package ahdruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSQLiteOpenWithOptionsReadOnlyAndValidation(t *testing.T) {
	buildSQLiteHelper(t)
	directory := t.TempDir()

	// Invalid option combinations fail before opening or creating anything.
	if _, err := SQLiteOpenWithOptions("", false, 1000); err == nil {
		t.Fatal("empty path accepted")
	}
	if _, err := SQLiteOpenWithOptions(":memory:", true, 0); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("read-only :memory: error = %v", err)
	}
	if _, err := SQLiteOpenWithOptions(":memory:", false, -1); err == nil || !strings.Contains(err.Error(), "busyTimeoutMilliseconds") {
		t.Fatalf("negative busy timeout error = %v", err)
	}
	if _, err := SQLiteOpenWithOptions(":memory:", false, ahdSQLiteMaxBusyTimeoutMilliseconds+1); err == nil || !strings.Contains(err.Error(), "busyTimeoutMilliseconds") {
		t.Fatalf("excessive busy timeout error = %v", err)
	}

	// Read-only open on a nonexistent file must fail and never create the file.
	missing := filepath.Join(directory, "does-not-exist.db")
	if _, err := SQLiteOpenWithOptions(missing, true, 1000); err == nil {
		t.Fatal("read-only open of nonexistent file succeeded")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("read-only open created missing file: %v", err)
	}

	// Seed an existing database via read-write openWithOptions.
	dbPath := filepath.Join(directory, "notes ü #1%.db")
	rw, err := SQLiteOpenWithOptions(dbPath, false, 1000)
	if err != nil {
		t.Fatalf("read-write openWithOptions: %v", err)
	}
	if _, err := SQLiteExecute(rw, "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL)", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := SQLiteExecute(rw, "INSERT INTO notes (body) VALUES (?)", []string{SQLiteFromString("committed")}); err != nil {
		t.Fatal(err)
	}
	if err := SQLiteClose(rw); err != nil {
		t.Fatal(err)
	}

	// Open the seeded database in read-only mode.
	ro, err := SQLiteOpenWithOptions(dbPath, true, 500)
	if err != nil {
		t.Fatalf("read-only openWithOptions: %v", err)
	}
	defer SQLiteClose(ro)

	_, rows, err := SQLiteQuery(ro, "SELECT body FROM notes", nil)
	if err != nil {
		t.Fatalf("read-only query: %v", err)
	}
	if len(rows) != 1 || rows[0][0] != SQLiteFromString("committed") {
		t.Fatalf("read-only rows = %v", rows)
	}

	// Writes through a read-only handle must fail.
	if _, err := SQLiteExecute(ro, "INSERT INTO notes (body) VALUES (?)", []string{SQLiteFromString("blocked")}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "readonly") {
		t.Fatalf("read-only insert error = %v", err)
	}
	if _, err := SQLiteExecute(ro, "CREATE TABLE illegal (id INTEGER)", nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "readonly") {
		t.Fatalf("read-only DDL error = %v", err)
	}

	// Backup from a read-only connection still succeeds and produces a valid snapshot.
	backupPath := filepath.Join(directory, "ro backup #2%.db")
	if err := SQLiteBackup(ro, backupPath); err != nil {
		t.Fatalf("backup from read-only handle: %v", err)
	}
	restored, err := SQLiteOpenWithOptions(backupPath, true, 0)
	if err != nil {
		t.Fatalf("open restored backup read-only: %v", err)
	}
	defer SQLiteClose(restored)
	if got := sqliteBackupScalar(t, restored, "SELECT body FROM notes"); got != "committed" {
		t.Fatalf("restored body = %q", got)
	}
	if got := sqliteBackupScalar(t, restored, "PRAGMA integrity_check"); got != "ok" {
		t.Fatalf("restored integrity = %q", got)
	}
}

func TestSQLiteOpenWithOptionsBusyTimeoutContention(t *testing.T) {
	buildSQLiteHelper(t)
	directory := t.TempDir()
	dbPath := filepath.Join(directory, "contention.db")

	writer, err := SQLiteOpen(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer SQLiteClose(writer)
	if _, err := SQLiteExecute(writer, "CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)", nil); err != nil {
		t.Fatal(err)
	}

	// Hold an exclusive write lock on writer.
	if err := SQLiteBegin(writer); err != nil {
		t.Fatal(err)
	}
	if _, err := SQLiteExecute(writer, "INSERT INTO items (name) VALUES (?)", []string{SQLiteFromString("first")}); err != nil {
		t.Fatal(err)
	}

	// A second connection with busyTimeoutMilliseconds=0 fails immediately without waiting.
	noWait, err := SQLiteOpenWithOptions(dbPath, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer SQLiteClose(noWait)
	start := time.Now()
	_, lockErr := SQLiteExecute(noWait, "INSERT INTO items (name) VALUES (?)", []string{SQLiteFromString("zero-wait")})
	elapsed := time.Since(start)
	if lockErr == nil || !strings.Contains(strings.ToLower(lockErr.Error()), "locked") {
		t.Fatalf("zero-timeout write error = %v; want locked error", lockErr)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("zero-timeout write waited %v; expected immediate failure", elapsed)
	}

	// Commit writer and verify a bounded-timeout connection writes cleanly.
	if err := SQLiteCommit(writer); err != nil {
		t.Fatal(err)
	}
	bounded, err := SQLiteOpenWithOptions(dbPath, false, 1500)
	if err != nil {
		t.Fatal(err)
	}
	defer SQLiteClose(bounded)
	if _, err := SQLiteExecute(bounded, "INSERT INTO items (name) VALUES (?)", []string{SQLiteFromString("after-commit")}); err != nil {
		t.Fatalf("bounded write after commit: %v", err)
	}
}

func TestSQLiteOpenWithOptionsOldHelperVersionError(t *testing.T) {
	// An older ahdsqlite helper only checks OperationOpen before looking up
	// request.Database (which is 0 for openWithOptions), so it fails with
	// "unknown Database handle". Verify that calling an unrecognized top-level
	// operation fails explicitly rather than silently opening read-write.
	buildSQLiteHelper(t)
	_, err := ahdSQLiteSession.call(ahdSQLiteRequest{Operation: "nonexistentPhase2Op"})
	if err == nil || !strings.Contains(err.Error(), "unknown Database handle") {
		t.Fatalf("unexpected error for unknown top-level operation: %v", err)
	}
}
