package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The Plot viewer's Save asks the ahdgui helper for the save dialog, so a
// program that uses Plot needs the ahdgui discovery hint even when it opens no
// window of its own. Without it, a Plot-only program compiled into a temporary
// directory left the viewer with no dialog helper and its Save reported the
// GUI helper as not installed.

const plotOnlyProgram = `bring Plot
from Plot bring (Chart)

chart: Chart := Plot.line(x: [1.0, 2.0, 3.0], y: [2.0, 4.0, 9.0]).title("Save")
chart.show()
`

const guiOnlyProgram = `bring GUI
from GUI bring (Window)

window: Window := GUI.window(title: "Window", width: 320, height: 200)
window.wait()
`

const plainProgram = `write("no helper")
`

// useStubHelper puts a file named like the helper in a directory of its own and
// points the helper's override variable at it, so the hint resolves to a known
// place on every machine.
func useStubHelper(t *testing.T, name, override string) string {
	t.Helper()
	directory := t.TempDir()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("stub"), 0o700); err != nil {
		t.Fatalf("could not write the stub helper: %v", err)
	}
	t.Setenv(override, path)
	return directory
}

func guiRuntimeHint(t *testing.T, source string) string {
	t.Helper()
	directory := writeSources(t, map[string]string{"main.ahd": source})
	result := Compile(filepath.Join(directory, "main.ahd"))
	if result.HasErrors() || result.Program == nil {
		t.Fatalf("compiling: %s", diagnosticText(result.Diagnostics))
	}
	program := configureGUIRuntime(result.Program)
	for _, file := range program.Files {
		if file.Name == "ahdcode_gui_runtime_hint.go" {
			return file.Content
		}
	}
	return ""
}

func TestPlotProgramCarriesTheGUIHelperHintForTheViewersSaveDialog(t *testing.T) {
	root := useStubHelper(t, "ahdgui", "AHDCODE_GUI_RUNTIME")

	hint := guiRuntimeHint(t, plotOnlyProgram)
	if hint == "" {
		t.Fatal("a program that uses Plot carries no ahdgui hint, so the viewer's Save has no dialog helper")
	}
	if !strings.Contains(hint, root) {
		t.Fatalf("the hint does not name the helper directory %s: %s", root, hint)
	}

	if guiRuntimeHint(t, guiOnlyProgram) == "" {
		t.Fatal("a program that uses GUI carries no ahdgui hint")
	}
	if hint := guiRuntimeHint(t, plainProgram); hint != "" {
		t.Fatalf("a program that uses neither GUI nor Plot carries an ahdgui hint: %s", hint)
	}
}
