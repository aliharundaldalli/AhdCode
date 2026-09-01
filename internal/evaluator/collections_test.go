package evaluator

import (
	"reflect"
	"testing"
)

func evaluatorList(items ...any) *List { return &List{Items: items} }

func evaluatorPair(keys []any, values []any) *Pair {
	pair := &Pair{Values: make(map[any]any, len(keys))}
	for index, key := range keys {
		pair.Keys = append(pair.Keys, key)
		pair.Values[key] = values[index]
	}
	return pair
}

func pairValuesInOrder(pair *Pair) []any {
	result := make([]any, len(pair.Keys))
	for index, key := range pair.Keys {
		result[index] = pair.Values[key]
	}
	return result
}

// TestListsEvaluatorPurityAndOrdering checks the properties the shared
// native/REPL program cannot inspect directly: that a source collection is
// untouched, and that each returned collection is structurally independent.
func TestListsEvaluatorPurityAndOrdering(t *testing.T) {
	session := newLatexTestSession()
	source := evaluatorList(int64(1), int64(2), int64(3), int64(4), int64(5))

	chunks := session.listsBuiltin("chunk", []any{source, int64(2)}).(*List)
	chunks.Items[0].(*List).Items = append(chunks.Items[0].(*List).Items, int64(99))
	if !reflect.DeepEqual(source.Items, []any{int64(1), int64(2), int64(3), int64(4), int64(5)}) {
		t.Fatalf("chunk aliased or mutated its source: %v", source.Items)
	}

	rows := evaluatorList(evaluatorList(int64(1), int64(2)), evaluatorList(int64(3), int64(4)))
	flat := session.listsBuiltin("flatten", []any{rows}).(*List)
	flat.Items = append(flat.Items, int64(9))
	if len(rows.Items[0].(*List).Items) != 2 {
		t.Fatal("flatten aliased an inner List")
	}

	turned := session.listsBuiltin("transpose", []any{rows}).(*List)
	if got := turned.Items[0].(*List).Items; !reflect.DeepEqual(got, []any{int64(1), int64(3)}) {
		t.Fatalf("transpose = %v", got)
	}

	distinct := session.listsBuiltin("unique", []any{evaluatorList(int64(3), int64(1), int64(3))}).(*List)
	if !reflect.DeepEqual(distinct.Items, []any{int64(3), int64(1)}) {
		t.Fatalf("unique = %v", distinct.Items)
	}
	// null is one ordinary distinct value, not a skipped element.
	withNull := session.listsBuiltin("unique", []any{evaluatorList(int64(1), nil, int64(1), nil)}).(*List)
	if !reflect.DeepEqual(withNull.Items, []any{int64(1), nil}) {
		t.Fatalf("nullable unique = %v", withNull.Items)
	}

	counts := session.listsBuiltin("valueCounts", []any{evaluatorList("b", "a", "b")}).(*Pair)
	if !reflect.DeepEqual(counts.Keys, []any{"b", "a"}) ||
		!reflect.DeepEqual(pairValuesInOrder(counts), []any{int64(2), int64(1)}) {
		t.Fatalf("valueCounts = %v -> %v", counts.Keys, pairValuesInOrder(counts))
	}
}

func TestListsEvaluatorRaisesListsError(t *testing.T) {
	session := newLatexTestSession()
	source := evaluatorList(int64(1), int64(2))
	expectEvaluatorRaise(t, "ListsError", func() { session.listsBuiltin("chunk", []any{source, int64(0)}) })
	expectEvaluatorRaise(t, "ListsError", func() { session.listsBuiltin("chunk", []any{source, int64(-2)}) })
	expectEvaluatorRaise(t, "ListsError", func() {
		session.listsBuiltin("transpose", []any{evaluatorList(
			evaluatorList(int64(1), int64(2), int64(3)), evaluatorList(int64(4), int64(5)))})
	})
}

