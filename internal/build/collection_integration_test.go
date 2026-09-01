package build

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestJSONObjectUpdateUsesKeyValueRatherThanStringSurgery is the workflow this
// release exists to make possible: read a JSON catalog, replace exactly one
// root field with KeyValue.with, write it back, and confirm the untouched
// fields survive byte-for-byte in structure. Before v0.1.18 the only way to do
// this was stringify -> String concatenation -> parse, which is what the
// program below never does.
func TestJSONObjectUpdateUsesKeyValueRatherThanStringSurgery(t *testing.T) {
	directory := t.TempDir()
	catalog := filepath.Join(directory, "library.json")
	source := `bring JSON
bring KeyValue
bring Lists
bring Data
from JSON bring JSONValue

path: String := ` + strconv.Quote(catalog) + `

category: Function := (identifier: Int, name: String) -> JSONValue {
    return JSON.object(KeyValue.combine(
        ["id", "name"]
        [JSON.fromInt(identifier), JSON.fromString(name)]
    ))
}

book: Function := (title: String, categoryId: Int) -> JSONValue {
    return JSON.object(KeyValue.combine(
        ["title", "categoryId"]
        [JSON.fromString(title), JSON.fromInt(categoryId)]
    ))
}

root: Pair<String, JSONValue> := KeyValue.combine(
    ["categories", "books"]
    [
        JSON.array([category(1, "Fiction"), category(2, "Science"), category(3, "History")])
        JSON.array([book("Dune", 1), book("Cosmos", 3)])
    ]
)
JSON.write(JSON.object(root), path, true)

loaded: JSONValue := JSON.read(path)
object: Pair<String, JSONValue> := loaded.object()
books: List<JSONValue> := object["books"].array()
books.add(book("Sapiens", 1))

updated: Pair<String, JSONValue> := KeyValue.with(object, "books", JSON.array(books))
JSON.write(JSON.object(updated), path, true)

reread: Pair<String, JSONValue> := JSON.read(path).object()
write(KeyValue.keys(reread))
write(len(reread["categories"].array()))
write(len(reread["books"].array()))

names: List<String> := []
for entry in reread["categories"].array() {
    label: Local JSONValue? := entry.get("name")
    if label != null {
        names.add(label.string())
    }
}
write(names)

categoryIds: List<Int> := []
for entry in reread["books"].array() {
    identifier: Local JSONValue? := entry.get("categoryId")
    if identifier != null {
        categoryIds.add(identifier.int())
    }
}
write(categoryIds)
write(Lists.valueCounts(categoryIds))

columns: List<String> := ["name", "score", "department"]
first: Pair<String, String> := KeyValue.combine(columns, ["Ali", "91", "Mathematics"])
second: Pair<String, String> := KeyValue.combine(columns, ["Ayse", "88", "Physics"])
table := Data.fromRecords([first, second])
write(table.columns())
write(table.rowCount())
write(table.row(1))
write(Lists.transpose([KeyValue.keys(first), KeyValue.values(first), KeyValue.values(second)]))
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("JSON/KeyValue workflow failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	want := strings.Join([]string{
		`["categories", "books"]`,
		`3`,
		`3`,
		`["Fiction", "Science", "History"]`,
		`[1, 3, 1]`,
		`{1: 2, 3: 1}`,
		`["name", "score", "department"]`,
		`2`,
		`{"name": "Ayse", "score": "88", "department": "Physics"}`,
		`[["name", "Ali", "Ayse"], ["score", "91", "88"], ["department", "Mathematics", "Physics"]]`,
	}, "\n") + "\n"
	if stdout != want {
		t.Fatalf("expected:\n%s\nreceived:\n%s", want, stdout)
	}
	// The rewritten file must still be ordinary JSON whose untouched
	// "categories" field survived the single-field update intact.
	written, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	text := string(written)
	for _, fragment := range []string{`"categories"`, `"Fiction"`, `"Science"`, `"History"`, `"Sapiens"`} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("the rewritten catalog lost %s:\n%s", fragment, text)
		}
	}
	if strings.Index(text, `"categories"`) > strings.Index(text, `"books"`) {
		t.Fatalf("KeyValue.with moved the replaced key out of its position:\n%s", text)
	}
}
