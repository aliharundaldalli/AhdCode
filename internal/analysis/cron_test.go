package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// Cron reaches editors through the compiler's module interface; nothing here
// is a hand-written table and no syntax was added for it.

func TestCompletionOffersCronModuleName(t *testing.T) {
	text := "bring Cr\n"
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	if items := store.Completion(path, len("bring Cr")); !hasLabel(items, "Cron") {
		t.Fatalf("expected Cron among module completions, got %#v", items)
	}
}

func TestCompletionOffersCronMembersAndClasses(t *testing.T) {
	text := "bring Cron\nvalue := Cron.\n"
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	items := store.Completion(path, len("bring Cron\nvalue := Cron."))
	for _, name := range []string{"scheduler", "next"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Cron member completions, got %#v", name, items)
		}
	}

	text = "from Cron bring \n"
	store.Open(path, text)
	items = store.Completion(path, len("from Cron bring "))
	for _, name := range []string{"Scheduler", "CronError"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Cron export completions, got %#v", name, items)
		}
	}
}

func TestHoverShowsCronNextSignature(t *testing.T) {
	text := "bring Cron\nbring Time\nvalue := Cron.next(\"0 9 * * *\", Time.utc())\n"
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	hover, ok := store.Hover(path, offsetOf(t, text, "next")+1)
	if !ok {
		t.Fatal("no hover for Cron.next")
	}
	if !strings.Contains(hover.Text, "next") || !strings.Contains(hover.Text, "expression: String") || !strings.Contains(hover.Text, "DateTime") {
		t.Fatalf("hover does not describe next's signature: %q", hover.Text)
	}
}
