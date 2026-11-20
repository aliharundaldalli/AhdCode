package migrate

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func TestMigrateRejectsTransactionEscapeAndHistoryTampering(t *testing.T) {
	attacks := []string{
		"CREATE TABLE escaped (id INTEGER PRIMARY KEY);\nCOMMIT;\n",
		"COMMIT;\nBEGIN;\nCREATE TABLE escaped (id INTEGER PRIMARY KEY);\n",
		"COMMIT;\nBEGIN;\nCREATE TABLE escaped (id INTEGER PRIMARY KEY);\nSELECT * FROM missing_table;\n",
		"ROLLBACK;\n",
		"END;\n",
		"BEGIN IMMEDIATE;\n",
		"ATTACH DATABASE 'other.db' AS other;\nCREATE TABLE other.escaped (id INTEGER);\n",
		"DETACH DATABASE main;\n",
		"INSERT INTO ahd_schema_migrations (version, name, up_checksum, down_checksum, applied_at) VALUES (99, 'x', 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', '', 1);\n",
		"UPDATE ahd_schema_migrations SET name = 'tampered';\n",
		"DELETE FROM ahd_schema_migrations;\n",
		"DROP TABLE ahd_schema_migrations;\n",
		"ALTER TABLE ahd_schema_migrations ADD COLUMN extra TEXT;\n",
		"ALTER TABLE \"ahd_schema_migrations\" RENAME TO stolen;\n",
		"DELETE FROM MAIN.ahd_schema_migrations;\n",
		"CREATE TEMP TABLE ahd_schema_migrations (version INTEGER);\n",
		"CREATE TRIGGER evil AFTER INSERT ON notes BEGIN\nDELETE FROM ahd_schema_migrations;\nEND;\nINSERT INTO notes (id) VALUES (2);\n",
		"CREATE VIEW history_view AS SELECT * FROM ahd_schema_migrations;\nCREATE TRIGGER history_view_delete INSTEAD OF DELETE ON history_view BEGIN\nDELETE FROM ahd_schema_migrations;\nEND;\n",
		"PRAGMA writable_schema = ON;\n",
		"PRAGMA journal_mode = OFF;\n",
		"PRAGMA schema_version = 1;\n",
		"CREATE TABLE escaped (id INTEGER PRIMARY KEY);\nVACUUM;\n",
		"UPDATE sqlite_master SET sql = sql;\n",
		"UPDATE sqlite_schema SET sql = sql;\n",
	}
	for _, script := range attacks {
		t.Run(script[:min(24, len(script))], func(t *testing.T) {
			directory := t.TempDir()
			database := filepath.Join(directory, "app.db")
			other := filepath.Join(directory, "other.db")
			writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\nINSERT INTO notes (id) VALUES (1);\n", "DROP TABLE notes;\n")
			if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
				t.Fatal(err)
			}
			before := historySnapshot(t, database)
			writePair(t, directory, "0002", "attack", script, "")
			_, err := Up(Options{Database: database, Directory: directory})
			if err == nil || !strings.Contains(err.Error(), "rejected") && !strings.Contains(err.Error(), "not allowed") && !strings.Contains(err.Error(), "failed") {
				t.Fatalf("error = %v", err)
			}
			if tableExists(t, database, "escaped") || tableExists(t, database, "stolen") || tableExists(t, database, "history_view") {
				t.Fatal("rejected SQL left a committed table or view")
			}
			if _, statErr := os.Stat(other); statErr == nil {
				t.Fatal("ATTACH created another database file")
			}
			if historySnapshot(t, database) != before {
				t.Fatalf("history changed from %s", before)
			}
			if countHistory(t, database) != 1 || !tableExists(t, database, "notes") {
				t.Fatal("the applied migration was not left intact")
			}
			if err := os.WriteFile(filepath.Join(directory, "0002_attack.up.sql"), []byte("ALTER TABLE notes ADD COLUMN body TEXT;\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			applied, err := Up(Options{Database: database, Directory: directory})
			if err != nil || len(applied) != 1 || countHistory(t, database) != 2 {
				t.Fatalf("corrected migration %+v %v", applied, err)
			}
		})
	}
}

func TestMigrateAllowsTriggersSavepointsAndQuotedSemicolons(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	script := "" +
		"CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL);\n" +
		"CREATE TRIGGER notes_bi BEFORE INSERT ON notes BEGIN\n" +
		"  SELECT CASE WHEN NEW.body = 'bad' THEN RAISE(ABORT, 'no') ELSE NULL END;\n" +
		"END;\n" +
		"INSERT INTO notes (id, body) VALUES (1, 'a;b');\n" +
		"SAVEPOINT keep;\n" +
		"UPDATE notes SET body = 'changed' WHERE id = 1;\n" +
		"ROLLBACK TO keep;\n" +
		"RELEASE keep;\n" +
		"INSERT INTO notes (id, body) VALUES (2, 'second');\n"
	writePair(t, directory, "0001", "create_notes", script, "DROP TABLE notes;\n")
	applied, err := Up(Options{Database: database, Directory: directory})
	if err != nil || len(applied) != 1 {
		t.Fatal(err)
	}
	if noteBody(t, database) != "a;b" || countHistory(t, database) != 1 {
		t.Fatalf("body %s history %d", noteBody(t, database), countHistory(t, database))
	}
}

func TestMigrateRejectsCommentOnlySQL(t *testing.T) {
	cases := []string{
		"",
		"   \n\t\n",
		"-- only a comment\n",
		"/* block */\n",
		"-- semicolon ; inside a comment\n/* another ; */\n",
		"-- line\n/* block; still a comment */\n",
	}
	for index, script := range cases {
		t.Run(fmt.Sprintf("%d", index), func(t *testing.T) {
			directory := t.TempDir()
			database := filepath.Join(directory, "app.db")
			writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "DROP TABLE notes;\n")
			if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
				t.Fatal(err)
			}
			writePair(t, directory, "0002", "empty", script, "")
			_, err := Up(Options{Database: database, Directory: directory})
			if err == nil || !strings.Contains(err.Error(), "empty") {
				t.Fatalf("up error = %v", err)
			}
			if countHistory(t, database) != 1 || tableExists(t, database, "extra") {
				t.Fatal("comment-only SQL was recorded")
			}
		})
	}
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "-- comment only\n")
	if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
		t.Fatal(err)
	}
	_, err := Down(Options{Database: database, Directory: directory}, 1)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("comment-only down = %v", err)
	}
	if countHistory(t, database) != 1 || !tableExists(t, database, "notes") {
		t.Fatal("comment-only down removed history or the table")
	}
	templateDir := t.TempDir()
	database = filepath.Join(templateDir, "app.db")
	if _, err := Create(templateDir, "create_notes"); err != nil {
		t.Fatal(err)
	}
	_, err = Up(Options{Database: database, Directory: templateDir})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("template up = %v", err)
	}
	if _, statErr := os.Stat(database); statErr == nil && tableExists(t, database, historyTable) {
		t.Fatal("the unedited template was recorded")
	}
	script := "-- leading comment;\n/* block; comment */\nCREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT);\nINSERT INTO notes (id, body) VALUES (1, 'a;b');\n"
	if err := os.WriteFile(filepath.Join(templateDir, "0001_create_notes.up.sql"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	applied, err := Up(Options{Database: database, Directory: templateDir})
	if err != nil || len(applied) != 1 || noteBody(t, database) != "a;b" {
		t.Fatalf("comment then SQL %+v %v", applied, err)
	}
}

func TestMigrateRejectsCorruptHistoryWithoutRepairingIt(t *testing.T) {
	validChecksum := strings.Repeat("ab", 32)
	shapes := []struct {
		name string
		sql  string
	}{
		{"extra nullable", `CREATE TABLE ahd_schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, up_checksum TEXT NOT NULL, down_checksum TEXT NOT NULL, applied_at INTEGER NOT NULL, extra TEXT)`},
		{"extra required", `CREATE TABLE ahd_schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, up_checksum TEXT NOT NULL, down_checksum TEXT NOT NULL, applied_at INTEGER NOT NULL, extra TEXT NOT NULL DEFAULT '')`},
		{"missing column", `CREATE TABLE ahd_schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, up_checksum TEXT NOT NULL, down_checksum TEXT NOT NULL)`},
		{"wrong type", `CREATE TABLE ahd_schema_migrations (version TEXT PRIMARY KEY, name TEXT NOT NULL, up_checksum TEXT NOT NULL, down_checksum TEXT NOT NULL, applied_at INTEGER NOT NULL)`},
		{"nullable name", `CREATE TABLE ahd_schema_migrations (version INTEGER PRIMARY KEY, name TEXT, up_checksum TEXT NOT NULL, down_checksum TEXT NOT NULL, applied_at INTEGER NOT NULL)`},
		{"unexpected index", `CREATE TABLE ahd_schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, up_checksum TEXT NOT NULL, down_checksum TEXT NOT NULL, applied_at INTEGER NOT NULL); CREATE INDEX ahd_schema_migrations_name ON ahd_schema_migrations (name)`},
	}
	for _, shape := range shapes {
		t.Run(shape.name, func(t *testing.T) {
			directory := t.TempDir()
			database := filepath.Join(directory, "app.db")
			execDB(t, database, shape.sql)
			before := masterText(t, database)
			writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "")
			_, err := Up(Options{Database: database, Directory: directory})
			if err == nil || !strings.Contains(err.Error(), "will not change it") {
				t.Fatalf("error = %v", err)
			}
			if masterText(t, database) != before || tableExists(t, database, "notes") {
				t.Fatal("corrupt history was repaired or the migration ran")
			}
		})
	}
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	execDB(t, database, `CREATE TABLE ahd_schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, up_checksum TEXT NOT NULL, down_checksum TEXT NOT NULL, applied_at INTEGER NOT NULL)`)
	execDB(t, database, `INSERT INTO ahd_schema_migrations (version, name, up_checksum, down_checksum, applied_at) VALUES (1, 'create_notes', x'abcd', '', 1)`)
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "")
	_, err := Up(Options{Database: database, Directory: directory})
	if err == nil || !strings.Contains(err.Error(), "non-text") {
		t.Fatalf("blob checksum = %v", err)
	}
	if tableExists(t, database, "notes") {
		t.Fatal("a non-text checksum was coerced or the migration ran")
	}
	execDB(t, database, `DELETE FROM ahd_schema_migrations`)
	execDB(t, database, `INSERT INTO ahd_schema_migrations (version, name, up_checksum, down_checksum, applied_at) VALUES (1, 'create_notes', 'NOT-HEX', '', 1)`)
	_, err = Up(Options{Database: database, Directory: directory})
	if err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("checksum = %v", err)
	}
	execDB(t, database, `DELETE FROM ahd_schema_migrations`)
	execDB(t, database, `INSERT INTO ahd_schema_migrations (version, name, up_checksum, down_checksum, applied_at) VALUES (1, 'create_notes', '`+validChecksum+`', '', 1), (3, 'later', '`+validChecksum+`', '', 1)`)
	_, err = Up(Options{Database: database, Directory: directory})
	if err == nil || !strings.Contains(err.Error(), "inconsistent") {
		t.Fatalf("gap = %v", err)
	}
	if tableExists(t, database, "notes") {
		t.Fatal("inconsistent history was rewritten")
	}
}

