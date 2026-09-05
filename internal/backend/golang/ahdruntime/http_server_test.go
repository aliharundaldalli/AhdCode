package ahdruntime

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func freeLoopbackPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func startHTTP(t *testing.T, maxBody int64, register func(handle string)) (base string, handle string) {
	t.Helper()
	port := freeLoopbackPort(t)
	handle = AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(port), maxBody)
	register(handle)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		AhdHTTPServerStart(AhdClassHTTPError, handle)
	}()
	t.Cleanup(func() {
		ahdHTTPTestShutdown(handle)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	})
	base = "http://127.0.0.1:" + strconv.Itoa(port)
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(base + "/__probe__")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			return base, handle
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("HTTP server did not start")
	return "", ""
}

func TestHTTPHelloPage(t *testing.T) {
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string {
			return AhdHTTPHTML(AhdClassHTTPError, "<!doctype html><html><body><h1>Hello from AhdCode</h1></body></html>", 200)
		})
	})
	response, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != 200 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if ct := response.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(string(body), "Hello from AhdCode") {
		t.Fatalf("body = %q", body)
	}
}

func TestHTTPExactRoutingAndErrors(t *testing.T) {
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/notes", func(data string) string {
			return AhdHTTPText(AhdClassHTTPError, "notes:"+AhdHTTPRequestPath(data)+":"+AhdHTTPRequestMethod(data), 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/ok", func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "ok", 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/throws", func(string) string {
			AhdRaiseClass(AhdClassDomainError, "boom")
			return AhdHTTPText(AhdClassHTTPError, "no", 200)
		})
	})
	client := &http.Client{Timeout: 2 * time.Second}
	get := func(path string) (int, string, http.Header) {
		t.Helper()
		response, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		return response.StatusCode, string(body), response.Header.Clone()
	}
	status, body, _ := get("/notes")
	if status != 200 || body != "notes:/notes:GET" {
		t.Fatalf("GET /notes = %d %q", status, body)
	}
	status, body, _ = get("/notes?q=x")
	if status != 200 || body != "notes:/notes:GET" {
		t.Fatalf("GET /notes?q=x = %d %q", status, body)
	}
	status, _, _ = get("/notes/")
	if status != 404 {
		t.Fatalf("GET /notes/ = %d, want 404", status)
	}
	request, _ := http.NewRequest(http.MethodPost, base+"/notes", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != 405 {
		t.Fatalf("POST /notes = %d, want 405", response.StatusCode)
	}
	if allow := response.Header.Get("Allow"); !strings.Contains(allow, "GET") {
		t.Fatalf("Allow = %q", allow)
	}
	status, _, _ = get("/unknown")
	if status != 404 {
		t.Fatalf("GET /unknown = %d, want 404", status)
	}

	status, body, _ = get("/ok")
	if status != 200 || body != "ok" {
		t.Fatalf("GET /ok = %d %q", status, body)
	}
	status, body, _ = get("/throws")
	if status != 500 || body != "Internal Server Error" {
		t.Fatalf("GET /throws = %d %q", status, body)
	}
	if strings.Contains(body, "boom") {
		t.Fatal("500 body leaked an internal message")
	}
	status, body, _ = get("/ok")
	if status != 200 || body != "ok" {
		t.Fatalf("server did not survive handler error: %d %q", status, body)
	}
}

