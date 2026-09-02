package ahdruntime

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
)

func TestHTTPClientRequestImmutabilityAndHeaders(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {}))
	defer server.Close()
	original := AhdHTTPClientRequest(class, "GET", server.URL)
	withHeader := AhdHTTPClientRequestWithHeader(class, original, "X-Test", "one")
	replaced := AhdHTTPClientRequestWithHeader(class, withHeader, "x-test", "two")
	appended := AhdHTTPClientRequestAddHeader(class, replaced, "X-Test", "three")
	withBody := AhdHTTPClientRequestWithBody(class, appended, "hello")
	if ahdHTTPDecodeClientRequest(class, original).Headers != nil && len(ahdHTTPDecodeClientRequest(class, original).Headers) != 0 {
		t.Fatal("original request gained headers")
	}
	if ahdHTTPDecodeClientRequest(class, original).Body != "" {
		t.Fatal("original request gained a body")
	}
	if ahdHTTPDecodeClientRequest(class, withHeader).Body != "" {
		t.Fatal("header-only request gained a body")
	}
	if got := ahdHTTPDecodeClientRequest(class, withHeader).Headers; len(got) != 1 || got[0].Value != "one" {
		t.Fatalf("withHeader = %#v", got)
	}
	if got := ahdHTTPDecodeClientRequest(class, replaced).Headers; len(got) != 1 || got[0].Value != "two" {
		t.Fatalf("withHeader replace = %#v", got)
	}
	if got := ahdHTTPDecodeClientRequest(class, appended).Headers; len(got) != 2 || got[0].Value != "two" || got[1].Value != "three" {
		t.Fatalf("addHeader = %#v", got)
	}
	if ahdHTTPDecodeClientRequest(class, appended).Body != "" {
		t.Fatal("addHeader mutated the body")
	}
	decoded := ahdHTTPDecodeClientRequest(class, withBody)
	if decoded.Body != "hello" || len(decoded.Headers) != 2 {
		t.Fatalf("withBody = %#v", decoded)
	}
}

func TestHTTPClientURLValidation(t *testing.T) {
	class := AhdClassHTTPError
	for _, raw := range []string{
		"/api", "ftp://example.com/", "file:///tmp/x", "javascript:alert(1)",
		"data:text/plain,hi", "http://", "https://", "not a url",
		"http://example.com/path#frag", "http://user:pass@127.0.0.1/",
	} {
		expectRaise(t, class, func() { AhdHTTPClientRequest(class, "GET", raw) })
	}
	valid := AhdHTTPClientRequest(class, "GET", "http://127.0.0.1/")
	if ahdHTTPDecodeClientRequest(class, valid).URL != "http://127.0.0.1/" {
		t.Fatal("valid loopback URL was rejected")
	}
	https := AhdHTTPClientRequest(class, "GET", "https://example.com/")
	if ahdHTTPDecodeClientRequest(class, https).Method != "GET" {
		t.Fatal("HTTPS URL constructor failed")
	}
}

func TestHTTPClientDoesNotUppercaseMethod(t *testing.T) {
	class := AhdClassHTTPError
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		seen = request.Method
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := AhdHTTPClient(class, 5, 1024, true)
	AhdHTTPClientSend(class, client, AhdHTTPClientRequest(class, "get", server.URL))
	if seen != "get" {
		t.Fatalf("method = %q; want get", seen)
	}
}

func TestHTTPClientHeaderSecurity(t *testing.T) {
	class := AhdClassHTTPError
	request := AhdHTTPClientRequest(class, "GET", "http://127.0.0.1/")
	expectRaise(t, class, func() { AhdHTTPClientRequestWithHeader(class, request, "X Test", "v") })
	expectRaise(t, class, func() { AhdHTTPClientRequestWithHeader(class, request, "X:Test", "v") })
	expectRaise(t, class, func() { AhdHTTPClientRequestWithHeader(class, request, "X-Test", "good\r\nInjected: bad") })
	expectRaise(t, class, func() { AhdHTTPClientRequestWithHeader(class, request, "X-Test", "good\nInjected: bad") })
	expectRaise(t, class, func() { AhdHTTPClientRequestWithHeader(class, request, "Content-Length", "9") })
	expectRaise(t, class, func() { AhdHTTPClientRequestWithHeader(class, request, "Host", "evil.test") })
	AhdHTTPClientRequestWithHeader(class, request, "Authorization", "Bearer ordinary-token")
}

