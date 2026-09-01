package ahdruntime

import (
	"archive/zip"
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func excelTestValue(value string, err error) string {
	if err != nil {
		panic(err)
	}
	return value
}

func TestExcelSemanticRoundTripAndDeterministicPackage(t *testing.T) {
	workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "Öğrenciler"))
	workbook = excelTestValue(ExcelWorkbookAddSheet(workbook, "Statistics"))
	sheet := excelTestValue(ExcelWorkbookSheet(workbook, "Öğrenciler"))
	original := sheet
	sheet = excelTestValue(ExcelSheetSetCell(sheet, 1, 1, excelTestValue(ExcelFromString("Üniversite Sonuçları"))))
	sheet = excelTestValue(ExcelSheetSetCell(sheet, 1, 2, excelTestValue(ExcelFromString("Do not lose me"))))
	merge := excelTestValue(ExcelSheetRange(sheet, 1, 1, 1, 2))
	if _, err := ExcelSheetMerge(sheet, merge); err == nil || !strings.Contains(err.Error(), "would lose") {
		t.Fatalf("unsafe merge error = %v", err)
	}
	if got := excelTestValue(ExcelSheetCell(sheet, 1, 2)); excelTestValue(ExcelCellString(got)) != "Do not lose me" {
		t.Fatal("failed merge changed the source Sheet")
	}
	sheet = excelTestValue(ExcelSheetSetCell(sheet, 1, 2, ExcelBlank()))
	sheet = excelTestValue(ExcelSheetMerge(sheet, merge))
	sheet = excelTestValue(ExcelSheetSetRow(sheet, 2, 1, []string{
		excelTestValue(ExcelFromString("Ali")), ExcelFromInt(91), excelTestValue(ExcelFromReal(91.5)), ExcelFromBool(true), excelTestValue(ExcelFormula("=SUM(B2:C2)")),
	}))
	sheet = excelTestValue(ExcelSheetSetCell(sheet, 3, 1, excelTestValue(ExcelFromReal(1.0))))
	style := ExcelStyle()
	style = excelTestValue(ExcelStyleString(style, "fillColor", "#1F4E79"))
	style = excelTestValue(ExcelStyleString(style, "textColor", "#FFFFFF"))
	style = excelTestValue(ExcelStyleBool(style, "bold", true))
	style = excelTestValue(ExcelStyleBool(style, "italic", true))
	style = excelTestValue(ExcelStyleBool(style, "underline", true))
	style = excelTestValue(ExcelStyleFontSize(style, 14))
	style = excelTestValue(ExcelStyleString(style, "horizontal", "center"))
	style = excelTestValue(ExcelStyleString(style, "vertical", "center"))
	style = excelTestValue(ExcelStyleBool(style, "wrap", true))
	style = excelTestValue(ExcelStyleString(style, "numberFormat", "0.00"))
	style = excelTestValue(ExcelStyleBorder(style, "thin", "#000000"))
	sheet = excelTestValue(ExcelSheetStyle(sheet, merge, style))
	sheet = excelTestValue(ExcelSheetColumnWidth(sheet, 1, 22.5))
	sheet = excelTestValue(ExcelSheetRowHeight(sheet, 1, 28))
	workbook = excelTestValue(ExcelWorkbookWithSheet(workbook, sheet))
	if original == sheet {
		t.Fatal("Sheet transformation did not produce a new snapshot")
	}

	directory := t.TempDir()
	first := filepath.Join(directory, "first.xlsx")
	second := filepath.Join(directory, "second.xlsx")
	if err := ExcelWorkbookSave(workbook, first); err != nil {
		t.Fatal(err)
	}
	if err := ExcelWorkbookSave(workbook, second); err != nil {
		t.Fatal(err)
	}
	left, _ := os.ReadFile(first)
	right, _ := os.ReadFile(second)
	if !bytes.Equal(left, right) {
		t.Fatal("same Workbook did not produce byte-identical XLSX packages")
	}
	loaded := excelTestValue(ExcelRead(first))
	if sheets := strings.Join(func() []string {
		values, err := ExcelWorkbookSheets(loaded)
		if err != nil {
			t.Fatal(err)
		}
		return values
	}(), ","); sheets != "Öğrenciler,Statistics" {
		t.Fatalf("sheet order = %q", sheets)
	}
	loadedSheet := excelTestValue(ExcelWorkbookSheet(loaded, "Öğrenciler"))
	formula := excelTestValue(ExcelSheetCell(loadedSheet, 2, 5))
	if got := excelTestValue(ExcelCellFormula(formula)); got != "=SUM(B2:C2)" {
		t.Fatalf("formula = %q", got)
	}
	if got := excelTestValue(ExcelSheetCell(loadedSheet, 2, 3)); excelTestValue(ExcelCellKind(got)) != "Real" {
		t.Fatal("Real lexical intent was not preserved")
	}
	if got := excelTestValue(ExcelSheetCell(loadedSheet, 3, 1)); excelTestValue(ExcelCellKind(got)) != "Real" {
		t.Fatal("integral Real lexical intent was not preserved")
	}
	resaved := filepath.Join(directory, "resaved.xlsx")
	if err := ExcelWorkbookSave(loaded, resaved); err != nil {
		t.Fatal(err)
	}
	if _, err := ExcelRead(resaved); err != nil {
		t.Fatal(err)
	}
}

