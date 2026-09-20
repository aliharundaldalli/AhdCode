# Student Performance Explorer (v2.2)

[English] · [Türkçe](README_TR.md)

> This example accompanies the AhdCode v2.2.0 release.

One desktop application that collects a student's grades and shows the same
fifteen numbers seven different ways. It is the program the v2.2 Plot
additions were designed against: the pie, the heatmap, and the named Surface
axes all answer questions the earlier charts could not.

```bash
cd student_performance
ahdcode run main.ahd
```

## What it does

[GUI](../../../docs/GUI.md) collects the data: a student name and a 5 × 3
grid of grades — Analysis, Algebra, Geometry, Statistics and Programming,
over three academic years. Every field is checked as it is typed: a grade
must be a number from 0 to 100, and until the name and all fifteen grades
are valid the chart buttons stay disabled. The summary `TableView` and the
yearly averages update on every keystroke.

[Numeric](../../../docs/NUMERIC.md) holds the grid as a `Matrix`, and
[Plot](../../../docs/PLOT.md) draws it:

| Button | Chart | Question it answers |
|---|---|---|
| Open selected chart | line, scatter or bar | how did one course change over three years? |
| Open histogram | histogram | how are all fifteen grades distributed? |
| Open box plot | box | what is the spread — median, quartiles, extremes? |
| Open yearly error-bar chart | error bar | how consistent was each year? |
| **Open selected-year pie chart** | **pie** | **how did one year break down by course?** |
| **Open performance heatmap** | **heatmap** | **where are the strong and weak cells, at a glance?** |
| Open performance surface | surface | what does the whole grid look like as a shape? |

The interface is bilingual: the language Select rewrites every label, button
and table heading, and the charts are titled in the chosen language too.

## What v2.2 adds here

**`Plot.pie`** shows one year's five courses as slices. The Select beside it
chooses the year. A pie has no axes, so the program sets no `xLabel` or
`yLabel` on it — its legend, which is on by default, names the courses:

```ahd
chart: Local Chart := Plot.pie(courseNames(), yearGrades).title(pTitle)
```

**`Plot.heatmap`** shows all fifteen grades at once, course by year. The
`Matrix` is row per y label and column per x label — five courses down,
three years across — and the colour scale replaces reading the numbers:

```ahd
chart: Local Chart := Plot.heatmap(yearNames(), courseNames(), Numeric.matrix(gradeMatrix()))
chart = chart.title(hTitle).xLabel(xLabelText).yLabel(yLabelText)
```

**`Surface.xCategories` and `Surface.yCategories`** name the surface's
coordinates. Before v2.2 the same surface was drawn over `1..3` and `1..5`
and its title had to explain what the numbers meant; now the axes say
"Year 1" and "Geometry" themselves. The geometry is unchanged — the labels
only replace what is shown:

```ahd
surface: Local Surface := Plot.surface(x, y, Numeric.matrix(matrixRows))
surface = surface.xCategories(yearNames()).yCategories(courseNames())
surface = surface.title(sTitle).xLabel(xLabelText).yLabel(yLabelText).zLabel(zLabelText)
```

`xLabel` and `yLabel` still name the axes; `xCategories` and `yCategories`
name the points along them.

## Try it

Type a name and this grid:

| Course | Year 1 | Year 2 | Year 3 |
|---|---|---|---|
| Analysis | 72 | 78 | 84 |
| Algebra | 68 | 75 | 81 |
| Geometry | 80 | 82 | 88 |
| Statistics | 65 | 74 | 83 |
| Programming | 85 | 91 | 95 |

Then open the heatmap and the surface side by side. Each chart opens in its
own [Plot viewer](../../../docs/PLOT.md#the-viewer) window: drag, zoom, turn
and Save, while the application stays usable behind them.

## Notes

Leaving a field empty, or typing a negative number, a number above 100, or
something that is not a number at all, is reported in the status line and
disables the chart buttons until it is fixed. Nothing is written to disk
unless you use a viewer's Save.
