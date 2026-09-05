package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/initweb"
	"ahdcode/internal/localdev"
)

func TestStudioHostsMapsLoopback(t *testing.T) {
	for _, tc := range []struct {
		name, hosts string
		want        bool
	}{
		{"missing", "127.0.0.1 localhost\n", false},
		{"comment", "# 127.0.0.1 ahddatabasestudio.test\n", false},
		{"aliases", "127.0.0.1\tlocalhost AHDDATABASESTUDIO.TEST # local\n", true},
		{"ipv6 only", "::1 ahddatabasestudio.test\n", false},
		{"remote", "192.0.2.1 ahddatabasestudio.test\n", false},
		{"conflict", "127.0.0.1 ahddatabasestudio.test\n::1 ahddatabasestudio.test\n", false},
		{"suffix", "127.0.0.1 ahddatabasestudio.test.invalid\n", false},
		{"absolute name", "127.0.0.1 ahddatabasestudio.test.\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := localdev.HostsMapsLoopback(tc.hosts, localdev.StudioHost); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestStudioEnvBootstrapPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("AHD_DATA_LANG=en\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureStudioEnv(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode %v", info.Mode())
	}
	existing := []byte("AHD_DATA_LANG=tr\n")
	if err := os.WriteFile(path, existing, 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureStudioEnv(dir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(existing) {
		t.Fatalf("existing env changed: %v", err)
	}
}

// The management subcommands are a registry, not a database administration
// CLI: they record and forget paths, and never touch the file itself.
func TestDatabasesAddListRemove(t *testing.T) {
	t.Setenv(localdev.HomeEnvKey, t.TempDir())
	project := t.TempDir()
	database := filepath.Join(project, "foo.db")
	original := []byte("rows that must survive")
	if err := os.WriteFile(database, original, 0o600); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if code := runDatabases([]string{"add", database}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("add returned %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Registered:") {
		t.Errorf("add did not confirm registration:\n%s", out.String())
	}

	// Adding it again is a no-op rather than a duplicate or an error.
	out.Reset()
	if code := runDatabases([]string{"add", database}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("re-add returned %d", code)
	}
	if !strings.Contains(out.String(), "Already registered") {
		t.Errorf("re-adding did not report the existing entry:\n%s", out.String())
	}

	out.Reset()
	if code := runDatabases([]string{"list"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("list returned %d", code)
	}
	listed := out.String()
	if strings.Count(listed, "foo.db") != 1 {
		t.Errorf("list did not show exactly one entry:\n%s", listed)
	}
	if !strings.Contains(listed, "sqlite\t") || !strings.Contains(listed, "available") {
		t.Errorf("list was not in the expected tab-separated shape:\n%s", listed)
	}

	out.Reset()
	if code := runDatabases([]string{"remove", database}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("remove returned %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "was not changed") {
		t.Errorf("remove did not state that the file was kept:\n%s", out.String())
	}
	content, err := os.ReadFile(database)
	if err != nil || string(content) != string(original) {
		t.Fatalf("remove disturbed the database file: %v %q", err, content)
	}

	out.Reset()
	if code := runDatabases([]string{"list"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("list returned %d", code)
	}
	if strings.Contains(out.String(), "foo.db") {
		t.Errorf("the removed entry is still listed:\n%s", out.String())
	}
}

// Bad usage is refused with a usage exit code, and never starts Studio.
func TestDatabasesRejectsBadUsage(t *testing.T) {
	t.Setenv(localdev.HomeEnvKey, t.TempDir())
	for _, arguments := range [][]string{
		{"list", "extra"},
		{"add"},
		{"add", "a.db", "b.db"},
		{"remove"},
		{"nonsense"},
	} {
		var out, errOut bytes.Buffer
		if code := runDatabases(arguments, strings.NewReader(""), &out, &errOut); code != 2 {
			t.Errorf("%v returned %d, expected 2", arguments, code)
		}
	}
	var out, errOut bytes.Buffer
	if code := runDatabases([]string{"remove", filepath.Join(t.TempDir(), "never.db")}, strings.NewReader(""), &out, &errOut); code != 1 {
		t.Errorf("removing an unregistered path returned %d, expected 1", code)
	}
	if code := runDatabases([]string{"add", filepath.Join(t.TempDir(), "missing.db")}, strings.NewReader(""), &out, &errOut); code != 1 {
		t.Errorf("adding a missing file returned %d, expected 1", code)
	}
}

// The canonical Studio URL is the clean name; the direct loopback URL stays
// supported and is what is used when the clean name is not resolvable here.
func TestStudioURLsKeepDirectCompatibility(t *testing.T) {
	if initweb.AhdDataStudioPublicURL() != "http://ahddatabasestudio.test/" {
		t.Errorf("canonical Studio URL was %q", initweb.AhdDataStudioPublicURL())
	}
	if initweb.AhdDataStudioLoopbackURL() != "http://127.0.0.1:8081/AhdDataStudio" {
		t.Errorf("direct Studio URL was %q", initweb.AhdDataStudioLoopbackURL())
	}
	if got := studioOpenURL(0); got != initweb.AhdDataStudioLoopbackURL() {
		t.Errorf("with no router, the opened URL was %q", got)
	}
}
