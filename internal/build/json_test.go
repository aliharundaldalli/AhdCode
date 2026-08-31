package build

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestJSONStandardLibraryRunsAsNativeExecutable(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "report.json")
	source := `bring JSON
from JSON bring JSONValue
from JSON bring JSONError

student: JSONValue := JSON.object({
    "name": JSON.fromString("Ali")
    "score": JSON.fromInt(91)
    "active": JSON.fromBool(true)
})

text: String := JSON.stringify(student, true)
write(text)

parsed: JSONValue := JSON.parse(text)
name: JSONValue? := parsed.get("name")
if name != null {
    write(name.string())
}

JSON.write(student, ` + strconv.Quote(output) + `, true)
loaded: JSONValue := JSON.read(` + strconv.Quote(output) + `)
score: JSONValue? := loaded.get("score")
if score != null {
    write(score.int())
}

array: JSONValue := JSON.array([JSON.fromInt(1), JSON.fromInt(2), JSON.fromInt(3)])
write(array.at(-1).int())
write(array.array())

attempt {
    JSON.parse(r"{bad}")
}
except JSONError as error {
    write(error.message)
}
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("JSON program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, want := range []string{
		"{\n  \"name\": \"Ali\",\n  \"score\": 91,\n  \"active\": true\n}",
		"Ali\n",
		"3\n",
		"[<JSONValue>, <JSONValue>, <JSONValue>]",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("native JSON output omitted %q:\n%s", want, stdout)
		}
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"score": 91`) {
		t.Fatalf("written JSON file content = %q", string(data))
	}
}
