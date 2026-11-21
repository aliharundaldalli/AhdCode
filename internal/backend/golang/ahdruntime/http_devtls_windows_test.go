//go:build windows

package ahdruntime

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDevHTTPSWindowsCredentialOpenRejectsReplacement(t *testing.T) {
	root := t.TempDir()
	credential := filepath.Join(root, "key.pem")
	target := filepath.Join(root, "target.pem")
	for path, body := range map[string]string{credential: "local", target: "target bytes"} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := AhdHTTPDevCheckTLSPath(credential); err != nil {
		t.Fatal(err)
	}
	if body, err := ahdDevTLSReadCredential(credential); err != nil || string(body) != "local" {
		t.Fatalf("regular leaf = %q, %v", body, err)
	}
	if err := os.Remove(credential); err != nil {
		t.Fatal(err)
	}

	t.Run("junction", func(t *testing.T) {
		directory := filepath.Join(root, "directory")
		if err := os.Mkdir(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command("cmd", "/c", "mklink", "/J", credential, directory).CombinedOutput(); err != nil {
			t.Skipf("could not create Windows junction: %v: %s", err, output)
		}
		if body, err := ahdDevTLSReadCredential(credential); err == nil || len(body) != 0 {
			t.Fatalf("replacement junction = %q, %v", body, err)
		}
		if err := os.Remove(credential); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("file symlink", func(t *testing.T) {
		if err := os.Symlink(target, credential); err != nil {
			t.Skipf("could not create Windows file symlink: %v", err)
		}
		if body, err := ahdDevTLSReadCredential(credential); err == nil || len(body) != 0 {
			t.Fatalf("replacement file symlink = %q, %v", body, err)
		}
	})
}
