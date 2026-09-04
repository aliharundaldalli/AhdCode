package build

import (
	"path/filepath"
	"strings"
	"testing"
)

// The v0.15 Web framework, exercised end to end: source in, native program
// out, real output compared. These cover the behaviour an application depends
// on -- the facade's types, the UI layer's escaping, and the configuration
// contract -- rather than one near-identical case per HTML tag.

func runWebProgram(t *testing.T, source string) string {
	t.Helper()
	directory := writeSources(t, map[string]string{"main.ahd": source})
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 {
		t.Fatalf("program exited with %d\nstderr:\n%s", code, errorOutput)
	}
	return out
}

// A. The facade's types are the underlying ones: a handler declared with
// Web's Request/Response registers on an HTTP.Server without conversion.
func TestWebTypesAreTheUnderlyingHTTPTypes(t *testing.T) {
	out := runWebProgram(t, `bring Web
from Web bring (Request, Response, HTMLNode)
bring HTTP
from HTTP bring Server

home: Function := (request: Request) -> Response {
    return Web.html(Web.UI.p("ok"))
}

server: Server := HTTP.server("127.0.0.1", 8199)
server.get("/", home)
node: HTMLNode := Web.UI.text("shared")
write(Web.render(node))
`)
	if strings.TrimSpace(out) != "shared" {
		t.Fatalf("expected \"shared\"; received %q", out)
	}
}

