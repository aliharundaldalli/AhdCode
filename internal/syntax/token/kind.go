package token

// Kind identifies a lexical token kind.
type Kind uint16

const (
	Invalid Kind = iota
	EOF
	Newline
	Identifier
	IntLiteral
	RealLiteral
	StringStart
	StringText
	InterpolationStart
	InterpolationEnd
	StringEnd

	KeywordAnd
	KeywordOr
	KeywordNot
	KeywordSame
	KeywordIs
	KeywordIn
	KeywordHas
	KeywordIf
	KeywordElse
	KeywordWhile
	KeywordUntil
	KeywordFor
	KeywordBreak
	KeywordContinue
	KeywordState
	KeywordCondition
	KeywordDefault
	KeywordAttempt
	KeywordExcept
	KeywordUltimately
	KeywordToss
	KeywordReturn
	KeywordBring
	KeywordFrom
	KeywordAll
	KeywordAs
	KeywordTrue
	KeywordFalse
	KeywordNull
	KeywordInt
	KeywordReal
	KeywordString
	KeywordBool
	KeywordNothing
	KeywordList
	KeywordPair
	KeywordFunction
	KeywordOverload
	KeywordOverride
	KeywordClass
	KeywordAttributes
	KeywordConstant
	KeywordLocal
	KeywordGlobal
	KeywordConfidential
	KeywordObject
	KeywordError

	Colon
	Declare
	Assign
	Arrow
	Plus
	Minus
	Star
	Slash
	Percent
	Caret
	PlusAssign
	MinusAssign
	StarAssign
	SlashAssign
	PercentAssign
	CaretAssign
	Increment
	Decrement
	Equal
	NotEqual
	Less
	LessEqual
	Greater
	GreaterEqual
	Dot
	Comma
	LeftParen
	RightParen
	LeftBracket
	RightBracket
	LeftBrace
	RightBrace
)

var names = [...]string{
	Invalid: "Invalid", EOF: "EOF", Newline: "Newline", Identifier: "Identifier",
	IntLiteral: "IntLiteral", RealLiteral: "RealLiteral", StringStart: "StringStart",
	StringText: "StringText", InterpolationStart: "InterpolationStart",
	InterpolationEnd: "InterpolationEnd", StringEnd: "StringEnd",
	KeywordAnd: "and", KeywordOr: "or", KeywordNot: "not", KeywordSame: "same",
	KeywordIs: "is", KeywordIn: "in", KeywordHas: "has", KeywordIf: "if",
	KeywordElse: "else", KeywordWhile: "while", KeywordUntil: "until", KeywordFor: "for",
	KeywordBreak: "break", KeywordContinue: "continue", KeywordState: "state",
	KeywordCondition: "condition", KeywordDefault: "default", KeywordAttempt: "attempt",
	KeywordExcept: "except", KeywordUltimately: "ultimately", KeywordToss: "toss",
	KeywordReturn: "return", KeywordBring: "bring", KeywordFrom: "from", KeywordAll: "all",
	KeywordAs: "as", KeywordTrue: "true", KeywordFalse: "false", KeywordNull: "null",
	KeywordInt: "Int", KeywordReal: "Real", KeywordString: "String", KeywordBool: "Bool",
	KeywordNothing: "Nothing", KeywordList: "List", KeywordPair: "Pair",
	KeywordFunction: "Function", KeywordOverload: "Overload", KeywordOverride: "Override",
	KeywordClass: "Class", KeywordAttributes: "Attributes", KeywordConstant: "Constant",
	KeywordLocal: "Local", KeywordGlobal: "Global", KeywordConfidential: "Confidential",
	KeywordObject: "Object", KeywordError: "Error", Colon: ":", Declare: ":=", Assign: "=",
	Arrow: "->", Plus: "+", Minus: "-", Star: "*", Slash: "/", Percent: "%", Caret: "^",
	PlusAssign: "+=", MinusAssign: "-=", StarAssign: "*=", SlashAssign: "/=",
	PercentAssign: "%=", CaretAssign: "^=", Increment: "++", Decrement: "--",
	Equal: "==", NotEqual: "!=", Less: "<", LessEqual: "<=", Greater: ">",
	GreaterEqual: ">=", Dot: ".", Comma: ",", LeftParen: "(", RightParen: ")",
	LeftBracket: "[", RightBracket: "]", LeftBrace: "{", RightBrace: "}",
}

func (k Kind) String() string {
	if int(k) < len(names) && names[k] != "" {
		return names[k]
	}
	return "Kind(?)"
}
