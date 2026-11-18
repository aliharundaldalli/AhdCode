package build

import (
	"bytes"
	"path/filepath"
	"testing"
)

// guiMenuProgram builds a menu, rejects a duplicate shortcut, and lets a
// focused TextInput keep Command/Ctrl+C. Widget ids follow creation order:
// File 1, Save 2, Bold 3, separator 4, Edit 5, Copy 6, Drop 7, column 8,
// TextInput 9, Label 10.
const guiMenuProgram = `bring GUI
from GUI bring (Window, Menu, MenuItem, TextInput, Label, GUIError)

log: String := ""
window: Window := GUI.window("Menu", 480, 280)
file: Menu := window.menu("File")
save: MenuItem := file.item("Save", "S")
bold: MenuItem := file.checkItem("Bold", "", false)
file.separator()
edit: Menu := file.menu("Edit")
copy: MenuItem := edit.item("Copy", "C")
drop: MenuItem := file.item("Drop", "D")
drop.setEnabled(false)
form := window.column()
name: TextInput := form.textInput()
status: Label := form.label("ready")

onSave: Function := () -> Nothing {
    log: Global String
    name: Global TextInput
    status: Global Label
    log = log + "save:" + name.text() + ";"
    status.setText("saved")
}
onBold: Function := () -> Nothing {
    log: Global String
    bold: Global MenuItem
    if bold.checked() {
        log = log + "bold;"
    } else {
        log = log + "plain;"
    }
}
onCopy: Function := () -> Nothing {
    log: Global String
    log = log + "copy;"
}
onDrop: Function := () -> Nothing {
    log: Global String
    log = log + "drop;"
}
save.onClick(onSave)
bold.onClick(onBold)
copy.onClick(onCopy)
drop.onClick(onDrop)
attempt {
    file.item("Also", "s")
    write("accepted")
} except GUIError as error {
    write(error.message)
}
window.wait()
write(log)
write(name.text())
write(GUI.clipboard())
write(status.text())
if bold.checked() {
    write("yes")
} else {
    write("no")
}
`

const guiMenuScript = `[{"event":"type","widget":9,"text":"Ali"},{"event":"edit","widget":9,"key":"selectAll"},` +
	`{"event":"focus","widget":9},{"event":"shortcut","key":"C"},{"event":"shortcut","key":"S"},` +
	`{"event":"shortcut","key":"D"},{"event":"menu","widget":3}]`

func TestGUIMenusNativeAndEvaluatorAgree(t *testing.T) {
	t.Run("actions", func(t *testing.T) {
		useHeadlessGUI(t, guiMenuScript)
		want := "shortcut \"S\" is already used\nsave:Ali;bold;\nAli\nAli\nsaved\nyes\n"
		assertGUI(t, guiMenuProgram, want)
	})
	t.Run("password", func(t *testing.T) {
		useHeadlessGUI(t, `[{"event":"type","widget":4,"text":"secret"},{"event":"edit","widget":4,"key":"selectAll"},{"event":"focus","widget":4},{"event":"shortcut","key":"C"}]`)
		program := `bring GUI
from GUI bring (Window, Menu, MenuItem, PasswordInput)
status: String := "ready"
GUI.setClipboard("")
window: Window := GUI.window("Secret", 420, 220)
file: Menu := window.menu("Edit")
copy: MenuItem := file.item("Copy", "C")
secret: PasswordInput := window.column().passwordInput()
onCopy: Function := () -> Nothing {
    status: Global String
    status = "copied"
}
copy.onClick(onCopy)
window.wait()
write(status)
write(secret.text())
write(GUI.clipboard())
`
		assertGUI(t, program, "ready\nsecret\n\n")
	})
	t.Run("hidden", func(t *testing.T) {
		useHeadlessGUI(t, `[{"event":"shortcut","key":"C"}]`)
		program := `bring GUI
from GUI bring (Window, Menu, MenuItem, TextInput)
status: String := "ready"
window: Window := GUI.window("Hide", 420, 220)
file: Menu := window.menu("Edit")
copy: MenuItem := file.item("Copy", "C")
name: TextInput := window.column().textInput()
name.setVisible(false)
onCopy: Function := () -> Nothing {
    status: Global String
    status = "menu"
}
copy.onClick(onCopy)
window.wait()
write(status)
write(name.text())
`
		assertGUI(t, program, "menu\n\n")
	})
}

func assertGUI(t *testing.T, program, want string) {
	t.Helper()
	directory := writeSources(t, map[string]string{"main.ahd": program})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stdout != want {
		t.Fatalf("native (exit %d, stderr %q):\n have %q\n want %q", code, stderr, stdout, want)
	}
	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, program, &output, &errorOutput)
	if output.String() != want {
		t.Fatalf("evaluator:\n have %q\n want %q\n stderr %q", output.String(), want, errorOutput.String())
	}
}
