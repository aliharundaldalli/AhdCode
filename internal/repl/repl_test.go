package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIncompleteUsesCompilerTokens(t *testing.T) {
	for _, text := range []string{"if true {\n", "add: Function := (\n", `text: String := """hello`} {
		if !Incomplete(text) {
			t.Fatalf("expected incomplete input: %q", text)
		}
	}
	for _, text := range []string{"x: Int := )\n", "write(1)\n", "if true {\n write(1)\n}\n\n"} {
		if Incomplete(text) {
			t.Fatalf("expected complete input: %q", text)
		}
	}
}

func TestPersistentSessionMutationErrorsAndDeclarations(t *testing.T) {
	input := strings.Join([]string{
		"x: Int := 5",
		"write(x ^ 2)",
		"x = 7",
		"write(x)",
		"x: Int := 9",
		"write(1 / 0)",
		"write(x)",
	}, "\n") + "\n"
	var output, errors bytes.Buffer
	if code := Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.5"); code != 0 {
		t.Fatalf("REPL exit = %d", code)
	}
	if !strings.Contains(output.String(), "25\n") || strings.Count(output.String(), "7\n") != 2 {
		t.Fatalf("REPL output:\n%s", output.String())
	}
	if !strings.Contains(errors.String(), "already declared") || !strings.Contains(errors.String(), "DivisionByZeroError") {
		t.Fatalf("REPL errors:\n%s", errors.String())
	}
}

func TestLambdaPersistsAndWorksAsListCallback(t *testing.T) {
	input := `square := lambda (x: Int) -> x ^ 2
write(square(5))
values := [1, 2, 3]
write(values.map(lambda (x: Int) -> x ^ 2))
write(values.filter(lambda (x: Int) -> x > 1))
values.sort(lambda (x: Int) -> -x)
write(values)
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.11")
	for _, want := range []string{"25\n", "[1, 4, 9]\n", "[2, 3]\n", "[3, 2, 1]\n"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("REPL output missing %q:\n%s", want, output.String())
		}
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}

func TestMultilineFunctionAndClassPersist(t *testing.T) {
	input := `add: Function := (
    x: Int
    y: Int
) -> Int {
    return x + y
}
write(add(2, 3))
Box: Class<> := {
    structure: Attributes := (
        value: Int
    )
}
box: Box := Box(value: 9)
write(box.value)
if true {
    write("block")
}
text: String := """first
second"""
write(text)
if false {
    write("wrong")
}
else {
    write("else")
}
attempt {
    write(1 % 0)
}
except DivisionByZeroError as error {
    write("caught")
}
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.5")
	if !strings.Contains(output.String(), "5\n") || !strings.Contains(output.String(), "9\n") || !strings.Contains(output.String(), "block\n") || !strings.Contains(output.String(), "first\nsecond\n") || !strings.Contains(output.String(), "else\n") || !strings.Contains(output.String(), "caught\n") {
		t.Fatalf("REPL output:\n%s\nerrors:\n%s", output.String(), errors.String())
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors:\n%s", errors.String())
	}
}

func TestNumericConversionsAndPowerUseTheSharedPipeline(t *testing.T) {
	input := "write(real(2))\nwrite(real(\"-2.5e-4\"))\nwrite(int(\"  +42  \"))\nwrite(real(2) ^ -3)\nwrite(2 ^ -3)\nwrite(int(3.7))\nwrite(int(\"3.0\"))\n"
	var output, errors bytes.Buffer
	if code := Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.5"); code != 0 {
		t.Fatalf("REPL exit = %d", code)
	}
	if !strings.Contains(output.String(), "2.0\n") || !strings.Contains(output.String(), "-0.00025\n") || !strings.Contains(output.String(), "42\n") || !strings.Contains(output.String(), "0.125\n") || !strings.Contains(output.String(), "3\n") {
		t.Fatalf("REPL output:\n%s", output.String())
	}
	if strings.Count(errors.String(), "DomainError") != 2 {
		t.Fatalf("REPL errors:\n%s", errors.String())
	}
}

