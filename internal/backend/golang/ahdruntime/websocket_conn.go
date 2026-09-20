package ahdruntime

// The WebSocket connection (v1.4.0). This file imports the vendored
// github.com/coder/websocket (see ahdruntime/websocketvendor), so, like the
// MySQL and codes runtimes, it joins a generated program only when the program
// creates an endpoint with HTTP.websocket. websocket.go holds everything that
// must always compile.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/coder/websocket"
)

func init() {
	ahdHTTPWebSocketServe = ahdWebSocketServe
}

// ahdWebSocketServe answers one GET on a WebSocket path: the opening handshake
// checks in their fixed order, then the connection's whole lifetime.
func ahdWebSocketServe(dispatcher ahdHTTPDispatcher, route *ahdHTTPWebSocketRoute, writer http.ResponseWriter, request *http.Request) {
	endpoint := route.endpoint
	if status := ahdWebSocketHandshakeProblem(request); status != 0 {
		ahdWebSocketRefuseHandshake(writer, status)
		return
	}
	if !route.reserve() {
		ahdHTTPWritePlain(writer, http.StatusServiceUnavailable, "Service Unavailable")
		return
	}
	defer route.release()
	if !ahdWebSocketOriginAllowed(endpoint, request) {
		ahdHTTPWritePlain(writer, http.StatusForbidden, "Forbidden")
		return
	}
	snapshot, uploads, err := ahdHTTPMaterialize(request, nil, 0, 0)
	defer ahdHTTPReleaseUploads(uploads)
	if err != nil {
		ahdHTTPWritePlain(writer, http.StatusBadRequest, "Bad Request")
		return
	}
	requestData := ahdHTTPEncodeRequest(snapshot)
	if endpoint.accept != nil {
		var refusal *string
		if ahdWebSocketCall(dispatcher.state, func() { refusal = endpoint.accept(requestData) }) {
			ahdHTTPWritePlain(writer, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		if refusal != nil {
			ahdHTTPWriteEncoded(writer, request, *refusal)
			return
		}
	}
	// The HTTP server set read and write deadlines for an ordinary request.
	// They would stay on the connection after the upgrade hijacks it and end
	// every socket after a few seconds, so they are cleared first.
	controller := http.NewResponseController(writer)
	_ = controller.SetReadDeadline(time.Time{})
	_ = controller.SetWriteDeadline(time.Time{})

	// onOpen runs before the 101 response. By the time the client sees the
	// connection open, the application has already registered the socket, so a
	// message another handler sends right afterwards cannot miss it. Messages
	// sent, or a close requested, before the upgrade completes wait in the
	// socket's queue.
	var connection atomic.Pointer[websocket.Conn]
	socket, created := ahdWebSocketNewSocket(endpoint.maxQueued, func() {
		if conn := connection.Load(); conn != nil {
			_ = conn.CloseNow()
		}
	})
	if !created {
		ahdHTTPWritePlain(writer, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer ahdWebSocketForget(socket.id)
	if endpoint.onOpen != nil && ahdWebSocketCall(dispatcher.state, func() { endpoint.onOpen(socket.id, requestData) }) {
		socket.requestClose(int64(websocket.StatusInternalError), "internal error", false)
	}

	conn, err := websocket.Accept(writer, request, &websocket.AcceptOptions{
		// The origin was already checked above, with AhdCode's own rules.
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionDisabled,
	})
	if err != nil {
		// onOpen has run, so onClose still runs exactly once.
		socket.markClosed()
		ahdWebSocketFinish(dispatcher.state, endpoint, socket, int64(websocket.StatusAbnormalClosure), "")
		return
	}
	connection.Store(conn)
	ahdWebSocketRun(dispatcher.state, endpoint, conn, socket)
}

// ahdWebSocketRun owns one upgraded connection until it ends. onOpen has
// already run; onMessage runs once per message in arrival order and onClose
// exactly once, last. Every callback holds the server mutex.
func ahdWebSocketRun(state *ahdHTTPServerState, endpoint *ahdWebSocketEndpointState, conn *websocket.Conn, socket *ahdWebSocketSocket) {
	// Allowing one byte past the limit lets AhdCode see an oversize message and
	// close with its own 1009 instead of the library's.
	conn.SetReadLimit(endpoint.maxMessageBytes + 1)

	ctx, cancel := context.WithCancel(context.Background())
	readerDone := make(chan struct{})
	writerDone := make(chan struct{})
	go ahdWebSocketWriter(conn, socket, readerDone, writerDone)
	go ahdWebSocketPinger(ctx, conn)

	code, reason := ahdWebSocketReadLoop(state, endpoint, conn, socket)
	socket.markClosed()
	close(readerDone)
	<-writerDone
	cancel()
	_ = conn.CloseNow()
	ahdWebSocketFinish(state, endpoint, socket, code, reason)
}

// ahdWebSocketFinish runs onClose exactly once. A close the server started
// reports its own code and reason; otherwise the peer's, or 1006 when the
// connection was lost.
func ahdWebSocketFinish(state *ahdHTTPServerState, endpoint *ahdWebSocketEndpointState, socket *ahdWebSocketSocket, code int64, reason string) {
	socket.mutex.Lock()
	if socket.initiated {
		code, reason = socket.code, socket.reason
	}
	socket.mutex.Unlock()
	if endpoint.onClose != nil {
		ahdWebSocketCall(state, func() { endpoint.onClose(socket.id, code, reason) })
	}
}

// ahdWebSocketReadLoop delivers messages until the connection ends and returns
// the peer's close code and reason, or 1006 when the connection was lost. The
// next frame is read only after the previous callback returned, so a slow
// application slows its peer instead of growing a queue.
func ahdWebSocketReadLoop(state *ahdHTTPServerState, endpoint *ahdWebSocketEndpointState, conn *websocket.Conn, socket *ahdWebSocketSocket) (int64, string) {
	for {
		messageType, reader, err := conn.Reader(context.Background())
		if err != nil {
			return ahdWebSocketPeerClose(err)
		}
		payload, err := io.ReadAll(io.LimitReader(reader, endpoint.maxMessageBytes+1))
		if err != nil {
			return ahdWebSocketPeerClose(err)
		}
		if int64(len(payload)) > endpoint.maxMessageBytes {
			// The rest of the message stays unread; the close handshake discards it.
			socket.requestClose(int64(websocket.StatusMessageTooBig), "message too large", false)
			return int64(websocket.StatusAbnormalClosure), ""
		}
		if socket.isClosing() {
			continue
		}
		if messageType != websocket.MessageText {
			socket.requestClose(int64(websocket.StatusUnsupportedData), "binary messages are not supported", false)
			continue
		}
		if !utf8.Valid(payload) {
			socket.requestClose(int64(websocket.StatusInvalidFramePayloadData), "text message is not valid UTF-8", false)
			continue
		}
		text := string(payload)
		if ahdWebSocketCall(state, func() { endpoint.onMessage(socket.id, text) }) {
			socket.requestClose(int64(websocket.StatusInternalError), "internal error", false)
		}
	}
}

// ahdWebSocketWriter drains one socket's queue and performs a server-initiated
// close. It is the only goroutine that writes data frames.
func ahdWebSocketWriter(conn *websocket.Conn, socket *ahdWebSocketSocket, readerDone <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	for {
		select {
		case text := <-socket.outbound:
			if !ahdWebSocketWrite(conn, text) {
				_ = conn.CloseNow()
				return
			}
		case request := <-socket.closeRequests:
			ahdWebSocketFinishClose(conn, socket, request)
			return
		case <-readerDone:
			select {
			case request := <-socket.closeRequests:
				ahdWebSocketFinishClose(conn, socket, request)
			default:
			}
			return
		}
	}
}

func ahdWebSocketFinishClose(conn *websocket.Conn, socket *ahdWebSocketSocket, request ahdWebSocketCloseRequest) {
	for request.flush {
		select {
		case text := <-socket.outbound:
			if !ahdWebSocketWrite(conn, text) {
				_ = conn.CloseNow()
				return
			}
			continue
		default:
		}
		break
	}
	_ = conn.Close(websocket.StatusCode(request.code), request.reason)
}

func ahdWebSocketWrite(conn *websocket.Conn, text string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), ahdWebSocketWriteTimeout)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, []byte(text)) == nil
}

// ahdWebSocketPinger sends a ping on an interval; a peer that does not answer
// in time is treated as lost.
func ahdWebSocketPinger(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(ahdWebSocketPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingContext, cancel := context.WithTimeout(ctx, ahdWebSocketPongTimeout)
			err := conn.Ping(pingContext)
			cancel()
			if err != nil {
				if ctx.Err() == nil {
					_ = conn.CloseNow()
				}
				return
			}
		}
	}
}

