package evaluator

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// wordGridSpanTestWriteZIP builds a minimal DOCX-shaped ZIP archive from raw
// entries, mirroring the native runtime's own wordTestWriteZIP helper, so a
// foreign document.xml with explicit gridSpan markup can be fed straight to
// Word.read without going through Document.table's own construction (which
// already guarantees rectangular rows and so cannot reproduce the bug).
func wordGridSpanTestWriteZIP(t *testing.T, path string, entries [][2][]byte) {
	t.Helper()
	var content bytes.Buffer
	writer := zip.NewWriter(&content)
	for _, entry := range entries {
		part, err := writer.Create(string(entry[0]))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(entry[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

const wordGridSpanOriginalXML = `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
	`<w:tbl>` +
	`<w:tr><w:tc><w:p><w:r><w:t>Student</w:t></w:r></w:p></w:tc><w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>Results</w:t></w:r></w:p></w:tc></w:tr>` +
	`<w:tr><w:tc><w:p/></w:tc><w:tc><w:p><w:r><w:t>Midterm</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Final</w:t></w:r></w:p></w:tc></w:tr>` +
	`<w:tr><w:tc><w:p><w:r><w:t>Ali</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>80</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>90</w:t></w:r></w:p></w:tc></w:tr>` +
	`<w:tr><w:tc><w:p><w:r><w:t>Ayşe</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>75</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>88</w:t></w:r></w:p></w:tc></w:tr>` +
	`<w:tr><w:tc><w:p><w:r><w:t>Mehmet</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>82</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>91</w:t></w:r></w:p></w:tc></w:tr>` +
	`</w:tbl></w:body></w:document>`

func wordGridSpanRowStrings(t *testing.T, grid *List) [][]string {
	t.Helper()
	rows := make([][]string, len(grid.Items))
	for index, item := range grid.Items {
		row := item.(*List)
		cells := make([]string, len(row.Items))
		for position, value := range row.Items {
			cells[position] = value.(string)
		}
		rows[index] = cells
	}
	return rows
}

// TestWordEvaluatorGridSpanOriginalFailureSurvivesReadSaveRead reproduces the
// confirmed v0.1.16 data-loss bug: a merged two-column header ("Results")
// sitting over three real data columns used to be read as a ragged table
// whose header width (2) silently capped every re-saved row, dropping the
// Final column ("90", "88", "91"). Case A from the v0.1.17 Word regression
// matrix.
func TestWordEvaluatorGridSpanOriginalFailureSurvivesReadSaveRead(t *testing.T) {
	session := newLatexTestSession()
	directory := t.TempDir()
	original := filepath.Join(directory, "original.docx")
	wordGridSpanTestWriteZIP(t, original, [][2][]byte{{[]byte("word/document.xml"), []byte(wordGridSpanOriginalXML)}})

	assertOriginalShape := func(loaded any) {
		tables := session.wordOperation("Document.tables", loaded, nil).(*List)
		if len(tables.Items) != 1 {
			t.Fatalf("expected exactly one table, got %d", len(tables.Items))
		}
		rows := wordGridSpanRowStrings(t, tables.Items[0].(*List))
		if len(rows) != 5 {
			t.Fatalf("expected 5 logical rows, got %d: %v", len(rows), rows)
		}
		for _, row := range rows {
			if len(row) != 3 {
				t.Fatalf("expected semantic width 3, got %d: %v", len(row), row)
			}
		}
		if rows[0][0] != "Student" || rows[0][1] != "Results" || rows[0][2] != "" {
			t.Fatalf("header row = %v", rows[0])
		}
		if rows[1][1] != "Midterm" || rows[1][2] != "Final" {
			t.Fatalf("sub-header row = %v", rows[1])
		}
		finals := []string{rows[2][2], rows[3][2], rows[4][2]}
		if finals[0] != "90" || finals[1] != "88" || finals[2] != "91" {
			t.Fatalf("Final column values lost: %v", finals)
		}
	}

	loaded := session.wordBuiltin("read", []any{original})
	assertOriginalShape(loaded)

	resaved := filepath.Join(directory, "resaved.docx")
	if got := session.wordOperation("Document.save", loaded, []any{resaved}); got != Nothing {
		t.Fatalf("Document.save returned %#v", got)
	}
	reloaded := session.wordBuiltin("read", []any{resaved})
	assertOriginalShape(reloaded)
}

// TestWordEvaluatorGridSpanInMiddleExpandsAtPosition covers case B: a merged
// cell in the middle of a row must expand to empty columns exactly at the
// point where the span occurred, not be appended at the row's end.
func TestWordEvaluatorGridSpanInMiddleExpandsAtPosition(t *testing.T) {
	session := newLatexTestSession()
	directory := t.TempDir()
	path := filepath.Join(directory, "middle.docx")
	xmlBody := `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		`<w:tbl><w:tr><w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc></w:tr></w:tbl>` +
		`</w:body></w:document>`
	wordGridSpanTestWriteZIP(t, path, [][2][]byte{{[]byte("word/document.xml"), []byte(xmlBody)}})

	loaded := session.wordBuiltin("read", []any{path})
	tables := session.wordOperation("Document.tables", loaded, nil).(*List)
	rows := wordGridSpanRowStrings(t, tables.Items[0].(*List))
	if len(rows) != 1 || rows[0][0] != "A" || rows[0][1] != "" || rows[0][2] != "B" {
		t.Fatalf("span-in-middle result = %v, want [A  B]", rows)
	}
}

// TestWordEvaluatorMultipleGridSpansInOneRow covers case C.
func TestWordEvaluatorMultipleGridSpansInOneRow(t *testing.T) {
	session := newLatexTestSession()
	directory := t.TempDir()
	path := filepath.Join(directory, "multi.docx")
	xmlBody := `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		`<w:tbl><w:tr>` +
		`<w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc>` +
		`<w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc>` +
		`<w:tc><w:p><w:r><w:t>C</w:t></w:r></w:p></w:tc>` +
		`</w:tr></w:tbl></w:body></w:document>`
	wordGridSpanTestWriteZIP(t, path, [][2][]byte{{[]byte("word/document.xml"), []byte(xmlBody)}})

	loaded := session.wordBuiltin("read", []any{path})
	tables := session.wordOperation("Document.tables", loaded, nil).(*List)
	rows := wordGridSpanRowStrings(t, tables.Items[0].(*List))
	want := []string{"A", "", "B", "", "C"}
	if len(rows) != 1 || len(rows[0]) != len(want) {
		t.Fatalf("multi-span result = %v, want %v", rows, want)
	}
	for index, value := range want {
		if rows[0][index] != value {
			t.Fatalf("multi-span result = %v, want %v", rows[0], want)
		}
	}
}

// TestWordEvaluatorRaggedTableRaisesWordErrorOnSave covers case D: the save
// path must defensively reject a ragged internal table with WordError rather
// than truncating it. wordFinishTable and Document.table both guarantee
// rectangular blocks, so a malformed block is constructed directly here to
// exercise the defensive invariant itself.
func TestWordEvaluatorRaggedTableRaisesWordErrorOnSave(t *testing.T) {
	session := newLatexTestSession()
	ragged := session.documentValue([]wordBlock{{
		Kind:    "table",
		Headers: []string{"A", "B"},
		Rows:    [][]string{{"1", "2", "3"}},
		Align:   "left",
	}})
	path := filepath.Join(t.TempDir(), "ragged.docx")
	expectEvaluatorRaise(t, "WordError", func() {
		session.wordOperation("Document.save", ragged, []any{path})
	})
}

// TestWordEvaluatorMergedDataRowReadBackStaysRectangular covers case E/F: a
// horizontal merge inside a data row (not just the header) must still read
// back as a rectangular table and must be safe to re-save, matching the
// native runtime's TestWordTablesValidateMergesAndCopyCallerLists coverage.
func TestWordEvaluatorMergedDataRowReadBackStaysRectangular(t *testing.T) {
	session := newLatexTestSession()
	directory := t.TempDir()
	base := session.wordBuiltin("new", nil)
	headers := &List{Items: []any{"A", "B", "C"}}
	row := &List{Items: []any{"1", "2", "3"}}
	rows := &List{Items: []any{row}}
	merge := &List{Items: []any{int64(1), int64(0), int64(1), int64(2)}}
	merges := &List{Items: []any{merge}}
	document := session.wordOperation("Document.table", base, []any{headers, rows, merges, "left"})
	path := filepath.Join(directory, "data-merge.docx")
	session.wordOperation("Document.save", document, []any{path})

	loaded := session.wordBuiltin("read", []any{path})
	tables := session.wordOperation("Document.tables", loaded, nil).(*List)
	grid := wordGridSpanRowStrings(t, tables.Items[0].(*List))
	if len(grid) != 2 {
		t.Fatalf("expected 2 rows, got %d: %v", len(grid), grid)
	}
	for _, gridRow := range grid {
		if len(gridRow) != 3 {
			t.Fatalf("merged data row read back ragged: %v", grid)
		}
	}
	resaved := filepath.Join(directory, "resaved.docx")
	if got := session.wordOperation("Document.save", loaded, []any{resaved}); got != Nothing {
		t.Fatalf("Document.save returned %#v", got)
	}
}
