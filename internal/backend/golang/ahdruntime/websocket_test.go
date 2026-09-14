package ahdruntime

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// startWebSocketServer starts an HTTP server, lets register add routes and
// endpoints, and returns its ws:// base URL. Open connections are dropped
// before the server shuts down, because shutdown does not reach hijacked ones.
func startWebSocketServer(t *testing.T, register func(handle string)) string {
	t.Helper()
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, register)
	t.Cleanup(ahdWebSocketTestCloseAll)
	return "ws" + strings.TrimPrefix(base, "http")
}

func dialWebSocket(t *testing.T, url string, header http.Header) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, response, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("dial %s failed: %v (status %d)", url, err, status)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

// dialStatus dials and returns the HTTP status of a refused handshake.
func dialStatus(t *testing.T, url string, header http.Header) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, response, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: header})
	if err == nil {
		_ = conn.CloseNow()
		t.Fatalf("dial %s unexpectedly succeeded", url)
	}
	if response == nil {
		t.Fatalf("dial %s failed without a response: %v", url, err)
	}
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

func readText(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	kind, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if kind != websocket.MessageText {
		t.Fatalf("read a %v message", kind)
	}
	return string(data)
}

func writeText(t *testing.T, conn *websocket.Conn, text string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, []byte(text)); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

// readClose reads until the server closes and returns its close code and reason.
func readClose(t *testing.T, conn *websocket.Conn) (websocket.StatusCode, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		_, _, err := conn.Read(ctx)
		if err == nil {
			continue
		}
		var closeError websocket.CloseError
		if errors.As(err, &closeError) {
			return closeError.Code, closeError.Reason
		}
		return -1, err.Error()
	}
}

// recorder collects callback events in order, safely for the test goroutine.
type recorder struct {
	mutex  sync.Mutex
	events []string
}

func (r *recorder) add(event string) {
	r.mutex.Lock()
	r.events = append(r.events, event)
	r.mutex.Unlock()
}

func (r *recorder) snapshot() []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return append([]string(nil), r.events...)
}

