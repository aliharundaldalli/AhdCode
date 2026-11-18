# GUI standard module

[English] · [Türkçe](GUI_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Graphics](GRAPHICS.md)

> Added in v1.8.0. Completed in v2.0.0.

`GUI` opens real desktop windows with the ordinary controls of a small
desktop application: Labels, Buttons, single-line and multi-line text,
password fields, Checkboxes, drop-down Selects, ListBoxes, and TableViews,
arranged in Columns and Rows, with file, folder, save, message, and
confirmation dialogs. Clicks, key presses, edits, and selections can run an
AhdCode Function.

With v2.0, AhdCode's first-party GUI covers the ordinary controls needed for
small forms, data-entry tools, database front ends, and file-oriented desktop
utilities. v2.9 continues that toolkit. It now has visibility, Label size and
weight, a program clipboard, text editing, window menus, grids, alignment,
scrollable containers, pictures, wrapping labels, tabs, radio buttons,
progress bars, date fields, timers, one background task, and a Canvas
embedded in the window. A TableView sorts from its headers, and
`GUI.resource` names a file in the application's resource folder.

AhdCode GUI is designed for learning and small desktop applications such as
forms, data-entry tools, simple database front ends, utilities, and API-based
applications. It does not aim to provide the specialized infrastructure needed
for professional video production, 3D authoring, game engines, or similarly
large creative applications. It is also not a Tkinter-compatible toolkit: its
surface is deliberately small, and it grows only where ordinary small programs
need it.

```ahd
bring GUI
from GUI bring (Window, Container, Label, Button, TextInput)

window: Window := GUI.window(title: "Greeter", width: 360, height: 180)
form: Container := window.column()
form.label("Your name")
name: TextInput := form.textInput(placeholder: "e.g. Ayşe")
greet: Button := form.button("Greet")
answer: Label := form.label("")

sayHello: Function := () -> Nothing {
    name: Global TextInput
    answer: Global Label
    answer.setText("Hello, {name.text()}!")
}

greet.onClick(sayHello)
window.wait()
```

GUI only handles user interaction. A TextInput gives a String, a Checkbox a
Bool, a TableView shows rows of Strings, and a Button runs a Function; what
that Function does with the values — store them with [SQLite](SQLITE.md),
write a file, call [HTTP](HTTP.md), compute with
[Statistics](STATISTICS.md) — is ordinary AhdCode. GUI knows nothing about
databases, files, or Excel. The v2.0 example
[`examples/v2.0/ledger_app`](../examples/v2.0/ledger_app/main.ahd) is a
complete SQLite-backed ledger with a table, dialogs, and CSV export, and
[Packaging](PACKAGING.md) turns such a program into a desktop application.

## Surface

```text
GUI.window(title: String := "AhdCode", width: Int := 800, height: Int := 600) -> Window

Window.column(spacing: Int := 8, padding: Int := 12) -> Container
Window.row(spacing: Int := 8, padding: Int := 12) -> Container
Window.onKey(handler: (key: String) -> Nothing) -> Nothing
Window.wait() -> Nothing
Window.close() -> Nothing
Window.isOpen() -> Bool
Window.setTitle(text: String) -> Nothing

Container.column(spacing: Int := 8, padding: Int := 0) -> Container
Container.row(spacing: Int := 8, padding: Int := 0) -> Container
Container.label(text: String) -> Label
Container.button(text: String) -> Button
Container.textInput(placeholder: String := "") -> TextInput
Container.checkbox(text: String, checked: Bool := false) -> Checkbox

Label.text() -> String              Label.setText(text: String) -> Nothing
Button.text() -> String             Button.setText(text: String) -> Nothing
Button.onClick(handler: () -> Nothing) -> Nothing
TextInput.text() -> String          TextInput.setText(text: String) -> Nothing
Checkbox.checked() -> Bool          Checkbox.setChecked(checked: Bool) -> Nothing
```

