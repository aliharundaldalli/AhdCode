# GUI standard module

[English] · [Türkçe](GUI_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [Graphics](GRAPHICS.md)

> Added in v1.8.0.

`GUI` opens real desktop windows with a small set of widgets: a Label, a
Button, a single-line TextInput, and a Checkbox, arranged in Columns and
Rows. A Button click and a key press can run an AhdCode Function.

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
Bool, and a Button runs a Function; what that Function does with the values —
store them with [SQLite](SQLITE.md), write a file, call [HTTP](HTTP.md),
compute with [Statistics](STATISTICS.md) — is ordinary AhdCode. GUI knows
nothing about databases, files, or Excel.

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

`GUIError` derives from `Error`. None of the Classes has a constructor: a
Window comes from `GUI.window`, a Container from `column` or `row`, and every
widget from a Container. Every call is entirely positional or entirely named.

## Windows

`GUI.window` opens the window at once and returns it; there is no main loop
to write. Width and height must each be between 1 and 4096; the title is any
text of at most 256 characters. An invalid size is a `GUIError`, never
silently clamped. The window cannot be resized in v1.8.

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

After a Window closed, a TextInput's `text()` and a Checkbox's `checked()`
still return their last values, so a program can read a form after `wait()`
returns. Changing a widget, adding a widget, or registering a callback on a
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
- Every widget keeps its natural size: a Label is as wide as its text, a
  Button as its text plus a margin (at least 64 points), a TextInput is 260
  points wide, and every widget is 32 points tall.
- A Column's children are aligned to its left edge; a Row's children are
  centered vertically.
- The root Container starts at the window's top-left corner. What does not fit
  in the window is clipped; choose a window size that fits the form.

There is no grid, absolute positioning, alignment option, stretching, or
scrolling. Forms are easy to build from Columns and Rows; not every desktop
layout can be expressed, on purpose.

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

Clicking a Button, TextInput, or Checkbox focuses it; Tab moves the focus to
the next one in the order they were created. Texts are at most 4096
characters. The text is drawn with the Go Regular font bundled with AhdCode,
so it looks the same on every computer; there is no font, color, or theme
option in v1.8.

## Callbacks

```text
Button.onClick(handler: () -> Nothing)
Window.onKey(handler: (key: String) -> Nothing)
```

The handler's shape is checked when the program is compiled: an `onClick`
Function takes no arguments, an `onKey` Function takes one String, and both
return Nothing. A mismatch is a compile error, not a runtime failure.

- Each event has at most one callback. Registering `onClick` or `onKey` again
  **replaces** the previous callback. There is no way to remove a callback in
  v1.8.
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

## Errors

`GUIError` is raised for GUI problems: an invalid window size, spacing, or
padding, a second root Container, changes to a closed Window, a missing or
failed `ahdgui` helper, and a malformed helper response. It is never used for
an error raised by a callback.

## How it works

Each open Window is drawn by one process of the bundled `ahdgui` helper,
installed next to the other AhdCode helpers. The program and the helper
exchange bounded JSON messages over the helper's standard input and output;
the helper sends back click, key, and close events. The helper runs no shell
command, uses no network, reads no files or environment files, and logs
nothing a user types. It is built with the same window library as the
Graphics helper (Ebitengine v2.10.2) and embeds the Go Regular font; see
[THIRD_PARTY_NOTICES_GUI.md](../THIRD_PARTY_NOTICES_GUI.md).

`ahdcode run`, compiled programs, and the REPL use the same implementation. A
compiled program finds the helper through the installation that compiled it,
or next to itself in an AhdCode installation; `PATH` is never searched.
`AHDCODE_GUI_RUNTIME` names the helper explicitly for development and tests.

## Platform notes

- macOS: live-tested on this release's development Mac. The menu bar shows
  the helper's name, `ahdgui`, while a Window is focused.
- Windows and Linux: the helper is compiled for both; a Linux desktop needs an
  X11 display (XWayland counts) and the system's OpenGL libraries.
- Automated tests: `AHDCODE_GUI_HEADLESS=1` opens every Window without a
  display.

## Not in v1.8

GUI v1.8 has no table or grid widget, list box, drop-down, radio buttons,
tabs, tree, menus, toolbars, status bars, multi-line text, password fields,
images or icons, clipboard, drag and drop, file or folder pickers, dialogs,
scrolling, resizable windows, themes, styles, fonts, animations, drawing
inside a Window, `TextInput.onChange`, `Checkbox.onChange`, background tasks,
threads, or application packaging. Some of these may come in later versions;
packaging programs as desktop applications is planned separately.
