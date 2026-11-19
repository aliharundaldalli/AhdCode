package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ncruces/go-sqlite3"
)

// historyTable is the only table migrate creates. It is not a session table
// and migrate never reads or writes ahd_session_* tables.
const historyTable = "ahd_schema_migrations"

const busyTimeout = 5 * time.Second

// interruptAfterScript, when set by a test, runs after the migration SQL and
// before the history change, still inside the open transaction. A non-nil
// error aborts the migration and rolls the transaction back.
var interruptAfterScript func(*sqlite3.Conn) error

// beforeImmediate, when set by a test, runs after the connection policy is
// restored and before BEGIN IMMEDIATE.
var beforeImmediate func(*sqlite3.Conn)

// panicDuringAuthorize, when set by a test, runs after the authorizer is
// installed. The authorizer is still cleared if it panics.
var panicDuringAuthorize func()

type appliedRow struct {
	Version      int64
	Name         string
	UpChecksum   string
	DownChecksum string
	AppliedAt    int64
}

type openMode int

const (
	openReadOnly openMode = iota
	openReadWrite
	openCreate
)

func openSQLite(path string, mode openMode) (*sqlite3.Conn, error) {
	if strings.TrimSpace(path) == "" || path == ":memory:" {
		return nil, errors.New("migrate needs a database file path")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("database path cannot be resolved: %s", path)
	}
	absolute = filepath.Clean(absolute)
	if err := ensurePathContained(absolute); err != nil {
		return nil, err
	}
	info, statErr := os.Lstat(absolute)
	if statErr == nil {
		reparse, err := componentReparse(absolute)
		if err != nil {
			return nil, fmt.Errorf("database path cannot be read: %s", absolute)
		}
		if reparse {
			return nil, fmt.Errorf("database path is a symbolic link or junction: %s", absolute)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("database path is a directory: %s", absolute)
		}
	} else if os.IsNotExist(statErr) {
		if mode != openCreate {
			return nil, errDatabaseMissing
		}
		parent := filepath.Dir(absolute)
		parentInfo, err := os.Lstat(parent)
		if err != nil || !parentInfo.IsDir() {
			return nil, fmt.Errorf("database directory does not exist: %s", parent)
		}
	} else {
		return nil, fmt.Errorf("database path cannot be read: %s", absolute)
	}
	// OPEN_URI is absent. The path is a filesystem path, never a file: URI.
	// backupTo is not used. status does not pass OPEN_CREATE. down passes
	// OPEN_READWRITE without OPEN_CREATE, so a missing file is not created.
	flags := sqlite3.OPEN_READONLY
	switch mode {
	case openReadWrite:
		flags = sqlite3.OPEN_READWRITE
	case openCreate:
		flags = sqlite3.OPEN_READWRITE | sqlite3.OPEN_CREATE
	}
	conn, err := sqlite3.OpenFlags(absolute, flags)
	if err != nil {
		if mode != openCreate {
			if _, statErr := os.Lstat(absolute); os.IsNotExist(statErr) {
				return nil, errDatabaseMissing
			}
		}
		return nil, fmt.Errorf("database cannot be opened: %s", err.Error())
	}
	if err := conn.BusyTimeout(busyTimeout); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("database cannot be opened: %s", err.Error())
	}
	return conn, nil
}

var errDatabaseMissing = errors.New("database file does not exist")

func (row appliedRow) matches(file File) error {
	if file.Name != row.Name {
		return fmt.Errorf("migration %d was applied as %s but the file is named %s", row.Version, row.Name, file.Name)
	}
	if file.UpChecksum != row.UpChecksum {
		return fmt.Errorf("migration %s changed after it was applied; the recorded checksum no longer matches the file", file.Label)
	}
	if file.DownChecksum != row.DownChecksum {
		return fmt.Errorf("the down script for applied migration %s changed after it was applied", file.Label)
	}
	return nil
}

