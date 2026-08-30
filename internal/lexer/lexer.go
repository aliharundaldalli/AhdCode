package lexer

import (
	"fmt"
	"unicode/utf8"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/token"
)

type lexer struct {
	file    source.File
	offset  int
	line    int
	column  int
	tokens  []token.Token
	pending []token.Trivia
	bag     diagnostics.Bag
}

type interpolationContext struct {
	triple             bool
	braceDepth         int
	sawExpressionToken bool
	opening            source.Position
}

// Lex tokenizes one AhdCode source file.
func Lex(file source.File) Result {
	l := &lexer{file: file, line: 1, column: 1}
	l.run()
	return Result{Tokens: l.tokens, Diagnostics: l.bag.Items()}
}

func (l *lexer) run() {
	for !l.atEnd() {
		l.scanTrivia()
		if l.atEnd() {
			break
		}
		if l.isNewline() {
			l.emitNewline()
			continue
		}
		l.scanCodeToken(nil)
	}
	l.emitSynthetic(token.EOF, l.position())
}

func (l *lexer) position() source.Position {
	return source.Position{Offset: l.offset, Line: l.line, Column: l.column}
}

func (l *lexer) span(start source.Position) source.Span {
	return source.NewSpan(l.file.ID, start, l.position())
}

func (l *lexer) atEnd() bool {
	return l.offset >= len(l.file.Text)
}

func (l *lexer) byteAt(relative int) byte {
	index := l.offset + relative
	if index < 0 || index >= len(l.file.Text) {
		return 0
	}
	return l.file.Text[index]
}

func (l *lexer) hasPrefix(text string) bool {
	return len(l.file.Text)-l.offset >= len(text) && l.file.Text[l.offset:l.offset+len(text)] == text
}

func (l *lexer) peekRune() (rune, int) {
	if l.atEnd() {
		return 0, 0
	}
	return utf8.DecodeRuneInString(l.file.Text[l.offset:])
}

func (l *lexer) advanceRune() (rune, int) {
	start := l.position()
	r, size := l.peekRune()
	if size == 0 {
		return 0, 0
	}
	if r == utf8.RuneError && size == 1 {
		l.offset++
		l.column++
		l.bag.Error(codeInvalidUTF8, "invalid UTF-8 byte in source", l.span(start), "save the source file as valid UTF-8")
		return r, size
	}
	l.offset += size
	l.column++
	return r, size
}

func (l *lexer) advanceASCII(count int) {
	for range count {
		if l.atEnd() {
			return
		}
		l.offset++
		l.column++
	}
}

func (l *lexer) isNewline() bool {
	return l.byteAt(0) == '\n' || l.byteAt(0) == '\r'
}

func (l *lexer) consumeNewline() {
	if l.hasPrefix("\r\n") {
		l.offset += 2
	} else {
		l.offset++
	}
	l.line++
	l.column = 1
}

func (l *lexer) emitNewline() {
	start := l.position()
	l.consumeNewline()
	l.emit(token.Newline, start, "\n", false)
}

func (l *lexer) emit(kind token.Kind, start source.Position, value string, synthetic bool) {
	span := l.span(start)
	lexeme := span.Text(l.file)
	leading := append([]token.Trivia(nil), l.pending...)
	l.pending = l.pending[:0]
	l.tokens = append(l.tokens, token.Token{
		Kind: kind, Lexeme: lexeme, Value: value, Span: span,
		LeadingTrivia: leading, Synthetic: synthetic,
	})
}

func (l *lexer) emitSynthetic(kind token.Kind, at source.Position) {
	l.emit(kind, at, "", true)
}

func (l *lexer) addTrivia(kind token.TriviaKind, start source.Position) {
	span := l.span(start)
	if span.Empty() {
		return
	}
	l.pending = append(l.pending, token.Trivia{Kind: kind, Lexeme: span.Text(l.file), Span: span})
}

func (l *lexer) scanTrivia() {
	for !l.atEnd() {
		switch {
		case isHorizontalWhitespace(l.byteAt(0)):
			start := l.position()
			for isHorizontalWhitespace(l.byteAt(0)) {
				l.advanceASCII(1)
			}
			l.addTrivia(token.WhitespaceTrivia, start)
		case l.hasPrefix("//"):
			start := l.position()
			l.advanceASCII(2)
			for !l.atEnd() && !l.isNewline() {
				l.advanceRune()
			}
			l.addTrivia(token.LineCommentTrivia, start)
		case l.hasPrefix("/*"):
			l.scanBlockComment()
		default:
			return
		}
	}
}

func (l *lexer) scanBlockComment() {
	opening := l.position()
	chunkStart := opening
	l.advanceASCII(2)
	for !l.atEnd() {
		if l.hasPrefix("*/") {
			l.advanceASCII(2)
			l.addTrivia(token.BlockCommentTrivia, chunkStart)
			return
		}
		if l.isNewline() {
			l.addTrivia(token.BlockCommentTrivia, chunkStart)
			l.emitNewline()
			chunkStart = l.position()
			continue
		}
		l.advanceRune()
	}
	l.addTrivia(token.BlockCommentTrivia, chunkStart)
	l.bag.Error(codeUnterminatedBlockComment, "unterminated multiline comment", l.span(opening), "add */ to close the comment")
}

