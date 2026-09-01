package ahdruntime

import (
	"reflect"
	"testing"
)

func listOf[T any](items ...T) *AhdList[T] { return AhdNewList(items...) }

func listItems[T any](list *AhdList[T]) []T { return list.Snapshot() }

func nestedItems[T any](list *AhdList[*AhdList[T]]) [][]T {
	result := make([][]T, 0, list.Len())
	for _, row := range list.Snapshot() {
		result = append(result, row.Snapshot())
	}
	return result
}

func pairEntries[K comparable, V any](pair *AhdPair[K, V]) ([]K, []V) {
	keys := pair.Keys()
	values := make([]V, len(keys))
	for index, key := range keys {
		values[index] = pair.Get(key)
	}
	return keys, values
}

func TestListsChunkSplitsWithoutPaddingOrMutation(t *testing.T) {
	source := listOf[int64](1, 2, 3, 4, 5)
	chunks := AhdListsChunk(AhdClassListsError, source, 2)
	if got := nestedItems(chunks); !reflect.DeepEqual(got, [][]int64{{1, 2}, {3, 4}, {5}}) {
		t.Fatalf("chunk = %v", got)
	}
	if got := listItems(source); !reflect.DeepEqual(got, []int64{1, 2, 3, 4, 5}) {
		t.Fatalf("chunk mutated its source: %v", got)
	}
	// Each inner List is fresh: mutating one never reaches the source.
	chunks.At(0).Add(99)
	if got := listItems(source); !reflect.DeepEqual(got, []int64{1, 2, 3, 4, 5}) {
		t.Fatalf("a chunk aliased its source: %v", got)
	}
	if got := nestedItems(AhdListsChunk(AhdClassListsError, source, 1)); len(got) != 5 {
		t.Fatalf("size 1 produced %v", got)
	}
	if got := nestedItems(AhdListsChunk(AhdClassListsError, source, 99)); !reflect.DeepEqual(got, [][]int64{{1, 2, 3, 4, 5}}) {
		t.Fatalf("an oversized size produced %v", got)
	}
	if got := nestedItems(AhdListsChunk(AhdClassListsError, AhdNewList[int64](), 3)); len(got) != 0 {
		t.Fatalf("an empty source produced %v", got)
	}
	expectRaise(t, AhdClassListsError, func() { AhdListsChunk(AhdClassListsError, source, 0) })
	expectRaise(t, AhdClassListsError, func() { AhdListsChunk(AhdClassListsError, source, -3) })
}

func TestListsFlattenRemovesExactlyOneLevel(t *testing.T) {
	rows := listOf(listOf[int64](1, 2), listOf[int64](3), AhdNewList[int64](), listOf[int64](4, 5))
	flat := AhdListsFlatten(rows)
	if got := listItems(flat); !reflect.DeepEqual(got, []int64{1, 2, 3, 4, 5}) {
		t.Fatalf("flatten = %v", got)
	}
	flat.Add(6)
	if got := listItems(rows.At(0)); !reflect.DeepEqual(got, []int64{1, 2}) {
		t.Fatalf("flatten aliased an inner List: %v", got)
	}
	if AhdListsFlatten(AhdNewList[*AhdList[int64]]()).Len() != 0 {
		t.Fatal("an empty outer List did not flatten to an empty List")
	}
	expectRaise(t, AhdClassNullError, func() {
		AhdListsFlatten(listOf[*AhdList[int64]](nil))
	})
}

func TestListsTransposeRequiresRectangularRows(t *testing.T) {
	rows := listOf(listOf[int64](1, 2, 3), listOf[int64](4, 5, 6))
	turned := AhdListsTranspose(AhdClassListsError, rows)
	if got := nestedItems(turned); !reflect.DeepEqual(got, [][]int64{{1, 4}, {2, 5}, {3, 6}}) {
		t.Fatalf("transpose = %v", got)
	}
	if got := nestedItems(AhdListsTranspose(AhdClassListsError, turned)); !reflect.DeepEqual(got, [][]int64{{1, 2, 3}, {4, 5, 6}}) {
		t.Fatalf("transposing twice did not restore the source: %v", got)
	}
	if got := nestedItems(AhdListsTranspose(AhdClassListsError, listOf(listOf[int64](1, 2, 3)))); !reflect.DeepEqual(got, [][]int64{{1}, {2}, {3}}) {
		t.Fatalf("1xN transpose = %v", got)
	}
	if got := nestedItems(AhdListsTranspose(AhdClassListsError, AhdNewList[*AhdList[int64]]())); len(got) != 0 {
		t.Fatalf("an empty outer List produced %v", got)
	}
	if got := nestedItems(AhdListsTranspose(AhdClassListsError, listOf(AhdNewList[int64](), AhdNewList[int64]()))); len(got) != 0 {
		t.Fatalf("all-empty rows produced %v", got)
	}
	// Ragged input is rejected in both directions rather than padded or cut.
	expectRaise(t, AhdClassListsError, func() {
		AhdListsTranspose(AhdClassListsError, listOf(listOf[int64](1, 2, 3), listOf[int64](4, 5)))
	})
	expectRaise(t, AhdClassListsError, func() {
		AhdListsTranspose(AhdClassListsError, listOf(listOf[int64](1, 2), listOf[int64](3, 4, 5)))
	})
	if got := listItems(rows.At(0)); !reflect.DeepEqual(got, []int64{1, 2, 3}) {
		t.Fatalf("transpose mutated its source: %v", got)
	}
}

