package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const excelModuleID = "builtin:Excel"

var (
	excelErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	excelErrorClass    = &types.ClassSymbol{ModuleID: excelModuleID, Name: "ExcelError", Parent: excelErrorParent}
	excelWorkbookClass = &types.ClassSymbol{ModuleID: excelModuleID, Name: "Workbook"}
	excelSheetClass    = &types.ClassSymbol{ModuleID: excelModuleID, Name: "Sheet"}
	excelCellClass     = &types.ClassSymbol{ModuleID: excelModuleID, Name: "Cell"}
	excelRangeClass    = &types.ClassSymbol{ModuleID: excelModuleID, Name: "Range"}
	excelStyleClass    = &types.ClassSymbol{ModuleID: excelModuleID, Name: "CellStyle"}
)

func ExcelErrorIdentity() *types.ClassSymbol    { return excelErrorClass }
func ExcelWorkbookIdentity() *types.ClassSymbol { return excelWorkbookClass }
func ExcelSheetIdentity() *types.ClassSymbol    { return excelSheetClass }
func ExcelCellIdentity() *types.ClassSymbol     { return excelCellClass }
func ExcelRangeIdentity() *types.ClassSymbol    { return excelRangeClass }
func ExcelStyleIdentity() *types.ClassSymbol    { return excelStyleClass }

var ExcelWorkbookOperations = []string{"addSheet", "sheet", "withSheet", "sheets", "sheetCount", "save"}
var ExcelSheetOperations = []string{
	"name", "cell", "setCell", "range", "setRow", "setRange", "cells", "usedRange",
	"merge", "merges", "style", "columnWidth", "rowHeight",
}
var ExcelCellOperations = []string{"kind", "isBlank", "string", "int", "real", "bool", "formula"}
var ExcelRangeOperations = []string{
	"startRow", "startColumn", "endRow", "endColumn", "rowCount", "columnCount", "address",
}
var ExcelStyleOperations = []string{
	"bold", "italic", "underline", "fontSize", "textColor", "fillColor", "horizontal",
	"vertical", "wrap", "numberFormat", "border",
}

func excelWorkbookType() types.Type { return types.Class{Symbol: excelWorkbookClass} }
func excelSheetType() types.Type    { return types.Class{Symbol: excelSheetClass} }
func excelCellType() types.Type     { return types.Class{Symbol: excelCellClass} }
func excelRangeType() types.Type    { return types.Class{Symbol: excelRangeClass} }
func excelStyleType() types.Type    { return types.Class{Symbol: excelStyleClass} }

