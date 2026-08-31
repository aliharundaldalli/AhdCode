package lexer

import (
	"strings"
	"testing"

	"ahdcode/internal/source"
	"ahdcode/internal/syntax/token"
)

func lexText(text string) Result {
	return Lex(source.NewFile(1, "test.ahd", text))
}

func tokenKinds(result Result) []token.Kind {
	kinds := make([]token.Kind, len(result.Tokens))
	for i, item := range result.Tokens {
		kinds[i] = item.Kind
	}
	return kinds
}

func assertKinds(t *testing.T, result Result, want ...token.Kind) {
	t.Helper()
	got := tokenKinds(result)
	if len(got) != len(want) {
		t.Fatalf("token kinds = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token[%d] = %v; want %v; all=%v", i, got[i], want[i], got)
		}
	}
}

func assertNoDiagnostics(t *testing.T, result Result) {
	t.Helper()
	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
}

func diagnosticCodes(result Result) []string {
	codes := make([]string, len(result.Diagnostics))
	for i, diagnostic := range result.Diagnostics {
		codes[i] = diagnostic.Code
	}
	return codes
}

func reconstruct(result Result) string {
	var builder strings.Builder
	for _, item := range result.Tokens {
		for _, trivia := range item.LeadingTrivia {
			builder.WriteString(trivia.Lexeme)
		}
		if !item.Synthetic {
			builder.WriteString(item.Lexeme)
		}
	}
	return builder.String()
}

func TestUnicodeIdentifiersNormalizeNFKCWithoutCaseFolding(t *testing.T) {
	result := lexText("öğrenci2 Ａli Ali ali structure attribute SuperClass")
	assertNoDiagnostics(t, result)
	assertKinds(t, result,
		token.Identifier, token.Identifier, token.Identifier, token.Identifier,
		token.Identifier, token.Identifier, token.Identifier, token.EOF,
	)
	values := []string{"öğrenci2", "Ali", "Ali", "ali", "structure", "attribute", "SuperClass"}
	for i, want := range values {
		if result.Tokens[i].Value != want {
			t.Fatalf("token[%d].Value = %q; want %q", i, result.Tokens[i].Value, want)
		}
	}
}

func TestNFKCNormalizedKeywordIsReserved(t *testing.T) {
	result := lexText("Ｉｎｔ")
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.KeywordInt, token.EOF)
	if result.Tokens[0].Value != "Int" {
		t.Fatalf("normalized keyword value = %q", result.Tokens[0].Value)
	}
}

func TestInvalidUTF8IsLexicalError(t *testing.T) {
	result := lexText(string([]byte{'x', ' ', 0xff, ' ', 'y'}))
	if got := diagnosticCodes(result); len(got) != 1 || got[0] != codeInvalidUTF8 {
		t.Fatalf("diagnostics = %v", got)
	}
	assertKinds(t, result, token.Identifier, token.Identifier, token.EOF)
}

func TestXIDStartAndContinueBoundaries(t *testing.T) {
	valid := lexText("a\u0301 a① ℘value")
	assertNoDiagnostics(t, valid)
	assertKinds(t, valid, token.Identifier, token.Identifier, token.Identifier, token.EOF)

	invalid := lexText("\u0301name 😀")
	if got := diagnosticCodes(invalid); strings.Join(got, ",") != "LEX002,LEX002" {
		t.Fatalf("invalid XID diagnostics = %v", got)
	}
}

func TestReservedKeywordsAndCaseSensitivity(t *testing.T) {
	result := lexText("and or not true false null lambda Lambda Int int Class class")
	assertNoDiagnostics(t, result)
	assertKinds(t, result,
		token.KeywordAnd, token.KeywordOr, token.KeywordNot,
		token.KeywordTrue, token.KeywordFalse, token.KeywordNull,
		token.KeywordLambda, token.Identifier,
		token.KeywordInt, token.Identifier, token.KeywordClass, token.Identifier, token.EOF,
	)
}

func TestNumericLiteralGrammarAndNoLexerRangeCheck(t *testing.T) {
	result := lexText("0 0012 1.25 1e6 1.2e-3 9223372036854775808 -9223372036854775808")
	assertNoDiagnostics(t, result)
	assertKinds(t, result,
		token.IntLiteral, token.IntLiteral, token.RealLiteral, token.RealLiteral,
		token.RealLiteral, token.IntLiteral, token.Minus, token.IntLiteral, token.EOF,
	)
	if got := result.Tokens[5].Lexeme; got != "9223372036854775808" {
		t.Fatalf("large Int lexeme = %q", got)
	}
}

