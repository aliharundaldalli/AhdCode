package semantic

import (
	"strings"
	"testing"
)

// v2.2 closes three visualization gaps: a pie, a heatmap, and names for a
// Surface's x and y coordinates. All three are published through the same
// standard-module metadata every other Plot entry point uses.

const plotV220Preamble = `bring Plot
bring Numeric
from Plot bring (Chart, Surface, PlotError)
from Numeric bring Matrix

courses: List<String> := ["Analysis", "Algebra", "Geometry", "Statistics", "Programming"]
years: List<String> := ["Year 1", "Year 2", "Year 3"]
grades: Matrix := Numeric.matrix([
    [72.0, 78.0, 84.0]
    [68.0, 75.0, 81.0]
    [80.0, 82.0, 88.0]
    [65.0, 74.0, 83.0]
    [85.0, 91.0, 95.0]
])

`

func TestPlotV220ValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, plotV220Preamble+`// A pie takes Int or Real values, like every other Plot function.
whole: Chart := Plot.pie(courses, [84, 81, 88, 83, 95])
fractional: Chart := Plot.pie(courses, [84.5, 81.0, 88.25, 83.0, 95.75])
whole = whole.title("Year 3").legend(true).size(700, 500)

heat: Chart := Plot.heatmap(years, courses, grades)
heat = heat.title("Student performance").xLabel("Academic year").yLabel("Course").legend(false)

surface: Surface := Plot.surface([1, 2, 3], [1, 2, 3, 4, 5], grades)
surface = surface.xCategories(years).yCategories(courses)
surface = surface.xLabel("Academic year").yLabel("Course").zLabel("Grade")

attempt {
    whole.save("pie.png")
    heat.save("heat.png")
    surface.save("surface.png")
    whole.show()
    heat.show()
    surface.show()
}
except PlotError as error {
    write(error.message)
}
fractional.save("fractional.png")
`)
	requireSemanticClean(t, result)
}

// The static type mistakes a program can make with the v2.2 surface.
func TestPlotV220RejectsStaticMistakes(t *testing.T) {
	for _, source := range []string{
		// Numbers written as text are not numbers.
		`bad: Chart := Plot.pie(["A"], ["10"])`,
		`bad: Chart := Plot.pie([1], [10])`,
		// A heatmap needs a real Matrix, not a nested List.
		`bad: Chart := Plot.heatmap(years, courses, [[90.0]])`,
		`bad: Chart := Plot.heatmap(years, courses, [90.0])`,
		`bad: Chart := Plot.heatmap(years, courses, grades, "extra")`,
		`bad: Chart := Plot.heatmap(years, grades, courses)`,
		// Wrong arity.
		`bad: Chart := Plot.pie(courses)`,
		`bad: Chart := Plot.pie(courses, [1, 2], [3])`,
		`bad: Chart := Plot.heatmap(years, courses)`,
		// Categories are text, one List, on a Surface.
		`surface: Surface := Plot.surface([1, 2, 3], [1, 2, 3, 4, 5], grades)
bad: Surface := surface.xCategories([1, 2, 3])`,
		`surface: Surface := Plot.surface([1, 2, 3], [1, 2, 3, 4, 5], grades)
bad: Surface := surface.yCategories("Analysis")`,
		`surface: Surface := Plot.surface([1, 2, 3], [1, 2, 3, 4, 5], grades)
bad: Surface := surface.xCategories()`,
		// A Chart is not a Surface and has no categories.
		`chart: Chart := Plot.pie(courses, [1, 2, 3, 4, 5])
bad: Chart := chart.xCategories(years)`,
		// A Surface is not a Chart.
		`surface: Surface := Plot.surface([1, 2, 3], [1, 2, 3, 4, 5], grades)
bad: Chart := surface`,
		`bad: Surface := Plot.pie(courses, [1, 2, 3, 4, 5])`,
		// Members AhdCode does not have.
		`chart: Chart := Plot.pie(courses, [1, 2, 3, 4, 5])
bad: Chart := chart.explode(true)`,
		`chart: Chart := Plot.heatmap(years, courses, grades)
bad: Chart := chart.colormap("viridis")`,
		`chart: Chart := Plot.heatmap(years, courses, grades)
bad: Chart := chart.annotate(true)`,
		`bad: Chart := Plot.donut(courses, [1, 2, 3, 4, 5])`,
		`bad: Chart := Plot.contour(years, courses, grades)`,
	} {
		program := source
		result := analyzeWithStandardModules(t, plotV220Preamble+program+"\n")
		requireSemanticFailure(t, result)
	}
}

// A Chart is still produced only by the Plot functions, and the hint now
// names the two new ones.
func TestPlotV220ConstructionHintNamesPieAndHeatmap(t *testing.T) {
	result := analyzeWithStandardModules(t, plotV220Preamble+"bad: Chart := Chart()\n")
	requireSemanticFailure(t, result)
	hints := semanticHintsOf(result)
	for _, wanted := range []string{"Plot.pie", "Plot.heatmap"} {
		if !strings.Contains(hints, wanted) {
			t.Fatalf("the Chart construction hint does not name %s: %q", wanted, hints)
		}
	}
}

// The v2.2 surface is exactly this and no more.
func TestPlotV220SurfaceIsExactlyTheIntendedOne(t *testing.T) {
	if got := strings.Join(PlotSurfaceOperations, ","); got != "title,xLabel,yLabel,zLabel,size,wireframe,xCategories,yCategories,show,save" {
		t.Fatalf("Surface operations = %s", got)
	}
	// Chart gained no member: a pie and a heatmap are Charts, configured by
	// the members a Chart already had.
	if got := strings.Join(PlotChartOperations, ","); got != "title,xLabel,yLabel,legend,size,line,scatter,save,show" {
		t.Fatalf("Chart operations = %s", got)
	}
	module := StandardModuleInterfaces()["Plot"]
	for _, name := range []string{"pie", "heatmap"} {
		if module.Exports[name] == nil {
			t.Fatalf("Plot must export %q", name)
		}
	}
	// None of the things v2.2 deliberately left out.
	for _, name := range []string{
		"donut", "contour", "violin", "polar", "area", "stem", "candlestick",
		"colormap", "palette", "theme", "axis",
	} {
		if module.Exports[name] != nil {
			t.Fatalf("Plot must not export %q", name)
		}
	}
}
