package ahdruntime

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestSQLiteValueEncodingKeepsKindsExact(t *testing.T) {
	real91, err := SQLiteFromReal(91.5)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		text   string
		kind   string
		isNull bool
	}{
		{SQLiteNullValue(), "Null", true},
		{SQLiteFromInt(0), "Int", false},
		{SQLiteFromInt(-7), "Int", false},
		{SQLiteFromInt(math.MaxInt64), "Int", false},
		{SQLiteFromInt(math.MinInt64), "Int", false},
		{real91, "Real", false},
		{SQLiteFromString(""), "String", false},
		{SQLiteFromString("Şişli 'çay' \"☕\"\n;--"), "String", false},
		{SQLiteFromString("42"), "String", false},
	}
	for _, testCase := range cases {
		if kind := AhdSQLiteValueKind(AhdClassSQLiteError, testCase.text); kind != testCase.kind {
			t.Fatalf("kind(%q) = %q; want %q", testCase.text, kind, testCase.kind)
		}
		if isNull := AhdSQLiteValueIsNull(AhdClassSQLiteError, testCase.text); isNull != testCase.isNull {
			t.Fatalf("isNull(%q) = %v; want %v", testCase.text, isNull, testCase.isNull)
		}
	}
	if AhdSQLiteValueInt(AhdClassSQLiteError, SQLiteFromInt(math.MinInt64)) != math.MinInt64 {
		t.Fatal("Int64 minimum did not round-trip")
	}
	if AhdSQLiteValueReal(AhdClassSQLiteError, real91) != 91.5 {
		t.Fatal("Real did not round-trip")
	}
	if AhdSQLiteValueReal(AhdClassSQLiteError, SQLiteFromInt(3)) != 3.0 {
		t.Fatal("real() must widen an Int exactly like a Real := Int assignment")
	}
	if AhdSQLiteValueString(AhdClassSQLiteError, SQLiteFromString("Şişli 'çay' \"☕\"\n;--")) != "Şişli 'çay' \"☕\"\n;--" {
		t.Fatal("String did not round-trip")
	}
	if _, err := SQLiteFromReal(math.NaN()); err == nil {
		t.Fatal("NaN was accepted as a Real")
	}
	if _, err := SQLiteFromReal(math.Inf(-1)); err == nil {
		t.Fatal("-Inf was accepted as a Real")
	}
	expectRaise(t, AhdClassSQLiteError, func() { AhdSQLiteFromReal(AhdClassSQLiteError, math.Inf(1)) })
}

func TestSQLiteValueWrongKindAccessRaisesInsteadOfConverting(t *testing.T) {
	message := func(body func()) (text string) {
		defer func() {
			signal, ok := recover().(*AhdSignal)
			if !ok {
				t.Fatal("expected an AhdSignal")
			}
			if signal.Instance.AhdClassOf() != AhdClassSQLiteError {
				t.Fatalf("expected SQLiteError; received %s", signal.Instance.AhdClassOf().Name)
			}
			text = signal.Message
		}()
		body()
		return ""
	}
	if got := message(func() { AhdSQLiteValueInt(AhdClassSQLiteError, SQLiteFromString("42")) }); !strings.Contains(got, "int() requires kind Int; this SQLiteValue has kind String") {
		t.Fatalf("String -> int() message %q", got)
	}
	real3, _ := SQLiteFromReal(3.0)
	if got := message(func() { AhdSQLiteValueInt(AhdClassSQLiteError, real3) }); !strings.Contains(got, "has kind Real") {
		t.Fatalf("Real -> int() message %q", got)
	}
	if got := message(func() { AhdSQLiteValueString(AhdClassSQLiteError, SQLiteFromInt(42)) }); !strings.Contains(got, "string() requires kind String") {
		t.Fatalf("Int -> string() message %q", got)
	}
	if got := message(func() { AhdSQLiteValueReal(AhdClassSQLiteError, SQLiteFromString("1.5")) }); !strings.Contains(got, "real() requires kind Real or Int") {
		t.Fatalf("String -> real() message %q", got)
	}
	for _, accessor := range []func(){
		func() { AhdSQLiteValueInt(AhdClassSQLiteError, SQLiteNullValue()) },
		func() { AhdSQLiteValueReal(AhdClassSQLiteError, SQLiteNullValue()) },
		func() { AhdSQLiteValueString(AhdClassSQLiteError, SQLiteNullValue()) },
	} {
		if got := message(accessor); !strings.Contains(got, "has kind Null (check kind() first)") {
			t.Fatalf("Null access message %q", got)
		}
	}
	for _, corrupted := range []string{"", "X1", "Iabc", "R1e999", "Rnan"} {
		expectRaise(t, AhdClassSQLiteError, func() { AhdSQLiteValueKind(AhdClassSQLiteError, corrupted) })
	}
}

