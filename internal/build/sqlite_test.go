package build

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// sqliteHelperForTest builds cmd/ahdsqlite into a temporary directory and
// points both the compiler (runtime hint) and the runtime (environment) at it.
func sqliteHelperForTest(t *testing.T) {
	t.Helper()
	helper := filepath.Join(t.TempDir(), "ahdsqlite")
	if runtime.GOOS == "windows" {
		helper += ".exe"
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "build", "-o", helper, "./cmd/ahdsqlite")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("could not build ahdsqlite: %v\n%s", err, output)
	}
	t.Setenv("AHDCODE_SQLITE_RUNTIME", helper)
}

// buildSQLiteProgram compiles one program and returns the executable path.
func buildSQLiteProgram(t *testing.T, source string) string {
	t.Helper()
	directory := writeSources(t, map[string]string{"main.ahd": source})
	output := filepath.Join(t.TempDir(), "program")
	path, result := BuildProgram(filepath.Join(directory, "main.ahd"), output)
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	return path
}

// runIn executes a built program with the given working directory so relative
// database paths land in a temporary directory, never in the repository.
func runIn(t *testing.T, executable, directory string) (string, string, int) {
	t.Helper()
	command := exec.Command(executable)
	command.Dir = directory
	var out, errorOutput strings.Builder
	command.Stdout = &out
	command.Stderr = &errorOutput
	code := 0
	if runError := command.Run(); runError != nil {
		var exit *exec.ExitError
		if !errors.As(runError, &exit) {
			t.Fatalf("could not run the executable: %v", runError)
		}
		code = exit.ExitCode()
	}
	return out.String(), errorOutput.String(), code
}

const sqliteStudentsProgram = `bring SQLite
from SQLite bring Database
from SQLite bring SQLiteValue

db: Database := SQLite.open("students.db")

db.execute(
    """
    CREATE TABLE students (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        score REAL NOT NULL
    )
    """
)

db.execute(
    "INSERT INTO students (name, score) VALUES (?, ?)",
    [
        SQLite.fromString("Ayşe")
        SQLite.fromReal(91.5)
    ]
)

db.execute(
    "INSERT INTO students (name, score) VALUES (?, ?)",
    [
        SQLite.fromString("Deniz")
        SQLite.fromReal(78.0)
    ]
)

rows: List<Pair<String, SQLiteValue>> := db.query(
    "SELECT id, name, score FROM students ORDER BY id"
)

for row in rows {
    write("{row["id"].int()} {row["name"].string()} {row["score"].real()}")
}

changed: Int := db.execute(
    "UPDATE students SET score = ? WHERE name = ?",
    [
        SQLite.fromReal(85.0)
        SQLite.fromString("Deniz")
    ]
)

write(changed)

db.execute(
    "DELETE FROM students WHERE name = ?",
    [SQLite.fromString("Ayşe")]
)

write(db.lastInsertId())
write(db.query("SELECT COUNT(*) AS n FROM students")[0]["n"].int())

db.close()
`

// TestSQLiteBasicCRUDAcceptanceProgram is the frozen acceptance program from
// the v0.3.0 specification, compiled natively and run from a temporary
// directory.
func TestSQLiteBasicCRUDAcceptanceProgram(t *testing.T) {
	sqliteHelperForTest(t)
	executable := buildSQLiteProgram(t, sqliteStudentsProgram)
	directory := t.TempDir()
	out, errorOutput, code := runIn(t, executable, directory)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errorOutput)
	}
	want := "1 Ayşe 91.5\n2 Deniz 78.0\n1\n2\n1\n"
	if out != want {
		t.Fatalf("stdout\n want %q\n have %q\n stderr %s", want, out, errorOutput)
	}
	if _, err := os.Stat(filepath.Join(directory, "students.db")); err != nil {
		t.Fatalf("students.db was not created next to the program: %v", err)
	}
	// A second run against the existing file fails on CREATE TABLE with a
	// catchable-by-class SQLiteError and a clean (trace-free) report.
	out, errorOutput, code = runIn(t, executable, directory)
	if code == 0 || !strings.HasPrefix(errorOutput, "SQLiteError: ") || !strings.Contains(errorOutput, "table students already exists") {
		t.Fatalf("second run: code %d stdout %q stderr %q", code, out, errorOutput)
	}
	if strings.Contains(errorOutput, "goroutine ") {
		t.Fatalf("a Go stack trace leaked: %q", errorOutput)
	}
}