func excelModuleInterface() *ModuleInterface {
	module := standardInterface(excelModuleID, "Excel")
	classes := []struct {
		name     string
		identity *types.ClassSymbol
	}{
		{"ExcelError", excelErrorClass}, {"Workbook", excelWorkbookClass}, {"Sheet", excelSheetClass},
		{"Cell", excelCellClass}, {"Range", excelRangeClass}, {"CellStyle", excelStyleClass},
	}
	for _, entry := range classes {
		symbol := &Symbol{
			Name: entry.name, Kind: ClassSymbol, Class: entry.identity,
			Type: types.Class{Symbol: entry.identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: excelModuleID,
			Members: make(map[string]*Symbol),
		}
		if entry.name == "ExcelError" {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[excelModuleID+"\x00"+entry.name] = symbol
		addStandardExport(module, symbol)
	}
	stringParameter := func(name string) types.Parameter { return types.Parameter{Name: name, Type: types.String} }
	addStandardExport(module, standardFunction(excelModuleID, "new", excelWorkbookType()))
	addStandardExport(module, standardFunction(excelModuleID, "read", excelWorkbookType(), stringParameter("path")))
	addStandardExport(module, standardFunction(excelModuleID, "blank", excelCellType()))
	addStandardExport(module, standardFunction(excelModuleID, "fromString", excelCellType(), stringParameter("value")))
	addStandardExport(module, standardFunction(excelModuleID, "fromInt", excelCellType(), types.Parameter{Name: "value", Type: types.Int}))
	addStandardExport(module, standardFunction(excelModuleID, "fromReal", excelCellType(), types.Parameter{Name: "value", Type: types.Real}))
	addStandardExport(module, standardFunction(excelModuleID, "fromBool", excelCellType(), types.Parameter{Name: "value", Type: types.Bool}))
	addStandardExport(module, standardFunction(excelModuleID, "formula", excelCellType(), stringParameter("expression")))
	addStandardExport(module, standardFunction(excelModuleID, "style", excelStyleType()))
	sort.Strings(module.ExportNames)
	return module
}

func excelConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != excelModuleID {
		return "", false
	}
	switch identity.Name {
	case "Workbook":
		return "create a Workbook with Excel.new() or Excel.read(path)", true
	case "Sheet":
		return "obtain a Sheet with Workbook.sheet(name)", true
	case "Cell":
		return "create a Cell with Excel.blank(), Excel.fromString/fromInt/fromReal/fromBool(value), or Excel.formula(expression)", true
	case "Range":
		return "create a Range with Sheet.range(startRow, startColumn, endRow, endColumn)", true
	case "CellStyle":
		return "create a CellStyle patch with Excel.style()", true
	}
	return "", false
}

type excelOperationShape struct {
	parameters     []types.Type
	result         types.Type
	resultNullable bool
	hint           string
}

func excelOperationShapes() map[TypeOperation]excelOperationShape {
	none := []types.Type{}
	workbook, sheet, cell, cellRange, style := excelWorkbookType(), excelSheetType(), excelCellType(), excelRangeType(), excelStyleType()
	cellList := types.List{Element: cell}
	cellGrid := types.List{Element: cellList}
	return map[TypeOperation]excelOperationShape{
		ExcelWorkbookAddSheet:   {[]types.Type{types.String}, workbook, false, "pass one new Sheet name"},
		ExcelWorkbookSheet:      {[]types.Type{types.String}, sheet, false, "pass one existing Sheet name"},
		ExcelWorkbookWithSheet:  {[]types.Type{sheet}, workbook, false, "pass one Sheet snapshot already present by name"},
		ExcelWorkbookSheets:     {none, types.List{Element: types.String}, false, "call sheets with no argument"},
		ExcelWorkbookSheetCount: {none, types.Int, false, "call sheetCount with no argument"},
		ExcelWorkbookSave:       {[]types.Type{types.String}, types.Nothing, false, "pass one destination .xlsx path"},

		ExcelSheetName:        {none, types.String, false, "call name with no argument"},
		ExcelSheetCell:        {[]types.Type{types.Int, types.Int}, cell, false, "pass 1-based row and column Int coordinates"},
		ExcelSheetSetCell:     {[]types.Type{types.Int, types.Int, cell}, sheet, false, "pass 1-based row and column Int coordinates and one Cell"},
		ExcelSheetRange:       {[]types.Type{types.Int, types.Int, types.Int, types.Int}, cellRange, false, "pass start row/column and end row/column Int coordinates"},
		ExcelSheetSetRow:      {[]types.Type{types.Int, types.Int, cellList}, sheet, false, "pass a row, start column, and List<Cell>"},
		ExcelSheetSetRange:    {[]types.Type{cellRange, cellGrid}, sheet, false, "pass a Range and an exact rectangular List<List<Cell>>"},
		ExcelSheetCells:       {[]types.Type{cellRange}, cellGrid, false, "pass one Range"},
		ExcelSheetUsedRange:   {none, cellRange, true, "call usedRange with no argument"},
		ExcelSheetMerge:       {[]types.Type{cellRange}, sheet, false, "pass one non-overlapping safe Range"},
		ExcelSheetMerges:      {none, types.List{Element: cellRange}, false, "call merges with no argument"},
		ExcelSheetStyle:       {[]types.Type{cellRange, style}, sheet, false, "pass a Range and CellStyle patch"},
		ExcelSheetColumnWidth: {[]types.Type{types.Int, types.Real}, sheet, false, "pass a 1-based column and positive finite width"},
		ExcelSheetRowHeight:   {[]types.Type{types.Int, types.Real}, sheet, false, "pass a 1-based row and positive finite height"},

		ExcelCellKind:    {none, types.String, false, "call kind with no argument"},
		ExcelCellIsBlank: {none, types.Bool, false, "call isBlank with no argument"},
		ExcelCellString:  {none, types.String, false, "call string on a String Cell"},
		ExcelCellInt:     {none, types.Int, false, "call int on an Int Cell"},
		ExcelCellReal:    {none, types.Real, false, "call real on a Real or Int Cell"},
		ExcelCellBool:    {none, types.Bool, false, "call bool on a Bool Cell"},
		ExcelCellFormula: {none, types.String, false, "call formula on a Formula Cell"},

		ExcelRangeStartRow:    {none, types.Int, false, "call startRow with no argument"},
		ExcelRangeStartColumn: {none, types.Int, false, "call startColumn with no argument"},
		ExcelRangeEndRow:      {none, types.Int, false, "call endRow with no argument"},
		ExcelRangeEndColumn:   {none, types.Int, false, "call endColumn with no argument"},
		ExcelRangeRowCount:    {none, types.Int, false, "call rowCount with no argument"},
		ExcelRangeColumnCount: {none, types.Int, false, "call columnCount with no argument"},
		ExcelRangeAddress:     {none, types.String, false, "call address with no argument"},

		ExcelStyleBold:         {[]types.Type{types.Bool}, style, false, "pass one Bool"},
		ExcelStyleItalic:       {[]types.Type{types.Bool}, style, false, "pass one Bool"},
		ExcelStyleUnderline:    {[]types.Type{types.Bool}, style, false, "pass one Bool"},
		ExcelStyleFontSize:     {[]types.Type{types.Real}, style, false, "pass a positive finite Real font size"},
		ExcelStyleTextColor:    {[]types.Type{types.String}, style, false, "pass a #RRGGBB color"},
		ExcelStyleFillColor:    {[]types.Type{types.String}, style, false, "pass a #RRGGBB color"},
		ExcelStyleHorizontal:   {[]types.Type{types.String}, style, false, "pass left, center, or right"},
		ExcelStyleVertical:     {[]types.Type{types.String}, style, false, "pass top, center, or bottom"},
		ExcelStyleWrap:         {[]types.Type{types.Bool}, style, false, "pass one Bool"},
		ExcelStyleNumberFormat: {[]types.Type{types.String}, style, false, "pass one explicit Excel number-format String"},
		ExcelStyleBorder:       {[]types.Type{types.String, types.String}, style, false, "pass a supported border style and #RRGGBB color"},
	}
}

var excelOperationNames = map[string]map[string]TypeOperation{
	"Workbook": {"addSheet": ExcelWorkbookAddSheet, "sheet": ExcelWorkbookSheet, "withSheet": ExcelWorkbookWithSheet,
		"sheets": ExcelWorkbookSheets, "sheetCount": ExcelWorkbookSheetCount, "save": ExcelWorkbookSave},
	"Sheet": {"name": ExcelSheetName, "cell": ExcelSheetCell, "setCell": ExcelSheetSetCell, "range": ExcelSheetRange,
		"setRow": ExcelSheetSetRow, "setRange": ExcelSheetSetRange, "cells": ExcelSheetCells, "usedRange": ExcelSheetUsedRange,
		"merge": ExcelSheetMerge, "merges": ExcelSheetMerges, "style": ExcelSheetStyle,
		"columnWidth": ExcelSheetColumnWidth, "rowHeight": ExcelSheetRowHeight},
	"Cell": {"kind": ExcelCellKind, "isBlank": ExcelCellIsBlank, "string": ExcelCellString, "int": ExcelCellInt,
		"real": ExcelCellReal, "bool": ExcelCellBool, "formula": ExcelCellFormula},
	"Range": {"startRow": ExcelRangeStartRow, "startColumn": ExcelRangeStartColumn, "endRow": ExcelRangeEndRow,
		"endColumn": ExcelRangeEndColumn, "rowCount": ExcelRangeRowCount, "columnCount": ExcelRangeColumnCount,
		"address": ExcelRangeAddress},
	"CellStyle": {"bold": ExcelStyleBold, "italic": ExcelStyleItalic, "underline": ExcelStyleUnderline,
		"fontSize": ExcelStyleFontSize, "textColor": ExcelStyleTextColor, "fillColor": ExcelStyleFillColor,
		"horizontal": ExcelStyleHorizontal, "vertical": ExcelStyleVertical, "wrap": ExcelStyleWrap,
		"numberFormat": ExcelStyleNumberFormat, "border": ExcelStyleBorder},
}

func excelOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != excelModuleID {
		return "", false
	}
	operation, known := excelOperationNames[class.Symbol.Name][name]
	return operation, known
}

