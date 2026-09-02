package analysis

import (
	"sort"

	"ahdcode/internal/lexer"
	"ahdcode/internal/module"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

// Semantic token type indices — must match the legend advertised in LSP
// initialize. Restrained to categories the compiler genuinely distinguishes.
const (
	SemTokenNamespace = iota
	SemTokenType
	SemTokenFunction
	SemTokenMethod
	SemTokenParameter
	SemTokenVariable
	SemTokenProperty
	SemTokenKeyword
	SemTokenString
	SemTokenNumber
	SemTokenComment
)

// Semantic token modifier bit flags — must match LSP legend order.
const (
	SemModDeclaration = 1 << iota
	SemModReadonly
)

// SemanticToken is one highlighted span in source order.
type SemanticToken struct {
	StartOffset int
	Length      int
	TokenType   int
	Modifiers   int
}

// SemanticTokens returns semantic highlighting spans for path's most recent
// analysis snapshot, sorted by start offset.
func (store *Store) SemanticTokens(path string) []SemanticToken {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()
	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return nil
	}
	text := cached.result.Text[entryModule.File.Path]
	file := source.File{ID: entryModule.File.ID, Path: entryModule.File.Path, Text: text}
	lexed := lexer.Lex(file)
	collector := &semanticTokenCollector{
		entryModule: entryModule,
		text:        text,
		fileID:      entryModule.File.ID,
	}
	collector.collectProgram(entryModule.Parsed.Program)
	collector.collectComments(lexed.Tokens)
	sort.Slice(collector.tokens, func(i, j int) bool {
		return collector.tokens[i].StartOffset < collector.tokens[j].StartOffset
	})
	return collector.tokens
}

type semanticTokenCollector struct {
	entryModule *module.Module
	text        string
	fileID      source.FileID
	tokens      []SemanticToken
	seen        map[int]bool
}

func (c *semanticTokenCollector) add(start, length, tokenType, modifiers int) {
	if length <= 0 || c.seen[start] {
		return
	}
	c.seen[start] = true
	c.tokens = append(c.tokens, SemanticToken{
		StartOffset: start,
		Length:      length,
		TokenType:   tokenType,
		Modifiers:   modifiers,
	})
}

func (c *semanticTokenCollector) addSpan(span source.Span, tokenType, modifiers int) {
	if span.FileID != c.fileID || span.Empty() {
		return
	}
	c.add(span.Start.Offset, span.End.Offset-span.Start.Offset, tokenType, modifiers)
}

func (c *semanticTokenCollector) collectProgram(program *ast.Program) {
	if program == nil {
		c.seen = make(map[int]bool)
		return
	}
	c.seen = make(map[int]bool)
	resolved := c.entryModule.Semantic.ResolvedSymbols
	for _, statement := range program.Statements {
		c.collectStmt(statement, resolved, true)
	}
}

func (c *semanticTokenCollector) collectStmt(statement ast.Stmt, resolved map[ast.Node]*semantic.Symbol, moduleRoot bool) {
	if statement == nil {
		return
	}
	if symbol, ok := resolved[statement]; ok && symbol != nil {
		c.collectSymbolNode(statement, symbol, moduleRoot)
	}
	switch node := statement.(type) {
	case *ast.VariableDecl:
		c.collectVariableDecl(node, resolved, moduleRoot)
	case *ast.FunctionDecl:
		c.collectFunctionDecl(node, resolved, moduleRoot)
	case *ast.ClassDecl:
		c.collectClassDecl(node, resolved)
	case *ast.StructureDecl:
		c.collectStructureDecl(node, resolved)
	case *ast.BringStmt:
		c.addKeywordSpan(node.Span(), "bring", "from")
	}
}

func (c *semanticTokenCollector) collectVariableDecl(decl *ast.VariableDecl, resolved map[ast.Node]*semantic.Symbol, moduleRoot bool) {
	if decl.Type != nil {
		c.collectTypeRef(decl.Type)
	}
	if symbol, ok := resolved[decl]; ok {
		c.collectSymbolNode(decl.Target, symbol, moduleRoot)
	}
	c.collectExpr(decl.Initializer, resolved)
}