func TestImaginaryLiteralRequiresImmediateUppercaseI(t *testing.T) {
	valid := lexText("3I 3.5I 1e2I")
	assertNoDiagnostics(t, valid)
	assertKinds(t, valid, token.ImaginaryLiteral, token.ImaginaryLiteral, token.ImaginaryLiteral, token.EOF)

	lowercase := lexText("3i")
	if got := diagnosticCodes(lowercase); len(got) != 1 || got[0] != codeInvalidNumericLiteral {
		t.Fatalf("lowercase suffix diagnostics = %v", got)
	}

	separated := lexText("3 I")
	assertNoDiagnostics(t, separated)
	assertKinds(t, separated, token.IntLiteral, token.Identifier, token.EOF)
}

func TestMalformedNumericLiterals(t *testing.T) {
	result := lexText("1e 1_000 123abc .5 5.")
	if got := diagnosticCodes(result); strings.Join(got, ",") != "LEX004,LEX004,LEX004,LEX004,LEX004" {
		t.Fatalf("diagnostics = %v", got)
	}
	assertKinds(t, result, token.RealLiteral, token.IntLiteral, token.IntLiteral, token.RealLiteral, token.RealLiteral, token.EOF)
}

func TestFiveDotRemainsIntAndMemberDotTokens(t *testing.T) {
	result := lexText("5.member")
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.IntLiteral, token.Dot, token.Identifier, token.EOF)
}

func TestOperatorsUseLongestMatch(t *testing.T) {
	text := ": := = -> + - * / % ^ += -= *= /= %= ^= ++ -- == != < <= > >= . , ( ) [ ] { }"
	result := lexText(text)
	assertNoDiagnostics(t, result)
	assertKinds(t, result,
		token.Colon, token.Declare, token.Assign, token.Arrow, token.Plus, token.Minus,
		token.Star, token.Slash, token.Percent, token.Caret, token.PlusAssign,
		token.MinusAssign, token.StarAssign, token.SlashAssign, token.PercentAssign,
		token.CaretAssign, token.Increment, token.Decrement, token.Equal, token.NotEqual,
		token.Less, token.LessEqual, token.Greater, token.GreaterEqual, token.Dot,
		token.Comma, token.LeftParen, token.RightParen, token.LeftBracket,
		token.RightBracket, token.LeftBrace, token.RightBrace, token.EOF,
	)
}

func TestNewlinesAreRetainedWithRuneColumns(t *testing.T) {
	result := lexText("ö\r\nx\ny")
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.Identifier, token.Newline, token.Identifier, token.Newline, token.Identifier, token.EOF)
	if got := result.Tokens[0].Span.End.Column; got != 2 {
		t.Fatalf("Unicode identifier end column = %d; want 2", got)
	}
	if got := result.Tokens[2].Span.Start; got.Line != 2 || got.Column != 1 {
		t.Fatalf("x start = %+v; want line 2 column 1", got)
	}
	if got := result.Tokens[1].Lexeme; got != "\r\n" {
		t.Fatalf("CRLF lexeme = %q", got)
	}
}

func TestCommentsAreTriviaAndMultilineCommentsDoNotNest(t *testing.T) {
	text := "x /* outer /* inner */ y // tail\n/*a\nb*/z"
	result := lexText(text)
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.Identifier, token.Identifier, token.Newline, token.Newline, token.Identifier, token.EOF)
	if got := result.Tokens[1].LeadingTrivia[1].Lexeme; got != "/* outer /* inner */" {
		t.Fatalf("block comment trivia = %q", got)
	}
	if got := reconstruct(result); got != text {
		t.Fatalf("reconstructed source = %q; want %q", got, text)
	}
}

func TestUnterminatedMultilineComment(t *testing.T) {
	result := lexText("x /* never closes")
	if got := diagnosticCodes(result); len(got) != 1 || got[0] != codeUnterminatedBlockComment {
		t.Fatalf("diagnostics = %v", got)
	}
	if got := reconstruct(result); got != "x /* never closes" {
		t.Fatalf("reconstructed source = %q", got)
	}
}

func TestExactEscapeSetDecodesStringText(t *testing.T) {
	result := lexText(`"a\n\r\t\\\"\'\{\}"`)
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.StringStart, token.StringText, token.StringEnd, token.EOF)
	want := "a\n\r\t\\\"'{}"
	if got := result.Tokens[1].Value; got != want {
		t.Fatalf("decoded StringText = %q; want %q", got, want)
	}
}

func TestUnsupportedEscapeIsError(t *testing.T) {
	result := lexText(`"\q"`)
	if got := diagnosticCodes(result); len(got) != 1 || got[0] != codeInvalidEscape {
		t.Fatalf("diagnostics = %v", got)
	}
}

func TestTripleStringPreservesContentWithoutDedentOrTrim(t *testing.T) {
	result := lexText("\"\"\"\n  first\nlast\n\"\"\"")
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.StringStart, token.StringText, token.StringEnd, token.EOF)
	if got, want := result.Tokens[1].Value, "\n  first\nlast\n"; got != want {
		t.Fatalf("triple content = %q; want %q", got, want)
	}
}

