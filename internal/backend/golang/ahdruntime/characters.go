package ahdruntime

// AhdCode Characters standard module runtime.
//
// This file is compiled twice: once as part of the compiler, where the
// interactive evaluator calls it directly, and once as generated program
// source. Both execution modes therefore run the same code, and it depends
// only on the Go standard library.
//
// The unit of this module is one Unicode code point (a scalar value), not a
// UTF-8 byte and not an extended grapheme cluster. A character is represented
// by an ordinary String that holds exactly one code point; AhdCode has no Char
// type. A decomposed sequence such as "e" followed by U+0301 COMBINING ACUTE
// ACCENT is two characters here even though it displays as one glyph.
//
// Every operation that takes one character validates that its String holds
// exactly one code point. Inspecting only the first code point would hide the
// caller's mistake, so an empty or longer String raises instead.

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

// The interactive evaluator raises CharactersError through this runtime, so the
// class needs a constructor in the compiler process too. A generated program
// raises through its own generated descriptor instead.
func init() {
	AhdRegisterError(AhdClassCharactersError, func(message string) AhdInstance {
		instance := &ahdModuleError{message: message}
		instance.AhdSetClass(AhdClassCharactersError)
		return instance
	})
}

// ahdCharactersDecode walks text one code point at a time. AhdCode Strings
// are valid UTF-8, but the check keeps a malformed byte from ever surfacing
// as a silent U+FFFD replacement character.
func ahdCharactersDecode(errorClass *AhdClass, operation, text string, visit func(start, size int)) {
	for index := 0; index < len(text); {
		value, size := utf8.DecodeRuneInString(text[index:])
		if value == utf8.RuneError && size <= 1 {
			AhdRaiseClass(errorClass, "Characters."+operation+" received text that is not valid UTF-8")
		}
		visit(index, size)
		index += size
	}
}

// AhdCharactersList returns each code point of text as its own String, in
// source order. The source String is not changed.
func AhdCharactersList(errorClass *AhdClass, text string) *AhdList[string] {
	items := make([]string, 0, len(text))
	ahdCharactersDecode(errorClass, "list", text, func(start, size int) {
		items = append(items, text[start:start+size])
	})
	return AhdNewList(items...)
}

// AhdCharactersCount returns the number of code points in text. It agrees
// with len(text), which AhdCode already defines in code points.
func AhdCharactersCount(errorClass *AhdClass, text string) int64 {
	count := int64(0)
	ahdCharactersDecode(errorClass, "count", text, func(int, int) { count++ })
	return count
}

// ahdCharactersOne returns the single code point a character String holds.
func ahdCharactersOne(errorClass *AhdClass, operation, character string) rune {
	if character == "" {
		AhdRaiseClass(errorClass, "Characters."+operation+" requires exactly one character; received an empty String")
	}
	value, size := utf8.DecodeRuneInString(character)
	if value == utf8.RuneError && size <= 1 {
		AhdRaiseClass(errorClass, "Characters."+operation+" received text that is not valid UTF-8")
	}
	if size != len(character) {
		count := AhdCharactersCount(errorClass, character)
		AhdRaiseClass(errorClass, "Characters."+operation+" requires exactly one character; received "+
			strconv.FormatInt(count, 10)+" characters")
	}
	return value
}

// AhdCharactersCodePoint returns the Unicode scalar value of one character.
func AhdCharactersCodePoint(errorClass *AhdClass, character string) int64 {
	return int64(ahdCharactersOne(errorClass, "codePoint", character))
}

// AhdCharactersFromCodePoint returns the one-character String for a Unicode
// scalar value. Negative values, values above U+10FFFF, and the surrogate
// range U+D800..U+DFFF are not scalar values and raise; nothing is truncated,
// wrapped, or replaced with U+FFFD.
func AhdCharactersFromCodePoint(errorClass *AhdClass, value int64) string {
	if value < 0 || value > unicode.MaxRune {
		AhdRaiseClass(errorClass, "Characters.fromCodePoint: "+strconv.FormatInt(value, 10)+
			" is outside the Unicode range 0..1114111")
	}
	if value >= 0xD800 && value <= 0xDFFF {
		AhdRaiseClass(errorClass, "Characters.fromCodePoint: "+strconv.FormatInt(value, 10)+
			" is a surrogate code point, not a Unicode scalar value")
	}
	return string(rune(value))
}

// AhdCharactersIsLetter reports Unicode General Category L (Lu, Ll, Lt, Lm, Lo).
func AhdCharactersIsLetter(errorClass *AhdClass, character string) bool {
	return unicode.IsLetter(ahdCharactersOne(errorClass, "isLetter", character))
}

// AhdCharactersIsDigit reports Unicode General Category Nd, decimal digits of
// any script. Other numeric characters such as "²" (No) or "Ⅻ" (Nl) are not
// digits.
func AhdCharactersIsDigit(errorClass *AhdClass, character string) bool {
	return unicode.IsDigit(ahdCharactersOne(errorClass, "isDigit", character))
}

// AhdCharactersIsWhitespace reports the Unicode White_Space property, the same
// definition String.trim uses.
func AhdCharactersIsWhitespace(errorClass *AhdClass, character string) bool {
	return unicode.IsSpace(ahdCharactersOne(errorClass, "isWhitespace", character))
}

// AhdCharactersIsUpper reports Unicode General Category Lu.
func AhdCharactersIsUpper(errorClass *AhdClass, character string) bool {
	return unicode.IsUpper(ahdCharactersOne(errorClass, "isUpper", character))
}

// AhdCharactersIsLower reports Unicode General Category Ll.
func AhdCharactersIsLower(errorClass *AhdClass, character string) bool {
	return unicode.IsLower(ahdCharactersOne(errorClass, "isLower", character))
}

// AhdCharactersIsAlphaNumeric reports a letter (L) or a decimal digit (Nd).
func AhdCharactersIsAlphaNumeric(errorClass *AhdClass, character string) bool {
	value := ahdCharactersOne(errorClass, "isAlphaNumeric", character)
	return unicode.IsLetter(value) || unicode.IsDigit(value)
}

// AhdCharactersIsPunctuation reports Unicode General Category P.
func AhdCharactersIsPunctuation(errorClass *AhdClass, character string) bool {
	return unicode.IsPunct(ahdCharactersOne(errorClass, "isPunctuation", character))
}

// AhdCharactersIsSymbol reports Unicode General Category S (Sm, Sc, Sk, So).
func AhdCharactersIsSymbol(errorClass *AhdClass, character string) bool {
	return unicode.IsSymbol(ahdCharactersOne(errorClass, "isSymbol", character))
}
