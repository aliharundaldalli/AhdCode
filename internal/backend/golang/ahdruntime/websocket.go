package ahdruntime

// WebSocket server support (v1.4.0).
//
// A WebSocket endpoint is registered on an HTTP Server like a route and shares
// that Server's one handler mutex: onOpen, onMessage, onClose, and the accept
// check run exactly like HTTP handlers, never two at the same time on the same
// Server, so AhdCode code never runs concurrently. A WebSocket value is a
// handle to one accepted connection; send and close never block, because they
// may run while that mutex is held.
//
// This file is standard library only and joins every generated program, so the
// HTTP dispatcher, Server.websocket, and the WebSocket and WebSocketEndpoint
// members always compile. The RFC 6455 connection itself -- framing, ping and
// pong, the close handshake -- is the vendored github.com/coder/websocket and
// lives in websocket_conn.go, which a generated program receives only when it
// creates an endpoint with HTTP.websocket.

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// The four callback shapes, each already adapted from an AhdCode Function by
// the backend or the evaluator. A socket and a request travel as their hidden
// handle and data Strings.
type (
	AhdWebSocketMessageHandler func(socket string, text string)
	AhdWebSocketOpenHandler    func(socket string, request string)
	AhdWebSocketCloseHandler   func(socket string, code int64, reason string)
	AhdWebSocketAcceptHandler  func(request string) *string
)

const (
	ahdWebSocketDefaultMaxMessageBytes = 65536
	ahdWebSocketMaximumMessageBytes    = 16777216
	ahdWebSocketDefaultMaxQueued       = 64
	ahdWebSocketMaximumQueued          = 4096
	ahdWebSocketDefaultMaxConnections  = 1024
	ahdWebSocketMaximumConnections     = 1000000
	ahdWebSocketMaximumReasonBytes     = 123
)

// Connection timing is internal, like the HTTP server's timeouts, and not a
// public API. These are variables only so the runtime's tests can shorten them.
var (
	ahdWebSocketPingInterval = 30 * time.Second
	ahdWebSocketPongTimeout  = 30 * time.Second
	ahdWebSocketWriteTimeout = 10 * time.Second
)

// ahdWebSocketEndpointState is one immutable WebSocketEndpoint configuration.
// Every with* member stores a changed copy under a new handle, the same way a
// Cookie builder returns a new Cookie.
type ahdWebSocketEndpointState struct {
	onMessage       AhdWebSocketMessageHandler
	onOpen          AhdWebSocketOpenHandler
	onClose         AhdWebSocketCloseHandler
	accept          AhdWebSocketAcceptHandler
	origins         []string // nil is the same-origin default
	maxMessageBytes int64
	maxQueued       int64
	maxConnections  int64
}

var (
	ahdWebSocketEndpoints    = map[string]*ahdWebSocketEndpointState{}
	ahdWebSocketEndpointsMu  sync.Mutex
	ahdWebSocketNextEndpoint atomic.Int64
)

// AhdHTTPWebSocket is HTTP.websocket(onMessage).
func AhdHTTPWebSocket(class *AhdClass, onMessage AhdWebSocketMessageHandler) string {
	if onMessage == nil {
		AhdRaiseClass(class, "HTTP.websocket onMessage handler is missing")
	}
	return ahdWebSocketStoreEndpoint(&ahdWebSocketEndpointState{
		onMessage:       onMessage,
		maxMessageBytes: ahdWebSocketDefaultMaxMessageBytes,
		maxQueued:       ahdWebSocketDefaultMaxQueued,
		maxConnections:  ahdWebSocketDefaultMaxConnections,
	})
}

func ahdWebSocketStoreEndpoint(state *ahdWebSocketEndpointState) string {
	id := strconv.FormatInt(ahdWebSocketNextEndpoint.Add(1), 10)
	ahdWebSocketEndpointsMu.Lock()
	ahdWebSocketEndpoints[id] = state
	ahdWebSocketEndpointsMu.Unlock()
	return id
}

