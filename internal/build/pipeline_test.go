package build

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	backend "ahdcode/internal/backend/golang"
	"ahdcode/internal/diagnostics"
)

// program is one end-to-end acceptance case: AhdCode sources in, real program
// output out.
type program struct {
	name     string
	entry    string
	sources  map[string]string
	stdin    string
	expected string
	exitCode int
	// errorClass is the AhdCode Error class an uncaught failure must report.
	errorClass string
}

func writeSources(t *testing.T, sources map[string]string) string {
	t.Helper()
	directory := t.TempDir()
	for name, text := range sources {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(text), 0o600); err != nil {
			t.Fatalf("could not write %s: %v", name, err)
		}
	}
	return directory
}

func diagnosticText(items []diagnostics.Diagnostic) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, item.Code+": "+item.Message)
	}
	return strings.Join(parts, "\n")
}

// buildAndRun compiles one entry module to a native executable and runs it.
func buildAndRun(t *testing.T, entry, stdin string) (string, string, int) {
	t.Helper()
	output := filepath.Join(t.TempDir(), "program")
	path, result := BuildProgram(entry, output)
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if path != output {
		t.Fatalf("expected the executable at %s; received %s", output, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("no executable was produced: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatal("the produced file is not executable")
	}
	command := exec.Command(path)
	command.Stdin = strings.NewReader(stdin)
	var out, errorOutput strings.Builder
	command.Stdout = &out
	command.Stderr = &errorOutput
	code := 0
	if runError := command.Run(); runError != nil {
		var exit *exec.ExitError
		if !errors.As(runError, &exit) {
			t.Fatalf("could not run the executable: %v", runError)
		}
		code = exit.ExitCode()
	}
	return out.String(), errorOutput.String(), code
}

// runProgramCases builds and runs each acceptance case as a native
// executable and checks its real streams and exit code.
func runProgramCases(t *testing.T, cases []program) {
	t.Helper()
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			entry := testCase.entry
			if entry == "" {
				entry = "main.ahd"
			}
			directory := writeSources(t, testCase.sources)
			out, errorOutput, code := buildAndRun(t, filepath.Join(directory, entry), testCase.stdin)
			if out != testCase.expected {
				t.Fatalf("stdout mismatch\n want %q\n have %q\n stderr: %s", testCase.expected, out, errorOutput)
			}
			if code != testCase.exitCode {
				t.Fatalf("exit code mismatch: want %d, have %d (stderr: %s)", testCase.exitCode, code, errorOutput)
			}
			if testCase.errorClass != "" && !strings.HasPrefix(errorOutput, testCase.errorClass+": ") {
				t.Fatalf("expected an uncaught %s on stderr; received %q", testCase.errorClass, errorOutput)
			}
			if testCase.exitCode != 0 && strings.Contains(errorOutput, "goroutine ") {
				t.Fatalf("a Go stack trace leaked into program output: %q", errorOutput)
			}
		})
	}
}

func TestAcceptanceProgramsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name:     "hello",
			sources:  map[string]string{"main.ahd": "write(\"Hello AhdCode\")\n"},
			expected: "Hello AhdCode\n",
		},
		{
			name:     "arithmetic",
			sources:  map[string]string{"main.ahd": "x: Int := 5\ny: Int := 7\nwrite(x + y)\n"},
			expected: "12\n",
		},
		{
			name:     "int to real widening",
			sources:  map[string]string{"main.ahd": "x: Real := 5\nwrite(x)\n"},
			expected: "5.0\n",
		},
		{
			name: "function",
			sources: map[string]string{"main.ahd": `add: Function := (
    a: Int
    b: Int
) -> Int {
    return a + b
}

write(add(5, 7))
`},
			expected: "12\n",
		},
		{
			name: "recursion",
			sources: map[string]string{"main.ahd": `factorial: Function := (
    n: Int
) -> Int {
    if n <= 1 {
        return 1
    }

    return n * factorial(n - 1)
}

write(factorial(10))
`},
			expected: "3628800\n",
		},
		{
			name: "condition and loop",
			sources: map[string]string{"main.ahd": `total: Int := 0
i: Int := 1
while i <= 5 {
    if i % 2 == 0 {
        total += i
    }

    i++
}

write(total)
`},
			expected: "6\n",
		},
		{
			name: "until post-check and continue",
			sources: map[string]string{"main.ahd": `i: Int := 0
body: Int := 0
until i >= 3 {
    i++
    body++
    if i == 1 {
        continue
    }
}

write(i)
write(body)
`},
			expected: "3\n3\n",
		},
		{
			name: "until body always runs once",
			sources: map[string]string{"main.ahd": `runs: Int := 0
until true {
    runs++
}

write(runs)
`},
			expected: "1\n",
		},
		{
			name:     "list alias and clear",
			sources:  map[string]string{"main.ahd": "a: List<Int> := [1, 2, 3]\nb: List<Int> := a\nclear(a)\nwrite(len(b))\n"},
			expected: "0\n",
		},
		{
			name: "pair alias and clear",
			sources: map[string]string{"main.ahd": `a: Pair<String, Int> := {
    "x": 1
    "y": 2
}

b: Pair<String, Int> := a
write(len(b))
clear(a)
write(len(b))
`},
			expected: "2\n0\n",
		},
		{
			name: "pair keeps insertion order",
			sources: map[string]string{"main.ahd": `scores: Pair<String, Int> := {
    "zebra": 1
    "alpha": 2
}

scores["zebra"] = 9
for key in scores {
    write(key)
}

write(str(scores))
`},
			expected: "zebra\nalpha\n{\"zebra\": 9, \"alpha\": 2}\n",
		},
		{
			name: "callback",
			sources: map[string]string{"main.ahd": `apply: Function := (
    value: Int
    action: Function
) -> Int {
    return action(value)
}

double: Function := (
    n: Int
) -> Int {
    return n * 2
}

write(apply(21, double))
`},
			expected: "42\n",
		},
		{
			name: "overload",
			sources: map[string]string{"main.ahd": `calculate: Function := (
    x: Int
) -> Int {
    return x ^ 2
}

calculate: Overload Function := (
    x: Real
) -> Real {
    return x * 2.0
}

write(calculate(5))
write(calculate(2.5))
`},
			expected: "25\n5.0\n",
		},
		{
			name: "class construction, field, and method",
			sources: map[string]string{"main.ahd": `Counter: Class<> := {
    structure: Attributes := (
        label: String
        value: Int
    )

    bump: Function := (
        amount: Int
    ) -> Nothing {
        attribute.value += amount
    }
}

counter: Counter := Counter(
    label: "hits"
    value: 1
)

counter.bump(4)
write("{counter.label}={counter.value}")
`},
			expected: "hits=5\n",
		},
		{
			name: "vector dot product",
			sources: map[string]string{"main.ahd": `Vector: Class<> := {
    structure: Attributes := (
        x: Real
        y: Real
    )

    dot: Function := (
        other: Vector
    ) -> Real {
        return attribute.x * other.x + attribute.y * other.y
    }
}

a: Vector := Vector(
    x: 2.0
    y: 3.0
)

b: Vector := Vector(
    x: 4.0
    y: 5.0
)

write(a.dot(b))
`},
			expected: "23.0\n",
		},
		{
			name: "class reference identity",
			sources: map[string]string{"main.ahd": `Box: Class<> := {
    structure: Attributes := (
        value: Int
    )
}

first: Box := Box(value: 1)
second: Box := first
second.value = 9
write(first.value)
write(str(first == second))
`},
			expected: "9\ntrue\n",
		},
		{
			name: "multi module bring",
			sources: map[string]string{
				"main.ahd": "from Greeting bring greet\n\nwrite(greet(\"AhdCode\"))\n",
				"Greeting.ahd": `greet: Function := (
    name: String
) -> String {
    return "Hello {name}"
}
`,
			},
			expected: "Hello AhdCode\n",
		},
		{
			name: "namespace module member",
			sources: map[string]string{
				"main.ahd": "bring Mathematics\n\nwrite(Mathematics.twice(21))\n",
				"Mathematics.ahd": `twice: Function := (
    value: Int
) -> Int {
    return value * 2
}
`,
			},
			expected: "42\n",
		},
		{
			name: "state has no fall-through",
			sources: map[string]string{"main.ahd": `value: Int := 2
state value {
    condition 1 {
        write("one")
    }

    condition 2 {
        write("two")
    }

    condition default {
        write("other")
    }
}
`},
			expected: "two\n",
		},
		{
			name: "for iterates a shallow snapshot",
			sources: map[string]string{"main.ahd": `numbers: List<Int> := [1, 2, 3]
seen: Int := 0
for number in numbers {
    seen++
    clear(numbers)
}

write(seen)
write(len(numbers))
`},
			expected: "3\n0\n",
		},
		{
			name:     "string iteration is by character",
			sources:  map[string]string{"main.ahd": "for character in \"añb\" {\n    write(character)\n}\n"},
			expected: "a\nñ\nb\n",
		},
		{
			name: "canonical str",
			sources: map[string]string{"main.ahd": `write(str(5))
write(str(5.0))
write(str(-0.0))
write(str(true))
write(str(null))
write(str([1, 2, 3]))
write(str(["Ali", "Ayşe"]))
write(str({"Ali": 90, "Ayşe": 95}))
`},
			expected: "5\n5.0\n-0.0\ntrue\nnull\n[1, 2, 3]\n[\"Ali\", \"Ayşe\"]\n{\"Ali\": 90, \"Ayşe\": 95}\n",
		},
		{
			name: "operators, indexing, slicing, and local bindings",
			sources: map[string]string{"main.ahd": `Box: Class<> := {
    structure: Attributes := (
        value: Int
    )
}

inspect: Function := (
) -> String {
    numbers: Local List<Int> := [10, 20, 30]
    text: Local String := "hello"
    scores: Local Pair<String, Int> := {
        "a": 1
    }

    numbers[0] = 15
    numbers[-1] = 99
    scores["a"] = 5
    box: Local Box := Box(value: 1)

    parts: Local List<String> := [
        str(numbers)
        str(numbers[1:])
        str(text[1:3])
        str(text[-1])
        str(20 in numbers)
        str("ell" in text)
        str("a" in scores)
        str(5 same 5)
        str(5 same 5.0)
        str(box is Box)
        str(box has value)
        str(box has missing)
        str(scores)
        str([1] + [2])
        str("ab" * 2)
        str(not false)
        str(true and false)
        str(true or false)
    ]

    return str(parts)
}

write(inspect())
`},
			expected: "[\"[15, 20, 99]\", \"[20, 99]\", \"el\", \"o\", \"true\", \"true\", \"true\", \"true\", \"false\", \"true\", \"true\", \"false\", \"{\\\"a\\\": 5}\", \"[1, 2]\", \"abab\", \"true\", \"false\", \"true\"]\n",
		},
		{
			name: "compound assignment and update evaluate their target once",
			sources: map[string]string{"main.ahd": `Counter: Class<> := {
    structure: Attributes := (
        value: Int
    )
}

calls: Int := 0

pick: Function := (
    counter: Counter
) -> Counter {
    calls: Global Int
    calls = calls + 1
    return counter
}

counter: Counter := Counter(value: 1)
pick(counter).value += 10
pick(counter).value++
write(counter.value)
write(calls)
`},
			expected: "12\n2\n",
		},
		{
			name:     "signed 64-bit minimum",
			sources:  map[string]string{"main.ahd": "minimum: Int := -9223372036854775808\nwrite(minimum)\n"},
			expected: "-9223372036854775808\n",
		},
		{
			name: "html string generation smoke test",
			sources: map[string]string{"main.ahd": `title: String := "AhdCode"

html: String := """
<html>
<body>
<h1>{title}</h1>
</body>
</html>
"""

write(html)
`},
			expected: "\n<html>\n<body>\n<h1>AhdCode</h1>\n</body>\n</html>\n\n",
		},
		{
			name:     "terminal input",
			sources:  map[string]string{"main.ahd": "name: String := take(\"Name: \")\nwrite(\"Hello {name}\")\n"},
			stdin:    "Ali\n",
			expected: "Name: Hello Ali\n",
		},
		{
			name:       "int overflow is a runtime error",
			sources:    map[string]string{"main.ahd": "big: Int := 9223372036854775807\nwrite(big + 1)\n"},
			expected:   "",
			exitCode:   1,
			errorClass: "OverflowError",
		},
		{
			name:       "division by zero is a runtime error",
			sources:    map[string]string{"main.ahd": "zero: Real := 0.0\nwrite(1.0 / zero)\n"},
			expected:   "",
			exitCode:   1,
			errorClass: "DivisionByZeroError",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			entry := testCase.entry
			if entry == "" {
				entry = "main.ahd"
			}
			directory := writeSources(t, testCase.sources)
			out, errorOutput, code := buildAndRun(t, filepath.Join(directory, entry), testCase.stdin)
			if out != testCase.expected {
				t.Fatalf("stdout mismatch\n want %q\n have %q\n stderr: %s", testCase.expected, out, errorOutput)
			}
			if code != testCase.exitCode {
				t.Fatalf("exit code mismatch: want %d, have %d (stderr: %s)", testCase.exitCode, code, errorOutput)
			}
			if testCase.errorClass != "" && !strings.HasPrefix(errorOutput, testCase.errorClass+": ") {
				t.Fatalf("expected an uncaught %s on stderr; received %q", testCase.errorClass, errorOutput)
			}
			if testCase.exitCode != 0 && strings.Contains(errorOutput, "goroutine ") {
				t.Fatalf("a Go stack trace leaked into program output: %q", errorOutput)
			}
		})
	}
}