func TestTripleStringPreservesPhysicalCRLF(t *testing.T) {
	result := lexText("\"\"\"a\r\nb\"\"\"")
	assertNoDiagnostics(t, result)
	if got, want := result.Tokens[1].Value, "a\r\nb"; got != want {
		t.Fatalf("triple CRLF content = %q; want %q", got, want)
	}
}

func TestEscapedQuoteDoesNotCloseTripleString(t *testing.T) {
	result := lexText(`"""a\"""b"""`)
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.StringStart, token.StringText, token.StringEnd, token.EOF)
	if got := result.Tokens[1].Value; got != `a"""b` {
		t.Fatalf("triple content = %q", got)
	}
}

func TestQuoteMismatchAndOrdinaryNewlineFailCleanly(t *testing.T) {
	mismatch := lexText(`"hello'`)
	if got := diagnosticCodes(mismatch); len(got) != 1 || got[0] != codeUnterminatedString {
		t.Fatalf("quote mismatch diagnostics = %v", got)
	}

	newline := lexText("\"hello\nnext")
	if got := diagnosticCodes(newline); len(got) == 0 || got[0] != codeNewlineInString {
		t.Fatalf("newline diagnostics = %v", got)
	}
}

func TestInterpolationTokenizesNestedExpression(t *testing.T) {
	result := lexText(`"Hello {user != null and user.age >= 18}: {make({"x": "value {x}"})}"`)
	assertNoDiagnostics(t, result)
	wantContains := []token.Kind{
		token.StringStart, token.StringText, token.InterpolationStart, token.Identifier,
		token.NotEqual, token.KeywordNull, token.KeywordAnd, token.Identifier, token.Dot,
		token.Identifier, token.GreaterEqual, token.IntLiteral, token.InterpolationEnd,
		token.StringText, token.InterpolationStart, token.Identifier, token.LeftParen,
		token.LeftBrace, token.StringStart, token.StringText, token.StringEnd, token.Colon,
		token.StringStart, token.InterpolationStart, token.LeftBrace,
	}
	got := tokenKinds(result)
	for _, kind := range wantContains {
		found := false
		for _, candidate := range got {
			if candidate == kind {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected token kind %v in %v", kind, got)
		}
	}
}

func TestEscapedAndMalformedInterpolationBraces(t *testing.T) {
	escaped := lexText(`"\{literal\}"`)
	assertNoDiagnostics(t, escaped)
	if got := escaped.Tokens[1].Value; got != "{literal}" {
		t.Fatalf("escaped braces value = %q", got)
	}

	tests := []struct {
		text string
		code string
	}{
		{`"{}"`, codeEmptyInterpolation},
		{`"}"`, codeUnmatchedStringBrace},
		{`"{user`, codeUnterminatedInterpolation},
	}
	for _, test := range tests {
		result := lexText(test.text)
		codes := diagnosticCodes(result)
		if len(codes) == 0 || codes[0] != test.code {
			t.Fatalf("Lex(%q) diagnostics = %v; want first %s", test.text, codes, test.code)
		}
	}
}

func TestTripleInterpolationMaySpanLines(t *testing.T) {
	result := lexText("\"\"\"{\nvalue\n}\"\"\"")
	assertNoDiagnostics(t, result)
	assertKinds(t, result,
		token.StringStart, token.InterpolationStart, token.Newline, token.Identifier,
		token.Newline, token.InterpolationEnd, token.StringEnd, token.EOF,
	)
}

func TestOrdinaryInterpolationCannotHideNewlineInComment(t *testing.T) {
	result := lexText("\"{/* a\nb */ value}\"")
	codes := diagnosticCodes(result)
	if len(codes) == 0 || codes[0] != codeNewlineInString {
		t.Fatalf("diagnostics = %v", codes)
	}
}

func TestCommentsContainingFakeSyntaxDoNotLeakTokens(t *testing.T) {
	result := lexText("// := \"fake {x}\"\nvalue/* ++ null */")
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.Newline, token.Identifier, token.EOF)
}

func TestValidLexingReconstructsExactSource(t *testing.T) {
	text := "from Mathematics bring (\n  sqrt // keep\n)\nvalue: String := \"x {sqrt(4)}\""
	result := lexText(text)
	assertNoDiagnostics(t, result)
	if got := reconstruct(result); got != text {
		t.Fatalf("reconstructed source = %q; want %q", got, text)
	}
}

func TestRawStringDoesNotDecodeEscapes(t *testing.T) {
	// A raw String cannot contain its own delimiter (nothing escapes it, so a
	// quote always closes the literal) -- this exercises every other escape
	// form staying literal: \n, \t, and a doubled backslash.
	result := lexText(`r"\n\t\\{x}"`)
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.StringStart, token.StringText, token.StringEnd, token.EOF)
	want := `\n\t\\{x}`
	if got := result.Tokens[1].Value; got != want {
		t.Fatalf("raw StringText = %q; want %q", got, want)
	}
}