func TestExcelStrictRangeMergeAndCellSafety(t *testing.T) {
	workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "Data"))
	sheet := excelTestValue(ExcelWorkbookSheet(workbook, "Data"))
	unchanged := excelTestValue(ExcelSheetStyle(sheet, excelTestValue(ExcelSheetRange(sheet, 1, 1, 2, 2)), ExcelStyle()))
	if used, err := ExcelSheetUsedRange(unchanged); err != nil || used != nil {
		t.Fatalf("empty style patch expanded usedRange: used=%v err=%v", used, err)
	}
	cellRange := excelTestValue(ExcelSheetRange(sheet, 1, 1, 2, 3))
	if _, err := ExcelSheetSetRange(sheet, cellRange, [][]string{{ExcelFromInt(1), ExcelFromInt(2), ExcelFromInt(3)}, {ExcelFromInt(4), ExcelFromInt(5)}}); err == nil {
		t.Fatal("ragged setRange succeeded")
	}
	if used, err := ExcelSheetUsedRange(sheet); err != nil || used != nil {
		t.Fatalf("failed setRange changed source: used=%v err=%v", used, err)
	}
	merged := excelTestValue(ExcelSheetMerge(sheet, excelTestValue(ExcelSheetRange(sheet, 1, 1, 1, 2))))
	if _, err := ExcelSheetSetCell(merged, 1, 2, ExcelFromInt(9)); err == nil {
		t.Fatal("merged non-anchor write succeeded")
	}
	if _, err := ExcelFromReal(math.Inf(1)); err == nil {
		t.Fatal("non-finite Real succeeded")
	}
	literal := excelTestValue(ExcelFromString("=SUM(A1:A100)"))
	if kind := excelTestValue(ExcelCellKind(literal)); kind != "String" {
		t.Fatalf("formula-like String kind = %q", kind)
	}
	if _, err := ExcelFormula("SUM(A1:A3)"); err == nil {
		t.Fatal("formula without = succeeded")
	}
	if _, err := ExcelCellInt(excelTestValue(ExcelFromReal(1.0))); err == nil {
		t.Fatal("Real narrowed to Int")
	}
}

func TestExcelStyleOverlayPreservesUnspecifiedProperties(t *testing.T) {
	workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "Style"))
	sheet := excelTestValue(ExcelWorkbookSheet(workbook, "Style"))
	cellRange := excelTestValue(ExcelSheetRange(sheet, 1, 1, 1, 1))
	fill := excelTestValue(ExcelStyleString(ExcelStyle(), "fillColor", "#1F4E79"))
	sheet = excelTestValue(ExcelSheetStyle(sheet, cellRange, fill))
	bold := excelTestValue(ExcelStyleBool(ExcelStyle(), "bold", true))
	sheet = excelTestValue(ExcelSheetStyle(sheet, cellRange, bold))
	decoded, err := excelSheet(sheet)
	if err != nil {
		t.Fatal(err)
	}
	style := decoded.Cells[0].Style
	if style == nil || style.FillColor == nil || *style.FillColor != "#1F4E79" || style.Bold == nil || !*style.Bold {
		t.Fatalf("style overlay = %#v", style)
	}
	disable := excelTestValue(ExcelStyleBool(ExcelStyle(), "bold", false))
	sheet = excelTestValue(ExcelSheetStyle(sheet, cellRange, disable))
	decoded, _ = excelSheet(sheet)
	style = decoded.Cells[0].Style
	if style.Bold == nil || *style.Bold || style.FillColor == nil || *style.FillColor != "#1F4E79" {
		t.Fatalf("bold(false) overlay = %#v", style)
	}
}

