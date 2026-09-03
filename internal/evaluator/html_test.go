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

func TestHTMLEvaluatorParseSelectAndAccessors(t *testing.T) {
	session := newLatexTestSession()
	source := `<article class="card" data-id="1"><h2>Riesz &amp; Banach</h2><a href="/notes/1">Read</a></article>`
	doc := session.htmlBuiltin("parse", []any{source})
	articles := session.htmlOperation("HTMLDocument.select", doc, []any{"article.card"}).(*List)
	if len(articles.Items) != 1 {
		t.Fatalf("expected 1 article.card, got %d", len(articles.Items))
	}
	article := articles.Items[0]
	if session.htmlOperation("HTMLElement.tag", article, nil) != "article" {
		t.Fatalf("expected tag() article")
	}
	h2 := session.htmlOperation("HTMLElement.first", article, []any{"h2"})
	if h2 == nil {
		t.Fatalf("expected h2 to be found")
	}
	if got := session.htmlOperation("HTMLElement.text", h2, nil); got != "Riesz & Banach" {
		t.Fatalf("expected decoded entity text, got %v", got)
	}
	a := session.htmlOperation("HTMLElement.first", article, []any{"a"})
	href := session.htmlOperation("HTMLElement.attr", a, []any{"href"})
	if href != "/notes/1" {
		t.Fatalf("expected relative href, got %v", href)
	}
	if session.htmlOperation("HTMLElement.hasAttr", article, []any{"data-id"}) != true {
		t.Fatalf("expected hasAttr data-id true")
	}
	missing := session.htmlOperation("HTMLDocument.first", doc, []any{"section"})
	if missing != nil {
		t.Fatalf("expected first() to return nil when nothing matches")
	}
}

func TestHTMLEvaluatorInvalidSelectorRaisesHTMLError(t *testing.T) {
	session := newLatexTestSession()
	doc := session.htmlBuiltin("parse", []any{"<div></div>"})
	expectEvaluatorRaise(t, "HTMLError", func() {
		session.htmlOperation("HTMLDocument.select", doc, []any{":nth-child(2)"})
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