func (l *lexer) scanCodeToken(context *interpolationContext) {
	start := l.position()
	r, size := l.peekRune()
	if r == utf8.RuneError && size == 1 {
		l.advanceRune()
		return
	}
	if isXIDStart(r) {
		l.scanIdentifier(context)
		return
	}
	if isASCIIDigit(l.byteAt(0)) {
		l.scanNumber(context)
		return
	}
	if l.byteAt(0) == '.' && isASCIIDigit(l.byteAt(1)) {
		l.scanLeadingDotNumber(context)
		return
	}
	if l.byteAt(0) == '\'' || l.byteAt(0) == '"' {
		if context != nil {
			context.sawExpressionToken = true
		}
		l.scanString()
		return
	}

	kind, width := l.operator()
	if kind == token.Invalid {
		l.advanceRune()
		l.bag.Error(codeUnexpectedCharacter, fmt.Sprintf("unexpected character %q", r), l.span(start), "remove the character or replace it with valid AhdCode syntax")
		return
	}
	l.advanceASCII(width)
	if context != nil {
		context.sawExpressionToken = true
		if kind == token.LeftBrace {
			context.braceDepth++
		} else if kind == token.RightBrace && context.braceDepth > 0 {
			context.braceDepth--
		}
	}
	l.emit(kind, start, "", false)
}

func (l *lexer) scanIdentifier(context *interpolationContext) {
	start := l.position()
	l.advanceRune()
	for !l.atEnd() {
		r, _ := l.peekRune()
		if !isXIDContinue(r) {
			break
		}
		l.advanceRune()
	}
	raw := l.span(start).Text(l.file)
	normalized := normalizeIdentifier(raw)
	kind := token.Identifier
	if keyword, ok := token.LookupKeyword(normalized); ok {
		kind = keyword
	}
	if context != nil {
		context.sawExpressionToken = true
	}
	l.emit(kind, start, normalized, false)
}

func (l *lexer) scanNumber(context *interpolationContext) {
	start := l.position()
	for isASCIIDigit(l.byteAt(0)) {
		l.advanceASCII(1)
	}
	kind := token.IntLiteral
	if l.byteAt(0) == '.' && isASCIIDigit(l.byteAt(1)) {
		kind = token.RealLiteral
		l.advanceASCII(1)
		for isASCIIDigit(l.byteAt(0)) {
			l.advanceASCII(1)
		}
	} else if l.byteAt(0) == '.' && !startsIdentifierAt(l.file.Text, l.offset+1) {
		kind = token.RealLiteral
		l.advanceASCII(1)
		l.bag.Error(codeInvalidNumericLiteral, "Real literals require digits after the decimal point", l.span(start), "add one or more digits after the decimal point")
	}
	if l.byteAt(0) == 'e' || l.byteAt(0) == 'E' {
		kind = token.RealLiteral
		l.advanceASCII(1)
		if l.byteAt(0) == '+' || l.byteAt(0) == '-' {
			l.advanceASCII(1)
		}
		if !isASCIIDigit(l.byteAt(0)) {
			l.bag.Error(codeInvalidNumericLiteral, "numeric exponent requires decimal digits", l.span(start), "add one or more digits after the exponent marker")
		} else {
			for isASCIIDigit(l.byteAt(0)) {
				l.advanceASCII(1)
			}
		}
	}
	if l.byteAt(0) == '_' || startsIdentifierAt(l.file.Text, l.offset) {
		for !l.atEnd() {
			r, _ := l.peekRune()
			if !isXIDContinue(r) {
				break
			}
			l.advanceRune()
		}
		l.bag.Error(codeInvalidNumericLiteral, "numeric suffixes and separators are not supported", l.span(start), "write plain decimal digits without a suffix or underscore")
	}
	if context != nil {
		context.sawExpressionToken = true
	}
	l.emit(kind, start, l.span(start).Text(l.file), false)
}

func (l *lexer) scanLeadingDotNumber(context *interpolationContext) {
	start := l.position()
	l.advanceASCII(1)
	for isASCIIDigit(l.byteAt(0)) {
		l.advanceASCII(1)
	}
	l.bag.Error(codeInvalidNumericLiteral, "Real literals require digits before the decimal point", l.span(start), "write 0 before the decimal point")
	if context != nil {
		context.sawExpressionToken = true
	}
	l.emit(token.RealLiteral, start, l.span(start).Text(l.file), false)
}

func (l *lexer) operator() (token.Kind, int) {
	pairs := map[string]token.Kind{
		":=": token.Declare, "->": token.Arrow, "+=": token.PlusAssign, "-=": token.MinusAssign,
		"*=": token.StarAssign, "/=": token.SlashAssign, "%=": token.PercentAssign,
		"^=": token.CaretAssign, "++": token.Increment, "--": token.Decrement,
		"==": token.Equal, "!=": token.NotEqual, "<=": token.LessEqual, ">=": token.GreaterEqual,
	}
	if kind, ok := pairs[string([]byte{l.byteAt(0), l.byteAt(1)})]; ok {
		return kind, 2
	}
	singles := map[byte]token.Kind{
		':': token.Colon, '=': token.Assign, '+': token.Plus, '-': token.Minus,
		'*': token.Star, '/': token.Slash, '%': token.Percent, '^': token.Caret,
		'<': token.Less, '>': token.Greater, '.': token.Dot, ',': token.Comma,
		'(': token.LeftParen, ')': token.RightParen, '[': token.LeftBracket,
		']': token.RightBracket, '{': token.LeftBrace, '}': token.RightBrace,
		'?': token.Question, '#': token.Hash, '@': token.At,
	}
	if kind, ok := singles[l.byteAt(0)]; ok {
		return kind, 1
	}
	return token.Invalid, 0
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isHorizontalWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\f' || b == '\v'
}

func startsIdentifierAt(text string, offset int) bool {
	if offset < 0 || offset >= len(text) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(text[offset:])
	return isXIDStart(r)
}
