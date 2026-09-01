package ahdruntime

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestPDFDocumentIsImmutable(t *testing.T) {
	base := AhdPDFNew()
	withHeading := AhdPDFHeading(base, "Title", 1)
	if len(base.Blocks) != 0 {
		t.Fatal("AhdPDFHeading mutated the base document")
	}
	if len(withHeading.Blocks) != 1 {
		t.Fatalf("blocks = %d; want 1", len(withHeading.Blocks))
	}
	withParagraph := AhdPDFParagraph(withHeading, "Body", "left", false, false, false)
	if len(withHeading.Blocks) != 1 {
		t.Fatal("AhdPDFParagraph mutated its receiver")
	}
	if len(withParagraph.Blocks) != 2 {
		t.Fatalf("blocks = %d; want 2", len(withParagraph.Blocks))
	}
	// Two documents derived from the same base must not share backing storage.
	branchA := AhdPDFHeading(base, "A", 1)
	branchB := AhdPDFHeading(base, "B", 1)
	if branchA.Blocks[0] == branchB.Blocks[0] {
		t.Fatal("expected different block content for independently derived branches")
	}
	if len(base.Blocks) != 0 {
		t.Fatal("base document changed after deriving two branches from it")
	}
}

func TestPDFHeadingLevelValidation(t *testing.T) {
	doc := AhdPDFNew()
	for _, level := range []int64{1, 2, 3, 4, 5, 6} {
		result := AhdPDFHeading(doc, "x", level)
		if len(result.Blocks) != 1 {
			t.Fatalf("level %d: expected one block", level)
		}
	}
	for _, level := range []int64{0, -1, 7, 100} {
		expectRaise(t, AhdClassPDFError, func() { AhdPDFHeading(doc, "x", level) })
	}
}

func TestPDFParagraphAlignValidation(t *testing.T) {
	doc := AhdPDFNew()
	for _, align := range []string{"left", "center", "right", "justify"} {
		if len(AhdPDFParagraph(doc, "x", align, false, false, false).Blocks) != 1 {
			t.Fatalf("align %q: expected one block", align)
		}
	}
	expectRaise(t, AhdClassPDFError, func() { AhdPDFParagraph(doc, "x", "top", false, false, false) })
	expectRaise(t, AhdClassPDFError, func() { AhdPDFParagraph(doc, "x", "", false, false, false) })
}

func TestPDFTableRaggedRowsRejected(t *testing.T) {
	doc := AhdPDFNew()
	headers := AhdNewList("A", "B", "C")
	goodRows := AhdNewList(AhdNewList("1", "2", "3"))
	result := AhdPDFTable(doc, headers, goodRows, "left")
	if len(result.Blocks) != 1 {
		t.Fatal("expected one table block")
	}
	raggedRows := AhdNewList(AhdNewList("1", "2"))
	expectRaise(t, AhdClassPDFError, func() { AhdPDFTable(doc, headers, raggedRows, "left") })

	noHeaders := AhdNewList[string]()
	expectRaise(t, AhdClassPDFError, func() { AhdPDFTable(doc, noHeaders, AhdNewList[*AhdList[string]](), "left") })

	expectRaise(t, AhdClassPDFError, func() { AhdPDFTable(doc, headers, goodRows, "middle") })
}

func TestPDFTableDoesNotMutateSourceLists(t *testing.T) {
	doc := AhdPDFNew()
	headers := AhdNewList("A", "B")
	row := AhdNewList("1", "2")
	rows := AhdNewList(row)
	_ = AhdPDFTable(doc, headers, rows, "left")
	if headers.Len() != 2 || row.Len() != 2 || rows.Len() != 1 {
		t.Fatal("PDFDocument.table mutated a source List")
	}
}

func pdfTestPNGBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	image := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			image.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestPDFImageValidationAndAspectRatio(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "test.png")
	if err := os.WriteFile(path, pdfTestPNGBytes(t, 200, 100), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := AhdPDFNew()

	// No size: block carries zero width/height, meaning "let the renderer
	// use the image's natural size".
	noSize := AhdPDFImage(doc, path, AhdBuildPair([]string{}, []float64{}))
	blocks := ahdPDFDecodeBlocks(noSize)
	if blocks[0].WidthCM != 0 || blocks[0].HeightCM != 0 {
		t.Fatalf("no-size image extent = (%v,%v); want (0,0)", blocks[0].WidthCM, blocks[0].HeightCM)
	}
	if blocks[0].MediaExt != "png" || len(blocks[0].Media) == 0 {
		t.Fatal("image block did not embed PNG bytes")
	}

	// Width only: height must follow the 200x100 (2:1) aspect ratio.
	widthOnly := AhdPDFImage(doc, path, AhdBuildPair([]string{"width"}, []float64{10}))
	widthBlocks := ahdPDFDecodeBlocks(widthOnly)
	if widthBlocks[0].WidthCM != 10 || widthBlocks[0].HeightCM != 5 {
		t.Fatalf("width-only extent = (%v,%v); want (10,5)", widthBlocks[0].WidthCM, widthBlocks[0].HeightCM)
	}

	// Both dimensions: used exactly as given, no aspect correction.
	both := AhdPDFImage(doc, path, AhdBuildPair([]string{"width", "height"}, []float64{3, 3}))
	bothBlocks := ahdPDFDecodeBlocks(both)
	if bothBlocks[0].WidthCM != 3 || bothBlocks[0].HeightCM != 3 {
		t.Fatalf("explicit extent = (%v,%v); want (3,3)", bothBlocks[0].WidthCM, bothBlocks[0].HeightCM)
	}

	expectRaise(t, AhdClassPDFError, func() { AhdPDFImage(doc, "", AhdBuildPair([]string{}, []float64{})) })
	expectRaise(t, AhdClassPDFError, func() {
		AhdPDFImage(doc, filepath.Join(directory, "missing.png"), AhdBuildPair([]string{}, []float64{}))
	})
	expectRaise(t, AhdClassPDFError, func() {
		AhdPDFImage(doc, path, AhdBuildPair([]string{"width"}, []float64{-1}))
	})
	expectRaise(t, AhdClassPDFError, func() {
		AhdPDFImage(doc, path, AhdBuildPair([]string{"depth"}, []float64{1}))
	})

	textPath := filepath.Join(directory, "not-an-image.png")
	if err := os.WriteFile(textPath, []byte("not a real image"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectRaise(t, AhdClassPDFError, func() { AhdPDFImage(doc, textPath, AhdBuildPair([]string{}, []float64{})) })
}

func TestPDFImageSurvivesSourceFileDeletion(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "test.png")
	if err := os.WriteFile(path, pdfTestPNGBytes(t, 10, 10), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := AhdPDFImage(AhdPDFNew(), path, AhdBuildPair([]string{}, []float64{}))
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	blocks := ahdPDFDecodeBlocks(doc)
	if len(blocks[0].Media) == 0 {
		t.Fatal("image bytes were not embedded independently of the source file")
	}
}

func TestPDFPageBreakAppendsBlock(t *testing.T) {
	doc := AhdPDFPageBreak(AhdPDFNew())
	blocks := ahdPDFDecodeBlocks(doc)
	if len(blocks) != 1 || blocks[0].Kind != "pageBreak" {
		t.Fatalf("blocks = %+v", blocks)
	}
}

func TestPDFSaveRejectsNonPDFExtension(t *testing.T) {
	doc := AhdPDFHeading(AhdPDFNew(), "x", 1)
	directory := t.TempDir()
	for _, name := range []string{"out.txt", "out.PDF.bak", "out"} {
		destination := filepath.Join(directory, name)
		expectRaise(t, AhdClassPDFError, func() { AhdPDFSave(doc, destination) })
	}
}

func TestPDFTextEscapingReachesLatexEscape(t *testing.T) {
	// PDFDocument.save() must never expose raw LaTeX: every reserved
	// character in user text is routed through AhdLatexEscape. This checks
	// the heading/paragraph/table body builders call it -- not the full
	// render, which needs a staged Tectonic runtime (see internal/build).
	block := ahdPDFBlock{Kind: "heading", Text: `\{}$&#%_^~`, Level: 1}
	command := ahdPDFHeadingCommand(block)
	if want := AhdLatexEscape(block.Text); command != `\section{`+want+"}\n" {
		t.Fatalf("heading command = %q", command)
	}
	paragraph := ahdPDFBlock{Kind: "paragraph", Text: `\{}$&#%_^~`, Align: "justify"}
	body := ahdPDFParagraphBody(paragraph)
	if want := AhdLatexEscape(paragraph.Text); body != want+"\n\n" {
		t.Fatalf("paragraph body = %q", body)
	}
	table := ahdPDFBlock{Kind: "table", Headers: []string{`&`}, Rows: [][]string{{`%`}}, Align: "left"}
	tableBody := ahdPDFTableBody(table)
	if !bytes.Contains([]byte(tableBody), []byte(AhdLatexEscape("&"))) || !bytes.Contains([]byte(tableBody), []byte(AhdLatexEscape("%"))) {
		t.Fatalf("table body did not escape reserved characters: %q", tableBody)
	}
}

func TestPDFFromExcelRequiresAtLeastOneSheet(t *testing.T) {
	empty := excelEncode(excelWorkbookData{Sheets: []excelSheetData{}})
	expectRaise(t, AhdClassPDFError, func() { AhdPDFFromExcel(empty) })
}

func TestPDFFromExcelColumnLimit(t *testing.T) {
	cells := make([]excelCellEntry, 0, ahdPDFMaxExcelColumns+1)
	for column := int64(1); column <= ahdPDFMaxExcelColumns+1; column++ {
		cells = append(cells, excelCellEntry{Row: 1, Column: column, Cell: excelCellData{Kind: "String", Text: "x"}})
	}
	workbook := excelEncode(excelWorkbookData{Sheets: []excelSheetData{{Name: "Wide", Cells: cells}}})
	expectRaise(t, AhdClassPDFError, func() { AhdPDFFromExcel(workbook) })

	withinLimit := make([]excelCellEntry, 0, ahdPDFMaxExcelColumns)
	for column := int64(1); column <= ahdPDFMaxExcelColumns; column++ {
		withinLimit = append(withinLimit, excelCellEntry{Row: 1, Column: column, Cell: excelCellData{Kind: "String", Text: "x"}})
	}
	fits := excelEncode(excelWorkbookData{Sheets: []excelSheetData{{Name: "Fits", Cells: withinLimit}}})
	result := AhdPDFFromExcel(fits)
	blocks := ahdPDFDecodeBlocks(result)
	if len(blocks) != 2 || blocks[0].Kind != "heading" || blocks[1].Kind != "table" {
		t.Fatalf("blocks = %+v", blocks)
	}
}

func TestPDFFromExcelFormulaShowsSourceNotFabricatedValue(t *testing.T) {
	cells := []excelCellEntry{
		{Row: 1, Column: 1, Cell: excelCellData{Kind: "Formula", Text: "=SUM(A1:A2)"}},
	}
	workbook := excelEncode(excelWorkbookData{Sheets: []excelSheetData{{Name: "S", Cells: cells}}})
	result := AhdPDFFromExcel(workbook)
	blocks := ahdPDFDecodeBlocks(result)
	if len(blocks) != 2 || blocks[1].Kind != "table" || len(blocks[1].Headers) != 1 {
		t.Fatalf("blocks = %+v", blocks)
	}
	if blocks[1].Headers[0] != "=SUM(A1:A2)" {
		t.Fatalf("formula cell text = %q; want the exact formula source", blocks[1].Headers[0])
	}
}

func TestPDFFromExcelMergedNonAnchorCellsStayBlank(t *testing.T) {
	cells := []excelCellEntry{
		{Row: 1, Column: 1, Cell: excelCellData{Kind: "String", Text: "anchor"}},
	}
	merges := []excelRangeData{{StartRow: 1, StartColumn: 1, EndRow: 1, EndColumn: 2}}
	workbook := excelEncode(excelWorkbookData{Sheets: []excelSheetData{{Name: "S", Cells: cells, Merges: merges}}})
	result := AhdPDFFromExcel(workbook)
	blocks := ahdPDFDecodeBlocks(result)
	if blocks[1].Headers[0] != "anchor" || blocks[1].Headers[1] != "" {
		t.Fatalf("headers = %+v; want [anchor, \"\"] (merge loses no value, no fabricated span)", blocks[1].Headers)
	}
}

func TestPDFFromWordMapsEveryBlockKind(t *testing.T) {
	word := AhdWordDocument{}
	word = ahdWordAppend(word, ahdWordBlock{Kind: "heading", Text: "Title", Level: 2})
	word = ahdWordAppend(word, ahdWordBlock{Kind: "paragraph", Text: "Body", Align: "center", Bold: true})
	word = ahdWordAppend(word, ahdWordBlock{Kind: "table", Headers: []string{"H"}, Rows: [][]string{{"v"}}, Align: "left"})
	word = ahdWordAppend(word, ahdWordBlock{Kind: "pageBreak"})
	word = ahdWordAppend(word, ahdWordBlock{Kind: "image", Media: []byte{1, 2, 3}, MediaExt: "png", WidthEMU: 360000, HeightEMU: 720000})

	result := AhdPDFFromWord(word)
	blocks := ahdPDFDecodeBlocks(result)
	if len(blocks) != 5 {
		t.Fatalf("blocks = %d; want 5", len(blocks))
	}
	if blocks[0].Kind != "heading" || blocks[0].Text != "Title" || blocks[0].Level != 2 {
		t.Fatalf("heading block = %+v", blocks[0])
	}
	if blocks[1].Kind != "paragraph" || blocks[1].Text != "Body" || blocks[1].Align != "center" || !blocks[1].Bold {
		t.Fatalf("paragraph block = %+v", blocks[1])
	}
	if blocks[2].Kind != "table" || len(blocks[2].Headers) != 1 || blocks[2].Rows[0][0] != "v" {
		t.Fatalf("table block = %+v", blocks[2])
	}
	if blocks[3].Kind != "pageBreak" {
		t.Fatalf("pageBreak block = %+v", blocks[3])
	}
	if blocks[4].Kind != "image" || blocks[4].WidthCM != 1 || blocks[4].HeightCM != 2 {
		t.Fatalf("image block = %+v; want widthCM=1 heightCM=2 (360000/720000 EMU -> cm)", blocks[4])
	}
}
