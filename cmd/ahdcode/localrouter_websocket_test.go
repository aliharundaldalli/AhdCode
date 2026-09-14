package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"ahdcode/internal/localdev"
)

// The local .test router forwards a WebSocket upgrade like any other request
// and keeps the Host that was typed, so an application's same-origin check
// works at its .test name exactly as it does on the loopback port.
func TestRouterForwardsWebSocketUpgrades(t *testing.T) {
	seenHost := make(chan string, 1)
	host, port := backend(t, func(writer http.ResponseWriter, request *http.Request) {
		select {
		case seenHost <- request.Host:
		default:
		}
		conn, err := websocket.Accept(writer, request, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.CloseNow()
		ctx := context.Background()
		for {
			kind, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			if err := conn.Write(ctx, kind, append([]byte("echo "), data...)); err != nil {
				return
			}
		}
	})
	router := routerServing(localdev.Route{
		Hostname: "ahdakademi.test", Kind: localdev.KindDev,
		BindHost: host, BindPort: port,
	})
	front := httptest.NewServer(router)
	t.Cleanup(front.Close)
	frontAddress := strings.TrimPrefix(front.URL, "http://")
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, frontAddress)
		},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws://ahdakademi.test/live", &websocket.DialOptions{HTTPClient: client})
	if err != nil {
		t.Fatalf("the upgrade did not pass through the router: %v", err)
	}
	defer conn.CloseNow()
	if err := conn.Write(ctx, websocket.MessageText, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	_, data, err := conn.Read(ctx)
	if err != nil || string(data) != "echo hi" {
		t.Fatalf("message through the router = %q, %v", data, err)
	}
	if got := <-seenHost; got != "ahdakademi.test" {
		t.Fatalf("the application saw Host %q", got)
	}
}
