package repl

import (
	"bytes"
	"strings"
	"testing"
)

// collectionsProgram and collectionsExpectedOutput are copies of the native
// end-to-end program in internal/build/collections_test.go. Keeping the two
// literally identical is the parity contract: the persistent evaluator must
// produce exactly what the compiled executable produces, including generic
// result shapes, ordering, and error text.
const collectionsProgram = `bring Lists
bring KeyValue
from Lists bring ListsError
from KeyValue bring KeyValueError

numbers: List<Int> := [1, 2, 3, 4, 5]
chunks: List<List<Int>> := Lists.chunk(numbers, 2)
write(chunks)
write(numbers)
write(Lists.chunk(numbers, 1))
write(Lists.chunk(numbers, 99))

grid: List<List<Int>> := [[1, 2], [3], [4, 5]]
write(Lists.flatten(grid))
write(Lists.transpose([[1, 2, 3], [4, 5, 6]]))
write(Lists.transpose([[1, 2, 3]]))
empty: List<List<Int>> := []
write(Lists.transpose(empty))

write(Lists.unique([3, 1, 3, 2, 1]))
write(Lists.unique(["b", "a", "b"]))
write(Lists.unique([true, false, true]))
write(Lists.valueCounts([1, 1, 3, 2, 1, 3]))
write(Lists.valueCounts(["Math", "Physics", "Math"]))
write(Lists.groupBy(["Ali", "Ayse", "Bora", "Ahmet"], lambda (name: String) -> name[0]))

record: Pair<String, String> := KeyValue.combine(["name", "score", "department"], ["Ali", "91", "Mathematics"])
write(record)
write(KeyValue.keys(record))
write(KeyValue.values(record))
write(KeyValue.with(record, "score", "95"))
write(KeyValue.with(record, "year", "2"))
write(record)
write(KeyValue.without(record, "name"))
write(KeyValue.select(record, ["department", "name"]))
write(KeyValue.drop(record, ["score"]))
write(KeyValue.rename(record, "name", "student"))
write(KeyValue.rename(record, "name", "name"))
write(KeyValue.mapValues(record, lambda (value: String) -> len(value)))
write(KeyValue.merge({"a": 1, "b": 2}, {"c": 3}))
write(KeyValue.overlay({"a": 1, "b": 2}, {"b": 9, "c": 3}))

snapshot: List<String> := KeyValue.keys(record)
snapshot.add("injected")
write(record)

attempt { write(Lists.chunk(numbers, 0)) } except ListsError as error { write(error.message) }
attempt { write(Lists.transpose([[1, 2, 3], [4, 5]])) } except ListsError as error { write(error.message) }
attempt { write(KeyValue.combine(["a", "b"], ["1"])) } except KeyValueError as error { write(error.message) }
attempt { write(KeyValue.combine(["a", "a"], ["1", "2"])) } except KeyValueError as error { write(error.message) }
attempt { write(KeyValue.select(record, ["name", "name"])) } except KeyValueError as error { write(error.message) }
attempt { write(KeyValue.rename(record, "name", "score")) } except KeyValueError as error { write(error.message) }
attempt { write(KeyValue.merge(record, record)) } except KeyValueError as error { write(error.message) }
attempt { write(KeyValue.without(record, "missing")) } except KeyError as error { write(error.message) }
attempt { write(KeyValue.drop(record, ["missing"])) } except KeyError as error { write(error.message) }
`

var collectionsExpectedOutput = []string{
	`[[1, 2], [3, 4], [5]]`,
	`[1, 2, 3, 4, 5]`,
	`[[1], [2], [3], [4], [5]]`,
	`[[1, 2, 3, 4, 5]]`,
	`[1, 2, 3, 4, 5]`,
	`[[1, 4], [2, 5], [3, 6]]`,
	`[[1], [2], [3]]`,
	`[]`,
	`[3, 1, 2]`,
	`["b", "a"]`,
	`[true, false]`,
	`{1: 3, 3: 2, 2: 1}`,
	`{"Math": 2, "Physics": 1}`,
	`{"A": ["Ali", "Ayse", "Ahmet"], "B": ["Bora"]}`,
	`{"name": "Ali", "score": "91", "department": "Mathematics"}`,
	`["name", "score", "department"]`,
	`["Ali", "91", "Mathematics"]`,
	`{"name": "Ali", "score": "95", "department": "Mathematics"}`,
	`{"name": "Ali", "score": "91", "department": "Mathematics", "year": "2"}`,
	`{"name": "Ali", "score": "91", "department": "Mathematics"}`,
	`{"score": "91", "department": "Mathematics"}`,
	`{"department": "Mathematics", "name": "Ali"}`,
	`{"name": "Ali", "department": "Mathematics"}`,
	`{"student": "Ali", "score": "91", "department": "Mathematics"}`,
	`{"name": "Ali", "score": "91", "department": "Mathematics"}`,
	`{"name": 3, "score": 2, "department": 11}`,
	`{"a": 1, "b": 2, "c": 3}`,
	`{"a": 1, "b": 9, "c": 3}`,
	`{"name": "Ali", "score": "91", "department": "Mathematics"}`,
	`chunk requires a size greater than zero; received 0`,
	`transpose requires rectangular rows: row 1 has 2 element(s); expected 3`,
	`combine requires equal lengths; received 2 key(s) and 1 value(s)`,
	`combine received the duplicate key "a"`,
	`select received the duplicate key "name"`,
	`rename cannot rename "name" to "score"; that key already exists`,
	`merge received the key "name" in both Pairs`,
	`Pair has no key "missing"`,
	`Pair has no key "missing"`,
}

func TestCollectionModulesInThePersistentREPL(t *testing.T) {
	var output, errorOutput bytes.Buffer
	Run(strings.NewReader(collectionsProgram), &output, &errorOutput, "AhdCode v0.2.0")
	if errorOutput.Len() != 0 {
		t.Fatalf("REPL errors: %s", errorOutput.String())
	}
	lines := replValueLines(output.String())
	if len(lines) != len(collectionsExpectedOutput) {
		t.Fatalf("expected %d value lines; received %d:\n%s",
			len(collectionsExpectedOutput), len(lines), output.String())
	}
	for index, want := range collectionsExpectedOutput {
		if lines[index] != want {
			t.Fatalf("line %d: expected %q; received %q", index+1, want, lines[index])
		}
	}
	if strings.Contains(output.String(), "injected") {
		t.Fatalf("a keys() snapshot mutated its source Pair:\n%s", output.String())
	}
}

// replValueLines strips the session banner and the interactive prompts the
// session interleaves with program output, leaving only the lines the program
// itself wrote.
func replValueLines(text string) []string {
	replaced := strings.ReplaceAll(text, "...> ", "")
	replaced = strings.ReplaceAll(replaced, "ahd> ", "")
	lines := []string{}
	for index, line := range strings.Split(replaced, "\n") {
		if line == "" || (index == 0 && strings.HasPrefix(line, "AhdCode v")) {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}
