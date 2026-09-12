package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const charactersModulePrefix = "builtin:Characters::"

var charactersErrorClass = ir.ClassID("builtin:Characters::class::CharactersError")

// charactersCall lowers the Characters module's functions. Every call maps to
// one plain runtime function in ahdruntime/characters.go, which the evaluator
// also calls, so both execution modes share one implementation. Each helper
// receives the error class because each can be given a value outside its
// domain: a String that is not exactly one code point, or an Int that is not a
// Unicode scalar value.
func (generator *generator) charactersCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), charactersModulePrefix)
	errorClass := generator.descriptorName(charactersErrorClass)
	text := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	switch name {
	case "list":
		return "AhdCharactersList(" + errorClass + ", " + text(0) + ")"
	case "count":
		return "AhdCharactersCount(" + errorClass + ", " + text(0) + ")"
	case "codePoint":
		return "AhdCharactersCodePoint(" + errorClass + ", " + text(0) + ")"
	case "fromCodePoint":
		return "AhdCharactersFromCodePoint(" + errorClass + ", " +
			generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.IntType}, false) + ")"
	case "isLetter":
		return "AhdCharactersIsLetter(" + errorClass + ", " + text(0) + ")"
	case "isDigit":
		return "AhdCharactersIsDigit(" + errorClass + ", " + text(0) + ")"
	case "isWhitespace":
		return "AhdCharactersIsWhitespace(" + errorClass + ", " + text(0) + ")"
	case "isUpper":
		return "AhdCharactersIsUpper(" + errorClass + ", " + text(0) + ")"
	case "isLower":
		return "AhdCharactersIsLower(" + errorClass + ", " + text(0) + ")"
	case "isAlphaNumeric":
		return "AhdCharactersIsAlphaNumeric(" + errorClass + ", " + text(0) + ")"
	case "isPunctuation":
		return "AhdCharactersIsPunctuation(" + errorClass + ", " + text(0) + ")"
	case "isSymbol":
		return "AhdCharactersIsSymbol(" + errorClass + ", " + text(0) + ")"
	default:
		return generator.unsupported("Characters function "+name, meta.Span)
	}
}
