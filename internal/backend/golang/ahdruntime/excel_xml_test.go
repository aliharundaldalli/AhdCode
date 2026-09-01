package ahdruntime

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExcelFormulaCellsCarryCalculationInteroperabilityMetadata is the
// regression for the confirmed real-world defect where a Formula Cell with
// no cached <v> and no workbook calcId left B2:B4 blank in real Microsoft
// Excel until the user pressed F9, even though fullCalcOnLoad/forceFullCalc
// were already present. Verified empirically against real Microsoft Excel
// and real Apple Numbers (both installed on the build machine): opening the
// generated file displays every formula's correct result immediately, with
// no editing, F9, or re-entry. This test checks the generated XML contract
// that produces that behavior, not the applications themselves.
func TestExcelFormulaCellsCarryCalculationInteroperabilityMetadata(t *testing.T) {
	workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "Sheet1"))
	sheet := excelTestValue(ExcelWorkbookSheet(workbook, "Sheet1"))
	formulas := []string{
		`=SUM(A2:A4)`, `=AVERAGE(A2:A4)`, `=MAX(A2:A4)`,
		`=1+2`, `=A2+A3`, `=IF(A2<20,"yes","no")`, `=A2*2.5`, `=IF(A2<>A3,"different","same")`,
	}
	for index, expression := range formulas {
		cell := excelTestValue(ExcelFormula(expression))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, int64(index+1), 1, cell))
	}
	// Excel.fromString must never be reinterpreted as a Formula, before or
	// after this fix.
	literal := excelTestValue(ExcelFromString(`=SUM(A2:A4)`))
	sheet = excelTestValue(ExcelSheetSetCell(sheet, int64(len(formulas)+1), 1, literal))
	workbook = excelTestValue(ExcelWorkbookWithSheet(workbook, sheet))

	directory := t.TempDir()
	path := filepath.Join(directory, "formulas.xlsx")
	if err := ExcelWorkbookSave(workbook, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	data := excelTestReadFile(t, path)
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a valid ZIP: %v", err)
	}
	parts := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		opened, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		content, readErr := io.ReadAll(opened)
		_ = opened.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		parts[file.Name] = string(content)
		decoder := xml.NewDecoder(bytes.NewReader(content))
		var parseErr error
		for {
			if _, parseErr = decoder.Token(); parseErr != nil {
				break
			}
		}
		if !errors.Is(parseErr, io.EOF) {
			t.Fatalf("%s is not well-formed XML: %v", file.Name, parseErr)
		}
	}

	workbookXML := parts["xl/workbook.xml"]
	for _, required := range []string{`calcId="124519"`, `fullCalcOnLoad="1"`, `forceFullCalc="1"`} {
		if !strings.Contains(workbookXML, required) {
			t.Fatalf("xl/workbook.xml calcPr missing %s:\n%s", required, workbookXML)
		}
	}

	sheetXML := parts["xl/worksheets/sheet1.xml"]
	for _, expression := range formulas {
		// Each formula's escaped source must be immediately followed by a
		// cached <v> placeholder within the same cell.
		escaped := excelEscapeXML(strings.TrimPrefix(expression, "="))
		want := "<f>" + escaped + "</f><v>0</v>"
		if !strings.Contains(sheetXML, want) {
			t.Fatalf("formula cell for %q missing cached placeholder %q:\n%s", expression, want, sheetXML)
		}
	}
	if !strings.Contains(sheetXML, `t="inlineStr"><is><t xml:space="preserve">=SUM(A2:A4)</t>`) {
		t.Fatalf("literal String cell lost its exact text:\n%s", sheetXML)
	}

	// The public contract: kind()/formula() only ever see the caller's exact
	// source, never the cached placeholder.
	loaded := excelTestValue(ExcelRead(path))
	loadedSheet := excelTestValue(ExcelWorkbookSheet(loaded, "Sheet1"))
	for index, expression := range formulas {
		cell := excelTestValue(ExcelSheetCell(loadedSheet, int64(index+1), 1))
		if kind := excelTestValue(ExcelCellKind(cell)); kind != "Formula" {
			t.Fatalf("row %d kind = %q; want Formula", index+1, kind)
		}
		if got := excelTestValue(ExcelCellFormula(cell)); got != expression {
			t.Fatalf("row %d formula = %q; want %q", index+1, got, expression)
		}
	}
	literalCell := excelTestValue(ExcelSheetCell(loadedSheet, int64(len(formulas)+1), 1))
	if kind := excelTestValue(ExcelCellKind(literalCell)); kind != "String" {
		t.Fatalf("literal cell kind = %q; want String", kind)
	}
	if got := excelTestValue(ExcelCellString(literalCell)); got != `=SUM(A2:A4)` {
		t.Fatalf("literal cell text = %q", got)
	}

	// save -> read -> save -> read must remain byte-for-byte deterministic
	// with the new cached placeholder and calcPr metadata in play.
	resaved := filepath.Join(directory, "formulas-2.xlsx")
	if err := ExcelWorkbookSave(loaded, resaved); err != nil {
		t.Fatalf("resave: %v", err)
	}
	first := excelTestReadFile(t, path)
	second := excelTestReadFile(t, resaved)
	if !bytes.Equal(first, second) {
		t.Fatal("save -> read -> save did not produce byte-identical output")
	}
}

func excelTestReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read %s: %v", path, err)
	}
	return data
}

// excelXMLEscapingStrings is the QA-reported matrix of XML-reserved
// characters, common combinations, an already-escaped-looking String, and
// Unicode combined with reserved characters. Every entry must survive
// save -> read byte-for-byte in the closed Cell model.
var excelXMLEscapingStrings = []string{
	"Tom & Jerry",
	"1 < 2",
	"3 > 2",
	`"hello"`,
	"it's fine",
	`<>&"'`,
	`<tag attr="x">A & B</tag>`,
	"🌍 < A & B > 🎉",
	"Tom &amp; Jerry", // literal already-escaped-looking text must not be re-interpreted
}

var excelXMLEscapingFormulas = []string{
	`=IF(A1<5,"yes","no")`,
	`=A1&B1`,
	`=IF(A1<>B1,"different","same")`,
}

// excelParseEveryXMLPart independently parses each XLSX package member with
// encoding/xml, failing the test if any part is not well-formed. It mirrors
// unzip -t by fully reading (and so CRC-checking) every stored entry first.
func excelParseEveryXMLPart(t *testing.T, data []byte) {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("XLSX package is not a valid ZIP archive: %v", err)
	}
	if len(reader.File) == 0 {
		t.Fatal("XLSX package contains no members")
	}
	for _, file := range reader.File {
		opened, err := file.Open()
		if err != nil {
			t.Fatalf("%s: could not open: %v", file.Name, err)
		}
		content, err := io.ReadAll(opened)
		closeErr := opened.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("%s: corrupted member (bad CRC or truncated data): read=%v close=%v", file.Name, err, closeErr)
		}
		if !strings.HasSuffix(file.Name, ".xml") && !strings.HasSuffix(file.Name, ".rels") {
			continue
		}
		decoder := xml.NewDecoder(bytes.NewReader(content))
		decoder.Strict = true
		var parseErr error
		for {
			if _, parseErr = decoder.Token(); parseErr != nil {
				break
			}
		}
		if !errors.Is(parseErr, io.EOF) {
			t.Fatalf("%s is not well-formed XML: %v\n%s", file.Name, parseErr, content)
		}
	}
}

// buildExcelXMLEscapingWorkbook constructs the required external regression
// workbook's content: reserved-character Strings, formulas, an XML-sensitive
// but Excel-legal Sheet name, and an XML-sensitive custom number format.
func buildExcelXMLEscapingWorkbook(t *testing.T) string {
	t.Helper()
	sheetName := `A & B <legal> it's ok`
	workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), sheetName))
	sheet := excelTestValue(ExcelWorkbookSheet(workbook, sheetName))
	for index, text := range excelXMLEscapingStrings {
		cell := excelTestValue(ExcelFromString(text))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, int64(index+1), 1, cell))
	}
	for index, expression := range excelXMLEscapingFormulas {
		cell := excelTestValue(ExcelFormula(expression))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, int64(index+1), 2, cell))
	}
	numberFormat := `0.00" <A&B> "`
	style := excelTestValue(ExcelStyleString(ExcelStyle(), "numberFormat", numberFormat))
	formatRow := int64(len(excelXMLEscapingStrings) + 1)
	sheet = excelTestValue(ExcelSheetStyle(sheet, excelTestValue(ExcelSheetRange(sheet, formatRow, 1, formatRow, 1)), style))
	workbook = excelTestValue(ExcelWorkbookWithSheet(workbook, sheet))
	return workbook
}

