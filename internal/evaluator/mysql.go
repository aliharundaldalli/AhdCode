package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The MySQL standard module's REPL/evaluated implementation. It drives the
// very same in-process runtime client the native backend emits
// (ahdruntime/mysql.go), so the persistent evaluator and compiled programs
// share one connection pool implementation, one value encoding, and one set
// of error messages.

var (
	evaluatorMySQLDatabaseClass    = ir.ClassID("builtin:MySQL::class::MySQLDatabase")
	evaluatorMySQLTransactionClass = ir.ClassID("builtin:MySQL::class::MySQLTransaction")
	evaluatorMySQLResultClass      = ir.ClassID("builtin:MySQL::class::MySQLResult")
	evaluatorMySQLValueClass       = ir.ClassID("builtin:MySQL::class::MySQLValue")

	evaluatorMySQLDatabaseField    = ir.FieldID("builtin:MySQL::class::MySQLDatabase::field::handle")
	evaluatorMySQLTransactionField = ir.FieldID("builtin:MySQL::class::MySQLTransaction::field::handle")
	evaluatorMySQLResultField      = ir.FieldID("builtin:MySQL::class::MySQLResult::field::data")
	evaluatorMySQLValueField       = ir.FieldID("builtin:MySQL::class::MySQLValue::field::data")
)

func (session *Session) mysqlDatabase(handle string) *Instance {
	return &Instance{Class: evaluatorMySQLDatabaseClass, Fields: map[ir.FieldID]any{evaluatorMySQLDatabaseField: handle}}
}

func (session *Session) mysqlTransaction(handle string) *Instance {
	return &Instance{Class: evaluatorMySQLTransactionClass, Fields: map[ir.FieldID]any{evaluatorMySQLTransactionField: handle}}
}

func (session *Session) mysqlResult(data string) *Instance {
	return &Instance{Class: evaluatorMySQLResultClass, Fields: map[ir.FieldID]any{evaluatorMySQLResultField: data}}
}

func (session *Session) mysqlValue(data string) *Instance {
	return &Instance{Class: evaluatorMySQLValueClass, Fields: map[ir.FieldID]any{evaluatorMySQLValueField: data}}
}

func (session *Session) mysqlHandleOf(value any, class ir.ClassID, field ir.FieldID, name string) string {
	instance := session.requireInstance(value)
	handle, ok := instance.Fields[field].(string)
	if !ok || instance.Class != class {
		session.raise("MySQLError", name+" storage is corrupted")
	}
	return handle
}

func (session *Session) mysqlDatabaseHandle(value any) string {
	return session.mysqlHandleOf(value, evaluatorMySQLDatabaseClass, evaluatorMySQLDatabaseField, "MySQLDatabase")
}

func (session *Session) mysqlTransactionHandle(value any) string {
	return session.mysqlHandleOf(value, evaluatorMySQLTransactionClass, evaluatorMySQLTransactionField, "MySQLTransaction")
}

func (session *Session) mysqlResultData(value any) string {
	return session.mysqlHandleOf(value, evaluatorMySQLResultClass, evaluatorMySQLResultField, "MySQLResult")
}

func (session *Session) mysqlValueData(value any) string {
	return session.mysqlHandleOf(value, evaluatorMySQLValueClass, evaluatorMySQLValueField, "MySQLValue")
}

func (session *Session) mysqlCheck(err error) {
	if err != nil {
		session.raise("MySQLError", err.Error())
	}
}

func (session *Session) mysqlOptionalDatabaseArg(args []any, index int) *string {
	if index >= len(args) || args[index] == nil {
		return nil
	}
	value := args[index].(string)
	return &value
}

func (session *Session) mysqlParameters(args []any, index int) []string {
	if index >= len(args) || args[index] == nil {
		return nil
	}
	list := session.requireList(args[index])
	result := make([]string, len(list.Items))
	for position, item := range list.Items {
		result[position] = session.mysqlValueData(item)
	}
	return result
}

func (session *Session) mysqlRows(columns []string, rows [][]string) *List {
	items := make([]any, len(rows))
	for index, row := range rows {
		pair := &Pair{Values: make(map[any]any, len(columns))}
		for column, text := range row {
			pairSet(pair, columns[column], session.mysqlValue(text))
		}
		items[index] = pair
	}
	return &List{Items: items}
}

