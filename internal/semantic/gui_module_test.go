package semantic

import (
	"testing"
)

const guiPreamble = `bring GUI
bring Graphics
from GUI bring (Window, Container, Label, Button, TextInput, Checkbox, GUIError)
from Graphics bring (Canvas)
window: Window := GUI.window()
form: Container := window.column()
button: Button := form.button("Save")
input: TextInput := form.textInput()
check: Checkbox := form.checkbox("Paid")
status: Label := form.label("Ready")
canvas: Canvas := Graphics.open()
`

func TestGUIModuleRegistered(t *testing.T) {
	module := StandardModuleInterfaces()["GUI"]
	if module == nil || module.ModuleID != "builtin:GUI" {
		t.Fatalf("GUI module = %#v", module)
	}
	want := "Button Checkbox Container GUIError Label TextInput Window window"
	got := ""
	for index, name := range module.ExportNames {
		if index > 0 {
			got += " "
		}
		got += name
	}
	if got != want {
		t.Fatalf("GUI exports = %s, want %s", got, want)
	}
}

func TestGUIValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, guiPreamble+`other: Window := GUI.window(title: "Two", width: 300, height: 200)
defaults: Window := GUI.window("Positional", 640, 480)
row: Container := form.row(spacing: 4, padding: 2)
nested: Container := row.column(4, 0)
nested.label(text: "Name")
named: TextInput := nested.textInput(placeholder: "e.g. Ayşe")
flag: Checkbox := row.checkbox(text: "Done", checked: true)
save: Button := row.button(text: "OK")
text: String := input.text()
input.setText("x")
status.setText(text: status.text() + button.text())
button.setText("Saved")
isChecked: Bool := check.checked()
check.setChecked(checked: false)
window.setTitle("Title")
open: Bool := window.isOpen()
clicked: Function := () -> Nothing {
    status: Global Label
    status.setText("clicked")
}
pressed: Function := (key: String) -> Nothing {
    write(key)
}
where: Function := (x: Real, y: Real) -> Nothing {
    write("{x} {y}")
}
button.onClick(clicked)
button.onClick(handler: lambda () -> write("inline"))
window.onKey(pressed)
window.onKey(handler: lambda (key: String) -> write(key))
canvas.onClick(where)
canvas.onClick(lambda (x: Real, y: Real) -> write(x + y))
canvas.onKey(pressed)
canvas.onKey(handler: pressed)
attempt {
    window.wait()
} except GUIError as error {
    write(error.message)
}
window.close()
`)
	requireSemanticClean(t, result)
}

// Static mistakes, including every wrong callback shape, are compiler
// diagnostics, never a runtime GUIError.
func TestGUIRejectsStaticMistakes(t *testing.T) {
	for _, source := range []string{
		`GUI.window(800)`,
		`GUI.window("t", "800")`,
		`GUI.window(title: "t", size: 800)`,
		`GUI.window("t", 800, 600, "extra")`,
		`Window()`,
		`Button()`,
		`Container()`,
		`window.column("8")`,
		`window.column(gap: 8)`,
		`window.grid()`,
		`form.label()`,
		`form.label(1)`,
		`form.button()`,
		`form.textInput(1)`,
		`form.checkbox()`,
		`form.checkbox("x", "yes")`,
		`form.table()`,
		`form.dropdown()`,
		`form.image("a.png")`,
		`label: Label := form.button("x")`,
		`input.setText()`,
		`input.setText(1)`,
		`input.onChange(lambda () -> write(1))`,
		`input.onKey(lambda (key: String) -> write(key))`,
		`check.setChecked("yes")`,
		`check.onChange(lambda () -> write(1))`,
		`count: Int := check.checked()`,
		`status.onClick(lambda () -> write(1))`,
		`window.wait(1)`,
		`window.setTitle()`,
		`window.onClick(lambda () -> write(1))`,
		`window.onKeyUp(lambda (key: String) -> write(key))`,
		`GUI.fileDialog()`,
		`GUI.alert("x")`,
		// Callback shapes are checked statically.
		`button.onClick(lambda (x: Int) -> write(x))`,
		`button.onClick(lambda (key: String) -> write(key))`,
		`button.onClick(1)`,
		`button.onClick()`,
		`button.onClick("save")`,
		`window.onKey(lambda () -> write(1))`,
		`window.onKey(lambda (key: Int) -> write(key))`,
		`window.onKey(lambda (a: String, b: String) -> write(a))`,
		`canvas.onClick(lambda (x: Real) -> write(x))`,
		`canvas.onClick(lambda (x: String, y: String) -> write(x))`,
		`canvas.onClick(lambda () -> write(1))`,
		`canvas.onKey(lambda (key: Int) -> write(key))`,
		`canvas.onKey(lambda () -> write(1))`,
		// Graphics stays visual programming, not a game engine.
		`canvas.onMouseMove(lambda (x: Real, y: Real) -> write(x))`,
		`canvas.isKeyDown("A")`,
		`canvas.mouseX()`,
		`result: Int := button.onClick(lambda () -> write(1))`,
		// A callback returns Nothing: a value-returning Function is rejected.
		`button.onClick(lambda () -> 1)`,
		`window.onKey(lambda (key: String) -> key)`,
		`canvas.onClick(lambda (x: Real, y: Real) -> x + y)`,
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, guiPreamble+source+"\n"))
	}
}
