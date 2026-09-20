package ahdruntime

// The WebSocket client (v2.1.0).
//
// v1.4.0 gave AhdCode a WebSocket server. This is the other end of the same
// connection, and it is deliberately not the same value: a WebSocketClient is
// immutable configuration with no socket behind it, and a WebSocketConnection
// is one live connection that connect() produced. Keeping them apart means a
// program can never call receive on something that was never dialled, and the
// type checker says so.
//
// receive is synchronous. There is no callback, no background event bus, and
// no automatic reconnect: a program asks for the next message and waits. One
// reader goroutine per connection does the actual framing read, so a receive
// that times out leaves the connection open and usable -- reading with a
// deadline straight off the library's connection would close it instead.
// The channel it hands messages over is unbuffered, so the next frame is read
// only after the previous message was taken: a slow program slows its peer
// rather than growing a queue.
//
// This file is standard library only and joins every generated program, the
// same way websocket.go does. The RFC 6455 connection itself is the vendored
// github.com/coder/websocket and lives in websocket_conn.go, which a program
// receives only when it uses WebSocket at all.

import (
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

const (
	ahdWebSocketClientDefaultTimeoutSeconds = 30
	// ahdWebSocketClientMaximumTimeoutSeconds is the largest whole-second
	// timeout that still fits in a time.Duration after conversion to
	// nanoseconds, the same bound the HTTP client uses.
	ahdWebSocketClientMaximumTimeoutSeconds = 9223372036
)

// The close codes a client sends for a protocol problem it detects itself.
// They are the RFC 6455 codes the server half uses for the same situations.
const (
	ahdWebSocketClientCodeUnsupportedData = 1003
	ahdWebSocketClientCodeInvalidPayload  = 1007
	ahdWebSocketClientCodeMessageTooBig   = 1009
	ahdWebSocketClientCodeAbnormal        = 1006
)

// ahdWebSocketClientConfig is one immutable WebSocketClient. Every with*
// member stores a changed copy under a new handle, exactly like
// WebSocketEndpoint.
type ahdWebSocketClientConfig struct {
	url             string
	headers         []ahdHTTPHeaderPair
	timeoutSeconds  int64
	maxMessageBytes int64
}

// ahdWebSocketClientLink is one live connection as the always-compiled half
// sees it: an interface, so this file never names the vendored library.
// websocket_conn.go installs the dialer that produces one.
type ahdWebSocketClientLink interface {
	// read blocks for the next frame. done reports that the connection ended
	// instead; code and reason then describe how.
	read() (message ahdWebSocketClientFrame, done bool)
	// write sends one text message within the write deadline.
	write(text string, timeout time.Duration) error
	// shutdown performs the closing handshake with the given code.
	shutdown(code int64, reason string)
	// drop ends the connection without a handshake.
	drop()
}

// ahdWebSocketClientFrame is one frame the reader goroutine delivers, or the
// terminal state that ended the connection.
type ahdWebSocketClientFrame struct {
	text   string
	binary bool
	// oversize reports a message past maxMessageBytes; invalidUTF8 a text
	// message that is not valid UTF-8. Both are protocol problems the client
	// answers with its own close code.
	oversize    bool
	invalidUTF8 bool
	// closed reports the end of the connection. code is the peer's close
	// code, or 1006 when the connection was lost.
	closed bool
	code   int64
	reason string
}

// ahdWebSocketClientDialer is installed by websocket_conn.go. It is nil only
// in a program that uses no WebSocket at all, which therefore never reaches
// connect.
var ahdWebSocketClientDialer func(config ahdWebSocketClientConfig) (ahdWebSocketClientLink, error)

var (
	ahdWebSocketClients    = map[string]*ahdWebSocketClientConfig{}
	ahdWebSocketClientsMu  sync.Mutex
	ahdWebSocketNextClient atomic.Int64
)

// AhdHTTPWebSocketClient is HTTP.webSocketClient(url). It performs no network
// activity: it validates the URL and stores the configuration.
func AhdHTTPWebSocketClient(class *AhdClass, rawURL string) string {
	ahdWebSocketRequireClientURL(class, rawURL)
	return ahdWebSocketStoreClient(&ahdWebSocketClientConfig{
		url:             rawURL,
		timeoutSeconds:  ahdWebSocketClientDefaultTimeoutSeconds,
		maxMessageBytes: ahdWebSocketDefaultMaxMessageBytes,
	})
}

// ahdWebSocketRequireClientURL accepts exactly ws:// and wss://. http:// and
// https:// are refused rather than quietly rewritten, so a program says which
// protocol it means.
func ahdWebSocketRequireClientURL(class *AhdClass, raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		AhdRaiseClass(class, "WebSocket client URL must be an absolute ws or wss URL with a host")
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		AhdRaiseClass(class, "WebSocket client URL scheme must be ws or wss")
	}
	if parsed.Fragment != "" || strings.Contains(raw, "#") {
		AhdRaiseClass(class, "WebSocket client URL must not contain a fragment")
	}
	if parsed.User != nil {
		AhdRaiseClass(class, "WebSocket client URL must not contain userinfo; use an Authorization header")
	}
	if parsed.Hostname() == "" {
		AhdRaiseClass(class, "WebSocket client URL must include a host")
	}
	return parsed
}