func TestListsUniqueKeepsFirstOccurrence(t *testing.T) {
	source := listOf[int64](3, 1, 3, 2, 1)
	if got := listItems(AhdListsUnique(source, AhdEqInt)); !reflect.DeepEqual(got, []int64{3, 1, 2}) {
		t.Fatalf("unique = %v", got)
	}
	if got := listItems(source); !reflect.DeepEqual(got, []int64{3, 1, 3, 2, 1}) {
		t.Fatalf("unique mutated its source: %v", got)
	}
	// A nullable element type treats null as one ordinary distinct value.
	one, two := int64(1), int64(2)
	nullable := listOf[*int64](&one, nil, &one, nil, &two)
	distinct := AhdListsUnique(nullable, AhdEqNull[int64](AhdEqInt))
	if distinct.Len() != 3 || distinct.At(1) != nil {
		t.Fatalf("nullable unique = %v", listItems(distinct))
	}
	// Deep List equality, not pointer identity, decides distinctness.
	rows := listOf(listOf[int64](1, 2), listOf[int64](1, 2), listOf[int64](3))
	if got := AhdListsUnique(rows, AhdEqList[int64](AhdEqInt)).Len(); got != 2 {
		t.Fatalf("unique over Lists kept %d rows", got)
	}
}

func TestListsValueCountsCountsInFirstOccurrenceOrder(t *testing.T) {
	keys, values := pairEntries(AhdListsValueCounts(listOf[int64](1, 1, 3, 2, 1, 3)))
	if !reflect.DeepEqual(keys, []int64{1, 3, 2}) || !reflect.DeepEqual(values, []int64{3, 2, 1}) {
		t.Fatalf("valueCounts = %v -> %v", keys, values)
	}
	textKeys, textValues := pairEntries(AhdListsValueCounts(listOf("Math", "Physics", "Math")))
	if !reflect.DeepEqual(textKeys, []string{"Math", "Physics"}) || !reflect.DeepEqual(textValues, []int64{2, 1}) {
		t.Fatalf("String valueCounts = %v -> %v", textKeys, textValues)
	}
	if AhdListsValueCounts(AhdNewList[string]()).Len() != 0 {
		t.Fatal("an empty source did not produce an empty Pair")
	}
}

func TestListsGroupByKeepsKeyAndMemberOrder(t *testing.T) {
	source := listOf("Ali", "Ayse", "Bora", "Ahmet")
	groups := AhdListsGroupBy(source, func(name string) string { return name[:1] })
	keys := groups.Keys()
	if !reflect.DeepEqual(keys, []string{"A", "B"}) {
		t.Fatalf("group keys = %v", keys)
	}
	if got := listItems(groups.Get("A")); !reflect.DeepEqual(got, []string{"Ali", "Ayse", "Ahmet"}) {
		t.Fatalf("group members = %v", got)
	}
	// The callback reads a snapshot, so appending inside it cannot extend the
	// iteration, exactly like List.map and List.filter.
	numbers := listOf[int64](1, 2, 3)
	seen := 0
	AhdListsGroupBy(numbers, func(value int64) int64 {
		seen++
		numbers.Add(value * 10)
		return value % 2
	})
	if seen != 3 {
		t.Fatalf("the key Function ran %d times over a 3-element snapshot", seen)
	}
}

func TestKeyValueKeysAndValuesAreSnapshots(t *testing.T) {
	pair := AhdBuildPair([]string{"a", "b"}, []int64{1, 2})
	keys := AhdKeyValueKeys(pair)
	values := AhdKeyValueValues(pair)
	if !reflect.DeepEqual(listItems(keys), []string{"a", "b"}) || !reflect.DeepEqual(listItems(values), []int64{1, 2}) {
		t.Fatalf("keys/values = %v / %v", listItems(keys), listItems(values))
	}
	keys.Add("c")
	values.Add(3)
	if pair.Len() != 2 {
		t.Fatal("mutating a keys/values snapshot reached the source Pair")
	}
	if AhdKeyValueKeys(AhdNewPair[string, int64]()).Len() != 0 {
		t.Fatal("an empty Pair did not produce an empty key List")
	}
}