func ahdWebSocketPeerClose(err error) (int64, string) {
	var closeError websocket.CloseError
	if errors.As(err, &closeError) {
		return int64(closeError.Code), closeError.Reason
	}
	return int64(websocket.StatusAbnormalClosure), ""
}

// ahdWebSocketCall runs one callback holding the server's handler mutex,
// exactly like an HTTP handler, and reports whether it raised. The failure is
// written to stderr, never to the peer.
func ahdWebSocketCall(state *ahdHTTPServerState, callback func()) (failed bool) {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	defer func() {
		if recovered := recover(); recovered != nil {
			failed = true
			ahdWebSocketLogFailure(recovered)
		}
	}()
	callback()
	return false
}

func ahdWebSocketLogFailure(recovered any) {
	if signal, ok := recovered.(*AhdSignal); ok && signal != nil {
		name := "Error"
		if signal.Instance != nil && signal.Instance.AhdClassOf() != nil {
			name = signal.Instance.AhdClassOf().Name
		}
		fmt.Fprintf(os.Stderr, "ahdcode: WebSocket handler %s: %s\n", name, signal.Message)
		return
	}
	fmt.Fprintf(os.Stderr, "ahdcode: WebSocket handler panic: %v\n", recovered)
}

// --- the client connection (v2.1.0) ---
//
// The client half of websocket_client.go: the opening handshake, and the one
// live connection behind a WebSocketConnection. Everything that names the
// vendored library lives here, so a program that uses no WebSocket at all
// still compiles without it.

