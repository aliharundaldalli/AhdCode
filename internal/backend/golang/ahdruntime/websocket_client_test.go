package ahdruntime

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// The WebSocket client matrix. The fixture is a plain coder/websocket server
// rather than AhdCode's own, so these tests exercise the client against the
// protocol and not against a matching bug in the server half.

type webSocketFixtureOptions struct {
	// echo answers every text message with the same text.
	echo bool
	// greet sends these messages as soon as the connection opens.
	greet []string
	// closeAfterGreeting closes normally once the greeting is sent.
	closeAfterGreeting bool
	closeCode          websocket.StatusCode
	closeReason        string
	// sendBinary sends one binary message instead of text.
	sendBinary bool
	// sendBytes sends one text message of exactly this many bytes.
	sendBytes int
	// dropWithoutHandshake ends the TCP connection with no close frame.
	dropWithoutHandshake bool
	// refuse answers the upgrade with an ordinary HTTP status.
	refuse int
	// requireHeader fails the handshake unless this header has this value.
	requireHeader, requireValue string
	// tls serves over HTTPS with a certificate the test trusts.
	tls bool
	// untrusted serves over HTTPS with a certificate nothing trusts.
	untrusted bool
	// hold keeps the connection open without sending anything.
	hold bool
}

type webSocketFixture struct {
	url       string
	server    *httptest.Server
	seenAuth  chan string
	transport http.RoundTripper
}

func startWebSocketFixture(t *testing.T, options webSocketFixtureOptions) *webSocketFixture {
	t.Helper()
	fixture := &webSocketFixture{seenAuth: make(chan string, 8)}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		select {
		case fixture.seenAuth <- request.Header.Get("Authorization"):
		default:
		}
		if options.refuse != 0 {
			http.Error(writer, http.StatusText(options.refuse), options.refuse)
			return
		}
		if options.requireHeader != "" && request.Header.Get(options.requireHeader) != options.requireValue {
			http.Error(writer, "Forbidden", http.StatusForbidden)
			return
		}
		conn, err := websocket.Accept(writer, request, &websocket.AcceptOptions{
			InsecureSkipVerify: true, CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			return
		}
		serveWebSocketFixture(conn, options)
	})
	switch {
	case options.tls || options.untrusted:
		fixture.server = httptest.NewTLSServer(handler)
		fixture.url = "wss://" + strings.TrimPrefix(fixture.server.URL, "https://")
		if options.tls {
			// The fixture's own certificate authority, and nothing else, is
			// what this handshake trusts. An untrusted fixture leaves the
			// default transport in place so verification really runs.
			pool := x509.NewCertPool()
			pool.AddCert(fixture.server.Certificate())
			fixture.transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}
		}
	default:
		fixture.server = httptest.NewServer(handler)
		fixture.url = "ws://" + strings.TrimPrefix(fixture.server.URL, "http://")
	}
	t.Cleanup(fixture.server.Close)
	if fixture.transport != nil {
		previous := ahdWebSocketClientTransport
		ahdWebSocketClientTransport = fixture.transport
		t.Cleanup(func() { ahdWebSocketClientTransport = previous })
	}
	return fixture
}

func serveWebSocketFixture(conn *websocket.Conn, options webSocketFixtureOptions) {
	defer func() { _ = conn.CloseNow() }()
	ctx := context.Background()
	for _, message := range options.greet {
		if conn.Write(ctx, websocket.MessageText, []byte(message)) != nil {
			return
		}
	}
	if options.sendBinary {
		_ = conn.Write(ctx, websocket.MessageBinary, []byte{0x00, 0x01, 0x02})
	}
	if options.sendBytes > 0 {
		_ = conn.Write(ctx, websocket.MessageText, []byte(strings.Repeat("x", options.sendBytes)))
	}
	if options.dropWithoutHandshake {
		_ = conn.CloseNow()
		return
	}
	if options.closeAfterGreeting {
		code := options.closeCode
		if code == 0 {
			code = websocket.StatusNormalClosure
		}
		_ = conn.Close(code, options.closeReason)
		return
	}
	if options.hold {
		<-ctx.Done()
		return
	}
	if !options.echo {
		return
	}
	conn.SetReadLimit(1 << 20)
	for {
		messageType, payload, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if conn.Write(ctx, messageType, payload) != nil {
			return
		}
	}
}

func connectFixture(t *testing.T, fixture *webSocketFixture, configure func(string) string) string {
	t.Helper()
	handle := AhdHTTPWebSocketClient(AhdClassHTTPError, fixture.url)
	if configure != nil {
		handle = configure(handle)
	}
	connection := AhdWebSocketClientConnect(AhdClassHTTPError, handle)
	t.Cleanup(func() { AhdWebSocketConnectionClose(AhdClassHTTPError, connection, 1000, "") })
	return connection
}