func TestKeyValueCombineRejectsMismatchAndDuplicates(t *testing.T) {
	keys, values := pairEntries(AhdKeyValueCombine(AhdClassKeyValueError,
		listOf("name", "score"), listOf("Ali", "91")))
	if !reflect.DeepEqual(keys, []string{"name", "score"}) || !reflect.DeepEqual(values, []string{"Ali", "91"}) {
		t.Fatalf("combine = %v -> %v", keys, values)
	}
	if AhdKeyValueCombine(AhdClassKeyValueError, AhdNewList[string](), AhdNewList[int64]()).Len() != 0 {
		t.Fatal("empty inputs did not produce an empty Pair")
	}
	expectRaise(t, AhdClassKeyValueError, func() {
		AhdKeyValueCombine(AhdClassKeyValueError, listOf("a", "b"), listOf[int64](1))
	})
	expectRaise(t, AhdClassKeyValueError, func() {
		AhdKeyValueCombine(AhdClassKeyValueError, listOf("a", "a"), listOf[int64](1, 2))
	})
}

func TestKeyValueWithReplacesInPlaceAndAppendsNewKeys(t *testing.T) {
	base := AhdBuildPair([]string{"name", "score"}, []string{"Ali", "90"})
	replaced := AhdKeyValueWith(base, "score", "95")
	if keys, values := pairEntries(replaced); !reflect.DeepEqual(keys, []string{"name", "score"}) ||
		!reflect.DeepEqual(values, []string{"Ali", "95"}) {
		t.Fatalf("with moved an existing key: %v -> %v", keys, values)
	}
	appended := AhdKeyValueWith(base, "department", "Mathematics")
	if keys := appended.Keys(); !reflect.DeepEqual(keys, []string{"name", "score", "department"}) {
		t.Fatalf("with did not append a new key last: %v", keys)
	}
	if base.Get("score") != "90" || base.Len() != 2 {
		t.Fatal("with mutated its source Pair")
	}
}

func TestKeyValueWithoutAndDropPreserveSurvivingOrder(t *testing.T) {
	base := AhdBuildPair([]string{"a", "b", "c"}, []int64{1, 2, 3})
	for _, test := range []struct {
		key  string
		want []string
	}{{"a", []string{"b", "c"}}, {"b", []string{"a", "c"}}, {"c", []string{"a", "b"}}} {
		if keys := AhdKeyValueWithout(base, test.key).Keys(); !reflect.DeepEqual(keys, test.want) {
			t.Fatalf("without(%q) = %v", test.key, keys)
		}
	}
	if keys := AhdKeyValueDrop(AhdClassKeyValueError, base, listOf("b")).Keys(); !reflect.DeepEqual(keys, []string{"a", "c"}) {
		t.Fatalf("drop kept %v", keys)
	}
	if keys := AhdKeyValueDrop(AhdClassKeyValueError, base, AhdNewList[string]()).Keys(); !reflect.DeepEqual(keys, []string{"a", "b", "c"}) {
		t.Fatalf("an empty drop changed the Pair: %v", keys)
	}
	if AhdKeyValueDrop(AhdClassKeyValueError, base, listOf("a", "b", "c")).Len() != 0 {
		t.Fatal("dropping every key did not empty the Pair")
	}
	if base.Len() != 3 {
		t.Fatal("without/drop mutated their source Pair")
	}
	expectRaise(t, AhdClassKeyError, func() { AhdKeyValueWithout(base, "z") })
	expectRaise(t, AhdClassKeyError, func() { AhdKeyValueDrop(AhdClassKeyValueError, base, listOf("z")) })
	expectRaise(t, AhdClassKeyValueError, func() { AhdKeyValueDrop(AhdClassKeyValueError, base, listOf("a", "a")) })
}

func TestKeyValueSelectFollowsRequestedOrder(t *testing.T) {
	base := AhdBuildPair([]string{"a", "b", "c"}, []int64{1, 2, 3})
	keys, values := pairEntries(AhdKeyValueSelect(AhdClassKeyValueError, base, listOf("c", "a")))
	if !reflect.DeepEqual(keys, []string{"c", "a"}) || !reflect.DeepEqual(values, []int64{3, 1}) {
		t.Fatalf("select = %v -> %v", keys, values)
	}
	if AhdKeyValueSelect(AhdClassKeyValueError, base, AhdNewList[string]()).Len() != 0 {
		t.Fatal("an empty select did not produce an empty Pair")
	}
	expectRaise(t, AhdClassKeyError, func() { AhdKeyValueSelect(AhdClassKeyValueError, base, listOf("z")) })
	expectRaise(t, AhdClassKeyValueError, func() { AhdKeyValueSelect(AhdClassKeyValueError, base, listOf("a", "a")) })
}

