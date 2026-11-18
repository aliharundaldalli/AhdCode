package build

import (
	"path/filepath"
	"testing"
)

// guiLayoutProgram uses the grid, scroll, alignment, and sizing members and
// rejects the values the runtime refuses. Geometry is covered by the helper
// tests; this program checks that native code and the evaluator agree.
const guiLayoutProgram = `bring GUI
from GUI bring (Window, Container, Label, Button, TextInput, GUIError)

window: Window := GUI.window("Layout", 420, 280)
page: Container := window.scroll(spacing: 8, padding: 12)
form: Container := page.grid(columns: 2, columnGap: 12, rowGap: 8)
form.label("Name")
name: TextInput := form.textInput()
name.setAlign("center", "start")
name.setMinWidth(80)
name.setPreferredWidth(120)
name.setSpan(1, 1)
extra: Button := form.button("Extra")
extra.setVisible(false)
nested: Container := form.column()
nested.setSpan(2, 1)
nested.label("Inside")
wide: Label := page.label("Wide")
wide.setMinWidth(640)
attempt {
    form.grid(0)
    write("accepted columns")
} except GUIError as error {
    write(error.message)
}
attempt {
    name.setAlign("middle")
    write("accepted align")
} except GUIError as error {
    write(error.message)
}
attempt {
    name.setMinWidth(-1)
    write("accepted width")
} except GUIError as error {
    write(error.message)
}
window.wait()
write("ready")
`

func TestGUILayoutNativeAndEvaluatorAgree(t *testing.T) {
	useHeadlessGUI(t, "")
	assertGUI(t, guiLayoutProgram, "a grid needs between 1 and 32 columns\nalignment must be \"start\", \"center\", \"end\", or \"stretch\"\nminimum width must be between 0 and 4096\nready\n")
	directory := filepath.Join("..", "..", "examples", "v2.9", "layout", "main.ahd")
	stdout, stderr, code := buildAndRun(t, directory, "")
	if code != 0 || stdout != "" {
		t.Fatalf("example exit %d stdout %q stderr %q", code, stdout, stderr)
	}
}