func TestMigrateCreateIsSerializedAcrossProcesses(t *testing.T) {
	if os.Getenv("AHD_MIGRATE_NEW") == "1" {
		if err := os.WriteFile(os.Getenv("AHD_MIGRATE_READY"), []byte("ready"), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(os.Getenv("AHD_MIGRATE_GO")); err == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if _, err := Create(os.Getenv("AHD_MIGRATE_DIR"), os.Getenv("AHD_MIGRATE_NAME")); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	directory := filepath.Join(t.TempDir(), "Göç Klasörü")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	const processes = 20
	commands := make([]*exec.Cmd, processes)
	outputs := make([]bytes.Buffer, processes)
	for index := range processes {
		name := fmt.Sprintf("name_%02d", index)
		if index < 10 {
			name = "create_notes"
		}
		ready := filepath.Join(directory, fmt.Sprintf("ready-%02d", index))
		command := exec.Command(os.Args[0], "-test.run=^TestMigrateCreateIsSerializedAcrossProcesses$")
		command.Env = append(os.Environ(), "AHD_MIGRATE_NEW=1", "AHD_MIGRATE_DIR="+directory, "AHD_MIGRATE_NAME="+name, "AHD_MIGRATE_READY="+ready, "AHD_MIGRATE_GO="+filepath.Join(directory, "go"))
		command.Stdout = &outputs[index]
		command.Stderr = &outputs[index]
		commands[index] = command
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, command := range commands {
			if command.Process != nil && command.ProcessState == nil {
				_ = command.Process.Kill()
				_, _ = command.Process.Wait()
			}
		}
	})
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		ready := 0
		for index := range processes {
			if _, err := os.Stat(filepath.Join(directory, fmt.Sprintf("ready-%02d", index))); err == nil {
				ready++
			}
		}
		if ready == processes {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(directory, "go"), []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	for index, command := range commands {
		if err := command.Wait(); err != nil {
			t.Fatalf("process %d: %v %s", index, err, outputs[index].String())
		}
	}
	files, err := Scan(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != processes {
		t.Fatalf("created %d migrations", len(files))
	}
	same := 0
	seen := map[int64]string{}
	for _, file := range files {
		if file.Version < 1 || file.Version > processes {
			t.Fatalf("version %d", file.Version)
		}
		if previous, ok := seen[file.Version]; ok {
			t.Fatalf("version %d created twice (%s and %s)", file.Version, previous, file.Name)
		}
		seen[file.Version] = file.Name
		if file.Name == "create_notes" {
			same++
		}
		if _, err := os.Stat(file.UpPath); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(file.DownPath); err != nil {
			t.Fatal(err)
		}
	}
	if same != 10 {
		t.Fatalf("identical names produced %d files", same)
	}
}

func TestMigrateCreateFailureDoesNotLeaveAnOrphan(t *testing.T) {
	directory := t.TempDir()
	previous := createStage
	t.Cleanup(func() { createStage = previous })
	createStage = func(stage string) error {
		if stage == "before-down" {
			return errors.New("down was not written")
		}
		return nil
	}
	if _, err := Create(directory, "create_notes"); err == nil || !strings.Contains(err.Error(), "down was not written") {
		t.Fatalf("error = %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") || strings.Contains(entry.Name(), ".incoming") {
			t.Fatalf("orphan %s", entry.Name())
		}
	}
	createStage = nil
	if _, err := Create(directory, "create_notes"); err != nil {
		t.Fatal(err)
	}
	files, err := Scan(directory)
	if err != nil || len(files) != 1 || files[0].Label != "0001_create_notes" {
		t.Fatalf("files %+v %v", files, err)
	}
}

func TestMigrateCreateRecoversAfterTheProcessIsKilled(t *testing.T) {
	if os.Getenv("AHD_MIGRATE_NEW_KILL") == "1" {
		createStage = func(stage string) error {
			if stage != "down-published" {
				return nil
			}
			if err := os.WriteFile(os.Getenv("AHD_MIGRATE_MARKER"), []byte("published"), 0o644); err != nil {
				return err
			}
			time.Sleep(time.Minute)
			return nil
		}
		_, err := Create(os.Getenv("AHD_MIGRATE_DIR"), "create_notes")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	directory := filepath.Join(t.TempDir(), "Göç Klasörü")
	marker := filepath.Join(filepath.Dir(directory), "marker")
	command := exec.Command(os.Args[0], "-test.run=^TestMigrateCreateRecoversAfterTheProcessIsKilled$")
	command.Env = append(os.Environ(), "AHD_MIGRATE_NEW_KILL=1", "AHD_MIGRATE_DIR="+directory, "AHD_MIGRATE_MARKER="+marker)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
			_, _ = command.Process.Wait()
		}
	})
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("the child did not publish the down file")
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_, _ = command.Process.Wait()
	if _, err := Create(directory, "add_author"); err != nil {
		t.Fatal(err)
	}
	files, err := Scan(directory)
	if err != nil || len(files) != 2 || files[0].Label != "0001_create_notes" || files[1].Label != "0002_add_author" {
		t.Fatalf("files %+v %v", files, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".incoming") {
			t.Fatalf("incoming file remained: %s", entry.Name())
		}
	}
	database := filepath.Join(filepath.Dir(directory), "app.db")
	status, err := Inspect(Options{Database: database, Directory: directory})
	if err != nil || status.DatabaseExists || len(status.Pending) != 2 {
		t.Fatalf("status %+v %v", status, err)
	}
	if _, statErr := os.Stat(database); statErr == nil {
		t.Fatal("status created a missing database")
	}
	if err := os.WriteFile(files[0].UpPath, []byte("CREATE TABLE notes (id INTEGER PRIMARY KEY);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(files[0].DownPath, []byte("DROP TABLE notes;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(files[1].UpPath, []byte("ALTER TABLE notes ADD COLUMN body TEXT;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	applied, err := Up(Options{Database: database, Directory: directory})
	if err != nil || len(applied) != 2 {
		t.Fatalf("up after recovery %+v %v", applied, err)
	}
}

func TestMigrateRejectsJunctions(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("unprivileged junctions are created with mklink /J")
	}
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	writePair(t, real, "0001", "create_notes", "CREATE TABLE escaped (id INTEGER PRIMARY KEY);\n", "")
	link := filepath.Join(root, "link")
	command := exec.Command("cmd", "/c", "mklink", "/J", link, real)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("junction: %v %s", err, output)
	}
	if _, err := Scan(link); err == nil || !strings.Contains(err.Error(), "junction") && !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("scan through junction: %v", err)
	}
	database := filepath.Join(root, "app.db")
	if _, err := Up(Options{Database: database, Directory: link}); err == nil {
		t.Fatal("up followed a junction")
	}
	if _, statErr := os.Stat(database); statErr == nil && tableExists(t, database, "escaped") {
		t.Fatal("SQL behind a junction was executed")
	}
	parent := filepath.Join(root, "parent-link")
	parentReal := filepath.Join(root, "parent-real")
	nested := filepath.Join(parentReal, "migrations")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	writePair(t, nested, "0001", "create_notes", "CREATE TABLE escaped (id INTEGER PRIMARY KEY);\n", "")
	if output, err := exec.Command("cmd", "/c", "mklink", "/J", parent, parentReal).CombinedOutput(); err != nil {
		t.Fatalf("parent junction: %v %s", err, output)
	}
	if _, err := Scan(filepath.Join(parent, "migrations")); err == nil || !strings.Contains(err.Error(), "junction") && !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("parent junction scan: %v", err)
	}
}

func TestMigrateStatusDoesNotCreateAndDownDoesNotCreate(t *testing.T) {
	directory := t.TempDir()
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "DROP TABLE notes;\n")
	missing := filepath.Join(directory, "missing.db")
	status, err := Inspect(Options{Database: missing, Directory: directory})
	if err != nil || status.DatabaseExists || len(status.Pending) != 1 {
		t.Fatalf("status %+v %v", status, err)
	}
	if _, statErr := os.Stat(missing); statErr == nil {
		t.Fatal("status created the database")
	}
	if _, err := Down(Options{Database: missing, Directory: directory}, 1); err == nil {
		t.Fatal("down created or rolled back a missing database")
	}
	if _, statErr := os.Stat(missing); statErr == nil {
		t.Fatal("down created the database")
	}
}

