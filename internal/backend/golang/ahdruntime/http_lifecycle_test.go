package ahdruntime

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestHTTPEndpointMatchIsExact(t *testing.T) {
	same := [][2]string{
		{"127.0.0.1", "127.0.0.1"},
		{"LOCALHOST", "localhost"},
		{"[::1]", "::1"},
		{"0.0.0.0", "0.0.0.0"},
		{"::", "::"},
	}
	for _, pair := range same {
		if !ahdHTTPSameEndpoint(pair[0], pair[1]) {
			t.Fatalf("%q and %q should be the same endpoint", pair[0], pair[1])
		}
	}
	different := [][2]string{
		{"127.0.0.1", "localhost"},
		{"127.0.0.1", "::1"},
		{"127.0.0.1", "0.0.0.0"},
		{"localhost", "::1"},
		{"0.0.0.0", "::"},
		{"::1", "::"},
	}
	for _, pair := range different {
		if ahdHTTPSameEndpoint(pair[0], pair[1]) {
			t.Fatalf("%q and %q must not select the same server", pair[0], pair[1])
		}
	}
}

func TestHTTPShutdownClosesTheListenerBeforeServe(t *testing.T) {
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	previous := ahdHTTPTestBeforeServe
	published := make(chan struct{})
	release := make(chan struct{})
	ahdHTTPTestBeforeServe = func() {
		close(published)
		<-release
	}
	t.Cleanup(func() {
		ahdHTTPTestBeforeServe = previous
		select {
		case <-release:
		default:
			close(release)
		}
		ahdHTTPTestShutdown(handle)
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		AhdHTTPServerStart(AhdClassHTTPError, handle)
	}()
	select {
	case <-published:
	case <-time.After(2 * time.Second):
		t.Fatal("listener was not published")
	}
	if !AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("shutdown did not claim the published listener")
	}
	// The port has to be free before Serve runs. Shutdown alone does not
	// close a listener that Serve has not registered.
	assertPortReusable(t, port)
	if AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("repeated shutdown reported an active server")
	}
	ahdHTTPTestBeforeServe = nil

	replacement := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	AhdHTTPServerGet(AhdClassHTTPError, replacement, "/", func(string) string {
		return AhdHTTPText(AhdClassHTTPError, "replacement", 200)
	})
	replacementDone := make(chan struct{})
	go func() {
		defer close(replacementDone)
		defer func() { _ = recover() }()
		AhdHTTPServerStart(AhdClassHTTPError, replacement)
	}()
	t.Cleanup(func() {
		ahdHTTPTestShutdown(replacement)
		select {
		case <-replacementDone:
		case <-time.After(3 * time.Second):
		}
	})
	waitForStatus(t, "http://127.0.0.1:"+strconv.Itoa(port)+"/", http.StatusOK)
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the original Server.start did not return after the barrier")
	}
	waitForStatus(t, "http://127.0.0.1:"+strconv.Itoa(port)+"/", http.StatusOK)
	if ahdHTTPHandleRegistered(handle) {
		t.Fatal("the shut-down server stayed registered")
	}
	if !ahdHTTPHandleRegistered(replacement) {
		t.Fatal("shutdown closed the replacement server that reused the address")
	}
}

func TestHTTPPublicationPanicReleasesTheListener(t *testing.T) {
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	previous := ahdHTTPTestBeforeServe
	ahdHTTPTestBeforeServe = func() { panic("publication failed") }
	t.Cleanup(func() { ahdHTTPTestBeforeServe = previous })
	func() {
		defer func() { _ = recover() }()
		AhdHTTPServerStart(AhdClassHTTPError, handle)
	}()
	if ahdHTTPHandleRegistered(handle) {
		t.Fatal("a panicked start left the server registered")
	}
	assertPortReusable(t, port)
}

func TestHTTPShutdownClaimsStartedServerBeforeListen(t *testing.T) {
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	ahdHTTPServersMu.Lock()
	server := ahdHTTPServers[handle]
	ahdHTTPServersMu.Unlock()
	if server == nil {
		t.Fatal("missing server state")
	}
	// Exercise the real startup transition without advancing to bind yet.
	httpServer := server.claimStart(AhdClassHTTPError, 0)
	if !AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("shutdown missed started server before listen")
	}
	if AhdHTTPServerActive() {
		t.Fatal("claimed server stayed active after pre-listen shutdown")
	}
	if ahdHTTPHandleRegistered(handle) {
		t.Fatal("claimed server stayed registered after pre-listen shutdown")
	}
	if AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("repeated pre-listen shutdown returned true")
	}
	server.serve(AhdClassHTTPError, httpServer, func(string, string) (net.Listener, error) {
		t.Fatal("cancelled start reached the bind step")
		return nil, nil
	})
	assertPortReusable(t, port)
}

