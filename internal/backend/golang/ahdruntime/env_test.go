package ahdruntime

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// envChildHelperFlag switches this same test binary into "child helper"
// mode: it prints the target variable's value (or envChildAbsentSentinel if
// unset) to stdout and returns, instead of running the parent test body.
// This is the standard os/exec-based technique for testing real child-
// process environment inheritance without any shell.
const (
	envChildHelperFlag     = "AHDCODE_ENV_CHILD_HELPER"
	envChildTargetVar      = "AHDCODE_TEST_ENV_CHILD_VAR"
	envChildAbsentSentinel = "<absent>"
)

func TestEnvSetUnsetVisibleToChildProcess(t *testing.T) {
	if os.Getenv(envChildHelperFlag) == "1" {
		// os.Exit, not return: returning would let the testing framework
		// print its own "PASS" line, polluting the output this test reads.
		value, present := os.LookupEnv(envChildTargetVar)
		if !present {
			value = envChildAbsentSentinel
		}
		os.Stdout.WriteString(value)
		os.Exit(0)
	}

	os.Unsetenv(envChildTargetVar)
	t.Cleanup(func() { os.Unsetenv(envChildTargetVar) })

	runChild := func() string {
		t.Helper()
		command := exec.Command(os.Args[0], "-test.run=^TestEnvSetUnsetVisibleToChildProcess$")
		command.Env = append(os.Environ(), envChildHelperFlag+"=1")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("child process failed: %v (output=%q)", err, output)
		}
		return strings.TrimRight(string(output), "\r\n")
	}

	AhdEnvSet(AhdClassEnvError, envChildTargetVar, "child-value")
	if got := runChild(); got != "child-value" {
		t.Fatalf("child process did not inherit Env.set: got %q, want %q", got, "child-value")
	}

	AhdEnvUnset(AhdClassEnvError, envChildTargetVar)
	if got := runChild(); got != envChildAbsentSentinel {
		t.Fatalf("child process still saw the unset variable: got %q, want %q", got, envChildAbsentSentinel)
	}
}

func TestEnvGetHasGetOrDistinguishMissingFromEmpty(t *testing.T) {
	name := "AHDCODE_TEST_ENV_VAR_MISSING"
	os.Unsetenv(name)
	if got := AhdEnvGet(name); got != nil {
		t.Fatalf("Get(missing) = %v, want nil", got)
	}
	if AhdEnvHas(name) {
		t.Fatal("Has(missing) = true")
	}
	if got := AhdEnvGetOr(name, "fallback"); got != "fallback" {
		t.Fatalf("GetOr(missing) = %q", got)
	}

	empty := "AHDCODE_TEST_ENV_VAR_EMPTY"
	t.Cleanup(func() { os.Unsetenv(empty) })
	if err := os.Setenv(empty, ""); err != nil {
		t.Fatal(err)
	}
	if got := AhdEnvGet(empty); got == nil || *got != "" {
		t.Fatalf("Get(present-empty) = %v, want a pointer to \"\"", got)
	}
	if !AhdEnvHas(empty) {
		t.Fatal("Has(present-empty) = false")
	}
	if got := AhdEnvGetOr(empty, "fallback"); got != "" {
		t.Fatalf("GetOr(present-empty) = %q, want \"\" (fallback only applies when absent)", got)
	}
}

func TestEnvSetAndUnset(t *testing.T) {
	name := "AHDCODE_TEST_ENV_VAR_SET"
	t.Cleanup(func() { os.Unsetenv(name) })
	AhdEnvSet(AhdClassEnvError, name, "value")
	if got := AhdEnvGet(name); got == nil || *got != "value" {
		t.Fatalf("Get after Set = %v", got)
	}
	AhdEnvUnset(AhdClassEnvError, name)
	if AhdEnvHas(name) {
		t.Fatal("Has after Unset = true")
	}
}

func TestEnvSetRejectsInvalidNames(t *testing.T) {
	expectRaise(t, AhdClassEnvError, func() { AhdEnvSet(AhdClassEnvError, "", "v") })
	expectRaise(t, AhdClassEnvError, func() { AhdEnvSet(AhdClassEnvError, "A=B", "v") })
	expectRaise(t, AhdClassEnvError, func() { AhdEnvSet(AhdClassEnvError, "A\x00B", "v") })
	expectRaise(t, AhdClassEnvError, func() { AhdEnvUnset(AhdClassEnvError, "") })
}

func TestEnvDotenvGrammar(t *testing.T) {
	content := "KEY1=value\n" +
		"KEY2=\"quoted value\"\n" +
		"KEY3='single quoted'\n" +
		"# a full-line comment\n" +
		"\n" +
		"EMPTY=\n" +
		"UNICODE=Ölçü\n" +
		"ESCAPED=\"line1\\nline2\\ttab\\\"quote\\\\backslash\"\n"
	entries := ahdEnvParseFile(AhdClassEnvError, content)
	want := map[string]string{
		"KEY1": "value", "KEY2": "quoted value", "KEY3": "single quoted",
		"EMPTY": "", "UNICODE": "Ölçü", "ESCAPED": "line1\nline2\ttab\"quote\\backslash",
	}
	if len(entries) != len(want) {
		t.Fatalf("entries = %v, want %d entries", entries, len(want))
	}
	for _, entry := range entries {
		if want[entry.Key] != entry.Value {
			t.Fatalf("%s = %q, want %q", entry.Key, entry.Value, want[entry.Key])
		}
	}
}

