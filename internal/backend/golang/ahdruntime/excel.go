package ahdruntime

// The Excel standard module's direct XLSX implementation. This file is also
// emitted verbatim into native programs (with only its package clause
// rewritten), so it intentionally depends on the Go standard library and the
// sibling AhdCode runtime only.

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	excelMaxRow           = 1048576
	excelMaxColumn        = 16384
	excelMaxCellText      = 32767
	excelMaxFormulaText   = 8192
	excelMaxArchiveBytes  = 128 << 20
	excelMaxEntryBytes    = 32 << 20
	excelMaxTotalBytes    = 128 << 20
	excelMaxEntries       = 2048
	excelMaxCompressRatio = 1000
)

type excelWorkbookData struct {
	Sheets []excelSheetData `json:"sheets"`
}

type excelSheetData struct {
	Name         string                 `json:"name"`
	Cells        []excelCellEntry       `json:"cells,omitempty"`
	Merges       []excelRangeData       `json:"merges,omitempty"`
	ColumnWidths []excelColumnDimension `json:"columnWidths,omitempty"`
	RowHeights   []excelRowDimension    `json:"rowHeights,omitempty"`
}

type excelCellEntry struct {
	Row    int64           `json:"row"`
	Column int64           `json:"column"`
	Cell   excelCellData   `json:"cell"`
	Style  *excelStyleData `json:"style,omitempty"`
}

type excelCellData struct {
	Kind string  `json:"kind"`
	Text string  `json:"text,omitempty"`
	Int  int64   `json:"int,omitempty"`
	Real float64 `json:"real,omitempty"`
	Bool bool    `json:"bool,omitempty"`
}

type excelRangeData struct {
	StartRow    int64 `json:"startRow"`
	StartColumn int64 `json:"startColumn"`
	EndRow      int64 `json:"endRow"`
	EndColumn   int64 `json:"endColumn"`
}

// Pointers are the style patch's tri-state fields: nil means unspecified;
// false means explicitly disabled. Applied cell styles use the same shape so
// overlays can preserve every property they do not mention.
type excelStyleData struct {
	Bold         *bool    `json:"bold,omitempty"`
	Italic       *bool    `json:"italic,omitempty"`
	Underline    *bool    `json:"underline,omitempty"`
	FontSize     *float64 `json:"fontSize,omitempty"`
	TextColor    *string  `json:"textColor,omitempty"`
	FillColor    *string  `json:"fillColor,omitempty"`
	Horizontal   *string  `json:"horizontal,omitempty"`
	Vertical     *string  `json:"vertical,omitempty"`
	Wrap         *bool    `json:"wrap,omitempty"`
	NumberFormat *string  `json:"numberFormat,omitempty"`
	BorderStyle  *string  `json:"borderStyle,omitempty"`
	BorderColor  *string  `json:"borderColor,omitempty"`
}

type excelColumnDimension struct {
	Column int64   `json:"column"`
	Width  float64 `json:"width"`
}

type excelRowDimension struct {
	Row    int64   `json:"row"`
	Height float64 `json:"height"`
}

func excelEncode(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("ahdcode: Excel internal value could not be encoded: " + err.Error())
	}
	return string(encoded)
}

func excelDecode(text string, target any, kind string) error {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%s storage is corrupted", kind)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%s storage is corrupted", kind)
	}
	return nil
}

func excelWorkbook(text string) (excelWorkbookData, error) {
	var value excelWorkbookData
	if err := excelDecode(text, &value, "Workbook"); err != nil {
		return value, err
	}
	if err := excelValidateWorkbook(value); err != nil {
		return value, err
	}
	return value, nil
}

func excelSheet(text string) (excelSheetData, error) {
	var value excelSheetData
	if err := excelDecode(text, &value, "Sheet"); err != nil {
		return value, err
	}
	if err := excelValidateSheet(value); err != nil {
		return value, err
	}
	return value, nil
}

func excelCell(text string) (excelCellData, error) {
	var value excelCellData
	if err := excelDecode(text, &value, "Cell"); err != nil {
		return value, err
	}
	if err := excelValidateCell(value); err != nil {
		return value, err
	}
	return value, nil
}

func excelRange(text string) (excelRangeData, error) {
	var value excelRangeData
	if err := excelDecode(text, &value, "Range"); err != nil {
		return value, err
	}
	if err := excelValidateRange(value); err != nil {
		return value, err
	}
	return value, nil
}

func excelStyle(text string) (excelStyleData, error) {
	var value excelStyleData
	if err := excelDecode(text, &value, "CellStyle"); err != nil {
		return value, err
	}
	if err := excelValidateStyle(value); err != nil {
		return value, err
	}
	return value, nil
}

func excelValidXMLText(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if r == 0x9 || r == 0xa || r == 0xd || (r >= 0x20 && r <= 0xd7ff) ||
			(r >= 0xe000 && r <= 0xfffd) || (r >= 0x10000 && r <= 0x10ffff) {
			continue
		}
		return false
	}
	return true
}

func excelValidateSheetName(name string) error {
	if name == "" {
		return errors.New("Sheet name must not be empty")
	}
	if utf8.RuneCountInString(name) > 31 {
		return errors.New("Sheet name must contain at most 31 Unicode characters")
	}
	if !excelValidXMLText(name) {
		return errors.New("Sheet name contains text that cannot be represented safely in XLSX")
	}
	if strings.ContainsAny(name, `:\/?*[]`) {
		return errors.New(`Sheet name contains one of the invalid characters : \ / ? * [ ]`)
	}
	if strings.HasPrefix(name, "'") || strings.HasSuffix(name, "'") {
		return errors.New("Sheet name must not begin or end with an apostrophe")
	}
	return nil
}

func excelValidateCoordinate(row, column int64) error {
	if row < 1 || row > excelMaxRow {
		return fmt.Errorf("row must be between 1 and %d", excelMaxRow)
	}
	if column < 1 || column > excelMaxColumn {
		return fmt.Errorf("column must be between 1 and %d", excelMaxColumn)
	}
	return nil
}

func excelValidateRange(value excelRangeData) error {
	if err := excelValidateCoordinate(value.StartRow, value.StartColumn); err != nil {
		return fmt.Errorf("Range start: %w", err)
	}
	if err := excelValidateCoordinate(value.EndRow, value.EndColumn); err != nil {
		return fmt.Errorf("Range end: %w", err)
	}
	if value.StartRow > value.EndRow {
		return errors.New("Range startRow must not exceed endRow")
	}
	if value.StartColumn > value.EndColumn {
		return errors.New("Range startColumn must not exceed endColumn")
	}
	return nil
}

func excelValidateCell(value excelCellData) error {
	switch value.Kind {
	case "Blank", "Int", "Bool":
		return nil
	case "String":
		if !excelValidXMLText(value.Text) {
			return errors.New("String Cell contains text that cannot be represented safely in XLSX")
		}
		if utf8.RuneCountInString(value.Text) > excelMaxCellText {
			return fmt.Errorf("String Cell exceeds Excel's %d-character limit", excelMaxCellText)
		}
		return nil
	case "Real":
		if math.IsNaN(value.Real) || math.IsInf(value.Real, 0) {
			return errors.New("Real Cell requires a finite value")
		}
		return nil
	case "Formula":
		return excelValidateFormula(value.Text)
	default:
		return errors.New("Cell kind is not one of Blank, String, Int, Real, Bool, or Formula")
	}
}

func excelValidateFormula(expression string) error {
	if !strings.HasPrefix(expression, "=") {
		return errors.New("Formula expression must begin with =")
	}
	if utf8.RuneCountInString(expression) <= 1 {
		return errors.New("Formula expression must contain text after =")
	}
	if utf8.RuneCountInString(expression) > excelMaxFormulaText {
		return fmt.Errorf("Formula expression exceeds Excel's %d-character limit", excelMaxFormulaText)
	}
	if !excelValidXMLText(expression) {
		return errors.New("Formula expression contains text that cannot be represented safely in XLSX")
	}
	return nil
}

func excelValidColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func excelValidateStyle(value excelStyleData) error {
	if value.FontSize != nil && (*value.FontSize <= 0 || *value.FontSize > 409 || math.IsNaN(*value.FontSize) || math.IsInf(*value.FontSize, 0)) {
		return errors.New("CellStyle fontSize must be a positive finite Real no greater than 409")
	}
	for _, color := range []*string{value.TextColor, value.FillColor, value.BorderColor} {
		if color != nil && !excelValidColor(*color) {
			return errors.New("CellStyle colors must use uppercase #RRGGBB spelling")
		}
	}
	if value.Horizontal != nil && *value.Horizontal != "left" && *value.Horizontal != "center" && *value.Horizontal != "right" {
		return errors.New("CellStyle horizontal must be left, center, or right")
	}
	if value.Vertical != nil && *value.Vertical != "top" && *value.Vertical != "center" && *value.Vertical != "bottom" {
		return errors.New("CellStyle vertical must be top, center, or bottom")
	}
	if value.NumberFormat != nil {
		if *value.NumberFormat == "" {
			return errors.New("CellStyle numberFormat must not be empty")
		}
		if utf8.RuneCountInString(*value.NumberFormat) > 255 || !excelValidXMLText(*value.NumberFormat) {
			return errors.New("CellStyle numberFormat cannot be represented safely in XLSX")
		}
	}
	if value.BorderStyle != nil {
		allowed := map[string]bool{"none": true, "thin": true, "medium": true, "thick": true, "dashed": true, "dotted": true, "double": true}
		if !allowed[*value.BorderStyle] {
			return errors.New("CellStyle border style must be none, thin, medium, thick, dashed, dotted, or double")
		}
		if value.BorderColor == nil {
			return errors.New("CellStyle border requires a color")
		}
	}
	return nil
}

func excelValidateWorkbook(value excelWorkbookData) error {
	for index, sheet := range value.Sheets {
		if err := excelValidateSheet(sheet); err != nil {
			return err
		}
		for previous := 0; previous < index; previous++ {
			if strings.EqualFold(value.Sheets[previous].Name, sheet.Name) {
				return fmt.Errorf("Workbook contains duplicate Sheet name %q (names are case-insensitive)", sheet.Name)
			}
		}
	}
	return nil
}