func TestHTTPShutdownStartsWhileAHandlerIsBlocked(t *testing.T) {
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseHandler := func() { releaseOnce.Do(func() { close(release) }) }
	// The handler stays blocked longer than the graceful deadline. Shutdown
	// has to return on that deadline, not when this timer fires.
	time.AfterFunc(4500*time.Millisecond, releaseHandler)
	entered := make(chan struct{})
	finished := make(chan struct{})
	AhdHTTPServerGet(AhdClassHTTPError, handle, "/slow", func(string) string {
		close(entered)
		<-release
		close(finished)
		return AhdHTTPText(AhdClassHTTPError, "late", 200)
	})
	AhdHTTPServerGet(AhdClassHTTPError, handle, "/ping", func(string) string {
		return AhdHTTPText(AhdClassHTTPError, "pong", 200)
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		AhdHTTPServerStart(AhdClassHTTPError, handle)
	}()
	t.Cleanup(func() {
		releaseHandler()
		ahdHTTPTestShutdown(handle)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	})
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	waitForStatus(t, base+"/ping", http.StatusOK)

	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		response, err := client.Get(base + "/slow")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("slow handler did not start")
	}

	shutdownDone := make(chan bool, 1)
	started := time.Now()
	go func() {
		shutdownDone <- AhdHTTPShutdownAddress("127.0.0.1", port)
	}()
	deadline := time.Now().Add(time.Second)
	refused := false
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 50*time.Millisecond)
		if err != nil {
			refused = true
			break
		}
		_ = conn.Close()
		time.Sleep(10 * time.Millisecond)
	}
	if !refused {
		t.Fatal("a new connection was accepted while shutdown had been requested and a handler was blocked")
	}
	if AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("a concurrent shutdown claimed a listener that was already claimed")
	}
	var stopped bool
	select {
	case stopped = <-shutdownDone:
	case <-time.After(4 * time.Second):
		t.Fatal("shutdown did not return within the forced-close bound")
	}
	elapsed := time.Since(started)
	if !stopped {
		t.Fatal("shutdown did not stop the listening server")
	}
	if elapsed < 1500*time.Millisecond || elapsed > 3500*time.Millisecond {
		t.Fatalf("shutdown took %s; graceful shutdown is 2s, then Close", elapsed)
	}
	select {
	case <-finished:
		t.Fatal("Close ended the blocked handler goroutine")
	default:
	}
	select {
	case <-finished:
	case <-time.After(4 * time.Second):
		t.Fatal("the handler did not finish after its own wait ended")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Server.start did not return after Close")
	}
	select {
	case <-requestDone:
	case <-time.After(2 * time.Second):
		t.Fatal("the blocked request did not finish")
	}
	if AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("repeated shutdown reported an active server")
	}
	if ahdHTTPHandleRegistered(handle) {
		t.Fatal("stopped server stayed registered")
	}
	assertPortReusable(t, port)
}

func TestHTTPRegistryDropsServersAcrossRestarts(t *testing.T) {
	port := freeLoopbackPort(t)
	before := ahdHTTPRegistryCount()
	for cycle := 1; cycle <= 3; cycle++ {
		handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "cycle", 200)
		})
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() { _ = recover() }()
			AhdHTTPServerStart(AhdClassHTTPError, handle)
		}()
		base := "http://127.0.0.1:" + strconv.Itoa(port)
		waitForStatus(t, base+"/", http.StatusOK)
		if !AhdHTTPShutdownAddress("127.0.0.1", port) {
			t.Fatalf("cycle %d did not shut down", cycle)
		}
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("cycle %d start did not return", cycle)
		}
		if AhdHTTPShutdownAddress("127.0.0.1", port) {
			t.Fatalf("cycle %d repeated shutdown reported an active server", cycle)
		}
		if ahdHTTPHandleRegistered(handle) {
			t.Fatalf("cycle %d left the server registered", cycle)
		}
		assertPortReusable(t, port)
	}
	if got := ahdHTTPRegistryCount(); got != before {
		t.Fatalf("registry grew from %d to %d across three start/stop cycles", before, got)
	}
}

func TestHTTPShutdownDoesNotStopADifferentEndpoint(t *testing.T) {
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string {
		return AhdHTTPText(AhdClassHTTPError, "loopback", 200)
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		AhdHTTPServerStart(AhdClassHTTPError, handle)
	}()
	t.Cleanup(func() {
		ahdHTTPTestShutdown(handle)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	})
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	waitForStatus(t, base+"/", http.StatusOK)
	if AhdHTTPShutdownAddress("localhost", port) {
		t.Fatal("localhost shutdown stopped a 127.0.0.1 server")
	}
	waitForStatus(t, base+"/", http.StatusOK)
	if AhdHTTPShutdownAddress("::1", port) {
		t.Fatal("IPv6 loopback shutdown stopped a 127.0.0.1 server")
	}
	if AhdHTTPShutdownAddress("0.0.0.0", port) {
		t.Fatal("wildcard shutdown stopped a 127.0.0.1 server")
	}
	if !AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("the matching address did not shut down")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("start did not return")
	}
}

