# GUI foundations and window events (v1.8)

[English] · [Türkçe](README_TR.md)

Two small programs for the v1.8 [GUI](../../docs/GUI.md) module and the new
[Graphics](../../docs/GRAPHICS.md) Canvas events.

## simple_ledger

```bash
cd simple_ledger
ahdcode run main.ahd
```

A form with a customer name, an amount, a Paid checkbox, and a Save button.
The Save callback validates the input with ordinary AhdCode (`trim`, `real`),
stores the entry with the existing [SQLite](../../docs/SQLITE.md) module in
`ledger.db` in the current directory, and shows the result in a Label. GUI
knows nothing about databases: the callback simply calls SQLite. Escape
closes the window. `ledger.db` is created on first run and is not part of the
repository.

This is not an accounting program: there is no table view, search, or edit.

## turtle_events

```bash
cd turtle_events
ahdcode run main.ahd
```

The arrow keys turn the Turtle and draw one 20-unit segment per press; a
click moves the Turtle to that point (Cartesian coordinates, origin in the
center, +y up) without drawing; S saves `turtle-events.png` and
`turtle-events.svg`. It reads no terminal input: the Canvas window reports
the keys and clicks through `Canvas.onKey` and `Canvas.onClick`.
