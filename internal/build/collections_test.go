package build

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildRelocateAndRun compiles one entry module, moves the executable into a
// separate directory, deletes the .ahd source, and runs the binary from there.
// It is how a module proves it needs no repository path, helper binary,
// runtime bundle, or environment variable at run time.
func buildRelocateAndRun(t *testing.T, entry, name string) string {
	t.Helper()
	built := filepath.Join(t.TempDir(), name)
	path, result := BuildProgram(entry, built)
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	relocated := filepath.Join(t.TempDir(), name)
	if err := os.Rename(path, relocated); err != nil {
		t.Fatalf("could not relocate the executable: %v", err)
	}
	if err := os.Remove(entry); err != nil {
		t.Fatalf("could not remove the source: %v", err)
	}
	command := exec.Command(relocated)
	command.Dir = filepath.Dir(relocated)
	// A deliberately bare environment: nothing about the repository, the
	// toolchain, or a helper binary may be needed to run.
	command.Env = []string{"PATH=/usr/bin:/bin"}
	var out, errorOutput strings.Builder
	command.Stdout = &out
	command.Stderr = &errorOutput
	if runError := command.Run(); runError != nil {
		var exit *exec.ExitError
		if !errors.As(runError, &exit) {
			t.Fatalf("could not run the relocated executable: %v", runError)
		}
		t.Fatalf("the relocated executable failed: code=%d stdout=%q stderr=%q",
			exit.ExitCode(), out.String(), errorOutput.String())
	}
	if errorOutput.Len() != 0 {
		t.Fatalf("the relocated executable wrote to stderr: %s", errorOutput.String())
	}
	return out.String()
}

// collectionsProgram exercises the whole Lists/KeyValue surface, including
// ordering, purity, and every error class, through one native executable.
// internal/repl's TestCollectionModulesInThePersistentREPL runs the same
// program text against the same expectations, so the native backend and the
// persistent evaluator are held to one literal contract.
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

// collectionsExpectedOutput is the exact, ordered output collectionsProgram
// produces. The REPL parity test keeps its own copy of both, because a test
// file cannot be imported across packages.
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

func TestCollectionModulesRunAsNativeExecutable(t *testing.T) {
	directory := t.TempDir()
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(collectionsProgram), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("collections program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(lines) != len(collectionsExpectedOutput) {
		t.Fatalf("expected %d output lines; received %d:\n%s",
			len(collectionsExpectedOutput), len(lines), stdout)
	}
	for index, want := range collectionsExpectedOutput {
		if lines[index] != want {
			t.Fatalf("line %d: expected %q; received %q", index+1, want, lines[index])
		}
	}
	if strings.Contains(stdout, "injected") {
		t.Fatalf("a keys() snapshot mutated its source Pair:\n%s", stdout)
	}
}

// TestCollectionModulesSurviveRelocation proves the two modules need no repo
// path, helper binary, runtime bundle, or environment variable: the compiled
// executable is moved away from its source and run from an unrelated
// directory.
func TestCollectionModulesSurviveRelocation(t *testing.T) {
	source := filepath.Join(t.TempDir(), "main.ahd")
	program := `bring Lists
bring KeyValue

write(Lists.transpose([["a", "b"], ["c", "d"]]))
write(Lists.valueCounts([1, 1, 2]))
write(KeyValue.overlay(KeyValue.combine(["a"], [1]), {"b": 2}))
`
	if err := os.WriteFile(source, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout := buildRelocateAndRun(t, source, "collections")
	for _, want := range []string{`[["a", "c"], ["b", "d"]]`, `{1: 2, 2: 1}`, `{"a": 1, "b": 2}`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("relocated executable output missing %q:\n%s", want, stdout)
		}
	}
}

// TestCollectionModulesAreNotShadowedByASiblingFile compiles a program whose
// own directory holds conflicting Lists.ahd and KeyValue.ahd files, and proves
// the built-in modules still win, all the way through to a running binary.
func TestCollectionModulesAreNotShadowedByASiblingFile(t *testing.T) {
	directory := t.TempDir()
	files := map[string]string{
		"Lists.ahd": "chunk: Function := (values: List<Int>, size: Int) -> String {\n" +
			"    return \"user Lists.chunk\"\n}\n",
		"KeyValue.ahd": "keys: Function := (pair: Pair<String, Int>) -> String {\n" +
			"    return \"user KeyValue.keys\"\n}\n",
		"main.ahd": "bring Lists\nbring KeyValue\nwrite(Lists.chunk([1, 2, 3], 2))\nwrite(KeyValue.keys({\"a\": 1}))\n",
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("shadowing program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if stdout != "[[1, 2], [3]]\n[\"a\"]\n" {
		t.Fatalf("a sibling file shadowed a built-in module: %q", stdout)
	}
}

// TestCollectionModulesShallowStructuralSemantics proves the transformations
// copy collection structure but never deep-copy referenced values: a Class
// instance reached through a new List or Pair is the same object.
func TestCollectionModulesShallowStructuralSemantics(t *testing.T) {
	directory := t.TempDir()
	source := `bring Lists
bring KeyValue

Box: Class<> := {
    structure: Attributes := (
        label: String
    )
}

first: Box := Box(label: "one")
boxes: List<Box> := [first, Box(label: "two")]
parts: List<List<Box>> := Lists.chunk(boxes, 1)
parts[0][0].label = "changed"
write(first.label)
write(boxes[0].label)

byName: Pair<String, Box> := KeyValue.combine(["first"], [first])
copied: Pair<String, Box> := KeyValue.with(byName, "second", Box(label: "two"))
write(copied["first"] same first)
write(len(byName))

frozen: Constant List<Int> := [1, 1, 2]
write(Lists.unique(frozen))
write(Lists.valueCounts(frozen))
locked: Constant Pair<String, Int> := {"a": 1}
write(KeyValue.with(locked, "b", 2))
write(locked)
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("shallow-semantics program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	want := "changed\nchanged\ntrue\n1\n[1, 2]\n{1: 2, 2: 1}\n{\"a\": 1, \"b\": 2}\n{\"a\": 1}\n"
	if stdout != want {
		t.Fatalf("expected:\n%s\nreceived:\n%s", want, stdout)
	}
}
