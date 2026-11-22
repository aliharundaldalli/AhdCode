package build

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPrivateAvatarAuthorizationNative(t *testing.T) {
	sqliteHelperForTest(t)
	root := t.TempDir()
	const alice = `<svg xmlns="http://www.w3.org/2000/svg"><title>Alice private avatar</title></svg>`
	const bob = `<svg xmlns="http://www.w3.org/2000/svg"><title>Bob private avatar</title></svg>`
	for path, body := range map[string]string{
		"private/alice.svg": alice, "private/bob.svg": bob,
		"public/style.css": "body{}", "managed/declared.css": "h1{}", "managed/other.css": "undeclared neighbour",
	} {
		name := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "public", "alias")
	target := filepath.Join(root, "private")
	if runtime.GOOS == "windows" {
		if output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
			t.Fatalf("create fixture junction: %v\n%s", err, output)
		}
	} else if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(link) })

	port := freeLoopbackPort(t)
	executable := buildSQLiteProgram(t, privateAvatarSource(port))
	process := startHTTPProcess(t, executable, root, port)
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	baseURL, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	newClient := func(withCookies bool) *http.Client {
		client := &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		if withCookies {
			jar, err := cookiejar.New(nil)
			if err != nil {
				t.Fatal(err)
			}
			client.Jar = jar
		}
		return client
	}
	request := func(client *http.Client, method, path string, form url.Values, headers map[string]string, status int, body string, sensitive bool) (string, []*http.Cookie) {
		t.Helper()
		req, err := http.NewRequest(method, base+path, strings.NewReader(form.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		bytes, err := io.ReadAll(response.Body)
		text := string(bytes)
		if err != nil || response.StatusCode != status || (body != "" && text != body) {
			t.Fatalf("%s %s returned %d %q (%v), want %d %q", method, path, response.StatusCode, text, err, status, body)
		}
		if (method == "HEAD" || status == 304) && len(bytes) != 0 {
			t.Fatalf("%s %s returned a body for a bodyless response", method, path)
		}
		if len(body) == 0 && (strings.Contains(text, alice) || strings.Contains(text, bob) || strings.Contains(text, root)) {
			t.Fatalf("%s %s disclosed fixture bytes or a storage path", method, path)
		}
		if response.Header.Get("Location") != "" {
			t.Fatalf("%s %s returned a redirect", method, path)
		}
		if sensitive && !strings.Contains(response.Header.Get("Cache-Control"), "no-store") {
			t.Fatalf("%s %s cache policy %q", method, path, response.Header.Get("Cache-Control"))
		}
		return text, response.Cookies()
	}
	anon, a, b := newClient(false), newClient(true), newClient(true)
	get := func(client *http.Client, path string, status int, body string) {
		t.Helper()
		request(client, "GET", path, nil, nil, status, body, true)
	}
	csrf := func(client *http.Client) string {
		t.Helper()
		token, _ := request(client, "GET", "/csrf", nil, nil, 200, "", true)
		if token == "" {
			t.Fatal("CSRF endpoint failed to mint a token")
		}
		return token
	}
	login := func(client *http.Client, credential string) {
		t.Helper()
		token := csrf(client)
		previous := client.Jar.Cookies(baseURL)
		if len(previous) != 1 {
			t.Fatal("CSRF token must belong to a pre-login session")
		}
		_, cookies := request(client, "POST", "/login", url.Values{"credential": {credential}, "_csrf": {token}}, nil, 200, "Signed in", true)
		if len(cookies) != 1 || cookies[0].Value == "" || cookies[0].Value == previous[0].Value ||
			!cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
			t.Fatal("login must rotate the session and issue an HttpOnly, SameSite=Lax cookie")
		}
	}
	get(anon, "/avatars/1", 401, "Sign in")
	request(anon, "GET", "/avatars/2", nil, map[string]string{
		"Cookie": "avatar_session=bob", "X-User": "2", "Authorization": "Bearer bob",
	}, 401, "Sign in", true)
	request(anon, "POST", "/login", url.Values{"credential": {"fixture-alice"}}, nil, 403, "Forbidden", true)
	anonymousToken := csrf(a)
	request(a, "POST", "/login", url.Values{"credential": {"unknown"}, "_csrf": {anonymousToken}}, nil, 401, "Sign in", true)
	login(a, "fixture-alice")
	login(b, "fixture-bob")
	get(a, "/avatars/1", 200, alice)
	get(b, "/avatars/2", 200, bob)
	get(a, "/avatars/2", 404, "Not Found")
	get(b, "/avatars/1", 404, "Not Found")
	get(a, "/avatars/2?owner=1&filename=alice.svg", 404, "Not Found")
	request(a, "GET", "/avatars/2", nil, map[string]string{
		"X-User": "2", "X-Original-URL": "/avatars/1", "Range": "bytes=0-5", "If-None-Match": "*",
	}, 404, "Not Found", true)
	request(a, "HEAD", "/avatars/2", nil, nil, 404, "", true)
	request(a, "HEAD", "/avatars/1", nil, nil, 200, "", true)
	request(a, "GET", "/avatars/1", nil, map[string]string{"If-None-Match": `"alice-avatar"`}, 304, "", true)
	request(b, "HEAD", "/avatars/1", nil, map[string]string{"If-None-Match": `"alice-avatar"`}, 404, "", true)
	request(anon, "GET", "/avatars/1", nil, map[string]string{"If-None-Match": `"alice-avatar"`}, 401, "Sign in", true)
	request(a, "GET", "/avatars/1", nil, map[string]string{"Range": "bytes=0-5"}, 206, alice[:6], true)
	request(a, "GET", "/avatars/1", nil, map[string]string{"Range": "bytes=900-999"}, 416, "", true)
	get(a, "/avatars/1?file=bob.svg", 200, alice)
	get(a, "/avatars/2.svg", 404, "Not Found")
	for _, path := range []string{
		"/private/bob.svg", "/assets/bob.svg", "/assets/alias/bob.svg", "/assets/%2e%2e/private/bob.svg",
		"/assets/..%5cprivate%5cbob.svg", "/managed/other.css", "/managed/bob.svg",
		"/sessions.db", "/assets/sessions.db", "/managed/sessions.db",
	} {
		request(a, "GET", path, nil, nil, 404, "", false)
	}
	request(anon, "GET", "/assets/style.css", nil, nil, 200, "body{}", false)
	request(anon, "GET", "/managed/declared.css", nil, nil, 200, "h1{}", false)
	if err := process.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if _, err := process.Process.Wait(); err != nil {
		t.Fatal(err)
	}
	startHTTPProcess(t, executable, root, port)
	get(a, "/avatars/1", 200, alice)
	get(b, "/avatars/2", 200, bob)
	get(a, "/avatars/2", 404, "Not Found")
	old := a.Jar.Cookies(baseURL)
	if len(old) != 1 {
		t.Fatalf("expected one session cookie, got %d", len(old))
	}
	bobToken := csrf(b)
	request(a, "POST", "/logout", url.Values{"_csrf": {bobToken}}, nil, 403, "Forbidden", true)
	get(a, "/avatars/1", 200, alice)
	aliceToken := csrf(a)
	request(a, "POST", "/logout", url.Values{"_csrf": {aliceToken}}, nil, 200, "Signed out", true)
	get(a, "/avatars/1", 401, "Sign in")
	login(a, "fixture-bob")
	get(a, "/avatars/1", 404, "Not Found")
	get(a, "/avatars/2", 200, bob)
	request(anon, "GET", "/avatars/1", nil, map[string]string{"Cookie": old[0].String()}, 401, "Sign in", true)
}

