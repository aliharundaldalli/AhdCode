package evaluator

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"ahdcode/internal/backend/golang/ahdruntime"
)

var (
	evaluatorSQLiteHelperOnce     sync.Once
	evaluatorSQLiteHelperPath     string
	evaluatorSQLiteHelperBuildErr error
	evaluatorSQLiteHelperWarmOnce sync.Once
	evaluatorSQLiteHelperWarmErr  error
)

func TestEvaluatorPersistentSessionLogin(t *testing.T) {
	helper := buildEvaluatorSQLiteHelper(t)
	t.Setenv("AHDCODE_SQLITE_RUNTIME", helper)
	port := freeEvaluatorPort(t)
	databaseDir := t.TempDir()
	t.Cleanup(func() {
		if err := ahdruntime.AhdHTTPReleasePersistentSessionStorageErr(); err != nil {
			t.Errorf("releasing persistent session storage: %v", err)
		}
	})
	database := filepath.ToSlash(filepath.Join(databaseDir, "sessions.db"))
	source := `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring SessionStore
from HTTP bring Session

sessions: SessionStore := HTTP.persistentSessions("` + database + `", "ahd_session", 3600, false, "Lax")

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

login: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    session.rotate()
    session.set("user", "Ali")
    return sessions.commit(session, HTTP.text("in"))
}

panel: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    user: Local String? := session.get("user")
    if user == null {
        return sessions.commit(session, HTTP.text("anonymous"))
    }
    return sessions.commit(session, HTTP.text(user))
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", ok)
app.post("/login", login)
app.get("/panel", panel)
app.start()
`
	base, finished, outcome := startEvaluatorHTTPWithDone(t, source, port)
	client := &http.Client{Timeout: 0, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader("name=Ali"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "in" || len(response.Cookies()) == 0 {
		t.Fatalf("evaluator login = %d %q cookies %#v", response.StatusCode, body, response.Header["Set-Cookie"])
	}
	panel, err := http.NewRequest(http.MethodGet, base+"/panel", nil)
	if err != nil {
		t.Fatal(err)
	}
	panel.AddCookie(response.Cookies()[0])
	view, err := client.Do(panel)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(view.Body)
	_ = view.Body.Close()
	if string(body) != "Ali" {
		t.Fatalf("evaluator panel = %q", body)
	}
	if !ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("shutdown did not claim the evaluator HTTP server")
	}
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("evaluator HTTP server did not finish after shutdown")
	}
	if outcome.panicked || outcome.result.Failure != nil {
		t.Fatalf("evaluator server ended with failure: %s", formatEvaluatorOutcome(*outcome))
	}

	again := ahdruntime.AhdHTTPPersistentSessions(ahdruntime.AhdClassHTTPError, filepath.FromSlash(database), "ahd_session", 3600, false, "Lax")
	opened := ahdruntime.AhdHTTPSessionStoreOpen(ahdruntime.AhdClassHTTPError, again, ahdruntimeRequest(response.Cookies()[0].Value))
	got := ahdruntime.AhdHTTPSessionGet(ahdruntime.AhdClassHTTPError, opened, "user")
	if got == nil || *got != "Ali" {
		t.Fatalf("evaluator session after shutdown = %v", got)
	}
}

func ahdruntimeRequest(cookie string) string {
	return `{"method":"GET","path":"/","query":[],"headers":[],"cookies":[{"name":"ahd_session","value":"` + cookie + `"}],"form":[],"files":[],"body":""}`
}

