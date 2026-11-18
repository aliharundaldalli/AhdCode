package semantic

import (
	"testing"
)

const guiPreamble = `bring GUI
bring Graphics
from GUI bring (Window, Container, Label, Button, TextInput, Checkbox, GUIError, Menu, MenuItem, Image)
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
	want := "Button Canvas Checkbox Container DateField GUIError Image Label ListBox Menu MenuItem PasswordInput Progress RadioButton RadioGroup Select TableView Tabs Task TextArea TextInput Window checkpoint clipboard confirm message openFile openFiles resource saveFile selectFolder setClipboard window"
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
button.setVisible(false)
form.setVisible(visible: true)
seen: Bool := button.isVisible() and form.isVisible() and status.isVisible()
status.setSize("title")
status.setBold(true)
status.setWrap(true)
status.setPreferredWidth(120)
photo: Image := form.image("photo.png", 80, 40)
photo.setSize(40)
photo.setEnabled(false)
photo.setBackground("#FFFFFF")
photoSeen: Bool := photo.isEnabled() and photo.isVisible()
GUI.setClipboard("note")
copied: String := GUI.clipboard()
file: Menu := window.menu("File")
editMenu: Menu := file.menu("Edit")
saveItem: MenuItem := file.item("Save", "S")
plain: MenuItem := file.item("Quit")
boldItem: MenuItem := editMenu.checkItem("Bold", "Shift+B", false)
file.separator()
saveItem.onClick(clicked)
saveItem.setEnabled(false)
itemEnabled: Bool := saveItem.isEnabled()
boldItem.setChecked(true)
marked: Bool := boldItem.checked()
saveItem.setText("Store")
page: Container := form.scroll(spacing: 8, padding: 0)
sheet: Container := page.grid(columns: 2, columnGap: 8, rowGap: 4)
sheet.label("A")
wideButton: Button := sheet.button("Wide")
wideButton.setMinWidth(80)
wideButton.setPreferredWidth(120)
wideButton.setAlign("center", "start")
wideButton.setSpan(2, 1)
window.close()
from GUI bring (Tabs, RadioGroup, RadioButton, Progress, DateField)
tabs: Tabs := form.tabs()
firstPage: Container := tabs.page("One")
firstPage.label("inside")
tabs.select(0)
tabAt: Int := tabs.selectedIndex()
tabs.onChange(lambda (index: Int) -> write(index))
group: RadioGroup := form.radioGroup()
choice: RadioButton := group.option("Alpha")
choice.setText("First")
choice.setEnabled(false)
group.select(null)
group.select(0)
picked: Int? := group.selectedIndex()
pickedText: String? := group.selectedText()
group.onChange(lambda (index: Int?, text: String?) -> write("kept"))
bar: Progress := form.progress(maximum: 10)
openBar: Progress := form.progress()
bar.setValue(3)
bar.setIndeterminate(true)
moving: Bool := bar.isIndeterminate()
amount: Int := bar.value()
limit: Int := bar.maximum()
when: DateField := form.date(2024, 2, 29)
standard: DateField := form.date()
when.setDate(2023, 1, 31)
y: Int := when.year()
m: Int := when.month()
d: Int := when.day()
when.onChange(lambda (year: Int, month: Int, day: Int) -> write(year + month + day))
form.canvas(120, 80).clear("white")
form.canvas().circle(0, 0, 10, "blue", "blue")
window.cancel(window.every(1000, lambda () -> write("t")))
window.task(lambda () -> 1).onDone(lambda (value: Int?, problem: String?) -> write("done"))
window.task(lambda () -> 1).cancel()
stopped: Bool := window.task(lambda () -> 1).cancelled()
again: Bool := GUI.checkpoint()
tabs.setVisible(true)
group.setBackground("white")
bar.setEnabled(false)
when.setForeground("black")
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
		`form.grid()`,
		`form.grid("2")`,
		`form.scroll("8")`,
		`button.setAlign()`,
		`button.setAlign(1)`,
		`button.setMinWidth()`,
		`button.setMinWidth("8")`,
		`button.setSpan()`,
		`button.setPreferredWidth(1, 2)`,
		`form.label()`,
		`form.label(1)`,
		`form.button()`,
		`form.textInput(1)`,
		`form.checkbox()`,
		`form.checkbox("x", "yes")`,
		`form.table()`,
		`form.dropdown()`,
		`form.image()`,
		`form.image(1)`,
		`status.setWrap()`,
		`status.setWrap("yes")`,
		`button.setWrap(true)`,
		`form.image("a.png").setSize("big")`,
		`form.tabs(1)`,
		`form.tabs().page()`,
		`form.tabs().select()`,
		`form.tabs().onChange(lambda () -> write(1))`,
		`form.radioGroup().option()`,
		`form.radioGroup().onChange(lambda (index: Int, text: String) -> write(index))`,
		`form.radioGroup().select("0")`,
		`form.progress("10")`,
		`form.progress().setValue()`,
		`form.date("2024")`,
		`form.date().setDate(2024, 2)`,
		`form.date().onChange(lambda (year: Int) -> write(year))`,
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
		`window.menu()`,
		`window.menu(1)`,
		`file: Menu := window.menu("File")
file.item()`,
		`file: Menu := window.menu("File")
file.item(1)`,
		`file: Menu := window.menu("File")
file.separator(1)`,
		`file: Menu := window.menu("File")
item: MenuItem := file.item("Save")
item.onClick(lambda (x: Int) -> write(x))`,
		`file: Menu := window.menu("File")
item: MenuItem := file.item("Save")
item.setEnabled("no")`,
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
		`button.setFont("x")`,
		`status.setSize(1)`,
		`status.setBold("yes")`,
		`button.setSize("large")`,
		`GUI.setClipboard()`,
		`GUI.clipboard("x")`,
		`button.setStyle("x")`,
		`window.setTheme("dark")`,
		`input.setPlaceholderColor("gray")`,
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, guiPreamble+source+"\n"))
	}
}
