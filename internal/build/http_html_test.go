package build

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func freeLoopbackPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func startBuiltHTTP(t *testing.T, executable, directory string, port int) string {
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
		response, err := http.Get(base + "/__probe__")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			return base
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("HTTP program did not start on %s\nstderr: %s", base, stderr.String())
	return ""
}

func httpHelloSource(port int) string {
	return `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response

home: Function := (request: Request) -> Response {
    return HTTP.html(
        r"""
        <!doctype html>
        <html>
        <body>
            <h1>Hello from AhdCode</h1>
        </body>
        </html>
        """
    )
}

throws: Function := (request: Request) -> Response {
    toss (DomainError("boom"))
}

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/", home)
app.get("/ok", ok)
app.get("/throws", throws)
app.start()
`
}

func httpRequestSource(port int) string {
	return `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response

hello: Function := (request: Request) -> Response {
    name: Local String? := request.query("name")
    if name != null {
        return HTTP.text(name)
    }
    return HTTP.text("missing")
}

form: Function := (request: Request) -> Response {
    title: Local String? := request.form("title")
    body: Local String? := request.form("body")
    if title == null {
        return HTTP.text("title is required", 400)
    }
    if body == null {
        return HTTP.text("body is required", 400)
    }
    return HTTP.text(title + "|" + body)
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/hello", hello)
app.post("/form", form)
app.start()
`
}

func webNotesSource(port int) string {
	return `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
bring HTML
from HTML bring HTMLNode
bring SQLite
from SQLite bring Database
from SQLite bring SQLiteValue

openNotes: Function := () -> Database {
    db: Local Database := SQLite.open("notes.db")
    db.execute("""
        CREATE TABLE IF NOT EXISTS notes (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            title TEXT NOT NULL,
            body TEXT NOT NULL
        )
        """)
    return db
}

home: Function := (request: Request) -> Response {
    db: Local Database := openNotes()
    q: Local String? := request.query("q")
    rows: Local List<Pair<String, SQLiteValue>> := []
    searchValue: Local String := ""
    if q != null {
        searchValue = q
        term: Local String := "%" + q + "%"
        rows = db.query(
            "SELECT id, title, body FROM notes WHERE title LIKE ? OR body LIKE ? ORDER BY id"
            [SQLite.fromString(term), SQLite.fromString(term)]
        )
    }
    else {
        rows = db.query("SELECT id, title, body FROM notes ORDER BY id")
    }
    db.close()

    items: Local List<HTMLNode> := []
    for row in rows {
        idText: Local String := "{row["id"].int()}"
        items.add(
            HTML.element(
                "li"
                {}
                [
                    HTML.element("strong", {}, [HTML.text(row["title"].string())])
                    HTML.text(" — ")
                    HTML.text(row["body"].string())
                    HTML.element(
                        "form"
                        {"method": "post", "action": "/delete"}
                        [
                            HTML.element("input", {"type": "hidden", "name": "id", "value": idText}, [])
                            HTML.element("button", {"type": "submit"}, [HTML.text("Delete")])
                        ]
                    )
                ]
            )
        )
    }

    page: Local String := HTML.document(
        "Notes"
        [
            HTML.element("h1", {}, [HTML.text("Notes")])
            HTML.element(
                "form"
                {"method": "get", "action": "/"}
                [
                    HTML.element("input", {"name": "q", "value": searchValue}, [])
                    HTML.element("button", {"type": "submit"}, [HTML.text("Search")])
                ]
            )
            HTML.element("ul", {}, items)
            HTML.element(
                "form"
                {"method": "post", "action": "/notes"}
                [
                    HTML.element("p", {}, [
                        HTML.text("Title: ")
                        HTML.element("input", {"name": "title"}, [])
                    ])
                    HTML.element("p", {}, [
                        HTML.text("Body: ")
                        HTML.element("input", {"name": "body"}, [])
                    ])
                    HTML.element("button", {"type": "submit"}, [HTML.text("Add")])
                ]
            )
        ]
    )
    return HTTP.html(page)
}

createNote: Function := (request: Request) -> Response {
    title: Local String? := request.form("title")
    body: Local String? := request.form("body")
    if title == null {
        return HTTP.text("title is required", 400)
    }
    if body == null {
        return HTTP.text("body is required", 400)
    }
    if len(title.trim()) == 0 {
        return HTTP.text("title is required", 400)
    }
    db: Local Database := openNotes()
    db.execute(
        "INSERT INTO notes (title, body) VALUES (?, ?)"
        [SQLite.fromString(title), SQLite.fromString(body)]
    )
    db.close()
    return HTTP.redirect("/")
}

deleteNote: Function := (request: Request) -> Response {
    idText: Local String? := request.form("id")
    if idText == null {
        return HTTP.text("id is required", 400)
    }
    id: Local Int := int(idText)
    db: Local Database := openNotes()
    db.execute("DELETE FROM notes WHERE id = ?", [SQLite.fromInt(id)])
    db.close()
    return HTTP.redirect("/")
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/", home)
app.post("/notes", createNote)
app.post("/delete", deleteNote)
app.start()
`
}

