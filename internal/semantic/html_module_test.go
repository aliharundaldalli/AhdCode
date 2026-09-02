package semantic

import (
	"strings"
	"testing"
)

const htmlPreamble = "bring HTML\nfrom HTML bring HTMLNode\nfrom HTML bring HTMLError\n\n"

func TestHTMLModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, htmlPreamble+`node: HTMLNode := HTML.text("Hello")
node = HTML.element("h1", {}, [HTML.text("Hello")])
node = HTML.element("input", {"name": "title", "value": "x"}, [])
page: String := HTML.render(node)
page = HTML.document("Notes", [HTML.element("p", {}, [HTML.text("Hi")])])
`)
	requireSemanticClean(t, result)
}

func TestHTMLOperationsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`HTML.text()`,
		`HTML.text(1)`,
		`HTML.element("p")`,
		`HTML.element("p", "x", [])`,
		`HTML.element("p", {}, "x")`,
		`HTML.render("x")`,
		`HTML.document("t", HTML.text("x"))`,
		`node: String := HTML.text("x")`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, htmlPreamble+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestHTMLNodeIsNotConstructedDirectly(t *testing.T) {
	result := analyzeWithStandardModules(t, htmlPreamble+"node: HTMLNode := HTMLNode()\n")
	requireSemanticFailure(t, result)
}

func TestHTMLModuleInterfaceExportsExactSurface(t *testing.T) {
	module := StandardModuleInterfaces()["HTML"]
	if module == nil || module.ModuleID != "builtin:HTML" {
		t.Fatalf("HTML is not a registered builtin module: %#v", module)
	}
	wantExports := []string{"HTMLError", "HTMLNode", "document", "element", "render", "text"}
	if strings.Join(module.ExportNames, ",") != strings.Join(wantExports, ",") {
		t.Fatalf("HTML exports %v; want %v", module.ExportNames, wantExports)
	}
	signatures := map[string]string{
		"text":     "(value: String) -> HTMLNode",
		"element":  "(name: String, attributes: Pair<String, String>, children: List<HTMLNode>) -> HTMLNode",
		"render":   "(node: HTMLNode) -> String",
		"document": "(title: String, body: List<HTMLNode>) -> String",
	}
	for name, want := range signatures {
		symbol := module.Exports[name]
		if symbol == nil || symbol.Callable == nil {
			t.Fatalf("HTML.%s is not an exported function", name)
		}
		if have := FormatSignature(symbol.Callable.Signature); have != want {
			t.Fatalf("HTML.%s signature %q; want %q", name, have, want)
		}
	}
	errorSymbol := module.Exports["HTMLError"]
	if errorSymbol.Class == nil || errorSymbol.Class.Parent == nil || errorSymbol.Class.Parent.Name != "Error" {
		t.Fatalf("HTMLError does not derive from Error: %#v", errorSymbol.Class)
	}
}
