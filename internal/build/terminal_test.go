package build

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/evaluator"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
)

// terminalProgram exercises every deterministic Terminal operation. Under a
// test both standard streams are pipes, so the capability queries report a
// non-terminal and styling degrades to plain text in both backends.
const terminalProgram = `bring Terminal
from Terminal bring TerminalError

Point: Class<> := {
    structure: Attributes := (x: Int, y: Int)
}

Badge: Class<> := {
    structure: Attributes := (label: String)

    CStr: Function := (
    ) -> String {
        return "Badge({attribute.label})"
    }
}

write("start")
Terminal.emit(["Ali", "95", "Passed"])
Terminal.emit(["Ali", "95", "Passed"], " | ")
Terminal.emit(parts: ["Loading"], ending: "")
Terminal.emit(parts: ["..."], separator: "", ending: "\n")
nothing: List<String> := []
Terminal.emit(nothing)
Terminal.emit(nothing, ", ", "<end>\n")
Terminal.emit(["ç", "ğ", "🙂"], "·", "")
Terminal.emit(["", "line\nbreak", ""], "|")
Terminal.error("failure", "")
Terminal.error("Configuration not found")
Terminal.error(text: "Unicode ✓ çğ", ending: "!\n")
Terminal.flush()
Terminal.flush()
write(str(Terminal.isInteractive()) + " " + str(Terminal.supportsColor()))
width: Int? := Terminal.width()
height: Int? := Terminal.height()
write(str(width == null) + " " + str(height == null))
write(Terminal.style(text: "Success", foreground: "green", bold: true) + "|" + Terminal.style("plain") + "|" + Terminal.style("x", "red", "white", true, true))
for name in ["purple", "Red", ""] {
    attempt {
        Terminal.style(text: "x", foreground: name)
    } except TerminalError as error {
        write(error.message)
    }
}
attempt {
    Terminal.style(text: "x", background: "bright-red")
} except TerminalError as error {
    write(error.message)
}
grades: Pair<String, List<Int>> := {"Ali": [90, 95, 100], "Ayşe": [85, 91, 97]}
Terminal.pretty(grades)
empty: Pair<String, Int> := {}
Terminal.pretty(empty)
Terminal.pretty(nothing)
Terminal.pretty(["a\"b", "line\nbreak"])
Terminal.pretty(42)
Terminal.pretty(2.5)
Terminal.pretty(true)
Terminal.pretty("plain text")
optional: List<Real?> := [1.5, null]
Terminal.pretty(optional)
nested: List<List<String>> := [["x"], []]
Terminal.pretty(nested)
flags: Pair<Int, Bool> := {1: true, 2: false}
Terminal.pretty(flags)
deep: Pair<String, Pair<String, List<Int>>> := {"outer": {"inner": [1]}}
Terminal.pretty(deep)
Terminal.pretty(Point(x: 1, y: 2))
Terminal.pretty([Point(x: 1, y: 2)])
badge: Badge := Badge(label: "gold")
write(badge)
Terminal.pretty(badge)
Terminal.pretty([badge])
write([badge])
write("end")
`

const terminalExpectedStdout = "start\n" +
	"Ali 95 Passed\n" +
	"Ali | 95 | Passed\n" +
	"Loading...\n" +
	"\n" +
	"<end>\n" +
	"ç·ğ·🙂|line\nbreak|\n" +
	"false false\n" +
	"true true\n" +
	"Success|plain|x\n" +
	"unsupported foreground color \"purple\"; use default, black, red, green, yellow, blue, magenta, cyan, or white\n" +
	"unsupported foreground color \"Red\"; use default, black, red, green, yellow, blue, magenta, cyan, or white\n" +
	"unsupported foreground color \"\"; use default, black, red, green, yellow, blue, magenta, cyan, or white\n" +
	"unsupported background color \"bright-red\"; use default, black, red, green, yellow, blue, magenta, cyan, or white\n" +
	"{\n    \"Ali\": [\n        90,\n        95,\n        100\n    ],\n    \"Ayşe\": [\n        85,\n        91,\n        97\n    ]\n}\n" +
	"{}\n" +
	"[]\n" +
	"[\n    \"a\\\"b\",\n    \"line\\nbreak\"\n]\n" +
	"42\n" +
	"2.5\n" +
	"true\n" +
	"plain text\n" +
	"[\n    1.5,\n    null\n]\n" +
	"[\n    [\n        \"x\"\n    ],\n    []\n]\n" +
	"{\n    1: true,\n    2: false\n}\n" +
	"{\n    \"outer\": {\n        \"inner\": [\n            1\n        ]\n    }\n}\n" +
	"<Point>\n" +
	"[\n    <Point>\n]\n" +
	// A top-level Class value is exactly what write shows, including its own
	// CStr; inside a collection it is what str shows inside a collection.
	"Badge(gold)\n" +
	"Badge(gold)\n" +
	"[\n    <Badge>\n]\n" +
	"[<Badge>]\n" +
	"end\n"

