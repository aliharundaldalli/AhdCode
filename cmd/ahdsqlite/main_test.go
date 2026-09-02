package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/sqliteproto"
)

func newEngine() *server {
	return &server{databases: make(map[int64]*database), next: 1}
}

func openDatabase(t *testing.T, engine *server, path string) int64 {
	t.Helper()
	response := engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationOpen, Path: path})
	if response.Error != "" {
		t.Fatalf("open(%q) failed: %s", path, response.Error)
	}
	return response.Database
}

func execute(t *testing.T, engine *server, db int64, sql string, parameters ...sqliteproto.Value) int64 {
	t.Helper()
	response := engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationExecute, Database: db, SQL: sql, Parameters: parameters})
	if response.Error != "" {
		t.Fatalf("execute(%q) failed: %s", sql, response.Error)
	}
	return response.Changed
}

func query(t *testing.T, engine *server, db int64, sql string, parameters ...sqliteproto.Value) sqliteproto.Response {
	t.Helper()
	response := engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationQuery, Database: db, SQL: sql, Parameters: parameters})
	if response.Error != "" {
		t.Fatalf("query(%q) failed: %s", sql, response.Error)
	}
	return response
}

func expectError(t *testing.T, response sqliteproto.Response, fragment string) {
	t.Helper()
	if response.Error == "" {
		t.Fatalf("expected an error containing %q; the operation succeeded: %+v", fragment, response)
	}
	if !strings.Contains(response.Error, fragment) {
		t.Fatalf("error %q does not mention %q", response.Error, fragment)
	}
}

func simple(engine *server, operation string, db int64) sqliteproto.Response {
	return engine.handle(sqliteproto.Request{Operation: operation, Database: db})
}

func integer(value int64) sqliteproto.Value {
	return sqliteproto.Value{Kind: sqliteproto.KindInt, Int: value}
}
func real(value float64) sqliteproto.Value {
	return sqliteproto.Value{Kind: sqliteproto.KindReal, Real: value}
}
func text(value string) sqliteproto.Value {
	return sqliteproto.Value{Kind: sqliteproto.KindString, String: value}
}
func null() sqliteproto.Value { return sqliteproto.Value{Kind: sqliteproto.KindNull} }
func count(t *testing.T, engine *server, db int64, table string) int64 {
	t.Helper()
	return query(t, engine, db, "SELECT COUNT(*) AS n FROM "+table).Rows[0][0].Int
}

func TestStorageClassesMapExactly(t *testing.T) {
	engine := newEngine()
	db := openDatabase(t, engine, ":memory:")
	// Declared types are deliberately misleading: only the runtime storage
	// class of each value may decide the SQLiteValue kind.
	execute(t, engine, db, "CREATE TABLE v (i INTEGER, r REAL, s TEXT, n TEXT, b BOOLEAN, d DATE, dyn)")
	execute(t, engine, db, "INSERT INTO v VALUES (?, ?, ?, ?, ?, ?, ?)",
		integer(math.MinInt64), real(-2.5), text(""), null(), integer(1), text("2026-09-02"), text("12"))
	execute(t, engine, db, "INSERT INTO v VALUES (?, ?, ?, ?, ?, ?, ?)",
		integer(math.MaxInt64), real(0.1), text("Şişli çay ☕"), text("not null"), integer(0), null(), integer(0))
	response := query(t, engine, db, "SELECT i, r, s, n, b, d, dyn FROM v ORDER BY i")
	if strings.Join(response.Columns, ",") != "i,r,s,n,b,d,dyn" {
		t.Fatalf("columns %v", response.Columns)
	}
	want := [][]sqliteproto.Value{
		{integer(math.MinInt64), real(-2.5), text(""), null(), integer(1), text("2026-09-02"), text("12")},
		{integer(math.MaxInt64), real(0.1), text("Şişli çay ☕"), text("not null"), integer(0), null(), integer(0)},
	}
	have, _ := json.Marshal(response.Rows)
	expected, _ := json.Marshal(want)
	if !bytes.Equal(have, expected) {
		t.Fatalf("rows\n have %s\n want %s", have, expected)
	}
	// Arithmetic that produces REAL from INTEGER columns is reported as Real.
	if row := query(t, engine, db, "SELECT 1 / 2.0 AS half, 7 % 3 AS rest, 'a' || 'b' AS glued").Rows[0]; row[0] != real(0.5) || row[1] != integer(1) || row[2] != text("ab") {
		t.Fatalf("expression storage classes %+v", row)
	}
}