// sqliteRepositoryRoot is captured before any test changes the working
// directory, so the helper can always be built from the module root.
var sqliteRepositoryRoot = func() string {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		panic(err)
	}
	return root
}()

// buildSQLiteHelper compiles cmd/ahdsqlite once into a temporary directory and
// points the runtime at it through AHDCODE_SQLITE_RUNTIME.
func buildSQLiteHelper(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}
	helper := filepath.Join(t.TempDir(), "ahdsqlite")
	if runtime.GOOS == "windows" {
		helper += ".exe"
	}
	command := exec.Command("go", "build", "-o", helper, "./cmd/ahdsqlite")
	command.Dir = sqliteRepositoryRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("could not build ahdsqlite: %v\n%s", err, output)
	}
	t.Setenv("AHDCODE_SQLITE_RUNTIME", helper)
}

func TestSQLiteRuntimeDrivesTheHelperEndToEnd(t *testing.T) {
	buildSQLiteHelper(t)
	directory := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	class := AhdClassSQLiteError

	// A relative path is resolved against this process's working directory.
	db := AhdSQLiteOpen(class, "notes.db")
	if AhdSQLiteExecute(class, db, "CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, body TEXT, score REAL)", nil) != 0 {
		t.Fatal("CREATE TABLE reported changes")
	}
	score, _ := SQLiteFromReal(9.5)
	if AhdSQLiteExecute(class, db, "INSERT INTO notes (title, body, score) VALUES (?, ?, ?)", []string{SQLiteFromString("Robert'); DROP TABLE notes;--"), SQLiteNullValue(), score}) != 1 {
		t.Fatal("INSERT did not report one change")
	}
	if AhdSQLiteLastInsertID(class, db) != 1 {
		t.Fatal("lastInsertId is not 1")
	}
	AhdSQLiteExecute(class, db, "INSERT INTO notes (title, body, score) VALUES (?, ?, ?)", []string{SQLiteFromString("İkinci"), SQLiteFromString("gövde"), SQLiteFromInt(7)})
	columns, rows := AhdSQLiteQuery(class, db, "SELECT id, title, body, score FROM notes WHERE id >= ? ORDER BY id", []string{SQLiteFromInt(1)})
	if strings.Join(columns, ",") != "id,title,body,score" || len(rows) != 2 {
		t.Fatalf("query shape %v %v", columns, rows)
	}
	if rows[0][1] != SQLiteFromString("Robert'); DROP TABLE notes;--") || rows[0][2] != SQLiteNullValue() || rows[0][3] != score {
		t.Fatalf("first row %v", rows[0])
	}
	// The REAL-affinity column stored the Int 7 as REAL 7.0, exactly as SQLite does.
	if AhdSQLiteValueKind(class, rows[1][3]) != "Real" || AhdSQLiteValueReal(class, rows[1][3]) != 7.0 {
		t.Fatalf("REAL affinity column %v", rows[1][3])
	}
	if AhdSQLiteExecute(class, db, "UPDATE notes SET body = ? WHERE id = ?", []string{SQLiteFromString("yeni"), SQLiteFromInt(2)}) != 1 {
		t.Fatal("UPDATE changed a different number of rows")
	}
	if AhdSQLiteExecute(class, db, "DELETE FROM notes WHERE id = ?", []string{SQLiteFromInt(1)}) != 1 {
		t.Fatal("DELETE changed a different number of rows")
	}
	_, empty := AhdSQLiteQuery(class, db, "SELECT id FROM notes WHERE id = 1", []string{})
	if len(empty) != 0 {
		t.Fatal("deleted row still visible")
	}

	AhdSQLiteBegin(class, db)
	AhdSQLiteExecute(class, db, "INSERT INTO notes (title) VALUES ('geçici')", nil)
	AhdSQLiteRollback(class, db)
	_, remaining := AhdSQLiteQuery(class, db, "SELECT COUNT(*) AS n FROM notes", nil)
	if AhdSQLiteValueInt(class, remaining[0][0]) != 1 {
		t.Fatal("rollback did not discard the insert")
	}
	expectRaise(t, class, func() { AhdSQLiteCommit(class, db) })
	AhdSQLiteBegin(class, db)
	expectRaise(t, class, func() { AhdSQLiteBegin(class, db) })
	expectRaise(t, class, func() { AhdSQLiteClose(class, db) })
	AhdSQLiteCommit(class, db)

	// Errors keep SQLite's text and become SQLiteError.
	expectRaise(t, class, func() { AhdSQLiteQuery(class, db, "SELECT * FROM missing", nil) })
	expectRaise(t, class, func() { AhdSQLiteExecute(class, db, "INSERT INTO notes (title) VALUES (NULL)", nil) })
	expectRaise(t, class, func() { AhdSQLiteExecute(class, db, "INSERT INTO notes (title) VALUES (?)", nil) })
	expectRaise(t, class, func() { AhdSQLiteQuery(class, db, "SELECT x'00' AS blob", nil) })
	expectRaise(t, class, func() { AhdSQLiteQuery(class, db, "SELECT 1 AS a, 2 AS a", nil) })

	AhdSQLiteClose(class, db)
	AhdSQLiteClose(class, db)
	expectRaise(t, class, func() { AhdSQLiteExecute(class, db, "SELECT 1", nil) })
	expectRaise(t, class, func() { AhdSQLiteQuery(class, db, "SELECT 1", nil) })
	expectRaise(t, class, func() { AhdSQLiteBegin(class, db) })
	expectRaise(t, class, func() { AhdSQLiteLastInsertID(class, db) })

	// The file is real and reopens with its data; a second handle is a new connection.
	if _, err := os.Stat(filepath.Join(directory, "notes.db")); err != nil {
		t.Fatalf("database file missing: %v", err)
	}
	reopened := AhdSQLiteOpen(class, filepath.Join(directory, "notes.db"))
	_, titles := AhdSQLiteQuery(class, reopened, "SELECT title, body FROM notes ORDER BY id", nil)
	if len(titles) != 1 || titles[0][0] != SQLiteFromString("İkinci") || titles[0][1] != SQLiteFromString("yeni") {
		t.Fatalf("reopened rows %v", titles)
	}
	AhdSQLiteClose(class, reopened)

	expectRaise(t, class, func() { AhdSQLiteOpen(class, filepath.Join(directory, "missing", "x.db")) })
	expectRaise(t, class, func() { AhdSQLiteOpen(class, "") })
	expectRaise(t, class, func() { AhdSQLiteExecute(class, "not-a-handle", "SELECT 1", nil) })
	expectRaise(t, class, func() { AhdSQLiteExecute(class, "1", "SELECT ?", []string{"Xbad"}) })
}

