package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestV180SimpleLedgerExampleRuns drives the tracked GUI ledger headless: an
// empty name, an empty (so invalid) amount, and two valid entries, the second
// marked as paid, then Escape. The rows reach SQLite, and a second run finds
// them. Widget ids follow creation order: customer 3, amount 5, paid 7, save 8.
func TestV180SimpleLedgerExampleRuns(t *testing.T) {
	sqliteHelperForTest(t)
	useHeadlessGUI(t, `[{"event":"click","widget":8},`+
		`{"event":"type","widget":3,"text":"Ayşe Yılmaz"},{"event":"click","widget":8},`+
		`{"event":"type","widget":5,"text":"12.5"},{"event":"click","widget":8},`+
		`{"event":"type","widget":3,"text":"Ali"},{"event":"type","widget":5,"text":"7.25"},{"event":"toggle","widget":7},{"event":"click","widget":8},`+
		`{"event":"key","key":"Escape"}]`)
	entry, err := filepath.Abs(filepath.Join("..", "..", "examples", "v1.8", "simple_ledger", "main.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	executable, result := BuildProgram(entry, filepath.Join(t.TempDir(), "ledger"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	directory := t.TempDir()
	first := strings.Join([]string{
		"The customer name is required.",
		"The amount must be a number such as 125.50.",
		"Saved #1: Ayşe Yılmaz, 12.5 (entries: 1)",
		"Saved #2: Ali, 7.25 (entries: 2)",
	}, "\n") + "\n"
	out, errorOutput, code := runIn(t, executable, directory)
	if code != 0 || out != first {
		t.Fatalf("first run exit %d (stderr %q)\n want:\n%s\n have:\n%s", code, errorOutput, first, out)
	}
	out, errorOutput, code = runIn(t, executable, directory)
	if code != 0 || !strings.Contains(out, "Saved #3: Ayşe Yılmaz, 12.5 (entries: 3)") {
		t.Fatalf("second run exit %d (stderr %q): %s", code, errorOutput, out)
	}
	if _, err := os.Stat(filepath.Join(directory, "ledger.db")); err != nil {
		t.Fatalf("ledger.db missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join("..", "..", "examples", "v1.8", "simple_ledger", "ledger.db")); !os.IsNotExist(err) {
		t.Fatal("the example wrote ledger.db into the repository")
	}
}

// TestV180TurtleEventsExampleRuns replays arrow keys, a click, and S against the
// tracked Turtle example: each arrow draws 20 units, and nothing reads stdin.
func TestV180TurtleEventsExampleRuns(t *testing.T) {
	useHeadlessGraphics(t)
	t.Setenv("AHDCODE_GRAPHICS_HEADLESS_EVENTS", `[{"event":"key","key":"ArrowUp"},{"event":"key","key":"ArrowRight"},`+
		`{"event":"click","x":-100,"y":-50},{"event":"key","key":"ArrowDown"},{"event":"key","key":"X"},{"event":"key","key":"S"}]`)
	entry, err := filepath.Abs(filepath.Join("..", "..", "examples", "v1.8", "turtle_events", "main.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	executable, result := BuildProgram(entry, filepath.Join(t.TempDir(), "turtle"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	directory := t.TempDir()
	want := strings.Join([]string{
		"Arrow keys draw 20 units, a click moves the Turtle, S saves, closing the window ends.",
		"ArrowUp: (0.0, 20.0)",
		"ArrowRight: (20.0, 20.0)",
		"Moved to (-100.0, -50.0)",
		"ArrowDown: (-100.0, -70.0)",
		"Saved turtle-events.png and turtle-events.svg",
		"Window closed.",
	}, "\n") + "\n"
	out, errorOutput, code := runIn(t, executable, directory)
	if code != 0 || out != want {
		t.Fatalf("exit %d (stderr %q)\n want:\n%s\n have:\n%s", code, errorOutput, want, out)
	}
	for _, name := range []string{"turtle-events.png", "turtle-events.svg"} {
		if _, err := os.Stat(filepath.Join(directory, name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}
}
