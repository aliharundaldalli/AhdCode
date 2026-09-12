package evaluator

import (
	"reflect"
	"testing"

	"ahdcode/internal/semantic"
)

// Expected values come from the Unicode Character Database, not from the
// implementation: U+015F ş is Ll, U+0130 İ is Lu, U+0131 ı is Ll, U+1F60A 😊 is
// So, U+0663 ٣ is Nd, U+00B2 ² is No, and U+0301 is a combining mark (Mn).

// evaluatorRaisedMessage runs fn, requires it to raise the named AhdCode error,
// and returns that error's message.
func evaluatorRaisedMessage(t *testing.T, name string, fn func()) (message string) {
	t.Helper()
	defer func() {
		failure, ok := recover().(raised)
		if !ok || failure.failure == nil || failure.failure.Name != name {
			t.Fatalf("expected a %s raise, got %#v", name, failure)
		}
		message = failure.failure.Message
	}()
	fn()
	return ""
}

func charactersList(t *testing.T, session *Session, text string) []any {
	t.Helper()
	return session.charactersBuiltin("list", []any{text}).(*List).Items
}

func TestCharactersListIsCodePointOrder(t *testing.T) {
	session := newSecurityTestSession()
	cases := []struct {
		text string
		want []any
	}{
		{"", []any{}},
		{"abc", []any{"a", "b", "c"}},
		{"çğıöşü", []any{"ç", "ğ", "ı", "ö", "ş", "ü"}},
		{"😊", []any{"😊"}},
		{"Aş😊", []any{"A", "ş", "😊"}},
		// e + U+0301 displays as one glyph but is two code points.
		{"e\u0301", []any{"e", "\u0301"}},
		{"a\nb", []any{"a", "\n", "b"}},
		{" \t ", []any{" ", "\t", " "}},
	}
	for _, test := range cases {
		got := charactersList(t, session, test.text)
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("list(%q) = %#v, want %#v", test.text, got, test.want)
		}
	}
}

func TestCharactersCountIsCodePoints(t *testing.T) {
	session := newSecurityTestSession()
	cases := []struct {
		text string
		want int64
	}{
		{"", 0},
		{"AhdCode", 7},
		{"çğıöşü", 6}, // twelve UTF-8 bytes
		{"😊", 1},      // four UTF-8 bytes
		{"Aş😊", 3},
		{"e\u0301", 2},
	}
	for _, test := range cases {
		if got := session.charactersBuiltin("count", []any{test.text}).(int64); got != test.want {
			t.Fatalf("count(%q) = %d, want %d", test.text, got, test.want)
		}
	}
}

func TestCharactersCodePointRoundTrip(t *testing.T) {
	session := newSecurityTestSession()
	cases := []struct {
		character string
		value     int64
	}{
		{"A", 65},
		{"ş", 351},
		{"İ", 304},
		{"😊", 128522},
		{"\u0301", 769},
	}
	for _, test := range cases {
		if got := session.charactersBuiltin("codePoint", []any{test.character}).(int64); got != test.value {
			t.Fatalf("codePoint(%q) = %d, want %d", test.character, got, test.value)
		}
		if got := session.charactersBuiltin("fromCodePoint", []any{test.value}).(string); got != test.character {
			t.Fatalf("fromCodePoint(%d) = %q, want %q", test.value, got, test.character)
		}
	}
	if got := session.charactersBuiltin("fromCodePoint", []any{int64(0x10FFFF)}).(string); got != "\U0010FFFF" {
		t.Fatalf("fromCodePoint(0x10FFFF) = %q", got)
	}
	if got := session.charactersBuiltin("fromCodePoint", []any{int64(0)}).(string); got != "\x00" {
		t.Fatalf("fromCodePoint(0) = %q", got)
	}
}

func TestCharactersRejectsNonScalarValues(t *testing.T) {
	session := newSecurityTestSession()
	for _, value := range []int64{-1, -9223372036854775808, 0x110000, 9223372036854775807, 0xD800, 0xDBFF, 0xDC00, 0xDFFF} {
		value := value
		expectEvaluatorRaise(t, "CharactersError", func() {
			session.charactersBuiltin("fromCodePoint", []any{value})
		})
	}
}

