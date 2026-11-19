package migrate

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func waitForFile(path string, limit time.Duration) error {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", path)
}

func waitProcess(t *testing.T, command *exec.Cmd, output *bytes.Buffer, limit time.Duration) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("exit: %v\n%s", err, output.String())
		}
	case <-time.After(limit):
		t.Fatalf("timed out\n%s", output.String())
	}
}

func writePair(t *testing.T, directory, version, name, up, down string) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	base := version + "_" + name
	if err := os.WriteFile(filepath.Join(directory, base+".up.sql"), []byte(up), 0o644); err != nil {
		t.Fatal(err)
	}
	if down != "" {
		if err := os.WriteFile(filepath.Join(directory, base+".down.sql"), []byte(down), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func countHistory(t *testing.T, database string) int {
	t.Helper()
	conn, err := sqlite3.Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	statement, _, err := conn.Prepare(`SELECT COUNT(*) FROM ahd_schema_migrations`)
	if err != nil {
		t.Fatal(err)
	}
	defer statement.Close()
	if !statement.Step() {
		t.Fatal(statement.Err())
	}
	return int(statement.ColumnInt64(0))
}

func tableExists(t *testing.T, database, name string) bool {
	t.Helper()
	conn, err := sqlite3.Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	statement, _, err := conn.Prepare(`SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?`)
	if err != nil {
		t.Fatal(err)
	}
	defer statement.Close()
	if err := statement.BindText(1, name); err != nil {
		t.Fatal(err)
	}
	return statement.Step()
}

func noteBody(t *testing.T, database string) string {
	t.Helper()
	conn, err := sqlite3.Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	statement, _, err := conn.Prepare(`SELECT body FROM notes WHERE id = 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer statement.Close()
	if !statement.Step() {
		t.Fatalf("notes row missing: %v", statement.Err())
	}
	return statement.ColumnText(0)
}

func TestMigrateUpgradesAndDoesNotRepeat(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "Göç Klasörü")
	database := filepath.Join(directory, "uygulama veri.db")
	writePair(t, directory, "0001", "create_notes",
		"CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL);\nINSERT INTO notes (id, body) VALUES (1, 'ilk');\n",
		"DROP TABLE notes;\n")
	first, err := Up(Options{Database: database, Directory: directory})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].Label != "0001_create_notes" {
		t.Fatalf("first = %+v", first)
	}
	if noteBody(t, database) != "ilk" {
		t.Fatal(noteBody(t, database))
	}
	writePair(t, directory, "0002", "add_author",
		"ALTER TABLE notes ADD COLUMN author TEXT NOT NULL DEFAULT 'Ali';\nUPDATE notes SET body = 'ikinci' WHERE id = 1;\n",
		"ALTER TABLE notes DROP COLUMN author;\nUPDATE notes SET body = 'ilk' WHERE id = 1;\n")
	second, err := Up(Options{Database: database, Directory: directory})
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].Name != "add_author" {
		t.Fatalf("second = %+v", second)
	}
	if noteBody(t, database) != "ikinci" {
		t.Fatalf("body = %s", noteBody(t, database))
	}
	again, err := Up(Options{Database: database, Directory: directory})
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 || countHistory(t, database) != 2 {
		t.Fatalf("again %+v history %d", again, countHistory(t, database))
	}
	absoluteDir, _ := filepath.Abs(directory)
	absoluteDB, _ := filepath.Abs(database)
	relativeDir, _ := filepath.Rel(".", absoluteDir) // may be absolute if different volume
	status, err := Inspect(Options{Database: absoluteDB, Directory: absoluteDir})
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Applied) != 2 || len(status.Pending) != 0 {
		t.Fatalf("status %+v pending %+v", status.Applied, status.Pending)
	}
	rolled, err := Down(Options{Database: database, Directory: directory}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rolled) != 1 || rolled[0].Label != "0002_add_author" || countHistory(t, database) != 1 {
		t.Fatalf("rolled %+v", rolled)
	}
	if noteBody(t, database) != "ilk" {
		t.Fatalf("after down, body = %s", noteBody(t, database))
	}
	_ = relativeDir
}

func TestMigrateFailedChangeIsNotRecorded(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "broken",
		"CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL);\nINSERT INTO notes (id, body) VALUES (1, 'x');\nSELECT * FROM missing_table;\n",
		"")
	_, err := Up(Options{Database: database, Directory: directory})
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Fatalf("error = %v", err)
	}
	if tableExists(t, database, "notes") || tableExists(t, database, historyTable) {
		t.Fatal("a failed migration left tables behind")
	}
	if err := os.WriteFile(filepath.Join(directory, "0001_broken.up.sql"), []byte("CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	applied, err := Up(Options{Database: database, Directory: directory})
	if err != nil || len(applied) != 1 || !tableExists(t, database, "notes") {
		t.Fatalf("recovery %+v %v", applied, err)
	}
}

func TestMigrateDetectsAChangedChecksum(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "DROP TABLE notes;\n")
	if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "0001_create_notes.up.sql"), []byte("CREATE TABLE notes (id INTEGER PRIMARY KEY);\n-- edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writePair(t, directory, "0002", "next", "CREATE TABLE extra (id INTEGER PRIMARY KEY);\n", "")
	_, err := Up(Options{Database: database, Directory: directory})
	if err == nil || !strings.Contains(err.Error(), "changed after it was applied") {
		t.Fatalf("error = %v", err)
	}
	if tableExists(t, database, "extra") {
		t.Fatal("a later migration ran after a checksum mismatch")
	}
	_, err = Inspect(Options{Database: database, Directory: directory})
	if err == nil || !strings.Contains(err.Error(), "changed after it was applied") {
		t.Fatalf("status error = %v", err)
	}
}

func TestMigrateRejectsBadFilesAndHistory(t *testing.T) {
	root := t.TempDir()
	if _, err := Up(Options{Database: filepath.Join(root, "app.db"), Directory: filepath.Join(root, "missing")}); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("missing directory: %v", err)
	}
	directory := filepath.Join(root, "migrations")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "notes.sql"), []byte("CREATE TABLE notes (id INTEGER);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Scan(directory); err == nil || !strings.Contains(err.Error(), "invalid migration file name") {
		t.Fatalf("invalid name: %v", err)
	}
	os.Remove(filepath.Join(directory, "notes.sql"))
	writePair(t, directory, "0001", "create_notes", "   \n", "")
	if _, err := Up(Options{Database: filepath.Join(root, "app.db"), Directory: directory}); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty: %v", err)
	}
	os.Remove(filepath.Join(directory, "0001_create_notes.up.sql"))
	writePair(t, directory, "0001", "one", "CREATE TABLE one (id INTEGER);\n", "")
	writePair(t, directory, "0001", "two", "CREATE TABLE two (id INTEGER);\n", "")
	if _, err := Scan(directory); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate: %v", err)
	}
	os.Remove(filepath.Join(directory, "0001_two.up.sql"))
	writePair(t, directory, "0003", "skipped", "CREATE TABLE three (id INTEGER);\n", "")
	if _, err := Scan(directory); err == nil || !strings.Contains(err.Error(), "skip 2") {
		t.Fatalf("gap: %v", err)
	}
	clean := filepath.Join(root, "fresh")
	if _, err := Create(clean, "Bad Name"); err == nil {
		t.Fatal("invalid migration name was accepted")
	}
	if _, err := Up(Options{Database: filepath.Join(root, "not-a-file"), Directory: directory, Driver: "mysql"}); err == nil || !strings.Contains(err.Error(), "MySQL") {
		t.Fatalf("mysql: %v", err)
	}
}

func TestMigrateRejectsUnknownHistory(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "DROP TABLE notes;\n")
	if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(directory, "0001_create_notes.up.sql"))
	os.Remove(filepath.Join(directory, "0001_create_notes.down.sql"))
	writePair(t, directory, "0001", "other", "CREATE TABLE other (id INTEGER);\n", "")
	_, err := Up(Options{Database: database, Directory: directory})
	if err == nil || !strings.Contains(err.Error(), "applied as create_notes") {
		t.Fatalf("error = %v", err)
	}
}

func TestMigrateDirectoryDatabaseIsRejected(t *testing.T) {
	directory := t.TempDir()
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER);\n", "")
	_, err := Up(Options{Database: directory, Directory: directory})
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("error = %v", err)
	}
}

func TestMigrateInterruptedProcessRecovers(t *testing.T) {
	if os.Getenv("AHD_MIGRATE_INTERRUPT") == "1" {
		interruptAfterScript = func(*sqlite3.Conn) error {
			if err := os.WriteFile(os.Getenv("AHD_MIGRATE_MARKER"), []byte("in transaction"), 0o644); err != nil {
				return err
			}
			time.Sleep(time.Minute)
			return errors.New("migration interrupted")
		}
		_, err := Up(Options{Database: os.Getenv("AHD_MIGRATE_DB"), Directory: os.Getenv("AHD_MIGRATE_DIR")})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	marker := filepath.Join(directory, "in-transaction")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT);\nINSERT INTO notes (id, body) VALUES (1, 'x');\n", "")
	command := exec.Command(os.Args[0], "-test.run=^TestMigrateInterruptedProcessRecovers$")
	command.Env = append(os.Environ(), "AHD_MIGRATE_INTERRUPT=1", "AHD_MIGRATE_DB="+database, "AHD_MIGRATE_DIR="+directory, "AHD_MIGRATE_MARKER="+marker)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
			_, _ = command.Process.Wait()
		}
	})
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("the child did not reach the open transaction")
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_, _ = command.Process.Wait()
	if tableExists(t, database, "notes") {
		t.Fatal("killing the process committed the migration")
	}
	applied, err := Up(Options{Database: database, Directory: directory})
	if err != nil || len(applied) != 1 || countHistory(t, database) != 1 || noteBody(t, database) != "x" {
		t.Fatalf("recovery %+v %v", applied, err)
	}
}

func TestMigrateConcurrentUpAppliesOnce(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\nINSERT INTO notes (id) VALUES (1);\n", "")
	var wait sync.WaitGroup
	errors := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := Up(Options{Database: database, Directory: directory})
			errors <- err
		}()
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if countHistory(t, database) != 1 {
		t.Fatalf("history = %d", countHistory(t, database))
	}
}

func TestMigrateTwoProcesses(t *testing.T) {
	if os.Getenv("AHD_MIGRATE_CHILD") == "1" {
		release := os.Getenv("AHD_MIGRATE_RELEASE")
		if os.Getenv("AHD_MIGRATE_HOLDER") == "1" {
			interruptAfterScript = func(*sqlite3.Conn) error {
				if err := os.WriteFile(os.Getenv("AHD_MIGRATE_ENTERED"), []byte("held"), 0o644); err != nil {
					return err
				}
				deadline := time.Now().Add(30 * time.Second)
				for time.Now().Before(deadline) {
					if _, err := os.Stat(release); err == nil {
						return nil
					}
					time.Sleep(10 * time.Millisecond)
				}
				return errors.New("the holder was not released")
			}
		} else {
			beforeImmediate = func(*sqlite3.Conn) {
				_ = os.WriteFile(os.Getenv("AHD_MIGRATE_WAITING"), []byte("waiting"), 0o644)
			}
		}
		_, err := Up(Options{Database: os.Getenv("AHD_MIGRATE_DB"), Directory: os.Getenv("AHD_MIGRATE_DIR")})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "DROP TABLE notes;\n")
	entered := filepath.Join(directory, "entered")
	release := filepath.Join(directory, "release")
	waiting := filepath.Join(directory, "waiting")
	start := func(holder bool) (*exec.Cmd, *bytes.Buffer) {
		var output bytes.Buffer
		command := exec.Command(os.Args[0], "-test.run=^TestMigrateTwoProcesses$")
		env := append(os.Environ(), "AHD_MIGRATE_CHILD=1", "AHD_MIGRATE_DB="+database, "AHD_MIGRATE_DIR="+directory, "AHD_MIGRATE_ENTERED="+entered, "AHD_MIGRATE_RELEASE="+release, "AHD_MIGRATE_WAITING="+waiting)
		if holder {
			env = append(env, "AHD_MIGRATE_HOLDER=1")
		}
		command.Env = env
		command.Stdout = &output
		command.Stderr = &output
		return command, &output
	}
	holder, holderOut := start(true)
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.WriteFile(release, []byte("go"), 0o644)
		if holder.Process != nil {
			_ = holder.Process.Kill()
		}
	})
	if err := waitForFile(entered, 20*time.Second); err != nil {
		t.Fatalf("holder did not enter the transaction: %v\n%s", err, holderOut.String())
	}
	if tableExists(t, database, historyTable) {
		t.Fatal("history was visible before the holder was released")
	}
	waiter, waiterOut := start(false)
	if err := waiter.Start(); err != nil {
		t.Fatal(err)
	}
	waiterDone := make(chan error, 1)
	go func() { waiterDone <- waiter.Wait() }()
	t.Cleanup(func() {
		if waiter.Process != nil {
			_ = waiter.Process.Kill()
		}
	})
	if err := waitForFile(waiting, 20*time.Second); err != nil {
		t.Fatalf("waiter did not reach the migration lock: %v\nwaiter: %s\nholder: %s", err, waiterOut.String(), holderOut.String())
	}
	select {
	case err := <-waiterDone:
		t.Fatalf("waiter finished before the holder released the transaction: %v\n%s", err, waiterOut.String())
	case <-time.After(200 * time.Millisecond):
	}
	if tableExists(t, database, historyTable) {
		t.Fatal("history was visible while the holder still had the transaction")
	}
	if err := os.WriteFile(release, []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitProcess(t, holder, holderOut, 20*time.Second)
	select {
	case err := <-waiterDone:
		if err != nil {
			t.Fatalf("waiter: %v\n%s", err, waiterOut.String())
		}
	case <-time.After(20 * time.Second):
		t.Fatalf("waiter did not exit\n%s", waiterOut.String())
	}
	if countHistory(t, database) != 1 || !tableExists(t, database, "notes") {
		t.Fatalf("history %d table %v", countHistory(t, database), tableExists(t, database, "notes"))
	}
}

func TestMigrateRelativePaths(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	directory := filepath.Join("migrations", "Göç Klasörü")
	if _, err := Create(directory, "create_notes"); err != nil {
		t.Fatal(err)
	}
	up := filepath.Join(directory, "0001_create_notes.up.sql")
	if err := os.WriteFile(up, []byte("CREATE TABLE notes (id INTEGER PRIMARY KEY);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Up(Options{Database: filepath.Join("data", "uygulama veri.db"), Directory: directory}); err == nil || !strings.Contains(err.Error(), "database directory does not exist") {
		t.Fatalf("missing parent: %v", err)
	}
	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatal(err)
	}
	applied, err := Up(Options{Database: filepath.ToSlash(filepath.Join("data", "uygulama veri.db")), Directory: directory})
	if err != nil || len(applied) != 1 {
		t.Fatalf("%+v %v", applied, err)
	}
	if !tableExists(t, filepath.Join(root, "data", "uygulama veri.db"), "notes") {
		t.Fatal("relative path did not create the table")
	}
}

func TestMigrateDownRefusesAnUnwrittenScript(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	file, err := Create(directory, "create_notes")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file.UpPath, []byte("CREATE TABLE notes (id INTEGER PRIMARY KEY);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
		t.Fatal(err)
	}
	_, err = Down(Options{Database: database, Directory: directory}, 1)
	if err == nil || !strings.Contains(err.Error(), "down") {
		t.Fatalf("down = %v", err)
	}
	if countHistory(t, database) != 1 || !tableExists(t, database, "notes") {
		t.Fatal("refused down changed the database")
	}
}

func TestMigrateRejectsASymlinkOutsideTheDirectory(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "migrations")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(outside, "secret.sql")
	if err := os.WriteFile(target, []byte("DROP TABLE notes;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "0001_create_notes.up.sql")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("this process cannot create a symbolic link: %v", err)
	}
	_, err := Scan(directory)
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("scan = %v", err)
	}
	database := filepath.Join(root, "app.db")
	if _, err := Up(Options{Database: database, Directory: directory}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("up = %v", err)
	}
	if _, statErr := os.Stat(database); statErr == nil && tableExists(t, database, "notes") {
		t.Fatal("a linked script was executed")
	}
}
