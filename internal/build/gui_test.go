package build

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	guiHelperBuild sync.Once
	guiHelperFile  string
	guiHelperFail  string
)

// useHeadlessGUI builds the real ahdgui helper once and runs every Window in
// this test headless, replaying script as the user's input.
func useHeadlessGUI(t *testing.T, script string) {
	t.Helper()
	guiHelperBuild.Do(func() {
		directory, err := os.MkdirTemp("", "ahdgui-build-test-")
		if err != nil {
			guiHelperFail = err.Error()
			return
		}
		name := "ahdgui"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		guiHelperFile = filepath.Join(directory, name)
		source, _ := filepath.Abs(filepath.Join("..", "..", "cmd", "ahdgui"))
		command := exec.Command("go", "build", "-o", guiHelperFile, ".")
		command.Dir = source
		if output, err := command.CombinedOutput(); err != nil {
			guiHelperFail = err.Error() + "\n" + string(output)
		}
	})
	if guiHelperFail != "" {
		t.Fatalf("building ahdgui: %s", guiHelperFail)
	}
	t.Setenv("AHDCODE_GUI_RUNTIME", guiHelperFile)
	t.Setenv("AHDCODE_GUI_HEADLESS", "1")
	t.Setenv("AHDCODE_GUI_HEADLESS_EVENTS", script)
}

// guiFormProgram is a small data-entry form: typing, a Checkbox, a Button
// callback that reads both and updates a Label, a key callback, and the
// layout rules. Widget ids in the script follow creation order: root 1,
// label 2, input 3, row 4, checkbox 5, button 6, status 7.
const guiFormProgram = `bring GUI
from GUI bring (Window, Container, Label, Button, TextInput, Checkbox, GUIError)

window: Window := GUI.window(title: "Form", width: 420, height: 240)
form: Container := window.column()
form.label("Customer")
name: TextInput := form.textInput(placeholder: "name")
row: Container := form.row(spacing: 12)
paid: Checkbox := row.checkbox("Paid")
save: Button := row.button("Save")
status: Label := form.label("Ready")

saveClicked: Function := () -> Nothing {
    name: Global TextInput
    paid: Global Checkbox
    status: Global Label
    status.setText("Saved {name.text()} paid={paid.checked()}")
    write(status.text())
}

save.onClick(lambda () -> write("replaced"))
save.onClick(saveClicked)
window.onKey(lambda (key: String) -> write("key {key}"))
attempt {
    window.row()
} except GUIError as error {
    write("second root: " + error.message)
}
attempt {
    form.column(spacing: -1)
} except GUIError as error {
    write(error.message)
}
write("before wait [{name.text()}] {status.text()} {save.text()} {paid.checked()}")
window.wait()
write("after wait open={window.isOpen()} name={name.text()} paid={paid.checked()}")
attempt {
    status.setText("late")
} except GUIError as error {
    write("after close: " + error.message)
}
`

const guiFormScript = `[{"event":"type","widget":3,"text":"Ali"},{"event":"click","widget":6},{"event":"toggle","widget":5},` +
	`{"event":"type","widget":3,"text":" Veli"},{"event":"key","key":"Enter"},{"event":"click","widget":6}]`

const guiFormExpected = `second root: the Window already has its root Container; add further Containers to it
spacing and padding must be 0 or greater, not -1 and 0
before wait [] Ready Save false
Saved Ali paid=false
key Enter
Saved Ali Veli paid=true
after wait open=false name=Ali Veli paid=true
after close: the Window is closed
`