func TestRunProgramPropagatesExitCodeAndStreams(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "write(\"out\")\nbig: Int := 9223372036854775807\nwrite(big + 1)\n"})
	stdout, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatalf("could not create a capture file: %v", err)
	}
	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatalf("could not create a capture file: %v", err)
	}
	code, result := RunProgram(filepath.Join(directory, "main.ahd"), nil, nil, stdout, stderr)
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if code != 1 {
		t.Fatalf("expected exit code 1; received %d", code)
	}
	out, _ := os.ReadFile(stdout.Name())
	errorOutput, _ := os.ReadFile(stderr.Name())
	if string(out) != "out\n" {
		t.Fatalf("stdout was not propagated: %q", out)
	}
	if !strings.Contains(string(errorOutput), "OverflowError") {
		t.Fatalf("stderr was not propagated: %q", errorOutput)
	}
}

func TestFrontendErrorsStopBeforeCodeGeneration(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "x: Int := \"text\"\n"})
	path, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
	if path != "" || !result.HasErrors() {
		t.Fatal("expected the build to stop on a frontend error")
	}
	if result.Program != nil {
		t.Fatal("no Go program may be generated from failing source")
	}
}

func TestUndefinedAssignmentStopsBeforeCodeGeneration(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "score = 10\n"})
	path, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
	if path != "" || !result.HasErrors() {
		t.Fatal("expected undefined assignment to stop on a frontend error")
	}
	if result.Program != nil {
		t.Fatal("no Go program may be generated from an undefined assignment")
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "SEM001" {
		t.Fatalf("diagnostics = %+v, want one SEM001", result.Diagnostics)
	}
}