// TestSQLitePersistsAcrossProcesses writes with one native program and reads
// with a different one; only the .db file connects them.
func TestSQLitePersistsAcrossProcesses(t *testing.T) {
	sqliteHelperForTest(t)
	directory := t.TempDir()
	writer := buildSQLiteProgram(t, `bring SQLite
from SQLite bring SQLiteError

db := SQLite.open("notes.db")
db.execute("CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, body TEXT NOT NULL)")
db.execute("INSERT INTO notes (title, body) VALUES (?, ?)", [SQLite.fromString("Alışveriş"), SQLite.fromString("süt, ekmek")])
write(db.lastInsertId())
db.execute("INSERT INTO notes (title, body) VALUES (?, ?)", [SQLite.fromString("Robert'); DROP TABLE notes;--"), SQLite.fromString("")])
write(db.lastInsertId())

db.begin()
attempt {
    db.execute("INSERT INTO notes (title, body) VALUES (?, ?)", [SQLite.fromString("committed"), SQLite.fromString("yes")])
    db.commit()
}
except SQLiteError as error {
    db.rollback()
    write("unexpected: " + error.message)
}

db.begin()
attempt {
    db.execute("INSERT INTO notes (title, body) VALUES (?, ?)", [SQLite.fromString("rolled back"), SQLite.fromString("no")])
    db.execute("INSERT INTO notes (title, body) VALUES (?, ?)", [SQLite.nullValue(), SQLite.fromString("constraint")])
    db.commit()
}
except SQLiteError as error {
    db.rollback()
    write("rolled back: " + error.message)
}
db.close()
`)
	out, errorOutput, code := runIn(t, writer, directory)
	if code != 0 {
		t.Fatalf("writer exit %d: %s", code, errorOutput)
	}
	if want := "1\n2\nrolled back: NOT NULL constraint failed: notes.title\n"; out != want {
		t.Fatalf("writer stdout %q; want %q", out, want)
	}

	reader := buildSQLiteProgram(t, `bring SQLite

db := SQLite.open("notes.db")
for row in db.query("SELECT id, title, body FROM notes ORDER BY id") {
    write("{row["id"].int()}|{row["title"].string()}|{row["body"].string()}")
}
found := db.query("SELECT id FROM notes WHERE title = ?", [SQLite.fromString("Robert'); DROP TABLE notes;--")])
write(len(found))
write(db.query("SELECT COUNT(*) AS n FROM sqlite_master WHERE type = 'table' AND name = 'notes'")[0]["n"].int())
db.close()
db.close()
`)
	out, errorOutput, code = runIn(t, reader, directory)
	if code != 0 {
		t.Fatalf("reader exit %d: %s", code, errorOutput)
	}
	want := "1|Alışveriş|süt, ekmek\n2|Robert'); DROP TABLE notes;--|\n3|committed|yes\n1\n1\n"
	if out != want {
		t.Fatalf("reader stdout\n want %q\n have %q", want, out)
	}
}

// sqliteCheck renders one attempt/except probe: AhdCode has no block lambdas,
// so every probe is spelled out as its own statement group.
func sqliteCheck(label, statement string) string {
	return "attempt {\n    " + statement + "\n    write(\"" + label + ": ok\")\n}\nexcept SQLiteError as error {\n    write(\"" + label + ": SQLiteError \" + error.message)\n}\n"
}