Added in v1.9.0 (see [Colors](#colors) and [Enabled state](#enabled-state)):

```text
Window.setBackground(color: String) -> Nothing
Container.setBackground(color: String) -> Nothing
Label.setForeground(color: String)      Label.setBackground(color: String)
Button.setForeground(color: String)     Button.setBackground(color: String)
TextInput.setForeground(color: String)  TextInput.setBackground(color: String)
Checkbox.setForeground(color: String)   Checkbox.setBackground(color: String)
Button.setEnabled(enabled: Bool)        Button.isEnabled() -> Bool
TextInput.setEnabled(enabled: Bool)     TextInput.isEnabled() -> Bool
Checkbox.setEnabled(enabled: Bool)      Checkbox.isEnabled() -> Bool
```

Added in v2.0.0:

```text
Window.setResizable(resizable: Bool) -> Nothing
Window.isResizable() -> Bool

Container.passwordInput(placeholder: String := "") -> PasswordInput
Container.textArea(placeholder: String := "") -> TextArea
Container.select(items: List<String>, selectedIndex: Int? := null) -> Select
Container.listBox(items: List<String> := []) -> ListBox
Container.table(columns: List<String>, rows: List<List<String>> := []) -> TableView

TextInput.onChange(handler: (text: String) -> Nothing) -> Nothing
Checkbox.onChange(handler: (checked: Bool) -> Nothing) -> Nothing

PasswordInput.text() -> String      PasswordInput.setText(text: String) -> Nothing
PasswordInput.onChange(handler: (text: String) -> Nothing) -> Nothing
TextArea.text() -> String           TextArea.setText(text: String) -> Nothing
TextArea.onChange(handler: (text: String) -> Nothing) -> Nothing

Select.items() -> List<String>      Select.setItems(items: List<String>) -> Nothing
Select.selectedIndex() -> Int?      Select.selectedText() -> String?
Select.select(index: Int?) -> Nothing
Select.onChange(handler: (index: Int?, text: String?) -> Nothing) -> Nothing
ListBox.items() -> List<String>     ListBox.setItems(items: List<String>) -> Nothing
ListBox.selectedIndex() -> Int?     ListBox.selectedText() -> String?
ListBox.select(index: Int?) -> Nothing
ListBox.onChange(handler: (index: Int?, text: String?) -> Nothing) -> Nothing

TableView.columns() -> List<String>
TableView.rows() -> List<List<String>>
TableView.setRows(rows: List<List<String>>) -> Nothing
TableView.selectedRow() -> Int?     TableView.selectRow(index: Int?) -> Nothing
TableView.onSelect(handler: (row: Int?) -> Nothing) -> Nothing
TableView.sort(column: Int, descending: Bool := false) -> Nothing
TableView.sortedColumn() -> Int?    TableView.sortDescending() -> Bool

GUI.openFile(title: String := "Open File", extensions: List<String> := []) -> String?
GUI.openFiles(title: String := "Open Files", extensions: List<String> := []) -> List<String>
GUI.selectFolder(title: String := "Select Folder") -> String?
GUI.saveFile(title: String := "Save File", suggestedName: String := "", extensions: List<String> := []) -> String?
GUI.message(title: String, text: String) -> Nothing
GUI.confirm(title: String, text: String) -> Bool
GUI.resource(name: String) -> String
```

PasswordInput, TextArea, Select, ListBox, and TableView also have
`setForeground`, `setBackground`, `setEnabled`, and `isEnabled`, exactly as
the v1.9 widgets.

`GUIError` derives from `Error`. None of the Classes has a constructor: a
Window comes from `GUI.window`, a Container from `column` or `row`, and every
widget from a Container. Every call is entirely positional or entirely named.

## Windows

`GUI.window` opens the window at once and returns it; there is no main loop
to write. Width and height must each be between 1 and 4096; the title is any
text of at most 256 characters. An invalid size is a `GUIError`, never
silently clamped. The window has a fixed size unless the program makes it
resizable (see [Resizable windows](#resizable-windows)).

- `wait()` shows the window until the user closes it, running callbacks in
  the meantime (see below), and then returns normally. Everything the program
  wrote before `wait()` appears in the terminal before the window starts
  waiting, and each callback's output appears as soon as it returns.
- `close()` closes the window from the program. Closing twice does nothing.
- `isOpen()` is false after `close()`, and after the user closed the window.
- `setTitle(text)` changes the title bar.

Several Windows can be open at once; each is independent, and closing one
leaves the others open. Every window a program leaves open closes when the
program ends.

After a Window closed, every text field's `text()`, a Checkbox's
`checked()`, and every selection still return their last values, so a
program can read a form after `wait()` returns. Changing a widget, adding a widget, or registering a callback on a
closed Window is a `GUIError`.

## Layout: one root, Columns, Rows, and grids

A Window has at most one root Container: the first `window.column()`,
`window.row()`, `window.grid()`, or `window.scroll()` creates it, and calling
another root is a `GUIError`. An empty Window is valid. Every other Container
is created inside a Container.

- A **Column** places its children top to bottom. A **Row** places them left
  to right. A **grid** places them left to right in a fixed number of columns,
  wrapping to the next row. `columnGap` is the gap between columns and
  `rowGap` the gap between rows; both default to 8.
- `spacing` is the gap between neighboring children of a Column, Row, or
  scroll container, and `padding` the margin inside the Container's edges,
  both in window points. Spacing, padding, and grid gaps must be between 0
  and 1000. A grid has between 1 and 32 columns.
- Every widget has a natural size: a Label is as wide as its text, a Button
  as its text plus a margin (at least 64 points), a TextInput and a
  PasswordInput are 260 points wide, and every single-line widget is 32
  points tall. A TextArea is 360 × 136 points, a ListBox 260 × 146, a
  Select as wide as its longest item (200 to 480 points), and a TableView as
  wide as its columns (200 to 640 points) and 222 points tall.
- With no explicit alignment, a Column's children stay on its left edge and a
  Row's children stay centered vertically. A grid cell starts its child at
  the top-left of the cell.
- The root starts at the window's top-left corner. When the window has a menu
  bar, the root starts 28 points below that bar, so the bar does not cover
  the form. What does not fit is clipped, unless the root is a scroll
  container (below).

`setMinWidth(width)` and `setPreferredWidth(width)` raise a widget's natural
width. They never shrink it: a preferred width smaller than the content is
ignored, and 0 clears the setting. A negative width, or a width above 4096,
is a `GUIError`.

`setAlign(horizontal, vertical := "")` sets alignment. Each value is
`"start"`, `"center"`, `"end"`, or `"stretch"`. Horizontal alignment applies
in a Column and in a grid cell. Vertical alignment applies in a Row and in a
grid cell. An empty vertical value keeps the historical Row behavior.
Stretch makes the child fill the slot on that axis. Alignment never shrinks
a widget and never pulls a widget that is already larger than its slot back
into the slot: the overflow stays clipped, or scrolls when the widget is
inside a scroll container.

`setSpan(columns, rows := 1)` is the number of grid cells a widget covers,
from 1 to 32. A span wider than the grid uses every column. Outside a grid
the span is stored and has no effect.

Grid rules, in order:

1. Hidden children are skipped. The next visible child moves into that cell.
   Hiding a widget does not leave an empty hole.
2. A child that does not fit on the current row wraps. A cell already covered
   by a row span is skipped.
3. A column's width is the widest child that occupies that column alone. A
   row's height is the tallest child that occupies that row alone. A child
   that spans several tracks widens or heightens those tracks, equally, until
   their sum and gaps can hold it.
4. The grid's natural size is the tracks, the gaps, and the padding.
5. When the grid is given extra width, every column receives an equal share
   and the first columns receive any remainder. Extra height goes only to
   rows that contain a growing child (a TextArea, ListBox, TableView, or a
   container that holds one). Otherwise the grid stays top-aligned.
6. A child is aligned in its cell. A growing child fills its cell. Nothing
   shrinks below its natural width.

There is no absolute positioning. The scrolling widgets — TextArea, ListBox,
and TableView — **grow**, and so does every Column, Row, or grid holding one.
When the window is larger than its content, a Column's extra height is shared
equally by its growing children, and a Row's extra width likewise; a growing
child also fills its Column's width (its Row's height).

### Scroll containers

`window.scroll(spacing := 8, padding := 12)` is a scrollable root.
`container.scroll(spacing := 8, padding := 0)` is a nested one. Children are
stacked top to bottom at their natural size; they do not grow inside the
scroll container. The stack's width is the container's natural width. Its
natural height is the stack, or 256 points when the stack is taller. A taller
scroll container then grows, so a Column can give it more of a large window.

A scroll **root** is the exception: it fills the window, below the menu bar
when there is one. A window that is shorter than the form scrolls vertically.
A window that is narrower than the form scrolls horizontally. That is the
ordinary way to scroll a wide form. A nested scroll container is as wide as
its content, so it scrolls vertically; make the scroll container the root
when the form should scroll in both directions.

The wheel scrolls the scroll container under the pointer, on either axis, and
the scroll bar can be dragged. A ListBox, TableView, or TextArea under the
pointer keeps its own scrolling. One notch moves 72 points. A disabled list
does not scroll.
Content is clipped to the container: nothing is drawn outside it, and a click
on a widget that has been scrolled out of view does not land on that widget.
The click lands on the widget now under the pointer, inside the viewport, and
not on the scroll bar. Tab and a scripted focus scroll a focused control into
view. A hidden control takes no space and is skipped by clicks and Tab. A
disabled control stays visible, keeps its space, and does not accept clicks,
keys, or Tab.
An open Select list and the menu bar are drawn above the scrolled content.

`examples/v2.9/layout` is a two-column form in a scrollable window, with a
nested column, a hidden button, and enough lines to scroll.

### Checking a layout by hand

Automated tests check geometry, clipping, wheel scrolling, focus, and which
widget a point hits. They do not move the physical mouse. On this development
Mac, a person used the physical mouse and keyboard on `examples/v2.9/layout`:
the File menu opened, the text field accepted typing, Command+A, Command+C,
Command+V, and Command+Z edited that field, the wheel scrolled the form, and
the window closed cleanly. Windows 11 and Wayland were not run.

The same window is the place to repeat that check:

1. Run `examples/v2.9/layout`. The menu bar sits above the form.
2. Click the Name field and type. Command (macOS) or Ctrl (Windows and Linux)
   +C copies the field. The same modifier with S runs Save.
3. Scroll with the wheel. Click the control now under the pointer and confirm
   that control receives the click.
4. Press Tab until a control that started below the window scrolls into view.
5. Confirm the hidden Extra button leaves no empty cell.

```ahd
bring GUI
from GUI bring (Window, Container)

window: Window := GUI.window(title: "Layout", width: 420, height: 200)
form: Container := window.column(spacing: 10)
form.label("Name")
form.textInput(placeholder: "name")
buttons: Container := form.row(spacing: 12, padding: 0)
buttons.button("Save")
buttons.button("Cancel")
window.wait()
```

## Widgets

- **Label** shows one line of text; `setText` replaces it.
- **Button** shows one line of text and runs its click callback. A click is a
  press and a release of the left mouse button on the same Button; Enter or
  Space activates the focused Button.
- **TextInput** is a single-line text field. The user types, moves the caret
  with the arrow keys, Home, and End, and deletes with Backspace and Delete.
  The placeholder is shown only while the field is empty and unfocused, and
  `text()` never returns it. `setText` replaces the whole text.
- **Checkbox** shows a box and a caption; a click, or Space when it is
  focused, toggles it. `checked()` reads it and `setChecked` sets it.

- **PasswordInput** (v2.0) is a TextInput that shows one dot per character.
  `text()` returns what was typed; the helper never draws or logs it.
- **TextArea** (v2.0) edits several lines of text. Enter starts a new line;
  the arrows, Home and End (of the line), PageUp and PageDown, Backspace, and
  Delete work as usual, and a click places the caret. Long text scrolls
  vertically with a scroll bar, and a long line scrolls sideways with the
  caret; there is no line wrapping, rich text, or syntax coloring. A TextArea
  holds at most 100,000 characters; `setText` turns `"\r\n"` and `"\r"`
  into `"\n"`.
- **Select** (v2.0) is a drop-down choice among Strings. A click, Enter, or
  Space opens its list; a click or Enter picks a choice and Escape closes it.
  While it is closed, the arrow keys change the choice directly. It is not
  editable. `selectedIndex` in `container.select(...)` preselects a choice.
- **ListBox** (v2.0) shows a list of Strings with at most one selected item.
  A click selects an item; the arrows, Home, End, PageUp, and PageDown move
  the selection. A long list scrolls with the wheel or its scroll bar.
- **TableView** (v2.0) — see [TableView](#tableview).

Clicking an interactive widget focuses it; Tab moves the focus to the next
enabled one in the order they were created, and Shift+Tab to the previous
one. Texts are at most 4096 characters (100,000 in a TextArea). The text is
drawn with the Go Regular font bundled with AhdCode, so it looks the same on
every computer, and it is measured for every display scale, so Turkish and
other letters are never clipped; there is no font or theme option.

## Selections

A Select, ListBox, or TableView has at most one selected entry, identified by
its index, or none, which is `null`:

- `selectedIndex()` (`selectedRow()` for a TableView) returns the index or
  `null`; `selectedText()` the selected item's text or `null`.
- `select(index)` (`selectRow`) selects an entry; `select(null)` clears the
  selection. An index outside the entries is a `GUIError`.
- `setItems` (`setRows`) replaces the entries. A selection that still exists
  is kept; one that no longer exists is cleared.
- Only the user's own actions call a change callback. `select`, `setItems`,
  `setRows`, `setText`, and `setChecked` never do, so a callback can update
  the widgets without calling itself.

A ListBox or Select holds at most 20,000 items of at most 1024 characters.

## TableView

`container.table(columns, rows)` shows rows of Strings under a header. Use the
name `TableView` for the widget; `Table` remains [Data](DATA.md)'s table.

```ahd
bring GUI
bring Data
from GUI bring (Window, Container, TableView)

sales := Data.fromRows(["Month", "Amount"], [["Jan", "120"], ["Feb", "95"]])
// A Table's rows are records; a TableView takes plain rows of cells.
cells: List<List<String>> := []
for record in sales.rows() {
    row: Local List<String> := []
    for column in sales.columns() {
        row.add(record[column])
    }
    cells.add(row)
}
window: Window := GUI.window(title: "Sales", width: 480, height: 320)
window.setResizable(true)
form: Container := window.column()
table: TableView := form.table(sales.columns(), cells)
table.onSelect(lambda (row: Int?) -> write("row {row}"))
window.wait()
```

- There must be at least one column, and every row needs exactly one cell
  per column; anything else is a `GUIError`. Column names may repeat: they
  are only labels. Cells are Strings — format numbers and dates yourself —
  and `null` is not a cell.
- Column widths are fixed by the header and the first 1000 rows: each is the
  widest text plus a margin, between 48 and 320 points. A wider text is
  clipped at its column.
- The table scrolls vertically with the wheel, its scroll bar, and the
  keyboard, and horizontally with a sideways wheel or Shift and the wheel.
  Only the visible rows are drawn, so tens of thousands of rows stay fast.
- A click in the body selects a row; when the table has the focus, the
  arrows, Home, End, PageUp, and PageDown move the selection. A click on a
  header sorts that column. The first click is ascending. The next click on
  the same header is descending. A click on another header sorts that column
  ascending. Cells compare as text, in Unicode code point order, case
  sensitive: an empty cell comes before any other text when ascending, and
  `"10"` comes before `"2"`. The order is not numeric and not locale-aware.
  Equal cells keep their current relative order. The selected row stays the
  same row and moves to its new index. A header click does not call
  `onSelect`. `sort(column, descending := false)` is the same order, chosen
  by the program, and it does not call `onSelect` either. `sortedColumn()`
  is null until the table has been sorted. `setRows` keeps the current sort
  and applies it to the new rows: the row that lands at the selected index
  in the new data stays selected and then moves with the sort. A disabled
  table ignores a header click. `sort` still works. Cells are not editable:
  change the data and call `setRows`.
- `columns()` and `rows()` return copies of the rows in the order on screen.
  A TableView holds at most 64
  columns, 20,000 rows, and 200,000 cells of at most 1024 characters.

A [Data](DATA.md) Table's `rows()` are records (`List<Pair<String, String>>`),
so a program turns them into rows of cells explicitly, as above; GUI does not
depend on Data and nothing is bound automatically.

## Callbacks

```text
Button.onClick(handler: () -> Nothing)
Window.onKey(handler: (key: String) -> Nothing)
TextInput.onChange, PasswordInput.onChange, TextArea.onChange(handler: (text: String) -> Nothing)
Checkbox.onChange(handler: (checked: Bool) -> Nothing)
Select.onChange, ListBox.onChange(handler: (index: Int?, text: String?) -> Nothing)
TableView.onSelect(handler: (row: Int?) -> Nothing)
```

The handler's shape is checked when the program is compiled: an `onClick`
Function takes no arguments, an `onKey` Function takes one String, a text
`onChange` one String, a Checkbox `onChange` one Bool, and all return
Nothing. A selection callback must declare its parameters nullable —
`lambda (index: Int?, text: String?) -> ...` — because the selection can be
empty; a mismatch is a compile error, not a runtime failure.

- A text field's `onChange` runs once for each edit the user makes (typing,
  Backspace, Delete, Enter in a TextArea) with the whole new text. A
  Checkbox's runs when the user toggles it; a Select's, ListBox's, or
  TableView's when the user changes the selection.
- Each event has at most one callback. Registering a callback again
  **replaces** the previous one. There is no way to remove a callback.
- Callbacks run during `wait()`, one at a time, in the order the events
  happened, on the program's own path of execution. Two callbacks never run
  at the same time. A background task does not run at the same time as a
  callback either; `GUI.checkpoint` is the only place it lets the window
  handle events.
- A callback may read and change any widget, open another Window, or call any
  module.
- **A long callback blocks the window.** While a callback runs, the window
  does not react; events that happen meanwhile are handled afterwards, in
  order. A callback that waits for an HTTP response or a large query freezes
  the window until it returns. That is the price of the simple serial model.
- If a callback raises an error, the Window is closed and the error
  propagates unchanged out of `wait()`: a `ValueError` stays a `ValueError`
  and can be caught with `except ValueError` around `wait()`.
- When the window closes, events that were still waiting are discarded.

## Key names

`Window.onKey` reports each key press once (key-down only; holding a key does
not repeat it). The key names are:

```text
ArrowUp ArrowDown ArrowLeft ArrowRight Enter Escape Space Tab
Backspace Delete Home End PageUp PageDown
A B C ... Z        0 1 ... 9
```

Letters and digits are named by the key's US-keyboard label, whatever the
active keyboard layout, and a letter is reported in upper case with or
without Shift. Other keys are not reported. There is no key-up event,
modifier API, or key-repeat setting.

`Window.onKey` also receives the keys typed into a focused TextInput: typing
"a" into a field reports "A" to `onKey`, and the TextInput still receives the
character. A program that reacts to letters in `onKey` should keep that in
mind; Escape, the arrows, and Enter are the usual choices.

## Colors

> Added in v1.9.0.

Colors are explicit, not a styling framework: a Window and a Container have a
background, and a Label, Button, TextInput, and Checkbox have a foreground
(their text) and a background.

```ahd
bring GUI
from GUI bring (Window, Container, Label, Button)

window: Window := GUI.window(title: "Colors", width: 360, height: 160)
window.setBackground("#EEF2F7")
form: Container := window.column(spacing: 10, padding: 16)
title: Label := form.label("New order")
title.setForeground("#1F4E79")
save: Button := form.button("Save")
save.setForeground("white")
save.setBackground("#2E7D32")
window.wait()
```

A color is spelled exactly as in [Graphics](GRAPHICS.md): one of the nine
lower-case names `black`, `white`, `red`, `green`, `blue`, `yellow`, `cyan`,
`magenta`, and `gray`, or `#RRGGBB`, or `#RRGGBBAA` (hex digits in either
case). Anything else — `"Red"`, `"purple"`, `"#fff"`, an empty text — is a
`GUIError`; no other color is substituted. Setting a color again replaces it.

- Where no color is set, everything looks exactly as in v1.8.
- A Button's own background replaces its gray fill; while it is held down it
  is drawn 15% darker. A TextInput's background replaces its white field. A
  Label's and a Checkbox's background fills its whole row box; the
  Checkbox's square keeps its own look.
- The placeholder of an empty TextInput stays gray, and focus outlines stay
  blue.
- `#RRGGBBAA` colors blend with normal "source over" compositing: the window
  starts opaque in its default light gray, the Window's background is
  painted over it, and every Container and widget is painted over its parent
  in creation order. The window itself is never transparent.

Changing a color on a closed Window is a `GUIError`.

## Enabled state

> Added in v1.9.0; v2.0 widgets included.

A Button, TextInput, Checkbox, PasswordInput, TextArea, Select, ListBox,
TableView, Image, RadioButton, DateField, or Progress can be disabled. Every
widget starts enabled; Labels, Containers, Tabs, and RadioGroups have no
enabled state. An Image and a Progress bar are never in the Tab order,
enabled or disabled.

- A disabled Button cannot be clicked or activated with Enter or Space, so
  its `onClick` callback does not run.
- A disabled TextInput ignores typing and editing keys.
- A disabled Checkbox cannot be toggled by a click or Space.
- A disabled Select, ListBox, or TableView cannot be opened, selected, or
  scrolled by the user.
- A disabled widget cannot take the focus: clicking it moves the focus
  nowhere, Tab skips it, and a focused widget that is disabled loses the
  focus.
- A disabled RadioButton cannot be selected by a click, Space, or the arrow
  keys. The program can still select it.
- A disabled DateField ignores clicks and typing. The program can still call
  `setDate`.
- A disabled Progress bar is faded. The program can still set its value.
- The program can still change a disabled widget: `setText` and
  `setChecked` work, and `text()` and `checked()` read it.
- A disabled widget is drawn as usual and then faded toward the window's
  default color.

`isEnabled()` reports the state, also after the Window closed; `setEnabled`
on a closed Window is a `GUIError`. A disabled control stays visible and
keeps its place in the layout. Hiding a control is separate: `setVisible`
removes it from the layout, from clicks, and from Tab.

## Resizable windows

> Added in v2.0.0.

`window.setResizable(true)` lets the user resize the window, and
`setResizable(false)` stops it; `isResizable()` reports the setting. Windows
are not resizable unless the program says so, exactly as in v1.9. A
resizable window keeps at least 160 × 120 points. When it grows, the
scrolling widgets and their Containers take the extra space as described in
[Layout](#layout-one-root-columns-and-rows); when it shrinks below the
content, the content is clipped as before.

## Dialogs

> Added in v2.0.0.

```ahd
bring GUI
bring CSV

path: String? := GUI.openFile(title: "Open prices", extensions: ["csv"])
if path != null {
    rows: Local := CSV.read(path)
    write("{len(rows)} rows")
}
target: String? := GUI.saveFile(suggestedName: "report.csv", extensions: ["csv"])
if target != null and GUI.confirm("Export", "Write the report to {target}?") {
    CSV.write(target, [["name"], ["Ayşe"]])
    GUI.message("Export", "Saved.")
}
```

- `openFile` returns the chosen file, `openFiles` every chosen file,
  `selectFolder` the chosen folder, and `saveFile` the path to save to. Every
  path is absolute and in the system's own form.
- **Cancelling is not an error.** A cancelled `openFile`, `selectFolder`, or
  `saveFile` returns `null`, and a cancelled `openFiles` an empty List.
  `GUIError` means the dialog could not be shown at all.
- A dialog only chooses a path: it never opens, creates, or changes a file.
  `saveFile` creates nothing; reading and writing are the program's own,
  separate steps.
- `extensions` offers only files with those extensions, such as
  `["png", "jpg"]`: letters and digits only (a leading dot is ignored);
  anything else, such as `"*.png"`, is a `GUIError`. An empty List offers
  every file. A saved name without an offered extension gets the first one.
- `suggestedName` is a file name without a folder.
- `message` shows a text with an OK button and returns when it is
  dismissed. `confirm` shows OK and Cancel and returns true for OK.

The dialogs are the system's own on macOS (the standard open and save panels
and alerts) and Windows (the common dialogs). On Linux, where AhdCode links no
desktop toolkit, the helper draws its own dialog with GUI widgets: a file
dialog lists one folder at a time, Up goes to the parent folder, and Open on
a selected folder enters it; that dialog chooses one file at a time, also
for `openFiles`. A dialog is modal to the program, which waits for it: an open
GUI window keeps drawing meanwhile but its callbacks run after the dialog
closes.

## Errors

`GUIError` is raised for GUI problems: an invalid window size, spacing, or
padding, an unsupported color, a second root Container, a TableView row of
the wrong width, an index outside the entries, an invalid file extension,
changes to a closed Window, a dialog that could not be shown, a missing,
damaged, or unsupported image, a missing or failed `ahdgui` helper, a
malformed helper response, a date that is not a real calendar day, a
progress value or maximum outside its bounds, a tab or radio index outside
its control, or an empty tab title or radio label. It is never used for
an error raised by a callback, and never for a cancelled dialog.

## Limits

| What | Limit |
| --- | --- |
| Window width and height | 1 to 4096 points |
| Title | 256 characters |
| Label, Button, Checkbox, TextInput, PasswordInput text | 4096 characters |
| TextArea text | 100,000 characters |
| ListBox and Select items | 20,000 items of 1024 characters |
| TableView | 64 columns, 20,000 rows, 200,000 cells of 1024 characters |
| Widgets in one Window | 1024 |
| Dialog title, text | 256, 4096 characters |
| Suggested file name, extensions | 255 characters, 32 extensions of 16 characters |
| Image file | PNG or JPEG, at most 64 MiB, 8192 pixels on a side |
| Image path | 4096 bytes |
| Image width and height | 0 to 4096 points; 0 means that side is not given |
| Tabs | 32 pages; a title is 1 to 256 characters |
| Radio group | 64 options; a label is 1 to 256 characters |
| Progress maximum | 1 to 1000000; the value is 0 through that maximum |
| Date | year 1 to 9999, a real Gregorian month and day |

A request over a limit is a `GUIError`; nothing is truncated silently.

## How it works

Each open Window is drawn by one process of the bundled `ahdgui` helper,
installed next to the other AhdCode helpers, and each dialog by one short-
lived process of the same helper. The program and the helper exchange
bounded JSON messages over the helper's standard input and output; the
helper sends back click, key, change, and close events. The helper runs no shell
command, uses no network, and logs nothing a user types. The only file it
reads is an image the program asks to show. It is built with the same window library as the
Graphics helper (Ebitengine v2.10.2) and embeds the Go Regular font; see
[THIRD_PARTY_NOTICES_GUI.md](../THIRD_PARTY_NOTICES_GUI.md).

`ahdcode run`, compiled programs, and the REPL use the same implementation. A
compiled program finds the helper through the installation that compiled it,
or next to itself in an AhdCode installation; `PATH` is never searched.
`AHDCODE_GUI_RUNTIME` names the helper explicitly for development and tests.

## Platform notes

- macOS: live-tested on this release's development Mac, including the physical
  mouse and keyboard on an in-window File menu, text editing, Command+A, C, V,
  and Z, scrolling, and closing the window. From v1.9.0 the menu
  bar shows **AhdCode** and the Dock shows the AhdCode icon while a Window is
  focused (v1.8.0 showed the helper's name, `ahdgui`); from v2.0 the Dock
  icon has the rounded macOS shape. That system menu bar is the application
  name only. AhdCode menus are drawn inside the window, on every platform.
  In an application made with [`ahdcode package`](PACKAGING.md), the
  application's own name and icon are shown instead.
- Windows and Linux: from v1.9.0 the window uses the AhdCode icon where the
  system shows window icons.
- Windows and Linux: the helper is compiled for both; a Linux desktop needs an
  X11 display (XWayland counts) and the system's OpenGL libraries.
- Windows: the native dialogs are compiled and structurally tested but were
  not run on a Windows machine for this release.
- Automated tests: `AHDCODE_GUI_HEADLESS=1` opens every Window without a
  display, and `AHDCODE_GUI_HEADLESS_DIALOGS` answers dialogs from a list.

## Visibility, Label size, and the clipboard

Every widget and every Container has `setVisible(visible)` and `isVisible()`.
A hidden widget takes no space in its Column, Row, grid, or scroll container,
receives no clicks, and is skipped by Tab. Hiding a Container hides everything inside it, so
`isVisible()` on a child is false until that Container is shown again.
Showing a widget lays it out with its siblings. A hidden control is not a
disabled control: a disabled control stays visible and keeps its space.

A Label has `setSize(size)` and `setBold(bold)`. `size` is `"small"`,
`"normal"`, `"large"`, or `"title"`. Any other text is a GUIError. There is
no free font choice.

`GUI.setClipboard(text)` replaces the program's text clipboard, and
`GUI.clipboard()` reads it. The text is at most 100000 characters. TextInput,
PasswordInput, and TextArea share that clipboard.

A click places the caret. Dragging selects, and a double click selects the
word under the pointer. Shift plus the arrow keys, Home, and End extends the
selection. Backspace, Delete, and typing act on the whole selection. The caret
moves by grapheme, so a combining mark or an emoji stays with its base
character. A paste that does not fit in the field is refused and the text
stays as it was. A PasswordInput can paste, and it does not copy or cut.

Consecutive typing is one undo step. A paste, a delete, or a selection
replaced by typing is its own step. Undo and redo restore the text and the
caret. On macOS the shortcuts are Command+A, C, X, V, and Z, and
Command+Shift+Z redoes. On Windows and Linux they are Control+A, C, X, V, and
Z, and both Control+Y and Control+Shift+Z redo. A hidden or disabled field
ignores these keys.

```ahd
bring GUI
form: Container := GUI.window("Form").column()
note: Label := form.label("Ready")
note.setSize("title")
note.setBold(true)
save: Button := form.button("Save")
save.setVisible(false)
GUI.setClipboard(note.text())
```

## Menus and shortcuts

A menu is drawn in a bar at the top of its Window. The same bar is used on
macOS, Windows, and Linux. It is not the operating system's menu bar.

```ahd
file: Menu := window.menu("File")
save: MenuItem := file.item("Save", "S")
file.separator()
view: Menu := file.menu("View")
bold: MenuItem := view.checkItem("Bold title", "", false)
save.onClick(onSave)
```

`menu(title)` adds a menu to the bar, or a submenu to a menu. A submenu
cannot contain another menu. `item(label, shortcut := "")` adds an item.
`checkItem(label, shortcut := "", checked := false)` adds an item with a
check. `separator()` draws a line. An item has `onClick`, `setEnabled`,
`isEnabled`, and `setText`. A check item also has `setChecked` and
`checked`. Calling `setChecked` on a plain item is a GUIError.

The shortcut text does not include the platform command key. `"S"` means
Command+S on macOS and Ctrl+S on Windows and Linux. `"Shift+S"`, `"Alt+S"`,
and `"Alt+Shift+S"` add Shift and Alt (Option on macOS). `Ctrl`, `Command`,
and `Cmd` are rejected, so the command key is not written twice. A letter
may be written in either case. Two items cannot use the same shortcut; the
second `item` or `checkItem` is a GUIError. An empty shortcut is not a
conflict.

A disabled item keeps its shortcut, and the shortcut does nothing. A check
item flips when it runs, before `onClick`. Callbacks run one at a time, as
Button clicks do, and they see the same `Global` names as any other Function.
Opening a menu does not capture local variables.

Shortcut precedence, for a Window that has the focus:

1. If a TextInput, TextArea, or PasswordInput has the focus and the chord is
   an editing chord, the field handles it and the menu item does not run.
   Those chords are A, C, X, V, and Z, Shift+Z on every platform, and Y on
   Windows and Linux. A PasswordInput still does not copy or cut, and it
   still consumes C and X.
2. Otherwise the one enabled item that owns the chord runs, once. The key is
   not inserted and is not delivered to `Window.onKey`.
3. If the only item for that chord is disabled, the chord does nothing.
4. A Window that is not focused receives no shortcuts. A hidden or disabled
   field is not focused, so it does not take an editing chord.

`examples/v2.9/menu` is a small note window: Save reads the TextInput, and
its shortcut is Command/Ctrl+S.

## Images

`container.image(path, width := 0, height := 0)` shows a PNG or JPEG file.
An Image is not a drawing surface and it is not a Button: it has no click
callback and it is not in the Tab order.

`width` and `height` are window points, from 0 to 4096. Zero means that side
was not given.

- Neither given: the picture's own pixel size, reduced so that neither side
  exceeds 4096. The proportions stay the same.
- Only the width: the height follows the picture. Only the height: the width
  follows the picture. If the side that follows would pass 4096, that side is
  4096 and the picture is fitted inside the box.
- Both given: the box is exactly that size. The picture is fitted inside,
  centered, and keeps its proportions. The empty part uses the Image's
  background, or shows whatever is behind the Image.

`setSize(width := 0, height := 0)` changes that choice later. `setMinWidth`
and `setPreferredWidth` can make the box wider than the picture. The picture
stays centered and is not stretched. `setAlign` and `setSpan` work as they do
for any other widget. An Image can sit in a Column, a Row, a grid, or a
scroll container, and it is clipped by a scroll container the same way a
Label is.

The picture is read when the Image is created and then kept. Later frames do
not read the file and do not decode it again. A missing file, a directory, a
file larger than 64 MiB, a picture larger than 8192 pixels on a side, a
damaged file, or a file that is not a PNG or JPEG is a GUIError, and no Image
is created. GIF and other formats are refused.

The path is relative to the program's working directory, the same rule as
Graphics and File. The runtime makes it absolute before the helper reads it.
The helper's own folder is not searched. A packaged application opened from
Finder uses the home folder as its working directory; one started from a
terminal uses that terminal's folder. `image` does not look in the
application's resource folder.

`GUI.resource(name)` returns the canonical absolute path of one file in
that resource folder. `name` is relative to the folder, with no `..` and no
absolute path. It is not the working directory and it is not the home
directory. The folder and the file are both resolved through symbolic links.
A link is accepted only when the file's canonical path stays inside the
folder's canonical path, including a link through a nested directory. A link
that points outside is a GUIError, and so is a link whose target does not
exist, a missing file, or a directory. `ahdcode run` uses the `resources`
folder next to the entry module. A packaged macOS application uses
`Contents/Resources/app`. A packaged Windows or Linux application uses
`resources` beside the program, not `runtime`. A program that wants a
bundled picture passes that path to `image`:

```ahd
mark: Image := form.image(GUI.resource("icons/mark.png"))
```

`ahdcode package` copies `.png`, `.jpg`, and `.jpeg` files from the `resources`
folder next to the entry into that application folder. Any other file there
is an error. Data files are not copied.

`setEnabled(false)` leaves the Image visible, in its place, and faded. It
still does not take clicks or Tab. `setVisible(false)` removes it from the
layout.

```ahd
mark: Image := form.image("mark.png", 160, 0)
mark.setSize(120, 80)
```

## Wrapping labels

`label.setWrap(wrap)` turns wrapping on or off. Wrapping is off unless the
program turns it on. A Label that does not wrap is still one line, and
`setSize` and `setBold` behave as they did before.

With wrapping on:

- The wrap width is the larger of the preferred width and the minimum width
  when either is greater than zero. That width is used even when the text is
  wider. The Label's width becomes that wrap width, and its height is the
  number of lines times the Label's line height.
- When neither width is set, the Label breaks only at explicit line breaks.
  Its width is its longest line.
- `\r\n` and `\r` are line breaks, as `\n` is. A blank line keeps its height.
- Spaces separate words, and a run of spaces becomes one space. A word wider
  than the line is split on grapheme clusters, so a Turkish letter or an
  emoji is not cut apart. One cluster that is itself wider than the line
  stays on its own line and is clipped to the Label's box. The embedded font
  has no emoji glyph, so that cluster is drawn as the font's missing-character
  box.
- Text is drawn from the top of the box, one line at a time, and is clipped
  to that box. A scroll container clips it again. Stretching the Label gives
  it a wider box and the text is wrapped for that width. The height does not
  shrink when the box gets wider.
- When the window or a scroll viewport is narrower than that measured width,
  the Label is measured again at the narrower width before the widgets below
  it are placed. The new height is the new line count, so those widgets move
  down and the lines do not overlap them. The Label does not become narrower
  than its minimum width or its widest grapheme cluster. A field, button,
  picture, or other widget that does not wrap keeps its measured width, and
  a grid column stays wide enough for that widget. The Label then wraps to
  the slot those widgets leave. If that slot is still wider than the window,
  the extra is scrolled or clipped. Drawing still keeps the measured line
  breaks if a box is narrower than the height it was given.
- `setSize` and `setBold` still change the line height and the weight. The
  extra width of a bold Label is the same extra width a one-line Label uses.

`setWrap` on anything other than a Label is a GUIError. `setWrap()` with no
argument is rejected by the compiler.

`examples/v2.9/gallery` puts an image, a wrapping label, a grid, and a scroll
container in one window.

## What GUI is not

These are intentional boundaries, not missing basics:

- no tree view, ribbon, docking, or multi-document windows, and no
  operating-system menu bar beyond the application name;
- no rich-text editor, syntax highlighting, or embedded browser or web view;
- no theme or CSS engine, and no font beyond a Label's four sizes and bold;
- no custom-widget or plug-in system, and no visual GUI designer;
- no editable spreadsheet cells, formulas, or filtering framework in
  TableView. Header sorting compares text, as described above. There is no
  automatic binding to databases or Data Tables;
- no drag and drop, and no general animation framework. `window.every` is a
  timer. The indeterminate progress bar still draws its own moving block.
  An Image shows a picture file. A Canvas inside the window draws a small
  set of shapes; [Graphics](GRAPHICS.md) remains the full drawing window;
- one cooperative background task per window, not async, threads, futures,
  or general multithreading. A callback still runs one at a time, and a
  long callback still blocks that window until it returns;
- no infrastructure for professional creative applications such as video
  editors, 3D authoring tools, or game engines.

## Tabs, radio buttons, progress, and dates

`container.tabs()` returns a Tabs control. `tabs.page(title)` adds a page
and returns a Container, so that page holds the same widgets a Column holds.
A title is 1 to 256 characters, and one Tabs control holds at most 32 pages.
The first page is selected. `select(index)` selects a page by creation
order, including a hidden page, and `selectedIndex()` returns that index.
Both raise `GUIError` when the control has no pages or the index is outside
them.

Only the selected page is drawn and receives clicks, keys, and Tab. The
other pages keep their widgets, but those widgets are not hit and are not
Tab stops. Clicking a header selects that page. Left and Right move among
the visible headers and leave the focus on the tab bar. Coming back to a
page by clicking its header restores the focus to the control that last had
it on that page, when that control can still take the focus. A hidden page
has no header. If the selected page is hidden, its content area is empty.
`onChange` receives the new index when the user changes the page. `select`
does not call it.

`container.radioGroup()` returns a RadioGroup. `group.option(text)` adds a
RadioButton whose text is 1 to 256 characters. A group holds at most 64
options. Nothing is selected until the user or the program selects one.
`select(index)` takes `Int?`, and `null` clears the selection.
`selectedIndex()` and `selectedText()` are null when nothing is selected.
The index counts every option, including a hidden one. A click or Space
selects an enabled option. Up and Down move among the enabled, visible
options and select the one they reach. A disabled option cannot be selected
by the user; `select` can still select it. `onChange` receives the index and
the text, both null when the selection is cleared, and only for a change
the user made. A RadioButton has text, colors, enabled, and visible. The
group itself is not enabled or disabled.

`container.progress(maximum := 100)` returns a Progress bar. The maximum is
an Int from 1 to 1000000. The value starts at 0. `setValue` requires an Int
from 0 through the maximum; anything else raises `GUIError` and leaves the
value unchanged. `setIndeterminate(true)` draws a moving block instead of a
fraction of the bar. The program can still read and set the value while it
moves. Setting the value does not run a callback and does not wait: the
window draws the new value on its next frame. A progress bar is not a Tab
stop. Disabling it fades it and does not stop the program from setting the
value. Its natural size is one text field wide and one row tall.

`container.date(year := 2000, month := 1, day := 1)` returns a DateField.
The date is one proleptic Gregorian day: year 1 through 9999, month 1
through 12, and a day that month has. 29 February is valid when the year is
divisible by 4, except a year divisible by 100 unless it is also divisible
by 400. The control always holds a real date. An invalid date raises
`GUIError`. `setDate` then leaves the previous date in place. The field is
drawn `YYYY-MM-DD`, with zeros and no month names, so macOS, Windows, and
Linux show the same numerals. It is one Tab stop with three equal parts.
A click chooses a part. Left and Right move between the parts. Up and Down
change that part by one. The year stays inside 1 through 9999, the month
inside 1 through 12, and the day is clamped to the month, including when
the month or the year changes. Digits replace the part, four for the year
and two for the month or the day, and they apply only when the resulting
date is real. `year()`, `month()`, and `day()` read it. `onChange` receives
all three when the user changes the date. `setDate` does not call it.

`examples/v2.9/controls` puts all four controls in one window.

## Timers, background tasks, and an embedded Canvas

`window.every(milliseconds, handler)` registers a repeating timer and
returns its Int identifier. `window.cancel(timer)` stops that timer.
Cancelling an identifier that names no timer does nothing. The interval is
1 millisecond through one day (86,400,000 milliseconds). A window holds at
most 64 timers; the next `every` raises `GUIError` until one is cancelled.
The handler is `() -> Nothing`.

A real window posts one tick at a time onto the same event queue as clicks
and keys. The handler runs inside `wait`, one at a time, in order with those
events, and may read and change widgets. A handler that runs longer than
its interval does not build a queue: the ticks it missed are skipped. Cancel
from inside the handler takes effect before the next tick. Closing the
window stops every timer, and a tick already queued after that close is not
run. A headless window has no clock. Its timers run only when the script
sends `{"event":"tick"}`. A tick with no `timer` field, or with `timer` 0,
runs every timer in the order they were started. A non-zero `timer` runs
that one.

`window.task(work)` starts one background task and returns a `Task`. `work`
is `() -> Int`. A window runs one task at a time; a second `task` before the
first has returned raises `GUIError`. The function runs while `wait` is
running, including a task created before `wait`. It does not run at the same
time as a click, a timer, or any other GUI callback. `GUI.checkpoint()` is
how it lets those callbacks run. Outside a task, `checkpoint` returns true
and changes nothing. Inside a task it returns false when the task was
cancelled or the window has started to close, and true when the window can
still deliver a result. It does not stop the function. The function decides
whether to return. A stretch of the function that never calls `checkpoint`,
including one file or database call, holds the GUI callbacks until it
returns. The window process keeps drawing; the program does not handle
clicks or timers during that stretch.

`task.onDone(handler)` registers `(value: Int?, problem: String?) -> Nothing`.
Registering again replaces the handler. The handler runs on the same path as
other callbacks, so it may change widgets. On success, `value` is the Int and
`problem` is null. When `work` raises an Error, `value` is null and `problem`
is that error's message; any other failure still propagates. `onDone` runs
only when the function has returned, the task was not cancelled, and the
window is still open. A task that is already finished when `onDone` is
registered is delivered then, under those same rules.

`task.cancel()` asks that the result be discarded. It does not interrupt the
function. If the function returns after `cancel`, `onDone` does not run.
`task.cancelled()` is true after `cancel`, after the window starts closing
while the function is still running, and for an identifier that never named
a task. Cancelling a finished task or an unknown task does nothing.

The task must not change the window. A widget call made from the task raises
`GUIError` (`a background task cannot change the window`). Update labels,
progress bars, and other widgets from `onDone`, from a timer, or from another
callback. The task and those callbacks do not share a hidden mutable store:
they exchange only values the program itself assigns, and they do not run at
the same time, except while `checkpoint` has let the callbacks run. Do not
read or write the same variable from both sides across `checkpoint` unless
the program can accept either order.

Closing the window cancels a task that has not returned. `wait` then waits
until the function returns, so a function that ignores `checkpoint` and never
returns also keeps `wait` from returning. The result is discarded and
`onDone` does not run. A task that already finished and whose completion was
queued while the window was open can still run `onDone` before `wait`
returns.

`container.canvas(width := 240, height := 160)` returns a Canvas and does
not open a Graphics window. It can be added to a Column, a Row, a Grid, a
Scroll container, or a Tabs page. It cannot be the root. The width and height
are the natural size, from 1 to 4096. Layout may stretch it. Coordinates use
the placed size: the origin is the center, x grows right, and y grows up, the
same arrangement Graphics uses. Drawing is clipped to the Canvas and to any
scroll viewport around it. A hidden Canvas takes no space and receives no
pointer events. A disabled Canvas stays visible, fades, and receives none.

This Canvas is a subset of Graphics drawing. It has no turtle, images, arcs,
polygons, ellipses, SVG, or batch. Rectangle and circle strokes are one unit
wide. A line's width is 1 through 64.

```text
Canvas.clear(color: String := "") -> Nothing
Canvas.line(x1: Real, y1: Real, x2: Real, y2: Real, stroke: String := "black", width: Real := 1) -> Nothing
Canvas.rectangle(x: Real, y: Real, width: Real, height: Real, stroke: String := "black", fill: String := "") -> Nothing
Canvas.circle(x: Real, y: Real, radius: Real, stroke: String := "black", fill: String := "") -> Nothing
Canvas.text(x: Real, y: Real, text: String, fill: String := "black", size: Real := 16) -> Nothing
Canvas.onMove(handler: (x: Real, y: Real) -> Nothing) -> Nothing
Canvas.onClick(handler: (x: Real, y: Real) -> Nothing) -> Nothing
Canvas.onDrag(handler: (x: Real, y: Real) -> Nothing) -> Nothing
```

An empty `clear` color is white. An empty fill draws no fill. Colors are the
Graphics names and the `#RRGGBB` and `#RRGGBBAA` forms. Coordinates must be
finite and at most 100,000 in absolute value. A rectangle's width and height
must be greater than 0. A circle's radius must be greater than 0 and at most
100,000. Text is 1 to 1000 characters and its size is 1 through 200. `(x, y)`
for text is the baseline. A Canvas holds at most 4096 drawing commands;
`clear` starts again. `setBackground`, `setEnabled`, `setVisible`,
`setMinWidth`, `setPreferredWidth`, `setAlign`, and `setSpan` are the same
members other widgets have. There is no `setForeground`. A Canvas is not a
Tab stop.

`onMove` reports the pointer while no button is held. `onClick` reports a
press that began on the Canvas. `onDrag` reports the pointer while that press
continues, including after it leaves the Canvas, so a coordinate may lie
outside. A disabled, hidden, or inactive-page Canvas reports none of these.
A drag that has already begun stops if the Canvas becomes disabled or hidden.
The coordinates are Canvas coordinates, including when the Canvas sits in a
scrolled container: the widget's place already includes the scroll, and the
pointer is measured from the Canvas center. GUI's Canvas and Graphics' Canvas
are different classes with the same name. `from GUI bring Canvas` and
`from Graphics bring Canvas` cannot both be in one scope. A program that only
needs the embedded surface brings GUI's Canvas, or it never names the class
and uses the value `container.canvas` returns.

`examples/v2.9/interactive` puts a timer, a progress bar, a background sum,
and a moving circle on one Canvas, with Start, Stop, and Cancel.

## Known limits still open

1. Windows 11 and a Wayland session have not been run for this GUI work.
   macOS has the automated tests and physical mouse and keyboard passes of
   the layout example, Controls, Interactive, and the ledger.