func ahdWebSocketLookupEndpoint(class *AhdClass, handle string) *ahdWebSocketEndpointState {
	ahdWebSocketEndpointsMu.Lock()
	state := ahdWebSocketEndpoints[handle]
	ahdWebSocketEndpointsMu.Unlock()
	if state == nil {
		AhdRaiseClass(class, "WebSocketEndpoint storage is corrupted")
	}
	return state
}

func ahdWebSocketDerive(class *AhdClass, handle string, change func(*ahdWebSocketEndpointState)) string {
	derived := *ahdWebSocketLookupEndpoint(class, handle)
	if derived.origins != nil {
		derived.origins = append([]string(nil), derived.origins...)
	}
	change(&derived)
	return ahdWebSocketStoreEndpoint(&derived)
}

func AhdWebSocketEndpointWithOpen(class *AhdClass, handle string, handler AhdWebSocketOpenHandler) string {
	if handler == nil {
		AhdRaiseClass(class, "WebSocketEndpoint.withOpen handler is missing")
	}
	return ahdWebSocketDerive(class, handle, func(state *ahdWebSocketEndpointState) { state.onOpen = handler })
}

func AhdWebSocketEndpointWithClose(class *AhdClass, handle string, handler AhdWebSocketCloseHandler) string {
	if handler == nil {
		AhdRaiseClass(class, "WebSocketEndpoint.withClose handler is missing")
	}
	return ahdWebSocketDerive(class, handle, func(state *ahdWebSocketEndpointState) { state.onClose = handler })
}

func AhdWebSocketEndpointWithAccept(class *AhdClass, handle string, check AhdWebSocketAcceptHandler) string {
	if check == nil {
		AhdRaiseClass(class, "WebSocketEndpoint.withAccept check is missing")
	}
	return ahdWebSocketDerive(class, handle, func(state *ahdWebSocketEndpointState) { state.accept = check })
}

// AhdWebSocketEndpointWithAllowedOrigins replaces the same-origin default with
// an exact list. There is no wildcard and no allow-all switch.
func AhdWebSocketEndpointWithAllowedOrigins(class *AhdClass, handle string, origins []string) string {
	if len(origins) == 0 {
		AhdRaiseClass(class, "WebSocketEndpoint.withAllowedOrigins needs at least one origin")
	}
	normalized := make([]string, 0, len(origins))
	for _, origin := range origins {
		value, valid := ahdWebSocketNormalizeOrigin(origin)
		if !valid {
			AhdRaiseClass(class, "WebSocketEndpoint origin "+ahdHTMLQuote(origin)+" must be exactly scheme://host or scheme://host:port with http or https")
		}
		if !ahdHTTPContains(normalized, value) {
			normalized = append(normalized, value)
		}
	}
	return ahdWebSocketDerive(class, handle, func(state *ahdWebSocketEndpointState) { state.origins = normalized })
}

func AhdWebSocketEndpointWithMaxMessageBytes(class *AhdClass, handle string, bytes int64) string {
	if bytes < 1 || bytes > ahdWebSocketMaximumMessageBytes {
		AhdRaiseClass(class, "WebSocketEndpoint maxMessageBytes must be in 1..16777216")
	}
	return ahdWebSocketDerive(class, handle, func(state *ahdWebSocketEndpointState) { state.maxMessageBytes = bytes })
}

func AhdWebSocketEndpointWithMaxQueuedMessages(class *AhdClass, handle string, count int64) string {
	if count < 1 || count > ahdWebSocketMaximumQueued {
		AhdRaiseClass(class, "WebSocketEndpoint maxQueuedMessages must be in 1..4096")
	}
	return ahdWebSocketDerive(class, handle, func(state *ahdWebSocketEndpointState) { state.maxQueued = count })
}

func AhdWebSocketEndpointWithMaxConnections(class *AhdClass, handle string, count int64) string {
	if count < 1 || count > ahdWebSocketMaximumConnections {
		AhdRaiseClass(class, "WebSocketEndpoint maxConnections must be in 1..1000000")
	}
	return ahdWebSocketDerive(class, handle, func(state *ahdWebSocketEndpointState) { state.maxConnections = count })
}

