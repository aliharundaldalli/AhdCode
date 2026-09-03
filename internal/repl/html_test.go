package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestHTMLWorksInTheREPL(t *testing.T) {
	input := `bring HTML
write(HTML.render(HTML.text("<script>alert(1)</script>")))
write(HTML.document("Tom & Jerry", [HTML.text("Ayşe ☕")]))
`
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(input), &output, &errorOutput, "AhdCode v0.5.0")
	text := output.String()
	if strings.Contains(text, "<script>") || !strings.Contains(text, "&lt;script&gt;") {
		t.Fatalf("REPL HTML output was not escaped:\n%s\nerrors:\n%s", text, errorOutput.String())
	}
	if !strings.Contains(text, "Tom &amp; Jerry") || !strings.Contains(text, "Ayşe ☕") {
		t.Fatalf("REPL HTML document omitted expected text:\n%s", text)
	}
	if errorOutput.Len() != 0 {
		t.Fatalf("REPL reported errors:\n%s", errorOutput.String())
	}
}

func TestHTMLParseWorksInTheREPL(t *testing.T) {
	input := `bring HTML
doc := HTML.parse("<h1>Hello &amp; AhdCode</h1>")
heading := doc.first("h1")
write(heading != null)
if heading != null {
    write(heading.text())
}
`
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(input), &output, &errorOutput, "AhdCode v0.7.0")
	text := output.String()
	if !strings.Contains(text, "true") {
		t.Fatalf("REPL did not report heading != null:\n%s\nerrors:\n%s", text, errorOutput.String())
	}
	if !strings.Contains(text, "Hello & AhdCode") {
		t.Fatalf("REPL HTML.parse text() did not decode the entity:\n%s", text)
	}
	if errorOutput.Len() != 0 {
		t.Fatalf("REPL reported errors:\n%s", errorOutput.String())
	}
}