func TestWebSocketClientURLValidation(t *testing.T) {
	class := AhdClassHTTPError
	for _, raw := range []string{
		"http://example.com/", "https://example.com/", "file:///tmp/x", "/relative",
		"ws://", "wss://", "not a url", "ws://example.com/path#frag",
		"ws://user:pass@example.com/", "", "ftp://example.com/",
	} {
		value := raw
		expectRaise(t, class, func() { AhdHTTPWebSocketClient(class, value) })
	}
	for _, raw := range []string{"ws://example.com/socket", "wss://example.com:8443/socket?room=1"} {
		if AhdHTTPWebSocketClient(class, raw) == "" {
			t.Fatalf("%s was refused", raw)
		}
	}
}

func TestWebSocketClientConfigurationIsImmutable(t *testing.T) {
	class := AhdClassHTTPError
	base := AhdHTTPWebSocketClient(class, "ws://example.com/socket")
	withHeader := AhdWebSocketClientWithHeader(class, base, "Authorization", "Bearer one")
	replaced := AhdWebSocketClientWithHeader(class, withHeader, "authorization", "Bearer two")
	tuned := AhdWebSocketClientWithMaxMessageBytes(class, AhdWebSocketClientWithTimeout(class, replaced, 7), 4096)

	if config := ahdWebSocketLookupClient(class, base); len(config.headers) != 0 ||
		config.timeoutSeconds != ahdWebSocketClientDefaultTimeoutSeconds ||
		config.maxMessageBytes != ahdWebSocketDefaultMaxMessageBytes {
		t.Fatalf("the original configuration changed: %#v", config)
	}
	if config := ahdWebSocketLookupClient(class, withHeader); len(config.headers) != 1 || config.headers[0].Value != "Bearer one" {
		t.Fatalf("withHeader = %#v", config.headers)
	}
	// Setting the same header again replaces it rather than sending it twice.
	if config := ahdWebSocketLookupClient(class, replaced); len(config.headers) != 1 || config.headers[0].Value != "Bearer two" {
		t.Fatalf("replacement = %#v", config.headers)
	}
	if config := ahdWebSocketLookupClient(class, tuned); config.timeoutSeconds != 7 || config.maxMessageBytes != 4096 {
		t.Fatalf("tuned = %#v", config)
	}
}

func TestWebSocketClientRejectsProtocolOwnedHeadersAndBadValues(t *testing.T) {
	class := AhdClassHTTPError
	base := AhdHTTPWebSocketClient(class, "ws://example.com/socket")
	for _, name := range []string{
		"Connection", "Upgrade", "upgrade", "Host", "Content-Length",
		"Sec-WebSocket-Key", "sec-websocket-version", "Sec-WebSocket-Extensions",
		"Sec-WebSocket-Protocol", "Sec-WebSocket-Accept",
	} {
		value := name
		expectRaise(t, class, func() { AhdWebSocketClientWithHeader(class, base, value, "x") })
	}
	for _, header := range [][2]string{
		{"Bad Name", "x"}, {"Bad\rName", "x"}, {"", "x"},
		{"X-Test", "line\rbreak"}, {"X-Test", "line\nbreak"},
	} {
		pair := header
		expectRaise(t, class, func() { AhdWebSocketClientWithHeader(class, base, pair[0], pair[1]) })
	}
	for _, seconds := range []int64{0, -1, ahdWebSocketClientMaximumTimeoutSeconds + 1} {
		value := seconds
		expectRaise(t, class, func() { AhdWebSocketClientWithTimeout(class, base, value) })
	}
	for _, bytes := range []int64{0, -1, ahdWebSocketMaximumMessageBytes + 1} {
		value := bytes
		expectRaise(t, class, func() { AhdWebSocketClientWithMaxMessageBytes(class, base, value) })
	}
}

func TestWebSocketClientSendsAndReceivesText(t *testing.T) {
	class := AhdClassHTTPError
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{echo: true})
	connection := connectFixture(t, fixture, nil)

	messages := []string{"first", "ikinci — Türkçe", `{"kind":"tick","value":42}`, "üçüncü 🎉"}
	for _, message := range messages {
		if !AhdWebSocketConnectionSend(class, connection, message) {
			t.Fatalf("send(%q) reported a closed connection", message)
		}
		received := AhdWebSocketConnectionReceive(class, connection, 5)
		if received == nil {
			t.Fatalf("receive after %q returned null", message)
		}
		if *received != message {
			t.Fatalf("received %q, want %q", *received, message)
		}
	}
	if !AhdWebSocketConnectionIsOpen(class, connection) {
		t.Fatal("the connection should still be open")
	}
	if AhdWebSocketConnectionCloseCode(class, connection) != nil {
		t.Fatal("an open connection reported a close code")
	}
	if AhdWebSocketConnectionCloseReason(class, connection) != "" {
		t.Fatal("an open connection reported a close reason")
	}
}

