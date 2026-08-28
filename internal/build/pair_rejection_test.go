package build

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestPairKeyProgramsAreRejectedBeforeCodeGeneration keeps the v0.1 Pair key
// rules enforced on the real compile path, so an invalid program never reaches
// a backend or produces an executable.
func TestPairKeyProgramsAreRejectedBeforeCodeGeneration(t *testing.T) {
	cases := []struct {
		name   string
		source string
		code   string
	}{
		{
			name: "duplicate literal key",
			source: `scores: Pair<String, Int> := {
    "Ali": 80
    "Ayşe": 90
    "Ali": 100
}

write(scores)
`,
			code: "SEM036",
		},
		{
			name: "Real key type",
			source: `values: Pair<Real, String> := {
    1.5: "A"
    2.5: "B"
}

write(values)
`,
			code: "SEM035",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			directory := writeSources(t, map[string]string{"main.ahd": testCase.source})
			result := Compile(filepath.Join(directory, "main.ahd"))
			if !result.HasErrors() {
				t.Fatal("expected the program to be rejected")
			}
			if result.Program != nil {
				t.Fatal("no Go program may be generated from a rejected Pair literal")
			}
			found := false
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Code == testCase.code {
					found = true
				}
				if strings.HasPrefix(diagnostic.Code, "BCK") {
					t.Fatalf("a backend diagnostic followed the semantic error: %s", diagnostic.Message)
				}
			}
			if !found {
				t.Fatalf("expected %s; received\n%s", testCase.code, diagnosticText(result.Diagnostics))
			}
		})
	}
}

// TestValidPairKeyProgramRunsAsNativeExecutable keeps the accepted Pair key
// types working end to end.
func TestValidPairKeyProgramRunsAsNativeExecutable(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `byName: Pair<String, Int> := {
    "Ali": 80
    "Ayşe": 90
}

byNumber: Pair<Int, String> := {
    1: "one"
    2: "two"
}

byFlag: Pair<Bool, String> := {
    true: "yes"
    false: "no"
}

write(byName)
write(byNumber)
write(byFlag)
write(len(byName))
`})
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	expected := "{\"Ali\": 80, \"Ayşe\": 90}\n{1: \"one\", 2: \"two\"}\n{true: \"yes\", false: \"no\"}\n2\n"
	if out != expected {
		t.Fatalf("stdout mismatch\n want %q\n have %q\n stderr: %s", expected, out, errorOutput)
	}
	if code != 0 {
		t.Fatalf("program exited with %d (stderr: %s)", code, errorOutput)
	}
}