// A one-character operation never inspects only the first code point.
func TestCharactersRequireExactlyOneCodePoint(t *testing.T) {
	session := newSecurityTestSession()
	operations := append([]string{"codePoint"}, semantic.CharactersClassifiers...)
	for _, name := range operations {
		for _, text := range []string{"", "AB", "hello", "e\u0301", "😊😊"} {
			name, text := name, text
			expectEvaluatorRaise(t, "CharactersError", func() {
				session.charactersBuiltin(name, []any{text})
			})
		}
	}
}

func TestCharactersClassification(t *testing.T) {
	session := newSecurityTestSession()
	classify := func(name, character string) bool {
		return session.charactersBuiltin(name, []any{character}).(bool)
	}
	type row struct {
		character                                                string
		letter, digit, space, upper, lower, alnum, punct, symbol bool
	}
	rows := []row{
		{"A", true, false, false, true, false, true, false, false},
		{"z", true, false, false, false, true, true, false, false},
		{"ç", true, false, false, false, true, true, false, false},
		{"Ç", true, false, false, true, false, true, false, false},
		{"ğ", true, false, false, false, true, true, false, false},
		{"Ğ", true, false, false, true, false, true, false, false},
		{"ı", true, false, false, false, true, true, false, false},
		{"İ", true, false, false, true, false, true, false, false},
		{"ö", true, false, false, false, true, true, false, false},
		{"Ö", true, false, false, true, false, true, false, false},
		{"ş", true, false, false, false, true, true, false, false},
		{"Ş", true, false, false, true, false, true, false, false},
		{"ü", true, false, false, false, true, true, false, false},
		{"Ü", true, false, false, true, false, true, false, false},
		{"ж", true, false, false, false, true, true, false, false},  // Cyrillic Ll
		{"語", true, false, false, false, false, true, false, false}, // Lo: neither case
		{"7", false, true, false, false, false, true, false, false},
		{"٣", false, true, false, false, false, true, false, false},   // Arabic-Indic digit, Nd
		{"²", false, false, false, false, false, false, false, false}, // No, not a digit
		{" ", false, false, true, false, false, false, false, false},
		{"\t", false, false, true, false, false, false, false, false},
		{"\n", false, false, true, false, false, false, false, false},
		{"\u00a0", false, false, true, false, false, false, false, false}, // no-break space
		{"!", false, false, false, false, false, false, true, false},
		{"–", false, false, false, false, false, false, true, false}, // en dash, Pd
		{"«", false, false, false, false, false, false, true, false},
		{"+", false, false, false, false, false, false, false, true},       // Sm
		{"$", false, false, false, false, false, false, false, true},       // Sc
		{"😊", false, false, false, false, false, false, false, true},       // So
		{"\u0301", false, false, false, false, false, false, false, false}, // Mn
	}
	for _, test := range rows {
		checks := []struct {
			name string
			want bool
		}{
			{"isLetter", test.letter}, {"isDigit", test.digit}, {"isWhitespace", test.space},
			{"isUpper", test.upper}, {"isLower", test.lower}, {"isAlphaNumeric", test.alnum},
			{"isPunctuation", test.punct}, {"isSymbol", test.symbol},
		}
		for _, check := range checks {
			if got := classify(check.name, test.character); got != check.want {
				t.Fatalf("%s(%q) = %v, want %v", check.name, test.character, got, check.want)
			}
		}
	}
}

func TestCharactersErrorMessagesAreStable(t *testing.T) {
	session := newSecurityTestSession()
	cases := []struct {
		name    string
		args    []any
		message string
	}{
		{"codePoint", []any{""}, "Characters.codePoint requires exactly one character; received an empty String"},
		{"isLetter", []any{"AB"}, "Characters.isLetter requires exactly one character; received 2 characters"},
		{"fromCodePoint", []any{int64(-1)}, "Characters.fromCodePoint: -1 is outside the Unicode range 0..1114111"},
		{"fromCodePoint", []any{int64(55296)}, "Characters.fromCodePoint: 55296 is a surrogate code point, not a Unicode scalar value"},
	}
	for _, test := range cases {
		got := evaluatorRaisedMessage(t, "CharactersError", func() {
			session.charactersBuiltin(test.name, test.args)
		})
		if got != test.message {
			t.Fatalf("%s%v message = %q, want %q", test.name, test.args, got, test.message)
		}
	}
}
