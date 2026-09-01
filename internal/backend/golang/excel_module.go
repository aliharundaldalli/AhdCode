package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const excelModulePrefix = "builtin:Excel::"

var (
	excelWorkbookClass = ir.ClassID("builtin:Excel::class::Workbook")
	excelSheetClass    = ir.ClassID("builtin:Excel::class::Sheet")
	excelCellClass     = ir.ClassID("builtin:Excel::class::Cell")
	excelRangeClass    = ir.ClassID("builtin:Excel::class::Range")
	excelStyleClass    = ir.ClassID("builtin:Excel::class::CellStyle")
	excelErrorClass    = ir.ClassID("builtin:Excel::class::ExcelError")

	excelWorkbookDataField = ir.FieldID("builtin:Excel::class::Workbook::field::data")
	excelSheetDataField    = ir.FieldID("builtin:Excel::class::Sheet::field::data")
	excelCellDataField     = ir.FieldID("builtin:Excel::class::Cell::field::data")
	excelRangeDataField    = ir.FieldID("builtin:Excel::class::Range::field::data")
	excelStyleDataField    = ir.FieldID("builtin:Excel::class::CellStyle::field::data")
)

func (generator *generator) excelCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), excelModulePrefix)
	errorClass := generator.descriptorName(excelErrorClass)
	text := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	switch name {
	case "new":
		return generator.excelValueFrom(excelWorkbookClass, "AhdExcelNew()", meta)
	case "read":
		return generator.excelValueFrom(excelWorkbookClass, "AhdExcelRead("+errorClass+", "+text(0)+")", meta)
	case "blank":
		return generator.excelValueFrom(excelCellClass, "AhdExcelBlank()", meta)
	case "fromString":
		return generator.excelValueFrom(excelCellClass, "AhdExcelFromString("+errorClass+", "+text(0)+")", meta)
	case "fromInt":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.IntType}, false)
		return generator.excelValueFrom(excelCellClass, "AhdExcelFromInt("+argument+")", meta)
	case "fromReal":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.RealType}, false)
		return generator.excelValueFrom(excelCellClass, "AhdExcelFromReal("+errorClass+", "+argument+")", meta)
	case "fromBool":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.BoolType}, false)
		return generator.excelValueFrom(excelCellClass, "AhdExcelFromBool("+argument+")", meta)
	case "formula":
		return generator.excelValueFrom(excelCellClass, "AhdExcelFormula("+errorClass+", "+text(0)+")", meta)
	case "style":
		return generator.excelValueFrom(excelStyleClass, "AhdExcelStyle()", meta)
	default:
		return generator.unsupported("Excel function "+name, meta.Span)
	}
}

func excelDataField(class ir.ClassID) ir.FieldID {
	switch class {
	case excelWorkbookClass:
		return excelWorkbookDataField
	case excelSheetClass:
		return excelSheetDataField
	case excelCellClass:
		return excelCellDataField
	case excelRangeClass:
		return excelRangeDataField
	case excelStyleClass:
		return excelStyleDataField
	default:
		return ""
	}
}