func ahdWebSocketStoreClient(config *ahdWebSocketClientConfig) string {
	id := "c" + strconv.FormatInt(ahdWebSocketNextClient.Add(1), 10)
	ahdWebSocketClientsMu.Lock()
	ahdWebSocketClients[id] = config
	ahdWebSocketClientsMu.Unlock()
	return id
}

func ahdWebSocketLookupClient(class *AhdClass, handle string) *ahdWebSocketClientConfig {
	ahdWebSocketClientsMu.Lock()
	config := ahdWebSocketClients[handle]
	ahdWebSocketClientsMu.Unlock()
	if config == nil {
		AhdRaiseClass(class, "WebSocketClient storage is corrupted")
	}
	return config
}

func ahdWebSocketDeriveClient(class *AhdClass, handle string, change func(*ahdWebSocketClientConfig)) string {
	derived := *ahdWebSocketLookupClient(class, handle)
	derived.headers = append([]ahdHTTPHeaderPair(nil), derived.headers...)
	change(&derived)
	return ahdWebSocketStoreClient(&derived)
}

// ahdWebSocketProtocolHeaders are the handshake headers the protocol and the
// connection library own. Letting a program set one would either be ignored
// or break the upgrade, so setting one is refused instead.
var ahdWebSocketProtocolHeaders = []string{
	"Connection", "Upgrade", "Host", "Content-Length",
	"Sec-Websocket-Key", "Sec-Websocket-Version", "Sec-Websocket-Accept",
	"Sec-Websocket-Extensions", "Sec-Websocket-Protocol",
}

// AhdWebSocketClientWithHeader is WebSocketClient.withHeader(name, value).
// It follows ClientRequest.withHeader: setting a header that is already
// present replaces it, so repeating a call never sends the header twice.
func AhdWebSocketClientWithHeader(class *AhdClass, handle, name, value string) string {
	if !ahdHTTPHeaderNameOK(name) {
		AhdRaiseClass(class, "HTTP header name "+ahdHTMLQuote(name)+" is not valid")
	}
	if strings.ContainsAny(value, "\r\n") {
		AhdRaiseClass(class, "HTTP header value must not contain CR or LF")
	}
	canonical := ahdWebSocketCanonicalHeader(name)
	for _, owned := range ahdWebSocketProtocolHeaders {
		if canonical == owned {
			AhdRaiseClass(class, "WebSocket handshake header "+ahdHTMLQuote(name)+" is set by the protocol and cannot be changed")
		}
	}
	return ahdWebSocketDeriveClient(class, handle, func(config *ahdWebSocketClientConfig) {
		headers := make([]ahdHTTPHeaderPair, 0, len(config.headers)+1)
		replaced := false
		for _, header := range config.headers {
			if ahdWebSocketCanonicalHeader(header.Name) == canonical {
				if !replaced {
					headers = append(headers, ahdHTTPHeaderPair{Name: name, Value: value})
					replaced = true
				}
				continue
			}
			headers = append(headers, header)
		}
		if !replaced {
			headers = append(headers, ahdHTTPHeaderPair{Name: name, Value: value})
		}
		config.headers = headers
	})
}

// ahdWebSocketCanonicalHeader is textproto's canonical form written out here
// so this file needs no import for one call: Sec-Websocket-Key, not
// Sec-WebSocket-Key. Only the comparison matters; the header a program set is
// sent with the spelling it wrote.
func ahdWebSocketCanonicalHeader(name string) string {
	parts := strings.Split(strings.ToLower(name), "-")
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "-")
}

