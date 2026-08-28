package lexer

const (
	codeInvalidUTF8               = "LEX001"
	codeUnexpectedCharacter       = "LEX002"
	codeUnterminatedBlockComment  = "LEX003"
	codeInvalidNumericLiteral     = "LEX004"
	codeInvalidEscape             = "LEX005"
	codeUnterminatedString        = "LEX006"
	codeNewlineInString           = "LEX007"
	codeUnmatchedStringBrace      = "LEX008"
	codeUnterminatedInterpolation = "LEX009"
	codeEmptyInterpolation        = "LEX010"
)
