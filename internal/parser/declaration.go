package parser

import (
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

func (p *parser) parseSimpleStatement(scope scopeKind) ast.Stmt {
	start := p.current().Span.Start
	expression := p.parseExpression(0)
	if p.match(token.Colon) {
		return p.parseDeclaration(start, expression, scope)
	}
	if isAssignmentOperator(p.current().Kind) {
		operator := p.advance()
		if !expressionCanAssign(expression) {
			p.errorSpan(codeInvalidAssignmentTarget, "expression is not assignable", expression.Span(), "assign only to an identifier, member, or index")
		}
		p.skipNewlines()
		value := p.parseExpression(0)
		return &ast.AssignmentStmt{
			Base: p.base(start, spanEnd(value)), Target: expression,
			Operator: operator.Kind.String(), Value: value,
		}
	}
	if p.match(token.Increment, token.Decrement) {
		operator := p.previous()
		if !expressionCanAssign(expression) {
			p.errorSpan(codeInvalidStandaloneUpdate, "increment/decrement target is not assignable", expression.Span(), "update only an identifier, member, or index")
		}
		return &ast.IncDecStmt{
			Base: p.base(start, operator.Span.End), Target: expression,
			Operator: operator.Kind.String(), Prefix: false,
		}
	}
	return &ast.ExprStmt{Base: p.base(start, spanEnd(expression)), Expression: expression}
}

func (p *parser) parsePrefixUpdate() ast.Stmt {
	operator := p.advance()
	p.skipNewlines()
	target := p.parseExpression(100)
	if !expressionCanAssign(target) {
		p.errorSpan(codeInvalidStandaloneUpdate, "increment/decrement target is not assignable", target.Span(), "update only an identifier, member, or index")
	}
	return &ast.IncDecStmt{
		Base: p.base(operator.Span.Start, spanEnd(target)), Target: target,
		Operator: operator.Kind.String(), Prefix: true,
	}
}

func isAssignmentOperator(kind token.Kind) bool {
	switch kind {
	case token.Assign, token.PlusAssign, token.MinusAssign, token.StarAssign,
		token.SlashAssign, token.PercentAssign, token.CaretAssign:
		return true
	default:
		return false
	}
}

func (p *parser) parseDeclaration(start source.Position, target ast.Expr, scope scopeKind) ast.Stmt {
	name := ""
	if identifier, ok := target.(*ast.IdentifierExpr); ok {
		name = identifier.Name
	} else if !expressionCanAssign(target) {
		p.errorSpan(codeInvalidAssignmentTarget, "declaration target is not a name or member", target.Span(), "declare an identifier or an allowed attribute member")
	}

	modifiers, flavor := p.parseDeclarationModifiers()
	typeRef := p.parseTypeRef()
	if flavor != ast.FunctionBase && typeRef.Name != "Function" {
		p.errorSpan(codeInvalidTypeSyntax, "Overload/Override may modify only Function declarations", typeRef.Span(), "remove the Function modifier or use Function as the declaration type")
	}

	if hasModifier(modifiers, ast.ModifierGlobal) && !p.check(token.Declare) {
		return &ast.VariableDecl{
			Base: p.base(start, typeRef.Span().End), Target: target, Name: name,
			Modifiers: modifiers, Type: typeRef, GlobalOnly: true,
		}
	}

	p.expect(token.Declare, "expected := in declaration")
	p.skipNewlines()

	if name == "structure" && typeRef.Name == "Attributes" && p.check(token.LeftParen) {
		return p.parseStructureDecl(start, scope)
	}
	if typeRef.Name == "Function" && (flavor != ast.FunctionBase || p.looksLikeFunctionSignature()) {
		return p.parseFunctionDecl(start, name, modifiers, flavor, scope)
	}
	if typeRef.Name == "Class" && p.check(token.LeftBrace) {
		return p.parseClassDecl(start, name, modifiers, typeRef, scope)
	}

	initializer := p.parseExpression(0)
	return &ast.VariableDecl{
		Base: p.base(start, spanEnd(initializer)), Target: target, Name: name,
		Modifiers: modifiers, Type: typeRef, Initializer: initializer,
	}
}

func (p *parser) parseDeclarationModifiers() ([]ast.Modifier, ast.FunctionFlavor) {
	var modifiers []ast.Modifier
	flavor := ast.FunctionBase
	for {
		switch p.current().Kind {
		case token.KeywordLocal:
			modifiers = append(modifiers, ast.ModifierLocal)
			p.advance()
		case token.KeywordGlobal:
			modifiers = append(modifiers, ast.ModifierGlobal)
			p.advance()
		case token.KeywordConstant:
			modifiers = append(modifiers, ast.ModifierConstant)
			p.advance()
		case token.KeywordConfidential:
			modifiers = append(modifiers, ast.ModifierConfidential)
			p.advance()
		case token.KeywordOverload:
			flavor = ast.FunctionOverload
			p.advance()
		case token.KeywordOverride:
			flavor = ast.FunctionOverride
			p.advance()
		default:
			return modifiers, flavor
		}
	}
}

func hasModifier(modifiers []ast.Modifier, wanted ast.Modifier) bool {
	for _, modifier := range modifiers {
		if modifier == wanted {
			return true
		}
	}
	return false
}

func (p *parser) parseTypeRef() *ast.TypeRef {
	startToken := p.current()
	name, ok := syntacticTypeName(startToken)
	if !ok {
		p.errorCurrent(codeInvalidTypeSyntax, "expected type name", "use a declared Class name or an AhdCode type")
		if !p.atEnd() {
			p.advance()
		}
		return &ast.TypeRef{Base: ast.Base{Range: startToken.Span}, Name: startToken.Value}
	}
	p.advance()
	typeRef := &ast.TypeRef{Base: ast.Base{Range: startToken.Span}, Name: name}
	if !p.match(token.Less) {
		return typeRef
	}
	p.skipNewlines()
	typeRef.ExplicitEmpty = p.check(token.Greater)
	for !p.check(token.Greater) && !p.atEnd() {
		typeRef.Arguments = append(typeRef.Arguments, p.parseTypeRef())
		if p.match(token.Comma) {
			p.skipNewlines()
			continue
		}
		if p.check(token.Newline) {
			p.skipNewlines()
			continue
		}
		if !p.check(token.Greater) {
			p.errorCurrent(codeExpectedSeparator, "expected comma or newline between generic type arguments", "write Pair<String, Int> on one line")
			// Treat the next type as a recovered item without inventing whitespace separation.
			continue
		}
	}
	closing := p.expect(token.Greater, "expected > after generic type arguments")
	typeRef.Range = source.NewSpan(p.file.ID, startToken.Span.Start, closing.Span.End)
	return typeRef
}

func syntacticTypeName(item token.Token) (string, bool) {
	if item.Kind == token.Identifier {
		return item.Value, true
	}
	switch item.Kind {
	case token.KeywordInt, token.KeywordReal, token.KeywordString, token.KeywordBool,
		token.KeywordNothing, token.KeywordList, token.KeywordPair, token.KeywordFunction,
		token.KeywordClass, token.KeywordAttributes, token.KeywordObject, token.KeywordError:
		return item.Kind.String(), true
	default:
		return "", false
	}
}

func (p *parser) looksLikeFunctionSignature() bool {
	if !p.check(token.LeftParen) {
		return false
	}
	depth := 0
	for index := p.index; index < len(p.tokens); index++ {
		switch p.tokens[index].Kind {
		case token.LeftParen:
			depth++
		case token.RightParen:
			depth--
			if depth == 0 {
				next := p.nextNonNewlineIndex(index + 1)
				return p.tokens[next].Kind == token.Arrow
			}
		case token.EOF:
			return false
		}
	}
	return false
}

func (p *parser) parseFunctionDecl(start source.Position, name string, modifiers []ast.Modifier, flavor ast.FunctionFlavor, scope scopeKind) ast.Stmt {
	if scope == scopeBlock {
		p.errorSpan(codeInvalidDeclarationScope, "Function declarations are not allowed in executable blocks", positionSpan(p.file.ID, start), "move the Function declaration to module root or Class member scope")
	}
	if flavor == ast.FunctionOverride && scope != scopeClass {
		p.errorSpan(codeInvalidDeclarationScope, "Override Function requires Class member scope", positionSpan(p.file.ID, start), "declare Override Function inside a derived Class")
	}
	parameters := p.parseParameterList(false)
	p.skipNewlines()
	p.expect(token.Arrow, "expected -> after Function parameters")
	p.skipNewlines()
	returnType := p.parseTypeRef()
	p.skipNewlines()
	body := p.parseBlock()
	return &ast.FunctionDecl{
		Base: p.base(start, body.Span().End), Name: name, Modifiers: modifiers,
		Flavor: flavor, Parameters: parameters, ReturnType: returnType, Body: body,
	}
}

func (p *parser) parseClassDecl(start source.Position, name string, modifiers []ast.Modifier, typeRef *ast.TypeRef, scope scopeKind) ast.Stmt {
	if scope != scopeModule {
		p.errorSpan(codeInvalidDeclarationScope, "Class declarations are allowed only at module root", positionSpan(p.file.ID, start), "move the Class declaration to module root")
	}
	opening := p.expect(token.LeftBrace, "expected { after Class declaration")
	_ = opening
	p.skipNewlines()
	members := p.parseStatementList(scopeClass, token.RightBrace)
	closing := p.expect(token.RightBrace, "expected } after Class body")
	var parent *ast.TypeRef
	if len(typeRef.Arguments) > 0 {
		parent = typeRef.Arguments[0]
	}
	if len(typeRef.Arguments) > 1 {
		p.errorSpan(codeInvalidTypeSyntax, "Class accepts at most one direct parent", typeRef.Span(), "write Class<Parent> with one parent")
	}
	return &ast.ClassDecl{
		Base: p.base(start, closing.Span.End), Name: name, Modifiers: modifiers, Parent: parent,
		ExplicitRoot: typeRef.ExplicitEmpty, Members: members,
	}
}

func (p *parser) parseStructureDecl(start source.Position, scope scopeKind) ast.Stmt {
	if scope != scopeClass {
		p.errorSpan(codeInvalidDeclarationScope, "structure declarations require Class member scope", positionSpan(p.file.ID, start), "move structure into a Class body")
	}
	parameters := p.parseParameterList(true)
	var body *ast.Block
	end := p.previous().Span.End
	if p.check(token.LeftBrace) {
		body = p.parseBlock()
		end = body.Span().End
	}
	return &ast.StructureDecl{Base: p.base(start, end), Parameters: parameters, Body: body}
}

func (p *parser) parseParameterList(allowInherited bool) []ast.Parameter {
	p.expect(token.LeftParen, "expected ( before parameters")
	p.skipNewlines()
	var parameters []ast.Parameter
	seenDefault := false
	for !p.check(token.RightParen) && !p.atEnd() {
		start := p.current().Span.Start
		if allowInherited && p.check(token.Identifier) && p.current().Value == "SuperClass" && p.peek(1).Kind == token.Dot && p.peek(2).Kind == token.Identifier && p.peek(2).Value == "attributes" {
			p.advance()
			p.advance()
			end := p.advance().Span.End
			parameters = append(parameters, ast.Parameter{Base: p.base(start, end), InheritedAttributes: true})
		} else {
			name := p.expect(token.Identifier, "expected parameter name")
			p.expect(token.Colon, "expected : after parameter name")
			modifiers, flavor := p.parseDeclarationModifiers()
			if flavor != ast.FunctionBase {
				p.errorCurrent(codeInvalidTypeSyntax, "Overload/Override are not parameter modifiers", "remove the Function declaration modifier")
			}
			typeRef := p.parseTypeRef()
			var defaultValue ast.Expr
			if p.match(token.Declare) {
				p.skipNewlines()
				defaultValue = p.parseExpression(0)
				seenDefault = true
			} else if seenDefault {
				p.errorSpan(codeInvalidTypeSyntax, "required parameter cannot follow a default parameter", name.Span, "move required parameters before default parameters")
			}
			end := typeRef.Span().End
			if defaultValue != nil {
				end = defaultValue.Span().End
			}
			parameters = append(parameters, ast.Parameter{
				Base: p.base(start, end), Name: name.Value, Modifiers: modifiers,
				Type: typeRef, Default: defaultValue,
			})
		}
		if p.consumeItemSeparator(token.RightParen) {
			continue
		}
		if !p.check(token.RightParen) {
			p.errorCurrent(codeExpectedSeparator, "expected comma or newline between parameters", "separate same-line parameters with commas")
		}
	}
	p.expect(token.RightParen, "expected ) after parameters")
	return parameters
}
