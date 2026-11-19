package ahdruntime

import (
	"fmt"
	"io"
	"net"
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

	"ahdcode/internal/migrate"
)

var persistentHelperOnce sync.Once
var persistentHelperPath string

func usePersistentHelper(t *testing.T) {
	t.Helper()
	persistentHelperOnce.Do(func() {
		directory, err := os.MkdirTemp("", "ahdcode-session-helper")
		if err != nil {
			t.Fatal(err)
		}
		helper := filepath.Join(directory, sessionHelperExecutable())
		command := exec.Command("go", "build", "-o", helper, "./cmd/ahdsqlite")
		command.Dir = sqliteRepositoryRoot
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("could not build ahdsqlite: %v\n%s", err, output)
		}
		persistentHelperPath = helper
	})
	if persistentHelperPath == "" {
		t.Fatal("ahdsqlite helper was not built")
	}
	t.Setenv("AHDCODE_SQLITE_RUNTIME", persistentHelperPath)
	t.Cleanup(func() {
		if err := ahdHTTPTestClosePersistentStores(); err != nil {
			t.Errorf("closing persistent session databases: %v", err)
		}
	})
}

func persistentSessionPath(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	usePersistentHelper(t)
	return filepath.Join(directory, "sessions.db")
}

func sessionCookie(t *testing.T, class *AhdClass, store, session string) string {
	t.Helper()
	committed := AhdHTTPSessionStoreCommit(class, store, session, AhdHTTPText(class, "ok", 200))
	cookies := ahdHTTPDecodeResponse(class, committed).Cookies
	if len(cookies) != 1 {
		t.Fatalf("cookies = %#v", cookies)
	}
	return cookies[0].Value
}

func sessionRequest(name, id string) string {
	return ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/",
		Cookies: []ahdHTTPCookieEntry{{Name: name, Value: id}},
	})
}

func expectSessionText(t *testing.T, fragment string, body func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		signal, ok := recovered.(*AhdSignal)
		if !ok || signal.Instance == nil || signal.Instance.AhdClassOf() != AhdClassHTTPError {
			t.Fatalf("expected HTTPError containing %q; recovered %#v", fragment, recovered)
		}
		if !strings.Contains(signal.Message, fragment) {
			t.Fatalf("HTTPError %q does not contain %q", signal.Message, fragment)
		}
	}()
	body()
}

func TestPersistentSessionSurvivesANewStore(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "ahd_session", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	committed := AhdHTTPSessionStoreCommit(class, store,
		AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"),
		AhdHTTPText(class, "ok", 200))
	cookie := ahdHTTPDecodeResponse(class, committed).Cookies[0]
	wire := ahdHTTPCookieWire(cookie)
	if !cookie.HttpOnly || cookie.Secure || cookie.Path != "/" || cookie.SameSite != "Lax" || cookie.MaxAge == nil || *cookie.MaxAge != 3600 {
		t.Fatalf("cookie = %#v", cookie)
	}
	if !strings.Contains(wire, "HttpOnly") || strings.Contains(wire, "Ali") {
		t.Fatalf("cookie wire = %s", wire)
	}
	reopened := AhdHTTPPersistentSessions(class, path, "ahd_session", 3600, false, "Lax")
	loaded := AhdHTTPSessionStoreOpen(class, reopened, sessionRequest("ahd_session", cookie.Value))
	if got := AhdHTTPSessionGet(class, loaded, "user"); got == nil || *got != "Ali" {
		t.Fatalf("after a new store, user = %v", got)
	}
}

func TestPersistentSessionsAreIsolated(t *testing.T) {
	class := AhdClassHTTPError
	directory := t.TempDir()
	usePersistentHelper(t)
	path := filepath.Join(directory, "shared.db")
	storeA := AhdHTTPPersistentSessions(class, path, "a", 3600, false, "Lax")
	storeB := AhdHTTPPersistentSessions(class, path, "b", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	idA := sessionCookie(t, class, storeA, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, storeA, empty), "who", "Ali"))
	idB := sessionCookie(t, class, storeB, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, storeB, empty), "who", "Mehmet"))
	fromA := AhdHTTPSessionStoreOpen(class, storeA, ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/",
		Cookies: []ahdHTTPCookieEntry{{Name: "a", Value: idA}, {Name: "b", Value: idB}},
	}))
	fromB := AhdHTTPSessionStoreOpen(class, storeB, ahdHTTPEncodeRequest(ahdHTTPRequestData{
		Method: "GET", Path: "/",
		Cookies: []ahdHTTPCookieEntry{{Name: "a", Value: idA}, {Name: "b", Value: idB}},
	}))
	if got := AhdHTTPSessionGet(class, fromA, "who"); got == nil || *got != "Ali" {
		t.Fatalf("A = %v", got)
	}
	if got := AhdHTTPSessionGet(class, fromB, "who"); got == nil || *got != "Mehmet" {
		t.Fatalf("B = %v", got)
	}
	destroyed := AhdHTTPSessionDestroy(class, fromA)
	AhdHTTPSessionStoreCommit(class, storeA, destroyed, AhdHTTPText(class, "bye", 200))
	again := AhdHTTPPersistentSessions(class, path, "a", 3600, false, "Lax")
	if AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, again, sessionRequest("a", idA)), "who") != nil {
		t.Fatal("destroyed session was readable after a new store")
	}
	stillB := AhdHTTPPersistentSessions(class, path, "b", 3600, false, "Lax")
	if got := AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, stillB, sessionRequest("b", idB)), "who"); got == nil || *got != "Mehmet" {
		t.Fatal("destroying A removed B")
	}
}

func TestPersistentSessionExpiryStaysExpired(t *testing.T) {
	class := AhdClassHTTPError
	defer ahdHTTPTestResetClock()
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	ahdHTTPTimeNow = func() time.Time { return now }
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 1, true, "Strict")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "k", "v"))
	cookie := ahdHTTPDecodeResponse(class, AhdHTTPSessionStoreCommit(class, store,
		AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "k", "v"),
		AhdHTTPText(class, "ok", 200))).Cookies[0]
	if !cookie.Secure || cookie.SameSite != "Strict" {
		t.Fatalf("secure cookie = %#v", cookie)
	}
	id = cookie.Value
	ahdHTTPTimeNow = func() time.Time { return now.Add(2 * time.Second) }
	if AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id)), "k") != nil {
		t.Fatal("expired session was accepted")
	}
	restarted := AhdHTTPPersistentSessions(class, path, "s", 1, true, "Strict")
	if AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, restarted, sessionRequest("s", id)), "k") != nil {
		t.Fatal("expired session became valid after a new store")
	}
}

