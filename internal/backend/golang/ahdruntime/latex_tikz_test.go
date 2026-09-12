package ahdruntime

import (
	"strings"
	"testing"
)

// The v1.2.0 TikZ bridge passes TikZ source through exactly. These tests pin
// the generated fragments and the preamble a document derives from them.

const latexOverlayOpen = "\\AddToHookNext{shipout/foreground}{\\put(0,0){\\begin{tikzpicture}[remember picture,overlay]\n"

func TestLatexTikZPreservesSourceExactly(t *testing.T) {
	// Backslashes, braces, brackets, a percent sign, coordinates, nested
	// commands, calc syntax, a TikZ comment, Unicode text, and several lines.
	source := "\\draw[thick, ->] (0,0) -- ++(2,1) node[midway, above] {\\textbf{Türkçe \\emph{çğıöşü}} 50\\%};\n" +
		"% a TikZ comment {with braces}\n" +
		"\\node[draw, fit=(a)(b)] at ($(a)!0.5!(b)$) {[x] \\{literal\\}};"
	text, problem := AhdLatexTikZText("tikz", source, []string{"calc"}, false)
	if problem != "" {
		t.Fatal(problem)
	}
	want := "% AHDCODE_TIKZ calc\n\\begin{tikzpicture}\n" + source + "\n\\end{tikzpicture}\n"
	if text != want {
		t.Fatalf("tikz fragment\n got %q\nwant %q", text, want)
	}
	overlay, problem := AhdLatexTikZText("overlay", source+"\n", nil, true)
	if problem != "" {
		t.Fatal(problem)
	}
	if want := "% AHDCODE_TIKZ\n" + latexOverlayOpen + source + "\n\\end{tikzpicture}}}%\n"; overlay != want {
		t.Fatalf("overlay fragment\n got %q\nwant %q", overlay, want)
	}
}

func TestLatexTikZLibrariesAreCanonicalAndClosed(t *testing.T) {
	text, problem := AhdLatexTikZText("tikz", "x", []string{"pgfornament", "positioning", "calc", "positioning"}, false)
	if problem != "" {
		t.Fatal(problem)
	}
	if first := strings.SplitN(text, "\n", 2)[0]; first != "% AHDCODE_TIKZ calc,positioning,pgfornament" {
		t.Fatalf("marker = %q", first)
	}
	for _, name := range []string{"shadows", "Calc", "", "calc ", "tikz"} {
		_, problem := AhdLatexTikZText("overlay", "x", []string{name}, true)
		want := "Latex.overlay library \"" + name + "\" is not bundled; supported names are calc, positioning, arrows.meta, shapes.geometric, decorations.pathmorphing, decorations.pathreplacing, patterns, fit, backgrounds, pgfornament"
		if problem != want {
			t.Fatalf("library %q problem\n got %q\nwant %q", name, problem, want)
		}
	}
}

func TestLatexBorderIsAnOrdinaryOverlay(t *testing.T) {
	text, problem := AhdLatexBorderText(1, 1, "")
	if problem != "" {
		t.Fatal(problem)
	}
	want := "% AHDCODE_TIKZ\n" + latexOverlayOpen +
		"\\draw[line width=1.0pt] ([shift={(1.0cm,1.0cm)}]current page.south west) rectangle ([shift={(-1.0cm,-1.0cm)}]current page.north east);\n" +
		"\\end{tikzpicture}}}%\n"
	if text != want {
		t.Fatalf("default border\n got %q\nwant %q", text, want)
	}
	colored, _ := AhdLatexBorderText(1.6, 0.6, "#1f4e79")
	for _, part := range []string{"\\definecolor{ahdborder}{HTML}{1F4E79}\n", "\\draw[line width=0.6pt,draw=ahdborder] ([shift={(1.6cm,1.6cm)}]"} {
		if !strings.Contains(colored, part) {
			t.Fatalf("colored border lacks %q:\n%s", part, colored)
		}
	}
	for _, test := range []struct {
		inset, thickness float64
		color, problem   string
	}{
		{-0.1, 1, "", "Latex.border inset must not be negative"},
		{1, 0, "", "Latex.border thickness must be positive"},
		{1, -2, "", "Latex.border thickness must be positive"},
		{1, 1, "red", "Latex.border color must use #RRGGBB"},
		{1, 1, "#12345G", "Latex.border color must use #RRGGBB"},
	} {
		if _, problem := AhdLatexBorderText(test.inset, test.thickness, test.color); problem != test.problem {
			t.Fatalf("border(%v, %v, %q) problem = %q, want %q", test.inset, test.thickness, test.color, problem, test.problem)
		}
	}
	// A zero inset is a border on the page edge, which is legitimate.
	if _, problem := AhdLatexBorderText(0, 1, ""); problem != "" {
		t.Fatalf("zero inset rejected: %s", problem)
	}
}

