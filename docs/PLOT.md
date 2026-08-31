# Plot standard module

[English] · [Türkçe](PLOT_TR.md)

[Back to README](../README.md) · [Statistics](STATISTICS.md) · [Modules](MODULES.md)

Plot renders charts from typed numeric Lists. It is explicit, like Math,
Time, Regex, CSV, Data, and Statistics:

```ahd
bring Plot
from Plot bring Chart
from Plot bring Figure
from Plot bring PlotError
```

The canonical identity is `builtin:Plot`; a sibling `Plot.ahd` cannot shadow
it. Every argument is `NonNull`.

Plot does **not** depend on Data. A `Table` cell is a `String`, so a program
converts explicitly before plotting a column — the same discipline
Statistics uses, and for the same reason.

## Chart types

```text
Plot.line(x, y)                              -> Chart
Plot.scatter(x, y)                           -> Chart
Plot.bar(labels: List<String>, values)       -> Chart
Plot.histogram(values, bins: Int)            -> Chart
Plot.box(values)                             -> Chart
Plot.errorBar(x, y, lowerErrors, upperErrors) -> Chart
Plot.new()                                   -> Chart
Plot.subplots(rows: Int, columns: Int, charts: List<Chart>) -> Figure
```

A single chart — line, scatter, bar, histogram, box, or error bar — produces
a `Chart`. A multi-chart composition produces a `Figure` (see
[Subplots](#subplots)).

## Strict numeric input, no String coercion

Every numeric argument accepts `List<Int>` or `List<Real>`, resolved by
ordinary overload resolution; an `Int` List is safely widened to `Real`
internally. `x` and `y` may independently be `List<Int>` or `List<Real>`:

```ahd
x: List<Int> := [1, 2, 3, 4]
y: List<Real> := [2.0, 5.0, 4.0, 8.0]

chart := Plot.line(x, y)
```

A `List<String>` is never accepted, even one holding digit text. This does
not compile:

```ahd
Plot.line(["1", "2", "3"], ["2", "5", "4"])
```

Data integration stays explicit, exactly like Statistics:

```ahd
scores: List<Int> := table.column("score").map(
    lambda (value: String) -> int(value)
)

chart := Plot.histogram(scores, 10)
```

## Empty data

Every chart constructor (`Plot.line`, `Plot.scatter`, `Plot.bar`,
`Plot.histogram`, `Plot.box`, `Plot.errorBar`) and `Chart.line`/`Chart.scatter`
raise `PlotError` for empty numeric input. There is nothing meaningful to
draw, so this is a domain error, the same way `Statistics.mean([])` is:

```ahd
attempt {
    Plot.line(empty, empty)
} except PlotError as error {
    write(error.message)  // "line chart data must not be empty"
}
```

## Chart metadata

```text
chart.title(text: String)   -> Chart
chart.xLabel(text: String)  -> Chart
chart.yLabel(text: String)  -> Chart
chart.legend(enabled: Bool) -> Chart
chart.size(width: Int, height: Int) -> Chart
```

Every Chart method is pure: it returns a **new** Chart and never modifies its
receiver, the same convention [`Table`](DATA.md) uses for every operation.
Configuration therefore chains through reassignment:

```ahd
chart := Plot.line(x, y)
chart = chart.title("Experiment")
chart = chart.xLabel("Time")
chart = chart.yLabel("Value")
```

`size` sets the output dimensions in pixels for PNG, or the equivalent page
size for SVG/PDF; both `width` and `height` must be positive. A Chart's
default size is 800x600.

## Multiple series

`chart.line(x, y, label)` and `chart.scatter(x, y, label)` add one more
series to a Chart, so a line and a scatter series — or several lines, or
several scatter series — can share one Chart with a legend:

```ahd
chart := Plot.new()
chart = chart.line(x, y1, "Experiment")
chart = chart.scatter(x, y2, "Observation")
chart = chart.legend(true)
```

`Plot.line(x, y)` and `Plot.scatter(x, y)` are shorthand for starting a
Chart with one unlabeled series; `chart.line`/`chart.scatter` extend it (or
extend a Chart already built this way). `x` and `y` follow the same
independent `List<Int>`/`List<Real>` rule as every other numeric argument.

Adding a line or scatter series to a `bar`, `histogram`, `box`, or `errorBar`
Chart raises `PlotError`: those chart kinds are self-contained and do not
compose with the series model.

## Save

```text
chart.save(path: String) -> Nothing
figure.save(path: String) -> Nothing
```

The output format is inferred from the file extension. Supported formats are
PNG (`.png`), SVG (`.svg`), and PDF (`.pdf`); anything else raises
`PlotError`:

```ahd
chart.save("result.png")
chart.save("result.svg")
chart.save("result.pdf")

attempt {
    chart.save("result.bmp")
} except PlotError as error {
    write(error.message)
}
```

A relative path resolves against the program's working directory, the same
rule [`File`](FILESYSTEM.md) uses. A rendering or filesystem failure raises
`PlotError`, never a raw Go error.

## Show

```text
chart.show() -> Nothing
figure.show() -> Nothing
```

`show()` renders to a unique temporary PNG and opens it with the platform's
standard image-opening mechanism (`open` on macOS, `xdg-open` on Linux, the
shell's `start` command on Windows), so inspecting a chart never requires
manually saving and locating a file. The temporary image lives in an
AhdCode-specific area under the system temporary directory; it is not
automatically deleted, since the external viewer needs to keep reading it
after `show()` returns.

`show()` requires a desktop session. A headless environment (CI, a container
with no display, no `xdg-open`/no handler registered) fails cleanly with a
`PlotError` rather than hanging — every render/open step runs under a short
timeout.

## Subplots

```ahd
figure := Plot.subplots(
    2, 2,
    [
        Plot.line(x1, y1),
        Plot.scatter(x2, y2),
        Plot.histogram(values, 10),
        Plot.box(values)
    ]
)

figure.show()
figure.save("summary.pdf")
```

`charts` is row-major. `rows` and `columns` must both be positive, and the
chart count must not exceed `rows * columns`; fewer charts than cells is
permitted and leaves the remaining cells blank, rather than requiring an
exact count. A `Figure` is an explicit, immutable value produced by
`Plot.subplots` — there is no mutable global "current subplot" state.

A `Figure`'s save/show size is derived deterministically from its grid
dimensions (a fixed per-cell budget scaled by `rows` and `columns`); v0.1.14
publishes no `Figure.size` method.

## PlotError

```ahd
bring Plot
from Plot bring PlotError
```

`PlotError` derives directly from `Error`. Plot raises it for every
plot-specific runtime failure: mismatched `x`/`y` lengths, empty chart data,
an invalid bin count, mismatched bar labels/values, mismatched error-bar
data, negative error magnitudes, an unsupported output format, invalid
subplot dimensions, more charts than subplot cells, a rendering failure, a
temporary-file failure, and a viewer-open failure. A static type mismatch —
passing a `List<String>` where a numeric List is expected — remains an
ordinary compile-time diagnostic; `PlotError` is reserved for domain and
runtime failures the type checker cannot rule out in advance.

## Input is never modified

Every Plot function and Chart method reads a snapshot of its List arguments;
none reorders or otherwise mutates the caller's List:

```ahd
values: List<Int> := [3, 1, 4, 1, 5]

chart := Plot.histogram(values, 5)
write(values)  // [3, 1, 4, 1, 5]
```

## Rendering

Plot renders with [Gonum](https://gonum.org)'s plotting library, out of
process, through a small bundled renderer helper (`ahdplot`) shipped
alongside the `ahdcode` toolchain. This keeps the implementation backend an
internal detail: both the persistent evaluator and natively-compiled
programs drive the same helper the same way, so `Plot.*` behaves identically
whether run through the REPL or `ahdcode build`/`ahdcode run`.

## What Plot is not

v0.1.14 supports exactly six chart families: line, scatter, bar, histogram,
box, and error bar. There is no pie, heatmap, contour, violin, stem, polar,
3D, candlestick, area, or surface chart, and no arbitrary custom plotter
injection — these may be considered in a future release. There is no numeric
scalar type beyond `Int`/`Real` widening (no `Numeric` type), no general GUI
framework, and no secondary axes.
