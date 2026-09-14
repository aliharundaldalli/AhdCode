package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const postgresqlModulePrefix = "builtin:PostgreSQL::"

var (
	postgresqlDatabaseClass          = ir.ClassID("builtin:PostgreSQL::class::PostgreSQLDatabase")
	postgresqlTransactionClass       = ir.ClassID("builtin:PostgreSQL::class::PostgreSQLTransaction")
	postgresqlResultClass            = ir.ClassID("builtin:PostgreSQL::class::PostgreSQLResult")
	postgresqlValueClass             = ir.ClassID("builtin:PostgreSQL::class::PostgreSQLValue")
	postgresqlErrorClass             = ir.ClassID("builtin:PostgreSQL::class::PostgreSQLError")
	postgresqlDatabaseHandleField    = ir.FieldID("builtin:PostgreSQL::class::PostgreSQLDatabase::field::handle")
	postgresqlTransactionHandleField = ir.FieldID("builtin:PostgreSQL::class::PostgreSQLTransaction::field::handle")
	postgresqlResultDataField        = ir.FieldID("builtin:PostgreSQL::class::PostgreSQLResult::field::data")
	postgresqlValueDataField         = ir.FieldID("builtin:PostgreSQL::class::PostgreSQLValue::field::data")
)

// postgresqlCall lowers the PostgreSQL module's plain functions. Any use marks
// the program as requiring the vendored pgx dependency tree at build time.
func (generator *generator) postgresqlCall(value *ir.CallExpr) string {
	generator.usesPostgreSQL = true
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), postgresqlModulePrefix)
	errorClass := generator.descriptorName(postgresqlErrorClass)
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
	// PostgreSQL.connect's database argument is nullable, like MySQL's: nil when
	// omitted or null, a boxed String otherwise.
	nullableText := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return "(*string)(nil)"
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, true)
	}
	switch name {
	case "connect":
		return generator.postgresqlValueFrom(postgresqlDatabaseClass, "AhdPostgreSQLConnect("+errorClass+", "+
			text(0, `""`)+", "+text(1, `""`)+", "+text(2, `""`)+", "+
			integer(3, "int64(5432)")+", "+nullableText(4)+", "+
			text(5, `"tls"`)+", "+integer(6, "int64(10)")+")", meta)
	case "nullValue":
		return generator.postgresqlValueFrom(postgresqlValueClass, "AhdPostgreSQLNullValue()", meta)
	case "fromInt":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.IntType}, false)
		return generator.postgresqlValueFrom(postgresqlValueClass, "AhdPostgreSQLFromInt("+argument+")", meta)
	case "fromReal":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.RealType}, false)
		return generator.postgresqlValueFrom(postgresqlValueClass, "AhdPostgreSQLFromReal("+errorClass+", "+argument+")", meta)
	case "fromString":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}, false)
		return generator.postgresqlValueFrom(postgresqlValueClass, "AhdPostgreSQLFromString("+argument+")", meta)
	case "fromBool":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.BoolType}, false)
		return generator.postgresqlValueFrom(postgresqlValueClass, "AhdPostgreSQLFromBool("+argument+")", meta)
	default:
		return generator.unsupported("PostgreSQL function "+name, meta.Span)
	}
}

func postgresqlDataField(class ir.ClassID) ir.FieldID {
	switch class {
	case postgresqlDatabaseClass:
		return postgresqlDatabaseHandleField
	case postgresqlTransactionClass:
		return postgresqlTransactionHandleField
	case postgresqlResultClass:
		return postgresqlResultDataField
	default:
		return postgresqlValueDataField
	}
}