func TestHTTPClientLocalGETPOSTHeadersAndStatuses(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("X-Echo", request.Header.Get("X-App"))
		writer.Header().Add("X-Dup", "one")
		writer.Header().Add("X-Dup", "two")
		switch {
		case request.URL.Path == "/missing":
			writer.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(writer, `{"error":"missing"}`)
		case request.URL.Path == "/fail":
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(writer, `{"error":"boom"}`)
		case request.URL.Path == "/denied":
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(writer, `{"error":"auth"}`)
		case request.URL.Path == "/limited":
			writer.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(writer, `{"error":"slow"}`)
		case request.URL.Path == "/echo":
			body, _ := io.ReadAll(request.Body)
			if request.Header.Get("Content-Type") != "application/json" {
				t.Errorf("content-type = %q", request.Header.Get("Content-Type"))
			}
			if request.Header.Get("X-App") != "" {
				if got := request.Header.Values("X-Trace"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
					t.Errorf("X-Trace = %#v", got)
				}
			}
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write(body)
		default:
			writer.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	client := AhdHTTPClient(class, 5, 1024, true)
	empty := ahdHTTPDecodeClientResponse(class, AhdHTTPClientGet(class, client, server.URL+"/ok"))
	if empty.Status != 204 || empty.Body != "" || empty.URL == "" {
		t.Fatalf("GET 204 = %#v", empty)
	}
	if ahdHTTPClientResponseHeaderValue(class, AhdHTTPClientGet(class, client, server.URL+"/ok"), "x-echo") != "" {
		t.Fatal("unexpected echo header on GET")
	}
	posted := AhdHTTPClientPost(class, client, server.URL+"/echo", `{"n":1}`, "application/json")
	if ahdHTTPDecodeClientResponse(class, posted).Body != `{"n":1}` {
		t.Fatalf("POST body = %q", ahdHTTPDecodeClientResponse(class, posted).Body)
	}
	request := AhdHTTPClientRequest(class, "POST", server.URL+"/echo")
	request = AhdHTTPClientRequestWithHeader(class, request, "Content-Type", "application/json")
	request = AhdHTTPClientRequestAddHeader(class, request, "X-Trace", "a")
	request = AhdHTTPClientRequestAddHeader(class, request, "X-Trace", "b")
	request = AhdHTTPClientRequestWithHeader(class, request, "X-App", "AhdCode")
	request = AhdHTTPClientRequestWithBody(class, request, `{"ok":true}`)
	custom := ahdHTTPDecodeClientResponse(class, AhdHTTPClientSend(class, client, request))
	if custom.Status != 200 || custom.Body != `{"ok":true}` {
		t.Fatalf("custom = %#v", custom)
	}
	if got := AhdHTTPClientResponseHeader(class, AhdHTTPClientSend(class, client, request), "x-echo"); got == nil || *got != "AhdCode" {
		t.Fatalf("header = %v", got)
	}
	values := AhdHTTPClientResponseHeaderAll(class, AhdHTTPClientSend(class, client, request), "X-Dup").Snapshot()
	if len(values) != 2 || values[0] != "one" || values[1] != "two" {
		t.Fatalf("headerAll = %#v", values)
	}
	for _, path := range []string{"/missing", "/fail", "/denied", "/limited"} {
		response := ahdHTTPDecodeClientResponse(class, AhdHTTPClientGet(class, client, server.URL+path))
		if response.Status < 400 || response.Body == "" {
			t.Fatalf("%s = %#v", path, response)
		}
	}
}

func TestHTTPClientTimeoutThenRemainsHealthy(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/slow" {
			time.Sleep(3 * time.Second)
		}
		_, _ = io.WriteString(writer, "ok")
	}))
	defer server.Close()
	client := AhdHTTPClient(class, 1, 1024, true)
	started := time.Now()
	message := mustRaiseHTTP(t, func() { AhdHTTPClientGet(class, client, server.URL+"/slow") })
	elapsed := time.Since(started)
	if !strings.Contains(message, "timed out") {
		t.Fatalf("timeout message = %q", message)
	}
	if elapsed < 800*time.Millisecond || elapsed > 2500*time.Millisecond {
		t.Fatalf("timeout window = %s", elapsed)
	}
	got := ahdHTTPDecodeClientResponse(class, AhdHTTPClientGet(class, client, server.URL+"/ok"))
	if got.Status != 200 || got.Body != "ok" {
		t.Fatalf("client died after timeout: %#v", got)
	}
}

