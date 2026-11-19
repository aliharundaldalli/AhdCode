//go:build windows

package ahdruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemplateWindowsLeafOpenRejectsReplacement(t *testing.T) {
	root := t.TempDir()
	template := filepath.Join(root, "template.html")
	target := filepath.Join(root, "target.html")
	for path, body := range map[string]string{template: "local", target: "target bytes"} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := ahdTemplateEnsureContained(template); err != nil {
		t.Fatal(err)
	}
	if body, err := ahdTemplateReadFile(template); err != nil || string(body) != "local" {
		t.Fatalf("regular leaf = %q, %v", body, err)
	}
	if err := os.Remove(template); err != nil {
		t.Fatal(err)
	}

	t.Run("junction", func(t *testing.T) {
		directory := filepath.Join(root, "directory")
		if err := os.Mkdir(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := makeTemplateJunction(t, directory, template); err != nil {
			t.Skipf("could not create Windows junction: %v", err)
		}
		if body, err := ahdTemplateReadFile(template); err == nil || len(body) != 0 {
			t.Fatalf("replacement junction = %q, %v", body, err)
		}
		if err := os.Remove(template); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("file symlink", func(t *testing.T) {
		if err := os.Symlink(target, template); err != nil {
			t.Skipf("could not create Windows file symlink: %v", err)
		}
		if body, err := ahdTemplateReadFile(template); err == nil || len(body) != 0 {
			t.Fatalf("replacement file symlink = %q, %v", body, err)
		}
	})
}
