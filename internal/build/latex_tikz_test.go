package build

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// latexTikZProgram carries TikZ through a raw triple String: backslashes,
// braces, brackets, a percent sign, coordinates, calc syntax, nested commands,
// an escaped brace pair, an interpolation-looking {name}, and Unicode text.
const latexTikZProgram = `bring Latex as L

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
page: String := L.document(body: drawing + L.overlay(r"\node at (current page.center) {x};", ["pgfornament"]), landscape: true)
write(page)
`

const latexTikZDrawing = `% AHDCODE_TIKZ calc,fit
\begin{tikzpicture}

\draw[thick, ->] (0,0) -- ++(2,1) node[midway, above] {\textbf{Türkçe \emph{çğıöşü}} 50\%};
% a comment {with braces} [and brackets]
\node[draw, fit=(a)(b)] at ($(a)!0.5!(b)$) {\{literal\} {name} 😊};
\end{tikzpicture}
`

// A raw String keeps TikZ exactly; a normal String interpolates {name}. Both
// behaviors are the existing String contract, carried unchanged into Latex.
func TestLatexTikZRawSourceSurvivesNativeCompilation(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": latexTikZProgram})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("TikZ program failed: code=%d stderr=%q", code, stderr)
	}
	if !strings.HasPrefix(stdout, latexTikZDrawing+"\n") {
		t.Fatalf("raw TikZ was not preserved exactly:\n%s", stdout)
	}
	for _, want := range []string{
		"% AHDCODE_TIKZ\n\\begin{tikzpicture}\n\\node Ali;\n\\end{tikzpicture}\n",
		"\\geometry{landscape,margin=2.54cm}\n",
		"\\usepackage{tikz}\n\\usetikzlibrary{calc,fit}\n\\usepackage{pgfornament}\n",
		"\\begin{document}\n" + latexTikZDrawing,
		"\\AddToHookNext{shipout/foreground}{\\put(0,0){\\begin{tikzpicture}[remember picture,overlay]\n\\node at (current page.center) {x};\n\\end{tikzpicture}}}%\n",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("output lacks %q:\n%s", want, stdout)
		}
	}
}

// Compiles real TikZ, libraries, pgfornament ornaments, a border, and a
// landscape page with the staged offline runtime, then keeps the exact source.
func TestLatexTikZCertificateCompilesWithStagedOfflineBundle(t *testing.T) {
	root := os.Getenv("AHDCODE_LATEX_TEST_RUNTIME")
	if root == "" {
		t.Skip("set AHDCODE_LATEX_TEST_RUNTIME to a staged Tectonic + ahdcode-latex.ttb directory")
	}
	t.Setenv("AHDCODE_LATEX_RUNTIME", root)
	directory := t.TempDir()
	output := filepath.Join(directory, "certificate.pdf")
	source := `bring Latex as L

frame: String := L.border(inset: 0.8, thickness: 2.4, color: "#1F4E79")
frame += L.border(inset: 1.25, thickness: 0.6)
corner: String := L.overlay(
    source: r"""
\node[anchor=north west] at ([shift={(1.5cm,-1.5cm)}]current page.north west) {\pgfornament[width=3cm]{63}};
\node[opacity=0.08, scale=8] at (current page.center) {AhdCode};
"""
    libraries: ["pgfornament"]
)
seal: String := L.tikz(
    source: r"""
\node[draw, regular polygon, regular polygon sides=6, minimum size=2cm] (h) {Seal};
\node[draw, star, star points=12, right=of h] (s) {\textbf{ş}};
\draw[-{Stealth[length=3mm]}, decorate, decoration={snake}] (h) -- (s);
\draw[decorate, decoration={brace, amplitude=6pt}] (h.north) -- (s.north);
\node[draw, fit=(h)(s), inner sep=6pt] {};
\begin{scope}[on background layer]
\fill[pattern=north east lines] (h.south west) rectangle (s.north east);
\end{scope}
\coordinate (m) at ($(h)!0.5!(s)$);
"""
    libraries: ["calc", "positioning", "arrows.meta", "shapes.geometric", "decorations.pathmorphing", "decorations.pathreplacing", "patterns", "fit", "backgrounds"]
)
body: String := r"\thispagestyle{empty}" + "\n" + frame + corner + L.center(r"{\Huge Certificate}\par" + "\n" + seal)
L.pdf(L.document(body: body, landscape: true), ` + strconv.Quote(output) + `, "tex")
write("ok")
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stdout != "ok\n" {
		t.Fatalf("certificate did not compile: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	pdf, err := os.ReadFile(output)
	if err != nil || !strings.HasPrefix(string(pdf), "%PDF-") {
		t.Fatalf("no valid PDF was produced: %v", err)
	}
	tex, err := os.ReadFile(strings.TrimSuffix(output, ".pdf") + ".tex")
	if err != nil || !strings.Contains(string(tex), `\pgfornament[width=3cm]{63}`) {
		t.Fatalf("the TeX sidecar does not hold the exact TikZ source: %v", err)
	}
}
