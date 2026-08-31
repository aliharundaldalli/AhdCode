package build

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestEnvStandardLibraryRunsAsNativeExecutable(t *testing.T) {
	directory := t.TempDir()
	envPath := filepath.Join(directory, "sample.env")
	envContent := "PORT=8080\n" +
		"NAME=\"Ali Harun\"\n" +
		"EMPTY=\n" +
		"# a comment\n"
	if err := os.WriteFile(envPath, []byte(envContent), 0o600); err != nil {
		t.Fatal(err)
	}

	source := `bring Env
from Env bring EnvError

Env.load(` + strconv.Quote(envPath) + `)
port: String := Env.getOr("PORT", "3000")
write(port)
name: String? := Env.get("NAME")
if name != null {
    write(name)
}
write(Env.exists("EMPTY"))
write(Env.exists("NOPE"))
write(Env.getOr("NOPE", "default"))

Env.set("CUSTOM", "value")
write(Env.get("CUSTOM"))
Env.unset("CUSTOM")
write(Env.exists("CUSTOM"))

record: Pair<String, String> := Env.read(` + strconv.Quote(envPath) + `)
write(record)

attempt {
    Env.set("BAD=NAME", "x")
}
except EnvError as error {
    write(error.message)
}
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("Env program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, want := range []string{
		"8080\n", "Ali Harun\n", "true\n", "false\n", "default\n",
		"value\n", "false\n", `{"PORT": "8080", "NAME": "Ali Harun", "EMPTY": ""}`,
		"environment variable name must not contain '='",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("native Env output omitted %q:\n%s", want, stdout)
		}
	}
}
