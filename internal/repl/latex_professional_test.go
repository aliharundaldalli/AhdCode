package repl

import (
	"bytes"
	"strings"
	"testing"
)

// v1.3.0 Latex helpers return the same fragments in the persistent REPL as in
// a compiled program; compiling a PDF stays outside the interactive evaluator.
func TestLatexProfessionalHelpersInPersistentREPL(t *testing.T) {
	input := `bring Latex as L
write(L.link("site", "https://ahdcode.org/?a=1&b=2"))
write(L.pageNumber())
footer: String := L.footer(center: "Page " + L.pageNumber() + " / " + L.pageCount())
write(str(len(footer) > 0))
code: String := L.qr("https://ahdcode.org")
write(str(code.contains("\\fill[black]")))
write(L.image("logo.svg", {"width": 4.0}, {"opacity": 0.5}))
L.place("x", 1.0, 1.0, "middle")
L.pdf(L.document(footer), "out.pdf")
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v1.3.0")
	for _, want := range []string{`\href{https://ahdcode.org/?a=1\&b=2}{site}`, "\\thepage{}", "true\n"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("REPL output lacks %q:\n%s\nerrors:\n%s", want, output.String(), errors.String())
		}
	}
	for _, want := range []string{"ValueError", "Latex.place anchor must be", "LatexError", "not available in the interactive evaluator"} {
		if !strings.Contains(errors.String(), want) {
			t.Fatalf("REPL errors lack %q:\n%s", want, errors.String())
		}
	}
}