func TestParametersAreBoundNotInterpolated(t *testing.T) {
	engine := newEngine()
	db := openDatabase(t, engine, ":memory:")
	execute(t, engine, db, "CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL)")
	hostile := []string{
		"Robert'); DROP TABLE notes;--",
		`she said "hi"; -- and left`,
		"line one\nline two\\n not a newline",
		"back\\slash and 'quote'",
		"İstanbul'da çağ ğüşıöç",
		"emoji 🎉🚀 done",
		"",
		"?",
		"'; SELECT 1; '",
	}
	for _, title := range hostile {
		if changed := execute(t, engine, db, "INSERT INTO notes (title) VALUES (?)", text(title)); changed != 1 {
			t.Fatalf("insert of %q changed %d rows", title, changed)
		}
	}
	response := query(t, engine, db, "SELECT title FROM notes ORDER BY id")
	if len(response.Rows) != len(hostile) {
		t.Fatalf("stored %d rows; want %d", len(response.Rows), len(hostile))
	}
	for index, title := range hostile {
		if response.Rows[index][0] != text(title) {
			t.Fatalf("row %d stored %q; want %q", index, response.Rows[index][0].String, title)
		}
	}
	// The table survived the "DROP TABLE" text and nothing else was executed.
	if n := query(t, engine, db, "SELECT COUNT(*) AS n FROM sqlite_master WHERE type = 'table' AND name = 'notes'").Rows[0][0]; n != integer(1) {
		t.Fatalf("notes table lookup returned %+v", n)
	}
	exact := query(t, engine, db, "SELECT id FROM notes WHERE title = ?", text("Robert'); DROP TABLE notes;--"))
	if len(exact.Rows) != 1 || exact.Rows[0][0] != integer(1) {
		t.Fatalf("parameterised lookup returned %+v", exact.Rows)
	}
}

func TestParameterCountAndKindsAreChecked(t *testing.T) {
	engine := newEngine()
	db := openDatabase(t, engine, ":memory:")
	execute(t, engine, db, "CREATE TABLE t (a, b)")
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationExecute, Database: db, SQL: "INSERT INTO t VALUES (?, ?)", Parameters: []sqliteproto.Value{integer(1)}}), "2 parameter placeholder(s); received 1")
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationExecute, Database: db, SQL: "INSERT INTO t VALUES (1, 2)", Parameters: []sqliteproto.Value{integer(1)}}), "0 parameter placeholder(s); received 1")
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationExecute, Database: db, SQL: "INSERT INTO t VALUES (?, ?)", Parameters: []sqliteproto.Value{integer(1), real(math.Inf(1))}}), "not a finite Real")
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationExecute, Database: db, SQL: "INSERT INTO t VALUES (?, ?)", Parameters: []sqliteproto.Value{integer(1), {Kind: "Blob"}}}), "unsupported kind")
	if count(t, engine, db, "t") != 0 {
		t.Fatal("a rejected statement still inserted rows")
	}
}

func TestOneCallIsOneStatement(t *testing.T) {
	engine := newEngine()
	db := openDatabase(t, engine, ":memory:")
	execute(t, engine, db, "CREATE TABLE t (a)")
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationExecute, Database: db, SQL: "INSERT INTO t VALUES (1); INSERT INTO t VALUES (2)"}), "exactly one SQL statement")
	if count(t, engine, db, "t") != 0 {
		t.Fatal("the rejected multi-statement text still ran")
	}
	// Trailing semicolons, whitespace, and comments are still one statement.
	execute(t, engine, db, "INSERT INTO t VALUES (1);   ")
	execute(t, engine, db, "INSERT INTO t VALUES (2); -- done")
	if count(t, engine, db, "t") != 2 {
		t.Fatal("single statements with trailing semicolons were not executed")
	}
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationExecute, Database: db, SQL: "   -- nothing here"}), "contains no statement")
	if response := engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationExecute, Database: db, SQL: "SELEC 1"}); response.Error != `near "SELEC": syntax error` {
		t.Fatalf("syntax error text %q", response.Error)
	}
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationQuery, Database: db, SQL: "SELECT * FROM missing"}), "no such table: missing")
}

