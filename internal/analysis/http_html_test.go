package analysis

import (
	"path/filepath"
	"testing"
)

func TestHTTPAndHTMLAreDiscoveredAfterBringAndFrom(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()

	store.Open(path, "bring HT\n")
	items := store.Completion(path, len("bring HT"))
	if !hasLabel(items, "HTTP") || detailOf(items, "HTTP") != "module HTTP" {
		t.Fatalf("expected HTTP after `bring HT`, got %#v", items)
	}
	if !hasLabel(items, "HTML") || detailOf(items, "HTML") != "module HTML" {
		t.Fatalf("expected HTML after `bring HT`, got %#v", items)
	}

	store.Open(path, "from HTTP bring HTTP\n")
	items = store.Completion(path, len("from HTTP bring HTTP"))
	if detailOf(items, "HTTPError") != "Class HTTPError" {
		t.Fatalf("expected exported Class HTTPError after `from HTTP bring HTTP`, got %#v", items)
	}
	store.Open(path, "from HTML bring HTML\n")
	items = store.Completion(path, len("from HTML bring HTML"))
	if detailOf(items, "HTMLError") != "Class HTMLError" || detailOf(items, "HTMLNode") != "Class HTMLNode" {
		t.Fatalf("expected HTMLError/HTMLNode after `from HTML bring HTML`, got %#v", items)
	}
}

func TestHTTPAndHTMLNamespaceMembersCarryRealSignatures(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()

	text := "bring HTTP\nx := HTTP.\n"
	store.Open(path, text)
	items := store.Completion(path, offsetOf(t, text, "HTTP.")+len("HTTP."))
	wantHTTP := map[string]string{
		"server":    "server: (host: String, port: Int, maxBodyBytes: Int := default) -> Server",
		"text":      "text: (body: String, status: Int := default) -> Response",
		"html":      "html: (body: String, status: Int := default) -> Response",
		"response":  "response: (status: Int, body: String, contentType: String) -> Response",
		"redirect":  "redirect: (location: String, status: Int := default) -> Response",
		"Server":    "Class Server",
		"Request":   "Class Request",
		"Response":  "Class Response",
		"HTTPError": "Class HTTPError",
	}
	for label, detail := range wantHTTP {
		if detailOf(items, label) != detail {
			t.Fatalf("HTTP.%s detail = %q; want %q (items %#v)", label, detailOf(items, label), detail, items)
		}
	}

	text = "bring HTML\nx := HTML.\n"
	store.Open(path, text)
	items = store.Completion(path, offsetOf(t, text, "HTML.")+len("HTML."))
	wantHTML := map[string]string{
		"text":      "text: (value: String) -> HTMLNode",
		"element":   "element: (name: String, attributes: Pair<String, String>, children: List<HTMLNode>) -> HTMLNode",
		"render":    "render: (node: HTMLNode) -> String",
		"document":  "document: (title: String, body: List<HTMLNode>) -> String",
		"HTMLNode":  "Class HTMLNode",
		"HTMLError": "Class HTMLError",
	}
	for label, detail := range wantHTML {
		if detailOf(items, label) != detail {
			t.Fatalf("HTML.%s detail = %q; want %q (items %#v)", label, detailOf(items, label), detail, items)
		}
	}
}

func TestHTTPAndHTMLHoverAndSignatureHelpComeFromTheCompiler(t *testing.T) {
	text := "bring HTTP\nfrom HTTP bring Server\nfrom HTTP bring Request\nfrom HTTP bring Response\n" +
		"bring HTML\nfrom HTML bring HTMLNode\n" +
		"app: Server := HTTP.server(\"127.0.0.1\", 8080)\n" +
		"page := HTTP.text(\"ok\")\n" +
		"markup := HTTP.html(\"<h1>Hi</h1>\")\n" +
		"node := HTML.element(\"p\", {}, [HTML.text(\"Hi\")])\n" +
		"doc := HTML.document(\"Notes\", [node])\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	if hover, ok := store.Hover(path, offsetOf(t, text, "HTTP.server")+len("HTTP.")+1); !ok || hover.Text != "server: (host: String, port: Int, maxBodyBytes: Int := default) -> Server" {
		t.Fatalf("hover on HTTP.server = %#v, ok = %v", hover, ok)
	}
	if hover, ok := store.Hover(path, offsetOf(t, text, "HTTP.text")+len("HTTP.")+1); !ok || hover.Text != "text: (body: String, status: Int := default) -> Response" {
		t.Fatalf("hover on HTTP.text = %#v, ok = %v", hover, ok)
	}
	if hover, ok := store.Hover(path, offsetOf(t, text, "HTTP.html")+len("HTTP.")+1); !ok || hover.Text != "html: (body: String, status: Int := default) -> Response" {
		t.Fatalf("hover on HTTP.html = %#v, ok = %v", hover, ok)
	}
	if hover, ok := store.Hover(path, offsetOf(t, text, "HTML.element")+len("HTML.")+1); !ok || hover.Text != "element: (name: String, attributes: Pair<String, String>, children: List<HTMLNode>) -> HTMLNode" {
		t.Fatalf("hover on HTML.element = %#v, ok = %v", hover, ok)
	}
	if hover, ok := store.Hover(path, offsetOf(t, text, "HTML.document")+len("HTML.")+1); !ok || hover.Text != "document: (title: String, body: List<HTMLNode>) -> String" {
		t.Fatalf("hover on HTML.document = %#v, ok = %v", hover, ok)
	}

	if help, ok := store.SignatureHelp(path, offsetOf(t, text, "HTTP.server(")+len("HTTP.server(")); !ok || help.Label != "(host: String, port: Int, maxBodyBytes: Int := default) -> Server" {
		t.Fatalf("signature help for HTTP.server = %#v, ok = %v", help, ok)
	}
	if help, ok := store.SignatureHelp(path, offsetOf(t, text, "HTTP.text(")+len("HTTP.text(")); !ok || help.Label != "(body: String, status: Int := default) -> Response" {
		t.Fatalf("signature help for HTTP.text = %#v, ok = %v", help, ok)
	}
	if help, ok := store.SignatureHelp(path, offsetOf(t, text, "HTML.element(")+len("HTML.element(")); !ok || help.Label != "(name: String, attributes: Pair<String, String>, children: List<HTMLNode>) -> HTMLNode" {
		t.Fatalf("signature help for HTML.element = %#v, ok = %v", help, ok)
	}
	if help, ok := store.SignatureHelp(path, offsetOf(t, text, "HTML.document(")+len("HTML.document(")); !ok || help.Label != "(title: String, body: List<HTMLNode>) -> String" {
		t.Fatalf("signature help for HTML.document = %#v, ok = %v", help, ok)
	}
}
