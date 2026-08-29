package build

import (
	"path/filepath"
	"testing"
)

// TestFlexibleCommaStylesCompileAndRunIdentically is a v0.1.6 smoke test for
// the "AI models mix formatting styles" requirement: the same call, written
// with a same-line comma, a multi-line comma (including a trailing comma),
// and a newline-only separator, must compile and run identically. This
// exercises the runtime end of the pipeline, not just parsing.
func TestFlexibleCommaStylesCompileAndRunIdentically(t *testing.T) {
	directory := writeSources(t, map[string]string{
		"main.ahd": `add: Function := (
    x: Int
    y: Int
) -> Int {
    return x + y
}

write(add(2, 3))
write(add(
    2,
    3,
))
write(add(
    2
    3
))
`,
	})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	want := "5\n5\n5\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q -- the comma, multi-line-comma, and newline-only call styles must run identically", stdout, want)
	}
}

// TestFlexibleCommaAcrossEveryConstructRunsIdentically mixes the flexible
// separator styles across List literals, Pair literals, named call
// arguments, and Function parameters within a single program, the way a
// language model asked to write AhdCode is liable to.
func TestFlexibleCommaAcrossEveryConstructRunsIdentically(t *testing.T) {
	directory := writeSources(t, map[string]string{
		"main.ahd": `describe: Function := (
    name: String,
    age: Int
) -> String {
    return "{name} is {age}"
}

numbers: List<Int> := [
    1
    2
    3,
]

totals: Pair<String, Int> := {"a": 1, "b": 2}

total: Int := 0
for value in numbers {
    total = total + value
}
write(total)
write(totals["a"])
write(totals["b"])
write(describe(name: "Ali", age: 25))
`,
	})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	want := "6\n1\n2\nAli is 25\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}
