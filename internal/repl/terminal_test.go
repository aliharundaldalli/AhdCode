package repl

import (
	"bytes"
	"strings"
	"testing"
)

// In the persistent REPL, Terminal.emit and Terminal.pretty write to the REPL's
// output and Terminal.error to its error stream, so the two never mix.
func TestTerminalStreamsInPersistentREPL(t *testing.T) {
	input := `bring Terminal
Terminal.emit(["Ali", "95"], " | ")
Terminal.error("only on stderr")
Terminal.pretty([1, 2])
write(str(Terminal.isInteractive()) + " " + str(Terminal.width() == null) + " " + Terminal.style(text: "plain", bold: true))
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v1.5.0")
	for _, want := range []string{"Ali | 95\n", "[\n    1,\n    2\n]\n", "false true plain\n"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("REPL output lacks %q:\n%s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "only on stderr") {
		t.Fatalf("Terminal.error reached the REPL output:\n%s", output.String())
	}
	if !strings.Contains(errors.String(), "only on stderr\n") {
		t.Fatalf("Terminal.error did not reach the REPL error stream: %q", errors.String())
	}
}