func excelValidateSheet(value excelSheetData) error {
	if err := excelValidateSheetName(value.Name); err != nil {
		return err
	}
	previousRow, previousColumn := int64(0), int64(0)
	for _, entry := range value.Cells {
		if err := excelValidateCoordinate(entry.Row, entry.Column); err != nil {
			return fmt.Errorf("Sheet %q Cell: %w", value.Name, err)
		}
		if entry.Row < previousRow || (entry.Row == previousRow && entry.Column <= previousColumn) {
			return fmt.Errorf("Sheet %q Cell coordinates are not unique and sorted", value.Name)
		}
		if err := excelValidateCell(entry.Cell); err != nil {
			return fmt.Errorf("Sheet %q Cell %s: %w", value.Name, excelCellAddress(entry.Row, entry.Column), err)
		}
		if entry.Style != nil {
			if err := excelValidateStyle(*entry.Style); err != nil {
				return fmt.Errorf("Sheet %q Cell %s: %w", value.Name, excelCellAddress(entry.Row, entry.Column), err)
			}
		}
		previousRow, previousColumn = entry.Row, entry.Column
	}
	for _, merge := range value.Merges {
		if err := excelValidateRange(merge); err != nil {
			return fmt.Errorf("Sheet %q merge: %w", value.Name, err)
		}
	}
	for left := 0; left < len(value.Merges); left++ {
		for right := left + 1; right < len(value.Merges); right++ {
			if excelRangesOverlap(value.Merges[left], value.Merges[right]) {
				return fmt.Errorf("Sheet %q merged ranges overlap", value.Name)
			}
		}
	}
	for _, merge := range value.Merges {
		for _, entry := range value.Cells {
			if entry.Row >= merge.StartRow && entry.Row <= merge.EndRow && entry.Column >= merge.StartColumn && entry.Column <= merge.EndColumn &&
				(entry.Row != merge.StartRow || entry.Column != merge.StartColumn) && entry.Cell.Kind != "Blank" {
				return fmt.Errorf("Sheet %q merged range would hide non-Blank Cell %s", value.Name, excelCellAddress(entry.Row, entry.Column))
			}
		}
	}
	for _, dimension := range value.ColumnWidths {
		if err := excelValidateCoordinate(1, dimension.Column); err != nil || dimension.Width <= 0 || dimension.Width > 255 || math.IsNaN(dimension.Width) || math.IsInf(dimension.Width, 0) {
			return fmt.Errorf("Sheet %q column width is invalid", value.Name)
		}
	}
	for _, dimension := range value.RowHeights {
		if err := excelValidateCoordinate(dimension.Row, 1); err != nil || dimension.Height <= 0 || dimension.Height > 409.5 || math.IsNaN(dimension.Height) || math.IsInf(dimension.Height, 0) {
			return fmt.Errorf("Sheet %q row height is invalid", value.Name)
		}
	}
	return nil
}

func excelBool(value bool) *bool       { result := value; return &result }
func excelReal(value float64) *float64 { result := value; return &result }
func excelText(value string) *string   { result := value; return &result }

// Pure semantic entry points are shared by the persistent evaluator. Native
// wrappers below translate their errors into the generated ExcelError Class.
func ExcelNew() string { return excelEncode(excelWorkbookData{Sheets: []excelSheetData{}}) }

func ExcelBlank() string { return excelEncode(excelCellData{Kind: "Blank"}) }

func ExcelFromString(value string) (string, error) {
	cell := excelCellData{Kind: "String", Text: value}
	if err := excelValidateCell(cell); err != nil {
		return "", fmt.Errorf("Excel.fromString: %w", err)
	}
	return excelEncode(cell), nil
}

func ExcelFromInt(value int64) string { return excelEncode(excelCellData{Kind: "Int", Int: value}) }

func ExcelFromReal(value float64) (string, error) {
	cell := excelCellData{Kind: "Real", Real: value}
	if err := excelValidateCell(cell); err != nil {
		return "", fmt.Errorf("Excel.fromReal: %w", err)
	}
	return excelEncode(cell), nil
}

func ExcelFromBool(value bool) string { return excelEncode(excelCellData{Kind: "Bool", Bool: value}) }

func ExcelFormula(expression string) (string, error) {
	cell := excelCellData{Kind: "Formula", Text: expression}
	if err := excelValidateCell(cell); err != nil {
		return "", fmt.Errorf("Excel.formula: %w", err)
	}
	return excelEncode(cell), nil
}

func ExcelStyle() string { return excelEncode(excelStyleData{}) }

func ExcelWorkbookAddSheet(workbookText, name string) (string, error) {
	workbook, err := excelWorkbook(workbookText)
	if err != nil {
		return "", fmt.Errorf("Workbook.addSheet: %w", err)
	}
	if err := excelValidateSheetName(name); err != nil {
		return "", fmt.Errorf("Workbook.addSheet: %w", err)
	}
	for _, sheet := range workbook.Sheets {
		if strings.EqualFold(sheet.Name, name) {
			return "", fmt.Errorf("Workbook.addSheet: Sheet name %q already exists (names are case-insensitive)", name)
		}
	}
	workbook.Sheets = append(workbook.Sheets, excelSheetData{Name: name})
	return excelEncode(workbook), nil
}

func ExcelWorkbookSheet(workbookText, name string) (string, error) {
	workbook, err := excelWorkbook(workbookText)
	if err != nil {
		return "", fmt.Errorf("Workbook.sheet: %w", err)
	}
	for _, sheet := range workbook.Sheets {
		if sheet.Name == name {
			return excelEncode(sheet), nil
		}
	}
	return "", fmt.Errorf("Workbook.sheet: Sheet %q was not found", name)
}

func ExcelWorkbookWithSheet(workbookText, sheetText string) (string, error) {
	workbook, err := excelWorkbook(workbookText)
	if err != nil {
		return "", fmt.Errorf("Workbook.withSheet: %w", err)
	}
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return "", fmt.Errorf("Workbook.withSheet: %w", err)
	}
	for index := range workbook.Sheets {
		if workbook.Sheets[index].Name == sheet.Name {
			workbook.Sheets[index] = sheet
			return excelEncode(workbook), nil
		}
	}
	return "", fmt.Errorf("Workbook.withSheet: Sheet %q is not already in the Workbook", sheet.Name)
}

func ExcelWorkbookSheets(workbookText string) ([]string, error) {
	workbook, err := excelWorkbook(workbookText)
	if err != nil {
		return nil, fmt.Errorf("Workbook.sheets: %w", err)
	}
	result := make([]string, len(workbook.Sheets))
	for index, sheet := range workbook.Sheets {
		result[index] = sheet.Name
	}
	return result, nil
}

func ExcelWorkbookSheetCount(workbookText string) (int64, error) {
	workbook, err := excelWorkbook(workbookText)
	if err != nil {
		return 0, fmt.Errorf("Workbook.sheetCount: %w", err)
	}
	return int64(len(workbook.Sheets)), nil
}

func ExcelSheetName(sheetText string) (string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return "", fmt.Errorf("Sheet.name: %w", err)
	}
	return sheet.Name, nil
}

func excelCellIndex(cells []excelCellEntry, row, column int64) (int, bool) {
	index := sort.Search(len(cells), func(index int) bool {
		return cells[index].Row > row || (cells[index].Row == row && cells[index].Column >= column)
	})
	return index, index < len(cells) && cells[index].Row == row && cells[index].Column == column
}

func excelCellAt(sheet excelSheetData, row, column int64) excelCellData {
	if index, found := excelCellIndex(sheet.Cells, row, column); found {
		return sheet.Cells[index].Cell
	}
	return excelCellData{Kind: "Blank"}
}

func ExcelSheetCell(sheetText string, row, column int64) (string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return "", fmt.Errorf("Sheet.cell: %w", err)
	}
	if err := excelValidateCoordinate(row, column); err != nil {
		return "", fmt.Errorf("Sheet.cell: %w", err)
	}
	return excelEncode(excelCellAt(sheet, row, column)), nil
}

func excelIsMergedNonAnchor(sheet excelSheetData, row, column int64) bool {
	for _, cellRange := range sheet.Merges {
		if row >= cellRange.StartRow && row <= cellRange.EndRow && column >= cellRange.StartColumn && column <= cellRange.EndColumn {
			return row != cellRange.StartRow || column != cellRange.StartColumn
		}
	}
	return false
}

func excelSetCell(sheet *excelSheetData, row, column int64, cell excelCellData) error {
	if cell.Kind != "Blank" && excelIsMergedNonAnchor(*sheet, row, column) {
		return fmt.Errorf("cannot write content to merged non-anchor Cell %s", excelCellAddress(row, column))
	}
	index, found := excelCellIndex(sheet.Cells, row, column)
	if found {
		style := sheet.Cells[index].Style
		if cell.Kind == "Blank" && style == nil {
			sheet.Cells = append(sheet.Cells[:index], sheet.Cells[index+1:]...)
		} else {
			sheet.Cells[index].Cell = cell
		}
		return nil
	}
	if cell.Kind == "Blank" {
		return nil
	}
	sheet.Cells = append(sheet.Cells, excelCellEntry{})
	copy(sheet.Cells[index+1:], sheet.Cells[index:])
	sheet.Cells[index] = excelCellEntry{Row: row, Column: column, Cell: cell}
	return nil
}

func ExcelSheetSetCell(sheetText string, row, column int64, cellText string) (string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return "", fmt.Errorf("Sheet.setCell: %w", err)
	}
	if err := excelValidateCoordinate(row, column); err != nil {
		return "", fmt.Errorf("Sheet.setCell: %w", err)
	}
	cell, err := excelCell(cellText)
	if err != nil {
		return "", fmt.Errorf("Sheet.setCell: %w", err)
	}
	if err := excelSetCell(&sheet, row, column, cell); err != nil {
		return "", fmt.Errorf("Sheet.setCell: %w", err)
	}
	return excelEncode(sheet), nil
}

func ExcelSheetRange(sheetText string, startRow, startColumn, endRow, endColumn int64) (string, error) {
	if _, err := excelSheet(sheetText); err != nil {
		return "", fmt.Errorf("Sheet.range: %w", err)
	}
	value := excelRangeData{StartRow: startRow, StartColumn: startColumn, EndRow: endRow, EndColumn: endColumn}
	if err := excelValidateRange(value); err != nil {
		return "", fmt.Errorf("Sheet.range: %w", err)
	}
	return excelEncode(value), nil
}

func ExcelSheetSetRow(sheetText string, row, startColumn int64, cellTexts []string) (string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return "", fmt.Errorf("Sheet.setRow: %w", err)
	}
	if err := excelValidateCoordinate(row, startColumn); err != nil {
		return "", fmt.Errorf("Sheet.setRow: %w", err)
	}
	if len(cellTexts) > 0 && startColumn+int64(len(cellTexts))-1 > excelMaxColumn {
		return "", fmt.Errorf("Sheet.setRow: cells extend beyond column %d", excelMaxColumn)
	}
	cells := make([]excelCellData, len(cellTexts))
	for index, text := range cellTexts {
		cell, decodeErr := excelCell(text)
		if decodeErr != nil {
			return "", fmt.Errorf("Sheet.setRow: value %d: %w", index+1, decodeErr)
		}
		cells[index] = cell
	}
	for index, cell := range cells {
		if err := excelSetCell(&sheet, row, startColumn+int64(index), cell); err != nil {
			return "", fmt.Errorf("Sheet.setRow: %w", err)
		}
	}
	return excelEncode(sheet), nil
}

func ExcelSheetSetRange(sheetText, rangeText string, cellTexts [][]string) (string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return "", fmt.Errorf("Sheet.setRange: %w", err)
	}
	cellRange, err := excelRange(rangeText)
	if err != nil {
		return "", fmt.Errorf("Sheet.setRange: %w", err)
	}
	rowCount := int(cellRange.EndRow - cellRange.StartRow + 1)
	columnCount := int(cellRange.EndColumn - cellRange.StartColumn + 1)
	if len(cellTexts) != rowCount {
		return "", fmt.Errorf("Sheet.setRange: Range requires %d rows; received %d", rowCount, len(cellTexts))
	}
	cells := make([][]excelCellData, rowCount)
	for row := range cellTexts {
		if len(cellTexts[row]) != columnCount {
			return "", fmt.Errorf("Sheet.setRange: row %d requires %d cells; received %d", row+1, columnCount, len(cellTexts[row]))
		}
		cells[row] = make([]excelCellData, columnCount)
		for column, text := range cellTexts[row] {
			cell, decodeErr := excelCell(text)
			if decodeErr != nil {
				return "", fmt.Errorf("Sheet.setRange: row %d column %d: %w", row+1, column+1, decodeErr)
			}
			cells[row][column] = cell
		}
	}
	for row := range cells {
		for column, cell := range cells[row] {
			if err := excelSetCell(&sheet, cellRange.StartRow+int64(row), cellRange.StartColumn+int64(column), cell); err != nil {
				return "", fmt.Errorf("Sheet.setRange: %w", err)
			}
		}
	}
	return excelEncode(sheet), nil
}

