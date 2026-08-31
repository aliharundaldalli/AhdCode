package evaluator

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestWordEvaluatorDocumentOperationsArePure(t *testing.T) {
	session := newLatexTestSession()
	base := session.wordBuiltin("new", nil)
	heading := session.wordOperation("Document.heading", base, []any{"Report", int64(1)})
	document := session.wordOperation("Document.paragraph", heading, []any{"Summary", "center", true, true, true})
	document = session.wordOperation("Document.pageBreak", document, nil)

	if got := session.wordOperation("Document.text", base, nil).(string); got != "" {
		t.Fatalf("base Document was mutated: %q", got)
	}
	if got := session.wordOperation("Document.text", document, nil).(string); got != "Report\nSummary" {
		t.Fatalf("Document.text = %q", got)
	}
	if got := session.wordOperation("Document.headings", document, nil).(*List).Items; len(got) != 1 || got[0] != "Report" {
		t.Fatalf("Document.headings = %v", got)
	}
	if got := session.wordOperation("Document.paragraphs", document, nil).(*List).Items; len(got) != 1 || got[0] != "Summary" {
		t.Fatalf("Document.paragraphs = %v", got)
	}
	expectEvaluatorRaise(t, "WordError", func() {
		session.wordOperation("Document.heading", base, []any{"bad", int64(7)})
	})
	expectEvaluatorRaise(t, "WordError", func() {
		session.wordOperation("Document.paragraph", base, []any{"bad", "middle"})
	})
}

func TestWordEvaluatorTablesCopyInputsAndValidateMerges(t *testing.T) {
	session := newLatexTestSession()
	base := session.wordBuiltin("new", nil)
	headers := &List{Items: []any{"A", "B"}}
	row := &List{Items: []any{"1", "2"}}
	rows := &List{Items: []any{row}}
	merge := &List{Items: []any{int64(0), int64(0), int64(1), int64(2)}}
	merges := &List{Items: []any{merge}}
	document := session.wordOperation("Document.table", base, []any{headers, rows, merges, "center"})
	headers.Items[0] = "changed"
	row.Items[0] = "changed"
	merge.Items[3] = int64(1)

	tables := session.wordOperation("Document.tables", document, nil).(*List)
	grid := tables.Items[0].(*List)
	if grid.Items[0].(*List).Items[0] != "A" || grid.Items[1].(*List).Items[0] != "1" {
		t.Fatalf("Document retained caller List aliases: %v", tables.Items)
	}
	invalid := [][]any{
		{int64(0), int64(0), int64(1)},
		{int64(-1), int64(0), int64(1), int64(2)},
		{int64(0), int64(0), int64(1), int64(1)},
		{int64(0), int64(1), int64(1), int64(2)},
	}
	for _, values := range invalid {
		values := values
		expectEvaluatorRaise(t, "WordError", func() {
			session.wordOperation("Document.table", base, []any{
				&List{Items: []any{"A", "B"}},
				&List{Items: []any{&List{Items: []any{"1", "2"}}}},
				&List{Items: []any{&List{Items: values}}},
			})
		})
	}
}

func TestWordEvaluatorImageSaveAndRead(t *testing.T) {
	session := newLatexTestSession()
	directory := t.TempDir()
	imagePath := filepath.Join(directory, "source.png")
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 40, 20))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imagePath, encoded.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	document := session.wordBuiltin("new", nil)
	document = session.wordOperation("Document.heading", document, []any{"Image", int64(1)})
	size := &Pair{Keys: []any{"width"}, Values: map[any]any{"width": 5.0}}
	document = session.wordOperation("Document.image", document, []any{imagePath, size})
	if err := os.Remove(imagePath); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(directory, "result.docx")
	if got := session.wordOperation("Document.save", document, []any{output}); got != Nothing {
		t.Fatalf("Document.save returned %#v", got)
	}
	content, err := os.ReadFile(output)
	if err != nil || len(content) < 100 || !bytes.HasPrefix(content, []byte("PK")) {
		t.Fatalf("invalid evaluator DOCX: size=%d err=%v", len(content), err)
	}
	loaded := session.wordBuiltin("read", []any{output})
	if got := session.wordOperation("Document.text", loaded, nil).(string); got != "Image" {
		t.Fatalf("read text = %q", got)
	}
	expectEvaluatorRaise(t, "WordError", func() {
		session.wordBuiltin("read", []any{filepath.Join(directory, "missing.docx")})
	})
}
