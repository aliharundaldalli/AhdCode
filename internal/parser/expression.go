package parser

import (
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

const (
	bpOr       = 10
	bpAnd      = 20
	bpNot      = 30
	bpEqual    = 40
	bpCompare  = 50
	bpAdd      = 60
	bpMultiply = 70
	bpUnary    = 80
	bpPower    = 90
)

func (p *parser) parseExpression(minBindingPower int) ast.Expr {
	left := p.parsePrefix()
	for {
		if p.check(token.Newline) {
			break
		}
		switch p.current().Kind {
		case token.LeftParen:
			if 100 < minBindingPower {
				return left
			}
			left = p.parseCall(left)
			continue
		case token.Dot:
			if 100 < minBindingPower {
				return left
			}
			left = p.parseMember(left)
			continue
		case token.LeftBracket:
			if 100 < minBindingPower {
				return left
			}
			left = p.parseIndexOrSlice(left)
			continue
		}

		operator, leftBP, rightBP, width := p.infixOperator()
		if width == 0 || leftBP < minBindingPower {
			break
		}
		for range width {
			p.advance()
		}
		p.skipNewlines()
		right := p.parseRequiredExpression(rightBP, "missing right operand after '"+operator+"'", "write an expression after the operator")
		left = &ast.BinaryExpr{
			Base: p.base(spanStart(left), spanEnd(right)), Left: left,
			Operator: operator, Right: right,
		}
	}
	return left
}

func (p *parser) parseRequiredExpression(minBindingPower int, message, hint string) ast.Expr {
	if p.atEnd() || p.check(token.RightParen) || p.check(token.RightBracket) || p.check(token.RightBrace) || p.check(token.InterpolationEnd) {
		item := p.current()
		p.errorCurrent(codeUnexpectedToken, message, hint)
		return &ast.BadExpr{Base: ast.Base{Range: item.Span}}
	}
	return p.parseExpression(minBindingPower)
}

func (p *parser) parsePrefix() ast.Expr {
	item := p.current()
	switch item.Kind {
	case token.IntLiteral:
		p.advance()
		return &ast.LiteralExpr{Base: ast.Base{Range: item.Span}, Kind: ast.IntLiteral, Raw: item.Lexeme, Value: item.Value}
	case token.RealLiteral:
		p.advance()
		return &ast.LiteralExpr{Base: ast.Base{Range: item.Span}, Kind: ast.RealLiteral, Raw: item.Lexeme, Value: item.Value}
	case token.KeywordTrue, token.KeywordFalse:
		p.advance()
		return &ast.LiteralExpr{Base: ast.Base{Range: item.Span}, Kind: ast.BoolLiteral, Raw: item.Lexeme, Value: item.Value}
	case token.KeywordNull:
		p.advance()
		return &ast.LiteralExpr{Base: ast.Base{Range: item.Span}, Kind: ast.NullLiteral, Raw: item.Lexeme, Value: "null"}
	case token.KeywordLambda:
		return p.parseLambda()
	case token.Identifier:
		p.advance()
		return &ast.IdentifierExpr{Base: ast.Base{Range: item.Span}, Name: item.Value, Raw: item.Lexeme}
	case token.KeywordObject, token.KeywordError:
		// The built-in Class names are usable wherever a declared Class name is,
		// including construction and type membership.
		p.advance()
		return &ast.IdentifierExpr{Base: ast.Base{Range: item.Span}, Name: item.Kind.String(), Raw: item.Lexeme}
	case token.StringStart:
		return p.parseString()
	case token.LeftParen:
		start := p.advance().Span.Start
		p.skipNewlines()
		if p.atEnd() {
			p.errorCurrent(codeExpectedToken, "expected an expression and ) to close the grouped expression", "write an expression followed by )")
			bad := &ast.BadExpr{Base: ast.Base{Range: p.current().Span}}
			return &ast.GroupExpr{Base: p.base(start, p.current().Span.End), Expression: bad}
		}
		expression := p.parseExpression(0)
		p.skipNewlines()
		closing := p.expect(token.RightParen, "expected ) to close the grouped expression")
		return &ast.GroupExpr{Base: p.base(start, closing.Span.End), Expression: expression}
	case token.LeftBracket:
		return p.parseList()
	case token.LeftBrace:
		return p.parsePair()
	case token.Plus, token.Minus:
		start := p.advance()
		p.skipNewlines()
		operand := p.parseExpression(bpUnary)
		return &ast.UnaryExpr{Base: p.base(start.Span.Start, spanEnd(operand)), Operator: start.Kind.String(), Operand: operand}
	case token.KeywordNot:
		start := p.advance()
		p.skipNewlines()
		operand := p.parseExpression(bpNot)
		return &ast.UnaryExpr{Base: p.base(start.Span.Start, spanEnd(operand)), Operator: "not", Operand: operand}
	default:
		p.errorCurrent(codeUnexpectedToken, "expected expression", "start an expression with a literal, identifier, grouping, or collection")
		if !p.atEnd() && !p.check(token.RightBrace) && !p.check(token.RightParen) && !p.check(token.RightBracket) && !p.check(token.InterpolationEnd) {
			p.advance()
		}
		return &ast.BadExpr{Base: ast.Base{Range: item.Span}}
	}
}

func (p *parser) parseLambda() ast.Expr {
	start := p.advance().Span.Start
	captures := p.parseCaptureList()
	parameters := p.parseParameterList(false)
	p.skipNewlines()
	p.expect(token.Arrow, "expected -> after lambda parameters")
	p.skipNewlines()
	if p.check(token.LeftBrace) {
		p.errorCurrent(codeInvalidLambdaSyntax, "lambda body must be one expression, not a block", "use lambda (...) -> expression, or write a normal Function for statements")
		block := p.parseBlock()
		return &ast.BadExpr{Base: p.base(start, block.Span().End)}
	}
	body := p.parseRequiredExpression(0, "missing lambda body expression after '->'", "write one expression after '->', or use a normal Function for statements")
	return &ast.LambdaExpr{Base: p.base(start, spanEnd(body)), Captures: captures, Parameters: parameters, Body: body}
}

// parseCaptureList reads the optional `[name, name]` capture list that may
// follow `lambda`. The list is names only: a capture reads an existing
// binding, so there is no type, no modifier, and no initializer to write.
func (p *parser) parseCaptureList() []ast.CaptureRef {
	if !p.check(token.LeftBracket) {
		return nil
	}
	p.advance()
	var captures []ast.CaptureRef
	p.skipNewlines()
	for !p.check(token.RightBracket) && !p.atEnd() {
		name := p.current()
		if name.Kind != token.Identifier {
			p.errorCurrent(codeInvalidLambdaSyntax, "lambda capture list expects a binding name",
				"list the names the lambda reads, as in lambda [minimum] (value: Int) -> value >= minimum")
			break
		}
		p.advance()
		captures = append(captures, ast.CaptureRef{Base: p.base(name.Span.Start, name.Span.End), Name: name.Value})
		p.skipNewlines()
		if p.check(token.Comma) {
			p.advance()
			p.skipNewlines()
			continue
		}
		break
	}
	p.skipNewlines()
	p.expect(token.RightBracket, "expected ] to close the lambda capture list")
	return captures
}

func (p *parser) infixOperator() (operator string, leftBP, rightBP, width int) {
	kind := p.current().Kind
	switch kind {
	case token.Caret:
		return "^", bpPower, bpPower - 1, 1
	case token.Star, token.Slash, token.Percent:
		return kind.String(), bpMultiply, bpMultiply + 1, 1
	case token.Plus, token.Minus:
		return kind.String(), bpAdd, bpAdd + 1, 1
	case token.Less, token.LessEqual, token.Greater, token.GreaterEqual:
		return kind.String(), bpCompare, bpCompare + 1, 1
	case token.Equal, token.NotEqual, token.KeywordSame, token.KeywordIn:
		return kind.String(), bpEqual, bpEqual + 1, 1
	case token.KeywordIs:
		if p.peek(1).Kind == token.KeywordNot {
			return "is not", bpEqual, bpEqual + 1, 2
		}
		return "is", bpEqual, bpEqual + 1, 1
	case token.KeywordHas:
		if p.peek(1).Kind == token.KeywordNot {
			return "has not", bpEqual, bpEqual + 1, 2
		}
		return "has", bpEqual, bpEqual + 1, 1
	case token.KeywordNot:
		if p.peek(1).Kind == token.KeywordIn {
			return "not in", bpEqual, bpEqual + 1, 2
		}
	case token.KeywordAnd:
		return "and", bpAnd, bpAnd + 1, 1
	case token.KeywordOr:
		return "or", bpOr, bpOr + 1, 1
	}
	return "", 0, 0, 0
}

func (p *parser) parseCall(callee ast.Expr) ast.Expr {
	start := spanStart(callee)
	p.expect(token.LeftParen, "expected ( to start call arguments")
	p.skipNewlines()
	var arguments []ast.CallArgument
	argumentStyle := -1 // 0 positional, 1 named
	for !p.check(token.RightParen) && !p.atEnd() {
		argumentStart := p.current().Span.Start
		name := ""
		style := 0
		if p.check(token.Identifier) && p.peek(1).Kind == token.Colon {
			name = p.advance().Value
			p.advance()
			style = 1
		}
		if argumentStyle == -1 {
			argumentStyle = style
		} else if style != argumentStyle {
			p.errorCurrent(codeMixedCallArguments, "positional and named arguments cannot be mixed", "make every argument positional or every argument named")
		}
		p.skipNewlines()
		value := p.parseExpression(0)
		arguments = append(arguments, ast.CallArgument{
			Base: p.base(argumentStart, spanEnd(value)), Name: name, Value: value,
		})
		if p.consumeItemSeparator(token.RightParen) {
			continue
		}
		if !p.check(token.RightParen) {
			p.errorCurrent(codeExpectedSeparator, "expected comma or newline between call arguments", "add a comma on the same line or place the next argument on a new line")
		}
	}
	closing := p.expect(token.RightParen, "expected ) to close the call argument list")
	return &ast.CallExpr{Base: p.base(start, closing.Span.End), Callee: callee, Arguments: arguments}
}

func (p *parser) parseMember(object ast.Expr) ast.Expr {
	p.advance()
	name := p.expect(token.Identifier, "expected member name after .")
	return &ast.MemberExpr{Base: p.base(spanStart(object), name.Span.End), Object: object, Name: name.Value}
}

func (p *parser) parseIndexOrSlice(object ast.Expr) ast.Expr {
	p.advance()
	p.skipNewlines()
	var first ast.Expr
	if p.atEnd() {
		p.errorCurrent(codeExpectedToken, "expected an index expression and ] to close the index", "write an index followed by ]")
		first = &ast.BadExpr{Base: ast.Base{Range: p.current().Span}}
		return &ast.IndexExpr{Base: p.base(spanStart(object), p.current().Span.End), Object: object, Index: first}
	}
	if !p.check(token.Colon) && !p.check(token.RightBracket) {
		first = p.parseExpression(0)
		p.skipNewlines()
	}
	if p.match(token.Colon) {
		p.skipNewlines()
		var end ast.Expr
		if !p.check(token.RightBracket) {
			end = p.parseExpression(0)
			p.skipNewlines()
		}
		closing := p.expect(token.RightBracket, "expected ] after slice")
		return &ast.SliceExpr{Base: p.base(spanStart(object), closing.Span.End), Object: object, Start: first, End: end}
	}
	closing := p.expect(token.RightBracket, "expected ] to close the index")
	if first == nil {
		p.errorSpan(codeUnexpectedToken, "empty index is not allowed", closing.Span, "provide an index or use : for a slice")
		first = &ast.BadExpr{Base: ast.Base{Range: closing.Span}}
	}
	return &ast.IndexExpr{Base: p.base(spanStart(object), closing.Span.End), Object: object, Index: first}
}

func (p *parser) parseList() ast.Expr {
	start := p.advance().Span.Start
	p.skipNewlines()
	var elements []ast.Expr
	for !p.check(token.RightBracket) && !p.atEnd() {
		elements = append(elements, p.parseExpression(0))
		if p.consumeItemSeparator(token.RightBracket) {
			continue
		}
		if !p.check(token.RightBracket) {
			p.errorCurrent(codeExpectedSeparator, "expected comma or newline between List elements", "separate same-line elements with commas")
		}
	}
	closing := p.expect(token.RightBracket, "expected ] to close the List literal")
	return &ast.ListExpr{Base: p.base(start, closing.Span.End), Elements: elements}
}

func (p *parser) parsePair() ast.Expr {
	start := p.advance().Span.Start
	p.skipNewlines()
	var entries []ast.PairEntry
	for !p.check(token.RightBrace) && !p.atEnd() {
		entryStart := p.current().Span.Start
		key := p.parseExpression(0)
		p.expect(token.Colon, "expected : between Pair key and value")
		p.skipNewlines()
		value := p.parseExpression(0)
		entries = append(entries, ast.PairEntry{Base: p.base(entryStart, spanEnd(value)), Key: key, Value: value})
		if p.consumeItemSeparator(token.RightBrace) {
			continue
		}
		if !p.check(token.RightBrace) {
			p.errorCurrent(codeExpectedSeparator, "expected comma or newline between Pair entries", "separate same-line entries with commas")
		}
	}
	closing := p.expect(token.RightBrace, "expected } to close the Pair literal")
	return &ast.PairExpr{Base: p.base(start, closing.Span.End), Entries: entries}
}

func (p *parser) consumeItemSeparator(closing token.Kind) bool {
	if p.match(token.Comma) {
		p.skipNewlines()
		return true
	}
	if p.check(token.Newline) {
		p.skipNewlines()
		return true
	}
	return p.check(closing)
}

func (p *parser) parseString() ast.Expr {
	opening := p.advance()
	parts := make([]ast.StringPart, 0, 1)
	for !p.check(token.StringEnd) && !p.atEnd() {
		if p.check(token.StringText) {
			text := p.advance()
			parts = append(parts, ast.StringPart{Base: ast.Base{Range: text.Span}, Text: text.Value})
			continue
		}
		if p.check(token.InterpolationStart) {
			start := p.advance().Span.Start
			p.skipNewlines()
			var expression ast.Expr
			if p.check(token.InterpolationEnd) {
				p.errorCurrent(codeUnexpectedToken, "empty interpolation expression", "place one expression between the braces")
				expression = &ast.BadExpr{Base: ast.Base{Range: p.current().Span}}
			} else {
				expression = p.parseExpression(0)
			}
			p.skipNewlines()
			closing := p.expect(token.InterpolationEnd, "expected } after interpolation expression")
			parts = append(parts, ast.StringPart{Base: p.base(start, closing.Span.End), Expression: expression})
			continue
		}
		p.errorCurrent(codeUnexpectedToken, "unexpected token in string", "use text or a valid interpolation expression")
		p.advance()
	}
	closing := p.expect(token.StringEnd, "expected matching string delimiter")
	return &ast.StringExpr{Base: p.base(opening.Span.Start, closing.Span.End), Delimiter: opening.Lexeme, Parts: parts}
}

func expressionCanAssign(expression ast.Expr) bool {
	switch expression.(type) {
	case *ast.IdentifierExpr, *ast.MemberExpr, *ast.IndexExpr:
		return true
	default:
		return false
	}
}

func positionSpan(fileID source.FileID, position source.Position) source.Span {
	return source.NewSpan(fileID, position, position)
}
