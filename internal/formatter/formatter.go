// Package formatter implements the canonical AhdCode source formatter.
//
// AhdCode's grammar accepts flexible spelling: commas between items are
// optional wherever a newline can disambiguate them, trailing commas are
// always optional, and indentation carries no structural meaning. This
// package is the other half of "strict semantics, flexible spelling,
// canonical formatting": it takes any validly parsed source and renders the
// single recommended AhdCode style from the AST, independent of how the
// input happened to be spelled.
//
// Rendering works by lowering the AST into a small Wadler/Lindig-style
// pretty-printing document (see doc.go) and laying that document out against
// an 80-column page. A bracketed, comma-flexible construct (call arguments,
// List literals, Pair literals, and Function/structure parameter lists)
// collapses onto one line, comma-separated, when it fits; otherwise it
// breaks to one item per line with no separator and no trailing comma. Every
// other line break is structural (statement boundaries, block braces).
//
// Comments and exact string/interpolation spelling survive formatting: a
// shared token cursor walks the original token stream in lockstep with the
// AST traversal, so trivia attached to any consumed or skipped token is
// re-emitted at the right point. A comment appearing between the items of a
// bracketed construct forces that construct to render broken, since a line
// comment cannot otherwise be placed without corrupting the layout.
package formatter

import (
	"strings"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/lexer"
	"ahdcode/internal/parser"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

type Result struct {
	Text        string
	Diagnostics []diagnostics.Diagnostic
}

func (result Result) HasErrors() bool {
	for _, item := range result.Diagnostics {
		if item.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}

// Format validates and formats one source file. Invalid input is never
// partially rewritten.
func Format(file source.File) Result {
	lexed := lexer.Lex(file)
	parsed := parser.Parse(file, lexed.Tokens)
	diags := append(append([]diagnostics.Diagnostic(nil), lexed.Diagnostics...), parsed.Diagnostics...)
	result := Result{Diagnostics: diags}
	if result.HasErrors() || parsed.Program == nil {
		return result
	}
	b := &builder{file: file, tokens: parsed.Tokens}
	root := b.program(parsed.Program)
	rendered := strings.TrimRight(render(root, maxLineWidth), " \t\r\n")
	if rendered == "" {
		return result
	}
	result.Text = rendered + "\n"
	return result
}

// builder walks the AST and the original token stream together: every
// render method consumes exactly the tokens the parser would have consumed
// for the node it is given, which keeps the shared cursor in lockstep with
// the AST so trivia (comments) attached to any token is found at the right
// place regardless of how the formatter chooses to lay the construct out.
type builder struct {
	file   source.File
	tokens []token.Token
	cursor int
	// openComment is true immediately after rendering a block-comment chunk
	// that has not yet reached its closing "*/". A multiline block comment
	// is split by the lexer into one trivia chunk per physical line (see
	// lexer.scanBlockComment); every chunk after the first already carries
	// its own original leading whitespace, so it must be rejoined with a
	// bare newline (rawNewline) rather than canonical indentation.
	openComment bool
}

func (b *builder) current() token.Token {
	if b.cursor >= len(b.tokens) {
		return b.tokens[len(b.tokens)-1]
	}
	return b.tokens[b.cursor]
}

func (b *builder) advance() token.Token {
	current := b.current()
	if b.cursor < len(b.tokens)-1 {
		b.cursor++
	}
	return current
}

// renderTrivia renders one comment trivia entry and reports whether it was
// a block-comment chunk continuing a still-open multiline comment (see
// builder.openComment).
func (b *builder) renderTrivia(trivia token.Trivia) (doc, bool) {
	switch trivia.Kind {
	case token.LineCommentTrivia:
		b.openComment = false
		return text(strings.TrimRight(trivia.Lexeme, " \t\r")), false
	case token.BlockCommentTrivia:
		continuation := b.openComment && !strings.HasPrefix(trivia.Lexeme, "/*")
		b.openComment = !strings.HasSuffix(strings.TrimRight(trivia.Lexeme, " \t\r"), "*/")
		if continuation {
			return concat(rawNewline(), text(trivia.Lexeme)), true
		}
		return text(trivia.Lexeme), false
	default:
		return concat(), false
	}
}

// leadingTriviaDoc renders the current token's leading trivia (comments)
// without advancing the cursor. A block comment stays inline; a line
// comment (which, lexically, only ever attaches to a Newline token) forces
// a following hard break.
func (b *builder) leadingTriviaDoc() doc {
	var parts []doc
	for _, trivia := range b.current().LeadingTrivia {
		switch trivia.Kind {
		case token.LineCommentTrivia:
			rendered, _ := b.renderTrivia(trivia)
			parts = append(parts, rendered)
		case token.BlockCommentTrivia:
			rendered, _ := b.renderTrivia(trivia)
			parts = append(parts, rendered, text(" "))
		}
	}
	return concat(parts...)
}

// leaf consumes exactly one token, preceded by whatever leading trivia it
// carries, and renders its original spelling.
func (b *builder) leaf() doc {
	trivia := b.leadingTriviaDoc()
	consumed := b.advance()
	return concat(trivia, text(consumed.Lexeme))
}

func isEmptyDoc(d doc) bool {
	return (d.kind == dConcat && len(d.items) == 0) || (d.kind == dText && d.text == "")
}

// gap drains a run of Newline/Comma separator tokens starting at the
// cursor -- tokens that carry no AST node of their own -- collecting any
// comment trivia they carry along the way. It stops as soon as the current
// token is neither a Newline nor a Comma, leaving the cursor there so the
// caller's own leaf/expr consumption picks up that token's trivia normally.
func (b *builder) gap() doc {
	inline, standalone, _ := b.gapParts()
	switch {
	case isEmptyDoc(inline):
		return standalone
	case isEmptyDoc(standalone):
		return inline
	default:
		return concat(inline, hardline(), standalone)
	}
}

// gapParts is gap's finer-grained form: inline is a comment trailing the
// content immediately before this gap on the same physical source line (so
// it belongs right after that content with a single space, not a hard
// break); standalone is every comment after that, each already preceded by
// a hard break where more than one appears; blankBeforeNext reports whether
// the source had a blank line right before whatever follows the gap (a
// statement, or a closing brace), so a caller separating statements can
// preserve up to one blank line the way the rest of this package preserves
// comments -- readability grouping the author already expressed, not
// structure the parser needs.
func (b *builder) gapParts() (inline doc, standalone doc, blankBeforeNext bool) {
	var standaloneParts []doc
	haveInline := false
	seenNewline := false
	blankRun := 0
	for b.current().Kind == token.Newline || b.current().Kind == token.Comma {
		current := b.current()
		hadComment := false
		for _, trivia := range current.LeadingTrivia {
			if trivia.Kind != token.LineCommentTrivia && trivia.Kind != token.BlockCommentTrivia {
				continue
			}
			hadComment = true
			rendered, continuation := b.renderTrivia(trivia)
			if !seenNewline && !haveInline {
				inline = rendered
				haveInline = true
				continue
			}
			if len(standaloneParts) > 0 && !continuation {
				standaloneParts = append(standaloneParts, hardline())
			}
			standaloneParts = append(standaloneParts, rendered)
		}
		if current.Kind == token.Newline {
			seenNewline = true
			if hadComment {
				blankRun = 0
			} else {
				blankRun++
			}
		}
		b.advance()
	}
	return inline, concat(standaloneParts...), blankRun >= 2
}

// withGapSeparator renders the mandatory hard break between two sibling
// constructs (statements, except clauses, if/else branches, ...), folding
// in any comments a drained gap collected between them. When blank is true,
// one blank line is preserved right before whatever follows: a rawNewline
// (bare "\n", no indent padding) ends the current line so the blank line it
// creates carries no trailing whitespace, and the hardline right after it
// supplies the indentation for what comes next.
func withGapSeparator(gapDoc doc, blank bool) doc {
	trailer := hardline()
	if blank {
		trailer = concat(rawNewline(), hardline())
	}
	if isEmptyDoc(gapDoc) {
		return trailer
	}
	return concat(hardline(), gapDoc, trailer)
}

// stmtSeq renders a list of statements separated by hard breaks, including
// any comments found between them, but is not itself responsible for the
// surrounding braces/indentation (see bracedStmts).
func (b *builder) stmtSeq(statements []ast.Stmt) []doc {
	var parts []doc
	if lead := b.gap(); !isEmptyDoc(lead) {
		parts = append(parts, lead, hardline())
	}
	for index, statement := range statements {
		parts = append(parts, b.stmt(statement))
		inline, standalone, blank := b.gapParts()
		if !isEmptyDoc(inline) {
			parts = append(parts, text(" "), inline)
		}
		if index == len(statements)-1 {
			if !isEmptyDoc(standalone) {
				parts = append(parts, hardline(), standalone)
			}
			continue
		}
		parts = append(parts, withGapSeparator(standalone, blank))
	}
	return parts
}

// bracedStmts renders `{ ... }` around a statement list. The opening brace
// must be the current token.
func (b *builder) bracedStmts(statements []ast.Stmt) doc {
	open := b.leaf()
	if len(statements) == 0 {
		inner := b.gap()
		close_ := b.leaf()
		if isEmptyDoc(inner) {
			return concat(open, close_)
		}
		return concat(open, indent(concat(hardline(), inner)), hardline(), close_)
	}
	body := b.stmtSeq(statements)
	close_ := b.leaf()
	return concat(open, indent(concat(append([]doc{hardline()}, body...)...)), hardline(), close_)
}

func (b *builder) block(blk *ast.Block) doc { return b.bracedStmts(blk.Statements) }

func (b *builder) program(p *ast.Program) doc {
	return concat(b.stmtSeq(p.Statements)...)
}

// declarationModifiers consumes the Local/Global/Constant/Confidential and
// Overload/Override modifier keywords in whatever order the source used,
// but always renders them in that canonical order (scope/visibility first,
// then Function flavor), mirroring parser.parseDeclarationModifiers'
// acceptance of any interleaving.
func (b *builder) declarationModifiers() doc {
	var scope []doc
	var flavor doc
	hasFlavor := false
	for {
		switch b.current().Kind {
		case token.KeywordLocal, token.KeywordGlobal, token.KeywordConstant, token.KeywordConfidential:
			scope = append(scope, b.leaf(), text(" "))
		case token.KeywordOverload, token.KeywordOverride:
			flavor = concat(b.leaf(), text(" "))
			hasFlavor = true
		default:
			if hasFlavor {
				return concat(append(scope, flavor)...)
			}
			return concat(scope...)
		}
	}
}

// typeRef renders a type reference. Generic type arguments (List<T>,
// Pair<K, V>, Class<Parent>) are outside the flexible-comma/canonical-break
// rules the rest of this package implements, so they always render flat.
func (b *builder) typeRef(t *ast.TypeRef) doc {
	name := b.leaf()
	if b.current().Kind != token.Less {
		return concat(name, b.nullableSuffix())
	}
	open := b.leaf()
	skipGapSilently(b)
	var parts []doc
	for index, argument := range t.Arguments {
		if index > 0 {
			parts = append(parts, text(", "))
		}
		parts = append(parts, b.typeRef(argument))
		if b.current().Kind == token.Comma {
			b.advance()
		}
		skipGapSilently(b)
	}
	close_ := b.leaf()
	return concat(name, open, concat(parts...), close_, b.nullableSuffix())
}

// nullableSuffix consumes a trailing `?` marking the type just rendered as
// nullable, if present.
func (b *builder) nullableSuffix() doc {
	if b.current().Kind != token.Question {
		return concat()
	}
	return b.leaf()
}

// skipGapSilently advances past stray newlines with no rendering
// obligation. Generic type argument lists are never broken across lines by
// the formatter, so a comment placed inside one (a vanishingly rare and
// discouraged pattern) is dropped rather than forcing this package to grow
// a second line-breaking construct for that one case.
func skipGapSilently(b *builder) {
	for b.current().Kind == token.Newline {
		b.advance()
	}
}

// delimitedGroup renders a bracketed, comma/newline-flexible construct:
// call arguments, List elements, Pair entries, or a Function/structure
// parameter list. It collapses to one line, comma-separated, when that fits
// within maxLineWidth (accounting for whatever precedes it and whatever
// immediately follows up to the next unavoidable break); otherwise it
// breaks to one item per line with no comma at all, matching the canonical
// style. A comment found between items always forces the broken form.
func (b *builder) delimitedGroup(itemCount int, renderItem func(index int) doc) doc {
	open := b.leaf()
	var body []doc
	forced := false
	if lead := b.gap(); !isEmptyDoc(lead) {
		forced = true
		body = append(body, lead, hardline())
	}
	for index := 0; index < itemCount; index++ {
		if index > 0 {
			body = append(body, ifBreak(hardline(), text(", ")))
		}
		body = append(body, renderItem(index))
		if trailing := b.gap(); !isEmptyDoc(trailing) {
			forced = true
			body = append(body, hardline(), trailing)
		}
	}
	close_ := b.leaf()
	inner := concat(
		open,
		indent(concat(append([]doc{ifBreak(hardline(), text(""))}, body...)...)),
		ifBreak(hardline(), text("")),
		close_,
	)
	if forced {
		return breakGroup(inner)
	}
	return group(inner)
}

func (b *builder) callArgsGroup(arguments []ast.CallArgument) doc {
	return b.delimitedGroup(len(arguments), func(index int) doc {
		argument := arguments[index]
		if argument.Name == "" {
			return b.expr(argument.Value)
		}
		name := b.leaf()
		colon := b.leaf()
		return concat(name, colon, text(" "), b.expr(argument.Value))
	})
}

func (b *builder) listGroup(elements []ast.Expr) doc {
	return b.delimitedGroup(len(elements), func(index int) doc { return b.expr(elements[index]) })
}

func (b *builder) pairGroup(entries []ast.PairEntry) doc {
	return b.delimitedGroup(len(entries), func(index int) doc {
		entry := entries[index]
		key := b.expr(entry.Key)
		colon := b.leaf()
		return concat(key, colon, text(" "), b.expr(entry.Value))
	})
}

// captureGroup renders a lambda's explicit dependency list. Each entry names
// its kind either compactly (`#name`/`@name`) or in full (`Local name`/
// `Global name`); the formatter preserves whichever spelling the source used
// rather than rewriting one into the other, since both are canonical.
func (b *builder) captureGroup(captures []ast.CaptureRef) doc {
	return b.delimitedGroup(len(captures), func(int) doc {
		if b.current().Kind == token.KeywordLocal || b.current().Kind == token.KeywordGlobal {
			keyword := b.leaf()
			return concat(keyword, text(" "), b.leaf())
		}
		return concat(b.leaf(), b.leaf())
	})
}

func (b *builder) parameterGroup(parameters []ast.Parameter) doc {
	return b.delimitedGroup(len(parameters), func(index int) doc {
		parameter := parameters[index]
		if parameter.InheritedAttributes {
			object := b.leaf()
			dot := b.leaf()
			member := b.leaf()
			return concat(object, dot, member)
		}
		name := b.leaf()
		colon := b.leaf()
		modifiers := b.declarationModifiers()
		typeDoc := b.typeRef(parameter.Type)
		result := concat(name, colon, text(" "), modifiers, typeDoc)
		if parameter.Default == nil {
			return result
		}
		declare := b.leaf()
		return concat(result, text(" "), declare, text(" "), b.expr(parameter.Default))
	})
}

// declHead renders the "target: [modifiers] Type" prefix shared by
// variable, Function, Class, and structure declarations.
func (b *builder) declHead(targetDoc doc, typeRef *ast.TypeRef) doc {
	colon := b.leaf()
	modifiers := b.declarationModifiers()
	return concat(targetDoc, colon, text(" "), modifiers, b.typeRef(typeRef))
}

func (b *builder) variableDecl(v *ast.VariableDecl) doc {
	target := b.expr(v.Target)
	head := target
	inferredAfterModifier := false
	if b.current().Kind == token.Colon {
		colon := b.leaf()
		modifiers := b.declarationModifiers()
		head = concat(target, colon, text(" "), modifiers)
		if v.Type != nil {
			head = concat(head, b.typeRef(v.Type))
		} else {
			inferredAfterModifier = true
		}
	}
	if v.GlobalOnly {
		return head
	}
	declare := b.leaf()
	if inferredAfterModifier {
		return concat(head, declare, text(" "), b.expr(v.Initializer))
	}
	return concat(head, text(" "), declare, text(" "), b.expr(v.Initializer))
}

func (b *builder) functionDecl(f *ast.FunctionDecl) doc {
	head := b.declHead(b.leaf(), &ast.TypeRef{})
	declare := b.leaf()
	parameters := b.parameterGroup(f.Parameters)
	arrow := b.leaf()
	returnType := b.typeRef(f.ReturnType)
	body := b.block(f.Body)
	return concat(head, text(" "), declare, text(" "), parameters, text(" "), arrow, text(" "), returnType, text(" "), body)
}

func (b *builder) classDecl(c *ast.ClassDecl) doc {
	var arguments []*ast.TypeRef
	if c.Parent != nil {
		arguments = []*ast.TypeRef{c.Parent}
	}
	classType := &ast.TypeRef{Arguments: arguments, ExplicitEmpty: c.ExplicitRoot}
	head := b.declHead(b.leaf(), classType)
	declare := b.leaf()
	body := b.bracedStmts(c.Members)
	return concat(head, text(" "), declare, text(" "), body)
}

func (b *builder) structureDecl(s *ast.StructureDecl) doc {
	head := b.declHead(b.leaf(), &ast.TypeRef{})
	declare := b.leaf()
	parameters := b.parameterGroup(s.Parameters)
	if s.Body == nil {
		return concat(head, text(" "), declare, text(" "), parameters)
	}
	return concat(head, text(" "), declare, text(" "), parameters, text(" "), b.block(s.Body))
}

func (b *builder) stateBody(conditions []ast.StateCondition) doc {
	open := b.leaf()
	if len(conditions) == 0 {
		inner := b.gap()
		close_ := b.leaf()
		if isEmptyDoc(inner) {
			return concat(open, close_)
		}
		return concat(open, indent(concat(hardline(), inner)), hardline(), close_)
	}
	var body []doc
	if lead := b.gap(); !isEmptyDoc(lead) {
		body = append(body, lead, hardline())
	}
	for index, condition := range conditions {
		keyword := b.leaf()
		var matchDoc doc
		if condition.Default {
			matchDoc = b.leaf()
		} else {
			matchDoc = b.expr(condition.Match)
		}
		body = append(body, keyword, text(" "), matchDoc, text(" "), b.block(condition.Body))
		trailingGap := b.gap()
		if index == len(conditions)-1 {
			if !isEmptyDoc(trailingGap) {
				body = append(body, hardline(), trailingGap)
			}
			continue
		}
		body = append(body, withGapSeparator(trailingGap, false))
	}
	close_ := b.leaf()
	return concat(open, indent(concat(append([]doc{hardline()}, body...)...)), hardline(), close_)
}

func (b *builder) stmt(s ast.Stmt) doc {
	switch v := s.(type) {
	case *ast.ExprStmt:
		return b.expr(v.Expression)
	case *ast.VariableDecl:
		return b.variableDecl(v)
	case *ast.AssignmentStmt:
		target := b.expr(v.Target)
		operator := b.leaf()
		return concat(target, text(" "), operator, text(" "), b.expr(v.Value))
	case *ast.IncDecStmt:
		if v.Prefix {
			operator := b.leaf()
			return concat(operator, b.expr(v.Target))
		}
		target := b.expr(v.Target)
		return concat(target, b.leaf())
	case *ast.ReturnStmt:
		keyword := b.leaf()
		if v.Value == nil {
			return keyword
		}
		return concat(keyword, text(" "), b.expr(v.Value))
	case *ast.TossStmt:
		keyword := b.leaf()
		return concat(keyword, text(" "), b.expr(v.Value))
	case *ast.BreakStmt:
		return b.leaf()
	case *ast.ContinueStmt:
		return b.leaf()
	case *ast.IfStmt:
		return b.ifStmt(v)
	case *ast.WhileStmt:
		keyword := b.leaf()
		condition := b.expr(v.Condition)
		return concat(keyword, text(" "), condition, text(" "), b.block(v.Body))
	case *ast.UntilStmt:
		keyword := b.leaf()
		condition := b.expr(v.Condition)
		return concat(keyword, text(" "), condition, text(" "), b.block(v.Body))
	case *ast.ForStmt:
		return b.forStmt(v)
	case *ast.StateStmt:
		keyword := b.leaf()
		value := b.expr(v.Value)
		return concat(keyword, text(" "), value, text(" "), b.stateBody(v.Conditions))
	case *ast.AttemptStmt:
		return b.attemptStmt(v)
	case *ast.BringStmt:
		return b.bringStmt(v)
	case *ast.FunctionDecl:
		return b.functionDecl(v)
	case *ast.ClassDecl:
		return b.classDecl(v)
	case *ast.StructureDecl:
		return b.structureDecl(v)
	default:
		return b.leaf()
	}
}

func (b *builder) ifStmt(v *ast.IfStmt) doc {
	var parts []doc
	for index, branch := range v.Branches {
		if index == 0 {
			keyword := b.leaf()
			condition := b.expr(branch.Condition)
			parts = append(parts, keyword, text(" "), condition, text(" "), b.block(branch.Body))
			continue
		}
		gapDoc := b.gap()
		elseKeyword := b.leaf()
		ifKeyword := b.leaf()
		condition := b.expr(branch.Condition)
		parts = append(parts, withGapSeparator(gapDoc, false), elseKeyword, text(" "), ifKeyword, text(" "), condition, text(" "), b.block(branch.Body))
	}
	if v.Else != nil {
		gapDoc := b.gap()
		elseKeyword := b.leaf()
		parts = append(parts, withGapSeparator(gapDoc, false), elseKeyword, text(" "), b.block(v.Else))
	}
	return concat(parts...)
}

func (b *builder) forStmt(v *ast.ForStmt) doc {
	keyword := b.leaf()
	name := b.leaf()
	var typeDoc doc
	if v.Type != nil {
		colon := b.leaf()
		typeDoc = concat(colon, text(" "), b.typeRef(v.Type))
	}
	inKeyword := b.leaf()
	iterable := b.expr(v.Iterable)
	return concat(keyword, text(" "), name, typeDoc, text(" "), inKeyword, text(" "), iterable, text(" "), b.block(v.Body))
}

func (b *builder) attemptStmt(v *ast.AttemptStmt) doc {
	keyword := b.leaf()
	parts := []doc{keyword, text(" "), b.block(v.Body)}
	for _, except := range v.Excepts {
		gapDoc := b.gap()
		exceptKeyword := b.leaf()
		typeDoc := b.typeRef(except.Type)
		var asDoc doc
		if except.Name != "" {
			asKeyword := b.leaf()
			name := b.leaf()
			asDoc = concat(text(" "), asKeyword, text(" "), name)
		}
		parts = append(parts, withGapSeparator(gapDoc, false), exceptKeyword, text(" "), typeDoc, asDoc, text(" "), b.block(except.Body))
	}
	if v.Ultimately != nil {
		gapDoc := b.gap()
		ultimatelyKeyword := b.leaf()
		parts = append(parts, withGapSeparator(gapDoc, false), ultimatelyKeyword, text(" "), b.block(v.Ultimately))
	}
	return concat(parts...)
}

func (b *builder) bringStmt(v *ast.BringStmt) doc {
	if v.Namespace {
		keyword := b.leaf()
		module := b.leaf()
		result := concat(keyword, text(" "), module)
		if v.Alias == "" {
			return result
		}
		asKeyword := b.leaf()
		alias := b.leaf()
		return concat(result, text(" "), asKeyword, text(" "), alias)
	}
	fromKeyword := b.leaf()
	module := b.leaf()
	bringKeyword := b.leaf()
	prefix := concat(fromKeyword, text(" "), module, text(" "), bringKeyword, text(" "))
	if v.All {
		return concat(prefix, b.leaf())
	}
	if b.current().Kind == token.LeftParen {
		return concat(prefix, b.delimitedGroup(len(v.Names), func(int) doc { return b.leaf() }))
	}
	return concat(prefix, b.leaf())
}

func (b *builder) expr(e ast.Expr) doc {
	switch v := e.(type) {
	case *ast.LiteralExpr:
		return b.leaf()
	case *ast.IdentifierExpr:
		return b.leaf()
	case *ast.GroupExpr:
		open := b.leaf()
		lead := b.gap()
		inner := b.expr(v.Expression)
		trail := b.gap()
		return concat(open, lead, inner, trail, b.leaf())
	case *ast.LambdaExpr:
		keyword := b.leaf()
		// An explicit capture list is part of the lambda's own syntax, so its
		// tokens are consumed here rather than left for the parameter list.
		captures := text("")
		if b.current().Kind == token.LeftBracket {
			captures = concat(b.captureGroup(v.Captures), text(" "))
		}
		parameters := b.parameterGroup(v.Parameters)
		arrow := b.leaf()
		gapDoc := b.gap()
		return concat(keyword, text(" "), captures, parameters, text(" "), arrow, text(" "), gapDoc, b.expr(v.Body))
	case *ast.UnaryExpr:
		operator := b.leaf()
		gapDoc := b.gap()
		operand := b.expr(v.Operand)
		if v.Operator == "not" {
			return concat(operator, text(" "), gapDoc, operand)
		}
		return concat(operator, gapDoc, operand)
	case *ast.BinaryExpr:
		left := b.expr(v.Left)
		operator := b.binaryOperatorLeaf(v.Operator)
		gapDoc := b.gap()
		right := b.expr(v.Right)
		return concat(left, text(" "), operator, text(" "), gapDoc, right)
	case *ast.CallExpr:
		callee := b.expr(v.Callee)
		return concat(callee, b.callArgsGroup(v.Arguments))
	case *ast.MemberExpr:
		object := b.expr(v.Object)
		dot := b.leaf()
		name := b.leaf()
		return concat(object, dot, name)
	case *ast.IndexExpr:
		object := b.expr(v.Object)
		open := b.leaf()
		lead := b.gap()
		index := b.expr(v.Index)
		trail := b.gap()
		return concat(object, open, lead, index, trail, b.leaf())
	case *ast.SliceExpr:
		return b.sliceExpr(v)
	case *ast.ListExpr:
		return b.listGroup(v.Elements)
	case *ast.PairExpr:
		return b.pairGroup(v.Entries)
	case *ast.StringExpr:
		return b.verbatimSpan(v.Span())
	default:
		return b.leaf()
	}
}

func (b *builder) sliceExpr(v *ast.SliceExpr) doc {
	object := b.expr(v.Object)
	open := b.leaf()
	parts := []doc{object, open, b.gap()}
	if v.Start != nil {
		parts = append(parts, b.expr(v.Start))
	}
	parts = append(parts, b.gap(), b.leaf())
	parts = append(parts, b.gap())
	if v.End != nil {
		parts = append(parts, b.expr(v.End))
	}
	parts = append(parts, b.gap(), b.leaf())
	return concat(parts...)
}

// binaryOperatorLeaf consumes the token(s) spelling a binary operator. Most
// operators are one token; "is not", "has not", and "not in" are two.
func (b *builder) binaryOperatorLeaf(operator string) doc {
	switch operator {
	case "is not", "has not", "not in":
		first := b.leaf()
		return concat(first, text(" "), b.leaf())
	default:
		return b.leaf()
	}
}

// verbatimSpan reproduces a StringExpr's exact source text -- including
// escapes, interpolation spacing, and multiline content -- unchanged. String
// interpolation is not part of the flexible-comma/canonical-break surface
// this package canonicalizes, so it is reproduced rather than re-derived.
func (b *builder) verbatimSpan(span source.Span) doc {
	trivia := b.leadingTriviaDoc()
	raw := b.file.Text[span.Start.Offset:span.End.Offset]
	for b.cursor < len(b.tokens) && b.tokens[b.cursor].Span.Start.Offset < span.End.Offset {
		b.cursor++
	}
	return concat(trivia, text(raw))
}