func TestExecuteReportsChangedRows(t *testing.T) {
	engine := newEngine()
	db := openDatabase(t, engine, ":memory:")
	if changed := execute(t, engine, db, "CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)"); changed != 0 {
		t.Fatalf("DDL reported %d changes", changed)
	}
	for _, v := range []string{"a", "b", "c"} {
		execute(t, engine, db, "INSERT INTO t (v) VALUES (?)", text(v))
	}
	if changed := execute(t, engine, db, "UPDATE t SET v = upper(v) WHERE id > ?", integer(1)); changed != 2 {
		t.Fatalf("UPDATE reported %d changes; want 2", changed)
	}
	if changed := execute(t, engine, db, "UPDATE t SET v = v WHERE id > 100"); changed != 0 {
		t.Fatalf("no-match UPDATE reported %d changes", changed)
	}
	if changed := execute(t, engine, db, "DELETE FROM t"); changed != 3 {
		t.Fatalf("DELETE reported %d changes; want 3", changed)
	}
	// A query result is fully materialized; execute on a SELECT changes nothing.
	if changed := execute(t, engine, db, "SELECT 1"); changed != 0 {
		t.Fatalf("SELECT via execute reported %d changes", changed)
	}
}

func TestLastInsertIDIsConnectionLocal(t *testing.T) {
	engine := newEngine()
	first := openDatabase(t, engine, ":memory:")
	second := openDatabase(t, engine, ":memory:")
	execute(t, engine, first, "CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL)")
	execute(t, engine, first, "INSERT INTO notes (title) VALUES ('a')")
	execute(t, engine, first, "INSERT INTO notes (title) VALUES ('b')")
	execute(t, engine, first, "INSERT INTO notes (id, title) VALUES (40, 'c')")
	if response := simple(engine, sqliteproto.OperationLastInsertID, first); response.RowID != 40 {
		t.Fatalf("lastInsertId %+v; want 40", response)
	}
	if response := simple(engine, sqliteproto.OperationLastInsertID, second); response.Error != "" || response.RowID != 0 {
		t.Fatalf("a different Database saw another connection's row id: %+v", response)
	}
	execute(t, engine, first, "INSERT INTO notes (title) VALUES ('d')")
	if response := simple(engine, sqliteproto.OperationLastInsertID, first); response.RowID != 41 {
		t.Fatalf("AUTOINCREMENT continued at %d; want 41", response.RowID)
	}
}

func TestTransactionsCommitRollbackAndRejectMisuse(t *testing.T) {
	engine := newEngine()
	db := openDatabase(t, engine, ":memory:")
	execute(t, engine, db, "CREATE TABLE t (v TEXT UNIQUE)")

	if r := simple(engine, sqliteproto.OperationBegin, db); r.Error != "" {
		t.Fatal(r.Error)
	}
	execute(t, engine, db, "INSERT INTO t VALUES ('A')")
	execute(t, engine, db, "INSERT INTO t VALUES ('B')")
	if count(t, engine, db, "t") != 2 {
		t.Fatal("queries inside the transaction do not see its writes")
	}
	if r := simple(engine, sqliteproto.OperationCommit, db); r.Error != "" {
		t.Fatal(r.Error)
	}
	if count(t, engine, db, "t") != 2 {
		t.Fatal("commit lost rows")
	}

	simple(engine, sqliteproto.OperationBegin, db)
	execute(t, engine, db, "INSERT INTO t VALUES ('C')")
	if r := simple(engine, sqliteproto.OperationRollback, db); r.Error != "" {
		t.Fatal(r.Error)
	}
	if count(t, engine, db, "t") != 2 {
		t.Fatal("rollback kept rows")
	}

	// A failing statement inside the transaction does not end it; rollback
	// removes the earlier valid insert too.
	simple(engine, sqliteproto.OperationBegin, db)
	execute(t, engine, db, "INSERT INTO t VALUES ('D')")
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationExecute, Database: db, SQL: "INSERT INTO t VALUES ('A')"}), "UNIQUE constraint failed")
	expectError(t, simple(engine, sqliteproto.OperationBegin, db), "already active")
	if r := simple(engine, sqliteproto.OperationRollback, db); r.Error != "" {
		t.Fatal(r.Error)
	}
	if count(t, engine, db, "t") != 2 {
		t.Fatal("rollback after a failed statement did not remove the transaction's earlier insert")
	}

	expectError(t, simple(engine, sqliteproto.OperationCommit, db), "no transaction is active")
	expectError(t, simple(engine, sqliteproto.OperationRollback, db), "no transaction is active")

	// close() during a transaction must neither commit nor rollback silently.
	simple(engine, sqliteproto.OperationBegin, db)
	execute(t, engine, db, "INSERT INTO t VALUES ('E')")
	expectError(t, simple(engine, sqliteproto.OperationClose, db), "active transaction")
	if count(t, engine, db, "t") != 3 {
		t.Fatal("the rejected close changed the transaction state")
	}
	simple(engine, sqliteproto.OperationRollback, db)
	if r := simple(engine, sqliteproto.OperationClose, db); r.Error != "" {
		t.Fatal(r.Error)
	}
}

