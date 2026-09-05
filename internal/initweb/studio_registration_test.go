package initweb

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStudioExternalAdminRegistration(t *testing.T) {
	root := t.TempDir()
	studio := filepath.Join(root, "tools", "AhdDataStudio")
	if err := os.MkdirAll(studio, 0755); err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(studio, ".env")
	initial := "# keep settings\nAHD_DATA_LANG=tr\nAHD_DATA_SQLITE_PATHS=/existing.db\n"
	if err := os.WriteFile(env, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	t.Chdir(project)
	t.Setenv("AHDCODE_ROOT", root)
	var out bytes.Buffer
	err := Web(project, &out, &out, Options{Starter: StarterAdmin, AppName: "External QA", DatabaseName: "app", Database: DriverSQLite, AdminName: "QA", AdminEmail: "qa@example.com", AdminPassword: "qa-only-password"})
	if err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(project, "database", "app.db")
	if _, err := os.Stat(db); err != nil {
		t.Fatal(err)
	}
	registered, err := os.ReadFile(env)
	if err != nil || !strings.Contains(string(registered), db) {
		t.Fatalf("Admin did not register its database: %v", err)
	}
	if !registerSQLiteWithStudio(db) {
		t.Fatal("repeat registration failed")
	}
	got, err := os.ReadFile(env)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "AHD_DATA_SQLITE_PATHS=/existing.db,"+db) || strings.Count(string(got), db) != 1 || !strings.Contains(string(got), "# keep settings\nAHD_DATA_LANG=tr\n") {
		t.Fatalf("unexpected env: %s", got)
	}
}

func TestStudioRegistrationDiscoveryAndSafety(t *testing.T) {
	root := t.TempDir()
	local := t.TempDir()
	t.Chdir(local)
	t.Setenv("AHDCODE_ROOT", root)
	for _, dir := range []string{root, local} {
		if err := os.MkdirAll(filepath.Join(dir, "tools", "AhdDataStudio"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	explicit := filepath.Join(root, "tools", "AhdDataStudio", ".env")
	nearby := filepath.Join(local, "tools", "AhdDataStudio", ".env")
	db := filepath.Join(local, "app.db")
	original := []byte("untouched database")
	if err := os.WriteFile(db, original, 0600); err != nil {
		t.Fatal(err)
	}
	if registerSQLiteWithStudio(db) {
		t.Fatal("registered without env")
	}
	if _, err := os.Stat(explicit); !os.IsNotExist(err) {
		t.Fatal("created env")
	}
	for _, path := range []string{explicit, nearby} {
		if err := os.WriteFile(path, []byte("KEEP=yes\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if !registerSQLiteWithStudio(db) {
		t.Fatal("registration failed")
	}
	got, _ := os.ReadFile(nearby)
	if string(got) != "KEEP=yes\n" {
		t.Fatal("mutated cwd env despite explicit root")
	}
	for _, path := range []string{"", "relative.db", "/tmp/bad,db", "/tmp/bad\nKEY=value", "/tmp/bad #comment", "/tmp/bad\"db"} {
		if registerSQLiteWithStudio(path) {
			t.Fatalf("accepted unsafe path %q", path)
		}
	}
	got, _ = os.ReadFile(db)
	if !bytes.Equal(got, original) {
		t.Fatal("database changed")
	}
}
