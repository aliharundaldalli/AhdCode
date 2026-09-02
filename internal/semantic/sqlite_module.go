package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const sqliteModuleID = "builtin:SQLite"

var (
	sqliteErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	sqliteErrorClass    = &types.ClassSymbol{ModuleID: sqliteModuleID, Name: "SQLiteError", Parent: sqliteErrorParent}
	sqliteDatabaseClass = &types.ClassSymbol{ModuleID: sqliteModuleID, Name: "Database"}
	sqliteValueClass    = &types.ClassSymbol{ModuleID: sqliteModuleID, Name: "SQLiteValue"}
)

// SQLiteErrorIdentity, SQLiteDatabaseIdentity, and SQLiteValueIdentity expose
// the canonical identities to the lowering layer without coupling the public
// module interface to a backend.
func SQLiteErrorIdentity() *types.ClassSymbol    { return sqliteErrorClass }
func SQLiteDatabaseIdentity() *types.ClassSymbol { return sqliteDatabaseClass }
func SQLiteValueIdentity() *types.ClassSymbol    { return sqliteValueClass }

// SQLiteDatabaseOperations and SQLiteValueOperations name the members each
// Class publishes through built-in type operations, so has/has not reports the
// real surface and the lowering layer's IR Class agrees with the frontend.
var SQLiteDatabaseOperations = []string{"execute", "query", "lastInsertId", "begin", "commit", "rollback", "close"}
var SQLiteValueOperations = []string{"kind", "isNull", "int", "real", "string"}

func sqliteDatabaseType() types.Type { return types.Class{Symbol: sqliteDatabaseClass} }
func sqliteValueType() types.Type    { return types.Class{Symbol: sqliteValueClass} }

// sqliteRowType is one query row: the result columns in result order, each
// holding exactly one SQLiteValue (SQL NULL is a SQLiteValue of kind Null,
// never an AhdCode null).
func sqliteRowType() types.Type { return types.Pair{Key: types.String, Value: sqliteValueType()} }

