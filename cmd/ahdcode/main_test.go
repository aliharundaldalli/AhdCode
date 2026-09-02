package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/source"
)

func write(t *testing.T, name, text string) string {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatalf("could not write %s: %v", name, err)
	}
	return path
}

func TestCommandDispatch(t *testing.T) {
	var out, errors bytes.Buffer
	if code := runWithIO(nil, bytes.NewBuffer(nil), &out, &errors); code != 0 {
		t.Fatalf("expected REPL exit 0; received %d", code)
	}
	if !strings.Contains(out.String(), "AhdCode v0.6.0\nahd> ") {
		t.Fatalf("REPL banner/prompt = %q", out.String())
	}
	if code := run([]string{"nonsense"}); code != 2 {
		t.Fatalf("expected usage exit 2 for an unknown command; received %d", code)
	}
	if code := run([]string{"version"}); code != 0 {
		t.Fatalf("expected version exit 0; received %d", code)
	}
	if code := run([]string{"help"}); code != 0 {
		t.Fatalf("expected help exit 0; received %d", code)
	}
	if code := run([]string{"build"}); code != 2 {
		t.Fatalf("expected usage exit 2 for build without an entry; received %d", code)
	}
	if code := run([]string{"build", "a.ahd", "-o"}); code != 2 {
		t.Fatalf("expected usage exit 2 for -o without a path; received %d", code)
	}
	if code := run([]string{"build", "a.ahd", "b.ahd"}); code != 2 {
		t.Fatalf("expected usage exit 2 for two entry modules; received %d", code)
	}
}

func TestBuildProducesAnExecutable(t *testing.T) {
	entry := write(t, "hello.ahd", "write(\"Hello AhdCode\")\n")
	output := filepath.Join(t.TempDir(), "hello")
	if code := run([]string{"build", entry, "-o", output}); code != 0 {
		t.Fatalf("expected build exit 0; received %d", code)
	}
	if info, err := os.Stat(output); err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("no executable was produced: %v", err)
	}
}

func TestRunPropagatesTheProgramExitCode(t *testing.T) {
	entry := write(t, "boom.ahd", "big: Int := 9223372036854775807\nwrite(big + 1)\n")
	if code := run([]string{"run", entry}); code != 1 {
		t.Fatalf("expected the program exit code 1; received %d", code)
	}
}

func TestRunAndBuildAcceptUnicodePathsWithSpaces(t *testing.T) {
	entry := write(t, "Merhaba Ayşe.ahd", "write(\"path ok\")\n")
	var out, errors bytes.Buffer
	if code := runWithIO([]string{"run", entry}, bytes.NewBuffer(nil), &out, &errors); code != 0 || out.String() != "path ok\n" {
		t.Fatalf("run exit=%d stdout=%q stderr=%q", code, out.String(), errors.String())
	}
	output := filepath.Join(t.TempDir(), "çıktı program")
	out.Reset()
	errors.Reset()
	if code := runWithIO([]string{"build", entry, "-o", output}, bytes.NewBuffer(nil), &out, &errors); code != 0 {
		t.Fatalf("build exit=%d stdout=%q stderr=%q", code, out.String(), errors.String())
	}
	if info, err := os.Stat(output); err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("Unicode/space output was not executable: %v", err)
	}
}

func TestBuildReportsFrontendErrors(t *testing.T) {
	entry := write(t, "bad.ahd", "x: Int := \"text\"\n")
	if code := run([]string{"build", entry, "-o", filepath.Join(t.TempDir(), "bad")}); code != 1 {
		t.Fatalf("expected build exit 1 for failing source; received %d", code)
	}
}

func TestDiagnosticFormattingIncludesLocationAndHint(t *testing.T) {
	files := map[source.FileID]source.File{1: source.NewFile(1, "main.ahd", "\n\n    unknown\n")}
	item := diagnostics.Diagnostic{
		Code: "SEM001", Severity: diagnostics.SeverityError, Message: "unknown name",
		Span: source.Span{FileID: 1, Start: source.Position{Line: 3, Column: 5}}, Hint: "declare it first",
	}
	text := format(item, files)
	if text != "error [SEM001] main.ahd:3:5\nunknown name\n  3 |     unknown\n    |     ^\n  hint: declare it first" {
		t.Fatalf("unexpected diagnostic rendering: %q", text)
	}
	warning := diagnostics.Diagnostic{Code: "BCK001", Severity: diagnostics.SeverityWarning, Message: "deferred"}
	if format(warning, files) != "warning [BCK001]\ndeferred" {
		t.Fatalf("unexpected warning rendering: %q", format(warning, files))
	}
}

func TestHelpVersionAndUnknownFlags(t *testing.T) {
	for _, testCase := range []struct {
		arguments []string
		code      int
		contains  string
	}{
		{[]string{"--help"}, 0, "ahdcode format"},
		{[]string{"--help"}, 0, "ahdcode lsp"},
		{[]string{"--version"}, 0, "AhdCode v0.6.0"},
		{[]string{"run", "--bad"}, 2, "unknown flag"},
		{[]string{"format", "--bad", "x.ahd"}, 2, "unknown flag"},
	} {
		var out, errors bytes.Buffer
		code := runWithIO(testCase.arguments, bytes.NewBuffer(nil), &out, &errors)
		if code != testCase.code || !strings.Contains(out.String()+errors.String(), testCase.contains) {
			t.Fatalf("%v => code %d, stdout %q, stderr %q", testCase.arguments, code, out.String(), errors.String())
		}
	}
}

