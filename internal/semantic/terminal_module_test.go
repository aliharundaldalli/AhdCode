package semantic

import (
	"strings"
	"testing"
)

const terminalPreamble = "bring Terminal\nfrom Terminal bring TerminalError\n\n"

func TestTerminalModuleRegistered(t *testing.T) {
	module, ok := StandardModuleInterfaces()["Terminal"]
	if !ok {
		t.Fatal("Terminal module not registered")
	}
	if module.ModuleID != "builtin:Terminal" {
		t.Fatalf("Terminal identity = %q", module.ModuleID)
	}
	want := []string{"TerminalError", "emit", "error", "flush", "height", "isInteractive", "pretty", "style", "supportsColor", "width"}
	if strings.Join(module.ExportNames, ",") != strings.Join(want, ",") {
		t.Fatalf("Terminal exports %v, want exactly %v", module.ExportNames, want)
	}
	// The v1.5.0 surface deliberately has no print clone, color switches,
	// cursor or keyboard control, progress output, or logging.
	for _, name := range []string{"print", "println", "write", "read", "take", "enableColor", "disableColor", "setColor",
		"clear", "cursor", "moveCursor", "readKey", "rawMode", "progress", "spinner", "menu", "log", "rgb", "color256"} {
		if module.Exports[name] != nil {
			t.Fatalf("Terminal must not export %q", name)
		}
	}
}

func TestTerminalModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, terminalPreamble+`Point: Class<> := {
    structure: Attributes := (x: Int, y: Int)
}

name := "Ali"
score := 95
Terminal.emit(["Ali", "95", "Passed"])
Terminal.emit([name, str(score)], " | ")
Terminal.emit([name, str(score)], " | ", "")
Terminal.emit(parts: ["{name}: {score}"])
Terminal.emit(parts: ["Loading"], ending: "")
Terminal.emit(ending: "\n", separator: ", ", parts: ["a", "b"])
none: List<String> := []
Terminal.emit(none)
Terminal.error("Configuration not found")
Terminal.error("partial", "")
Terminal.error(text: "failure", ending: "!\n")
Terminal.flush()
interactive: Bool := Terminal.isInteractive()
color: Bool := Terminal.supportsColor()
width: Int? := Terminal.width()
height: Int? := Terminal.height()
if width != null {
    columns: Local Int := width
    write(str(columns))
}
styled: String := Terminal.style("OK")
bright: String := Terminal.style("OK", "green", "default", true, false)
named: String := Terminal.style(text: "Warning", foreground: "yellow", underline: true)
nested: String := Terminal.style(text: "Status: " + named, bold: true)
Terminal.pretty(42)
Terminal.pretty(2.5)
Terminal.pretty(true)
Terminal.pretty("text")
Terminal.pretty([1, 2, 3])
Terminal.pretty({"Ali": [90, 95, 100]})
Terminal.pretty(Point(x: 1, y: 2))
maybe: String? := null
Terminal.pretty(maybe)
optional: List<Real?> := [1.5, null]
Terminal.pretty(optional)
attempt {
    Terminal.style(text: "x", foreground: "purple")
} except TerminalError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

func TestTerminalSelectiveAndAliasedImports(t *testing.T) {
	for _, source := range []string{
		"from Terminal bring emit\nemit([\"a\"])\n",
		"from Terminal bring (emit, style, pretty, TerminalError)\nemit([style(\"a\")])\npretty([1])\n",
		"bring Terminal as T\nT.emit([\"a\"], \"-\")\nT.flush()\n",
	} {
		requireSemanticClean(t, analyzeWithStandardModules(t, source))
	}
}

// Static mistakes are compiler diagnostics, never a runtime TerminalError.
// Terminal.emit takes exactly List<String>: nothing is converted for the
// caller, and there is no variadic call form.
func TestTerminalRejectsStaticMistakes(t *testing.T) {
	for _, source := range []string{
		`Terminal.emit("Ali")`,
		`Terminal.emit("Ali", "95")`,
		`Terminal.emit([1, 2])`,
		`Terminal.emit(["a"], 1)`,
		`Terminal.emit(["a"], " ", "\n", "extra")`,
		`Terminal.emit(parts: ["a"], separator: 1)`,
		`Terminal.emit(values: ["a"])`,
		`Terminal.emit()`,
		"values: List<String?> := [\"a\", null]\nTerminal.emit(values)",
		"parts: List<String>? := null\nTerminal.emit(parts)",
		`Terminal.error(1)`,
		`Terminal.error()`,
		`Terminal.error("a", 1)`,
		`Terminal.flush(1)`,
		`Terminal.isInteractive(true)`,
		`Terminal.width(80)`,
		`columns: Int := Terminal.width()`,
		`rows: Int := Terminal.height()`,
		`Terminal.supportsColor("stdout")`,
		`Terminal.style(1)`,
		`Terminal.style()`,
		`Terminal.style("x", "red", "blue", "yes")`,
		`Terminal.style(text: "x", color: "red")`,
		`count: Int := Terminal.style("x")`,
		`Terminal.pretty()`,
		`Terminal.pretty(1, 2)`,
		`Terminal.pretty(value: 1)`,
		`Terminal.pretty(between(1, 3))`,
		`printer := Terminal.pretty`,
		`result: Int := Terminal.pretty(1)`,
		`Terminal.print("x")`,
		`Terminal.enableColor()`,
		`Terminal.clear()`,
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, terminalPreamble+source+"\n"))
	}
}