func TestPersistentSessionRotateInvalidatesTheOldIdentifier(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	session := AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali")
	oldID := sessionCookie(t, class, store, session)
	current := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", oldID))
	current = AhdHTTPSessionRotate(class, current)
	current = AhdHTTPSessionSet(class, current, "user", "Ali")
	newID := sessionCookie(t, class, store, current)
	if newID == oldID {
		t.Fatal("rotate kept the old identifier")
	}
	if AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", oldID)), "user") != nil {
		t.Fatal("old identifier still loaded the session")
	}
	restarted := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	if AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, restarted, sessionRequest("s", oldID)), "user") != nil {
		t.Fatal("old identifier worked after a new store")
	}
	if got := AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, restarted, sessionRequest("s", newID)), "user"); got == nil || *got != "Ali" {
		t.Fatalf("new identifier = %v", got)
	}
}

func TestPersistentSessionRejectsAStaleCommit(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	stale := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	live := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	AhdHTTPSessionStoreCommit(class, store, AhdHTTPSessionDestroy(class, live), AhdHTTPText(class, "out", 200))
	stale = AhdHTTPSessionSet(class, stale, "user", "intruder")
	expectSessionText(t, "HTTP session is no longer valid", func() {
		AhdHTTPSessionStoreCommit(class, store, stale, AhdHTTPText(class, "no", 200))
	})
	restarted := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	if AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, restarted, sessionRequest("s", id)), "user") != nil {
		t.Fatal("stale commit resurrected a destroyed session")
	}
}

func TestPersistentSessionRejectsACommitAfterExpiry(t *testing.T) {
	class := AhdClassHTTPError
	defer ahdHTTPTestResetClock()
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	ahdHTTPTimeNow = func() time.Time { return now }
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 1, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	open := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	ahdHTTPTimeNow = func() time.Time { return now.Add(2 * time.Second) }
	expectSessionText(t, "HTTP session is no longer valid", func() {
		AhdHTTPSessionStoreCommit(class, store, AhdHTTPSessionSet(class, open, "user", "later"), AhdHTTPText(class, "no", 200))
	})
	if AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id)), "user") != nil {
		t.Fatal("expired commit refreshed the session")
	}
}

func TestPersistentSessionConcurrentClientsAndUpdates(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	var wg sync.WaitGroup
	ids := make([]string, 24)
	failures := make([]string, 24)
	wg.Add(24)
	for index := 0; index < 24; index++ {
		go func(index int) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					failures[index] = fmt.Sprint(recovered)
				}
			}()
			name := "user-" + strings.Repeat("x", index%3) + string(rune('a'+index))
			session := AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "who", name)
			committed := AhdHTTPSessionStoreCommit(class, store, session, AhdHTTPText(class, "ok", 200))
			ids[index] = ahdHTTPDecodeResponse(class, committed).Cookies[0].Value
		}(index)
	}
	wg.Wait()
	for index, failure := range failures {
		if failure != "" {
			t.Fatalf("client %d: %s", index, failure)
		}
	}
	for index, id := range ids {
		name := "user-" + strings.Repeat("x", index%3) + string(rune('a'+index))
		got := AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id)), "who")
		if got == nil || *got != name {
			t.Fatalf("client %d = %v, want %s", index, got, name)
		}
	}

	base := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", ids[0]))
	recorded := make([]map[string]string, 24)
	wg.Add(24)
	for index := 0; index < 24; index++ {
		go func(index int) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					failures[index] = fmt.Sprint(recovered)
				}
			}()
			current := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", ids[0]))
			current = AhdHTTPSessionSet(class, current, "note", string(rune('a'+index)))
			recorded[index] = ahdHTTPDecodeSession(class, current).Values
			AhdHTTPSessionStoreCommit(class, store, current, AhdHTTPText(class, "ok", 200))
		}(index)
	}
	wg.Wait()
	for index, failure := range failures {
		if failure != "" {
			t.Fatalf("update %d: %s", index, failure)
		}
	}
	final := ahdHTTPDecodeSession(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", ids[0]))).Values
	if final["who"] != ahdHTTPDecodeSession(class, base).Values["who"] && final["who"] == "" {
		t.Fatalf("final map = %#v", final)
	}
	matched := false
	for _, snapshot := range recorded {
		if mapsEqual(snapshot, final) {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("final map %#v is not one committed snapshot", final)
	}
}

func mapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func TestPersistentSessionStorageFaultDoesNotAuthenticate(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	open := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	ahdHTTPTestSetSessionStorageFault("disk full")
	defer ahdHTTPTestSetSessionStorageFault("")
	expectSessionText(t, "HTTP session storage failed: disk full", func() {
		AhdHTTPSessionStoreCommit(class, store, AhdHTTPSessionSet(class, open, "user", "other"), AhdHTTPText(class, "no", 200))
	})
	expectSessionText(t, "HTTP session storage failed: disk full", func() {
		AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	})
	ahdHTTPTestSetSessionStorageFault("")
	if got := AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id)), "user"); got == nil || *got != "Ali" {
		t.Fatalf("fault changed the stored user to %v", got)
	}
}

