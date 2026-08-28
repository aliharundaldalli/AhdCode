package token

import "ahdcode/internal/source"

// TriviaKind identifies non-semantic source text retained for formatting.
type TriviaKind uint8

const (
	WhitespaceTrivia TriviaKind = iota
	LineCommentTrivia
	BlockCommentTrivia
)

// Trivia retains exact source text and placement. A multiline block comment
// may be split at newline token boundaries; concatenating stream lexemes still
// reconstructs the original source.
type Trivia struct {
	Kind   TriviaKind
	Lexeme string
	Span   source.Span
}

// Token is one lexical token. Value contains normalized identifier text or
// decoded StringText; Lexeme always preserves original source spelling.
type Token struct {
	Kind          Kind
	Lexeme        string
	Value         string
	Span          source.Span
	LeadingTrivia []Trivia
	Synthetic     bool
}
