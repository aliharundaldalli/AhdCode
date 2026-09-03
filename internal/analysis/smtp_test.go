package analysis

import (
	"path/filepath"
	"testing"
)

func TestSMTPIsDiscoveredAfterBringAndFrom(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()

	store.Open(path, "bring SM\n")
	items := store.Completion(path, len("bring SM"))
	if !hasLabel(items, "SMTP") || detailOf(items, "SMTP") != "module SMTP" {
		t.Fatalf("expected SMTP after `bring SM`, got %#v", items)
	}

	store.Open(path, "from SMTP bring SMTP\n")
	items = store.Completion(path, len("from SMTP bring SMTP"))
	if detailOf(items, "SMTPError") != "Class SMTPError" ||
		detailOf(items, "SMTPClient") != "Class SMTPClient" ||
		detailOf(items, "SMTPMessage") != "Class SMTPMessage" {
		t.Fatalf("expected SMTP classes after `from SMTP bring SMTP`, got %#v", items)
	}
}

func TestSMTPNamespaceMembersCarryRealSignatures(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()

	text := "bring SMTP\nx := SMTP.\n"
	store.Open(path, text)
	items := store.Completion(path, offsetOf(t, text, "SMTP.")+len("SMTP."))
	want := map[string]string{
		"client":      "client: (host: String, port: Int, security: String := default, timeoutSeconds: Int := default) -> SMTPClient",
		"message":     "message: (from: String, to: List<String>, subject: String) -> SMTPMessage",
		"SMTPClient":  "Class SMTPClient",
		"SMTPMessage": "Class SMTPMessage",
		"SMTPError":   "Class SMTPError",
	}
	for label, detail := range want {
		if detailOf(items, label) != detail {
			t.Fatalf("SMTP.%s detail = %q; want %q (items %#v)", label, detailOf(items, label), detail, items)
		}
	}
}
