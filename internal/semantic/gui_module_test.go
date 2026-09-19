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
	want := "Button Checkbox Container GUIError Label ListBox PasswordInput Select TableView TextArea TextInput Window confirm message openFile openFiles saveFile selectFolder window"
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
window.setBackground("#F0F4F8")
form.setBackground(color: "white")
status.setForeground("blue")
status.setBackground("#FFFF0080")
button.setForeground("white")
button.setBackground("#0066CC")
input.setForeground("black")
input.setBackground("yellow")
check.setForeground("red")
check.setBackground("gray")
button.setEnabled(false)
input.setEnabled(enabled: false)
check.setEnabled(true)
enabled: Bool := button.isEnabled() and input.isEnabled() and check.isEnabled()
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
		// v1.9 colors and enabled state: exact shapes, and only where they apply.
		`window.setForeground("red")`,
		`window.setEnabled(false)`,
		`form.setForeground("red")`,
		`form.setEnabled(false)`,
		`status.setEnabled(false)`,
		`status.isEnabled()`,
		`button.setBackground(1)`,
		`button.setBackground()`,
		`button.setBackground("red", "blue")`,
		`button.setEnabled("yes")`,
		`button.setEnabled()`,
		`count: Int := button.isEnabled()`,
		`result: Int := button.setBackground("red")`,
		`button.setVisible(false)`,
		`button.setFont("x")`,
		`button.setStyle("x")`,
		`window.setTheme("dark")`,
		`input.setPlaceholderColor("gray")`,
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, guiPreamble+source+"\n"))
	}
}
