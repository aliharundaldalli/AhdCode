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
	"testing"
	"time"
)

func TestHTTPExamplesV05Compile(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"01_cookie.ahd", "02_session_counter.ahd", "03_session_login.ahd"} {
		entry := filepath.Join(root, "examples", "v0.5", name)
		result := Compile(entry)
		if result.HasErrors() {
			t.Fatalf("%s failed:\n%s", name, diagnosticText(result.Diagnostics))
		}
	}
}

func TestHTTPCookieNativeProgram(t *testing.T) {
	port := freeLoopbackPort(t)
	source := `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response

read: Function := (request: Request) -> Response {
    value: Local String? := request.cookie("theme")
    if value == null {
        return HTTP.text("missing")
    }
    return HTTP.text(value)
}

set: Function := (request: Request) -> Response {
    return HTTP.text("set").withCookie(HTTP.cookie("theme", "dark").withHttpOnly(true).withMaxAge(60))
}

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", ok)
app.get("/read", read)
app.get("/set", set)
app.start()
`
	executable := buildSQLiteProgram(t, source)
	base := startBuiltHTTP(t, executable, t.TempDir(), port)
	response, err := http.Get(base + "/read")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "missing" {
		t.Fatalf("absent = %q", body)
	}
	response, err = http.Get(base + "/set")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if len(response.Header["Set-Cookie"]) != 1 || !strings.Contains(response.Header["Set-Cookie"][0], "theme=dark") {
		t.Fatalf("Set-Cookie = %#v", response.Header["Set-Cookie"])
	}
}

func TestHTTPSessionNativeTwoClients(t *testing.T) {
	port := freeLoopbackPort(t)
	executable := buildSQLiteProgram(t, nativeSessionLoginSource(port, false))
	base := startBuiltHTTP(t, executable, t.TempDir(), port)
	assertTwoClientIndependence(t, base)
}

func TestHTTPSessionNativeRestartLosesMemory(t *testing.T) {
	port := freeLoopbackPort(t)
	executable := buildSQLiteProgram(t, nativeSessionLoginSource(port, false))
	directory := t.TempDir()
	command := startHTTPProcess(t, executable, directory, port)
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader("name=Ali"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	response, err = client.Get(base + "/panel")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 || string(body) != "Ali" {
		t.Fatalf("before restart = %d %q", response.StatusCode, body)
	}
	if command.Process != nil {
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
	}
	_ = startHTTPProcess(t, executable, directory, port)
	response, err = client.Get(base + "/panel")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 303 {
		t.Fatalf("after restart expected anonymous redirect, got %d", response.StatusCode)
	}
}

func startHTTPProcess(t *testing.T, executable, directory string, port int) *exec.Cmd {
	t.Helper()
	command := exec.Command(executable)
	command.Dir = directory
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatalf("could not start HTTP program: %v", err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
			_, _ = command.Process.Wait()
		}
		if t.Failed() && stderr.Len() > 0 {
			t.Logf("program stderr:\n%s", stderr.String())
		}
	})
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(base + "/ok")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				return command
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("HTTP program did not start on %s\nstderr: %s", base, stderr.String())
	return command
}

func TestHTTPSessionHTMLNativeProgram(t *testing.T) {
	port := freeLoopbackPort(t)
	source := `bring HTTP
bring HTML
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring SessionStore
from HTTP bring Session

sessions: SessionStore := HTTP.sessions()

count: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    raw: Local String? := session.get("count")
    value: Local Int := 0
    if raw != null {
        value = int(raw)
    }
    value = value + 1
    session.set("count", str(value))
    page: Local String := HTML.document("Count", [HTML.element("p", {}, [HTML.text(str(value))])])
    return sessions.commit(session, HTTP.html(page))
}

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", ok)
app.get("/", count)
app.start()
`
	executable := buildSQLiteProgram(t, source)
	base := startBuiltHTTP(t, executable, t.TempDir(), port)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	response, err := client.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if !strings.Contains(string(body), ">1<") {
		t.Fatalf("first count = %q", body)
	}
	response, err = client.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if !strings.Contains(string(body), ">2<") {
		t.Fatalf("second count = %q", body)
	}
}

