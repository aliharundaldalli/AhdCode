package build

import (
	"path/filepath"
	"testing"
)

// The v1.5.0 Terminal demo builds with no external runtime requirement and,
// with both standard streams redirected as they are under a test, prints the
// documented non-terminal output.
func TestV150TerminalDemoRuns(t *testing.T) {
	path, err := filepath.Abs("../../examples/v1.5/terminal_demo/main.ahd")
	if err != nil {
		t.Fatal(err)
	}
	result := Compile(path)
	if result.HasErrors() {
		t.Fatalf("terminal demo:\n%s", diagnosticText(result.Diagnostics))
	}
	program := result.Program
	if program.RequiresLatex || program.RequiresPlot || program.RequiresNumeric || program.RequiresSQLite ||
		program.RequiresMySQL || program.RequiresCodes || program.RequiresWebSocket || program.RequiresPostgreSQL {
		t.Fatalf("the terminal demo acquired an external runtime requirement: %+v", program)
	}
	stdout, stderr, code := buildAndRun(t, path, "")
	const wantStdout = "1. write works exactly as before\n" +
		"2. emit joins Strings\n" +
		"3. Ali | 95 | Passed\n" +
		"4. Loading...\n" +
		"6. interactive: false\n" +
		"7. size: unknown, because standard output is not a terminal\n" +
		"8. color: false\n" +
		"9. success / warning / error\n" +
		"10. Unicode: çğıöşü İstanbul 🙂\n" +
		"11. grades:\n" +
		"{\n    \"Ali\": [\n        90,\n        95,\n        100\n    ],\n    \"Ayşe\": [\n        85,\n        91,\n        97\n    ]\n}\n"
	if code != 0 || stdout != wantStdout || stderr != "5. this line goes to standard error\n" {
		t.Fatalf("terminal demo: code=%d\nstdout %q\nwant   %q\nstderr %q", code, stdout, wantStdout, stderr)
	}
}
