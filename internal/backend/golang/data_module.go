package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const dataModulePrefix = "builtin:Data::"

var (
	dataTableClass        = ir.ClassID("builtin:Data::class::Table")
	dataTableColumnsField = ir.FieldID("builtin:Data::class::Table::field::columns")
	dataTableCellsField   = ir.FieldID("builtin:Data::class::Table::field::cells")
)

// The runtime Class catalog names a Data helper raises through. They are used
// instead of generated descriptors because Data can be imported without CSV,
// and a descriptor exists only for a Class the program itself declares. Each
// name denotes the same runtime Class object its descriptor would, so
// attempt/except matching is unaffected.
const (
	dataErrorRuntime = "AhdClassDataError"
	csvErrorRuntime  = "AhdClassCSVError"
	fileErrorRuntime = "AhdClassFileError"
)

// dataRowType is the row shape every Data callback receives: Pair<String,
// String>, matching the frontend's published contract.
func dataRowType() ir.Type {
	key := ir.Type{Kind: ir.StringType}
	value := ir.Type{Kind: ir.StringType}
	return ir.Type{Kind: ir.PairType, Key: &key, Value: &value}
}

// dataCall lowers the Data module's factory functions. The frontend has
// already checked every signature, so this layer only maps stable builtin
// identities onto runtime helpers and the generated Table constructor.
func (generator *generator) dataCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), dataModulePrefix)
	argument := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].UsesDefault ||
			value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.expr(value.Arguments[index].Value)
	}
	dataError, csvError := dataErrorRuntime, csvErrorRuntime
	switch name {
	case "fromRows":
		return generator.tableFrom("AhdDataFromRows("+dataError+", "+argument(0, "nil")+", "+argument(1, "nil")+")", meta)
	case "fromRecords":
		return generator.tableFrom("AhdDataFromRecords("+dataError+", "+argument(0, "nil")+")", meta)
	case "fromCSV":
		return generator.tableFrom("AhdDataFromCSV("+dataError+", "+csvError+", "+
			argument(0, `""`)+", "+argument(1, `","`)+")", meta)
	case "readCSV":
		return generator.tableFrom("AhdDataReadCSV("+dataError+", "+csvError+", "+fileErrorRuntime+", "+
			argument(0, `""`)+", "+argument(1, `","`)+")", meta)
	default:
		return generator.unsupported("Data function "+name, meta.Span)
	}
}

// tableFrom builds a Table instance from one runtime AhdTable reading, the
// same way a DateTime is built from one civil-time reading.
func (generator *generator) tableFrom(table string, meta ir.ExprBase) string {
	helper, ok := generator.dataHelper()
	if !ok {
		return generator.unsupported("a Table value without its Class declaration", meta.Span)
	}
	return helper + "(" + table + ")"
}

// tableOf evaluates one Table expression exactly once and reads its two hidden
// storage fields into the runtime interchange shape.
func (generator *generator) tableOf(expression ir.Expr) string {
	rendered := generator.expr(expression)
	columns := "value." + generator.fieldName(dataTableColumnsField) + "_get()"
	cells := "value." + generator.fieldName(dataTableCellsField) + "_get()"
	return "func(value " + generator.interfaceName(dataTableClass) + ") AhdTable { " +
		"return AhdTable{Columns: " + columns + ", Cells: " + cells + "} }(" + rendered + ")"
}

// dataHelper registers the generated constructor wrapper for Table.
func (generator *generator) dataHelper() (string, bool) {
	if generator.layouts[dataTableClass] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[dataTableClass]; known {
		return name, true
	}
	name := mangleNamed("dh_", generator.classDisplayName(dataTableClass), string(dataTableClass))
	generator.timeHelpers[dataTableClass] = name
	return name, true
}

// emitDataHelpers writes the Table wrapper, turning a runtime reading into a
// constructed AhdCode value.
func (generator *generator) emitDataHelpers(writer *emitter) {
	name, known := generator.timeHelpers[dataTableClass]
	if !known {
		return
	}
	layout := generator.layouts[dataTableClass]
	if layout == nil {
		return
	}
	constructor := generator.functions[layout.class.Constructor]
	if constructor == nil {
		return
	}
	writer.line("// Table value built from one runtime table reading.")
	writer.open("func " + name + "(table AhdTable) " + generator.interfaceName(dataTableClass) + " {")
	writer.line("return " + generator.callableName(constructor) + "(table.Columns, table.Cells)")
	writer.close("}")
	writer.blank()
}