func TestWebSocketClientKeepsMessageOrder(t *testing.T) {
	class := AhdClassHTTPError
	expected := []string{"one", "two", "three", "four", "five"}
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{greet: expected, closeAfterGreeting: true})
	connection := connectFixture(t, fixture, nil)
	for _, want := range expected {
		received := AhdWebSocketConnectionReceive(class, connection, 5)
		if received == nil || *received != want {
			t.Fatalf("received %v, want %q", received, want)
		}
	}
	if received := AhdWebSocketConnectionReceive(class, connection, 5); received != nil {
		t.Fatalf("the close was reported as a message: %q", *received)
	}
}

func TestWebSocketClientReportsANormalPeerClose(t *testing.T) {
	class := AhdClassHTTPError
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{
		greet: []string{"bye soon"}, closeAfterGreeting: true,
		closeCode: websocket.StatusGoingAway, closeReason: "server restarting",
	})
	connection := connectFixture(t, fixture, nil)
	if first := AhdWebSocketConnectionReceive(class, connection, 5); first == nil || *first != "bye soon" {
		t.Fatalf("first message = %v", first)
	}
	if received := AhdWebSocketConnectionReceive(class, connection, 5); received != nil {
		t.Fatalf("expected null at the close; received %q", *received)
	}
	code := AhdWebSocketConnectionCloseCode(class, connection)
	if code == nil || *code != 1001 {
		t.Fatalf("close code = %v", code)
	}
	if reason := AhdWebSocketConnectionCloseReason(class, connection); reason != "server restarting" {
		t.Fatalf("close reason = %q", reason)
	}
	if AhdWebSocketConnectionIsOpen(class, connection) {
		t.Fatal("a closed connection reports itself open")
	}
	// A send after the close reports false rather than raising or pretending.
	if AhdWebSocketConnectionSend(class, connection, "too late") {
		t.Fatal("send after close returned true")
	}
	// There is no reconnect: the connection stays closed.
	if AhdWebSocketConnectionIsOpen(class, connection) {
		t.Fatal("the connection reopened by itself")
	}
}

func TestWebSocketClientLocalCloseRecordsItsOwnCodeAndReason(t *testing.T) {
	class := AhdClassHTTPError
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{echo: true})
	handle := AhdHTTPWebSocketClient(class, fixture.url)
	connection := AhdWebSocketClientConnect(class, handle)

	AhdWebSocketConnectionClose(class, connection, 1001, "done here")
	if AhdWebSocketConnectionIsOpen(class, connection) {
		t.Fatal("still open after close")
	}
	code := AhdWebSocketConnectionCloseCode(class, connection)
	if code == nil || *code != 1001 {
		t.Fatalf("close code = %v", code)
	}
	if reason := AhdWebSocketConnectionCloseReason(class, connection); reason != "done here" {
		t.Fatalf("close reason = %q", reason)
	}
	// Closing twice is not an error and does not change what was recorded.
	AhdWebSocketConnectionClose(class, connection, 1011, "different")
	if again := AhdWebSocketConnectionCloseCode(class, connection); again == nil || *again != 1001 {
		t.Fatalf("the second close changed the recorded code: %v", again)
	}
	// The close-code policy is the server's.
	for _, code := range []int64{0, 999, 1005, 1006, 1015, 2999, 5000} {
		value := code
		expectRaise(t, class, func() { AhdWebSocketConnectionClose(class, connection, value, "") })
	}
	expectRaise(t, class, func() {
		AhdWebSocketConnectionClose(class, connection, 1000, strings.Repeat("r", ahdWebSocketMaximumReasonBytes+1))
	})
}

func TestWebSocketClientReceiveTimeoutLeavesTheConnectionOpen(t *testing.T) {
	class := AhdClassHTTPError
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{echo: true})
	connection := connectFixture(t, fixture, nil)

	// Nothing has been sent, so this receive can only time out.
	expectRaise(t, class, func() { AhdWebSocketConnectionReceive(class, connection, 1) })
	if !AhdWebSocketConnectionIsOpen(class, connection) {
		t.Fatal("a receive timeout closed the connection")
	}
	// The same connection still works afterwards; this is what the reader
	// goroutine exists for.
	if !AhdWebSocketConnectionSend(class, connection, "still here") {
		t.Fatal("send after a receive timeout failed")
	}
	received := AhdWebSocketConnectionReceive(class, connection, 5)
	if received == nil || *received != "still here" {
		t.Fatalf("received %v after a timeout", received)
	}
	// A negative timeout is refused rather than treated as "wait forever".
	expectRaise(t, class, func() { AhdWebSocketConnectionReceive(class, connection, -1) })
}