func sqliteModuleInterface() *ModuleInterface {
	module := standardInterface(sqliteModuleID, "SQLite")
	classes := []struct {
		name     string
		identity *types.ClassSymbol
	}{
		{"SQLiteError", sqliteErrorClass}, {"Database", sqliteDatabaseClass}, {"SQLiteValue", sqliteValueClass},
	}
	for _, entry := range classes {
		symbol := &Symbol{
			Name: entry.name, Kind: ClassSymbol, Class: entry.identity,
			Type: types.Class{Symbol: entry.identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: sqliteModuleID,
			Members: make(map[string]*Symbol),
		}
		if entry.name == "SQLiteError" {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[sqliteModuleID+"\x00"+entry.name] = symbol
		addStandardExport(module, symbol)
	}
	value := sqliteValueType()
	addStandardExport(module, standardFunction(sqliteModuleID, "open", sqliteDatabaseType(), types.Parameter{Name: "path", Type: types.String}))
	// nullValue, not null: `null` is a reserved keyword and cannot appear as a
	// member name after `.`, the same reason JSON publishes JSON.nullValue().
	addStandardExport(module, standardFunction(sqliteModuleID, "nullValue", value))
	addStandardExport(module, standardFunction(sqliteModuleID, "fromInt", value, types.Parameter{Name: "value", Type: types.Int}))
	addStandardExport(module, standardFunction(sqliteModuleID, "fromReal", value, types.Parameter{Name: "value", Type: types.Real}))
	addStandardExport(module, standardFunction(sqliteModuleID, "fromString", value, types.Parameter{Name: "value", Type: types.String}))
	sort.Strings(module.ExportNames)
	return module
}

// sqliteConstructionHint names the SQLite functions that produce each value,
// so direct construction has an actionable message.
func sqliteConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != sqliteModuleID {
		return "", false
	}
	switch identity.Name {
	case "Database":
		return "open a Database with SQLite.open(path)", true
	case "SQLiteValue":
		return "create a SQLiteValue with SQLite.nullValue(), SQLite.fromInt(value), SQLite.fromReal(value), or SQLite.fromString(value)", true
	}
	return "", false
}

// sqliteOperationShape is the fixed call shape of one Database or SQLiteValue
// member. optional counts how many trailing parameters may be omitted: the
// `parameters: List<SQLiteValue> = []` argument of execute and query.
type sqliteOperationShape struct {
	parameters []types.Type
	optional   int
	result     types.Type
	hint       string
}

func sqliteOperationShapes() map[TypeOperation]sqliteOperationShape {
	none := []types.Type{}
	parameters := types.List{Element: sqliteValueType()}
	statement := []types.Type{types.String, parameters}
	return map[TypeOperation]sqliteOperationShape{
		SQLiteDatabaseExecute:      {statement, 1, types.Int, "pass one SQL String and optionally a List<SQLiteValue> of ? parameters"},
		SQLiteDatabaseQuery:        {statement, 1, types.List{Element: sqliteRowType()}, "pass one SQL String and optionally a List<SQLiteValue> of ? parameters"},
		SQLiteDatabaseLastInsertID: {none, 0, types.Int, "call lastInsertId with no argument"},
		SQLiteDatabaseBegin:        {none, 0, types.Nothing, "call begin with no argument"},
		SQLiteDatabaseCommit:       {none, 0, types.Nothing, "call commit with no argument"},
		SQLiteDatabaseRollback:     {none, 0, types.Nothing, "call rollback with no argument"},
		SQLiteDatabaseClose:        {none, 0, types.Nothing, "call close with no argument"},

		SQLiteValueKind:   {none, 0, types.String, "call kind with no argument"},
		SQLiteValueIsNull: {none, 0, types.Bool, "call isNull with no argument"},
		SQLiteValueInt:    {none, 0, types.Int, "call int on an Int SQLiteValue"},
		SQLiteValueReal:   {none, 0, types.Real, "call real on a Real or Int SQLiteValue"},
		SQLiteValueString: {none, 0, types.String, "call string on a String SQLiteValue"},
	}
}

var sqliteOperationNames = map[string]map[string]TypeOperation{
	"Database": {"execute": SQLiteDatabaseExecute, "query": SQLiteDatabaseQuery, "lastInsertId": SQLiteDatabaseLastInsertID,
		"begin": SQLiteDatabaseBegin, "commit": SQLiteDatabaseCommit, "rollback": SQLiteDatabaseRollback, "close": SQLiteDatabaseClose},
	"SQLiteValue": {"kind": SQLiteValueKind, "isNull": SQLiteValueIsNull, "int": SQLiteValueInt,
		"real": SQLiteValueReal, "string": SQLiteValueString},
}

// sqliteOperationFor names the built-in member a Database or SQLiteValue
// instance publishes. Only the compiler-supplied identities match, so a user
// Class named Database never collides with them.
func sqliteOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != sqliteModuleID {
		return "", false
	}
	operation, known := sqliteOperationNames[class.Symbol.Name][name]
	return operation, known
}

// analyzeSQLiteOperation checks one Database or SQLiteValue member. Every
// argument is a NonNull value of the declared type, and the trailing
// parameter List may be omitted where the shape allows it. The exact
// per-call signature is recorded so lowering gives each argument its own
// expected type, the same way analyzeExcelOperation does.
func (a *analyzer) analyzeSQLiteOperation(call *ast.CallExpr, operation TypeOperation, shape sqliteOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	minimum := len(shape.parameters) - shape.optional
	if len(call.Arguments) < minimum || len(call.Arguments) > len(shape.parameters) {
		expected := fmt.Sprintf("%d", len(shape.parameters))
		if shape.optional > 0 {
			expected = fmt.Sprintf("%d to %d", minimum, len(shape.parameters))
		}
		a.error(codeCallArguments, fmt.Sprintf("%s expects %s argument(s); received %d", operation, expected, len(call.Arguments)), call.Span(), shape.hint)
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
	parameters := make([]types.Parameter, len(call.Arguments))
	for index := range call.Arguments {
		parameters[index] = types.Parameter{Type: shape.parameters[index]}
	}
	a.result.SelectedCallables[call] = &Callable{
		Signature:  &types.Signature{Parameters: parameters, Return: shape.result},
		ReturnNull: NonNull,
	}
	return result
}
