package lexer

import (
	"strings"

	"ahdcode/internal/source"
	"ahdcode/internal/syntax/token"
)

type stringDelimiter struct {
	quote  byte
	width  int
	triple bool
}

func (l *lexer) scanString() {
	delimiter := l.readStringDelimiter()
	start := l.position()
	l.advanceASCII(delimiter.width)
	l.emit(token.StringStart, start, string(delimiter.quote), false)

	textStart := l.position()
	var decoded strings.Builder
	flush := func() {
		if l.position().Offset == textStart.Offset {
			return
		}
		l.emit(token.StringText, textStart, decoded.String(), false)
		decoded.Reset()
		textStart = l.position()
	}

	for !l.atEnd() {
		if l.matchesDelimiter(delimiter) {
			flush()
			endStart := l.position()
			l.advanceASCII(delimiter.width)
			l.emit(token.StringEnd, endStart, string(delimiter.quote), false)
			return
		}
		if !delimiter.triple && l.isNewline() {
			flush()
			l.bag.Error(codeNewlineInString, "physical newline in ordinary string", source.NewSpan(l.file.ID, l.position(), l.position()), "close the string before the newline or use a triple string")
			l.emitSynthetic(token.StringEnd, l.position())
			return
		}
		if l.byteAt(0) == '\\' {
			l.scanEscape(&decoded)
			continue
		}
		if l.byteAt(0) == '{' {
			flush()
			opening := l.position()
			l.advanceASCII(1)
			l.emit(token.InterpolationStart, opening, "", false)
			context := &interpolationContext{triple: delimiter.triple, opening: opening}
			if !l.scanInterpolation(context) {
				l.bag.Error(codeUnterminatedString, "unterminated string", l.span(start), "add the matching string delimiter")
				l.emitSynthetic(token.StringEnd, l.position())
				return
			}
			textStart = l.position()
			continue
		}
		if l.byteAt(0) == '}' {
			braceStart := l.position()
			l.advanceASCII(1)
			decoded.WriteByte('}')
			l.bag.Error(codeUnmatchedStringBrace, "unmatched } in string text", l.span(braceStart), "escape a literal closing brace as \\}")
			continue
		}
		if l.isNewline() {
			if l.hasPrefix("\r\n") {
				decoded.WriteString("\r\n")
			} else {
				decoded.WriteByte(l.byteAt(0))
			}
			l.consumeNewline()
			continue
		}
		r, _ := l.advanceRune()
		decoded.WriteRune(r)
	}

	flush()
	l.bag.Error(codeUnterminatedString, "unterminated string", l.span(start), "add the matching string delimiter")
	l.emitSynthetic(token.StringEnd, l.position())
}

// scanRawString scans r"...", r'...', r"""...""", and r"'..."' literals.
// A raw String has no escape processing and no interpolation: backslashes
// and braces are ordinary characters, and the resulting value is copied
// verbatim from source. The StringStart token's span is widened to start at
// the r prefix, so the parser and formatter need no raw-specific handling --
// StringExpr.Span() already covers the prefix, and the formatter reproduces
// source text verbatim.
func (l *lexer) scanRawString() {
	prefixStart := l.position()
	l.advanceASCII(1)
	delimiter := l.readStringDelimiter()
	l.advanceASCII(delimiter.width)
	l.emit(token.StringStart, prefixStart, string(delimiter.quote), false)

	textStart := l.position()
	flush := func() {
		if l.position().Offset == textStart.Offset {
			return
		}
		l.emit(token.StringText, textStart, l.span(textStart).Text(l.file), false)
		textStart = l.position()
	}

	for !l.atEnd() {
		if l.matchesDelimiter(delimiter) {
			flush()
			endStart := l.position()
			l.advanceASCII(delimiter.width)
			l.emit(token.StringEnd, endStart, string(delimiter.quote), false)
			return
		}
		if !delimiter.triple && l.isNewline() {
			flush()
			l.bag.Error(codeNewlineInString, "physical newline in ordinary string", source.NewSpan(l.file.ID, l.position(), l.position()), "close the string before the newline or use a triple string")
			l.emitSynthetic(token.StringEnd, l.position())
			return
		}
		if l.isNewline() {
			l.consumeNewline()
			continue
		}
		l.advanceRune()
	}

	flush()
	l.bag.Error(codeUnterminatedString, "unterminated string", l.span(prefixStart), "add the matching string delimiter")
	l.emitSynthetic(token.StringEnd, l.position())
}