func (generator *generator) excelValueFrom(class ir.ClassID, data string, meta ir.ExprBase) string {
	helper, ok := generator.excelHelper(class)
	if !ok {
		return generator.unsupported("an Excel value without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

func (generator *generator) excelDataOf(class ir.ClassID, expression ir.Expr) string {
	rendered := generator.expr(expression)
	getter := generator.fieldName(excelDataField(class)) + "_get()"
	return "func(value " + generator.interfaceName(class) + ") string { return value." + getter + " }(" + rendered + ")"
}

func (generator *generator) excelHelper(class ir.ClassID) (string, bool) {
	if generator.layouts[class] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[class]; known {
		return name, true
	}
	name := mangleNamed("eh_", generator.classDisplayName(class), string(class))
	generator.timeHelpers[class] = name
	return name, true
}

func (generator *generator) emitExcelHelpers(writer *emitter) {
	for _, class := range []ir.ClassID{excelWorkbookClass, excelSheetClass, excelCellClass, excelRangeClass, excelStyleClass} {
		name, known := generator.timeHelpers[class]
		if !known {
			continue
		}
		layout := generator.layouts[class]
		if layout == nil {
			continue
		}
		constructor := generator.functions[layout.class.Constructor]
		if constructor == nil {
			continue
		}
		writer.open("func " + name + "(data string) " + generator.interfaceName(class) + " {")
		writer.line("return " + generator.callableName(constructor) + "(data)")
		writer.close("}")
		writer.blank()
	}
}

func (generator *generator) excelOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(excelErrorClass)
	text := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	integer := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	real := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.RealType}, false)
	}
	boolean := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}
	switch name {
	case "Workbook.addSheet":
		receiver := generator.excelDataOf(excelWorkbookClass, value.Callee)
		return generator.excelValueFrom(excelWorkbookClass, "AhdExcelWorkbookAddSheet("+errorClass+", "+receiver+", "+text(0)+")", meta)
	case "Workbook.sheet":
		receiver := generator.excelDataOf(excelWorkbookClass, value.Callee)
		return generator.excelValueFrom(excelSheetClass, "AhdExcelWorkbookSheet("+errorClass+", "+receiver+", "+text(0)+")", meta)
	case "Workbook.withSheet":
		receiver := generator.excelDataOf(excelWorkbookClass, value.Callee)
		sheet := generator.excelDataOf(excelSheetClass, value.Arguments[0].Value)
		return generator.excelValueFrom(excelWorkbookClass, "AhdExcelWorkbookWithSheet("+errorClass+", "+receiver+", "+sheet+")", meta)
	case "Workbook.sheets":
		return "AhdNewList(AhdExcelWorkbookSheets(" + errorClass + ", " + generator.excelDataOf(excelWorkbookClass, value.Callee) + ")...)"
	case "Workbook.sheetCount":
		return "AhdExcelWorkbookSheetCount(" + errorClass + ", " + generator.excelDataOf(excelWorkbookClass, value.Callee) + ")"
	case "Workbook.save":
		return "AhdExcelWorkbookSave(" + errorClass + ", " + generator.excelDataOf(excelWorkbookClass, value.Callee) + ", " + text(0) + ")"

	case "Sheet.name":
		return "AhdExcelSheetName(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ")"
	case "Sheet.cell":
		data := "AhdExcelSheetCell(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ", " + integer(0) + ", " + integer(1) + ")"
		return generator.excelValueFrom(excelCellClass, data, meta)
	case "Sheet.setCell":
		data := "AhdExcelSheetSetCell(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ", " + integer(0) + ", " + integer(1) + ", " + generator.excelDataOf(excelCellClass, value.Arguments[2].Value) + ")"
		return generator.excelValueFrom(excelSheetClass, data, meta)
	case "Sheet.range":
		data := "AhdExcelSheetRange(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ", " + integer(0) + ", " + integer(1) + ", " + integer(2) + ", " + integer(3) + ")"
		return generator.excelValueFrom(excelRangeClass, data, meta)
	case "Sheet.setRow":
		data := "AhdExcelSheetSetRow(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ", " + integer(0) + ", " + integer(1) + ", " + generator.excelCellTexts(value.Arguments[2].Value) + ")"
		return generator.excelValueFrom(excelSheetClass, data, meta)
	case "Sheet.setRange":
		data := "AhdExcelSheetSetRange(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ", " + generator.excelDataOf(excelRangeClass, value.Arguments[0].Value) + ", " + generator.excelCellGridTexts(value.Arguments[1].Value) + ")"
		return generator.excelValueFrom(excelSheetClass, data, meta)
	case "Sheet.cells":
		data := "AhdExcelSheetCells(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ", " + generator.excelDataOf(excelRangeClass, value.Arguments[0].Value) + ")"
		return generator.excelCellGridResult(data, meta)
	case "Sheet.usedRange":
		data := "AhdExcelSheetUsedRange(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ")"
		return generator.excelNullableResult(excelRangeClass, data, meta)
	case "Sheet.merge":
		data := "AhdExcelSheetMerge(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ", " + generator.excelDataOf(excelRangeClass, value.Arguments[0].Value) + ")"
		return generator.excelValueFrom(excelSheetClass, data, meta)
	case "Sheet.merges":
		data := "AhdExcelSheetMerges(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ")"
		return generator.excelListResult(excelRangeClass, data, meta)
	case "Sheet.style":
		data := "AhdExcelSheetStyle(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ", " + generator.excelDataOf(excelRangeClass, value.Arguments[0].Value) + ", " + generator.excelDataOf(excelStyleClass, value.Arguments[1].Value) + ")"
		return generator.excelValueFrom(excelSheetClass, data, meta)
	case "Sheet.columnWidth":
		data := "AhdExcelSheetColumnWidth(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ", " + integer(0) + ", " + real(1) + ")"
		return generator.excelValueFrom(excelSheetClass, data, meta)
	case "Sheet.rowHeight":
		data := "AhdExcelSheetRowHeight(" + errorClass + ", " + generator.excelDataOf(excelSheetClass, value.Callee) + ", " + integer(0) + ", " + real(1) + ")"
		return generator.excelValueFrom(excelSheetClass, data, meta)

	case "Cell.kind":
		return "AhdExcelCellKind(" + errorClass + ", " + generator.excelDataOf(excelCellClass, value.Callee) + ")"
	case "Cell.isBlank":
		return "AhdExcelCellIsBlank(" + errorClass + ", " + generator.excelDataOf(excelCellClass, value.Callee) + ")"
	case "Cell.string":
		return "AhdExcelCellString(" + errorClass + ", " + generator.excelDataOf(excelCellClass, value.Callee) + ")"
	case "Cell.int":
		return "AhdExcelCellInt(" + errorClass + ", " + generator.excelDataOf(excelCellClass, value.Callee) + ")"
	case "Cell.real":
		return "AhdExcelCellReal(" + errorClass + ", " + generator.excelDataOf(excelCellClass, value.Callee) + ")"
	case "Cell.bool":
		return "AhdExcelCellBool(" + errorClass + ", " + generator.excelDataOf(excelCellClass, value.Callee) + ")"
	case "Cell.formula":
		return "AhdExcelCellFormula(" + errorClass + ", " + generator.excelDataOf(excelCellClass, value.Callee) + ")"

	case "Range.startRow", "Range.startColumn", "Range.endRow", "Range.endColumn", "Range.rowCount", "Range.columnCount":
		operation := strings.TrimPrefix(name, "Range.")
		return "AhdExcelRangeInt(" + errorClass + ", " + generator.excelDataOf(excelRangeClass, value.Callee) + ", " + quote(operation) + ")"
	case "Range.address":
		return "AhdExcelRangeAddress(" + errorClass + ", " + generator.excelDataOf(excelRangeClass, value.Callee) + ")"

	case "CellStyle.bold", "CellStyle.italic", "CellStyle.underline", "CellStyle.wrap":
		operation := strings.TrimPrefix(name, "CellStyle.")
		data := "AhdExcelStyleBool(" + errorClass + ", " + generator.excelDataOf(excelStyleClass, value.Callee) + ", " + quote(operation) + ", " + boolean(0) + ")"
		return generator.excelValueFrom(excelStyleClass, data, meta)
	case "CellStyle.fontSize":
		data := "AhdExcelStyleFontSize(" + errorClass + ", " + generator.excelDataOf(excelStyleClass, value.Callee) + ", " + real(0) + ")"
		return generator.excelValueFrom(excelStyleClass, data, meta)
	case "CellStyle.textColor", "CellStyle.fillColor", "CellStyle.horizontal", "CellStyle.vertical", "CellStyle.numberFormat":
		operation := strings.TrimPrefix(name, "CellStyle.")
		data := "AhdExcelStyleString(" + errorClass + ", " + generator.excelDataOf(excelStyleClass, value.Callee) + ", " + quote(operation) + ", " + text(0) + ")"
		return generator.excelValueFrom(excelStyleClass, data, meta)
	case "CellStyle.border":
		data := "AhdExcelStyleBorder(" + errorClass + ", " + generator.excelDataOf(excelStyleClass, value.Callee) + ", " + text(0) + ", " + text(1) + ")"
		return generator.excelValueFrom(excelStyleClass, data, meta)
	default:
		return generator.unsupported("Excel operation "+name, meta.Span)
	}
}