func TestRawStringHasNoInterpolation(t *testing.T) {
	result := lexText(`r"{name}"`)
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.StringStart, token.StringText, token.StringEnd, token.EOF)
	if got := result.Tokens[1].Value; got != "{name}" {
		t.Fatalf("raw StringText = %q; want %q", got, "{name}")
	}
}

func TestRawStringUnsupportedEscapeIsNotAnError(t *testing.T) {
	result := lexText(`r"\q"`)
	assertNoDiagnostics(t, result)
	if got := result.Tokens[1].Value; got != `\q` {
		t.Fatalf("raw StringText = %q; want %q", got, `\q`)
	}
}

func TestRawStringRegexQuantifierIsExact(t *testing.T) {
	result := lexText(`r"^MATH-[0-9]{3}$"`)
	assertNoDiagnostics(t, result)
	if got := result.Tokens[1].Value; got != "^MATH-[0-9]{3}$" {
		t.Fatalf("raw StringText = %q", got)
	}
}

func TestRawStringSingleQuoteForm(t *testing.T) {
	result := lexText(`r'abc\n{x}'`)
	assertNoDiagnostics(t, result)
	if got := result.Tokens[1].Value; got != `abc\n{x}` {
		t.Fatalf("raw StringText = %q", got)
	}
}

func TestRawTripleStringPreservesContentAndUnicode(t *testing.T) {
	result := lexText("r\"\"\"\n\\frac{x}{y} öğrenci\n\"\"\"")
	assertNoDiagnostics(t, result)
	want := "\n\\frac{x}{y} öğrenci\n"
	if got := result.Tokens[1].Value; got != want {
		t.Fatalf("raw triple content = %q; want %q", got, want)
	}
}

func TestRawTripleSingleQuoteForm(t *testing.T) {
	result := lexText("r'''\\frac{x}{y}'''")
	assertNoDiagnostics(t, result)
	if got := result.Tokens[1].Value; got != `\frac{x}{y}` {
		t.Fatalf("raw triple content = %q", got)
	}
}

func TestRawStringSpanIncludesPrefix(t *testing.T) {
	result := lexText(`r"abc"`)
	assertNoDiagnostics(t, result)
	if got := result.Tokens[0].Lexeme; got != `r"` {
		t.Fatalf("StringStart lexeme = %q; want %q", got, `r"`)
	}
}

func TestEmptyRawString(t *testing.T) {
	result := lexText(`r""`)
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.StringStart, token.StringEnd, token.EOF)
}

func TestUnterminatedRawStringDiagnostics(t *testing.T) {
	singleQuote := lexText(`r'unterminated`)
	if got := diagnosticCodes(singleQuote); len(got) != 1 || got[0] != codeUnterminatedString {
		t.Fatalf("raw single-quote diagnostics = %v", got)
	}

	doubleQuote := lexText(`r"unterminated`)
	if got := diagnosticCodes(doubleQuote); len(got) != 1 || got[0] != codeUnterminatedString {
		t.Fatalf("raw double-quote diagnostics = %v", got)
	}

	triple := lexText(`r"""unterminated`)
	if got := diagnosticCodes(triple); len(got) != 1 || got[0] != codeUnterminatedString {
		t.Fatalf("raw triple-quote diagnostics = %v", got)
	}

	newline := lexText("r\"hello\nnext")
	if got := diagnosticCodes(newline); len(got) == 0 || got[0] != codeNewlineInString {
		t.Fatalf("raw newline diagnostics = %v", got)
	}
}

func TestUppercaseRIsNotARawStringPrefix(t *testing.T) {
	result := lexText(`R"not raw"`)
	assertNoDiagnostics(t, result)
	assertKinds(t, result, token.Identifier, token.StringStart, token.StringText, token.StringEnd, token.EOF)
	if got := result.Tokens[0].Value; got != "R" {
		t.Fatalf("identifier value = %q; want %q", got, "R")
	}
}

func TestBareIdentifierNamedRIsUnaffected(t *testing.T) {
	result := lexText("r := 5\nr + 1")
	assertNoDiagnostics(t, result)
	assertKinds(t, result,
		token.Identifier, token.Declare, token.IntLiteral, token.Newline,
		token.Identifier, token.Plus, token.IntLiteral, token.EOF,
	)
}

func TestRawStringRoundTripsExactSource(t *testing.T) {
	text := "value: String := r\"x\\y{z}\" + r'''raw'''"
	result := lexText(text)
	assertNoDiagnostics(t, result)
	if got := reconstruct(result); got != text {
		t.Fatalf("reconstructed source = %q; want %q", got, text)
	}
}