func TestHTTPClientResponseSizeBoundary(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		n := 8
		if request.URL.Path == "/over" {
			n = 9
		}
		_, _ = writer.Write([]byte(strings.Repeat("a", n)))
	}))
	defer server.Close()
	client := AhdHTTPClient(class, 5, 8, true)
	exact := ahdHTTPDecodeClientResponse(class, AhdHTTPClientGet(class, client, server.URL+"/exact"))
	if exact.Body != strings.Repeat("a", 8) {
		t.Fatalf("exact = %q", exact.Body)
	}
	message := mustRaiseHTTP(t, func() { AhdHTTPClientGet(class, client, server.URL+"/over") })
	if !strings.Contains(message, "maxResponseBytes") {
		t.Fatalf("oversize message = %q", message)
	}
}

func TestHTTPClientRejectsInvalidUTF8(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte{0xff, 0xfe})
	}))
	defer server.Close()
	client := AhdHTTPClient(class, 5, 1024, true)
	message := mustRaiseHTTP(t, func() { AhdHTTPClientGet(class, client, server.URL) })
	if !strings.Contains(message, "UTF-8") || strings.Contains(message, "\uFFFD") {
		t.Fatalf("utf-8 message = %q", message)
	}
	if utf8.Valid([]byte{0xff, 0xfe}) {
		t.Fatal("fixture is valid UTF-8")
	}
}

func TestHTTPClientRedirects(t *testing.T) {
	class := AhdClassHTTPError
	final := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" {
			t.Errorf("sensitive headers leaked: %#v", request.Header)
		}
		_, _ = io.WriteString(writer, "done")
	}))
	defer final.Close()
	same := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/next" {
			if request.Header.Get("Authorization") != "Bearer keep" {
				t.Errorf("same-host Authorization missing: %q", request.Header.Get("Authorization"))
			}
			_, _ = io.WriteString(writer, "same")
			return
		}
		http.Redirect(writer, request, "/next", http.StatusFound)
	}))
	defer same.Close()
	cross := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, final.URL, http.StatusFound)
	}))
	defer cross.Close()
	loop := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, request.URL.String(), http.StatusFound)
	}))
	defer loop.Close()

	held := AhdHTTPClient(class, 5, 1024, false)
	redirected := ahdHTTPDecodeClientResponse(class, AhdHTTPClientGet(class, held, same.URL+"/start"))
	if redirected.Status != 302 {
		t.Fatalf("follow disabled = %#v", redirected)
	}
	follower := AhdHTTPClient(class, 5, 1024, true)
	sameHost := AhdHTTPClientRequest(class, "GET", same.URL+"/start")
	sameHost = AhdHTTPClientRequestWithHeader(class, sameHost, "Authorization", "Bearer keep")
	got := ahdHTTPDecodeClientResponse(class, AhdHTTPClientSend(class, follower, sameHost))
	if got.Status != 200 || got.Body != "same" || !strings.Contains(got.URL, "/next") {
		t.Fatalf("same-host follow = %#v", got)
	}
	crossRequest := AhdHTTPClientRequest(class, "GET", cross.URL)
	crossRequest = AhdHTTPClientRequestWithHeader(class, crossRequest, "Authorization", "Bearer secret")
	crossRequest = AhdHTTPClientRequestWithHeader(class, crossRequest, "Cookie", "sid=1")
	crossed := ahdHTTPDecodeClientResponse(class, AhdHTTPClientSend(class, follower, crossRequest))
	if crossed.Body != "done" {
		t.Fatalf("cross-host follow = %#v", crossed)
	}
	message := mustRaiseHTTP(t, func() { AhdHTTPClientGet(class, follower, loop.URL) })
	if !strings.Contains(message, "redirect") {
		t.Fatalf("loop message = %q", message)
	}
}

