package parser

import (
	"fmt"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

type scopeKind uint8

const (
	scopeModule scopeKind = iota
	scopeClass
	scopeBlock
)

type parser struct {
	file   source.File
	tokens []token.Token
	index  int
	bag    diagnostics.Bag
}

// Parse builds a syntax-only typed AST. It performs no name or type resolution.
func Parse(file source.File, tokens []token.Token) Result {
	owned := append([]token.Token(nil), tokens...)
	if len(owned) == 0 || owned[len(owned)-1].Kind != token.EOF {
		at := source.Position{Offset: len(file.Text), Line: 1, Column: 1}
		owned = append(owned, token.Token{Kind: token.EOF, Span: source.NewSpan(file.ID, at, at), Synthetic: true})
	}
	p := &parser{file: file, tokens: owned}
	program := p.parseProgram()
	return Result{Program: program, Tokens: owned, Diagnostics: p.bag.Items()}
}

func (p *parser) parseProgram() *ast.Program {
	start := p.current().Span.Start
	statements := p.parseStatementList(scopeModule, token.EOF)
	end := p.current().Span.End
	return &ast.Program{Base: p.base(start, end), Statements: statements}
}

func (p *parser) parseStatementList(scope scopeKind, terminator token.Kind) []ast.Stmt {
	var statements []ast.Stmt
	p.skipNewlines()
	for !p.check(terminator) && !p.atEnd() {
		before := p.index
		statement := p.parseStatement(scope)
		if statement != nil {
			statements = append(statements, statement)
		}
		if p.index == before {
			p.advance()
		}
		if p.check(token.Newline) {
			p.skipNewlines()
			continue
		}
		if p.check(terminator) || p.atEnd() {
			break
		}
		p.errorCurrent(codeExpectedSeparator, "expected newline between statements", "place each independent statement on its own line")
		p.synchronize(terminator)
		p.skipNewlines()
	}
	return statements
}

func (p *parser) parseStatement(scope scopeKind) ast.Stmt {
	switch p.current().Kind {
	case token.KeywordIf:
		return p.parseIf()
	case token.KeywordWhile:
		return p.parseWhile()
	case token.KeywordUntil:
		return p.parseUntil()
	case token.KeywordFor:
		return p.parseFor()
	case token.KeywordState:
		return p.parseState()
	case token.KeywordAttempt:
		return p.parseAttempt()
	case token.KeywordBreak:
		start := p.advance().Span.Start
		return &ast.BreakStmt{Base: p.base(start, p.previous().Span.End)}
	case token.KeywordContinue:
		start := p.advance().Span.Start
		return &ast.ContinueStmt{Base: p.base(start, p.previous().Span.End)}
	case token.KeywordReturn:
		return p.parseReturn()
	case token.KeywordToss:
		return p.parseToss()
	case token.KeywordBring, token.KeywordFrom:
		return p.parseBring()
	case token.Increment, token.Decrement:
		return p.parsePrefixUpdate()
	default:
		return p.parseSimpleStatement(scope)
	}
}

func (p *parser) current() token.Token {
	if p.index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.index]
}

func (p *parser) previous() token.Token {
	if p.index <= 0 {
		return p.tokens[0]
	}
	return p.tokens[p.index-1]
}

func (p *parser) peek(distance int) token.Token {
	index := p.index + distance
	if index < 0 {
		index = 0
	}
	if index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[index]
}

func (p *parser) advance() token.Token {
	current := p.current()
	if p.index < len(p.tokens)-1 {
		p.index++
	}
	return current
}

func (p *parser) atEnd() bool { return p.current().Kind == token.EOF }

func (p *parser) check(kind token.Kind) bool { return p.current().Kind == kind }

func (p *parser) match(kinds ...token.Kind) bool {
	for _, kind := range kinds {
		if p.check(kind) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *parser) expect(kind token.Kind, message string) token.Token {
	if p.check(kind) {
		return p.advance()
	}
	p.errorCurrent(codeExpectedToken, message, fmt.Sprintf("expected %s", kind))
	at := p.current().Span.Start
	return token.Token{Kind: kind, Span: source.NewSpan(p.file.ID, at, at), Synthetic: true}
}

func (p *parser) skipNewlines() bool {
	skipped := false
	for p.match(token.Newline) {
		skipped = true
	}
	return skipped
}

func (p *parser) nextNonNewlineIndex(from int) int {
	for from < len(p.tokens) && p.tokens[from].Kind == token.Newline {
		from++
	}
	if from >= len(p.tokens) {
		return len(p.tokens) - 1
	}
	return from
}

func (p *parser) errorCurrent(code, message, hint string) {
	p.bag.Error(code, message, p.current().Span, hint)
}

func (p *parser) errorSpan(code, message string, span source.Span, hint string) {
	p.bag.Error(code, message, span, hint)
}

func (p *parser) base(start, end source.Position) ast.Base {
	return ast.Base{Range: source.NewSpan(p.file.ID, start, end)}
}

// requireSameLineRHS enforces that the first token of an assigned expression
// begins on the same physical line as the := or = operator introducing it.
// Line breaks are otherwise non-semantic, but this one placement rule keeps
// declarations and assignments visually anchored to their operator. On
// violation it reports the exact diagnostic and skips the offending newlines
// so parsing can recover and keep finding further errors.
func (p *parser) requireSameLineRHS(operator token.Token) {
	if !p.check(token.Newline) {
		return
	}
	symbol := operator.Kind.String()
	message := fmt.Sprintf("expected the assigned expression to begin after '%s' on the same line", symbol)
	p.errorSpan(codeExpectedSameLineRHS, message, operator.Span, fmt.Sprintf("move the expression after '%s' onto the same line", symbol))
	p.skipNewlines()
}

func (p *parser) synchronize(terminator token.Kind) {
	for !p.atEnd() && !p.check(terminator) && !p.check(token.Newline) {
		if p.current().Kind == token.RightBrace {
			return
		}
		p.advance()
	}
}

func spanStart(node ast.Node) source.Position { return node.Span().Start }
func spanEnd(node ast.Node) source.Position   { return node.Span().End }
