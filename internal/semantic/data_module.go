package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const dataModuleID = "builtin:Data"

var (
	dataErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	dataErrorClass = &types.ClassSymbol{ModuleID: dataModuleID, Name: "DataError", Parent: dataErrorParent}
	dataTableClass = &types.ClassSymbol{ModuleID: dataModuleID, Name: "Table"}
)

// DataErrorIdentity and DataTableIdentity expose the canonical identities to
// the lowering layer without coupling the public module interface to a backend.
func DataErrorIdentity() *types.ClassSymbol { return dataErrorClass }
func DataTableIdentity() *types.ClassSymbol { return dataTableClass }

// DataTableOperations names the members a Table publishes through built-in type
// operations, so has/has not reports what a Table value really offers. The
// lowering layer reuses this list, keeping the IR Class and the frontend in
// agreement about the published surface.
var DataTableOperations = []string{
	"rowCount", "columnCount", "columns", "rows", "row", "column",
	"head", "tail", "select", "drop", "rename", "reverse",
	"filter", "sort", "transform", "derive",
	"unique", "valueCounts", "groupBy", "pivotCount", "toCSV", "writeCSV",
}

// dataRow is the canonical row shape AhdCode source sees. Every cell is a
// String: Data is a table structure layer, not a typed-value system, so a user
// converts explicitly with int(...) / real(...) when they want a number.
func dataRow() types.Type { return types.Pair{Key: types.String, Value: types.String} }

func dataTableType() types.Type { return types.Class{Symbol: dataTableClass} }

