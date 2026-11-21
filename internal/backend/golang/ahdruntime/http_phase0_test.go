package ahdruntime

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPBindFailureClearsOwnershipBeforeShutdownAndRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed-start.db")
	usePersistentHelper(t)
	class := AhdClassHTTPError
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = occupied.Close() })
	port := occupied.Addr().(*net.TCPAddr).Port
	handle := AhdHTTPServer(class, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	server := ahdHTTPLookup(class, handle)
	t.Cleanup(server.retire)
	scope := AhdHTTPEvaluatorScopeBegin(nil)
	endScope := sync.OnceFunc(func() { AhdHTTPEvaluatorScopeEnd(scope) })
	t.Cleanup(endScope)
	AhdHTTPPersistentSessionsScoped(class, scope, path, "ahd_session", 3600, false, "Lax")
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("the evaluator scope did not open its database")
	}

	start := server.claimStart(class, scope)
	bindEntered := make(chan struct{})
	bindRelease := make(chan struct{})
	unblockBind := sync.OnceFunc(func() { close(bindRelease) })
	bindResult := make(chan any, 1)
	t.Cleanup(unblockBind)
	go func() {
		defer func() { bindResult <- recover() }()
		server.serve(class, start, func(network, address string) (net.Listener, error) {
			close(bindEntered)
			<-bindRelease
			return net.Listen(network, address)
		})
	}()
	waitPhase0(t, bindEntered)

	// Hold the route mutex so the failed start cannot unfreeze routes.
	// Shutdown must see the bind result without waiting for that mutex.
	stopped := func() bool {
		server.mutex.Lock()
		defer server.mutex.Unlock()
		shutdownStarted := make(chan struct{})
		shutdownResult := make(chan bool, 1)
		go func() {
			close(shutdownStarted)
			shutdownResult <- AhdHTTPShutdownAddress("127.0.0.1", port)
		}()
		waitPhase0(t, shutdownStarted)
		unblockBind()
		return waitPhase0(t, shutdownResult)
	}()
	if stopped {
		t.Fatal("shutdown claimed a server whose bind failed")
	}
	assertHTTPBindError(t, waitPhase0(t, bindResult))
	if !ahdHTTPHandleRegistered(handle) || server.owner.Load() || server.closing.Load() || server.routesFrozen.Load() {
		t.Fatal("failed bind did not leave a reusable, idle handle")
	}
	endScope()
	if sessionDatabaseHeld(t, path) {
		t.Fatal("failed start retained a database after its scope ended")
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("failed start left the database locked: %v", err)
	}

	// A second real bind still fails while the port is occupied.
	retry := func() (failure any) {
		defer func() { failure = recover() }()
		AhdHTTPServerStart(class, handle)
		return nil
	}
	assertHTTPBindError(t, retry())
	if err := occupied.Close(); err != nil {
		t.Fatal(err)
	}
	AhdHTTPServerGet(class, handle, "/retry", func(string) string {
		return AhdHTTPText(class, "reused", http.StatusOK)
	})
	// Start the same handle with a real listener and wait for a response
	// through it before checking shutdown.
	ready := make(chan struct{})
	retryResult := make(chan any, 1)
	retryServer := server.claimStart(class, 0)
	go func() {
		defer func() { retryResult <- recover() }()
		server.serve(class, retryServer, func(network, address string) (net.Listener, error) {
			listener, err := net.Listen(network, address)
			close(ready)
			return listener, err
		})
	}()
	t.Cleanup(func() { _ = AhdHTTPShutdownAddress("127.0.0.1", port) })
	waitPhase0(t, ready)
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get("http://" + occupied.Addr().String() + "/retry")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("reused handle served status %d", response.StatusCode)
	}
	if !AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("successful retry did not publish a listener")
	}
	if recovered := waitPhase0(t, retryResult); recovered != nil {
		t.Fatalf("successful retry failed after shutdown: %v", recovered)
	}
	assertPortReusable(t, port)
}

