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
	// recoveredDotContinuation records that the statement just parsed was a
	// rejected leading-dot continuation, so the statement list can mark the
	// receiver it continued as malformed.
	recoveredDotContinuation bool
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
		p.recoveredDotContinuation = false
		statement := p.parseStatement(scope)
		if statement != nil {
			if p.recoveredDotContinuation {
				// The receiver that this rejected chain continues was already
				// parsed as a complete statement. Its initializer is only
				// "complete" because the parser stopped at the newline, so
				// leaving it intact would invite a second, misleading
				// diagnostic about the truncated value's type. Marking it
				// malformed keeps PAR013 the single explanation.
				invalidateTruncatedReceiver(statements)
			}
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

// invalidateTruncatedReceiver replaces the value of the statement a rejected
// leading-dot chain was continuing with a BadExpr, so semantic analysis does
// not type-check a receiver the parser cut short. Only the value is replaced:
// the declared name, type, and modifiers stay, so unrelated later uses of the
// binding still resolve normally instead of cascading unknown-name errors.
func invalidateTruncatedReceiver(statements []ast.Stmt) {
	if len(statements) == 0 {
		return
	}
	switch previous := statements[len(statements)-1].(type) {
	case *ast.VariableDecl:
		if previous.Initializer != nil {
			previous.Initializer = &ast.BadExpr{Base: ast.Base{Range: previous.Initializer.Span()}}
		}
	case *ast.AssignmentStmt:
		if previous.Value != nil {
			previous.Value = &ast.BadExpr{Base: ast.Base{Range: previous.Value.Span()}}
		}
	}
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
	case token.Dot:
		return p.parseLeadingDotContinuation()
	default:
		return p.parseSimpleStatement(scope)
	}
}

// parseLeadingDotContinuation rejects a member chain continued from a new
// line. AhdCode does not have leading-dot continuation, and this does not make
// it valid; it reports the first precise PAR013 and then consumes the whole
// malformed chain as one region.
//
// The chain is consumed with bracket awareness, because a rejected
// `.filter(` normally opens arguments that run over several physical lines. A
// newline inside those brackets does not end the chain, so the argument list
// is never reinterpreted as independent statements and its closing bracket
// never becomes a spurious "expected expression". Once the brackets balance, a
// further leading-dot member on the next line belongs to the same rejected
// chain and is absorbed silently rather than reported again.
func (p *parser) parseLeadingDotContinuation() ast.Stmt {
	leading := p.advance()
	p.errorSpan(codeLeadingDotContinuation, "method chain cannot continue from a new line", leading.Span,
		"keep the member call on the same expression as its receiver, or store the intermediate result in a variable")
	end := p.consumeDotContinuation(leading.Span.End)
	p.recoveredDotContinuation = true
	bad := &ast.BadExpr{Base: p.base(leading.Span.Start, end)}
	return &ast.ExprStmt{Base: p.base(leading.Span.Start, end), Expression: bad}
}

// consumeDotContinuation skips the remainder of one rejected chain, including
// every bracketed argument list and every further leading-dot member that
// continues it.
func (p *parser) consumeDotContinuation(end source.Position) source.Position {
	for {
		depth := 0
		for !p.atEnd() {
			if depth == 0 && (p.check(token.Newline) || p.check(token.RightBrace)) {
				break
			}
			switch p.current().Kind {
			case token.LeftParen, token.LeftBracket:
				depth++
			case token.RightParen, token.RightBracket:
				// A closing bracket with nothing open belongs to an enclosing
				// construct, so recovery stops before consuming it.
				if depth == 0 {
					return end
				}
				depth--
			case token.LeftBrace:
				// A block brace is not part of an expression chain; leaving it
				// alone keeps the enclosing statement list recoverable.
				if depth == 0 {
					return end
				}
				depth++
			case token.RightBrace:
				depth--
			}
			end = p.advance().Span.End
		}
		// Look past the newline for another member of the same chain.
		next := 0
		for p.peek(next).Kind == token.Newline {
			next++
		}
		if p.peek(next).Kind != token.Dot {
			return end
		}
		for range next + 1 {
			end = p.advance().Span.End
		}
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
func (p *parser) requireSameLineRHS(operator token.Token) bool {
	if !p.check(token.Newline) {
		return false
	}
	symbol := operator.Kind.String()
	message := fmt.Sprintf("expected the assigned expression to begin after '%s' on the same line", symbol)
	p.errorSpan(codeExpectedSameLineRHS, message, operator.Span, fmt.Sprintf("write the assigned expression on the same line as '%s'", symbol))
	p.skipNewlines()
	return true
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
