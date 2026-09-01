package lsp

import "testing"

func TestOffsetToPositionASCII(t *testing.T) {
	text := "write(\"hi\")\n"
	index := newLineIndex(text)
	position := index.OffsetToPosition(0)
	if position != (Position{Line: 0, Character: 0}) {
		t.Fatalf("position = %+v", position)
	}
	position = index.OffsetToPosition(6)
	if position != (Position{Line: 0, Character: 6}) {
		t.Fatalf("position = %+v", position)
	}
}

func TestOffsetToPositionMultipleLines(t *testing.T) {
	text := "x: Int := 1\ny: Int := 2\nz: Int := 3\n"
	index := newLineIndex(text)
	// "y" is the first byte of the second line.
	offset := len("x: Int := 1\n")
	position := index.OffsetToPosition(offset)
	if position != (Position{Line: 1, Character: 0}) {
		t.Fatalf("position = %+v", position)
	}
}

func TestPositionToOffsetASCIIRoundTrip(t *testing.T) {
	text := "x: Int := 1\ny: Int := 2\n"
	index := newLineIndex(text)
	for _, offset := range []int{0, 3, 11, 12, 15, len(text)} {
		position := index.OffsetToPosition(offset)
		roundTripped := index.PositionToOffset(position)
		if roundTripped != offset {
			t.Fatalf("offset %d -> %+v -> %d", offset, position, roundTripped)
		}
	}
}

func TestOffsetToPositionTurkishCharacters(t *testing.T) {
	// "isim" then a Turkish string containing ş, ı, ç -- each a 2-byte UTF-8
	// sequence but exactly one UTF-16 code unit (all within the BMP).
	text := "isim: String := \"İşçi\"\n"
	index := newLineIndex(text)
	// The opening quote is at byte offset len("isim: String := ").
	quoteOffset := len("isim: String := ")
	position := index.OffsetToPosition(quoteOffset)
	// Character count must equal the *rune* count up to that point, not the
	// byte count, since every preceding rune here is ASCII.
	if position.Character != quoteOffset {
		t.Fatalf("position = %+v, want character %d", position, quoteOffset)
	}
	// Two UTF-16 units past the quote's own position skips the quote itself
	// (1 unit) plus "İ" (2 UTF-8 bytes, but still 1 UTF-16 unit -- it's in
	// the BMP), landing right after "İ".
	iBytes := len("İ")
	afterI := index.PositionToOffset(Position{Line: 0, Character: position.Character + 2})
	want := quoteOffset + 1 + iBytes
	if afterI != want {
		t.Fatalf("afterI = %d, want %d (quoteOffset=%d, iBytes=%d)", afterI, want, quoteOffset, iBytes)
	}
}

func TestPositionToOffsetUnicodeGeneral(t *testing.T) {
	// A mix of ASCII and multi-byte-but-BMP Unicode (Greek, CJK).
	text := "α: String := \"日本語\"\n"
	index := newLineIndex(text)
	// "α" is 2 UTF-8 bytes, 1 UTF-16 unit, 1 rune.
	afterAlpha := index.PositionToOffset(Position{Line: 0, Character: 1})
	if afterAlpha != len("α") {
		t.Fatalf("afterAlpha = %d, want %d", afterAlpha, len("α"))
	}
	position := index.OffsetToPosition(len(text) - 2) // just before the closing quote
	roundTripped := index.PositionToOffset(position)
	if roundTripped != len(text)-2 {
		t.Fatalf("round trip failed: %d -> %+v -> %d", len(text)-2, position, roundTripped)
	}
}

