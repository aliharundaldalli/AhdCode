package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The SQLite standard module's REPL implementation. It drives the very same
// stdlib-only runtime client the native backend emits (ahdruntime/sqlite.go),
// so the persistent evaluator and compiled programs share one helper
// protocol, one value encoding, and one set of error messages. The helper
// process lives as long as the REPL, which is what lets an in-memory Database
// survive successive entries of the same session.

var (
	evaluatorSQLiteDatabaseClass = ir.ClassID("builtin:SQLite::class::Database")
	evaluatorSQLiteValueClass    = ir.ClassID("builtin:SQLite::class::SQLiteValue")

	evaluatorSQLiteDatabaseField = ir.FieldID("builtin:SQLite::class::Database::field::handle")
	evaluatorSQLiteValueField    = ir.FieldID("builtin:SQLite::class::SQLiteValue::field::data")
)

func (session *Session) sqliteDatabase(handle string) *Instance {
	return &Instance{Class: evaluatorSQLiteDatabaseClass, Fields: map[ir.FieldID]any{evaluatorSQLiteDatabaseField: handle}}
}

func (session *Session) sqliteValue(data string) *Instance {
	return &Instance{Class: evaluatorSQLiteValueClass, Fields: map[ir.FieldID]any{evaluatorSQLiteValueField: data}}
}

func (session *Session) sqliteHandleOf(value any) string {
	instance := session.requireInstance(value)
	handle, ok := instance.Fields[evaluatorSQLiteDatabaseField].(string)
	if !ok || instance.Class != evaluatorSQLiteDatabaseClass {
		session.raise("SQLiteError", "Database storage is corrupted")
	}
	return handle
}

func (session *Session) sqliteDataOf(value any) string {
	instance := session.requireInstance(value)
	data, ok := instance.Fields[evaluatorSQLiteValueField].(string)
	if !ok || instance.Class != evaluatorSQLiteValueClass {
		session.raise("SQLiteError", "SQLiteValue storage is corrupted")
	}
	return data
}

func (session *Session) sqliteCheck(err error) {
	if err != nil {
		session.raise("SQLiteError", err.Error())
	}
}

func (session *Session) sqliteParameters(args []any, index int) []string {
	if index >= len(args) || args[index] == nil {
		return nil
	}
	list := session.requireList(args[index])
	result := make([]string, len(list.Items))
	for position, item := range list.Items {
		result[position] = session.sqliteDataOf(item)
	}
	return result
}

func (session *Session) sqliteBuiltin(name string, args []any) any {
	switch name {
	case "open":
		path := args[0].(string)
		if path != ":memory:" {
			// The REPL resolves relative paths against its own session
			// directory, exactly like File and the other path-taking modules.
			path = session.sessionPath(path)
		}
		handle, err := ahdruntime.SQLiteOpen(path)
		session.sqliteCheck(err)
		return session.sqliteDatabase(handle)
	case "nullValue":
		return session.sqliteValue(ahdruntime.SQLiteNullValue())
	case "fromInt":
		return session.sqliteValue(ahdruntime.SQLiteFromInt(args[0].(int64)))
	case "fromReal":
		text, err := ahdruntime.SQLiteFromReal(args[0].(float64))
		session.sqliteCheck(err)
		return session.sqliteValue(text)
	case "fromString":
		return session.sqliteValue(ahdruntime.SQLiteFromString(args[0].(string)))
	}
	session.raise("Error", "unsupported SQLite function "+name)
	return nil
}

func (session *Session) sqliteOperation(name string, receiver any, args []any) any {
	switch name {
	case "Database.execute":
		changed, err := ahdruntime.SQLiteExecute(session.sqliteHandleOf(receiver), args[0].(string), session.sqliteParameters(args, 1))
		session.sqliteCheck(err)
		return changed
	case "Database.query":
		columns, rows, err := ahdruntime.SQLiteQuery(session.sqliteHandleOf(receiver), args[0].(string), session.sqliteParameters(args, 1))
		session.sqliteCheck(err)
		items := make([]any, len(rows))
		for index, row := range rows {
			pair := &Pair{Values: make(map[any]any, len(columns))}
			for column, text := range row {
				pairSet(pair, columns[column], session.sqliteValue(text))
			}
			items[index] = pair
		}
		return &List{Items: items}
	case "Database.lastInsertId":
		value, err := ahdruntime.SQLiteLastInsertID(session.sqliteHandleOf(receiver))
		session.sqliteCheck(err)
		return value
	case "Database.begin":
		session.sqliteCheck(ahdruntime.SQLiteBegin(session.sqliteHandleOf(receiver)))
		return Nothing
	case "Database.commit":
		session.sqliteCheck(ahdruntime.SQLiteCommit(session.sqliteHandleOf(receiver)))
		return Nothing
	case "Database.rollback":
		session.sqliteCheck(ahdruntime.SQLiteRollback(session.sqliteHandleOf(receiver)))
		return Nothing
	case "Database.close":
		session.sqliteCheck(ahdruntime.SQLiteClose(session.sqliteHandleOf(receiver)))
		return Nothing

	case "SQLiteValue.kind":
		kind, err := ahdruntime.SQLiteValueKind(session.sqliteDataOf(receiver))
		session.sqliteCheck(err)
		return kind
	case "SQLiteValue.isNull":
		isNull, err := ahdruntime.SQLiteValueIsNull(session.sqliteDataOf(receiver))
		session.sqliteCheck(err)
		return isNull
	case "SQLiteValue.int":
		value, err := ahdruntime.SQLiteValueInt(session.sqliteDataOf(receiver))
		session.sqliteCheck(err)
		return value
	case "SQLiteValue.real":
		value, err := ahdruntime.SQLiteValueReal(session.sqliteDataOf(receiver))
		session.sqliteCheck(err)
		return value
	case "SQLiteValue.string":
		value, err := ahdruntime.SQLiteValueString(session.sqliteDataOf(receiver))
		session.sqliteCheck(err)
		return value
	}
	session.raise("Error", "unsupported SQLite operation "+name)
	return nil
}
