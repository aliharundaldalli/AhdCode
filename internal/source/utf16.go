package source

import (
	"sort"
	"unicode/utf8"
)

// UTF16Position is an editor position: zero-based line, zero-based
// character counted in UTF-16 code units -- deliberately different on every
// axis from AhdCode's own Position (one-based line/column, Column
// counted in Unicode code points). Converting between the two always goes
// through a LineIndex built from the real source text; neither side is ever
// approximated by a bare "subtract one" or a byte-count shortcut.
type UTF16Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// UTF16Range is a half-open [Start, End) range.
type UTF16Range struct {
	Start UTF16Position `json:"start"`
	End   UTF16Position `json:"end"`
}

// LineIndex precomputes the byte offset of the start of every line in one
// document's text, so repeated offset<->UTF16Position conversions for that
// snapshot (e.g. every diagnostic in one publish) do not each rescan the
// whole document.
type LineIndex struct {
	text   string
	starts []int
}

// NewLineIndex indexes the line boundaries in a source snapshot.
func NewLineIndex(text string) *LineIndex {
	starts := make([]int, 1, 16)
	starts[0] = 0
	for index := 0; index < len(text); index++ {
		if text[index] == '\n' {
			starts = append(starts, index+1)
		}
	}
	return &LineIndex{text: text, starts: starts}
}

// lineContentRange returns the byte range of one line's content, excluding
// its trailing line terminator (\n or \r\n) so character-counting math never
// includes it.
func (index *LineIndex) lineContentRange(line int) (start, end int) {
	if line < 0 {
		line = 0
	}
	if line >= len(index.starts) {
		return len(index.text), len(index.text)
	}
	start = index.starts[line]
	end = len(index.text)
	if line+1 < len(index.starts) {
		end = index.starts[line+1]
	}
	if end > start && index.text[end-1] == '\n' {
		end--
		if end > start && index.text[end-1] == '\r' {
			end--
		}
	}
	return start, end
}

// OffsetToPosition converts a document byte offset into a UTF16Position.
func (index *LineIndex) OffsetToPosition(offset int) UTF16Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(index.text) {
		offset = len(index.text)
	}
	// The greatest line whose start is <= offset.
	line := sort.Search(len(index.starts), func(candidate int) bool {
		return index.starts[candidate] > offset
	}) - 1
	if line < 0 {
		line = 0
	}
	lineStart, lineEnd := index.lineContentRange(line)
	if offset > lineEnd {
		offset = lineEnd
	}
	return UTF16Position{Line: line, Character: utf16Length(index.text[lineStart:offset])}
}

// PositionToOffset converts a UTF16Position back into a byte offset in the
// document, decoding UTF-16 code units against the real line text. A
// character count that would land inside a surrogate pair (malformed input)
// stops at the start of that rune rather than guessing.
func (index *LineIndex) PositionToOffset(position UTF16Position) int {
	line := position.Line
	if line < 0 {
		line = 0
	}
	lineStart, lineEnd := index.lineContentRange(line)
	offset := lineStart
	remaining := position.Character
	for offset < lineEnd && remaining > 0 {
		codePoint, size := utf8.DecodeRuneInString(index.text[offset:lineEnd])
		width := utf16Width(codePoint)
		if remaining < width {
			break
		}
		remaining -= width
		offset += size
	}
	return offset
}

// utf16Width is 1 for any code point in the Basic Multilingual Plane and 2
// for one that requires a UTF-16 surrogate pair (U+10000 and above, e.g.
// most emoji).
func utf16Width(codePoint rune) int {
	if codePoint > 0xFFFF {
		return 2
	}
	return 1
}

func utf16Length(text string) int {
	length := 0
	for _, codePoint := range text {
		length += utf16Width(codePoint)
	}
	return length
}