func buildEvaluatorSQLiteHelper(t *testing.T) string {
	t.Helper()
	evaluatorSQLiteHelperOnce.Do(func() {
		root, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			evaluatorSQLiteHelperBuildErr = err
			return
		}
		// The helper process is shared across tests and outlives t.TempDir
		// cleanup, so build it once outside any per-test temporary directory.
		directory, err := os.MkdirTemp("", "ahd-evaluator-sqlite")
		if err != nil {
			evaluatorSQLiteHelperBuildErr = err
			return
		}
		helper := filepath.Join(directory, "ahdsqlite")
		if runtime.GOOS == "windows" {
			helper += ".exe"
		}
		command := exec.Command("go", "build", "-o", helper, "./cmd/ahdsqlite")
		command.Dir = root
		output, err := command.CombinedOutput()
		if err != nil {
			evaluatorSQLiteHelperBuildErr = fmt.Errorf("could not build ahdsqlite: %w\n%s", err, output)
			return
		}
		evaluatorSQLiteHelperPath = helper
	})
	if evaluatorSQLiteHelperBuildErr != nil {
		t.Fatal(evaluatorSQLiteHelperBuildErr)
	}
	t.Setenv("AHDCODE_SQLITE_RUNTIME", evaluatorSQLiteHelperPath)
	evaluatorSQLiteHelperWarmOnce.Do(func() {
		handle, err := ahdruntime.SQLiteOpen(":memory:")
		if err != nil {
			evaluatorSQLiteHelperWarmErr = fmt.Errorf("ahdsqlite helper health check failed: %w", err)
			return
		}
		if err := ahdruntime.SQLiteClose(handle); err != nil {
			evaluatorSQLiteHelperWarmErr = fmt.Errorf("ahdsqlite helper close failed: %w", err)
		}
	})
	if evaluatorSQLiteHelperWarmErr != nil {
		t.Fatal(evaluatorSQLiteHelperWarmErr)
	}
	return evaluatorSQLiteHelperPath
}

func TestEvaluatorPersistentSessionRelativePathUsesSessionCWD(t *testing.T) {
	helper := buildEvaluatorSQLiteHelper(t)
	t.Setenv("AHDCODE_SQLITE_RUNTIME", helper)
	session := newLatexTestSession()
	directory := filepath.Join(t.TempDir(), "Oturum Klasörü")
	t.Cleanup(func() {
		if err := ahdruntime.AhdHTTPReleasePersistentSessionStorageErr(); err != nil {
			t.Errorf("releasing persistent session storage: %v", err)
		}
	})
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	session.CWD = directory
	session.httpBuiltin("persistentSessions", []any{filepath.ToSlash("kalıcı oturum.db"), "ahd_session", int64(60), false, "Lax"})
	created := filepath.Join(directory, "kalıcı oturum.db")
	if _, err := os.Stat(created); err != nil {
		t.Fatalf("relative session file was not created in the session directory: %v", err)
	}
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(packageDir, "kalıcı oturum.db")); !os.IsNotExist(err) {
		t.Fatalf("relative session file appeared in the evaluator package directory: %v", err)
	}
	absolute := filepath.Join(directory, "absolute oturum.db")
	session.httpBuiltin("persistentSessions", []any{absolute, "ahd_session", int64(60), false, "Lax"})
	if _, err := os.Stat(absolute); err != nil {
		t.Fatalf("absolute session file was not created: %v", err)
	}
}

func TestEvaluatorPersistentSessionRejectsADirectory(t *testing.T) {
	helper := buildEvaluatorSQLiteHelper(t)
	t.Setenv("AHDCODE_SQLITE_RUNTIME", helper)
	session := newLatexTestSession()
	expectEvaluatorRaise(t, "HTTPError", func() {
		session.httpBuiltin("persistentSessions", []any{t.TempDir(), "ahd_session", int64(60), false, "Lax"})
	})
}

