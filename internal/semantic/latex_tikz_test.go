package semantic

import (
	"strings"
	"testing"

	"ahdcode/internal/types"
)

func TestLatexVectorHelpersAreRegistered(t *testing.T) {
	module := StandardModuleInterfaces()["Latex"]
	for _, name := range []string{"tikz", "overlay", "border"} {
		if module.Exports[name] == nil {
			t.Fatalf("Latex module missing export %q", name)
		}
	}
	// TikZ stays TikZ: there is no drawing API mirroring its commands.
	for _, name := range []string{"line", "rectangle", "circle", "path", "node", "certificate", "svg", "canvas", "landscapePage"} {
		if module.Exports[name] != nil {
			t.Fatalf("Latex module must not export %q", name)
		}
	}
	document := module.Exports["document"].Callable.Signature
	// landscape keeps its v1.2.0 position; v1.3.0 appends page layout and PDF
	// properties after it, so existing positional calls keep their meaning.
	var names []string
	for _, parameter := range document.Parameters {
		names = append(names, parameter.Name)
	}
	want := []string{"body", "title", "author", "date", "type", "margin", "color", "cover", "theorems", "theme", "landscape",
		"paper", "pageSize", "margins", "subject", "keywords", "creator"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("document parameters = %v, want %v", names, want)
	}
	for _, parameter := range document.Parameters[1:] {
		if !parameter.HasDefault {
			t.Fatalf("document parameter %s must be optional", parameter.Name)
		}
	}
	if landscape := document.Parameters[10]; landscape.Type != types.Bool {
		t.Fatalf("landscape = %+v, want an optional Bool", landscape)
	}
}

func TestLatexVectorHelpersValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, `bring Latex as L

drawing: String := L.tikz(r"\draw (0,0) rectangle (4,2);")
labelled: String := L.tikz(
    source: r"\node[draw, right=of a] {b};"
    libraries: ["positioning", "calc"]
)
frame: String := L.overlay(r"\draw (current page.north west) -- (current page.south east);", ["pgfornament"])
edge: String := L.border()
inner: String := L.border(inset: 1.6, thickness: 0.6, color: "#1F4E79")
source: String := L.document(body: edge + inner + frame + drawing + labelled, landscape: true)
portrait: String := L.document(drawing, "Title", "Author")
`)
	requireSemanticClean(t, result)
}

func TestLatexVectorHelpersRejectWrongStaticTypes(t *testing.T) {
	tests := []struct {
		source string
		code   string
	}{
		{`L.tikz(42)`, codeTypeMismatch},
		{`L.tikz("x", "calc")`, codeTypeMismatch},
		{`L.tikz("x", [1, 2])`, codeTypeMismatch},
		{`L.overlay(true)`, codeTypeMismatch},
		{`L.border("1cm")`, codeTypeMismatch},
		{`L.border(1.0, 1.0, 5)`, codeTypeMismatch},
		{`L.document(body: "x", landscape: "yes")`, codeTypeMismatch},
		{`L.tikz()`, codeCallArguments},
		{`L.overlay("x", [], "extra")`, codeCallArguments},
		{`L.border(1.0, 1.0, "", "extra")`, codeCallArguments},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, "bring Latex as L\nvalue := "+test.source+"\n")
			requireSemanticFailure(t, result)
			if result.Diagnostics[0].Code != test.code {
				t.Fatalf("%s: first diagnostic %s %q, want %s", test.source,
					result.Diagnostics[0].Code, result.Diagnostics[0].Message, test.code)
			}
		})
	}
}

// A library name is data the runtime validates against the bundled set, so an
// unknown name compiles and raises ValueError when the fragment is built.
func TestLatexTikZLibraryNamesAreRuntimeValidated(t *testing.T) {
	result := analyzeWithStandardModules(t, "bring Latex as L\nvalue := L.tikz(\"x\", [\"shadows\"])\n")
	requireSemanticClean(t, result)
}
