package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const mysqlModulePrefix = "builtin:MySQL::"

var (
	mysqlDatabaseClass          = ir.ClassID("builtin:MySQL::class::MySQLDatabase")
	mysqlTransactionClass       = ir.ClassID("builtin:MySQL::class::MySQLTransaction")
	mysqlResultClass            = ir.ClassID("builtin:MySQL::class::MySQLResult")
	mysqlValueClass             = ir.ClassID("builtin:MySQL::class::MySQLValue")
	mysqlErrorClass             = ir.ClassID("builtin:MySQL::class::MySQLError")
	mysqlDatabaseHandleField    = ir.FieldID("builtin:MySQL::class::MySQLDatabase::field::handle")
	mysqlTransactionHandleField = ir.FieldID("builtin:MySQL::class::MySQLTransaction::field::handle")
	mysqlResultDataField        = ir.FieldID("builtin:MySQL::class::MySQLResult::field::data")
	mysqlValueDataField         = ir.FieldID("builtin:MySQL::class::MySQLValue::field::data")
)

// mysqlCall lowers the MySQL module's plain functions. Any use marks the
// program as requiring the vendored go-sql-driver/mysql dependency tree at
// build time.
func (generator *generator) mysqlCall(value *ir.CallExpr) string {
	generator.usesMySQL = true
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), mysqlModulePrefix)
	errorClass := generator.descriptorName(mysqlErrorClass)
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
	// nullableText renders MySQL.connect's database argument as a *string:
	// nil when omitted or passed null, a boxed pointer to the String
	// otherwise. This is the module's one nullable *parameter* (every other
	// standard-module parameter across the compiler is NonNull), so it is
	// the one call site that asks generator.value for the boxed
	// representation instead of the usual unboxed scalar.
	nullableText := func(index int) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return "(*string)(nil)"
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, true)
	}
	switch name {
	case "connect":
		return generator.mysqlValueFrom(mysqlDatabaseClass, "AhdMySQLConnect("+errorClass+", "+
			text(0, `""`)+", "+text(1, `""`)+", "+text(2, `""`)+", "+
			integer(3, "int64(3306)")+", "+nullableText(4)+", "+
			text(5, `"tls"`)+", "+integer(6, "int64(10)")+")", meta)
	case "nullValue":
		return generator.mysqlValueFrom(mysqlValueClass, "AhdMySQLNullValue()", meta)
	case "fromInt":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.IntType}, false)
		return generator.mysqlValueFrom(mysqlValueClass, "AhdMySQLFromInt("+argument+")", meta)
	case "fromReal":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.RealType}, false)
		return generator.mysqlValueFrom(mysqlValueClass, "AhdMySQLFromReal("+errorClass+", "+argument+")", meta)
	case "fromString":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}, false)
		return generator.mysqlValueFrom(mysqlValueClass, "AhdMySQLFromString("+argument+")", meta)
	default:
		return generator.unsupported("MySQL function "+name, meta.Span)
	}
}

func mysqlDataField(class ir.ClassID) ir.FieldID {
	switch class {
	case mysqlDatabaseClass:
		return mysqlDatabaseHandleField
	case mysqlTransactionClass:
		return mysqlTransactionHandleField
	case mysqlResultClass:
		return mysqlResultDataField
	default:
		return mysqlValueDataField
	}
}

