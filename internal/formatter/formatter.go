// Package formatter implements the canonical AhdCode source formatter. Syntax
// validity and structural layout come from the ordinary lexer/parser; exact
// comments and string spellings come from the retained token/trivia stream.
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
	diagnostics := append(append([]diagnostics.Diagnostic(nil), lexed.Diagnostics...), parsed.Diagnostics...)
	result := Result{Diagnostics: diagnostics}
	if result.HasErrors() || parsed.Program == nil {
		return result
	}
	layout := collectLayout(parsed.Program, parsed.Tokens)
	printer := tokenPrinter{file: file, tokens: parsed.Tokens, layout: layout, lineStart: true}
	result.Text = printer.print()
	return result
}

type syntaxLayout struct {
	pairs      map[int]bool
	unary      map[int]bool
	typeAngles map[int]bool
	multiline  map[int]bool
	closing    map[int]int
}

func collectLayout(program *ast.Program, tokens []token.Token) syntaxLayout {
	layout := syntaxLayout{
		pairs: make(map[int]bool), unary: make(map[int]bool), typeAngles: make(map[int]bool),
		multiline: make(map[int]bool), closing: make(map[int]int),
	}
	walker := layoutWalker{layout: &layout, tokens: tokens}
	walker.program(program)
	type delimiter struct {
		kind   token.Kind
		offset int
		line   int
	}
	var stack []delimiter
	for _, item := range tokens {
		switch item.Kind {
		case token.LeftParen, token.LeftBracket, token.LeftBrace:
			stack = append(stack, delimiter{kind: item.Kind, offset: item.Span.Start.Offset, line: item.Span.Start.Line})
		case token.RightParen, token.RightBracket, token.RightBrace:
			if len(stack) == 0 || !matchingDelimiter(stack[len(stack)-1].kind, item.Kind) {
				continue
			}
			opening := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			layout.closing[opening.offset] = item.Span.Start.Offset
			if opening.line != item.Span.Start.Line {
				layout.multiline[opening.offset] = true
			}
		}
	}
	return layout
}

func matchingDelimiter(open, close token.Kind) bool {
	return open == token.LeftParen && close == token.RightParen ||
		open == token.LeftBracket && close == token.RightBracket ||
		open == token.LeftBrace && close == token.RightBrace
}

type layoutWalker struct {
	layout *syntaxLayout
	tokens []token.Token
}

func (walker *layoutWalker) program(program *ast.Program) {
	for _, statement := range program.Statements {
		walker.statement(statement)
	}
}

func (walker *layoutWalker) statement(statement ast.Stmt) {
	if statement == nil {
		return
	}
	switch value := statement.(type) {
	case *ast.ExprStmt:
		walker.expression(value.Expression)
	case *ast.VariableDecl:
		walker.expression(value.Target)
		walker.typeRef(value.Type)
		walker.expression(value.Initializer)
	case *ast.AssignmentStmt:
		walker.expression(value.Target)
		walker.expression(value.Value)
	case *ast.IncDecStmt:
		walker.expression(value.Target)
	case *ast.ReturnStmt:
		walker.expression(value.Value)
	case *ast.TossStmt:
		walker.expression(value.Value)
	case *ast.IfStmt:
		for _, branch := range value.Branches {
			walker.expression(branch.Condition)
			walker.block(branch.Body)
		}
		walker.block(value.Else)
	case *ast.WhileStmt:
		walker.expression(value.Condition)
		walker.block(value.Body)
	case *ast.UntilStmt:
		walker.expression(value.Condition)
		walker.block(value.Body)
	case *ast.ForStmt:
		walker.expression(value.Iterable)
		walker.block(value.Body)
	case *ast.StateStmt:
		walker.expression(value.Value)
		for _, condition := range value.Conditions {
			walker.expression(condition.Match)
			walker.block(condition.Body)
		}
	case *ast.AttemptStmt:
		walker.block(value.Body)
		for _, clause := range value.Excepts {
			walker.typeRef(clause.Type)
			walker.block(clause.Body)
		}
		walker.block(value.Ultimately)
	case *ast.FunctionDecl:
		walker.forceFirst(value.Span(), token.LeftParen)
		walker.parameters(value.Parameters)
		walker.typeRef(value.ReturnType)
		walker.block(value.Body)
	case *ast.ClassDecl:
		walker.markAnglesBeforeBrace(value.Span())
		walker.typeRef(value.Parent)
		for _, member := range value.Members {
			walker.statement(member)
		}
	case *ast.StructureDecl:
		walker.forceFirst(value.Span(), token.LeftParen)
		walker.parameters(value.Parameters)
		walker.block(value.Body)
	}
}