// ahdWebSocketNormalizeOrigin accepts exactly scheme://host[:port] with http or
// https, and returns it lowercased without a default port -- the form a
// browser writes into the Origin header.
func ahdWebSocketNormalizeOrigin(origin string) (string, bool) {
	if origin == "" || strings.ContainsAny(origin, " \t\r\n#?") {
		return "", false
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Opaque != "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if (scheme != "http" && scheme != "https") || parsed.Hostname() == "" || strings.HasSuffix(parsed.Host, ":") {
		return "", false
	}
	host := strings.ToLower(parsed.Host)
	if port := parsed.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", false
		}
		if (scheme == "http" && number == 80) || (scheme == "https" && number == 443) {
			host = strings.TrimSuffix(host, ":"+port)
		}
	}
	return scheme + "://" + host, true
}

// ahdWebSocketOriginAllowed applies the endpoint's origin policy. A request
// without an Origin header is a non-browser client and is allowed; browsers
// always send one. The default compares the Origin's host with the request's
// Host header; a proxy that rewrites Host therefore fails closed.
func ahdWebSocketOriginAllowed(endpoint *ahdWebSocketEndpointState, request *http.Request) bool {
	values := request.Header.Values("Origin")
	if len(values) == 0 {
		return true
	}
	if len(values) != 1 {
		return false
	}
	origin, valid := ahdWebSocketNormalizeOrigin(values[0])
	if !valid {
		return false
	}
	if endpoint.origins != nil {
		return ahdHTTPContains(endpoint.origins, origin)
	}
	return origin[strings.Index(origin, "://")+3:] == ahdWebSocketComparableHost(request.Host)
}

// ahdWebSocketComparableHost lowercases a Host header and drops :80 or :443,
// which a browser never writes into an Origin.
func ahdWebSocketComparableHost(host string) string {
	host = strings.ToLower(host)
	for _, port := range []string{":80", ":443"} {
		if strings.HasSuffix(host, port) {
			return strings.TrimSuffix(host, port)
		}
	}
	return host
}

// ahdWebSocketHandshakeProblem checks the RFC 6455 opening handshake fields
// itself, so each refusal has a fixed status: 426 when the request is not an
// upgrade attempt at all, 400 when it is a malformed one, 0 when it is valid.
func ahdWebSocketHandshakeProblem(request *http.Request) int {
	if !ahdWebSocketHeaderToken(request.Header, "Upgrade", "websocket") {
		return http.StatusUpgradeRequired
	}
	if !request.ProtoAtLeast(1, 1) || !ahdWebSocketHeaderToken(request.Header, "Connection", "upgrade") ||
		request.Header.Get("Sec-WebSocket-Version") != "13" {
		return http.StatusBadRequest
	}
	key, err := base64.StdEncoding.DecodeString(request.Header.Get("Sec-WebSocket-Key"))
	if err != nil || len(key) != 16 {
		return http.StatusBadRequest
	}
	return 0
}

