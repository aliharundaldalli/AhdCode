package parser

import (
	"fmt"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/syntax/token"
)

// A reserved keyword written where a name belongs is one mistake, and it gets
// one diagnostic: PAR015 names the keyword and says to pick another name. The
// parser then reads that token as the identifier the author meant, so the rest
// of the declaration parses normally instead of collapsing into a run of
// "expected expression" and "expected newline" errors.
//
// Later lines usually use the same name again (`write(not)`). There the
// keyword keeps its real meaning, so those lines can still fail to parse, but
// every such failure is an echo of the mistake PAR015 already explained and is
// dropped from the result.

// reservedNameSuggestions offers a replacement that reads naturally for the
// keywords most often chosen as names; the rest get a generic one.
var reservedNameSuggestions = map[token.Kind]string{
	token.KeywordNot: "score", token.KeywordState: "status", token.KeywordCondition: "rule",
	token.KeywordDefault: "fallback", token.KeywordFrom: "source", token.KeywordAll: "items",
	token.KeywordAs: "alias", token.KeywordIn: "input", token.KeywordIs: "matches",
	token.KeywordHas: "contains", token.KeywordSame: "identical", token.KeywordAnd: "both",
	token.KeywordOr: "either", token.KeywordFor: "target", token.KeywordIf: "check",
	token.KeywordElse: "other", token.KeywordWhile: "duration", token.KeywordUntil: "deadline",
	token.KeywordBreak: "pause", token.KeywordContinue: "next", token.KeywordAttempt: "tries",
	token.KeywordExcept: "excluded", token.KeywordUltimately: "finalStep", token.KeywordToss: "throwCount",
	token.KeywordReturn: "result", token.KeywordBring: "imported", token.KeywordRequire: "needed",
	token.KeywordTrue: "yes", token.KeywordFalse: "no", token.KeywordNull: "empty",
	token.KeywordLambda: "callback", token.KeywordUses: "usage",
	token.KeywordInt: "count", token.KeywordReal: "amount", token.KeywordComplex: "number",
	token.KeywordString: "text", token.KeywordBool: "flag", token.KeywordNothing: "none",
	token.KeywordList: "items", token.KeywordPair: "entries", token.KeywordFunction: "action",
	token.KeywordClass: "kind", token.KeywordObject: "item", token.KeywordError: "problem",
	token.KeywordLocal: "nearby", token.KeywordGlobal: "shared", token.KeywordConstant: "fixed",
}

func isReservedKeyword(kind token.Kind) bool {
	found, ok := token.LookupKeyword(kind.String())
	return ok && found == kind
}

// reportReservedName reports PAR015 for the current token, the first time
// each keyword is used as a name, and rewrites the token into the identifier
// the author meant. Repeating the mistake with the same keyword is read the
// same way without a second report.
func (p *parser) reportReservedName() {
	item := p.current()
	word := item.Kind.String()
	if p.reservedNames == nil {
		p.reservedNames = make(map[token.Kind]int)
	}
	if _, seen := p.reservedNames[item.Kind]; !seen {
		suggestion, known := reservedNameSuggestions[item.Kind]
		if !known {
			suggestion = "my" + word
		}
		p.errorSpan(codeReservedKeywordName,
			fmt.Sprintf("'%s' is a reserved keyword in AhdCode", word), item.Span,
			fmt.Sprintf("choose a different identifier, such as '%s'", suggestion))
		p.reservedNames[item.Kind] = item.Span.Start.Offset
	}
	p.readAsName()
}

// readAsName rewrites the current keyword token into an identifier.
func (p *parser) readAsName() {
	if p.index >= len(p.tokens) {
		return
	}
	item := &p.tokens[p.index]
	item.Value = item.Kind.String()
	item.Kind = token.Identifier
}

// reservedOperandName reads a keyword that PAR015 already reported as a name
// as that name when it appears where an operand belongs, so `write(not)` and
// `not + 5` refer to the binding the author declared.
func (p *parser) reservedOperandName() {
	if _, reported := p.reservedNames[p.current().Kind]; reported {
		// What else goes wrong on this line only echoes PAR015.
		if p.echoLines == nil {
			p.echoLines = make(map[int]bool)
		}
		p.echoLines[p.current().Span.Start.Line] = true
		p.readAsName()
	}
}

// reservedStatementName recognizes a statement that begins by declaring,
// assigning, or updating a reserved keyword as if it were a name. No valid
// statement starts with a keyword followed by one of these operators.
func (p *parser) reservedStatementName() {
	if !isReservedKeyword(p.current().Kind) {
		return
	}
	switch next := p.peek(1).Kind; {
	case next == token.Colon, next == token.Declare, isAssignmentOperator(next),
		next == token.Increment, next == token.Decrement:
		p.reportReservedName()
	}
}

// expectName reads a binding or member name. A reserved keyword in that
// place is reported as PAR015 and read as the name.
func (p *parser) expectName(message string) token.Token {
	if isReservedKeyword(p.current().Kind) {
		p.reportReservedName()
	}
	return p.expect(token.Identifier, message)
}

// withoutReservedNameEchoes drops the other parse errors on lines that use a
// keyword PAR015 reported as a name. PAR015 itself always stays.
func (p *parser) withoutReservedNameEchoes(items []diagnostics.Diagnostic) []diagnostics.Diagnostic {
	if len(p.reservedNames) == 0 {
		return items
	}
	echoLines := p.echoLines
	if echoLines == nil {
		echoLines = make(map[int]bool)
	}
	for _, item := range p.tokens {
		if first, reported := p.reservedNames[item.Kind]; reported && item.Span.Start.Offset > first {
			echoLines[item.Span.Start.Line] = true
		}
	}
	kept := items[:0]
	for _, item := range items {
		if item.Code != codeReservedKeywordName && echoLines[item.Span.Start.Line] {
			continue
		}
		kept = append(kept, item)
	}
	return kept
}