func (a *analyzer) analyzeExcelOperation(call *ast.CallExpr, operation TypeOperation, shape excelOperationShape, current *scope, flow flowState) expressionInfo {
	nullState := NonNull
	if shape.resultNullable {
		nullState = MaybeNull
	}
	result := expressionInfo{typeValue: shape.result, nullState: nullState}
	if len(call.Arguments) != len(shape.parameters) {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument(s); received %d", operation, len(shape.parameters), len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for index, expected := range shape.parameters {
		argument := a.analyzeExpressionExpected(call.Arguments[index].Value, current, flow, expected)
		if argument.invalid() {
			continue
		}
		if argument.nullState != NonNull {
			a.nullableError(string(operation), call.Arguments[index].Value, argument.nullState)
			continue
		}
		if !types.Assignable(expected, argument.typeValue) {
			a.typeMismatch(call.Arguments[index].Span(), expected, argument.typeValue, string(operation)+" argument")
		}
	}
	// Excel operations take positional-only arguments whose parameter types
	// differ per index (for example CellStyle.fontSize expects Real while
	// Sheet.columnWidth expects Int then Real), so a single receiver-derived
	// expected type cannot describe them the way typeOperationArgument does
	// for the homogeneous TypeOperation modules. Recording the exact
	// per-call signature here lets lowering give each argument its own
	// expected type -- including safe Int -> Real widening -- the same way
	// moduleOperationResult already does for Lists and KeyValue.
	parameters := make([]types.Parameter, len(shape.parameters))
	for index, parameterType := range shape.parameters {
		parameters[index] = types.Parameter{Type: parameterType}
	}
	a.result.SelectedCallables[call] = &Callable{
		Signature:  &types.Signature{Parameters: parameters, Return: shape.result},
		ReturnNull: nullState,
	}
	return result
}