func TestHTTPQueryFormHeadersAndDuplicates(t *testing.T) {
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/hello", func(data string) string {
			name := AhdHTTPRequestQuery(AhdClassHTTPError, data, "name")
			all := AhdHTTPRequestQueryAll(AhdClassHTTPError, data, "q")
			header := AhdHTTPRequestHeader(AhdClassHTTPError, data, "X-Test")
			headerAll := AhdHTTPRequestHeaderAll(AhdClassHTTPError, data, "x-test")
			first := ""
			if name != nil {
				first = *name
			}
			headerText := ""
			if header != nil {
				headerText = *header
			}
			return AhdHTTPText(AhdClassHTTPError, first+"|"+strings.Join(all.Snapshot(), ",")+":"+headerText+":"+strconv.Itoa(len(headerAll.Snapshot())), 200)
		})
		AhdHTTPServerPost(AhdClassHTTPError, handle, "/form", func(data string) string {
			title := AhdHTTPRequestForm(AhdClassHTTPError, data, "title")
			body := AhdHTTPRequestForm(AhdClassHTTPError, data, "body")
			tags := AhdHTTPRequestFormAll(AhdClassHTTPError, data, "tag")
			titleText, bodyText := "", ""
			if title != nil {
				titleText = *title
			}
			if body != nil {
				bodyText = *body
			}
			return AhdHTTPText(AhdClassHTTPError, titleText+"|"+bodyText+"|"+strings.Join(tags.Snapshot(), ","), 200)
		})
	})
	response, err := http.Get(base + "/hello?name=" + url.QueryEscape("Ayşe") + "&q=a&q=b&q=c")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 || string(body) != "Ayşe|a,b,c::0" {
		t.Fatalf("query = %d %q", response.StatusCode, body)
	}

	request, _ := http.NewRequest(http.MethodGet, base+"/hello?name=x", nil)
	request.Header.Add("X-Test", "one")
	request.Header.Add("x-test", "two")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if !strings.HasPrefix(string(body), "x|:") || !strings.Contains(string(body), ":one:") {
		t.Fatalf("headers = %q", body)
	}

	form := url.Values{}
	form.Set("title", "First Note")
	form.Set("body", "Hello World")
	form.Add("tag", "a")
	form.Add("tag", "b")
	response, err = http.Post(base+"/form", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "First Note|Hello World|a,b" {
		t.Fatalf("form = %q", body)
	}

	encoded := "title=First+Note&body=Hello%20World"
	response, err = http.Post(base+"/form", "application/x-www-form-urlencoded", strings.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "First Note|Hello World|" {
		t.Fatalf("urlencoded form = %q", body)
	}
}

func TestHTTPRejectsMalformedQueryAndFormBeforeHandler(t *testing.T) {
	var queryHits atomic.Int64
	var formHits atomic.Int64
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/search", func(data string) string {
			queryHits.Add(1)
			all := AhdHTTPRequestQueryAll(AhdClassHTTPError, data, "q")
			first := AhdHTTPRequestQuery(AhdClassHTTPError, data, "q")
			text := strings.Join(all.Snapshot(), ",")
			if first != nil {
				text = *first + "|" + text
			}
			return AhdHTTPText(AhdClassHTTPError, text, 200)
		})
		AhdHTTPServerPost(AhdClassHTTPError, handle, "/form", func(data string) string {
			formHits.Add(1)
			title := AhdHTTPRequestForm(AhdClassHTTPError, data, "title")
			text := ""
			if title != nil {
				text = *title
			}
			all := AhdHTTPRequestFormAll(AhdClassHTTPError, data, "title")
			return AhdHTTPText(AhdClassHTTPError, text+"|"+strings.Join(all.Snapshot(), ","), 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/ok", func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "ok", 200)
		})
	})
	client := &http.Client{Timeout: 2 * time.Second}

	getRaw := func(rawQuery string) (int, string) {
		t.Helper()
		request, err := http.NewRequest(http.MethodGet, base+"/search", nil)
		if err != nil {
			t.Fatal(err)
		}
		request.URL.RawQuery = rawQuery
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		return response.StatusCode, string(body)
	}
	postForm := func(raw string) (int, string) {
		t.Helper()
		response, err := client.Post(base+"/form", "application/x-www-form-urlencoded", strings.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		return response.StatusCode, string(body)
	}
	assertOK := func() {
		t.Helper()
		response, err := client.Get(base + "/ok")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode != 200 || string(body) != "ok" {
			t.Fatalf("GET /ok = %d %q", response.StatusCode, body)
		}
	}

	invalid := []string{"q=%", "q=%2", "q=%ZZ", "q=%80", "q=%C0%80"}
	for _, raw := range invalid {
		queryHits.Store(0)
		status, body := getRaw(raw)
		if status != 400 {
			t.Fatalf("GET /search?%s = %d %q, want 400", raw, status, body)
		}
		if strings.Contains(body, "\uFFFD") {
			t.Fatalf("GET /search?%s body contained U+FFFD: %q", raw, body)
		}
		if queryHits.Load() != 0 {
			t.Fatalf("GET /search?%s entered handler (hits=%d)", raw, queryHits.Load())
		}
		assertOK()
	}
	for _, raw := range []string{"title=%", "title=%2", "title=%ZZ", "title=%80", "title=%C0%80"} {
		formHits.Store(0)
		status, body := postForm(raw)
		if status != 400 {
			t.Fatalf("POST /form %s = %d %q, want 400", raw, status, body)
		}
		if strings.Contains(body, "\uFFFD") {
			t.Fatalf("POST /form %s body contained U+FFFD: %q", raw, body)
		}
		if formHits.Load() != 0 {
			t.Fatalf("POST /form %s entered handler (hits=%d)", raw, formHits.Load())
		}
		assertOK()
	}

	queryHits.Store(0)
	status, body := getRaw("q=Ay%C5%9Fe")
	if status != 200 || body != "Ayşe|Ayşe" || queryHits.Load() != 1 {
		t.Fatalf("Turkish query = %d %q hits=%d", status, body, queryHits.Load())
	}
	status, body = getRaw("q=%F0%9F%98%80")
	if status != 200 || body != "😀|😀" {
		t.Fatalf("emoji query = %d %q", status, body)
	}
	status, body = getRaw("q=AhdCode&q=SQLite")
	if status != 200 || body != "AhdCode|AhdCode,SQLite" {
		t.Fatalf("duplicate query = %d %q", status, body)
	}
	status, body = getRaw("q=a+b")
	if status != 200 || body != "a b|a b" {
		t.Fatalf("+ query = %d %q", status, body)
	}
	status, body = getRaw("q=a%20b")
	if status != 200 || body != "a b|a b" {
		t.Fatalf("%%20 query = %d %q", status, body)
	}

	formHits.Store(0)
	status, body = postForm("title=Ay%C5%9Fe")
	if status != 200 || body != "Ayşe|Ayşe" || formHits.Load() != 1 {
		t.Fatalf("Turkish form = %d %q hits=%d", status, body, formHits.Load())
	}
	status, body = postForm("title=%F0%9F%98%80")
	if status != 200 || body != "😀|😀" {
		t.Fatalf("emoji form = %d %q", status, body)
	}
	status, body = postForm("title=a&title=b&title=c")
	if status != 200 || body != "a|a,b,c" {
		t.Fatalf("duplicate form = %d %q", status, body)
	}
	status, body = postForm("title=First+Note")
	if status != 200 || body != "First Note|First Note" {
		t.Fatalf("+ form = %d %q", status, body)
	}
	status, body = postForm("title=Hello%20World")
	if status != 200 || body != "Hello World|Hello World" {
		t.Fatalf("%%20 form = %d %q", status, body)
	}
	assertOK()
}