func TestHTTPHelloNativeProgram(t *testing.T) {
	port := freeLoopbackPort(t)
	executable := buildSQLiteProgram(t, httpHelloSource(port))
	directory := t.TempDir()
	base := startBuiltHTTP(t, executable, directory, port)

	response, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if ct := response.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(string(body), "Hello from AhdCode") {
		t.Fatalf("body = %q", body)
	}

	get := func(path string) (int, string) {
		t.Helper()
		response, err := http.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		payload, _ := io.ReadAll(response.Body)
		return response.StatusCode, string(payload)
	}
	status, payload := get("/ok")
	if status != 200 || payload != "ok" {
		t.Fatalf("GET /ok = %d %q", status, payload)
	}
	status, payload = get("/throws")
	if status != 500 || payload != "Internal Server Error" {
		t.Fatalf("GET /throws = %d %q", status, payload)
	}
	status, payload = get("/ok")
	if status != 200 || payload != "ok" {
		t.Fatalf("server did not survive: %d %q", status, payload)
	}
	status, _ = get("/unknown")
	if status != 404 {
		t.Fatalf("GET /unknown = %d", status)
	}
	request, _ := http.NewRequest(http.MethodPost, base+"/ok", nil)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != 405 {
		t.Fatalf("POST /ok = %d", response.StatusCode)
	}
	status, payload = get("/ok")
	if status != 200 || payload != "ok" {
		t.Fatalf("after 405: %d %q", status, payload)
	}
}

func TestHTTPRequestNativeQueryAndForm(t *testing.T) {
	port := freeLoopbackPort(t)
	executable := buildSQLiteProgram(t, httpRequestSource(port))
	base := startBuiltHTTP(t, executable, t.TempDir(), port)

	response, err := http.Get(base + "/hello?name=" + url.QueryEscape("Ayşe"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "Ayşe" {
		t.Fatalf("query = %q", body)
	}

	form := url.Values{}
	form.Set("title", "First Note")
	form.Set("body", "Hello World")
	response, err = http.Post(base+"/form", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "First Note|Hello World" {
		t.Fatalf("form = %q", body)
	}

	encoded := "title=First+Note&body=Hello%20World"
	response, err = http.Post(base+"/form", "application/x-www-form-urlencoded", strings.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "First Note|Hello World" {
		t.Fatalf("urlencoded form = %q", body)
	}
}

func TestHTTPBodyLimitSurvivesOnNativeProgram(t *testing.T) {
	port := freeLoopbackPort(t)
	source := `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response

echo: Function := (request: Request) -> Response {
    return HTTP.text(request.body())
}

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `, 16)
app.post("/echo", echo)
app.get("/ok", ok)
app.start()
`
	executable := buildSQLiteProgram(t, source)
	base := startBuiltHTTP(t, executable, t.TempDir(), port)
	response, err := http.Post(base+"/echo", "text/plain", strings.NewReader("12345678901234567"))
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != 413 {
		t.Fatalf("oversize = %d", response.StatusCode)
	}
	response, err = http.Get(base + "/ok")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 || string(body) != "ok" {
		t.Fatalf("after 413: %d %q", response.StatusCode, body)
	}
}

func TestHTTPOnlyProgramDoesNotRequireSQLiteHelper(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": httpHelloSource(8080)})
	result := Compile(filepath.Join(directory, "main.ahd"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if result.Program == nil || result.Program.RequiresSQLite {
		t.Fatal("an HTTP-only program must not require the SQLite helper")
	}
	var sawHTTP, sawHTML bool
	for _, file := range result.Program.Files {
		if strings.Contains(file.Content, `"github.com/gin-gonic/gin"`) || strings.Contains(file.Content, `"github.com/labstack/echo`) {
			t.Fatalf("generated file %s imports a third-party HTTP framework", file.Name)
		}
		if file.Name == "ahdcode_http_runtime.go" {
			sawHTTP = true
		}
		if file.Name == "ahdcode_html_runtime.go" {
			sawHTML = true
		}
	}
	if !sawHTTP || !sawHTML {
		t.Fatal("HTTP/HTML runtime files were not emitted")
	}
}

func TestHTMLRenderNativeProgram(t *testing.T) {
	out, errorOutput, code := buildAndRun(t, filepath.Join(writeSources(t, map[string]string{"main.ahd": `bring HTML
write(HTML.render(HTML.text("<script>alert(1)</script>")))
write(HTML.document("Tom & Jerry", [HTML.element("p", {}, [HTML.text("Ayşe ☕")])]))
`}), "main.ahd"), "")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errorOutput)
	}
	if strings.Contains(out, "<script>") || !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("stdout = %q", out)
	}
	if !strings.Contains(out, "Tom &amp; Jerry") || !strings.Contains(out, "Ayşe ☕") {
		t.Fatalf("document stdout = %q", out)
	}
}

