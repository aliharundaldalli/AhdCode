package evaluator

import (
	"bufio"
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
)

// The interpreter lock must be free while Server.start is blocked in Serve.
// Request callbacks take that lock one at a time. Shutdown lets start return
// and releases the port so another server can bind it.
func TestEvaluatorHTTPServerCallbacksShutdownAndRestart(t *testing.T) {
	port := freeEvaluatorPort(t)
	base, done, output := startLockedHTTP(t, evaluatorHTTPProgram(port), "/note")
	client := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}

	query := url.Values{}
	query.Set("q", "İstanbul")
	status, body, contentType := evaluatorHTTPRead(t, client, base+"/note?"+query.Encode())
	if status != 200 || body != "İstanbul" || contentType != "text/plain; charset=utf-8" {
		t.Fatalf("GET /note = %d %q %q", status, body, contentType)
	}

	const requests = 24
	var group sync.WaitGroup
	values := make([]string, requests)
	for index := 0; index < requests; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			response, err := client.Get(base + "/hit")
			if err != nil {
				values[index] = "error:" + err.Error()
				return
			}
			payload, err := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if err != nil || response.StatusCode != http.StatusOK {
				values[index] = "bad"
				return
			}
			values[index] = string(payload)
		}(index)
	}
	group.Wait()
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	if len(seen) != requests {
		t.Fatalf("concurrent callbacks lost updates: %v", values)
	}
	for number := 1; number <= requests; number++ {
		if !seen[strconv.Itoa(number)] {
			t.Fatalf("missing callback result %d in %v", number, values)
		}
	}
	if _, hits, _ := evaluatorHTTPRead(t, client, base+"/hits"); hits != strconv.Itoa(requests) {
		t.Fatalf("stored hits = %q", hits)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	browser := &http.Client{Timeout: 2 * time.Second, Jar: jar, Transport: &http.Transport{DisableKeepAlives: true}}
	if _, first, _ := evaluatorHTTPRead(t, browser, base+"/count"); first != "1" {
		t.Fatalf("first session read = %q", first)
	}
	if _, second, _ := evaluatorHTTPRead(t, browser, base+"/count"); second != "2" {
		t.Fatalf("second session read = %q", second)
	}

	stopEvaluatorHTTP(t, "127.0.0.1", port, done)
	if !strings.Contains(output.String(), "stopped") {
		t.Fatalf("program did not continue after shutdown: %q", output.String())
	}
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("port was not released: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	secondBase, secondDone, _ := startLockedHTTP(t, `bring HTTP
from HTTP bring (Server, Request, Response)

home: Function := (request: Request) -> Response {
    return HTTP.text("second")
}

server: Server := HTTP.server("127.0.0.1", `+strconv.Itoa(port)+`)
server.get("/", home)
server.start()
`, "/")
	if _, body, _ := evaluatorHTTPRead(t, client, secondBase+"/"); body != "second" {
		t.Fatalf("restarted server = %q", body)
	}
	stopEvaluatorHTTP(t, "127.0.0.1", port, secondDone)
}

func TestEvaluatorYieldReleasesTheInterpreterLock(t *testing.T) {
	session := New(bufio.NewReader(strings.NewReader("")), io.Discard, t.TempDir())
	session.enter()
	depth := session.yield()
	if depth == 0 {
		t.Fatal("yield released a lock this goroutine did not hold")
	}
	entered := make(chan struct{})
	go func() {
		session.enter()
		close(entered)
		session.leave()
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("a callback could not enter while the owner had yielded")
	}
	session.resume(depth)
	session.leave()
	if session.owner.Load() != 0 {
		t.Fatal("the interpreter lock was left owned")
	}
}

func TestEvaluatorCrossServerMutationDoesNotDeadlock(t *testing.T) {
	outerPort := freeEvaluatorPort(t)
	innerPort := freeEvaluatorPort(t)
	source := `bring HTTP
from HTTP bring (Server, Request, Response)

page: Function := (request: Request) -> Response {
    return HTTP.text("inner")
}

outer: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(outerPort) + `)

mutate: Function := (request: Request) -> Response {
    outer: Global Server
    added: Function := (request: Request) -> Response {
        return HTTP.text("added")
    }
    outer.get("/added", added)
    return HTTP.text("mutated")
}

nest: Function := (request: Request) -> Response {
    inner: Local Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(innerPort) + `)
    inner.get("/", page)
    inner.get("/mutate", mutate)
    inner.start()
    return HTTP.text("outer-returned")
}