// dataOperation lowers the built-in members of Table. Every member reaches
// this through the ordinary type-operation path, so Data adds no static-method
// or operator semantics to the language.
func (generator *generator) dataOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	receiver := generator.tableOf(value.Callee)
	dataError, csvError := dataErrorRuntime, csvErrorRuntime
	text := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	integer := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	list := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			generator.fail(CodeGenerationFailure, name+" has a missing argument", meta.Span, "the IR call is malformed")
			return "nil"
		}
		return generator.expr(value.Arguments[index].Value)
	}
	switch name {
	case "Table.rowCount":
		return "AhdDataRowCount(" + receiver + ")"
	case "Table.columnCount":
		return "AhdDataColumnCount(" + receiver + ")"
	case "Table.columns":
		return "AhdDataColumns(" + receiver + ")"
	case "Table.rows":
		return "AhdDataRows(" + receiver + ")"
	case "Table.row":
		return "AhdDataRow(" + receiver + ", " + integer(0, "int64(0)") + ")"
	case "Table.column":
		return "AhdDataColumn(" + dataError + ", " + receiver + ", " + text(0, `""`) + ")"
	case "Table.head":
		return generator.tableFrom("AhdDataHead("+dataError+", "+receiver+", "+integer(0, "int64(5)")+")", meta)
	case "Table.tail":
		return generator.tableFrom("AhdDataTail("+dataError+", "+receiver+", "+integer(0, "int64(5)")+")", meta)
	case "Table.select":
		return generator.tableFrom("AhdDataSelect("+dataError+", "+receiver+", "+list(0)+")", meta)
	case "Table.drop":
		return generator.tableFrom("AhdDataDrop("+dataError+", "+receiver+", "+list(0)+")", meta)
	case "Table.rename":
		return generator.tableFrom("AhdDataRename("+dataError+", "+receiver+", "+text(0, `""`)+", "+text(1, `""`)+")", meta)
	case "Table.reverse":
		return generator.tableFrom("AhdDataReverse("+receiver+")", meta)
	case "Table.filter":
		adapter := generator.dataCallback(value, 0, dataRowType(), ir.Type{Kind: ir.BoolType}, meta)
		return generator.tableFrom("AhdDataFilter("+receiver+", "+adapter+")", meta)
	case "Table.sort":
		return generator.dataSort(value, receiver, dataError, meta)
	case "Table.transform":
		adapter := generator.dataCallback(value, 1, ir.Type{Kind: ir.StringType}, ir.Type{Kind: ir.StringType}, meta)
		return generator.tableFrom("AhdDataTransform("+dataError+", "+receiver+", "+text(0, `""`)+", "+adapter+")", meta)
	case "Table.derive":
		adapter := generator.dataCallback(value, 1, dataRowType(), ir.Type{Kind: ir.StringType}, meta)
		return generator.tableFrom("AhdDataDerive("+dataError+", "+receiver+", "+text(0, `""`)+", "+adapter+")", meta)
	case "Table.unique":
		return "AhdDataUnique(" + dataError + ", " + receiver + ", " + text(0, `""`) + ")"
	case "Table.valueCounts":
		return "AhdDataValueCounts(" + dataError + ", " + receiver + ", " + text(0, `""`) + ")"
	case "Table.groupBy":
		return generator.dataGroupBy(receiver, dataError, text(0, `""`), meta)
	case "Table.pivotCount":
		return generator.tableFrom("AhdDataPivotCount("+dataError+", "+receiver+", "+
			text(0, `""`)+", "+text(1, `""`)+")", meta)
	case "Table.toCSV":
		return "AhdDataToCSV(" + csvError + ", " + receiver + ", " + text(0, `","`) + ")"
	case "Table.writeCSV":
		return "AhdDataWriteCSV(" + csvError + ", " + fileErrorRuntime + ", " + receiver + ", " +
			text(0, `""`) + ", " + text(1, `","`) + ")"
	default:
		return generator.unsupported("Table operation "+name, meta.Span)
	}
}

// dataCallback adapts one Function argument to the fixed runtime signature.
// The result stays boxed, matching the representation a Function value's
// return already uses, exactly as List.filter and List.sort do.
func (generator *generator) dataCallback(value *ir.CallExpr, index int, parameter, result ir.Type, meta ir.ExprBase) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		generator.fail(CodeGenerationFailure, "a Table callback is missing", meta.Span, "the IR call is malformed")
		return "nil"
	}
	return generator.adaptElementCallback(value.Arguments[index].Value, parameter, false, result, true)
}

// dataSort selects the column or keyed form. The argument's static type decides
// which, so no ordering is ever inferred from rendered text.
func (generator *generator) dataSort(value *ir.CallExpr, receiver, dataError string, meta ir.ExprBase) string {
	if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
		generator.fail(CodeGenerationFailure, "Table.sort has no argument", meta.Span, "the IR call is malformed")
		return "nil"
	}
	argument := value.Arguments[0].Value
	signature := argument.ExprMeta().Type.Signature
	if argument.ExprMeta().Type.Kind != ir.FunctionType || signature == nil {
		column := generator.value(argument, ir.Type{Kind: ir.StringType}, false)
		return generator.tableFrom("AhdDataSortColumn("+dataError+", "+receiver+", "+column+")", meta)
	}
	helper, known := orderedHelper("AhdDataSortKey", signature.Return)
	if !known {
		return generator.unsupported("sort by a "+signature.Return.String()+" key", meta.Span)
	}
	adapter := generator.adaptElementCallback(argument, dataRowType(), false, signature.Return, true)
	return generator.tableFrom(helper+"("+receiver+", "+adapter+")", meta)
}

// dataGroupBy builds Pair<String, Table>. The runtime supplies the key order
// and each group's storage; the generated code wraps every group in a Table,
// because only generated code knows the Table Class type.
func (generator *generator) dataGroupBy(receiver, dataError, column string, meta ir.ExprBase) string {
	helper, ok := generator.dataHelper()
	if !ok {
		return generator.unsupported("a Table value without its Class declaration", meta.Span)
	}
	result := generator.interfaceName(dataTableClass)
	source := generator.temporaryName()
	name := generator.temporaryName()
	groups := generator.temporaryName()
	key := generator.temporaryName()
	return "func(" + source + " AhdTable, " + name + " string) *AhdPair[string, " + result + "] { " +
		groups + " := AhdNewPair[string, " + result + "](); " +
		"for _, " + key + " := range AhdDataGroupKeys(" + dataError + ", " + source + ", " + name + ").Snapshot() { " +
		groups + ".Set(" + key + ", " + helper + "(AhdDataGroupTable(" + dataError + ", " + source + ", " + name + ", " + key + "))) }; " +
		"return " + groups + " }(" + receiver + ", " + column + ")"
}
