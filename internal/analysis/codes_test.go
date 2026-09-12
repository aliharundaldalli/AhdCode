package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// QR and Barcode reach editors through the compiler's module interfaces, like
// every other standard module.

func TestCompletionOffersCodesModuleNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, "bring \n")
	items := store.Completion(path, len("bring "))
	for _, name := range []string{"QR", "Barcode"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %s among module completions, got %#v", name, items)
		}
	}
}

func TestCompletionOffersCodesMembers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	text := "bring Barcode\nvalue := Barcode.\n"
	store.Open(path, text)
	items := store.Completion(path, len("bring Barcode\nvalue := Barcode."))
	for _, name := range []string{"code128", "ean13", "upca"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Barcode completions, got %#v", name, items)
		}
	}
	text = "from QR bring \n"
	store.Open(path, text)
	items = store.Completion(path, len("from QR bring "))
	for _, name := range []string{"create", "QRCode", "QRError"} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among QR exports, got %#v", name, items)
		}
	}
}

func TestHoverShowsQRCreateSignature(t *testing.T) {
	text := "bring QR\ncode := QR.create(\"x\")\n"
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, text)
	hover, ok := store.Hover(path, offsetOf(t, text, "create")+1)
	if !ok {
		t.Fatal("no hover for QR.create")
	}
	if !strings.Contains(hover.Text, "value: String") || !strings.Contains(hover.Text, "QRCode") {
		t.Fatalf("hover does not describe QR.create: %q", hover.Text)
	}
}
