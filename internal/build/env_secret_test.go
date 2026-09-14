package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvSecretRunsAsNativeExecutable(t *testing.T) {
	directory := t.TempDir()
	secretPath := filepath.Join(directory, "db_password")
	if err := os.WriteFile(secretPath, []byte("s3cr3t with spaces \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_SECRET_DIRECT", "hunter2")
	t.Setenv("AHDCODE_SECRET_FILE_BASED_FILE", secretPath)
	t.Setenv("AHDCODE_SECRET_MISSING", "")
	os.Unsetenv("AHDCODE_SECRET_MISSING")
	t.Setenv("AHDCODE_SECRET_BOTH", "one")
	t.Setenv("AHDCODE_SECRET_BOTH_FILE", secretPath)
	t.Setenv("AHDCODE_SECRET_GONE_FILE", filepath.Join(directory, "does-not-exist"))

	source := `bring Env
from Env bring EnvError

direct: String? := Env.secret("AHDCODE_SECRET_DIRECT")
if direct != null {
    write("direct " + direct)
}
fromFile: String? := Env.secret("AHDCODE_SECRET_FILE_BASED")
if fromFile != null {
    write("file [" + fromFile + "]")
}
missing: String? := Env.secret("AHDCODE_SECRET_MISSING")
write(missing == null)
attempt {
    Env.secret("AHDCODE_SECRET_BOTH")
} except EnvError as error {
    write(error.message)
}
attempt {
    Env.secret("AHDCODE_SECRET_GONE")
} except EnvError as error {
    write(error.message)
}
`
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRun(t, entry, "")
	if code != 0 || stderr != "" {
		t.Fatalf("Env.secret program failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	want := "direct hunter2\n" +
		"file [s3cr3t with spaces ]\n" +
		"true\n" +
		"AHDCODE_SECRET_BOTH and AHDCODE_SECRET_BOTH_FILE are both set; set only one\n" +
		"the secret file named by AHDCODE_SECRET_GONE_FILE could not be read\n"
	if stdout != want {
		t.Fatalf("Env.secret native output mismatch:\n got: %q\nwant: %q", stdout, want)
	}
	if strings.Contains(stdout, directory) {
		t.Fatalf("output leaks a secret file path: %q", stdout)
	}
}