func (c *semanticTokenCollector) collectFunctionDecl(decl *ast.FunctionDecl, resolved map[ast.Node]*semantic.Symbol, moduleRoot bool) {
	if symbol, ok := resolved[decl]; ok && symbol != nil {
		tokenType := SemTokenFunction
		if !moduleRoot {
			tokenType = SemTokenMethod
		}
		c.addDeclName(decl.Span(), decl.Name, tokenType, 0)
	}
	for index := range decl.Parameters {
		parameter := &decl.Parameters[index]
		if symbol, ok := resolved[parameter]; ok && symbol != nil {
			c.addSpan(parameter.Span(), SemTokenParameter, SemModDeclaration)
		}
	}
	if decl.ReturnType != nil {
		c.collectTypeRef(decl.ReturnType)
	}
	if decl.Body != nil {
		for _, statement := range decl.Body.Statements {
			c.collectStmt(statement, resolved, false)
		}
	}
}

func (c *semanticTokenCollector) collectClassDecl(decl *ast.ClassDecl, resolved map[ast.Node]*semantic.Symbol) {
	if symbol, ok := resolved[decl]; ok && symbol != nil {
		c.addDeclName(decl.Span(), symbol.Name, SemTokenType, SemModDeclaration)
	}
	if decl.Parent != nil {
		c.collectTypeRef(decl.Parent)
	}
	for _, member := range decl.Members {
		c.collectStmt(member, resolved, false)
	}
}

func (c *semanticTokenCollector) collectStructureDecl(decl *ast.StructureDecl, resolved map[ast.Node]*semantic.Symbol) {
	for index := range decl.Parameters {
		parameter := &decl.Parameters[index]
		if symbol, ok := resolved[parameter]; ok && symbol != nil {
			modifiers := SemModDeclaration
			tokenType := SemTokenParameter
			if symbol.Kind == semantic.MemberSymbol {
				tokenType = SemTokenProperty
			}
			c.addSpan(parameter.Span(), tokenType, modifiers)
		}
	}
	if decl.Body != nil {
		for _, statement := range decl.Body.Statements {
			c.collectStmt(statement, resolved, false)
		}
	}
}

func (c *semanticTokenCollector) collectSymbolNode(node ast.Node, symbol *semantic.Symbol, moduleRoot bool) {
	modifiers := 0
	if isDeclarationNode(node, symbol) {
		modifiers |= SemModDeclaration
	}
	if symbol.Constant {
		modifiers |= SemModReadonly
	}
	tokenType := symbolTokenType(symbol, moduleRoot)
	switch typed := node.(type) {
	case *ast.IdentifierExpr:
		c.addSpan(typed.Span(), tokenType, modifiers)
	case *ast.VariableDecl:
		if identifier, ok := typed.Target.(*ast.IdentifierExpr); ok {
			c.addSpan(identifier.Span(), tokenType, modifiers)
		}
	case *ast.FunctionDecl:
		c.addDeclName(typed.Span(), typed.Name, tokenType, modifiers)
	case *ast.ClassDecl:
		c.addDeclName(typed.Span(), typed.Name, tokenType, modifiers)
	case *ast.MemberExpr:
		c.addSpan(memberNameSpan(typed), tokenType, modifiers)
	default:
		c.addSpan(symbol.Span, tokenType, modifiers)
	}
}

func symbolTokenType(symbol *semantic.Symbol, moduleRoot bool) int {
	switch symbol.Kind {
	case semantic.NamespaceSymbol:
		return SemTokenNamespace
	case semantic.ClassSymbol:
		return SemTokenType
	case semantic.FunctionSymbol:
		if moduleRoot {
			return SemTokenFunction
		}
		return SemTokenMethod
	case semantic.MemberSymbol:
		return SemTokenProperty
	case semantic.ParameterSymbol:
		return SemTokenParameter
	default:
		return SemTokenVariable
	}
}

func isDeclarationNode(node ast.Node, symbol *semantic.Symbol) bool {
	if symbol.Declaration == node {
		return true
	}
	if variableDecl, ok := symbol.Declaration.(*ast.VariableDecl); ok {
		if identifier, ok := variableDecl.Target.(*ast.IdentifierExpr); ok && identifier == node {
			return true
		}
	}
	if functionDecl, ok := symbol.Declaration.(*ast.FunctionDecl); ok && functionDecl == node {
		return true
	}
	if classDecl, ok := symbol.Declaration.(*ast.ClassDecl); ok && classDecl == node {
		return true
	}
	return false
}

