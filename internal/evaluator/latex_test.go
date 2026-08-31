package evaluator

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func newLatexTestSession() *Session {
	return New(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, "")
}

func expectEvaluatorRaise(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		failure, ok := recovered.(raised)
		if !ok {
			t.Fatalf("expected a %s raise, got %#v", name, recovered)
		}
		if failure.failure.Name != name {
			t.Fatalf("expected a %s raise, got %s", name, failure.failure.Name)
		}
	}()
	fn()
}

func TestLatexTableMathColumnBoundsMatchNativeRuntime(t *testing.T) {
	s := newLatexTestSession()
	headers := &List{Items: []any{"A", "B"}}
	rows := &List{Items: []any{&List{Items: []any{"1", "2"}}}}

	if got := s.latexBuiltin("table", []any{headers, rows, &List{Items: []any{int64(0), int64(1)}}}); !strings.Contains(got.(string), `\(1\)`) {
		t.Fatalf("in-range math column was not applied: %q", got)
	}
	expectEvaluatorRaise(t, "ValueError", func() {
		s.latexBuiltin("table", []any{headers, rows, &List{Items: []any{int64(5)}}})
	})
	expectEvaluatorRaise(t, "ValueError", func() {
		s.latexBuiltin("table", []any{headers, rows, &List{Items: []any{int64(-1)}}})
	})
}

func TestLatexDocumentRejectsUndeclaredTheoremType(t *testing.T) {
	s := newLatexTestSession()
	body := "\\begin{ahdthmdeadbeefcafe}\nStatement.\n\\end{ahdthmdeadbeefcafe}\n"
	expectEvaluatorRaise(t, "ValueError", func() {
		s.latexBuiltin("document", []any{body, "", "", "", "Article", nil, "", "", &Pair{}})
	})
}

func TestLatexDocumentAcceptsDeclaredTheoremType(t *testing.T) {
	s := newLatexTestSession()
	theorems := &Pair{Keys: []any{"Theorem"}, Values: map[any]any{"Theorem": ""}}
	theorem := s.latexBuiltin("theorem", []any{"Theorem", "Every proof needs a claim.", ""}).(string)
	got := s.latexBuiltin("document", []any{theorem, "", "", "", "Article", nil, "", "", theorems}).(string)
	if !strings.Contains(got, "\\newtheorem{") || !strings.Contains(got, "Every proof needs a claim.") {
		t.Fatalf("declared theorem type was rejected:\n%s", got)
	}
}

func TestLatexDocumentRejectsChapterCounterOutsideReport(t *testing.T) {
	s := newLatexTestSession()
	theorems := &Pair{Keys: []any{"Theorem"}, Values: map[any]any{"Theorem": "chapter"}}
	expectEvaluatorRaise(t, "ValueError", func() {
		s.latexBuiltin("document", []any{"", "", "", "", "Article", nil, "", "", theorems})
	})
}