func (session *Session) mysqlBuiltin(name string, args []any) any {
	switch name {
	case "connect":
		host := args[0].(string)
		username := args[1].(string)
		password := args[2].(string)
		port := session.httpIntArg(args, 3, 3306)
		database := session.mysqlOptionalDatabaseArg(args, 4)
		security := session.httpStringArg(args, 5, "tls")
		timeoutSeconds := session.httpIntArg(args, 6, 10)
		handle, err := ahdruntime.MySQLConnect(host, username, password, port, database, security, timeoutSeconds)
		session.mysqlCheck(err)
		return session.mysqlDatabase(handle)
	case "nullValue":
		return session.mysqlValue(ahdruntime.MySQLNullValue())
	case "fromInt":
		return session.mysqlValue(ahdruntime.MySQLFromInt(args[0].(int64)))
	case "fromReal":
		text, err := ahdruntime.MySQLFromReal(args[0].(float64))
		session.mysqlCheck(err)
		return session.mysqlValue(text)
	case "fromString":
		return session.mysqlValue(ahdruntime.MySQLFromString(args[0].(string)))
	}
	session.raise("Error", "unsupported MySQL function "+name)
	return nil
}

func (session *Session) mysqlOperation(name string, receiver any, args []any) any {
	switch name {
	case "MySQLDatabase.ping":
		session.mysqlCheck(ahdruntime.MySQLPing(session.mysqlDatabaseHandle(receiver)))
		return Nothing
	case "MySQLDatabase.execute":
		data, err := ahdruntime.MySQLExecute(session.mysqlDatabaseHandle(receiver), args[0].(string), session.mysqlParameters(args, 1))
		session.mysqlCheck(err)
		return session.mysqlResult(data)
	case "MySQLDatabase.query":
		columns, rows, err := ahdruntime.MySQLQuery(session.mysqlDatabaseHandle(receiver), args[0].(string), session.mysqlParameters(args, 1))
		session.mysqlCheck(err)
		return session.mysqlRows(columns, rows)
	case "MySQLDatabase.begin":
		handle, err := ahdruntime.MySQLBegin(session.mysqlDatabaseHandle(receiver))
		session.mysqlCheck(err)
		return session.mysqlTransaction(handle)
	case "MySQLDatabase.close":
		session.mysqlCheck(ahdruntime.MySQLClose(session.mysqlDatabaseHandle(receiver)))
		return Nothing

	case "MySQLTransaction.execute":
		data, err := ahdruntime.MySQLTransactionExecute(session.mysqlTransactionHandle(receiver), args[0].(string), session.mysqlParameters(args, 1))
		session.mysqlCheck(err)
		return session.mysqlResult(data)
	case "MySQLTransaction.query":
		columns, rows, err := ahdruntime.MySQLTransactionQuery(session.mysqlTransactionHandle(receiver), args[0].(string), session.mysqlParameters(args, 1))
		session.mysqlCheck(err)
		return session.mysqlRows(columns, rows)
	case "MySQLTransaction.commit":
		session.mysqlCheck(ahdruntime.MySQLTransactionCommit(session.mysqlTransactionHandle(receiver)))
		return Nothing
	case "MySQLTransaction.rollback":
		session.mysqlCheck(ahdruntime.MySQLTransactionRollback(session.mysqlTransactionHandle(receiver)))
		return Nothing

	case "MySQLResult.affectedRows":
		value, err := ahdruntime.MySQLResultAffectedRows(session.mysqlResultData(receiver))
		session.mysqlCheck(err)
		return value
	case "MySQLResult.lastInsertId":
		value, err := ahdruntime.MySQLResultLastInsertID(session.mysqlResultData(receiver))
		session.mysqlCheck(err)
		if value == nil {
			return nil
		}
		return *value

	case "MySQLValue.kind":
		value, err := ahdruntime.MySQLValueKind(session.mysqlValueData(receiver))
		session.mysqlCheck(err)
		return value
	case "MySQLValue.isNull":
		value, err := ahdruntime.MySQLValueIsNull(session.mysqlValueData(receiver))
		session.mysqlCheck(err)
		return value
	case "MySQLValue.int":
		value, err := ahdruntime.MySQLValueInt(session.mysqlValueData(receiver))
		session.mysqlCheck(err)
		return value
	case "MySQLValue.real":
		value, err := ahdruntime.MySQLValueReal(session.mysqlValueData(receiver))
		session.mysqlCheck(err)
		return value
	case "MySQLValue.string":
		value, err := ahdruntime.MySQLValueString(session.mysqlValueData(receiver))
		session.mysqlCheck(err)
		return value
	case "MySQLValue.isBinary":
		value, err := ahdruntime.MySQLValueIsBinary(session.mysqlValueData(receiver))
		session.mysqlCheck(err)
		return value
	case "MySQLValue.binarySize":
		value, err := ahdruntime.MySQLValueBinarySize(session.mysqlValueData(receiver))
		session.mysqlCheck(err)
		return value
	case "MySQLValue.binaryBase64":
		value, err := ahdruntime.MySQLValueBinaryBase64(session.mysqlValueData(receiver))
		session.mysqlCheck(err)
		return value
	}
	session.raise("Error", "unsupported MySQL operation "+name)
	return nil
}