const terminalExpectedStderr = "failureConfiguration not found\nUnicode ✓ çğ!\n"

// runTerminalEvaluator runs one program in the evaluator with separate or
// shared output writers.
func runTerminalEvaluator(t *testing.T, source string, output, errorOutput *bytes.Buffer) {
	t.Helper()
	workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": source})
	frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
	if frontend.HasErrors() {
		t.Fatalf("frontend diagnostics: %+v", frontend.Diagnostics)
	}
	lowered := lowering.LowerCompilation(frontend)
	if lowered.HasErrors() {
		t.Fatalf("lowering diagnostics: %+v", lowered.Diagnostics)
	}
	session := evaluator.New(bufio.NewReader(strings.NewReader("")), output, t.TempDir())
	session.ErrorOutput = errorOutput
	if result := session.Execute(lowered.Compilation, 0); result.Failure != nil {
		t.Fatalf("evaluator raised %s: %s", result.Failure.Name, result.Failure.Message)
	}
}

func TestTerminalNativeAndEvaluatorAgree(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": terminalProgram})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 {
		t.Fatalf("native program failed: code=%d stderr=%q", code, stderr)
	}
	if stdout != terminalExpectedStdout {
		t.Fatalf("native stdout mismatch:\n got: %q\nwant: %q", stdout, terminalExpectedStdout)
	}
	if stderr != terminalExpectedStderr {
		t.Fatalf("native stderr mismatch:\n got: %q\nwant: %q", stderr, terminalExpectedStderr)
	}
	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, terminalProgram, &output, &errorOutput)
	if output.String() != stdout {
		t.Fatalf("evaluator stdout differs from native:\n evaluator: %q\n    native: %q", output.String(), stdout)
	}
	if errorOutput.String() != stderr {
		t.Fatalf("evaluator stderr differs from native:\n evaluator: %q\n    native: %q", errorOutput.String(), stderr)
	}
}

func TestTerminalMinimalStreamsSeparate(t *testing.T) {
	source := "bring Terminal\nTerminal.emit([\"a\", \"b\"], \"|\", \"\")\nTerminal.error(\"failure\", \"\")\n"
	directory := writeSources(t, map[string]string{"main.ahd": source})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stdout != "a|b" || stderr != "failure" {
		t.Fatalf("native: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, source, &output, &errorOutput)
	if output.String() != "a|b" || errorOutput.String() != "failure" {
		t.Fatalf("evaluator: stdout=%q stderr=%q", output.String(), errorOutput.String())
	}
}

// Terminal.error writes pending standard output first, so when both streams
// share one destination the text appears in program order even though the
// native standard output is buffered.
func TestTerminalErrorKeepsProgramOrderOnASharedDestination(t *testing.T) {
	source := `bring Terminal
write("one")
Terminal.error("two")
Terminal.emit(["three"])
Terminal.error("four", "")
Terminal.emit(["five"])
`
	const want = "one\ntwo\nthree\nfourfive\n"
	directory := writeSources(t, map[string]string{"main.ahd": source})
	executable := filepath.Join(t.TempDir(), "program")
	if _, result := BuildProgram(filepath.Join(directory, "main.ahd"), executable); result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	var combined bytes.Buffer
	command := exec.Command(executable)
	command.Stdout = &combined
	command.Stderr = &combined
	if err := command.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			t.Fatalf("program exited %d: %q", exit.ExitCode(), combined.String())
		}
		t.Fatal(err)
	}
	if combined.String() != want {
		t.Fatalf("native combined output %q, want %q", combined.String(), want)
	}
	var shared bytes.Buffer
	runTerminalEvaluator(t, source, &shared, &shared)
	if shared.String() != want {
		t.Fatalf("evaluator combined output %q, want %q", shared.String(), want)
	}
}

func TestTerminalUncaughtStyleErrorIsAhdCodeLevel(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "bring Terminal\nwrite(Terminal.style(text: \"x\", foreground: \"orange\"))\n"})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 1 || stdout != "" || !strings.HasPrefix(stderr, "TerminalError: unsupported foreground color \"orange\"") || strings.Contains(stderr, "goroutine ") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}
