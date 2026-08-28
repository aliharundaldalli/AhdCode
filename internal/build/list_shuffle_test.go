package build

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestListShuffleRunsAsNativeCode(t *testing.T) {
	cases := []program{
		{
			name: "ordinary shuffle mutates in place and aliases observe it",
			sources: map[string]string{"main.ahd": `bring Math
Math.seed(42)
values: List<Int> := [1, 2, 3, 4, 5]
alias: List<Int> := values
values.shuffle()
write(values)
write(alias)
write(alias same values)
`},
			expected: "[2, 3, 1, 5, 4]\n[2, 3, 1, 5, 4]\ntrue\n",
		},
		{
			name: "seed 42 reproduces the same shuffle",
			sources: map[string]string{"main.ahd": `bring Math
first: List<Int> := [1, 2, 3, 4, 5]
second: List<Int> := [1, 2, 3, 4, 5]
Math.seed(42)
first.shuffle()
Math.seed(42)
second.shuffle()
write(first)
write(second)
write(first == second)
`},
			expected: "[2, 3, 1, 5, 4]\n[2, 3, 1, 5, 4]\ntrue\n",
		},
		{
			name: "entropy-initialized startup shuffle remains a permutation",
			sources: map[string]string{"main.ahd": `values: List<Int> := [1, 2, 3, 4, 5]
values.shuffle()
values.sort()
write(values)
`},
			expected: "[1, 2, 3, 4, 5]\n",
		},
		{
			name: "explicit seed 557 preserves its shuffle",
			sources: map[string]string{"main.ahd": `bring Math
Math.seed(557)
values: List<Int> := [1, 2, 3, 4, 5]
values.shuffle()
write(values)
`},
			expected: "[5, 2, 3, 1, 4]\n",
		},
		{
			name: "empty and singleton Lists do not consume RNG state",
			sources: map[string]string{"main.ahd": `bring Math
Math.seed(42)
expectedAfterEmpty: Real := Math.random()
Math.seed(42)
empty: List<Int> := []
empty.shuffle()
write(Math.random() == expectedAfterEmpty)

Math.seed(42)
expectedAfterSingleton: Real := Math.random()
Math.seed(42)
single: List<Int> := [7]
single.shuffle()
write(Math.random() == expectedAfterSingleton)
write(empty)
write(single)
`},
			expected: "true\ntrue\n[]\n[7]\n",
		},
		{
			name: "shuffle advances the shared random and randomInt sequence",
			sources: map[string]string{"main.ahd": `bring Math
Math.seed(42)
expectedBefore: Real := Math.random()
discard4: Int := Math.randomInt(0, 4)
discard3: Int := Math.randomInt(0, 3)
discard2: Int := Math.randomInt(0, 2)
discard1: Int := Math.randomInt(0, 1)
expectedAfter: Int := Math.randomInt(1, 10)

Math.seed(42)
actualBefore: Real := Math.random()
values: List<Int> := [1, 2, 3, 4, 5]
values.shuffle()
actualAfter: Int := Math.randomInt(1, 10)
write(actualBefore == expectedBefore)
write(actualAfter == expectedAfter)
`},
			expected: "true\ntrue\n",
		},
		{
			name: "a deep-frozen List rejects shuffle through an alias",
			sources: map[string]string{"main.ahd": `values: Constant List<Int> := [1, 2, 3]
alias: List<Int> := values
attempt {
    alias.shuffle()
} except ConstantError as error {
    write("blocked")
}
write(values)
write(alias)
`},
			expected: "blocked\n[1, 2, 3]\n[1, 2, 3]\n",
		},
		{
			name: "reverse and sort retain their existing behavior",
			sources: map[string]string{"main.ahd": `reversed: List<Int> := [1, 2, 3]
sorted: List<Int> := [8, 3, 12, 5]
reversed.reverse()
sorted.sort()
write(reversed)
write(sorted)
`},
			expected: "[3, 2, 1]\n[3, 5, 8, 12]\n",
		},
	}
	runProgramCases(t, cases)
}

func TestSeededListShuffleExecutionsAndGeneratedOutputAreDeterministic(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `bring Math
Math.seed(557)
values: List<Int> := [1, 2, 3, 4, 5]
values.shuffle()
write(values)
`})
	entry := filepath.Join(directory, "main.ahd")
	output := filepath.Join(t.TempDir(), "program")
	path, result := BuildProgram(entry, output)
	if result.HasErrors() {
		t.Fatalf("shuffle build failed: %s", diagnosticText(result.Diagnostics))
	}
	run := func() string {
		command := exec.Command(path)
		bytes, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("shuffle executable failed: %v (%s)", err, bytes)
		}
		return string(bytes)
	}
	if first, second := run(), run(); first != second || first != "[5, 2, 3, 1, 4]\n" {
		t.Fatalf("explicitly seeded shuffle outputs = %q and %q", first, second)
	}

	first := Compile(entry)
	second := Compile(entry)
	if first.HasErrors() || second.HasErrors() || len(first.Program.Files) != len(second.Program.Files) {
		t.Fatal("repeated shuffle generation failed")
	}
	for index := range first.Program.Files {
		if first.Program.Files[index] != second.Program.Files[index] {
			t.Fatalf("generated shuffle file %d changed across identical compilations", index)
		}
	}
}

func TestListShuffleFrontendRejections(t *testing.T) {
	cases := map[string]string{
		"Constant receiver":   "values: Constant List<Int> := [1, 2, 3]\nvalues.shuffle()\n",
		"nullable receiver":   "values: List<Int> := null\nvalues.shuffle()\n",
		"positional argument": "values: List<Int> := [1, 2, 3]\nvalues.shuffle(1)\n",
		"named argument":      "values: List<Int> := [1, 2, 3]\nvalues.shuffle(value: 1)\n",
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			directory := writeSources(t, map[string]string{"main.ahd": source})
			_, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
			if !result.HasErrors() {
				t.Fatal("expected a compile-time rejection")
			}
			for _, diagnostic := range result.Diagnostics {
				if !strings.HasPrefix(diagnostic.Code, "SEM") {
					t.Fatalf("diagnostics = %s, want only semantic diagnostics", diagnosticText(result.Diagnostics))
				}
			}
		})
	}
}