func TestMigratePartialDownReportsCommittedSteps(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT);\n", "SELECT * FROM missing_table;\n")
	writePair(t, directory, "0002", "add_author", "ALTER TABLE notes ADD COLUMN author TEXT NOT NULL DEFAULT 'Ali';\n", "ALTER TABLE notes DROP COLUMN author;\n")
	if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
		t.Fatal(err)
	}
	rolled, err := Down(Options{Database: database, Directory: directory}, 2)
	if err == nil || len(rolled) != 1 || rolled[0].Label != "0002_add_author" {
		t.Fatalf("partial %+v %v", rolled, err)
	}
	if countHistory(t, database) != 1 || !tableExists(t, database, "notes") || columnExists(t, database, "notes", "author") {
		t.Fatal("the failed down did not keep the earlier rollback")
	}
}

func TestMigratePartialUpReportsTheCommittedVersion(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\nINSERT INTO notes (id) VALUES (1);\n", "DROP TABLE notes;\n")
	writePair(t, directory, "0002", "broken", "ALTER TABLE notes ADD COLUMN body TEXT;\nSELECT * FROM missing_table;\n", "")
	applied, err := Up(Options{Database: database, Directory: directory})
	if err == nil || len(applied) != 1 || applied[0].Label != "0001_create_notes" {
		t.Fatalf("partial %+v %v", applied, err)
	}
	if countHistory(t, database) != 1 || !tableExists(t, database, "notes") {
		t.Fatal("the committed migration was not kept")
	}
}

func TestMigrateBusyTimeoutDoesNotApply(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "")
	holder, err := sqlite3.OpenFlags(database, sqlite3.OPEN_READWRITE|sqlite3.OPEN_CREATE)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	if err := holder.Exec("BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	started := time.Now()
	go func() {
		_, err := Up(Options{Database: database, Directory: directory})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "lock") && !strings.Contains(strings.ToLower(err.Error()), "busy") {
			t.Fatalf("error = %v", err)
		}
		if time.Since(started) < 5*time.Second {
			t.Fatalf("up returned before the 5s busy timeout: %s", time.Since(started))
		}
	case <-time.After(8 * time.Second):
		t.Fatal("up did not return after the busy timeout")
	}
	_ = holder.Exec("ROLLBACK")
	if tableExists(t, database, "notes") || tableExists(t, database, historyTable) {
		t.Fatal("a timed-out migration was recorded")
	}
}

