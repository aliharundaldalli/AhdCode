package semantic

import (
	"strings"
	"testing"
)

const htmlPreamble = "bring HTML\nfrom HTML bring HTMLNode\nfrom HTML bring HTMLError\n\n"

const htmlParsePreamble = "bring HTML\nfrom HTML bring HTMLDocument\nfrom HTML bring HTMLElement\nfrom HTML bring HTMLError\n\n"

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

func TestHTMLParseModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, htmlParsePreamble+`document: HTMLDocument := HTML.parse("<h1>Hi</h1>")
elements: List<HTMLElement> := document.select("h1")
heading: HTMLElement? := document.first("h1")
if heading != null {
    tag: Local String := heading.tag()
    text: Local String := heading.text()
    attr: Local String? := heading.attr("id")
    present: Local Bool := heading.hasAttr("id")
    nested: Local List<HTMLElement> := heading.select("span")
    firstNested: Local HTMLElement? := heading.first("span")
}
`)
	requireSemanticClean(t, result)
}

func TestHTMLParseOperationsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`HTML.parse()`,
		`HTML.parse(1)`,
		`document: HTMLDocument := HTML.parse("x")
document.select()`,
		`document: HTMLDocument := HTML.parse("x")
document.select(1)`,
		`document: HTMLDocument := HTML.parse("x")
document.first()`,
		`document: HTMLDocument := HTML.parse("x")
element: HTMLElement := document.first("h1")`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, htmlParsePreamble+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestHTMLDocumentAndElementAreNotConstructedDirectly(t *testing.T) {
	result := analyzeWithStandardModules(t, htmlParsePreamble+"document: HTMLDocument := HTMLDocument()\n")
	requireSemanticFailure(t, result)
	result = analyzeWithStandardModules(t, htmlParsePreamble+"element: HTMLElement := HTMLElement()\n")
	requireSemanticFailure(t, result)
}

func TestHTMLModuleInterfaceExportsExactSurface(t *testing.T) {
	module := StandardModuleInterfaces()["HTML"]
	if module == nil || module.ModuleID != "builtin:HTML" {
		t.Fatalf("HTML is not a registered builtin module: %#v", module)
	}
	wantExports := []string{"HTMLDocument", "HTMLElement", "HTMLError", "HTMLNode", "document", "element", "parse", "render", "text"}
	if strings.Join(module.ExportNames, ",") != strings.Join(wantExports, ",") {
		t.Fatalf("HTML exports %v; want %v", module.ExportNames, wantExports)
	}
	signatures := map[string]string{
		"text":     "(value: String) -> HTMLNode",
		"element":  "(name: String, attributes: Pair<String, String>, children: List<HTMLNode>) -> HTMLNode",
		"render":   "(node: HTMLNode) -> String",
		"document": "(title: String, body: List<HTMLNode>) -> String",
		"parse":    "(source: String) -> HTMLDocument",
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
