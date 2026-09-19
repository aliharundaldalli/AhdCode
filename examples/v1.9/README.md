# GUI colors and the interactive Plot viewer (v1.9)

[English] · [Türkçe](README_TR.md)

Two small programs for v1.9: [GUI](../../docs/GUI.md) colors and enabled
state, and AhdCode's own [Plot](../../docs/PLOT.md#show) viewer.

## gui_colors

```bash
cd gui_colors
ahdcode run main.ahd
```

An order form with explicit colors: a colored window and header, a tinted
TextInput, a colored Checkbox caption, and a green Save button. Save starts
disabled; press Check, and Save is enabled only when a customer name is typed
and the terms are accepted. Saving writes one line and disables Save again.
Escape closes the window. Colors use the Graphics spellings: nine lower-case
names, `#RRGGBB`, or `#RRGGBBAA`.

This is an ordinary small form, not a styling showcase: there are no themes,
fonts, or custom widgets.

## interactive_plot

```bash
cd interactive_plot
ahdcode run main.ahd
```

A line and a scatter series with a title, axis labels, and a legend, shown in
AhdCode's own viewer:

- scroll (or use the trackpad) to zoom around the pointer,
- drag with the left button to pan,
- press Q or E to turn the view a quarter turn left or right,
- press R to reset, and
- press Escape to close.

The program continues as soon as the viewer is open and then saves
`result.png` — the chart exactly as defined, whatever you did in the viewer:
the viewer changes only the view. `result.png` is not part of the repository.