// TestSQLiteErrorsAndValueKindsNatively covers NULL fidelity, storage-class
// mapping, wrong-kind access, BLOB and duplicate-column rejection, use after
// close, and transaction misuse -- every failure as a catchable SQLiteError.
func TestSQLiteErrorsAndValueKindsNatively(t *testing.T) {
	sqliteHelperForTest(t)
	source := `bring SQLite
from SQLite bring Database
from SQLite bring SQLiteValue
from SQLite bring SQLiteError

db: Database := SQLite.open(":memory:")
db.execute("CREATE TABLE people (id INTEGER PRIMARY KEY, nickname TEXT, height REAL, flag BOOLEAN, photo BLOB)")
db.execute("INSERT INTO people (id, nickname, height, flag, photo) VALUES (?, ?, ?, ?, x'00ff')", [SQLite.fromInt(1), SQLite.nullValue(), SQLite.fromReal(1.75), SQLite.fromInt(1)])
db.execute("INSERT INTO people (id, nickname, height, flag) VALUES (?, ?, ?, ?)", [SQLite.fromInt(-2), SQLite.fromString("Çağrı ☕"), SQLite.fromInt(2), SQLite.fromInt(0)])

rows: List<Pair<String, SQLiteValue>> := db.query("SELECT id, nickname, height, flag FROM people ORDER BY id DESC")
for row in rows {
    nickname: Local SQLiteValue := row["nickname"]
    write("{row["id"].int()} {nickname.kind()} {nickname.isNull()} {row["height"].kind()} {row["height"].real()} {row["flag"].kind()}")
}
write(len(rows[0]))
write(rows[1]["nickname"].string())
` +
		sqliteCheck("null as int", `rows[0]["nickname"].int()`) +
		sqliteCheck("int as string", `rows[0]["id"].string()`) +
		sqliteCheck("real as int", `rows[0]["height"].int()`) +
		sqliteCheck("blob", `db.query("SELECT photo FROM people WHERE id = 1")`) +
		sqliteCheck("duplicate", `db.query("SELECT a.id, b.id FROM people a JOIN people b ON a.id = b.id")`) +
		sqliteCheck("aliased", `db.query("SELECT a.id AS a_id, b.id AS b_id FROM people a JOIN people b ON a.id = b.id")`) +
		sqliteCheck("malformed", `db.execute("SELEC 1")`) +
		sqliteCheck("missing table", `db.query("SELECT * FROM ghosts")`) +
		sqliteCheck("unique", `db.execute("INSERT INTO people (id) VALUES (1)")`) +
		sqliteCheck("placeholders", `db.execute("INSERT INTO people (id) VALUES (?)", [])`) +
		sqliteCheck("two statements", `db.execute("DELETE FROM people; DELETE FROM people")`) +
		sqliteCheck("commit without begin", `db.commit()`) +
		sqliteCheck("rollback without begin", `db.rollback()`) +
		"db.begin()\n" +
		sqliteCheck("nested begin", `db.begin()`) +
		sqliteCheck("close in transaction", `db.close()`) +
		"db.rollback()\n" +
		"attempt {\n    SQLite.open(\"no/such/dir/x.db\")\n    write(\"open missing directory: ok\")\n}\nexcept SQLiteError as error {\n    write(\"open missing directory: SQLiteError\")\n    write(error.message.startsWith(\"unable to open database file\"))\n}\n" +
		"db.close()\nalias: Database := db\n" +
		sqliteCheck("execute after close", `alias.execute("SELECT 1")`) +
		sqliteCheck("query after close", `alias.query("SELECT 1")`) +
		sqliteCheck("begin after close", `alias.begin()`) +
		sqliteCheck("lastInsertId after close", `alias.lastInsertId()`) +
		sqliteCheck("close twice", `alias.close()`)
	executable := buildSQLiteProgram(t, source)
	out, errorOutput, code := runIn(t, executable, t.TempDir())
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errorOutput)
	}
	want := strings.Join([]string{
		"1 Null true Real 1.75 Int",
		"-2 String false Real 2.0 Int",
		"4",
		"Çağrı ☕",
		"null as int: SQLiteError int() requires kind Int; this SQLiteValue has kind Null (check kind() first)",
		"int as string: SQLiteError string() requires kind String; this SQLiteValue has kind Int (check kind() first)",
		"real as int: SQLiteError int() requires kind Int; this SQLiteValue has kind Real (check kind() first)",
		`blob: SQLiteError column "photo" holds a BLOB value; BLOB results are not supported by AhdCode SQLite v0.3.0`,
		`duplicate: SQLiteError the result has the duplicate column label "id"; give each column a distinct name with AS`,
		"aliased: ok",
		`malformed: SQLiteError near "SELEC": syntax error`,
		"missing table: SQLiteError no such table: ghosts",
		"unique: SQLiteError UNIQUE constraint failed: people.id",
		"placeholders: SQLiteError the SQL statement has 1 parameter placeholder(s); received 0 value(s)",
		"two statements: SQLiteError execute and query run exactly one SQL statement; split the text into one call per statement",
		"commit without begin: SQLiteError no transaction is active; call begin() before commit()",
		"rollback without begin: SQLiteError no transaction is active; there is nothing to roll back",
		"nested begin: SQLiteError a transaction is already active; call commit() or rollback() before begin()",
		"close in transaction: SQLiteError the Database still has an active transaction; call commit() or rollback() before close()",
		"open missing directory: SQLiteError",
		"true",
		"execute after close: SQLiteError the Database is closed",
		"query after close: SQLiteError the Database is closed",
		"begin after close: SQLiteError the Database is closed",
		"lastInsertId after close: SQLiteError the Database is closed",
		"close twice: ok",
	}, "\n") + "\n"
	if out != want {
		t.Fatalf("stdout\n want:\n%s\n have:\n%s\n stderr: %s", want, out, errorOutput)
	}
}

