package parser

import (
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

func (p *parser) parseBlock() *ast.Block {
	opening := p.expect(token.LeftBrace, "expected { to start block")
	p.skipNewlines()
	statements := p.parseStatementList(scopeBlock, token.RightBrace)
	closing := p.expect(token.RightBrace, "expected } after block")
	return &ast.Block{Base: p.base(opening.Span.Start, closing.Span.End), Statements: statements}
}

func (p *parser) parseIf() ast.Stmt {
	start := p.advance().Span.Start
	condition := p.parseExpression(0)
	body := p.parseBlock()
	branches := []ast.ConditionalBlock{{
		Base: p.base(condition.Span().Start, body.Span().End), Condition: condition, Body: body,
	}}
	end := body.Span().End
	var elseBody *ast.Block

	for p.consumeClauseKeyword(token.KeywordElse) {
		if p.match(token.KeywordIf) {
			condition = p.parseExpression(0)
			body = p.parseBlock()
			branches = append(branches, ast.ConditionalBlock{
				Base: p.base(condition.Span().Start, body.Span().End), Condition: condition, Body: body,
			})
			end = body.Span().End
			continue
		}
		elseBody = p.parseBlock()
		end = elseBody.Span().End
		break
	}
	return &ast.IfStmt{Base: p.base(start, end), Branches: branches, Else: elseBody}
}

func (p *parser) parseWhile() ast.Stmt {
	start := p.advance().Span.Start
	condition := p.parseExpression(0)
	body := p.parseBlock()
	return &ast.WhileStmt{Base: p.base(start, body.Span().End), Condition: condition, Body: body}
}

func (p *parser) parseUntil() ast.Stmt {
	start := p.advance().Span.Start
	condition := p.parseExpression(0)
	body := p.parseBlock()
	return &ast.UntilStmt{Base: p.base(start, body.Span().End), Condition: condition, Body: body}
}

func (p *parser) parseFor() ast.Stmt {
	start := p.advance().Span.Start
	name := p.expect(token.Identifier, "expected iteration variable after for")
	var typeRef *ast.TypeRef
	if p.match(token.Colon) {
		// The iteration binding is implicitly Local, so a scope modifier here is
		// a syntax error rather than an unknown type name. Recover by dropping
		// the modifiers so the declared type still parses.
		if isDeclarationModifier(p.current().Kind) {
			p.errorCurrent(codeInvalidControlSyntax, "for iteration bindings are implicitly Local", "write for name: Type in iterable")
			for isDeclarationModifier(p.current().Kind) {
				p.advance()
			}
		}
		typeRef = p.parseTypeRef()
	}
	p.expect(token.KeywordIn, "expected in after iteration variable")
	iterable := p.parseExpression(0)
	body := p.parseBlock()
	return &ast.ForStmt{
		Base: p.base(start, body.Span().End), Name: name.Value, Type: typeRef, Iterable: iterable, Body: body,
	}
}

func (p *parser) parseState() ast.Stmt {
	start := p.advance().Span.Start
	value := p.parseExpression(0)
	p.expect(token.LeftBrace, "expected { after state expression")
	p.skipNewlines()
	var conditions []ast.StateCondition
	for !p.check(token.RightBrace) && !p.atEnd() {
		clauseStart := p.current().Span.Start
		if !p.match(token.KeywordCondition) {
			p.errorCurrent(codeInvalidControlSyntax, "expected condition in state body", "state bodies contain only condition clauses")
			p.synchronize(token.RightBrace)
			p.skipNewlines()
			continue
		}
		isDefault := p.match(token.KeywordDefault)
		var match ast.Expr
		if !isDefault {
			match = p.parseExpression(0)
		}
		body := p.parseBlock()
		conditions = append(conditions, ast.StateCondition{
			Base: p.base(clauseStart, body.Span().End), Match: match, Default: isDefault, Body: body,
		})
		p.skipNewlines()
	}
	closing := p.expect(token.RightBrace, "expected } after state body")
	return &ast.StateStmt{Base: p.base(start, closing.Span.End), Value: value, Conditions: conditions}
}

func (p *parser) parseAttempt() ast.Stmt {
	start := p.advance().Span.Start
	body := p.parseBlock()
	end := body.Span().End
	var excepts []ast.ExceptClause
	var ultimately *ast.Block

	for p.consumeClauseKeyword(token.KeywordExcept) {
		clauseStart := p.previous().Span.Start
		typeRef := p.parseTypeRef()
		name := ""
		if p.match(token.KeywordAs) {
			name = p.expect(token.Identifier, "expected exception binding name after as").Value
		}
		clauseBody := p.parseBlock()
		excepts = append(excepts, ast.ExceptClause{
			Base: p.base(clauseStart, clauseBody.Span().End), Type: typeRef, Name: name, Body: clauseBody,
		})
		end = clauseBody.Span().End
	}
	if p.consumeClauseKeyword(token.KeywordUltimately) {
		ultimately = p.parseBlock()
		end = ultimately.Span().End
	}
	if len(excepts) == 0 && ultimately == nil {
		p.errorSpan(codeInvalidControlSyntax, "attempt requires except or ultimately", body.Span(), "add an except clause, an ultimately clause, or both")
	}
	return &ast.AttemptStmt{
		Base: p.base(start, end), Body: body, Excepts: excepts, Ultimately: ultimately,
	}
}

func (p *parser) parseReturn() ast.Stmt {
	keyword := p.advance()
	if p.check(token.Newline) || p.check(token.RightBrace) || p.atEnd() {
		return &ast.ReturnStmt{Base: ast.Base{Range: keyword.Span}}
	}
	value := p.parseExpression(0)
	return &ast.ReturnStmt{Base: p.base(keyword.Span.Start, value.Span().End), Value: value}
}

func (p *parser) parseToss() ast.Stmt {
	start := p.advance().Span.Start
	if p.check(token.Newline) || p.check(token.RightBrace) || p.atEnd() {
		p.errorCurrent(codeInvalidControlSyntax, "toss requires an Error expression", "provide the Error value to toss")
		bad := &ast.BadExpr{Base: ast.Base{Range: p.current().Span}}
		return &ast.TossStmt{Base: p.base(start, bad.Span().End), Value: bad}
	}
	value := p.parseExpression(0)
	return &ast.TossStmt{Base: p.base(start, value.Span().End), Value: value}
}

func (p *parser) parseBring() ast.Stmt {
	start := p.current().Span.Start
	if p.match(token.KeywordBring) {
		module := p.expect(token.Identifier, "expected module name after bring")
		alias := token.Token{}
		if p.match(token.KeywordAs) {
			alias = p.expect(token.Identifier, "expected alias name after as")
		}
		end := module.Span.End
		if alias.Value != "" {
			end = alias.Span.End
		}
		return &ast.BringStmt{
			Base: p.base(start, end), Module: module.Value, Alias: alias.Value, Namespace: true,
		}
	}

	p.expect(token.KeywordFrom, "expected from")
	module := p.expect(token.Identifier, "expected module name after from")
	p.expect(token.KeywordBring, "expected bring after module name")
	statement := &ast.BringStmt{Module: module.Value}
	if p.match(token.KeywordAll) {
		statement.All = true
		statement.Base = p.base(start, p.previous().Span.End)
		return statement
	}
	if p.match(token.LeftParen) {
		p.skipNewlines()
		for !p.check(token.RightParen) && !p.atEnd() {
			name := p.expect(token.Identifier, "expected imported symbol name")
			if name.Value != "" {
				statement.Names = append(statement.Names, name.Value)
			}
			if p.consumeItemSeparator(token.RightParen) {
				continue
			}
			if !p.check(token.RightParen) {
				p.errorCurrent(codeExpectedSeparator, "expected comma or newline between imported names", "separate same-line names with commas")
			}
		}
		closing := p.expect(token.RightParen, "expected ) after imported names")
		statement.Base = p.base(start, closing.Span.End)
		return statement
	}
	name := p.expect(token.Identifier, "expected imported symbol name or all")
	if name.Value != "" {
		statement.Names = []string{name.Value}
	}
	statement.Base = p.base(start, name.Span.End)
	return statement
}

// Clause keywords may appear immediately after a closing brace or after one or
// more statement newlines. Newlines are restored when the requested clause is
// absent so the outer statement list still owns statement termination.
func (p *parser) consumeClauseKeyword(kind token.Kind) bool {
	saved := p.index
	p.skipNewlines()
	if p.match(kind) {
		return true
	}
	p.index = saved
	return false
}