func ahdWebSocketHeaderToken(header http.Header, name, token string) bool {
	for _, value := range header.Values(name) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func ahdWebSocketRefuseHandshake(writer http.ResponseWriter, status int) {
	writer.Header().Set("Sec-WebSocket-Version", "13")
	if status == http.StatusUpgradeRequired {
		writer.Header().Set("Upgrade", "websocket")
		writer.Header().Set("Connection", "Upgrade")
	}
	ahdHTTPWritePlain(writer, status, http.StatusText(status))
}

// ahdHTTPWebSocketRoute is one Server.websocket registration. It keeps the
// endpoint configuration it was registered with and counts open connections.
type ahdHTTPWebSocketRoute struct {
	endpoint *ahdWebSocketEndpointState
	active   atomic.Int64
}

// reserve claims one connection slot, or reports that maxConnections is reached.
func (route *ahdHTTPWebSocketRoute) reserve() bool {
	for {
		current := route.active.Load()
		if current >= route.endpoint.maxConnections {
			return false
		}
		if route.active.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func (route *ahdHTTPWebSocketRoute) release() {
	route.active.Add(-1)
}

// AhdHTTPServerWebSocket is Server.websocket(path, endpoint). The path follows
// the HTTP route rules, and a path cannot be both a GET route and a WebSocket
// endpoint.
func AhdHTTPServerWebSocket(class *AhdClass, handle, path, endpointHandle string) {
	endpoint := ahdWebSocketLookupEndpoint(class, endpointHandle)
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#") {
		AhdRaiseClass(class, "HTTP route path "+ahdHTMLQuote(path)+" must begin with / and must not contain ? or #")
	}
	if strings.Contains(path, "*") && ahdHTTPWildcardPrefix(path) == "" {
		AhdRaiseClass(class, "HTTP route path "+ahdHTMLQuote(path)+" may use a trailing /* for one path segment")
	}
	server := ahdHTTPLookup(class, handle)
	server.mutex.Lock()
	defer server.mutex.Unlock()
	if server.started {
		AhdRaiseClass(class, "HTTP routes cannot be changed after start")
	}
	if _, exists := server.routes[ahdHTTPRouteKey{method: http.MethodGet, path: path}]; exists {
		AhdRaiseClass(class, "HTTP route GET "+path+" is already registered")
	}
	if server.webSockets == nil {
		server.webSockets = make(map[string]*ahdHTTPWebSocketRoute)
	}
	if _, exists := server.webSockets[path]; exists {
		AhdRaiseClass(class, "HTTP WebSocket route "+path+" is already registered")
	}
	server.webSockets[path] = &ahdHTTPWebSocketRoute{endpoint: endpoint}
}

// ahdHTTPLookupWebSocket finds the endpoint for path: an exact registration
// first, then a trailing /* one, which an exact GET route overrides. The caller
// holds the server mutex.
func ahdHTTPLookupWebSocket(state *ahdHTTPServerState, path string) *ahdHTTPWebSocketRoute {
	if route := state.webSockets[path]; route != nil {
		return route
	}
	if len(state.webSockets) == 0 {
		return nil
	}
	if _, exact := state.routes[ahdHTTPRouteKey{method: http.MethodGet, path: path}]; exact {
		return nil
	}
	for pattern, route := range state.webSockets {
		if prefix := ahdHTTPWildcardPrefix(pattern); prefix != "" && ahdHTTPWildcardMatch(prefix, path) {
			return route
		}
	}
	return nil
}

// ahdHTTPWebSocketServe is installed by websocket_conn.go. It is nil only in a
// program that never created an endpoint, which therefore has no WebSocket
// route to serve.
var ahdHTTPWebSocketServe func(dispatcher ahdHTTPDispatcher, route *ahdHTTPWebSocketRoute, writer http.ResponseWriter, request *http.Request)

func ahdHTTPServeWebSocket(dispatcher ahdHTTPDispatcher, route *ahdHTTPWebSocketRoute, writer http.ResponseWriter, request *http.Request) {
	serve := ahdHTTPWebSocketServe
	if serve == nil {
		fmt.Fprintln(os.Stderr, "ahdcode: WebSocket support is not part of this program")
		ahdHTTPWritePlain(writer, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	serve(dispatcher, route, writer, request)
}

// ahdWebSocketCloseRequest is one server-initiated close waiting for the
// connection's writer. flush writes messages already queued first.
type ahdWebSocketCloseRequest struct {
	code   int64
	reason string
	flush  bool
}

// ahdWebSocketSocket is the state behind one WebSocket value.
type ahdWebSocketSocket struct {
	id            string
	outbound      chan string
	closeRequests chan ahdWebSocketCloseRequest
	mutex         sync.Mutex
	closing       bool
	initiated     bool
	code          int64
	reason        string
	// forceClose drops the connection without a close handshake. The
	// runtime's tests use it to end connections a test server leaves open.
	forceClose func()
}

var (
	ahdWebSockets   = map[string]*ahdWebSocketSocket{}
	ahdWebSocketsMu sync.Mutex
)

// ahdWebSocketNewSocket registers a socket under a 128-bit random identifier
// that is never reused. Every field is set before the socket becomes visible
// in the registry.
func ahdWebSocketNewSocket(queue int64, forceClose func()) (*ahdWebSocketSocket, bool) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return nil, false
	}
	socket := &ahdWebSocketSocket{
		id:            base64.RawURLEncoding.EncodeToString(raw[:]),
		outbound:      make(chan string, queue),
		closeRequests: make(chan ahdWebSocketCloseRequest, 1),
		forceClose:    forceClose,
	}
	ahdWebSocketsMu.Lock()
	ahdWebSockets[socket.id] = socket
	ahdWebSocketsMu.Unlock()
	return socket, true
}

func ahdWebSocketForget(id string) {
	ahdWebSocketsMu.Lock()
	delete(ahdWebSockets, id)
	ahdWebSocketsMu.Unlock()
}

func ahdWebSocketLookup(id string) *ahdWebSocketSocket {
	ahdWebSocketsMu.Lock()
	defer ahdWebSocketsMu.Unlock()
	return ahdWebSockets[id]
}

// requestClose starts a server-initiated close exactly once and never blocks.
// The request is handed over while the mutex is held, so anyone who then sees
// closing also finds the request waiting.
func (socket *ahdWebSocketSocket) requestClose(code int64, reason string, flush bool) {
	socket.mutex.Lock()
	defer socket.mutex.Unlock()
	if socket.closing {
		return
	}
	socket.closing = true
	socket.initiated = true
	socket.code = code
	socket.reason = reason
	socket.closeRequests <- ahdWebSocketCloseRequest{code: code, reason: reason, flush: flush}
}

// markClosed records that the peer closed or the connection was lost.
func (socket *ahdWebSocketSocket) markClosed() {
	socket.mutex.Lock()
	socket.closing = true
	socket.mutex.Unlock()
}

func (socket *ahdWebSocketSocket) isClosing() bool {
	socket.mutex.Lock()
	defer socket.mutex.Unlock()
	return socket.closing
}

// AhdWebSocketID is WebSocket.id(): the socket's opaque identifier.
func AhdWebSocketID(class *AhdClass, socket string) string {
	if socket == "" {
		AhdRaiseClass(class, "WebSocket storage is corrupted")
	}
	return socket
}

// AhdWebSocketSend is WebSocket.send(text). It queues the message and returns
// true, or returns false when the socket is closed or closing. A full queue
// closes the socket with 1008 rather than waiting or growing.
func AhdWebSocketSend(class *AhdClass, socketID, text string) bool {
	socket := ahdWebSocketLookup(socketID)
	if socket == nil {
		return false
	}
	socket.mutex.Lock()
	if socket.closing {
		socket.mutex.Unlock()
		return false
	}
	select {
	case socket.outbound <- text:
		socket.mutex.Unlock()
		return true
	default:
		socket.mutex.Unlock()
	}
	socket.requestClose(1008, "outbound queue full", false)
	return false
}

// AhdWebSocketClose is WebSocket.close(code, reason). Closing a closed or
// closing socket does nothing; an invalid code or reason raises HTTPError.
func AhdWebSocketClose(class *AhdClass, socketID string, code int64, reason string) {
	if code != 1000 && code != 1001 && code != 1008 && code != 1011 && (code < 3000 || code > 4999) {
		AhdRaiseClass(class, "WebSocket close code must be 1000, 1001, 1008, 1011, or 3000..4999")
	}
	if len(reason) > ahdWebSocketMaximumReasonBytes || !utf8.ValidString(reason) {
		AhdRaiseClass(class, "WebSocket close reason must be at most 123 bytes of UTF-8")
	}
	if socket := ahdWebSocketLookup(socketID); socket != nil {
		socket.requestClose(code, reason, true)
	}
}

// AhdWebSocketIsOpen is WebSocket.isOpen().
func AhdWebSocketIsOpen(socketID string) bool {
	socket := ahdWebSocketLookup(socketID)
	return socket != nil && !socket.isClosing()
}

// ahdWebSocketTestCloseAll drops every open connection. Only the runtime's
// tests call it, because HTTP server shutdown does not reach hijacked
// connections.
func ahdWebSocketTestCloseAll() {
	ahdWebSocketsMu.Lock()
	sockets := make([]*ahdWebSocketSocket, 0, len(ahdWebSockets))
	for _, socket := range ahdWebSockets {
		sockets = append(sockets, socket)
	}
	ahdWebSocketsMu.Unlock()
	for _, socket := range sockets {
		if socket.forceClose != nil {
			socket.forceClose()
		}
	}
}
