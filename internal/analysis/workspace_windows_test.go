package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWorkspaceDiscoveryDoesNotFollowJunction(t *testing.T) {
	root, outside := canonicalPath(t.TempDir()), t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "outside.ahd"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked")
	if output, err := exec.Command("cmd", "/c", "mklink", "/J", link, outside).CombinedOutput(); err != nil {
		t.Fatalf("cannot create fixture junction: %v\n%s", err, output)
	}
	t.Cleanup(func() { _ = os.Remove(link) })
	if files, complete := workspaceSourceFiles([]string{root}, workspaceScanBudget); !complete || len(files) != 0 {
		t.Fatalf("junction discovery = %v, complete=%v", files, complete)
	}
}