func TestWebSocketClientRefusesBinaryMessages(t *testing.T) {
	class := AhdClassHTTPError
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{sendBinary: true, hold: true})
	connection := connectFixture(t, fixture, nil)
	expectRaise(t, class, func() { AhdWebSocketConnectionReceive(class, connection, 5) })
	code := AhdWebSocketConnectionCloseCode(class, connection)
	if code == nil || *code != ahdWebSocketClientCodeUnsupportedData {
		t.Fatalf("a binary message closed with %v, want 1003", code)
	}
}

func TestWebSocketClientEnforcesTheMessageSizeLimit(t *testing.T) {
	class := AhdClassHTTPError
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{sendBytes: 2048, hold: true})
	connection := connectFixture(t, fixture, func(handle string) string {
		return AhdWebSocketClientWithMaxMessageBytes(class, handle, 512)
	})
	expectRaise(t, class, func() { AhdWebSocketConnectionReceive(class, connection, 5) })
	code := AhdWebSocketConnectionCloseCode(class, connection)
	if code == nil || *code != ahdWebSocketClientCodeMessageTooBig {
		t.Fatalf("an oversize message closed with %v, want 1009", code)
	}

	// Sending past the limit is refused before anything reaches the wire.
	other := startWebSocketFixture(t, webSocketFixtureOptions{echo: true})
	small := connectFixture(t, other, func(handle string) string {
		return AhdWebSocketClientWithMaxMessageBytes(class, handle, 8)
	})
	expectRaise(t, class, func() { AhdWebSocketConnectionSend(class, small, strings.Repeat("x", 9)) })
	if !AhdWebSocketConnectionSend(class, small, "12345678") {
		t.Fatal("a message exactly at the limit was refused")
	}
}

func TestWebSocketClientReportsAnAbruptTransportLoss(t *testing.T) {
	class := AhdClassHTTPError
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{
		greet: []string{"before the drop"}, dropWithoutHandshake: true,
	})
	connection := connectFixture(t, fixture, nil)
	if first := AhdWebSocketConnectionReceive(class, connection, 5); first == nil || *first != "before the drop" {
		t.Fatalf("first message = %v", first)
	}
	// A lost connection is a failure, not the null a clean close returns.
	expectRaise(t, class, func() { AhdWebSocketConnectionReceive(class, connection, 5) })
	code := AhdWebSocketConnectionCloseCode(class, connection)
	if code == nil || *code != ahdWebSocketClientCodeAbnormal {
		t.Fatalf("close code after a drop = %v, want 1006", code)
	}
}

func TestWebSocketClientSendsHandshakeHeaders(t *testing.T) {
	class := AhdClassHTTPError
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{
		echo: true, requireHeader: "Authorization", requireValue: "Bearer token-value",
	})
	// Without the header the handshake is refused.
	expectRaise(t, class, func() {
		AhdWebSocketClientConnect(class, AhdHTTPWebSocketClient(class, fixture.url))
	})
	connection := connectFixture(t, fixture, func(handle string) string {
		return AhdWebSocketClientWithHeader(class, handle, "Authorization", "Bearer token-value")
	})
	if !AhdWebSocketConnectionSend(class, connection, "hello") {
		t.Fatal("send")
	}
	if received := AhdWebSocketConnectionReceive(class, connection, 5); received == nil || *received != "hello" {
		t.Fatalf("received %v", received)
	}
}

// A refused handshake must not put the Authorization header, or anything
// else from the request, into the message an AhdCode program sees.
func TestWebSocketClientFailureNeverNamesTheCredential(t *testing.T) {
	class := AhdClassHTTPError
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{refuse: http.StatusUnauthorized})
	handle := AhdWebSocketClientWithHeader(class, AhdHTTPWebSocketClient(class, fixture.url),
		"Authorization", "Bearer super-secret-token")
	message := captureWebSocketRaise(t, class, func() { AhdWebSocketClientConnect(class, handle) })
	if strings.Contains(message, "super-secret-token") || strings.Contains(strings.ToLower(message), "authorization") {
		t.Fatalf("the failure names the credential: %s", message)
	}
	if !strings.HasPrefix(message, "HTTP WebSocket") {
		t.Fatalf("unexpected failure wording: %s", message)
	}
}

