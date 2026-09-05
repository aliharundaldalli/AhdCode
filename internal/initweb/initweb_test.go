package initweb

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func emptyOpts() Options {
	return Options{Starter: StarterEmpty, AppName: "AhdCode Web"}
}

func TestWebFreshDirectory(t *testing.T) {
	root := t.TempDir()
	var out, errBuf bytes.Buffer
	if err := Web(root, &out, &errBuf, emptyOpts()); err != nil {
		t.Fatalf("Web: %v\nstderr=%s", err, errBuf.String())
	}
	want := []string{
		"app.ahd",
		".env",
		".env.example",
		".gitignore",
		"Config/App.ahd",
		"Components/Navbar.ahd",
		"Components/Footer.ahd",
		"Layouts/Main.ahd",
		"Pages/Home.ahd",
		"public/style.css",
		"public/main.js",
		"public/ahdcode-logo.png",
		"public/vendor/bootstrap/bootstrap.min.css",
		"public/vendor/bootstrap/bootstrap.bundle.min.js",
		"public/vendor/bootstrap/LICENSE",
	}
	for _, rel := range want {
		info, err := os.Lstat(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("%s is a symlink", rel)
		}
	}
	env, err := os.ReadFile(filepath.Join(root, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(env)
	for _, key := range []string{
		"APP_NAME=AhdCode Web",
		"APP_ENV=development",
		"APP_HOST=localhost",
		"APP_PROTOCOL=http",
		"SERVER_HOST=127.0.0.1",
		"SERVER_PORT=8080",
	} {
		if !strings.Contains(text, key) {
			t.Fatalf(".env missing %s:\n%s", key, text)
		}
	}
	if strings.Contains(text, "MAIL_") || strings.Contains(text, "DB_") {
		t.Fatalf("Empty .env has extra keys:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "password") || strings.Contains(text, "TOKEN") {
		t.Fatalf(".env looks secret-bearing:\n%s", text)
	}
	ignore, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []string{".env", "*.dev", "*.run"} {
		if !gitignoreHas(string(ignore), entry) {
			t.Fatalf(".gitignore missing %s:\n%s", entry, ignore)
		}
	}
	if gitignoreHas(string(ignore), "database/*.db") {
		t.Fatal("Empty starter should not ignore database files")
	}
	home, err := os.ReadFile(filepath.Join(root, "Pages/Home.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(home), "home: Function") {
		t.Fatalf("home handler not named home:\n%s", home)
	}
	if strings.Contains(string(home), "/login") || strings.Contains(string(home), "dashboard") {
		t.Fatal("Empty home mentions auth")
	}
	if _, err := os.Stat(filepath.Join(root, "database")); !os.IsNotExist(err) {
		t.Fatal("Empty starter created database/")
	}
	if _, err := os.Stat(filepath.Join(root, "Pages/Login.ahd")); !os.IsNotExist(err) {
		t.Fatal("Empty starter created Login")
	}
	if _, err := os.Stat(filepath.Join(root, "Config/Mail.ahd")); !os.IsNotExist(err) {
		t.Fatal("Empty starter created Mail config")
	}
	layout, err := os.ReadFile(filepath.Join(root, "Layouts/Main.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		`Web.UI.stylesheet("/assets/vendor/bootstrap/bootstrap.min.css")`,
		`Web.UI.stylesheet("/assets/style.css")`,
		`{"src": "/assets/vendor/bootstrap/bootstrap.bundle.min.js"}`,
		`Web.UI.element("script", {"src": "/assets/main.js"}, [])`,
	} {
		if !strings.Contains(string(layout), needle) {
			t.Fatalf("layout missing %s:\n%s", needle, layout)
		}
	}
	if strings.Contains(strings.ToLower(string(layout)), "cdn") {
		t.Fatal("layout mentions a CDN")
	}
	css, err := os.ReadFile(filepath.Join(root, "public/style.css"))
	if err != nil {
		t.Fatal(err)
	}
	if len(bytes.TrimSpace(css)) == 0 {
		t.Fatal("public/style.css should contain starter styles")
	}
	if !strings.Contains(out.String(), "Starter: Empty") || !strings.Contains(out.String(), "ahdcode dev app.ahd") {
		t.Fatalf("success output missing:\n%s", out.String())
	}
}

func TestWizardEmptySkipsDatabase(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	err := Web(root, &out, ioDiscard{}, Options{
		Input:  strings.NewReader("1\nMy Portal\n"),
		Output: &out,
		IsTTY:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Database:") || strings.Contains(out.String(), "SQLite") {
		t.Fatalf("Empty wizard asked about the database:\n%s", out.String())
	}
	env, _ := os.ReadFile(filepath.Join(root, ".env"))
	if !strings.Contains(string(env), "APP_NAME=My Portal") {
		t.Fatalf("app name not preserved:\n%s", env)
	}
	if strings.Contains(string(env), "DB_") {
		t.Fatal("Empty .env has DB keys")
	}
}

func TestWizardBasicSkipsAuthAndDatabase(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	err := Web(root, &out, ioDiscard{}, Options{
		Input:  strings.NewReader("2\nDemo\n"),
		Output: &out,
		IsTTY:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Database:") || strings.Contains(out.String(), "Admin") && strings.Contains(out.String(), "password") {
		t.Fatalf("Basic wizard asked auth/database questions:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "Config/Mail.ahd")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "database")); !os.IsNotExist(err) {
		t.Fatal("Basic created database/")
	}
	if _, err := os.Stat(filepath.Join(root, "Pages/Login.ahd")); !os.IsNotExist(err) {
		t.Fatal("Basic created Login")
	}
	env, _ := os.ReadFile(filepath.Join(root, ".env"))
	text := string(env)
	for _, key := range []string{"MAIL_HOST=", "MAIL_PORT=587", "MAIL_SECURITY=starttls"} {
		if !strings.Contains(text, key) {
			t.Fatalf("Basic .env missing %s:\n%s", key, text)
		}
	}
	if strings.Contains(text, "DB_") {
		t.Fatal("Basic .env has DB keys")
	}
	if !strings.Contains(out.String(), "Starter: Basic") {
		t.Fatalf("stdout=%s", out.String())
	}
}

func TestAdminSQLiteBootstrap(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	password := "qa-admin-pass"
	err := Web(root, &out, ioDiscard{}, Options{
		Starter:       StarterAdmin,
		AppName:       "QA Portal",
		Database:      DriverSQLite,
		DatabaseName:  "qa_portal",
		AdminName:     "Ada",
		AdminEmail:    "ada@example.com",
		AdminPassword: password,
	})
	if err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(root, "database", "qa_portal.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("sqlite file: %v", err)
	}
	schema, err := os.ReadFile(filepath.Join(root, "database/schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(schema), "CREATE TABLE users") {
		t.Fatalf("schema:\n%s", schema)
	}
	hash, err := inspectSQLiteAdminHash(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || hash == password {
		t.Fatal("stored password was plaintext or empty")
	}
	ok, err := verifyPassword(password, hash)
	if err != nil || !ok {
		t.Fatalf("hash did not verify: %v", err)
	}
	env, _ := os.ReadFile(filepath.Join(root, ".env"))
	if !strings.Contains(string(env), "DB_DRIVER=sqlite") || !strings.Contains(string(env), "DB_DATABASE=database/qa_portal.db") {
		t.Fatalf(".env:\n%s", env)
	}
	if strings.Contains(string(env), password) {
		t.Fatal(".env contains the admin password")
	}
	example, _ := os.ReadFile(filepath.Join(root, ".env.example"))
	if strings.Contains(string(example), password) {
		t.Fatal(".env.example contains a secret")
	}
	ignore, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if !gitignoreHas(string(ignore), "database/*.db") {
		t.Fatal("Admin gitignore missing database/*.db")
	}
	if !strings.Contains(out.String(), "Starter: Admin") || !strings.Contains(out.String(), "ada@example.com") {
		t.Fatalf("stdout=%s", out.String())
	}
	if strings.Contains(out.String(), password) {
		t.Fatal("success output echoed the password")
	}
	for _, rel := range []string{"Pages/Login.ahd", "Pages/Dashboard.ahd", "Services/Auth.ahd", "Repositories/Users.ahd"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("missing %s", rel)
		}
	}
}

func TestAdminConflictWritesNothingAndSkipsDatabase(t *testing.T) {
	root := t.TempDir()
	sentinel := []byte("keep-me\n")
	if err := os.WriteFile(filepath.Join(root, "app.ahd"), sentinel, 0o644); err != nil {
		t.Fatal(err)
	}
	err := Web(root, ioDiscard{}, ioDiscard{}, Options{
		Starter:       StarterAdmin,
		AppName:       "Blocked",
		Database:      DriverSQLite,
		DatabaseName:  "blocked",
		AdminName:     "Ada",
		AdminEmail:    "ada@example.com",
		AdminPassword: "qa-admin-pass",
	})
	if err == nil {
		t.Fatal("expected conflict")
	}
	if !strings.Contains(err.Error(), "app.ahd already exists") {
		t.Fatalf("error = %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "app.ahd"))
	if string(got) != string(sentinel) {
		t.Fatalf("sentinel changed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "database")); !os.IsNotExist(err) {
		t.Fatal("database/ created after local conflict")
	}
}

func TestExistingSQLiteFileIsLeftAlone(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "database"), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(root, "database", "kept.db")
	if err := os.WriteFile(existing, []byte("sqlite-sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Web(root, ioDiscard{}, ioDiscard{}, Options{
		Starter:       StarterAdmin,
		AppName:       "Kept",
		Database:      DriverSQLite,
		DatabaseName:  "kept",
		AdminName:     "Ada",
		AdminEmail:    "ada@example.com",
		AdminPassword: "qa-admin-pass",
	})
	if err == nil {
		t.Fatal("expected existing db conflict")
	}
	got, _ := os.ReadFile(existing)
	if string(got) != "sqlite-sentinel" {
		t.Fatalf("existing db changed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "app.ahd")); !os.IsNotExist(err) {
		t.Fatal("partial scaffold after existing db")
	}
}

func TestWebConflictWritesNothing(t *testing.T) {
	root := t.TempDir()
	sentinel := []byte("keep-me\n")
	if err := os.WriteFile(filepath.Join(root, "app.ahd"), sentinel, 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errBuf bytes.Buffer
	err := Web(root, &out, &errBuf, emptyOpts())
	if err == nil {
		t.Fatal("expected conflict")
	}
	if !strings.Contains(err.Error(), "app.ahd already exists") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "No files were written.") {
		t.Fatalf("error missing no-write note: %v", err)
	}
	got, readErr := os.ReadFile(filepath.Join(root, "app.ahd"))
	if readErr != nil || string(got) != string(sentinel) {
		t.Fatalf("sentinel changed: %q %v", got, readErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, "Config")); !os.IsNotExist(statErr) {
		t.Fatalf("Config/ was created on conflict")
	}
	if _, statErr := os.Stat(filepath.Join(root, ".env")); !os.IsNotExist(statErr) {
		t.Fatalf(".env was created on conflict")
	}
}

func TestWebDirectoryPathConflict(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Pages"), []byte("not-a-dir\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Web(root, ioDiscard{}, ioDiscard{}, emptyOpts())
	if err == nil {
		t.Fatal("expected directory conflict")
	}
	if !strings.Contains(err.Error(), "Pages is a file") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "app.ahd")); !os.IsNotExist(statErr) {
		t.Fatalf("partial scaffold after directory conflict")
	}
}

func TestWebGitignoreMerge(t *testing.T) {
	root := t.TempDir()
	existing := "# keep\nbuild/\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Web(root, ioDiscard{}, ioDiscard{}, emptyOpts()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "# keep") || !strings.Contains(text, "build/") {
		t.Fatalf("lost existing rules:\n%s", text)
	}
	for _, entry := range []string{".env", "*.dev", "*.run"} {
		if !gitignoreHas(text, entry) {
			t.Fatalf("missing %s after merge:\n%s", entry, text)
		}
	}
}

func TestWebSymlinkConflict(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "elsewhere.ahd")
	if err := os.WriteFile(target, []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "app.ahd")); err != nil {
		t.Fatal(err)
	}
	err := Web(root, ioDiscard{}, ioDiscard{}, emptyOpts())
	if err == nil {
		t.Fatal("expected symlink conflict")
	}
	if !strings.Contains(err.Error(), "app.ahd is a symlink") {
		t.Fatalf("error = %v", err)
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil || string(got) != "secret\n" {
		t.Fatalf("symlink destination changed: %q %v", got, readErr)
	}
}

func TestWebSecondInitDoesNotOverwrite(t *testing.T) {
	root := t.TempDir()
	if err := Web(root, ioDiscard{}, ioDiscard{}, emptyOpts()); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "Pages/Home.ahd")
	original, err := os.ReadFile(home)
	if err != nil {
		t.Fatal(err)
	}
	edited := append([]byte("// edited\n"), original...)
	if err := os.WriteFile(home, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	err = Web(root, ioDiscard{}, ioDiscard{}, emptyOpts())
	if err == nil {
		t.Fatal("second init should fail")
	}
	got, readErr := os.ReadFile(home)
	if readErr != nil || string(got) != string(edited) {
		t.Fatalf("user edit overwritten")
	}
}

func TestNonTTYBareInitFails(t *testing.T) {
	err := Web(t.TempDir(), ioDiscard{}, ioDiscard{}, Options{Input: strings.NewReader(""), IsTTY: false})
	if err == nil {
		t.Fatal("expected non-tty failure")
	}
	if !strings.Contains(err.Error(), "empty|basic|admin") {
		t.Fatalf("error = %v", err)
	}
}

func TestNamesAndIdentifiers(t *testing.T) {
	if got := defaultDatabaseName("My Portal"); got != "my_portal" {
		t.Fatalf("slug = %q", got)
	}
	if err := validateDatabaseName("my_portal"); err != nil {
		t.Fatal(err)
	}
	if err := validateDatabaseName("1bad"); err == nil {
		t.Fatal("expected leading digit rejection")
	}
	if err := validateDatabaseName("bad-name"); err == nil {
		t.Fatal("expected hyphen rejection")
	}
	if err := validateEmail("ada@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := validateEmail("not-an-email"); err == nil {
		t.Fatal("expected email rejection")
	}
	if err := validateAdminPassword("short", "short"); err == nil {
		t.Fatal("expected short password rejection")
	}
	if err := validateAdminPassword("long-enough", "different"); err == nil || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("mismatch = %v", err)
	}
}

func TestMySQLSequenceExistingRefused(t *testing.T) {
	fake := &scriptedMySQL{exists: true}
	created, err := bootstrapMySQL(Options{
		Starter:       StarterAdmin,
		Database:      DriverMySQL,
		DatabaseName:  "my_portal",
		MySQLHost:     "127.0.0.1",
		MySQLPort:     3306,
		MySQLUser:     "app",
		MySQLPassword: "unused",
		MySQLSecurity: "tls",
		AdminName:     "Ada",
		AdminEmail:    "ada@example.com",
		AdminPassword: "qa-admin-pass",
	}, fake)
	if err == nil || created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v", err)
	}
	if containsCall(fake.calls, "CreateDatabase") {
		t.Fatalf("created an existing database: %v", fake.calls)
	}
}

func TestMySQLSequenceCreateSchemaInsert(t *testing.T) {
	fake := &scriptedMySQL{}
	created, err := bootstrapMySQL(Options{
		Starter:       StarterAdmin,
		Database:      DriverMySQL,
		DatabaseName:  "my_portal",
		MySQLHost:     "127.0.0.1",
		MySQLPort:     3306,
		MySQLUser:     "app",
		MySQLPassword: "unused",
		MySQLSecurity: "tls",
		AdminName:     "Ada",
		AdminEmail:    "ada@example.com",
		AdminPassword: "qa-admin-pass",
	}, fake)
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	want := []string{"Connect", "DatabaseExists", "CreateDatabase", "UseDatabase", "Exec", "Exec"}
	if strings.Join(fake.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %v", fake.calls)
	}
	if !strings.Contains(fake.sql[0], "CREATE TABLE users") {
		t.Fatalf("schema sql = %q", fake.sql[0])
	}
	if !strings.Contains(fake.sql[1], "INSERT INTO users") {
		t.Fatalf("insert sql = %q", fake.sql[1])
	}
}

func TestMySQLCreateDenied(t *testing.T) {
	fake := &scriptedMySQL{createErr: fmtCreateDenied()}
	_, err := bootstrapMySQL(Options{
		Starter:       StarterAdmin,
		Database:      DriverMySQL,
		DatabaseName:  "my_portal",
		MySQLHost:     "127.0.0.1",
		MySQLPort:     3306,
		MySQLUser:     "app",
		MySQLSecurity: "tls",
		AdminName:     "Ada",
		AdminEmail:    "ada@example.com",
		AdminPassword: "qa-admin-pass",
	}, fake)
	if err == nil || !strings.Contains(err.Error(), "CREATE DATABASE permission") {
		t.Fatalf("error = %v", err)
	}
}

func fmtCreateDenied() error {
	return errCreateDenied
}

var errCreateDenied = errString("Unable to create database \"my_portal\".\nThe MySQL account does not have CREATE DATABASE permission.")

type errString string

func (e errString) Error() string { return string(e) }

type scriptedMySQL struct {
	exists    bool
	createErr error
	calls     []string
	sql       []string
}

func (s *scriptedMySQL) Connect(host, username, password string, port int, security string) error {
	s.calls = append(s.calls, "Connect")
	return nil
}
func (s *scriptedMySQL) DatabaseExists(name string) (bool, error) {
	s.calls = append(s.calls, "DatabaseExists")
	return s.exists, nil
}
func (s *scriptedMySQL) CreateDatabase(name string) error {
	s.calls = append(s.calls, "CreateDatabase")
	return s.createErr
}
func (s *scriptedMySQL) UseDatabase(name string) error {
	s.calls = append(s.calls, "UseDatabase")
	return nil
}
func (s *scriptedMySQL) Exec(sqlText string, parameters []string) error {
	s.calls = append(s.calls, "Exec")
	s.sql = append(s.sql, sqlText)
	return nil
}
func (s *scriptedMySQL) Close() {}

func containsCall(calls []string, name string) bool {
	for _, call := range calls {
		if call == name {
			return true
		}
	}
	return false
}

func gitignoreHas(text, entry string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == entry {
			return true
		}
	}
	return false
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