func TestMigrateConcurrentDownAndUpStayConsistent(t *testing.T) {
	if os.Getenv("AHD_MIGRATE_OP") != "" {
		ready := os.Getenv("AHD_MIGRATE_READY")
		gate := os.Getenv("AHD_MIGRATE_GATE")
		if err := os.WriteFile(ready, []byte("ready"), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		deadline := time.Now().Add(20 * time.Second)
		opened := false
		for time.Now().Before(deadline) {
			if _, err := os.Stat(gate); err == nil {
				opened = true
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if !opened {
			fmt.Fprintln(os.Stderr, "the other process was not started")
			os.Exit(1)
		}
		var err error
		options := Options{Database: os.Getenv("AHD_MIGRATE_DB"), Directory: os.Getenv("AHD_MIGRATE_DIR")}
		switch os.Getenv("AHD_MIGRATE_OP") {
		case "up":
			_, err = Up(options)
		case "down":
			_, err = Down(options, 1)
		default:
			fmt.Fprintln(os.Stderr, "unknown operation")
			os.Exit(1)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	notes := "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT);\nINSERT INTO notes (id, body) VALUES (1, 'ilk');\n"
	addAuthor := "ALTER TABLE notes ADD COLUMN author TEXT NOT NULL DEFAULT 'Ali';\n"
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", notes, "DROP TABLE notes;\n")
	writePair(t, directory, "0002", "add_author", addAuthor, "ALTER TABLE notes DROP COLUMN author;\n")
	if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
		t.Fatal(err)
	}
	err1, err2 := raceMigrate(t, database, directory, "down", "down")
	if err1 != nil && err2 != nil {
		t.Fatalf("both down processes failed: %v | %v", err1, err2)
	}
	if _, err := Inspect(Options{Database: database, Directory: directory}); err != nil {
		t.Fatalf("history after parallel down: %v", err)
	}
	assertNotesMatchHistory(t, database)

	freshDir := t.TempDir()
	fresh := filepath.Join(freshDir, "app.db")
	writePair(t, freshDir, "0001", "create_notes", notes, "DROP TABLE notes;\n")
	if _, err := Up(Options{Database: fresh, Directory: freshDir}); err != nil {
		t.Fatal(err)
	}
	writePair(t, freshDir, "0002", "add_author", addAuthor, "ALTER TABLE notes DROP COLUMN author;\n")
	upErr, downErr := raceMigrate(t, fresh, freshDir, "up", "down")
	if upErr != nil && downErr != nil {
		t.Fatalf("up and down both failed: %v | %v", upErr, downErr)
	}
	if _, err := Inspect(Options{Database: fresh, Directory: freshDir}); err != nil {
		t.Fatalf("history inconsistent after up versus down: %v", err)
	}
	assertNotesMatchHistory(t, fresh)
}

type migrateProc struct {
	command *exec.Cmd
	output  bytes.Buffer
	done    chan error
}

func raceMigrate(t *testing.T, database, directory, left, right string) (error, error) {
	t.Helper()
	gate := filepath.Join(directory, "race-gate-"+left+"-"+right)
	_ = os.Remove(gate)
	start := func(op, ready string) *migrateProc {
		proc := &migrateProc{done: make(chan error, 1)}
		proc.command = exec.Command(os.Args[0], "-test.run=^TestMigrateConcurrentDownAndUpStayConsistent$")
		proc.command.Env = append(os.Environ(), "AHD_MIGRATE_OP="+op, "AHD_MIGRATE_DB="+database, "AHD_MIGRATE_DIR="+directory, "AHD_MIGRATE_READY="+ready, "AHD_MIGRATE_GATE="+gate)
		proc.command.Stdout = &proc.output
		proc.command.Stderr = &proc.output
		if err := proc.command.Start(); err != nil {
			t.Fatal(err)
		}
		go func() { proc.done <- proc.command.Wait() }()
		return proc
	}
	first := start(left, filepath.Join(directory, "ready-"+left+"-a"))
	second := start(right, filepath.Join(directory, "ready-"+right+"-b"))
	t.Cleanup(func() {
		_ = os.WriteFile(gate, []byte("go"), 0o644)
		if first.command.Process != nil {
			_ = first.command.Process.Kill()
		}
		if second.command.Process != nil {
			_ = second.command.Process.Kill()
		}
	})
	if err := waitForFile(filepath.Join(directory, "ready-"+left+"-a"), 20*time.Second); err != nil {
		t.Fatalf("%s did not start: %v\n%s", left, err, first.output.String())
	}
	if err := waitForFile(filepath.Join(directory, "ready-"+right+"-b"), 20*time.Second); err != nil {
		t.Fatalf("%s did not start: %v\n%s", right, err, second.output.String())
	}
	select {
	case err := <-first.done:
		t.Fatalf("%s finished before %s started the operation: %v\n%s", left, right, err, first.output.String())
	case err := <-second.done:
		t.Fatalf("%s finished before %s started the operation: %v\n%s", right, left, err, second.output.String())
	case <-time.After(200 * time.Millisecond):
	}
	if err := os.WriteFile(gate, []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	return waitMigrate(t, first, left), waitMigrate(t, second, right)
}

func waitMigrate(t *testing.T, proc *migrateProc, name string) error {
	t.Helper()
	select {
	case err := <-proc.done:
		if err != nil {
			return fmt.Errorf("%s: %w\n%s", name, err, proc.output.String())
		}
		return nil
	case <-time.After(20 * time.Second):
		t.Fatalf("%s did not exit\n%s", name, proc.output.String())
		return nil
	}
}

func assertNotesMatchHistory(t *testing.T, database string) {
	t.Helper()
	applied := 0
	if tableExists(t, database, historyTable) {
		applied = countHistory(t, database)
	}
	notes := tableExists(t, database, "notes")
	author := columnExists(t, database, "notes", "author")
	switch applied {
	case 0:
		if notes {
			t.Fatal("history is empty but notes still exists")
		}
	case 1:
		if !notes || author {
			t.Fatalf("version 1 does not match the schema (notes=%v author=%v)", notes, author)
		}
	case 2:
		if !notes || !author {
			t.Fatalf("version 2 does not match the schema (notes=%v author=%v)", notes, author)
		}
	default:
		t.Fatalf("unexpected history count %d", applied)
	}
}

func columnExists(t *testing.T, database, table, column string) bool {
	t.Helper()
	if !tableExists(t, database, table) {
		return false
	}
	conn, err := sqlite3.Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	statement, _, err := conn.Prepare(`SELECT 1 FROM pragma_table_info(?) WHERE name = ? COLLATE NOCASE`)
	if err != nil {
		t.Fatal(err)
	}
	defer statement.Close()
	if err := statement.BindText(1, table); err != nil {
		t.Fatal(err)
	}
	if err := statement.BindText(2, column); err != nil {
		t.Fatal(err)
	}
	return statement.Step()
}

func historySnapshot(t *testing.T, database string) string {
	t.Helper()
	conn, err := sqlite3.Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	statement, _, err := conn.Prepare(`SELECT version, name, up_checksum, down_checksum, applied_at FROM ahd_schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	defer statement.Close()
	var buffer bytes.Buffer
	for statement.Step() {
		fmt.Fprintf(&buffer, "%s|%s|%s|%s|%s\n", statement.ColumnText(0), statement.ColumnText(1), statement.ColumnText(2), statement.ColumnText(3), statement.ColumnText(4))
	}
	if err := statement.Err(); err != nil {
		t.Fatal(err)
	}
	return buffer.String()
}

func masterText(t *testing.T, database string) string {
	t.Helper()
	conn, err := sqlite3.Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	statement, _, err := conn.Prepare(`SELECT sql FROM sqlite_master ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer statement.Close()
	var buffer bytes.Buffer
	for statement.Step() {
		fmt.Fprintln(&buffer, statement.ColumnText(0))
	}
	return buffer.String()
}

func execDB(t *testing.T, database, sql string) {
	t.Helper()
	conn, err := sqlite3.OpenFlags(database, sqlite3.OPEN_READWRITE|sqlite3.OPEN_CREATE)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.Exec(sql); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func TestMigrateTriggerAuthorizationMatrix(t *testing.T) {
	notes := "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT, n INTEGER NOT NULL DEFAULT 0);\nINSERT INTO notes (id, body) VALUES (1, 'ilk');\n"
	view := notes + "CREATE VIEW notes_view AS SELECT id, body, n FROM notes;\n"
	cases := []struct {
		name   string
		setup  string
		script string
		allow  bool
		app    string
		want   string
	}{
		{
			name: "T1 instead of update on a view",
			script: "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT);\n" +
				"CREATE VIEW notes_view AS SELECT id, body FROM notes;\n" +
				"CREATE TRIGGER evil INSTEAD OF UPDATE ON notes_view BEGIN\n" +
				"UPDATE ahd_schema_migrations SET name = 'stolen';\nEND;\n",
		},
		{
			name:  "T2 after update of column",
			setup: notes,
			script: "CREATE TRIGGER evil AFTER UPDATE OF body ON notes BEGIN\n" +
				"UPDATE ahd_schema_migrations SET name = 'stolen';\nEND;\n",
		},
		{
			name:  "T3 before update of column",
			setup: notes,
			script: "CREATE TRIGGER evil BEFORE UPDATE OF body ON notes BEGIN\n" +
				"UPDATE ahd_schema_migrations SET name = 'stolen';\nEND;\n",
		},
		{
			name:  "T4 instead of update of column",
			setup: view,
			script: "CREATE TRIGGER evil INSTEAD OF UPDATE OF body ON notes_view BEGIN\n" +
				"UPDATE ahd_schema_migrations SET name = 'stolen';\nEND;\n",
		},
		{
			name:  "T5 update on without rowid",
			setup: "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT) WITHOUT ROWID;\nINSERT INTO notes (id, body) VALUES (1, 'ilk');\n",
			script: "CREATE TRIGGER evil AFTER UPDATE ON notes BEGIN\n" +
				"UPDATE ahd_schema_migrations SET name = 'stolen';\nEND;\n",
		},
		{
			name:  "T6 trigger introduced by a later migration",
			setup: notes,
			script: "CREATE TRIGGER evil AFTER UPDATE OF body ON notes BEGIN\n" +
				"DELETE FROM ahd_schema_migrations;\nEND;\n",
			app: "UPDATE notes SET body = 'later' WHERE id = 1;",
		},
		{
			name: "T7 legitimate multi-statement trigger",
			script: notes + "CREATE TRIGGER notes_touch AFTER UPDATE OF body ON notes BEGIN\n" +
				"UPDATE notes SET n = n + 1 WHERE id = NEW.id;\n" +
				"UPDATE notes SET n = n + 1 WHERE id = NEW.id;\nEND;\n",
			allow: true,
			app:   "UPDATE notes SET body = 'iki' WHERE id = 1;",
			want:  "2",
		},
		{
			name:  "T8 quoted identifiers",
			setup: notes,
			script: "CREATE TRIGGER \"evil trig\" AFTER UPDATE OF \"body\" ON \"notes\" BEGIN\n" +
				"UPDATE \"ahd_schema_migrations\" SET name = 'stolen';\nEND;\n",
		},
		{
			name: "T9 legitimate instead of update",
			script: "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT);\n" +
				"CREATE VIEW notes_view AS SELECT id, body FROM notes;\n" +
				"CREATE TRIGGER notes_view_upd INSTEAD OF UPDATE ON notes_view BEGIN\n" +
				"UPDATE notes SET body = NEW.body WHERE id = OLD.id;\nEND;\n" +
				"INSERT INTO notes (id, body) VALUES (1, 'ilk');\n",
			allow: true,
			app:   "UPDATE notes_view SET body = 'iki' WHERE id = 1;",
			want:  "iki",
		},
		{
			name:  "T10 update of a column that does not exist",
			setup: notes,
			script: "CREATE TRIGGER evil AFTER UPDATE OF not_a_column ON notes BEGIN\n" +
				"UPDATE ahd_schema_migrations SET name = 'stolen';\nEND;\n",
		},
		{
			name:  "T11 temporary update-of trigger",
			setup: notes,
			script: "CREATE TEMP TRIGGER evil AFTER UPDATE OF body ON notes BEGIN\n" +
				"DELETE FROM ahd_schema_migrations;\nEND;\n",
		},
		{
			name: "nested trigger",
			script: notes + "CREATE TABLE other (id INTEGER PRIMARY KEY);\n" +
				"INSERT INTO other (id) VALUES (1);\n" +
				"CREATE TRIGGER inner_evil AFTER UPDATE ON other BEGIN\n" +
				"UPDATE ahd_schema_migrations SET name = 'stolen';\nEND;\n" +
				"CREATE TRIGGER outer_notes AFTER UPDATE ON notes BEGIN\n" +
				"UPDATE other SET id = id;\nEND;\n",
		},
		{
			name:  "schema-qualified target",
			setup: notes,
			script: "CREATE TRIGGER evil AFTER UPDATE OF body ON main.notes BEGIN\n" +
				"UPDATE ahd_schema_migrations SET name = 'stolen';\nEND;\n",
		},
		{
			name: "legitimate before update and generated column",
			script: "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT, total INTEGER GENERATED ALWAYS AS (id) VIRTUAL);\n" +
				"CREATE TRIGGER notes_bu BEFORE UPDATE OF body ON notes BEGIN\n" +
				"SELECT CASE WHEN NEW.body = 'bad' THEN RAISE(ABORT, 'no') ELSE NULL END;\nEND;\n" +
				"INSERT INTO notes (id, body) VALUES (1, 'ilk');\n",
			allow: true,
			app:   "UPDATE notes SET body = 'iki' WHERE id = 1;",
			want:  "iki",
		},
		{
			name: "legitimate without rowid trigger",
			script: "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT) WITHOUT ROWID;\n" +
				"CREATE TRIGGER notes_bu BEFORE UPDATE ON notes BEGIN\n" +
				"SELECT CASE WHEN NEW.body = 'bad' THEN RAISE(ABORT, 'no') ELSE NULL END;\nEND;\n" +
				"INSERT INTO notes (id, body) VALUES (1, 'ilk');\n",
			allow: true,
			app:   "UPDATE notes SET body = 'iki' WHERE id = 1;",
			want:  "iki",
		},
	}
	for _, candidate := range cases {
		t.Run(candidate.name, func(t *testing.T) {
			directory := t.TempDir()
			database := filepath.Join(directory, "app.db")
			if candidate.setup != "" {
				writePair(t, directory, "0001", "setup", candidate.setup, "SELECT 1;\n")
				if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
					t.Fatal(err)
				}
			}
			version, name := "0001", "candidate"
			if candidate.setup != "" {
				version, name = "0002", "candidate"
			}
			writePair(t, directory, version, name, candidate.script, "SELECT 1;\n")
			before := ""
			if candidate.setup != "" {
				before = historySnapshot(t, database)
			}
			_, err := Up(Options{Database: database, Directory: directory})
			if candidate.allow {
				if err != nil {
					t.Fatal(err)
				}
				snapshot := historySnapshot(t, database)
				if candidate.app != "" {
					execDB(t, database, candidate.app)
					if historySnapshot(t, database) != snapshot {
						t.Fatal("application SQL changed migration history through an accepted trigger")
					}
					if candidate.want != "" {
						if got := scalarDB(t, database, wantQuery(candidate.want)); got != candidate.want {
							t.Fatalf("trigger effect = %s, want %s", got, candidate.want)
						}
					}
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "rejected") {
				t.Fatalf("error = %v", err)
			}
			if candidate.setup == "" {
				if tableExists(t, database, "notes") || tableExists(t, database, historyTable) {
					t.Fatal("a rejected trigger migration committed")
				}
				return
			}
			if historySnapshot(t, database) != before {
				t.Fatal("a rejected trigger changed migration history")
			}
			if triggerExists(t, database, "evil") || triggerExists(t, database, "evil trig") || triggerExists(t, database, "inner_evil") {
				t.Fatal("a rejected trigger was committed")
			}
			if candidate.app != "" {
				execDB(t, database, candidate.app)
				if historySnapshot(t, database) != before {
					t.Fatal("application SQL changed migration history after a rejected trigger")
				}
			}
		})
	}
}

func wantQuery(want string) string {
	if want == "iki" || want == "2" {
		if want == "2" {
			return "SELECT CAST(n AS TEXT) FROM notes"
		}
		return "SELECT body FROM notes"
	}
	return "SELECT body FROM notes"
}

func scalarDB(t *testing.T, database, sql string) string {
	t.Helper()
	conn, err := sqlite3.Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	statement, _, err := conn.Prepare(sql)
	if err != nil {
		t.Fatal(err)
	}
	defer statement.Close()
	if !statement.Step() {
		t.Fatal(statement.Err())
	}
	return statement.ColumnText(0)
}

func triggerExists(t *testing.T, database, name string) bool {
	t.Helper()
	conn, err := sqlite3.Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	statement, _, err := conn.Prepare(`SELECT 1 FROM sqlite_master WHERE type = 'trigger' AND name = ?`)
	if err != nil {
		t.Fatal(err)
	}
	defer statement.Close()
	if err := statement.BindText(1, name); err != nil {
		t.Fatal(err)
	}
	return statement.Step()
}

func TestMigrateHistoryNameIsCaseInsensitive(t *testing.T) {
	shape := ` (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		up_checksum TEXT NOT NULL,
		down_checksum TEXT NOT NULL,
		applied_at INTEGER NOT NULL
	)`
	for _, name := range []string{"ahd_schema_migrations", "AHD_SCHEMA_MIGRATIONS", "Ahd_Schema_Migrations"} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			database := filepath.Join(directory, "app.db")
			execDB(t, database, `CREATE TABLE "`+name+`"`+shape+";")
			writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "")
			applied, err := Up(Options{Database: database, Directory: directory})
			if err != nil || len(applied) != 1 || countHistory(t, database) != 1 {
				t.Fatalf("applied %+v %v", applied, err)
			}
			if objectsNamed(t, database, historyTable) != 1 {
				t.Fatal("history discovery created a second object")
			}
		})
	}
	t.Run("view", func(t *testing.T) {
		directory := t.TempDir()
		database := filepath.Join(directory, "app.db")
		execDB(t, database, `CREATE VIEW "AHD_SCHEMA_MIGRATIONS" AS SELECT 1 AS version;`)
		writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "")
		_, err := Up(Options{Database: database, Directory: directory})
		if err == nil || !strings.Contains(err.Error(), "view") {
			t.Fatalf("error = %v", err)
		}
		if tableExists(t, database, "notes") || tableExists(t, database, historyTable) {
			t.Fatal("a view was treated as a missing history table")
		}
	})
	t.Run("collision", func(t *testing.T) {
		directory := t.TempDir()
		database := filepath.Join(directory, "app.db")
		conn, err := sqlite3.OpenFlags(database, sqlite3.OPEN_READWRITE|sqlite3.OPEN_CREATE)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if err := conn.Exec(`CREATE TABLE ahd_schema_migrations` + shape); err != nil {
			t.Fatal(err)
		}
		insertErr := conn.Exec(`PRAGMA writable_schema = ON;
			INSERT INTO sqlite_master (type, name, tbl_name, rootpage, sql)
			VALUES ('view', 'AHD_SCHEMA_MIGRATIONS', 'AHD_SCHEMA_MIGRATIONS', 0, 'CREATE VIEW "AHD_SCHEMA_MIGRATIONS" AS SELECT 1');`)
		kind, kindErr := historyKind(conn)
		if insertErr == nil && kindErr == nil {
			t.Fatalf("colliding history object was readable as %q", kind)
		}
		if insertErr != nil && !strings.Contains(insertErr.Error(), "already exists") && !strings.Contains(insertErr.Error(), "malformed") {
			t.Fatal(insertErr)
		}
		if kindErr != nil && !strings.Contains(kindErr.Error(), "matches") && !strings.Contains(kindErr.Error(), "malformed") && !strings.Contains(kindErr.Error(), "unreadable") {
			t.Fatal(kindErr)
		}
	})
}

func objectsNamed(t *testing.T, database, name string) int {
	t.Helper()
	conn, err := sqlite3.Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	statement, _, err := conn.Prepare(`SELECT COUNT(*) FROM sqlite_master WHERE name = ? COLLATE NOCASE`)
	if err != nil {
		t.Fatal(err)
	}
	defer statement.Close()
	if err := statement.BindText(1, name); err != nil {
		t.Fatal(err)
	}
	if !statement.Step() {
		t.Fatal(statement.Err())
	}
	return int(statement.ColumnInt64(0))
}

func TestMigrateRejectsReparseLockAndIncoming(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("unprivileged junctions are created with mklink /J")
	}
	t.Run("lock junction", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "elsewhere")
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(target, "marker.txt")
		if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(directory, ".ahdcode-migrate.lock")
		if output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
			t.Fatalf("junction: %v %s", err, output)
		}
		if _, err := Create(directory, "create_notes"); err == nil || !strings.Contains(err.Error(), "junction") && !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("create = %v", err)
		}
		if _, err := os.Stat(filepath.Join(directory, "0001_create_notes.up.sql")); err == nil {
			t.Fatal("create followed the lock junction")
		}
		body, err := os.ReadFile(marker)
		if err != nil || string(body) != "keep" {
			t.Fatalf("marker = %q %v", body, err)
		}
	})
	t.Run("incoming junction", func(t *testing.T) {
		directory := t.TempDir()
		outside := filepath.Join(directory, "outside")
		if err := os.MkdirAll(outside, 0o755); err != nil {
			t.Fatal(err)
		}
		secret := filepath.Join(outside, "secret.sql")
		if err := os.WriteFile(secret, []byte("DROP TABLE notes;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "0001_create_notes.down.sql"), []byte("DROP TABLE notes;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		incoming := filepath.Join(directory, "0001_create_notes.up.sql.incoming")
		if output, err := exec.Command("cmd", "/c", "mklink", "/J", incoming, outside).CombinedOutput(); err != nil {
			t.Fatalf("junction: %v %s", err, output)
		}
		if _, err := Create(directory, "create_notes"); err == nil || !strings.Contains(err.Error(), "junction") && !strings.Contains(err.Error(), "symbolic link") && !strings.Contains(err.Error(), "non-regular") {
			t.Fatalf("create = %v", err)
		}
		if _, err := os.Stat(filepath.Join(directory, "0001_create_notes.up.sql")); err == nil {
			t.Fatal("recovery renamed a junction into a migration file")
		}
		body, err := os.ReadFile(secret)
		if err != nil || string(body) != "DROP TABLE notes;\n" {
			t.Fatalf("secret = %q %v", body, err)
		}
	})
}

func TestMigrateRejectsSymlinkLock(t *testing.T) {
	directory := t.TempDir()
	real := filepath.Join(directory, "real.lock")
	if err := os.WriteFile(real, []byte("lock"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(directory, ".ahdcode-migrate.lock")); err != nil {
		t.Skip(err.Error())
	}
	if _, err := Create(directory, "create_notes"); err == nil {
		t.Fatal("create followed a symlink lock file")
	}
	body, err := os.ReadFile(real)
	if err != nil || string(body) != "lock" {
		t.Fatalf("lock target = %q %v", body, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "0001_create_notes.up.sql")); err == nil {
		t.Fatal("create wrote a migration through the symlink lock")
	}
}

func TestMigrateRejectsConnectionPragmas(t *testing.T) {
	pragmas := []string{
		"PRAGMA busy_timeout = 0;\n",
		"PRAGMA locking_mode = EXCLUSIVE;\n",
		"PRAGMA trusted_schema = OFF;\n",
		"PRAGMA recursive_triggers = ON;\n",
		"PRAGMA foreign_keys = ON;\n",
		"PRAGMA synchronous = OFF;\n",
		"PRAGMA wal_autocheckpoint = 0;\n",
		"PRAGMA case_sensitive_like = ON;\n",
		"PRAGMA secure_delete = ON;\n",
		"PRAGMA reverse_unordered_selects = ON;\n",
		"PRAGMA temp_store = MEMORY;\n",
	}
	for _, script := range pragmas {
		t.Run(script, func(t *testing.T) {
			directory := t.TempDir()
			database := filepath.Join(directory, "app.db")
			writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "")
			if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
				t.Fatal(err)
			}
			before := historySnapshot(t, database)
			writePair(t, directory, "0002", "pragma", script, "")
			_, err := Up(Options{Database: database, Directory: directory})
			if err == nil || !strings.Contains(err.Error(), "rejected") && !strings.Contains(err.Error(), "not allowed") {
				t.Fatalf("error = %v", err)
			}
			if historySnapshot(t, database) != before {
				t.Fatal("a rejected pragma changed history")
			}
		})
	}
}