func (l *lexer) readStringDelimiter() stringDelimiter {
	quote := l.byteAt(0)
	tripleText := string([]byte{quote, quote, quote})
	if l.hasPrefix(tripleText) {
		return stringDelimiter{quote: quote, width: 3, triple: true}
	}
	return stringDelimiter{quote: quote, width: 1}
}

func (l *lexer) matchesDelimiter(delimiter stringDelimiter) bool {
	if delimiter.triple {
		return l.hasPrefix(string([]byte{delimiter.quote, delimiter.quote, delimiter.quote}))
	}
	return l.byteAt(0) == delimiter.quote
}

func (l *lexer) scanEscape(decoded *strings.Builder) {
	start := l.position()
	l.advanceASCII(1)
	if l.atEnd() {
		l.bag.Error(codeInvalidEscape, "incomplete escape sequence", l.span(start), "add one of the supported escape characters")
		decoded.WriteByte('\\')
		return
	}
	if l.isNewline() {
		l.bag.Error(codeInvalidEscape, "physical newline cannot be escaped in a string", l.span(start), "use \\n for a newline value or a triple string for physical newlines")
		return
	}
	escaped := l.byteAt(0)
	l.advanceASCII(1)
	switch escaped {
	case 'n':
		decoded.WriteByte('\n')
	case 'r':
		decoded.WriteByte('\r')
	case 't':
		decoded.WriteByte('\t')
	case '\\', '"', '\'', '{', '}':
		decoded.WriteByte(escaped)
	default:
		l.bag.Error(codeInvalidEscape, "unsupported escape sequence", l.span(start), "use only \\n, \\r, \\t, \\\\, \\\", \\', \\{, or \\}")
		decoded.WriteByte(escaped)
	}
}

func (l *lexer) scanInterpolation(context *interpolationContext) bool {
	for !l.atEnd() {
		lineBeforeTrivia := l.line
		l.scanTrivia()
		if !context.triple && l.line != lineBeforeTrivia {
			l.bag.Error(codeNewlineInString, "physical newline in ordinary string interpolation", source.NewSpan(l.file.ID, context.opening, l.position()), "use a triple string when interpolation syntax spans physical lines")
			l.emitSynthetic(token.InterpolationEnd, l.position())
			return false
		}
		if l.atEnd() {
			l.reportUnterminatedInterpolation(context)
			return false
		}
		if l.isNewline() {
			if !context.triple {
				l.bag.Error(codeNewlineInString, "physical newline in ordinary string interpolation", source.NewSpan(l.file.ID, l.position(), l.position()), "close the interpolation and string before the newline")
				l.emitSynthetic(token.InterpolationEnd, l.position())
				return false
			}
			l.emitNewline()
			continue
		}
		if l.byteAt(0) == '}' && context.braceDepth == 0 {
			if !context.sawExpressionToken {
				l.bag.Error(codeEmptyInterpolation, "empty string interpolation", source.NewSpan(l.file.ID, context.opening, l.position()), "place one expression between the braces")
			}
			start := l.position()
			l.advanceASCII(1)
			l.emit(token.InterpolationEnd, start, "", false)
			return true
		}
		l.scanCodeToken(context)
	}
	l.reportUnterminatedInterpolation(context)
	return false
}

func (l *lexer) reportUnterminatedInterpolation(context *interpolationContext) {
	l.bag.Error(codeUnterminatedInterpolation, "unterminated string interpolation", source.NewSpan(l.file.ID, context.opening, l.position()), "add } to close the interpolation")
	l.emitSynthetic(token.InterpolationEnd, l.position())
}