func TestLatexDocumentLoadsTikZOnlyForTikZFragments(t *testing.T) {
	empty := AhdBuildPair([]string{}, []string{})
	plain := AhdLatexDocumentFull("Body", "Title", "", "", "Article", 2.54, "", "", empty, "Default", false)
	if strings.Contains(plain, "tikz") || strings.Contains(plain, "landscape") {
		t.Fatalf("a document without TikZ changed:\n%s", plain)
	}
	// Raw TikZ written by hand is the caller's own source; only generated
	// fragments carry the marker that loads the package.
	manual := AhdLatexDocumentFull("\\begin{tikzpicture}\\end{tikzpicture}", "", "", "", "Article", 2.54, "", "", empty, "Default", false)
	if strings.Contains(manual, "\\usepackage{tikz}") {
		t.Fatal("a hand-written tikzpicture must not silently load TikZ")
	}

	first, _ := AhdLatexTikZText("tikz", "\\draw (0,0) -- (1,1);", []string{"positioning"}, false)
	second, _ := AhdLatexTikZText("overlay", "\\node {x};", []string{"pgfornament", "calc"}, true)
	cover, _ := AhdLatexBorderText(1, 1, "")
	source := AhdLatexDocumentFull(first+second, "", "", "", "Report", 2.0, "#1F4E79", cover, empty, "Default", true)
	preamble := source[:strings.Index(source, "\\begin{document}")]
	for _, want := range []string{
		"\\geometry{landscape,margin=2.0cm}\n",
		"\\definecolor{ahdaccent}{HTML}{1F4E79}\n\\usepackage{tikz}\n\\usetikzlibrary{calc,positioning}\n\\usepackage{pgfornament}\n",
	} {
		if !strings.Contains(preamble, want) {
			t.Fatalf("preamble lacks %q:\n%s", want, preamble)
		}
	}
	if strings.Count(source, "\\usepackage{tikz}") != 1 {
		t.Fatal("TikZ must be loaded exactly once")
	}
}

func TestLatexDocumentRejectsInvalidVectorLayouts(t *testing.T) {
	empty := AhdBuildPair([]string{}, []string{})
	requireLatexValueError(t, "Latex.document landscape requires an Article or Report document", func() {
		AhdLatexDocumentFull("Body", "", "", "", "Beamer", 2.54, "", "", empty, "Default", true)
	})
	requireLatexValueError(t, "Latex.document found a TikZ fragment naming an unbundled library \"shadows\"", func() {
		AhdLatexDocumentFull("% AHDCODE_TIKZ calc,shadows\n", "", "", "", "Article", 2.54, "", "", empty, "Default", false)
	})
	requireLatexValueError(t, "Latex.tikz library \"shadows\" is not bundled; supported names are calc, positioning, arrows.meta, shapes.geometric, decorations.pathmorphing, decorations.pathreplacing, patterns, fit, backgrounds, pgfornament", func() {
		AhdLatexTikZ("x", AhdNewList("shadows"))
	})
	requireLatexValueError(t, "Latex.border thickness must be positive", func() {
		AhdLatexBorder(1, 0, "")
	})
}

func requireLatexValueError(t *testing.T, message string, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		signal, ok := recovered.(*AhdSignal)
		if !ok {
			t.Fatalf("expected ValueError %q, got %#v", message, recovered)
		}
		if signal.Message != message {
			t.Fatalf("ValueError message\n got %q\nwant %q", signal.Message, message)
		}
	}()
	fn()
}
