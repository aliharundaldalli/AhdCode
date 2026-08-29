package formatter

import (
	"os"
	"path/filepath"
	"testing"

	"ahdcode/internal/lexer"
	"ahdcode/internal/parser"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
)

// TestEntireExampleCorpusFormatsIdempotentlyWithoutChangingSemantics is a
// broad regression net for the canonical-formatting rewrite: every .ahd file
// under examples/ and docs/ that already parses cleanly must format without
// diagnostics, format idempotently, and keep the same semantic validity
// before and after formatting.
func TestEntireExampleCorpusFormatsIdempotentlyWithoutChangingSemantics(t *testing.T) {
	root := filepath.Join("..", "..")
	var files []string
	for _, dir := range []string{"examples", "docs"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".ahd" {
				files = append(files, path)
			}
			return nil
		})
	}
	t.Logf("scanning %d .ahd files", len(files))
	for _, path := range files {
		path := path
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			file0 := source.NewFile(1, path, string(content))
			lexed0 := lexer.Lex(file0)
			parsed0 := parser.Parse(file0, lexed0.Tokens)
			if lexed0.HasErrors() || parsed0.HasErrors() {
				t.Skip("input does not parse cleanly; not a formatter target")
			}
			first := Format(file0)
			if first.HasErrors() {
				t.Fatalf("formatting a validly-parsed file produced errors: %+v", first.Diagnostics)
			}
			second := Format(source.NewFile(1, path, first.Text))
			if second.HasErrors() {
				t.Fatalf("re-formatting produced errors: %+v\n%s", second.Diagnostics, first.Text)
			}
			if second.Text != first.Text {
				t.Fatalf("not idempotent:\n---first---\n%s\n---second---\n%s", first.Text, second.Text)
			}
			file1 := source.NewFile(1, path, first.Text)
			lexed1 := lexer.Lex(file1)
			parsed1 := parser.Parse(file1, lexed1.Tokens)
			analyzed0 := semantic.Analyze(parsed0)
			analyzed1 := semantic.Analyze(parsed1)
			if analyzed0.HasErrors() != analyzed1.HasErrors() {
				t.Fatalf("semantic validity changed by formatting (before=%v after=%v)\nbefore diags=%+v\nafter diags=%+v\n%s",
					!analyzed0.HasErrors(), !analyzed1.HasErrors(), analyzed0.Diagnostics, analyzed1.Diagnostics, first.Text)
			}
		})
	}
}
