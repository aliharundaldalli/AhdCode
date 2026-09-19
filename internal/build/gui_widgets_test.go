package build

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The v2.0 GUI surface, end to end through the real compiler and the real
// ahdgui helper, headless: every new widget, the change and selection
// callbacks, programmatic changes that call no callback, resizable Windows,
// and the dialogs, with identical output from a native program and the
// evaluator. Widget ids follow creation order: root 1, name 2, pin 3,
// notes 4, method 5, customers 6, ledger 7, paid 8, save 9.
const guiWidgetsProgram = `bring GUI
from GUI bring (Window, Container, TextInput, PasswordInput, TextArea, Select, ListBox, TableView, Checkbox, Button, GUIError)

window: Window := GUI.window(title: "Ledger", width: 640, height: 480)
window.setResizable(true)
form: Container := window.column()
name: TextInput := form.textInput(placeholder: "Customer")
pin: PasswordInput := form.passwordInput()
notes: TextArea := form.textArea(placeholder: "Notes")
method: Select := form.select(items: ["Cash", "Card", "Transfer"], selectedIndex: 0)
customers: ListBox := form.listBox(["Ayşe", "Ali"])
ledger: TableView := form.table(["Customer", "Amount"], [["Ayşe", "10"], ["Ali", "25"]])
paid: Checkbox := form.checkbox("Paid")
save: Button := form.button("Save")

name.onChange(lambda (text: String) -> write("name {text}"))
pin.onChange(lambda (text: String) -> write("pin length {len(text)}"))
notes.onChange(lambda (text: String) -> write("notes lines {len(text.split("\n"))}"))
paid.onChange(lambda (checked: Bool) -> write("paid {checked}"))
method.onChange(lambda (index: Int?, text: String?) -> write("method {index} {text}"))
customers.onChange(lambda (index: Int?, text: String?) -> write("customer {index} {text}"))
ledger.onSelect(lambda (row: Int?) -> write("row {row}"))

addRow: Function := () -> Nothing {
    ledger: Global TableView
    name: Global TextInput
    rows: Local List<List<String>> := ledger.rows()
    rows.add([name.text(), "5"])
    ledger.setRows(rows)
    ledger.selectRow(len(rows) - 1)
    write("rows {len(ledger.rows())} selected {ledger.selectedRow()}")
}
save.onClick(addRow)

write("resizable {window.isResizable()} method {method.selectedIndex()} {method.selectedText()} customer {customers.selectedIndex()}")
customers.select(1)
customers.setItems(["Ayşe"])
write("after setItems {customers.selectedIndex()} {customers.items()}")
method.select(null)
write("columns {ledger.columns()} rows {len(ledger.rows())}")
attempt {
    form.table([], [])
} except GUIError as error {
    write(error.message)
}
attempt {
    ledger.setRows([["only one"]])
} except GUIError as error {
    write(error.message)
}
attempt {
    customers.select(5)
} except GUIError as error {
    write(error.message)
}
window.wait()
write("after name={name.text()} pin={pin.text()} notes={notes.text()} method={method.selectedText()} customer={customers.selectedText()} row={ledger.selectedRow()} paid={paid.checked()}")

chosen: String? := GUI.openFile(extensions: ["csv"])
write("open {chosen}")
cancelled: String? := GUI.saveFile(suggestedName: "ledger.csv", extensions: ["csv"])
write("save {cancelled == null}")
files: List<String> := GUI.openFiles()
write("files {len(files)}")
folder: String? := GUI.selectFolder(title: "Export to")
write("folder {folder}")
write("confirm {GUI.confirm("Delete", "Delete the row?")}")
GUI.message("Done", "Saved")
attempt {
    GUI.openFile(extensions: ["*.csv"])
} except GUIError as error {
    write(error.message)
}
attempt {
    GUI.openFile()
} except GUIError as error {
    write(error.message)
}
`

const guiWidgetsScript = `[{"event":"type","widget":2,"text":"Zeynep"},{"event":"type","widget":3,"text":"1234"},` +
	`{"event":"type","widget":4,"text":"first\nsecond"},{"event":"select","widget":5,"selected":2},` +
	`{"event":"select","widget":6,"selected":0},{"event":"select","widget":7,"selected":1},{"event":"toggle","widget":8},` +
	`{"event":"resize","width":900,"height":700},{"event":"click","widget":9}]`

const guiWidgetsDialogs = `[{"paths":["/tmp/ahdcode-ledger/in.csv"]},{"cancel":true},{"cancel":true},{"paths":["/tmp/ahdcode-ledger"]},` +
	`{"confirm":true},{},{"fail":"no display"}]`

