package semantic

import (
	"strings"
	"testing"

	"ahdcode/internal/types"
)

const httpPreamble = "bring HTTP\nfrom HTTP bring Server\nfrom HTTP bring Request\nfrom HTTP bring Response\nfrom HTTP bring Cookie\nfrom HTTP bring SessionStore\nfrom HTTP bring Session\nfrom HTTP bring Client\nfrom HTTP bring ClientRequest\nfrom HTTP bring ClientResponse\nfrom HTTP bring HTTPError\n\n"

func TestHTTPModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, httpPreamble+`home: Function := (request: Request) -> Response {
    method: Local String := request.method()
    path: Local String := request.path()
    name: Local String? := request.query("name")
    names: Local List<String> := request.queryAll("name")
    header: Local String? := request.header("Content-Type")
    headers: Local List<String> := request.headerAll("Accept")
    body: Local String := request.body()
    title: Local String? := request.form("title")
    tags: Local List<String> := request.formAll("tag")
    theme: Local String? := request.cookie("theme")
    themes: Local List<String> := request.cookieAll("theme")
    cookie: Local Cookie := HTTP.cookie("theme", "dark")
    cookie = cookie.withPath("/")
    cookie = cookie.withHttpOnly(true)
    cookie = cookie.withSecure(false)
    cookie = cookie.withSameSite("Lax")
    cookie = cookie.withMaxAge(60)
    store: Local SessionStore := HTTP.sessions()
    session: Local Session := store.open(request)
    present: Local Bool := session.has("count")
    count: Local String? := session.get("count")
    session.set("count", "1")
    session.remove("gone")
    session.clear()
    session.rotate()
    session.destroy()
    return store.commit(session, HTTP.text("ok").withCookie(cookie).withCookie(HTTP.deleteCookie("old")))
}

created: Response := HTTP.text("ok")
created = HTTP.text("ok", 201)
created = HTTP.html("<h1>Hi</h1>")
created = HTTP.response(204, "", "text/plain")
created = HTTP.redirect("/")
created = HTTP.redirect("/notes", 303)
created = created.withHeader("X-App", "AhdCode")
created = created.withCookie(HTTP.cookie("a", "1"))

store: SessionStore := HTTP.sessions()
store = HTTP.sessions("user_session")
store = HTTP.sessions("user_session", 3600)
store = HTTP.sessions("user_session", 3600, false)
store = HTTP.sessions("user_session", 3600, false, "Lax")

client: Client := HTTP.client()
client = HTTP.client(15)
client = HTTP.client(15, 1024)
client = HTTP.client(15, 1024, false)
request: ClientRequest := HTTP.clientRequest("GET", "https://example.com/")
request = request.withHeader("Accept", "application/json")
request = request.addHeader("X-Trace", "a")
request = request.withBody("")
fetched: ClientResponse := client.get("https://example.com/")
posted: ClientResponse := client.post("https://example.com/", "hi")
posted = client.post("https://example.com/", r'{"ok":true}', "application/json")
sent: ClientResponse := client.send(request)
status: Int := sent.status()
payload: String := sent.body()
header: String? := sent.header("Content-Type")
headers: List<String> := sent.headerAll("Set-Cookie")
finalURL: String := sent.url()

app: Server := HTTP.server("127.0.0.1", 8080)
app = HTTP.server("127.0.0.1", 8080, 2048)
app.get("/", home)
app.post("/notes", home)
app.route("PUT", "/item", home)
app.start()
`)
	requireSemanticClean(t, result)
}

