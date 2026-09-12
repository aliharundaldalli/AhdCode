package repl

import (
	"bytes"
	"strings"
	"testing"
)

// The persistent REPL builds the same TikZ fragments and document preamble as a
// compiled program; raw triple Strings keep TikZ exact across session lines.
func TestLatexTikZMatchesNativeInPersistentREPL(t *testing.T) {
	input := `bring Latex as L
name: String := "Ali"
drawing: String := L.tikz(
    source: r"""
\draw[thick, ->] (0,0) -- ++(2,1) node[midway, above] {\textbf{Türkçe \emph{çğıöşü}} 50\%};
% a comment {with braces} [and brackets]
\node[draw, fit=(a)(b)] at ($(a)!0.5!(b)$) {\{literal\} {name} 😊};
"""
    libraries: ["fit", "calc"]
)
write(drawing)
write(L.tikz("\\node {name};"))
write(L.document(body: drawing + L.overlay(r"\node at (current page.center) {x};", ["pgfornament"]), landscape: true))
write(L.tikz("x", ["shadows"]))
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v1.2.0")
	drawing := "% AHDCODE_TIKZ calc,fit\n\\begin{tikzpicture}\n\n" +
		"\\draw[thick, ->] (0,0) -- ++(2,1) node[midway, above] {\\textbf{Türkçe \\emph{çğıöşü}} 50\\%};\n" +
		"% a comment {with braces} [and brackets]\n" +
		"\\node[draw, fit=(a)(b)] at ($(a)!0.5!(b)$) {\\{literal\\} {name} 😊};\n" +
		"\\end{tikzpicture}\n"
	for _, want := range []string{
		drawing,
		"% AHDCODE_TIKZ\n\\begin{tikzpicture}\n\\node Ali;\n\\end{tikzpicture}\n",
		"\\geometry{landscape,margin=2.54cm}\n",
		"\\usepackage{tikz}\n\\usetikzlibrary{calc,fit}\n\\usepackage{pgfornament}\n",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("REPL output lacks %q:\n%s\nerrors:\n%s", want, output.String(), errors.String())
		}
	}
	if !strings.Contains(errors.String(), `ValueError: Latex.tikz library "shadows" is not bundled`) {
		t.Fatalf("REPL did not report the unbundled library:\n%s", errors.String())
	}
}