const guiWidgetsExpected = `resizable true method 0 Cash customer null
after setItems null ["Ayşe"]
columns ["Customer", "Amount"] rows 2
a TableView needs at least one column
row 0 has 1 cells; every row needs one cell per column (2)
index 5 is outside the 1 entries
name Zeynep
pin length 4
notes lines 2
method 2 Transfer
customer 0 Ayşe
row 1
paid true
rows 3 selected 2
after name=Zeynep pin=1234 notes=first
second method=Transfer customer=Ayşe row=2 paid=true
open /tmp/ahdcode-ledger/in.csv
save true
files 0
folder /tmp/ahdcode-ledger
confirm true
"*.csv" is not a file extension; use letters and digits, such as "csv"
the dialog could not be shown: no display
`

func TestGUIWidgetsDialogsNativeAndEvaluatorAgree(t *testing.T) {
	useHeadlessGUI(t, guiWidgetsScript)
	directory := writeSources(t, map[string]string{"main.ahd": guiWidgetsProgram})
	for _, mode := range []string{"native", "evaluator"} {
		t.Run(mode, func(t *testing.T) {
			// Each program answers its dialogs from the start of the list.
			t.Setenv("AHDCODE_GUI_HEADLESS_DIALOGS", guiWidgetsDialogs)
			var out string
			if mode == "native" {
				stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
				if code != 0 {
					t.Fatalf("exit %d, stderr %q, stdout %q", code, stderr, stdout)
				}
				out = stdout
			} else {
				var output, errorOutput bytes.Buffer
				runTerminalEvaluator(t, guiWidgetsProgram, &output, &errorOutput)
				out = output.String()
			}
			if out != guiWidgetsExpected {
				t.Fatalf("%s output:\n%s\nwant:\n%s", mode, out, guiWidgetsExpected)
			}
		})
	}
}

func TestGUISelectionCallbacksMustAcceptNull(t *testing.T) {
	for _, bad := range []string{
		`list.onChange(lambda (index: Int, text: String) -> write(index))`,
		`list.onChange(lambda (index: Int?, text: String) -> write(index))`,
		`table.onSelect(lambda (row: Int) -> write(row))`,
	} {
		source := "bring GUI\nwindow := GUI.window()\nform := window.column()\nlist := form.listBox()\ntable := form.table([\"a\"])\n" + bad + "\n"
		directory := writeSources(t, map[string]string{"main.ahd": source})
		result := Compile(filepath.Join(directory, "main.ahd"))
		if !result.HasErrors() || !strings.Contains(diagnosticText(result.Diagnostics), "declare the handler's parameters nullable") {
			t.Fatalf("%s accepted:\n%s", bad, diagnosticText(result.Diagnostics))
		}
	}
}

// TestLedgerExampleRunsHeadless drives examples/v2.0/ledger_app as a user
// would: two entries, a row selection, a declined delete, and a CSV export
// through the save dialog, in a native program and the evaluator.
func TestLedgerExampleRunsHeadless(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "v2.0", "ledger_app", "main.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	sqliteHelperForTest(t)
	// Widget ids follow creation order: customer 7, kind 9, amount 11,
	// add 14, table 21, remove 25, export 26.
	useHeadlessGUI(t, `[{"event":"type","widget":7,"text":"Ayşe"},{"event":"type","widget":11,"text":"250.5"},{"event":"click","widget":14},`+
		`{"event":"select","widget":9,"selected":1},{"event":"type","widget":11,"text":"100"},{"event":"click","widget":14},`+
		`{"event":"select","widget":21,"selected":0},{"event":"click","widget":25},{"event":"click","widget":26}]`)
	want := "No,Customer,Entry,Amount,Note\n1,Ayşe,Debt,250.5,\n2,Ayşe,Payment,100.0,\n"
	for _, mode := range []string{"native", "evaluator"} {
		t.Run(mode, func(t *testing.T) {
			directory := t.TempDir()
			exported := filepath.Join(directory, "export", "ledger.csv")
			if err := os.MkdirAll(filepath.Dir(exported), 0o755); err != nil {
				t.Fatal(err)
			}
			answers, _ := json.Marshal([]any{map[string]any{}, map[string]any{"paths": []string{exported}}, map[string]any{}})
			t.Setenv("AHDCODE_GUI_HEADLESS_DIALOGS", string(answers))
			sources := writeSources(t, map[string]string{"main.ahd": string(source)})
			if mode == "native" {
				stdout, stderr, code := buildAndRunIn(t, filepath.Join(sources, "main.ahd"), directory)
				if code != 0 {
					t.Fatalf("exit %d: %s %s", code, stdout, stderr)
				}
			} else {
				var output, errorOutput bytes.Buffer
				runEvaluatorIn(t, directory, string(source), &output, &errorOutput)
			}
			data, err := os.ReadFile(exported)
			if err != nil || string(data) != want {
				t.Fatalf("exported %q (%v), want %q", data, err, want)
			}
			if _, err := os.Stat(filepath.Join(directory, "ahdcode-ledger.db")); err != nil {
				t.Fatal("the ledger database was not created in the working folder")
			}
		})
	}
}
