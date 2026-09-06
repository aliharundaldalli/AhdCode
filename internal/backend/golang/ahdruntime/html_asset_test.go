package ahdruntime

import (
	"strings"
	"testing"
)

func TestHTMLAssetIsInvisibleInOrdinaryRender(t *testing.T) {
	t.Cleanup(ahdHTMLResetExposedAssets)
	asset := AhdHTMLAsset(AhdClassHTMLError, "stylesheet", "app.css")
	card := AhdHTMLElement(AhdClassHTMLError, "div", []string{"class"}, []string{"card"}, []string{
		asset,
		AhdHTMLText("Hello"),
	})
	if got := AhdHTMLRender(AhdClassHTMLError, card); got != `<div class="card">Hello</div>` {
		t.Fatalf("render = %q", got)
	}
	if !ahdHTMLAssetExposed("app.css") {
		t.Fatal("declaring a stylesheet should expose the managed path")
	}
}

func TestHTMLComposeDocumentCollectsLayoutPageAndComponentAssetsOnce(t *testing.T) {
	t.Cleanup(ahdHTMLResetExposedAssets)
	ahdHTMLSetManagedAssetPrefix("/assets/")
	layoutCSS := AhdHTMLAsset(AhdClassHTMLError, "stylesheet", "app.css")
	pageCSS := AhdHTMLAsset(AhdClassHTMLError, "stylesheet", "page.css")
	componentCSS := AhdHTMLAsset(AhdClassHTMLError, "stylesheet", "component.css")
	component := AhdHTMLElement(AhdClassHTMLError, "article", nil, nil, []string{
		componentCSS,
		AhdHTMLText("Card"),
	})
	page := AhdHTMLElement(AhdClassHTMLError, "main", nil, nil, []string{
		pageCSS,
		component,
		component,
	})
	layout := AhdHTMLElement(AhdClassHTMLError, "div", nil, nil, []string{
		layoutCSS,
		page,
	})
	document := AhdHTMLComposeDocument(AhdClassHTMLError, "Home", []string{layout}, nil)
	if strings.Count(document, `href="/assets/app.css"`) != 1 {
		t.Fatalf("app.css count: %s", document)
	}
	if strings.Count(document, `href="/assets/page.css"`) != 1 {
		t.Fatalf("page.css count: %s", document)
	}
	if strings.Count(document, `href="/assets/component.css"`) != 1 {
		t.Fatalf("component.css count: %s", document)
	}
	appAt := strings.Index(document, `href="/assets/app.css"`)
	pageAt := strings.Index(document, `href="/assets/page.css"`)
	componentAt := strings.Index(document, `href="/assets/component.css"`)
	if !(appAt < pageAt && pageAt < componentAt) {
		t.Fatalf("stylesheet order is not layout, page, component: %s", document)
	}
	if !strings.Contains(document, "<article>Card</article><article>Card</article>") {
		t.Fatalf("component body should render twice: %s", document)
	}
}

func TestHTMLComposeDocumentOmitsUnusedComponentAssets(t *testing.T) {
	t.Cleanup(ahdHTMLResetExposedAssets)
	used := AhdHTMLAsset(AhdClassHTMLError, "stylesheet", "used.css")
	_ = AhdHTMLAsset(AhdClassHTMLError, "stylesheet", "unused.css")
	page := AhdHTMLElement(AhdClassHTMLError, "main", nil, nil, []string{used, AhdHTMLText("ok")})
	document := AhdHTMLComposeDocument(AhdClassHTMLError, "Home", []string{page}, nil)
	if !strings.Contains(document, `href="/assets/used.css"`) {
		t.Fatalf("used.css missing: %s", document)
	}
	if strings.Contains(document, "unused.css") {
		t.Fatalf("unused component asset was emitted: %s", document)
	}
}