func (walker *layoutWalker) forceFirst(span source.Span, kind token.Kind) {
	for _, item := range walker.tokens {
		if item.Span.Start.Offset < span.Start.Offset || item.Span.End.Offset > span.End.Offset {
			continue
		}
		if item.Kind == kind {
			walker.layout.multiline[item.Span.Start.Offset] = true
			return
		}
	}
}

func (walker *layoutWalker) markAnglesBeforeBrace(span source.Span) {
	for _, item := range walker.tokens {
		if item.Span.Start.Offset < span.Start.Offset || item.Span.End.Offset > span.End.Offset {
			continue
		}
		if item.Kind == token.LeftBrace {
			return
		}
		if item.Kind == token.Less || item.Kind == token.Greater {
			walker.layout.typeAngles[item.Span.Start.Offset] = true
		}
	}
}

func (walker *layoutWalker) block(block *ast.Block) {
	if block == nil {
		return
	}
	for _, statement := range block.Statements {
		walker.statement(statement)
	}
}

func (walker *layoutWalker) parameters(parameters []ast.Parameter) {
	for index := range parameters {
		walker.typeRef(parameters[index].Type)
		walker.expression(parameters[index].Default)
	}
}

func (walker *layoutWalker) typeRef(value *ast.TypeRef) {
	if value == nil {
		return
	}
	for _, item := range walker.tokens {
		if item.Span.Start.Offset < value.Span().Start.Offset || item.Span.End.Offset > value.Span().End.Offset {
			continue
		}
		if item.Kind == token.Less || item.Kind == token.Greater {
			walker.layout.typeAngles[item.Span.Start.Offset] = true
		}
	}
	for _, argument := range value.Arguments {
		walker.typeRef(argument)
	}
}

func (walker *layoutWalker) expression(expression ast.Expr) {
	if expression == nil {
		return
	}
	switch value := expression.(type) {
	case *ast.GroupExpr:
		walker.expression(value.Expression)
	case *ast.UnaryExpr:
		walker.layout.unary[value.Span().Start.Offset] = true
		walker.expression(value.Operand)
	case *ast.BinaryExpr:
		walker.expression(value.Left)
		walker.expression(value.Right)
	case *ast.CallExpr:
		walker.expression(value.Callee)
		for _, argument := range value.Arguments {
			walker.expression(argument.Value)
		}
	case *ast.MemberExpr:
		walker.expression(value.Object)
	case *ast.IndexExpr:
		walker.expression(value.Object)
		walker.expression(value.Index)
	case *ast.SliceExpr:
		walker.expression(value.Object)
		walker.expression(value.Start)
		walker.expression(value.End)
	case *ast.ListExpr:
		for _, element := range value.Elements {
			walker.expression(element)
		}
	case *ast.PairExpr:
		walker.layout.pairs[value.Span().Start.Offset] = true
		for _, entry := range value.Entries {
			walker.expression(entry.Key)
			walker.expression(entry.Value)
		}
	case *ast.StringExpr:
		for _, part := range value.Parts {
			walker.expression(part.Expression)
		}
	}
}

type tokenPrinter struct {
	file       source.File
	tokens     []token.Token
	layout     syntaxLayout
	out        strings.Builder
	indent     int
	lineStart  bool
	lastKind   token.Kind
	lastOffset int
}

func (printer *tokenPrinter) print() string {
	for index := 0; index < len(printer.tokens); index++ {
		item := printer.tokens[index]
		printer.trivia(item.LeadingTrivia)
		if item.Kind == token.EOF {
			break
		}
		if item.Kind == token.Newline {
			printer.sourceNewline()
			continue
		}
		if item.Kind == token.StringStart {
			index = printer.string(index)
			continue
		}
		printer.token(item, nextKind(printer.tokens, index+1))
	}
	text := strings.TrimRight(printer.out.String(), " \t\r\n")
	if text == "" {
		return ""
	}
	return text + "\n"
}

func nextKind(tokens []token.Token, index int) token.Kind {
	if index >= len(tokens) {
		return token.EOF
	}
	return tokens[index].Kind
}

func (printer *tokenPrinter) string(index int) int {
	start := printer.tokens[index]
	end := start.Span.End.Offset
	for index+1 < len(printer.tokens) {
		index++
		end = printer.tokens[index].Span.End.Offset
		if printer.tokens[index].Kind == token.StringEnd || printer.tokens[index].Kind == token.EOF {
			break
		}
	}
	printer.before(start)
	printer.write(printer.file.Text[start.Span.Start.Offset:end])
	printer.lastKind = token.StringEnd
	printer.lastOffset = start.Span.Start.Offset
	return index
}

