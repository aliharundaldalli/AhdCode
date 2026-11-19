package evaluator

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

type evaluatorRunOutcome struct {
	result   ExecutionResult
	panicked bool
	panicVal any
}

func formatEvaluatorOutcome(outcome evaluatorRunOutcome) string {
	if outcome.panicked {
		return fmt.Sprintf("evaluator panicked: %v", outcome.panicVal)
	}
	if outcome.result.Failure != nil {
		return fmt.Sprintf("evaluator failed (%s): %s", outcome.result.Failure.Name, outcome.result.Failure.Message)
	}
	return "evaluator returned before HTTP server became ready (no error reported)"
}

// launchEvaluatorProgram runs a compiled program on a background goroutine,
// capturing both ExecutionResult and any recovered panic without masking them.
// Once finished closes, reading *outcome is race-free.
func launchEvaluatorProgram(compilation *ir.Compilation) (finished <-chan struct{}, outcome *evaluatorRunOutcome) {
	session := newLatexTestSession()
	done := make(chan struct{})
	captured := &evaluatorRunOutcome{}
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				captured.panicked = true
				captured.panicVal = r
			}
		}()
		captured.result = session.Execute(compilation, 0)
	}()
	return done, captured
}

// stopEvaluatorProgram shuts down host:port and waits until the evaluator
// goroutine exits, nudging shutdown briefly in case Server.start had not yet
// bound when Cleanup began.
func stopEvaluatorProgram(host string, port int, finished <-chan struct{}) {
	_ = ahdruntime.AhdHTTPShutdownAddress(host, port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-finished:
			return
		case <-time.After(10 * time.Millisecond):
			_ = ahdruntime.AhdHTTPShutdownAddress(host, port)
		}
	}
}

// evaluatorHTTPGet fails within half a second. A callback blocked on the
// interpreter lock must make the test fail here instead of hanging the suite.
func evaluatorHTTPGet(url string) (*http.Response, error) {
	client := &http.Client{
		Timeout:   500 * time.Millisecond,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	return client.Get(url)
}

func waitForEvaluatorHTTP(compilation *ir.Compilation, port int, timeout time.Duration) (string, <-chan struct{}, *evaluatorRunOutcome, func(), error) {
	finished, outcome := launchEvaluatorProgram(compilation)
	var stopOnce sync.Once
	cleanup := func() {
		stopOnce.Do(func() {
			stopEvaluatorProgram("127.0.0.1", port, finished)
		})
	}
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case <-finished:
			cleanup()
			return "", finished, outcome, cleanup, fmt.Errorf("evaluator HTTP server did not start: %s", formatEvaluatorOutcome(*outcome))
		default:
		}
		response, err := evaluatorHTTPGet(base + "/ok")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return base, finished, outcome, cleanup, nil
			}
			lastErr = fmt.Errorf("GET %s/ok returned HTTP %d", base, response.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(10 * time.Millisecond)
	}
	cleanup()
	select {
	case <-finished:
		if outcome.panicked || outcome.result.Failure != nil {
			return "", finished, outcome, cleanup, fmt.Errorf("evaluator HTTP server did not start within %s: %s (last probe: %v)", timeout, formatEvaluatorOutcome(*outcome), lastErr)
		}
	default:
		return "", finished, outcome, cleanup, fmt.Errorf("evaluator HTTP server did not start within %s (evaluator still running at deadline; last probe: %v)", timeout, lastErr)
	}
	return "", finished, outcome, cleanup, fmt.Errorf("evaluator HTTP server did not start within %s (last probe: %v)", timeout, lastErr)
}

func startEvaluatorHTTPWithDone(t *testing.T, source string, port int) (string, <-chan struct{}, *evaluatorRunOutcome) {
	t.Helper()
	compilation := compileAhd(t, source)
	base, finished, outcome, cleanup, err := waitForEvaluatorHTTP(compilation, port, 2*time.Second)
	t.Cleanup(cleanup)
	if err != nil {
		t.Fatal(err)
	}
	return base, finished, outcome
}

func startEvaluatorHTTP(t *testing.T, source string, port int) string {
	t.Helper()
	base, _, _ := startEvaluatorHTTPWithDone(t, source, port)
	return base
}

