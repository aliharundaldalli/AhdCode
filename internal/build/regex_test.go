package build

import (
	"path/filepath"
	"testing"
)

func TestRegexModuleBuildsAndRunsNatively(t *testing.T) {
	source := `bring Regex
from Regex bring RegexError
from Regex bring Pattern

digits: Pattern := Regex.compile("[0-9]+")

write(digits.matches("abc123"))
write(digits.matches("abcdef"))
write(digits.find("abc123def456"))
write(digits.findAll("abc123def456"))
write(digits.replace("abc123def456", "#"))
write(digits.split("abc123def456ghi"))

pair: Pattern := Regex.compile("([a-z]+)-([0-9]+)")
write(pair.groups("item-42"))

missing: String? := digits.find("no numbers here")
write(missing == null)

attempt {
    Regex.compile("(unclosed")
    write("unreachable")
}
except RegexError as error {
    write("caught")
}
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || errorOutput != "" {
		t.Fatalf("regex program failed: code=%d stderr=%s", code, errorOutput)
	}
	want := "true\nfalse\n123\n[\"123\", \"456\"]\nabc#def#\n[\"abc\", \"def\", \"ghi\"]\n" +
		"[\"item-42\", \"item\", \"42\"]\ntrue\ncaught\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}
