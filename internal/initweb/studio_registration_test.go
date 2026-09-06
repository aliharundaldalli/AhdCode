package initweb

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/localdev"
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

// The Admin starter's SQLite database is registered automatically, so nothing
// has to be added to AHD_DATA_SQLITE_PATHS or AHD_DATA_PROJECT_ROOT by hand
// before it shows up in AhdDataStudio.
func TestAdminSQLiteRegistersItselfInTheRegistry(t *testing.T) {
	t.Setenv(localdev.HomeEnvKey, t.TempDir())
	project := t.TempDir()
	t.Chdir(project)
	t.Setenv("AHDCODE_ROOT", "")
	_ = os.Unsetenv("AHDCODE_ROOT")
	t.Setenv("AHDCODE_STUDIO_CACHE", t.TempDir())

	var out bytes.Buffer
	err := Web(project, &out, &out, Options{
		Starter: StarterAdmin, AppName: "Registry QA", DatabaseName: "app",
		Database: DriverSQLite, AdminName: "QA", AdminEmail: "qa@example.com",
		AdminPassword: "qa-only-password",
	})
	if err != nil {
		t.Fatal(err)
	}

	database := filepath.Join(project, "database", "app.db")
	canonical, err := localdev.CanonicalDatabasePath(database)
	if err != nil {
		t.Fatal(err)
	}
	databases, err := localdev.LoadDatabases()
	if err != nil {
		t.Fatal(err)
	}
	if len(databases) != 1 {
		t.Fatalf("expected exactly the database init created, found %#v", databases)
	}
	if databases[0].Path != canonical || databases[0].Driver != localdev.DriverSQLite {
		t.Fatalf("unexpected registry entry %#v", databases[0])
	}
	if !strings.Contains(out.String(), "Registered for AhdDataStudio") {
		t.Errorf("init did not report the registration:\n%s", out.String())
	}
	if strings.Contains(out.String(), "Add it to AhdDataStudio as AHD_DATA_SQLITE_PATHS") {
		t.Errorf("init still told the user to edit an environment variable:\n%s", out.String())
	}
}

// Registration is limited to the one database init created. A project that
// happens to contain other SQLite files is not scanned.
func TestInitRegistersOnlyTheDatabaseItCreated(t *testing.T) {
	t.Setenv(localdev.HomeEnvKey, t.TempDir())
	project := t.TempDir()
	unrelated := filepath.Join(project, "someone-elses.db")
	if err := os.WriteFile(unrelated, []byte("not ours"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	t.Setenv("AHDCODE_ROOT", "")
	_ = os.Unsetenv("AHDCODE_ROOT")
	t.Setenv("AHDCODE_STUDIO_CACHE", t.TempDir())

	var out bytes.Buffer
	if err := Web(project, &out, &out, Options{
		Starter: StarterAdmin, AppName: "Scan QA", DatabaseName: "app",
		Database: DriverSQLite, AdminName: "QA", AdminEmail: "qa@example.com",
		AdminPassword: "qa-only-password",
	}); err != nil {
		t.Fatal(err)
	}
	databases, err := localdev.LoadDatabases()
	if err != nil {
		t.Fatal(err)
	}
	for _, database := range databases {
		if strings.Contains(database.Path, "someone-elses.db") {
			t.Fatalf("init registered a database it did not create: %#v", database)
		}
	}
	if len(databases) != 1 {
		t.Fatalf("expected one entry, found %#v", databases)
	}
}

// A registry that cannot be written must not cost the user their database or
// their generated application. The failure is reported and nothing is undone.
func TestRegistrationFailureKeepsTheDatabaseAndTheApp(t *testing.T) {
	home := t.TempDir()
	t.Setenv(localdev.HomeEnvKey, home)
	// A directory where the registry file's name is already taken by a
	// directory: the write fails, everything else must not.
	if err := os.MkdirAll(filepath.Join(home, "databases.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	t.Chdir(project)
	t.Setenv("AHDCODE_ROOT", "")
	_ = os.Unsetenv("AHDCODE_ROOT")
	t.Setenv("AHDCODE_STUDIO_CACHE", t.TempDir())

	var out bytes.Buffer
	if err := Web(project, &out, &out, Options{
		Starter: StarterAdmin, AppName: "Failure QA", DatabaseName: "app",
		Database: DriverSQLite, AdminName: "QA", AdminEmail: "qa@example.com",
		AdminPassword: "qa-only-password",
	}); err != nil {
		t.Fatalf("a registry failure failed the whole init: %v", err)
	}
	database := filepath.Join(project, "database", "app.db")
	if _, err := os.Stat(database); err != nil {
		t.Fatalf("the database was removed after a registry failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "app.ahd")); err != nil {
		t.Fatalf("the generated application was removed after a registry failure: %v", err)
	}
	printed := out.String()
	if !strings.Contains(printed, "could not be registered") {
		t.Errorf("the registration failure was not reported:\n%s", printed)
	}
	if !strings.Contains(printed, "ahdcode databases add") {
		t.Errorf("the failure did not say how to recover:\n%s", printed)
	}
}
