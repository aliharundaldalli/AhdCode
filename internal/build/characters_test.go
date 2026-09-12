package build

import (
	"path/filepath"
	"strings"
	"testing"
)

// assertNoGoInternals fails when a user-facing failure leaks Go runtime detail.
func assertNoGoInternals(t *testing.T, output string) {
	t.Helper()
	for _, forbidden := range []string{"panic:", "goroutine ", "runtime error", "ahdruntime", ".go:"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("output leaks Go internals (%q): %q", forbidden, output)
		}
	}
}

func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}

// charactersProgram exercises every Characters operation, including the
// runtime value-domain failures, so its output pins native behavior exactly.
const charactersProgram = `bring Characters as C
from Characters bring CharactersError

text: String := "AhdCode – Türkçe: çğıöşü 😊"
write(str(C.count(text)))
write(str(len(text)))
write(C.list("Aş😊"))
write(C.list(""))
write(C.list("e" + C.fromCodePoint(769)))
write(str(C.codePoint("😊")))
write(C.fromCodePoint(C.codePoint("ğ")))
write(str(C.isLetter("İ")) + " " + str(C.isUpper("İ")) + " " + str(C.isLower("ı")))
write(str(C.isDigit("٣")) + " " + str(C.isDigit("²")))
write(str(C.isWhitespace("\t")) + " " + str(C.isPunctuation("–")) + " " + str(C.isSymbol("😊")))
write(str(C.isAlphaNumeric("7")) + " " + str(C.isAlphaNumeric("_")))

letters: Int := 0
for character in C.list(text) {
    if C.isLetter(character) {
        letters += 1
    }
}
write(str(letters))

attempt {
    write(str(C.codePoint("")))
} except CharactersError as error {
    write(error.message)
}
attempt {
    write(str(C.isLetter("AB")))
} except CharactersError as error {
    write(error.message)
}
attempt {
    write(C.fromCodePoint(1114112))
} except CharactersError as error {
    write(error.message)
}
attempt {
    write(C.fromCodePoint(57343))
} except CharactersError as error {
    write(error.message)
}
`

const charactersExpected = `26
26
["A", "ş", "😊"]
[]
["e", "́"]
128522
ğ
true true true
true false
true true true
true false
19
Characters.codePoint requires exactly one character; received an empty String
Characters.isLetter requires exactly one character; received 2 characters
Characters.fromCodePoint: 1114112 is outside the Unicode range 0..1114111
Characters.fromCodePoint: 57343 is a surrogate code point, not a Unicode scalar value
`

func TestCharactersRunsThroughNativeBackend(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": charactersProgram})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("Characters program failed: code=%d stderr=%q", code, stderr)
	}
	if stdout != charactersExpected {
		t.Fatalf("Characters native output mismatch:\n got: %q\nwant: %q", stdout, charactersExpected)
	}
}

// An uncaught CharactersError ends the program as an ordinary AhdCode error,
// never as a Go panic or stack trace.
func TestCharactersUncaughtErrorIsAhdCodeLevel(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "bring Characters\nwrite(str(Characters.codePoint(\"hello\")))\n"})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code == 0 {
		t.Fatalf("expected a failing exit code, got 0 (stdout=%q)", stdout)
	}
	assertNoGoInternals(t, stderr)
	if !containsAll(stderr, "CharactersError", "Characters.codePoint requires exactly one character; received 5 characters") {
		t.Fatalf("uncaught CharactersError was not reported at AhdCode level: %q", stderr)
	}
}

// A sibling Characters.ahd cannot hijack the builtin import.
func TestCharactersBuiltinWinsOverSiblingFile(t *testing.T) {
	directory := writeSources(t, map[string]string{
		"main.ahd":       "bring Characters\nwrite(str(Characters.count(\"şğ\")))\n",
		"Characters.ahd": "Public count: Function := (text: String) -> Int {\n    return 99\n}\n",
	})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stdout != "2\n" {
		t.Fatalf("builtin Characters did not take precedence: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}
