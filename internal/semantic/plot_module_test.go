package semantic

import "testing"

const plotPreamble = "bring Plot\nfrom Plot bring Chart\nfrom Plot bring Figure\nfrom Plot bring PlotError\n\n"

func TestPlotModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, plotPreamble+`x: List<Int> := [1, 2, 3]
y: List<Real> := [1.0, 4.0, 9.0]

chart: Chart := Plot.line(x, y)
chart = chart.title("Title").xLabel("X").yLabel("Y").legend(true).size(640, 480)
chart = chart.scatter(x, y, "second series")

bar: Chart := Plot.bar(["a", "b"], [1, 2])
histogram: Chart := Plot.histogram(x, 5)
box: Chart := Plot.box(y)
errorBar: Chart := Plot.errorBar(x, y, x, y)

figure: Figure := Plot.subplots(2, 2, [chart, bar, histogram, box])

attempt {
    chart.save("out.png")
    chart.show()
    figure.save("out.png")
    figure.show()
}
except PlotError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

func TestPlotLineAcceptsIndependentIntOrRealLists(t *testing.T) {
	result := analyzeWithStandardModules(t, plotPreamble+`ints: List<Int> := [1, 2, 3]
reals: List<Real> := [1.0, 2.0, 3.0]
a: Chart := Plot.line(ints, reals)
b: Chart := Plot.line(reals, ints)
c: Chart := Plot.line(ints, ints)
d: Chart := Plot.line(reals, reals)
`)
	requireSemanticClean(t, result)
}

func TestPlotRejectsStringListData(t *testing.T) {
	result := analyzeWithStandardModules(t, plotPreamble+`values: List<String> := ["1", "2", "3"]
chart: Chart := Plot.line(values, values)
`)
	requireSemanticCode(t, result, codeNoMatchingOverload)
}

func TestPlotChartSeriesOperationAcceptsIntOrRealLists(t *testing.T) {
	result := analyzeWithStandardModules(t, plotPreamble+`ints: List<Int> := [1, 2, 3]
reals: List<Real> := [1.0, 2.0, 3.0]
chart: Chart := Plot.new()
chart = chart.line(ints, reals, "a")
chart = chart.scatter(reals, ints, "b")
`)
	requireSemanticClean(t, result)
}

func TestPlotChartSeriesOperationRejectsStringList(t *testing.T) {
	result := analyzeWithStandardModules(t, plotPreamble+`values: List<String> := ["1", "2"]
chart: Chart := Plot.new()
chart = chart.line(values, values, "a")
`)
	requireSemanticCode(t, result, codeTypeMismatch)
}

func TestPlotHistogramBinsMustBeInt(t *testing.T) {
	result := analyzeWithStandardModules(t, plotPreamble+`values: List<Real> := [1.0, 2.0]
chart: Chart := Plot.histogram(values, 3.5)
`)
	requireSemanticCode(t, result, codeNoMatchingOverload)
}

func TestPlotChartValuesAreNotConstructedDirectly(t *testing.T) {
	result := analyzeWithStandardModules(t, plotPreamble+`chart: Chart := Chart()
`)
	requireSemanticCode(t, result, codeCallArguments)
}

func TestPlotErrorInheritsError(t *testing.T) {
	result := analyzeWithStandardModules(t, plotPreamble+`attempt {
    write("noop")
}
except PlotError as error {
    write(error.message)
}
except Error as generic {
    write(generic.message)
}
`)
	requireSemanticClean(t, result)
}
