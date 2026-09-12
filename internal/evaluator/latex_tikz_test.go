package evaluator

import (
	"testing"

	"ahdcode/internal/backend/golang/ahdruntime"
)

// The evaluator's Latex implementation is separate from the native runtime for
// most helpers, so the v1.2.0 additions are compared byte for byte against the
// runtime a compiled program uses.

func TestLatexTikZFragmentsMatchNativeRuntime(t *testing.T) {
	session := newLatexTestSession()
	source := "\\draw[thick] (0,0) -- ++(2,1) node[above] {\\textbf{Türkçe} 50\\%};\n% {x} [y]\n"
	libraries := &List{Items: []any{"pgfornament", "calc"}}

	for _, overlay := range []bool{false, true} {
		name := "tikz"
		if overlay {
			name = "overlay"
		}
		got := session.latexBuiltin(name, []any{source, libraries}).(string)
		want, _ := ahdruntime.AhdLatexTikZText(name, source, []string{"pgfornament", "calc"}, overlay)
		if got != want {
			t.Fatalf("evaluator %s\n got %q\nwant %q", name, got, want)
		}
	}

	defaultBorder := session.latexBuiltin("border", []any{nil, nil, nil}).(string)
	if want, _ := ahdruntime.AhdLatexBorderText(1, 1, ""); defaultBorder != want {
		t.Fatalf("evaluator default border\n got %q\nwant %q", defaultBorder, want)
	}
	colored := session.latexBuiltin("border", []any{1.6, 0.6, "#1F4E79"}).(string)
	if want, _ := ahdruntime.AhdLatexBorderText(1.6, 0.6, "#1F4E79"); colored != want {
		t.Fatalf("evaluator colored border\n got %q\nwant %q", colored, want)
	}
}

func TestLatexDocumentLandscapeAndTikZMatchNativeRuntime(t *testing.T) {
	session := newLatexTestSession()
	body, _ := ahdruntime.AhdLatexTikZText("overlay", "\\node {x};", []string{"positioning", "pgfornament"}, true)
	cover, _ := ahdruntime.AhdLatexBorderText(1, 2, "#B08D57")
	empty := ahdruntime.AhdBuildPair([]string{}, []string{})
	cases := []struct {
		args      []any
		kind      string
		landscape bool
		body      string
		cover     string
	}{
		{[]any{"Body", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil}, "Article", false, "Body", ""},
		{[]any{body, "Title", nil, nil, "Report", nil, "#1F4E79", cover, nil, nil, true}, "Report", true, body, cover},
		{[]any{body, nil, nil, nil, nil, nil, nil, nil, nil, nil, false}, "Article", false, body, ""},
	}
	for index, test := range cases {
		got := session.latexBuiltin("document", test.args).(string)
		title, color := "", ""
		if test.args[1] != nil {
			title = test.args[1].(string)
		}
		if test.args[6] != nil {
			color = test.args[6].(string)
		}
		want := ahdruntime.AhdLatexDocumentFull(test.body, title, "", "", test.kind, 2.54, color, test.cover, empty, "Default", test.landscape)
		if got != want {
			t.Fatalf("case %d: evaluator document differs from the native runtime\n got %q\nwant %q", index, got, want)
		}
	}
}

func TestLatexVectorValidationMatchesNativeRuntime(t *testing.T) {
	session := newLatexTestSession()
	expectEvaluatorRaise(t, "ValueError", func() {
		session.latexBuiltin("tikz", []any{"x", &List{Items: []any{"shadows"}}})
	})
	expectEvaluatorRaise(t, "ValueError", func() {
		session.latexBuiltin("border", []any{-1.0, nil, nil})
	})
	expectEvaluatorRaise(t, "ValueError", func() {
		session.latexBuiltin("border", []any{nil, 0.0, nil})
	})
	expectEvaluatorRaise(t, "ValueError", func() {
		session.latexBuiltin("border", []any{nil, nil, "navy"})
	})
	expectEvaluatorRaise(t, "ValueError", func() {
		session.latexBuiltin("document", []any{"Body", nil, nil, nil, "Beamer", nil, nil, nil, nil, nil, true})
	})
	got := evaluatorRaisedMessage(t, "ValueError", func() {
		session.latexBuiltin("overlay", []any{"x", &List{Items: []any{"Calc"}}})
	})
	_, want := ahdruntime.AhdLatexTikZText("overlay", "x", []string{"Calc"}, true)
	if got != want {
		t.Fatalf("evaluator message %q, native %q", got, want)
	}
}