func TestWebNotesNativeXSSAndSQLStayData(t *testing.T) {
	sqliteHelperForTest(t)
	port := freeLoopbackPort(t)
	source := webNotesSource(port)
	executable := buildSQLiteProgram(t, source)
	directory := t.TempDir()
	base := startBuiltHTTP(t, executable, directory, port)

	post := func(path string, fields url.Values) {
		t.Helper()
		response, err := http.Post(base+path, "application/x-www-form-urlencoded", strings.NewReader(fields.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if response.StatusCode != 200 && response.StatusCode != 303 {
			t.Fatalf("POST %s = %d", path, response.StatusCode)
		}
	}

	xssTitle := `<script>alert("owned")</script>`
	xssBody := `<img src=x onerror=alert(1)>`
	post("/notes", url.Values{"title": {xssTitle}, "body": {xssBody}})
	post("/notes", url.Values{"title": {"Tom & Jerry"}, "body": {"Ayşe ☕"}})
	post("/notes", url.Values{"title": {"Robert'); DROP TABLE notes;--"}, "body": {"this stays data"}})

	response, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	html := string(page)
	if strings.Contains(html, `<script>alert("owned")</script>`) || strings.Contains(html, `<img src=x onerror=alert(1)>`) {
		t.Fatalf("unescaped markup reached the page:\n%s", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") || !strings.Contains(html, "&lt;img") {
		t.Fatalf("escaped markup missing:\n%s", html)
	}
	if !strings.Contains(html, "Tom &amp; Jerry") || !strings.Contains(html, "Ayşe ☕") {
		t.Fatalf("ordinary text missing:\n%s", html)
	}
	if !strings.Contains(html, "DROP TABLE notes") {
		t.Fatalf("SQL-looking title missing from escaped page:\n%s", html)
	}

	response, err = http.Get(base + "/?q=" + url.QueryEscape("DROP TABLE"))
	if err != nil {
		t.Fatal(err)
	}
	searchPage, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if !strings.Contains(string(searchPage), "DROP TABLE notes") {
		t.Fatalf("search did not find the SQL-looking title:\n%s", searchPage)
	}

	if _, err := os.Stat(filepath.Join(directory, "notes.db")); err != nil {
		t.Fatalf("notes.db was not created: %v", err)
	}
}

func TestWebNotesRequiresSQLiteHelper(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": webNotesSource(8080)})
	result := Compile(filepath.Join(directory, "main.ahd"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if result.Program == nil || !result.Program.RequiresSQLite {
		t.Fatal("Web Notes must require the SQLite helper")
	}
}
