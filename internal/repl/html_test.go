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