func verifyExcelXMLEscapingWorkbook(t *testing.T, workbookText, sheetName string) {
	t.Helper()
	sheet := excelTestValue(ExcelWorkbookSheet(workbookText, sheetName))
	for index, want := range excelXMLEscapingStrings {
		cell := excelTestValue(ExcelSheetCell(sheet, int64(index+1), 1))
		got := excelTestValue(ExcelCellString(cell))
		if got != want {
			t.Fatalf("String round trip at row %d: want %q, got %q", index+1, want, got)
		}
	}
	for index, want := range excelXMLEscapingFormulas {
		cell := excelTestValue(ExcelSheetCell(sheet, int64(index+1), 2))
		got := excelTestValue(ExcelCellFormula(cell))
		if got != want {
			t.Fatalf("Formula round trip at row %d: want %q, got %q", index+1, want, got)
		}
	}
	if got := excelTestValue(ExcelSheetName(sheet)); got != sheetName {
		t.Fatalf("Sheet name round trip: want %q, got %q", sheetName, got)
	}
}

// TestExcelXMLEscapingExternalRegressionRoundTrip is the required external
// regression: save -> independently parse every XML part -> read -> verify
// exact semantic equality -> save again -> parse again -> read again ->
// verify equality again, for the full QA-reported character matrix.
func TestExcelXMLEscapingExternalRegressionRoundTrip(t *testing.T) {
	sheetName := `A & B <legal> it's ok`
	workbook := buildExcelXMLEscapingWorkbook(t)

	directory := t.TempDir()
	firstPath := filepath.Join(directory, "Excel XML Escaping QA.xlsx")
	if err := ExcelWorkbookSave(workbook, firstPath); err != nil {
		t.Fatalf("first save: %v", err)
	}
	firstData := excelTestReadFile(t, firstPath)
	excelParseEveryXMLPart(t, firstData)

	loaded := excelTestValue(ExcelRead(firstPath))
	verifyExcelXMLEscapingWorkbook(t, loaded, sheetName)

	secondPath := filepath.Join(directory, "Excel XML Escaping QA (resaved).xlsx")
	if err := ExcelWorkbookSave(loaded, secondPath); err != nil {
		t.Fatalf("second save: %v", err)
	}
	secondData := excelTestReadFile(t, secondPath)
	excelParseEveryXMLPart(t, secondData)

	reloaded := excelTestValue(ExcelRead(secondPath))
	verifyExcelXMLEscapingWorkbook(t, reloaded, sheetName)
}