func TestCloseIsIdempotentAndUseAfterCloseFails(t *testing.T) {
	engine := newEngine()
	db := openDatabase(t, engine, ":memory:")
	if r := simple(engine, sqliteproto.OperationClose, db); r.Error != "" {
		t.Fatal(r.Error)
	}
	if r := simple(engine, sqliteproto.OperationClose, db); r.Error != "" {
		t.Fatalf("second close failed: %s", r.Error)
	}
	for _, operation := range []string{sqliteproto.OperationExecute, sqliteproto.OperationQuery, sqliteproto.OperationBegin, sqliteproto.OperationCommit, sqliteproto.OperationRollback, sqliteproto.OperationLastInsertID} {
		expectError(t, engine.handle(sqliteproto.Request{Operation: operation, Database: db, SQL: "SELECT 1"}), "the Database is closed")
	}
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationQuery, Database: 99, SQL: "SELECT 1"}), "unknown Database handle")
	expectError(t, engine.handle(sqliteproto.Request{Operation: "vacuum", Database: openDatabase(t, engine, ":memory:")}), "unknown SQLite helper operation")
}

func TestQueryShapeRejectsDuplicateLabelsAndBlobs(t *testing.T) {
	engine := newEngine()
	db := openDatabase(t, engine, ":memory:")
	execute(t, engine, db, "CREATE TABLE a (id INTEGER PRIMARY KEY, name TEXT)")
	execute(t, engine, db, "CREATE TABLE b (id INTEGER PRIMARY KEY, a_id INTEGER, payload BLOB)")
	execute(t, engine, db, "INSERT INTO a VALUES (1, 'one')")
	execute(t, engine, db, "INSERT INTO b VALUES (7, 1, x'00ff')")
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationQuery, Database: db, SQL: "SELECT a.id, b.id FROM a JOIN b ON b.a_id = a.id"}), `duplicate column label "id"`)
	aliased := query(t, engine, db, "SELECT a.id AS a_id, b.id AS b_id, a.name FROM a JOIN b ON b.a_id = a.id")
	if strings.Join(aliased.Columns, ",") != "a_id,b_id,name" || aliased.Rows[0][0] != integer(1) || aliased.Rows[0][1] != integer(7) {
		t.Fatalf("aliased join %+v", aliased)
	}
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationQuery, Database: db, SQL: "SELECT payload FROM b"}), `column "payload" holds a BLOB value`)
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationQuery, Database: db, SQL: "SELECT x'41' AS raw"}), "BLOB")
	// Non-BLOB columns of the same table remain readable.
	if row := query(t, engine, db, "SELECT id, a_id FROM b").Rows[0]; row[0] != integer(7) || row[1] != integer(1) {
		t.Fatalf("plain columns %+v", row)
	}
	// Empty results still report the column list.
	empty := query(t, engine, db, "SELECT id, name FROM a WHERE id > 100")
	if strings.Join(empty.Columns, ",") != "id,name" || len(empty.Rows) != 0 {
		t.Fatalf("empty result %+v", empty)
	}
}