func TestUnsupportedNodeStopsTheBuildWithABackendDiagnostic(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `describe: Function := (
    action: Function
    value: Int
) -> String {
    result: Local Int := action(value)
    return "{str(action)}={result}"
}

double: Function := (
    n: Int
) -> Int {
    return n * 2
}

write(describe(double, 2))
`})
	path, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
	if path != "" {
		t.Fatal("expected no executable for an unsupported node")
	}
	found := false
	for _, item := range result.Diagnostics {
		if item.Code == backend.CodeUnsupportedNode {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s; received\n%s", backend.CodeUnsupportedNode, diagnosticText(result.Diagnostics))
	}
}

func TestBuildWorkspaceIsRemovedAndSourceTreeUntouched(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "write(\"hi\")\n"})
	before, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("could not read the source directory: %v", err)
	}
	result := Compile(filepath.Join(directory, "main.ahd"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	workspace, failures := NewWorkspace(result.Program)
	if len(failures) != 0 {
		t.Fatalf("workspace failures: %s", diagnosticText(failures))
	}
	location := workspace.Directory
	if _, err := os.Stat(filepath.Join(location, "go.mod")); err != nil {
		t.Fatalf("the workspace has no go.mod: %v", err)
	}
	workspace.Close()
	if _, err := os.Stat(location); !os.IsNotExist(err) {
		t.Fatal("the temporary workspace was not removed")
	}
	after, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("could not read the source directory: %v", err)
	}
	if len(before) != len(after) {
		t.Fatal("the user source tree was modified by the build")
	}
}

func TestGoToolchainIsDiscoveredThroughPath(t *testing.T) {
	located, err := FindGoToolchain()
	if err != nil {
		t.Skipf("no Go toolchain is available: %v", err)
	}
	if !filepath.IsAbs(located) {
		t.Fatalf("expected an absolute toolchain path; received %q", located)
	}
	if info, statError := os.Stat(located); statError != nil || info.IsDir() {
		t.Fatalf("the discovered toolchain is not an executable file: %v", statError)
	}
}

func TestDefaultOutputPathUsesTheEntryModuleName(t *testing.T) {
	path := DefaultOutputPath(filepath.Join("some", "where", "hello.ahd"))
	if filepath.Base(path) != "hello" {
		t.Fatalf("expected the default output to be named hello; received %q", filepath.Base(path))
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("expected an absolute default output path; received %q", path)
	}
}

func TestMissingToolchainIsReportedNotPanicked(t *testing.T) {
	original := os.Getenv("PATH")
	root := os.Getenv("GOROOT")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("GOROOT", t.TempDir())
	defer func() {
		_ = os.Setenv("PATH", original)
		_ = os.Setenv("GOROOT", root)
	}()
	if _, err := FindGoToolchain(); err == nil {
		t.Skip("a Go toolchain is still reachable from a standard install location")
	}
}
