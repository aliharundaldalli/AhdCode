package ahdruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResourceRootFollowsTheApplicationNotTheHomeDirectory(t *testing.T) {
	executable := filepath.Join("/Applications", "Ledger.app", "Contents", "MacOS", "Ledger")
	mac, problem := ahdResourceRootFrom(executable, true, "darwin")
	wantMac := filepath.Clean(filepath.Join("/Applications", "Ledger.app", "Contents", "Resources", "app"))
	if problem != "" || mac != wantMac {
		t.Fatalf("macOS %q %s", mac, problem)
	}
	program := filepath.Join("/opt", "ledger", "Ledger")
	for _, goos := range []string{"windows", "linux"} {
		got, problem := ahdResourceRootFrom(program, true, goos)
		want := filepath.Join("/opt", "ledger", "resources")
		if problem != "" || got != want {
			t.Fatalf("%s %q %s", goos, got, problem)
		}
	}
	_, problem = ahdResourceRootFrom(filepath.Join(t.TempDir(), "program"), false, runtime.GOOS)
	if problem == "" {
		t.Fatal("a development program without a resources folder was accepted")
	}
}

func TestResourcePathIsExplicit(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "icons"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "icons", "mark.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_RESOURCE_ROOT", root)
	got, problem := AhdGUIResource("icons/mark.png")
	want := canonicalResource(t, filepath.Join(root, "icons", "mark.png"))
	if problem != "" || got != want {
		t.Fatalf("resource %q %s", got, problem)
	}
	for _, name := range []string{"", "../mark.png", "/tmp/mark.png", `icons\..\mark.png`} {
		if _, problem := AhdGUIResource(name); problem == "" {
			t.Fatalf("accepted %q", name)
		}
	}
	if _, problem := AhdGUIResource("icons/missing.png"); problem == "" || !strings.Contains(problem, "does not exist") {
		t.Fatalf("missing: %s", problem)
	}
}

func TestResourceSymlinkStaysInsideTheCanonicalRoot(t *testing.T) {
	root := t.TempDir()
	icons := filepath.Join(root, "icons")
	if err := os.Mkdir(icons, 0o755); err != nil {
		t.Fatal(err)
	}
	mark := filepath.Join(icons, "mark.png")
	if err := os.WriteFile(mark, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("mark.png", filepath.Join(icons, "alias.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(mark, filepath.Join(icons, "absolute.png")); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "icons", "mark.png"), filepath.Join(nested, "up.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(icons, filepath.Join(root, "also")); err != nil {
		t.Fatal(err)
	}
	linkedRoot := filepath.Join(t.TempDir(), "resources")
	if err := os.Symlink(root, linkedRoot); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	real := filepath.Join(parent, "resources")
	sibling := filepath.Join(parent, "resources-evil", "secret.png")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if ahdResourceInside(real, sibling) {
		t.Fatal("a sibling directory was treated as inside the resource folder")
	}
	if !ahdResourceInside(real, filepath.Join(real, "icons", "mark.png")) {
		t.Fatal("a nested file was treated as outside the resource folder")
	}

	t.Setenv("AHDCODE_RESOURCE_ROOT", linkedRoot)
	want := canonicalResource(t, mark)
	for _, name := range []string{"icons/mark.png", "icons/alias.png", "icons/absolute.png", "nested/up.png", "also/mark.png"} {
		got, problem := AhdGUIResource(name)
		if problem != "" || got != want {
			t.Fatalf("%s -> %q %s", name, got, problem)
		}
	}

	bundle := t.TempDir()
	executable := filepath.Join(bundle, "Ledger.app", "Contents", "MacOS", "Ledger")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	macRoot, problem := ahdResourceRootFrom(executable, true, "darwin")
	if problem != "" {
		t.Fatal(problem)
	}
	macIcons := filepath.Join(macRoot, "icons", "nested")
	if err := os.MkdirAll(macIcons, 0o755); err != nil {
		t.Fatal(err)
	}
	macMark := filepath.Join(macIcons, "mark.png")
	if err := os.WriteFile(macMark, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_RESOURCE_ROOT", macRoot)
	got, problem := AhdGUIResource("icons/nested/mark.png")
	if problem != "" || got != canonicalResource(t, macMark) {
		t.Fatalf("packaged macOS resource %q %s", got, problem)
	}
}

func TestResourceSymlinkCannotLeaveTheRoot(t *testing.T) {
	root := t.TempDir()
	icons := filepath.Join(root, "icons")
	if err := os.Mkdir(icons, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.png")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(outside, "folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "folder", "mark.png"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(icons, "escape.png")); err != nil {
		t.Fatal(err)
	}
	relativeOutside, err := filepath.Rel(icons, secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(relativeOutside, filepath.Join(icons, "relative.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "folder"), filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-target.png", filepath.Join(icons, "gone.png")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_RESOURCE_ROOT", root)
	for _, name := range []string{"icons/escape.png", "icons/relative.png", "linked/mark.png"} {
		got, problem := AhdGUIResource(name)
		if got != "" || problem != "the resource path leaves the resource folder" {
			t.Fatalf("%s -> %q %s", name, got, problem)
		}
	}
	_, problem := AhdGUIResource("icons/gone.png")
	if problem != "could not read the resource icons/gone.png: the file does not exist" {
		t.Fatalf("dangling: %s", problem)
	}
}

func canonicalResource(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		t.Fatal(err)
	}
	return absolute
}