ping: Function := (request: Request) -> Response {
    return HTTP.text("pong")
}

outer.get("/nest", nest)
outer.get("/ping", ping)
outer.start()
write("outer-stopped")
`
	compilation := compileAhd(t, source)
	output := &strings.Builder{}
	session := New(bufio.NewReader(strings.NewReader("")), output, t.TempDir())
	done := make(chan ExecutionResult, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		done <- session.Execute(compilation, 0)
	}()
	t.Cleanup(func() {
		stopEvaluatorProgram("127.0.0.1", innerPort, finished)
		stopEvaluatorProgram("127.0.0.1", outerPort, finished)
	})
	outerBase := "http://127.0.0.1:" + strconv.Itoa(outerPort)
	innerBase := "http://127.0.0.1:" + strconv.Itoa(innerPort)
	if !evaluatorWaitOK(t, outerBase+"/ping") {
		t.Fatal("outer server did not answer")
	}
	go func() {
		client := &http.Client{Timeout: 8 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
		response, err := client.Get(outerBase + "/nest")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}()
	if !evaluatorWaitOK(t, innerBase+"/") {
		t.Fatal("inner server did not answer")
	}
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	response, err := client.Get(innerBase + "/mutate")
	if err != nil {
		t.Fatalf("cross-server route mutation blocked: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("mutation status = %d %q", response.StatusCode, body)
	}
	if _, still, _ := evaluatorHTTPRead(t, client, innerBase+"/"); still != "inner" {
		t.Fatalf("inner server after the rejected mutation = %q", still)
	}
	if !ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", innerPort) {
		t.Fatal("inner shutdown failed")
	}
	if _, ping, _ := evaluatorHTTPRead(t, client, outerBase+"/ping"); ping != "pong" {
		t.Fatalf("outer server did not accept a request after the inner server stopped: %q", ping)
	}
	added, err := client.Get(outerBase + "/added")
	if err != nil {
		t.Fatal(err)
	}
	_ = added.Body.Close()
	if added.StatusCode != http.StatusNotFound {
		t.Fatalf("rejected cross-server route was stored, status = %d", added.StatusCode)
	}
	if !ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", outerPort) {
		t.Fatal("outer shutdown failed")
	}
	select {
	case result := <-done:
		if result.Failure != nil {
			t.Fatalf("program failed: %s: %s", result.Failure.Name, result.Failure.Message)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("evaluator did not finish after both servers stopped")
	}
	if !strings.Contains(output.String(), "outer-stopped") {
		t.Fatalf("program did not continue: %q", output.String())
	}
	if ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", outerPort) || ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", innerPort) {
		t.Fatal("repeated shutdown reported an active server")
	}
}

func evaluatorWaitOK(t *testing.T, url string) bool {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		response, err := evaluatorHTTPGet(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestEvaluatorNestedServerStartCanBeShutDown(t *testing.T) {
	outerPort := freeEvaluatorPort(t)
	innerPort := freeEvaluatorPort(t)
	source := `bring HTTP
from HTTP bring (Server, Request, Response)

page: Function := (request: Request) -> Response {
    return HTTP.text("inner")
}

outer: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(outerPort) + `)

nest: Function := (request: Request) -> Response {
    inner: Local Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(innerPort) + `)
    inner.get("/", page)
    inner.start()
    return HTTP.text("outer-returned")
}

ping: Function := (request: Request) -> Response {
    return HTTP.text("pong")
}

