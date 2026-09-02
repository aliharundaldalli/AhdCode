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
