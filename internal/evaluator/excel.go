package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

var (
	evaluatorExcelWorkbookClass = ir.ClassID("builtin:Excel::class::Workbook")
	evaluatorExcelSheetClass    = ir.ClassID("builtin:Excel::class::Sheet")
	evaluatorExcelCellClass     = ir.ClassID("builtin:Excel::class::Cell")
	evaluatorExcelRangeClass    = ir.ClassID("builtin:Excel::class::Range")
	evaluatorExcelStyleClass    = ir.ClassID("builtin:Excel::class::CellStyle")

	evaluatorExcelWorkbookField = ir.FieldID("builtin:Excel::class::Workbook::field::data")
	evaluatorExcelSheetField    = ir.FieldID("builtin:Excel::class::Sheet::field::data")
	evaluatorExcelCellField     = ir.FieldID("builtin:Excel::class::Cell::field::data")
	evaluatorExcelRangeField    = ir.FieldID("builtin:Excel::class::Range::field::data")
	evaluatorExcelStyleField    = ir.FieldID("builtin:Excel::class::CellStyle::field::data")
)

func evaluatorExcelField(class ir.ClassID) ir.FieldID {
	switch class {
	case evaluatorExcelWorkbookClass:
		return evaluatorExcelWorkbookField
	case evaluatorExcelSheetClass:
		return evaluatorExcelSheetField
	case evaluatorExcelCellClass:
		return evaluatorExcelCellField
	case evaluatorExcelRangeClass:
		return evaluatorExcelRangeField
	case evaluatorExcelStyleClass:
		return evaluatorExcelStyleField
	default:
		return ""
	}
}

func (session *Session) excelValue(class ir.ClassID, data string) *Instance {
	return &Instance{Class: class, Fields: map[ir.FieldID]any{evaluatorExcelField(class): data}}
}

func (session *Session) excelData(value any, class ir.ClassID) string {
	instance := session.requireInstance(value)
	data, ok := instance.Fields[evaluatorExcelField(class)].(string)
	if !ok || instance.Class != class {
		session.raise("ExcelError", "Excel value storage is corrupted")
	}
	return data
}

func (session *Session) excelResult(value string, err error) string {
	if err != nil {
		session.raise("ExcelError", err.Error())
	}
	return value
}

func (session *Session) excelBuiltin(name string, args []any) any {
	switch name {
	case "new":
		return session.excelValue(evaluatorExcelWorkbookClass, ahdruntime.ExcelNew())
	case "read":
		value, err := ahdruntime.ExcelRead(args[0].(string))
		return session.excelValue(evaluatorExcelWorkbookClass, session.excelResult(value, err))
	case "blank":
		return session.excelValue(evaluatorExcelCellClass, ahdruntime.ExcelBlank())
	case "fromString":
		value, err := ahdruntime.ExcelFromString(args[0].(string))
		return session.excelValue(evaluatorExcelCellClass, session.excelResult(value, err))
	case "fromInt":
		return session.excelValue(evaluatorExcelCellClass, ahdruntime.ExcelFromInt(args[0].(int64)))
	case "fromReal":
		value, err := ahdruntime.ExcelFromReal(args[0].(float64))
		return session.excelValue(evaluatorExcelCellClass, session.excelResult(value, err))
	case "fromBool":
		return session.excelValue(evaluatorExcelCellClass, ahdruntime.ExcelFromBool(args[0].(bool)))
	case "formula":
		value, err := ahdruntime.ExcelFormula(args[0].(string))
		return session.excelValue(evaluatorExcelCellClass, session.excelResult(value, err))
	case "style":
		return session.excelValue(evaluatorExcelStyleClass, ahdruntime.ExcelStyle())
	default:
		session.raise("Error", "unsupported Excel function "+name)
	}
	return nil
}

func (session *Session) excelCellTexts(value any) []string {
	list := session.requireList(value)
	result := make([]string, len(list.Items))
	for index, item := range list.Items {
		result[index] = session.excelData(item, evaluatorExcelCellClass)
	}
	return result
}

func (session *Session) excelCellGridTexts(value any) [][]string {
	list := session.requireList(value)
	result := make([][]string, len(list.Items))
	for index, row := range list.Items {
		result[index] = session.excelCellTexts(row)
	}
	return result
}

