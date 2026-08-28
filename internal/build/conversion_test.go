package build

import (
	"path/filepath"
	"testing"
)

func TestExplicitNumericConversionsRunAsNativeCode(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `write(int(3.7))
write(int(-3.7))
write(real(2))
write(int("  +42  "))
write(int("-9223372036854775808"))
write(real("3"))
write(real("3.14"))
write(real("1e3"))
write(real("-2.5e-4"))
write(2 ^ 3)
attempt {
    write(2 ^ -3)
}
except DomainError {
    write("DomainError")
}
write(2.0 ^ -3)
write(real(2) ^ -3)
attempt {
    write(int("3.0"))
}
except DomainError {
    write("Int DomainError")
}
attempt {
    write(real("NaN"))
}
except DomainError {
    write("Real DomainError")
}
attempt {
    write(int("9223372036854775808"))
}
except OverflowError {
    write("Int OverflowError")
}
attempt {
    write(real("1e309"))
}
except OverflowError {
    write("Real OverflowError")
}
`})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q", code, stderr)
	}
	want := "3\n-3\n2.0\n42\n-9223372036854775808\n3.0\n3.14\n1000.0\n-0.00025\n8\nDomainError\n0.125\n0.125\nInt DomainError\nReal DomainError\nInt OverflowError\nReal OverflowError\n"
	if stdout != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", stdout, want)
	}
}

func TestInvalidNumericInputTypesFailWithOneSignatureDiagnostic(t *testing.T) {
	for _, source := range []string{`write(int(true))`, `write(real(false))`} {
		path := filepath.Join(writeSources(t, map[string]string{"main.ahd": source + "\n"}), "main.ahd")
		result := Compile(path)
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "SEM016" {
			t.Fatalf("diagnostics for %q = %+v", source, result.Diagnostics)
		}
	}
}
