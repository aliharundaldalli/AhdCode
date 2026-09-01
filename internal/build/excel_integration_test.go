package build

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestExcelComposesWithListsKeyValueDataAndJSON(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "integration.xlsx")
	source := `bring Excel
bring Lists
bring KeyValue
bring Data
bring JSON
from Excel bring Workbook
from Excel bring Sheet
from Excel bring Cell
from Data bring Table
from JSON bring JSONValue

columns: List<String> := ["Name", "Score", "Department"]
first: Pair<String, String> := KeyValue.combine(columns, ["Ali", "91", "Mathematics"])
second: Pair<String, String> := KeyValue.combine(columns, ["Ayse", "88", "Physics"])
table: Table := Data.fromRecords([first, second])

book: Workbook := Excel.new().addSheet("Data").addSheet("Transpose").addSheet("JSON")
sheet: Sheet := book.sheet("Data")
headerCells: List<Cell> := table.columns().map(lambda (text: String) -> Excel.fromString(text))
sheet = sheet.setRow(1, 1, headerCells)
rowIndex: Int := 2
for record in table.rows() {
    cells: Local List<Cell> := KeyValue.values(record).map(lambda (text: String) -> Excel.fromString(text))
    sheet = sheet.setRow(rowIndex, 1, cells)
    rowIndex = rowIndex + 1
}
book = book.withSheet(sheet)

transposeSheet: Sheet := book.sheet("Transpose")
sourceGrid: List<List<Cell>> := [
    [Excel.fromString("A"), Excel.fromString("B"), Excel.fromString("C")]
    [Excel.fromInt(1), Excel.fromInt(2), Excel.fromInt(3)]
]
transposed: List<List<Cell>> := Lists.transpose(sourceGrid)
restored: List<List<Cell>> := Lists.transpose(transposed)
transposeSheet = transposeSheet.setRange(transposeSheet.range(1, 1, 2, 3), restored)
book = book.withSheet(transposeSheet)

jsonRecord: JSONValue := JSON.object(KeyValue.combine(
    ["Name", "Score", "Active"]
    [JSON.fromString("Bora"), JSON.fromInt(84), JSON.fromBool(true)]
))
jsonObject: Pair<String, JSONValue> := jsonRecord.object()
jsonSheet: Sheet := book.sheet("JSON")
jsonSheet = jsonSheet.setRow(1, 1, KeyValue.keys(jsonObject).map(lambda (text: String) -> Excel.fromString(text)))
jsonCells: List<Cell> := []
for value in KeyValue.values(jsonObject) {
    if value.kind() == "String" {
        jsonCells.add(Excel.fromString(value.string()))
    }
    else if value.kind() == "Int" {
        jsonCells.add(Excel.fromInt(value.int()))
    }
    else if value.kind() == "Bool" {
        jsonCells.add(Excel.fromBool(value.bool()))
    }
}
jsonSheet = jsonSheet.setRow(2, 1, jsonCells)
book = book.withSheet(jsonSheet)
book.save(` + strconv.Quote(output) + `)

loaded: Workbook := Excel.read(` + strconv.Quote(output) + `)
write(loaded.sheet("Data").cell(3, 1).string())
write(loaded.sheet("Transpose").cell(2, 3).int())
write(loaded.sheet("JSON").cell(2, 2).int())
write(loaded.sheet("JSON").cell(2, 3).bool())
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("Excel integration failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if stdout != strings.Join([]string{"Ayse", "3", "84", "true", ""}, "\n") {
		t.Fatalf("integration output = %q", stdout)
	}
}