// TestSQLiteProgramsStayStdlibOnly proves the generated Go workspace has no
// third-party import: the SQLite engine lives in the helper, not the program.
func TestSQLiteProgramsStayStdlibOnly(t *testing.T) {
	sqliteHelperForTest(t)
	directory := writeSources(t, map[string]string{"main.ahd": "bring SQLite\ndb := SQLite.open(\":memory:\")\ndb.close()\n"})
	result := Compile(filepath.Join(directory, "main.ahd"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if result.Program == nil || !result.Program.RequiresSQLite {
		t.Fatal("a program using SQLite must be marked RequiresSQLite")
	}
	var sawRuntime bool
	for _, file := range result.Program.Files {
		if strings.Contains(file.Content, "ncruces") || strings.Contains(file.Content, "modernc") || strings.Contains(file.Content, "database/sql") {
			t.Fatalf("generated file %s imports a SQLite driver directly", file.Name)
		}
		if file.Name == "ahdcode_sqlite_runtime.go" {
			sawRuntime = true
		}
	}
	if !sawRuntime {
		names := make([]string, 0, len(result.Program.Files))
		for _, file := range result.Program.Files {
			names = append(names, file.Name)
		}
		t.Fatalf("the SQLite runtime client was not emitted; files: %v", names)
	}
	// The client file is stdlib-only and always emitted like the other module
	// runtimes; only programs that use SQLite are marked as needing the helper.
	plain := Compile(filepath.Join(writeSources(t, map[string]string{"main.ahd": "write(1)\n"}), "main.ahd"))
	if plain.Program == nil || plain.Program.RequiresSQLite {
		t.Fatal("a program without SQLite must not require the SQLite helper")
	}
}

// TestSQLiteNotesExampleRunsTwiceFromATemporaryDirectory is the documented
// notes-app workflow: the tracked example runs once, creates notes.db, and a
// second run of the same program finds the earlier notes still there.
func TestSQLiteNotesExampleRunsTwiceFromATemporaryDirectory(t *testing.T) {
	sqliteHelperForTest(t)
	entry, err := filepath.Abs(filepath.Join("..", "..", "examples", "v0.3", "01_sqlite_notes.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "notes")
	executable, result := BuildProgram(entry, output)
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	directory := t.TempDir()
	out, errorOutput, code := runIn(t, executable, directory)
	if code != 0 {
		t.Fatalf("first run exit %d: %s", code, errorOutput)
	}
	firstRun := strings.Join([]string{
		"Notes already stored from earlier runs: 0",
		"Added notes #1 and #2",
		"All notes:",
		"  #1 Shopping - milk, bread, tea",
		"  #2 Robert'); DROP TABLE notes;-- - parameters keep this as plain text",
		"Updated 1 note",
		"Deleted 1 note",
		"Notes whose title contains 'Shop': 1",
		"After reopening notes.db:",
		"  #1 Shopping - milk, bread, tea, honey",
	}, "\n") + "\n"
	if out != firstRun {
		t.Fatalf("first run stdout\n want:\n%s\n have:\n%s", firstRun, out)
	}
	out, errorOutput, code = runIn(t, executable, directory)
	if code != 0 {
		t.Fatalf("second run exit %d: %s", code, errorOutput)
	}
	secondRun := strings.Join([]string{
		"Notes already stored from earlier runs: 1",
		"Added notes #3 and #4",
		"All notes:",
		"  #1 Shopping - milk, bread, tea, honey",
		"  #3 Shopping - milk, bread, tea",
		"  #4 Robert'); DROP TABLE notes;-- - parameters keep this as plain text",
		"Updated 1 note",
		"Deleted 1 note",
		"Notes whose title contains 'Shop': 2",
		"After reopening notes.db:",
		"  #1 Shopping - milk, bread, tea, honey",
		"  #3 Shopping - milk, bread, tea, honey",
	}, "\n") + "\n"
	if out != secondRun {
		t.Fatalf("second run stdout\n want:\n%s\n have:\n%s", secondRun, out)
	}
	if _, err := os.Stat(filepath.Join(directory, "notes.db")); err != nil {
		t.Fatalf("notes.db missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join("..", "..", "examples", "v0.3", "notes.db")); !os.IsNotExist(err) {
		t.Fatal("the example wrote notes.db into the repository")
	}
}
