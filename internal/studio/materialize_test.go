package studio

import (
	"os"
	"path/filepath"
	"testing"

	"ahdcode/internal/ahdversion"
)

func TestMaterializeWritesExactVersionStudio(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("AHDCODE_STUDIO_CACHE", cache)
	dir, err := Materialize()
	if err != nil {
		t.Fatal(err)
	}
	if dir != cache {
		t.Fatalf("cache dir = %s; want %s", dir, cache)
	}
	if _, err := os.Stat(filepath.Join(dir, "app.ahd")); err != nil {
		t.Fatal(err)
	}
	stamp, err := os.ReadFile(filepath.Join(dir, stampName))
	if err != nil {
		t.Fatal(err)
	}
	if string(stamp) != ahdversion.Number+"\n" {
		t.Fatalf("stamp = %q", stamp)
	}
	again, err := Materialize()
	if err != nil || again != dir {
		t.Fatalf("reuse failed: %s %v", again, err)
	}
}