func TestHTTPCallbackRetirementPreservesOnlyAdmittedOwnership(t *testing.T) {
	usePersistentHelper(t)
	for _, kind := range []string{"route", "error-page", "websocket"} {
		t.Run(kind, func(t *testing.T) {
			class := AhdClassHTTPError
			path := filepath.Join(t.TempDir(), "callback.db")
			scope := AhdHTTPEvaluatorScopeBegin(nil)
			endScope := sync.OnceFunc(func() { AhdHTTPEvaluatorScopeEnd(scope) })
			t.Cleanup(endScope)
			store := AhdHTTPPersistentSessionsScoped(class, scope, path, "ahd_session", 3600, false, "Lax")
			handle := AhdHTTPServer(class, "127.0.0.1", int64(freeLoopbackPort(t)), ahdHTTPDefaultMaxBody)
			state := ahdHTTPLookup(class, handle)
			t.Cleanup(state.retire)
			entered := make(chan struct{})
			resume := make(chan struct{})
			unblock := sync.OnceFunc(func() { close(resume) })
			var calls atomic.Int32
			var id string
			handler := func(string) string {
				if calls.Add(1) == 1 {
					close(entered)
					<-resume
					id = saveSessionUser(class, store, "retained")
				}
				return AhdHTTPText(class, "ok", http.StatusOK)
			}
			AhdHTTPServerGet(class, handle, "/callback", handler)
			dispatcher := state.claimStart(class, scope).Handler
			invoke := func() int {
				request := httptest.NewRequest(http.MethodGet, "/callback", nil)
				response := httptest.NewRecorder()
				switch kind {
				case "route":
					dispatcher.ServeHTTP(response, request)
				case "error-page":
					ahdHTTPServeInstalledPage(state, response, request, handler, ahdHTTPRequestData{}, http.StatusNotFound, "Not Found")
				case "websocket":
					if ahdWebSocketCall(state, func() { handler("") }) {
						return http.StatusServiceUnavailable
					}
					return http.StatusOK
				}
				return response.Code
			}
			result := make(chan int, 1)
			complete := make(chan struct{})
			go func() {
				defer close(complete)
				result <- invoke()
			}()
			t.Cleanup(func() { unblock(); waitPhase0(t, complete) })
			waitPhase0(t, entered)
			if !ahdHTTPShutdownState(state) {
				t.Fatal("shutdown missed the started server")
			}
			endScope()
			if !sessionDatabaseHeld(t, path) {
				t.Fatal("retirement lost the admitted callback's database")
			}
			unblock()
			status := waitPhase0(t, result)
			if want := map[string]int{"route": http.StatusOK, "error-page": http.StatusNotFound, "websocket": http.StatusOK}[kind]; status != want || id == "" {
				t.Fatalf("admitted callback result = %d, session saved = %v", status, id != "")
			}

			state.mutex.Lock()
			admitted := state.tryEnterCallback()
			if admitted {
				state.leaveCallback()
			}
			state.mutex.Unlock()
			if admitted {
				state.finishCallback()
				t.Fatal("the lifetime admission gate accepted a retired server")
			}
			if status := invoke(); status != http.StatusServiceUnavailable || calls.Load() != 1 {
				t.Fatalf("retired callback result = %d, calls = %d", status, calls.Load())
			}
			if sessionDatabaseHeld(t, path) {
				t.Fatal("the callback left a scoped database open")
			}
			if err := os.Remove(path); err != nil {
				t.Fatalf("the callback left a scoped file locked: %v", err)
			}
		})
	}
}

func waitPhase0[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for lifecycle synchronization")
		var zero T
		return zero
	}
}

func assertHTTPBindError(t *testing.T, recovered any) {
	t.Helper()
	signal, ok := recovered.(*AhdSignal)
	if !ok || signal.Instance.AhdClassOf() != AhdClassHTTPError || !strings.Contains(signal.Message, "could not bind") {
		t.Fatalf("expected HTTP bind error, got %v", recovered)
	}
}
