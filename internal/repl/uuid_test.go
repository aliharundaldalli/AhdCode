package repl

import (
	"bytes"
	"strings"
	"testing"
)

// The persistent REPL creates and compares UUIDs through the evaluator, which
// calls the same runtime a compiled program does, so its results match
// internal/build's native expectations.
func TestUUIDMatchesNativeInPersistentREPL(t *testing.T) {
	input := `bring UUID
from UUID bring (UUIDValue, UUIDError)
parsed: UUIDValue := UUID.parse("017F22E2-79B0-7CC3-98C4-DC0C0C07398F")
write(parsed.string() + " " + str(parsed.version()))
first: UUIDValue := UUID.v7()
second: UUIDValue := UUID.v7()
write(str(first.compare(second)) + " " + str(first.equals(first)) + " " + str(first == UUID.parse(first.string())))
write(str(UUID.zero().isZero()) + " " + str(UUID.isValid(r"{017f22e2-79b0-7cc3-98c4-dc0c0c07398f}")))
write(str(parsed))
attempt {
    UUID.parse("urn:uuid:017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
} except UUIDError as error {
    write(error.message)
}
UUID.parse("")
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v1.3.0")
	want := []string{
		"017f22e2-79b0-7cc3-98c4-dc0c0c07398f 7\n",
		"-1 true false\n",
		"true false\n",
		"<UUIDValue>\n",
		"UUID text must be 36 characters in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx\n",
	}
	rest := output.String()
	for _, line := range want {
		index := strings.Index(rest, line)
		if index < 0 {
			t.Fatalf("REPL output missing %q in order:\n%s\nerrors:\n%s", line, output.String(), errors.String())
		}
		rest = rest[index+len(line):]
	}
	if !strings.Contains(errors.String(), "UUIDError") || !strings.Contains(errors.String(), "UUID text must be 36 characters") {
		t.Fatalf("uncaught UUIDError not reported by the REPL: %q", errors.String())
	}
}