func TestHTTPClientRedirectPolicy(t *testing.T) {
	state := &ahdHTTPClientState{followRedirects: true}
	httpsFrom, _ := url.Parse("https://api.example.com/v1")
	httpTo, _ := url.Parse("http://api.example.com/v1")
	next, _ := http.NewRequest(http.MethodGet, httpTo.String(), nil)
	previous, _ := http.NewRequest(http.MethodGet, httpsFrom.String(), nil)
	if err := ahdHTTPClientRedirect(state, next, []*http.Request{previous}); !errors.Is(err, errHTTPHTTPSDowngrade) {
		t.Fatalf("downgrade = %v", err)
	}
	other, _ := http.NewRequest(http.MethodGet, "https://other.example.com/", nil)
	other.Header.Set("Authorization", "Bearer leak")
	other.Header.Set("Cookie", "sid=1")
	from, _ := http.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	if err := ahdHTTPClientRedirect(state, other, []*http.Request{from}); err != nil {
		t.Fatal(err)
	}
	if other.Header.Get("Authorization") != "" || other.Header.Get("Cookie") != "" {
		t.Fatalf("credentials survived host change: %#v", other.Header)
	}
	disabled := &ahdHTTPClientState{followRedirects: false}
	if err := ahdHTTPClientRedirect(disabled, next, []*http.Request{previous}); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("disabled follow = %v", err)
	}
}

func TestHTTPClientUntrustedTLSFails(t *testing.T) {
	class := AhdClassHTTPError
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(writer, "no")
	}))
	defer server.Close()
	client := AhdHTTPClient(class, 5, 1024, true)
	request := AhdHTTPClientRequest(class, "GET", server.URL)
	request = AhdHTTPClientRequestWithHeader(class, request, "Authorization", "Bearer SUPER_SECRET_TEST_VALUE")
	message := mustRaiseHTTP(t, func() { AhdHTTPClientSend(class, client, request) })
	if !strings.Contains(message, "TLS") {
		t.Fatalf("tls message = %q", message)
	}
	if strings.Contains(message, "SUPER_SECRET_TEST_VALUE") {
		t.Fatalf("secret leaked: %q", message)
	}
}

func TestHTTPClientTrustedHTTPSIfNetworkAvailable(t *testing.T) {
	class := AhdClassHTTPError
	client := AhdHTTPClient(class, 15, ahdHTTPDefaultClientMaxBody, true)
	var response string
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				signal, ok := recovered.(*AhdSignal)
				if !ok {
					t.Fatalf("unexpected panic: %v", recovered)
				}
				t.Skipf("public HTTPS environment-blocked: %s", signal.Message)
			}
		}()
		response = AhdHTTPClientGet(class, client, "https://example.com/")
	}()
	decoded := ahdHTTPDecodeClientResponse(class, response)
	if decoded.Status < 200 || decoded.Status >= 400 || decoded.Body == "" || !strings.Contains(decoded.URL, "https://") {
		t.Fatalf("public HTTPS = %#v", decoded)
	}
}

func TestHTTPClientInvalidConfiguration(t *testing.T) {
	class := AhdClassHTTPError
	expectRaise(t, class, func() { AhdHTTPClient(class, 0, 1024, true) })
	expectRaise(t, class, func() { AhdHTTPClient(class, 1, 0, true) })
}

func TestHTTPClientRace(t *testing.T) {
	class := AhdClassHTTPError
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		hits.Add(1)
		_, _ = io.WriteString(writer, "ok")
	}))
	defer server.Close()
	client := AhdHTTPClient(class, 5, 1024, true)
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			got := ahdHTTPDecodeClientResponse(class, AhdHTTPClientGet(class, client, server.URL))
			if got.Body != "ok" {
				t.Errorf("body = %q", got.Body)
			}
		}()
	}
	wait.Wait()
	if hits.Load() != 8 {
		t.Fatalf("hits = %d", hits.Load())
	}
}

func ahdHTTPClientResponseHeaderValue(class *AhdClass, data, name string) string {
	value := AhdHTTPClientResponseHeader(class, data, name)
	if value == nil {
		return ""
	}
	return *value
}

func mustRaiseHTTP(t *testing.T, body func()) string {
	t.Helper()
	var message string
	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatal("expected HTTPError")
			}
			signal, ok := recovered.(*AhdSignal)
			if !ok || signal.Instance.AhdClassOf() != AhdClassHTTPError {
				t.Fatalf("expected HTTPError; received %v", recovered)
			}
			message = signal.Message
		}()
		body()
	}()
	return message
}
