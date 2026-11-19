package evaluator

import (
	"strings"
	"testing"
)

func TestTemplatesEvaluatorEscapesAndRendersALayout(t *testing.T) {
	session := newLatexTestSession()
	set := session.templatesBuiltin("parse", []any{"page.html", "<p>{{.name}}</p>"})
	data := session.templatesBuiltin("object", []any{&Pair{
		Keys:   []any{"name"},
		Values: map[any]any{"name": session.templatesBuiltin("fromString", []any{"<Ayşe>"})},
	}})
	got := session.templatesOperation("TemplateSet.render", set, []any{"page.html", data}).(string)
	if got != "<p>&lt;Ayşe&gt;</p>" {
		t.Fatalf("render = %q", got)
	}
	if strings.Contains(got, "<Ayşe>") {
		t.Fatal("name was not escaped")
	}
}
