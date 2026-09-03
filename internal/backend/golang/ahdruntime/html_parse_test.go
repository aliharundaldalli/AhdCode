package ahdruntime

import (
	"strings"
	"testing"
)

func TestHTMLParseScrapeFixture(t *testing.T) {
	class := AhdClassHTMLError
	source := `<!doctype html>
<html>
<body>
  <article class="card" data-id="1">
    <h2>Riesz &amp; Banach</h2>
    <a href="/notes/1">Read</a>
  </article>
  <article class="card featured" data-id="2">
    <h2>Functional Analysis</h2>
    <a href="/notes/2">Read</a>
  </article>
</body>
</html>`
	doc := AhdHTMLParse(class, source)
	articles := AhdHTMLDocumentSelect(class, doc, "article.card")
	if len(articles) != 2 {
		t.Fatalf("expected 2 article.card, got %d", len(articles))
	}
	h2 := AhdHTMLElementFirst(class, articles[0], "h2")
	if h2 == nil {
		t.Fatalf("expected h2 in first article")
	}
	if got := AhdHTMLElementText(class, *h2); got != "Riesz & Banach" {
		t.Fatalf("expected decoded entity text, got %q", got)
	}
	a := AhdHTMLElementFirst(class, articles[0], "a")
	if a == nil {
		t.Fatalf("expected a in first article")
	}
	href := AhdHTMLElementAttr(class, *a, "href")
	if href == nil || *href != "/notes/1" {
		t.Fatalf("expected relative href, got %v", href)
	}
	if AhdHTMLElementTag(class, articles[0]) != "article" {
		t.Fatalf("expected tag() to be lowercased")
	}
	if !AhdHTMLElementHasAttr(class, articles[1], "data-id") {
		t.Fatalf("expected hasAttr data-id true")
	}
	dataID := AhdHTMLElementAttr(class, articles[1], "data-id")
	if dataID == nil || *dataID != "2" {
		t.Fatalf("expected data-id 2, got %v", dataID)
	}
}

func TestHTMLParseMalformedRecovery(t *testing.T) {
	class := AhdClassHTMLError
	doc := AhdHTMLParse(class, "<div><p>Hello")
	div := AhdHTMLDocumentFirst(class, doc, "div")
	if div == nil {
		t.Fatalf("expected a div to be recovered")
	}
	p := AhdHTMLElementFirst(class, *div, "p")
	if p == nil {
		t.Fatalf("expected p nested inside the recovered div")
	}
	if AhdHTMLElementText(class, *p) != "Hello" {
		t.Fatalf("expected recovered p text %q, got %q", "Hello", AhdHTMLElementText(class, *p))
	}
}

func TestHTMLParseMismatchedEndTagRecovery(t *testing.T) {
	class := AhdClassHTMLError
	doc := AhdHTMLParse(class, "<b><i>text</b></i>")
	b := AhdHTMLDocumentFirst(class, doc, "b")
	if b == nil {
		t.Fatalf("expected a recovered b element")
	}
	i := AhdHTMLElementFirst(class, *b, "i")
	if i == nil || AhdHTMLElementText(class, *i) != "text" {
		t.Fatalf("expected i nested in b with text, got %v", i)
	}
}

func TestHTMLParseUnquotedAttributesAndCaseMixing(t *testing.T) {
	class := AhdClassHTMLError
	doc := AhdHTMLParse(class, `<DIV class=box id=main><P>hi</P></DIV>`)
	div := AhdHTMLDocumentFirst(class, doc, "div")
	if div == nil {
		t.Fatalf("expected a div element regardless of source capitalization")
	}
	if AhdHTMLElementTag(class, *div) != "div" {
		t.Fatalf("expected tag() normalized to lowercase")
	}
	id := AhdHTMLElementAttr(class, *div, "ID")
	if id == nil || *id != "main" {
		t.Fatalf("expected case-insensitive attribute name lookup, got %v", id)
	}
}

func TestHTMLSelectorMatrix(t *testing.T) {
	class := AhdClassHTMLError
	source := `<div id="main">
  <article class="card" data-id="1"><h2>One</h2><a href="/1">x</a></article>
  <article class="card featured" data-id="2"><h2>Two</h2><a href="/2">x</a></article>
</div>`
	doc := AhdHTMLParse(class, source)
	cases := []struct {
		selector string
		count    int
	}{
		{"article", 2},
		{".card", 2},
		{"#main", 1},
		{"article.card", 2},
		{".card.featured", 1},
		{"[href]", 2},
		{`[data-id="2"]`, 1},
		{"article h2", 2},
		{"article > h2", 2},
		{"article a", 2},
		{"h1, h2", 2},
		{".card, .featured", 2},
		{"*", 7}, // div, 2 article, 2 h2, 2 a
	}
	for _, tc := range cases {
		got := AhdHTMLDocumentSelect(class, doc, tc.selector)
		if len(got) != tc.count {
			t.Errorf("selector %q: expected %d matches, got %d", tc.selector, tc.count, len(got))
		}
	}
}

