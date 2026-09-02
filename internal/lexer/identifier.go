package lexer

import "unicode/utf8"

// ValidIdentifier reports whether text is a legal AhdCode identifier using the
// same XID-based rules the lexer applies when scanning identifiers.
func ValidIdentifier(text string) bool {
	if text == "" {
		return false
	}
	first, size := utf8.DecodeRuneInString(text)
	if !isXIDStart(first) {
		return false
	}
	for _, r := range text[size:] {
		if !isXIDContinue(r) {
			return false
		}
	}
	return true
}