func (generator *generator) excelCellTexts(expression ir.Expr) string {
	rendered := generator.expr(expression)
	element := generator.interfaceName(excelCellClass)
	getter := generator.fieldName(excelCellDataField) + "_get()"
	return "func(list *AhdList[" + element + "]) []string { items := list.Snapshot(); result := make([]string, len(items)); for index, item := range items { result[index] = item." + getter + " }; return result }(" + rendered + ")"
}

func (generator *generator) excelCellGridTexts(expression ir.Expr) string {
	rendered := generator.expr(expression)
	element := generator.interfaceName(excelCellClass)
	getter := generator.fieldName(excelCellDataField) + "_get()"
	return "func(grid *AhdList[*AhdList[" + element + "]]) [][]string { rows := grid.Snapshot(); result := make([][]string, len(rows)); for r, row := range rows { items := row.Snapshot(); result[r] = make([]string, len(items)); for c, item := range items { result[r][c] = item." + getter + " } }; return result }(" + rendered + ")"
}

func (generator *generator) excelListResult(class ir.ClassID, data string, meta ir.ExprBase) string {
	helper, ok := generator.excelHelper(class)
	if !ok {
		return generator.unsupported("an Excel List result without its Class declaration", meta.Span)
	}
	element := generator.interfaceName(class)
	return "func(values []string) *AhdList[" + element + "] { result := make([]" + element + ", len(values)); for index, value := range values { result[index] = " + helper + "(value) }; return AhdNewList(result...) }(" + data + ")"
}

func (generator *generator) excelCellGridResult(data string, meta ir.ExprBase) string {
	helper, ok := generator.excelHelper(excelCellClass)
	if !ok {
		return generator.unsupported("an Excel cell grid without the Cell Class declaration", meta.Span)
	}
	element := generator.interfaceName(excelCellClass)
	return "func(values [][]string) *AhdList[*AhdList[" + element + "]] { rows := make([]*AhdList[" + element + "], len(values)); for r, row := range values { cells := make([]" + element + ", len(row)); for c, cell := range row { cells[c] = " + helper + "(cell) }; rows[r] = AhdNewList(cells...) }; return AhdNewList(rows...) }(" + data + ")"
}

func (generator *generator) excelNullableResult(class ir.ClassID, data string, meta ir.ExprBase) string {
	helper, ok := generator.excelHelper(class)
	if !ok {
		return generator.unsupported("a nullable Excel result without its Class declaration", meta.Span)
	}
	element := generator.interfaceName(class)
	return "func(value *string) " + element + " { if value == nil { return nil }; return " + helper + "(*value) }(" + data + ")"
}
