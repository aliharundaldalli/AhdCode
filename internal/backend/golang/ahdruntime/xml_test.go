package ahdruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestXMLSimpleRootAndAttributes(t *testing.T) {
	document := AhdXMLParse(AhdClassXMLError, `<student id="42"><name>Ali</name></student>`)
	if got := AhdXMLKind(AhdClassXMLError, document); got != "Element" {
		t.Fatalf("root kind = %q", got)
	}
	if got := AhdXMLName(AhdClassXMLError, document); got != "student" {
		t.Fatalf("root name = %q", got)
	}
	if got := AhdXMLAttribute(AhdClassXMLError, document, "id"); got == nil || *got != "42" {
		t.Fatalf("attribute(id) = %v", got)
	}
	if got := AhdXMLAttribute(AhdClassXMLError, document, "missing"); got != nil {
		t.Fatalf("attribute(missing) = %v, want nil", got)
	}
	elements := AhdXMLElementsData(AhdClassXMLError, document)
	if len(elements) != 1 || AhdXMLName(AhdClassXMLError, elements[0]) != "name" {
		t.Fatalf("elements() = %v", elements)
	}
	if got := AhdXMLNodeText(AhdClassXMLError, elements[0]); got != "Ali" {
		t.Fatalf("name text = %q", got)
	}
}

func TestXMLEmptyElement(t *testing.T) {
	document := AhdXMLParse(AhdClassXMLError, `<root/>`)
	if got := AhdXMLChildrenData(AhdClassXMLError, document); len(got) != 0 {
		t.Fatalf("empty element children = %v", got)
	}
	if got := AhdXMLStringify(AhdClassXMLError, document, false); got != "<root/>" {
		t.Fatalf("empty element stringify = %q", got)
	}
}

func TestXMLNestedElementsAndTextOrder(t *testing.T) {
	document := AhdXMLParse(AhdClassXMLError, `<a><b>1</b><c>2</c></a>`)
	children := AhdXMLChildrenData(AhdClassXMLError, document)
	if len(children) != 2 {
		t.Fatalf("children = %v", children)
	}
	if AhdXMLName(AhdClassXMLError, children[0]) != "b" || AhdXMLNodeText(AhdClassXMLError, children[0]) != "1" {
		t.Fatalf("first child = %v", children[0])
	}
	if AhdXMLName(AhdClassXMLError, children[1]) != "c" || AhdXMLNodeText(AhdClassXMLError, children[1]) != "2" {
		t.Fatalf("second child = %v", children[1])
	}
}

func TestXMLMixedContentOrderPreserved(t *testing.T) {
	document := AhdXMLParse(AhdClassXMLError, `<p>one<b>two</b>three</p>`)
	children := AhdXMLChildrenData(AhdClassXMLError, document)
	if len(children) != 3 {
		t.Fatalf("mixed content children = %v", children)
	}
	if AhdXMLKind(AhdClassXMLError, children[0]) != "Text" || AhdXMLNodeText(AhdClassXMLError, children[0]) != "one" {
		t.Fatalf("first mixed child = %v", children[0])
	}
	if AhdXMLKind(AhdClassXMLError, children[1]) != "Element" || AhdXMLNodeText(AhdClassXMLError, children[1]) != "two" {
		t.Fatalf("second mixed child = %v", children[1])
	}
	if AhdXMLKind(AhdClassXMLError, children[2]) != "Text" || AhdXMLNodeText(AhdClassXMLError, children[2]) != "three" {
		t.Fatalf("third mixed child = %v", children[2])
	}
	if got := AhdXMLNodeText(AhdClassXMLError, document); got != "onethree" {
		t.Fatalf("element text() (direct Text children only) = %q, want %q", got, "onethree")
	}
}

func TestXMLEscapedEntitiesAndUnicode(t *testing.T) {
	document := AhdXMLParse(AhdClassXMLError, `<x attr="a &amp; b">1 &lt; 2 &amp; Ögüzhan İçöz</x>`)
	if got := AhdXMLAttribute(AhdClassXMLError, document, "attr"); got == nil || *got != "a & b" {
		t.Fatalf("escaped attribute = %v", got)
	}
	if got := AhdXMLNodeText(AhdClassXMLError, document); got != "1 < 2 & Ögüzhan İçöz" {
		t.Fatalf("escaped/unicode text = %q", got)
	}
}

func TestXMLCDATARecoveredAsText(t *testing.T) {
	document := AhdXMLParse(AhdClassXMLError, `<x><![CDATA[<raw> & stuff]]></x>`)
	if got := AhdXMLNodeText(AhdClassXMLError, document); got != "<raw> & stuff" {
		t.Fatalf("CDATA text = %q", got)
	}
}

func TestXMLNamespaceBasics(t *testing.T) {
	document := AhdXMLParse(AhdClassXMLError, `<root xmlns="http://example.com/ns"><child/></root>`)
	if got := AhdXMLNamespace(AhdClassXMLError, document); got != "http://example.com/ns" {
		t.Fatalf("root namespace = %q", got)
	}
	unqualified := AhdXMLParse(AhdClassXMLError, `<root><child/></root>`)
	if got := AhdXMLNamespace(AhdClassXMLError, unqualified); got != "" {
		t.Fatalf("unqualified namespace = %q, want \"\"", got)
	}
	prefixed := AhdXMLParse(AhdClassXMLError, `<p:root xmlns:p="http://example.com/p"><p:child/></p:root>`)
	if got := AhdXMLNamespace(AhdClassXMLError, prefixed); got != "http://example.com/p" {
		t.Fatalf("prefixed namespace = %q", got)
	}
	child := AhdXMLChildrenData(AhdClassXMLError, prefixed)[0]
	if got := AhdXMLNamespace(AhdClassXMLError, child); got != "http://example.com/p" {
		t.Fatalf("child namespace = %q", got)
	}
}

