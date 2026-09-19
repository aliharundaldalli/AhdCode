package build

import (
	"os"
	"path/filepath"
	"testing"
)

// TestV190GUIColorsExampleRuns drives the tracked colors example headless.
// Widget ids follow creation order: customer 5, terms 6, check 8, save 9.
// The first click on the still-disabled Save is ignored.
func TestV190GUIColorsExampleRuns(t *testing.T) {
	useHeadlessGUI(t, `[{"event":"click","widget":9},{"event":"click","widget":8},`+
		`{"event":"type","widget":5,"text":"Ayşe Yılmaz"},{"event":"toggle","widget":6},{"event":"click","widget":8},`+
		`{"event":"click","widget":9},{"event":"click","widget":9},{"event":"key","key":"Escape"}]`)
	entry, err := filepath.Abs(filepath.Join("..", "..", "examples", "v1.9", "gui_colors", "main.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRunIn(t, entry, t.TempDir())
	if code != 0 || stdout != "saved Ayşe Yılmaz\n" {
		t.Fatalf("exit %d (stderr %q): %q", code, stderr, stdout)
	}
}

// TestV190InteractivePlotExampleRuns shows the tracked chart in a headless
// viewer, then checks the saved file and that no preview was left behind.
func TestV190InteractivePlotExampleRuns(t *testing.T) {
	report, previews := usePlotViewer(t, `[{"action":"rotateRight"},{"action":"zoom","factor":2,"x":400,"y":300},{"action":"close"}]`)
	entry, err := filepath.Abs(filepath.Join("..", "..", "examples", "v1.9", "interactive_plot", "main.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	stdout, stderr, code := buildAndRunIn(t, entry, directory)
	want := "Scroll to zoom, drag to pan, Q/E to rotate, R to reset, Escape to close.\n" +
		"Saved result.png, the chart as it was defined: the viewer changed nothing.\n"
	if code != 0 || stdout != want {
		t.Fatalf("exit %d (stderr %q): %q", code, stderr, stdout)
	}
	if info, err := os.Stat(filepath.Join(directory, "result.png")); err != nil || info.Size() == 0 {
		t.Fatal("result.png was not saved")
	}
	if states := readReport(t, report); states[1].Rotation != 270 {
		t.Fatalf("viewer states %+v", states)
	}
	if left := leftoverPreviews(t, previews); len(left) != 0 {
		t.Fatalf("previews left behind: %v", left)
	}
	if _, err := os.Stat(filepath.Join("..", "..", "examples", "v1.9", "interactive_plot", "result.png")); !os.IsNotExist(err) {
		t.Fatal("the example wrote into the repository")
	}
}
