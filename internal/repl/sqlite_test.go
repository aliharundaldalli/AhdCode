package repl

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	sqliteRuntimeOnce sync.Once
	sqliteRuntimePath string
)

// sqliteRuntimeForTest builds the bundled ahdsqlite helper once so the
// persistent evaluator can reach a real SQLite engine during REPL tests.
func sqliteRuntimeForTest(t *testing.T) string {
	t.Helper()
	sqliteRuntimeOnce.Do(func() {
		dir, err := os.MkdirTemp("", "ahdsqlite-test-*")
		if err != nil {
			return
		}
		path := filepath.Join(dir, "ahdsqlite")
		root, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			return
		}
		command := exec.Command("go", "build", "-o", path, "./cmd/ahdsqlite")
		command.Dir = root
		if err := command.Run(); err == nil {
			sqliteRuntimePath = path
		}
	})
	if sqliteRuntimePath == "" {
		t.Skip("ahdsqlite helper could not be built in this environment; skipping SQLite REPL test")
	}
	return sqliteRuntimePath
}

// TestSQLiteDatabaseSurvivesSuccessiveREPLEntries is the required REPL
// workflow: an in-memory Database persists across entries, a failing SQL
// statement leaves the session (and the Database) intact, and SQLiteError is
// catchable exactly like every other Error.
func TestSQLiteDatabaseSurvivesSuccessiveREPLEntries(t *testing.T) {
	t.Setenv("AHDCODE_SQLITE_RUNTIME", sqliteRuntimeForTest(t))
	input := `bring SQLite
from SQLite bring SQLiteError
db := SQLite.open(":memory:")
db.execute("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL, price REAL)")
db.execute("INSERT INTO items (name, price) VALUES (?, ?)", [SQLite.fromString("Çay"), SQLite.fromReal(12.5)])
db.lastInsertId()
db.execute("INSERT INTO items (name) VALUES (?)", [SQLite.fromString("Robert'); DROP TABLE items;--")])
db.query("SELECT * FROM missing")
db.execute("INSERT INTO items (name) VALUES (NULL)")
rows := db.query("SELECT id, name, price FROM items ORDER BY id")
len(rows)
rows[0]["name"].string()
rows[0]["price"].real()
rows[1]["name"].string()
rows[1]["price"].kind()
rows[1]["price"].isNull()
attempt {
    rows[1]["name"].int()
}
except SQLiteError as error {
    write("caught: " + error.message)
}
db.begin()
db.execute("INSERT INTO items (name) VALUES ('temporary')")
db.rollback()
db.query("SELECT COUNT(*) AS n FROM items")[0]["n"].int()
db.close()
attempt {
    db.query("SELECT 1 AS one")
}
except SQLiteError as error {
    write("after close: " + error.message)
}
write("session alive")
`
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(input), &output, &errorOutput, "AhdCode v0.5.0")
	text := output.String()
	for _, want := range []string{
		"1\n",   // lastInsertId
		"2\n",   // len(rows)
		"Çay\n", // rows[0]["name"].string()
		"12.5\n",
		"Robert'); DROP TABLE items;--\n",
		"Null\n",
		"true\n",
		"caught: int() requires kind Int; this SQLiteValue has kind String (check kind() first)\n",
		"after close: the Database is closed\n",
		"session alive\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("REPL output omitted %q:\n%s\nerrors:\n%s", want, text, errorOutput.String())
		}
	}
	errors := errorOutput.String()
	for _, want := range []string{"SQLiteError", "no such table: missing", "NOT NULL constraint failed: items.name"} {
		if !strings.Contains(errors, want) {
			t.Fatalf("REPL error stream omitted %q:\n%s", want, errors)
		}
	}
	// Exactly the two failing SQL entries were reported; the session continued.
	if got := strings.Count(errors, "SQLiteError"); got != 2 {
		t.Fatalf("expected 2 uncaught SQLiteError reports, found %d:\n%s", got, errors)
	}
	if _, err := os.Stat(".ahdcode-repl-session.ahd"); err == nil {
		t.Fatal("REPL left its session file behind")
	}
}

// TestSQLiteFileDatabaseIsReadBackInAFreshREPL proves persistence across
// evaluator sessions: the second Run is a new process-level session that only
// shares the .db file on disk.
func TestSQLiteFileDatabaseIsReadBackInAFreshREPL(t *testing.T) {
	t.Setenv("AHDCODE_SQLITE_RUNTIME", sqliteRuntimeForTest(t))
	path := filepath.ToSlash(filepath.Join(t.TempDir(), "boşluklu dizin", "notes.db"))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	first := `bring SQLite
db := SQLite.open("` + path + `")
db.execute("CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, body TEXT NOT NULL)")
db.execute("INSERT INTO notes (title, body) VALUES (?, ?)", [SQLite.fromString("İlk not"), SQLite.fromString("gövde")])
db.close()
`
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(first), &output, &errorOutput, "AhdCode v0.5.0")
	if errorOutput.Len() != 0 {
		t.Fatalf("first REPL session errors: %s", errorOutput.String())
	}
	second := `bring SQLite
db := SQLite.open("` + path + `")
for row in db.query("SELECT id, title, body FROM notes ORDER BY id") {
    write("{row["id"].int()}: {row["title"].string()} / {row["body"].string()}")
}
db.close()
`
	output.Reset()
	errorOutput.Reset()
	Run(strings.NewReader(second), &output, &errorOutput, "AhdCode v0.5.0")
	if errorOutput.Len() != 0 {
		t.Fatalf("second REPL session errors: %s", errorOutput.String())
	}
	if !strings.Contains(output.String(), "1: İlk not / gövde\n") {
		t.Fatalf("persisted note not read back:\n%s", output.String())
	}
}