func freeEvaluatorPort(t *testing.T) int {
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

func TestHTTPEvaluatorCookiePrimitives(t *testing.T) {
	session := newLatexTestSession()
	cookie := session.httpBuiltin("cookie", []any{"theme", "dark"})
	other := session.httpOperation("Cookie.withHttpOnly", cookie, []any{true})
	if other == cookie {
		t.Fatal("Cookie.withHttpOnly must return a new Cookie")
	}
	expectEvaluatorRaise(t, "HTTPError", func() {
		session.httpBuiltin("cookie", []any{"bad name", "x"})
	})
	expectEvaluatorRaise(t, "HTTPError", func() {
		session.httpBuiltin("cookie", []any{"ok", "x\r\nY: 1"})
	})
	expectEvaluatorRaise(t, "HTTPError", func() {
		session.httpOperation("Cookie.withSameSite", cookie, []any{"lax"})
	})
	expectEvaluatorRaise(t, "HTTPError", func() {
		session.httpOperation("Cookie.withSameSite", cookie, []any{"None"})
	})
	text := session.httpBuiltin("text", []any{"ok"})
	headed := session.httpOperation("Response.withCookie", text, []any{cookie})
	if headed == text {
		t.Fatal("withCookie must return a new Response")
	}
	expectEvaluatorRaise(t, "HTTPError", func() {
		session.httpOperation("Response.withHeader", text, []any{"Set-Cookie", "a=1"})
	})
}

func TestHTTPEvaluatorRequestAndResponseCookies(t *testing.T) {
	port := freeEvaluatorPort(t)
	base := startEvaluatorHTTP(t, `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring Cookie

read: Function := (request: Request) -> Response {
    first: Local String? := request.cookie("theme")
    values: Local List<String> := request.cookieAll("theme")
    if first == null {
        return HTTP.text("missing")
    }
    return HTTP.text("{first}:{str(len(values))}")
}

set: Function := (request: Request) -> Response {
    response: Local Response := HTTP.text("set")
    response = response.withCookie(HTTP.cookie("a", "1"))
    response = response.withCookie(HTTP.cookie("b", "2").withHttpOnly(true).withMaxAge(60))
    return response
}

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

app: Server := HTTP.server("127.0.0.1", `+strconv.Itoa(port)+`)
app.get("/ok", ok)
app.get("/read", read)
app.get("/set", set)
app.start()
`, port)

	response, err := http.Get(base + "/read")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "missing" {
		t.Fatalf("absent cookie = %q", body)
	}

	request, err := http.NewRequest(http.MethodGet, base+"/read", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("Cookie", "theme=dark")
	request.Header.Add("Cookie", "theme=light")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "dark:2" {
		t.Fatalf("duplicate cookies = %q", body)
	}

	response, err = http.Get(base + "/set")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	headers := response.Header["Set-Cookie"]
	if len(headers) != 2 {
		t.Fatalf("Set-Cookie = %#v", headers)
	}
	joined := strings.Join(headers, "\n")
	if !strings.Contains(joined, "a=1") || !strings.Contains(joined, "b=2") || !strings.Contains(joined, "HttpOnly") {
		t.Fatalf("cookie attributes = %#v", headers)
	}
}

func TestHTTPEvaluatorTwoClientSessions(t *testing.T) {
	port := freeEvaluatorPort(t)
	base := startEvaluatorHTTP(t, sessionLoginSource(port), port)
	jarA, _ := cookiejar.New(nil)
	jarB, _ := cookiejar.New(nil)
	clientA := &http.Client{Jar: jarA, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	clientB := &http.Client{Jar: jarB, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	login := func(client *http.Client, name string) {
		t.Helper()
		response, err := client.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader("name="+url.QueryEscape(name)))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if response.StatusCode != 303 {
			t.Fatalf("login %s = %d", name, response.StatusCode)
		}
	}
	panel := func(client *http.Client) string {
		t.Helper()
		response, err := client.Get(base + "/panel")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode == 303 {
			return "anon"
		}
		return string(body)
	}

	login(clientA, "Ali")
	login(clientB, "Mehmet")
	if panel(clientA) != "Ali" || panel(clientB) != "Mehmet" {
		t.Fatalf("A=%q B=%q", panel(clientA), panel(clientB))
	}
	if panel(clientA) != "Ali" || panel(clientB) != "Mehmet" {
		t.Fatal("second panel read mixed clients")
	}
	response, err := clientA.Post(base+"/logout", "application/x-www-form-urlencoded", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if panel(clientA) != "anon" {
		t.Fatal("A logout did not clear A")
	}
	if panel(clientB) != "Mehmet" {
		t.Fatal("A logout logged out B")
	}
}

func TestHTTPEvaluatorSessionFixation(t *testing.T) {
	port := freeEvaluatorPort(t)
	base := startEvaluatorHTTP(t, sessionLoginSource(port), port)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get(base + "/anon")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	cookies := jar.Cookies(mustParse(t, base+"/anon"))
	if len(cookies) != 1 {
		t.Fatalf("pre-login cookies = %#v", cookies)
	}
	oldID := cookies[0].Value
	response, err = client.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader("name=Ali"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	newCookies := jar.Cookies(mustParse(t, base+"/panel"))
	if len(newCookies) != 1 || newCookies[0].Value == oldID {
		t.Fatalf("login did not rotate: old=%q new=%#v", oldID, newCookies)
	}
	request, err := http.NewRequest(http.MethodGet, base+"/panel", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: "ahd_session", Value: oldID})
	bare := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err = bare.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 303 {
		t.Fatalf("old id still authenticated: %d %q", response.StatusCode, body)
	}
	response, err = client.Get(base + "/panel")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "Ali" {
		t.Fatalf("new id lost values: %q", body)
	}
}

func TestHTTPEvaluatorDeleteCookieClearsJar(t *testing.T) {
	port := freeEvaluatorPort(t)
	base := startEvaluatorHTTP(t, `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response

set: Function := (request: Request) -> Response {
    return HTTP.text("set").withCookie(HTTP.cookie("theme", "dark"))
}

del: Function := (request: Request) -> Response {
    return HTTP.text("del").withCookie(HTTP.deleteCookie("theme"))
}

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

app: Server := HTTP.server("127.0.0.1", `+strconv.Itoa(port)+`)
app.get("/ok", ok)
app.get("/set", set)
app.get("/del", del)
app.start()
`, port)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	if _, err := client.Get(base + "/set"); err != nil {
		t.Fatal(err)
	}
	if len(jar.Cookies(mustParse(t, base+"/set"))) != 1 {
		t.Fatal("jar did not store cookie")
	}
	if _, err := client.Get(base + "/del"); err != nil {
		t.Fatal(err)
	}
	if remaining := jar.Cookies(mustParse(t, base+"/del")); len(remaining) != 0 {
		t.Fatalf("jar still has %#v", remaining)
	}
}

func TestHTTPEvaluatorHandlerCookieErrorIsContained(t *testing.T) {
	port := freeEvaluatorPort(t)
	base := startEvaluatorHTTP(t, `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response

boom: Function := (request: Request) -> Response {
    return HTTP.text("x").withCookie(HTTP.cookie("bad name", "x"))
}

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

app: Server := HTTP.server("127.0.0.1", `+strconv.Itoa(port)+`)
app.get("/ok", ok)
app.get("/boom", boom)
app.start()
`, port)
	response, err := http.Get(base + "/boom")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 500 || strings.Contains(string(body), "bad name") {
		t.Fatalf("contained error = %d %q", response.StatusCode, body)
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
}

func sessionLoginSource(port int) string {
	return `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring SessionStore
from HTTP bring Session

sessions: SessionStore := HTTP.sessions("ahd_session", 3600, false, "Lax")

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

anon: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    session.set("flash", "1")
    return sessions.commit(session, HTTP.text("anon"))
}

login: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    name: Local String? := request.form("name")
    if name == null {
        return sessions.commit(session, HTTP.text("name is required", 400))
    }
    session.rotate()
    session.set("name", name)
    return sessions.commit(session, HTTP.redirect("/panel"))
}

panel: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    name: Local String? := session.get("name")
    if name == null {
        return sessions.commit(session, HTTP.redirect("/"))
    }
    return sessions.commit(session, HTTP.text(name))
}

logout: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    session.destroy()
    return sessions.commit(session, HTTP.redirect("/"))
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", ok)
app.get("/anon", anon)
app.post("/login", login)
app.get("/panel", panel)
app.post("/logout", logout)
app.start()
`
}

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