func TestEnvDotenvOrderPreserved(t *testing.T) {
	entries := ahdEnvParseFile(AhdClassEnvError, "Z=1\nA=2\nM=3\n")
	if len(entries) != 3 || entries[0].Key != "Z" || entries[1].Key != "A" || entries[2].Key != "M" {
		t.Fatalf("entries = %v, order not preserved", entries)
	}
}

func TestEnvDotenvMalformedInputRejected(t *testing.T) {
	malformed := []string{
		"NOEQUALS",
		"1KEY=value",
		"KE-Y=value",
		`KEY="unterminated`,
		`KEY='unterminated`,
		"KEY=\"bad\\xescape\"",
		"DUP=1\nDUP=2\n",
		`KEY="value" trailing`,
	}
	for _, content := range malformed {
		content := content
		expectRaise(t, AhdClassEnvError, func() { ahdEnvParseFile(AhdClassEnvError, content) })
	}
}

func TestEnvDotenvNoShellInterpolationOrCommandExecution(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "should_not_exist")
	content := "VALUE=$(touch " + marker + ")\nHOME_REF=${HOME}\nBACKTICK=`echo hi`\n"
	entries := ahdEnvParseFile(AhdClassEnvError, content)
	if len(entries) != 3 {
		t.Fatalf("entries = %v", entries)
	}
	if entries[0].Value != "$(touch "+marker+")" {
		t.Fatalf("VALUE was interpreted, not treated literally: %q", entries[0].Value)
	}
	if entries[1].Value != "${HOME}" {
		t.Fatalf("HOME_REF was interpolated: %q", entries[1].Value)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("command substitution was executed - file was created")
	}
}

func TestEnvReadDoesNotMutateProcessEnvironment(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	name := "AHDCODE_TEST_ENV_READ_ONLY"
	os.Unsetenv(name)
	if err := os.WriteFile(path, []byte(name+"=value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries := AhdEnvReadEntries(AhdClassEnvError, path)
	if len(entries) != 1 || entries[0].Key != name || entries[0].Value != "value" {
		t.Fatalf("ReadEntries = %v", entries)
	}
	if AhdEnvHas(name) {
		t.Fatal("Env.read mutated the process environment")
	}
	expectRaise(t, AhdClassEnvError, func() { AhdEnvReadEntries(AhdClassEnvError, filepath.Join(directory, "missing.env")) })
}

func TestEnvLoadOverridePrecedence(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	existingName := "AHDCODE_TEST_ENV_EXISTING"
	newName := "AHDCODE_TEST_ENV_NEW"
	t.Cleanup(func() { os.Unsetenv(existingName); os.Unsetenv(newName) })
	if err := os.Setenv(existingName, "process-value"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(existingName+"=dotenv-value\n"+newName+"=dotenv-only\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	AhdEnvLoad(AhdClassEnvError, path, false)
	if got := AhdEnvGet(existingName); got == nil || *got != "process-value" {
		t.Fatalf("override=false: existing = %v, want process-value to win", got)
	}
	if got := AhdEnvGet(newName); got == nil || *got != "dotenv-only" {
		t.Fatalf("override=false: new variable = %v, want it applied", got)
	}

	AhdEnvLoad(AhdClassEnvError, path, true)
	if got := AhdEnvGet(existingName); got == nil || *got != "dotenv-value" {
		t.Fatalf("override=true: existing = %v, want dotenv value to win", got)
	}
}

func TestEnvLoadValidatesWholeFileBeforeApplying(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	untouchedName := "AHDCODE_TEST_ENV_HALF_APPLY"
	os.Unsetenv(untouchedName)
	t.Cleanup(func() { os.Unsetenv(untouchedName) })
	if err := os.WriteFile(path, []byte(untouchedName+"=value\nBADLINE\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectRaise(t, AhdClassEnvError, func() { AhdEnvLoad(AhdClassEnvError, path, false) })
	if AhdEnvHas(untouchedName) {
		t.Fatal("Env.load applied an earlier entry before failing on a later malformed line")
	}
}

func TestEnvErrorMessagesDoNotLeakSecretValues(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	secret := "super-secret-value-should-not-appear"
	if err := os.WriteFile(path, []byte("A="+secret+"\nA="+secret+"2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	defer func() {
		recovered := recover()
		signal, ok := recovered.(*AhdSignal)
		if !ok {
			t.Fatalf("expected an EnvError, got %v", recovered)
		}
		if strings.Contains(signal.Message, secret) {
			t.Fatalf("error message leaked a secret value: %q", signal.Message)
		}
	}()
	AhdEnvReadEntries(AhdClassEnvError, path)
}
