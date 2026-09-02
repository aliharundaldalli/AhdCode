package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/semantic"
)

func applyTextEdits(text string, edits []TextEdit) string {
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		start := edit.Span.Start.Offset
		end := edit.Span.End.Offset
		if start < 0 || end < start || end > len(text) {
			continue
		}
		text = text[:start] + edit.NewText + text[end:]
	}
	return text
}

func diagnosticAtCode(result Result, path, code string) (diagnostics.Diagnostic, bool) {
	for owner, items := range result.Diagnostics {
		if canonicalPath(owner) != canonicalPath(path) {
			continue
		}
		for _, item := range items {
			if item.Code == code {
				return item, true
			}
		}
	}
	return diagnostics.Diagnostic{}, false
}

func hasDiagnosticCode(result Result, path, code string) bool {
	_, ok := diagnosticAtCode(result, path, code)
	return ok
}

func assertQuickFixRegression(
	t *testing.T,
	name string,
	path string,
	before string,
	targetCode string,
	negativeOffset int,
	wantTitle string,
	wantAfter string,
) {
	t.Helper()
	store := NewStore()
	result := store.Open(path, before)
	diag, ok := diagnosticAtCode(result, path, targetCode)
	if !ok {
		t.Fatalf("%s: expected diagnostic %s in %#v", name, targetCode, result.Diagnostics)
	}
	actions := store.CodeActions(path, diag.Span.Start.Offset)
	if len(actions) != 1 {
		t.Fatalf("%s: expected exactly one code action, got %#v", name, actions)
	}
	action := actions[0]
	if action.Title != wantTitle {
		t.Fatalf("%s: title = %q, want %q", name, action.Title, wantTitle)
	}
	if len(action.Edits) != 1 {
		t.Fatalf("%s: expected one edit, got %#v", name, action.Edits)
	}
	got := applyTextEdits(before, action.Edits)
	if got != wantAfter {
		t.Fatalf("%s: edited source mismatch\n got:  %q\n want: %q\n edit: %#v", name, got, wantAfter, action.Edits[0])
	}
	after := store.Open(path, got)
	if hasDiagnosticCode(after, path, targetCode) {
		t.Fatalf("%s: target diagnostic %s still present after fix: %#v", name, targetCode, after.Diagnostics)
	}
	if negativeOffset >= 0 {
		for _, candidate := range store.CodeActions(path, negativeOffset) {
			if candidate.Title == wantTitle {
				t.Fatalf("%s: negative offset %d unexpectedly received %q", name, negativeOffset, wantTitle)
			}
		}
	}
}

func TestQuickFixSEM006MissingLocal(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	before := "f: Function := () -> Nothing {\n    x: Int := 1\n}\n"
	wantAfter := "f: Function := () -> Nothing {\n    x: Local Int := 1\n}\n"
	assertQuickFixRegression(
		t,
		"SEM006",
		path,
		before,
		codeMissingLocal,
		offsetOf(t, before, "f: Function"),
		"Add Local modifier",
		wantAfter,
	)
}

func TestQuickFixPAR009RemoveForLocal(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	before := "for i: Local Int in [1] {\n    write(i)\n}\n"
	wantAfter := "for i: Int in [1] {\n    write(i)\n}\n"
	assertQuickFixRegression(
		t,
		"PAR009",
		path,
		before,
		codeInvalidControlSyntax,
		offsetOf(t, before, "write"),
		"Remove invalid Local from for binding",
		wantAfter,
	)
}

func TestQuickFixSEM029ImportSymbol(t *testing.T) {
	directory := t.TempDir()
	toolsPath := filepath.Join(directory, "Tools.ahd")
	mainPath := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(toolsPath, []byte("helper: Function := () -> Nothing { write(1) }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	before := "from Tools bring absent\nwrite(1)\n"
	wantAfter := "write(1)\n"
	assertQuickFixRegression(
		t,
		"SEM029",
		mainPath,
		before,
		semantic.CodeExportNotFound,
		offsetOf(t, before, "write"),
		"Import absent from Tools",
		wantAfter,
	)
}
