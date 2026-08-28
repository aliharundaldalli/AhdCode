package build

import (
	"path/filepath"
	"testing"
)

func TestCrossModuleFunctionParameterNamesDoNotChangeRuntimeCompatibility(t *testing.T) {
	directory := writeSources(t, map[string]string{
		"Numbers.ahd": `triple: Function := (
    value: Int
) -> Int {
    return value * 3
}`,
		"Engine.ahd": `applyTwice: Function := (
    operation: Function
    value: Int
) -> Int {
    first: Local Int := operation(value)
    return operation(first)
}`,
		"main.ahd": `bring Engine
from Numbers bring triple

operation: Function := triple

write(Engine.applyTwice(operation, 2))
write(Engine.applyTwice(triple, 3))
`,
	})

	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" || stdout != "18\n27\n" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}