func TestWebSocketClientHandshakeRefusalAndUnreachableHost(t *testing.T) {
	class := AhdClassHTTPError
	refused := startWebSocketFixture(t, webSocketFixtureOptions{refuse: http.StatusForbidden})
	expectRaise(t, class, func() {
		AhdWebSocketClientConnect(class, AhdHTTPWebSocketClient(class, refused.url))
	})
	// A port nothing listens on fails at the transport, not at the protocol.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	expectRaise(t, class, func() {
		AhdWebSocketClientConnect(class, AhdHTTPWebSocketClient(class, "ws://"+address+"/socket"))
	})
}

func TestWebSocketClientVerifiesTLS(t *testing.T) {
	class := AhdClassHTTPError
	// A wss:// fixture whose certificate the transport trusts works exactly
	// like a ws:// one.
	trusted := startWebSocketFixture(t, webSocketFixtureOptions{tls: true, echo: true})
	if !strings.HasPrefix(trusted.url, "wss://") {
		t.Fatalf("fixture url = %s", trusted.url)
	}
	connection := connectFixture(t, trusted, nil)
	if !AhdWebSocketConnectionSend(class, connection, "over tls") {
		t.Fatal("send over wss")
	}
	if received := AhdWebSocketConnectionReceive(class, connection, 5); received == nil || *received != "over tls" {
		t.Fatalf("received %v over wss", received)
	}
}

func TestWebSocketClientRefusesAnUntrustedCertificate(t *testing.T) {
	class := AhdClassHTTPError
	// No transport is injected here, so the handshake uses the real default
	// and must reject a certificate the system roots do not know.
	fixture := startWebSocketFixture(t, webSocketFixtureOptions{untrusted: true, echo: true})
	message := captureWebSocketRaise(t, class, func() {
		AhdWebSocketClientConnect(class, AhdHTTPWebSocketClient(class, fixture.url))
	})
	if !strings.Contains(message, "TLS") {
		t.Fatalf("an untrusted certificate failed with %q", message)
	}
}

func TestWebSocketClientConnectTimesOut(t *testing.T) {
	class := AhdClassHTTPError
	// A listener that accepts the TCP connection and then says nothing at
	// all, so only the handshake timeout can end the attempt.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				_, _ = io.Copy(io.Discard, conn)
			}()
		}
	}()
	handle := AhdWebSocketClientWithTimeout(class,
		AhdHTTPWebSocketClient(class, "ws://"+listener.Addr().String()+"/socket"), 1)
	started := time.Now()
	expectRaise(t, class, func() { AhdWebSocketClientConnect(class, handle) })
	if elapsed := time.Since(started); elapsed > 10*time.Second {
		t.Fatalf("the handshake timeout took %s", elapsed)
	}
}

func TestWebSocketClientCorruptedStorageIsReported(t *testing.T) {
	class := AhdClassHTTPError
	expectRaise(t, class, func() { AhdWebSocketClientWithTimeout(class, "nonexistent", 5) })
	expectRaise(t, class, func() { AhdWebSocketConnectionSend(class, "nonexistent", "x") })
	expectRaise(t, class, func() { AhdWebSocketConnectionReceive(class, "nonexistent", 1) })
	expectRaise(t, class, func() { AhdWebSocketConnectionIsOpen(class, "nonexistent") })
}

// captureWebSocketRaise runs body, requires that it raised the expected
// class, and returns the message so a test can check what it does and does
// not say.
func captureWebSocketRaise(t *testing.T, class *AhdClass, body func()) (message string) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected an AhdCode %s", class.Name)
		}
		signal, ok := recovered.(*AhdSignal)
		if !ok {
			t.Fatalf("expected an AhdSignal; received %v", recovered)
		}
		if signal.Instance.AhdClassOf() != class {
			t.Fatalf("expected %s; received %s", class.Name, signal.Instance.AhdClassOf().Name)
		}
		message = signal.Message
	}()
	body()
	return ""
}

// Closing every client connection at the end keeps a fixture's reader
// goroutine from outliving the test binary's last test.
func TestWebSocketClientConnectionsAreAllReleased(t *testing.T) {
	ahdWebSocketClientTestCloseAll()
	ahdWebSocketConnectionsMu.Lock()
	defer ahdWebSocketConnectionsMu.Unlock()
	for _, connection := range ahdWebSocketConnections {
		connection.mutex.Lock()
		open := !connection.closed
		connection.mutex.Unlock()
		if open {
			t.Fatal("a client connection was left open")
		}
	}
}
