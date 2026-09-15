package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// Terminal reaches editors through the compiler's module interface, like every
// other standard module; there is no second symbol catalog.

func TestCompletionOffersTerminalModuleAndMembers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, "bring \n")
	if items := store.Completion(path, len("bring ")); !hasLabel(items, "Terminal") {
		t.Fatalf("expected Terminal among module completions, got %#v", items)
	}
	text := "bring Terminal\nTerminal.\n"
	store.Open(path, text)
	items := store.Completion(path, len("bring Terminal\nTerminal."))
	for _, name := range []string{"emit", "error", "flush", "isInteractive", "width", "height", "supportsColor", "style", "pretty"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Terminal completions, got %#v", name, items)
		}
	}
	store.Open(path, "from Terminal bring \n")
	items = store.Completion(path, len("from Terminal bring "))
	for _, name := range []string{"TerminalError", "emit", "style"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Terminal exports, got %#v", name, items)
		}
	}
}

func TestHoverShowsTerminalSignatures(t *testing.T) {
	text := "bring Terminal\nTerminal.emit([\"a\"])\nstyled := Terminal.style(\"ok\")\n"
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	hover, ok := store.Hover(path, offsetOf(t, text, "emit")+1)
	if !ok {
		t.Fatal("no hover for Terminal.emit")
	}
	for _, part := range []string{"parts: List<String>", "separator: String", "ending: String"} {
		if !strings.Contains(hover.Text, part) {
			t.Fatalf("hover does not describe Terminal.emit (%q missing): %q", part, hover.Text)
		}
	}
	hover, ok = store.Hover(path, offsetOf(t, text, "Terminal.style")+len("Terminal.")+1)
	if !ok {
		t.Fatal("no hover for Terminal.style")
	}
	for _, part := range []string{"text: String", "foreground: String", "background: String", "bold: Bool", "underline: Bool"} {
		if !strings.Contains(hover.Text, part) {
			t.Fatalf("hover does not describe Terminal.style (%q missing): %q", part, hover.Text)
		}
	}
}

func TestHoverAndSignatureAllTerminalOperations(t *testing.T) {
	text := `bring Terminal
Terminal.emit(["a"])
Terminal.error("err")
Terminal.flush()
b := Terminal.isInteractive()
w := Terminal.width()
h := Terminal.height()
c := Terminal.supportsColor()
s := Terminal.style("text")
Terminal.pretty([1, 2, 3])
`
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)

	ops := []struct {
		name     string
		contains string
	}{
		{"emit", "parts: List<String>"},
		{"error", "text: String"},
		{"flush", "() -> Nothing"},
		{"isInteractive", "() -> Bool"},
		{"width", "() -> Int?"},
		{"height", "() -> Int?"},
		{"supportsColor", "() -> Bool"},
		{"style", "text: String"},
		{"pretty", "pretty"},
	}

	for _, op := range ops {
		offset := offsetOf(t, text, "Terminal."+op.name) + len("Terminal.") + 1
		hover, ok := store.Hover(path, offset)
		if !ok {
			t.Fatalf("no hover for Terminal.%s", op.name)
		}
		if !strings.Contains(hover.Text, op.contains) {
			t.Fatalf("hover for Terminal.%s (%q) does not contain %q", op.name, hover.Text, op.contains)
		}
	}

	// Also verify signature help inside Terminal.pretty call
	sigHelp, ok := store.SignatureHelp(path, offsetOf(t, text, "Terminal.pretty([")+len("Terminal.pretty("))
	if !ok {
		t.Fatalf("expected signature help for Terminal.pretty")
	}
	if !strings.Contains(sigHelp.Label, "value") {
		t.Fatalf("expected signature help label for Terminal.pretty to contain 'value', got %q", sigHelp.Label)
	}
}