func TestHTTPWildcardShutdownMatchesOnlyTheWildcard(t *testing.T) {
	port := freeLoopbackPort(t)
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Skip("wildcard bind is not available")
	}
	port = listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	handle := AhdHTTPServer(AhdClassHTTPError, "0.0.0.0", int64(port), ahdHTTPDefaultMaxBody)
	AhdHTTPServerGet(AhdClassHTTPError, handle, "/", func(string) string {
		return AhdHTTPText(AhdClassHTTPError, "wild", 200)
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		AhdHTTPServerStart(AhdClassHTTPError, handle)
	}()
	t.Cleanup(func() {
		ahdHTTPTestShutdown(handle)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	})
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	waitForStatus(t, base+"/", http.StatusOK)
	if AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("127.0.0.1 shutdown stopped a 0.0.0.0 server")
	}
	waitForStatus(t, base+"/", http.StatusOK)
	if !AhdHTTPShutdownAddress("0.0.0.0", port) {
		t.Fatal("the wildcard server did not shut down")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("wildcard start did not return")
	}
}

func TestHTTPNestedStartShutsDownOuterWhileInnerStaysUp(t *testing.T) {
	outerPort := freeLoopbackPort(t)
	innerPort := freeLoopbackPort(t)
	outer := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(outerPort), ahdHTTPDefaultMaxBody)
	inner := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(innerPort), ahdHTTPDefaultMaxBody)
	AhdHTTPServerGet(AhdClassHTTPError, inner, "/", func(string) string {
		return AhdHTTPText(AhdClassHTTPError, "inner", 200)
	})
	innerServing := make(chan struct{})
	AhdHTTPServerGet(AhdClassHTTPError, outer, "/nest", func(string) string {
		close(innerServing)
		AhdHTTPServerStart(AhdClassHTTPError, inner)
		return AhdHTTPText(AhdClassHTTPError, "outer-returned", 200)
	})
	AhdHTTPServerGet(AhdClassHTTPError, outer, "/ping", func(string) string {
		return AhdHTTPText(AhdClassHTTPError, "pong", 200)
	})
	outerDone := make(chan struct{})
	go func() {
		defer close(outerDone)
		defer func() { _ = recover() }()
		AhdHTTPServerStart(AhdClassHTTPError, outer)
	}()
	t.Cleanup(func() {
		ahdHTTPTestShutdown(inner)
		ahdHTTPTestShutdown(outer)
	})
	waitForStatus(t, "http://127.0.0.1:"+strconv.Itoa(outerPort)+"/ping", http.StatusOK)
	go func() {
		client := &http.Client{Timeout: 8 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
		response, err := client.Get("http://127.0.0.1:" + strconv.Itoa(outerPort) + "/nest")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}()
	select {
	case <-innerServing:
	case <-time.After(2 * time.Second):
		t.Fatal("nested start did not begin")
	}
	waitForStatus(t, "http://127.0.0.1:"+strconv.Itoa(innerPort)+"/", http.StatusOK)

	started := time.Now()
	if !AhdHTTPShutdownAddress("127.0.0.1", outerPort) {
		t.Fatal("outer shutdown failed while its handler was blocked in the inner server")
	}
	if elapsed := time.Since(started); elapsed > 4*time.Second {
		t.Fatalf("outer shutdown took %s while a handler was blocked", elapsed)
	}
	waitForStatus(t, "http://127.0.0.1:"+strconv.Itoa(innerPort)+"/", http.StatusOK)
	if !AhdHTTPShutdownAddress("127.0.0.1", innerPort) {
		t.Fatal("inner shutdown failed")
	}
	select {
	case <-outerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("outer Server.start did not return")
	}
	if AhdHTTPShutdownAddress("127.0.0.1", outerPort) || AhdHTTPShutdownAddress("127.0.0.1", innerPort) {
		t.Fatal("repeated shutdown reported an active server")
	}
}

