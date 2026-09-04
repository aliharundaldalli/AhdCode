package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const mysqlModuleID = "builtin:MySQL"

var (
	mysqlErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	mysqlErrorClass       = &types.ClassSymbol{ModuleID: mysqlModuleID, Name: "MySQLError", Parent: mysqlErrorParent}
	mysqlDatabaseClass    = &types.ClassSymbol{ModuleID: mysqlModuleID, Name: "MySQLDatabase"}
	mysqlTransactionClass = &types.ClassSymbol{ModuleID: mysqlModuleID, Name: "MySQLTransaction"}
	mysqlResultClass      = &types.ClassSymbol{ModuleID: mysqlModuleID, Name: "MySQLResult"}
	mysqlValueClass       = &types.ClassSymbol{ModuleID: mysqlModuleID, Name: "MySQLValue"}
)

// MySQLErrorIdentity, MySQLDatabaseIdentity, MySQLTransactionIdentity,
// MySQLResultIdentity, and MySQLValueIdentity expose the canonical identities
// to the lowering layer without coupling the public module interface to a
// backend.
func MySQLErrorIdentity() *types.ClassSymbol       { return mysqlErrorClass }
func MySQLDatabaseIdentity() *types.ClassSymbol    { return mysqlDatabaseClass }
func MySQLTransactionIdentity() *types.ClassSymbol { return mysqlTransactionClass }
func MySQLResultIdentity() *types.ClassSymbol      { return mysqlResultClass }
func MySQLValueIdentity() *types.ClassSymbol       { return mysqlValueClass }

// MySQLDatabaseOperations, MySQLTransactionOperations, MySQLResultOperations,
// and MySQLValueOperations name the members each Class publishes through
// built-in type operations, so has/has not reports the real surface and the
// lowering layer's IR Class agrees with the frontend.
var MySQLDatabaseOperations = []string{"ping", "execute", "query", "begin", "close"}
var MySQLTransactionOperations = []string{"execute", "query", "commit", "rollback"}
var MySQLResultOperations = []string{"affectedRows", "lastInsertId"}
var MySQLValueOperations = []string{"kind", "isNull", "int", "real", "string", "isBinary", "binarySize", "binaryBase64"}

func mysqlDatabaseType() types.Type    { return types.Class{Symbol: mysqlDatabaseClass} }
func mysqlTransactionType() types.Type { return types.Class{Symbol: mysqlTransactionClass} }
func mysqlResultType() types.Type      { return types.Class{Symbol: mysqlResultClass} }
func mysqlValueType() types.Type       { return types.Class{Symbol: mysqlValueClass} }

// mysqlRowType is one query row: the result columns in result order, each
// holding exactly one MySQLValue (SQL NULL is a MySQLValue of kind Null,
// never an AhdCode null) -- the same row shape SQLite publishes.
func mysqlRowType() types.Type { return types.Pair{Key: types.String, Value: mysqlValueType()} }