// mysqlValueFrom wraps one runtime text (a handle or an encoded value) into
// an instance through the generated class helper.
func (generator *generator) mysqlValueFrom(class ir.ClassID, data string, meta ir.ExprBase) string {
	helper, ok := generator.mysqlHelper(class)
	if !ok {
		return generator.unsupported("a MySQL value without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

// mysqlDataOf evaluates one MySQLDatabase, MySQLTransaction, MySQLResult, or
// MySQLValue expression exactly once and reads its hidden text field.
func (generator *generator) mysqlDataOf(class ir.ClassID, expression ir.Expr) string {
	rendered := generator.expr(expression)
	getter := generator.fieldName(mysqlDataField(class)) + "_get()"
	return "func(value " + generator.interfaceName(class) + ") string { return value." + getter + " }(" + rendered + ")"
}

func (generator *generator) mysqlHelper(class ir.ClassID) (string, bool) {
	if generator.layouts[class] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[class]; known {
		return name, true
	}
	name := mangleNamed("mh_", generator.classDisplayName(class), string(class))
	generator.timeHelpers[class] = name
	return name, true
}

func (generator *generator) emitMySQLHelpers(writer *emitter) {
	for _, class := range []ir.ClassID{mysqlDatabaseClass, mysqlTransactionClass, mysqlResultClass, mysqlValueClass} {
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

// mysqlParameterTexts evaluates the optional List<MySQLValue> argument
// exactly once and returns every element's encoded text, in order. An
// omitted argument binds no parameters.
func (generator *generator) mysqlParameterTexts(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	element := generator.interfaceName(mysqlValueClass)
	getter := generator.fieldName(mysqlValueDataField) + "_get()"
	return "func(list *AhdList[" + element + "]) []string { " +
		"items := list.Snapshot(); result := make([]string, len(items)); " +
		"for index, item := range items { result[index] = item." + getter + " }; " +
		"return result }(" + rendered + ")"
}

// mysqlOperation lowers the built-in members of MySQLDatabase,
// MySQLTransaction, MySQLResult, and MySQLValue.
func (generator *generator) mysqlOperation(name string, value *ir.CallExpr) string {
	generator.usesMySQL = true
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(mysqlErrorClass)
	text := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	switch name {
	case "MySQLDatabase.ping":
		return "AhdMySQLPing(" + errorClass + ", " + generator.mysqlDataOf(mysqlDatabaseClass, value.Callee) + ")"
	case "MySQLDatabase.execute":
		handle := generator.mysqlDataOf(mysqlDatabaseClass, value.Callee)
		return generator.mysqlValueFrom(mysqlResultClass,
			"AhdMySQLExecute("+errorClass+", "+handle+", "+text(0)+", "+generator.mysqlParameterTexts(value, 1)+")", meta)
	case "MySQLDatabase.query":
		handle := generator.mysqlDataOf(mysqlDatabaseClass, value.Callee)
		return generator.mysqlRowsResult("AhdMySQLQuery("+errorClass+", "+handle+", "+text(0)+", "+generator.mysqlParameterTexts(value, 1)+")", meta)
	case "MySQLDatabase.begin":
		handle := generator.mysqlDataOf(mysqlDatabaseClass, value.Callee)
		return generator.mysqlValueFrom(mysqlTransactionClass, "AhdMySQLBegin("+errorClass+", "+handle+")", meta)
	case "MySQLDatabase.close":
		return "AhdMySQLClose(" + errorClass + ", " + generator.mysqlDataOf(mysqlDatabaseClass, value.Callee) + ")"

	case "MySQLTransaction.execute":
		handle := generator.mysqlDataOf(mysqlTransactionClass, value.Callee)
		return generator.mysqlValueFrom(mysqlResultClass,
			"AhdMySQLTransactionExecute("+errorClass+", "+handle+", "+text(0)+", "+generator.mysqlParameterTexts(value, 1)+")", meta)
	case "MySQLTransaction.query":
		handle := generator.mysqlDataOf(mysqlTransactionClass, value.Callee)
		return generator.mysqlRowsResult("AhdMySQLTransactionQuery("+errorClass+", "+handle+", "+text(0)+", "+generator.mysqlParameterTexts(value, 1)+")", meta)
	case "MySQLTransaction.commit":
		return "AhdMySQLTransactionCommit(" + errorClass + ", " + generator.mysqlDataOf(mysqlTransactionClass, value.Callee) + ")"
	case "MySQLTransaction.rollback":
		return "AhdMySQLTransactionRollback(" + errorClass + ", " + generator.mysqlDataOf(mysqlTransactionClass, value.Callee) + ")"

	case "MySQLResult.affectedRows":
		return "AhdMySQLResultAffectedRows(" + errorClass + ", " + generator.mysqlDataOf(mysqlResultClass, value.Callee) + ")"
	case "MySQLResult.lastInsertId":
		// The runtime already returns *int64: this is the boxed
		// representation a MaybeNull Int uses throughout generated code, so
		// no extra AhdBox wrapping belongs here.
		return "AhdMySQLResultLastInsertID(" + errorClass + ", " + generator.mysqlDataOf(mysqlResultClass, value.Callee) + ")"

	case "MySQLValue.kind":
		return "AhdMySQLValueKind(" + errorClass + ", " + generator.mysqlDataOf(mysqlValueClass, value.Callee) + ")"
	case "MySQLValue.isNull":
		return "AhdMySQLValueIsNull(" + errorClass + ", " + generator.mysqlDataOf(mysqlValueClass, value.Callee) + ")"
	case "MySQLValue.int":
		return "AhdMySQLValueInt(" + errorClass + ", " + generator.mysqlDataOf(mysqlValueClass, value.Callee) + ")"
	case "MySQLValue.real":
		return "AhdMySQLValueReal(" + errorClass + ", " + generator.mysqlDataOf(mysqlValueClass, value.Callee) + ")"
	case "MySQLValue.string":
		return "AhdMySQLValueString(" + errorClass + ", " + generator.mysqlDataOf(mysqlValueClass, value.Callee) + ")"
	case "MySQLValue.isBinary":
		return "AhdMySQLValueIsBinary(" + errorClass + ", " + generator.mysqlDataOf(mysqlValueClass, value.Callee) + ")"
	case "MySQLValue.binarySize":
		return "AhdMySQLValueBinarySize(" + errorClass + ", " + generator.mysqlDataOf(mysqlValueClass, value.Callee) + ")"
	case "MySQLValue.binaryBase64":
		return "AhdMySQLValueBinaryBase64(" + errorClass + ", " + generator.mysqlDataOf(mysqlValueClass, value.Callee) + ")"
	default:
		return generator.unsupported("MySQL operation "+name, meta.Span)
	}
}

// mysqlRowsResult wraps the runtime's (columns, rows) reading into the
// List<Pair<String, MySQLValue>> query() returns: one Pair per row whose
// keys are the result columns in result order.
func (generator *generator) mysqlRowsResult(data string, meta ir.ExprBase) string {
	helper, ok := generator.mysqlHelper(mysqlValueClass)
	if !ok {
		return generator.unsupported("a MySQL query result without the MySQLValue Class declaration", meta.Span)
	}
	element := generator.interfaceName(mysqlValueClass)
	return "func(columns []string, rows [][]string) *AhdList[*AhdPair[string, " + element + "]] { " +
		"result := make([]*AhdPair[string, " + element + "], len(rows)); " +
		"for r, row := range rows { values := make([]" + element + ", len(row)); " +
		"for c, text := range row { values[c] = " + helper + "(text) }; " +
		"result[r] = AhdBuildPair(columns, values) }; " +
		"return AhdNewList(result...) }(" + data + ")"
}
