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
utilities. Future releases may add focused widgets when a real use case
requires them, but GUI is no longer an active foundational roadmap.

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

GUI.openFile(title: String := "Open File", extensions: List<String> := []) -> String?
GUI.openFiles(title: String := "Open Files", extensions: List<String> := []) -> List<String>
GUI.selectFolder(title: String := "Select Folder") -> String?
GUI.saveFile(title: String := "Save File", suggestedName: String := "", extensions: List<String> := []) -> String?
GUI.message(title: String, text: String) -> Nothing
GUI.confirm(title: String, text: String) -> Bool
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

## Layout: one root, Columns and Rows

A Window has at most one root Container: the first `window.column()` or
`window.row()` creates it, and calling either again is a `GUIError`. An empty
Window is valid. Every other Container is created inside a Container.

- A **Column** places its children top to bottom; a **Row** places them left
  to right.
- `spacing` is the gap between neighboring children and `padding` the margin
  inside the Container's edges, both in window points. Both must be between 0
  and 1000; a negative value is a `GUIError`.
- Every widget has a natural size: a Label is as wide as its text, a Button
  as its text plus a margin (at least 64 points), a TextInput and a
  PasswordInput are 260 points wide, and every single-line widget is 32
  points tall. A TextArea is 360 × 136 points, a ListBox 260 × 146, a
  Select as wide as its longest item (200 to 480 points), and a TableView as
  wide as its columns (200 to 640 points) and 222 points tall.
- A Column's children are aligned to its left edge; a Row's children are
  centered vertically.
- The root Container starts at the window's top-left corner. What does not fit
  in the window is clipped; choose a window size that fits the form.

The scrolling widgets — TextArea, ListBox, and TableView — **grow**, and so
does every Container holding one. When the window is larger than its content
(a resizable window the user enlarged, or a window created larger), a
Column's extra height is shared equally by its growing children, and a Row's
extra width likewise; a growing child also fills its Column's width (its
Row's height). Labels, Buttons, text fields, Checkboxes, and Selects never
stretch, and nothing shrinks below its natural size. There is no grid,
absolute positioning, alignment option, or growth setting: this one rule
lets a table or a list use the space of a large window.

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
- A click selects a row; when the table has the focus, the arrows, Home,
  End, PageUp, and PageDown move the selection. Cells are not editable:
  change the data and call `setRows`.
- `columns()` and `rows()` return copies. A TableView holds at most 64
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
  at the same time, and there are no threads to manage.
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

A Button, TextInput, Checkbox, PasswordInput, TextArea, Select, ListBox, or
TableView can be disabled. Every widget starts enabled; Labels and Containers
have no enabled state.

- A disabled Button cannot be clicked or activated with Enter or Space, so
  its `onClick` callback does not run.
- A disabled TextInput ignores typing and editing keys.
- A disabled Checkbox cannot be toggled by a click or Space.
- A disabled Select, ListBox, or TableView cannot be opened, selected, or
  scrolled by the user.
- A disabled widget cannot take the focus: clicking it moves the focus
  nowhere, Tab skips it, and a focused widget that is disabled loses the
  focus.
- The program can still change a disabled widget: `setText` and
  `setChecked` work, and `text()` and `checked()` read it.
- A disabled widget is drawn as usual and then faded toward the window's
  default color.

`isEnabled()` reports the state, also after the Window closed; `setEnabled`
on a closed Window is a `GUIError`. There is no visibility, hide, or show
API.

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
changes to a closed Window, a dialog that could not be shown, a missing or
failed `ahdgui` helper, and a malformed helper response. It is never used for
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

A request over a limit is a `GUIError`; nothing is truncated silently.

## How it works

Each open Window is drawn by one process of the bundled `ahdgui` helper,
installed next to the other AhdCode helpers, and each dialog by one short-
lived process of the same helper. The program and the helper exchange
bounded JSON messages over the helper's standard input and output; the
helper sends back click, key, change, and close events. The helper runs no shell
command, uses no network, reads no files or environment files, and logs
nothing a user types. It is built with the same window library as the
Graphics helper (Ebitengine v2.10.2) and embeds the Go Regular font; see
[THIRD_PARTY_NOTICES_GUI.md](../THIRD_PARTY_NOTICES_GUI.md).

`ahdcode run`, compiled programs, and the REPL use the same implementation. A
compiled program finds the helper through the installation that compiled it,
or next to itself in an AhdCode installation; `PATH` is never searched.
`AHDCODE_GUI_RUNTIME` names the helper explicitly for development and tests.

## Platform notes

- macOS: live-tested on this release's development Mac. From v1.9.0 the menu
  bar shows **AhdCode** and the Dock shows the AhdCode icon while a Window is
  focused (v1.8.0 showed the helper's name, `ahdgui`); from v2.0 the Dock
  icon has the rounded macOS shape. In an application made with
  [`ahdcode package`](PACKAGING.md), the application's own name and icon are
  shown instead.
- Windows and Linux: from v1.9.0 the window uses the AhdCode icon where the
  system shows window icons.
- Windows and Linux: the helper is compiled for both; a Linux desktop needs an
  X11 display (XWayland counts) and the system's OpenGL libraries.
- Windows: the native dialogs are compiled and structurally tested but were
  not run on a Windows machine for this release.
- Automated tests: `AHDCODE_GUI_HEADLESS=1` opens every Window without a
  display, and `AHDCODE_GUI_HEADLESS_DIALOGS` answers dialogs from a list.

## What GUI is not

These are intentional boundaries, not missing basics:

- no tree view, tabs, menus, ribbon, docking, or multi-document windows;
- no rich-text editor, syntax highlighting, or embedded browser or web view;
- no theme or CSS engine, fonts, or styles beyond the explicit colors;
- no custom-widget or plug-in system, and no visual GUI designer;
- no editable spreadsheet cells, formulas, sorting, or filtering framework
  in TableView, and no automatic binding to databases or Data Tables;
- no drag and drop, clipboard API, animations, or drawing inside a Window
  (use [Graphics](GRAPHICS.md) for drawing);
- no async, threads, futures, or background tasks: callbacks run one at a
  time, and a long callback blocks its window;
- no infrastructure for professional creative applications such as video
  editors, 3D authoring tools, or game engines.

Radio buttons are covered by Select, and visibility (hide/show) is not part
of v2.0.