// TestKeyValueEvaluatorOrderingAndPurity pins the ordering contract of every
// KeyValue operation, and proves each result is a fresh Pair.
func TestKeyValueEvaluatorOrderingAndPurity(t *testing.T) {
	session := newLatexTestSession()
	base := evaluatorPair([]any{"a", "b", "c"}, []any{int64(1), int64(2), int64(3)})

	replaced := session.keyValueBuiltin("with", []any{base, "b", int64(9)}).(*Pair)
	if !reflect.DeepEqual(replaced.Keys, []any{"a", "b", "c"}) || replaced.Values["b"] != int64(9) {
		t.Fatalf("with moved an existing key: %v", replaced.Keys)
	}
	appended := session.keyValueBuiltin("with", []any{base, "d", int64(4)}).(*Pair)
	if !reflect.DeepEqual(appended.Keys, []any{"a", "b", "c", "d"}) {
		t.Fatalf("with did not append a new key last: %v", appended.Keys)
	}
	if base.Values["b"] != int64(2) || len(base.Keys) != 3 {
		t.Fatal("with mutated its source Pair")
	}

	picked := session.keyValueBuiltin("select", []any{base, evaluatorList("c", "a")}).(*Pair)
	if !reflect.DeepEqual(picked.Keys, []any{"c", "a"}) {
		t.Fatalf("select did not follow the requested order: %v", picked.Keys)
	}
	dropped := session.keyValueBuiltin("drop", []any{base, evaluatorList("b")}).(*Pair)
	if !reflect.DeepEqual(dropped.Keys, []any{"a", "c"}) {
		t.Fatalf("drop did not keep the source order: %v", dropped.Keys)
	}
	renamed := session.keyValueBuiltin("rename", []any{base, "b", "middle"}).(*Pair)
	if !reflect.DeepEqual(renamed.Keys, []any{"a", "middle", "c"}) {
		t.Fatalf("rename did not preserve the position: %v", renamed.Keys)
	}
	same := session.keyValueBuiltin("rename", []any{base, "b", "b"}).(*Pair)
	same.Keys = append(same.Keys, "z")
	if len(base.Keys) != 3 {
		t.Fatal("a no-op rename aliased its source")
	}

	overlaid := session.keyValueBuiltin("overlay", []any{
		base, evaluatorPair([]any{"b", "z"}, []any{int64(9), int64(26)})}).(*Pair)
	if !reflect.DeepEqual(overlaid.Keys, []any{"a", "b", "c", "z"}) ||
		!reflect.DeepEqual(pairValuesInOrder(overlaid), []any{int64(1), int64(9), int64(3), int64(26)}) {
		t.Fatalf("overlay = %v -> %v", overlaid.Keys, pairValuesInOrder(overlaid))
	}

	keys := session.keyValueBuiltin("keys", []any{base}).(*List)
	values := session.keyValueBuiltin("values", []any{base}).(*List)
	keys.Items = append(keys.Items, "injected")
	values.Items = append(values.Items, int64(0))
	if len(base.Keys) != 3 {
		t.Fatal("a keys/values snapshot reached the source Pair")
	}
	if !reflect.DeepEqual(values.Items[:3], []any{int64(1), int64(2), int64(3)}) {
		t.Fatalf("values = %v", values.Items)
	}
}

func TestKeyValueEvaluatorRaisesTheDocumentedErrorClasses(t *testing.T) {
	session := newLatexTestSession()
	base := evaluatorPair([]any{"a", "b"}, []any{int64(1), int64(2)})

	expectEvaluatorRaise(t, "KeyValueError", func() {
		session.keyValueBuiltin("combine", []any{evaluatorList("a", "b"), evaluatorList(int64(1))})
	})
	expectEvaluatorRaise(t, "KeyValueError", func() {
		session.keyValueBuiltin("combine", []any{evaluatorList("a", "a"), evaluatorList(int64(1), int64(2))})
	})
	expectEvaluatorRaise(t, "KeyValueError", func() {
		session.keyValueBuiltin("select", []any{base, evaluatorList("a", "a")})
	})
	expectEvaluatorRaise(t, "KeyValueError", func() {
		session.keyValueBuiltin("drop", []any{base, evaluatorList("a", "a")})
	})
	expectEvaluatorRaise(t, "KeyValueError", func() {
		session.keyValueBuiltin("rename", []any{base, "a", "b"})
	})
	expectEvaluatorRaise(t, "KeyValueError", func() {
		session.keyValueBuiltin("merge", []any{base, base})
	})
	// A genuinely missing key keeps the language's existing KeyError.
	expectEvaluatorRaise(t, "KeyError", func() { session.keyValueBuiltin("without", []any{base, "z"}) })
	expectEvaluatorRaise(t, "KeyError", func() { session.keyValueBuiltin("select", []any{base, evaluatorList("z")}) })
	expectEvaluatorRaise(t, "KeyError", func() { session.keyValueBuiltin("drop", []any{base, evaluatorList("z")}) })
	expectEvaluatorRaise(t, "KeyError", func() { session.keyValueBuiltin("rename", []any{base, "z", "y"}) })
}