func TestExcelReaderRejectsUnsafeAndDuplicateZIPMembers(t *testing.T) {
	makeZIP := func(names []string) []byte {
		var buffer bytes.Buffer
		writer := zip.NewWriter(&buffer)
		for _, name := range names {
			part, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = part.Write([]byte("x"))
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return buffer.Bytes()
	}
	if _, err := excelReadPackage(makeZIP([]string{"../escape.xml"})); err == nil {
		t.Fatal("unsafe ZIP member succeeded")
	}
	if _, err := excelReadPackage(makeZIP([]string{"same.xml", "same.xml"})); err == nil {
		t.Fatal("duplicate ZIP member succeeded")
	}
}

func TestExcelReaderSupportsSharedStringsAndLexicalNumbers(t *testing.T) {
	entries := [][2]string{
		{"[Content_Types].xml", excelXMLHeader + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`},
		{"_rels/.rels", excelPackageRelsXML},
		{"xl/workbook.xml", excelXMLHeader + `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Data" sheetId="1" r:id="rId1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", excelXMLHeader + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/></Relationships>`},
		{"xl/sharedStrings.xml", excelXMLHeader + `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><si><t>plain</t></si><si><r><t>rich </t></r><r><t>text</t></r></si></sst>`},
		{"xl/worksheets/sheet1.xml", excelXMLHeader + `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c><c r="C1"><v>91</v></c><c r="D1"><v>1.0</v></c><c r="E1"><v>1e3</v></c><c r="F1" t="b"><v>1</v></c><c r="G1"><f>A1&amp;B1</f><v>cached</v></c></row></sheetData></worksheet>`},
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		part, err := writer.Create(entry[0])
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte(entry[1]))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	workbookData, err := excelReadPackage(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	workbook := excelEncode(workbookData)
	sheet := excelTestValue(ExcelWorkbookSheet(workbook, "Data"))
	checks := []struct {
		column     int64
		kind, text string
	}{
		{1, "String", "plain"}, {2, "String", "rich text"}, {3, "Int", ""}, {4, "Real", ""},
		{5, "Real", ""}, {6, "Bool", ""}, {7, "Formula", "=A1&B1"},
	}
	for _, check := range checks {
		cell := excelTestValue(ExcelSheetCell(sheet, 1, check.column))
		if kind := excelTestValue(ExcelCellKind(cell)); kind != check.kind {
			t.Fatalf("column %d kind=%q want=%q", check.column, kind, check.kind)
		}
		if check.kind == "String" && excelTestValue(ExcelCellString(cell)) != check.text {
			t.Fatalf("column %d String mismatch", check.column)
		}
		if check.kind == "Formula" && excelTestValue(ExcelCellFormula(cell)) != check.text {
			t.Fatalf("column %d Formula mismatch", check.column)
		}
	}
}

func TestExcelReaderRejectsAdvancedFormulaAndDTD(t *testing.T) {
	if err := excelParseXML([]byte(`<!DOCTYPE x><x/>`), &struct{}{}, "Excel.read", "test.xml"); err == nil {
		t.Fatal("DTD was accepted")
	}
	if _, err := excelDecodeWorksheetCell(excelWorksheetCellXML{Formula: &struct {
		Type string `xml:"t,attr"`
		Text string `xml:",chardata"`
	}{Type: "shared", Text: "A1+1"}}, nil); err == nil {
		t.Fatal("shared formula was accepted")
	}
}

func TestExcelFailedSavePreservesExistingDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.xlsx")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ExcelWorkbookSave(ExcelNew(), path); err == nil {
		t.Fatal("empty Workbook save succeeded")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "keep" {
		t.Fatalf("destination changed: %q err=%v", content, err)
	}
}

func TestExcelCoordinateRangeAndWorkbookValidationMatrix(t *testing.T) {
	workbook := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "Unicode Şube"))
	for _, name := range []string{"unicode şube", "", "bad/name", strings.Repeat("x", 32), "'edge"} {
		if _, err := ExcelWorkbookAddSheet(workbook, name); err == nil {
			t.Fatalf("invalid or duplicate Sheet name %q succeeded", name)
		}
	}
	sheet := excelTestValue(ExcelWorkbookSheet(workbook, "Unicode Şube"))
	last := excelTestValue(ExcelSheetSetCell(sheet, excelMaxRow, excelMaxColumn, ExcelFromInt(1)))
	if got := excelTestValue(ExcelSheetRange(last, excelMaxRow, excelMaxColumn, excelMaxRow, excelMaxColumn)); excelTestValue(ExcelRangeAddress(got)) != "XFD1048576:XFD1048576" {
		t.Fatal("last Range address is wrong")
	}
	for _, coordinate := range [][2]int64{{0, 1}, {1, 0}, {-1, 1}, {excelMaxRow + 1, 1}, {1, excelMaxColumn + 1}} {
		if _, err := ExcelSheetCell(sheet, coordinate[0], coordinate[1]); err == nil {
			t.Fatalf("invalid coordinate %v succeeded", coordinate)
		}
	}
	for _, bounds := range [][4]int64{{2, 1, 1, 1}, {1, 2, 1, 1}, {0, 1, 1, 1}, {1, 1, excelMaxRow + 1, 1}} {
		if _, err := ExcelSheetRange(sheet, bounds[0], bounds[1], bounds[2], bounds[3]); err == nil {
			t.Fatalf("invalid Range %v succeeded", bounds)
		}
	}
	if _, err := ExcelWorkbookSheet(workbook, "missing"); err == nil {
		t.Fatal("unknown Sheet succeeded")
	}
	missing := excelTestValue(ExcelWorkbookAddSheet(ExcelNew(), "Other"))
	other := excelTestValue(ExcelWorkbookSheet(missing, "Other"))
	if _, err := ExcelWorkbookWithSheet(workbook, other); err == nil {
		t.Fatal("withSheet added a missing Sheet")
	}
}

