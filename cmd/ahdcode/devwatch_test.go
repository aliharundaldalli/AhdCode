package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// touch creates or updates a file's mtime, distinctly from any prior write,
// so pollChanges reliably observes a change.
func touch(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	future := time.Now().Add(time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}

func TestDevWatcherPathSetReplaceIsGraphAware(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "app.ahd")
	dep := filepath.Join(dir, "Dep.ahd")
	pruned := filepath.Join(dir, "Pruned.ahd")
	touch(t, entry, "entry v1")
	touch(t, dep, "dep v1")
	touch(t, pruned, "pruned v1")

	events := make(chan devControllerEvent, 4)
	watcher := newDevWatcher([]string{entry}, events)

	// Growing the set: dep is newly tracked. Its baseline must come from its
	// CURRENT state, not be reported as a spurious first change.
	watcher.applyPathSet([]string{entry, dep, pruned})
	if changed := watcher.pollChanges(); changed {
		t.Fatalf("newly tracked paths must not report as changed on their first poll")
	}

	// A tracked path's real edit is detected.
	touch(t, dep, "dep v2")
	if changed := watcher.pollChanges(); !changed {
		t.Fatalf("expected pollChanges to observe the edit to a tracked dependency")
	}

	// Shrinking the set: pruned drops out. Editing it afterward must not be
	// observed -- this is the "removed dependency stops being watched"
	// requirement (section 25.C).
	watcher.applyPathSet([]string{entry, dep})
	touch(t, pruned, "pruned v2, but no longer watched")
	if changed := watcher.pollChanges(); changed {
		t.Fatalf("editing a pruned dependency must not be reported as a change")
	}
	if _, tracked := watcher.snapshots[pruned]; tracked {
		t.Fatalf("pruned path must not remain in the tracked set")
	}

	// A path re-added after having been pruned is treated as newly tracked
	// again (fresh baseline), exactly like any other new path -- not
	// compared against its stale, no-longer-relevant last snapshot.
	watcher.applyPathSet([]string{entry, dep, pruned})
	if changed := watcher.pollChanges(); changed {
		t.Fatalf("re-adding a path must take a fresh baseline, not report a spurious change")
	}

	// A require(...) target that does not exist yet is tracked as missing,
	// not an error, and its later creation is observed as a change.
	unresolved := filepath.Join(dir, "NotYetCreated.ahd")
	watcher.applyPathSet([]string{entry, unresolved})
	if changed := watcher.pollChanges(); changed {
		t.Fatalf("a still-missing tracked path must not itself report as changed")
	}
	touch(t, unresolved, "now it exists")
	if changed := watcher.pollChanges(); !changed {
		t.Fatalf("expected pollChanges to observe the previously-missing path being created")
	}
}
