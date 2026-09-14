package build

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func webSocketNativeSource(port int) string {
	return `bring HTTP
bring KeyValue
from HTTP bring (Server, Request, Response, WebSocket, WebSocketEndpoint)

clients: Pair<String, WebSocket> := {}
closed: String := ""

liveOpen: Function := (socket: WebSocket, request: Request) -> Nothing {
    clients: Global Pair<String, WebSocket>
    clients[socket.id()] = socket
    room: Local String? := request.query("room")
    if room != null {
        socket.send("welcome to " + room)
    }
}

liveMessage: Function := (socket: WebSocket, text: String) -> Nothing {
    if text == "close" {
        socket.close(4000, "requested")
    } else {
        socket.send("echo " + text)
    }
}

liveClose: Function := (socket: WebSocket, code: Int, reason: String) -> Nothing {
    clients: Global Pair<String, WebSocket>
    closed: Global String
    clients = KeyValue.without(clients, socket.id())
    closed = closed + "{code} {reason};"
}

broadcast: Function := (request: Request) -> Response {
    clients: Global Pair<String, WebSocket>
    count: Local Int := 0
    for socket in KeyValue.values(clients) {
        if socket.send("news") {
            count += 1
        }
    }
    return HTTP.text("sent {count}")
}

closedLog: Function := (request: Request) -> Response {
    closed: Global String
    return HTTP.text(closed)
}

endpoint: WebSocketEndpoint := HTTP.websocket(liveMessage)
endpoint = endpoint.withOpen(liveOpen)
endpoint = endpoint.withClose(liveClose)

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.websocket("/live", endpoint)
app.post("/broadcast", broadcast)
app.get("/closed", closedLog)
app.start()
`
}

func dialNativeWebSocket(t *testing.T, url string, header http.Header) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

func nativeWebSocketRead(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(data)
}

func nativeWebSocketWrite(t *testing.T, conn *websocket.Conn, text string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, []byte(text)); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func nativeResponseBody(t *testing.T, response *http.Response, err error) string {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return string(body)
}

func nativeGet(t *testing.T, url string) string {
	t.Helper()
	response, err := http.Get(url)
	return nativeResponseBody(t, response, err)
}

func nativePost(t *testing.T, url string) string {
	t.Helper()
	response, err := http.Post(url, "text/plain", nil)
	return nativeResponseBody(t, response, err)
}