func TestBusyTimeoutIsReassertedBeforeEachMigration(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "")
	writePair(t, directory, "0002", "add_body", "ALTER TABLE notes ADD COLUMN body TEXT;\n", "")
	var timeouts []int64
	beforeImmediate = func(conn *sqlite3.Conn) {
		statement, _, err := conn.Prepare(`PRAGMA busy_timeout`)
		if err != nil {
			timeouts = append(timeouts, -1)
			return
		}
		if statement.Step() {
			timeouts = append(timeouts, statement.ColumnInt64(0))
		}
		statement.Close()
	}
	interruptAfterScript = func(conn *sqlite3.Conn) error {
		return conn.BusyTimeout(time.Millisecond)
	}
	defer func() {
		beforeImmediate = nil
		interruptAfterScript = nil
	}()
	if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
		t.Fatal(err)
	}
	if len(timeouts) < 2 {
		t.Fatalf("timeouts before each lock = %v", timeouts)
	}
	for _, timeout := range timeouts {
		if timeout != 5000 {
			t.Fatalf("timeouts before each lock = %v", timeouts)
		}
	}
}

func TestBusyTimeoutStillWaitsFiveSeconds(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	conn, err := openSQLite(database, openCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.Exec("CREATE TABLE notes (id INTEGER PRIMARY KEY);"); err != nil {
		t.Fatal(err)
	}
	if err := conn.BusyTimeout(time.Millisecond); err != nil {
		t.Fatal(err)
	}
	holder, err := sqlite3.OpenFlags(database, sqlite3.OPEN_READWRITE)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	if err := holder.Exec("BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	err = beginImmediate(conn)
	elapsed := time.Since(started)
	_ = holder.Exec("ROLLBACK")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "lock") && !strings.Contains(strings.ToLower(err.Error()), "busy") {
		t.Fatalf("error = %v", err)
	}
	if elapsed < 5*time.Second {
		t.Fatalf("lock wait was %s after the timeout had been set to 1ms", elapsed)
	}
}

func TestAuthorizerClearedOnPanicAndRejection(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	conn, err := openSQLite(database, openCreate)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := beginImmediate(conn); err != nil {
		t.Fatal(err)
	}
	panicDuringAuthorize = func() { panic("authorizer probe") }
	func() {
		defer func() { _ = recover() }()
		_ = execScript(conn, "SELECT 1;", "0001_probe")
	}()
	panicDuringAuthorize = nil
	if err := conn.Exec("ROLLBACK"); err != nil {
		t.Fatalf("authorizer still installed after panic: %v", err)
	}
	if err := beginImmediate(conn); err != nil {
		t.Fatal(err)
	}
	if err := execScript(conn, "COMMIT;\n", "0001_probe"); err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Fatalf("commit = %v", err)
	}
	if err := conn.Exec("ROLLBACK"); err != nil {
		t.Fatalf("authorizer still installed after rejection: %v", err)
	}
}

