package evaluator

import (
	"strings"
	"testing"
)

func TestHTMLEvaluatorEscapesTextAndAttributes(t *testing.T) {
	session := newLatexTestSession()
	script := session.htmlBuiltin("text", []any{`<script>alert("x")</script>`})
	got := session.htmlBuiltin("render", []any{script}).(string)
	if strings.Contains(got, "<script>") || !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("text render = %q", got)
	}
	heading := session.htmlBuiltin("element", []any{
		"h1",
		&Pair{Keys: []any{}, Values: map[any]any{}},
		&List{Items: []any{session.htmlBuiltin("text", []any{"Hello"})}},
	})
	if session.htmlBuiltin("render", []any{heading}) != "<h1>Hello</h1>" {
		t.Fatalf("element render = %v", session.htmlBuiltin("render", []any{heading}))
	}
	injected := session.htmlBuiltin("element", []any{
		"input",
		&Pair{Keys: []any{"name", "value"}, Values: map[any]any{"name": "title", "value": `" autofocus onfocus="alert(1)`}},
		&List{},
	})
	markup := session.htmlBuiltin("render", []any{injected}).(string)
	if strings.Contains(markup, `" autofocus`) || strings.Contains(markup, "</input>") {
		t.Fatalf("attribute injection survived: %q", markup)
	}
	page := session.htmlBuiltin("document", []any{
		`Notes & <x>`,
		&List{Items: []any{session.htmlBuiltin("element", []any{
			"p",
			&Pair{Keys: []any{}, Values: map[any]any{}},
			&List{Items: []any{session.htmlBuiltin("text", []any{"Ayşe ☕"})}},
		})}},
	}).(string)
	if !strings.Contains(page, `<meta charset="utf-8">`) || strings.Contains(page, "<x>") || !strings.Contains(page, "Ayşe ☕") {
		t.Fatalf("document = %q", page)
	}
}

func TestHTMLEvaluatorRejectsInvalidNamesAndVoidChildren(t *testing.T) {
	session := newLatexTestSession()
	expectEvaluatorRaise(t, "HTMLError", func() {
		session.htmlBuiltin("element", []any{"h1 onclick", &Pair{Keys: []any{}, Values: map[any]any{}}, &List{}})
	})
	expectEvaluatorRaise(t, "HTMLError", func() {
		session.htmlBuiltin("element", []any{
			"br",
			&Pair{Keys: []any{}, Values: map[any]any{}},
			&List{Items: []any{session.htmlBuiltin("text", []any{"x"})}},
		})
	})
}

func TestHTMLEvaluatorDoesNotMutateSourceCollections(t *testing.T) {
	session := newLatexTestSession()
	child := session.htmlBuiltin("text", []any{"Hi"})
	children := &List{Items: []any{child}}
	attributes := &Pair{Keys: []any{"class"}, Values: map[any]any{"class": "note"}}
	_ = session.htmlBuiltin("element", []any{"p", attributes, children})
	if len(children.Items) != 1 || children.Items[0] != child {
		t.Fatal("HTML.element mutated the children List")
	}
	if len(attributes.Keys) != 1 || attributes.Keys[0] != "class" || attributes.Values["class"] != "note" {
		t.Fatal("HTML.element mutated the attributes Pair")
	}
}
