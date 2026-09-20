# Plot standard module

[English] · [Türkçe](PLOT_TR.md)

[Back to README](../README.md) · [Statistics](STATISTICS.md) · [Modules](MODULES.md)

If you are learning this module, start with the [Plot workshop](PRACTICAL_MODULES.md#3-plot-turn-data-into-a-readable-chart)
for Data conversion, chart choice, quality checks, and embedding one figure in
Word and Latex; use this page as the full chart reference.

Plot renders charts from typed numeric Lists. It is explicit, like Math,
Time, Regex, CSV, Data, and Statistics:

```ahd
bring Plot
from Plot bring Chart
from Plot bring Figure
from Plot bring Surface
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

> Since v1.9.0. Earlier releases opened the chart with the operating system's
> image viewer.

`show()` opens the chart in AhdCode's own interactive viewer, a window named
**AhdCode Plot** with the AhdCode icon. The chart is drawn exactly as
`save()` would draw it, and the viewer lets you inspect it. From v2.0 a
toolbar along the top of the window shows the controls:

| Toolbar button | Keys and mouse | Action |
| --- | --- | --- |
| Save | — | Save the chart as PNG, SVG, or PDF (see below) |
| Zoom Out / Zoom In | Mouse wheel or trackpad scroll | Zoom out and in (the wheel zooms around the pointer) |
| — | Drag with the left button | Pan |
| Rotate Left / Rotate Right | `Q` / `E` | Turn the view a quarter turn left / right |
| Fit | `R` | Reset: no turn, whole chart fitted, centered |
| — | `Escape` | Close the viewer |

Each button shows its name when the pointer rests on it. The toolbar belongs
to the viewer: it is not a GUI widget and programs cannot add to it.

A small overlay shows the zoom and the turn, and a one-line hint of the
controls appears for the first seconds. The viewer opens at the chart's size
(within the screen) and can be resized. Zoom ranges from a quarter of the
fitted size up to 1600%; a chart smaller than the window stays centered, and
a zoomed chart always covers the window, so it can never be lost — `R`
brings back the whole chart. Turns are quarter turns only (0°, 90°, 180°,
270°).

**The viewer changes only the view.** Zooming, panning, and turning never
change the `Chart` or `Figure`, its data or axes, or a file saved later:
`chart.save("result.png")` after `show()` writes exactly what it would have
written without it. There is no viewer API; `show()` has no parameters and
the viewer has no menus, editing, or data picking.

**Save** (v2.0) asks for a file name in the system's save dialog, offering
PNG, SVG, and PDF; the extension chooses the format. The viewer does not
screenshot its window: it runs the same renderer on the chart's own
definition, so the file is byte for byte what `chart.save(path)` or
`figure.save(path)` writes, however the view is zoomed or turned. Saving
works after the program that called `show()` has ended. The result — or why
it failed — appears beside the toolbar for a few seconds.

> **Fixed after v2.0.0.** In v2.0.0, a program that uses Plot but not GUI
> did not find the helper that opens the save dialog, and Save reported
> "Save needs AhdCode's GUI helper (ahdgui), which is not installed". The
> compiler recorded the helper's location only for a program that used GUI
> itself, so a Plot-only program compiled into a temporary directory had no
> other way to find it. `chart.save(path)` and a packaged application were
> never affected.
>
> It is fixed in the repository and in v2.1: the location is now recorded
> for a program that uses GUI **or** Plot, and Save needs no environment
> variable. If you are running the published v2.0.0 release, name the
> helper when you run the program until you upgrade:
>
> ```sh
> AHDCODE_GUI_RUNTIME=~/Library/AhdCode/current/libexec/ahdcode/ahdgui ahdcode run chart.ahd
> ```

`show()` returns as soon as the viewer window is open. The program continues
and may call `show()` again; each call opens its own viewer, and a viewer
stays open until you close it, even after the program ends. `show()` renders
the chart to a temporary PNG in an AhdCode-specific area of the system
temporary directory; the viewer reads it completely and deletes it before
`show()` returns, so no preview files accumulate.

`show()` raises `PlotError` when the viewer cannot open: the bundled viewer
is missing, no display is available (CI, a container, a remote shell), or
the chart is larger than 6144 units on a side (show the chart smaller, or
`save()` it). AhdCode never falls back to another application.

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

## Surface

> Added in v2.0.0.

```ahd
bring Plot
bring Math
bring Numeric
from Plot bring Surface

x: List<Real> := [-2.0, -1.0, 0.0, 1.0, 2.0]
y: List<Real> := [-1.0, 0.0, 1.0]
rows: List<List<Real>> := []
for yValue in y {
    row: Local List<Real> := []
    for xValue in x {
        row.add(xValue * xValue - yValue * yValue)
    }
    rows.add(row)
}
surface: Surface := Plot.surface(x, y, Numeric.matrix(rows)).title("Saddle").zLabel("height")
surface.save("saddle.png")
surface.show()
```

```text
Plot.surface(x, y, z: Matrix) -> Surface

Surface.title(text: String) -> Surface
Surface.xLabel(text: String) -> Surface
Surface.yLabel(text: String) -> Surface
Surface.zLabel(text: String) -> Surface
Surface.size(width: Int, height: Int) -> Surface
Surface.wireframe(enabled: Bool) -> Surface
Surface.save(path: String) -> Nothing
Surface.show() -> Nothing
```

`Plot.surface` draws the height field `z` over a grid: `x` and `y` are each a
`List<Int>`, a `List<Real>`, or a Numeric `Vector`, and `z` is a
[Numeric](NUMERIC.md) `Matrix` with **one row per `y` value and one column
per `x` value** — `z[j][i]` is the height at `(x[i], y[j])`.

- `x` and `y` each hold 2 to 256 values, strictly increasing; the grid holds
  at most 65,536 points. A mismatched Matrix, too few or too many values, or
  unordered coordinates raise `PlotError`. Every value is finite (NaN and
  infinity never reach an AhdCode program).
- A Surface is immutable, like a Chart: `title`, the labels, `size`, and
  `wireframe` return a new Surface. The axis labels default to `x`, `y`, and
  `z`; the size defaults to 800 × 600 units and is at most 2000 on a side.
- The surface is filled with one fixed height color scale (deep blue through
  green to yellow) and shaded by one fixed light, so its shape reads at a
  glance; `wireframe(true)` draws only its grid lines, colored by height. The
  floor of the box, the axis names, and the range of each axis frame it.
  There is no colormap, lighting, material, or camera setting.
- `save(path)` writes a **PNG only**, at 4/3 pixels per unit (800 × 600 units
  make 1067 × 800 pixels), drawn from the initial view; another extension is
  a `PlotError`. It is deterministic: the same Surface always writes the same
  bytes on the same computer. There is no SVG or PDF for a Surface, rather
  than a vector file that merely wraps a picture.

`show()` opens the Surface in the same viewer, in 3D:

| Toolbar button | Keys and mouse | Action |
| --- | --- | --- |
| Save | — | Save the Surface's PNG, exactly as `save(path)` writes it |
| Zoom Out / Zoom In | Mouse wheel | Zoom out and in |
| — | Drag with the left button | Orbit around the surface |
| — | Shift+drag, or drag with the right button | Pan |
| Reset View | `R` | Back to the initial view |
| — | `Escape` | Close the viewer |

The view is an orthographic projection. It starts from a fixed angle with the
whole surface in view; orbiting turns freely around the vertical axis and
stops just short of straight up and straight down, so the view never flips;
zoom ranges from 0.3× to 6×, and panning keeps the surface within reach.
Like the Chart viewer, the camera belongs to the view only: it is not part of
the Surface, there is no camera API, and saving always writes the initial
view.

This is small scientific 3D plotting, not a 3D engine: there are no meshes,
imported models, textures, lighting or material settings, scene graph, or
other 3D primitives, and no 3D scatter plot in v2.0.

## PlotError

```ahd
bring Plot
from Plot bring PlotError
```

`PlotError` derives directly from `Error`. Plot raises it for every
plot-specific runtime failure: mismatched `x`/`y` lengths, empty chart data,
an invalid bin count, mismatched bar labels/values, mismatched error-bar
data, negative error magnitudes, an unsupported output format, invalid
subplot dimensions, more charts than subplot cells, an invalid Surface grid
or size, a rendering failure, a temporary-file failure, and a viewer-open
failure. A static type mismatch —
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

Plot renders charts with [Gonum](https://gonum.org)'s plotting library, out
of process, through a small bundled renderer helper (`ahdplot`) shipped
alongside the `ahdcode` toolchain. A Surface is drawn by the viewer helper
(`ahdplotview`) with its own small software projection, both on screen and
for `Surface.save`; the viewer's Save button uses the GUI helper (`ahdgui`)
for the save dialog. This keeps the implementation backend an
internal detail: both the persistent evaluator and natively-compiled
programs drive the same helper the same way, so `Plot.*` behaves identically
whether run through the REPL or `ahdcode build`/`ahdcode run`.

## What Plot is not

Plot supports six 2D chart families — line, scatter, bar, histogram, box,
and error bar — and, from v2.0, the 3D Surface. There is no pie, heatmap,
contour, violin, stem, polar, 3D scatter, candlestick, or area chart, and no
arbitrary custom plotter injection — these may be considered in a future
release. There is no numeric
scalar type beyond `Int`/`Real` widening (no `Numeric` type), no general GUI
framework, and no secondary axes.