func TestXMLConstruction(t *testing.T) {
	nameChild := AhdXMLElement(AhdClassXMLError, "name", nil, nil, []string{AhdXMLText("Ali")})
	scoreChild := AhdXMLElement(AhdClassXMLError, "score", nil, nil, []string{AhdXMLText("91")})
	student := AhdXMLElement(AhdClassXMLError, "student", []string{"id"}, []string{"42"}, []string{nameChild, scoreChild})
	document := AhdXMLDocument(AhdClassXMLError, student)
	if got := AhdXMLStringify(AhdClassXMLError, document, false); got != `<student id="42"><name>Ali</name><score>91</score></student>` {
		t.Fatalf("constructed stringify = %q", got)
	}
	expectRaise(t, AhdClassXMLError, func() { AhdXMLDocument(AhdClassXMLError, AhdXMLText("just text")) })
}

func TestXMLWrongKindAccessorsRaiseXMLError(t *testing.T) {
	text := AhdXMLText("x")
	expectRaise(t, AhdClassXMLError, func() { AhdXMLName(AhdClassXMLError, text) })
	expectRaise(t, AhdClassXMLError, func() { AhdXMLNamespace(AhdClassXMLError, text) })
	expectRaise(t, AhdClassXMLError, func() { AhdXMLAttribute(AhdClassXMLError, text, "a") })
	expectRaise(t, AhdClassXMLError, func() { AhdXMLAttributeKeys(AhdClassXMLError, text) })
	expectRaise(t, AhdClassXMLError, func() { AhdXMLChildrenData(AhdClassXMLError, text) })
	expectRaise(t, AhdClassXMLError, func() { AhdXMLElementsData(AhdClassXMLError, text) })
	// text() is valid on both kinds and must not raise.
	if got := AhdXMLNodeText(AhdClassXMLError, text); got != "x" {
		t.Fatalf("text() on a Text node = %q", got)
	}
}

func TestXMLParseErrors(t *testing.T) {
	malformed := []string{
		"",
		"<a>",
		"<a></b>",
		"<a><b></a></b>",
		`<a attr="1" attr="2"/>`,
		"not xml",
		"<a/><b/>",
		"text<a/>",
	}
	for _, source := range malformed {
		source := source
		expectRaise(t, AhdClassXMLError, func() { AhdXMLParse(AhdClassXMLError, source) })
	}
}

func TestXMLExcessiveDepthRejected(t *testing.T) {
	var open, close strings.Builder
	for i := 0; i < ahdXMLMaxDepth+10; i++ {
		open.WriteString("<a>")
		close.WriteString("</a>")
	}
	expectRaise(t, AhdClassXMLError, func() { AhdXMLParse(AhdClassXMLError, open.String()+close.String()) })
}

func TestXMLStringifyCompactAndPretty(t *testing.T) {
	document := AhdXMLParse(AhdClassXMLError, `<a><b/><c/></a>`)
	if got := AhdXMLStringify(AhdClassXMLError, document, false); got != "<a><b/><c/></a>" {
		t.Fatalf("compact stringify = %q", got)
	}
	pretty := AhdXMLStringify(AhdClassXMLError, document, true)
	want := "<a>\n  <b/>\n  <c/>\n</a>"
	if pretty != want {
		t.Fatalf("pretty stringify = %q, want %q", pretty, want)
	}
	if AhdXMLStringify(AhdClassXMLError, document, true) != pretty {
		t.Fatal("pretty stringify is not deterministic")
	}
	mixed := AhdXMLParse(AhdClassXMLError, `<p>one<b>two</b>three</p>`)
	if got := AhdXMLStringify(AhdClassXMLError, mixed, true); got != "<p>one<b>two</b>three</p>" {
		t.Fatalf("pretty stringify of mixed content added whitespace: %q", got)
	}
}

func TestXMLParseStringifyParseSemanticEquality(t *testing.T) {
	source := `<a id="1"><b>text</b><c/></a>`
	first := AhdXMLParse(AhdClassXMLError, source)
	stringified := AhdXMLStringify(AhdClassXMLError, first, false)
	second := AhdXMLParse(AhdClassXMLError, stringified)
	if AhdXMLStringify(AhdClassXMLError, first, false) != AhdXMLStringify(AhdClassXMLError, second, false) {
		t.Fatalf("parse -> stringify -> parse changed content: %q vs %q",
			AhdXMLStringify(AhdClassXMLError, first, false), AhdXMLStringify(AhdClassXMLError, second, false))
	}
}

func TestXMLFileIO(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "a b", "rapor.xml")
	document := AhdXMLDocument(AhdClassXMLError, AhdXMLElement(AhdClassXMLError, "ok", nil, nil, nil))
	AhdXMLWrite(AhdClassXMLError, document, path, true)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(content)) != "<ok/>" {
		t.Fatalf("written file content = %q", string(content))
	}
	read := AhdXMLRead(AhdClassXMLError, path)
	if AhdXMLName(AhdClassXMLError, read) != "ok" {
		t.Fatal("AhdXMLRead did not round-trip the written file")
	}
	expectRaise(t, AhdClassXMLError, func() { AhdXMLRead(AhdClassXMLError, filepath.Join(directory, "missing.xml")) })
}

func TestXMLNoExternalEntityExpansion(t *testing.T) {
	// A custom general entity requires an internal DTD subset Go's decoder
	// does not process, so referencing one is a parse error, not expansion.
	expectRaise(t, AhdClassXMLError, func() {
		AhdXMLParse(AhdClassXMLError, `<!DOCTYPE x [<!ENTITY e "value">]><a>&e;</a>`)
	})
}
