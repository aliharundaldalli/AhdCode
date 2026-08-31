package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONEvaluatorScalarConstructionAndAccessors(t *testing.T) {
	session := newLatexTestSession()
	if got := session.jsonBuiltin("nullValue", nil).(*Instance); session.jsonOperation("JSONValue.kind", got, nil) != "Null" {
		t.Fatalf("nullValue kind = %v", session.jsonOperation("JSONValue.kind", got, nil))
	}
	boolValue := session.jsonBuiltin("fromBool", []any{true})
	if session.jsonOperation("JSONValue.bool", boolValue, nil) != true {
		t.Fatal("fromBool/bool round trip")
	}
	intValue := session.jsonBuiltin("fromInt", []any{int64(91)})
	if session.jsonOperation("JSONValue.int", intValue, nil) != int64(91) {
		t.Fatal("fromInt/int round trip")
	}
	realValue := session.jsonBuiltin("fromReal", []any{5.0})
	if got := session.jsonValueText(realValue); got != "5.0" {
		t.Fatalf("fromReal(5.0) canonical text = %q, want 5.0 (must not read back as Int)", got)
	}
	if session.jsonOperation("JSONValue.real", intValue, nil) != float64(91) {
		t.Fatal("real() must accept Int widening")
	}
	stringValue := session.jsonBuiltin("fromString", []any{"hello"})
	if session.jsonOperation("JSONValue.string", stringValue, nil) != "hello" {
		t.Fatal("fromString/string round trip")
	}
}

func TestJSONEvaluatorFromRealRejectsNonFinite(t *testing.T) {
	session := newLatexTestSession()
	nan := 0.0
	nan = nan / nan
	expectEvaluatorRaise(t, "JSONError", func() {
		session.jsonBuiltin("fromReal", []any{nan})
	})
}

func TestJSONEvaluatorArrayAndObjectConstruction(t *testing.T) {
	session := newLatexTestSession()
	one := session.jsonBuiltin("fromInt", []any{int64(1)})
	two := session.jsonBuiltin("fromInt", []any{int64(2)})
	array := session.jsonBuiltin("array", []any{&List{Items: []any{one, two}}})
	if got := session.jsonValueText(array); got != "[1,2]" {
		t.Fatalf("array() canonical text = %q", got)
	}
	elements := session.jsonOperation("JSONValue.array", array, nil).(*List)
	if len(elements.Items) != 2 || session.jsonOperation("JSONValue.int", elements.Items[0], nil) != int64(1) {
		t.Fatalf("array() accessor = %v", elements.Items)
	}

	name := session.jsonBuiltin("fromString", []any{"Ali"})
	pair := &Pair{Keys: []any{"name"}, Values: map[any]any{"name": name}}
	object := session.jsonBuiltin("object", []any{pair})
	if got := session.jsonValueText(object); got != `{"name":"Ali"}` {
		t.Fatalf("object() canonical text = %q", got)
	}
	found := session.jsonOperation("JSONValue.get", object, []any{"name"})
	if found == nil || session.jsonOperation("JSONValue.string", found, nil) != "Ali" {
		t.Fatalf("get(present key) = %v", found)
	}
	missing := session.jsonOperation("JSONValue.get", object, []any{"missing"})
	if missing != nil {
		t.Fatalf("get(missing key) = %v, want nil", missing)
	}
}

func TestJSONEvaluatorParseOrderingAndDuplicateKeys(t *testing.T) {
	session := newLatexTestSession()
	value := session.jsonBuiltin("parse", []any{`{"z":1,"a":2}`})
	object := session.jsonOperation("JSONValue.object", value, nil).(*Pair)
	if len(object.Keys) != 2 || object.Keys[0] != "z" || object.Keys[1] != "a" {
		t.Fatalf("object() did not preserve insertion order: %v", object.Keys)
	}
	expectEvaluatorRaise(t, "JSONError", func() {
		session.jsonBuiltin("parse", []any{`{"a":1,"a":2}`})
	})
}

func TestJSONEvaluatorWrongKindAccessorsRaiseJSONError(t *testing.T) {
	session := newLatexTestSession()
	stringValue := session.jsonBuiltin("fromString", []any{"x"})
	expectEvaluatorRaise(t, "JSONError", func() {
		session.jsonOperation("JSONValue.int", stringValue, nil)
	})
	nullValue := session.jsonBuiltin("nullValue", nil)
	expectEvaluatorRaise(t, "JSONError", func() {
		session.jsonOperation("JSONValue.get", nullValue, []any{"k"})
	})
	arrayValue := session.jsonBuiltin("array", []any{&List{}})
	expectEvaluatorRaise(t, "JSONError", func() {
		session.jsonOperation("JSONValue.at", arrayValue, []any{int64(0)})
	})
}

func TestJSONEvaluatorAtIndexBoundsAndNegative(t *testing.T) {
	session := newLatexTestSession()
	one := session.jsonBuiltin("fromInt", []any{int64(10)})
	two := session.jsonBuiltin("fromInt", []any{int64(20)})
	array := session.jsonBuiltin("array", []any{&List{Items: []any{one, two}}})
	if got := session.jsonOperation("JSONValue.at", array, []any{int64(-1)}); session.jsonOperation("JSONValue.int", got, nil) != int64(20) {
		t.Fatalf("at(-1) = %v", got)
	}
	expectEvaluatorRaise(t, "JSONError", func() {
		session.jsonOperation("JSONValue.at", array, []any{int64(5)})
	})
}

func TestJSONEvaluatorStringifyCompactAndPretty(t *testing.T) {
	session := newLatexTestSession()
	value := session.jsonBuiltin("parse", []any{`{"a":1,"b":[1,2]}`})
	if got := session.jsonBuiltin("stringify", []any{value}); got != `{"a":1,"b":[1,2]}` {
		t.Fatalf("compact stringify = %v", got)
	}
	pretty := session.jsonBuiltin("stringify", []any{value, true}).(string)
	want := "{\n  \"a\": 1,\n  \"b\": [\n    1,\n    2\n  ]\n}"
	if pretty != want {
		t.Fatalf("pretty stringify = %q, want %q", pretty, want)
	}
}

func TestJSONEvaluatorFileIO(t *testing.T) {
	session := newLatexTestSession()
	directory := t.TempDir()
	path := filepath.Join(directory, "a b", "rapor.json")
	value := session.jsonBuiltin("fromBool", []any{true})
	if got := session.jsonBuiltin("write", []any{value, path, true}); got != Nothing {
		t.Fatalf("write returned %#v", got)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(content)) != "true" {
		t.Fatalf("written content = %q", string(content))
	}
	read := session.jsonBuiltin("read", []any{path})
	if session.jsonOperation("JSONValue.bool", read, nil) != true {
		t.Fatal("read did not round-trip the written file")
	}
	expectEvaluatorRaise(t, "JSONError", func() {
		session.jsonBuiltin("read", []any{filepath.Join(directory, "missing.json")})
	})
}

func TestJSONEvaluatorMalformedInputRaisesJSONError(t *testing.T) {
	session := newLatexTestSession()
	for _, source := range []string{"", "{", `{"a":1,}`, "01", "NaN", `"unterminated`} {
		source := source
		expectEvaluatorRaise(t, "JSONError", func() {
			session.jsonBuiltin("parse", []any{source})
		})
	}
}
