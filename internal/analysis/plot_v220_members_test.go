package analysis

import (
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/semantic"
)

// The v2.2 Plot surface reaches editors from the compiler's own published
// member Symbols, the same metadata the compiler checks calls against.
// There is no editor-side list of Plot members, and v2.0's Chart/Figure
// metadata bug is the reason this is checked rather than assumed.

func TestCompletionOffersThePlotV220Surface(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()

	// Plot. offers the two new entry points beside the old ones.
	prefix := "bring Plot\nx := Plot."
	store.Open(path, prefix+"\n")
	items := store.Completion(path, len(prefix))
	for _, name := range []string{"pie", "heatmap", "line", "scatter", "bar", "histogram", "box", "errorBar", "surface", "subplots", "new"} {
		if !hasLabel(items, name) {
			t.Fatalf("Plot. offers no %q; got %#v", name, items)
		}
	}
	for _, absent := range []string{"donut", "contour", "violin", "polar"} {
		if hasLabel(items, absent) {
			t.Fatalf("Plot. unexpectedly offers %q", absent)
		}
	}

	// A Surface offers its two new members and keeps the v2.0 ones.
	surfacePrefix := `bring Plot
bring Numeric
from Plot bring Surface
from Numeric bring Matrix
grades: Matrix := Numeric.matrix([[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]])
surface: Surface := Plot.surface([1, 2, 3], [1, 2], grades)
surface.`
	store.Open(path, surfacePrefix+"\n")
	items = store.Completion(path, len(surfacePrefix))
	for _, name := range semantic.PlotSurfaceOperations {
		if !hasLabel(items, name) {
			t.Fatalf("Surface. offers no %q; got %#v", name, items)
		}
	}
	// A Surface is not a Chart: it has no legend and no series.
	for _, absent := range []string{"legend", "line", "scatter"} {
		if hasLabel(items, absent) {
			t.Fatalf("Surface. unexpectedly offers %q", absent)
		}
	}
}

func TestHoverAndSignatureHelpForThePlotV220Members(t *testing.T) {
	source := `bring Plot
bring Numeric
from Plot bring (Chart, Surface)
from Numeric bring Matrix

courses: List<String> := ["Analysis", "Algebra"]
years: List<String> := ["Year 1", "Year 2", "Year 3"]
grades: Matrix := Numeric.matrix([[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]])
surface: Surface := Plot.surface([1, 2, 3], [1, 2], grades)
named: Surface := surface.xCategories(years)
again: Surface := named.yCategories(courses)
write(str(len(courses)) + str(len(years)))
`
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, source)
	for needle, wanted := range map[string]string{
		"surface.xCategories(": "xCategories: (labels: List<String>) -> Surface",
		"named.yCategories(":   "yCategories: (labels: List<String>) -> Surface",
	} {
		receiver := needle[:strings.Index(needle, ".")]
		offset := offsetOf(t, source, needle)
		hover, found := store.Hover(path, offset+len(receiver)+2)
		if !found || hover.Text != wanted {
			t.Fatalf("hover on %s = %q (%v), want %q", needle, hover.Text, found, wanted)
		}
		help, found := store.SignatureHelp(path, offset+len(needle))
		if !found || help.Label == "" {
			t.Fatalf("no signature help for %s", needle)
		}
		if !strings.HasSuffix(wanted, help.Label) {
			t.Fatalf("signature help for %s = %q, want the tail of %q", needle, help.Label, wanted)
		}
	}
}

// Every name a Surface publishes resolves to a member Symbol carrying the
// signature the compiler checks calls against -- including the two v2.2
// added, and every v2.0 one beside them.
func TestPlotV220EditorMembersMatchCompilerMembers(t *testing.T) {
	names := semantic.PlotSurfaceOperations
	for _, name := range names {
		if !semantic.TypeOperationBindsArguments(semantic.TypeOperation("Surface." + name)) {
			t.Fatalf("Surface.%s has no published Symbol", name)
		}
	}
	members := semantic.BuiltinClassMembers(semantic.PlotSurfaceIdentity())
	if len(members) != len(names) {
		t.Fatalf("Surface publishes %d members for %d names", len(members), len(names))
	}
	for index, member := range members {
		if member == nil || member.Name != names[index] || member.Callable == nil || member.Callable.Signature == nil {
			t.Fatalf("Surface member %d = %#v", index, member)
		}
	}
	// The v2.0 Chart and Figure metadata is still there.
	for class, operations := range map[string][]string{
		"Chart":  semantic.PlotChartOperations,
		"Figure": semantic.PlotFigureOperations,
	} {
		for _, name := range operations {
			if !semantic.TypeOperationBindsArguments(semantic.TypeOperation(class + "." + name)) {
				t.Fatalf("%s.%s lost its published Symbol", class, name)
			}
		}
	}
}
