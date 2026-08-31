package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestXMLEvaluatorConstructionAndAccessors(t *testing.T) {
	session := newLatexTestSession()
	leaf := session.xmlBuiltin("text", []any{"Ali"})
	attrs := &Pair{Keys: []any{"lang"}, Values: map[any]any{"lang": "en"}}
	node := session.xmlBuiltin("element", []any{"name", attrs, &List{Items: []any{leaf}}})

	if session.xmlOperation("XMLNode.kind", node, nil) != "Element" {
		t.Fatal("element kind")
	}
	if session.xmlOperation("XMLNode.name", node, nil) != "name" {
		t.Fatal("element name")
	}
	if session.xmlOperation("XMLNode.text", node, nil) != "Ali" {
		t.Fatal("element text = concatenation of direct Text children")
	}
	if got := session.xmlOperation("XMLNode.attribute", node, []any{"lang"}); got != "en" {
		t.Fatalf("attribute(lang) = %v", got)
	}
	if got := session.xmlOperation("XMLNode.attribute", node, []any{"missing"}); got != nil {
		t.Fatalf("attribute(missing) = %v, want nil", got)
	}
	pair := session.xmlOperation("XMLNode.attributes", node, nil).(*Pair)
	if len(pair.Keys) != 1 || pair.Keys[0] != "lang" {
		t.Fatalf("attributes() = %v", pair)
	}
	children := session.xmlOperation("XMLNode.children", node, nil).(*List)
	if len(children.Items) != 1 || session.xmlOperation("XMLNode.kind", children.Items[0], nil) != "Text" {
		t.Fatalf("children() = %v", children.Items)
	}
}

func TestXMLEvaluatorDocumentRequiresElementRoot(t *testing.T) {
	session := newLatexTestSession()
	textNode := session.xmlBuiltin("text", []any{"x"})
	expectEvaluatorRaise(t, "XMLError", func() {
		session.xmlBuiltin("document", []any{textNode})
	})
	element := session.xmlBuiltin("element", []any{"a", &Pair{}, &List{}})
	document := session.xmlBuiltin("document", []any{element})
	root := session.xmlOperation("XMLDocument.root", document, nil)
	if session.xmlOperation("XMLNode.name", root, nil) != "a" {
		t.Fatal("document.root() did not return the original root")
	}
}

func TestXMLEvaluatorMixedContentOrderPreserved(t *testing.T) {
	session := newLatexTestSession()
	document := session.xmlBuiltin("parse", []any{`<p>one<b>two</b>three</p>`})
	root := session.xmlOperation("XMLDocument.root", document, nil)
	children := session.xmlOperation("XMLNode.children", root, nil).(*List)
	if len(children.Items) != 3 {
		t.Fatalf("mixed content children = %v", children.Items)
	}
	kinds := []string{"Text", "Element", "Text"}
	for index, kind := range kinds {
		if session.xmlOperation("XMLNode.kind", children.Items[index], nil) != kind {
			t.Fatalf("child %d kind = %v, want %s", index, session.xmlOperation("XMLNode.kind", children.Items[index], nil), kind)
		}
	}
	if session.xmlOperation("XMLNode.text", root, nil) != "onethree" {
		t.Fatal("root text() should concatenate only direct Text children")
	}
}

func TestXMLEvaluatorNamespaceBasics(t *testing.T) {
	session := newLatexTestSession()
	document := session.xmlBuiltin("parse", []any{`<root xmlns="http://example.com/ns"><child/></root>`})
	root := session.xmlOperation("XMLDocument.root", document, nil)
	if got := session.xmlOperation("XMLNode.namespace", root, nil); got != "http://example.com/ns" {
		t.Fatalf("namespace() = %v", got)
	}
}

func TestXMLEvaluatorWrongKindAccessorsRaiseXMLError(t *testing.T) {
	session := newLatexTestSession()
	text := session.xmlBuiltin("text", []any{"x"})
	expectEvaluatorRaise(t, "XMLError", func() { session.xmlOperation("XMLNode.name", text, nil) })
	expectEvaluatorRaise(t, "XMLError", func() { session.xmlOperation("XMLNode.attribute", text, []any{"a"}) })
	expectEvaluatorRaise(t, "XMLError", func() { session.xmlOperation("XMLNode.children", text, nil) })
	if got := session.xmlOperation("XMLNode.text", text, nil); got != "x" {
		t.Fatalf("text() on Text node = %v", got)
	}
}

func TestXMLEvaluatorDuplicateAttributeAndMalformedInputRaiseXMLError(t *testing.T) {
	session := newLatexTestSession()
	for _, source := range []string{
		`<a attr="1" attr="2"/>`,
		"<a><b></a></b>",
		"",
		"<a/><b/>",
	} {
		source := source
		expectEvaluatorRaise(t, "XMLError", func() {
			session.xmlBuiltin("parse", []any{source})
		})
	}
}

func TestXMLEvaluatorStringifyCompactAndPretty(t *testing.T) {
	session := newLatexTestSession()
	document := session.xmlBuiltin("parse", []any{`<a><b/><c/></a>`})
	if got := session.xmlBuiltin("stringify", []any{document}); got != "<a><b/><c/></a>" {
		t.Fatalf("compact stringify = %v", got)
	}
	pretty := session.xmlBuiltin("stringify", []any{document, true}).(string)
	if pretty != "<a>\n  <b/>\n  <c/>\n</a>" {
		t.Fatalf("pretty stringify = %q", pretty)
	}
}

func TestXMLEvaluatorFileIO(t *testing.T) {
	session := newLatexTestSession()
	directory := t.TempDir()
	path := filepath.Join(directory, "a b", "rapor.xml")
	element := session.xmlBuiltin("element", []any{"ok", &Pair{}, &List{}})
	document := session.xmlBuiltin("document", []any{element})
	if got := session.xmlBuiltin("write", []any{document, path, true}); got != Nothing {
		t.Fatalf("write returned %#v", got)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(content)) != "<ok/>" {
		t.Fatalf("written content = %q", string(content))
	}
	read := session.xmlBuiltin("read", []any{path})
	root := session.xmlOperation("XMLDocument.root", read, nil)
	if session.xmlOperation("XMLNode.name", root, nil) != "ok" {
		t.Fatal("read did not round-trip the written file")
	}
	expectEvaluatorRaise(t, "XMLError", func() {
		session.xmlBuiltin("read", []any{filepath.Join(directory, "missing.xml")})
	})
}
