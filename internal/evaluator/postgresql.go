package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The PostgreSQL standard module's REPL implementation. It drives the same
// in-process runtime client the native backend emits (ahdruntime/postgresql.go),
// so the evaluator and compiled programs share one pool implementation, one
// value encoding, and one set of error messages.

var (
	evaluatorPostgreSQLDatabaseClass    = ir.ClassID("builtin:PostgreSQL::class::PostgreSQLDatabase")
	evaluatorPostgreSQLTransactionClass = ir.ClassID("builtin:PostgreSQL::class::PostgreSQLTransaction")
	evaluatorPostgreSQLResultClass      = ir.ClassID("builtin:PostgreSQL::class::PostgreSQLResult")
	evaluatorPostgreSQLValueClass       = ir.ClassID("builtin:PostgreSQL::class::PostgreSQLValue")

	evaluatorPostgreSQLDatabaseField    = ir.FieldID("builtin:PostgreSQL::class::PostgreSQLDatabase::field::handle")
	evaluatorPostgreSQLTransactionField = ir.FieldID("builtin:PostgreSQL::class::PostgreSQLTransaction::field::handle")
	evaluatorPostgreSQLResultField      = ir.FieldID("builtin:PostgreSQL::class::PostgreSQLResult::field::data")
	evaluatorPostgreSQLValueField       = ir.FieldID("builtin:PostgreSQL::class::PostgreSQLValue::field::data")
)

func (session *Session) postgresqlInstance(class ir.ClassID, field ir.FieldID, data string) *Instance {
	return &Instance{Class: class, Fields: map[ir.FieldID]any{field: data}}
}

func (session *Session) postgresqlValue(data string) *Instance {
	return session.postgresqlInstance(evaluatorPostgreSQLValueClass, evaluatorPostgreSQLValueField, data)
}

func (session *Session) postgresqlHandleOf(value any, class ir.ClassID, field ir.FieldID, name string) string {
	instance := session.requireInstance(value)
	handle, ok := instance.Fields[field].(string)
	if !ok || instance.Class != class {
		session.raise("PostgreSQLError", name+" storage is corrupted")
	}
	return handle
}

func (session *Session) postgresqlDatabaseHandle(value any) string {
	return session.postgresqlHandleOf(value, evaluatorPostgreSQLDatabaseClass, evaluatorPostgreSQLDatabaseField, "PostgreSQLDatabase")
}

func (session *Session) postgresqlTransactionHandle(value any) string {
	return session.postgresqlHandleOf(value, evaluatorPostgreSQLTransactionClass, evaluatorPostgreSQLTransactionField, "PostgreSQLTransaction")
}

func (session *Session) postgresqlValueData(value any) string {
	return session.postgresqlHandleOf(value, evaluatorPostgreSQLValueClass, evaluatorPostgreSQLValueField, "PostgreSQLValue")
}

func (session *Session) postgresqlCheck(err error) {
	if err != nil {
		session.raise("PostgreSQLError", err.Error())
	}
}

func (session *Session) postgresqlParameters(args []any, index int) []string {
	if index >= len(args) || args[index] == nil {
		return nil
	}
	list := session.requireList(args[index])
	result := make([]string, len(list.Items))
	for position, item := range list.Items {
		result[position] = session.postgresqlValueData(item)
	}
	return result
}

func (session *Session) postgresqlRows(columns []string, rows [][]string) *List {
	items := make([]any, len(rows))
	for index, row := range rows {
		pair := &Pair{Values: make(map[any]any, len(columns))}
		for column, text := range row {
			pairSet(pair, columns[column], session.postgresqlValue(text))
		}
		items[index] = pair
	}
	return &List{Items: items}
}

func (session *Session) postgresqlBuiltin(name string, args []any) any {
	switch name {
	case "connect":
		handle, err := ahdruntime.PostgreSQLConnect(args[0].(string), args[1].(string), args[2].(string),
			session.httpIntArg(args, 3, 5432), session.mysqlOptionalDatabaseArg(args, 4),
			session.httpStringArg(args, 5, "tls"), session.httpIntArg(args, 6, 10))
		session.postgresqlCheck(err)
		return session.postgresqlInstance(evaluatorPostgreSQLDatabaseClass, evaluatorPostgreSQLDatabaseField, handle)
	case "nullValue":
		return session.postgresqlValue(ahdruntime.PostgreSQLNullValue())
	case "fromInt":
		return session.postgresqlValue(ahdruntime.PostgreSQLFromInt(args[0].(int64)))
	case "fromReal":
		text, err := ahdruntime.PostgreSQLFromReal(args[0].(float64))
		session.postgresqlCheck(err)
		return session.postgresqlValue(text)
	case "fromString":
		return session.postgresqlValue(ahdruntime.PostgreSQLFromString(args[0].(string)))
	case "fromBool":
		return session.postgresqlValue(ahdruntime.PostgreSQLFromBool(args[0].(bool)))
	}
	session.raise("Error", "unsupported PostgreSQL function "+name)
	return nil
}