func (generator *generator) postgresqlValueFrom(class ir.ClassID, data string, meta ir.ExprBase) string {
	helper, ok := generator.postgresqlHelper(class)
	if !ok {
		return generator.unsupported("a PostgreSQL value without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

// postgresqlDataOf evaluates one PostgreSQL value expression exactly once and
// reads its hidden text field.
func (generator *generator) postgresqlDataOf(class ir.ClassID, expression ir.Expr) string {
	rendered := generator.expr(expression)
	getter := generator.fieldName(postgresqlDataField(class)) + "_get()"
	return "func(value " + generator.interfaceName(class) + ") string { return value." + getter + " }(" + rendered + ")"
}

func (generator *generator) postgresqlHelper(class ir.ClassID) (string, bool) {
	if generator.layouts[class] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[class]; known {
		return name, true
	}
	name := mangleNamed("pg_", generator.classDisplayName(class), string(class))
	generator.timeHelpers[class] = name
	return name, true
}

func (generator *generator) emitPostgreSQLHelpers(writer *emitter) {
	for _, class := range []ir.ClassID{postgresqlDatabaseClass, postgresqlTransactionClass, postgresqlResultClass, postgresqlValueClass} {
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

// postgresqlParameterTexts evaluates the optional List<PostgreSQLValue>
// argument exactly once and returns every element's encoded text, in order.
func (generator *generator) postgresqlParameterTexts(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	element := generator.interfaceName(postgresqlValueClass)
	getter := generator.fieldName(postgresqlValueDataField) + "_get()"
	return "func(list *AhdList[" + element + "]) []string { " +
		"items := list.Snapshot(); result := make([]string, len(items)); " +
		"for index, item := range items { result[index] = item." + getter + " }; " +
		"return result }(" + rendered + ")"
}

// postgresqlOperation lowers the built-in members of PostgreSQLDatabase,
// PostgreSQLTransaction, PostgreSQLResult, and PostgreSQLValue.
func (generator *generator) postgresqlOperation(name string, value *ir.CallExpr) string {
	generator.usesPostgreSQL = true
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(postgresqlErrorClass)
	text := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	valueCall := func(runtime string) string {
		return runtime + "(" + errorClass + ", " + generator.postgresqlDataOf(postgresqlValueClass, value.Callee) + ")"
	}
	switch name {
	case "PostgreSQLDatabase.ping":
		return "AhdPostgreSQLPing(" + errorClass + ", " + generator.postgresqlDataOf(postgresqlDatabaseClass, value.Callee) + ")"
	case "PostgreSQLDatabase.execute":
		handle := generator.postgresqlDataOf(postgresqlDatabaseClass, value.Callee)
		return generator.postgresqlValueFrom(postgresqlResultClass,
			"AhdPostgreSQLExecute("+errorClass+", "+handle+", "+text(0)+", "+generator.postgresqlParameterTexts(value, 1)+")", meta)
	case "PostgreSQLDatabase.query":
		handle := generator.postgresqlDataOf(postgresqlDatabaseClass, value.Callee)
		return generator.postgresqlRowsResult("AhdPostgreSQLQuery("+errorClass+", "+handle+", "+text(0)+", "+generator.postgresqlParameterTexts(value, 1)+")", meta)
	case "PostgreSQLDatabase.begin":
		handle := generator.postgresqlDataOf(postgresqlDatabaseClass, value.Callee)
		return generator.postgresqlValueFrom(postgresqlTransactionClass, "AhdPostgreSQLBegin("+errorClass+", "+handle+")", meta)
	case "PostgreSQLDatabase.close":
		return "AhdPostgreSQLClose(" + errorClass + ", " + generator.postgresqlDataOf(postgresqlDatabaseClass, value.Callee) + ")"

	case "PostgreSQLTransaction.execute":
		handle := generator.postgresqlDataOf(postgresqlTransactionClass, value.Callee)
		return generator.postgresqlValueFrom(postgresqlResultClass,
			"AhdPostgreSQLTransactionExecute("+errorClass+", "+handle+", "+text(0)+", "+generator.postgresqlParameterTexts(value, 1)+")", meta)
	case "PostgreSQLTransaction.query":
		handle := generator.postgresqlDataOf(postgresqlTransactionClass, value.Callee)
		return generator.postgresqlRowsResult("AhdPostgreSQLTransactionQuery("+errorClass+", "+handle+", "+text(0)+", "+generator.postgresqlParameterTexts(value, 1)+")", meta)
	case "PostgreSQLTransaction.commit":
		return "AhdPostgreSQLTransactionCommit(" + errorClass + ", " + generator.postgresqlDataOf(postgresqlTransactionClass, value.Callee) + ")"
	case "PostgreSQLTransaction.rollback":
		return "AhdPostgreSQLTransactionRollback(" + errorClass + ", " + generator.postgresqlDataOf(postgresqlTransactionClass, value.Callee) + ")"

	case "PostgreSQLResult.affectedRows":
		return "AhdPostgreSQLResultAffectedRows(" + errorClass + ", " + generator.postgresqlDataOf(postgresqlResultClass, value.Callee) + ")"

	case "PostgreSQLValue.kind":
		return valueCall("AhdPostgreSQLValueKind")
	case "PostgreSQLValue.isNull":
		return valueCall("AhdPostgreSQLValueIsNull")
	case "PostgreSQLValue.bool":
		return valueCall("AhdPostgreSQLValueBool")
	case "PostgreSQLValue.int":
		return valueCall("AhdPostgreSQLValueInt")
	case "PostgreSQLValue.real":
		return valueCall("AhdPostgreSQLValueReal")
	case "PostgreSQLValue.string":
		return valueCall("AhdPostgreSQLValueString")
	case "PostgreSQLValue.isBinary":
		return valueCall("AhdPostgreSQLValueIsBinary")
	case "PostgreSQLValue.binarySize":
		return valueCall("AhdPostgreSQLValueBinarySize")
	case "PostgreSQLValue.binaryBase64":
		return valueCall("AhdPostgreSQLValueBinaryBase64")
	default:
		return generator.unsupported("PostgreSQL operation "+name, meta.Span)
	}
}

// postgresqlRowsResult wraps the runtime's (columns, rows) reading into the
// List<Pair<String, PostgreSQLValue>> query() returns.
func (generator *generator) postgresqlRowsResult(data string, meta ir.ExprBase) string {
	helper, ok := generator.postgresqlHelper(postgresqlValueClass)
	if !ok {
		return generator.unsupported("a PostgreSQL query result without the PostgreSQLValue Class declaration", meta.Span)
	}
	element := generator.interfaceName(postgresqlValueClass)
	return "func(columns []string, rows [][]string) *AhdList[*AhdPair[string, " + element + "]] { " +
		"result := make([]*AhdPair[string, " + element + "], len(rows)); " +
		"for r, row := range rows { values := make([]" + element + ", len(row)); " +
		"for c, text := range row { values[c] = " + helper + "(text) }; " +
		"result[r] = AhdBuildPair(columns, values) }; " +
		"return AhdNewList(result...) }(" + data + ")"
}