func TestExcelCellFormulaAndStyleValidationMatrix(t *testing.T) {
	cells := []struct{ text, kind string }{
		{ExcelBlank(), "Blank"}, {excelTestValue(ExcelFromString("Türkçe 世界")), "String"},
		{ExcelFromInt(0), "Int"}, {ExcelFromInt(-9223372036854775807 - 1), "Int"},
		{excelTestValue(ExcelFromReal(-3.5)), "Real"}, {ExcelFromBool(false), "Bool"},
		{excelTestValue(ExcelFormula(`=IF(A1>0,"yes","no")`)), "Formula"},
	}
	for _, check := range cells {
		if kind := excelTestValue(ExcelCellKind(check.text)); kind != check.kind {
			t.Fatalf("kind=%q want=%q", kind, check.kind)
		}
	}
	if realValue := func() float64 {
		value, err := ExcelCellReal(ExcelFromInt(91))
		if err != nil {
			t.Fatal(err)
		}
		return value
	}(); realValue != 91 {
		t.Fatal("Int-to-Real widening failed")
	}
	for _, formula := range []string{"=SUM(A1:A3)", "=A1+B1", `=IF(A1>0,"yes","no")`, "=$A$1*B2", `=\"Türkçe\"`} {
		if _, err := ExcelFormula(formula); err != nil {
			t.Fatalf("valid Formula %q: %v", formula, err)
		}
	}
	for _, formula := range []string{"SUM(A1:A3)", "=", "=bad\x01", "=" + strings.Repeat("x", excelMaxFormulaText)} {
		if _, err := ExcelFormula(formula); err == nil {
			t.Fatalf("invalid Formula %q succeeded", formula[:min(len(formula), 20)])
		}
	}
	if _, err := ExcelFromString(strings.Repeat("x", excelMaxCellText+1)); err == nil {
		t.Fatal("oversized String succeeded")
	}
	for _, invalid := range []func() error{
		func() error { _, err := ExcelStyleString(ExcelStyle(), "textColor", "red"); return err },
		func() error { _, err := ExcelStyleString(ExcelStyle(), "fillColor", "#abcdef"); return err },
		func() error { _, err := ExcelStyleString(ExcelStyle(), "horizontal", "justify"); return err },
		func() error { _, err := ExcelStyleString(ExcelStyle(), "vertical", "middle"); return err },
		func() error { _, err := ExcelStyleBorder(ExcelStyle(), "hair", "#000000"); return err },
		func() error { _, err := ExcelStyleFontSize(ExcelStyle(), 0); return err },
		func() error { _, err := ExcelStyleFontSize(ExcelStyle(), math.NaN()); return err },
	} {
		if err := invalid(); err == nil {
			t.Fatal("invalid style succeeded")
		}
	}
}
