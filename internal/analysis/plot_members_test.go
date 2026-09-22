package analysis

import (
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/semantic"
	"ahdcode/internal/types"
)

// Chart and Figure members reach editors from the compiler's published member
// Symbols, the metadata the compiler checks calls against; there is no
// editor-side list of Plot members.

const plotMemberSource = `bring Plot
bring Numeric
x: List<Int> := [1, 2, 3]
y: List<Real> := [2.0, 4.0, 3.0]
chart := Plot.new()
chart = chart.title("Growth")
chart = chart.xLabel("t")
chart = chart.yLabel("v")
chart = chart.legend(true)
chart = chart.legendPosition(Plot.LegendPosition.topRight)
chart = chart.size(800, 600)
chart = chart.lineStyle(Plot.LineStyle.dashed)
chart = chart.lineWidth(2.0)
chart = chart.marker(Plot.Marker.circle)
chart = chart.markerSize(6.0)
chart = chart.line(x, y, "line")
chart = chart.scatter(x, y, "points")
chart.save("chart.png")
chart.show()
figure := Plot.subplots(1, 1, [chart])
figure.save("figure.png")
figure.show()
`

func TestCompletionOffersEveryChartAndFigureMember(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	for _, testCase := range []struct {
		prefix string
		want   []string
		absent []string
	}{
		{"bring Plot\nchart := Plot.new()\nchart.", semantic.PlotChartOperations, []string{"zLabel", "wireframe"}},
		{"bring Plot\nfigure := Plot.subplots(1, 1, [Plot.new()])\nfigure.", semantic.PlotFigureOperations, []string{"title", "line"}},
	} {
		store.Open(path, testCase.prefix+"\n")
		items := store.Completion(path, len(testCase.prefix))
		if len(testCase.want) == 0 {
			t.Fatal("no published members to check")
		}
		for _, name := range testCase.want {
			if !hasLabel(items, name) {
				t.Fatalf("after %q expected %q, got %#v", testCase.prefix, name, items)
			}
		}
		for _, name := range testCase.absent {
			if hasLabel(items, name) {
				t.Fatalf("after %q unexpected %q", testCase.prefix, name)
			}
		}
	}
}

func TestHoverAndSignatureHelpForEveryChartAndFigureMember(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, plotMemberSource)
	hovers := map[string]string{
		"chart.title":          "title: (text: String) -> Chart",
		"chart.xLabel":         "xLabel: (text: String) -> Chart",
		"chart.yLabel":         "yLabel: (text: String) -> Chart",
		"chart.legend":         "legend: (enabled: Bool) -> Chart",
		"chart.legendPosition": "legendPosition: (position: LegendPosition) -> Chart",
		"chart.size":           "size: (width: Int, height: Int) -> Chart",
		"chart.lineStyle":      "lineStyle: (style: LineStyle) -> Chart",
		"chart.lineWidth":      "lineWidth: (width: Real) -> Chart",
		"chart.marker":         "marker: (shape: Marker) -> Chart",
		"chart.markerSize":     "markerSize: (size: Real) -> Chart",
		"chart.save":           "save: (path: String) -> Nothing",
		"chart.show":           "show: () -> Nothing",
		"figure.save":          "save: (path: String) -> Nothing",
		"figure.show":          "show: () -> Nothing",
		"chart.line":           "line",
		"chart.scatter":        "scatter",
	}
	checked := 0
	for _, pair := range []struct {
		receiver string
		names    []string
	}{{"chart", semantic.PlotChartOperations}, {"figure", semantic.PlotFigureOperations}} {
		for _, name := range pair.names {
			target := pair.receiver + "." + name
			want, ok := hovers[target]
			if !ok {
				t.Fatalf("no expectation for published member %s", target)
			}
			hover, found := store.Hover(path, offsetOf(t, plotMemberSource, target+"(")+len(pair.receiver)+2)
			if !found || !strings.HasPrefix(hover.Text, want) {
				t.Fatalf("hover on %s = %q (%v), want %q", target, hover.Text, found, want)
			}
			help, found := store.SignatureHelp(path, offsetOf(t, plotMemberSource, target+"(")+len(target)+1)
			if !found || help.Label == "" {
				t.Fatalf("signature help for %s = %#v (%v)", target, help, found)
			}
			if strings.Contains(want, ":") && !strings.HasSuffix(want, help.Label) {
				t.Fatalf("signature help for %s = %q, want the tail of %q", target, help.Label, want)
			}
			checked++
		}
	}
	if checked != len(semantic.PlotChartOperations)+len(semantic.PlotFigureOperations) {
		t.Fatalf("checked %d members", checked)
	}
	// The series members select the overload matching their arguments.
	help, _ := store.SignatureHelp(path, offsetOf(t, plotMemberSource, "chart.line(")+len("chart.line("))
	if help.Label != "(x: List<Int>, y: List<Real>, label: String) -> Chart" {
		t.Fatalf("chart.line signature help = %q", help.Label)
	}
}

// TestEditorMembersMatchCompilerMembers keeps the editor aligned with the
// compiler: every name a built-in Class publishes for completion resolves to
// a checked member Symbol with a signature.
func TestEditorMembersMatchCompilerMembers(t *testing.T) {
	for class, names := range map[string][]string{"Chart": semantic.PlotChartOperations, "Figure": semantic.PlotFigureOperations} {
		for _, name := range names {
			if !semantic.TypeOperationBindsArguments(semantic.TypeOperation(class + "." + name)) {
				t.Fatalf("%s.%s has no published Symbol", class, name)
			}
		}
		members := semantic.BuiltinClassMembers(map[string]*types.ClassSymbol{"Chart": semantic.PlotChartIdentity(), "Figure": semantic.PlotFigureIdentity()}[class])
		if len(members) != len(names) {
			t.Fatalf("%s publishes %d members for %d names", class, len(members), len(names))
		}
		for index, member := range members {
			if member == nil || member.Name != names[index] || member.Callable == nil {
				t.Fatalf("%s member %d = %#v", class, index, member)
			}
		}
	}
}