func (c *semanticTokenCollector) collectExpr(expression ast.Expr, resolved map[ast.Node]*semantic.Symbol) {
	if expression == nil {
		return
	}
	if symbol, ok := resolved[expression]; ok && symbol != nil {
		c.collectSymbolNode(expression, symbol, symbol.ModuleRoot)
	}
	switch node := expression.(type) {
	case *ast.CallExpr:
		c.collectExpr(node.Callee, resolved)
		for index := range node.Arguments {
			c.collectExpr(node.Arguments[index].Value, resolved)
		}
	case *ast.MemberExpr:
		c.collectExpr(node.Object, resolved)
	case *ast.BinaryExpr:
		c.collectExpr(node.Left, resolved)
		c.collectExpr(node.Right, resolved)
	case *ast.UnaryExpr:
		c.collectExpr(node.Operand, resolved)
	case *ast.IdentifierExpr:
		if symbol, ok := resolved[node]; ok && symbol != nil {
			c.collectSymbolNode(node, symbol, false)
		}
	case *ast.LiteralExpr:
		switch node.Kind {
		case ast.IntLiteral, ast.RealLiteral:
			c.addSpan(node.Span(), SemTokenNumber, 0)
		}
	case *ast.StringExpr:
		c.addSpan(node.Span(), SemTokenString, 0)
	case *ast.ListExpr:
		for _, element := range node.Elements {
			c.collectExpr(element, resolved)
		}
	case *ast.PairExpr:
		for index := range node.Entries {
			c.collectExpr(node.Entries[index].Key, resolved)
			c.collectExpr(node.Entries[index].Value, resolved)
		}
	}
}

func (c *semanticTokenCollector) collectTypeRef(typeRef *ast.TypeRef) {
	if typeRef == nil {
		return
	}
	c.addSpan(typeRef.Span(), SemTokenType, 0)
	for _, argument := range typeRef.Arguments {
		if argument != nil {
			c.collectTypeRef(argument)
		}
	}
}

func (c *semanticTokenCollector) addDeclName(fullSpan source.Span, name string, tokenType, modifiers int) {
	if name == "" || fullSpan.Empty() {
		return
	}
	nameLen := len(name)
	if fullSpan.End.Offset-fullSpan.Start.Offset >= nameLen {
		c.add(fullSpan.Start.Offset, nameLen, tokenType, modifiers)
	}
}

func (c *semanticTokenCollector) addKeywordSpan(span source.Span, keywords ...string) {
	_ = keywords
	_ = span
}

func (c *semanticTokenCollector) collectComments(tokens []token.Token) {
	for _, item := range tokens {
		for _, trivia := range item.LeadingTrivia {
			if trivia.Kind == token.LineCommentTrivia || trivia.Kind == token.BlockCommentTrivia {
				c.add(trivia.Span.Start.Offset, trivia.Span.End.Offset-trivia.Span.Start.Offset, SemTokenComment, 0)
			}
		}
	}
}

// EncodeSemanticTokens converts semantic tokens to LSP delta encoding using
// UTF-16 code unit lengths.
func EncodeSemanticTokens(text string, tokens []SemanticToken) []uint32 {
	index := newUTF16LineIndex(text)
	var data []uint32
	var prevLine, prevChar uint32
	for _, item := range tokens {
		start := index.offsetToLineChar(item.StartOffset)
		length := index.utf16Length(item.StartOffset, item.StartOffset+item.Length)
		deltaLine := uint32(start.line) - prevLine
		var deltaChar uint32
		if deltaLine == 0 {
			deltaChar = uint32(start.char) - prevChar
		} else {
			deltaChar = uint32(start.char)
		}
		data = append(data, deltaLine, deltaChar, uint32(length), uint32(item.TokenType), uint32(item.Modifiers))
		prevLine = uint32(start.line)
		prevChar = uint32(start.char)
	}
	return data
}

type utf16LineIndex struct {
	text   string
	starts []int
}

type lineChar struct {
	line int
	char int
}

func newUTF16LineIndex(text string) *utf16LineIndex {
	starts := []int{0}
	for index := 0; index < len(text); index++ {
		if text[index] == '\n' {
			starts = append(starts, index+1)
		}
	}
	return &utf16LineIndex{text: text, starts: starts}
}

func (index *utf16LineIndex) offsetToLineChar(offset int) lineChar {
	line := sort.Search(len(index.starts), func(candidate int) bool {
		return index.starts[candidate] > offset
	}) - 1
	if line < 0 {
		line = 0
	}
	lineStart := index.starts[line]
	return lineChar{line: line, char: utf16Length(index.text[lineStart:offset])}
}

func (index *utf16LineIndex) utf16Length(start, end int) int {
	if start < 0 {
		start = 0
	}
	if end > len(index.text) {
		end = len(index.text)
	}
	return utf16Length(index.text[start:end])
}

func utf16Length(text string) int {
	length := 0
	for _, codePoint := range text {
		if codePoint > 0xFFFF {
			length += 2
		} else {
			length++
		}
	}
	return length
}
