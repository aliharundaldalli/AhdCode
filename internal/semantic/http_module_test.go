package semantic

import (
	"strings"
	"testing"

	"ahdcode/internal/types"
)

const httpPreamble = "bring HTTP\nfrom HTTP bring Server\nfrom HTTP bring Request\nfrom HTTP bring Response\nfrom HTTP bring HTTPError\n\n"

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
    return HTTP.text("ok")
}

created: Response := HTTP.text("ok")
created = HTTP.text("ok", 201)
created = HTTP.html("<h1>Hi</h1>")
created = HTTP.response(204, "", "text/plain")
created = HTTP.redirect("/")
created = HTTP.redirect("/notes", 303)
created = created.withHeader("X-App", "AhdCode")

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
}

func TestHTTPModuleInterfaceExportsExactSurface(t *testing.T) {
	module := StandardModuleInterfaces()["HTTP"]
	if module == nil || module.ModuleID != "builtin:HTTP" {
		t.Fatalf("HTTP is not a registered builtin module: %#v", module)
	}
	wantExports := []string{"HTTPError", "Request", "Response", "Server", "html", "redirect", "response", "server", "text"}
	if strings.Join(module.ExportNames, ",") != strings.Join(wantExports, ",") {
		t.Fatalf("HTTP exports %v; want %v", module.ExportNames, wantExports)
	}
	signatures := map[string]string{
		"server":   "(host: String, port: Int, maxBodyBytes: Int := default) -> Server",
		"text":     "(body: String, status: Int := default) -> Response",
		"html":     "(body: String, status: Int := default) -> Response",
		"response": "(status: Int, body: String, contentType: String) -> Response",
		"redirect": "(location: String, status: Int := default) -> Response",
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

func TestHTTPHandlerTypeShape(t *testing.T) {
	handler, ok := httpHandlerType().(types.Function)
	if !ok || handler.Signature == nil || len(handler.Signature.Parameters) != 1 {
		t.Fatalf("handler type = %#v", httpHandlerType())
	}
	if !types.Equal(handler.Signature.Parameters[0].Type, httpRequestType()) || !types.Equal(handler.Signature.Return, httpResponseType()) {
		t.Fatalf("handler signature = %s", types.Display(handler))
	}
}
