package semantic

import "testing"

func TestPlotV240StylesAreNominalAndChainable(t *testing.T) {
	result := analyzeWithStandardModules(t, `bring Plot
	from Plot bring (Chart, LineStyle, Marker, LegendPosition)
x: List<Real> := [1.0, 2.0, 3.0]
y: List<Real> := [3.0, 2.0, 1.0]
chart: Chart := Plot.line(x, y)
chart = chart.lineStyle(LineStyle.dashed).lineWidth(2.0).marker(Marker.diamond).markerSize(7.0)
	chart = chart.lineStyle(LineStyle.dashDot).marker(Marker.none)
	chart = chart.legendPosition(LegendPosition.bottomLeft)
`)
	requireSemanticClean(t, result)
}

func TestPlotV240StylesRejectStrings(t *testing.T) {
	for _, source := range []string{
		`chart: Chart := Plot.line([1.0], [2.0]).lineStyle("dashed")`,
		`chart: Chart := Plot.line([1.0], [2.0]).marker("circle")`,
		`chart: Chart := Plot.line([1.0], [2.0]).legendPosition("topLeft")`,
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, "bring Plot\nfrom Plot bring Chart\n"+source+"\n"))
	}
}

func TestPlotV240StyleMetadataPublishesNominalTypes(t *testing.T) {
	module := StandardModuleInterfaces()["Plot"]
	for _, name := range []string{"LineStyle", "Marker", "LegendPosition"} {
		if module.Exports[name] == nil {
			t.Fatalf("Plot must export %q", name)
		}
	}
	for _, name := range []string{"lineStyle", "lineWidth", "marker", "markerSize", "legendPosition"} {
		found := false
		for _, operation := range PlotChartOperations {
			if operation == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Chart metadata is missing %q", name)
		}
	}
}
