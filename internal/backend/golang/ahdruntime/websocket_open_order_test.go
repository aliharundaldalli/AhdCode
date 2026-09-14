package ahdruntime

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// When the client sees the connection open, onOpen has already returned. An
// application that registers the socket in onOpen therefore cannot miss a
// message another handler sends right after the client connects. The v1.4.0
// dogfood application found the opposite order losing such a broadcast.
func TestWebSocketOnOpenCompletesBeforeTheClientSeesOpen(t *testing.T) {
	var registered atomic.Bool
	base := startWebSocketServer(t, func(handle string) {
		endpoint := AhdHTTPWebSocket(AhdClassHTTPError, func(socket string, text string) {})
		endpoint = AhdWebSocketEndpointWithOpen(AhdClassHTTPError, endpoint, func(socket string, request string) {
			time.Sleep(150 * time.Millisecond)
			if !AhdWebSocketSend(AhdClassHTTPError, socket, "sent from onOpen") {
				t.Error("send from onOpen was refused")
			}
			registered.Store(true)
		})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/slow", endpoint)
	})
	for round := 0; round < 3; round++ {
		registered.Store(false)
		conn := dialWebSocket(t, base+"/slow", nil)
		if !registered.Load() {
			t.Fatal("the client saw the connection open before onOpen returned")
		}
		if got := readText(t, conn); got != "sent from onOpen" {
			t.Fatalf("first message = %q", got)
		}
		_ = conn.CloseNow()
	}
}

// A client that disappears while onOpen is still running gets onClose exactly
// once, whether the upgrade itself fails or the connection is found lost.
func TestWebSocketPeerLostDuringOnOpenStillClosesOnce(t *testing.T) {
	events := &recorder{}
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	base := startWebSocketServer(t, func(handle string) {
		endpoint := AhdHTTPWebSocket(AhdClassHTTPError, func(socket string, text string) {})
		endpoint = AhdWebSocketEndpointWithOpen(AhdClassHTTPError, endpoint, func(socket string, request string) {
			events.add("open")
			<-release
		})
		endpoint = AhdWebSocketEndpointWithClose(AhdClassHTTPError, endpoint, func(socket string, code int64, reason string) {
			events.add("close " + strconv.FormatInt(code, 10))
		})
		AhdHTTPServerWebSocket(AhdClassHTTPError, handle, "/vanish", endpoint)
	})
	host := strings.TrimPrefix(base, "ws://")
	raw, err := net.DialTimeout("tcp", host, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = raw.Write([]byte("GET /vanish HTTP/1.1\r\nHost: " + host + "\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	events.waitFor(t, "open")
	_ = raw.Close()
	unblock()
	events.waitFor(t, "close")
	time.Sleep(100 * time.Millisecond)
	if got := strings.Join(events.snapshot(), "|"); got != "open|close 1006" {
		t.Fatalf("events = %q, want open|close 1006", got)
	}
}