func (session *Session) postgresqlOperation(name string, receiver any, args []any) any {
	switch name {
	case "PostgreSQLDatabase.ping":
		session.postgresqlCheck(ahdruntime.PostgreSQLPing(session.postgresqlDatabaseHandle(receiver)))
		return Nothing
	case "PostgreSQLDatabase.execute":
		data, err := ahdruntime.PostgreSQLExecute(session.postgresqlDatabaseHandle(receiver), args[0].(string), session.postgresqlParameters(args, 1))
		session.postgresqlCheck(err)
		return session.postgresqlInstance(evaluatorPostgreSQLResultClass, evaluatorPostgreSQLResultField, data)
	case "PostgreSQLDatabase.query":
		columns, rows, err := ahdruntime.PostgreSQLQuery(session.postgresqlDatabaseHandle(receiver), args[0].(string), session.postgresqlParameters(args, 1))
		session.postgresqlCheck(err)
		return session.postgresqlRows(columns, rows)
	case "PostgreSQLDatabase.begin":
		handle, err := ahdruntime.PostgreSQLBegin(session.postgresqlDatabaseHandle(receiver))
		session.postgresqlCheck(err)
		return session.postgresqlInstance(evaluatorPostgreSQLTransactionClass, evaluatorPostgreSQLTransactionField, handle)
	case "PostgreSQLDatabase.close":
		session.postgresqlCheck(ahdruntime.PostgreSQLClose(session.postgresqlDatabaseHandle(receiver)))
		return Nothing

	case "PostgreSQLTransaction.execute":
		data, err := ahdruntime.PostgreSQLTransactionExecute(session.postgresqlTransactionHandle(receiver), args[0].(string), session.postgresqlParameters(args, 1))
		session.postgresqlCheck(err)
		return session.postgresqlInstance(evaluatorPostgreSQLResultClass, evaluatorPostgreSQLResultField, data)
	case "PostgreSQLTransaction.query":
		columns, rows, err := ahdruntime.PostgreSQLTransactionQuery(session.postgresqlTransactionHandle(receiver), args[0].(string), session.postgresqlParameters(args, 1))
		session.postgresqlCheck(err)
		return session.postgresqlRows(columns, rows)
	case "PostgreSQLTransaction.commit":
		session.postgresqlCheck(ahdruntime.PostgreSQLTransactionCommit(session.postgresqlTransactionHandle(receiver)))
		return Nothing
	case "PostgreSQLTransaction.rollback":
		session.postgresqlCheck(ahdruntime.PostgreSQLTransactionRollback(session.postgresqlTransactionHandle(receiver)))
		return Nothing

	case "PostgreSQLResult.affectedRows":
		value, err := ahdruntime.PostgreSQLResultAffectedRows(
			session.postgresqlHandleOf(receiver, evaluatorPostgreSQLResultClass, evaluatorPostgreSQLResultField, "PostgreSQLResult"))
		session.postgresqlCheck(err)
		return value
	}

	data := session.postgresqlValueData(receiver)
	var result any
	var err error
	switch name {
	case "PostgreSQLValue.kind":
		result, err = ahdruntime.PostgreSQLValueKind(data)
	case "PostgreSQLValue.isNull":
		result, err = ahdruntime.PostgreSQLValueIsNull(data)
	case "PostgreSQLValue.bool":
		result, err = ahdruntime.PostgreSQLValueBool(data)
	case "PostgreSQLValue.int":
		result, err = ahdruntime.PostgreSQLValueInt(data)
	case "PostgreSQLValue.real":
		result, err = ahdruntime.PostgreSQLValueReal(data)
	case "PostgreSQLValue.string":
		result, err = ahdruntime.PostgreSQLValueString(data)
	case "PostgreSQLValue.isBinary":
		result, err = ahdruntime.PostgreSQLValueIsBinary(data)
	case "PostgreSQLValue.binarySize":
		result, err = ahdruntime.PostgreSQLValueBinarySize(data)
	case "PostgreSQLValue.binaryBase64":
		result, err = ahdruntime.PostgreSQLValueBinaryBase64(data)
	default:
		session.raise("Error", "unsupported PostgreSQL operation "+name)
	}
	session.postgresqlCheck(err)
	return result
}