func TestExplainMarkerIsOnlyTheTriggerInit(t *testing.T) {
	conn, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.Exec(`
		CREATE TABLE notes (
			id INTEGER PRIMARY KEY,
			body TEXT DEFAULT '-- TRIGGER evil',
			n INTEGER NOT NULL DEFAULT 0,
			CONSTRAINT ck CHECK (body != '-- TRIGGER evil2')
		);
		CREATE TRIGGER decoy AFTER INSERT ON notes BEGIN
			SELECT '-- TRIGGER evil';
		END;
		CREATE TRIGGER notes_ok AFTER UPDATE OF body ON notes BEGIN
			SELECT 1;
		END;
	`); err != nil {
		t.Fatal(err)
	}
	names, err := explainTriggerNames(conn, `INSERT INTO "main"."notes" ("id", "body", "n") SELECT "id", "body", "n" FROM "main"."notes" WHERE 0`)
	if err != nil {
		t.Fatal(err)
	}
	if !sameNames(names, []string{"decoy"}) {
		t.Fatalf("insert markers = %v", names)
	}
	names, err = explainTriggerNames(conn, `UPDATE "main"."notes" SET "body" = "body" WHERE 0`)
	if err != nil {
		t.Fatal(err)
	}
	if !sameNames(names, []string{"notes_ok"}) {
		t.Fatalf("update markers = %v", names)
	}
}