func ExcelSheetCells(sheetText, rangeText string) ([][]string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return nil, fmt.Errorf("Sheet.cells: %w", err)
	}
	cellRange, err := excelRange(rangeText)
	if err != nil {
		return nil, fmt.Errorf("Sheet.cells: %w", err)
	}
	rowCount := int(cellRange.EndRow - cellRange.StartRow + 1)
	columnCount := int(cellRange.EndColumn - cellRange.StartColumn + 1)
	result := make([][]string, rowCount)
	for row := 0; row < rowCount; row++ {
		result[row] = make([]string, columnCount)
		for column := 0; column < columnCount; column++ {
			result[row][column] = excelEncode(excelCellAt(sheet, cellRange.StartRow+int64(row), cellRange.StartColumn+int64(column)))
		}
	}
	return result, nil
}

// excelUsedRange computes the Sheet's supported used range: any Cell with
// non-Blank content or an applied style, plus every merged Range. Row-height
// and column-width metadata alone never extends it. This is the single
// source of truth behind both Sheet.usedRange and the worksheet's derived
// <dimension> element, so the two can never disagree.
func excelUsedRange(sheet excelSheetData) (excelRangeData, bool) {
	var result excelRangeData
	set := false
	include := func(startRow, startColumn, endRow, endColumn int64) {
		if !set {
			result = excelRangeData{startRow, startColumn, endRow, endColumn}
			set = true
			return
		}
		if startRow < result.StartRow {
			result.StartRow = startRow
		}
		if startColumn < result.StartColumn {
			result.StartColumn = startColumn
		}
		if endRow > result.EndRow {
			result.EndRow = endRow
		}
		if endColumn > result.EndColumn {
			result.EndColumn = endColumn
		}
	}
	for _, entry := range sheet.Cells {
		if entry.Cell.Kind != "Blank" || entry.Style != nil {
			include(entry.Row, entry.Column, entry.Row, entry.Column)
		}
	}
	for _, merge := range sheet.Merges {
		include(merge.StartRow, merge.StartColumn, merge.EndRow, merge.EndColumn)
	}
	return result, set
}

func ExcelSheetUsedRange(sheetText string) (*string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return nil, fmt.Errorf("Sheet.usedRange: %w", err)
	}
	result, set := excelUsedRange(sheet)
	if !set {
		return nil, nil
	}
	text := excelEncode(result)
	return &text, nil
}

func excelRangesEqual(left, right excelRangeData) bool {
	return left == right
}

func excelRangesOverlap(left, right excelRangeData) bool {
	return left.StartRow <= right.EndRow && right.StartRow <= left.EndRow &&
		left.StartColumn <= right.EndColumn && right.StartColumn <= left.EndColumn
}

func ExcelSheetMerge(sheetText, rangeText string) (string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return "", fmt.Errorf("Sheet.merge: %w", err)
	}
	cellRange, err := excelRange(rangeText)
	if err != nil {
		return "", fmt.Errorf("Sheet.merge: %w", err)
	}
	for _, existing := range sheet.Merges {
		if excelRangesEqual(existing, cellRange) {
			return excelEncode(sheet), nil
		}
		if excelRangesOverlap(existing, cellRange) {
			return "", errors.New("Sheet.merge: merged ranges must not overlap")
		}
	}
	for _, entry := range sheet.Cells {
		if entry.Row < cellRange.StartRow || entry.Row > cellRange.EndRow || entry.Column < cellRange.StartColumn || entry.Column > cellRange.EndColumn {
			continue
		}
		if entry.Row == cellRange.StartRow && entry.Column == cellRange.StartColumn {
			continue
		}
		if entry.Cell.Kind != "Blank" {
			return "", fmt.Errorf("Sheet.merge: non-anchor Cell %s is not Blank; merge would lose its value", excelCellAddress(entry.Row, entry.Column))
		}
	}
	sheet.Merges = append(sheet.Merges, cellRange)
	return excelEncode(sheet), nil
}

func ExcelSheetMerges(sheetText string) ([]string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return nil, fmt.Errorf("Sheet.merges: %w", err)
	}
	result := make([]string, len(sheet.Merges))
	for index, cellRange := range sheet.Merges {
		result[index] = excelEncode(cellRange)
	}
	return result, nil
}

func excelOverlayStyle(current *excelStyleData, patch excelStyleData) *excelStyleData {
	result := excelStyleData{}
	if current != nil {
		result = *current
	}
	if patch.Bold != nil {
		result.Bold = excelBool(*patch.Bold)
	}
	if patch.Italic != nil {
		result.Italic = excelBool(*patch.Italic)
	}
	if patch.Underline != nil {
		result.Underline = excelBool(*patch.Underline)
	}
	if patch.FontSize != nil {
		result.FontSize = excelReal(*patch.FontSize)
	}
	if patch.TextColor != nil {
		result.TextColor = excelText(*patch.TextColor)
	}
	if patch.FillColor != nil {
		result.FillColor = excelText(*patch.FillColor)
	}
	if patch.Horizontal != nil {
		result.Horizontal = excelText(*patch.Horizontal)
	}
	if patch.Vertical != nil {
		result.Vertical = excelText(*patch.Vertical)
	}
	if patch.Wrap != nil {
		result.Wrap = excelBool(*patch.Wrap)
	}
	if patch.NumberFormat != nil {
		result.NumberFormat = excelText(*patch.NumberFormat)
	}
	if patch.BorderStyle != nil {
		result.BorderStyle = excelText(*patch.BorderStyle)
	}
	if patch.BorderColor != nil {
		result.BorderColor = excelText(*patch.BorderColor)
	}
	return &result
}

func excelStyleEmpty(style excelStyleData) bool {
	return style.Bold == nil && style.Italic == nil && style.Underline == nil && style.FontSize == nil &&
		style.TextColor == nil && style.FillColor == nil && style.Horizontal == nil && style.Vertical == nil &&
		style.Wrap == nil && style.NumberFormat == nil && style.BorderStyle == nil && style.BorderColor == nil
}

func ExcelSheetStyle(sheetText, rangeText, styleText string) (string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return "", fmt.Errorf("Sheet.style: %w", err)
	}
	cellRange, err := excelRange(rangeText)
	if err != nil {
		return "", fmt.Errorf("Sheet.style: %w", err)
	}
	patch, err := excelStyle(styleText)
	if err != nil {
		return "", fmt.Errorf("Sheet.style: %w", err)
	}
	if excelStyleEmpty(patch) {
		return excelEncode(sheet), nil
	}
	for row := cellRange.StartRow; row <= cellRange.EndRow; row++ {
		for column := cellRange.StartColumn; column <= cellRange.EndColumn; column++ {
			index, found := excelCellIndex(sheet.Cells, row, column)
			if found {
				sheet.Cells[index].Style = excelOverlayStyle(sheet.Cells[index].Style, patch)
				continue
			}
			entry := excelCellEntry{Row: row, Column: column, Cell: excelCellData{Kind: "Blank"}, Style: excelOverlayStyle(nil, patch)}
			sheet.Cells = append(sheet.Cells, excelCellEntry{})
			copy(sheet.Cells[index+1:], sheet.Cells[index:])
			sheet.Cells[index] = entry
		}
	}
	return excelEncode(sheet), nil
}

func ExcelSheetColumnWidth(sheetText string, column int64, width float64) (string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return "", fmt.Errorf("Sheet.columnWidth: %w", err)
	}
	if err := excelValidateCoordinate(1, column); err != nil {
		return "", fmt.Errorf("Sheet.columnWidth: %w", err)
	}
	if width <= 0 || width > 255 || math.IsNaN(width) || math.IsInf(width, 0) {
		return "", errors.New("Sheet.columnWidth: width must be a positive finite Real no greater than 255")
	}
	index := sort.Search(len(sheet.ColumnWidths), func(i int) bool { return sheet.ColumnWidths[i].Column >= column })
	if index < len(sheet.ColumnWidths) && sheet.ColumnWidths[index].Column == column {
		sheet.ColumnWidths[index].Width = width
	} else {
		sheet.ColumnWidths = append(sheet.ColumnWidths, excelColumnDimension{})
		copy(sheet.ColumnWidths[index+1:], sheet.ColumnWidths[index:])
		sheet.ColumnWidths[index] = excelColumnDimension{Column: column, Width: width}
	}
	return excelEncode(sheet), nil
}

func ExcelSheetRowHeight(sheetText string, row int64, height float64) (string, error) {
	sheet, err := excelSheet(sheetText)
	if err != nil {
		return "", fmt.Errorf("Sheet.rowHeight: %w", err)
	}
	if err := excelValidateCoordinate(row, 1); err != nil {
		return "", fmt.Errorf("Sheet.rowHeight: %w", err)
	}
	if height <= 0 || height > 409.5 || math.IsNaN(height) || math.IsInf(height, 0) {
		return "", errors.New("Sheet.rowHeight: height must be a positive finite Real no greater than 409.5")
	}
	index := sort.Search(len(sheet.RowHeights), func(i int) bool { return sheet.RowHeights[i].Row >= row })
	if index < len(sheet.RowHeights) && sheet.RowHeights[index].Row == row {
		sheet.RowHeights[index].Height = height
	} else {
		sheet.RowHeights = append(sheet.RowHeights, excelRowDimension{})
		copy(sheet.RowHeights[index+1:], sheet.RowHeights[index:])
		sheet.RowHeights[index] = excelRowDimension{Row: row, Height: height}
	}
	return excelEncode(sheet), nil
}

func ExcelCellKind(cellText string) (string, error) {
	cell, err := excelCell(cellText)
	if err != nil {
		return "", fmt.Errorf("Cell.kind: %w", err)
	}
	return cell.Kind, nil
}

func ExcelCellIsBlank(cellText string) (bool, error) {
	cell, err := excelCell(cellText)
	if err != nil {
		return false, fmt.Errorf("Cell.isBlank: %w", err)
	}
	return cell.Kind == "Blank", nil
}

func ExcelCellString(cellText string) (string, error) {
	cell, err := excelCell(cellText)
	if err != nil {
		return "", fmt.Errorf("Cell.string: %w", err)
	}
	if cell.Kind != "String" {
		return "", fmt.Errorf("Cell.string: expected String Cell; received %s", cell.Kind)
	}
	return cell.Text, nil
}

func ExcelCellInt(cellText string) (int64, error) {
	cell, err := excelCell(cellText)
	if err != nil {
		return 0, fmt.Errorf("Cell.int: %w", err)
	}
	if cell.Kind != "Int" {
		return 0, fmt.Errorf("Cell.int: expected Int Cell; received %s", cell.Kind)
	}
	return cell.Int, nil
}