func TestHTTPHandlerSignatureIsCheckedStatically(t *testing.T) {
	tests := []string{
		`bad: Function := () -> Response { return HTTP.text("x") }
app.get("/", bad)`,
		`bad: Function := (request: Request) -> String { return "x" }
app.get("/", bad)`,
		`bad: Function := (value: Int) -> Response { return HTTP.text("x") }
app.get("/", bad)`,
		`bad: Function := (request: Request) -> Int { return 1 }
app.post("/notes", bad)`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, httpPreamble+
				"app: Server := HTTP.server(\"127.0.0.1\", 8080)\n"+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestHTTPOperationsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`HTTP.server()`,
		`HTTP.server(8080, "127.0.0.1")`,
		`HTTP.text(200)`,
		`HTTP.redirect(303)`,
		`app.get(home, "/")`,
		`app.start("/")`,
		`request.method("GET")`,
		`request.query()`,
		`created.withHeader("X")`,
		`status: Int := HTTP.text("ok")`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, httpPreamble+
				"app: Server := HTTP.server(\"127.0.0.1\", 8080)\n"+
				"home: Function := (request: Request) -> Response { return HTTP.text(\"ok\") }\n"+
				"request: Request? := null\n"+
				"created: Response := HTTP.text(\"ok\")\n"+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestHTTPTypesAreNotConstructedDirectly(t *testing.T) {
	result := analyzeWithStandardModules(t, httpPreamble+"app: Server := Server()\n")
	requireSemanticFailure(t, result)
	result = analyzeWithStandardModules(t, httpPreamble+"request: Request := Request()\n")
	requireSemanticFailure(t, result)
	result = analyzeWithStandardModules(t, httpPreamble+"response: Response := Response()\n")
	requireSemanticFailure(t, result)
	result = analyzeWithStandardModules(t, httpPreamble+"cookie: Cookie := Cookie()\n")
	requireSemanticFailure(t, result)
	result = analyzeWithStandardModules(t, httpPreamble+"store: SessionStore := SessionStore()\n")
	requireSemanticFailure(t, result)
	result = analyzeWithStandardModules(t, httpPreamble+"session: Session := Session()\n")
	requireSemanticFailure(t, result)
	result = analyzeWithStandardModules(t, httpPreamble+"client: Client := Client()\n")
	requireSemanticFailure(t, result)
	result = analyzeWithStandardModules(t, httpPreamble+"request: ClientRequest := ClientRequest()\n")
	requireSemanticFailure(t, result)
	result = analyzeWithStandardModules(t, httpPreamble+"response: ClientResponse := ClientResponse()\n")
	requireSemanticFailure(t, result)
}

func TestHTTPModuleInterfaceExportsExactSurface(t *testing.T) {
	module := StandardModuleInterfaces()["HTTP"]
	if module == nil || module.ModuleID != "builtin:HTTP" {
		t.Fatalf("HTTP is not a registered builtin module: %#v", module)
	}
	wantExports := []string{
		"Client", "ClientRequest", "ClientResponse", "Cookie", "HTTPError", "Request", "Response",
		"Server", "Session", "SessionStore", "UploadedFile",
		"client", "clientRequest", "contextHandler", "cookie", "deleteCookie", "download", "file", "html", "redirect", "response", "server", "sessions", "text",
	}
	if strings.Join(module.ExportNames, ",") != strings.Join(wantExports, ",") {
		t.Fatalf("HTTP exports %v; want %v", module.ExportNames, wantExports)
	}
	signatures := map[string]string{
		"server":        "(host: String, port: Int, maxBodyBytes: Int := default) -> Server",
		"text":          "(body: String, status: Int := default) -> Response",
		"html":          "(body: String, status: Int := default) -> Response",
		"response":      "(status: Int, body: String, contentType: String) -> Response",
		"redirect":      "(location: String, status: Int := default) -> Response",
		"file":          "(path: String, contentType: String) -> Response",
		"download":      "(path: String, contentType: String, fileName: String) -> Response",
		"cookie":        "(name: String, value: String) -> Cookie",
		"deleteCookie":  "(name: String, path: String := default) -> Cookie",
		"sessions":      "(cookieName: String := default, maxAgeSeconds: Int := default, secure: Bool := default, sameSite: String := default) -> SessionStore",
		"client":        "(timeoutSeconds: Int := default, maxResponseBytes: Int := default, followRedirects: Bool := default) -> Client",
		"clientRequest":   "(method: String, url: String) -> ClientRequest",
		"contextHandler": "(store: SessionStore, opener: Function(Request, SessionStore) -> RequestContext, handler: Function(RequestContext) -> Response, first: Function(RequestContext) -> Response, second: Function(RequestContext) -> Response) -> Function(Request) -> Response",
	}
	for name, want := range signatures {
		symbol := module.Exports[name]
		if symbol == nil || symbol.Callable == nil {
			t.Fatalf("HTTP.%s is not an exported function", name)
		}
		if have := FormatSignature(symbol.Callable.Signature); have != want {
			t.Fatalf("HTTP.%s signature %q; want %q", name, have, want)
		}
	}
	errorSymbol := module.Exports["HTTPError"]
	if errorSymbol.Class == nil || errorSymbol.Class.Parent == nil || errorSymbol.Class.Parent.Name != "Error" {
		t.Fatalf("HTTPError does not derive from Error: %#v", errorSymbol.Class)
	}
}

func TestHTTPNullableQueryType(t *testing.T) {
	result := analyzeWithStandardModules(t, httpPreamble+`home: Function := (request: Request) -> Response {
    name: Local String? := request.query("name")
    names: Local List<String> := request.queryAll("name")
    return HTTP.text("ok")
}
app: Server := HTTP.server("127.0.0.1", 8080)
app.get("/", home)
`)
	requireSemanticClean(t, result)
	result = analyzeWithStandardModules(t, httpPreamble+`home: Function := (request: Request) -> Response {
    name: Local String := request.query("name")
    return HTTP.text("ok")
}
app: Server := HTTP.server("127.0.0.1", 8080)
app.get("/", home)
`)
	requireSemanticFailure(t, result)
}

func TestHTTPSessionValuesAreStringsOnly(t *testing.T) {
	result := analyzeWithStandardModules(t, httpPreamble+`home: Function := (request: Request) -> Response {
    store: Local SessionStore := HTTP.sessions()
    session: Local Session := store.open(request)
    session.set("count", 1)
    return HTTP.text("ok")
}
`)
	requireSemanticFailure(t, result)
}

func TestHTTPNullableCookieAndSessionGet(t *testing.T) {
	result := analyzeWithStandardModules(t, httpPreamble+`home: Function := (request: Request) -> Response {
    theme: Local String := request.cookie("theme")
    return HTTP.text("ok")
}
`)
	requireSemanticFailure(t, result)
}

func TestHTTPHandlerTypeShape(t *testing.T) {
	handler, ok := httpHandlerType().(types.Function)
	if !ok || handler.Signature == nil || len(handler.Signature.Parameters) != 1 {
		t.Fatalf("handler type = %#v", httpHandlerType())
	}
	if !types.Equal(handler.Signature.Parameters[0].Type, httpRequestType()) || !types.Equal(handler.Signature.Return, httpResponseType()) {
		t.Fatalf("handler signature = %s", types.Display(handler))
	}
}

const httpUploadPreamble = "bring HTTP\nfrom HTTP bring Request\nfrom HTTP bring Response\nfrom HTTP bring UploadedFile\n\n"

func TestHTTPUploadedFileValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, httpUploadPreamble+`handle: Function := (request: Request) -> Response {
    paper: Local UploadedFile? := request.file("paper")
    every: Local List<UploadedFile> := request.files("papers")
    if paper != null {
        name: Local String := paper.originalName()
        declared: Local String? := paper.declaredContentType()
        detected: Local String := paper.detectedContentType()
        size: Local Int := paper.size()
        stored: Local String := paper.save("uploads/papers")
    }
    return HTTP.text("ok")
}
`)
	requireSemanticClean(t, result)
}