func (r *recorder) waitFor(t *testing.T, prefix string) []string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		events := r.snapshot()
		for _, event := range events {
			if strings.HasPrefix(event, prefix) {
				return events
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("no %q event; events: %q", prefix, r.snapshot())
	return nil
}

func recordingEndpoint(events *recorder, onMessage AhdWebSocketMessageHandler) string {
	endpoint := AhdHTTPWebSocket(AhdClassHTTPError, onMessage)
	endpoint = AhdWebSocketEndpointWithOpen(AhdClassHTTPError, endpoint, func(socket string, request string) {
		events.add("open " + AhdHTTPRequestPath(request))
	})
	return AhdWebSocketEndpointWithClose(AhdClassHTTPError, endpoint, func(socket string, code int64, reason string) {
		events.add("close " + strconv.FormatInt(code, 10) + " " + reason)
	})
}

func TestWebSocketEchoAndCallbackOrder(t *testing.T) {
	events := &recorder{}
	var sawRequest atomic.Bool
	base := startWebSocketServer(t, func(handle string) {
		endpoint := AhdHTTPWebSocket(AhdClassHTTPError, func(socket string, text string) {
			events.add("message " + text)
			if !AhdWebSocketSend(AhdClassHTTPError, socket, "echo:"+text) {
				events.add("send failed")
			}
		})
		endpoint = AhdWebSocketEndpointWithOpen(AhdClassHTTPError, endpoint, func(socket string, request string) {
			if room := AhdHTTPRequestQuery(AhdClassHTTPError, request, "room"); room != nil && *room == "42" {
				sawRequest.Store(true)
			}
			if !AhdWebSocketIsOpen(socket) || AhdWebSocketID(AhdClassHTTPError, socket) != socket || len(socket) != 22 {
				events.add("bad socket")
			}
			events.add("open " + AhdHTTPRequestPath(request))
		})
		endpoint = AhdWebSocketEndpointWithClose(AhdClassHTTPError, endpoint, func(socket string, code int64, reason string) {
			if AhdWebSocketIsOpen(socket) || AhdWebSocketSend(AhdClassHTTPError, socket, "late") {
				events.add("still open during onClose")
			}
			events.add("close " + strconv.FormatInt(code, 10) + " " + reason)
		})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/echo", endpoint)
	})
	conn := dialWebSocket(t, base+"/echo?room=42", nil)
	writeText(t, conn, "merhaba 😊")
	if got := readText(t, conn); got != "echo:merhaba 😊" {
		t.Fatalf("echo = %q", got)
	}
	writeText(t, conn, "second")
	if got := readText(t, conn); got != "echo:second" {
		t.Fatalf("echo = %q", got)
	}
	if err := conn.Close(websocket.StatusNormalClosure, "bye"); err != nil {
		t.Fatalf("client close: %v", err)
	}
	got := events.waitFor(t, "close")
	want := []string{"open /echo", "message merhaba 😊", "message second", "close 1000 bye"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("events = %q, want %q", got, want)
	}
	if !sawRequest.Load() {
		t.Fatal("onOpen did not receive the upgrade Request")
	}
}

func TestWebSocketHandshakeRefusals(t *testing.T) {
	var connections atomic.Int64
	base := startWebSocketServer(t, func(handle string) {
		plain := AhdHTTPWebSocket(AhdClassHTTPError, func(string, string) {})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/plain", plain)

		guarded := AhdWebSocketEndpointWithAccept(AhdClassHTTPError, plain, func(request string) *string {
			if token := AhdHTTPRequestHeader(AhdClassHTTPError, request, "X-Token"); token != nil && *token == "ok" {
				return nil
			}
			refusal := AhdHTTPText(AhdClassHTTPError, "Unauthorized", 401)
			return &refusal
		})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/guarded", guarded)

		failing := AhdWebSocketEndpointWithAccept(AhdClassHTTPError, plain, func(request string) *string {
			AhdRaiseClass(AhdClassHTTPError, "accept failed")
			return nil
		})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/failing", failing)

		limited := AhdWebSocketEndpointWithMaxConnections(AhdClassHTTPError, plain, 1)
		limited = AhdWebSocketEndpointWithOpen(AhdClassHTTPError, limited, func(string, string) { connections.Add(1) })
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/limited", limited)

		origins := AhdWebSocketEndpointWithAllowedOrigins(AhdClassHTTPError, plain, []string{"https://Example.com", "http://localhost:3000"})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/origins", origins)
	})
	httpBase := "http" + strings.TrimPrefix(base, "ws")

	response, err := http.Get(httpBase + "/plain")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusUpgradeRequired || !strings.EqualFold(response.Header.Get("Upgrade"), "websocket") ||
		response.Header.Get("Sec-WebSocket-Version") != "13" {
		t.Fatalf("plain GET = %d %v", response.StatusCode, response.Header)
	}

	malformed, _ := http.NewRequest(http.MethodGet, httpBase+"/plain", nil)
	malformed.Header.Set("Upgrade", "websocket")
	malformed.Header.Set("Connection", "Upgrade")
	malformed.Header.Set("Sec-WebSocket-Version", "8")
	malformed.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	response, err = http.DefaultClient.Do(malformed)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong version = %d", response.StatusCode)
	}

	response, err = http.Post(httpBase+"/plain", "text/plain", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed || response.Header.Get("Allow") != "GET" {
		t.Fatalf("POST to a WebSocket path = %d Allow=%q", response.StatusCode, response.Header.Get("Allow"))
	}

	if status, body := dialStatus(t, base+"/guarded", nil); status != 401 || body != "Unauthorized" {
		t.Fatalf("refused accept = %d %q", status, body)
	}
	dialWebSocket(t, base+"/guarded", http.Header{"X-Token": []string{"ok"}})

	if status, body := dialStatus(t, base+"/failing", nil); status != 500 || strings.Contains(body, "accept failed") {
		t.Fatalf("raising accept = %d %q", status, body)
	}

	dialWebSocket(t, base+"/limited", nil)
	deadline := time.Now().Add(5 * time.Second)
	for connections.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if status, _ := dialStatus(t, base+"/limited", nil); status != http.StatusServiceUnavailable {
		t.Fatalf("second connection over the limit = %d", status)
	}

	if status, _ := dialStatus(t, base+"/plain", http.Header{"Origin": []string{"https://evil.example"}}); status != http.StatusForbidden {
		t.Fatalf("cross-origin default = %d", status)
	}
	host := strings.TrimPrefix(base, "ws://")
	dialWebSocket(t, base+"/plain", http.Header{"Origin": []string{"http://" + host}})
	dialWebSocket(t, base+"/origins", http.Header{"Origin": []string{"https://example.com"}})
	dialWebSocket(t, base+"/origins", http.Header{"Origin": []string{"http://localhost:3000"}})
	for _, origin := range []string{"http://" + host, "https://example.com:8443", "null", "https://example.com.evil"} {
		if status, _ := dialStatus(t, base+"/origins", http.Header{"Origin": []string{origin}}); status != http.StatusForbidden {
			t.Fatalf("origin %q against an allowlist = %d", origin, status)
		}
	}
}

func TestWebSocketOriginNormalization(t *testing.T) {
	valid := map[string]string{
		"https://Example.COM":      "https://example.com",
		"https://example.com:443":  "https://example.com",
		"http://example.com:80":    "http://example.com",
		"http://localhost:3000":    "http://localhost:3000",
		"http://[::1]:8080":        "http://[::1]:8080",
		"https://admin.ahd.com.tr": "https://admin.ahd.com.tr",
	}
	for input, want := range valid {
		if got, ok := ahdWebSocketNormalizeOrigin(input); !ok || got != want {
			t.Fatalf("normalize(%q) = %q %v, want %q", input, got, ok, want)
		}
	}
	for _, input := range []string{"*", "", "null", "example.com", "ftp://example.com", "https://example.com/",
		"https://example.com/app", "https://example.com?x=1", "https://example.com#top", "https://user@example.com",
		"https://example.com:0", "https://example.com:99999", "https://example.com:", " https://example.com", "ws://example.com"} {
		if got, ok := ahdWebSocketNormalizeOrigin(input); ok {
			t.Fatalf("normalize(%q) accepted as %q", input, got)
		}
	}
}

func TestWebSocketPolicyCloseCodes(t *testing.T) {
	events := &recorder{}
	base := startWebSocketServer(t, func(handle string) {
		endpoint := recordingEndpoint(events, func(socket string, text string) {
			if text == "raise" {
				AhdRaiseClass(AhdClassHTTPError, "handler failed")
			}
			AhdWebSocketSend(AhdClassHTTPError, socket, "ok:"+text)
		})
		endpoint = AhdWebSocketEndpointWithMaxMessageBytes(AhdClassHTTPError, endpoint, 8)
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/policy", endpoint)
	})
	cases := []struct {
		name   string
		kind   websocket.MessageType
		data   []byte
		code   websocket.StatusCode
		reason string
	}{
		{"binary", websocket.MessageBinary, []byte{1, 2, 3}, websocket.StatusUnsupportedData, "binary messages are not supported"},
		{"oversize", websocket.MessageText, []byte("123456789"), websocket.StatusMessageTooBig, "message too large"},
		{"invalid UTF-8", websocket.MessageText, []byte{'a', 0xff}, websocket.StatusInvalidFramePayloadData, "text message is not valid UTF-8"},
		{"handler raises", websocket.MessageText, []byte("raise"), websocket.StatusInternalError, "internal error"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			conn := dialWebSocket(t, base+"/policy", nil)
			writeText(t, conn, "12345678")
			if got := readText(t, conn); got != "ok:12345678" {
				t.Fatalf("message at the limit = %q", got)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := conn.Write(ctx, testCase.kind, testCase.data); err != nil {
				t.Fatalf("write: %v", err)
			}
			code, reason := readClose(t, conn)
			if code != testCase.code || reason != testCase.reason {
				t.Fatalf("close = %d %q, want %d %q", code, reason, testCase.code, testCase.reason)
			}
		})
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		closes := 0
		for _, event := range events.snapshot() {
			if strings.HasPrefix(event, "close ") {
				closes++
			}
		}
		if closes == len(cases) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	var closes []string
	for _, event := range events.snapshot() {
		if strings.HasPrefix(event, "close ") {
			closes = append(closes, event)
		}
	}
	// onClose is ordered per socket, not across sockets: each connection
	// finishes its own close handshake, so these four may report in any order.
	want := []string{"close 1003 binary messages are not supported", "close 1009 message too large",
		"close 1007 text message is not valid UTF-8", "close 1011 internal error"}
	slices.Sort(closes)
	slices.Sort(want)
	if !slices.Equal(closes, want) {
		t.Fatalf("onClose events = %q, want %q in any order", closes, want)
	}
}

func TestWebSocketServerCloseFlushesQueuedMessages(t *testing.T) {
	events := &recorder{}
	base := startWebSocketServer(t, func(handle string) {
		endpoint := recordingEndpoint(events, func(socket string, text string) {
			AhdWebSocketSend(AhdClassHTTPError, socket, "first")
			AhdWebSocketSend(AhdClassHTTPError, socket, "second")
			AhdWebSocketClose(AhdClassHTTPError, socket, 4001, "done")
			if AhdWebSocketSend(AhdClassHTTPError, socket, "after close") {
				events.add("send after close succeeded")
			}
			if AhdWebSocketIsOpen(socket) {
				events.add("open after close")
			}
			AhdWebSocketClose(AhdClassHTTPError, socket, 1000, "ignored second close")
		})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/close", endpoint)
	})
	conn := dialWebSocket(t, base+"/close", nil)
	writeText(t, conn, "go")
	if first, second := readText(t, conn), readText(t, conn); first != "first" || second != "second" {
		t.Fatalf("queued messages = %q %q", first, second)
	}
	if code, reason := readClose(t, conn); code != 4001 || reason != "done" {
		t.Fatalf("close = %d %q", code, reason)
	}
	got := events.waitFor(t, "close")
	if strings.Join(got, "|") != "open /close|close 4001 done" {
		t.Fatalf("events = %q", got)
	}
}

func TestWebSocketQueueOverflowClosesWith1008(t *testing.T) {
	previous := ahdWebSocketWriteTimeout
	ahdWebSocketWriteTimeout = 500 * time.Millisecond
	t.Cleanup(func() { ahdWebSocketWriteTimeout = previous })
	events := &recorder{}
	payload := strings.Repeat("x", 60000)
	base := startWebSocketServer(t, func(handle string) {
		endpoint := recordingEndpoint(events, func(socket string, text string) {
			for index := 0; index < 100000; index++ {
				if !AhdWebSocketSend(AhdClassHTTPError, socket, payload) {
					events.add("send refused")
					return
				}
			}
		})
		endpoint = AhdWebSocketEndpointWithMaxQueuedMessages(AhdClassHTTPError, endpoint, 1)
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/flood", endpoint)
	})
	conn := dialWebSocket(t, base+"/flood", nil)
	writeText(t, conn, "flood me")
	// The client never reads, so the server's writes stall and its queue fills.
	got := events.waitFor(t, "close")
	if strings.Join(got, "|") != "open /flood|send refused|close 1008 outbound queue full" {
		t.Fatalf("events = %q", got)
	}
}

func TestWebSocketSurvivesHTTPServerTimeouts(t *testing.T) {
	base := startWebSocketServer(t, func(handle string) {
		AhdHTTPServerSetLimits(AhdClassHTTPError, handle, ahdHTTPDefaultMaxBody, 0, 0, 200*time.Millisecond, 200*time.Millisecond, 200*time.Millisecond)
		endpoint := AhdHTTPWebSocket(AhdClassHTTPError, func(socket string, text string) {
			AhdWebSocketSend(AhdClassHTTPError, socket, "still here: "+text)
		})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/slow", endpoint)
	})
	conn := dialWebSocket(t, base+"/slow", nil)
	time.Sleep(900 * time.Millisecond)
	writeText(t, conn, "ping")
	if got := readText(t, conn); got != "still here: ping" {
		t.Fatalf("after the HTTP timeouts = %q", got)
	}
}

func TestWebSocketUnresponsivePeerIsClosedAsAbnormal(t *testing.T) {
	previousInterval, previousTimeout := ahdWebSocketPingInterval, ahdWebSocketPongTimeout
	ahdWebSocketPingInterval, ahdWebSocketPongTimeout = 100*time.Millisecond, 150*time.Millisecond
	t.Cleanup(func() { ahdWebSocketPingInterval, ahdWebSocketPongTimeout = previousInterval, previousTimeout })
	events := &recorder{}
	base := startWebSocketServer(t, func(handle string) {
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/quiet", recordingEndpoint(events, func(string, string) {}))
	})
	// A client that never reads never answers a ping.
	dialWebSocket(t, base+"/quiet", nil)
	got := events.waitFor(t, "close")
	if strings.Join(got, "|") != "open /quiet|close 1006 " {
		t.Fatalf("events = %q", got)
	}
}

func TestWebSocketCallbacksNeverOverlapWithHandlers(t *testing.T) {
	const clients, messages = 12, 40
	var inside atomic.Int32
	var overlaps, delivered atomic.Int64
	registry := map[string]bool{}
	enter := func() {
		if inside.Add(1) != 1 {
			overlaps.Add(1)
		}
		time.Sleep(50 * time.Microsecond)
		inside.Add(-1)
	}
	base := startWebSocketServer(t, func(handle string) {
		endpoint := AhdHTTPWebSocket(AhdClassHTTPError, func(socket string, text string) {
			enter()
			delivered.Add(1)
		})
		endpoint = AhdWebSocketEndpointWithOpen(AhdClassHTTPError, endpoint, func(socket string, request string) {
			enter()
			registry[socket] = true
		})
		endpoint = AhdWebSocketEndpointWithClose(AhdClassHTTPError, endpoint, func(socket string, code int64, reason string) {
			enter()
			delete(registry, socket)
		})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/live", endpoint)
		AhdHTTPServerPost(AhdClassHTTPError, handle, "/broadcast", func(data string) string {
			enter()
			for socket := range registry {
				AhdWebSocketSend(AhdClassHTTPError, socket, "news")
			}
			return AhdHTTPText(AhdClassHTTPError, "sent", 200)
		})
	})
	httpBase := "http" + strings.TrimPrefix(base, "ws")
	var group sync.WaitGroup
	for client := 0; client < clients; client++ {
		group.Add(1)
		go func() {
			defer group.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			conn, _, err := websocket.Dial(ctx, base+"/live", nil)
			if err != nil {
				t.Errorf("dial: %v", err)
				return
			}
			defer conn.CloseNow()
			go func() {
				for {
					if _, _, err := conn.Read(ctx); err != nil {
						return
					}
				}
			}()
			for index := 0; index < messages; index++ {
				if err := conn.Write(ctx, websocket.MessageText, []byte("m")); err != nil {
					t.Errorf("write: %v", err)
					return
				}
			}
			_ = conn.Close(websocket.StatusNormalClosure, "")
		}()
	}
	for index := 0; index < 30; index++ {
		response, err := http.Post(httpBase+"/broadcast", "text/plain", nil)
		if err == nil {
			_ = response.Body.Close()
		}
	}
	group.Wait()
	deadline := time.Now().Add(15 * time.Second)
	for delivered.Load() < clients*messages && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if delivered.Load() != clients*messages {
		t.Fatalf("delivered %d of %d messages", delivered.Load(), clients*messages)
	}
	if overlaps.Load() != 0 {
		t.Fatalf("%d callbacks overlapped another callback or handler", overlaps.Load())
	}
}

func TestWebSocketRegistrationRules(t *testing.T) {
	endpoint := AhdHTTPWebSocket(AhdClassHTTPError, func(string, string) {})
	handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(freeLoopbackPort(t)), ahdHTTPDefaultMaxBody)
	AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/events", endpoint)
	AhdHTTPServerPost(AhdClassHTTPError, handle, "/events", func(string) string { return AhdHTTPText(AhdClassHTTPError, "", 200) })
	AhdHTTPServerGet(AhdClassHTTPError, handle, "/page", func(string) string { return AhdHTTPText(AhdClassHTTPError, "", 200) })
	for want, register := range map[string]func(){
		"HTTP WebSocket route /events is already registered": func() {
			AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/events", endpoint)
		},
		"HTTP route GET /events is already registered as a WebSocket endpoint": func() {
			AhdHTTPServerGet(AhdClassHTTPError, handle, "/events", func(string) string { return "" })
		},
		"HTTP route GET /page is already registered": func() {
			AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/page", endpoint)
		},
		`HTTP route path "events" must begin with / and must not contain ? or #`: func() {
			AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "events", endpoint)
		},
		"WebSocketEndpoint storage is corrupted": func() {
			AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/other", "missing")
		},
		"WebSocketEndpoint maxMessageBytes must be in 1..16777216": func() {
			AhdWebSocketEndpointWithMaxMessageBytes(AhdClassHTTPError, endpoint, 0)
		},
		"WebSocketEndpoint maxQueuedMessages must be in 1..4096": func() {
			AhdWebSocketEndpointWithMaxQueuedMessages(AhdClassHTTPError, endpoint, 4097)
		},
		"WebSocketEndpoint maxConnections must be in 1..1000000": func() {
			AhdWebSocketEndpointWithMaxConnections(AhdClassHTTPError, endpoint, -1)
		},
		"WebSocketEndpoint.withAllowedOrigins needs at least one origin": func() {
			AhdWebSocketEndpointWithAllowedOrigins(AhdClassHTTPError, endpoint, nil)
		},
		`WebSocketEndpoint origin "*" must be exactly scheme://host or scheme://host:port with http or https`: func() {
			AhdWebSocketEndpointWithAllowedOrigins(AhdClassHTTPError, endpoint, []string{"*"})
		},
		"WebSocket close code must be 1000, 1001, 1008, 1011, or 3000..4999": func() {
			AhdWebSocketClose(AhdClassHTTPError, "missing", 1006, "")
		},
		"WebSocket close reason must be at most 123 bytes of UTF-8": func() {
			AhdWebSocketClose(AhdClassHTTPError, "missing", 1000, strings.Repeat("ş", 62))
		},
	} {
		if got := codesRaised(t, register); got != want {
			t.Fatalf("message %q, want %q", got, want)
		}
	}
	// Builders never change the endpoint they were called on.
	if AhdWebSocketEndpointWithMaxConnections(AhdClassHTTPError, endpoint, 5) == endpoint {
		t.Fatal("a builder returned the same endpoint handle")
	}
	if ahdWebSocketLookupEndpoint(AhdClassHTTPError, endpoint).maxConnections != ahdWebSocketDefaultMaxConnections {
		t.Fatal("a builder changed the original endpoint")
	}
	// An unknown or closed socket is simply not open.
	if AhdWebSocketSend(AhdClassHTTPError, "missing", "x") || AhdWebSocketIsOpen("missing") {
		t.Fatal("an unknown socket reports open")
	}
	AhdWebSocketClose(AhdClassHTTPError, "missing", 1000, "")
}

