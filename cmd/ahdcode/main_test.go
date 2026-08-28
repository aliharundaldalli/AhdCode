package main

import (
	"os"
	"path/filepath"
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
	if code := run(nil); code != 2 {
		t.Fatalf("expected usage exit 2; received %d", code)
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

func TestBuildReportsFrontendErrors(t *testing.T) {
	entry := write(t, "bad.ahd", "x: Int := \"text\"\n")
	if code := run([]string{"build", entry, "-o", filepath.Join(t.TempDir(), "bad")}); code != 1 {
		t.Fatalf("expected build exit 1 for failing source; received %d", code)
	}
}

func TestDiagnosticFormattingIncludesLocationAndHint(t *testing.T) {
	files := map[source.FileID]source.File{1: source.NewFile(1, "main.ahd", "")}
	item := diagnostics.Diagnostic{
		Code: "SEM001", Severity: diagnostics.SeverityError, Message: "unknown name",
		Span: source.Span{FileID: 1, Start: source.Position{Line: 3, Column: 5}}, Hint: "declare it first",
	}
	text := format(item, files)
	if text != "main.ahd:3:5: error [SEM001] unknown name\n  hint: declare it first" {
		t.Fatalf("unexpected diagnostic rendering: %q", text)
	}
	warning := diagnostics.Diagnostic{Code: "BCK001", Severity: diagnostics.SeverityWarning, Message: "deferred"}
	if format(warning, files) != "warning [BCK001] deferred" {
		t.Fatalf("unexpected warning rendering: %q", format(warning, files))
	}
}
