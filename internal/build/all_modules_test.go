package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAllStandardModulesCoexist imports every standard module of the current
// release in one ordinary program and confirms there is no module identity
// conflict, no runtime configurator overwrite, and no helper-discovery
// regression when they are all wired together. Latex is imported to prove
// identity coexistence but not exercised, so no offline payload is needed.
func TestAllStandardModulesCoexist(t *testing.T) {
	directory := t.TempDir()
	source := `bring Lists
bring KeyValue
bring Excel
bring Latex
bring JSON
bring XML
bring Env
bring Data
bring Numeric
bring Statistics
bring Plot
bring Word
from JSON bring JSONValue
from XML bring XMLNode
from Word bring Document
from Excel bring Workbook
from Excel bring Sheet

write(Lists.chunk([1, 2, 3, 4, 5], 2))
write(Lists.valueCounts(["a", "b", "a"]))
record: Pair<String, String> := KeyValue.combine(["name", "score"], ["Ali", "91"])
write(KeyValue.overlay(record, {"score": "95"}))

value: JSONValue := JSON.fromInt(91)
write(value.int())

node: XMLNode := XML.text("hello")
write(node.text())

write(Env.exists("PATH_TO_NOWHERE_SPECIAL"))

table := Data.fromRows(["a"], [["1"], ["2"]])
write(table.rowCount())

vector := Numeric.vector([1.0, 2.0, 3.0])
write(Statistics.mean(vector.values()))

document: Document := Word.new()
document = document.heading("Coexistence", 1)
write(document.text())

workbook: Workbook := Excel.new().addSheet("Data")
excelSheet: Sheet := workbook.sheet("Data").setCell(1, 1, Excel.fromString("xlsx"))
workbook = workbook.withSheet(excelSheet)
write(workbook.sheetCount())
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("all-module coexistence program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, want := range []string{
		"[[1, 2], [3, 4], [5]]\n", "{\"a\": 2, \"b\": 1}\n",
		"{\"name\": \"Ali\", \"score\": \"95\"}\n",
		"91\n", "hello\n", "false\n", "2\n", "2.0\n", "Coexistence\n", "1\n",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("coexistence output omitted %q:\n%s", want, stdout)
		}
	}
}