func loadHistory(conn *sqlite3.Conn) ([]appliedRow, error) {
	exists, err := historyExists(conn)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	if err := checkHistoryShape(conn); err != nil {
		return nil, err
	}
	statement, _, err := conn.Prepare(`SELECT version, name, up_checksum, down_checksum, applied_at FROM main.ahd_schema_migrations ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer statement.Close()
	var rows []appliedRow
	for statement.Step() {
		row, err := readHistoryRow(statement)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if err := statement.Err(); err != nil {
		return nil, err
	}
	for index, row := range rows {
		if row.Version != int64(index+1) {
			return nil, fmt.Errorf("database migration history is inconsistent: version %d is present where version %d belongs", row.Version, index+1)
		}
	}
	return rows, nil
}

func historyExists(conn *sqlite3.Conn) (bool, error) {
	kind, err := historyKind(conn)
	if err != nil || kind == "" {
		return false, err
	}
	if kind != "table" {
		return false, fmt.Errorf("ahd_schema_migrations is a %s, not a table; migrate will not change it", kind)
	}
	return true, nil
}

func historyKind(conn *sqlite3.Conn) (string, error) {
	// SQLite folds unquoted ASCII identifiers and compares quoted ones
	// without regard to ASCII case. A different spelling is the same object.
	statement, _, err := conn.Prepare(`SELECT type FROM main.sqlite_master WHERE name = ? COLLATE NOCASE`)
	if err != nil {
		return "", err
	}
	defer statement.Close()
	if err := statement.BindText(1, historyTable); err != nil {
		return "", err
	}
	var kinds []string
	for statement.Step() {
		if statement.ColumnType(0) != sqlite3.TEXT {
			return "", fmt.Errorf("ahd_schema_migrations has an unreadable sqlite_master row; migrate will not change it")
		}
		kinds = append(kinds, statement.ColumnText(0))
	}
	if err := statement.Err(); err != nil {
		return "", err
	}
	switch len(kinds) {
	case 0:
		return "", nil
	case 1:
		return kinds[0], nil
	default:
		return "", fmt.Errorf("ahd_schema_migrations matches %d objects; migrate will not change it", len(kinds))
	}
}

func checkHistoryShape(conn *sqlite3.Conn) error {
	statement, _, err := conn.Prepare(`PRAGMA table_info(ahd_schema_migrations)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	type column struct {
		name, typ  string
		notNull    int64
		pk         int64
		hasDefault bool
	}
	var columns []column
	for statement.Step() {
		if statement.ColumnType(1) != sqlite3.TEXT || statement.ColumnType(2) != sqlite3.TEXT || statement.ColumnType(3) != sqlite3.INTEGER || statement.ColumnType(5) != sqlite3.INTEGER {
			return fmt.Errorf("ahd_schema_migrations has an unreadable column description; migrate will not change it")
		}
		columns = append(columns, column{
			name: statement.ColumnText(1), typ: statement.ColumnText(2),
			notNull: statement.ColumnInt64(3), pk: statement.ColumnInt64(5),
			hasDefault: statement.ColumnType(4) != sqlite3.NULL,
		})
	}
	if err := statement.Err(); err != nil {
		return err
	}
	expected := []column{
		{name: "version", typ: "INTEGER", pk: 1},
		{name: "name", typ: "TEXT", notNull: 1},
		{name: "up_checksum", typ: "TEXT", notNull: 1},
		{name: "down_checksum", typ: "TEXT", notNull: 1},
		{name: "applied_at", typ: "INTEGER", notNull: 1},
	}
	if len(columns) != len(expected) {
		return fmt.Errorf("ahd_schema_migrations has %d columns, expected %d; migrate will not change it", len(columns), len(expected))
	}
	for index, want := range expected {
		got := columns[index]
		if !strings.EqualFold(got.name, want.name) || !strings.EqualFold(got.typ, want.typ) || got.notNull != want.notNull || got.pk != want.pk || got.hasDefault {
			return fmt.Errorf("ahd_schema_migrations column %d is %s %s, expected %s %s; migrate will not change it", index+1, got.name, got.typ, want.name, want.typ)
		}
	}
	indexes, _, err := conn.Prepare(`PRAGMA index_list(ahd_schema_migrations)`)
	if err != nil {
		return err
	}
	defer indexes.Close()
	if indexes.Step() {
		return fmt.Errorf("ahd_schema_migrations has an unexpected index; migrate will not change it")
	}
	return indexes.Err()
}

func readHistoryRow(statement *sqlite3.Stmt) (appliedRow, error) {
	if statement.ColumnType(0) != sqlite3.INTEGER {
		return appliedRow{}, fmt.Errorf("ahd_schema_migrations version is %s, not an integer; migrate will not change it", statement.ColumnType(0))
	}
	if statement.ColumnType(1) != sqlite3.TEXT || statement.ColumnType(2) != sqlite3.TEXT || statement.ColumnType(3) != sqlite3.TEXT {
		return appliedRow{}, fmt.Errorf("ahd_schema_migrations has a NULL or non-text name or checksum; migrate will not change it")
	}
	if statement.ColumnType(4) != sqlite3.INTEGER {
		return appliedRow{}, fmt.Errorf("ahd_schema_migrations applied_at is %s, not an integer; migrate will not change it", statement.ColumnType(4))
	}
	row := appliedRow{
		Version: statement.ColumnInt64(0), Name: statement.ColumnText(1),
		UpChecksum: statement.ColumnText(2), DownChecksum: statement.ColumnText(3),
		AppliedAt: statement.ColumnInt64(4),
	}
	if row.Version < 1 {
		return appliedRow{}, fmt.Errorf("ahd_schema_migrations version %d is not a positive integer; migrate will not change it", row.Version)
	}
	if !validName(row.Name) {
		return appliedRow{}, fmt.Errorf("ahd_schema_migrations name %q is not a migration name; migrate will not change it", row.Name)
	}
	if !validChecksum(row.UpChecksum, false) || !validChecksum(row.DownChecksum, true) {
		return appliedRow{}, fmt.Errorf("ahd_schema_migrations checksum for version %d is malformed; migrate will not change it", row.Version)
	}
	if row.AppliedAt < 0 {
		return appliedRow{}, fmt.Errorf("ahd_schema_migrations applied_at for version %d is negative; migrate will not change it", row.Version)
	}
	return row, nil
}

func validChecksum(value string, allowEmpty bool) bool {
	if value == "" {
		return allowEmpty
	}
	if len(value) != 64 || utf8.RuneCountInString(value) != 64 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

func ensureHistory(conn *sqlite3.Conn) error {
	exists, err := historyExists(conn)
	if err != nil {
		return err
	}
	if exists {
		return checkHistoryShape(conn)
	}
	return conn.Exec(`CREATE TABLE main.ahd_schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		up_checksum TEXT NOT NULL,
		down_checksum TEXT NOT NULL,
		applied_at INTEGER NOT NULL
	)`)
}

func checkConsistent(files []File, rows []appliedRow) (pending []File, err error) {
	byVersion := map[int64]File{}
	for _, file := range files {
		byVersion[file.Version] = file
	}
	for _, row := range rows {
		file, ok := byVersion[row.Version]
		if !ok {
			return nil, fmt.Errorf("database has applied migration %d (%s), which is not in the migration directory", row.Version, row.Name)
		}
		if err := row.matches(file); err != nil {
			return nil, err
		}
	}
	if len(files) < len(rows) {
		return nil, fmt.Errorf("database migration history has %d version(s) but the directory has %d", len(rows), len(files))
	}
	return files[len(rows):], nil
}

func beginImmediate(conn *sqlite3.Conn) error {
	if !conn.GetAutocommit() {
		return errors.New("a migration transaction is already open")
	}
	// The same connection applies every version in one Up or Down. Restoring
	// the timeout here is what keeps an earlier migration from shortening it.
	if err := restoreConnectionPolicy(conn); err != nil {
		return fmt.Errorf("could not set the database lock timeout: %s", err.Error())
	}
	if beforeImmediate != nil {
		beforeImmediate(conn)
	}
	if err := conn.Exec("BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("could not lock the database: %s", err.Error())
	}
	return nil
}

// restoreConnectionPolicy puts back the connection settings a migration is
// not allowed to change. It runs with no authorizer, before the database lock.
func restoreConnectionPolicy(conn *sqlite3.Conn) error {
	if err := conn.BusyTimeout(busyTimeout); err != nil {
		return err
	}
	return conn.Exec(`
		PRAGMA locking_mode = NORMAL;
		PRAGMA trusted_schema = ON;
		PRAGMA recursive_triggers = OFF;
		PRAGMA foreign_keys = OFF;
		PRAGMA defer_foreign_keys = OFF;
		PRAGMA ignore_check_constraints = OFF;
		PRAGMA query_only = OFF;
		PRAGMA read_uncommitted = OFF;
		PRAGMA cell_size_check = OFF;
		PRAGMA legacy_alter_table = OFF;
		PRAGMA synchronous = FULL;
		PRAGMA wal_autocheckpoint = 1000;
		PRAGMA case_sensitive_like = OFF;
		PRAGMA secure_delete = OFF;
		PRAGMA reverse_unordered_selects = OFF;
		PRAGMA temp_store = DEFAULT;
	`)
}

func rollback(conn *sqlite3.Conn) {
	defer func() { _ = recover() }()
	if conn.GetAutocommit() {
		return
	}
	_ = conn.Exec("ROLLBACK")
}

// execScript gives the whole file to SQLite's own parser. Triggers, quotes,
// and semicolons inside text stay intact because this process does not split
// the script. The authorizer is installed only for this call and cleared
// before the migration manager writes ahd_schema_migrations.
func execScript(conn *sqlite3.Conn, sql, label string) (err error) {
	reason := ""
	if err := conn.SetAuthorizer(func(action sqlite3.AuthorizerActionCode, arg1, arg2, schema, inner string) sqlite3.AuthorizerReturnCode {
		code, why := authorizeMigration(action, arg1, arg2, schema, inner)
		if code != sqlite3.AUTH_OK && reason == "" {
			reason = why
		}
		return code
	}); err != nil {
		return fmt.Errorf("migration %s could not be authorized: %s", label, err.Error())
	}
	defer func() {
		if clearErr := conn.SetAuthorizer(nil); clearErr != nil && err == nil {
			err = fmt.Errorf("migration %s could not restore the authorizer: %s", label, clearErr.Error())
		}
	}()
	if panicDuringAuthorize != nil {
		panicDuringAuthorize()
	}
	if err := requireExecutable(conn, sql); err != nil {
		if reason != "" {
			return fmt.Errorf("migration %s was rejected (%s): %s", label, reason, err.Error())
		}
		return fmt.Errorf("migration %s failed: %s", label, err.Error())
	}
	if err := conn.Exec(sql); err != nil {
		if reason != "" {
			return fmt.Errorf("migration %s was rejected (%s)", label, reason)
		}
		return fmt.Errorf("migration %s failed: %s", label, err.Error())
	}
	// Creating a trigger does not authorize its body. Compile every persistent
	// trigger under this authorizer before the history row can commit.
	if err := compileTriggers(conn, &reason); err != nil || reason != "" {
		if reason != "" {
			return fmt.Errorf("migration %s was rejected (%s)", label, reason)
		}
		return fmt.Errorf("migration %s was rejected (%s)", label, err.Error())
	}
	if conn.GetAutocommit() {
		return fmt.Errorf("migration %s ended the SQLite transaction and was not recorded. Inspect the database before trying again", label)
	}
	return nil
}

func requireExecutable(conn *sqlite3.Conn, sql string) error {
	// Only the first statement is prepared here. Later statements may depend
	// on tables created by earlier ones, so they are prepared by SQLite while
	// the script runs, still under the authorizer.
	statement, _, err := conn.Prepare(sql)
	if err != nil {
		return err
	}
	if statement == nil {
		return fmt.Errorf("empty of executable SQL")
	}
	statement.Close()
	return nil
}

func insertHistory(conn *sqlite3.Conn, file File, appliedAt int64) error {
	statement, _, err := conn.Prepare(`INSERT INTO main.ahd_schema_migrations (version, name, up_checksum, down_checksum, applied_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	binds := []struct {
		text bool
		n    int64
		s    string
	}{
		{n: file.Version},
		{text: true, s: file.Name},
		{text: true, s: file.UpChecksum},
		{text: true, s: file.DownChecksum},
		{n: appliedAt},
	}
	for index, bind := range binds {
		var err error
		if bind.text {
			err = statement.BindText(index+1, bind.s)
		} else {
			err = statement.BindInt64(index+1, bind.n)
		}
		if err != nil {
			return err
		}
	}
	if err := statement.Exec(); err != nil {
		return fmt.Errorf("could not record migration %s: %s", file.Label, err.Error())
	}
	return nil
}

func deleteHistory(conn *sqlite3.Conn, version int64) error {
	statement, _, err := conn.Prepare(`DELETE FROM main.ahd_schema_migrations WHERE version = ?`)
	if err != nil {
		return err
	}
	defer statement.Close()
	if err := statement.BindInt64(1, version); err != nil {
		return err
	}
	if err := statement.Exec(); err != nil {
		return err
	}
	changes := conn.Changes()
	if changes != 1 {
		return fmt.Errorf("migration history for version %d could not be removed", version)
	}
	return nil
}