func TestPersistentSessionRejectsBadConfigurationAndSchema(t *testing.T) {
	class := AhdClassHTTPError
	directory := t.TempDir()
	usePersistentHelper(t)
	expectSessionText(t, "must not be empty", func() {
		AhdHTTPPersistentSessions(class, "  ", "s", 60, false, "Lax")
	})
	expectSessionText(t, `not ":memory:"`, func() {
		AhdHTTPPersistentSessions(class, ":memory:", "s", 60, false, "Lax")
	})
	expectSessionText(t, "directory does not exist", func() {
		AhdHTTPPersistentSessions(class, filepath.Join(directory, "missing", "sessions.db"), "s", 60, false, "Lax")
	})
	expectSessionText(t, "is a directory", func() {
		AhdHTTPPersistentSessions(class, directory, "s", 60, false, "Lax")
	})
	expectSessionText(t, "SameSite", func() {
		AhdHTTPPersistentSessions(class, filepath.Join(directory, "sessions.db"), "s", 60, false, "None")
	})

	foreign := filepath.Join(directory, "foreign.db")
	handle, err := SQLiteOpen(foreign)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SQLiteExecute(handle, `CREATE TABLE ahd_session_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := SQLiteExecute(handle, `INSERT INTO ahd_session_meta (key, value) VALUES (?, ?)`, []string{SQLiteFromString("schema"), SQLiteFromString("2")}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = SQLiteClose(handle) })
	expectSessionText(t, "schema version 2 is not supported", func() {
		AhdHTTPPersistentSessions(class, foreign, "s", 60, false, "Lax")
	})
	expectSessionText(t, "Automatic migration is not performed", func() {
		AhdHTTPPersistentSessions(class, foreign, "s", 60, false, "Lax")
	})
}

func TestPersistentSessionRejectsCorruptAndOversizedData(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	handle, err := SQLiteOpen(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SQLiteExecute(handle, `UPDATE ahd_sessions SET values_json = ?`, []string{SQLiteFromString("{")}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = SQLiteClose(handle) })
	expectSessionText(t, "HTTP session data is corrupted", func() {
		AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	})
	expectSessionText(t, "HTTP session value is too large", func() {
		AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "blob", strings.Repeat("x", ahdHTTPSessionValueMaxBytes+1))
	})
	expectSessionText(t, "HTTP session key is too long", func() {
		AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), strings.Repeat("k", ahdHTTPSessionKeyMaxBytes+1), "v")
	})
}

func TestPersistentSessionUnicodePath(t *testing.T) {
	class := AhdClassHTTPError
	directory := filepath.Join(t.TempDir(), "Oturum Klasörü")
	usePersistentHelper(t)
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "kalıcı oturum.db")
	store := AhdHTTPPersistentSessions(class, path, "ahd_session", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "şehir", "İstanbul"))
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	restarted := AhdHTTPPersistentSessions(class, path, "ahd_session", 3600, false, "Lax")
	got := AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, restarted, sessionRequest("ahd_session", id)), "şehir")
	if got == nil || *got != "İstanbul" {
		t.Fatalf("unicode session = %v", got)
	}
}

func sessionHelperExecutable() string {
	if runtime.GOOS == "windows" {
		return "ahdsqlite.exe"
	}
	return "ahdsqlite"
}

func TestSessionHelperNameMatchesTheHost(t *testing.T) {
	name := sessionHelperExecutable()
	switch runtime.GOOS {
	case "windows":
		if name != "ahdsqlite.exe" {
			t.Fatalf("windows helper = %s", name)
		}
	case "linux", "darwin":
		if name != "ahdsqlite" {
			t.Fatalf("posix helper = %s", name)
		}
	default:
		if strings.HasSuffix(name, ".exe") {
			t.Fatalf("non-windows helper = %s", name)
		}
	}
}

func TestPersistentSessionUnknownCookieAllocatesNothing(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 60, false, "Lax")
	for _, cookie := range []string{"", "not a session", strings.Repeat("a", 43)} {
		session := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", cookie))
		AhdHTTPSessionStoreCommit(class, store, session, AhdHTTPText(class, "ok", 200))
	}
	if ahdHTTPTestStoreSize(store) != 0 {
		t.Fatalf("unknown cookies allocated %d sessions", ahdHTTPTestStoreSize(store))
	}
}

func sessionCounts(t *testing.T, path, namespace string) (live int, revoked int) {
	t.Helper()
	handle, err := SQLiteOpen(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = SQLiteClose(handle) }()
	count := func(table string) int {
		t.Helper()
		_, rows, err := SQLiteQuery(handle, `SELECT COUNT(*) AS total FROM `+table+` WHERE namespace = ?`, []string{ahdHTTPSessionText(namespace)})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || len(rows[0]) != 1 {
			t.Fatalf("%s rows = %#v", table, rows)
		}
		value, err := SQLiteValueInt(rows[0][0])
		if err != nil {
			t.Fatal(err)
		}
		return int(value)
	}
	return count("ahd_sessions"), count("ahd_session_revoked")
}

func sessionUser(class *AhdClass, store, name, id string) *string {
	return AhdHTTPSessionGet(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest(name, id)), "user")
}

func TestPersistentSessionStaleRotateCannotResurrect(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	other := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	held := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	heldByOther := AhdHTTPSessionStoreOpen(class, other, sessionRequest("s", id))

	t.Run("rotate versus destroy", func(t *testing.T) {
		destroyed := AhdHTTPSessionStoreCommit(class, store, AhdHTTPSessionDestroy(class, held), AhdHTTPText(class, "out", 200))
		wire := ahdHTTPCookieWire(ahdHTTPDecodeResponse(class, destroyed).Cookies[0])
		if !strings.Contains(wire, "Max-Age=0") || !strings.Contains(wire, "HttpOnly") || !strings.Contains(wire, "SameSite=Lax") {
			t.Fatalf("deletion cookie = %s", wire)
		}
		before := ahdHTTPDecodeSession(class, heldByOther).ID
		expectSessionText(t, "HTTP session is no longer valid", func() {
			AhdHTTPSessionRotate(class, heldByOther)
		})
		if ahdHTTPDecodeSession(class, heldByOther).ID != before {
			t.Fatal("failed rotate changed the caller's session id")
		}
		if sessionUser(class, other, "s", id) != nil {
			t.Fatal("destroyed cookie authenticated")
		}
		live, revoked := sessionCounts(t, path, "s")
		if live != 0 || revoked != 1 {
			t.Fatalf("after destroy, live=%d revoked=%d", live, revoked)
		}
	})
}

func TestPersistentSessionTwoRotatesOneWinner(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	peer := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	first := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	second := AhdHTTPSessionStoreOpen(class, peer, sessionRequest("s", id))
	rotated := AhdHTTPSessionRotate(class, first)
	expectSessionText(t, "HTTP session is no longer valid", func() {
		AhdHTTPSessionRotate(class, second)
	})
	newID := sessionCookie(t, class, store, rotated)
	if newID == id {
		t.Fatal("winner kept the old identifier")
	}
	if sessionUser(class, store, "s", id) != nil {
		t.Fatal("old cookie authenticated after rotation")
	}
	if got := sessionUser(class, store, "s", newID); got == nil || *got != "Ali" {
		t.Fatalf("new cookie = %v", got)
	}
	live, revoked := sessionCounts(t, path, "s")
	if live != 1 || revoked != 1 {
		t.Fatalf("after one rotation, live=%d revoked=%d", live, revoked)
	}
}

func TestPersistentSessionConcurrentRotateOneWinner(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	left := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	right := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, left, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, left, empty), "user", "Ali"))
	copies := []string{
		AhdHTTPSessionStoreOpen(class, left, sessionRequest("s", id)),
		AhdHTTPSessionStoreOpen(class, right, sessionRequest("s", id)),
	}
	owners := []string{left, right}
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	var winners []string
	failures := 0
	wg.Add(2)
	for index := 0; index < 2; index++ {
		go func(index int) {
			defer wg.Done()
			<-start
			defer func() {
				if recover() != nil {
					mu.Lock()
					failures++
					mu.Unlock()
				}
			}()
			rotated := AhdHTTPSessionRotate(class, copies[index])
			mu.Lock()
			winners = append(winners, owners[index]+"\n"+rotated)
			mu.Unlock()
		}(index)
	}
	close(start)
	wg.Wait()
	if len(winners) != 1 || failures != 1 {
		t.Fatalf("winners=%d failures=%d", len(winners), failures)
	}
	parts := strings.SplitN(winners[0], "\n", 2)
	newID := sessionCookie(t, class, parts[0], parts[1])
	if sessionUser(class, left, "s", id) != nil || sessionUser(class, right, "s", newID) == nil {
		t.Fatal("concurrent rotation did not leave exactly the replacement cookie")
	}
	live, revoked := sessionCounts(t, path, "s")
	if live != 1 || revoked != 1 {
		t.Fatalf("concurrent rotation live=%d revoked=%d", live, revoked)
	}
}

func TestPersistentSessionRotateAfterExpiryDoesNotRecreate(t *testing.T) {
	class := AhdClassHTTPError
	defer ahdHTTPTestResetClock()
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	ahdHTTPTimeNow = func() time.Time { return now }
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 1, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	held := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	ahdHTTPTimeNow = func() time.Time { return now.Add(2 * time.Second) }
	expectSessionText(t, "HTTP session is no longer valid", func() {
		AhdHTTPSessionRotate(class, held)
	})
	live, _ := sessionCounts(t, path, "s")
	if live != 1 {
		t.Fatalf("failed rotate changed the expired row, live=%d", live)
	}
	if sessionUser(class, store, "s", id) != nil {
		t.Fatal("expired cookie authenticated")
	}
	restarted := AhdHTTPPersistentSessions(class, path, "s", 1, false, "Lax")
	if sessionUser(class, restarted, "s", id) != nil {
		t.Fatal("expired cookie authenticated after a new store")
	}
}

func TestPersistentSessionRotateAndDestroyBeatStaleCommit(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	peer := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	stale := AhdHTTPSessionStoreOpen(class, peer, sessionRequest("s", id))
	rotated := AhdHTTPSessionRotate(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id)))
	newID := sessionCookie(t, class, store, rotated)
	expectSessionText(t, "HTTP session is no longer valid", func() {
		AhdHTTPSessionStoreCommit(class, peer, AhdHTTPSessionSet(class, stale, "user", "intruder"), AhdHTTPText(class, "no", 200))
	})
	if sessionUser(class, peer, "s", id) != nil {
		t.Fatal("rotated-away cookie authenticated")
	}
	if got := sessionUser(class, peer, "s", newID); got == nil || *got != "Ali" {
		t.Fatalf("replacement = %v", got)
	}

	again := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Veli"))
	stale = AhdHTTPSessionStoreOpen(class, peer, sessionRequest("s", again))
	AhdHTTPSessionStoreCommit(class, store, AhdHTTPSessionDestroy(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", again))), AhdHTTPText(class, "out", 200))
	expectSessionText(t, "HTTP session is no longer valid", func() {
		AhdHTTPSessionStoreCommit(class, peer, AhdHTTPSessionSet(class, stale, "user", "intruder"), AhdHTTPText(class, "no", 200))
	})
	if sessionUser(class, peer, "s", again) != nil {
		t.Fatal("destroyed cookie authenticated from the other connection")
	}
}

func TestPersistentSessionAbandonedRotateStaysRevoked(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	rotated := AhdHTTPSessionRotate(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id)))
	replacement := ahdHTTPDecodeSession(class, rotated).ID
	if err := ahdHTTPTestClosePersistentStores(); err != nil {
		t.Fatal(err)
	}
	restarted := AhdHTTPPersistentSessions(class, path, "s", 3600, false, "Lax")
	if sessionUser(class, restarted, "s", id) != nil {
		t.Fatal("old cookie survived an abandoned rotate")
	}
	if sessionUser(class, restarted, "s", replacement) != nil {
		t.Fatal("uncommitted replacement authenticated after the store was closed")
	}
	live, revoked := sessionCounts(t, path, "s")
	if live != 0 || revoked != 1 {
		t.Fatalf("abandoned rotate left live=%d revoked=%d", live, revoked)
	}
}

func TestMemorySessionStaleRotateCannotResurrect(t *testing.T) {
	class := AhdClassHTTPError
	store := AhdHTTPSessions(class, "s", 60, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	first := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	second := AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	AhdHTTPSessionStoreCommit(class, store, AhdHTTPSessionDestroy(class, first), AhdHTTPText(class, "out", 200))
	expectSessionText(t, "HTTP session is no longer valid", func() {
		AhdHTTPSessionRotate(class, second)
	})
	if sessionUser(class, store, "s", id) != nil {
		t.Fatal("destroyed memory session authenticated")
	}

	id = sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	first = AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	second = AhdHTTPSessionStoreOpen(class, store, sessionRequest("s", id))
	winner := AhdHTTPSessionRotate(class, first)
	expectSessionText(t, "HTTP session is no longer valid", func() {
		AhdHTTPSessionRotate(class, second)
	})
	newID := sessionCookie(t, class, store, winner)
	if sessionUser(class, store, "s", id) != nil {
		t.Fatal("old memory cookie survived rotation")
	}
	if got := sessionUser(class, store, "s", newID); got == nil || *got != "Ali" {
		t.Fatalf("memory replacement = %v", got)
	}

	defer ahdHTTPTestResetClock()
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	ahdHTTPTimeNow = func() time.Time { return now }
	expiring := AhdHTTPSessions(class, "e", 1, false, "Lax")
	id = sessionCookie(t, class, expiring, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, expiring, empty), "user", "Ali"))
	held := AhdHTTPSessionStoreOpen(class, expiring, sessionRequest("e", id))
	ahdHTTPTimeNow = func() time.Time { return now.Add(2 * time.Second) }
	expectSessionText(t, "HTTP session is no longer valid", func() {
		AhdHTTPSessionRotate(class, held)
	})
	if sessionUser(class, expiring, "e", id) != nil {
		t.Fatal("expired memory cookie authenticated")
	}
}

func TestSessionDeletionCookieKeepsSecurityAttributes(t *testing.T) {
	class := AhdClassHTTPError
	cases := []struct {
		name     string
		secure   bool
		sameSite string
		want     []string
		absent   []string
	}{
		{name: "ahd_session", secure: false, sameSite: "Lax", want: []string{"ahd_session=", "Path=/", "HttpOnly", "SameSite=Lax", "Max-Age=0"}, absent: []string{"Secure", "Domain="}},
		{name: "ahd_session", secure: true, sameSite: "Strict", want: []string{"Secure", "HttpOnly", "SameSite=Strict", "Path=/", "Max-Age=0"}, absent: []string{"Domain="}},
		{name: "ahd_session", secure: true, sameSite: "None", want: []string{"Secure", "HttpOnly", "SameSite=None", "Path=/", "Max-Age=0"}, absent: []string{"Domain="}},
		{name: "__Host-ahd", secure: true, sameSite: "Lax", want: []string{"__Host-ahd=", "Secure", "HttpOnly", "Path=/", "Max-Age=0"}, absent: []string{"Domain="}},
		{name: "__Secure-ahd", secure: true, sameSite: "Strict", want: []string{"__Secure-ahd=", "Secure", "HttpOnly", "SameSite=Strict", "Max-Age=0"}, absent: []string{"Domain="}},
	}
	for _, item := range cases {
		store := AhdHTTPSessions(class, item.name, 60, item.secure, item.sameSite)
		empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
		session := AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali")
		id := sessionCookie(t, class, store, session)
		cleared := AhdHTTPSessionStoreCommit(class, store, AhdHTTPSessionDestroy(class, AhdHTTPSessionStoreOpen(class, store, sessionRequest(item.name, id))), AhdHTTPText(class, "out", 200))
		wire := ahdHTTPCookieWire(ahdHTTPDecodeResponse(class, cleared).Cookies[0])
		for _, want := range item.want {
			if !strings.Contains(wire, want) {
				t.Errorf("%s cookie %q missing %q", item.name, wire, want)
			}
		}
		for _, absent := range item.absent {
			if strings.Contains(wire, absent) {
				t.Errorf("%s cookie %q contains %q", item.name, wire, absent)
			}
		}
	}
	expectSessionText(t, "requires Secure=true", func() {
		AhdHTTPSessions(class, "__Host-ahd", 60, false, "Lax")
	})
	expectSessionText(t, "requires Secure=true", func() {
		AhdHTTPSessions(class, "__Secure-ahd", 60, false, "Strict")
	})
}

func TestPersistentSessionRelativePathUsesTheWorkingDirectory(t *testing.T) {
	class := AhdClassHTTPError
	directory := filepath.Join(t.TempDir(), "Oturum Klasörü")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(directory)
	usePersistentHelper(t)
	store := AhdHTTPPersistentSessions(class, "kalıcı oturum.db", "s", 60, false, "Lax")
	path := filepath.Join(directory, "kalıcı oturum.db")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	if err := ahdHTTPTestClosePersistentStores(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("session file stayed locked after release: %v", err)
	}
	reopened := AhdHTTPPersistentSessions(class, path, "s", 60, false, "Lax")
	if sessionUser(class, reopened, "s", id) != nil {
		t.Fatal("removing the file left the old cookie authenticated")
	}
}

func TestPersistentSessionReleaseUnlocksTheFile(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "s", 60, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	id := sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", "Ali"))
	AhdHTTPReleasePersistentSessionStorage()
	if err := os.Remove(path); err != nil {
		t.Fatalf("release left the database locked: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("released database still exists: %v", err)
	}
	restored := AhdHTTPPersistentSessions(class, path, "s", 60, false, "Lax")
	if sessionUser(class, restored, "s", id) != nil {
		t.Fatal("a deleted database authenticated the old cookie")
	}
}

func sessionDatabaseHeld(t *testing.T, path string) bool {
	t.Helper()
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	ahdHTTPSessionDatabasesMu.Lock()
	database := ahdHTTPSessionDatabases[absolute]
	ahdHTTPSessionDatabasesMu.Unlock()
	if database == nil {
		return false
	}
	database.mutex.Lock()
	defer database.mutex.Unlock()
	return database.handle != ""
}

func commitNamedSession(t *testing.T, path, user string) string {
	t.Helper()
	class := AhdClassHTTPError
	store := AhdHTTPPersistentSessions(class, path, "ahd_session", 3600, false, "Lax")
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	return sessionCookie(t, class, store, AhdHTTPSessionSet(class, AhdHTTPSessionStoreOpen(class, store, empty), "user", user))
}

func expectNamedSession(t *testing.T, path, id, user string) {
	t.Helper()
	class := AhdClassHTTPError
	store := AhdHTTPPersistentSessions(class, path, "ahd_session", 3600, false, "Lax")
	got := sessionUser(class, store, "ahd_session", id)
	if got == nil || *got != user {
		t.Fatalf("user = %v, want %s", got, user)
	}
}

func startOwningServer(t *testing.T) (port int) {
	t.Helper()
	port = freeLoopbackPort(t)
	handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	go func() {
		defer func() { _ = recover() }()
		AhdHTTPServerStart(AhdClassHTTPError, handle)
	}()
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			t.Cleanup(func() { _ = AhdHTTPShutdownAddress("127.0.0.1", port) })
			return port
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("HTTP server did not accept a connection")
	return port
}

func stopOwningServer(t *testing.T, port int) {
	t.Helper()
	if !AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("shutdown did not claim the running server")
	}
}

func TestPersistentSessionUnstartedServerDoesNotPinTheFile(t *testing.T) {
	path := persistentSessionPath(t)
	id := commitNamedSession(t, path, "Ali")
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("opening a persistent store did not hold the database")
	}
	idlePort := freeLoopbackPort(t)
	idle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(idlePort), ahdHTTPDefaultMaxBody)
	t.Cleanup(func() {
		ahdHTTPServersMu.Lock()
		delete(ahdHTTPServers, idle)
		ahdHTTPServersMu.Unlock()
	})
	if AhdHTTPServerActive() {
		t.Fatal("an unstarted server counted as active")
	}
	port := startOwningServer(t)
	if !AhdHTTPServerActive() {
		t.Fatal("a running server did not count as active")
	}
	stopOwningServer(t, port)
	if AhdHTTPServerActive() {
		t.Fatal("stopping the running server left an active owner")
	}
	if sessionDatabaseHeld(t, path) {
		t.Fatal("the unstarted server kept the database open")
	}
	expectNamedSession(t, path, id, "Ali")
	AhdHTTPReleasePersistentSessionStorage()
	if err := os.Remove(path); err != nil {
		t.Fatalf("the session file stayed locked after the last owner stopped: %v", err)
	}
}

func TestPersistentSessionSharedDatabaseTracksRunningServers(t *testing.T) {
	path := persistentSessionPath(t)
	id := commitNamedSession(t, path, "Mehmet")
	first := startOwningServer(t)
	second := startOwningServer(t)
	stopOwningServer(t, first)
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("stopping one of two running servers released the shared database")
	}
	stopOwningServer(t, second)
	if sessionDatabaseHeld(t, path) {
		t.Fatal("the last running server left the database open")
	}
	expectNamedSession(t, path, id, "Mehmet")
	for range 4 {
		port := startOwningServer(t)
		stopOwningServer(t, port)
		if sessionDatabaseHeld(t, path) {
			t.Fatal("a stop/start cycle left the database open")
		}
	}
	expectNamedSession(t, path, id, "Mehmet")
	AhdHTTPReleasePersistentSessionStorage()
	if err := os.Remove(path); err != nil {
		t.Fatalf("repeated server cycles left the database locked: %v", err)
	}
}

func TestPersistentSessionEvaluatorEndReleasesWithoutAServer(t *testing.T) {
	path := persistentSessionPath(t)
	AhdHTTPEvaluatorBegin()
	ended := false
	end := func() {
		if !ended {
			ended = true
			AhdHTTPEvaluatorEnd()
		}
	}
	t.Cleanup(end)
	id := commitNamedSession(t, path, "Ayşe")
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("the evaluator execution did not hold the database")
	}
	idlePort := freeLoopbackPort(t)
	idle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(idlePort), ahdHTTPDefaultMaxBody)
	t.Cleanup(func() {
		ahdHTTPServersMu.Lock()
		delete(ahdHTTPServers, idle)
		ahdHTTPServersMu.Unlock()
	})
	port := startOwningServer(t)
	stopOwningServer(t, port)
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("the server stop released a database the evaluator was still using")
	}
	end()
	if sessionDatabaseHeld(t, path) {
		t.Fatal("the evaluator left the database open after it finished")
	}
	expectNamedSession(t, path, id, "Ayşe")
	AhdHTTPReleasePersistentSessionStorage()
	if err := os.Remove(path); err != nil {
		t.Fatalf("the session file stayed locked after the evaluator finished: %v", err)
	}
}

func TestPersistentSessionEvaluatorEndLeavesARunningServerAlone(t *testing.T) {
	path := persistentSessionPath(t)
	AhdHTTPEvaluatorBegin()
	ended := false
	end := func() {
		if !ended {
			ended = true
			AhdHTTPEvaluatorEnd()
		}
	}
	t.Cleanup(end)
	id := commitNamedSession(t, path, "Ali")
	port := startOwningServer(t)
	end()
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("ending the evaluator closed a database a running server still owns")
	}
	stopOwningServer(t, port)
	if sessionDatabaseHeld(t, path) {
		t.Fatal("the last running server did not release the database")
	}
	expectNamedSession(t, path, id, "Ali")
	AhdHTTPReleasePersistentSessionStorage()
	if err := os.Remove(path); err != nil {
		t.Fatalf("the session file stayed locked after the server stopped: %v", err)
	}
}

func TestPersistentSessionTwoEvaluatorsReleaseOnTheLastEnd(t *testing.T) {
	path := persistentSessionPath(t)
	AhdHTTPEvaluatorBegin()
	AhdHTTPEvaluatorBegin()
	remaining := 2
	end := func() {
		if remaining > 0 {
			remaining--
			AhdHTTPEvaluatorEnd()
		}
	}
	t.Cleanup(func() {
		for remaining > 0 {
			end()
		}
	})
	id := commitNamedSession(t, path, "Ali")
	end()
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("the first evaluator to finish released the other execution's database")
	}
	end()
	if sessionDatabaseHeld(t, path) {
		t.Fatal("the last evaluator left the database open")
	}
	expectNamedSession(t, path, id, "Ali")
	AhdHTTPReleasePersistentSessionStorage()
	if err := os.Remove(path); err != nil {
		t.Fatalf("the session file stayed locked after both evaluators finished: %v", err)
	}
}

func saveSessionUser(class *AhdClass, store, user string) string {
	empty := ahdHTTPEncodeRequest(ahdHTTPRequestData{Method: "GET", Path: "/"})
	opened := AhdHTTPSessionStoreOpen(class, store, empty)
	committed := AhdHTTPSessionStoreCommit(class, store, AhdHTTPSessionSet(class, opened, "user", user), AhdHTTPText(class, "ok", 200))
	cookies := ahdHTTPDecodeResponse(class, committed).Cookies
	if len(cookies) != 1 {
		panic("session commit did not return one cookie")
	}
	return cookies[0].Value
}

func listenSessionServer(t *testing.T, class *AhdClass, port int, handle string) <-chan struct{} {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		AhdHTTPServerStart(class, handle)
	}()
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return done
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("HTTP server did not accept a connection")
	return done
}

func fetchSessionPath(port int, path string) error {
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	response, err := client.Get("http://127.0.0.1:" + strconv.Itoa(port) + path)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %d", path, response.StatusCode)
	}
	return nil
}

func waitUntilSessionFileReleased(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for sessionDatabaseHeld(t, path) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
}

func TestPersistentSessionGracefulShutdownReleasesAHandlerReopen(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "ahd_session", 3600, false, "Lax")
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(class, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	entered := make(chan struct{})
	resume := make(chan struct{})
	var resumeOnce sync.Once
	resumeHandler := func() { resumeOnce.Do(func() { close(resume) }) }
	var id string
	AhdHTTPServerGet(class, handle, "/save", func(string) string {
		close(entered)
		<-resume
		id = saveSessionUser(class, store, "Ali")
		return AhdHTTPText(class, "saved", 200)
	})
	started := listenSessionServer(t, class, port, handle)
	t.Cleanup(func() {
		resumeHandler()
		_ = AhdHTTPShutdownAddress("127.0.0.1", port)
		<-started
	})
	requestDone := make(chan error, 1)
	go func() { requestDone <- fetchSessionPath(port, "/save") }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the handler did not enter")
	}
	shutdownDone := make(chan bool, 1)
	go func() { shutdownDone <- AhdHTTPShutdownAddress("127.0.0.1", port) }()
	deadline := time.Now().Add(2 * time.Second)
	for AhdHTTPServerActive() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if AhdHTTPServerActive() {
		t.Fatal("shutdown did not retire the server while the handler was still running")
	}
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("graceful shutdown released the database before the handler returned")
	}
	resumeHandler()
	select {
	case stopped := <-shutdownDone:
		if !stopped {
			t.Fatal("graceful shutdown did not claim the server")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("graceful shutdown did not return after the handler finished")
	}
	select {
	case err := <-requestDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the handler request did not finish")
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("Server.start did not return")
	}
	if sessionDatabaseHeld(t, path) {
		t.Fatal("the handler reopened the database and shutdown left it open")
	}
	expectNamedSession(t, path, id, "Ali")
	AhdHTTPReleasePersistentSessionStorage()
	if err := os.Remove(path); err != nil {
		t.Fatalf("Windows could not delete the database after graceful shutdown: %v", err)
	}
}

func TestPersistentSessionForcedShutdownReleasesAfterTheHandlerReturns(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "ahd_session", 3600, false, "Lax")
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(class, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	entered := make(chan struct{})
	resume := make(chan struct{})
	var resumeOnce sync.Once
	resumeHandler := func() { resumeOnce.Do(func() { close(resume) }) }
	var id string
	committed := make(chan struct{})
	AhdHTTPServerGet(class, handle, "/save", func(string) string {
		id = saveSessionUser(class, store, "Ali")
		close(entered)
		<-resume
		id = saveSessionUser(class, store, "Ayşe")
		close(committed)
		return AhdHTTPText(class, "saved", 200)
	})
	started := listenSessionServer(t, class, port, handle)
	t.Cleanup(func() {
		resumeHandler()
		_ = AhdHTTPShutdownAddress("127.0.0.1", port)
		<-started
	})
	go func() { _ = fetchSessionPath(port, "/save") }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the handler did not enter")
	}
	shutdownStarted := time.Now()
	shutdownDone := make(chan bool, 1)
	go func() { shutdownDone <- AhdHTTPShutdownAddress("127.0.0.1", port) }()
	var stopped bool
	select {
	case stopped = <-shutdownDone:
	case <-time.After(4 * time.Second):
		t.Fatal("forced shutdown did not return")
	}
	elapsed := time.Since(shutdownStarted)
	if !stopped {
		t.Fatal("forced shutdown did not claim the server")
	}
	if elapsed < 1500*time.Millisecond || elapsed > 3500*time.Millisecond {
		t.Fatalf("shutdown took %s; graceful shutdown is 2s, then Close", elapsed)
	}
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("forced shutdown closed the database while the handler was still running")
	}
	resumeHandler()
	select {
	case <-committed:
	case <-time.After(2 * time.Second):
		t.Fatal("the handler did not commit after forced shutdown")
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("Server.start did not return after the handler")
	}
	waitUntilSessionFileReleased(t, path)
	if sessionDatabaseHeld(t, path) {
		t.Fatal("the handler's later commit left the database open")
	}
	expectNamedSession(t, path, id, "Ayşe")
	AhdHTTPReleasePersistentSessionStorage()
	if err := os.Remove(path); err != nil {
		t.Fatalf("Windows could not delete the database after the forced-shutdown handler returned: %v", err)
	}
}

func TestPersistentSessionShutdownKeepsAnotherServersDatabase(t *testing.T) {
	class := AhdClassHTTPError
	path := persistentSessionPath(t)
	store := AhdHTTPPersistentSessions(class, path, "ahd_session", 3600, false, "Lax")
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(class, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	entered := make(chan struct{})
	resume := make(chan struct{})
	var resumeOnce sync.Once
	resumeHandler := func() { resumeOnce.Do(func() { close(resume) }) }
	var id string
	AhdHTTPServerGet(class, handle, "/save", func(string) string {
		close(entered)
		<-resume
		id = saveSessionUser(class, store, "Mehmet")
		return AhdHTTPText(class, "saved", 200)
	})
	started := listenSessionServer(t, class, port, handle)
	other := startOwningServer(t)
	t.Cleanup(func() {
		resumeHandler()
		_ = AhdHTTPShutdownAddress("127.0.0.1", port)
		<-started
	})
	go func() { _ = fetchSessionPath(port, "/save") }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the handler did not enter")
	}
	shutdownDone := make(chan bool, 1)
	go func() { shutdownDone <- AhdHTTPShutdownAddress("127.0.0.1", port) }()
	resumeHandler()
	select {
	case stopped := <-shutdownDone:
		if !stopped {
			t.Fatal("shutdown did not claim the first server")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not return")
	}
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("stopping one server released the database still owned by the other")
	}
	stopOwningServer(t, other)
	if sessionDatabaseHeld(t, path) {
		t.Fatal("the last server left the database open")
	}
	expectNamedSession(t, path, id, "Mehmet")
	AhdHTTPReleasePersistentSessionStorage()
	if err := os.Remove(path); err != nil {
		t.Fatalf("Windows could not delete the database after the last server stopped: %v", err)
	}
}

func TestMigrateDoesNotDisturbPersistentSessions(t *testing.T) {
	path := persistentSessionPath(t)
	id := commitNamedSession(t, path, "Ali")
	directory := filepath.Join(filepath.Dir(path), "Göç Klasörü")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "0001_create_notes.up.sql"), []byte("CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	applied, err := migrate.Up(migrate.Options{Database: path, Directory: directory})
	if err != nil || len(applied) != 1 {
		t.Fatalf("migrate %+v %v", applied, err)
	}
	expectNamedSession(t, path, id, "Ali")
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("reading the session did not reopen its database")
	}
}

func TestPersistentSessionScopedEvaluatorReleasesIndependentlyOfUnrelatedServer(t *testing.T) {
	usePersistentHelper(t)
	unrelated := startOwningServer(t)
	defer stopOwningServer(t, unrelated)

	unscopedPath := filepath.Join(t.TempDir(), "unscoped.db")
	unscopedID := commitNamedSession(t, unscopedPath, "Unscoped")
	if !sessionDatabaseHeld(t, unscopedPath) {
		t.Fatal("unscoped database was not held")
	}

	scopedPath := filepath.Join(t.TempDir(), "scoped.db")
	scopeID := AhdHTTPEvaluatorScopeBegin(nil)
	store := AhdHTTPPersistentSessionsScoped(AhdClassHTTPError, scopeID, scopedPath, "ahd_session", 3600, false, "Lax")
	id := saveSessionUser(AhdClassHTTPError, store, "Scoped")
	if !sessionDatabaseHeld(t, scopedPath) {
		t.Fatal("scoped database was not held while scope was active")
	}

	AhdHTTPEvaluatorScopeEnd(scopeID)
	if sessionDatabaseHeld(t, scopedPath) {
		t.Fatal("ending the scope left the scoped database open while an unrelated server ran")
	}
	if err := os.Remove(scopedPath); err != nil {
		t.Fatalf("scoped database stayed locked while unrelated server ran: %v", err)
	}
	if !sessionDatabaseHeld(t, unscopedPath) {
		t.Fatal("ending an unrelated scope closed an unscoped database still owned by a running server")
	}
	_ = id
	_ = unscopedID
}

func TestPersistentSessionScopedForcedShutdownHandsOffToActiveHandler(t *testing.T) {
	usePersistentHelper(t)
	path := filepath.Join(t.TempDir(), "handoff.db")
	class := AhdClassHTTPError
	scopeID := AhdHTTPEvaluatorScopeBegin(nil)
	endedScope := false
	endScope := func() {
		if !endedScope {
			endedScope = true
			AhdHTTPEvaluatorScopeEnd(scopeID)
		}
	}
	t.Cleanup(endScope)

	store := AhdHTTPPersistentSessionsScoped(class, scopeID, path, "ahd_session", 3600, false, "Lax")
	port := freeLoopbackPort(t)
	handle := AhdHTTPServer(class, "127.0.0.1", int64(port), ahdHTTPDefaultMaxBody)
	entered := make(chan struct{})
	resume := make(chan struct{})
	var resumeOnce sync.Once
	resumeHandler := func() { resumeOnce.Do(func() { close(resume) }) }
	committed := make(chan struct{})
	var id string
	AhdHTTPServerGet(class, handle, "/save", func(string) string {
		id = saveSessionUser(class, store, "Before")
		close(entered)
		<-resume
		id = saveSessionUser(class, store, "After")
		close(committed)
		return AhdHTTPText(class, "saved", 200)
	})

	started := make(chan struct{})
	go func() {
		defer close(started)
		defer func() { _ = recover() }()
		AhdHTTPServerStartScoped(class, scopeID, handle)
	}()
	requestDone := make(chan struct{})
	t.Cleanup(func() {
		resumeHandler()
		_ = AhdHTTPShutdownAddress("127.0.0.1", port)
		<-started
	})
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	go func() {
		defer close(requestDone)
		_ = fetchSessionPath(port, "/save")
	}()
	t.Cleanup(func() {
		resumeHandler()
		<-requestDone
	})
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not enter")
	}

	shutdownDone := make(chan bool, 1)
	go func() { shutdownDone <- AhdHTTPShutdownAddress("127.0.0.1", port) }()
	select {
	case stopped := <-shutdownDone:
		if !stopped {
			t.Fatal("forced shutdown did not claim server")
		}
	case <-time.After(4 * time.Second):
		t.Fatal("forced shutdown did not return")
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("Server.start did not return after forced shutdown")
	}

	// End the evaluator scope BEFORE the blocked handler returns: the database
	// must be handed off to the active handler, not closed out from under it.
	endScope()
	if !sessionDatabaseHeld(t, path) {
		t.Fatal("ending scope closed the database while an active handler still needed it")
	}

	resumeHandler()
	select {
	case <-committed:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not finish committing")
	}
	waitUntilSessionFileReleased(t, path)
	if sessionDatabaseHeld(t, path) {
		t.Fatal("handler completion left the scoped database open")
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("Windows could not delete handed-off scoped database: %v", err)
	}
	_ = id
}