// B. Every String that becomes page content is escaped, whatever helper it
// went through. This is the security property the whole UI layer rests on.
func TestWebUIEscapesEveryTextEntryPoint(t *testing.T) {
	out := runWebProgram(t, `bring Web

payload: String := "<script>alert(1)</script>"
write(Web.render(Web.UI.p(payload)))
write(Web.render(Web.UI.h1(payload)))
write(Web.render(Web.UI.span(payload)))
write(Web.render(Web.UI.li(payload)))
write(Web.render(Web.UI.button(payload)))
write(Web.render(Web.UI.td(payload)))
write(Web.render(Web.UI.a("/x", payload)))
write(Web.render(Web.UI.option("v", payload)))
write(Web.render(Web.UI.textarea("n", payload)))
write(Web.render(Web.UI.text(payload)))
write(Web.render(Web.UI.img("/i.png", payload)))
`)
	if strings.Contains(out, "<script>") {
		t.Fatalf("raw markup reached the rendered output:\n%s", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 11 {
		t.Fatalf("expected 11 rendered lines; received %d:\n%s", len(lines), out)
	}
	for _, line := range lines {
		if !strings.Contains(line, "&lt;script&gt;") {
			t.Errorf("value was not escaped in %q", line)
		}
	}
}

// C. Structure, attributes, void elements, and nested composition all render
// the markup the low-level builder would have produced.
func TestWebUIComposesTheSameMarkupAsTheHTMLModule(t *testing.T) {
	out := runWebProgram(t, `bring Web
bring HTML
from Web bring HTMLNode

viaUI: HTMLNode := Web.UI.section(
    [
        Web.UI.h1("Title")
        Web.UI.p("Body")
        Web.UI.img("/logo.png", "Logo")
        Web.UI.br()
        Web.UI.ul([Web.UI.li("one"), Web.UI.liNodes([Web.UI.a("/two", "two")])])
    ]
    {"class": "hero"}
)

viaHTML: HTMLNode := HTML.element(
    "section"
    {"class": "hero"}
    [
        HTML.element("h1", {}, [HTML.text("Title")])
        HTML.element("p", {}, [HTML.text("Body")])
        HTML.element("img", {"src": "/logo.png", "alt": "Logo"}, [])
        HTML.element("br", {}, [])
        HTML.element(
            "ul"
            {}
            [
                HTML.element("li", {}, [HTML.text("one")])
                HTML.element("li", {}, [HTML.element("a", {"href": "/two"}, [HTML.text("two")])])
            ]
        )
    ]
)

write(HTML.render(viaUI))
write(HTML.render(viaUI) == HTML.render(viaHTML))
`)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || lines[1] != "true" {
		t.Fatalf("Web.UI markup differs from the low-level builder:\n%s", out)
	}
	expected := `<section class="hero"><h1>Title</h1><p>Body</p>` +
		`<img src="/logo.png" alt="Logo"><br><ul><li>one</li><li><a href="/two">two</a></li></ul></section>`
	if lines[0] != expected {
		t.Fatalf("unexpected markup:\n  want %s\n  got  %s", expected, lines[0])
	}
}

// D. A helper that adds an attribute of its own must not write into the Pair
// the caller passed: a Pair is a reference, and a shared attribute map would
// otherwise accumulate every element's attributes.
func TestWebUIDoesNotMutateCallerAttributes(t *testing.T) {
	out := runWebProgram(t, `bring Web

shared: Pair<String, String> := {"class": "card"}
write(Web.render(Web.UI.a("/one", "One", shared)))
write(Web.render(Web.UI.img("/i.png", "Alt", shared)))
write(Web.render(Web.UI.a("/two", "Two", shared)))
write(str(shared))
`)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines; received %d:\n%s", len(lines), out)
	}
	if strings.Contains(lines[2], "src=") || strings.Contains(lines[2], "/one") {
		t.Errorf("an earlier call leaked attributes into a later one: %q", lines[2])
	}
	if lines[3] != `{"class": "card"}` {
		t.Errorf("the caller's Pair was mutated: %q", lines[3])
	}
}

// E. Tables, forms, and the label/input pairing compose as ordinary nodes.
func TestWebUIBuildsTablesAndForms(t *testing.T) {
	out := runWebProgram(t, `bring Web

write(Web.render(Web.UI.table(
    [
        Web.UI.thead([Web.UI.tr([Web.UI.th("Name")])])
        Web.UI.tbody([Web.UI.tr([Web.UI.td("Ada")])])
    ]
)))
write(Web.render(Web.UI.formTo(
    "/save"
    "post"
    [
        Web.UI.labelFor("name", "Name")
        Web.UI.input("text", "name", "", {"id": "name"})
        Web.UI.button("Save", {"type": "submit"})
    ]
)))
`)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	wantTable := "<table><thead><tr><th>Name</th></tr></thead><tbody><tr><td>Ada</td></tr></tbody></table>"
	if lines[0] != wantTable {
		t.Errorf("table markup:\n  want %s\n  got  %s", wantTable, lines[0])
	}
	// A helper's own attributes are appended after the caller's, which is why
	// id comes before type here: the order is deterministic, not accidental.
	wantForm := `<form action="/save" method="post"><label for="name">Name</label>` +
		`<input id="name" type="text" name="name" value=""><button type="submit">Save</button></form>`
	if lines[1] != wantForm {
		t.Errorf("form markup:\n  want %s\n  got  %s", wantForm, lines[1])
	}
}

// F. A Layout is an ordinary Function wrapping a Page's nodes, and the
// document shell escapes the title like any other text.
func TestWebDocumentShellWrapsPageContent(t *testing.T) {
	out := runWebProgram(t, `bring Web
from Web bring HTMLNode

card: Function := (title: String) -> HTMLNode {
    return Web.UI.article([Web.UI.h2(title)])
}

layout: Function := (title: String, content: List<HTMLNode>) -> String {
    body: Local List<HTMLNode> := [Web.UI.main(content)]
    return Web.document(title, body, [Web.UI.stylesheet("/assets/app.css")])
}

write(layout("A <b>title</b>", [card("Card")]))
`)
	rendered := strings.TrimSpace(out)
	for _, fragment := range []string{
		"<!doctype html>",
		`<title>A &lt;b&gt;title&lt;/b&gt;</title>`,
		`<link rel="stylesheet" href="/assets/app.css">`,
		"<main><article><h2>Card</h2></article></main>",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Errorf("document is missing %q:\n%s", fragment, rendered)
		}
	}
}