func init() {
	ahdWebSocketClientDialer = ahdWebSocketClientDial
}

// ahdWebSocketClientConn implements ahdWebSocketClientLink over one vendored
// connection. Only the reader goroutine calls read; write, shutdown, and drop
// may be called from the program's goroutine at the same time, which the
// library allows.
type ahdWebSocketClientConn struct {
	conn  *websocket.Conn
	limit int64
}

// ahdWebSocketClientTransport is the transport every handshake goes through.
// It is a variable only so the runtime's own tests can point a wss:// fixture
// at their certificate authority, the same way the ping interval is a
// variable so they can shorten it. Nothing in AhdCode can reach it: its
// default is the standard transport, which verifies the certificate and the
// host name against the system roots, and there is no insecure mode.
var ahdWebSocketClientTransport http.RoundTripper = http.DefaultTransport

// ahdWebSocketClientDial performs the opening handshake.
func ahdWebSocketClientDial(config ahdWebSocketClientConfig) (ahdWebSocketClientLink, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.timeoutSeconds)*time.Second)
	defer cancel()
	header := make(http.Header)
	for _, pair := range config.headers {
		header.Set(pair.Name, pair.Value)
	}
	conn, response, err := websocket.Dial(ctx, config.url, &websocket.DialOptions{
		HTTPClient:      &http.Client{Transport: ahdWebSocketClientTransport},
		HTTPHeader:      header,
		CompressionMode: websocket.CompressionDisabled,
	})
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	// One byte past the limit is allowed through so an oversize message is
	// recognized by AhdCode and answered with its own 1009, exactly as the
	// server half does.
	conn.SetReadLimit(config.maxMessageBytes + 1)
	return &ahdWebSocketClientConn{conn: conn, limit: config.maxMessageBytes}, nil
}

// read blocks for the next message. A control frame, a ping, and a pong are
// handled inside the library; only a data message or the end of the
// connection comes back here.
func (link *ahdWebSocketClientConn) read() (ahdWebSocketClientFrame, bool) {
	messageType, reader, err := link.conn.Reader(context.Background())
	if err != nil {
		code, reason := ahdWebSocketPeerClose(err)
		return ahdWebSocketClientFrame{closed: true, code: code, reason: reason}, true
	}
	payload, err := io.ReadAll(io.LimitReader(reader, link.limit+1))
	if err != nil {
		code, reason := ahdWebSocketPeerClose(err)
		return ahdWebSocketClientFrame{closed: true, code: code, reason: reason}, true
	}
	if int64(len(payload)) > link.limit {
		return ahdWebSocketClientFrame{oversize: true}, false
	}
	if messageType != websocket.MessageText {
		return ahdWebSocketClientFrame{binary: true}, false
	}
	if !utf8.Valid(payload) {
		return ahdWebSocketClientFrame{invalidUTF8: true}, false
	}
	return ahdWebSocketClientFrame{text: string(payload)}, false
}

func (link *ahdWebSocketClientConn) write(text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return link.conn.Write(ctx, websocket.MessageText, []byte(text))
}

func (link *ahdWebSocketClientConn) shutdown(code int64, reason string) {
	_ = link.conn.Close(websocket.StatusCode(code), reason)
}

func (link *ahdWebSocketClientConn) drop() {
	_ = link.conn.CloseNow()
}