func TestAhdHTTPParseEncodedRejectsMalformedAndInvalidUTF8(t *testing.T) {
	for _, raw := range []string{"q=%", "q=%2", "q=%ZZ", "q=%80", "q=%C0%80"} {
		values, err := ahdHTTPParseEncoded(raw)
		if err == nil {
			t.Fatalf("ParseEncoded(%q) succeeded: %#v", raw, values)
		}
	}
	values, err := ahdHTTPParseEncoded("q=Ay%C5%9Fe&q=%F0%9F%98%80&plus=a+b&sp=a%20b")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(values["q"], ","); got != "Ayşe,😀" {
		t.Fatalf("q = %q", got)
	}
	if values["plus"][0] != "a b" || values["sp"][0] != "a b" {
		t.Fatalf("plus/sp = %#v", values)
	}
}

func TestHTTPBodyLimitAndHeaderInjection(t *testing.T) {
	base, _ := startHTTP(t, 16, func(handle string) {
		AhdHTTPServerPost(AhdClassHTTPError, handle, "/echo", func(data string) string {
			return AhdHTTPText(AhdClassHTTPError, AhdHTTPRequestBody(AhdClassHTTPError, data), 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/ok", func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "ok", 200)
		})
	})
	response, err := http.Post(base+"/echo", "text/plain", strings.NewReader("1234567890123456"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 || string(body) != "1234567890123456" {
		t.Fatalf("exact limit = %d %q", response.StatusCode, body)
	}
	response, err = http.Post(base+"/echo", "text/plain", strings.NewReader("12345678901234567"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 413 {
		t.Fatalf("oversize = %d %q", response.StatusCode, body)
	}
	response, err = http.Get(base + "/ok")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("server died after 413: %d", response.StatusCode)
	}

	expectRaise(t, AhdClassHTTPError, func() {
		AhdHTTPResponseWithHeader(AhdClassHTTPError, AhdHTTPText(AhdClassHTTPError, "ok", 200), "X-Test", "good\r\nInjected: bad")
	})
	expectRaise(t, AhdClassHTTPError, func() {
		AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", 0, 10)
	})
	expectRaise(t, AhdClassHTTPError, func() {
		AhdHTTPRedirect(AhdClassHTTPError, "/", 200)
	})
}

func TestHTTPDuplicateRouteAndInvalidPath(t *testing.T) {
	handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", 8080, 1024)
	AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string { return AhdHTTPText(AhdClassHTTPError, "ok", 200) })
	expectRaise(t, AhdClassHTTPError, func() {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string { return AhdHTTPText(AhdClassHTTPError, "x", 200) })
	})
	expectRaise(t, AhdClassHTTPError, func() {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "notes", func(string) string { return AhdHTTPText(AhdClassHTTPError, "x", 200) })
	})
	expectRaise(t, AhdClassHTTPError, func() {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/notes?q=1", func(string) string { return AhdHTTPText(AhdClassHTTPError, "x", 200) })
	})
	expectRaise(t, AhdClassHTTPError, func() {
		AhdHTTPServerRoute(AhdClassHTTPError, handle, "GET GET", "/x", func(string) string { return AhdHTTPText(AhdClassHTTPError, "x", 200) })
	})
	expectRaise(t, AhdClassHTTPError, func() {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/*", func(string) string { return AhdHTTPText(AhdClassHTTPError, "x", 200) })
	})
	expectRaise(t, AhdClassHTTPError, func() {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/notes/*/edit", func(string) string { return AhdHTTPText(AhdClassHTTPError, "x", 200) })
	})
}

func TestHTTPWildcardOneSegment(t *testing.T) {
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/question/*", func(data string) string {
			return AhdHTTPText(AhdClassHTTPError, AhdHTTPRequestPath(data), 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/questions", func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "list", 200)
		})
	})
	client := &http.Client{Timeout: 2 * time.Second}
	get := func(path string) (int, string) {
		t.Helper()
		response, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		return response.StatusCode, string(body)
	}
	status, body := get("/question/2")
	if status != 200 || body != "/question/2" {
		t.Fatalf("GET /question/2 = %d %q", status, body)
	}
	status, body = get("/questions")
	if status != 200 || body != "list" {
		t.Fatalf("GET /questions = %d %q", status, body)
	}
	status, _ = get("/question")
	if status != 404 {
		t.Fatalf("GET /question = %d, want 404", status)
	}
	status, _ = get("/question/2/extra")
	if status != 404 {
		t.Fatalf("GET /question/2/extra = %d, want 404", status)
	}
	request, _ := http.NewRequest(http.MethodPost, base+"/question/2", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != 405 {
		t.Fatalf("POST /question/2 = %d, want 405", response.StatusCode)
	}
}

func TestHTTPHandlersAreSerialized(t *testing.T) {
	var running atomic.Int64
	var max atomic.Int64
	var count atomic.Int64
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/inc", func(string) string {
			current := running.Add(1)
			for {
				seen := max.Load()
				if current <= seen || max.CompareAndSwap(seen, current) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			running.Add(-1)
			return AhdHTTPText(AhdClassHTTPError, strconv.FormatInt(count.Add(1), 10), 200)
		})
	})
	var wait sync.WaitGroup
	wait.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wait.Done()
			response, err := http.Get(base + "/inc")
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode != 200 {
				t.Errorf("status = %d", response.StatusCode)
			}
		}()
	}
	wait.Wait()
	if max.Load() != 1 {
		t.Fatalf("handlers overlapped: max concurrent = %d", max.Load())
	}
	if count.Load() != 20 {
		t.Fatalf("count = %d, want 20", count.Load())
	}
}

func TestHTTPSequentialHundredsComplete(t *testing.T) {
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/ok", func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "ok", 200)
		})
	})
	for i := 0; i < 200; i++ {
		response, err := http.Get(base + "/ok")
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode != 200 || string(body) != "ok" {
			t.Fatalf("request %d = %d %q", i, response.StatusCode, body)
		}
	}
}

func TestHTTPRedirectDefaultIs303(t *testing.T) {
	data := AhdHTTPRedirect(AhdClassHTTPError, "/", 303)
	response := ahdHTTPDecodeResponse(AhdClassHTTPError, data)
	if response.Status != 303 {
		t.Fatalf("status = %d", response.Status)
	}
	if len(response.Headers) != 1 || response.Headers[0].Name != "Location" || response.Headers[0].Value != "/" {
		t.Fatalf("headers = %#v", response.Headers)
	}
}