// G. The environment contract: the canonical URL is APP_PROTOCOL://APP_HOST,
// development derives the same identity with .test appended, and production
// never gains that suffix.
func TestWebConfigDerivesEnvironmentURLs(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `bring Web
from Web bring AppConfig

config: AppConfig := Web.configure()
write(config.url())
write(config.developmentURL())
write(config.effectiveURL())
write(config.address())
`})
	for _, testCase := range []struct{ environment, expected string }{
		{"development", "https://example.com.test"},
		{"test", "https://example.com"},
		{"production", "https://example.com"},
	} {
		out := runWebEnvProgram(t, directory, map[string]string{
			"APP_NAME": "Example", "APP_ENV": testCase.environment,
			"APP_HOST": "example.com", "APP_PROTOCOL": "https",
			"SERVER_HOST": "127.0.0.1", "SERVER_PORT": "8080",
		})
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if lines[0] != "https://example.com" {
			t.Errorf("%s: canonical URL was %q", testCase.environment, lines[0])
		}
		if lines[1] != "https://example.com.test" {
			t.Errorf("%s: development URL was %q", testCase.environment, lines[1])
		}
		if lines[2] != testCase.expected {
			t.Errorf("%s: effective URL was %q, expected %q", testCase.environment, lines[2], testCase.expected)
		}
		if lines[3] != "127.0.0.1:8080" {
			t.Errorf("%s: bind address was %q", testCase.environment, lines[3])
		}
	}
}

// H. Invalid configuration fails with a message that names the key, and the
// error path never repeats a secret's value.
func TestWebConfigRejectsInvalidValues(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `bring Web
from Web bring AppConfig

config: AppConfig := Web.configure()
write(config.url())
`})
	valid := map[string]string{
		"APP_NAME": "Example", "APP_ENV": "development",
		"APP_HOST": "example.com", "APP_PROTOCOL": "https",
		"SERVER_HOST": "127.0.0.1", "SERVER_PORT": "8080",
		"DB_PASSWORD": "s3cret-canary-value",
	}
	for _, testCase := range []struct{ key, value, expected string }{
		{"APP_ENV", "dev", "APP_ENV must be one of"},
		{"APP_ENV", "DEVELOPMENT", "APP_ENV must be one of"},
		{"APP_ENV", "", "APP_ENV is required"},
		{"APP_PROTOCOL", "HTTP", "APP_PROTOCOL must be http or https"},
		{"APP_PROTOCOL", "ftp", "APP_PROTOCOL must be http or https"},
		{"APP_PROTOCOL", "", "APP_PROTOCOL is required"},
		{"APP_HOST", "https://example.com", "not a URL"},
		{"APP_HOST", "example.com/path", "without a path"},
		{"APP_HOST", "example.com:8080", "without a port"},
		{"SERVER_PORT", "eight", "whole number"},
		{"SERVER_PORT", "70000", "between 1 and 65535"},
	} {
		environment := make(map[string]string, len(valid))
		for key, value := range valid {
			environment[key] = value
		}
		environment[testCase.key] = testCase.value
		_, errorOutput := runWebEnvProgramExpectingFailure(t, directory, environment)
		if !strings.Contains(errorOutput, testCase.expected) {
			t.Errorf("%s=%q: expected a message containing %q; received %q",
				testCase.key, testCase.value, testCase.expected, strings.TrimSpace(errorOutput))
		}
		if strings.Contains(errorOutput, "s3cret-canary-value") {
			t.Errorf("%s=%q: a secret's value appeared in a configuration error", testCase.key, testCase.value)
		}
	}
}

// I. The low-level modules remain usable on their own after Web exists: an
// application is never forced through the facade.
func TestLowLevelHTTPAndHTMLStillWorkWithoutWeb(t *testing.T) {
	out := runWebProgram(t, `bring HTTP
from HTTP bring (Server, Request, Response)
bring HTML
from HTML bring HTMLNode

home: Function := (request: Request) -> Response {
    return HTTP.html(HTML.render(HTML.element("p", {}, [HTML.text("low level")])))
}

server: Server := HTTP.server("127.0.0.1", 8198)
server.get("/", home)
node: HTMLNode := HTML.text("<b>")
write(HTML.render(node))
`)
	if strings.TrimSpace(out) != "&lt;b&gt;" {
		t.Fatalf("low-level HTML changed behaviour: %q", out)
	}
}
