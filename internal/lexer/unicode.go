package lexer

import (
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

func normalizeIdentifier(text string) string {
	return norm.NFKC.String(text)
}

func isXIDStart(r rune) bool {
	if r == '_' {
		return true
	}
	normalized := norm.NFKC.String(string(r))
	first, size := utf8.DecodeRuneInString(normalized)
	if size == 0 || !isIDStart(first) {
		return false
	}
	for _, continuation := range normalized[size:] {
		if !isIDContinue(continuation) {
			return false
		}
	}
	return true
}

func isXIDContinue(r rune) bool {
	if r == '_' {
		return true
	}
	normalized := norm.NFKC.String(string(r))
	if normalized == "" {
		return false
	}
	for _, continuation := range normalized {
		if !isIDContinue(continuation) {
			return false
		}
	}
	return true
}

func isIDStart(r rune) bool {
	if unicode.Is(unicode.Pattern_Syntax, r) || unicode.Is(unicode.Pattern_White_Space, r) {
		return false
	}
	return unicode.IsLetter(r) || unicode.Is(unicode.Nl, r) || unicode.Is(unicode.Other_ID_Start, r)
}

func isIDContinue(r rune) bool {
	if unicode.Is(unicode.Pattern_Syntax, r) || unicode.Is(unicode.Pattern_White_Space, r) {
		return false
	}
	return isIDStart(r) || unicode.IsOneOf([]*unicode.RangeTable{
		unicode.Mn,
		unicode.Mc,
		unicode.Nd,
		unicode.Pc,
		unicode.Other_ID_Continue,
	}, r)
}