// TestNonBMPEmojiUTF16SurrogatePair is the release-blocking case from the
// task: a single non-BMP character (an emoji) must occupy exactly two
// UTF-16 code units, and a position after it must land one whole code point
// later in byte terms, not one UTF-16 unit (half a surrogate pair) later.
func TestNonBMPEmojiUTF16SurrogatePair(t *testing.T) {
	text := "value: String := \"🙂\"\n"
	index := newLineIndex(text)
	quoteContentOffset := len("value: String := \"")
	beforeEmoji := index.OffsetToPosition(quoteContentOffset)

	emojiBytes := len("🙂")
	if emojiBytes != 4 {
		t.Fatalf("test fixture assumption broken: 🙂 is %d UTF-8 bytes, expected 4", emojiBytes)
	}

	afterEmojiOffset := quoteContentOffset + emojiBytes
	afterEmojiPosition := index.OffsetToPosition(afterEmojiOffset)
	// The emoji is one code point occupying two UTF-16 units.
	if afterEmojiPosition.Character != beforeEmoji.Character+2 {
		t.Fatalf("emoji width in UTF-16 units = %d, want 2 (before=%+v after=%+v)",
			afterEmojiPosition.Character-beforeEmoji.Character, beforeEmoji, afterEmojiPosition)
	}
	// Converting that exact post-emoji Position back must land on the real
	// byte offset right after the emoji's 4 UTF-8 bytes -- not 1 UTF-16 unit
	// (2 bytes) in, which would split the surrogate pair and land mid-rune.
	roundTripped := index.PositionToOffset(afterEmojiPosition)
	if roundTripped != afterEmojiOffset {
		t.Fatalf("round trip after emoji = %d, want %d", roundTripped, afterEmojiOffset)
	}
	// A diagnostic/hover exactly at the character following the emoji (the
	// closing quote) must be at the correct editor column, not one UTF-16
	// code unit short (which is what a naive rune-count-as-UTF16 bug would
	// produce).
	closingQuoteOffset := afterEmojiOffset
	closingQuotePosition := index.OffsetToPosition(closingQuoteOffset)
	if closingQuotePosition.Character != beforeEmoji.Character+2 {
		t.Fatalf("closing quote character = %d, want %d", closingQuotePosition.Character, beforeEmoji.Character+2)
	}
}

func TestPositionToOffsetSurrogateBoundaryDoesNotPanic(t *testing.T) {
	text := "\"🙂\"\n"
	index := newLineIndex(text)
	// Character 2 lands exactly between the emoji's two surrogate halves.
	// This must not panic and must land at a defensible boundary (the start
	// of the emoji rune, since splitting it is meaningless).
	offset := index.PositionToOffset(Position{Line: 0, Character: 2})
	if offset != 1 {
		t.Fatalf("mid-surrogate offset = %d, want 1 (start of the emoji rune)", offset)
	}
}

func TestCRLFLineEndingsExcludedFromCharacterCount(t *testing.T) {
	text := "x: Int := 1\r\ny: Int := 2\r\n"
	index := newLineIndex(text)
	lineTwoStart := len("x: Int := 1\r\n")
	position := index.OffsetToPosition(lineTwoStart)
	if position != (Position{Line: 1, Character: 0}) {
		t.Fatalf("position = %+v", position)
	}
	// The end of line 0's *content* must be right before the \r, i.e.
	// character count equal to len("x: Int := 1"), not len("x: Int := 1\r").
	endOfLineZero := index.OffsetToPosition(len("x: Int := 1"))
	if endOfLineZero.Character != len("x: Int := 1") {
		t.Fatalf("endOfLineZero = %+v", endOfLineZero)
	}
	roundTripped := index.PositionToOffset(Position{Line: 0, Character: len("x: Int := 1")})
	if roundTripped != len("x: Int := 1") {
		t.Fatalf("round trip = %d, want %d", roundTripped, len("x: Int := 1"))
	}
}

func TestLineFeedOnlyLineEndings(t *testing.T) {
	text := "x: Int := 1\ny: Int := 2\n"
	index := newLineIndex(text)
	endOfLineZero := index.OffsetToPosition(len("x: Int := 1"))
	if endOfLineZero.Character != len("x: Int := 1") {
		t.Fatalf("endOfLineZero = %+v", endOfLineZero)
	}
}

func TestPositionPastEndOfDocumentClampsRatherThanPanics(t *testing.T) {
	text := "x: Int := 1\n"
	index := newLineIndex(text)
	position := index.OffsetToPosition(len(text) + 100)
	if position.Line != 1 || position.Character != 0 {
		t.Fatalf("position past end = %+v", position)
	}
	offset := index.PositionToOffset(Position{Line: 100, Character: 0})
	if offset != len(text) {
		t.Fatalf("offset for out-of-range line = %d, want %d", offset, len(text))
	}
}