// A native program serves a WebSocket endpoint end to end: the upgrade
// Request, echo, a broadcast from an ordinary HTTP handler, server- and
// client-initiated closes seen by onClose, and the handshake refusals. The
// program builds from the vendored tree with an empty module cache.
func TestWebSocketNativeProgramServesAnEndpoint(t *testing.T) {
	port := freeLoopbackPort(t)
	directory := writeSources(t, map[string]string{"main.ahd": webSocketNativeSource(port)})
	entry := filepath.Join(directory, "main.ahd")
	compiled := Compile(entry)
	if compiled.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(compiled.Diagnostics))
	}
	if !compiled.Program.RequiresWebSocket || compiled.Program.RequiresPostgreSQL || compiled.Program.RequiresMySQL {
		t.Fatalf("program flags: websocket=%v postgresql=%v mysql=%v", compiled.Program.RequiresWebSocket, compiled.Program.RequiresPostgreSQL, compiled.Program.RequiresMySQL)
	}
	t.Setenv("GOMODCACHE", t.TempDir())
	executable := filepath.Join(t.TempDir(), "live")
	if _, result := BuildProgram(entry, executable); result.HasErrors() {
		t.Fatalf("build failed:\n%s", diagnosticText(result.Diagnostics))
	}
	base := startBuiltHTTP(t, executable, directory, port)
	socketBase := "ws" + strings.TrimPrefix(base, "http")

	first := dialNativeWebSocket(t, socketBase+"/live?room=lab", nil)
	if got := nativeWebSocketRead(t, first); got != "welcome to lab" {
		t.Fatalf("welcome = %q", got)
	}
	nativeWebSocketWrite(t, first, "merhaba")
	if got := nativeWebSocketRead(t, first); got != "echo merhaba" {
		t.Fatalf("echo = %q", got)
	}
	second := dialNativeWebSocket(t, socketBase+"/live", http.Header{"Origin": []string{base}})
	nativeWebSocketWrite(t, second, "ready")
	if got := nativeWebSocketRead(t, second); got != "echo ready" {
		t.Fatalf("second echo = %q", got)
	}

	if body := nativePost(t, base+"/broadcast"); body != "sent 2" {
		t.Fatalf("broadcast = %q", body)
	}
	if firstNews, secondNews := nativeWebSocketRead(t, first), nativeWebSocketRead(t, second); firstNews != "news" || secondNews != "news" {
		t.Fatalf("broadcast delivered %q and %q", firstNews, secondNews)
	}

	nativeWebSocketWrite(t, first, "close")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _, err := first.Read(ctx)
	var closeError websocket.CloseError
	if !errors.As(err, &closeError) || closeError.Code != 4000 || closeError.Reason != "requested" {
		t.Fatalf("server close = %v", err)
	}
	if err := second.Close(websocket.StatusNormalClosure, "bye"); err != nil {
		t.Fatalf("client close: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	closed := ""
	for time.Now().Before(deadline) {
		closed = nativeGet(t, base+"/closed")
		if strings.Contains(closed, "4000 requested;") && strings.Contains(closed, "1000 bye;") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(closed, "4000 requested;") || !strings.Contains(closed, "1000 bye;") {
		t.Fatalf("onClose log = %q", closed)
	}
	if body := nativePost(t, base+"/broadcast"); body != "sent 0" {
		t.Fatalf("broadcast after both closed = %q", body)
	}

	response, err := http.Get(base + "/live")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusUpgradeRequired {
		t.Fatalf("plain GET = %d", response.StatusCode)
	}
	dialContext, dialCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dialCancel()
	_, refused, err := websocket.Dial(dialContext, socketBase+"/live", &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"https://evil.example"}},
	})
	if err == nil || refused == nil || refused.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin dial = %v %v", refused, err)
	}
}

// A Web application registers an endpoint through Web.websocket and
// App.websocket under the ordinary configuration contract and Web limits.
func TestWebSocketNativeWebApplication(t *testing.T) {
	port := freeLoopbackPort(t)
	directory := writeSources(t, map[string]string{"main.ahd": `bring Web
from Web bring (App, Request, Response, WebSocket)

echo: Function := (socket: WebSocket, text: String) -> Nothing {
    socket.send("web " + text)
}

home: Function := (request: Request) -> Response {
    return Web.text("home")
}

site: App := Web.app(Web.configure())
site.get("/", home)
site.websocket("/live", Web.websocket(echo))
site.start()
`})
	for key, value := range map[string]string{
		"APP_NAME": "Live", "APP_ENV": "test", "APP_HOST": "live.example", "APP_PROTOCOL": "http",
		"SERVER_HOST": "127.0.0.1", "SERVER_PORT": strconv.Itoa(port),
	} {
		t.Setenv(key, value)
	}
	executable := filepath.Join(t.TempDir(), "web-live")
	if _, result := BuildProgram(filepath.Join(directory, "main.ahd"), executable); result.HasErrors() {
		t.Fatalf("build failed:\n%s", diagnosticText(result.Diagnostics))
	}
	base := startBuiltHTTP(t, executable, directory, port)
	if body := nativeGet(t, base+"/"); body != "home" {
		t.Fatalf("home = %q", body)
	}
	conn := dialNativeWebSocket(t, "ws"+strings.TrimPrefix(base, "http")+"/live", http.Header{"Origin": []string{base}})
	nativeWebSocketWrite(t, conn, "hi")
	if got := nativeWebSocketRead(t, conn); got != "web hi" {
		t.Fatalf("Web echo = %q", got)
	}
}
