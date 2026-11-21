package ahdruntime

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func installHTTPSFixture(t *testing.T, names []string) *x509.CertPool {
	t.Helper()
	ca, pair := smtpTestCert(t, names, []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback})
	root, err := os.ReadFile(ca)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(root) {
		t.Fatal("invalid test CA")
	}
	key, err := x509.MarshalPKCS8PrivateKey(pair.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	certPath, keyPath := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	var chain []byte
	for _, cert := range pair.Certificate {
		chain = append(chain, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})...)
	}
	if err := os.WriteFile(certPath, chain, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0o600); err != nil {
		t.Fatal(err)
	}
	previousCert, previousKey := ahdDevTLSCert, ahdDevTLSKey
	ahdDevTLSCert, ahdDevTLSKey = certPath, keyPath
	t.Cleanup(func() { ahdDevTLSCert, ahdDevTLSKey = previousCert, previousKey })
	return pool
}

func TestDevHTTPSStartupRollsBackFailedClaims(t *testing.T) {
	installHTTPSFixture(t, []string{"wrong.test"})
	class := AhdClassHTTPError
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port
	handle := AhdHTTPServer(class, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	state := ahdHTTPLookup(class, handle)
	t.Cleanup(state.retire)
	assertFailure := func(handle, message string) {
		t.Helper()
		state := ahdHTTPLookup(class, handle)
		defer func() {
			signal, ok := recover().(*AhdSignal)
			if !ok || !strings.Contains(signal.Message, message) {
				t.Errorf("expected %q, got %v", message, signal)
			}
			if state.owner.Load() || state.closing.Load() || state.routesFrozen.Load() || !ahdHTTPHandleRegistered(handle) {
				t.Error("failed TLS start did not leave a reusable handle")
			}
		}()
		AhdHTTPServerStart(class, handle)
	}
	assertFailure(handle, "must cover localhost")
	installHTTPSFixture(t, []string{"localhost"})
	assertFailure(handle, "could not bind")
	unsafe := AhdHTTPServer(class, "0.0.0.0", int64(port), ahdHTTPDefaultMaxBody)
	t.Cleanup(ahdHTTPLookup(class, unsafe).retire)
	assertFailure(unsafe, "can bind only localhost")
}

func TestDevHTTPSWebSocketShutdownRetainsScopedSessions(t *testing.T) {
	pool := installHTTPSFixture(t, []string{"localhost"})
	ahdDevReloadTestInstall("00000000000000000000000000000001", "1")
	t.Cleanup(ahdDevReloadTestReset)
	database := filepath.Join(t.TempDir(), "sessions.db")
	usePersistentHelper(t)
	class := AhdClassHTTPError
	scope := AhdHTTPEvaluatorScopeBegin(nil)
	endScope := sync.OnceFunc(func() { AhdHTTPEvaluatorScopeEnd(scope) })
	t.Cleanup(endScope)
	store := AhdHTTPPersistentSessionsScoped(class, scope, database, "https_session", 3600, true, "Strict")
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(class, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	entered := make(chan struct{})
	resume := make(chan struct{})
	release := sync.OnceFunc(func() { close(resume) })
	callbackDone := make(chan string, 1)
	AhdHTTPServerGet(class, handle, "/", func(string) string {
		return AhdHTTPHTML(class, "<html><body>secure</body></html>", 200)
	})
	AhdHTTPServerPost(class, handle, "/login", func(request string) string {
		session := AhdHTTPSessionStoreOpen(class, store, request)
		session = AhdHTTPSessionRotate(class, session)
		session = AhdHTTPSessionSet(class, session, "user", "one")
		return AhdHTTPSessionStoreCommit(class, store, session, AhdHTTPText(class, "ok", 200))
	})
	AhdHTTPServerGet(class, handle, "/panel", func(request string) string {
		session := AhdHTTPSessionStoreOpen(class, store, request)
		user := AhdHTTPSessionGet(class, session, "user")
		if user == nil {
			return AhdHTTPText(class, "anonymous", 200)
		}
		return AhdHTTPText(class, *user, 200)
	})
	ws := AhdHTTPWebSocket(class, func(socket, text string) {
		if text == "block" {
			close(entered)
			<-resume
			callbackDone <- saveSessionUser(class, store, "retained")
			return
		}
		AhdWebSocketSend(class, socket, text)
	})
	AhdHTTPServerWebSocket(class, handle, "/socket", ws)
	startDone := make(chan any, 1)
	go func() {
		defer func() { startDone <- recover() }()
		AhdHTTPServerStartScoped(class, scope, handle)
	}()
	t.Cleanup(func() { release(); _ = AhdHTTPShutdownAddress("127.0.0.1", port) })
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	base := "https://" + addr
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	deadline := time.Now().Add(5 * time.Second)
	var page *http.Response
	for time.Now().Before(deadline) {
		var err error
		page, err = client.Get(base + "/")
		if err == nil {
			break
		}
		select {
		case failed := <-startDone:
			t.Fatalf("TLS startup failed: %v", failed)
		default:
		}
		time.Sleep(15 * time.Millisecond)
	}
	if page == nil {
		t.Fatal("HTTPS did not start")
	}
	body, err := io.ReadAll(page.Body)
	_ = page.Body.Close()
	if err != nil || page.TLS == nil || !strings.Contains(string(body), ahdDevReloadScriptPath) {
		t.Fatalf("HTTPS HTML/reload: %q %v", body, err)
	}
	response, err := client.Get(base + ahdDevReloadPath)
	if err != nil {
		t.Fatal(err)
	}
	probe, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil || response.StatusCode != 200 || !strings.Contains(string(probe), "00000000000000000000000000000001") {
		t.Fatalf("HTTPS reload probe = %d %q %v", response.StatusCode, probe, err)
	}
	for _, config := range []*tls.Config{
		{ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12},
		{RootCAs: pool, ServerName: "wrong.test", MinVersion: tls.VersionTLS12},
	} {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 2 * time.Second}, "tcp", addr, config)
		if err == nil {
			_ = conn.Close()
			t.Fatal("invalid trust or hostname was accepted")
		}
	}
	raw, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = raw.SetDeadline(time.Now().Add(2 * time.Second))
	_, _ = io.WriteString(raw, "GET / HTTP/1.1\r\nHost: "+addr+"\r\nConnection: close\r\n\r\n")
	plain, _ := io.ReadAll(raw)
	_ = raw.Close()
	if strings.Contains(string(plain), "200 OK") || strings.Contains(string(plain), ahdDevReloadScriptPath) {
		t.Fatalf("plaintext unexpectedly reached the application: %q", plain)
	}

	login := func(previous *http.Cookie) *http.Cookie {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, base+"/login", nil)
		if err != nil {
			t.Fatal(err)
		}
		if previous != nil {
			req.AddCookie(previous)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		cookies := resp.Cookies()
		if resp.StatusCode != 200 || len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
			t.Fatalf("HTTPS cookie status/attributes = %d, %d cookies", resp.StatusCode, len(cookies))
		}
		return cookies[0]
	}
	first := login(nil)
	second := login(first)
	if first.Value == second.Value {
		t.Fatal("HTTPS rotation reused the session ID")
	}
	for _, check := range []struct {
		cookie *http.Cookie
		want   string
	}{{first, "anonymous"}, {second, "one"}} {
		req, _ := http.NewRequest(http.MethodGet, base+"/panel", nil)
		req.AddCookie(check.cookie)
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil || string(data) != check.want {
			t.Fatalf("persistent HTTPS panel = %q, %v", data, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	bad, refused, err := websocket.Dial(ctx, "wss://"+addr+"/socket", &websocket.DialOptions{HTTPClient: client, HTTPHeader: http.Header{"Origin": []string{"http://" + addr}}})
	if bad != nil {
		_ = bad.CloseNow()
	}
	if refused != nil {
		_ = refused.Body.Close()
	}
	if err == nil || refused == nil || refused.StatusCode != http.StatusForbidden {
		t.Fatalf("plaintext origin was not refused: %v", err)
	}
	socket, _, err := websocket.Dial(ctx, "wss://"+addr+"/socket", &websocket.DialOptions{HTTPClient: client, HTTPHeader: http.Header{"Origin": []string{base}}})
	if err != nil {
		t.Fatal(err)
	}
	defer socket.CloseNow()
	writeText(t, socket, "secure echo")
	if got := readText(t, socket); got != "secure echo" {
		t.Fatalf("WSS echo = %q", got)
	}
	writeText(t, socket, "block")
	waitPhase0(t, entered)
	shutdown := make(chan bool, 1)
	go func() { shutdown <- AhdHTTPShutdownAddress("127.0.0.1", port) }()
	if !waitPhase0(t, shutdown) {
		t.Fatal("HTTPS shutdown did not claim the listener")
	}
	if failed := waitPhase0(t, startDone); failed != nil {
		t.Fatalf("HTTPS serve failed: %v", failed)
	}
	if _, _, err := socket.Read(ctx); err == nil || ctx.Err() != nil {
		t.Fatalf("WSS transport was not closed by shutdown: %v", err)
	}
	endScope()
	if !sessionDatabaseHeld(t, database) {
		t.Fatal("blocked WSS callback lost its scoped session file")
	}
	release()
	if id := waitPhase0(t, callbackDone); id == "" {
		t.Fatal("WSS callback could not save after shutdown")
	}
	waitUntilSessionFileReleased(t, database)
	if err := os.Remove(database); err != nil {
		t.Fatalf("WSS left SQLite file locked: %v", err)
	}
	assertPortReusable(t, port)
}

func TestDevHTTPSForcedHTTPShutdownReleasesSQLite(t *testing.T) {
	pool := installHTTPSFixture(t, []string{"localhost"})
	database := filepath.Join(t.TempDir(), "sessions.db")
	usePersistentHelper(t)
	class := AhdClassHTTPError
	scope := AhdHTTPEvaluatorScopeBegin(nil)
	endScope := sync.OnceFunc(func() { AhdHTTPEvaluatorScopeEnd(scope) })
	t.Cleanup(endScope)
	store := AhdHTTPPersistentSessionsScoped(class, scope, database, "https_session", 3600, true, "Lax")
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(class, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	entered, resume := make(chan struct{}), make(chan struct{})
	release := sync.OnceFunc(func() { close(resume) })
	callbackDone := make(chan string, 1)
	AhdHTTPServerGet(class, handle, "/block", func(string) string {
		close(entered)
		<-resume
		callbackDone <- saveSessionUser(class, store, "after forced shutdown")
		return AhdHTTPText(class, "ok", 200)
	})
	bound := make(chan struct{})
	previous := ahdHTTPTestBeforeServe
	ahdHTTPTestBeforeServe = func() { close(bound) }
	server := ahdHTTPLookup(class, handle)
	httpServer := server.claimStart(class, scope)
	handler := httpServer.Handler
	handlerDone := make(chan struct{})
	httpServer.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// A forced close releases the listener before the HTTP response
		// writer returns. Join the dispatcher before later tests replace
		// the in-process development reload identity.
		defer close(handlerDone)
		handler.ServeHTTP(writer, request)
	})
	serveResult := make(chan any, 1)
	serveDone := make(chan struct{})
	go func() {
		defer close(serveDone)
		defer func() { serveResult <- recover() }()
		server.serve(class, httpServer, ahdHTTPDevListen)
	}()
	t.Cleanup(func() {
		release()
		_ = AhdHTTPShutdownAddress("127.0.0.1", port)
		waitPhase0(t, serveDone)
		select {
		case <-entered:
			waitPhase0(t, handlerDone)
		default:
		}
		ahdHTTPTestBeforeServe = previous
	})
	waitPhase0(t, bound)
	client := &http.Client{
		Timeout:   6 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}},
	}
	defer client.CloseIdleConnections()
	reqDone := make(chan error, 1)
	go func() {
		response, err := client.Get("https://127.0.0.1:" + strconv.Itoa(port) + "/block")
		if response != nil {
			_ = response.Body.Close()
		}
		reqDone <- err
	}()
	waitPhase0(t, entered)
	stopped := make(chan bool, 1)
	go func() { stopped <- AhdHTTPShutdownAddress("127.0.0.1", port) }()
	if !waitPhase0(t, stopped) {
		t.Fatal("HTTPS shutdown failed")
	}
	if err := waitPhase0(t, reqDone); err == nil {
		t.Fatal("blocked HTTPS connection was not forced closed")
	}
	if err := waitPhase0(t, serveResult); err != nil {
		t.Fatalf("HTTPS serve failed: %v", err)
	}
	endScope()
	if !sessionDatabaseHeld(t, database) {
		t.Fatal("forced HTTPS shutdown released the active handler's database")
	}
	release()
	if id := waitPhase0(t, callbackDone); id == "" {
		t.Fatal("callback did not commit")
	}
	waitPhase0(t, handlerDone)
	waitUntilSessionFileReleased(t, database)
	if err := os.Remove(database); err != nil {
		t.Fatalf("HTTPS left SQLite file locked: %v", err)
	}
	assertPortReusable(t, port)
}