outer.get("/nest", nest)
outer.get("/ping", ping)
outer.start()
write("outer-stopped")
`
	compilation := compileAhd(t, source)
	output := &strings.Builder{}
	session := New(bufio.NewReader(strings.NewReader("")), output, t.TempDir())
	done := make(chan ExecutionResult, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		done <- session.Execute(compilation, 0)
	}()
	t.Cleanup(func() {
		stopEvaluatorProgram("127.0.0.1", innerPort, finished)
		stopEvaluatorProgram("127.0.0.1", outerPort, finished)
	})
	base := "http://127.0.0.1:" + strconv.Itoa(outerPort)
	deadline := time.Now().Add(3 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		response, err := evaluatorHTTPGet(base + "/ping")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				ready = true
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ready {
		t.Fatal("outer server did not answer")
	}
	go func() {
		client := &http.Client{Timeout: 8 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
		response, err := client.Get(base + "/nest")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}()
	innerBase := "http://127.0.0.1:" + strconv.Itoa(innerPort)
	deadline = time.Now().Add(3 * time.Second)
	innerReady := false
	for time.Now().Before(deadline) {
		response, err := evaluatorHTTPGet(innerBase + "/")
		if err == nil {
			body, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if response.StatusCode == 200 && string(body) == "inner" {
				innerReady = true
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !innerReady {
		t.Fatal("inner server did not answer")
	}
	started := time.Now()
	if !ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", outerPort) {
		t.Fatal("outer shutdown failed while its handler was blocked in the inner server")
	}
	if elapsed := time.Since(started); elapsed > 4*time.Second {
		t.Fatalf("outer shutdown took %s", elapsed)
	}
	response, err := evaluatorHTTPGet(innerBase + "/")
	if err != nil {
		t.Fatalf("inner server stopped with the outer server: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 || string(body) != "inner" {
		t.Fatalf("inner after outer shutdown = %d %q", response.StatusCode, body)
	}
	if !ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", innerPort) {
		t.Fatal("inner shutdown failed")
	}
	select {
	case result := <-done:
		if result.Failure != nil {
			t.Fatalf("program failed: %s: %s", result.Failure.Name, result.Failure.Message)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("outer Server.start did not return")
	}
	if !strings.Contains(output.String(), "outer-stopped") {
		t.Fatalf("program did not continue: %q", output.String())
	}
	if ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", outerPort) || ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", innerPort) {
		t.Fatal("repeated shutdown reported an active server")
	}
}

func evaluatorHTTPProgram(port int) string {
	return `bring HTTP
from HTTP bring (Server, Request, Response, SessionStore, Session)

hits: Int := 0
sessions: SessionStore := HTTP.sessions()

note: Function := (request: Request) -> Response {
    q: Local String? := request.query("q")
    if q != null {
        return HTTP.text(q)
    }
    return HTTP.text("missing")
}

hit: Function := (request: Request) -> Response {
    hits: Global Int
    hits = hits + 1
    return HTTP.text(str(hits))
}

total: Function := (request: Request) -> Response {
    hits: Global Int
    return HTTP.text(str(hits))
}

count: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    raw: Local String? := session.get("n")
    value: Local Int := 0
    if raw != null {
        value = int(raw)
    }
    value = value + 1
    session.set("n", str(value))
    return sessions.commit(session, HTTP.text(str(value)))
}

server: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
server.get("/note", note)
server.get("/hit", hit)
server.get("/hits", total)
server.get("/count", count)
server.start()
write("stopped")
`
}

func startLockedHTTP(t *testing.T, source, readyPath string) (string, <-chan ExecutionResult, *strings.Builder) {
	t.Helper()
	compilation := compileAhd(t, source)
	output := &strings.Builder{}
	session := New(bufio.NewReader(strings.NewReader("")), output, t.TempDir())
	done := make(chan ExecutionResult, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		done <- session.Execute(compilation, 0)
	}()
	t.Cleanup(func() {
		stopEvaluatorProgram("127.0.0.1", portFromSource(source), finished)
	})
	base := "http://127.0.0.1:" + strconv.Itoa(portFromSource(source))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-finished:
			early := <-done
			t.Fatalf("evaluator HTTP server exited before answering: failure=%v", early.Failure)
		default:
		}
		response, err := evaluatorHTTPGet(base + readyPath)
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				return base, done, output
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("evaluator HTTP server did not answer before the deadline")
	return "", nil, nil
}

func stopEvaluatorHTTP(t *testing.T, host string, port int, done <-chan ExecutionResult) {
	t.Helper()
	if !ahdruntime.AhdHTTPShutdownAddress(host, port) {
		t.Fatalf("server %s:%d did not shut down", host, port)
	}
	select {
	case result := <-done:
		if result.Failure != nil {
			t.Fatalf("program failed after shutdown: %s: %s", result.Failure.Name, result.Failure.Message)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Server.start did not return after shutdown")
	}
}

func evaluatorHTTPRead(t *testing.T, client *http.Client, url string) (int, string, string) {
	t.Helper()
	response, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(body), response.Header.Get("Content-Type")
}

func portFromSource(source string) int {
	marker := `HTTP.server("127.0.0.1", `
	start := strings.Index(source, marker)
	if start < 0 {
		return 0
	}
	start += len(marker)
	end := strings.Index(source[start:], ")")
	if end < 0 {
		return 0
	}
	port, _ := strconv.Atoi(source[start : start+end])
	return port
}
