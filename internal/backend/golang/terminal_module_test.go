package golang

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// The Terminal runtime is standard library only, so every generated program
// carries it without a vendored dependency tree, and each platform file is
// selected by its operating-system suffix and build constraint.
func TestTerminalRuntimeIsStandardLibraryOnly(t *testing.T) {
	allowed := map[string]bool{"os": true, "strconv": true, "strings": true, "syscall": true, "unsafe": true}
	constraints := map[string]string{
		terminalRuntimeFileName:        "",
		terminalDarwinRuntimeFileName:  "//go:build darwin",
		terminalLinuxRuntimeFileName:   "//go:build linux",
		terminalWindowsRuntimeFileName: "//go:build windows",
		terminalOtherRuntimeFileName:   "//go:build !darwin && !linux && !windows",
	}
	program := generate(t, "bring Terminal\nTerminal.emit([\"a\", \"b\"], \"|\", \"\")\n")
	if program.RequiresMySQL || program.RequiresCodes || program.RequiresWebSocket || program.RequiresPostgreSQL ||
		program.RequiresSQLite || program.RequiresLatex || program.RequiresPlot || program.RequiresNumeric {
		t.Fatalf("a Terminal program acquired an external runtime requirement: %+v", program)
	}
	seen := map[string]int{}
	for _, file := range program.Files {
		constraint, isTerminal := constraints[file.Name]
		if !isTerminal {
			continue
		}
		seen[file.Name]++
		parsed, err := parser.ParseFile(token.NewFileSet(), file.Name, file.Content, parser.ParseComments)
		if err != nil {
			t.Fatalf("%s is not valid Go: %v", file.Name, err)
		}
		if parsed.Name.Name != "main" {
			t.Fatalf("%s package clause not rewritten: %s", file.Name, parsed.Name.Name)
		}
		for _, spec := range parsed.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			if !allowed[path] {
				t.Fatalf("%s imports %s", file.Name, path)
			}
		}
		if constraint != "" && !strings.HasPrefix(file.Content, constraint+"\n") {
			t.Fatalf("%s does not start with %q", file.Name, constraint)
		}
		if constraint == "" && strings.Contains(file.Content, "//go:build") {
			t.Fatalf("%s must compile on every platform", file.Name)
		}
	}
	for name := range constraints {
		if seen[name] != 1 {
			t.Fatalf("%s emitted %d times, want 1", name, seen[name])
		}
	}
}

func TestTerminalCallsLowerToRuntimeFunctions(t *testing.T) {
	program := generate(t, `bring Terminal
Terminal.emit(["a"])
Terminal.emit(parts: ["a"], separator: "-")
Terminal.error("x")
Terminal.flush()
wide: Int? := Terminal.width()
tall: Int? := Terminal.height()
write(str(Terminal.isInteractive()) + str(Terminal.supportsColor()))
write(Terminal.style(text: "ok", foreground: "green"))
values: List<Int> := [1]
Terminal.pretty(values)
Terminal.pretty(1)
`)
	var source string
	for _, file := range program.Files {
		if file.Name == programFileName {
			source = file.Content
		}
	}
	for _, fragment := range []string{
		`AhdTerminalEmit(`, `" ", "\n")`, `"-", "\n")`, `AhdTerminalError(`, `AhdTerminalFlush(`, `AhdTerminalWidth()`,
		`AhdTerminalHeight()`, `AhdTerminalIsInteractive()`, `AhdTerminalSupportsColor()`, `AhdTerminalStyleText(`,
		`"green", "default", false, false)`, `AhdTerminalPrettyList[int64](AhdTerminalPrettyLeaf[int64](AhdStrInt))`,
		`AhdTerminalPretty(AhdStrInt(`,
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("generated program lacks %q:\n%s", fragment, source)
		}
	}
}