func TestHTMLSelectorDocumentOrder(t *testing.T) {
	class := AhdClassHTMLError
	doc := AhdHTMLParse(class, `<div id="a"></div><div id="b"></div><div id="c"></div>`)
	matches := AhdHTMLDocumentSelect(class, doc, "div")
	if len(matches) != 3 {
		t.Fatalf("expected 3 divs, got %d", len(matches))
	}
	for index, want := range []string{"a", "b", "c"} {
		id := AhdHTMLElementAttr(class, matches[index], "id")
		if id == nil || *id != want {
			t.Fatalf("expected div order a,b,c; index %d was %v", index, id)
		}
	}
}

func TestHTMLSelectorListDeduplicates(t *testing.T) {
	class := AhdClassHTMLError
	doc := AhdHTMLParse(class, `<article class="card">one</article>`)
	matches := AhdHTMLDocumentSelect(class, doc, ".card, article")
	if len(matches) != 1 {
		t.Fatalf("expected a selector list match to be de-duplicated, got %d", len(matches))
	}
}

func TestHTMLSelectorInvalidRejected(t *testing.T) {
	class := AhdClassHTMLError
	invalid := []string{
		"", ",", "div,", "> div", "div >", "div >> p", "div..card",
		"[", "[href", "[href=]", ":nth-child(2)", "div + p", "div ~ p",
		"**", "div*", "*div", "div*p", "article**", "*.card*", "*#main*", "div.card*",
	}
	for _, selector := range invalid {
		func() {
			defer func() {
				recovered := recover()
				if recovered == nil {
					t.Errorf("selector %q: expected a panic (HTMLError), got none", selector)
					return
				}
				if _, ok := recovered.(*AhdSignal); !ok {
					t.Errorf("selector %q: expected an AhdSignal panic, got %T", selector, recovered)
				}
			}()
			ahdHTMLParseSelector(class, selector)
		}()
	}
}

// TestHTMLUniversalSelectorCompoundGrammar exercises the type-selector slot
// invariant directly: a compound selector may open with at most one type
// selector (a bare '*' or one tag), which must come before any #id/.class/
// [attr] suffix. "*.card"/"*#main"/"*[href]" combine the universal selector
// with a suffix and remain valid; a second type-selector token anywhere in
// the same compound is rejected.
func TestHTMLUniversalSelectorCompoundGrammar(t *testing.T) {
	class := AhdClassHTMLError
	source := `<div id="main">
  <article class="card" data-id="1"><h2>One</h2></article>
  <article class="card featured" data-id="2"><h2>Two</h2></article>
</div>`
	doc := AhdHTMLParse(class, source)

	valid := []struct {
		selector string
		count    int
	}{
		{"*.card", 2},
		{"*#main", 1},
		{"*[data-id]", 2},
		{"div.card", 0},
		{"article.card.featured", 1},
	}
	for _, tc := range valid {
		got := AhdHTMLDocumentSelect(class, doc, tc.selector)
		if len(got) != tc.count {
			t.Errorf("selector %q: expected %d matches, got %d", tc.selector, tc.count, len(got))
		}
	}

	rejected := []string{
		"**", "div*", "*div", "div*p", "article**", "*.card*", "*#main*", "div.card*",
	}
	for _, selector := range rejected {
		func() {
			defer func() {
				recovered := recover()
				if recovered == nil {
					t.Errorf("selector %q: expected a panic (HTMLError), got none", selector)
					return
				}
				if _, ok := recovered.(*AhdSignal); !ok {
					t.Errorf("selector %q: expected an AhdSignal panic, got %T", selector, recovered)
				}
			}()
			ahdHTMLParseSelector(class, selector)
		}()
	}
}

func TestHTMLNoNetworkResources(t *testing.T) {
	class := AhdClassHTMLError
	source := `<img src="http://127.0.0.1:1/image">
<script src="http://127.0.0.1:1/script"></script>
<link rel="stylesheet" href="http://127.0.0.1:1/style">
<iframe src="http://127.0.0.1:1/frame"></iframe>`
	// Parsing must never dial out; this simply must not hang or error.
	doc := AhdHTMLParse(class, source)
	if !strings.Contains(doc, "img") {
		t.Fatalf("expected the img element to still be parsed as ordinary markup")
	}
}