func TestHTTPNestedStartInnerShutdownReleasesTheOuterHandler(t *testing.T) {
	outerPort := freeLoopbackPort(t)
	innerPort := freeLoopbackPort(t)
	outer := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(outerPort), ahdHTTPDefaultMaxBody)
	inner := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(innerPort), ahdHTTPDefaultMaxBody)
	AhdHTTPServerGet(AhdClassHTTPError, inner, "/", func(string) string {
		return AhdHTTPText(AhdClassHTTPError, "inner", 200)
	})
	innerServing := make(chan struct{})
	AhdHTTPServerGet(AhdClassHTTPError, outer, "/nest", func(string) string {
		close(innerServing)
		AhdHTTPServerStart(AhdClassHTTPError, inner)
		return AhdHTTPText(AhdClassHTTPError, "outer-returned", 200)
	})
	outerDone := make(chan struct{})
	go func() {
		defer close(outerDone)
		defer func() { _ = recover() }()
		AhdHTTPServerStart(AhdClassHTTPError, outer)
	}()
	t.Cleanup(func() {
		ahdHTTPTestShutdown(inner)
		ahdHTTPTestShutdown(outer)
		select {
		case <-outerDone:
		case <-time.After(3 * time.Second):
		}
	})
	waitForTCP(t, outerPort)
	body := make(chan string, 1)
	go func() {
		client := &http.Client{Timeout: 8 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
		response, err := client.Get("http://127.0.0.1:" + strconv.Itoa(outerPort) + "/nest")
		if err != nil {
			body <- "error:" + err.Error()
			return
		}
		payload, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil || response.StatusCode != http.StatusOK {
			body <- "bad"
			return
		}
		body <- string(payload)
	}()
	select {
	case <-innerServing:
	case <-time.After(2 * time.Second):
		t.Fatal("nested start did not begin")
	}
	waitForStatus(t, "http://127.0.0.1:"+strconv.Itoa(innerPort)+"/", http.StatusOK)
	if !AhdHTTPShutdownAddress("127.0.0.1", innerPort) {
		t.Fatal("inner shutdown failed")
	}
	select {
	case got := <-body:
		if got != "outer-returned" {
			t.Fatalf("outer handler after inner shutdown = %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("inner shutdown did not release the outer handler")
	}
	started := time.Now()
	if !AhdHTTPShutdownAddress("127.0.0.1", outerPort) {
		t.Fatal("outer shutdown failed after the handler returned")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("outer shutdown took %s after the nested handler had returned", elapsed)
	}
	select {
	case <-outerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("outer Server.start did not return")
	}
}

func TestHTTPSameServerReentryIsRejected(t *testing.T) {
	var serverHandle string
	base, serverHandle := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		serverHandle = handle
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/again", func(string) string {
			AhdHTTPServerStart(AhdClassHTTPError, handle)
			return AhdHTTPText(AhdClassHTTPError, "started-again", 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/add", func(string) string {
			AhdHTTPServerGet(AhdClassHTTPError, handle, "/extra", func(string) string {
				return AhdHTTPText(AhdClassHTTPError, "extra", 200)
			})
			return AhdHTTPText(AhdClassHTTPError, "added", 200)
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/ping", func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "pong", 200)
		})
	})
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	again, err := client.Get(base + "/again")
	if err != nil {
		t.Fatalf("same-server start from a handler blocked the request: %v", err)
	}
	_, _ = io.Copy(io.Discard, again.Body)
	_ = again.Body.Close()
	if again.StatusCode != http.StatusInternalServerError {
		t.Fatalf("re-entrant start status = %d", again.StatusCode)
	}
	add, err := client.Get(base + "/add")
	if err != nil {
		t.Fatalf("same-server route change from a handler blocked the request: %v", err)
	}
	_, _ = io.Copy(io.Discard, add.Body)
	_ = add.Body.Close()
	if add.StatusCode != http.StatusInternalServerError {
		t.Fatalf("re-entrant route status = %d", add.StatusCode)
	}
	extra, err := client.Get(base + "/extra")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, extra.Body)
	_ = extra.Body.Close()
	if extra.StatusCode != http.StatusNotFound {
		t.Fatalf("re-entrant route was stored, status = %d", extra.StatusCode)
	}
	ping, err := client.Get(base + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(ping.Body)
	_ = ping.Body.Close()
	if ping.StatusCode != http.StatusOK || string(body) != "pong" {
		t.Fatalf("server after rejected re-entry = %d %q", ping.StatusCode, body)
	}
	if !ahdHTTPHandleRegistered(serverHandle) {
		t.Fatal("rejected re-entry removed the running server")
	}
}

func ahdHTTPHandleRegistered(handle string) bool {
	ahdHTTPServersMu.Lock()
	defer ahdHTTPServersMu.Unlock()
	_, ok := ahdHTTPServers[handle]
	return ok
}

func ahdHTTPRegistryCount() int {
	ahdHTTPServersMu.Lock()
	defer ahdHTTPServersMu.Unlock()
	return len(ahdHTTPServers)
}

func assertPortReusable(t *testing.T, port int) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("port %d was not released: %v", port, err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
}

func waitForStatus(t *testing.T, url string, status int) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond, Transport: &http.Transport{DisableKeepAlives: true}}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == status {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s did not return %d before the deadline", url, status)
}

func waitForTCP(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(port), 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("port %d did not accept a connection", port)
}
