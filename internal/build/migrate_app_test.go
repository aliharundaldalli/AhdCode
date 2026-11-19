package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/migrate"
)

func TestMigrateExampleAppKeepsItsData(t *testing.T) {
	sqliteHelperForTest(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	sourceMigrations := filepath.Join(root, "examples", "v2.10", "migrations", "migrations")
	migrations := filepath.Join(directory, "migrations")
	if err := os.MkdirAll(migrations, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"0001_create_notes.up.sql", "0001_create_notes.down.sql"} {
		data, err := os.ReadFile(filepath.Join(sourceMigrations, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(migrations, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	database := filepath.Join(directory, "notes.db")
	first, err := migrate.Up(migrate.Options{Database: database, Directory: migrations})
	if err != nil || len(first) != 1 {
		t.Fatalf("first %+v %v", first, err)
	}
	firstApp := []byte("bring SQLite\nfrom SQLite bring Database\nfrom SQLite bring SQLiteValue\n\ndb: Database := SQLite.open(\"notes.db\")\nrows: List<Pair<String, SQLiteValue>> := db.query(\"SELECT id, body FROM notes ORDER BY id\")\nfor row in rows {\n    write(\"{row[\"id\"].int()} {row[\"body\"].string()}\")\n}\ndb.close()\n")
	if err := os.WriteFile(filepath.Join(directory, "first.ahd"), firstApp, 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := buildAndRunIn(t, filepath.Join(directory, "first.ahd"), directory)
	if code != 0 || !strings.Contains(stdout, "1 ilk not") {
		t.Fatalf("after first migration exit %d\n%s\n%s", code, stderr, stdout)
	}
	for _, name := range []string{"0002_add_author.up.sql", "0002_add_author.down.sql"} {
		data, err := os.ReadFile(filepath.Join(sourceMigrations, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(migrations, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	second, err := migrate.Up(migrate.Options{Database: database, Directory: migrations})
	if err != nil || len(second) != 1 {
		t.Fatalf("second %+v %v", second, err)
	}
	app, err := os.ReadFile(filepath.Join(root, "examples", "v2.10", "migrations", "app.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "app.ahd"), app, 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code = buildAndRunIn(t, filepath.Join(directory, "app.ahd"), directory)
	if code != 0 || !strings.Contains(stdout, "1 ilk not Ali") || !strings.Contains(stdout, "total 2") {
		t.Fatalf("after second migration exit %d\n%s\n%s", code, stderr, stdout)
	}
	again, err := migrate.Up(migrate.Options{Database: database, Directory: migrations})
	if err != nil || len(again) != 0 {
		t.Fatalf("re-run %+v %v", again, err)
	}
	stdout, stderr, code = buildAndRunIn(t, filepath.Join(directory, "app.ahd"), directory)
	if code != 0 || !strings.Contains(stdout, "1 ilk not Ali") || !strings.Contains(stdout, "total 3") {
		t.Fatalf("after a repeated migration exit %d\n%s\n%s", code, stderr, stdout)
	}
}
