package evaluator

import (
	"context"
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
	session := newLatexTestSession()
	go func() {
		defer func() { _ = recover() }()
		_ = session.Execute(compilation, 0)
	}()
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
	for time.Now().Before(deadline) {
		if _, ok := closedLog(); ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
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
