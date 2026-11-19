package build

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWebPersistentSessionsCompiles(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `bring Web
from Web bring SessionStore

sessions: SessionStore := Web.persistentSessions("sessions.db", "ahdsession", 3600, false, "Lax")
`})
	result := Compile(filepath.Join(directory, "main.ahd"))
	if result.HasErrors() {
		t.Fatalf("Web.persistentSessions failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if result.Program == nil || !result.Program.RequiresSQLite {
		t.Fatal("Web.persistentSessions must require the SQLite helper")
	}
}

func requireSQLite(t *testing.T, source string, want bool) {
	t.Helper()
	directory := writeSources(t, map[string]string{"main.ahd": source})
	result := Compile(filepath.Join(directory, "main.ahd"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if result.Program == nil || result.Program.RequiresSQLite != want {
		t.Fatalf("RequiresSQLite = %v, want %v", result.Program != nil && result.Program.RequiresSQLite, want)
	}
}

func TestSessionHelperDependencyFollowsUse(t *testing.T) {
	requireSQLite(t, "bring Web\nwrite(Web.render(Web.UI.text(\"ok\")))\n", false)
	requireSQLite(t, "bring Web\nfrom Web bring SessionStore\nsessions: SessionStore := Web.sessions()\n", false)
	requireSQLite(t, "bring HTTP\nfrom HTTP bring SessionStore\nsessions: SessionStore := HTTP.sessions()\n", false)
	requireSQLite(t, "bring HTTP\nfrom HTTP bring SessionStore\nsessions: SessionStore := HTTP.persistentSessions(\"sessions.db\")\n", true)
	requireSQLite(t, "bring Web\nfrom Web bring SessionStore\nsessions: SessionStore := Web.persistentSessions(\"sessions.db\", \"ahdsession\", 3600, false, \"Lax\")\n", true)
	requireSQLite(t, "bring SQLite\nfrom SQLite bring Database\ndb: Database := SQLite.open(\":memory:\")\ndb.close()\n", true)
	requireSQLite(t, `bring Web
from Web bring SessionStore
unused: Function := (path: String) -> SessionStore {
    return Web.persistentSessions(path, "ahdsession", 3600, false, "Lax")
}
write(Web.render(Web.UI.text("ok")))
`, false)
	requireSQLite(t, `bring Web
from Web bring SessionStore
openStore: Function := (path: String) -> SessionStore {
    return Web.persistentSessions(path, "ahdsession", 3600, false, "Lax")
}
store: SessionStore := openStore("sessions.db")
`, true)
}

func TestCStrReachabilityIncludesSQLite(t *testing.T) {
	const ledger = `bring HTTP
from HTTP bring SessionStore

Ledger: Class<> := {
    structure: Attributes := (name: String)
    CStr: Function := () -> String {
        store: Local SessionStore := HTTP.persistentSessions("sessions.db")
        return attribute.name
    }
}

book: Ledger := Ledger(name: "kasa")
`
	requireSQLite(t, ledger+"write(book)\n", true)
	requireSQLite(t, ledger+"text: String := str(book)\nwrite(text)\n", true)
	requireSQLite(t, ledger+"line: String := \"book {book}\"\nwrite(line)\n", true)
	requireSQLite(t, "bring Terminal\n"+ledger+"Terminal.pretty(book)\n", true)
	requireSQLite(t, ledger+"write(\"ok\")\n", false)
	requireSQLite(t, `bring HTTP
from HTTP bring SessionStore

Ledger: Class<> := {
    structure: Attributes := (name: String)
    CStr: Function := () -> String {
        store: Local SessionStore := HTTP.persistentSessions("sessions.db")
        return attribute.name
    }
}

write("ok")
`, false)
	requireSQLite(t, `bring SQLite
from SQLite bring Database

Ledger: Class<> := {
    structure: Attributes := (name: String)
    CStr: Function := () -> String {
        db: Local Database := SQLite.open(":memory:")
        db.close()
        return attribute.name
    }
}

book: Ledger := Ledger(name: "kasa")
write(str(book))
`, true)
	requireSQLite(t, `Ledger: Class<> := {
    structure: Attributes := (name: String)
    CStr: Function := () -> String { return attribute.name }
}

book: Ledger := Ledger(name: "kasa")
write(book)
`, false)
	requireSQLite(t, `bring HTTP
from HTTP bring SessionStore

Animal: Class<> := {
    structure: Attributes := (name: String)
    CStr: Function := () -> String { return attribute.name }
}
Dog: Class<Animal> := {
    structure: Attributes := (SuperClass.attributes)
    CStr: Override Function := () -> String {
        store: Local SessionStore := HTTP.persistentSessions("sessions.db")
        return attribute.name
    }
}
Cat: Class<Animal> := {
    structure: Attributes := (SuperClass.attributes)
    CStr: Override Function := () -> String { return attribute.name }
}

pet: Animal := Dog(name: "Karabaş")
write(pet)
`, true)
	requireSQLite(t, `bring HTTP
from HTTP bring SessionStore

Animal: Class<> := {
    structure: Attributes := (name: String)
    CStr: Function := () -> String { return attribute.name }
}
Dog: Class<Animal> := {
    structure: Attributes := (SuperClass.attributes)
    CStr: Override Function := () -> String { return attribute.name }
}
Cat: Class<Animal> := {
    structure: Attributes := (SuperClass.attributes)
    CStr: Override Function := () -> String {
        store: Local SessionStore := HTTP.persistentSessions("sessions.db")
        return attribute.name
    }
}

pet: Dog := Dog(name: "Karabaş")
write(pet)
`, false)
	requireSQLite(t, "bring Web\nwrite(Web.render(Web.UI.text(\"ok\")))\n", false)
}

func TestPersistentSessionProcessAcceptance(t *testing.T) {
	sqliteHelperForTest(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(root, "examples", "v2.10", "persistent_sessions", "app.ahd")
	output := filepath.Join(t.TempDir(), "persistent-sessions")
	executable, result := BuildProgram(entry, output)
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}

	t.Run("restart", func(t *testing.T) {
		base, database := startPersistentApp(t, executable, 3600)
		client := noRedirectClient(t)
		loginPersistent(t, client, base, "Ali")
		if got := bodyOf(t, client, base+"/panel"); got != "Ali" {
			t.Fatalf("before restart = %q", got)
		}
		stopPersistentApp(t)
		base = startPersistentAppAt(t, executable, database, 3600)
		if got := bodyOf(t, client, base+"/panel"); got != "Ali" {
			t.Fatalf("after restart = %q", got)
		}
	})

	t.Run("two clients and logout", func(t *testing.T) {
		base, database := startPersistentApp(t, executable, 3600)
		clientA := noRedirectClient(t)
		clientB := noRedirectClient(t)
		loginPersistent(t, clientA, base, "Ali")
		loginPersistent(t, clientB, base, "Mehmet")
		if got := bodyOf(t, clientA, base+"/panel"); got != "Ali" {
			t.Fatalf("A = %q", got)
		}
		if got := bodyOf(t, clientB, base+"/panel"); got != "Mehmet" {
			t.Fatalf("B = %q", got)
		}
		cookie := sessionCookieOf(t, clientA, base)
		response, err := clientA.Post(base+"/logout", "application/x-www-form-urlencoded", strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if got := bodyOf(t, clientA, base+"/panel"); got != "anonymous" {
			t.Fatalf("A after logout = %q", got)
		}
		if got := bodyOf(t, clientB, base+"/panel"); got != "Mehmet" {
			t.Fatalf("B after A's logout = %q", got)
		}
		stopPersistentApp(t)
		base = startPersistentAppAt(t, executable, database, 3600)
		if got := replayPanel(t, base, cookie); got != "anonymous" {
			t.Fatalf("A after restart = %q", got)
		}
		if got := bodyOf(t, clientB, base+"/panel"); got != "Mehmet" {
			t.Fatalf("B after restart = %q", got)
		}
	})

	t.Run("expiration", func(t *testing.T) {
		base, database := startPersistentApp(t, executable, 1)
		response := loginRaw(t, base, "Ali")
		cookie := cookieFrom(t, response, "ahd_session")
		if got := replayPanel(t, base, cookie); got != "Ali" {
			t.Fatalf("before expiry = %q", got)
		}
		time.Sleep(1500 * time.Millisecond)
		if got := replayPanel(t, base, cookie); got != "anonymous" {
			t.Fatalf("after expiry = %q", got)
		}
		stopPersistentApp(t)
		base = startPersistentAppAt(t, executable, database, 1)
		if got := replayPanel(t, base, cookie); got != "anonymous" {
			t.Fatalf("expired after restart = %q", got)
		}
	})

	t.Run("rotation", func(t *testing.T) {
		base, database := startPersistentApp(t, executable, 3600)
		response := loginRaw(t, base, "Ali")
		oldCookie := cookieFrom(t, response, "ahd_session")
		client := noRedirectClient(t)
		client.Jar.SetCookies(mustURL(t, base), response.Cookies())
		rotated, err := client.Post(base+"/rotate", "application/x-www-form-urlencoded", strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, rotated.Body)
		_ = rotated.Body.Close()
		newCookie := cookieFrom(t, rotated, "ahd_session")
		if newCookie == oldCookie {
			t.Fatal("rotate did not change the cookie")
		}
		if got := replayPanel(t, base, oldCookie); got != "anonymous" {
			t.Fatalf("old cookie = %q", got)
		}
		if got := replayPanel(t, base, newCookie); got != "Ali" {
			t.Fatalf("new cookie = %q", got)
		}
		stopPersistentApp(t)
		base = startPersistentAppAt(t, executable, database, 3600)
		if got := replayPanel(t, base, oldCookie); got != "anonymous" {
			t.Fatalf("old cookie after restart = %q", got)
		}
		if got := replayPanel(t, base, newCookie); got != "Ali" {
			t.Fatalf("new cookie after restart = %q", got)
		}
	})

	t.Run("concurrent", func(t *testing.T) {
		base, _ := startPersistentApp(t, executable, 3600)
		var wg sync.WaitGroup
		failures := make([]string, 24)
		wg.Add(24)
		for index := 0; index < 24; index++ {
			go func(index int) {
				defer wg.Done()
				name := "user" + strconv.Itoa(index)
				jar, err := cookiejar.New(nil)
				if err != nil {
					failures[index] = err.Error()
					return
				}
				client := &http.Client{Jar: jar, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
					return http.ErrUseLastResponse
				}}
				response, err := client.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader("name="+url.QueryEscape(name)))
				if err != nil {
					failures[index] = err.Error()
					return
				}
				body, _ := io.ReadAll(response.Body)
				_ = response.Body.Close()
				if string(body) != "in" {
					failures[index] = "login " + string(body)
					return
				}
				response, err = client.Get(base + "/panel")
				if err != nil {
					failures[index] = err.Error()
					return
				}
				body, _ = io.ReadAll(response.Body)
				_ = response.Body.Close()
				if string(body) != name {
					failures[index] = string(body)
				}
			}(index)
		}
		wg.Wait()
		for index, got := range failures {
			if got != "" {
				t.Fatalf("client %d: %s", index, got)
			}
		}
	})

	t.Run("storage failure", func(t *testing.T) {
		port := freeLoopbackPort(t)
		directory := filepath.Join(t.TempDir(), "not-a-file")
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		command := exec.Command(executable)
		command.Env = append(os.Environ(),
			"AHD_PORT="+strconv.Itoa(port),
			"AHD_SESSION_DB="+directory,
			"AHD_SESSION_MAX_AGE=3600",
		)
		output, err := command.CombinedOutput()
		if err == nil {
			t.Fatalf("directory database exited 0: %s", output)
		}
		if !strings.Contains(string(output), "is a directory") {
			t.Fatalf("storage failure output = %s", output)
		}
	})

	t.Run("two processes", func(t *testing.T) {
		baseA, database := startPersistentApp(t, executable, 3600)
		baseB := startPersistentPeer(t, executable, database, 3600)
		clientB := noRedirectClient(t)
		loginPersistent(t, clientB, baseB, "Mehmet")
		response := loginRaw(t, baseA, "Ali")
		cookie := cookieFrom(t, response, "ahd_session")
		if got := replayPanel(t, baseB, cookie); got != "Ali" {
			t.Fatalf("process B read A's session as %q", got)
		}
		request, err := http.NewRequest(http.MethodPost, baseA+"/logout", strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Cookie", "ahd_session="+cookie)
		logout, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, logout.Body)
		_ = logout.Body.Close()
		if got := replayPanel(t, baseB, cookie); got != "anonymous" {
			t.Fatalf("process B accepted A's logged-out cookie as %q", got)
		}
		if got := bodyOf(t, clientB, baseB+"/panel"); got != "Mehmet" {
			t.Fatalf("process B's own session = %q", got)
		}
	})
}

func startPersistentPeer(t *testing.T, executable, database string, maxAge int) string {
	t.Helper()
	port := freeLoopbackPort(t)
	command := exec.Command(executable)
	command.Env = append(os.Environ(),
		"AHD_PORT="+strconv.Itoa(port),
		"AHD_SESSION_DB="+database,
		"AHD_SESSION_MAX_AGE="+strconv.Itoa(maxAge),
	)
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
			_, _ = command.Process.Wait()
		}
	})
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(base + "/ok")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				return base
			}
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("peer did not start\n%s", stderr.String())
	return ""
}