func mysqlModuleInterface() *ModuleInterface {
	module := standardInterface(mysqlModuleID, "MySQL")
	classes := []struct {
		name     string
		identity *types.ClassSymbol
	}{
		{"MySQLError", mysqlErrorClass}, {"MySQLDatabase", mysqlDatabaseClass},
		{"MySQLTransaction", mysqlTransactionClass}, {"MySQLResult", mysqlResultClass},
		{"MySQLValue", mysqlValueClass},
	}
	for _, entry := range classes {
		symbol := &Symbol{
			Name: entry.name, Kind: ClassSymbol, Class: entry.identity,
			Type: types.Class{Symbol: entry.identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: mysqlModuleID,
			Members: make(map[string]*Symbol),
		}
		if entry.name == "MySQLError" {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[mysqlModuleID+"\x00"+entry.name] = symbol
		addStandardExport(module, symbol)
	}
	addStandardExport(module, mysqlConnectFunction())
	value := mysqlValueType()
	// nullValue, not null: `null` is a reserved keyword and cannot appear as
	// a member name after `.`, the same reason SQLite/JSON publish it this way.
	addStandardExport(module, standardFunction(mysqlModuleID, "nullValue", value))
	addStandardExport(module, standardFunction(mysqlModuleID, "fromInt", value, types.Parameter{Name: "value", Type: types.Int}))
	addStandardExport(module, standardFunction(mysqlModuleID, "fromReal", value, types.Parameter{Name: "value", Type: types.Real}))
	addStandardExport(module, standardFunction(mysqlModuleID, "fromString", value, types.Parameter{Name: "value", Type: types.String}))
	sort.Strings(module.ExportNames)
	return module
}

// mysqlConnectFunction builds MySQL.connect by hand rather than through
// standardFunction: it is the one standard-module function whose database
// *parameter* (not its result) is statically nullable, so future
// AhdDataStudio can connect to a server before any database is selected
// (SHOW DATABASES needs no default database). standardFunction always marks
// every parameter NonNull, so this mirrors it with one ParameterNull entry
// changed to MaybeNull instead.
func mysqlConnectFunction() *Symbol {
	parameters := []types.Parameter{
		{Name: "host", Type: types.String},
		{Name: "username", Type: types.String},
		{Name: "password", Type: types.String},
		{Name: "port", Type: types.Int, HasDefault: true},
		{Name: "database", Type: types.String, HasDefault: true},
		{Name: "security", Type: types.String, HasDefault: true},
		{Name: "timeoutSeconds", Type: types.Int, HasDefault: true},
	}
	signature := &types.Signature{Parameters: parameters, Return: mysqlDatabaseType()}
	parameterNull := nonNullParameters(len(parameters))
	parameterNull[4] = MaybeNull // database
	return &Symbol{
		Name: "connect", Kind: FunctionSymbol, Type: types.Function{Signature: signature},
		ModuleRoot: true, Builtin: true, InitialNull: NonNull, OriginModuleID: mysqlModuleID,
		Callable: &Callable{Signature: signature, ParameterNull: parameterNull, ReturnNull: NonNull},
	}
}

// mysqlConstructionHint names the MySQL functions that produce each value, so
// direct construction has an actionable message.
func mysqlConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != mysqlModuleID {
		return "", false
	}
	switch identity.Name {
	case "MySQLDatabase":
		return "connect a MySQLDatabase with MySQL.connect(...)", true
	case "MySQLTransaction":
		return "open a MySQLTransaction with MySQLDatabase.begin()", true
	case "MySQLResult":
		return "a MySQLResult is produced by MySQLDatabase.execute(...) or MySQLTransaction.execute(...)", true
	case "MySQLValue":
		return "create a MySQLValue with MySQL.nullValue(), MySQL.fromInt(value), MySQL.fromReal(value), or MySQL.fromString(value)", true
	}
	return "", false
}

// mysqlOperationShape is the fixed call shape of one MySQLDatabase,
// MySQLTransaction, MySQLResult, or MySQLValue member. optional counts how
// many trailing parameters may be omitted: the
// `params: List<MySQLValue> = []` argument of execute and query. resultNullable
// marks lastInsertId, whose absence of a generated id is a genuine, statically
// expected possibility rather than an error -- the same convention
// JSONValue.get uses.
type mysqlOperationShape struct {
	parameters     []types.Type
	optional       int
	result         types.Type
	resultNullable bool
	hint           string
}

func mysqlOperationShapes() map[TypeOperation]mysqlOperationShape {
	none := []types.Type{}
	parameters := types.List{Element: mysqlValueType()}
	statement := []types.Type{types.String, parameters}
	rows := types.List{Element: mysqlRowType()}
	return map[TypeOperation]mysqlOperationShape{
		MySQLDatabasePing:    {none, 0, types.Nothing, false, "call ping with no argument"},
		MySQLDatabaseExecute: {statement, 1, mysqlResultType(), false, "pass one SQL String and optionally a List<MySQLValue> of ? parameters"},
		MySQLDatabaseQuery:   {statement, 1, rows, false, "pass one SQL String and optionally a List<MySQLValue> of ? parameters"},
		MySQLDatabaseBegin:   {none, 0, mysqlTransactionType(), false, "call begin with no argument"},
		MySQLDatabaseClose:   {none, 0, types.Nothing, false, "call close with no argument"},

		MySQLTransactionExecute:  {statement, 1, mysqlResultType(), false, "pass one SQL String and optionally a List<MySQLValue> of ? parameters"},
		MySQLTransactionQuery:    {statement, 1, rows, false, "pass one SQL String and optionally a List<MySQLValue> of ? parameters"},
		MySQLTransactionCommit:   {none, 0, types.Nothing, false, "call commit with no argument"},
		MySQLTransactionRollback: {none, 0, types.Nothing, false, "call rollback with no argument"},

		MySQLResultAffectedRows: {none, 0, types.Int, false, "call affectedRows with no argument"},
		MySQLResultLastInsertID: {none, 0, types.Int, true, "call lastInsertId with no argument"},

		MySQLValueKind:         {none, 0, types.String, false, "call kind with no argument"},
		MySQLValueIsNull:       {none, 0, types.Bool, false, "call isNull with no argument"},
		MySQLValueInt:          {none, 0, types.Int, false, "call int on an Int MySQLValue"},
		MySQLValueReal:         {none, 0, types.Real, false, "call real on a Real or Int MySQLValue"},
		MySQLValueString:       {none, 0, types.String, false, "call string on a String MySQLValue"},
		MySQLValueIsBinary:     {none, 0, types.Bool, false, "call isBinary with no argument"},
		MySQLValueBinarySize:   {none, 0, types.Int, false, "call binarySize on a Binary MySQLValue"},
		MySQLValueBinaryBase64: {none, 0, types.String, false, "call binaryBase64 on a Binary MySQLValue"},
	}
}

var mysqlOperationNames = map[string]map[string]TypeOperation{
	"MySQLDatabase": {
		"ping": MySQLDatabasePing, "execute": MySQLDatabaseExecute, "query": MySQLDatabaseQuery,
		"begin": MySQLDatabaseBegin, "close": MySQLDatabaseClose,
	},
	"MySQLTransaction": {
		"execute": MySQLTransactionExecute, "query": MySQLTransactionQuery,
		"commit": MySQLTransactionCommit, "rollback": MySQLTransactionRollback,
	},
	"MySQLResult": {
		"affectedRows": MySQLResultAffectedRows, "lastInsertId": MySQLResultLastInsertID,
	},
	"MySQLValue": {
		"kind": MySQLValueKind, "isNull": MySQLValueIsNull, "int": MySQLValueInt,
		"real": MySQLValueReal, "string": MySQLValueString,
		"isBinary": MySQLValueIsBinary, "binarySize": MySQLValueBinarySize, "binaryBase64": MySQLValueBinaryBase64,
	},
}

// mysqlOperationFor names the built-in member a MySQLDatabase,
// MySQLTransaction, MySQLResult, or MySQLValue instance publishes. Only the
// compiler-supplied identities match, so a user Class with one of these
// names never collides with them.
func mysqlOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != mysqlModuleID {
		return "", false
	}
	operation, known := mysqlOperationNames[class.Symbol.Name][name]
	return operation, known
}

// analyzeMySQLOperation checks one MySQLDatabase, MySQLTransaction,
// MySQLResult, or MySQLValue member. Every argument is a NonNull value of the
// declared type, the trailing parameter List may be omitted where the shape
// allows it, and the result is MaybeNull only for lastInsertId.
func (a *analyzer) analyzeMySQLOperation(call *ast.CallExpr, operation TypeOperation, shape mysqlOperationShape, current *scope, flow flowState) expressionInfo {
	nullState := NonNull
	if shape.resultNullable {
		nullState = MaybeNull
	}
	result := expressionInfo{typeValue: shape.result, nullState: nullState}
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
		ReturnNull: nullState,
	}
	return result
}