func TestEvaluatorPersistentSessionReleasesWhenNoServerRuns(t *testing.T) {
	helper := buildEvaluatorSQLiteHelper(t)
	t.Setenv("AHDCODE_SQLITE_RUNTIME", helper)
	directory := t.TempDir()
	t.Cleanup(func() {
		if err := ahdruntime.AhdHTTPReleasePersistentSessionStorageErr(); err != nil {
			t.Errorf("releasing persistent session storage: %v", err)
		}
	})
	database := filepath.Join(directory, "sessions.db")
	source := `bring HTTP
from HTTP bring SessionStore
sessions: SessionStore := HTTP.persistentSessions("` + filepath.ToSlash(database) + `")
write("ok")
`

	session := newLatexTestSession()
	result := session.Execute(compileAhd(t, source), 0)
	if result.Failure != nil {
		t.Fatalf("program failed: %s", result.Failure.Message)
	}
	if _, err := os.Stat(database); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(database); err != nil {
		t.Fatalf("evaluator left the session file locked: %v", err)
	}
}

func TestEvaluatorUnstartedServerDoesNotPinTheSessionFile(t *testing.T) {
	helper := buildEvaluatorSQLiteHelper(t)
	t.Setenv("AHDCODE_SQLITE_RUNTIME", helper)
	port := freeEvaluatorPort(t)
	idle := freeEvaluatorPort(t)
	directory := t.TempDir()
	t.Cleanup(func() {
		if err := ahdruntime.AhdHTTPReleasePersistentSessionStorageErr(); err != nil {
			t.Errorf("releasing persistent session storage: %v", err)
		}
	})
	database := filepath.ToSlash(filepath.Join(directory, "sessions.db"))
	source := `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring SessionStore

sessions: SessionStore := HTTP.persistentSessions("` + database + `")
idle: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(idle) + `)
ok: Function := (request: Request) -> Response { return HTTP.text("ok") }
app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", ok)
app.start()
`
	base, finished, outcome := startEvaluatorHTTPWithDone(t, source, port)
	response, err := evaluatorHTTPGet(base + "/ok")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /ok = %d", response.StatusCode)
	}
	if !ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("shutdown did not claim the running server")
	}
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("the evaluator did not finish after the server stopped")
	}
	if outcome.panicked || outcome.result.Failure != nil {
		t.Fatalf("evaluator server ended with failure: %s", formatEvaluatorOutcome(*outcome))
	}
	if err := os.Remove(filepath.FromSlash(database)); err != nil {
		t.Fatalf("an unstarted server left the session file locked: %v", err)
	}
}

func TestEvaluatorHTTPStartupReportsActualExecutionFailure(t *testing.T) {
	port := freeEvaluatorPort(t)
	source := `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring SessionStore

sessions: SessionStore := HTTP.persistentSessions(":memory:")
ok: Function := (request: Request) -> Response { return HTTP.text("ok") }
app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", ok)
app.start()
`
	started := time.Now()
	_, _, _, cleanup, err := waitForEvaluatorHTTP(compileAhd(t, source), port, 2*time.Second)
	defer cleanup()
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("expected startup error for invalid persistent session path")
	}
	if elapsed >= time.Second {
		t.Fatalf("startup failure took %s; should fail immediately without waiting for readiness timeout: %v", elapsed, err)
	}
	if !strings.Contains(err.Error(), ":memory:") || !strings.Contains(err.Error(), "HTTPError") {
		t.Fatalf("startup error hid actual cause: %v", err)
	}
	if ahdruntime.AhdHTTPServerActive() {
		t.Fatal("failed startup left an active HTTP server")
	}
}

func TestEvaluatorPersistentSessionReleasesWhileUnrelatedServerRuns(t *testing.T) {
	helper := buildEvaluatorSQLiteHelper(t)
	t.Setenv("AHDCODE_SQLITE_RUNTIME", helper)
	t.Cleanup(func() {
		if err := ahdruntime.AhdHTTPReleasePersistentSessionStorageErr(); err != nil {
			t.Errorf("releasing persistent session storage: %v", err)
		}
	})

	unrelatedPort := freeEvaluatorPort(t)
	unrelatedBase := startEvaluatorHTTP(t, `bring HTTP
from HTTP bring (Server, Request, Response)

