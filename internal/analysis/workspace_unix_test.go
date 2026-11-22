//go:build unix

package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceDiscoveryDoesNotFollowSymlinks(t *testing.T) {
	root, outside := canonicalPath(t.TempDir()), t.TempDir()
	file := filepath.Join(outside, "outside.ahd")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{"linked": outside, "linked.ahd": file} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	if files, complete := workspaceSourceFiles([]string{root}, workspaceScanBudget); !complete || len(files) != 0 {
		t.Fatalf("symlink discovery = %v, complete=%v", files, complete)
	}
}
