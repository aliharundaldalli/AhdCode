//go:build go1.27

package ahdruntime

import (
	"testing"
	"uuid"
)

// Go 1.27 ships a standard uuid package. AhdCode does not use it -- go.mod and
// CI name an earlier Go, and its parser accepts forms AhdCode rejects -- but
// where it is available it independently confirms the RFC 9562 bit layout.
func TestUUIDAgreesWithGoStandardLibrary(t *testing.T) {
	for index := 0; index < 500; index++ {
		for _, text := range []string{AhdUUIDV4(AhdClassUUIDError), AhdUUIDV7(AhdClassUUIDError)} {
			parsed, err := uuid.Parse(text)
			if err != nil {
				t.Fatalf("standard library rejects %q: %v", text, err)
			}
			if parsed.String() != text {
				t.Fatalf("standard library formats %q as %q", text, parsed.String())
			}
		}
		for version, text := range map[int64]string{4: uuid.NewV4().String(), 7: uuid.NewV7().String()} {
			if AhdUUIDParse(AhdClassUUIDError, text) != text {
				t.Fatalf("AhdCode does not round-trip the standard library's %q", text)
			}
			if got := AhdUUIDVersion(AhdClassUUIDError, text); got != version {
				t.Fatalf("version of %q = %d, want %d", text, got, version)
			}
		}
	}
	// Forms the standard parser accepts but AhdCode's strict grammar rejects.
	for _, text := range []string{
		"{017f22e2-79b0-7cc3-98c4-dc0c0c07398f}",
		"urn:uuid:017f22e2-79b0-7cc3-98c4-dc0c0c07398f",
		"017f22e279b07cc398c4dc0c0c07398f",
	} {
		if _, err := uuid.Parse(text); err != nil {
			t.Fatalf("standard library now rejects %q; update this comparison", text)
		}
		if AhdUUIDIsValid(text) {
			t.Fatalf("AhdCode accepts the non-canonical form %q", text)
		}
	}
}