// AhdWebSocketClientWithTimeout is WebSocketClient.withTimeout(seconds): how
// long the opening handshake, and each later send, may take.
func AhdWebSocketClientWithTimeout(class *AhdClass, handle string, seconds int64) string {
	if seconds < 1 || seconds > ahdWebSocketClientMaximumTimeoutSeconds {
		AhdRaiseClass(class, "WebSocketClient timeoutSeconds must be between 1 and 9223372036")
	}
	return ahdWebSocketDeriveClient(class, handle, func(config *ahdWebSocketClientConfig) {
		config.timeoutSeconds = seconds
	})
}

// AhdWebSocketClientWithMaxMessageBytes is
// WebSocketClient.withMaxMessageBytes(bytes). The default and the range are
// the server endpoint's, so the two ends of one connection are configured the
// same way.
func AhdWebSocketClientWithMaxMessageBytes(class *AhdClass, handle string, bytes int64) string {
	if bytes < 1 || bytes > ahdWebSocketMaximumMessageBytes {
		AhdRaiseClass(class, "WebSocketClient maxMessageBytes must be in 1..16777216")
	}
	return ahdWebSocketDeriveClient(class, handle, func(config *ahdWebSocketClientConfig) {
		config.maxMessageBytes = bytes
	})
}

// --- the live connection ---

// ahdWebSocketConnection is the state behind one WebSocketConnection value.
// The reader goroutine owns link.read; everything else is guarded by mutex.
type ahdWebSocketConnection struct {
	link            ahdWebSocketClientLink
	maxMessageBytes int64
	writeTimeout    time.Duration

	messages chan ahdWebSocketClientFrame
	finished chan struct{}
	// localClose is closed when the program closes the connection, so a
	// reader waiting to hand a message over stops waiting.
	localClose chan struct{}

	mutex       sync.Mutex
	closed      bool
	hasCloseEnd bool
	closeCode   int64
	closeReason string
	// failure is the protocol problem the reader found, reported once to the
	// receive that observes it.
	failure string
}

var (
	ahdWebSocketConnections   = map[string]*ahdWebSocketConnection{}
	ahdWebSocketConnectionsMu sync.Mutex
	ahdWebSocketNextConn      atomic.Int64
)

// AhdWebSocketClientConnect is WebSocketClient.connect(). It performs the
// opening handshake and returns a live WebSocketConnection, or raises
// HTTPError.
func AhdWebSocketClientConnect(class *AhdClass, handle string) string {
	config := ahdWebSocketLookupClient(class, handle)
	dial := ahdWebSocketClientDialer
	if dial == nil {
		AhdRaiseClass(class, "WebSocket support is not part of this program")
	}
	link, err := dial(*config)
	if err != nil {
		ahdWebSocketRaiseClientFailure(class, err)
	}
	connection := &ahdWebSocketConnection{
		link:            link,
		maxMessageBytes: config.maxMessageBytes,
		writeTimeout:    time.Duration(config.timeoutSeconds) * time.Second,
		messages:        make(chan ahdWebSocketClientFrame),
		finished:        make(chan struct{}),
		localClose:      make(chan struct{}),
	}
	go connection.readLoop()
	id := "w" + strconv.FormatInt(ahdWebSocketNextConn.Add(1), 10)
	ahdWebSocketConnectionsMu.Lock()
	ahdWebSocketConnections[id] = connection
	ahdWebSocketConnectionsMu.Unlock()
	return id
}

// readLoop is the one goroutine that reads frames. It hands each message over
// an unbuffered channel, so it reads the next frame only after the program
// took the previous one.
//
// A protocol problem is recorded and published before the closing handshake
// is performed, never after: the handshake waits for the peer's close frame,
// and a receive waiting for the failure must not wait for that too.
func (connection *ahdWebSocketConnection) readLoop() {
	code, reason, failure := connection.readMessages()
	connection.recordEnd(code, reason, failure)
	close(connection.finished)
	if failure != "" {
		connection.link.shutdown(code, reason)
	}
}

