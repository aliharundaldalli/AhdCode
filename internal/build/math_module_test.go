package build

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMathModuleRunsAsNativeCode(t *testing.T) {
	cases := []program{
		{
			name: "constants and classic functions",
			sources: map[string]string{"main.ahd": `bring Math
write(Math.PI)
write(Math.E)
write(Math.sqrt(25))
write(Math.sin(0) == 0.0)
write(Math.cos(0) == 1.0)
write(Math.tan(0.5) > 0.0)
write(Math.log(Math.E) == 1.0)
write(Math.log10(100) == 2.0)
write(Math.exp(0) == 1.0)
`},
			expected: "3.141592653589793\n2.718281828459045\n5.0\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\n",
		},
		{
			name: "round floor and ceil",
			sources: map[string]string{"main.ahd": `bring Math
write(Math.round(3.4))
write(Math.round(3.5))
write(Math.round(-3.5))
write(Math.round(3.14159, 2))
write(Math.round(2.675, 2))
write(Math.floor(3.9))
write(Math.floor(-3.1))
write(Math.ceil(3.1))
write(Math.ceil(-3.9))
`},
			expected: "3.0\n4.0\n-4.0\n3.14\n2.68\n3\n-4\n4\n-3\n",
		},
		{
			name: "direct selective imports",
			sources: map[string]string{"main.ahd": `from Math bring (
    PI
    sqrt
)
write(PI)
write(sqrt(81))
`},
			expected: "3.141592653589793\n9.0\n",
		},
		{
			name: "Math callables remain ordinary Function values",
			sources: map[string]string{"main.ahd": `bring Math
apply: Function := (
    operation: Function
    value: Real
) -> Real {
    return operation(value)
}
values: List<Real> := [1.0, 4.0, 9.0]
write(values.map(Math.sqrt))
operation: Function := Math.sqrt
write(operation(16.0))
write(str(Math.sqrt))
write(apply(Math.round, 3.5))
`},
			expected: "[1.0, 2.0, 3.0]\n4.0\n<Function sqrt>\n4.0\n",
		},
		{
			name: "entropy-initialized random remains in range",
			sources: map[string]string{"main.ahd": `bring Math
value: Real := Math.random()
write(value >= 0.0 and value < 1.0)
`},
			expected: "true\n",
		},
		{
			name: "explicit seed 557 preserves its golden sequence",
			sources: map[string]string{"main.ahd": `bring Math
Math.seed(557)
write(Math.random())
write(Math.random())
write(Math.random())
`},
			expected: "0.4121990632081577\n0.4686510900868295\n0.5840201876345011\n",
		},
		{
			name: "seed reset and inclusive randomInt",
			sources: map[string]string{"main.ahd": `bring Math
Math.seed(42)
first: Real := Math.random()
Math.seed(42)
write(first == Math.random())
Math.seed(42)
for i: Int in between(5) {
    write(Math.randomInt(1, 10))
}
`},
			expected: "true\n4\n2\n9\n5\n1\n",
		},
		{
			name: "singleton randomInt does not consume state",
			sources: map[string]string{"main.ahd": `bring Math
Math.seed(42)
expected: Real := Math.random()
Math.seed(42)
write(Math.randomInt(-7, -7))
write(expected == Math.random())
`},
			expected: "-7\ntrue\n",
		},
		{
			name:  "shared sequence across modules",
			entry: "Main.ahd",
			sources: map[string]string{
				"A.ahd":    "bring Math\nMath.seed(557)\nfirst: Real := Math.random()",
				"B.ahd":    "bring Math\nsecond: Real := Math.random()",
				"Main.ahd": "bring Math\nfrom A bring first\nfrom B bring second\nwrite(first)\nwrite(second)\nwrite(Math.random())",
			},
			expected: "0.4121990632081577\n0.4686510900868295\n0.5840201876345011\n",
		},
	}
	runProgramCases(t, cases)
}

func TestMathRuntimeErrorsAreCatchable(t *testing.T) {
	source := `bring Math
attempt { write(Math.sqrt(-1.0)) } except DomainError as error { write("sqrt") }
attempt { write(Math.log(0.0)) } except DomainError as error { write("log zero") }
attempt { write(Math.log(-1.0)) } except DomainError as error { write("log negative") }
attempt { write(Math.round(1.2, -1)) } except DomainError as error { write("round low") }
attempt { write(Math.round(1.2, 16)) } except DomainError as error { write("round high") }
attempt { write(Math.floor(9223372036854775808.0)) } except OverflowError as error { write("floor") }
attempt { write(Math.ceil(9223372036854775808.0)) } except OverflowError as error { write("ceil") }
attempt { write(Math.exp(1000.0)) } except OverflowError as error { write("exp") }
attempt { write(Math.randomInt(10, 1)) } except DomainError as error { write("bounds") }
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	want := "sqrt\nlog zero\nlog negative\nround low\nround high\nfloor\nceil\nexp\nbounds\n"
	if out != want || code != 0 {
		t.Fatalf("Math catchability\n want %q\n have %q (exit %d, stderr %s)", want, out, code, errorOutput)
	}
}

func TestMathWrongTypesStopAtSemanticDiagnostics(t *testing.T) {
	cases := map[string]string{
		"sqrt String":       "bring Math\nwrite(Math.sqrt(\"9\"))",
		"random arity":      "bring Math\nwrite(Math.random(1))",
		"randomInt Real":    "bring Math\nwrite(Math.randomInt(1.0, 10))",
		"seed String":       "bring Math\nMath.seed(\"557\")",
		"round Real digits": "bring Math\nwrite(Math.round(1.2, 2.0))",
		"nullable input":    "bring Math\nvalue: Real := null\nwrite(Math.sqrt(value))",
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			directory := writeSources(t, map[string]string{"main.ahd": text})
			_, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
			if !result.HasErrors() {
				t.Fatal("expected a semantic rejection")
			}
			if len(result.Diagnostics) != 1 {
				t.Fatalf("diagnostics = %s, want one root diagnostic", diagnosticText(result.Diagnostics))
			}
			for _, diagnostic := range result.Diagnostics {
				if !strings.HasPrefix(diagnostic.Code, "SEM") {
					t.Fatalf("diagnostics = %s, want only semantic diagnostics", diagnosticText(result.Diagnostics))
				}
			}
		})
	}
}

func TestMathExplicitSeedExecutionsAndGeneratedOutputAreDeterministic(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "bring Math\nMath.seed(557)\nwrite(Math.random())\nwrite(Math.randomInt(1, 10))"})
	entry := filepath.Join(directory, "main.ahd")
	output := filepath.Join(t.TempDir(), "program")
	path, result := BuildProgram(entry, output)
	if result.HasErrors() {
		t.Fatalf("Math build failed: %s", diagnosticText(result.Diagnostics))
	}
	run := func() string {
		command := exec.Command(path)
		bytes, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("Math executable failed: %v (%s)", err, bytes)
		}
		return string(bytes)
	}
	if first, second := run(), run(); first != second || first != "0.4121990632081577\n3\n" {
		t.Fatalf("explicitly seeded execution outputs = %q and %q", first, second)
	}
	first := Compile(entry)
	second := Compile(entry)
	if first.HasErrors() || second.HasErrors() || len(first.Program.Files) != len(second.Program.Files) {
		t.Fatal("repeated Math generation failed")
	}
	for index := range first.Program.Files {
		if first.Program.Files[index] != second.Program.Files[index] {
			t.Fatalf("generated file %d changed across identical compilations", index)
		}
	}
}
