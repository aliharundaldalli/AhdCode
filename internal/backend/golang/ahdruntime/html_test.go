package ahdruntime

import (
	"strings"
	"testing"
)

func TestHTMLTextEscapesMarkup(t *testing.T) {
	node := AhdHTMLText(`<script>alert("x")</script>`)
	got := AhdHTMLRender(AhdClassHTMLError, node)
	if strings.Contains(got, "<script>") || !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("text render = %q", got)
	}
	amp := AhdHTMLRender(AhdClassHTMLError, AhdHTMLText("Tom & Jerry"))
	if amp != "Tom &amp; Jerry" {
		t.Fatalf("ampersand render = %q", amp)
	}
	lt := AhdHTMLRender(AhdClassHTMLError, AhdHTMLText("1 < 2"))
	if lt != "1 &lt; 2" {
		t.Fatalf("less-than render = %q", lt)
	}
}

func TestHTMLElementRendersDeterministically(t *testing.T) {
	node := AhdHTMLElement(AhdClassHTMLError, "h1", nil, nil, []string{AhdHTMLText("Hello")})
	if got := AhdHTMLRender(AhdClassHTMLError, node); got != "<h1>Hello</h1>" {
		t.Fatalf("render = %q", got)
	}
	input := AhdHTMLElement(AhdClassHTMLError, "input", []string{"name", "value"}, []string{"title", `" autofocus onfocus="alert(1)`}, nil)
	got := AhdHTMLRender(AhdClassHTMLError, input)
	if strings.Contains(got, "onfocus=") && !strings.Contains(got, "onfocus") {
		t.Fatalf("attribute was not escaped: %q", got)
	}
	if !strings.Contains(got, `name="title"`) || strings.Count(got, "<input") != 1 || strings.Contains(got, "</input>") {
		t.Fatalf("void input render = %q", got)
	}
	if strings.Contains(got, `" autofocus`) {
		t.Fatalf("attribute injection survived: %q", got)
	}
}

func TestHTMLVoidElementRejectsChildren(t *testing.T) {
	expectRaise(t, AhdClassHTMLError, func() {
		AhdHTMLElement(AhdClassHTMLError, "br", nil, nil, []string{AhdHTMLText("x")})
	})
}

func TestHTMLRejectsInvalidNames(t *testing.T) {
	expectRaise(t, AhdClassHTMLError, func() {
		AhdHTMLElement(AhdClassHTMLError, "h1 onclick", nil, nil, nil)
	})
	expectRaise(t, AhdClassHTMLError, func() {
		AhdHTMLElement(AhdClassHTMLError, "h1", []string{`onclick="alert(1)"`}, []string{"x"}, nil)
	})
	expectRaise(t, AhdClassHTMLError, func() {
		AhdHTMLElement(AhdClassHTMLError, "", nil, nil, nil)
	})
}

func TestHTMLDocumentEscapesTitleAndHasUTF8Meta(t *testing.T) {
	page := AhdHTMLDocument(AhdClassHTMLError, `Notes & <x>`, []string{AhdHTMLElement(AhdClassHTMLError, "p", nil, nil, []string{AhdHTMLText("Hi")})})
	if !strings.HasPrefix(page, "<!doctype html>") {
		t.Fatalf("missing doctype: %q", page)
	}
	if !strings.Contains(page, `<meta charset="utf-8">`) {
		t.Fatalf("missing charset meta: %q", page)
	}
	if strings.Contains(page, "<x>") || !strings.Contains(page, "&amp;") || !strings.Contains(page, "&lt;x&gt;") {
		t.Fatalf("title was not escaped: %q", page)
	}
	if !strings.Contains(page, "<p>Hi</p>") {
		t.Fatalf("body missing: %q", page)
	}
}

func TestHTMLPreservesSourceTextAndTurkishEmoji(t *testing.T) {
	source := "Ayşe ☕"
	node := AhdHTMLText(source)
	if got := AhdHTMLRender(AhdClassHTMLError, node); got != source {
		t.Fatalf("render = %q", got)
	}
}
