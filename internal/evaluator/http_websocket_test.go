package evaluator

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// The evaluator serves a WebSocket endpoint through the same runtime a native
// program uses: callbacks are AhdCode Functions run by the session.
func TestHTTPEvaluatorServesAWebSocketEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	source := `bring HTTP
from HTTP bring (Server, Request, Response, WebSocket, WebSocketEndpoint)

closed: String := ""

echo: Function := (socket: WebSocket, text: String) -> Nothing {
    socket.send("echo " + text)
}

opened: Function := (socket: WebSocket, request: Request) -> Nothing {
    socket.send("hello " + request.path())
}

recordClose: Function := (socket: WebSocket, code: Int, reason: String) -> Nothing {
    closed: Global String
    closed = closed + "{code} {reason};"
}

closedLog: Function := (request: Request) -> Response {
    closed: Global String
    return HTTP.text(closed)
}

endpoint: WebSocketEndpoint := HTTP.websocket(echo)
endpoint = endpoint.withOpen(opened)
endpoint = endpoint.withClose(recordClose)
app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.websocket("/live", endpoint)
app.get("/closed", closedLog)
app.start()
`
	compilation := compileAhd(t, source)
	finished, outcome := launchEvaluatorProgram(compilation)
	t.Cleanup(func() {
		stopEvaluatorProgram("127.0.0.1", port, finished)
	})
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	closedLog := func() (string, bool) {
		response, err := http.Get(base + "/closed")
		if err != nil {
			return "", false
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		return string(body), true
	}
	deadline := time.Now().Add(3 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		select {
		case <-finished:
			t.Fatalf("WebSocket server exited early: %s", formatEvaluatorOutcome(*outcome))
		default:
		}
		if _, ok := closedLog(); ok {
			ready = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ready {
		t.Fatal("WebSocket server did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws://127.0.0.1:"+strconv.Itoa(port)+"/live", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()
	if _, data, err := conn.Read(ctx); err != nil || string(data) != "hello /live" {
		t.Fatalf("onOpen message = %q %v", data, err)
	}
	if err := conn.Write(ctx, websocket.MessageText, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	if _, data, err := conn.Read(ctx); err != nil || string(data) != "echo hi" {
		t.Fatalf("echo = %q %v", data, err)
	}
	if err := conn.Close(websocket.StatusNormalClosure, "bye"); err != nil {
		t.Fatalf("close: %v", err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if log, ok := closedLog(); ok && strings.Contains(log, "1000 bye;") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	log, _ := closedLog()
	t.Fatalf("onClose log = %q", log)
}

func TestHTTPEvaluatorWebSocketSameServerMutationDoesNotDeadlock(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	source := `bring HTTP
from HTTP bring (Server, Request, Response, WebSocket, WebSocketEndpoint)

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)

mutate: Function := (socket: WebSocket, text: String) -> Nothing {
    app: Global Server
    extra: Function := (request: Request) -> Response {
        return HTTP.text("no")
    }
    app.get("/extra", extra)
}

echo: Function := (socket: WebSocket, text: String) -> Nothing {
    socket.send("echo " + text)
}

ping: Function := (request: Request) -> Response {
    return HTTP.text("pong")
}

bad: WebSocketEndpoint := HTTP.websocket(mutate)
good: WebSocketEndpoint := HTTP.websocket(echo)
app.websocket("/mut", bad)
app.websocket("/echo", good)
app.get("/ping", ping)
app.start()
`
	compilation := compileAhd(t, source)
	finished, outcome := launchEvaluatorProgram(compilation)
	t.Cleanup(func() {
		stopEvaluatorProgram("127.0.0.1", port, finished)
	})
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(3 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		select {
		case <-finished:
			t.Fatalf("server exited early: %s", formatEvaluatorOutcome(*outcome))
		default:
		}
		response, err := http.Get(base + "/ping")
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
		t.Fatal("server did not answer")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws://127.0.0.1:"+strconv.Itoa(port)+"/mut", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := conn.Write(ctx, websocket.MessageText, []byte("change")); err != nil {
		t.Fatal(err)
	}
	_, _, err = conn.Read(ctx)
	if err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("same-server route change from onMessage did not reject promptly: %v", err)
	}
	_ = conn.CloseNow()
	echoCtx, echoCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer echoCancel()

	response, err := http.Get(base + "/ping")
	if err != nil {
		t.Fatalf("HTTP after the WebSocket error: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 || string(body) != "pong" {
		t.Fatalf("ping = %d %q", response.StatusCode, body)
	}
	extra, err := http.Get(base + "/extra")
	if err != nil {
		t.Fatal(err)
	}
	_ = extra.Body.Close()
	if extra.StatusCode != http.StatusNotFound {
		t.Fatalf("rejected route was stored, status = %d", extra.StatusCode)
	}
	echo, _, err := websocket.Dial(echoCtx, "ws://127.0.0.1:"+strconv.Itoa(port)+"/echo", nil)
	if err != nil {
		t.Fatalf("echo dial after the rejected mutation: %v", err)
	}
	defer echo.CloseNow()
	if err := echo.Write(echoCtx, websocket.MessageText, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	_, data, err := echo.Read(echoCtx)
	if err != nil || string(data) != "echo hi" {
		t.Fatalf("echo = %q %v", data, err)
	}
}
