package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// Editor support for a new standard module is not written by hand: completion,
// hover and signature help all read the interface the compiler builds. These
// tests prove Bits actually arrives through that path, so an editor offers it
// like any other module.

func TestCompletionOffersBitsModuleName(t *testing.T) {
	text := "bring Bi\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, len("bring Bi"))
	if !hasLabel(items, "Bits") {
		t.Fatalf("expected Bits among module completions, got %#v", items)
	}
}

func TestCompletionOffersBitsMembers(t *testing.T) {
	text := "bring Bits\nvalue: Int := Bits.\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, len("bring Bits\nvalue: Int := Bits."))
	for _, name := range []string{
		"bitAnd", "bitOr", "bitXor", "bitNot",
		"shiftLeft", "shiftRight", "shiftRightUnsigned",
		"rotateLeft", "rotateRight",
		"count", "leadingZeros", "trailingZeros",
	} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Bits member completions, got %#v", name, items)
		}
	}
}

func TestCompletionOffersBitsErrorAfterFromBring(t *testing.T) {
	text := "from Bits bring \n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, len("from Bits bring "))
	if !hasLabel(items, "BitsError") {
		t.Fatalf("expected BitsError among export completions, got %#v", items)
	}
}

// The crypto functions added to Security must be offered too, since editors
// read the same interface.
func TestCompletionOffersSecurityCryptoMembers(t *testing.T) {
	text := "bring Security\nvalue: String := Security.\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	items := store.Completion(path, len("bring Security\nvalue: String := Security."))
	for _, name := range []string{
		"sha256", "sha512", "hmacSHA256", "hmacVerify",
		"base64Encode", "base64UrlEncode", "hexEncode", "randomHex",
		"rsaSignSHA256", "rsaVerifySHA256", "aesEncrypt", "aesDecrypt",
		// the original surface must still be offered
		"passwordHash", "token",
	} {
		if !hasLabel(items, name) {
			t.Fatalf("expected %q among Security member completions, got %#v", name, items)
		}
	}
}

// Hover must show the real signature, which comes from the semantic
// interface rather than from a hand-written table.
func TestHoverShowsBitsSignature(t *testing.T) {
	text := "bring Bits\nvalue: Int := Bits.bitAnd(12, 10)\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	hover, ok := store.Hover(path, offsetOf(t, text, "bitAnd")+1)
	if !ok {
		t.Fatal("no hover for Bits.bitAnd")
	}
	if !strings.Contains(hover.Text, "bitAnd") || !strings.Contains(hover.Text, "Int") {
		t.Fatalf("hover does not describe bitAnd's signature: %q", hover.Text)
	}
}

func TestHoverShowsSecuritySHA256Signature(t *testing.T) {
	text := "bring Security\nvalue: String := Security.sha256(\"abc\")\n"
	directory := t.TempDir()
	path := filepath.Join(directory, "main.ahd")
	store := NewStore()
	store.Open(path, text)

	hover, ok := store.Hover(path, offsetOf(t, text, "sha256")+1)
	if !ok {
		t.Fatal("no hover for Security.sha256")
	}
	if !strings.Contains(hover.Text, "sha256") || !strings.Contains(hover.Text, "String") {
		t.Fatalf("hover does not describe sha256's signature: %q", hover.Text)
	}
}
