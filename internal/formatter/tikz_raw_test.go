package formatter

import (
	"strings"
	"testing"
)

// TikZ source is text inside an AhdCode raw String. The formatter lays out the
// surrounding call but must never touch the literal's content: TikZ uses
// backslashes, braces, brackets, percent signs, and its own indentation.
func TestFormatterPreservesRawTikZExactly(t *testing.T) {
	tikz := "\n" +
		"\\draw[thick,  ->]   (0,0) -- ++(2,1)  node[midway,above] {\\textbf{Türkçe}  50\\%};\n" +
		"  % a comment {with braces} [and brackets]\n" +
		"\t\\node[draw, fit=(a)(b)] at ($(a)!0.5!(b)$) {\\{literal\\} {name} 😊};\n" +
		"\\foreach \\angle in {0, 15, ..., 345} {\n" +
		"        \\draw (\\angle:1cm) -- (\\angle:1.2cm);\n" +
		"}\n"
	text := "bring Latex as L\n\n" +
		"drawing:String:=L.tikz(source: r\"\"\"" + tikz + "\"\"\", libraries: [\"calc\",\"fit\"])\n" +
		"line := L.tikz(r\"\\node   {x};  % keep\")\n"
	formatted := formatText(t, text)
	if !strings.Contains(formatted, `r"""`+tikz+`"""`) {
		t.Fatalf("the raw triple String content changed:\n%s", formatted)
	}
	if !strings.Contains(formatted, `r"\node   {x};  % keep"`) {
		t.Fatalf("the raw String content changed:\n%s", formatted)
	}
	if again := formatText(t, formatted); again != formatted {
		t.Fatalf("formatting TikZ source is not idempotent:\nfirst:\n%s\nsecond:\n%s", formatted, again)
	}
}
