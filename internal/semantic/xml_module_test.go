package semantic

import (
	"strings"
	"testing"
)

const xmlPreamble = "bring XML\nfrom XML bring XMLNode\nfrom XML bring XMLDocument\nfrom XML bring XMLError\n\n"

func TestXMLModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, xmlPreamble+`leaf: XMLNode := XML.text("Ali")
node: XMLNode := XML.element("name", {"lang": "en"}, [leaf])
document: XMLDocument := XML.document(node)
parsed: XMLDocument := XML.parse("<a/>")
read: XMLDocument := XML.read("data.xml")
text: String := XML.stringify(document)
text = XML.stringify(document, true)
XML.write(document, "out.xml")
XML.write(document, "out.xml", true)

kind: String := node.kind()
name: String := node.name()
namespace: String := node.namespace()
content: String := node.text()
found: String? := node.attribute("lang")
attributes: Pair<String, String> := node.attributes()
kids: List<XMLNode> := node.children()
elements: List<XMLNode> := node.elements()
root: XMLNode := document.root()
`)
	requireSemanticClean(t, result)
}

func TestXMLOperationsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`node.kind(1)`,
		`node.name(1)`,
		`node.namespace(1)`,
		`node.text(1)`,
		`node.attribute()`,
		`node.attribute(1)`,
		`node.attributes(1)`,
		`node.children(1)`,
		`node.elements(1)`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, xmlPreamble+"node: XMLNode := XML.text(\"x\")\n"+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestXMLFunctionsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`XML.text(1)`,
		`XML.element(1, {}, [])`,
		`XML.document(1)`,
		`XML.parse(1)`,
		`XML.read(1)`,
		`XML.stringify(1)`,
		`XML.write(XML.document(XML.text("x")), 1)`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, xmlPreamble+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestXMLNodeAttributeIsNullableWithoutGuard(t *testing.T) {
	result := analyzeWithStandardModules(t, xmlPreamble+`node: XMLNode := XML.text("x")
found: String := node.attribute("a")
`)
	requireSemanticFailure(t, result)
}

func TestXMLOperationsArePositionalOnly(t *testing.T) {
	result := analyzeWithStandardModules(t, xmlPreamble+`node: XMLNode := XML.text("x")
node.attribute(name: "a")
`)
	requireSemanticCode(t, result, codeCallArguments)
}

func TestXMLConstructionHintNamesFactories(t *testing.T) {
	result := analyzeWithStandardModules(t, xmlPreamble+`node: XMLNode := XMLNode()
`)
	requireSemanticCode(t, result, codeCallArguments)
	found := false
	for _, diagnostic := range result.Diagnostics {
		if strings.Contains(diagnostic.Hint, "XML.text(value)") && strings.Contains(diagnostic.Hint, "XML.element(") {
			found = true
		}
	}
	if !found {
		t.Fatalf("XMLNode construction diagnostic omitted the XML factories: %+v", result.Diagnostics)
	}
	documentResult := analyzeWithStandardModules(t, xmlPreamble+`document: XMLDocument := XMLDocument()
`)
	requireSemanticCode(t, documentResult, codeCallArguments)
}

func TestXMLHiddenStorageAndUnknownMembersAreRejected(t *testing.T) {
	for _, member := range []string{"data", "root", "describe"} {
		result := analyzeWithStandardModules(t, xmlPreamble+
			"node: XMLNode := XML.text(\"x\")\nwrite(node."+member+")\n")
		requireSemanticFailure(t, result)
	}
	// root() belongs to XMLDocument, not XMLNode, so it is still unknown on
	// a plain XMLNode value even though it is a real member elsewhere.
	result := analyzeWithStandardModules(t, xmlPreamble+"document: XMLDocument := XML.document(XML.element(\"a\", {}, []))\nwrite(document.root())\n")
	requireSemanticClean(t, result)
}

func TestXMLErrorCatchable(t *testing.T) {
	result := analyzeWithStandardModules(t, xmlPreamble+`attempt {
    XML.parse("not xml")
} except XMLError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}