func TestNonFiniteRealsAreRejected(t *testing.T) {
	engine := newEngine()
	db := openDatabase(t, engine, ":memory:")
	// 1e308 * 10 overflows to +Inf inside SQLite.
	expectError(t, engine.handle(sqliteproto.Request{Operation: sqliteproto.OperationQuery, Database: db, SQL: "SELECT 1e308 * 10 AS huge"}), "non-finite REAL")
}

func TestFileDatabasesPersistAndRejectMissingDirectories(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "boşluklu klasör", "notlar.db")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	engine := newEngine()
	db := openDatabase(t, engine, path)
	execute(t, engine, db, "CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL)")
	execute(t, engine, db, "INSERT INTO notes (title) VALUES (?)", text("kalıcı not"))
	simple(engine, sqliteproto.OperationClose, db)
	engine.closeAll()

	again := newEngine()
	reopened := openDatabase(t, again, path)
	if row := query(t, again, reopened, "SELECT id, title FROM notes").Rows[0]; row[0] != integer(1) || row[1] != text("kalıcı not") {
		t.Fatalf("reopened data %+v", row)
	}
	simple(again, sqliteproto.OperationClose, reopened)

	expectError(t, again.handle(sqliteproto.Request{Operation: sqliteproto.OperationOpen, Path: filepath.Join(directory, "missing", "x.db")}), "unable to open database file")
	expectError(t, again.handle(sqliteproto.Request{Operation: sqliteproto.OperationOpen, Path: ""}), "path is empty")
	if _, err := os.Stat(filepath.Join(directory, "missing", "x.db")); !os.IsNotExist(err) {
		t.Fatal("open created a parent directory")
	}
}

func TestServeSpeaksOneJSONLinePerRequest(t *testing.T) {
	requests := []sqliteproto.Request{
		{Operation: sqliteproto.OperationOpen, Path: ":memory:"},
		{Operation: sqliteproto.OperationExecute, Database: 1, SQL: "CREATE TABLE t (v TEXT)"},
		{Operation: sqliteproto.OperationExecute, Database: 1, SQL: "INSERT INTO t VALUES (?)", Parameters: []sqliteproto.Value{text("<html> & ünïcode")}},
		{Operation: sqliteproto.OperationQuery, Database: 1, SQL: "SELECT v FROM t"},
		{Operation: sqliteproto.OperationQuery, Database: 1, SQL: "SELECT * FROM nope"},
		{Operation: sqliteproto.OperationClose, Database: 1},
	}
	var input bytes.Buffer
	encoder := json.NewEncoder(&input)
	for _, request := range requests {
		if err := encoder.Encode(request); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	serve(&input, &output)
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != len(requests) {
		t.Fatalf("received %d response lines for %d requests:\n%s", len(lines), len(requests), output.String())
	}
	var responses []sqliteproto.Response
	for _, line := range lines {
		var response sqliteproto.Response
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("response %q is not JSON: %v", line, err)
		}
		responses = append(responses, response)
	}
	if responses[0].Database != 1 || responses[1].Error != "" || responses[2].Changed != 1 {
		t.Fatalf("unexpected responses %+v", responses[:3])
	}
	if responses[3].Rows[0][0] != text("<html> & ünïcode") || strings.Contains(lines[3], `\u003c`) {
		t.Fatalf("query response %s", lines[3])
	}
	if !strings.Contains(responses[4].Error, "no such table: nope") || responses[5].Error != "" {
		t.Fatalf("error/close responses %+v", responses[4:])
	}
	if responses[4].Error != "no such table: nope" {
		t.Fatalf("driver/result-code prefix leaked: %q", responses[4].Error)
	}
	// Malformed input is answered and ends the session without panicking.
	output.Reset()
	serve(strings.NewReader("{not json"), &output)
	if !strings.Contains(output.String(), "malformed SQLite helper request") {
		t.Fatalf("malformed input response %q", output.String())
	}
}
