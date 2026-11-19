package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateCLI(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	directory := filepath.Join("Göç Klasörü", "migrations")
	database := filepath.Join("Göç Klasörü", "uygulama veri.db")

	var stdout, stderr bytes.Buffer
	if code := runMigrate([]string{"new", "create_notes", "--dir", directory}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %d %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "0001_create_notes.up.sql") {
		t.Fatalf("new output: %s", stdout.String())
	}
	up := filepath.Join(directory, "0001_create_notes.up.sql")
	if err := os.WriteFile(up, []byte("CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL);\nINSERT INTO notes (id, body) VALUES (1, 'kalıcı');\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	down := filepath.Join(directory, "0001_create_notes.down.sql")
	if err := os.WriteFile(down, []byte("DROP TABLE notes;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runMigrate([]string{"up", "--db", database, "--dir", directory}, &stdout, &stderr); code != 0 {
		t.Fatalf("up: %d %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Applied 0001_create_notes") {
		t.Fatalf("up output: %s", stdout.String())
	}
	stdout.Reset()
	if code := runMigrate([]string{"up", "--db", database, "--dir", directory}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "No pending migrations.") {
		t.Fatalf("second up: %d %s %s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runMigrate([]string{"status", "--db", database, "--dir", directory}, &stdout, &stderr); code != 0 {
		t.Fatalf("status: %d %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "applied\t0001_create_notes") || !strings.Contains(stdout.String(), "No pending migrations.") {
		t.Fatalf("status output: %s", stdout.String())
	}
	if err := os.WriteFile(up, []byte("CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL);\n-- changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if code := runMigrate([]string{"up", "--db", database, "--dir", directory}, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), "changed after it was applied") {
		t.Fatalf("checksum: %d %s", code, stderr.String())
	}
	if code := runMigrate(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("usage code %d", code)
	}
	if code := runMigrate([]string{"up", "--driver", "mysql", "--db", database, "--dir", directory}, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), "MySQL") {
		t.Fatalf("mysql code %d %s", code, stderr.String())
	}
	if code := runMigrate([]string{"down", "--steps", "0", "--db", database, "--dir", directory}, &stdout, &stderr); code != 2 {
		t.Fatalf("steps code %d", code)
	}
}

func TestMigrateCLIFlagsAndHelp(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	var stdout, stderr bytes.Buffer
	for _, args := range [][]string{{"--help"}, {"-h"}, {"new", "--help"}, {"status", "-h"}, {"up", "--help"}, {"down", "-h"}} {
		stdout.Reset()
		stderr.Reset()
		if code := runMigrate(args, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "ahdcode migrate new") {
			t.Fatalf("%v code %d\n%s%s", args, code, stdout.String(), stderr.String())
		}
	}
	if code := runMigrate(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("missing command %d", code)
	}
	cases := [][]string{
		{"new", "create_notes", "--db", "app.db"},
		{"new", "create_notes", "--driver", "sqlite"},
		{"status", "--db", "app.db", "--steps", "1"},
		{"up", "--db", "app.db", "--driver", "sqlite", "--driver", "sqlite"},
		{"up", "--db", "one.db", "--db", "two.db"},
		{"down", "--db", "app.db", "--steps", "1", "--steps", "2"},
		{"status", "--db"},
		{"up", "--driver", "--db", "app.db"},
		{"new", "--verbose"},
	}
	for _, args := range cases {
		stdout.Reset()
		stderr.Reset()
		if code := runMigrate(args, &stdout, &stderr); code != 2 {
			t.Fatalf("%v code %d %s", args, code, stderr.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, "migrations")); err == nil {
		t.Fatal("a rejected new command created a migration directory")
	}
	directory := "migrations"
	database := "notes.db"
	if code := runMigrate([]string{"new", "create_notes", "--dir", directory}, &stdout, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	if err := os.WriteFile(filepath.Join(directory, "0001_create_notes.up.sql"), []byte("CREATE TABLE notes (id INTEGER PRIMARY KEY);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "0002_broken.up.sql"), []byte("SELECT * FROM missing_table;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runMigrate([]string{"up", "--db", database, "--dir", directory}, &stdout, &stderr); code != 1 {
		t.Fatalf("partial up code %d\n%s%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Applied 0001_create_notes") || !strings.Contains(stderr.String(), "failed") {
		t.Fatalf("partial output\n%s%s", stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	missing := "missing.db"
	if code := runMigrate([]string{"status", "--db", missing, "--dir", directory}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "does not exist") || !strings.Contains(stdout.String(), "pending") {
		t.Fatalf("missing status %d\n%s%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(missing); err == nil {
		t.Fatal("status created a missing database")
	}
}

func TestMigrateCLIPartialDown(t *testing.T) {
	root := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	directory := "migrations"
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"0001_create_notes.up.sql":   "CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT);\n",
		"0001_create_notes.down.sql": "SELECT * FROM missing_table;\n",
		"0002_add_author.up.sql":     "ALTER TABLE notes ADD COLUMN author TEXT NOT NULL DEFAULT 'Ali';\n",
		"0002_add_author.down.sql":   "ALTER TABLE notes DROP COLUMN author;\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	if code := runMigrate([]string{"up", "--db", "notes.db", "--dir", directory}, &stdout, &stderr); code != 0 {
		t.Fatalf("up %d %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := runMigrate([]string{"down", "--db", "notes.db", "--dir", directory, "--steps", "2"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("partial down code %d\n%s%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Rolled back 0002_add_author") || !strings.Contains(stderr.String(), "1 migration") {
		t.Fatalf("partial down output\n%s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "missing_table") && !strings.Contains(stderr.String(), "failed") {
		t.Fatalf("partial down error\n%s", stderr.String())
	}
}
