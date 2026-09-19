package build

import (
	"bytes"
	"path/filepath"
	"testing"
)

// guiStyleProgram exercises every v1.9 color setter and the enabled state.
// Widget ids follow creation order: root 1, label 2, input 3, checkbox 4,
// button 5. The script first uses the disabled widgets, which must ignore
// the user, then presses Enter, whose callback enables them again.
const guiStyleProgram = `bring GUI
from GUI bring (Window, Container, Label, Button, TextInput, Checkbox, GUIError)

window: Window := GUI.window(title: "Styles", width: 420, height: 260)
window.setBackground("#F0F4F8")
form: Container := window.column()
form.setBackground("white")
title: Label := form.label("Başlık")
title.setForeground("blue")
title.setBackground("#FFFF0080")
name: TextInput := form.textInput(placeholder: "name")
name.setForeground("#202020")
name.setBackground(color: "yellow")
paid: Checkbox := form.checkbox("Paid")
paid.setForeground("red")
paid.setBackground("gray")
save: Button := form.button("Save")
save.setForeground("white")
save.setBackground("#0066CCFF")
write("enabled by default {save.isEnabled()} {name.isEnabled()} {paid.isEnabled()}")
for bad in ["Red", "purple", "#fff", ""] {
    attempt {
        save.setBackground(bad)
    } except GUIError as error {
        write(error.message)
    }
}
save.setEnabled(false)
name.setEnabled(enabled: false)
paid.setEnabled(false)
name.setText("programmatic")
paid.setChecked(true)
write("disabled {save.isEnabled()} {name.isEnabled()} {paid.isEnabled()} [{name.text()}] {paid.checked()}")

saveClicked: Function := () -> Nothing {
    name: Global TextInput
    paid: Global Checkbox
    write("saved [{name.text()}] {paid.checked()}")
}

keys: Function := (key: String) -> Nothing {
    save: Global Button
    name: Global TextInput
    paid: Global Checkbox
    window: Global Window
    if key == "Enter" {
        save.setEnabled(true)
        name.setEnabled(true)
        paid.setEnabled(true)
        write("enabled again {save.isEnabled()} {name.isEnabled()} {paid.isEnabled()}")
    }
    if key == "Escape" {
        window.close()
    }
}

save.onClick(saveClicked)
window.onKey(keys)
window.wait()
write("after close {save.isEnabled()} [{name.text()}] {paid.checked()}")
attempt {
    save.setBackground("red")
} except GUIError as error {
    write("after close: " + error.message)
}
attempt {
    save.setEnabled(false)
} except GUIError as error {
    write("after close: " + error.message)
}
`

const guiStyleScript = `[{"event":"click","widget":5},{"event":"type","widget":3,"text":"ignored"},{"event":"toggle","widget":4},` +
	`{"event":"key","key":"Enter"},{"event":"type","widget":3,"text":"!"},{"event":"toggle","widget":4},` +
	`{"event":"click","widget":5},{"event":"key","key":"Escape"}]`

const guiStyleExpected = `enabled by default true true true
unsupported background color "Red"; use black, white, red, green, blue, yellow, cyan, magenta, gray, #RRGGBB, or #RRGGBBAA
unsupported background color "purple"; use black, white, red, green, blue, yellow, cyan, magenta, gray, #RRGGBB, or #RRGGBBAA
unsupported background color "#fff"; use black, white, red, green, blue, yellow, cyan, magenta, gray, #RRGGBB, or #RRGGBBAA
unsupported background color ""; use black, white, red, green, blue, yellow, cyan, magenta, gray, #RRGGBB, or #RRGGBBAA
disabled false false false [programmatic] true
enabled again true true true
saved [programmatic!] false
after close true [programmatic!] false
after close: the Window is closed
after close: the Window is closed
`

func TestGUIStylesAndEnabledStateNativeAndEvaluatorAgree(t *testing.T) {
	useHeadlessGUI(t, guiStyleScript)
	directory := writeSources(t, map[string]string{"main.ahd": guiStyleProgram})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stdout != guiStyleExpected {
		t.Fatalf("native (exit %d, stderr %q):\n have %q\n want %q", code, stderr, stdout, guiStyleExpected)
	}
	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, guiStyleProgram, &output, &errorOutput)
	if output.String() != guiStyleExpected {
		t.Fatalf("evaluator:\n have %q\n want %q\n stderr %q", output.String(), guiStyleExpected, errorOutput.String())
	}
}