func (printer *tokenPrinter) token(item token.Token, next token.Kind) {
	offset := item.Span.Start.Offset
	isPair := printer.layout.pairs[offset]
	isBlockOpen := item.Kind == token.LeftBrace && !isPair
	isMultilineOpen := (item.Kind == token.LeftParen || item.Kind == token.LeftBracket || isPair) && printer.layout.multiline[offset]
	isClosing := item.Kind == token.RightParen || item.Kind == token.RightBracket || item.Kind == token.RightBrace
	if isClosing {
		opening := printer.openingOffset(offset)
		if item.Kind == token.RightBrace && !printer.layout.pairs[opening] || printer.layout.multiline[opening] {
			if printer.indent > 0 {
				printer.indent--
			}
			printer.ensureNewline()
		}
	}
	printer.before(item)
	printer.write(item.Lexeme)
	printer.lastKind = item.Kind
	printer.lastOffset = offset
	if isBlockOpen || isMultilineOpen {
		printer.indent++
		if next != token.Newline {
			printer.ensureNewline()
		}
	}
}

func (printer *tokenPrinter) openingOffset(closing int) int {
	for opening, end := range printer.layout.closing {
		if end == closing {
			return opening
		}
	}
	return -1
}

func (printer *tokenPrinter) before(item token.Token) {
	if printer.lineStart {
		printer.out.WriteString(strings.Repeat("    ", printer.indent))
		printer.lineStart = false
	}
	if needsSpace(printer.lastKind, printer.lastOffset, item, printer.layout) && !endsSpace(printer.out.String()) {
		printer.out.WriteByte(' ')
	}
}

func needsSpace(previous token.Kind, previousOffset int, current token.Token, layout syntaxLayout) bool {
	if previous == token.Invalid || previous == token.Newline {
		return false
	}
	kind := current.Kind
	if kind == token.RightParen || kind == token.RightBracket || kind == token.RightBrace || kind == token.Comma || kind == token.Dot || kind == token.Colon {
		return false
	}
	if previous == token.LeftParen || previous == token.LeftBracket || previous == token.LeftBrace || previous == token.Dot {
		return false
	}
	if (kind == token.Less || kind == token.Greater) && layout.typeAngles[current.Span.Start.Offset] ||
		previous == token.Less && layout.typeAngles[previousOffset] {
		return false
	}
	if kind == token.LeftParen && (previous == token.Identifier || previous == token.RightParen || previous == token.RightBracket || previous == token.StringEnd || previous == token.KeywordObject || previous == token.KeywordError) {
		return false
	}
	if kind == token.LeftBracket && (previous == token.Identifier || previous == token.RightParen || previous == token.RightBracket) {
		return false
	}
	if kind == token.Increment || kind == token.Decrement || previous == token.Increment || previous == token.Decrement {
		return false
	}
	if layout.unary[previousOffset] && (previous == token.Plus || previous == token.Minus) {
		return false
	}
	return true
}

func (printer *tokenPrinter) trivia(items []token.Trivia) {
	for _, item := range items {
		if item.Kind == token.WhitespaceTrivia {
			continue
		}
		if printer.lineStart {
			continuation := item.Kind == token.BlockCommentTrivia && !strings.HasPrefix(item.Lexeme, "/*")
			if !continuation {
				printer.out.WriteString(strings.Repeat("    ", printer.indent))
			}
			printer.lineStart = false
		} else if !endsSpace(printer.out.String()) {
			printer.out.WriteByte(' ')
		}
		printer.out.WriteString(item.Lexeme)
	}
}

func (printer *tokenPrinter) sourceNewline() {
	printer.newline(true)
	printer.lastKind = token.Newline
}

func (printer *tokenPrinter) ensureNewline() { printer.newline(false) }

func (printer *tokenPrinter) newline(allowBlank bool) {
	text := strings.TrimRight(printer.out.String(), " \t")
	printer.out.Reset()
	printer.out.WriteString(text)
	if strings.HasSuffix(text, "\n") {
		if allowBlank && !strings.HasSuffix(text, "\n\n") {
			printer.out.WriteByte('\n')
		}
	} else {
		printer.out.WriteByte('\n')
	}
	printer.lineStart = true
}

func (printer *tokenPrinter) write(text string) {
	printer.out.WriteString(text)
	if strings.Contains(text, "\n") || strings.Contains(text, "\r") {
		printer.lineStart = strings.HasSuffix(text, "\n") || strings.HasSuffix(text, "\r")
	}
}

func endsSpace(text string) bool {
	if text == "" {
		return false
	}
	last := text[len(text)-1]
	return last == ' ' || last == '\t' || last == '\n' || last == '\r'
}