func TestHTTPSessionSQLiteNativeProgram(t *testing.T) {
	sqliteHelperForTest(t)
	port := freeLoopbackPort(t)
	source := `bring HTTP
bring SQLite
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring SessionStore
from HTTP bring Session
from SQLite bring Database

sessions: SessionStore := HTTP.sessions("ahd_session", 3600, false, "Lax")

setup: Function := (
) -> Nothing {
    db: Local Database := SQLite.open("users.db")
    db.execute("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT NOT NULL)", [])
    db.close()
    return
}

login: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    name: Local String? := request.form("name")
    if name == null {
        return sessions.commit(session, HTTP.text("name is required", 400))
    }
    db: Local Database := SQLite.open("users.db")
    db.execute("INSERT INTO users(name) VALUES (?)", [SQLite.fromString(name)])
    id: Local Int := db.lastInsertId()
    db.close()
    session.rotate()
    session.set("user_id", str(id))
    session.set("name", name)
    return sessions.commit(session, HTTP.redirect("/panel"))
}

panel: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    name: Local String? := session.get("name")
    if name == null {
        return sessions.commit(session, HTTP.redirect("/"))
    }
    return sessions.commit(session, HTTP.text(name))
}

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

setup()
app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", ok)
app.post("/login", login)
app.get("/panel", panel)
app.start()
`
	executable := buildSQLiteProgram(t, source)
	directory := t.TempDir()
	base := startBuiltHTTP(t, executable, directory, port)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader("name=Ali"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	response, err = client.Get(base + "/panel")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "Ali" {
		t.Fatalf("sqlite session panel = %q", body)
	}
}

func TestHTTPSessionOnlyProgramDoesNotRequireHelpers(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": nativeSessionLoginSource(8080, false)})
	result := Compile(filepath.Join(directory, "main.ahd"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if result.Program == nil || result.Program.RequiresSQLite {
		t.Fatal("a session-only program must not require the SQLite helper")
	}
	for _, file := range result.Program.Files {
		if strings.Contains(file.Content, "ahdsession") || strings.Contains(file.Content, "ahdcookie") {
			t.Fatalf("generated file %s mentions a cookie/session helper", file.Name)
		}
	}
}

func TestHTTPSessionNativeRelocatesWithoutToolchain(t *testing.T) {
	port := freeLoopbackPort(t)
	executable := buildSQLiteProgram(t, nativeSessionLoginSource(port, false))
	relocated := filepath.Join(t.TempDir(), "sessionapp")
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(relocated, data, 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(relocated)
	command.Dir = filepath.Dir(relocated)
	command.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + t.TempDir()}
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
		if t.Failed() && stderr.Len() > 0 {
			t.Logf("stderr:\n%s", stderr.String())
		}
	})
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(5 * time.Second)
	started := false
	for time.Now().Before(deadline) {
		response, err := http.Get(base + "/ok")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				started = true
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !started {
		t.Fatalf("relocated session binary did not start\nstderr: %s", stderr.String())
	}
	assertTwoClientIndependence(t, base)
}

func nativeSessionLoginSource(port int, withHTML bool) string {
	htmlBring := ""
	if withHTML {
		htmlBring = "bring HTML\n"
	}
	return `bring HTTP
` + htmlBring + `from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring SessionStore
from HTTP bring Session

sessions: SessionStore := HTTP.sessions("ahd_session", 3600, false, "Lax")

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

login: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    name: Local String? := request.form("name")
    if name == null {
        return sessions.commit(session, HTTP.text("name is required", 400))
    }
    session.rotate()
    session.set("name", name)
    return sessions.commit(session, HTTP.redirect("/panel"))
}

panel: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    name: Local String? := session.get("name")
    if name == null {
        return sessions.commit(session, HTTP.redirect("/"))
    }
    return sessions.commit(session, HTTP.text(name))
}

logout: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    session.destroy()
    return sessions.commit(session, HTTP.redirect("/"))
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", ok)
app.post("/login", login)
app.get("/panel", panel)
app.post("/logout", logout)
app.start()
`
}

func assertTwoClientIndependence(t *testing.T, base string) {
	t.Helper()
	jarA, _ := cookiejar.New(nil)
	jarB, _ := cookiejar.New(nil)
	clientA := &http.Client{Jar: jarA, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	clientB := &http.Client{Jar: jarB, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	login := func(client *http.Client, name string) {
		t.Helper()
		response, err := client.Post(base+"/login", "application/x-www-form-urlencoded", strings.NewReader("name="+url.QueryEscape(name)))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if response.StatusCode != 303 {
			t.Fatalf("login %s = %d", name, response.StatusCode)
		}
	}
	panel := func(client *http.Client) (int, string) {
		t.Helper()
		response, err := client.Get(base + "/panel")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		return response.StatusCode, string(body)
	}
	login(clientA, "Ali")
	login(clientB, "Mehmet")
	status, body := panel(clientA)
	if status != 200 || body != "Ali" {
		t.Fatalf("A panel = %d %q", status, body)
	}
	status, body = panel(clientB)
	if status != 200 || body != "Mehmet" {
		t.Fatalf("B panel = %d %q", status, body)
	}
	response, err := clientA.Post(base+"/logout", "application/x-www-form-urlencoded", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	status, _ = panel(clientA)
	if status != 303 {
		t.Fatalf("A after logout = %d", status)
	}
	status, body = panel(clientB)
	if status != 200 || body != "Mehmet" {
		t.Fatalf("B after A logout = %d %q", status, body)
	}
}