ok: Function := (request: Request) -> Response { return HTTP.text("ok") }
app: Server := HTTP.server("127.0.0.1", `+strconv.Itoa(unrelatedPort)+`)
app.get("/ok", ok)
app.start()
`, unrelatedPort)

	directory := t.TempDir()
	database := filepath.Join(directory, "scoped.db")
	session := newLatexTestSession()
	result := session.Execute(compileAhd(t, `bring HTTP
from HTTP bring SessionStore
sessions: SessionStore := HTTP.persistentSessions("`+filepath.ToSlash(database)+`")
write("ok")
`), 0)
	if result.Failure != nil {
		t.Fatalf("scoped session program failed: %s", result.Failure.Message)
	}
	if err := os.Remove(database); err != nil {
		t.Fatalf("unrelated running server kept scoped session database locked: %v", err)
	}

	response, err := evaluatorHTTPGet(unrelatedBase + "/ok")
	if err != nil {
		t.Fatalf("unrelated server stopped answering: %v", err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unrelated GET /ok = %d", response.StatusCode)
	}
}

func TestEvaluatorPersistentSessionSharedDBStaysOpenWhilePeerServerRuns(t *testing.T) {
	helper := buildEvaluatorSQLiteHelper(t)
	t.Setenv("AHDCODE_SQLITE_RUNTIME", helper)
	t.Cleanup(func() {
		if err := ahdruntime.AhdHTTPReleasePersistentSessionStorageErr(); err != nil {
			t.Errorf("releasing persistent session storage: %v", err)
		}
	})

	port := freeEvaluatorPort(t)
	directory := t.TempDir()
	database := filepath.ToSlash(filepath.Join(directory, "shared.db"))
	serverSource := `bring HTTP
from HTTP bring (Server, Request, Response, SessionStore, Session)

sessions: SessionStore := HTTP.persistentSessions("` + database + `", "ahd_session", 3600, false, "Lax")

ok: Function := (request: Request) -> Response { return HTTP.text("ok") }

login: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    session.rotate()
    session.set("user", "Ali")
    return sessions.commit(session, HTTP.text("in"))
}

panel: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    user: Local String? := session.get("user")
    if user == null {
        return sessions.commit(session, HTTP.text("anonymous"))
    }
    return sessions.commit(session, HTTP.text(user))
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", ok)
app.post("/login", login)
app.get("/panel", panel)
app.start()
`
	base, finished, outcome := startEvaluatorHTTPWithDone(t, serverSource, port)
	client := &http.Client{Timeout: 5 * time.Second}
	loginResp, err := client.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, loginResp.Body)
	_ = loginResp.Body.Close()
	if len(loginResp.Cookies()) != 1 {
		t.Fatalf("login cookies = %#v", loginResp.Header["Set-Cookie"])
	}

	// Second evaluator opens the same database without a server and finishes.
	peer := newLatexTestSession()
	peerResult := peer.Execute(compileAhd(t, `bring HTTP
from HTTP bring SessionStore
sessions: SessionStore := HTTP.persistentSessions("`+database+`", "other_ns", 3600, false, "Lax")
write("ok")
`), 0)
	if peerResult.Failure != nil {
		t.Fatalf("peer session failed: %s", peerResult.Failure.Message)
	}

	// The running server must still read its session cleanly after the peer ends.
	req, err := http.NewRequest(http.MethodGet, base+"/panel", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(loginResp.Cookies()[0])
	view, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(view.Body)
	_ = view.Body.Close()
	if view.StatusCode != http.StatusOK || string(body) != "Ali" {
		t.Fatalf("panel after peer evaluator finished = %d %q", view.StatusCode, body)
	}

	if !ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("shutdown did not claim server")
	}
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("server evaluator did not exit after shutdown")
	}
	if outcome.panicked || outcome.result.Failure != nil {
		t.Fatalf("server evaluator failed: %s", formatEvaluatorOutcome(*outcome))
	}
	if err := os.Remove(filepath.FromSlash(database)); err != nil {
		t.Fatalf("shared database stayed locked after server exited: %v", err)
	}
}
