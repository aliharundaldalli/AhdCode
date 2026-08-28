package lexer

import (
	"testing"

	"ahdcode/internal/source"
	"ahdcode/internal/syntax/token"
)

func FuzzLexNeverPanics(f *testing.F) {
	for _, seed := range [][]byte{
		{},
		[]byte("öğrenci: Int := 5\n"),
		[]byte(`"hello {name}"`),
		[]byte("/* unterminated"),
		{0xff, 0xfe, '{', '}'},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		result := Lex(source.NewFile(1, "fuzz.ahd", string(data)))
		if len(result.Tokens) == 0 || result.Tokens[len(result.Tokens)-1].Kind != token.EOF {
			t.Fatal("lexer result does not end in EOF")
		}
		previous := 0
		for _, item := range result.Tokens {
			if item.Span.Start.Offset < previous || item.Span.End.Offset < item.Span.Start.Offset || item.Span.End.Offset > len(data) {
				t.Fatalf("invalid or non-monotonic token span: %+v after %d", item.Span, previous)
			}
			previous = item.Span.End.Offset
		}
	})
}