func TestSQLiteMemoryDatabasesAreIndependentConnections(t *testing.T) {
	buildSQLiteHelper(t)
	class := AhdClassSQLiteError
	first := AhdSQLiteOpen(class, ":memory:")
	second := AhdSQLiteOpen(class, ":memory:")
	if first == second {
		t.Fatal("two :memory: databases share one handle")
	}
	AhdSQLiteExecute(class, first, "CREATE TABLE t (v)", nil)
	expectRaise(t, class, func() { AhdSQLiteQuery(class, second, "SELECT * FROM t", nil) })
	AhdSQLiteClose(class, first)
	AhdSQLiteClose(class, second)

	// The helper session is shared by every goroutine of the program.
	shared := AhdSQLiteOpen(class, ":memory:")
	AhdSQLiteExecute(class, shared, "CREATE TABLE t (v INTEGER)", nil)
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			for index := 0; index < 20; index++ {
				AhdSQLiteExecute(class, shared, "INSERT INTO t VALUES (?)", []string{SQLiteFromInt(int64(worker*100 + index))})
			}
		}(worker)
	}
	group.Wait()
	_, rows := AhdSQLiteQuery(class, shared, "SELECT COUNT(*) AS n FROM t", nil)
	if AhdSQLiteValueInt(class, rows[0][0]) != 160 {
		t.Fatalf("concurrent inserts produced %v rows", rows[0][0])
	}
	AhdSQLiteClose(class, shared)
}