func TestWebSocketWildcardAndMethodRouting(t *testing.T) {
	base := startWebSocketServer(t, func(handle string) {
		endpoint := AhdHTTPWebSocket(AhdClassHTTPError, func(socket string, text string) {
			AhdWebSocketSend(AhdClassHTTPError, socket, "room:"+text)
		})
		endpoint = AhdWebSocketEndpointWithOpen(AhdClassHTTPError, endpoint, func(socket string, request string) {
			AhdWebSocketSend(AhdClassHTTPError, socket, AhdHTTPRequestPath(request))
		})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/rooms/*", endpoint)
		AhdHTTPServerPost(AhdClassHTTPError, handle, "/rooms/1", func(string) string { return AhdHTTPText(AhdClassHTTPError, "posted", 200) })
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/rooms/lobby", func(string) string { return AhdHTTPText(AhdClassHTTPError, "lobby page", 200) })
	})
	httpBase := "http" + strings.TrimPrefix(base, "ws")
	conn := dialWebSocket(t, base+"/rooms/1", nil)
	if got := readText(t, conn); got != "/rooms/1" {
		t.Fatalf("onOpen path = %q", got)
	}
	response, err := http.Post(httpBase+"/rooms/1", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "posted" {
		t.Fatalf("POST beside a WebSocket = %q", body)
	}
	request, _ := http.NewRequest(http.MethodPut, httpBase+"/rooms/1", nil)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed || response.Header.Get("Allow") != "POST, GET" {
		t.Fatalf("PUT = %d Allow=%q", response.StatusCode, response.Header.Get("Allow"))
	}
	// An exact GET route wins over the wildcard endpoint.
	response, err = http.Get(httpBase + "/rooms/lobby")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "lobby page" {
		t.Fatalf("exact GET = %d %q", response.StatusCode, body)
	}
}