// readMessages delivers messages until the connection ends, and reports how
// it ended: a close code and reason, plus the failure message when AhdCode
// itself ended it over a protocol problem.
func (connection *ahdWebSocketConnection) readMessages() (int64, string, string) {
	for {
		frame, done := connection.link.read()
		if done {
			return frame.code, frame.reason, ""
		}
		switch {
		case frame.oversize:
			return ahdWebSocketClientCodeMessageTooBig, "message too large",
				"HTTP WebSocket message exceeds maxMessageBytes"
		case frame.binary:
			return ahdWebSocketClientCodeUnsupportedData, "binary messages are not supported",
				"HTTP WebSocket binary messages are not supported"
		case frame.invalidUTF8:
			return ahdWebSocketClientCodeInvalidPayload, "text message is not valid UTF-8",
				"HTTP WebSocket text message is not valid UTF-8"
		}
		select {
		case connection.messages <- frame:
		case <-connection.localClose:
			return ahdWebSocketClientCodeAbnormal, "", ""
		}
	}
}

// recordEnd records how the connection ended. A close the program started
// already set these fields, and keeps them: what it asked for is what
// closeCode and closeReason report.
func (connection *ahdWebSocketConnection) recordEnd(code int64, reason, failure string) {
	connection.mutex.Lock()
	defer connection.mutex.Unlock()
	connection.closed = true
	if !connection.hasCloseEnd {
		connection.hasCloseEnd = true
		connection.closeCode = code
		connection.closeReason = reason
	}
	if failure != "" && connection.failure == "" {
		connection.failure = failure
	}
}

func ahdWebSocketLookupConnection(class *AhdClass, handle string) *ahdWebSocketConnection {
	ahdWebSocketConnectionsMu.Lock()
	connection := ahdWebSocketConnections[handle]
	ahdWebSocketConnectionsMu.Unlock()
	if connection == nil {
		AhdRaiseClass(class, "WebSocketConnection storage is corrupted")
	}
	return connection
}

// AhdWebSocketConnectionSend is WebSocketConnection.send(text). It returns
// false when the connection is already closed or closing, and raises
// HTTPError when the write itself fails -- a lost connection is not silently
// reported as a delivered message.
func AhdWebSocketConnectionSend(class *AhdClass, handle, text string) bool {
	connection := ahdWebSocketLookupConnection(class, handle)
	if !utf8.ValidString(text) {
		AhdRaiseClass(class, "HTTP WebSocket message must be valid UTF-8")
	}
	if int64(len(text)) > connection.maxMessageBytes {
		AhdRaiseClass(class, "HTTP WebSocket message exceeds maxMessageBytes")
	}
	connection.mutex.Lock()
	closed := connection.closed
	connection.mutex.Unlock()
	if closed {
		return false
	}
	if err := connection.link.write(text, connection.writeTimeout); err != nil {
		connection.mutex.Lock()
		alreadyClosed := connection.closed
		connection.mutex.Unlock()
		if alreadyClosed {
			return false
		}
		ahdWebSocketRaiseClientFailure(class, err)
	}
	return true
}

// AhdWebSocketConnectionReceive is
// WebSocketConnection.receive(timeoutSeconds).
//
//   - a text message returns that String
//   - a normal close by the peer returns null; closeCode and closeReason then
//     describe it
//   - a lost connection, a protocol problem, or an expired timeout raises
//     HTTPError
//
// An expired timeout leaves the connection open: the reader goroutine is
// still waiting, and a later receive picks the message up. That is the whole
// reason the read happens in its own goroutine rather than under a deadline.
func AhdWebSocketConnectionReceive(class *AhdClass, handle string, timeoutSeconds int64) *string {
	connection := ahdWebSocketLookupConnection(class, handle)
	if timeoutSeconds < 0 || timeoutSeconds > ahdWebSocketClientMaximumTimeoutSeconds {
		AhdRaiseClass(class, "WebSocketConnection receive timeoutSeconds must be between 0 and 9223372036")
	}
	var deadline <-chan time.Time
	if timeoutSeconds > 0 {
		timer := time.NewTimer(time.Duration(timeoutSeconds) * time.Second)
		defer timer.Stop()
		deadline = timer.C
	}
	select {
	case frame := <-connection.messages:
		text := frame.text
		return &text
	case <-connection.finished:
		return connection.endOfStream(class)
	case <-deadline:
		AhdRaiseClass(class, "HTTP WebSocket receive timed out")
		return nil
	}
}

