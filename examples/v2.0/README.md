# Desktop applications (v2.0)

[English] · [Türkçe](README_TR.md)

Three programs for v2.0: a complete [GUI](../../docs/GUI.md) application, a
3D [Plot](../../docs/PLOT.md#surface) Surface, and a small application to
[package](../../docs/PACKAGING.md).

## ledger_app

```bash
cd ledger_app
ahdcode run main.ahd
```

A customer ledger kept in SQLite. Type a customer, choose Debt or Payment in
the Select, type an amount and an optional note in the TextArea, and press
Add entry — it is enabled only while the customer and a positive amount are
filled in, through `onChange`. The TableView lists the entries with a
running balance; the ListBox filters by customer when "Show every customer"
is cleared. Select a row to delete it after a confirmation, and export the
visible rows to CSV through the save dialog. The window is resizable: the
table and the list grow with it. Escape closes it.

The database `ahdcode-ledger.db` is created in the working folder. It is a
generated file; it is not part of the example.

## surface_plot

```bash
cd surface_plot
ahdcode run main.ahd
```

Draws z = sin(x) · cos(y) over a 41 × 41 grid, saves `surface.png` and
`surface-wireframe.png`, and opens the Surface in the Plot viewer: drag to
orbit, Shift+drag to pan, scroll to zoom, R or Reset View to reset, and Save
to write the same PNG again.

## package_hello

```bash
cd package_hello
ahdcode package main.ahd --name "Hello AhdCode"
```

The smallest desktop application: one window with a greeting. The command
makes `dist/Hello AhdCode.app` on macOS (a folder and an archive on Windows
and Linux) that holds the program and the `ahdgui` helper only, and runs
without AhdCode installed. The ledger packages the same way:

```bash
ahdcode package ../ledger_app/main.ahd --name Ledger
```
