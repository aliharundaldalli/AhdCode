package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const sqliteModulePrefix = "builtin:SQLite::"

var (
	sqliteDatabaseClass       = ir.ClassID("builtin:SQLite::class::Database")
	sqliteValueClass          = ir.ClassID("builtin:SQLite::class::SQLiteValue")
	sqliteErrorClass          = ir.ClassID("builtin:SQLite::class::SQLiteError")
	sqliteDatabaseHandleField = ir.FieldID("builtin:SQLite::class::Database::field::handle")
	sqliteValueDataField      = ir.FieldID("builtin:SQLite::class::SQLiteValue::field::data")
)

// sqliteCall lowers the SQLite module's plain functions. Any use marks the
// program as requiring the bundled ahdsqlite helper at run time.
func (generator *generator) sqliteCall(value *ir.CallExpr) string {
	generator.usesSQLite = true
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), sqliteModulePrefix)
	errorClass := generator.descriptorName(sqliteErrorClass)
	switch name {
	case "open":
		path := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}, false)
		return generator.sqliteValueFrom(sqliteDatabaseClass, "AhdSQLiteOpen("+errorClass+", "+path+")", meta)
	case "nullValue":
		return generator.sqliteValueFrom(sqliteValueClass, "AhdSQLiteNullValue()", meta)
	case "fromInt":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.IntType}, false)
		return generator.sqliteValueFrom(sqliteValueClass, "AhdSQLiteFromInt("+argument+")", meta)
	case "fromReal":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.RealType}, false)
		return generator.sqliteValueFrom(sqliteValueClass, "AhdSQLiteFromReal("+errorClass+", "+argument+")", meta)
	case "fromString":
		argument := generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.StringType}, false)
		return generator.sqliteValueFrom(sqliteValueClass, "AhdSQLiteFromString("+argument+")", meta)
	default:
		return generator.unsupported("SQLite function "+name, meta.Span)
	}
}

func sqliteDataField(class ir.ClassID) ir.FieldID {
	if class == sqliteDatabaseClass {
		return sqliteDatabaseHandleField
	}
	return sqliteValueDataField
}