func (session *Session) excelValues(class ir.ClassID, values []string) *List {
	items := make([]any, len(values))
	for index, value := range values {
		items[index] = session.excelValue(class, value)
	}
	return &List{Items: items}
}

func (session *Session) excelCellGrid(values [][]string) *List {
	rows := make([]any, len(values))
	for index, row := range values {
		rows[index] = session.excelValues(evaluatorExcelCellClass, row)
	}
	return &List{Items: rows}
}

func (session *Session) excelOperation(name string, receiver any, args []any) any {
	workbook := func() string { return session.excelData(receiver, evaluatorExcelWorkbookClass) }
	sheet := func() string { return session.excelData(receiver, evaluatorExcelSheetClass) }
	cell := func() string { return session.excelData(receiver, evaluatorExcelCellClass) }
	cellRange := func() string { return session.excelData(receiver, evaluatorExcelRangeClass) }
	style := func() string { return session.excelData(receiver, evaluatorExcelStyleClass) }
	resultValue := func(class ir.ClassID, value string, err error) any {
		return session.excelValue(class, session.excelResult(value, err))
	}
	switch name {
	case "Workbook.addSheet":
		value, err := ahdruntime.ExcelWorkbookAddSheet(workbook(), args[0].(string))
		return resultValue(evaluatorExcelWorkbookClass, value, err)
	case "Workbook.sheet":
		value, err := ahdruntime.ExcelWorkbookSheet(workbook(), args[0].(string))
		return resultValue(evaluatorExcelSheetClass, value, err)
	case "Workbook.withSheet":
		value, err := ahdruntime.ExcelWorkbookWithSheet(workbook(), session.excelData(args[0], evaluatorExcelSheetClass))
		return resultValue(evaluatorExcelWorkbookClass, value, err)
	case "Workbook.sheets":
		values, err := ahdruntime.ExcelWorkbookSheets(workbook())
		if err != nil {
			session.raise("ExcelError", err.Error())
		}
		items := make([]any, len(values))
		for i, value := range values {
			items[i] = value
		}
		return &List{Items: items}
	case "Workbook.sheetCount":
		value, err := ahdruntime.ExcelWorkbookSheetCount(workbook())
		if err != nil {
			session.raise("ExcelError", err.Error())
		}
		return value
	case "Workbook.save":
		if err := ahdruntime.ExcelWorkbookSave(workbook(), args[0].(string)); err != nil {
			session.raise("ExcelError", err.Error())
		}
		return Nothing

	case "Sheet.name":
		value, err := ahdruntime.ExcelSheetName(sheet())
		return session.excelResult(value, err)
	case "Sheet.cell":
		value, err := ahdruntime.ExcelSheetCell(sheet(), args[0].(int64), args[1].(int64))
		return resultValue(evaluatorExcelCellClass, value, err)
	case "Sheet.setCell":
		value, err := ahdruntime.ExcelSheetSetCell(sheet(), args[0].(int64), args[1].(int64), session.excelData(args[2], evaluatorExcelCellClass))
		return resultValue(evaluatorExcelSheetClass, value, err)
	case "Sheet.range":
		value, err := ahdruntime.ExcelSheetRange(sheet(), args[0].(int64), args[1].(int64), args[2].(int64), args[3].(int64))
		return resultValue(evaluatorExcelRangeClass, value, err)
	case "Sheet.setRow":
		value, err := ahdruntime.ExcelSheetSetRow(sheet(), args[0].(int64), args[1].(int64), session.excelCellTexts(args[2]))
		return resultValue(evaluatorExcelSheetClass, value, err)
	case "Sheet.setRange":
		value, err := ahdruntime.ExcelSheetSetRange(sheet(), session.excelData(args[0], evaluatorExcelRangeClass), session.excelCellGridTexts(args[1]))
		return resultValue(evaluatorExcelSheetClass, value, err)
	case "Sheet.cells":
		values, err := ahdruntime.ExcelSheetCells(sheet(), session.excelData(args[0], evaluatorExcelRangeClass))
		if err != nil {
			session.raise("ExcelError", err.Error())
		}
		return session.excelCellGrid(values)
	case "Sheet.usedRange":
		value, err := ahdruntime.ExcelSheetUsedRange(sheet())
		if err != nil {
			session.raise("ExcelError", err.Error())
		}
		if value == nil {
			return nil
		}
		return session.excelValue(evaluatorExcelRangeClass, *value)
	case "Sheet.merge":
		value, err := ahdruntime.ExcelSheetMerge(sheet(), session.excelData(args[0], evaluatorExcelRangeClass))
		return resultValue(evaluatorExcelSheetClass, value, err)
	case "Sheet.merges":
		values, err := ahdruntime.ExcelSheetMerges(sheet())
		if err != nil {
			session.raise("ExcelError", err.Error())
		}
		return session.excelValues(evaluatorExcelRangeClass, values)
	case "Sheet.style":
		value, err := ahdruntime.ExcelSheetStyle(sheet(), session.excelData(args[0], evaluatorExcelRangeClass), session.excelData(args[1], evaluatorExcelStyleClass))
		return resultValue(evaluatorExcelSheetClass, value, err)
	case "Sheet.columnWidth":
		value, err := ahdruntime.ExcelSheetColumnWidth(sheet(), args[0].(int64), args[1].(float64))
		return resultValue(evaluatorExcelSheetClass, value, err)
	case "Sheet.rowHeight":
		value, err := ahdruntime.ExcelSheetRowHeight(sheet(), args[0].(int64), args[1].(float64))
		return resultValue(evaluatorExcelSheetClass, value, err)

	case "Cell.kind":
		value, err := ahdruntime.ExcelCellKind(cell())
		return session.excelResult(value, err)
	case "Cell.isBlank":
		value, err := ahdruntime.ExcelCellIsBlank(cell())
		if err != nil {
			session.raise("ExcelError", err.Error())
		}
		return value
	case "Cell.string":
		value, err := ahdruntime.ExcelCellString(cell())
		return session.excelResult(value, err)
	case "Cell.int":
		value, err := ahdruntime.ExcelCellInt(cell())
		if err != nil {
			session.raise("ExcelError", err.Error())
		}
		return value
	case "Cell.real":
		value, err := ahdruntime.ExcelCellReal(cell())
		if err != nil {
			session.raise("ExcelError", err.Error())
		}
		return value
	case "Cell.bool":
		value, err := ahdruntime.ExcelCellBool(cell())
		if err != nil {
			session.raise("ExcelError", err.Error())
		}
		return value
	case "Cell.formula":
		value, err := ahdruntime.ExcelCellFormula(cell())
		return session.excelResult(value, err)

	case "Range.startRow", "Range.startColumn", "Range.endRow", "Range.endColumn", "Range.rowCount", "Range.columnCount":
		value, err := ahdruntime.ExcelRangeInt(cellRange(), name[len("Range."):])
		if err != nil {
			session.raise("ExcelError", err.Error())
		}
		return value
	case "Range.address":
		value, err := ahdruntime.ExcelRangeAddress(cellRange())
		return session.excelResult(value, err)

	case "CellStyle.bold", "CellStyle.italic", "CellStyle.underline", "CellStyle.wrap":
		value, err := ahdruntime.ExcelStyleBool(style(), name[len("CellStyle."):], args[0].(bool))
		return resultValue(evaluatorExcelStyleClass, value, err)
	case "CellStyle.fontSize":
		value, err := ahdruntime.ExcelStyleFontSize(style(), args[0].(float64))
		return resultValue(evaluatorExcelStyleClass, value, err)
	case "CellStyle.textColor", "CellStyle.fillColor", "CellStyle.horizontal", "CellStyle.vertical", "CellStyle.numberFormat":
		value, err := ahdruntime.ExcelStyleString(style(), name[len("CellStyle."):], args[0].(string))
		return resultValue(evaluatorExcelStyleClass, value, err)
	case "CellStyle.border":
		value, err := ahdruntime.ExcelStyleBorder(style(), args[0].(string), args[1].(string))
		return resultValue(evaluatorExcelStyleClass, value, err)
	default:
		session.raise("Error", "unsupported Excel operation "+name)
	}
	return nil
}
