package migrate

import (
	"strings"

	"github.com/ncruces/go-sqlite3"
)

// authorizeMigration is installed only while user migration SQL runs.
// Trusted history statements run with no authorizer. SAVEPOINT, ROLLBACK TO,
// and RELEASE stay available because they do not end the outer transaction.
func authorizeMigration(action sqlite3.AuthorizerActionCode, arg1, arg2, schema, inner string) (sqlite3.AuthorizerReturnCode, string) {
	switch action {
	case sqlite3.AUTH_TRANSACTION:
		return sqlite3.AUTH_DENY, "BEGIN, COMMIT, and ROLLBACK are not allowed inside a migration"
	case sqlite3.AUTH_ATTACH, sqlite3.AUTH_DETACH:
		return sqlite3.AUTH_DENY, "ATTACH and DETACH are not allowed inside a migration"
	case sqlite3.AUTH_PRAGMA:
		if reason, denied := deniedPragma(arg1); denied {
			return sqlite3.AUTH_DENY, reason
		}
	}
	// CREATE TABLE and other ordinary DDL insert into sqlite_master through a
	// nested statement. Denying that insert would reject every migration.
	// Direct edits of sqlite_master still require PRAGMA writable_schema,
	// which is denied above.
	if actionWrites(action) && (protectedObject(arg1) || protectedObject(arg2) || protectedObject(inner)) {
		return sqlite3.AUTH_DENY, "ahd_schema_migrations cannot be changed by a migration"
	}
	return sqlite3.AUTH_OK, ""
}

func actionWrites(action sqlite3.AuthorizerActionCode) bool {
	switch action {
	case sqlite3.AUTH_INSERT, sqlite3.AUTH_UPDATE, sqlite3.AUTH_DELETE,
		sqlite3.AUTH_DROP_TABLE, sqlite3.AUTH_DROP_INDEX, sqlite3.AUTH_DROP_TRIGGER, sqlite3.AUTH_DROP_VIEW, sqlite3.AUTH_DROP_VTABLE,
		sqlite3.AUTH_DROP_TEMP_TABLE, sqlite3.AUTH_DROP_TEMP_INDEX, sqlite3.AUTH_DROP_TEMP_TRIGGER, sqlite3.AUTH_DROP_TEMP_VIEW,
		sqlite3.AUTH_ALTER_TABLE,
		sqlite3.AUTH_CREATE_INDEX, sqlite3.AUTH_CREATE_TABLE, sqlite3.AUTH_CREATE_TRIGGER, sqlite3.AUTH_CREATE_VIEW, sqlite3.AUTH_CREATE_VTABLE,
		sqlite3.AUTH_CREATE_TEMP_INDEX, sqlite3.AUTH_CREATE_TEMP_TABLE, sqlite3.AUTH_CREATE_TEMP_TRIGGER, sqlite3.AUTH_CREATE_TEMP_VIEW,
		sqlite3.AUTH_REINDEX, sqlite3.AUTH_ANALYZE:
		return true
	default:
		return false
	}
}

func protectedObject(name string) bool {
	return strings.EqualFold(name, historyTable)
}

// deniedPragma rejects connection settings that would change authorization,
// locking, or integrity for a later migration on the same connection.
// Schema queries such as table_info stay allowed.
func deniedPragma(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "writable_schema", "journal_mode", "wal_checkpoint", "incremental_vacuum", "schema_version",
		"busy_timeout", "locking_mode", "trusted_schema", "recursive_triggers", "foreign_keys",
		"defer_foreign_keys", "ignore_check_constraints", "query_only", "read_uncommitted",
		"cell_size_check", "legacy_alter_table", "synchronous",
		"wal_autocheckpoint", "case_sensitive_like", "secure_delete",
		"reverse_unordered_selects", "temp_store":
		return "PRAGMA " + name + " is not allowed inside a migration", true
	default:
		return "", false
	}
}
