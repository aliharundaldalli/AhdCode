package ahdruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unsetEnvironment removes names for the duration of a test and restores them.
func unsetEnvironment(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		previous, present := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if present {
				os.Setenv(name, previous)
			} else {
				os.Unsetenv(name)
			}
		})
	}
}

func writeSecret(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEnvSecretPrecedence(t *testing.T) {
	const name = "AHDCODE_RUNTIME_SECRET"
	unsetEnvironment(t, name, name+"_FILE")
	if value := AhdEnvSecret(AhdClassEnvError, name); value != nil {
		t.Fatalf("neither variable set, got %q", *value)
	}

	t.Setenv(name, "")
	if value := AhdEnvSecret(AhdClassEnvError, name); value == nil || *value != "" {
		t.Fatalf("a present empty NAME is its value, got %v", value)
	}
	t.Setenv(name, "hunter2\n")
	if value := AhdEnvSecret(AhdClassEnvError, name); value == nil || *value != "hunter2\n" {
		t.Fatalf("NAME is returned unchanged, got %v", value)
	}

	path := writeSecret(t, []byte("from-file"))
	t.Setenv(name+"_FILE", path)
	message := codesRaised(t, func() { AhdEnvSecret(AhdClassEnvError, name) })
	if message != name+" and "+name+"_FILE are both set; set only one" {
		t.Fatalf("both set message %q", message)
	}
	if strings.Contains(message, "hunter2") || strings.Contains(message, path) {
		t.Fatalf("message leaks the value or path: %q", message)
	}

	unsetEnvironment(t, name)
	if value := AhdEnvSecret(AhdClassEnvError, name); value == nil || *value != "from-file" {
		t.Fatalf("NAME_FILE contents, got %v", value)
	}

	t.Setenv(name+"_FILE", "")
	if message := codesRaised(t, func() { AhdEnvSecret(AhdClassEnvError, name) }); message != name+"_FILE is set but empty" {
		t.Fatalf("empty NAME_FILE message %q", message)
	}
}

func TestEnvSecretFileContents(t *testing.T) {
	const name = "AHDCODE_RUNTIME_SECRET_FILE_CONTENTS"
	unsetEnvironment(t, name, name+"_FILE")
	cases := map[string]string{
		"plain":              "plain",
		"trailing\n":         "trailing",
		"windows\r\n":        "windows",
		"two lines\n\n":      "two lines\n",
		" spaced  \n":        " spaced  ",
		"multi\nline\nvalue": "multi\nline\nvalue",
		"Şifre 😊\n":          "Şifre 😊",
		"\n":                 "",
		"":                   "",
	}
	for content, want := range cases {
		t.Setenv(name+"_FILE", writeSecret(t, []byte(content)))
		value := AhdEnvSecret(AhdClassEnvError, name)
		if value == nil || *value != want {
			t.Fatalf("file %q read as %v, want %q", content, value, want)
		}
	}

	exact := writeSecret(t, []byte(strings.Repeat("a", ahdEnvSecretMaximumBytes)))
	t.Setenv(name+"_FILE", exact)
	if value := AhdEnvSecret(AhdClassEnvError, name); value == nil || len(*value) != ahdEnvSecretMaximumBytes {
		t.Fatal("a file of exactly 1 MiB must be accepted")
	}

	directory := t.TempDir()
	link := filepath.Join(directory, "link")
	if err := os.Symlink(writeSecret(t, []byte("through-link\n")), link); err == nil {
		t.Setenv(name+"_FILE", link)
		if value := AhdEnvSecret(AhdClassEnvError, name); value == nil || *value != "through-link" {
			t.Fatalf("symlinked secret read as %v", value)
		}
	}

	failures := map[string]string{
		filepath.Join(directory, "missing"): "could not be read",
		directory:                           "could not be read",
		writeSecret(t, []byte(strings.Repeat("a", ahdEnvSecretMaximumBytes+1))): "is larger than 1 MiB",
		writeSecret(t, []byte{0xff, 0xfe}):                                      "is not valid UTF-8",
		writeSecret(t, []byte("before\x00after")):                               "contains a NUL byte",
	}
	for path, want := range failures {
		t.Setenv(name+"_FILE", path)
		message := codesRaised(t, func() { AhdEnvSecret(AhdClassEnvError, name) })
		if message != "the secret file named by "+name+"_FILE "+want {
			t.Fatalf("path %q message %q", path, message)
		}
		if strings.Contains(message, path) {
			t.Fatalf("message leaks the path: %q", message)
		}
	}
}

func TestEnvSecretValidatesTheName(t *testing.T) {
	for _, name := range []string{"", "BAD=NAME", "BAD\x00NAME"} {
		codesRaised(t, func() { AhdEnvSecret(AhdClassEnvError, name) })
	}
}
