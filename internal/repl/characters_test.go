package repl

import (
	"bytes"
	"strings"
	"testing"
)

// The persistent REPL runs Characters through the evaluator, which calls the
// same runtime functions a compiled program does. Every line below must match
// internal/build's native expectation for the same program.
func TestCharactersMatchesNativeInPersistentREPL(t *testing.T) {
	input := `bring Characters as C
from Characters bring CharactersError
text: String := "AhdCode – Türkçe: çğıöşü 😊"
write(str(C.count(text)))
write(C.list("Aş😊"))
write(C.list("e" + C.fromCodePoint(769)))
write(str(C.codePoint("😊")))
write(C.fromCodePoint(C.codePoint("ğ")))
write(str(C.isLetter("İ")) + " " + str(C.isUpper("İ")) + " " + str(C.isLower("ı")))
write(str(C.isDigit("٣")) + " " + str(C.isDigit("²")))
write(str(C.isWhitespace("\t")) + " " + str(C.isPunctuation("–")) + " " + str(C.isSymbol("😊")))
attempt {
    write(str(C.codePoint("")))
} except CharactersError as error {
    write(error.message)
}
attempt {
    write(C.fromCodePoint(57343))
} except CharactersError as error {
    write(error.message)
}
write(str(C.isLetter("AB")))
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v1.2.0")
	want := []string{
		"26\n",
		"[\"A\", \"ş\", \"😊\"]\n",
		"[\"e\", \"\u0301\"]\n",
		"128522\n",
		"ğ\n",
		"true true true\n",
		"true false\n",
		"true true true\n",
		"Characters.codePoint requires exactly one character; received an empty String\n",
		"Characters.fromCodePoint: 57343 is a surrogate code point, not a Unicode scalar value\n",
	}
	rest := output.String()
	for _, line := range want {
		index := strings.Index(rest, line)
		if index < 0 {
			t.Fatalf("REPL output missing %q in order:\n%s\nerrors:\n%s", line, output.String(), errors.String())
		}
		rest = rest[index+len(line):]
	}
	// The uncaught failure is reported by the REPL, which keeps running.
	if !strings.Contains(errors.String(), "CharactersError") ||
		!strings.Contains(errors.String(), "Characters.isLetter requires exactly one character; received 2 characters") {
		t.Fatalf("REPL did not report the uncaught CharactersError:\n%s", errors.String())
	}
	for _, forbidden := range []string{"panic:", "goroutine "} {
		if strings.Contains(errors.String(), forbidden) {
			t.Fatalf("REPL leaked Go internals: %s", errors.String())
		}
	}
}