func TestHTTPUploadedFileRejectsWrongUsage(t *testing.T) {
	tests := []string{
		// file() is nullable; it may not bind to a non-null UploadedFile.
		`handle: Function := (request: Request) -> Response {
    paper: Local UploadedFile := request.file("paper")
    return HTTP.text("ok")
}`,
		// save requires a directory argument.
		`handle: Function := (request: Request) -> Response {
    paper: Local UploadedFile? := request.file("paper")
    if paper != null {
        stored: Local String := paper.save()
    }
    return HTTP.text("ok")
}`,
		// size is an Int, not a String.
		`handle: Function := (request: Request) -> Response {
    paper: Local UploadedFile? := request.file("paper")
    if paper != null {
        size: Local String := paper.size()
    }
    return HTTP.text("ok")
}`,
		// There is no public bytes() escape hatch.
		`handle: Function := (request: Request) -> Response {
    paper: Local UploadedFile? := request.file("paper")
    if paper != null {
        raw: Local String := paper.bytes()
    }
    return HTTP.text("ok")
}`,
		// UploadedFile is not constructed directly.
		`paper: UploadedFile := UploadedFile()`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			requireSemanticFailure(t, analyzeWithStandardModules(t, httpUploadPreamble+source+"\n"))
		})
	}
}