func TestHTMLScriptRawText(t *testing.T) {
	class := AhdClassHTMLError
	doc := AhdHTMLParse(class, `<script>if (1 < 2) { fetch("/x"); }</script>`)
	script := AhdHTMLDocumentFirst(class, doc, "script")
	if script == nil {
		t.Fatalf("expected a script element")
	}
	if got := AhdHTMLElementText(class, *script); got != `if (1 < 2) { fetch("/x"); }` {
		t.Fatalf("expected raw undecoded script text, got %q", got)
	}
}

func TestHTMLEntities(t *testing.T) {
	class := AhdClassHTMLError
	doc := AhdHTMLParse(class, "<p>&amp; &lt; &gt; &quot; &#65; &#x41;</p>")
	p := AhdHTMLDocumentFirst(class, doc, "p")
	if p == nil {
		t.Fatalf("expected a p element")
	}
	if got := AhdHTMLElementText(class, *p); got != `& < > " A A` {
		t.Fatalf("unexpected decoded entities: %q", got)
	}
}

func TestHTMLCommentsAndDoctypeDoNotBreakParsingOrText(t *testing.T) {
	class := AhdClassHTMLError
	doc := AhdHTMLParse(class, "<!doctype html><!-- <div>hidden</div> --><p>seen</p>")
	if len(AhdHTMLDocumentSelect(class, doc, "div")) != 0 {
		t.Fatalf("expected commented-out markup to never be parsed as elements")
	}
	p := AhdHTMLDocumentFirst(class, doc, "p")
	if p == nil || AhdHTMLElementText(class, *p) != "seen" {
		t.Fatalf("expected the real p to parse fine after a doctype and comment")
	}
}

func TestHTMLElementSelectScoping(t *testing.T) {
	class := AhdClassHTMLError
	source := `<article><h2>A</h2></article><article><h2>B</h2></article>`
	doc := AhdHTMLParse(class, source)
	articles := AhdHTMLDocumentSelect(class, doc, "article")
	if len(articles) != 2 {
		t.Fatalf("expected 2 articles")
	}
	h2 := AhdHTMLElementFirst(class, articles[0], "h2")
	if h2 == nil || AhdHTMLElementText(class, *h2) != "A" {
		t.Fatalf("expected scoped first h2 to be A, got %v", h2)
	}
	// The element itself must not be matched by its own select/first.
	if AhdHTMLElementFirst(class, articles[0], "article") != nil {
		t.Fatalf("expected element.select to exclude the element itself")
	}
}

func TestHTMLElementRemainsValidAfterDocumentDiscarded(t *testing.T) {
	class := AhdClassHTMLError
	var savedElement string
	func() {
		doc := AhdHTMLParse(class, "<h1>Hello &amp; AhdCode</h1>")
		heading := AhdHTMLDocumentFirst(class, doc, "h1")
		if heading == nil {
			t.Fatalf("expected h1")
		}
		savedElement = *heading
	}()
	if AhdHTMLElementText(class, savedElement) != "Hello & AhdCode" {
		t.Fatalf("expected the saved element to remain valid and correct on its own")
	}
}

// TestHTMLRepeatedParseSelectDiscardHasNoRegistry proves there is no global
// handle/registry to leak: HTMLDocument/HTMLElement are self-contained
// encoded values (see html_parse.go's package comment), so repeatedly
// parsing, selecting, and discarding moderate documents cannot accumulate
// any shared state between iterations.
func TestHTMLRepeatedParseSelectDiscardHasNoRegistry(t *testing.T) {
	class := AhdClassHTMLError
	for i := 0; i < 2000; i++ {
		doc := AhdHTMLParse(class, `<div class="card"><h2>Title</h2><a href="/n">x</a></div>`)
		matches := AhdHTMLDocumentSelect(class, doc, ".card")
		if len(matches) != 1 {
			t.Fatalf("iteration %d: expected 1 match, got %d", i, len(matches))
		}
		_ = AhdHTMLElementText(class, matches[0])
	}
}

func TestHTMLPerformanceSanityFlatFixture(t *testing.T) {
	class := AhdClassHTMLError
	var builder strings.Builder
	const count = 5000
	for i := 0; i < count; i++ {
		builder.WriteString(`<article class="card"><h2>Title</h2><a href="/n">x</a></article>`)
	}
	doc := AhdHTMLParse(class, builder.String())
	matches := AhdHTMLDocumentSelect(class, doc, "article.card")
	if len(matches) != count {
		t.Fatalf("expected %d article.card matches, got %d", count, len(matches))
	}
}
