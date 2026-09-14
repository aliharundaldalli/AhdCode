package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

// PostgreSQL (v1.4.0) is a network database module in the MySQL family: the same
// connect/execute/query/begin shape, its own type names, and no shared Database
// abstraction. Placeholders are PostgreSQL's own $1, $2, ...; generated values
// come back through RETURNING, so a PostgreSQLResult has no lastInsertId.

const postgresqlModuleID = "builtin:PostgreSQL"

var (
	postgresqlErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	postgresqlErrorClass       = &types.ClassSymbol{ModuleID: postgresqlModuleID, Name: "PostgreSQLError", Parent: postgresqlErrorParent}
	postgresqlDatabaseClass    = &types.ClassSymbol{ModuleID: postgresqlModuleID, Name: "PostgreSQLDatabase"}
	postgresqlTransactionClass = &types.ClassSymbol{ModuleID: postgresqlModuleID, Name: "PostgreSQLTransaction"}
	postgresqlResultClass      = &types.ClassSymbol{ModuleID: postgresqlModuleID, Name: "PostgreSQLResult"}
	postgresqlValueClass       = &types.ClassSymbol{ModuleID: postgresqlModuleID, Name: "PostgreSQLValue"}
)

// The canonical identities, exposed to lowering without coupling the public
// module interface to a backend.
func PostgreSQLErrorIdentity() *types.ClassSymbol       { return postgresqlErrorClass }
func PostgreSQLDatabaseIdentity() *types.ClassSymbol    { return postgresqlDatabaseClass }
func PostgreSQLTransactionIdentity() *types.ClassSymbol { return postgresqlTransactionClass }
func PostgreSQLResultIdentity() *types.ClassSymbol      { return postgresqlResultClass }
func PostgreSQLValueIdentity() *types.ClassSymbol       { return postgresqlValueClass }

// The members each Class publishes through built-in type operations, so
// has/has not reports the real surface and the IR Class agrees with the
// frontend.
var PostgreSQLDatabaseOperations = []string{"ping", "execute", "query", "begin", "close"}
var PostgreSQLTransactionOperations = []string{"execute", "query", "commit", "rollback"}
var PostgreSQLResultOperations = []string{"affectedRows"}
var PostgreSQLValueOperations = []string{"kind", "isNull", "bool", "int", "real", "string", "isBinary", "binarySize", "binaryBase64"}

func postgresqlDatabaseType() types.Type    { return types.Class{Symbol: postgresqlDatabaseClass} }
func postgresqlTransactionType() types.Type { return types.Class{Symbol: postgresqlTransactionClass} }
func postgresqlResultType() types.Type      { return types.Class{Symbol: postgresqlResultClass} }
func postgresqlValueType() types.Type       { return types.Class{Symbol: postgresqlValueClass} }

// postgresqlRowType is one query row: the result columns in result order, each
// holding one PostgreSQLValue. SQL NULL is a value of kind Null, never an
// AhdCode null.
func postgresqlRowType() types.Type {
	return types.Pair{Key: types.String, Value: postgresqlValueType()}
}