func TestTakeUsesTheSharedInteractiveInput(t *testing.T) {
	input := strings.Join([]string{
		`name := take("Name: ")`,
		"Ali",
		`write("[{name}]")`,
		`write(len(name))`,
		"",
	}, "\n")
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.5")
	text := output.String()
	if !strings.Contains(text, "Name: ") {
		t.Fatalf("the take prompt was not written: %q", text)
	}
	if !strings.Contains(text, "[Ali]") {
		t.Fatalf("take did not consume the intended answer: %q", text)
	}
	if !strings.Contains(text, "3\n") {
		t.Fatalf("the captured answer did not persist: %q", text)
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}

func TestPersistentEvaluatorDoesNotReplayEffectsAndPreservesAliasesAndRNG(t *testing.T) {
	input := `write("once")
a := [1, 2]
b := a
a.add(3)
write(b)
bring Math
Math.seed(42)
write(Math.random())
write(Math.random())
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.7")
	text := output.String()
	if strings.Count(text, "once\n") != 1 {
		t.Fatalf("a prior side effect replayed:\n%s", text)
	}
	if !strings.Contains(text, `[1, 2, 3]`) {
		t.Fatalf("List alias identity was lost:\n%s", text)
	}
	if !strings.Contains(text, "0.7415648787718233") || strings.Count(text, "0.7415648787718233") != 1 {
		t.Fatalf("Math RNG state did not progress:\n%s", text)
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}

// TestPersistentIdentitySurvivesAcrossCommandsAndMutation matches the
// v0.1.8 spec example verbatim: id() must return the same number for the
// same List across separate REPL submissions, before and after mutation, and
// for a later alias of the same object.
func TestPersistentIdentitySurvivesAcrossCommandsAndMutation(t *testing.T) {
	input := `x := [1, 2]
firstId := id(x)
write(firstId)
x.add(3)
write(id(x) == firstId)
y := x
write(id(y) == firstId)
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.8")
	text := output.String()
	if strings.Count(text, "true\n") != 2 {
		t.Fatalf("id() did not remain stable across commands/mutation/alias:\n%s", text)
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}

// TestClassProtocolAndIntrospectionInREPL exercises Class Protocol Methods,
// type(), and id() together in the persistent REPL and confirms the results
// match the native backend's behavior for the same program.
func TestClassProtocolAndIntrospectionInREPL(t *testing.T) {
	input := `Vector2: Class<> := {
    structure: Attributes := (x: Real, y: Real)
    CAdd: Function := (other: Vector2) -> Vector2 {
        return Vector2(x: attribute.x + other.x, y: attribute.y + other.y)
    }
    CEqual: Function := (other: Vector2) -> Bool {
        return attribute.x == other.x and attribute.y == other.y
    }
    CStr: Function := () -> String {
        return "Vector2({attribute.x}, {attribute.y})"
    }
}
a := Vector2(x: 1.0, y: 2.0)
b := Vector2(x: 3.0, y: 4.0)
write(a + b)
write(str(a))
write(type(a))
write(a == Vector2(x: 1.0, y: 2.0))
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.8")
	text := output.String()
	for _, want := range []string{"Vector2(4.0, 6.0)", "Vector2(1.0, 2.0)", "Vector2\n", "true\n"} {
		if !strings.Contains(text, want) {
			t.Fatalf("REPL output missing %q:\n%s", want, text)
		}
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}

func TestREPLUsesLaunchDirectoryForModulesAndFiles(t *testing.T) {
	directory := t.TempDir()
	engine := `write("engine init")
tick: Function := () -> Int {
    return 7
}
`
	if err := os.WriteFile(filepath.Join(directory, "Engine.ahd"), []byte(engine), 0o600); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(previous) }()
	input := `bring Engine
write(Engine.tick())
bring File
File.writeText("note.txt", "hello")
write(File.readText("note.txt"))
bring CSV
CSV.write("data.csv", [["name"], ["Ali"]])
write(CSV.read("data.csv")[1][0])
attempt {
    CSV.read("missing.csv")
}
except IOError as error {
    write("csv file caught")
}
bring Time
from Time bring DateTime
epoch: DateTime := Time.fromTimestamp(0)
write(epoch.toOffset(180).timestamp())
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.7")
	if strings.Count(output.String(), "engine init\n") != 1 || !strings.Contains(output.String(), "7\n") || !strings.Contains(output.String(), "hello\n") || !strings.Contains(output.String(), "Ali\n") || !strings.Contains(output.String(), "csv file caught\n") || !strings.Contains(output.String(), "0\n") {
		t.Fatalf("REPL launch-directory behavior:\n%s", output.String())
	}
	if content, err := os.ReadFile(filepath.Join(directory, "note.txt")); err != nil || string(content) != "hello" {
		t.Fatalf("relative File write = %q, %v", content, err)
	}
	if content, err := os.ReadFile(filepath.Join(directory, "data.csv")); err != nil || string(content) != "name\nAli\n" {
		t.Fatalf("relative CSV write = %q, %v", content, err)
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}

// TestDataTablesInThePersistentREPL exercises the Data table layer in the
// persistent evaluator and pins the results the native backend produces for the
// same program, so the two implementations cannot drift apart.
func TestDataTablesInThePersistentREPL(t *testing.T) {
	input := `bring Data
from Data bring Table
from Data bring DataError
table: Table := Data.fromCSV("name,department,score\nAli,Math,91\nAyse,Physics,78\nMehmet,Math,84\n")
write(table.rowCount())
write(table.columns())
write(table.row(-1)["name"])
write(table.filter(lambda (row: Pair<String, String>) -> int(row["score"]) >= 84).column("name"))
write(table.sort(lambda (row: Pair<String, String>) -> -int(row["score"])).column("name"))
write(table.derive("flag", lambda (row: Pair<String, String>) -> str(int(row["score"]) >= 85)).column("flag"))
write(table.transform("name", lambda (value: String) -> value.upper()).column("name"))
write(table.unique("department"))
write(table.valueCounts("department"))
groups: Pair<String, Table> := table.groupBy("department")
write(groups["Math"].column("name"))
snapshot: List<String> := table.columns()
snapshot.add("injected")
write(table.columns())
write(type(table))
write(type(table.row(0)["score"]))
attempt { table.column("missing") } except DataError as error { write(error.message) }
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.12")
	expected := []string{
		"3\n",
		"[\"name\", \"department\", \"score\"]\n",
		"Mehmet\n",
		"[\"Ali\", \"Mehmet\"]\n",
		"[\"Ali\", \"Mehmet\", \"Ayse\"]\n",
		"[\"true\", \"false\", \"false\"]\n",
		"[\"ALI\", \"AYSE\", \"MEHMET\"]\n",
		"[\"Math\", \"Physics\"]\n",
		"{\"Math\": 2, \"Physics\": 1}\n",
		"[\"Ali\", \"Mehmet\"]\n",
		"Table\n",
		"String\n",
		"Table has no column \"missing\"\n",
	}
	text := output.String()
	for _, want := range expected {
		if !strings.Contains(text, want) {
			t.Fatalf("REPL output missing %q:\n%s", want, text)
		}
	}
	// The snapshot mutation must not have reached the Table's own schema.
	if strings.Contains(text, "injected") {
		t.Fatalf("a columns() snapshot mutated the Table:\n%s", text)
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}