// turtleEventProgram drives a Turtle with Canvas.onKey and Canvas.onClick,
// the replacement for reading arrow keys from the terminal.
const turtleEventProgram = `bring Graphics
from Graphics bring (Canvas, Turtle)

canvas: Canvas := Graphics.open(400, 400)
pen: Turtle := canvas.turtle()

step: Function := (key: String) -> Nothing {
    pen: Global Turtle
    state key {
        condition "ArrowUp" {
            pen.setHeading(90)
        }
        condition "ArrowDown" {
            pen.setHeading(270)
        }
        condition "ArrowLeft" {
            pen.setHeading(180)
        }
        condition "ArrowRight" {
            pen.setHeading(0)
        }
        condition default {
            write("ignored {key}")
            return
        }
    }
    pen.forward(20)
    write("{key} -> ({pen.x()}, {pen.y()})")
}

jump: Function := (x: Real, y: Real) -> Nothing {
    pen: Global Turtle
    pen.penUp()
    pen.moveTo(x, y)
    pen.penDown()
    write("click ({x}, {y})")
}

canvas.onKey(step)
canvas.onClick(handler: jump)
canvas.wait()
write("after wait open={canvas.isOpen()}")
`

const turtleEventScript = `[{"event":"key","key":"ArrowUp"},{"event":"key","key":"ArrowRight"},{"event":"key","key":"Q"},` +
	`{"event":"click","x":-50,"y":25.5},{"event":"key","key":"ArrowDown"},{"event":"key","key":"ArrowLeft"}]`

const turtleEventExpected = `ArrowUp -> (0.0, 20.0)
ArrowRight -> (20.0, 20.0)
ignored Q
click (-50.0, 25.5)
ArrowDown -> (-50.0, 5.5)
ArrowLeft -> (-70.0, 5.5)
after wait open=false
`

func TestGUIAndCanvasEventsNativeAndEvaluatorAgree(t *testing.T) {
	cases := []struct {
		name, source, expected string
		setup                  func(t *testing.T)
	}{
		{"GUI form", guiFormProgram, guiFormExpected, func(t *testing.T) { useHeadlessGUI(t, guiFormScript) }},
		{"Turtle events", turtleEventProgram, turtleEventExpected, func(t *testing.T) {
			useHeadlessGraphics(t)
			t.Setenv("AHDCODE_GRAPHICS_HEADLESS_EVENTS", turtleEventScript)
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.setup(t)
			directory := writeSources(t, map[string]string{"main.ahd": testCase.source})
			stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
			if code != 0 || stdout != testCase.expected {
				t.Fatalf("native (exit %d, stderr %q):\n have %q\n want %q", code, stderr, stdout, testCase.expected)
			}
			var output, errorOutput bytes.Buffer
			runTerminalEvaluator(t, testCase.source, &output, &errorOutput)
			if output.String() != testCase.expected {
				t.Fatalf("evaluator:\n have %q\n want %q", output.String(), testCase.expected)
			}
		})
	}
}

// TestGUICallbackErrorPropagatesUnchanged checks that a callback's own error
// reaches the program as that error, not as a GUIError, and that the
// program's except clause sees it after the Window was released.
func TestGUICallbackErrorPropagatesUnchanged(t *testing.T) {
	useHeadlessGUI(t, `[{"event":"click","widget":2},{"event":"click","widget":2}]`)
	source := `bring GUI
from GUI bring (Window, Button, GUIError)

window: Window := GUI.window()
button: Button := window.column().button("Fail")
fail: Function := () -> Nothing {
    toss(ValueError("bad amount"))
}
button.onClick(fail)
attempt {
    window.wait()
} except GUIError as error {
    write("GUIError " + error.message)
} except ValueError as error {
    write("ValueError " + error.message)
}
write("open={window.isOpen()}")
`
	expected := "ValueError bad amount\nopen=false\n"
	directory := writeSources(t, map[string]string{"main.ahd": source})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stdout != expected {
		t.Fatalf("native (exit %d, stderr %q): %q", code, stderr, stdout)
	}
	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, source, &output, &errorOutput)
	if output.String() != expected {
		t.Fatalf("evaluator: %q", output.String())
	}
	if strings.Contains(stdout, "GUIError") {
		t.Fatal("a callback error became a GUIError")
	}
}
