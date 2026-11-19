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

func TestDevReloadEnvironmentNamesOneSession(t *testing.T) {
	session := "0123456789abcdef0123456789abcdef"
	if env := devReloadEnvironment(session, 0); env != nil {
		t.Fatalf("generation 0 = %v", env)
	}
	if env := devReloadEnvironment("not-a-session", 7); env != nil {
		t.Fatalf("bad session = %v", env)
	}
	env := devReloadEnvironment(session, 7)
	if len(env) != 2 || env[0] != "AHDCODE_DEV_SESSION="+session || env[1] != "AHDCODE_DEV_GENERATION=7" {
		t.Fatalf("environment = %v", env)
	}
}

func TestDevWatcherSeesTemplateFilesOnce(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "app.ahd")
	page := filepath.Join(dir, "templates", "pages", "home.html")
	if err := os.MkdirAll(filepath.Dir(page), 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, entry, "entry")
	touch(t, page, "<p>Ana sayfa</p>")
	found := devTemplatePaths(entry)
	if len(found) != 1 || found[0] != page {
		t.Fatalf("template paths = %v", found)
	}
	events := make(chan devControllerEvent, 2)
	watcher := newDevWatcher([]string{entry}, events)
	if watcher.pollChanges() {
		t.Fatal("an existing template must not trigger a rebuild on startup")
	}
	watcher.applyPathSet(append([]string{entry}, found...))
	touch(t, page, "<p>Ana sayfa değişti</p>")
	if !watcher.pollChanges() {
		t.Fatal("a template edit was not observed")
	}
	if watcher.pollChanges() {
		t.Fatal("the same template edit was observed twice")
	}
	assertChangeOnce := func(t *testing.T, operation string) {
		t.Helper()
		if !watcher.pollChanges() {
			t.Fatalf("template %s was not observed", operation)
		}
		if watcher.pollChanges() {
			t.Fatalf("template %s was observed twice", operation)
		}
	}
	t.Run("create", func(t *testing.T) {
		touch(t, filepath.Join(filepath.Dir(page), "about.HTML"), "<p>Hakkında</p>")
		assertChangeOnce(t, "creation")
	})
	t.Run("delete", func(t *testing.T) {
		if err := os.Remove(page); err != nil {
			t.Fatal(err)
		}
		assertChangeOnce(t, "deletion")
	})
	t.Run("restore", func(t *testing.T) {
		touch(t, page, "<p>Ana sayfa</p>")
		assertChangeOnce(t, "restoration")
	})
	t.Run("rename", func(t *testing.T) {
		renamed := filepath.Join(filepath.Dir(page), "renamed.html")
		if err := os.Rename(page, renamed); err != nil {
			t.Fatal(err)
		}
		assertChangeOnce(t, "rename")
		if snapshot, tracked := watcher.snapshots[renamed]; !watcher.snapshots[page].missing || !tracked || snapshot.missing {
			t.Fatal("rename did not record both the old and new paths")
		}
	})
}

func TestDevWatcherMissingPolicy(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "app.ahd")
	dep := filepath.Join(dir, "templates", "data.ahd")
	otherHTML := filepath.Join(dir, "templates-other", "home.html")
	page := filepath.Join(dir, "templates", "home.html")
	for _, path := range []string{dep, otherHTML} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	paths := []string{entry, dep, otherHTML, page}
	for _, path := range paths {
		touch(t, path, "initial")
	}
	watcher := newDevWatcher(paths, nil)
	for _, path := range paths[1:] {
		t.Run(filepath.Base(filepath.Dir(path))+"/"+filepath.Base(path), func(t *testing.T) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if changed := watcher.pollChanges(); changed != (path == page) {
				t.Fatalf("poll after removing %s = %v", path, changed)
			}
			touch(t, path, "restored")
			if !watcher.pollChanges() {
				t.Fatalf("restoring %s did not trigger a change", path)
			}
		})
	}
}

func TestDevWatcherDebouncesABurstAndSeesReplace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "öğrenci proje")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "Sayfa.ahd")
	touch(t, entry, "version-one")
	events := make(chan devControllerEvent, 4)
	watcher := newDevWatcher([]string{entry}, events)
	watcher.start()
	defer watcher.close()

	touch(t, entry, "version-two")
	time.Sleep(40 * time.Millisecond)
	touch(t, entry, "version-three-longer")
	waitDevEvent(t, events)
	expectNoDevEvent(t, events)

	if err := os.Remove(entry); err != nil {
		t.Fatal(err)
	}
	expectNoDevEvent(t, events)
	touch(t, entry, "created-again")
	waitDevEvent(t, events)

	replaceFile(t, entry, "replaced-by-rename")
	waitDevEvent(t, events)
}

func waitDevEvent(t *testing.T, events <-chan devControllerEvent) {
	t.Helper()
	select {
	case event := <-events:
		if event.kind != eventSourceChanged {
			t.Fatalf("event = %v", event.kind)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for a source change")
	}
}

func expectNoDevEvent(t *testing.T, events <-chan devControllerEvent) {
	t.Helper()
	select {
	case event := <-events:
		t.Fatalf("unexpected event %v", event.kind)
	case <-time.After(450 * time.Millisecond):
	}
}
