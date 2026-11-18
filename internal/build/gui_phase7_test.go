package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTableSortAgrees(t *testing.T) {
	useHeadlessGUI(t, "")
	program := `bring GUI
from GUI bring (Window, Container, TableView, GUIError)

window: Window := GUI.window(title: "Sort", width: 360, height: 240)
form: Container := window.column()
table: TableView := form.table(["Name"], [["b"], ["a"], ["b"]])
table.selectRow(0)
table.onSelect(lambda (row: Int?) -> write("selected"))
table.sort(column: 0, descending: true)
for row in table.rows() {
    write(row[0])
}
column: Int? := table.sortedColumn()
write("{column}")
if table.sortDescending() {
    write("desc")
}
else {
    write("asc")
}
table.setRows([["c"], ["a"]])
for row in table.rows() {
    write(row[0])
}
attempt {
    table.sort(5)
} except GUIError as error {
    write(error.message)
}
window.close()
`
	assertGUI(t, program, "b\nb\na\n0\ndesc\nc\na\ncolumn 5 is outside the 1 columns\n")
}

func TestTableHeaderSortAgrees(t *testing.T) {
	useHeadlessGUI(t, `[{"event":"sort","widget":2,"selected":0}]`)
	program := `bring GUI
from GUI bring (Window, Container, TableView)

window: Window := GUI.window(title: "Header", width: 360, height: 240)
form: Container := window.column()
table: TableView := form.table(["Name", "Note"], [["b", ""], ["a", "m"], ["b", "m"]])
table.selectRow(0)
table.onSelect(lambda (row: Int?) -> write("selected"))
window.wait()
row: Int? := table.selectedRow()
write("{row}")
for line in table.rows() {
    write("{line[0]} {line[1]}")
}
if table.sortDescending() {
    write("desc")
}
else {
    write("asc")
}
`
	assertGUI(t, program, "1\na m\nb \nb m\nasc\n")
}

func TestHeaderSortFollowsTheOpenWindow(t *testing.T) {
	useHeadlessGUI(t, `[{"event":"sort","widget":2,"selected":0}]`)
	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, `bring GUI
window := GUI.window(title: "Earlier", width: 200, height: 120)
form := window.column()
form.label("not a table")
window.close()
`, &output, &errorOutput)
	program := `bring GUI
from GUI bring (Window, Container, TableView)

window: Window := GUI.window(title: "Header", width: 360, height: 240)
form: Container := window.column()
table: TableView := form.table(["Name", "Note"], [["b", ""], ["a", "m"], ["b", "m"]])
table.selectRow(0)
table.onSelect(lambda (row: Int?) -> write("selected"))
window.wait()
row: Int? := table.selectedRow()
write("{row}")
for line in table.rows() {
    write("{line[0]} {line[1]}")
}
if table.sortDescending() {
    write("desc")
}
else {
    write("asc")
}
`
	assertGUI(t, program, "1\na m\nb \nb m\nasc\n")
}

func TestLedgerRecordsSurviveReopenAndSort(t *testing.T) {
	sqliteHelperForTest(t)
	useHeadlessGUI(t, `[{"event":"sort","widget":2,"selected":0}]`)
	database := filepath.Join(t.TempDir(), "ledger.db")
	program := `bring GUI
bring SQLite
from GUI bring (Window, Container, TableView)
from SQLite bring (Database, SQLiteValue)

db: Database := SQLite.open(` + quote(database) + `)
db.execute("DROP TABLE IF EXISTS entries")
db.execute("CREATE TABLE entries (id INTEGER PRIMARY KEY, customer TEXT NOT NULL, kind TEXT NOT NULL, amount REAL NOT NULL, note TEXT NOT NULL)")
db.execute("INSERT INTO entries (customer, kind, amount, note) VALUES (?, ?, ?, ?)", [SQLite.fromString("B"), SQLite.fromString("Debt"), SQLite.fromReal(10.0), SQLite.fromString("second")])
db.execute("INSERT INTO entries (customer, kind, amount, note) VALUES (?, ?, ?, ?)", [SQLite.fromString("A"), SQLite.fromString("Payment"), SQLite.fromReal(4.0), SQLite.fromString("first")])
db.execute("INSERT INTO entries (customer, kind, amount, note) VALUES (?, ?, ?, ?)", [SQLite.fromString("A"), SQLite.fromString("Debt"), SQLite.fromReal(2.0), SQLite.fromString("third")])
db.close()

again: Database := SQLite.open(` + quote(database) + `)
again.execute("UPDATE entries SET note = ? WHERE customer = ?", [SQLite.fromString("edited"), SQLite.fromString("B")])
rows: List<List<String>> := []
for record in again.query("SELECT customer, note FROM entries ORDER BY id") {
    rows.add([record["customer"].string(), record["note"].string()])
}
again.close()

window: Window := GUI.window(title: "Ledger", width: 480, height: 320)
form: Container := window.column()
table: TableView := form.table(["Customer", "Note"], rows)
table.selectRow(0)
window.wait()
for line in table.rows() {
    write("{line[0]} {line[1]}")
}
window.close()

check: Database := SQLite.open(` + quote(database) + `)
for record in check.query("SELECT note FROM entries WHERE customer = ?", [SQLite.fromString("B")]) {
    write(record["note"].string())
}
check.close()
`
	assertGUI(t, program, "A first\nA third\nB edited\nedited\n")
}

func TestResourceLoadingAgrees(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "icons"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "icons", "mark.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("mark.png", filepath.Join(root, "icons", "alias.png")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret.png")
	if err := os.WriteFile(outside, []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "icons", "escape.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-target.png", filepath.Join(root, "icons", "gone.png")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_RESOURCE_ROOT", root)
	useHeadlessGUI(t, "")
	program := `bring GUI
from GUI bring (Window, Container, GUIError)

path: String := GUI.resource("icons/mark.png")
write(path)
alias: String := GUI.resource("icons/alias.png")
write(alias)
attempt {
    GUI.resource("icons/escape.png")
} except GUIError as error {
    write(error.message)
}
attempt {
    GUI.resource("icons/gone.png")
} except GUIError as error {
    write(error.message)
}
attempt {
    GUI.resource("missing.png")
} except GUIError as error {
    write(error.message)
}
attempt {
    GUI.resource("../mark.png")
} except GUIError as error {
    write("rejected")
}
window: Window := GUI.window(title: "Resource", width: 200, height: 120)
form: Container := window.column()
attempt {
    form.image("mark.png")
} except GUIError as error {
    write("cwd")
}
window.close()
`
	wantSuffix := string(filepath.Separator) + filepath.Join("icons", "mark.png")
	check := func(output string) {
		t.Helper()
		lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
		if len(lines) != 7 || !strings.HasSuffix(lines[0], wantSuffix) || lines[1] != lines[0] ||
			lines[2] != "the resource path leaves the resource folder" ||
			lines[3] != "could not read the resource icons/gone.png: the file does not exist" ||
			lines[4] != "could not read the resource missing.png: the file does not exist" ||
			lines[5] != "rejected" || lines[6] != "cwd" {
			t.Fatalf("output:\n%s", output)
		}
	}
	directory := writeSources(t, map[string]string{"main.ahd": program})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 {
		t.Fatalf("native exit %d stderr %q\n%s", code, stderr, stdout)
	}
	check(stdout)
	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, program, &output, &errorOutput)
	if errorOutput.Len() != 0 {
		t.Fatalf("evaluator stderr %s", errorOutput.String())
	}
	check(output.String())
}

func quote(path string) string {
	return `"` + path + `"`
}