func dataModuleInterface() *ModuleInterface {
	module := standardInterface(dataModuleID, "Data")

	errorSymbol := &Symbol{
		Name: "DataError", Kind: ClassSymbol, Class: dataErrorClass,
		Type: types.Class{Symbol: dataErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: dataModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[dataModuleID+"\x00DataError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	tableSymbol := &Symbol{
		Name: "Table", Kind: ClassSymbol, Class: dataTableClass,
		Type: types.Class{Symbol: dataTableClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: dataModuleID,
		Members: make(map[string]*Symbol),
	}
	module.Classes[dataModuleID+"\x00Table"] = tableSymbol
	addStandardExport(module, tableSymbol)

	table := dataTableType()
	rows := types.List{Element: types.List{Element: types.String}}
	records := types.List{Element: dataRow()}
	delimiter := types.Parameter{Name: "delimiter", Type: types.String, HasDefault: true}

	addStandardExport(module, standardFunction(dataModuleID, "fromRows", table,
		types.Parameter{Name: "columns", Type: types.List{Element: types.String}},
		types.Parameter{Name: "rows", Type: rows}))
	addStandardExport(module, standardFunction(dataModuleID, "fromRecords", table,
		types.Parameter{Name: "records", Type: records}))
	addStandardExport(module, standardFunction(dataModuleID, "fromCSV", table,
		types.Parameter{Name: "text", Type: types.String}, delimiter))
	addStandardExport(module, standardFunction(dataModuleID, "readCSV", table,
		types.Parameter{Name: "path", Type: types.String}, delimiter))

	sort.Strings(module.ExportNames)
	return module
}

// dataConstructionHint names the Data functions that produce a Table, so
// direct construction has an actionable message instead of a generic
// missing-constructor diagnostic.
func dataConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != dataModuleID || identity.Name != "Table" {
		return "", false
	}
	return "create a Table with Data.fromRows, Data.fromRecords, Data.fromCSV, or Data.readCSV, " +
		"or derive one from an existing Table", true
}

// dataOperationShape is the call shape of one Table member that takes only
// ordinary values. minimum and maximum bound the argument count, so an
// operation with a trailing default (head, tail, toCSV, writeCSV) is described
// by the same table as a fixed-arity one.
type dataOperationShape struct {
	parameters []types.Type
	minimum    int
	result     types.Type
	hint       string
}

// dataOperationShapes describes every Table member whose arguments are plain
// values. The callback-taking members (filter, sort, transform, derive) are
// checked separately, because their Function argument needs the existing
// callback machinery rather than a plain assignability test.
func dataOperationShapes() map[TypeOperation]dataOperationShape {
	none := []types.Type{}
	name := []types.Type{types.String}
	columnList := []types.Type{types.List{Element: types.String}}
	strings := types.List{Element: types.String}
	return map[TypeOperation]dataOperationShape{
		DataRowCount:    {none, 0, types.Int, "call rowCount with no argument"},
		DataColumnCount: {none, 0, types.Int, "call columnCount with no argument"},
		DataColumns:     {none, 0, strings, "call columns with no argument"},
		DataRows:        {none, 0, types.List{Element: dataRow()}, "call rows with no argument"},
		DataRow:         {[]types.Type{types.Int}, 1, dataRow(), "pass one Int row index"},
		DataColumn:      {name, 1, strings, "pass one existing column name"},
		DataHead:        {[]types.Type{types.Int}, 0, dataTableType(), "pass a non-negative Int row count, or no argument for 5"},
		DataTail:        {[]types.Type{types.Int}, 0, dataTableType(), "pass a non-negative Int row count, or no argument for 5"},
		DataSelect:      {columnList, 1, dataTableType(), "pass a List<String> of existing column names"},
		DataDrop:        {columnList, 1, dataTableType(), "pass a List<String> of existing column names"},
		DataRename:      {[]types.Type{types.String, types.String}, 2, dataTableType(), "pass the existing column name and its new name"},
		DataReverse:     {none, 0, dataTableType(), "call reverse with no argument"},
		DataUnique:      {name, 1, strings, "pass one existing column name"},
		DataValueCounts: {name, 1, types.Pair{Key: types.String, Value: types.Int}, "pass one existing column name"},
		DataGroupBy:     {name, 1, types.Pair{Key: types.String, Value: dataTableType()}, "pass one existing column name"},
		DataPivotCount:  {[]types.Type{types.String, types.String}, 2, dataTableType(), "pass the row column name and the column column name"},
		DataToCSV:       {name, 0, types.String, "pass a single-character String delimiter, or no argument for \",\""},
		DataWriteCSV:    {[]types.Type{types.String, types.String}, 1, types.Nothing, "pass the destination path, and optionally a single-character String delimiter"},
	}
}

var dataOperationNames = map[string]TypeOperation{
	"rowCount": DataRowCount, "columnCount": DataColumnCount, "columns": DataColumns,
	"rows": DataRows, "row": DataRow, "column": DataColumn,
	"head": DataHead, "tail": DataTail, "select": DataSelect, "drop": DataDrop,
	"rename": DataRename, "reverse": DataReverse, "filter": DataFilter, "sort": DataSort,
	"transform": DataTransform, "derive": DataDerive, "unique": DataUnique,
	"valueCounts": DataValueCounts, "groupBy": DataGroupBy, "pivotCount": DataPivotCount,
	"toCSV": DataToCSV, "writeCSV": DataWriteCSV,
}

// dataOperationFor names the built-in member a Table instance publishes. Only
// the compiler-supplied Table identity matches, so a user Class named Table
// never collides with it.
func dataOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil ||
		class.Symbol.ModuleID != dataModuleID || class.Symbol.Name != "Table" {
		return "", false
	}
	operation, known := dataOperationNames[name]
	return operation, known
}

// dataCallbackOperation reports whether one Table member takes a Function, so
// the shape table can skip it and the callback checker can own it.
func dataCallbackOperation(operation TypeOperation) bool {
	switch operation {
	case DataFilter, DataSort, DataTransform, DataDerive:
		return true
	default:
		return false
	}
}

// dataOperationResult is the statically known result of one Table member, used
// both by a successful check and by a rejection so a failure does not cascade.
func dataOperationResult(operation TypeOperation) (types.Type, bool) {
	if shape, known := dataOperationShapes()[operation]; known {
		return shape.result, true
	}
	switch operation {
	case DataFilter, DataSort, DataTransform, DataDerive:
		return dataTableType(), true
	default:
		return nil, false
	}
}

// analyzeDataOperation checks one Table member. Arguments are NonNull values of
// the declared type; a trailing defaulted argument may be omitted.
func (a *analyzer) analyzeDataOperation(call *ast.CallExpr, operation TypeOperation, shape dataOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	if len(call.Arguments) < shape.minimum || len(call.Arguments) > len(shape.parameters) {
		a.error(codeCallArguments, dataArityMessage(operation, shape, len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for index, argument := range call.Arguments {
		expected := shape.parameters[index]
		info := a.analyzeExpressionExpected(argument.Value, current, flow, expected)
		if info.invalid() {
			continue
		}
		if info.nullState != NonNull {
			a.nullableError(string(operation), argument.Value, info.nullState)
			continue
		}
		if !types.Assignable(expected, info.typeValue) {
			a.typeMismatch(argument.Span(), expected, info.typeValue, string(operation)+" argument")
		}
	}
	return result
}

func dataArityMessage(operation TypeOperation, shape dataOperationShape, received int) string {
	if shape.minimum == len(shape.parameters) {
		return fmt.Sprintf("%s expects %d argument(s); received %d", operation, shape.minimum, received)
	}
	return fmt.Sprintf("%s expects %d to %d argument(s); received %d",
		operation, shape.minimum, len(shape.parameters), received)
}

// analyzeDataCallbackOperation checks the four Table members that take a
// Function. Each contract is fixed and known to the compiler, so a wrong
// callback is a compile-time diagnostic rather than a runtime surprise.
func (a *analyzer) analyzeDataCallbackOperation(call *ast.CallExpr, operation TypeOperation, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: dataTableType(), nullState: NonNull}
	switch operation {
	case DataFilter:
		if !a.requireTypeOperationArity(call, operation, dataTableType(), 1, current, flow) {
			return result
		}
		a.analyzeDataCallback(call, 0, operation, dataRow(), types.Bool, current, flow)
	case DataTransform:
		if !a.requireTypeOperationArity(call, operation, dataTableType(), 2, current, flow) {
			return result
		}
		a.analyzeDataName(call, 0, operation, current, flow)
		a.analyzeDataCallback(call, 1, operation, types.String, types.String, current, flow)
	case DataDerive:
		if !a.requireTypeOperationArity(call, operation, dataTableType(), 2, current, flow) {
			return result
		}
		a.analyzeDataName(call, 0, operation, current, flow)
		a.analyzeDataCallback(call, 1, operation, dataRow(), types.String, current, flow)
	case DataSort:
		return a.analyzeDataSort(call, current, flow)
	}
	return result
}

// analyzeDataSort checks both sort forms. A String argument names a column and
// sorts lexically; a Function argument is an Int, Real, or String key, matching
// the existing List keyed-sort contract. The form is chosen by the argument's
// static type, so no ordering is ever inferred from source text.
func (a *analyzer) analyzeDataSort(call *ast.CallExpr, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: dataTableType(), nullState: NonNull}
	if !a.requireTypeOperationArity(call, DataSort, dataTableType(), 1, current, flow) {
		return result
	}
	reported := a.bag.Len()
	info := a.analyzeExpression(call.Arguments[0].Value, current, flow)
	if info.invalid() || a.bag.Len() != reported {
		return result
	}
	if info.nullState != NonNull {
		a.nullableError(string(DataSort), call.Arguments[0].Value, info.nullState)
		return result
	}
	if function, isFunction := info.typeValue.(types.Function); isFunction {
		if function.Signature == nil {
			return result
		}
		signature := function.Signature
		if len(signature.Parameters) != 1 || !types.Equal(signature.Parameters[0].Type, dataRow()) {
			a.error(codeCallArguments,
				fmt.Sprintf("%s requires a Function taking exactly one %s", DataSort, types.Display(dataRow())),
				call.Arguments[0].Span(), dataCallbackHint(DataSort))
			return result
		}
		if !sortableKey(signature.Return) {
			a.error(codeCallArguments, fmt.Sprintf("sort key Function returns %s", types.Display(signature.Return)),
				call.Arguments[0].Span(), "return Int, Real, or String from the key Function")
		}
		if a.callbackReturnsNull(call.Arguments[0].Value, info) {
			a.error(codeNullableUse, fmt.Sprintf("Function argument for %s may return null", DataSort),
				call.Arguments[0].Span(), "return a NonNull key from the callback")
		}
		return result
	}
	if !types.Assignable(types.String, info.typeValue) {
		a.typeMismatch(call.Arguments[0].Span(), types.String, info.typeValue, string(DataSort)+" argument")
	}
	return result
}

// analyzeDataName checks a leading String column/name argument of a callback
// operation.
func (a *analyzer) analyzeDataName(call *ast.CallExpr, index int, operation TypeOperation, current *scope, flow flowState) {
	info := a.analyzeExpressionExpected(call.Arguments[index].Value, current, flow, types.String)
	if info.invalid() {
		return
	}
	if info.nullState != NonNull {
		a.nullableError(string(operation), call.Arguments[index].Value, info.nullState)
		return
	}
	if !types.Assignable(types.String, info.typeValue) {
		a.typeMismatch(call.Arguments[index].Span(), types.String, info.typeValue, string(operation)+" column name")
	}
}

// analyzeDataCallback checks one Function argument against a fixed contract.
// It mirrors analyzeListCallback: the parameter type must match exactly,
// because no value is converted on the way into the call.
func (a *analyzer) analyzeDataCallback(call *ast.CallExpr, index int, operation TypeOperation, parameter, expectedReturn types.Type, current *scope, flow flowState) {
	expected := types.Function{Signature: &types.Signature{
		Parameters: []types.Parameter{{Name: "value", Type: parameter}}, Return: expectedReturn,
	}}
	reported := a.bag.Len()
	info := a.analyzeExpressionExpected(call.Arguments[index].Value, current, flow, expected)
	if info.invalid() || a.bag.Len() != reported {
		return
	}
	if info.nullState != NonNull {
		a.nullableError(string(operation), call.Arguments[index].Value, info.nullState)
		return
	}
	function, isFunction := info.typeValue.(types.Function)
	if !isFunction || function.Signature == nil {
		a.typeMismatch(call.Arguments[index].Span(), expected, info.typeValue, string(operation)+" argument")
		return
	}
	signature := function.Signature
	if len(signature.Parameters) != 1 || !types.Equal(signature.Parameters[0].Type, parameter) {
		a.error(codeCallArguments,
			fmt.Sprintf("%s requires a Function taking exactly one %s", operation, types.Display(parameter)),
			call.Arguments[index].Span(), dataCallbackHint(operation))
		return
	}
	if !types.Equal(signature.Return, expectedReturn) {
		a.typeMismatch(call.Arguments[index].Span(), expectedReturn, signature.Return, string(operation)+" callback result")
		return
	}
	if a.callbackReturnsNull(call.Arguments[index].Value, info) {
		a.error(codeNullableUse, fmt.Sprintf("Function argument for %s may return null", operation),
			call.Arguments[index].Span(), "return a NonNull value from the callback")
	}
}

// dataCallbackHint states the exact contract a Table callback must satisfy.
func dataCallbackHint(operation TypeOperation) string {
	switch operation {
	case DataFilter:
		return "filter expects a Function compatible with (Pair<String, String>) -> Bool"
	case DataSort:
		return "sort expects a column name String, or a Function compatible with (Pair<String, String>) -> Int|Real|String"
	case DataTransform:
		return "transform expects a Function compatible with (String) -> String"
	case DataDerive:
		return "derive expects a Function compatible with (Pair<String, String>) -> String"
	default:
		return ""
	}
}