var persistentProcess *exec.Cmd

func startPersistentApp(t *testing.T, executable string, maxAge int) (string, string) {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "Oturum Klasörü")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(directory, "kalıcı oturum.db")
	base := startPersistentAppAt(t, executable, database, maxAge)
	return base, database
}

func startPersistentAppAt(t *testing.T, executable, database string, maxAge int) string {
	t.Helper()
	stopPersistentApp(t)
	port := freeLoopbackPort(t)
	command := exec.Command(executable)
	command.Env = append(os.Environ(),
		"AHD_PORT="+strconv.Itoa(port),
		"AHD_SESSION_DB="+database,
		"AHD_SESSION_MAX_AGE="+strconv.Itoa(maxAge),
	)
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	persistentProcess = command
	t.Cleanup(func() { stopPersistentApp(t) })
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(base + "/ok")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				return base
			}
		}
		time.Sleep(30 * time.Millisecond)
	}
	_ = command.Process.Kill()
	t.Fatalf("persistent session app did not start\n%s", stderr.String())
	return ""
}

func stopPersistentApp(t *testing.T) {
	t.Helper()
	if persistentProcess != nil && persistentProcess.Process != nil {
		_ = persistentProcess.Process.Kill()
		_, _ = persistentProcess.Process.Wait()
	}
	persistentProcess = nil
}

func noRedirectClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

func loginPersistent(t *testing.T, client *http.Client, base, name string) {
	t.Helper()
	response, err := client.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader("name="+url.QueryEscape(name)))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 || string(body) != "in" {
		t.Fatalf("login %s = %d %q", name, response.StatusCode, body)
	}
}

func bodyOf(t *testing.T, client *http.Client, rawURL string) string {
	t.Helper()
	response, err := client.Get(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("%s = %d %q", rawURL, response.StatusCode, body)
	}
	return string(body)
}

func loginRaw(t *testing.T, base, name string) *http.Response {
	t.Helper()
	response, err := http.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader("name="+url.QueryEscape(name)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != 200 || string(body) != "in" {
		t.Fatalf("login = %d %q", response.StatusCode, body)
	}
	return response
}

func cookieFrom(t *testing.T, response *http.Response, name string) string {
	t.Helper()
	for _, cookie := range response.Cookies() {
		if cookie.Name == name && cookie.Value != "" {
			return cookie.Value
		}
	}
	t.Fatalf("missing %s cookie in %#v", name, response.Header["Set-Cookie"])
	return ""
}

func replayPanel(t *testing.T, base, cookie string) string {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, base+"/panel", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Cookie", "ahd_session="+cookie)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	return string(body)
}

func sessionCookieOf(t *testing.T, client *http.Client, base string) string {
	t.Helper()
	parsed := mustURL(t, base+"/panel")
	for _, cookie := range client.Jar.Cookies(parsed) {
		if cookie.Name == "ahd_session" {
			return cookie.Value
		}
	}
	return ""
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