func ExcelCellReal(cellText string) (float64, error) {
	cell, err := excelCell(cellText)
	if err != nil {
		return 0, fmt.Errorf("Cell.real: %w", err)
	}
	if cell.Kind == "Int" {
		return float64(cell.Int), nil
	}
	if cell.Kind != "Real" {
		return 0, fmt.Errorf("Cell.real: expected Real or Int Cell; received %s", cell.Kind)
	}
	return cell.Real, nil
}

func ExcelCellBool(cellText string) (bool, error) {
	cell, err := excelCell(cellText)
	if err != nil {
		return false, fmt.Errorf("Cell.bool: %w", err)
	}
	if cell.Kind != "Bool" {
		return false, fmt.Errorf("Cell.bool: expected Bool Cell; received %s", cell.Kind)
	}
	return cell.Bool, nil
}

func ExcelCellFormula(cellText string) (string, error) {
	cell, err := excelCell(cellText)
	if err != nil {
		return "", fmt.Errorf("Cell.formula: %w", err)
	}
	if cell.Kind != "Formula" {
		return "", fmt.Errorf("Cell.formula: expected Formula Cell; received %s", cell.Kind)
	}
	return cell.Text, nil
}

func ExcelRangeInt(rangeText, operation string) (int64, error) {
	cellRange, err := excelRange(rangeText)
	if err != nil {
		return 0, fmt.Errorf("Range.%s: %w", operation, err)
	}
	switch operation {
	case "startRow":
		return cellRange.StartRow, nil
	case "startColumn":
		return cellRange.StartColumn, nil
	case "endRow":
		return cellRange.EndRow, nil
	case "endColumn":
		return cellRange.EndColumn, nil
	case "rowCount":
		return cellRange.EndRow - cellRange.StartRow + 1, nil
	case "columnCount":
		return cellRange.EndColumn - cellRange.StartColumn + 1, nil
	default:
		return 0, fmt.Errorf("Range.%s: unsupported Range operation", operation)
	}
}

func ExcelRangeAddress(rangeText string) (string, error) {
	cellRange, err := excelRange(rangeText)
	if err != nil {
		return "", fmt.Errorf("Range.address: %w", err)
	}
	return excelCellAddress(cellRange.StartRow, cellRange.StartColumn) + ":" + excelCellAddress(cellRange.EndRow, cellRange.EndColumn), nil
}

func ExcelStyleBool(styleText, operation string, value bool) (string, error) {
	style, err := excelStyle(styleText)
	if err != nil {
		return "", fmt.Errorf("CellStyle.%s: %w", operation, err)
	}
	switch operation {
	case "bold":
		style.Bold = excelBool(value)
	case "italic":
		style.Italic = excelBool(value)
	case "underline":
		style.Underline = excelBool(value)
	case "wrap":
		style.Wrap = excelBool(value)
	default:
		return "", fmt.Errorf("CellStyle.%s: unsupported style operation", operation)
	}
	return excelEncode(style), nil
}

func ExcelStyleFontSize(styleText string, value float64) (string, error) {
	style, err := excelStyle(styleText)
	if err != nil {
		return "", fmt.Errorf("CellStyle.fontSize: %w", err)
	}
	style.FontSize = excelReal(value)
	if err := excelValidateStyle(style); err != nil {
		return "", fmt.Errorf("CellStyle.fontSize: %w", err)
	}
	return excelEncode(style), nil
}

func ExcelStyleString(styleText, operation, value string) (string, error) {
	style, err := excelStyle(styleText)
	if err != nil {
		return "", fmt.Errorf("CellStyle.%s: %w", operation, err)
	}
	switch operation {
	case "textColor":
		style.TextColor = excelText(value)
	case "fillColor":
		style.FillColor = excelText(value)
	case "horizontal":
		style.Horizontal = excelText(value)
	case "vertical":
		style.Vertical = excelText(value)
	case "numberFormat":
		style.NumberFormat = excelText(value)
	default:
		return "", fmt.Errorf("CellStyle.%s: unsupported style operation", operation)
	}
	if err := excelValidateStyle(style); err != nil {
		return "", fmt.Errorf("CellStyle.%s: %w", operation, err)
	}
	return excelEncode(style), nil
}

func ExcelStyleBorder(styleText, borderStyle, color string) (string, error) {
	style, err := excelStyle(styleText)
	if err != nil {
		return "", fmt.Errorf("CellStyle.border: %w", err)
	}
	style.BorderStyle, style.BorderColor = excelText(borderStyle), excelText(color)
	if err := excelValidateStyle(style); err != nil {
		return "", fmt.Errorf("CellStyle.border: %w", err)
	}
	return excelEncode(style), nil
}

func excelColumnName(column int64) string {
	var reversed []byte
	for column > 0 {
		column--
		reversed = append(reversed, byte('A'+column%26))
		column /= 26
	}
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return string(reversed)
}

func excelCellAddress(row, column int64) string {
	return excelColumnName(column) + strconv.FormatInt(row, 10)
}