// sqliteValueFrom wraps one runtime text (a Database handle or a SQLiteValue
// encoding) into an instance through the generated class helper.
func (generator *generator) sqliteValueFrom(class ir.ClassID, data string, meta ir.ExprBase) string {
	helper, ok := generator.sqliteHelper(class)
	if !ok {
		return generator.unsupported("a SQLite value without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

// sqliteDataOf evaluates one Database or SQLiteValue expression exactly once
// and reads its hidden text field.
func (generator *generator) sqliteDataOf(class ir.ClassID, expression ir.Expr) string {
	rendered := generator.expr(expression)
	getter := generator.fieldName(sqliteDataField(class)) + "_get()"
	return "func(value " + generator.interfaceName(class) + ") string { return value." + getter + " }(" + rendered + ")"
}

func (generator *generator) sqliteHelper(class ir.ClassID) (string, bool) {
	if generator.layouts[class] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[class]; known {
		return name, true
	}
	name := mangleNamed("sh_", generator.classDisplayName(class), string(class))
	generator.timeHelpers[class] = name
	return name, true
}

func (generator *generator) emitSQLiteHelpers(writer *emitter) {
	for _, class := range []ir.ClassID{sqliteDatabaseClass, sqliteValueClass} {
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

// sqliteParameterTexts evaluates the optional List<SQLiteValue> argument
// exactly once and returns every element's encoded text, in order. An omitted
// argument binds no parameters.
func (generator *generator) sqliteParameterTexts(value *ir.CallExpr, index int) string {
	if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
		return "nil"
	}
	rendered := generator.expr(value.Arguments[index].Value)
	element := generator.interfaceName(sqliteValueClass)
	getter := generator.fieldName(sqliteValueDataField) + "_get()"
	return "func(list *AhdList[" + element + "]) []string { " +
		"items := list.Snapshot(); result := make([]string, len(items)); " +
		"for index, item := range items { result[index] = item." + getter + " }; " +
		"return result }(" + rendered + ")"
}

// sqliteOperation lowers the built-in members of Database and SQLiteValue.
func (generator *generator) sqliteOperation(name string, value *ir.CallExpr) string {
	generator.usesSQLite = true
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(sqliteErrorClass)
	text := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	switch name {
	case "Database.execute":
		handle := generator.sqliteDataOf(sqliteDatabaseClass, value.Callee)
		return "AhdSQLiteExecute(" + errorClass + ", " + handle + ", " + text(0) + ", " + generator.sqliteParameterTexts(value, 1) + ")"
	case "Database.query":
		handle := generator.sqliteDataOf(sqliteDatabaseClass, value.Callee)
		return generator.sqliteRowsResult("AhdSQLiteQuery("+errorClass+", "+handle+", "+text(0)+", "+generator.sqliteParameterTexts(value, 1)+")", meta)
	case "Database.lastInsertId":
		return "AhdSQLiteLastInsertID(" + errorClass + ", " + generator.sqliteDataOf(sqliteDatabaseClass, value.Callee) + ")"
	case "Database.begin":
		return "AhdSQLiteBegin(" + errorClass + ", " + generator.sqliteDataOf(sqliteDatabaseClass, value.Callee) + ")"
	case "Database.commit":
		return "AhdSQLiteCommit(" + errorClass + ", " + generator.sqliteDataOf(sqliteDatabaseClass, value.Callee) + ")"
	case "Database.rollback":
		return "AhdSQLiteRollback(" + errorClass + ", " + generator.sqliteDataOf(sqliteDatabaseClass, value.Callee) + ")"
	case "Database.close":
		return "AhdSQLiteClose(" + errorClass + ", " + generator.sqliteDataOf(sqliteDatabaseClass, value.Callee) + ")"

	case "SQLiteValue.kind":
		return "AhdSQLiteValueKind(" + errorClass + ", " + generator.sqliteDataOf(sqliteValueClass, value.Callee) + ")"
	case "SQLiteValue.isNull":
		return "AhdSQLiteValueIsNull(" + errorClass + ", " + generator.sqliteDataOf(sqliteValueClass, value.Callee) + ")"
	case "SQLiteValue.int":
		return "AhdSQLiteValueInt(" + errorClass + ", " + generator.sqliteDataOf(sqliteValueClass, value.Callee) + ")"
	case "SQLiteValue.real":
		return "AhdSQLiteValueReal(" + errorClass + ", " + generator.sqliteDataOf(sqliteValueClass, value.Callee) + ")"
	case "SQLiteValue.string":
		return "AhdSQLiteValueString(" + errorClass + ", " + generator.sqliteDataOf(sqliteValueClass, value.Callee) + ")"
	default:
		return generator.unsupported("SQLite operation "+name, meta.Span)
	}
}

// sqliteRowsResult wraps the runtime's (columns, rows) reading into the
// List<Pair<String, SQLiteValue>> query() returns: one Pair per row whose keys
// are the result columns in result order.
func (generator *generator) sqliteRowsResult(data string, meta ir.ExprBase) string {
	helper, ok := generator.sqliteHelper(sqliteValueClass)
	if !ok {
		return generator.unsupported("a SQLite query result without the SQLiteValue Class declaration", meta.Span)
	}
	element := generator.interfaceName(sqliteValueClass)
	return "func(columns []string, rows [][]string) *AhdList[*AhdPair[string, " + element + "]] { " +
		"result := make([]*AhdPair[string, " + element + "], len(rows)); " +
		"for r, row := range rows { values := make([]" + element + ", len(row)); " +
		"for c, text := range row { values[c] = " + helper + "(text) }; " +
		"result[r] = AhdBuildPair(columns, values) }; " +
		"return AhdNewList(result...) }(" + data + ")"
}
