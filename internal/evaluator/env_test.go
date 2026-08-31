package evaluator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvEvaluatorGetHasGetOrDistinguishMissingFromEmpty(t *testing.T) {
	session := newLatexTestSession()
	name := "AHDCODE_EVAL_TEST_MISSING"
	os.Unsetenv(name)
	if session.envBuiltin("get", []any{name}) != nil {
		t.Fatal("get(missing) != nil")
	}
	if session.envBuiltin("exists", []any{name}) != false {
		t.Fatal("exists(missing) != false")
	}
	if session.envBuiltin("getOr", []any{name, "fallback"}) != "fallback" {
		t.Fatal("getOr(missing) did not use fallback")
	}

	empty := "AHDCODE_EVAL_TEST_EMPTY"
	t.Cleanup(func() { os.Unsetenv(empty) })
	os.Setenv(empty, "")
	if session.envBuiltin("get", []any{empty}) != "" {
		t.Fatal("get(present-empty) did not return \"\"")
	}
	if session.envBuiltin("exists", []any{empty}) != true {
		t.Fatal("exists(present-empty) != true")
	}
	if session.envBuiltin("getOr", []any{empty, "fallback"}) != "" {
		t.Fatal("getOr(present-empty) used the fallback instead of the empty value")
	}
}

func TestEnvEvaluatorSetAndUnset(t *testing.T) {
	session := newLatexTestSession()
	name := "AHDCODE_EVAL_TEST_SET"
	t.Cleanup(func() { os.Unsetenv(name) })
	if got := session.envBuiltin("set", []any{name, "value"}); got != Nothing {
		t.Fatalf("set returned %#v", got)
	}
	if session.envBuiltin("get", []any{name}) != "value" {
		t.Fatal("get after set")
	}
	session.envBuiltin("unset", []any{name})
	if session.envBuiltin("exists", []any{name}) != false {
		t.Fatal("exists after unset")
	}
}

func TestEnvEvaluatorSetRejectsInvalidNames(t *testing.T) {
	session := newLatexTestSession()
	expectEvaluatorRaise(t, "EnvError", func() { session.envBuiltin("set", []any{"", "v"}) })
	expectEvaluatorRaise(t, "EnvError", func() { session.envBuiltin("set", []any{"A=B", "v"}) })
	expectEvaluatorRaise(t, "EnvError", func() { session.envBuiltin("unset", []any{""}) })
}

func TestEnvEvaluatorDotenvGrammarAndOrder(t *testing.T) {
	session := newLatexTestSession()
	content := "Z=1\nA=\"two\"\nM='three'\n# comment\n\nEMPTY=\n"
	entries := session.envParseFile(content)
	if len(entries) != 4 {
		t.Fatalf("entries = %v", entries)
	}
	if entries[0].Key != "Z" || entries[1].Key != "A" || entries[2].Key != "M" || entries[3].Key != "EMPTY" {
		t.Fatalf("order not preserved: %v", entries)
	}
	if entries[1].Value != "two" || entries[2].Value != "three" || entries[3].Value != "" {
		t.Fatalf("values wrong: %v", entries)
	}
}

func TestEnvEvaluatorMalformedInputRaisesEnvError(t *testing.T) {
	session := newLatexTestSession()
	for _, content := range []string{
		"NOEQUALS",
		"1KEY=value",
		`KEY="unterminated`,
		"DUP=1\nDUP=2\n",
	} {
		content := content
		expectEvaluatorRaise(t, "EnvError", func() { session.envParseFile(content) })
	}
}

func TestEnvEvaluatorNoShellInterpolationOrExecution(t *testing.T) {
	session := newLatexTestSession()
	marker := filepath.Join(t.TempDir(), "should_not_exist")
	entries := session.envParseFile("VALUE=$(touch " + marker + ")\n")
	if entries[0].Value != "$(touch "+marker+")" {
		t.Fatalf("value was interpreted: %q", entries[0].Value)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("command substitution was executed")
	}
}

func TestEnvEvaluatorReadDoesNotMutateProcessEnvironment(t *testing.T) {
	session := newLatexTestSession()
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	name := "AHDCODE_EVAL_TEST_READ_ONLY"
	os.Unsetenv(name)
	os.WriteFile(path, []byte(name+"=value\n"), 0o600)
	pair := session.envBuiltin("read", []any{path}).(*Pair)
	if len(pair.Keys) != 1 || pair.Keys[0] != name || pair.Values[name] != "value" {
		t.Fatalf("read result = %v", pair)
	}
	if session.envBuiltin("exists", []any{name}) != false {
		t.Fatal("Env.read mutated the process environment")
	}
}

func TestEnvEvaluatorLoadOverridePrecedence(t *testing.T) {
	session := newLatexTestSession()
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	existing := "AHDCODE_EVAL_TEST_EXISTING"
	t.Cleanup(func() { os.Unsetenv(existing) })
	os.Setenv(existing, "process-value")
	os.WriteFile(path, []byte(existing+"=dotenv-value\n"), 0o600)

	session.envBuiltin("load", []any{path, false})
	if session.envBuiltin("get", []any{existing}) != "process-value" {
		t.Fatal("override=false did not preserve the existing value")
	}
	session.envBuiltin("load", []any{path, true})
	if session.envBuiltin("get", []any{existing}) != "dotenv-value" {
		t.Fatal("override=true did not apply the .env value")
	}
}

func TestEnvEvaluatorLoadValidatesWholeFileBeforeApplying(t *testing.T) {
	session := newLatexTestSession()
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	untouched := "AHDCODE_EVAL_TEST_HALF_APPLY"
	os.Unsetenv(untouched)
	t.Cleanup(func() { os.Unsetenv(untouched) })
	os.WriteFile(path, []byte(untouched+"=value\nBADLINE\n"), 0o600)
	expectEvaluatorRaise(t, "EnvError", func() { session.envBuiltin("load", []any{path, false}) })
	if session.envBuiltin("exists", []any{untouched}) != false {
		t.Fatal("Env.load applied an entry before failing on a later malformed line")
	}
}
