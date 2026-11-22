package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestWorkspaceVolumeRoot(t *testing.T) {
	root := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		root = filepath.VolumeName(t.TempDir()) + root
	}
	project := filepath.Join(root, "workspace-root-fixture")
	if !isWorkspaceVolumeRoot(root) || isWorkspaceVolumeRoot(project) {
		t.Fatalf("unexpected root classification for %q", root)
	}
	if runtime.GOOS == "windows" {
		for _, path := range []string{`C:\`, `C:/`, `\\server\share\`} {
			if !isWorkspaceVolumeRoot(path) {
				t.Errorf("%q is a Windows volume root", path)
			}
		}
	}
	store := NewStore()
	store.SetWorkspaceRoots([]string{root, project})
	if roots := store.workspaceRootsFor(filepath.Join(root, "main.ahd")); !reflect.DeepEqual(roots, []string{canonicalPath(project)}) {
		t.Fatalf("workspace roots = %v", roots)
	}
}

func TestWorkspaceDiscoveryBudgets(t *testing.T) {
	root := canonicalPath(t.TempDir())
	for _, name := range []string{"b.ahd", "a.ahd", "ignore.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	child := filepath.Join(root, "z")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "more.ahd"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	a, b, more := filepath.Join(root, "a.ahd"), filepath.Join(root, "b.ahd"), filepath.Join(child, "more.ahd")
	for _, tc := range []struct {
		name     string
		budget   workspaceBudget
		want     []string
		complete bool
	}{
		{"exact limits", workspaceBudget{2, 5, 3}, []string{a, b, more}, true},
		{"ordered file cap", workspaceBudget{2, 5, 1}, []string{a}, false},
		{"directory cap", workspaceBudget{1, 5, 3}, []string{a, b}, false},
		{"oversized directory", workspaceBudget{2, 3, 3}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, complete := workspaceSourceFiles([]string{root, child}, tc.budget)
			if !reflect.DeepEqual(got, tc.want) || complete != tc.complete {
				t.Fatalf("workspace paths %v, complete=%v", got, complete)
			}
		})
	}
}

func TestRenameRefusesIncompleteDiscovery(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.ahd")
	source := "answer: Int := 42\nwrite(answer)\n"
	store := NewStore()
	store.Open(path, source)
	position := offsetOf(t, source, "write(answer)") + len("write(")
	if edits, ok := store.Rename(path, position, "result"); !ok || len(edits) != 2 {
		t.Fatalf("complete rename = %v, ok=%v", edits, ok)
	}
	for i := 0; i <= workspaceScanBudget.files; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("unit%04d.ahd", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if edits, ok := store.Rename(path, position, "result"); ok || len(edits) != 0 {
		t.Fatalf("incomplete rename returned edits: %v", edits)
	}
}

func TestWorkspaceRequestsRefreshChangedImportOverlay(t *testing.T) {
	root := t.TempDir()
	mainPath, helper := filepath.Join(root, "main.ahd"), filepath.Join(root, "Helper.ahd")
	saved := "answer: Int := 42\n"
	if err := os.WriteFile(helper, []byte(saved), 0o600); err != nil {
		t.Fatal(err)
	}
	main := "from Helper bring answer\nwrite(answer)\n"
	store := NewStore()
	store.Open(mainPath, main)
	store.Open(helper, "// unsaved edit\n"+saved)
	offset := offsetOf(t, main, "write(answer)") + len("write(")
	edits, ok := store.Rename(mainPath, offset, "result")
	if !ok || len(edits) != 3 {
		t.Fatalf("rename with updated overlay = %v, ok=%v", edits, ok)
	}
	for _, edit := range edits {
		if edit.Path == canonicalPath(helper) && edit.Span.Start.Offset != len("// unsaved edit\n") {
			t.Fatalf("rename offset used the saved file: %v", edit)
		}
	}
	store.Close(helper)
	locations := store.References(mainPath, offset, true)
	for _, location := range locations {
		if location.Path == canonicalPath(helper) && location.Span.Start.Offset == 0 {
			return
		}
	}
	t.Fatalf("closed overlay did not fall back to the saved declaration: %v", locations)
}
