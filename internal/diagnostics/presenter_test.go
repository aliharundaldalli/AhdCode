package diagnostics

import (
	"testing"

	"ahdcode/internal/source"
)

func TestRenderIncludesSourceCaretAndHint(t *testing.T) {
	file := source.NewFile(1, "main.ahd", "first\nvalue + bad\n")
	item := Diagnostic{
		Code: "SEM001", Severity: SeverityError, Message: "unknown name",
		Span: source.Span{FileID: 1, Start: source.Position{Offset: 14, Line: 2, Column: 9}, End: source.Position{Offset: 17, Line: 2, Column: 12}},
		Hint: "declare it first",
	}
	want := "error [SEM001] main.ahd:2:9\nunknown name\n  2 | value + bad\n    |         ^^^\n  hint: declare it first"
	if got := Render(item, map[source.FileID]source.File{1: file}); got != want {
		t.Fatalf("rendered diagnostic:\n%s\nwant:\n%s", got, want)
	}
}
