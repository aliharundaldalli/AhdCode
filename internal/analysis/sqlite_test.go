package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// The SQLite module is an acceptance test of the v0.2.2 language-server
// architecture: nothing in internal/analysis or internal/lsp names SQLite,
// yet every editor feature below derives from the compiler's module interface.

func detailOf(items []CompletionItem, label string) string {
	for _, item := range items {
		if item.Label == label {
			return item.Detail
		}
	}
	return ""
}

func TestSQLiteModuleIsDiscoveredAfterBringAndFrom(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()

	store.Open(path, "bring SQL\n")
	items := store.Completion(path, len("bring SQL"))
	if !hasLabel(items, "SQLite") || detailOf(items, "SQLite") != "module SQLite" {
		t.Fatalf("expected SQLite after `bring SQL`, got %#v", items)
	}
	if hasLabel(items, "Math") {
		t.Fatalf("prefix SQL should filter Math out, got %#v", items)
	}

	store.Open(path, "from SQL\n")
	if items := store.Completion(path, len("from SQL")); !hasLabel(items, "SQLite") {
		t.Fatalf("expected SQLite after `from SQL`, got %#v", items)
	}

	store.Open(path, "from SQLite bring SQL\n")
	items = store.Completion(path, len("from SQLite bring SQL"))
	for _, want := range []string{"SQLiteError", "SQLiteValue"} {
		if detailOf(items, want) != "Class "+want {
			t.Fatalf("expected exported Class %s after `from SQLite bring SQL`, got %#v", want, items)
		}
	}
	if hasLabel(items, "Database") || hasLabel(items, "open") {
		t.Fatalf("prefix SQL should filter Database/open out, got %#v", items)
	}
	store.Open(path, "from SQLite bring D\n")
	if items := store.Completion(path, len("from SQLite bring D")); detailOf(items, "Database") != "Class Database" {
		t.Fatalf("expected exported Class Database, got %#v", items)
	}
}

func TestSQLiteNamespaceMembersCarryRealSignatures(t *testing.T) {
	text := "bring SQLite\nx := SQLite.\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, offsetOf(t, text, "SQLite.")+len("SQLite."))
	want := map[string]string{
		"open":        "open: (path: String) -> Database",
		"nullValue":   "nullValue: () -> SQLiteValue",
		"fromInt":     "fromInt: (value: Int) -> SQLiteValue",
		"fromReal":    "fromReal: (value: Real) -> SQLiteValue",
		"fromString":  "fromString: (value: String) -> SQLiteValue",
		"Database":    "Class Database",
		"SQLiteValue": "Class SQLiteValue",
		"SQLiteError": "Class SQLiteError",
	}
	if len(items) != len(want) {
		t.Fatalf("SQLite exposes %d members; want %d: %#v", len(items), len(want), items)
	}
	for label, detail := range want {
		if detailOf(items, label) != detail {
			t.Fatalf("SQLite.%s detail = %q; want %q", label, detailOf(items, label), detail)
		}
	}
}

func TestSQLiteHoverAndSignatureHelpComeFromTheCompiler(t *testing.T) {
	text := "bring SQLite\nfrom SQLite bring Database\nfrom SQLite bring SQLiteValue\n" +
		"db: Database := SQLite.open(\"notes.db\")\n" +
		"rows: List<Pair<String, SQLiteValue>> := db.query(\"SELECT id FROM notes\", [])\n" +
		"changed := db.execute(\"DELETE FROM notes\")\n" +
		"value := SQLite.fromInt(1)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	if hover, ok := store.Hover(path, offsetOf(t, text, "SQLite.open")+len("SQLite.")+1); !ok || hover.Text != "open: (path: String) -> Database" {
		t.Fatalf("hover on SQLite.open = %#v, ok = %v", hover, ok)
	}
	if hover, ok := store.Hover(path, offsetOf(t, text, "db: Database")+1); !ok || hover.Text != "db: Database" {
		t.Fatalf("hover on db = %#v, ok = %v", hover, ok)
	}
	if hover, ok := store.Hover(path, offsetOf(t, text, "changed := ")+1); !ok || hover.Text != "changed: Int" {
		t.Fatalf("hover on changed = %#v, ok = %v", hover, ok)
	}
	if hover, ok := store.Hover(path, offsetOf(t, text, "value := ")+1); !ok || hover.Text != "value: SQLiteValue" {
		t.Fatalf("hover on value = %#v, ok = %v", hover, ok)
	}
	if hover, ok := store.Hover(path, offsetOf(t, text, "rows: List")+1); !ok || hover.Text != "rows: List<Pair<String, SQLiteValue>>" {
		t.Fatalf("hover on rows = %#v, ok = %v", hover, ok)
	}

	help, ok := store.SignatureHelp(path, offsetOf(t, text, "SQLite.open(")+len("SQLite.open("))
	if !ok || help.Label != "(path: String) -> Database" || len(help.Parameters) != 1 || help.Parameters[0] != "path: String" {
		t.Fatalf("signature help for SQLite.open = %#v, ok = %v", help, ok)
	}
	if help, ok := store.SignatureHelp(path, offsetOf(t, text, "SQLite.fromInt(")+len("SQLite.fromInt(")); !ok || help.Label != "(value: Int) -> SQLiteValue" {
		t.Fatalf("signature help for SQLite.fromInt = %#v, ok = %v", help, ok)
	}
}

func TestSQLiteDiagnosticsFlowThroughTheStore(t *testing.T) {
	text := "bring SQLite\ndb := SQLite.open(\":memory:\")\ndb.execute(42)\nvalue: Int := SQLite.fromInt(1)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	result := store.Open(path, text)

	var messages []string
	for _, diagnostic := range result.Diagnostics[path] {
		messages = append(messages, diagnostic.Code+": "+diagnostic.Message)
	}
	joined := strings.Join(messages, "\n")
	if !strings.Contains(joined, "execute") || !strings.Contains(joined, "String") {
		t.Fatalf("expected a diagnostic for execute(42), got:\n%s", joined)
	}
	if !strings.Contains(joined, "SQLiteValue") {
		t.Fatalf("expected a diagnostic for Int := SQLiteValue, got:\n%s", joined)
	}
}