func TestFormatInPlaceCheckAndUnicodeSpacePath(t *testing.T) {
	entry := write(t, "Ayşe program.ahd", "// yorum\nx:Int:=5\nwrite(x)\n")
	var out, errors bytes.Buffer
	if code := runWithIO([]string{"format", "--check", entry}, bytes.NewBuffer(nil), &out, &errors); code != 1 {
		t.Fatalf("unformatted --check exit = %d, stderr = %q", code, errors.String())
	}
	before, _ := os.ReadFile(entry)
	if string(before) != "// yorum\nx:Int:=5\nwrite(x)\n" {
		t.Fatal("--check modified the source")
	}
	out.Reset()
	errors.Reset()
	if code := runWithIO([]string{"format", entry}, bytes.NewBuffer(nil), &out, &errors); code != 0 {
		t.Fatalf("format exit = %d, stderr = %q", code, errors.String())
	}
	formatted, _ := os.ReadFile(entry)
	if !strings.Contains(string(formatted), "// yorum\nx: Int := 5") {
		t.Fatalf("formatted source:\n%s", formatted)
	}
	if code := runWithIO([]string{"format", "--check", entry}, bytes.NewBuffer(nil), &out, &errors); code != 0 {
		t.Fatalf("formatted --check exit = %d, stderr = %q", code, errors.String())
	}
}

func TestFormatRejectsInvalidSourceWithoutWriting(t *testing.T) {
	entry := write(t, "invalid.ahd", "x: Int := )\n")
	var out, errors bytes.Buffer
	if code := runWithIO([]string{"format", entry}, bytes.NewBuffer(nil), &out, &errors); code != 1 {
		t.Fatalf("invalid format exit = %d", code)
	}
	content, _ := os.ReadFile(entry)
	if string(content) != "x: Int := )\n" {
		t.Fatal("invalid source was modified")
	}
	if !strings.Contains(errors.String(), "error [PAR") || !strings.Contains(errors.String(), "invalid.ahd:1:") || !strings.Contains(errors.String(), "^") {
		t.Fatalf("invalid source diagnostic:\n%s", errors.String())
	}
}

func TestCLIReplUsesPersistentOrdinaryDeclarations(t *testing.T) {
	input := bytes.NewBufferString("x: Int := 5\nwrite(x)\nx: Int := 7\nx = 7\nwrite(x)\n")
	var out, errors bytes.Buffer
	if code := runWithIO(nil, input, &out, &errors); code != 0 {
		t.Fatalf("REPL exit = %d", code)
	}
	if !strings.Contains(out.String(), "5\n") || !strings.Contains(out.String(), "7\n") || !strings.Contains(errors.String(), "already declared") {
		t.Fatalf("stdout:\n%s\nstderr:\n%s", out.String(), errors.String())
	}
}

func lspFrame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

// TestLSPSubcommandStdoutIsProtocolOnly proves `ahdcode lsp` never pollutes
// its stdout with the version banner or any other human-readable text --
// only Content-Length-framed JSON-RPC, exactly like every other subcommand
// keeps diagnostics/human text off stdout and onto stderr.
func TestLSPSubcommandStdoutIsProtocolOnly(t *testing.T) {
	input := lspFrame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var out, errors bytes.Buffer
	code := runWithIO([]string{"lsp"}, strings.NewReader(input), &out, &errors)
	if code != 0 {
		t.Fatalf("ahdcode lsp exit code = %d, stderr = %q", code, errors.String())
	}
	if strings.Contains(out.String(), version) {
		t.Fatalf("ahdcode lsp stdout leaked the human-readable version banner: %q", out.String())
	}
	if strings.Contains(out.String(), "usage:") {
		t.Fatalf("ahdcode lsp stdout leaked CLI usage text: %q", out.String())
	}
	if !strings.HasPrefix(out.String(), "Content-Length:") {
		t.Fatalf("ahdcode lsp stdout does not start with a valid frame header: %q", out.String())
	}
}

// TestLSPSubcommandAcceptsStdioFlag is a regression test: real LSP client
// libraries (vscode-languageclient's Node transport, confirmed against its
// installed source) unconditionally append "--stdio" when launching a
// server configured for stdio transport. ahdcode lsp must accept it as a
// no-op rather than exit with "unexpected argument", or every real editor
// integration fails immediately with "Server process exited with code 2"
// before a single protocol byte is exchanged.
func TestLSPSubcommandAcceptsStdioFlag(t *testing.T) {
	input := lspFrame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`) +
		lspFrame(`{"jsonrpc":"2.0","method":"exit","params":{}}`)
	var out, errors bytes.Buffer
	code := runWithIO([]string{"lsp", "--stdio"}, strings.NewReader(input), &out, &errors)
	if code != 0 {
		t.Fatalf("ahdcode lsp --stdio exit code = %d, stderr = %q", code, errors.String())
	}
	if errors.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", errors.String())
	}
	if !strings.HasPrefix(out.String(), "Content-Length:") {
		t.Fatalf("ahdcode lsp --stdio did not behave like a normal LSP session: %q", out.String())
	}
}

func TestLSPSubcommandRejectsUnexpectedArguments(t *testing.T) {
	var out, errors bytes.Buffer
	code := runWithIO([]string{"lsp", "--port", "9999"}, strings.NewReader(""), &out, &errors)
	if code == 0 {
		t.Fatal("expected a nonzero exit for an unexpected ahdcode lsp argument")
	}
	if !strings.Contains(errors.String(), "unexpected argument") {
		t.Fatalf("stderr = %q", errors.String())
	}
	if out.Len() != 0 {
		t.Fatalf("expected no stdout output when rejecting arguments, got %q", out.String())
	}
}
