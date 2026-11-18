package build

import (
	"bytes"
	"path/filepath"
	"testing"
)

// guiEditProgram drives TextInput through the helper: paste from the program
// clipboard, copy the result back, then undo and redo. Widget 2 is the field.
const guiEditProgram = `bring GUI
from GUI bring (Window, TextInput)

GUI.setClipboard("çağrı")
window: Window := GUI.window("Edit", 420, 240)
name: TextInput := window.column().textInput()
window.wait()
write(name.text())
write(GUI.clipboard())
`

const guiEditScript = `[{"event":"type","widget":2,"text":"Ali "},{"event":"edit","widget":2,"key":"paste"},` +
	`{"event":"edit","widget":2,"key":"selectAll"},{"event":"edit","widget":2,"key":"copy"},` +
	`{"event":"edit","widget":2,"key":"undo"},{"event":"edit","widget":2,"key":"redo"}]`

const guiEditExpected = "Ali çağrı\nAli çağrı\n"

func TestGUITextEditingNativeAndEvaluatorAgree(t *testing.T) {
	useHeadlessGUI(t, guiEditScript)
	directory := writeSources(t, map[string]string{"main.ahd": guiEditProgram})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stdout != guiEditExpected {
		t.Fatalf("native (exit %d, stderr %q):\n have %q\n want %q", code, stderr, stdout, guiEditExpected)
	}
	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, guiEditProgram, &output, &errorOutput)
	if output.String() != guiEditExpected {
		t.Fatalf("evaluator:\n have %q\n want %q\n stderr %q", output.String(), guiEditExpected, errorOutput.String())
	}
}