func TestKeyValueRenamePreservesPosition(t *testing.T) {
	base := AhdBuildPair([]string{"a", "b", "c"}, []int64{1, 2, 3})
	keys, values := pairEntries(AhdKeyValueRename(AhdClassKeyValueError, base, "b", "middle"))
	if !reflect.DeepEqual(keys, []string{"a", "middle", "c"}) || !reflect.DeepEqual(values, []int64{1, 2, 3}) {
		t.Fatalf("rename = %v -> %v", keys, values)
	}
	// Renaming a key to itself is a harmless structural copy, not an error.
	same := AhdKeyValueRename(AhdClassKeyValueError, base, "b", "b")
	if !reflect.DeepEqual(same.Keys(), []string{"a", "b", "c"}) {
		t.Fatalf("a no-op rename changed the order: %v", same.Keys())
	}
	same.Set("d", 4)
	if base.Len() != 3 {
		t.Fatal("a no-op rename aliased its source")
	}
	expectRaise(t, AhdClassKeyError, func() { AhdKeyValueRename(AhdClassKeyValueError, base, "z", "y") })
	expectRaise(t, AhdClassKeyValueError, func() { AhdKeyValueRename(AhdClassKeyValueError, base, "a", "b") })
}

func TestKeyValueMapValuesKeepsKeyOrderAndRunsOncePerValue(t *testing.T) {
	base := AhdBuildPair([]string{"a", "b"}, []string{"10", "20"})
	visited := []string{}
	mapped := AhdKeyValueMapValues(base, func(value string) int64 {
		visited = append(visited, value)
		return int64(len(value))
	})
	if keys, values := pairEntries(mapped); !reflect.DeepEqual(keys, []string{"a", "b"}) ||
		!reflect.DeepEqual(values, []int64{2, 2}) {
		t.Fatalf("mapValues = %v -> %v", keys, values)
	}
	if !reflect.DeepEqual(visited, []string{"10", "20"}) {
		t.Fatalf("the transform ran as %v, not once per value in Pair order", visited)
	}
	if base.Get("a") != "10" {
		t.Fatal("mapValues mutated its source Pair")
	}
}

func TestKeyValueMergeRejectsCollisionsAndOverlayAcceptsThem(t *testing.T) {
	left := AhdBuildPair([]string{"a", "b"}, []int64{1, 2})
	right := AhdBuildPair([]string{"c"}, []int64{3})
	keys, values := pairEntries(AhdKeyValueMerge(AhdClassKeyValueError, left, right))
	if !reflect.DeepEqual(keys, []string{"a", "b", "c"}) || !reflect.DeepEqual(values, []int64{1, 2, 3}) {
		t.Fatalf("merge = %v -> %v", keys, values)
	}
	empty := AhdNewPair[string, int64]()
	if AhdKeyValueMerge(AhdClassKeyValueError, empty, empty).Len() != 0 {
		t.Fatal("merging two empty Pairs did not produce an empty Pair")
	}
	if AhdKeyValueMerge(AhdClassKeyValueError, empty, left).Len() != 2 {
		t.Fatal("merging into an empty Pair lost entries")
	}
	expectRaise(t, AhdClassKeyValueError, func() {
		AhdKeyValueMerge(AhdClassKeyValueError, left, AhdBuildPair([]string{"a"}, []int64{9}))
	})

	changes := AhdBuildPair([]string{"b", "c"}, []int64{9, 3})
	overlaidKeys, overlaidValues := pairEntries(AhdKeyValueOverlay(left, changes))
	if !reflect.DeepEqual(overlaidKeys, []string{"a", "b", "c"}) ||
		!reflect.DeepEqual(overlaidValues, []int64{1, 9, 3}) {
		t.Fatalf("overlay = %v -> %v", overlaidKeys, overlaidValues)
	}
	if left.Get("b") != 2 || left.Len() != 2 || changes.Len() != 2 {
		t.Fatal("overlay mutated one of its sources")
	}
	if AhdKeyValueOverlay(left, empty).Len() != 2 || AhdKeyValueOverlay(empty, changes).Len() != 2 {
		t.Fatal("overlay with an empty operand lost entries")
	}
}
