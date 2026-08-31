package repl

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	plotRuntimeOnce sync.Once
	plotRuntimePath string
)

// plotRuntimeForTest builds the ahdplot renderer helper once per test binary
// run and returns its path, skipping the calling test if the toolchain
// cannot build it in this environment -- render/save parity is exercised
// when possible, but a missing helper degrades gracefully rather than
// failing the whole suite.
func plotRuntimeForTest(t *testing.T) string {
	t.Helper()
	plotRuntimeOnce.Do(func() {
		dir, err := os.MkdirTemp("", "ahdplot-test-*")
		if err != nil {
			return
		}
		path := filepath.Join(dir, "ahdplot")
		root, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			return
		}
		command := exec.Command("go", "build", "-o", path, "./cmd/ahdplot")
		command.Dir = root
		if err := command.Run(); err == nil {
			plotRuntimePath = path
		}
	})
	if plotRuntimePath == "" {
		t.Skip("ahdplot helper could not be built in this environment; skipping Plot render/save test")
	}
	return plotRuntimePath
}

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

// TestExplicitLambdaCaptureInThePersistentREPL pins closure behavior in the
// persistent evaluator against the results the native backend produces for the
// same programs, so the two closure implementations cannot drift.
func TestExplicitLambdaCaptureInThePersistentREPL(t *testing.T) {
	input := `keep: Function := (
    minimum: Int
    scores: List<Int>
) -> List<Int> {
    return scores.filter(lambda [#minimum] (score: Int) -> score >= minimum)
}
write(keep(70, [50, 70, 90, 65]))
run: Function := (
) -> Bool {
    minimum: Local Int := 70
    check: Local Function := lambda [#minimum] (score: Int) -> score >= minimum
    return check(80)
}
write(run())
bump: Function := (
    start: Int
) -> Int {
    step: Local Int := start
    first: Local Function := lambda [#step] (x: Int) -> x + step
    step = step + 100
    second: Local Function := lambda [#step] (x: Int) -> x + step
    return first(0) + second(0)
}
write(bump(1))
write([1, 2, 3].map(lambda (x: Int) -> x * 2))
write([1, 2, 3].map(lambda [] (x: Int) -> x * 3))
Maximum: Int := 100
check: Function := lambda [@Maximum] (score: Int) -> score <= Maximum
write(check(50))
Maximum = 40
write(check(50))
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.14")
	text := output.String()
	for _, want := range []string{
		"[70, 90]\n",
		"true\n",
		// Capture by value: 1 + 101, not 101 + 101.
		"102\n",
		"[2, 4, 6]\n",
		"[3, 6, 9]\n",
		// A Global dependency observes the live binding: true before the
		// mutation, false after -- not a snapshot from lambda creation.
		"true\n",
		"false\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("REPL output missing %q:\n%s", want, text)
		}
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}

// TestRawStringsInThePersistentREPL pins raw String literal semantics --
// no escape processing, no interpolation -- in the persistent evaluator
// against the native backend's results for the same input.
func TestRawStringsInThePersistentREPL(t *testing.T) {
	input := `name: String := "Ali"
write(r"{name}")
write(r"\n")
write("{name}")
pattern: String := r"^MATH-[0-9]{3}$"
write(pattern)
write(r'abc\n{x}')
multi: String := r"""
\frac{x+1}{x-1}
"""
write(multi)
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.14")
	text := output.String()
	for _, want := range []string{
		"{name}\n",
		`\n` + "\n",
		"Ali\n",
		"^MATH-[0-9]{3}$\n",
		`abc\n{x}` + "\n",
		"\n\\frac{x+1}{x-1}\n\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("REPL output missing %q:\n%s", want, text)
		}
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}

