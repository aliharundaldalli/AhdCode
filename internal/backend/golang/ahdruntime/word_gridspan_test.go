package ahdruntime

import (
	"path/filepath"
	"testing"
)

const ahdWordGridSpanOriginalXML = `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
	`<w:tbl>` +
	`<w:tr><w:tc><w:p><w:r><w:t>Student</w:t></w:r></w:p></w:tc><w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>Results</w:t></w:r></w:p></w:tc></w:tr>` +
	`<w:tr><w:tc><w:p/></w:tc><w:tc><w:p><w:r><w:t>Midterm</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Final</w:t></w:r></w:p></w:tc></w:tr>` +
	`<w:tr><w:tc><w:p><w:r><w:t>Ali</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>80</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>90</w:t></w:r></w:p></w:tc></w:tr>` +
	`<w:tr><w:tc><w:p><w:r><w:t>Ayşe</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>75</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>88</w:t></w:r></w:p></w:tc></w:tr>` +
	`<w:tr><w:tc><w:p><w:r><w:t>Mehmet</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>82</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>91</w:t></w:r></w:p></w:tc></w:tr>` +
	`</w:tbl></w:body></w:document>`

// TestWordNativeGridSpanOriginalFailureSurvivesReadSaveRead reproduces the
// confirmed v0.1.16 data-loss bug in the native (compiled) runtime and proves
// case A of the v0.1.17 Word regression matrix: the Final column ("90", "88",
// "91") must survive Word.read, and must still survive a save/re-read cycle.
func TestWordNativeGridSpanOriginalFailureSurvivesReadSaveRead(t *testing.T) {
	directory := t.TempDir()
	original := filepath.Join(directory, "original.docx")
	wordTestWriteZIP(t, original, [][2][]byte{{[]byte("word/document.xml"), []byte(ahdWordGridSpanOriginalXML)}})

	assertOriginalShape := func(document AhdWordDocument) {
		tables := AhdWordTables(document).Snapshot()
		if len(tables) != 1 {
			t.Fatalf("expected exactly one table, got %d", len(tables))
		}
		rows := tables[0].Snapshot()
		if len(rows) != 5 {
			t.Fatalf("expected 5 logical rows, got %d", len(rows))
		}
		var grid [][]string
		for _, row := range rows {
			grid = append(grid, row.Snapshot())
		}
		for _, row := range grid {
			if len(row) != 3 {
				t.Fatalf("expected semantic width 3, got %d: %v", len(row), grid)
			}
		}
		if grid[0][0] != "Student" || grid[0][1] != "Results" || grid[0][2] != "" {
			t.Fatalf("header row = %v", grid[0])
		}
		if grid[1][1] != "Midterm" || grid[1][2] != "Final" {
			t.Fatalf("sub-header row = %v", grid[1])
		}
		if grid[2][2] != "90" || grid[3][2] != "88" || grid[4][2] != "91" {
			t.Fatalf("Final column values lost: %v / %v / %v", grid[2][2], grid[3][2], grid[4][2])
		}
	}

	loaded := AhdWordRead(original)
	assertOriginalShape(loaded)

	resaved := filepath.Join(directory, "resaved.docx")
	AhdWordSave(loaded, resaved)
	reloaded := AhdWordRead(resaved)
	assertOriginalShape(reloaded)
}

// TestWordNativeGridSpanInMiddleExpandsAtPosition covers case B.
func TestWordNativeGridSpanInMiddleExpandsAtPosition(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "middle.docx")
	xmlBody := `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		`<w:tbl><w:tr><w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc></w:tr></w:tbl>` +
		`</w:body></w:document>`
	wordTestWriteZIP(t, path, [][2][]byte{{[]byte("word/document.xml"), []byte(xmlBody)}})

	tables := AhdWordTables(AhdWordRead(path)).Snapshot()
	row := tables[0].Snapshot()[0].Snapshot()
	if len(row) != 3 || row[0] != "A" || row[1] != "" || row[2] != "B" {
		t.Fatalf("span-in-middle result = %v, want [A  B]", row)
	}
}

// TestWordNativeMultipleGridSpansInOneRow covers case C.
func TestWordNativeMultipleGridSpansInOneRow(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "multi.docx")
	xmlBody := `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		`<w:tbl><w:tr>` +
		`<w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc>` +
		`<w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc>` +
		`<w:tc><w:p><w:r><w:t>C</w:t></w:r></w:p></w:tc>` +
		`</w:tr></w:tbl></w:body></w:document>`
	wordTestWriteZIP(t, path, [][2][]byte{{[]byte("word/document.xml"), []byte(xmlBody)}})

	tables := AhdWordTables(AhdWordRead(path)).Snapshot()
	row := tables[0].Snapshot()[0].Snapshot()
	want := []string{"A", "", "B", "", "C"}
	if len(row) != len(want) {
		t.Fatalf("multi-span result = %v, want %v", row, want)
	}
	for index, value := range want {
		if row[index] != value {
			t.Fatalf("multi-span result = %v, want %v", row, want)
		}
	}
}

// TestWordNativeRaggedTableRaisesWordErrorOnSave covers case D: the native
// save path must defensively reject a ragged internal table with WordError
// rather than truncating it.
func TestWordNativeRaggedTableRaisesWordErrorOnSave(t *testing.T) {
	ragged := ahdWordAppend(AhdWordNew(), ahdWordBlock{
		Kind:    "table",
		Headers: []string{"A", "B"},
		Rows:    [][]string{{"1", "2", "3"}},
		Align:   "left",
	})
	path := filepath.Join(t.TempDir(), "ragged.docx")
	expectRaise(t, AhdClassWordError, func() { AhdWordSave(ragged, path) })
}
