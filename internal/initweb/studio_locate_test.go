package initweb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/ahdversion"
)

func TestLocateBundledStudioWithoutRepository(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("AHDCODE_STUDIO_CACHE", cache)
	t.Setenv("AHDCODE_ROOT", "")
	_ = os.Unsetenv("AHDCODE_ROOT")
	t.Chdir(t.TempDir())

	dir, err := LocateAhdDataStudio()
	if err != nil {
		t.Fatal(err)
	}
	if dir != cache {
		t.Fatalf("studio dir = %s; want cache %s", dir, cache)
	}
	if _, err := os.Stat(filepath.Join(dir, "app.ahd")); err != nil {
		t.Fatal(err)
	}
	stamp, err := os.ReadFile(filepath.Join(dir, "AHDSTUDIO_VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(stamp)) != ahdversion.Number {
		t.Fatalf("bundled Studio version = %q; want %s", stamp, ahdversion.Number)
	}
}

func TestLocateHonorsDeveloperRootOverride(t *testing.T) {
	root := t.TempDir()
	studio := filepath.Join(root, "tools", "AhdDataStudio")
	if err := os.MkdirAll(studio, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(studio, "app.ahd"), []byte("write(1)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AHDCODE_ROOT", root)
	t.Setenv("AHDCODE_STUDIO_CACHE", t.TempDir())
	dir, err := LocateAhdDataStudio()
	if err != nil {
		t.Fatal(err)
	}
	if dir != studio {
		t.Fatalf("override dir = %s; want %s", dir, studio)
	}
}
