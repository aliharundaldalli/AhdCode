# Math labels in Plot and Surface (v2.3.0)

Plot keeps its existing APIs. A complete String enclosed by `$...$` is sent
through Gonum's internal math-text path with the project's offline fallback;
any other String remains normal text. The example covers chart titles and axes, a pie legend label, a series
legend, and Surface labels/categories.

The example writes PNG/SVG chart files and a PNG Surface. It has no network or
GUI requirement for the chart saves.