func excelEscapeXML(value string) string {
	var builder strings.Builder
	for _, r := range value {
		switch r {
		case '&':
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		case '"':
			builder.WriteString("&quot;")
		case '\'':
			builder.WriteString("&apos;")
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

const excelXMLHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`

func excelContentTypesXML(sheetCount int) string {
	var builder strings.Builder
	builder.WriteString(excelXMLHeader)
	builder.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	builder.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	builder.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	builder.WriteString(`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`)
	builder.WriteString(`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`)
	for index := 1; index <= sheetCount; index++ {
		builder.WriteString(`<Override PartName="/xl/worksheets/sheet` + strconv.Itoa(index) + `.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`)
	}
	builder.WriteString(`</Types>`)
	return builder.String()
}

const excelPackageRelsXML = excelXMLHeader +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
	`</Relationships>`

func excelWorkbookXML(workbook excelWorkbookData) string {
	var builder strings.Builder
	builder.WriteString(excelXMLHeader)
	builder.WriteString(`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	for index, sheet := range workbook.Sheets {
		identifier := strconv.Itoa(index + 1)
		builder.WriteString(`<sheet name="` + excelEscapeXML(sheet.Name) + `" sheetId="` + identifier + `" r:id="rId` + identifier + `"/>`)
	}
	builder.WriteString(`</sheets><calcPr calcMode="auto" fullCalcOnLoad="1" forceFullCalc="1"/></workbook>`)
	return builder.String()
}

func excelWorkbookRelsXML(sheetCount int) string {
	var builder strings.Builder
	builder.WriteString(excelXMLHeader)
	builder.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for index := 1; index <= sheetCount; index++ {
		identifier := strconv.Itoa(index)
		builder.WriteString(`<Relationship Id="rId` + identifier + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet` + identifier + `.xml"/>`)
	}
	builder.WriteString(`<Relationship Id="rId` + strconv.Itoa(sheetCount+1) + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`)
	builder.WriteString(`</Relationships>`)
	return builder.String()
}

type excelStyleCatalog struct {
	Styles []excelStyleData
	IDs    map[string]int
}

func excelCollectStyles(workbook excelWorkbookData) excelStyleCatalog {
	result := excelStyleCatalog{IDs: make(map[string]int)}
	for _, sheet := range workbook.Sheets {
		for _, entry := range sheet.Cells {
			if entry.Style == nil {
				continue
			}
			key := excelEncode(*entry.Style)
			if _, known := result.IDs[key]; known {
				continue
			}
			result.Styles = append(result.Styles, *entry.Style)
			result.IDs[key] = len(result.Styles)
		}
	}
	return result
}

func excelStyleID(catalog excelStyleCatalog, style *excelStyleData) int {
	if style == nil {
		return 0
	}
	return catalog.IDs[excelEncode(*style)]
}

func excelFontXML(style excelStyleData) string {
	var builder strings.Builder
	builder.WriteString(`<font>`)
	if style.Bold != nil {
		if *style.Bold {
			builder.WriteString(`<b/>`)
		} else {
			builder.WriteString(`<b val="0"/>`)
		}
	}
	if style.Italic != nil {
		if *style.Italic {
			builder.WriteString(`<i/>`)
		} else {
			builder.WriteString(`<i val="0"/>`)
		}
	}
	if style.Underline != nil {
		if *style.Underline {
			builder.WriteString(`<u/>`)
		} else {
			builder.WriteString(`<u val="none"/>`)
		}
	}
	if style.FontSize != nil {
		builder.WriteString(`<sz val="` + strconv.FormatFloat(*style.FontSize, 'g', -1, 64) + `"/>`)
	}
	if style.TextColor != nil {
		builder.WriteString(`<color rgb="FF` + (*style.TextColor)[1:] + `"/>`)
	}
	builder.WriteString(`</font>`)
	return builder.String()
}

func excelFillXML(style excelStyleData) string {
	if style.FillColor == nil {
		return `<fill><patternFill patternType="none"/></fill>`
	}
	return `<fill><patternFill patternType="solid"><fgColor rgb="FF` + (*style.FillColor)[1:] + `"/><bgColor indexed="64"/></patternFill></fill>`
}

func excelBorderXML(style excelStyleData) string {
	var builder strings.Builder
	builder.WriteString(`<border>`)
	for _, edge := range []string{"left", "right", "top", "bottom"} {
		builder.WriteString(`<` + edge)
		if style.BorderStyle != nil && *style.BorderStyle != "none" {
			builder.WriteString(` style="` + *style.BorderStyle + `"`)
		}
		builder.WriteString(`>`)
		if style.BorderStyle != nil && style.BorderColor != nil {
			builder.WriteString(`<color rgb="FF` + (*style.BorderColor)[1:] + `"/>`)
		}
		builder.WriteString(`</` + edge + `>`)
	}
	builder.WriteString(`<diagonal/></border>`)
	return builder.String()
}

func excelStylesXML(catalog excelStyleCatalog) string {
	var builder strings.Builder
	builder.WriteString(excelXMLHeader)
	builder.WriteString(`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	formats := make([]string, 0)
	formatIDs := make(map[string]int)
	for _, style := range catalog.Styles {
		if style.NumberFormat == nil || *style.NumberFormat == "General" {
			continue
		}
		if _, known := formatIDs[*style.NumberFormat]; known {
			continue
		}
		formatIDs[*style.NumberFormat] = 164 + len(formats)
		formats = append(formats, *style.NumberFormat)
	}
	if len(formats) > 0 {
		builder.WriteString(`<numFmts count="` + strconv.Itoa(len(formats)) + `">`)
		for _, format := range formats {
			builder.WriteString(`<numFmt numFmtId="` + strconv.Itoa(formatIDs[format]) + `" formatCode="` + excelEscapeXML(format) + `"/>`)
		}
		builder.WriteString(`</numFmts>`)
	}
	builder.WriteString(`<fonts count="` + strconv.Itoa(1+len(catalog.Styles)) + `"><font><sz val="11"/><name val="Calibri"/><family val="2"/><scheme val="minor"/></font>`)
	for _, style := range catalog.Styles {
		builder.WriteString(excelFontXML(style))
	}
	builder.WriteString(`</fonts>`)
	builder.WriteString(`<fills count="` + strconv.Itoa(2+len(catalog.Styles)) + `"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill>`)
	for _, style := range catalog.Styles {
		builder.WriteString(excelFillXML(style))
	}
	builder.WriteString(`</fills>`)
	builder.WriteString(`<borders count="` + strconv.Itoa(1+len(catalog.Styles)) + `"><border><left/><right/><top/><bottom/><diagonal/></border>`)
	for _, style := range catalog.Styles {
		builder.WriteString(excelBorderXML(style))
	}
	builder.WriteString(`</borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>`)
	builder.WriteString(`<cellXfs count="` + strconv.Itoa(1+len(catalog.Styles)) + `"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>`)
	for index, style := range catalog.Styles {
		numberFormatID := 0
		if style.NumberFormat != nil && *style.NumberFormat != "General" {
			numberFormatID = formatIDs[*style.NumberFormat]
		}
		builder.WriteString(`<xf numFmtId="` + strconv.Itoa(numberFormatID) + `" fontId="` + strconv.Itoa(index+1) + `" fillId="` + strconv.Itoa(index+2) + `" borderId="` + strconv.Itoa(index+1) + `" xfId="0" applyFont="1" applyFill="1" applyBorder="1"`)
		if style.NumberFormat != nil {
			builder.WriteString(` applyNumberFormat="1"`)
		}
		if style.Horizontal != nil || style.Vertical != nil || style.Wrap != nil {
			builder.WriteString(` applyAlignment="1"><alignment`)
			if style.Horizontal != nil {
				builder.WriteString(` horizontal="` + *style.Horizontal + `"`)
			}
			if style.Vertical != nil {
				builder.WriteString(` vertical="` + *style.Vertical + `"`)
			}
			if style.Wrap != nil {
				if *style.Wrap {
					builder.WriteString(` wrapText="1"`)
				} else {
					builder.WriteString(` wrapText="0"`)
				}
			}
			builder.WriteString(`/></xf>`)
		} else {
			builder.WriteString(`/>`)
		}
	}
	builder.WriteString(`</cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`)
	return builder.String()
}

func excelCellXML(entry excelCellEntry, styleID int) string {
	address := excelCellAddress(entry.Row, entry.Column)
	style := ""
	if styleID != 0 {
		style = ` s="` + strconv.Itoa(styleID) + `"`
	}
	switch entry.Cell.Kind {
	case "Blank":
		return `<c r="` + address + `"` + style + `/>`
	case "String":
		return `<c r="` + address + `"` + style + ` t="inlineStr"><is><t xml:space="preserve">` + excelEscapeXML(entry.Cell.Text) + `</t></is></c>`
	case "Int":
		return `<c r="` + address + `"` + style + `><v>` + strconv.FormatInt(entry.Cell.Int, 10) + `</v></c>`
	case "Real":
		value := strconv.FormatFloat(entry.Cell.Real, 'g', -1, 64)
		if !strings.ContainsAny(value, ".eE") {
			value += ".0"
		}
		return `<c r="` + address + `"` + style + `><v>` + value + `</v></c>`
	case "Bool":
		value := "0"
		if entry.Cell.Bool {
			value = "1"
		}
		return `<c r="` + address + `"` + style + ` t="b"><v>` + value + `</v></c>`
	case "Formula":
		return `<c r="` + address + `"` + style + `><f>` + excelEscapeXML(strings.TrimPrefix(entry.Cell.Text, "=")) + `</f></c>`
	default:
		panic("ahdcode: unsupported validated Excel Cell kind")
	}
}

func excelWorksheetXML(sheet excelSheetData, catalog excelStyleCatalog) string {
	var builder strings.Builder
	builder.WriteString(excelXMLHeader)
	builder.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	if used, set := excelUsedRange(sheet); set {
		start := excelCellAddress(used.StartRow, used.StartColumn)
		if used.StartRow == used.EndRow && used.StartColumn == used.EndColumn {
			builder.WriteString(`<dimension ref="` + start + `"/>`)
		} else {
			builder.WriteString(`<dimension ref="` + start + `:` + excelCellAddress(used.EndRow, used.EndColumn) + `"/>`)
		}
	} else {
		builder.WriteString(`<dimension ref="A1"/>`)
	}
	if len(sheet.ColumnWidths) > 0 {
		builder.WriteString(`<cols>`)
		for _, dimension := range sheet.ColumnWidths {
			column := strconv.FormatInt(dimension.Column, 10)
			builder.WriteString(`<col min="` + column + `" max="` + column + `" width="` + strconv.FormatFloat(dimension.Width, 'g', -1, 64) + `" customWidth="1"/>`)
		}
		builder.WriteString(`</cols>`)
	}
	heights := make(map[int64]float64, len(sheet.RowHeights))
	for _, dimension := range sheet.RowHeights {
		heights[dimension.Row] = dimension.Height
	}
	rows := make(map[int64][]excelCellEntry)
	rowNumbers := make([]int64, 0)
	for _, entry := range sheet.Cells {
		if _, known := rows[entry.Row]; !known {
			rowNumbers = append(rowNumbers, entry.Row)
		}
		rows[entry.Row] = append(rows[entry.Row], entry)
	}
	for _, dimension := range sheet.RowHeights {
		if _, known := rows[dimension.Row]; !known {
			rows[dimension.Row] = nil
			rowNumbers = append(rowNumbers, dimension.Row)
		}
	}
	sort.Slice(rowNumbers, func(left, right int) bool { return rowNumbers[left] < rowNumbers[right] })
	builder.WriteString(`<sheetData>`)
	for _, row := range rowNumbers {
		builder.WriteString(`<row r="` + strconv.FormatInt(row, 10) + `"`)
		if height, known := heights[row]; known {
			builder.WriteString(` ht="` + strconv.FormatFloat(height, 'g', -1, 64) + `" customHeight="1"`)
		}
		builder.WriteString(`>`)
		for _, entry := range rows[row] {
			builder.WriteString(excelCellXML(entry, excelStyleID(catalog, entry.Style)))
		}
		builder.WriteString(`</row>`)
	}
	builder.WriteString(`</sheetData>`)
	if len(sheet.Merges) > 0 {
		builder.WriteString(`<mergeCells count="` + strconv.Itoa(len(sheet.Merges)) + `">`)
		for _, cellRange := range sheet.Merges {
			builder.WriteString(`<mergeCell ref="` + excelCellAddress(cellRange.StartRow, cellRange.StartColumn) + `:` + excelCellAddress(cellRange.EndRow, cellRange.EndColumn) + `"/>`)
		}
		builder.WriteString(`</mergeCells>`)
	}
	builder.WriteString(`</worksheet>`)
	return builder.String()
}

func excelWriteZIPEntry(writer *zip.Writer, name, content string) error {
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	header.SetMode(0644)
	part, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.WriteString(part, content)
	return err
}

func excelBuildPackage(workbook excelWorkbookData) ([]byte, error) {
	catalog := excelCollectStyles(workbook)
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entries := []struct{ name, content string }{
		{"[Content_Types].xml", excelContentTypesXML(len(workbook.Sheets))},
		{"_rels/.rels", excelPackageRelsXML},
		{"xl/workbook.xml", excelWorkbookXML(workbook)},
		{"xl/_rels/workbook.xml.rels", excelWorkbookRelsXML(len(workbook.Sheets))},
		{"xl/styles.xml", excelStylesXML(catalog)},
	}
	for index, sheet := range workbook.Sheets {
		entries = append(entries, struct{ name, content string }{"xl/worksheets/sheet" + strconv.Itoa(index+1) + ".xml", excelWorksheetXML(sheet, catalog)})
	}
	for _, entry := range entries {
		if err := excelWriteZIPEntry(writer, entry.name, entry.content); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type excelRelationship struct {
	ID         string `xml:"Id,attr"`
	Type       string `xml:"Type,attr"`
	Target     string `xml:"Target,attr"`
	TargetMode string `xml:"TargetMode,attr"`
}

type excelRelationships struct {
	Items []excelRelationship `xml:"Relationship"`
}

type excelWorkbookXMLData struct {
	Sheets []struct {
		Name string `xml:"name,attr"`
		ID   string `xml:"id,attr"`
	} `xml:"sheets>sheet"`
}

type excelSharedStringsXML struct {
	Items []excelStringItemXML `xml:"si"`
}

type excelStringItemXML struct {
	Text string `xml:"t"`
	Runs []struct {
		Text string `xml:"t"`
	} `xml:"r"`
}

func (item excelStringItemXML) value() string {
	if len(item.Runs) == 0 {
		return item.Text
	}
	var builder strings.Builder
	if item.Text != "" {
		builder.WriteString(item.Text)
	}
	for _, run := range item.Runs {
		builder.WriteString(run.Text)
	}
	return builder.String()
}

type excelStylesXMLData struct {
	NumberFormats []struct {
		ID   int    `xml:"numFmtId,attr"`
		Code string `xml:"formatCode,attr"`
	} `xml:"numFmts>numFmt"`
	Fonts []struct {
		Bold *struct {
			Value string `xml:"val,attr"`
		} `xml:"b"`
		Italic *struct {
			Value string `xml:"val,attr"`
		} `xml:"i"`
		Underline *struct {
			Value string `xml:"val,attr"`
		} `xml:"u"`
		Size struct {
			Value string `xml:"val,attr"`
		} `xml:"sz"`
		Color struct {
			RGB string `xml:"rgb,attr"`
		} `xml:"color"`
	} `xml:"fonts>font"`
	Fills []struct {
		Pattern struct {
			Type       string `xml:"patternType,attr"`
			Foreground struct {
				RGB string `xml:"rgb,attr"`
			} `xml:"fgColor"`
		} `xml:"patternFill"`
	} `xml:"fills>fill"`
	Borders []struct {
		Left   excelBorderSideXML `xml:"left"`
		Right  excelBorderSideXML `xml:"right"`
		Top    excelBorderSideXML `xml:"top"`
		Bottom excelBorderSideXML `xml:"bottom"`
	} `xml:"borders>border"`
	CellFormats []excelCellFormatXML `xml:"cellXfs>xf"`
}

type excelBorderSideXML struct {
	Style string `xml:"style,attr"`
	Color struct {
		RGB string `xml:"rgb,attr"`
	} `xml:"color"`
}

type excelCellFormatXML struct {
	NumberFormatID    int    `xml:"numFmtId,attr"`
	ApplyNumberFormat string `xml:"applyNumberFormat,attr"`
	FontID            int    `xml:"fontId,attr"`
	FillID            int    `xml:"fillId,attr"`
	BorderID          int    `xml:"borderId,attr"`
	Alignment         *struct {
		Horizontal string `xml:"horizontal,attr"`
		Vertical   string `xml:"vertical,attr"`
		Wrap       string `xml:"wrapText,attr"`
	} `xml:"alignment"`
}

type excelWorksheetXMLData struct {
	Columns []struct {
		Min   int64  `xml:"min,attr"`
		Max   int64  `xml:"max,attr"`
		Width string `xml:"width,attr"`
	} `xml:"cols>col"`
	Rows []struct {
		Index  int64                   `xml:"r,attr"`
		Height string                  `xml:"ht,attr"`
		Cells  []excelWorksheetCellXML `xml:"c"`
	} `xml:"sheetData>row"`
	Merges []struct {
		Reference string `xml:"ref,attr"`
	} `xml:"mergeCells>mergeCell"`
}

type excelWorksheetCellXML struct {
	Reference string              `xml:"r,attr"`
	Type      string              `xml:"t,attr"`
	Style     int                 `xml:"s,attr"`
	Value     *string             `xml:"v"`
	Inline    *excelStringItemXML `xml:"is"`
	Formula   *struct {
		Type string `xml:"t,attr"`
		Text string `xml:",chardata"`
	} `xml:"f"`
}

func excelParseXML(content []byte, target any, operation, part string) error {
	if bytes.Contains(bytes.ToUpper(content), []byte("<!DOCTYPE")) {
		return fmt.Errorf("%s: %s contains a forbidden DTD", operation, part)
	}
	decoder := xml.NewDecoder(bytes.NewReader(content))
	decoder.Strict = true
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%s: %s is malformed XML", operation, part)
	}
	return nil
}

func excelCleanPart(basePart, target string) (string, error) {
	if target == "" || strings.Contains(target, "\\") || strings.HasPrefix(target, "/") {
		return "", errors.New("relationship target is not a safe internal package path")
	}
	clean := path.Clean(path.Join(path.Dir(basePart), target))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", errors.New("relationship target escapes the XLSX package")
	}
	return clean, nil
}

func excelRelationshipPart(part string) string {
	return path.Join(path.Dir(part), "_rels", path.Base(part)+".rels")
}

func excelReadZIP(data []byte) (map[string][]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("XLSX package is empty")
	}
	if len(data) > excelMaxArchiveBytes {
		return nil, fmt.Errorf("XLSX archive exceeds the %d-byte limit", excelMaxArchiveBytes)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.New("file is not a valid XLSX ZIP package")
	}
	if len(reader.File) > excelMaxEntries {
		return nil, fmt.Errorf("XLSX package exceeds the %d-entry limit", excelMaxEntries)
	}
	entries := make(map[string][]byte, len(reader.File))
	var total uint64
	for _, file := range reader.File {
		name := file.Name
		if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") {
			return nil, fmt.Errorf("XLSX package contains unsafe member path %q", name)
		}
		if _, duplicate := entries[name]; duplicate {
			return nil, fmt.Errorf("XLSX package contains duplicate member %q", name)
		}
		if file.UncompressedSize64 > excelMaxEntryBytes {
			return nil, fmt.Errorf("XLSX member %q exceeds the individual size limit", name)
		}
		total += file.UncompressedSize64
		if total > excelMaxTotalBytes {
			return nil, errors.New("XLSX package exceeds the total uncompressed size limit")
		}
		if file.UncompressedSize64 > 0 {
			if file.CompressedSize64 == 0 || file.UncompressedSize64/file.CompressedSize64 > excelMaxCompressRatio {
				return nil, fmt.Errorf("XLSX member %q has an unreasonable compression ratio", name)
			}
		}
		opened, openErr := file.Open()
		if openErr != nil {
			return nil, fmt.Errorf("XLSX member %q cannot be read", name)
		}
		content, readErr := io.ReadAll(io.LimitReader(opened, excelMaxEntryBytes+1))
		closeErr := opened.Close()
		if readErr != nil || closeErr != nil || len(content) > excelMaxEntryBytes {
			return nil, fmt.Errorf("XLSX member %q cannot be read safely", name)
		}
		entries[name] = content
	}
	return entries, nil
}

func excelRelationshipMap(content []byte, operation, part string) (map[string]excelRelationship, error) {
	var relationships excelRelationships
	if err := excelParseXML(content, &relationships, operation, part); err != nil {
		return nil, err
	}
	result := make(map[string]excelRelationship, len(relationships.Items))
	for _, relationship := range relationships.Items {
		if relationship.ID == "" || relationship.Type == "" || relationship.Target == "" {
			return nil, fmt.Errorf("%s: %s contains an invalid relationship", operation, part)
		}
		if _, duplicate := result[relationship.ID]; duplicate {
			return nil, fmt.Errorf("%s: %s contains duplicate relationship ID %q", operation, part, relationship.ID)
		}
		result[relationship.ID] = relationship
	}
	return result, nil
}

func excelFindRelationshipBySuffix(relationships map[string]excelRelationship, suffix string) (excelRelationship, bool) {
	identifiers := make([]string, 0, len(relationships))
	for identifier := range relationships {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)
	for _, identifier := range identifiers {
		relationship := relationships[identifier]
		if strings.HasSuffix(relationship.Type, suffix) {
			return relationship, true
		}
	}
	return excelRelationship{}, false
}

func excelARGBColor(rgb string) *string {
	if len(rgb) == 8 {
		rgb = rgb[2:]
	}
	if len(rgb) != 6 {
		return nil
	}
	rgb = strings.ToUpper(rgb)
	value := "#" + rgb
	if !excelValidColor(value) {
		return nil
	}
	return excelText(value)
}

var excelBuiltinNumberFormats = map[int]string{
	0: "General", 1: "0", 2: "0.00", 9: "0%", 10: "0.00%", 14: "mm-dd-yy",
	49: "@",
}

func excelDecodeStyles(content []byte) ([]*excelStyleData, error) {
	var source excelStylesXMLData
	if err := excelParseXML(content, &source, "Excel.read", "styles part"); err != nil {
		return nil, err
	}
	formats := make(map[int]string)
	for id, value := range excelBuiltinNumberFormats {
		formats[id] = value
	}
	for _, format := range source.NumberFormats {
		if format.ID < 0 || format.Code == "" || !excelValidXMLText(format.Code) {
			return nil, errors.New("Excel.read: styles part contains an invalid number format")
		}
		formats[format.ID] = format.Code
	}
	result := make([]*excelStyleData, len(source.CellFormats))
	for index, format := range source.CellFormats {
		if index == 0 {
			continue
		}
		style := excelStyleData{}
		if format.FontID < 0 || format.FontID >= len(source.Fonts) {
			return nil, errors.New("Excel.read: Cell style references an unknown font")
		}
		if format.FontID != 0 {
			font := source.Fonts[format.FontID]
			if font.Bold != nil {
				style.Bold = excelBool(font.Bold.Value != "0" && !strings.EqualFold(font.Bold.Value, "false"))
			}
			if font.Italic != nil {
				style.Italic = excelBool(font.Italic.Value != "0" && !strings.EqualFold(font.Italic.Value, "false"))
			}
			if font.Underline != nil {
				style.Underline = excelBool(font.Underline.Value != "0" && font.Underline.Value != "none" && !strings.EqualFold(font.Underline.Value, "false"))
			}
			if font.Size.Value != "" {
				size, err := strconv.ParseFloat(font.Size.Value, 64)
				if err != nil || size <= 0 || math.IsNaN(size) || math.IsInf(size, 0) {
					return nil, errors.New("Excel.read: Cell style contains an invalid font size")
				}
				style.FontSize = excelReal(size)
			}
			style.TextColor = excelARGBColor(font.Color.RGB)
		}
		if format.FillID < 0 || format.FillID >= len(source.Fills) {
			return nil, errors.New("Excel.read: Cell style references an unknown fill")
		}
		if format.FillID != 0 && source.Fills[format.FillID].Pattern.Type == "solid" {
			style.FillColor = excelARGBColor(source.Fills[format.FillID].Pattern.Foreground.RGB)
		}
		if format.BorderID < 0 || format.BorderID >= len(source.Borders) {
			return nil, errors.New("Excel.read: Cell style references an unknown border")
		}
		if format.BorderID != 0 {
			border := source.Borders[format.BorderID]
			side := border.Left
			if side.Style == "" && side.Color.RGB == "" {
				side = border.Right
			}
			if side.Style == "" && side.Color.RGB == "" {
				side = border.Top
			}
			if side.Style == "" && side.Color.RGB == "" {
				side = border.Bottom
			}
			if side.Style != "" {
				allowed := map[string]bool{"thin": true, "medium": true, "thick": true, "dashed": true, "dotted": true, "double": true}
				if allowed[side.Style] {
					style.BorderStyle = excelText(side.Style)
					style.BorderColor = excelARGBColor(side.Color.RGB)
					if style.BorderColor == nil {
						style.BorderColor = excelText("#000000")
					}
				}
			} else if side.Color.RGB != "" {
				style.BorderStyle = excelText("none")
				style.BorderColor = excelARGBColor(side.Color.RGB)
				if style.BorderColor == nil {
					return nil, errors.New("Excel.read: Cell style contains an invalid border color")
				}
			}
		}
		if format.NumberFormatID != 0 || format.ApplyNumberFormat == "1" || strings.EqualFold(format.ApplyNumberFormat, "true") {
			if value, known := formats[format.NumberFormatID]; known {
				style.NumberFormat = excelText(value)
			}
		}
		if format.Alignment != nil {
			if value := format.Alignment.Horizontal; value == "left" || value == "center" || value == "right" {
				style.Horizontal = excelText(value)
			}
			if value := format.Alignment.Vertical; value == "top" || value == "center" || value == "bottom" {
				style.Vertical = excelText(value)
			}
			if format.Alignment.Wrap != "" {
				style.Wrap = excelBool(format.Alignment.Wrap == "1" || strings.EqualFold(format.Alignment.Wrap, "true"))
			}
		}
		if err := excelValidateStyle(style); err != nil {
			return nil, fmt.Errorf("Excel.read: supported Cell style is invalid: %w", err)
		}
		result[index] = &style
	}
	return result, nil
}

func excelParseCellReference(reference string) (int64, int64, error) {
	reference = strings.ReplaceAll(reference, "$", "")
	if reference == "" {
		return 0, 0, errors.New("Cell reference is empty")
	}
	position := 0
	column := int64(0)
	for position < len(reference) {
		character := reference[position]
		if character >= 'a' && character <= 'z' {
			character -= 'a' - 'A'
		}
		if character < 'A' || character > 'Z' {
			break
		}
		column = column*26 + int64(character-'A'+1)
		position++
	}
	if position == 0 || position == len(reference) {
		return 0, 0, errors.New("Cell reference is invalid")
	}
	row, err := strconv.ParseInt(reference[position:], 10, 64)
	if err != nil || excelValidateCoordinate(row, column) != nil {
		return 0, 0, errors.New("Cell reference is outside Excel limits")
	}
	return row, column, nil
}

func excelParseRangeReference(reference string) (excelRangeData, error) {
	parts := strings.Split(reference, ":")
	if len(parts) == 1 {
		parts = append(parts, parts[0])
	}
	if len(parts) != 2 {
		return excelRangeData{}, errors.New("Range reference is invalid")
	}
	startRow, startColumn, err := excelParseCellReference(parts[0])
	if err != nil {
		return excelRangeData{}, err
	}
	endRow, endColumn, err := excelParseCellReference(parts[1])
	if err != nil {
		return excelRangeData{}, err
	}
	result := excelRangeData{startRow, startColumn, endRow, endColumn}
	if err := excelValidateRange(result); err != nil {
		return excelRangeData{}, err
	}
	return result, nil
}

func excelDecodeNumeric(value string) (excelCellData, error) {
	if value == "" {
		return excelCellData{Kind: "Blank"}, nil
	}
	if !strings.ContainsAny(value, ".eE") {
		integer, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return excelCellData{Kind: "Int", Int: integer}, nil
		}
	}
	realValue, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(realValue) || math.IsInf(realValue, 0) {
		return excelCellData{}, errors.New("numeric Cell value is invalid")
	}
	return excelCellData{Kind: "Real", Real: realValue}, nil
}

func excelDecodeWorksheetCell(source excelWorksheetCellXML, sharedStrings []string) (excelCellData, error) {
	if source.Formula != nil {
		if source.Formula.Type != "" && source.Formula.Type != "normal" {
			return excelCellData{}, fmt.Errorf("advanced formula representation %q is not supported", source.Formula.Type)
		}
		cell := excelCellData{Kind: "Formula", Text: "=" + source.Formula.Text}
		if err := excelValidateCell(cell); err != nil {
			return excelCellData{}, err
		}
		return cell, nil
	}
	value := ""
	if source.Value != nil {
		value = *source.Value
	}
	switch source.Type {
	case "", "n":
		return excelDecodeNumeric(value)
	case "s":
		index, err := strconv.Atoi(value)
		if err != nil || index < 0 || index >= len(sharedStrings) {
			return excelCellData{}, errors.New("shared String Cell references an unknown entry")
		}
		cell := excelCellData{Kind: "String", Text: sharedStrings[index]}
		if err := excelValidateCell(cell); err != nil {
			return excelCellData{}, err
		}
		return cell, nil
	case "inlineStr":
		if source.Inline == nil {
			return excelCellData{Kind: "Blank"}, nil
		}
		cell := excelCellData{Kind: "String", Text: source.Inline.value()}
		if err := excelValidateCell(cell); err != nil {
			return excelCellData{}, err
		}
		return cell, nil
	case "str":
		cell := excelCellData{Kind: "String", Text: value}
		if err := excelValidateCell(cell); err != nil {
			return excelCellData{}, err
		}
		return cell, nil
	case "b":
		if value == "1" || strings.EqualFold(value, "true") {
			return excelCellData{Kind: "Bool", Bool: true}, nil
		}
		if value == "0" || strings.EqualFold(value, "false") {
			return excelCellData{Kind: "Bool", Bool: false}, nil
		}
		return excelCellData{}, errors.New("Bool Cell value must be 0 or 1")
	case "e":
		return excelCellData{}, errors.New("error Cells are not supported because their value cannot be represented by the closed Cell model")
	case "d":
		return excelCellData{}, errors.New("ISO date Cells are not supported; dates are not inferred in Excel v0.1.19")
	default:
		return excelCellData{}, fmt.Errorf("Cell type %q is not supported", source.Type)
	}
}

func excelReadWorksheet(content []byte, name string, sharedStrings []string, styles []*excelStyleData) (excelSheetData, error) {
	var source excelWorksheetXMLData
	if err := excelParseXML(content, &source, "Excel.read", "worksheet for Sheet "+strconv.Quote(name)); err != nil {
		return excelSheetData{}, err
	}
	sheet := excelSheetData{Name: name}
	seenCells := make(map[[2]int64]bool)
	seenRows := make(map[int64]bool)
	for _, row := range source.Rows {
		if row.Index < 1 || row.Index > excelMaxRow || seenRows[row.Index] {
			return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q contains an invalid or duplicate row", name)
		}
		seenRows[row.Index] = true
		if row.Height != "" {
			height, err := strconv.ParseFloat(row.Height, 64)
			if err != nil || row.Index < 1 || row.Index > excelMaxRow || height <= 0 || height > 409.5 || math.IsNaN(height) || math.IsInf(height, 0) {
				return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q contains an invalid row height", name)
			}
			sheet.RowHeights = append(sheet.RowHeights, excelRowDimension{Row: row.Index, Height: height})
		}
		for _, sourceCell := range row.Cells {
			if sourceCell.Reference == "" {
				return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q contains a Cell without a coordinate", name)
			}
			cellRow, cellColumn, err := excelParseCellReference(sourceCell.Reference)
			if err != nil {
				return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q contains invalid Cell reference %q", name, sourceCell.Reference)
			}
			key := [2]int64{cellRow, cellColumn}
			if seenCells[key] {
				return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q contains duplicate Cell %s", name, excelCellAddress(cellRow, cellColumn))
			}
			seenCells[key] = true
			cell, err := excelDecodeWorksheetCell(sourceCell, sharedStrings)
			if err != nil {
				return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q Cell %s: %w", name, excelCellAddress(cellRow, cellColumn), err)
			}
			var style *excelStyleData
			if sourceCell.Style != 0 {
				if sourceCell.Style < 0 || sourceCell.Style >= len(styles) {
					return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q Cell %s references unknown style %d", name, excelCellAddress(cellRow, cellColumn), sourceCell.Style)
				}
				style = styles[sourceCell.Style]
				if style != nil {
					copyStyle := *style
					style = &copyStyle
				}
			}
			if cell.Kind != "Blank" || style != nil {
				sheet.Cells = append(sheet.Cells, excelCellEntry{Row: cellRow, Column: cellColumn, Cell: cell, Style: style})
			}
		}
	}
	columnWidths := make(map[int64]float64)
	for _, column := range source.Columns {
		if column.Min < 1 || column.Max < column.Min || column.Max > excelMaxColumn {
			return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q contains an invalid column range", name)
		}
		width, err := strconv.ParseFloat(column.Width, 64)
		if err != nil || width <= 0 || width > 255 || math.IsNaN(width) || math.IsInf(width, 0) {
			return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q contains an invalid column width", name)
		}
		for index := column.Min; index <= column.Max; index++ {
			columnWidths[index] = width
		}
	}
	for column, width := range columnWidths {
		sheet.ColumnWidths = append(sheet.ColumnWidths, excelColumnDimension{Column: column, Width: width})
	}
	for _, merge := range source.Merges {
		cellRange, err := excelParseRangeReference(merge.Reference)
		if err != nil {
			return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q contains invalid merged range %q", name, merge.Reference)
		}
		for _, existing := range sheet.Merges {
			if excelRangesOverlap(existing, cellRange) {
				return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q contains overlapping merged ranges", name)
			}
		}
		for _, entry := range sheet.Cells {
			if entry.Row >= cellRange.StartRow && entry.Row <= cellRange.EndRow && entry.Column >= cellRange.StartColumn && entry.Column <= cellRange.EndColumn &&
				(entry.Row != cellRange.StartRow || entry.Column != cellRange.StartColumn) && entry.Cell.Kind != "Blank" {
				return excelSheetData{}, fmt.Errorf("Excel.read: Sheet %q merged range %s would hide non-Blank Cell %s", name, merge.Reference, excelCellAddress(entry.Row, entry.Column))
			}
		}
		sheet.Merges = append(sheet.Merges, cellRange)
	}
	sort.Slice(sheet.Cells, func(left, right int) bool {
		return sheet.Cells[left].Row < sheet.Cells[right].Row || (sheet.Cells[left].Row == sheet.Cells[right].Row && sheet.Cells[left].Column < sheet.Cells[right].Column)
	})
	sort.Slice(sheet.ColumnWidths, func(left, right int) bool { return sheet.ColumnWidths[left].Column < sheet.ColumnWidths[right].Column })
	sort.Slice(sheet.RowHeights, func(left, right int) bool { return sheet.RowHeights[left].Row < sheet.RowHeights[right].Row })
	return sheet, nil
}

func excelReadPackage(data []byte) (excelWorkbookData, error) {
	entries, err := excelReadZIP(data)
	if err != nil {
		return excelWorkbookData{}, fmt.Errorf("Excel.read: %w", err)
	}
	if entries["[Content_Types].xml"] == nil || entries["_rels/.rels"] == nil {
		return excelWorkbookData{}, errors.New("Excel.read: XLSX package is missing required root parts")
	}
	var contentTypes struct{}
	if err := excelParseXML(entries["[Content_Types].xml"], &contentTypes, "Excel.read", "[Content_Types].xml"); err != nil {
		return excelWorkbookData{}, err
	}
	rootRelationships, err := excelRelationshipMap(entries["_rels/.rels"], "Excel.read", "_rels/.rels")
	if err != nil {
		return excelWorkbookData{}, err
	}
	office, found := excelFindRelationshipBySuffix(rootRelationships, "/officeDocument")
	if !found {
		return excelWorkbookData{}, errors.New("Excel.read: XLSX package has no workbook relationship")
	}
	if strings.EqualFold(office.TargetMode, "External") {
		return excelWorkbookData{}, errors.New("Excel.read: workbook relationship must not be external")
	}
	workbookPart, err := excelCleanPart("_rels/.rels", office.Target)
	if err != nil {
		// Root relationships resolve from the package root, not the _rels directory.
		workbookPart, err = excelCleanPart("root.xml", office.Target)
	} else if strings.HasPrefix(workbookPart, "_rels/") {
		workbookPart, err = excelCleanPart("root.xml", office.Target)
	}
	if err != nil {
		return excelWorkbookData{}, fmt.Errorf("Excel.read: invalid workbook relationship: %w", err)
	}
	workbookContent := entries[workbookPart]
	if workbookContent == nil {
		return excelWorkbookData{}, errors.New("Excel.read: workbook part is missing")
	}
	var workbookSource excelWorkbookXMLData
	if err := excelParseXML(workbookContent, &workbookSource, "Excel.read", workbookPart); err != nil {
		return excelWorkbookData{}, err
	}
	workbookRelsPart := excelRelationshipPart(workbookPart)
	workbookRelsContent := entries[workbookRelsPart]
	if workbookRelsContent == nil {
		return excelWorkbookData{}, errors.New("Excel.read: workbook relationships part is missing")
	}
	workbookRelationships, err := excelRelationshipMap(workbookRelsContent, "Excel.read", workbookRelsPart)
	if err != nil {
		return excelWorkbookData{}, err
	}

	sharedStrings := []string{}
	if relationship, known := excelFindRelationshipBySuffix(workbookRelationships, "/sharedStrings"); known {
		if strings.EqualFold(relationship.TargetMode, "External") {
			return excelWorkbookData{}, errors.New("Excel.read: sharedStrings relationship must not be external")
		}
		part, cleanErr := excelCleanPart(workbookPart, relationship.Target)
		if cleanErr != nil || entries[part] == nil {
			return excelWorkbookData{}, errors.New("Excel.read: sharedStrings relationship is broken")
		}
		var source excelSharedStringsXML
		if err := excelParseXML(entries[part], &source, "Excel.read", part); err != nil {
			return excelWorkbookData{}, err
		}
		sharedStrings = make([]string, len(source.Items))
		for index, item := range source.Items {
			sharedStrings[index] = item.value()
			cell := excelCellData{Kind: "String", Text: sharedStrings[index]}
			if err := excelValidateCell(cell); err != nil {
				return excelWorkbookData{}, fmt.Errorf("Excel.read: shared String %d: %w", index, err)
			}
		}
	}
	styles := []*excelStyleData{nil}
	if relationship, known := excelFindRelationshipBySuffix(workbookRelationships, "/styles"); known {
		if strings.EqualFold(relationship.TargetMode, "External") {
			return excelWorkbookData{}, errors.New("Excel.read: styles relationship must not be external")
		}
		part, cleanErr := excelCleanPart(workbookPart, relationship.Target)
		if cleanErr != nil || entries[part] == nil {
			return excelWorkbookData{}, errors.New("Excel.read: styles relationship is broken")
		}
		styles, err = excelDecodeStyles(entries[part])
		if err != nil {
			return excelWorkbookData{}, err
		}
		if len(styles) == 0 {
			styles = []*excelStyleData{nil}
		}
	}

	workbook := excelWorkbookData{Sheets: make([]excelSheetData, 0, len(workbookSource.Sheets))}
	for sheetIndex, sourceSheet := range workbookSource.Sheets {
		if err := excelValidateSheetName(sourceSheet.Name); err != nil {
			return excelWorkbookData{}, fmt.Errorf("Excel.read: invalid Sheet name %q: %w", sourceSheet.Name, err)
		}
		for previous := 0; previous < sheetIndex; previous++ {
			if strings.EqualFold(workbookSource.Sheets[previous].Name, sourceSheet.Name) {
				return excelWorkbookData{}, fmt.Errorf("Excel.read: duplicate Sheet name %q (names are case-insensitive)", sourceSheet.Name)
			}
		}
		relationship, known := workbookRelationships[sourceSheet.ID]
		if !known || !strings.HasSuffix(relationship.Type, "/worksheet") {
			return excelWorkbookData{}, fmt.Errorf("Excel.read: Sheet %q has a broken worksheet relationship", sourceSheet.Name)
		}
		if strings.EqualFold(relationship.TargetMode, "External") {
			return excelWorkbookData{}, fmt.Errorf("Excel.read: Sheet %q worksheet relationship must not be external", sourceSheet.Name)
		}
		part, cleanErr := excelCleanPart(workbookPart, relationship.Target)
		if cleanErr != nil || entries[part] == nil {
			return excelWorkbookData{}, fmt.Errorf("Excel.read: Sheet %q worksheet relationship is broken", sourceSheet.Name)
		}
		sheet, readErr := excelReadWorksheet(entries[part], sourceSheet.Name, sharedStrings, styles)
		if readErr != nil {
			return excelWorkbookData{}, readErr
		}
		workbook.Sheets = append(workbook.Sheets, sheet)
	}
	if err := excelValidateWorkbook(workbook); err != nil {
		return excelWorkbookData{}, fmt.Errorf("Excel.read: %w", err)
	}
	return workbook, nil
}

func ExcelRead(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("Excel.read: could not read XLSX file %q", filePath)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() < 0 || info.Size() > excelMaxArchiveBytes {
		return "", fmt.Errorf("Excel.read: XLSX file %q exceeds the archive size limit or cannot be inspected", filePath)
	}
	data, err := io.ReadAll(io.LimitReader(file, excelMaxArchiveBytes+1))
	if err != nil || len(data) > excelMaxArchiveBytes {
		return "", fmt.Errorf("Excel.read: could not read XLSX file %q safely", filePath)
	}
	workbook, err := excelReadPackage(data)
	if err != nil {
		return "", err
	}
	return excelEncode(workbook), nil
}

func excelAtomicWrite(filePath string, data []byte) error {
	absolute, err := filepath.Abs(filePath)
	if err != nil {
		return errors.New("destination path is invalid")
	}
	directory := filepath.Dir(absolute)
	temporary, err := os.CreateTemp(directory, ".ahdcode-excel-output-*")
	if err != nil {
		return errors.New("could not create an atomic output file")
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0644); err != nil {
		_ = temporary.Close()
		return errors.New("could not prepare the atomic output file")
	}
	_, writeErr := temporary.Write(data)
	syncErr := temporary.Sync()
	closeErr := temporary.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return errors.New("could not write the complete XLSX package")
	}
	if err := os.Rename(temporaryPath, absolute); err != nil {
		return errors.New("could not atomically replace the destination XLSX file")
	}
	return nil
}

func ExcelWorkbookSave(workbookText, filePath string) error {
	if !strings.EqualFold(filepath.Ext(filePath), ".xlsx") {
		return errors.New("Workbook.save: destination must use the .xlsx extension")
	}
	workbook, err := excelWorkbook(workbookText)
	if err != nil {
		return fmt.Errorf("Workbook.save: %w", err)
	}
	if len(workbook.Sheets) == 0 {
		return errors.New("Workbook.save: a Workbook must contain at least one Sheet")
	}
	data, err := excelBuildPackage(workbook)
	if err != nil {
		return errors.New("Workbook.save: could not build the XLSX package")
	}
	validated, err := excelReadPackage(data)
	if err != nil {
		return fmt.Errorf("Workbook.save: generated XLSX validation failed: %w", err)
	}
	if excelEncode(validated) != excelEncode(workbook) {
		return errors.New("Workbook.save: generated XLSX validation changed supported Workbook semantics")
	}
	if err := excelAtomicWrite(filePath, data); err != nil {
		return fmt.Errorf("Workbook.save: %w", err)
	}
	return nil
}

func excelRuntimeValue(class *AhdClass, value string, err error) string {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	return value
}

func excelRuntimeStrings(class *AhdClass, value []string, err error) []string {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	return value
}

func excelRuntimeGrid(class *AhdClass, value [][]string, err error) [][]string {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	return value
}

func excelRuntimeInt(class *AhdClass, value int64, err error) int64 {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	return value
}

func excelRuntimeReal(class *AhdClass, value float64, err error) float64 {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	return value
}

func excelRuntimeBool(class *AhdClass, value bool, err error) bool {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	return value
}

func AhdExcelNew() string { return ExcelNew() }
func AhdExcelRead(class *AhdClass, filePath string) string {
	value, err := ExcelRead(filePath)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelBlank() string { return ExcelBlank() }
func AhdExcelFromString(class *AhdClass, value string) string {
	result, err := ExcelFromString(value)
	return excelRuntimeValue(class, result, err)
}
func AhdExcelFromInt(value int64) string { return ExcelFromInt(value) }
func AhdExcelFromReal(class *AhdClass, value float64) string {
	result, err := ExcelFromReal(value)
	return excelRuntimeValue(class, result, err)
}
func AhdExcelFromBool(value bool) string { return ExcelFromBool(value) }
func AhdExcelFormula(class *AhdClass, value string) string {
	result, err := ExcelFormula(value)
	return excelRuntimeValue(class, result, err)
}
func AhdExcelStyle() string { return ExcelStyle() }

func AhdExcelWorkbookAddSheet(class *AhdClass, workbook, name string) string {
	value, err := ExcelWorkbookAddSheet(workbook, name)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelWorkbookSheet(class *AhdClass, workbook, name string) string {
	value, err := ExcelWorkbookSheet(workbook, name)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelWorkbookWithSheet(class *AhdClass, workbook, sheet string) string {
	value, err := ExcelWorkbookWithSheet(workbook, sheet)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelWorkbookSheets(class *AhdClass, workbook string) []string {
	value, err := ExcelWorkbookSheets(workbook)
	return excelRuntimeStrings(class, value, err)
}
func AhdExcelWorkbookSheetCount(class *AhdClass, workbook string) int64 {
	value, err := ExcelWorkbookSheetCount(workbook)
	return excelRuntimeInt(class, value, err)
}
func AhdExcelWorkbookSave(class *AhdClass, workbook, filePath string) {
	if err := ExcelWorkbookSave(workbook, filePath); err != nil {
		AhdRaiseClass(class, err.Error())
	}
}

func AhdExcelSheetName(class *AhdClass, sheet string) string {
	value, err := ExcelSheetName(sheet)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelSheetCell(class *AhdClass, sheet string, row, column int64) string {
	value, err := ExcelSheetCell(sheet, row, column)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelSheetSetCell(class *AhdClass, sheet string, row, column int64, cell string) string {
	value, err := ExcelSheetSetCell(sheet, row, column, cell)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelSheetRange(class *AhdClass, sheet string, startRow, startColumn, endRow, endColumn int64) string {
	value, err := ExcelSheetRange(sheet, startRow, startColumn, endRow, endColumn)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelSheetSetRow(class *AhdClass, sheet string, row, startColumn int64, cells []string) string {
	value, err := ExcelSheetSetRow(sheet, row, startColumn, cells)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelSheetSetRange(class *AhdClass, sheet, cellRange string, cells [][]string) string {
	value, err := ExcelSheetSetRange(sheet, cellRange, cells)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelSheetCells(class *AhdClass, sheet, cellRange string) [][]string {
	value, err := ExcelSheetCells(sheet, cellRange)
	return excelRuntimeGrid(class, value, err)
}
func AhdExcelSheetUsedRange(class *AhdClass, sheet string) *string {
	value, err := ExcelSheetUsedRange(sheet)
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
	return value
}
func AhdExcelSheetMerge(class *AhdClass, sheet, cellRange string) string {
	value, err := ExcelSheetMerge(sheet, cellRange)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelSheetMerges(class *AhdClass, sheet string) []string {
	value, err := ExcelSheetMerges(sheet)
	return excelRuntimeStrings(class, value, err)
}
func AhdExcelSheetStyle(class *AhdClass, sheet, cellRange, style string) string {
	value, err := ExcelSheetStyle(sheet, cellRange, style)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelSheetColumnWidth(class *AhdClass, sheet string, column int64, width float64) string {
	value, err := ExcelSheetColumnWidth(sheet, column, width)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelSheetRowHeight(class *AhdClass, sheet string, row int64, height float64) string {
	value, err := ExcelSheetRowHeight(sheet, row, height)
	return excelRuntimeValue(class, value, err)
}

func AhdExcelCellKind(class *AhdClass, cell string) string {
	value, err := ExcelCellKind(cell)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelCellIsBlank(class *AhdClass, cell string) bool {
	value, err := ExcelCellIsBlank(cell)
	return excelRuntimeBool(class, value, err)
}
func AhdExcelCellString(class *AhdClass, cell string) string {
	value, err := ExcelCellString(cell)
	return excelRuntimeValue(class, value, err)
}
func AhdExcelCellInt(class *AhdClass, cell string) int64 {
	value, err := ExcelCellInt(cell)
	return excelRuntimeInt(class, value, err)
}
func AhdExcelCellReal(class *AhdClass, cell string) float64 {
	value, err := ExcelCellReal(cell)
	return excelRuntimeReal(class, value, err)
}
func AhdExcelCellBool(class *AhdClass, cell string) bool {
	value, err := ExcelCellBool(cell)
	return excelRuntimeBool(class, value, err)
}
func AhdExcelCellFormula(class *AhdClass, cell string) string {
	value, err := ExcelCellFormula(cell)
	return excelRuntimeValue(class, value, err)
}

func AhdExcelRangeInt(class *AhdClass, cellRange, operation string) int64 {
	value, err := ExcelRangeInt(cellRange, operation)
	return excelRuntimeInt(class, value, err)
}
func AhdExcelRangeAddress(class *AhdClass, cellRange string) string {
	value, err := ExcelRangeAddress(cellRange)
	return excelRuntimeValue(class, value, err)
}

func AhdExcelStyleBool(class *AhdClass, style, operation string, value bool) string {
	result, err := ExcelStyleBool(style, operation, value)
	return excelRuntimeValue(class, result, err)
}
func AhdExcelStyleFontSize(class *AhdClass, style string, value float64) string {
	result, err := ExcelStyleFontSize(style, value)
	return excelRuntimeValue(class, result, err)
}
func AhdExcelStyleString(class *AhdClass, style, operation, value string) string {
	result, err := ExcelStyleString(style, operation, value)
	return excelRuntimeValue(class, result, err)
}
func AhdExcelStyleBorder(class *AhdClass, style, borderStyle, color string) string {
	value, err := ExcelStyleBorder(style, borderStyle, color)
	return excelRuntimeValue(class, value, err)
}
