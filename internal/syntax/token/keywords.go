package token

var keywords = map[string]Kind{
	"and": KeywordAnd, "or": KeywordOr, "not": KeywordNot, "same": KeywordSame,
	"is": KeywordIs, "in": KeywordIn, "has": KeywordHas, "if": KeywordIf,
	"else": KeywordElse, "while": KeywordWhile, "until": KeywordUntil, "for": KeywordFor,
	"break": KeywordBreak, "continue": KeywordContinue, "state": KeywordState,
	"condition": KeywordCondition, "default": KeywordDefault, "attempt": KeywordAttempt,
	"except": KeywordExcept, "ultimately": KeywordUltimately, "toss": KeywordToss,
	"return": KeywordReturn, "bring": KeywordBring, "from": KeywordFrom, "all": KeywordAll,
	"as": KeywordAs, "true": KeywordTrue, "false": KeywordFalse, "null": KeywordNull,
	"Int": KeywordInt, "Real": KeywordReal, "Complex": KeywordComplex, "String": KeywordString, "Bool": KeywordBool,
	"Nothing": KeywordNothing, "List": KeywordList, "Pair": KeywordPair,
	"Function": KeywordFunction, "lambda": KeywordLambda, "Overload": KeywordOverload, "Override": KeywordOverride,
	"Class": KeywordClass, "Attributes": KeywordAttributes, "Constant": KeywordConstant,
	"Local": KeywordLocal, "Global": KeywordGlobal, "Confidential": KeywordConfidential,
	"Object": KeywordObject, "Error": KeywordError,
}

// LookupKeyword returns the reserved keyword kind for normalized text.
func LookupKeyword(normalized string) (Kind, bool) {
	kind, ok := keywords[normalized]
	return kind, ok
}
