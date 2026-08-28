package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplicitNumericConversionsRunAsNativeCode(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `write(int(3.7))
write(int(-3.7))
write(real(2))
write(2 ^ 3)
attempt {
    write(2 ^ -3)
}
except DomainError {
    write("DomainError")
}
write(2.0 ^ -3)
write(real(2) ^ -3)
`})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q", code, stderr)
	}
	want := "3\n-3\n2.0\n8\nDomainError\n0.125\n0.125\n"
	if stdout != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", stdout, want)
	}
}

func TestInvalidNumericParsingFailsWithOneSignatureDiagnostic(t *testing.T) {
	for _, source := range []string{`write(int("1"))`, `write(real("1.5"))`} {
		path := filepath.Join(t.TempDir(), "main.ahd")
		if err := os.WriteFile(path, []byte(source+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		result := Compile(path)
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "SEM016" {
			t.Fatalf("diagnostics for %q = %+v", source, result.Diagnostics)
		}
		if strings.Contains(result.Diagnostics[0].Message, "unknown") || strings.Contains(result.Diagnostics[0].Message, "<invalid>") {
			t.Fatalf("unclean conversion diagnostic: %+v", result.Diagnostics[0])
		}
	}
}
