package evaluator

import (
	"os"
	"path/filepath"
	"testing"
)

// The REPL shares the native lookup and resolves a relative NAME_FILE path
// against the session's working directory.
func TestEnvSecretThroughEvaluator(t *testing.T) {
	session := newLatexTestSession()
	session.CWD = t.TempDir()
	const name = "AHDCODE_EVAL_SECRET"
	t.Setenv(name, "")
	os.Unsetenv(name)
	t.Setenv(name+"_FILE", "")
	os.Unsetenv(name + "_FILE")
	if value := session.envBuiltin("secret", []any{name}); value != nil {
		t.Fatalf("neither set, got %#v", value)
	}
	if err := os.WriteFile(filepath.Join(session.CWD, "db_password"), []byte("from-session-dir\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(name+"_FILE", "db_password")
	if value := session.envBuiltin("secret", []any{name}); value != "from-session-dir" {
		t.Fatalf("relative NAME_FILE read as %#v", value)
	}
	t.Setenv(name, "direct")
	message := evaluatorRaisedMessage(t, "EnvError", func() { session.envBuiltin("secret", []any{name}) })
	if message != name+" and "+name+"_FILE are both set; set only one" {
		t.Fatalf("both set message %q", message)
	}
	evaluatorRaisedMessage(t, "EnvError", func() { session.envBuiltin("secret", []any{""}) })
}
