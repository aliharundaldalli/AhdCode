package gopdf

import "testing"

func TestParseCFFEncodingUsesEmbeddedCustomCodes(t *testing.T) {
	encoding, ok := parseCFFEncoding([]byte{0, 3, 88, 100, 112}, 0, 4)
	if !ok {
		t.Fatal("custom CFF format 0 encoding was rejected")
	}
	for code, want := range map[byte]uint16{88: 1, 100: 2, 112: 3} {
		if got := encoding[code]; got != want {
			t.Errorf("encoding[%d] = %d, want %d", code, got, want)
		}
	}
	font := &glyphFont{cff: &cffOutlines{encoding: encoding, hasEncoding: true}}
	if got, ok := font.gid(100); !ok || got != 2 {
		t.Fatalf("simple CFF code resolved to (%d, %t), want (2, true)", got, ok)
	}
	rangeEncoding, ok := parseCFFEncoding([]byte{1, 1, 10, 2}, 0, 4)
	if !ok || rangeEncoding[10] != 1 || rangeEncoding[11] != 2 || rangeEncoding[12] != 3 {
		t.Fatalf("custom CFF format 1 range was not mapped: %v", rangeEncoding)
	}
}

func TestParseCFFEncodingRejectsUnsafeOrIncompleteMaps(t *testing.T) {
	for _, data := range [][]byte{
		{0x80},          // supplement requires SID name resolution
		{2, 1},          // unknown format
		{0, 2, 1},       // truncated format 0
		{1, 1, 255, 1},  // range wraps past byte 255
		{0, 3, 1, 2, 3}, // more glyphs than the program carries
	} {
		if _, ok := parseCFFEncoding(data, 0, 3); ok {
			t.Errorf("parseCFFEncoding(%v) unexpectedly succeeded", data)
		}
	}
}
