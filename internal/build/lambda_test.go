package build

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLambdaExpressionsRunAsNativeFunctionValues(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `Student: Class<> := {
    structure: Attributes := (score: Int)
}

offset: Int := 10
useGlobal: Function := () -> Int {
    offset: Global Int
    addOffset: Local := lambda (value: Int) -> value + offset
    return addOffset(5)
}

square := lambda (x: Int) -> x ^ 2
positive: Function := lambda (x: Int) -> x > 0
constant := lambda () -> 42
combine := lambda (x: Int, y: Int, z: Real, text: String) -> real(x + y) > z and text != ""
score := lambda (student: Student) -> student.score
present := lambda (student: Student?) -> student != null
announce := lambda (text: String) -> write(text)
usesPredicate := lambda (operation: Function) -> [1].filter(operation)[0] > 0

values := [1, 2, 3]
write(square(5))
write(positive(-1))
write(constant())
write(combine(2, 3, 4.5, "ok"))
write(score(Student(score: 9)))
write(present(null))
announce("ready")
write(usesPredicate(positive))
write(useGlobal())
write(values.map(lambda (x: Int) -> x ^ 2))
write(values.filter(lambda (x: Int) -> x > 1))
values.sort(lambda (x: Int) -> -x)
write(values)
`})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q", code, stderr)
	}
	want := strings.Join([]string{"25", "false", "42", "true", "9", "false", "ready", "true", "15", "[1, 4, 9]", "[2, 3]", "[3, 2, 1]", ""}, "\n")
	if stdout != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", stdout, want)
	}
}

func TestInvalidLambdasFailBeforeNativeGeneration(t *testing.T) {
	tests := map[string]string{
		"wrong argument": `f := lambda (x: Int) -> x ^ 2
write(f("5"))`,
		"capture": `outer: Function := (offset: Int) -> Function {
    return lambda (x: Int) -> x + offset
}`,
		"nullable member": `Student: Class<> := { structure: Attributes := (score: Int) }
f := lambda (student: Student?) -> student.score`,
		"default parameter": `f := lambda (x: Int := 1) -> x + 1`,
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			directory := writeSources(t, map[string]string{"main.ahd": source + "\n"})
			result := Compile(filepath.Join(directory, "main.ahd"))
			if !result.HasErrors() {
				t.Fatal("invalid lambda compiled without diagnostics")
			}
		})
	}
}
