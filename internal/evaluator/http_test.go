package evaluator

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"ahdcode/internal/ir"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
)

func TestHTTPEvaluatorConstructsResponses(t *testing.T) {
	session := newLatexTestSession()
	text := session.httpBuiltin("text", []any{"ok", int64(201)})
	html := session.httpBuiltin("html", []any{"<h1>Hi</h1>"})
	redirect := session.httpBuiltin("redirect", []any{"/"})
	custom := session.httpBuiltin("response", []any{int64(204), "", "text/plain"})
	if text == nil || html == nil || redirect == nil || custom == nil {
		t.Fatal("HTTP constructors returned nil")
	}
	headed := session.httpOperation("Response.withHeader", text, []any{"X-App", "AhdCode"})
	if headed == text {
		t.Fatal("withHeader must return a copy")
	}
	expectEvaluatorRaise(t, "HTTPError", func() {
		session.httpBuiltin("server", []any{"127.0.0.1", int64(0), int64(10)})
	})
	expectEvaluatorRaise(t, "HTTPError", func() {
		session.httpBuiltin("redirect", []any{"/", int64(200)})
	})
	expectEvaluatorRaise(t, "HTTPError", func() {
		session.httpOperation("Response.withHeader", text, []any{"X-Test", "good\r\nInjected: bad"})
	})
}

func compileAhd(t *testing.T, source string) *ir.Compilation {
	t.Helper()
	workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": source})
	frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
	if frontend.HasErrors() {
		t.Fatalf("frontend diagnostics: %+v", frontend.Diagnostics)
	}
	result := lowering.LowerCompilation(frontend)
	if result.HasErrors() {
		t.Fatalf("lowering diagnostics: %+v", result.Diagnostics)
	}
	return result.Compilation
}

func TestHTTPEvaluatorServesAHandler(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	source := `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response

home: Function := (request: Request) -> Response {
    name: Local String? := request.query("name")
    if name != null {
        return HTTP.text(name)
    }
    return HTTP.text("ok")
}

throws: Function := (request: Request) -> Response {
    toss (DomainError("boom"))
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", home)
app.get("/throws", throws)
app.start()
`
	compilation := compileAhd(t, source)
	session := newLatexTestSession()
	go func() {
		defer func() { _ = recover() }()
		_ = session.Execute(compilation, 0)
	}()
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(base + "/ok")
		if err == nil {
			body, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if response.StatusCode != 200 || string(body) != "ok" {
				t.Fatalf("GET /ok = %d %q", response.StatusCode, body)
			}
			response, err = http.Get(base + "/ok?name=" + url.QueryEscape("Ayşe"))
			if err != nil {
				t.Fatal(err)
			}
			body, _ = io.ReadAll(response.Body)
			_ = response.Body.Close()
			if string(body) != "Ayşe" {
				t.Fatalf("query name = %q", body)
			}
			response, err = http.Get(base + "/throws")
			if err != nil {
				t.Fatal(err)
			}
			body, _ = io.ReadAll(response.Body)
			_ = response.Body.Close()
			if response.StatusCode != 500 || !strings.Contains(string(body), "Internal Server Error") {
				t.Fatalf("GET /throws = %d %q", response.StatusCode, body)
			}
			response, err = http.Get(base + "/ok")
			if err != nil {
				t.Fatal(err)
			}
			body, _ = io.ReadAll(response.Body)
			_ = response.Body.Close()
			if response.StatusCode != 200 || string(body) != "ok" {
				t.Fatalf("server did not survive: %d %q", response.StatusCode, body)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("evaluator HTTP server did not start")
}

func TestHTTPEvaluatorRejectsMalformedQueryAndFormBeforeHandler(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	source := `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response

hits: Int := 0

search: Function := (request: Request) -> Response {
    hits: Global Int
    hits = hits + 1
    q: Local String? := request.query("q")
    if q != null {
        return HTTP.text(q)
    }
    return HTTP.text("missing")
}

save: Function := (request: Request) -> Response {
    hits: Global Int
    hits = hits + 1
    title: Local String? := request.form("title")
    if title != null {
        return HTTP.text(title)
    }
    return HTTP.text("missing")
}

status: Function := (request: Request) -> Response {
    hits: Global Int
    return HTTP.text(str(hits))
}

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/search", search)
app.post("/form", save)
app.get("/hits", status)
app.get("/ok", ok)
app.start()
`
	compilation := compileAhd(t, source)
	session := newLatexTestSession()
	go func() {
		defer func() { _ = recover() }()
		_ = session.Execute(compilation, 0)
	}()
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	client := &http.Client{Timeout: 2 * time.Second}
	started := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(base + "/ok")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				started = true
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !started {
		t.Fatal("evaluator HTTP server did not start")
	}

	hits := func() string {
		t.Helper()
		response, err := client.Get(base + "/hits")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		return string(body)
	}
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

	if hits() != "0" {
		t.Fatalf("hits before = %q", hits())
	}
	for _, raw := range []string{"q=%", "q=%2", "q=%ZZ", "q=%80", "q=%C0%80"} {
		status, body := getRaw(raw)
		if status != 400 || strings.Contains(body, "\uFFFD") {
			t.Fatalf("GET /search?%s = %d %q", raw, status, body)
		}
		if hits() != "0" {
			t.Fatalf("query handler ran after %s: hits=%s", raw, hits())
		}
	}
	for _, raw := range []string{"title=%", "title=%2", "title=%ZZ", "title=%80", "title=%C0%80"} {
		status, body := postForm(raw)
		if status != 400 || strings.Contains(body, "\uFFFD") {
			t.Fatalf("POST /form %s = %d %q", raw, status, body)
		}
		if hits() != "0" {
			t.Fatalf("form handler ran after %s: hits=%s", raw, hits())
		}
	}
	status, body := getRaw("q=Ay%C5%9Fe")
	if status != 200 || body != "Ayşe" {
		t.Fatalf("Turkish query = %d %q", status, body)
	}
	status, body = postForm("title=Hello%20World")
	if status != 200 || body != "Hello World" {
		t.Fatalf("form %%20 = %d %q", status, body)
	}
	response, err := client.Get(base + "/ok")
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 || string(payload) != "ok" {
		t.Fatalf("GET /ok afterward = %d %q", response.StatusCode, payload)
	}
}

func TestHTMLEvaluatorProgramOutputIsEscaped(t *testing.T) {
	var output bytes.Buffer
	session := New(bufio.NewReader(strings.NewReader("")), &output, "")
	compilation := compileAhd(t, `bring HTML
write(HTML.render(HTML.text("<script>alert(1)</script>")))
write(HTML.document("Tom & Jerry", [HTML.text("Ayşe ☕")]))
`)
	if result := session.Execute(compilation, 0); result.Failure != nil {
		t.Fatalf("execute: %v", result.Failure)
	}
	got := output.String()
	if strings.Contains(got, "<script>") || !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("output = %q", got)
	}
	if !strings.Contains(got, "Tom &amp; Jerry") || !strings.Contains(got, "Ayşe ☕") {
		t.Fatalf("document output = %q", got)
	}
}