func privateAvatarSource(port int) string {
	return `bring HTTP
bring Web
from HTTP bring (Server, Request, Response, SessionStore, Session)
from HTML bring HTMLNode
from Web bring RequestContext

sessions: SessionStore := HTTP.persistentSessions("sessions.db", "avatar_session", 3600, false, "Lax")
declared: HTMLNode := Web.Assets.stylesheet("declared.css")

health: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

csrf: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    context: Local RequestContext := Web.context(request, sessions)
    return context.respond(HTTP.text(context.csrfToken()).withHeader("Cache-Control", "no-store"))
}

login: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    context: Local RequestContext := Web.context(request, sessions)
    if not context.csrfValid() { return HTTP.text("Forbidden", 403).withHeader("Cache-Control", "no-store") }
    credential: Local String? := request.form("credential")
    owner: Local String := ""
    if credential == "fixture-alice" { owner = "1" }
    if credential == "fixture-bob" { owner = "2" }
    if owner == "" { return HTTP.text("Sign in", 401).withHeader("Cache-Control", "no-store") }
    session: Local Session := sessions.open(request)
    session.rotate()
    session.set("owner", owner)
    return sessions.commit(session, HTTP.text("Signed in").withHeader("Cache-Control", "no-store"))
}

logout: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    context: Local RequestContext := Web.context(request, sessions)
    if not context.csrfValid() { return HTTP.text("Forbidden", 403).withHeader("Cache-Control", "no-store") }
    session: Local Session := sessions.open(request)
    session.destroy()
    return sessions.commit(session, HTTP.text("Signed out").withHeader("Cache-Control", "no-store"))
}

avatar: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    owner: Local String? := session.get("owner")
    if owner == null { return HTTP.text("Sign in", 401).withHeader("Cache-Control", "no-store") }
    if owner == "1" and request.path() == "/avatars/1" { return HTTP.file("private/alice.svg", "image/svg+xml").withHeader("ETag", "\"alice-avatar\"") }
    if owner == "2" and request.path() == "/avatars/2" { return HTTP.file("private/bob.svg", "image/svg+xml") }
    return HTTP.text("Not Found", 404).withHeader("Cache-Control", "no-store")
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.static("/assets", "public")
app.managed("/managed", "managed")
app.get("/ok", health)
app.get("/csrf", csrf)
app.post("/login", login)
app.post("/logout", logout)
app.get("/avatars/1", avatar)
app.get("/avatars/2", avatar)
app.get("/avatars/2.svg", avatar)
app.route("HEAD", "/avatars/1", avatar)
app.route("HEAD", "/avatars/2", avatar)
app.start()
`
}