// endOfStream answers a receive that found the connection already ended: a
// protocol problem or a lost connection raises, a normal close returns null.
// No message can be missed here: the reader hands each one over an unbuffered
// channel and only then reads on, so finished is closed with nothing pending.
func (connection *ahdWebSocketConnection) endOfStream(class *AhdClass) *string {
	connection.mutex.Lock()
	failure := connection.failure
	code := connection.closeCode
	connection.mutex.Unlock()
	if failure != "" {
		AhdRaiseClass(class, failure)
	}
	if code == ahdWebSocketClientCodeAbnormal {
		AhdRaiseClass(class, "HTTP WebSocket connection was lost")
	}
	return nil
}

// AhdWebSocketConnectionClose is WebSocketConnection.close(code, reason). It
// follows the server's close-code policy, and closing an already closed
// connection does nothing.
func AhdWebSocketConnectionClose(class *AhdClass, handle string, code int64, reason string) {
	connection := ahdWebSocketLookupConnection(class, handle)
	if code != 1000 && code != 1001 && code != 1008 && code != 1011 && (code < 3000 || code > 4999) {
		AhdRaiseClass(class, "WebSocket close code must be 1000, 1001, 1008, 1011, or 3000..4999")
	}
	if len(reason) > ahdWebSocketMaximumReasonBytes || !utf8.ValidString(reason) {
		AhdRaiseClass(class, "WebSocket close reason must be at most 123 bytes of UTF-8")
	}
	connection.mutex.Lock()
	if connection.closed {
		connection.mutex.Unlock()
		return
	}
	connection.closed = true
	connection.hasCloseEnd = true
	connection.closeCode = code
	connection.closeReason = reason
	close(connection.localClose)
	connection.mutex.Unlock()
	connection.link.shutdown(code, reason)
}

func AhdWebSocketConnectionIsOpen(class *AhdClass, handle string) bool {
	connection := ahdWebSocketLookupConnection(class, handle)
	connection.mutex.Lock()
	defer connection.mutex.Unlock()
	return !connection.closed
}

// AhdWebSocketConnectionCloseCode is WebSocketConnection.closeCode(): null
// while the connection is open, then the code that ended it -- the one this
// program sent when it closed, the peer's otherwise, or 1006 when the
// connection was lost.
func AhdWebSocketConnectionCloseCode(class *AhdClass, handle string) *int64 {
	connection := ahdWebSocketLookupConnection(class, handle)
	connection.mutex.Lock()
	defer connection.mutex.Unlock()
	if !connection.hasCloseEnd {
		return nil
	}
	code := connection.closeCode
	return &code
}

func AhdWebSocketConnectionCloseReason(class *AhdClass, handle string) string {
	connection := ahdWebSocketLookupConnection(class, handle)
	connection.mutex.Lock()
	defer connection.mutex.Unlock()
	return connection.closeReason
}

// ahdWebSocketRaiseClientFailure maps one transport failure to AhdCode's
// fixed wording. A handshake header such as Authorization never appears in
// it: the message names the stage, not the request.
func ahdWebSocketRaiseClientFailure(class *AhdClass, err error) {
	if err == nil {
		AhdRaiseClass(class, "HTTP WebSocket connection failed")
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "tls") || strings.Contains(message, "certificate") || strings.Contains(message, "x509"):
		AhdRaiseClass(class, "HTTP WebSocket TLS verification failed")
	case strings.Contains(message, "timeout") || strings.Contains(message, "deadline exceeded"):
		AhdRaiseClass(class, "HTTP WebSocket connection timed out")
	case strings.Contains(message, "expected handshake response status code 101"),
		strings.Contains(message, "failed to websocket dial"):
		AhdRaiseClass(class, "HTTP WebSocket handshake was refused")
	}
	AhdRaiseClass(class, "HTTP WebSocket connection failed")
}

// ahdWebSocketClientTestCloseAll drops every open client connection. Only the
// runtime's tests call it.
func ahdWebSocketClientTestCloseAll() {
	ahdWebSocketConnectionsMu.Lock()
	connections := make([]*ahdWebSocketConnection, 0, len(ahdWebSocketConnections))
	for _, connection := range ahdWebSocketConnections {
		connections = append(connections, connection)
	}
	ahdWebSocketConnectionsMu.Unlock()
	for _, connection := range connections {
		connection.mutex.Lock()
		if !connection.closed {
			connection.closed = true
			close(connection.localClose)
		}
		connection.mutex.Unlock()
		connection.link.drop()
	}
}