func TestHTMLComposeDocumentScriptModesAndDedupe(t *testing.T) {
	t.Cleanup(ahdHTMLResetExposedAssets)
	normal := AhdHTMLAsset(AhdClassHTMLError, "script", "app.js")
	deferScript := AhdHTMLAsset(AhdClassHTMLError, "script-defer", "later.js")
	module := AhdHTMLAsset(AhdClassHTMLError, "script-module", "mod.js")
	again := AhdHTMLAsset(AhdClassHTMLError, "script", "app.js")
	page := AhdHTMLElement(AhdClassHTMLError, "main", nil, nil, []string{normal, deferScript, module, again})
	document := AhdHTMLComposeDocument(AhdClassHTMLError, "Home", []string{page}, nil)
	if strings.Count(document, `src="/assets/app.js"`) != 1 {
		t.Fatalf("app.js should appear once: %s", document)
	}
	if !strings.Contains(document, `<script src="/assets/later.js" defer></script>`) {
		t.Fatalf("defer script missing: %s", document)
	}
	if !strings.Contains(document, `<script type="module" src="/assets/mod.js"></script>`) {
		t.Fatalf("module script missing: %s", document)
	}
	scriptAt := strings.Index(document, `<script src="/assets/app.js"></script>`)
	bodyClose := strings.LastIndex(document, "</body>")
	if scriptAt < 0 || scriptAt > bodyClose {
		t.Fatalf("scripts should be emitted before </body>: %s", document)
	}
}

func TestHTMLInlineAssetsAreDistinctFromEscapedText(t *testing.T) {
	t.Cleanup(ahdHTMLResetExposedAssets)
	css := AhdHTMLInlineAsset(AhdClassHTMLError, "inline-css", "card", ".card{color:red}")
	js := AhdHTMLInlineAsset(AhdClassHTMLError, "inline-js", "boot", `console.log("ok")`)
	text := AhdHTMLText(`<script>alert(1)</script>`)
	page := AhdHTMLElement(AhdClassHTMLError, "main", nil, nil, []string{css, js, text})
	document := AhdHTMLComposeDocument(AhdClassHTMLError, "Home", []string{page}, nil)
	if !strings.Contains(document, "<style>.card{color:red}</style>") {
		t.Fatalf("inline css missing: %s", document)
	}
	if !strings.Contains(document, `<script>console.log("ok")</script>`) {
		t.Fatalf("inline js missing: %s", document)
	}
	if !strings.Contains(document, `&lt;script&gt;alert(1)&lt;/script&gt;`) {
		t.Fatalf("ordinary text must stay escaped: %s", document)
	}
	duplicate := AhdHTMLInlineAsset(AhdClassHTMLError, "inline-css", "card", ".card{color:blue}")
	again := AhdHTMLComposeDocument(AhdClassHTMLError, "Home", []string{
		AhdHTMLElement(AhdClassHTMLError, "main", nil, nil, []string{css, duplicate}),
	}, nil)
	if strings.Count(again, "<style>") != 1 || !strings.Contains(again, ".card{color:red}") {
		t.Fatalf("identical inline keys should keep the first: %s", again)
	}
}

func TestHTMLInlineAssetsRejectHostBreakout(t *testing.T) {
	expectRaise(t, AhdClassHTMLError, func() {
		AhdHTMLInlineAsset(AhdClassHTMLError, "inline-css", "x", "x</style><script>")
	})
	expectRaise(t, AhdClassHTMLError, func() {
		AhdHTMLInlineAsset(AhdClassHTMLError, "inline-js", "x", "x</script><script>")
	})
}

func TestHTMLAssetRejectsTraversal(t *testing.T) {
	expectRaise(t, AhdClassHTMLError, func() {
		AhdHTMLAsset(AhdClassHTMLError, "stylesheet", "../secret.css")
	})
	expectRaise(t, AhdClassHTMLError, func() {
		AhdHTMLAsset(AhdClassHTMLError, "stylesheet", ".hidden.css")
	})
}