func sameNames(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func TestMigrateAdversarialTriggerFamilies(t *testing.T) {
	base := "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT, n INTEGER NOT NULL DEFAULT 0);\nINSERT INTO notes (id, body) VALUES (1, 'ilk');\n"
	withDefault := "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT DEFAULT '-- TRIGGER evil', n INTEGER NOT NULL DEFAULT 0);\nINSERT INTO notes (id) VALUES (1);\n"
	withCheck := "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT, n INTEGER NOT NULL DEFAULT 0, CONSTRAINT ck CHECK (body != '-- TRIGGER evil'));\nINSERT INTO notes (id, body) VALUES (1, 'ilk');\n"
	withGenerated := "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT, total INTEGER GENERATED ALWAYS AS (id) VIRTUAL);\nINSERT INTO notes (id, body) VALUES (1, 'ilk');\n"
	erase := "BEGIN\nDELETE FROM ahd_schema_migrations;\nEND;\n"
	cases := []struct {
		name, setup, script string
		absent              []string
		allow               bool
		app, want           string
	}{
		{name: "A1 default marker and update of rowid", setup: withDefault, script: "CREATE TRIGGER evil AFTER UPDATE OF rowid ON notes " + erase, absent: []string{"evil"}},
		{name: "A2 check marker and update of oid", setup: withCheck, script: "CREATE TRIGGER evil AFTER UPDATE OF oid ON notes " + erase, absent: []string{"evil"}},
		{name: "A3 check marker and update of _rowid_", setup: withCheck, script: "CREATE TRIGGER evil AFTER UPDATE OF _rowid_ ON notes " + erase, absent: []string{"evil"}},
		{name: "A4 decoy trigger and update of a missing column", setup: base, script: "CREATE TRIGGER decoy AFTER INSERT ON notes BEGIN SELECT '-- TRIGGER evil'; END;\nCREATE TRIGGER evil AFTER UPDATE OF not_a_column ON notes " + erase, absent: []string{"evil", "decoy"}},
		{name: "A5 update of rowid", setup: base, script: "CREATE TRIGGER evil AFTER UPDATE OF rowid ON notes " + erase, absent: []string{"evil"}},
		{name: "A6 update of oid", setup: base, script: "CREATE TRIGGER evil BEFORE UPDATE OF oid ON notes " + erase, absent: []string{"evil"}},
		{name: "A8 update of _rowid_", setup: base, script: "CREATE TRIGGER evil AFTER UPDATE OF _rowid_ ON notes " + erase, absent: []string{"evil"}},
		{name: "A17 main and temp share a trigger name", setup: base, script: "CREATE TRIGGER evil AFTER UPDATE OF not_a_column ON notes " + erase + "CREATE TEMP TRIGGER evil AFTER UPDATE OF body ON notes BEGIN SELECT 1; END;\n", absent: []string{"evil"}},
		{name: "A18 case-only main and temp names", setup: base, script: "CREATE TRIGGER Evil AFTER UPDATE OF not_a_column ON notes " + erase + "CREATE TEMP TRIGGER evil AFTER UPDATE OF body ON notes BEGIN SELECT 1; END;\n", absent: []string{"evil", "Evil"}},
		{name: "A20 trigger added by a later migration", setup: base, script: "CREATE TRIGGER evil AFTER UPDATE OF body ON notes " + erase, absent: []string{"evil"}},
		{name: "update of a generated column", setup: withGenerated, script: "CREATE TRIGGER evil AFTER UPDATE OF total ON notes " + erase, absent: []string{"evil"}},
		{name: "legitimate update of rowid", setup: base, script: "CREATE TRIGGER notes_row AFTER UPDATE OF rowid ON notes BEGIN UPDATE notes SET n = n + 1 WHERE id = NEW.id; END;\n", allow: true, app: "UPDATE notes SET rowid = 2 WHERE id = 1;", want: "1"},
		{name: "legitimate update of oid", setup: base, script: "CREATE TRIGGER notes_oid AFTER UPDATE OF oid ON notes BEGIN UPDATE notes SET n = 4 WHERE id = NEW.id; END;\n", allow: true, app: "UPDATE notes SET oid = 3 WHERE id = 1;", want: "4"},
		{name: "legitimate update of _rowid_", setup: base, script: "CREATE TRIGGER notes_under AFTER UPDATE OF _rowid_ ON notes BEGIN UPDATE notes SET n = 5 WHERE id = NEW.id; END;\n", allow: true, app: "UPDATE notes SET \"_rowid_\" = 4 WHERE id = 1;", want: "5"},
		{name: "legitimate nested triggers", setup: base, script: "CREATE TABLE other (id INTEGER PRIMARY KEY, n INTEGER NOT NULL DEFAULT 0);\nINSERT INTO other (id) VALUES (1);\nCREATE TRIGGER other_au AFTER UPDATE ON other BEGIN UPDATE other SET n = n + 1 WHERE id = NEW.id; END;\nCREATE TRIGGER notes_au AFTER UPDATE OF body ON notes BEGIN UPDATE other SET id = id WHERE id = 1; END;\n", allow: true, app: "UPDATE notes SET body = 'iki' WHERE id = 1;", want: "1"},
	}
	for _, candidate := range cases {
		t.Run(candidate.name, func(t *testing.T) {
			directory := t.TempDir()
			database := filepath.Join(directory, "app.db")
			writePair(t, directory, "0001", "setup", candidate.setup, "SELECT 1;\n")
			if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
				t.Fatal(err)
			}
			before := historySnapshot(t, database)
			writePair(t, directory, "0002", "attack", candidate.script, "SELECT 1;\n")
			_, err := Up(Options{Database: database, Directory: directory})
			if candidate.allow {
				if err != nil {
					t.Fatal(err)
				}
				snapshot := historySnapshot(t, database)
				execDB(t, database, candidate.app)
				if historySnapshot(t, database) != snapshot {
					t.Fatal("application SQL changed migration history")
				}
				query := "SELECT CAST(n AS TEXT) FROM notes"
				if candidate.name == "legitimate nested triggers" {
					query = "SELECT CAST(n AS TEXT) FROM other"
				}
				if got := scalarDB(t, database, query); got != candidate.want {
					t.Fatalf("effect = %s, want %s", got, candidate.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "rejected") {
				t.Fatalf("error = %v", err)
			}
			if historySnapshot(t, database) != before || countHistory(t, database) != 1 {
				t.Fatal("rejected SQL changed migration history")
			}
			for _, name := range candidate.absent {
				if triggerExists(t, database, name) {
					t.Fatalf("rejected trigger %s is still installed", name)
				}
			}
			execDB(t, database, "UPDATE notes SET body = 'later' WHERE id = 1; UPDATE notes SET rowid = rowid WHERE id = 1;")
			if historySnapshot(t, database) != before {
				t.Fatal("application SQL changed history after the rejected trigger")
			}
			if err := os.WriteFile(filepath.Join(directory, "0002_attack.up.sql"), []byte("ALTER TABLE notes ADD COLUMN extra TEXT;\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			applied, err := Up(Options{Database: database, Directory: directory})
			if err != nil || len(applied) != 1 || countHistory(t, database) != 2 || !columnExists(t, database, "notes", "extra") {
				t.Fatalf("corrected migration %+v %v", applied, err)
			}
			for _, name := range candidate.absent {
				if triggerExists(t, database, name) {
					t.Fatalf("corrected database still has trigger %s", name)
				}
			}
		})
	}
}

func TestConnectionPragmasAreRestoredBeforeTheNextLock(t *testing.T) {
	directory := t.TempDir()
	database := filepath.Join(directory, "app.db")
	writePair(t, directory, "0001", "create_notes", "CREATE TABLE notes (id INTEGER PRIMARY KEY);\n", "")
	writePair(t, directory, "0002", "add_body", "ALTER TABLE notes ADD COLUMN body TEXT;\n", "")
	var checks []string
	beforeImmediate = func(conn *sqlite3.Conn) {
		checks = append(checks, pragmaText(conn, "wal_autocheckpoint")+"/"+likeFold(conn)+"/"+pragmaText(conn, "secure_delete")+"/"+pragmaText(conn, "reverse_unordered_selects")+"/"+pragmaText(conn, "temp_store"))
	}
	interruptAfterScript = func(conn *sqlite3.Conn) error {
		// temp_store cannot change inside the open migration transaction.
		// It is restored with the others before the next BEGIN IMMEDIATE.
		return conn.Exec(`
			PRAGMA wal_autocheckpoint = 1;
			PRAGMA case_sensitive_like = ON;
			PRAGMA secure_delete = ON;
			PRAGMA reverse_unordered_selects = ON;
		`)
	}
	defer func() {
		beforeImmediate = nil
		interruptAfterScript = nil
	}()
	if _, err := Up(Options{Database: database, Directory: directory}); err != nil {
		t.Fatal(err)
	}
	if len(checks) < 2 {
		t.Fatalf("checks = %v", checks)
	}
	for _, check := range checks {
		if check != "1000/1/0/0/0" {
			t.Fatalf("connection state before the lock = %s", check)
		}
	}
	outside, err := openSQLite(database, openReadWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer outside.Close()
	if err := outside.Exec(`PRAGMA temp_store = MEMORY`); err != nil {
		t.Fatal(err)
	}
	if err := restoreConnectionPolicy(outside); err != nil {
		t.Fatal(err)
	}
	if got := pragmaText(outside, "temp_store"); got != "0" {
		t.Fatalf("temp_store after restore = %s", got)
	}
}

func likeFold(conn *sqlite3.Conn) string {
	statement, _, err := conn.Prepare(`SELECT 'A' LIKE 'a'`)
	if err != nil {
		return err.Error()
	}
	defer statement.Close()
	if !statement.Step() {
		return "missing"
	}
	return statement.ColumnText(0)
}

func pragmaText(conn *sqlite3.Conn, name string) string {
	statement, _, err := conn.Prepare("PRAGMA " + name)
	if err != nil {
		return err.Error()
	}
	defer statement.Close()
	if !statement.Step() {
		return "missing"
	}
	return statement.ColumnText(0)
}