func postgresqlModuleInterface() *ModuleInterface {
	module := standardInterface(postgresqlModuleID, "PostgreSQL")
	classes := []struct {
		name     string
		identity *types.ClassSymbol
	}{
		{"PostgreSQLError", postgresqlErrorClass}, {"PostgreSQLDatabase", postgresqlDatabaseClass},
		{"PostgreSQLTransaction", postgresqlTransactionClass}, {"PostgreSQLResult", postgresqlResultClass},
		{"PostgreSQLValue", postgresqlValueClass},
	}
	for _, entry := range classes {
		symbol := &Symbol{
			Name: entry.name, Kind: ClassSymbol, Class: entry.identity,
			Type: types.Class{Symbol: entry.identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: postgresqlModuleID,
			Members: make(map[string]*Symbol),
		}
		if entry.name == "PostgreSQLError" {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[postgresqlModuleID+"\x00"+entry.name] = symbol
		addStandardExport(module, symbol)
	}
	addStandardExport(module, postgresqlConnectFunction())
	value := postgresqlValueType()
	addStandardExport(module, standardFunction(postgresqlModuleID, "nullValue", value))
	addStandardExport(module, standardFunction(postgresqlModuleID, "fromInt", value, types.Parameter{Name: "value", Type: types.Int}))
	addStandardExport(module, standardFunction(postgresqlModuleID, "fromReal", value, types.Parameter{Name: "value", Type: types.Real}))
	addStandardExport(module, standardFunction(postgresqlModuleID, "fromString", value, types.Parameter{Name: "value", Type: types.String}))
	addStandardExport(module, standardFunction(postgresqlModuleID, "fromBool", value, types.Parameter{Name: "value", Type: types.Bool}))
	sort.Strings(module.ExportNames)
	return module
}

// postgresqlConnectFunction mirrors MySQL.connect, including its one nullable
// parameter: database null (or "") uses PostgreSQL's own default, the database
// named after the role.
func postgresqlConnectFunction() *Symbol {
	parameters := []types.Parameter{
		{Name: "host", Type: types.String},
		{Name: "username", Type: types.String},
		{Name: "password", Type: types.String},
		{Name: "port", Type: types.Int, HasDefault: true},
		{Name: "database", Type: types.String, HasDefault: true},
		{Name: "security", Type: types.String, HasDefault: true},
		{Name: "timeoutSeconds", Type: types.Int, HasDefault: true},
	}
	signature := &types.Signature{Parameters: parameters, Return: postgresqlDatabaseType()}
	parameterNull := nonNullParameters(len(parameters))
	parameterNull[4] = MaybeNull // database
	return &Symbol{
		Name: "connect", Kind: FunctionSymbol, Type: types.Function{Signature: signature},
		ModuleRoot: true, Builtin: true, InitialNull: NonNull, OriginModuleID: postgresqlModuleID,
		Callable: &Callable{Signature: signature, ParameterNull: parameterNull, ReturnNull: NonNull},
	}
}

func postgresqlConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != postgresqlModuleID {
		return "", false
	}
	switch identity.Name {
	case "PostgreSQLDatabase":
		return "connect a PostgreSQLDatabase with PostgreSQL.connect(...)", true
	case "PostgreSQLTransaction":
		return "open a PostgreSQLTransaction with PostgreSQLDatabase.begin()", true
	case "PostgreSQLResult":
		return "a PostgreSQLResult is produced by PostgreSQLDatabase.execute(...) or PostgreSQLTransaction.execute(...)", true
	case "PostgreSQLValue":
		return "create a PostgreSQLValue with PostgreSQL.nullValue(), PostgreSQL.fromInt(value), PostgreSQL.fromReal(value), PostgreSQL.fromString(value), or PostgreSQL.fromBool(value)", true
	}
	return "", false
}

// postgresqlOperationShapes reuses MySQL's member shape: the trailing
// List<PostgreSQLValue> of execute and query may be omitted.
func postgresqlOperationShapes() map[TypeOperation]mysqlOperationShape {
	none := []types.Type{}
	statement := []types.Type{types.String, types.List{Element: postgresqlValueType()}}
	rows := types.List{Element: postgresqlRowType()}
	statementHint := "pass one SQL String and optionally a List<PostgreSQLValue> of $1, $2, ... parameters"
	return map[TypeOperation]mysqlOperationShape{
		PostgreSQLDatabasePing:    {none, 0, types.Nothing, false, "call ping with no argument"},
		PostgreSQLDatabaseExecute: {statement, 1, postgresqlResultType(), false, statementHint},
		PostgreSQLDatabaseQuery:   {statement, 1, rows, false, statementHint},
		PostgreSQLDatabaseBegin:   {none, 0, postgresqlTransactionType(), false, "call begin with no argument"},
		PostgreSQLDatabaseClose:   {none, 0, types.Nothing, false, "call close with no argument"},

		PostgreSQLTransactionExecute:  {statement, 1, postgresqlResultType(), false, statementHint},
		PostgreSQLTransactionQuery:    {statement, 1, rows, false, statementHint},
		PostgreSQLTransactionCommit:   {none, 0, types.Nothing, false, "call commit with no argument"},
		PostgreSQLTransactionRollback: {none, 0, types.Nothing, false, "call rollback with no argument"},

		PostgreSQLResultAffectedRows: {none, 0, types.Int, false, "call affectedRows with no argument"},

		PostgreSQLValueKind:         {none, 0, types.String, false, "call kind with no argument"},
		PostgreSQLValueIsNull:       {none, 0, types.Bool, false, "call isNull with no argument"},
		PostgreSQLValueBool:         {none, 0, types.Bool, false, "call bool on a Bool PostgreSQLValue"},
		PostgreSQLValueInt:          {none, 0, types.Int, false, "call int on an Int PostgreSQLValue"},
		PostgreSQLValueReal:         {none, 0, types.Real, false, "call real on a Real or Int PostgreSQLValue"},
		PostgreSQLValueString:       {none, 0, types.String, false, "call string on a String PostgreSQLValue"},
		PostgreSQLValueIsBinary:     {none, 0, types.Bool, false, "call isBinary with no argument"},
		PostgreSQLValueBinarySize:   {none, 0, types.Int, false, "call binarySize on a Binary PostgreSQLValue"},
		PostgreSQLValueBinaryBase64: {none, 0, types.String, false, "call binaryBase64 on a Binary PostgreSQLValue"},
	}
}

var postgresqlOperationNames = map[string]map[string]TypeOperation{
	"PostgreSQLDatabase": {
		"ping": PostgreSQLDatabasePing, "execute": PostgreSQLDatabaseExecute, "query": PostgreSQLDatabaseQuery,
		"begin": PostgreSQLDatabaseBegin, "close": PostgreSQLDatabaseClose,
	},
	"PostgreSQLTransaction": {
		"execute": PostgreSQLTransactionExecute, "query": PostgreSQLTransactionQuery,
		"commit": PostgreSQLTransactionCommit, "rollback": PostgreSQLTransactionRollback,
	},
	"PostgreSQLResult": {"affectedRows": PostgreSQLResultAffectedRows},
	"PostgreSQLValue": {
		"kind": PostgreSQLValueKind, "isNull": PostgreSQLValueIsNull, "bool": PostgreSQLValueBool,
		"int": PostgreSQLValueInt, "real": PostgreSQLValueReal, "string": PostgreSQLValueString,
		"isBinary": PostgreSQLValueIsBinary, "binarySize": PostgreSQLValueBinarySize, "binaryBase64": PostgreSQLValueBinaryBase64,
	},
}

// postgresqlOperationFor names the built-in member a PostgreSQL value publishes.
// Only the compiler-supplied identities match.
func postgresqlOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != postgresqlModuleID {
		return "", false
	}
	operation, known := postgresqlOperationNames[class.Symbol.Name][name]
	return operation, known
}