// TestPlotDomainErrorsAndImmutabilityInThePersistentREPL exercises Plot's
// runtime domain validation and input-immutability guarantees. None of this
// needs the ahdplot renderer helper, since every case here is rejected (or
// snapshotted) before rendering would begin.
func TestPlotDomainErrorsAndImmutabilityInThePersistentREPL(t *testing.T) {
	input := `bring Plot
from Plot bring PlotError

empty: List<Int> := []
emptyLabels: List<String> := []
attempt { Plot.line(empty, empty) } except PlotError as error { write(error.message) }
attempt { Plot.scatter(empty, empty) } except PlotError as error { write(error.message) }
attempt { Plot.bar(emptyLabels, empty) } except PlotError as error { write(error.message) }
attempt { Plot.histogram(empty, 5) } except PlotError as error { write(error.message) }
attempt { Plot.box(empty) } except PlotError as error { write(error.message) }
attempt { Plot.errorBar(empty, empty, empty, empty) } except PlotError as error { write(error.message) }

attempt { Plot.line([1, 2], [1]) } except PlotError as error { write(error.message) }
attempt { Plot.bar(["a"], [1.0, 2.0]) } except PlotError as error { write(error.message) }
attempt { Plot.histogram([1, 2, 3], 0) } except PlotError as error { write(error.message) }
attempt { Plot.errorBar([1], [1], [-1.0], [1.0]) } except PlotError as error { write(error.message) }

bars := Plot.bar(["a", "b"], [1.0, 2.0])
attempt { bars.line([1], [1], "x") } except PlotError as error { write(error.message) }

values: List<Int> := [3, 1, 4, 1, 5]
histogram := Plot.histogram(values, 3)
write(values)
box := Plot.box(values)
write(values)

x: List<Int> := [1, 2, 3]
base := Plot.line(x, x)
extended := base.line(x, x, "again")
write(base != extended)

attempt { Plot.subplots(0, 2, []) } except PlotError as error { write(error.message) }
attempt { Plot.subplots(1, 1, [base, extended]) } except PlotError as error { write(error.message) }
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.14")
	text := output.String()
	for _, want := range []string{
		"line chart data must not be empty\n",
		"scatter chart data must not be empty\n",
		"bar chart data must not be empty\n",
		"histogram data must not be empty\n",
		"box plot data must not be empty\n",
		"errorBar data must not be empty\n",
		"x and y must have the same length\n",
		"bar labels and values must have the same length\n",
		"histogram bin count must be positive\n",
		"lowerErrors must be non-negative\n",
		"cannot add a line series to a bar chart\n",
		"true\n",
		"subplot rows and columns must be positive\n",
		"more charts than subplot cells\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("REPL output missing %q:\n%s", want, text)
		}
	}
	// histogram/box never reorder or otherwise mutate the caller's List.
	if got := strings.Count(text, "[3, 1, 4, 1, 5]"); got != 2 {
		t.Fatalf("expected the caller's List to survive histogram+box unmutated twice, got %d occurrences:\n%s", got, text)
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}

// TestPlotRenderAndSaveParityInThePersistentREPL exercises real rendering
// through the bundled ahdplot helper: PNG/SVG/PDF save, an unsupported
// extension, and a subplot Figure. This is render/save parity, not the
// show() viewer-launch path -- see PART Q/V, which explicitly keep the OS
// image viewer out of automated coverage.
func TestPlotRenderAndSaveParityInThePersistentREPL(t *testing.T) {
	t.Setenv("AHDCODE_PLOT_RUNTIME", plotRuntimeForTest(t))
	directory := t.TempDir()
	png := filepath.Join(directory, "chart.png")
	svg := filepath.Join(directory, "chart.svg")
	pdf := filepath.Join(directory, "chart.pdf")
	bad := filepath.Join(directory, "chart.bmp")
	figurePath := filepath.Join(directory, "figure.png")

	input := `bring Plot
from Plot bring PlotError

x: List<Int> := [1, 2, 3, 4]
y: List<Real> := [2.0, 5.0, 4.0, 8.0]
chart := Plot.line(x, y)
chart = chart.title("Test").xLabel("X").yLabel("Y").legend(true)
chart.save("` + filepath.ToSlash(png) + `")
chart.save("` + filepath.ToSlash(svg) + `")
chart.save("` + filepath.ToSlash(pdf) + `")
write("saved")

attempt {
    chart.save("` + filepath.ToSlash(bad) + `")
}
except PlotError as error {
    write(error.message)
}

bar := Plot.bar(["Math", "Physics"], [90.0, 86.5])
figure := Plot.subplots(1, 2, [chart, bar])
figure.save("` + filepath.ToSlash(figurePath) + `")
write("figure saved")
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.14")
	text := output.String()
	if !strings.Contains(text, "saved\n") || !strings.Contains(text, "figure saved\n") {
		t.Fatalf("REPL output missing expected save confirmations:\n%s", text)
	}
	if !strings.Contains(text, "unsupported output format") {
		t.Fatalf("REPL output missing unsupported-format PlotError:\n%s", text)
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
	for _, path := range []string{png, svg, pdf, figurePath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("expected %s to be non-empty", path)
		}
	}
	if _, err := os.Stat(bad); err == nil {
		t.Fatalf("expected %s to not be created after an unsupported-format error", bad)
	}
}

// TestStatisticsAndPivotCountInThePersistentREPL pins both new surfaces in the
// persistent evaluator against the native backend's results for the same input.
func TestStatisticsAndPivotCountInThePersistentREPL(t *testing.T) {
	input := `bring Statistics
from Statistics bring StatisticsError
bring Data
from Data bring Table
values: List<Int> := [3, 1, 4, 1, 5]
write(Statistics.sum(values))
write(Statistics.mean(values))
write(Statistics.median(values))
write(Statistics.mode(values))
write(Statistics.variance(values))
write(Statistics.sampleVariance(values))
write(Statistics.quantile(values, 0.5))
write(values)
empty: List<Int> := []
attempt { Statistics.mean(empty) } except StatisticsError as error { write(error.message) }
students: Table := Data.fromCSV("name,department,grade\nAli,Math,A\nAyse,Physics,B\nMehmet,Math,A\nZeynep,Physics,A\n")
write(students.pivotCount("department", "grade").toCSV())
write(students.rowCount())
scores: List<Real> := students.column("grade").map(lambda (value: String) -> real(len(value)))
write(Statistics.mean(scores))
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v0.1.14")
	text := output.String()
	for _, want := range []string{
		"14\n", "2.8\n", "3.0\n", "1\n", "2.56\n", "3.2\n",
		// The input List is never reordered by a median or quantile.
		"[3, 1, 4, 1, 5]\n",
		"mean is undefined for an empty List\n",
		"department,A,B\nMath,2,0\nPhysics,1,1\n",
		"4\n", "1.0\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("REPL output missing %q:\n%s", want, text)
		}
	}
	if errors.Len() != 0 {
		t.Fatalf("REPL errors: %s", errors.String())
	}
}
