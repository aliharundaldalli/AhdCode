package ahdruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sqliteBackupScalar(t *testing.T, handle, sql string) string {
	t.Helper()
	_, rows, err := SQLiteQuery(handle, sql, nil)
	if err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	if len(rows) != 1 || len(rows[0]) != 1 {
		t.Fatalf("%s: rows = %v", sql, rows)
	}
	return rows[0][0][1:] // strip the kind byte
}

func sqliteBackupExec(t *testing.T, handle, sql string) {
	t.Helper()
	if _, err := SQLiteExecute(handle, sql, nil); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func requireNoBackupLeftovers(t *testing.T, directory string) {
	t.Helper()
	entries, _ := os.ReadDir(directory)
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".ahdbackup-") {
			t.Fatalf("temporary backup left behind: %s", entry.Name())
		}
	}
}

// TestSQLiteBackupIsAConsistentLiveSnapshot covers the v2.5.0 contract: a
// backup of an open WAL database while another connection holds uncommitted
// writes is a valid, self-contained file holding exactly the committed state
// at backup time, and later changes to the source never reach it.
func TestSQLiteBackupIsAConsistentLiveSnapshot(t *testing.T) {
	buildSQLiteHelper(t)
	directory := t.TempDir()
	source := filepath.Join(directory, "panel.db")
	db, err := SQLiteOpen(source)
	if err != nil {
		t.Fatal(err)
	}
	defer SQLiteClose(db)
	if mode := sqliteBackupScalar(t, db, "PRAGMA journal_mode=WAL"); mode != "wal" {
		t.Fatalf("journal mode = %q", mode)
	}
	sqliteBackupExec(t, db, "CREATE TABLE apps (id INTEGER PRIMARY KEY, name TEXT NOT NULL)")
	sqliteBackupExec(t, db, "CREATE TABLE audit (id INTEGER PRIMARY KEY, action TEXT NOT NULL)")
	for _, name := range []string{"ata", "panel", "chat"} {
		if _, err := SQLiteExecute(db, "INSERT INTO apps (name) VALUES (?)", []string{SQLiteFromString(name)}); err != nil {
			t.Fatal(err)
		}
	}
	sqliteBackupExec(t, db, "INSERT INTO audit (action) VALUES ('deploy')")

	// A second connection holds an uncommitted write during the backup.
	writer, err := SQLiteOpen(source)
	if err != nil {
		t.Fatal(err)
	}
	defer SQLiteClose(writer)
	if err := SQLiteBegin(writer); err != nil {
		t.Fatal(err)
	}
	sqliteBackupExec(t, writer, "INSERT INTO apps (name) VALUES ('uncommitted')")

	backup := filepath.Join(directory, "backups", "panel-1.db")
	if err := os.Mkdir(filepath.Dir(backup), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SQLiteBackup(db, backup); err != nil {
		t.Fatal(err)
	}
	if err := SQLiteRollback(writer); err != nil {
		t.Fatal(err)
	}

	// The source keeps working and changes after the snapshot.
	sqliteBackupExec(t, db, "INSERT INTO apps (name) VALUES ('after')")
	sqliteBackupExec(t, db, "DELETE FROM audit")

	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat(backup + suffix); err == nil {
			t.Fatalf("backup is not self-contained: %s exists", backup+suffix)
		}
	}
	copyHandle, err := SQLiteOpen(backup)
	if err != nil {
		t.Fatal(err)
	}
	defer SQLiteClose(copyHandle)
	if got := sqliteBackupScalar(t, copyHandle, "SELECT group_concat(name, ',') FROM (SELECT name FROM apps ORDER BY id)"); got != "ata,panel,chat" {
		t.Fatalf("backup apps = %q", got)
	}
	if got := sqliteBackupScalar(t, copyHandle, "SELECT count(*) FROM audit"); got != "1" {
		t.Fatalf("backup audit rows = %q", got)
	}
	if got := sqliteBackupScalar(t, copyHandle, "PRAGMA integrity_check"); got != "ok" {
		t.Fatalf("integrity = %q", got)
	}
	if got := sqliteBackupScalar(t, copyHandle, "PRAGMA journal_mode"); got != "delete" {
		t.Fatalf("backup journal mode = %q", got)
	}
	if got := sqliteBackupScalar(t, db, "SELECT count(*) FROM apps"); got != "4" {
		t.Fatalf("source apps after backup = %q", got)
	}

	// Repeated backup to the same path never overwrites; a new name works.
	if err := SQLiteBackup(db, backup); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second backup to the same path: %v", err)
	}
	second := filepath.Join(directory, "backups", "panel-2.db")
	if err := SQLiteBackup(db, second); err != nil {
		t.Fatal(err)
	}
	requireNoBackupLeftovers(t, filepath.Dir(backup))
}

func TestSQLiteBackupFailuresLeaveNoBackup(t *testing.T) {
	buildSQLiteHelper(t)
	directory := t.TempDir()
	db, err := SQLiteOpen(filepath.Join(directory, "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer SQLiteClose(db)
	sqliteBackupExec(t, db, "CREATE TABLE t (x INTEGER)")

	if err := SQLiteBackup(db, filepath.Join(directory, "missing", "b.db")); err == nil || !strings.Contains(err.Error(), "backup directory") {
		t.Fatalf("missing directory: %v", err)
	}
	if err := SQLiteBackup(db, ""); err == nil {
		t.Fatal("empty path accepted")
	}
	if err := SQLiteBackup(db, ":memory:"); err == nil {
		t.Fatal(":memory: destination accepted")
	}
	if err := SQLiteBegin(db); err != nil {
		t.Fatal(err)
	}
	inTransaction := filepath.Join(directory, "tx.db")
	if err := SQLiteBackup(db, inTransaction); err == nil || !strings.Contains(err.Error(), "transaction is active") {
		t.Fatalf("backup inside a transaction: %v", err)
	}
	if err := SQLiteRollback(db); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(inTransaction); err == nil {
		t.Fatal("a refused backup created its destination")
	}
	requireNoBackupLeftovers(t, directory)

	// An in-memory database can be backed up to a file.
	memory, err := SQLiteOpen(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer SQLiteClose(memory)
	sqliteBackupExec(t, memory, "CREATE TABLE m (v TEXT)")
	sqliteBackupExec(t, memory, "INSERT INTO m VALUES ('bellek')")
	fromMemory := filepath.Join(directory, "memory.db")
	if err := SQLiteBackup(memory, fromMemory); err != nil {
		t.Fatal(err)
	}
	check, err := SQLiteOpen(fromMemory)
	if err != nil {
		t.Fatal(err)
	}
	defer SQLiteClose(check)
	if got := sqliteBackupScalar(t, check, "SELECT v FROM m"); got != "bellek" {
		t.Fatalf("memory backup = %q", got)
	}
	// A path with URI-significant characters is a plain file name.
	odd := filepath.Join(directory, "odd name?#%.db")
	if err := SQLiteBackup(db, odd); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(odd); err != nil {
		t.Fatalf("odd path not created verbatim: %v", err)
	}
	expectRaise(t, AhdClassSQLiteError, func() { AhdSQLiteBackup(AhdClassSQLiteError, db, odd) })
}
