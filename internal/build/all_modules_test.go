package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAllStandardModulesCoexist imports every v0.1.17 standard module in
// one ordinary program and confirms there is no module identity conflict,
// no runtime configurator overwrite, and no helper-discovery regression
// when they are all wired together. Latex is intentionally excluded: it
// requires staging an offline compiler payload that is not available in
// this test environment (see examples/v0.1/16_latex.ahd's note), so
// including it here would make this test's failures about missing local
// staging rather than about module coexistence.
func TestAllStandardModulesCoexist(t *testing.T) {
	directory := t.TempDir()
	source := `bring JSON
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
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("all-module coexistence program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, want := range []string{"91\n", "hello\n", "false\n", "2\n", "2.0\n", "Coexistence\n"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("coexistence output omitted %q:\n%s", want, stdout)
		}
	}
}
