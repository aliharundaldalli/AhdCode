package module

import "testing"

func TestBuiltinHTTPCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring HTTP\napp := HTTP.server(\"127.0.0.1\", 8080)",
		"/HTTP.ahd": `server: String := "shadow"`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/HTTP.ahd").ID) != 0 {
		t.Fatal("the sibling HTTP.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "HTTP")
	if !module.Source.Builtin || module.ID != "builtin:HTTP" {
		t.Fatalf("HTTP did not keep its built-in identity: %#v", module)
	}
}

func TestHTTPIsExplicit(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd": `app := HTTP.server("127.0.0.1", 8080)`,
	}, "/Main.ahd")
	requireCode(t, result, "SEM001")
}

func TestHTTPTypesRequireExplicitFromBring(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring HTTP\napp: Server := HTTP.server(\"127.0.0.1\", 8080)",
	}, "/Main.ahd")
	if !result.HasErrors() {
		t.Fatal("Server was usable as a type without `from HTTP bring Server`")
	}
	_, result = compileMemory(t, map[string]string{
		"/Main.ahd": "bring HTTP\nfrom HTTP bring Server\nfrom HTTP bring Request\nfrom HTTP bring Response\nfrom HTTP bring HTTPError\n" +
			"home: Function := (request: Request) -> Response {\n    return HTTP.text(\"ok\")\n}\n" +
			"app: Server := HTTP.server(\"127.0.0.1\", 8080)\napp.get(\"/\", home)\n" +
			"attempt {\n    app.start()\n} except HTTPError as error {\n    write(error.message)\n}\n",
	}, "/Main.ahd")
	requireClean(t, result)
}

func TestBuiltinHTMLCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring HTML\nnode := HTML.text(\"x\")",
		"/HTML.ahd": `text: String := "shadow"`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/HTML.ahd").ID) != 0 {
		t.Fatal("the sibling HTML.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "HTML")
	if !module.Source.Builtin || module.ID != "builtin:HTML" {
		t.Fatalf("HTML did not keep its built-in identity: %#v", module)
	}
}