// TestExcelWorksheetDimension exercises every used-range scenario the
// worksheet's derived <dimension> element must report, and confirms
// row-height/column-width metadata alone never extends it.
func TestExcelWorksheetDimension(t *testing.T) {
	dimensionRef := func(worksheetXML string) string {
		start := strings.Index(worksheetXML, `<dimension ref="`)
		if start < 0 {
			t.Fatalf("worksheet XML has no <dimension> element:\n%s", worksheetXML)
		}
		start += len(`<dimension ref="`)
		end := strings.Index(worksheetXML[start:], `"`)
		if end < 0 {
			t.Fatalf("malformed <dimension> element:\n%s", worksheetXML)
		}
		return worksheetXML[start : start+end]
	}
	dimensionBeforeSheetData := func(worksheetXML string) bool {
		dimensionIndex := strings.Index(worksheetXML, "<dimension")
		sheetDataIndex := strings.Index(worksheetXML, "<sheetData")
		return dimensionIndex >= 0 && sheetDataIndex >= 0 && dimensionIndex < sheetDataIndex
	}

	catalog := excelStyleCatalog{IDs: make(map[string]int)}

	t.Run("empty Sheet", func(t *testing.T) {
		sheet := excelTestValue(ExcelWorkbookSheet(excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "S")), "S"))
		xmlText := excelWorksheetXML(excelTestDecodeSheet(t, sheet), catalog)
		if got := dimensionRef(xmlText); got != "A1" {
			t.Fatalf("empty Sheet dimension = %q; want A1", got)
		}
		if !dimensionBeforeSheetData(xmlText) {
			t.Fatal("dimension must precede sheetData")
		}
	})

	t.Run("only A1", func(t *testing.T) {
		workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "S"))
		sheet := excelTestValue(ExcelWorkbookSheet(workbook, "S"))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, 1, 1, excelTestValue(ExcelFromString("x"))))
		xmlText := excelWorksheetXML(excelTestDecodeSheet(t, sheet), catalog)
		if got := dimensionRef(xmlText); got != "A1" {
			t.Fatalf("A1-only dimension = %q; want A1", got)
		}
	})

	t.Run("only B3", func(t *testing.T) {
		workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "S"))
		sheet := excelTestValue(ExcelWorkbookSheet(workbook, "S"))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, 3, 2, excelTestValue(ExcelFromString("x"))))
		xmlText := excelWorksheetXML(excelTestDecodeSheet(t, sheet), catalog)
		if got := dimensionRef(xmlText); got != "B3" {
			t.Fatalf("B3-only dimension = %q; want B3", got)
		}
	})

	t.Run("rectangular used range", func(t *testing.T) {
		workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "S"))
		sheet := excelTestValue(ExcelWorkbookSheet(workbook, "S"))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, 1, 1, excelTestValue(ExcelFromString("x"))))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, 20, 4, excelTestValue(ExcelFromString("y"))))
		xmlText := excelWorksheetXML(excelTestDecodeSheet(t, sheet), catalog)
		if got := dimensionRef(xmlText); got != "A1:D20" {
			t.Fatalf("rectangular dimension = %q; want A1:D20", got)
		}
	})

	t.Run("Formula extends range", func(t *testing.T) {
		workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "S"))
		sheet := excelTestValue(ExcelWorkbookSheet(workbook, "S"))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, 1, 1, excelTestValue(ExcelFromString("x"))))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, 5, 5, excelTestValue(ExcelFormula("=A1"))))
		xmlText := excelWorksheetXML(excelTestDecodeSheet(t, sheet), catalog)
		if got := dimensionRef(xmlText); got != "A1:E5" {
			t.Fatalf("Formula-extended dimension = %q; want A1:E5", got)
		}
	})

	t.Run("styled Blank extends range", func(t *testing.T) {
		workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "S"))
		sheet := excelTestValue(ExcelWorkbookSheet(workbook, "S"))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, 1, 1, excelTestValue(ExcelFromString("x"))))
		style := excelTestValue(ExcelStyleBool(ExcelStyle(), "bold", true))
		sheet = excelTestValue(ExcelSheetStyle(sheet, excelTestValue(ExcelSheetRange(sheet, 4, 4, 4, 4)), style))
		xmlText := excelWorksheetXML(excelTestDecodeSheet(t, sheet), catalog)
		if got := dimensionRef(xmlText); got != "A1:D4" {
			t.Fatalf("styled-Blank-extended dimension = %q; want A1:D4", got)
		}
	})

	t.Run("merge extends range", func(t *testing.T) {
		workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "S"))
		sheet := excelTestValue(ExcelWorkbookSheet(workbook, "S"))
		sheet = excelTestValue(ExcelSheetSetCell(sheet, 1, 1, excelTestValue(ExcelFromString("x"))))
		sheet = excelTestValue(ExcelSheetMerge(sheet, excelTestValue(ExcelSheetRange(sheet, 6, 1, 8, 3))))
		xmlText := excelWorksheetXML(excelTestDecodeSheet(t, sheet), catalog)
		if got := dimensionRef(xmlText); got != "A1:C8" {
			t.Fatalf("merge-extended dimension = %q; want A1:C8", got)
		}
	})

	t.Run("width/height metadata alone does not extend range", func(t *testing.T) {
		workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "S"))
		sheet := excelTestValue(ExcelWorkbookSheet(workbook, "S"))
		sheet = excelTestValue(ExcelSheetColumnWidth(sheet, 10, 30))
		sheet = excelTestValue(ExcelSheetRowHeight(sheet, 50, 20))
		xmlText := excelWorksheetXML(excelTestDecodeSheet(t, sheet), catalog)
		if got := dimensionRef(xmlText); got != "A1" {
			t.Fatalf("width/height-only dimension = %q; want A1 (unaffected by dimension metadata)", got)
		}
	})
}

func excelTestDecodeSheet(t *testing.T, sheetText string) excelSheetData {
	t.Helper()
	sheet, err := excelSheet(sheetText)
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}
