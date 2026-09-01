package golang

import (
	"strings"
	"testing"
)

// TestCollectionCallsGenerateFullyTypedGo is the code-generation half of the
// no-erasure proof. The frontend specializes every Lists/KeyValue call to one
// exact concrete signature; this checks that the specialization survives all
// the way into Go, where each runtime helper is instantiated at the concrete
// element, key, and value types rather than at an interface.
func TestCollectionCallsGenerateFullyTypedGo(t *testing.T) {
	program := generate(t, `bring Lists
bring KeyValue

numbers: List<Int> := [1, 2, 3]
words: List<String> := ["a"]
grid: List<List<Int>> := [[1, 2]]
record: Pair<String, Int> := {"a": 1}

write(Lists.chunk(numbers, 2))
write(Lists.chunk(words, 2))
write(Lists.flatten(grid))
write(Lists.transpose(grid))
write(Lists.unique(numbers))
write(Lists.valueCounts(words))
write(Lists.groupBy(numbers, lambda (value: Int) -> value % 2))

write(KeyValue.keys(record))
write(KeyValue.values(record))
write(KeyValue.combine(words, numbers))
write(KeyValue.with(record, "b", 2))
write(KeyValue.without(record, "a"))
write(KeyValue.select(record, ["a"]))
write(KeyValue.drop(record, ["a"]))
write(KeyValue.rename(record, "a", "z"))
write(KeyValue.mapValues(record, lambda (value: Int) -> str(value)))
write(KeyValue.merge(record, {"c": 3}))
write(KeyValue.overlay(record, {"b": 2}))
`)
	text := programSource(t, program)

	// The two element types reach two distinct instantiations of one helper,
	// which is exactly what a family of per-type overloads would have been
	// needed for, and what an erased element type would have destroyed.
	for _, want := range []string{
		"AhdStrList[*AhdList[int64]](AhdStrList[int64](AhdStrInt))(AhdListsChunk(",
		"AhdStrList[*AhdList[string]](AhdStrList[string](AhdStrQuoted))(AhdListsChunk(",
		"AhdListsUnique(", "AhdEqInt",
		"AhdStrPair[string, int64](AhdStrQuoted, AhdStrInt)(AhdListsValueCounts(",
		"AhdStrPair[int64, *AhdList[int64]](AhdStrInt, AhdStrList[int64](AhdStrInt))(AhdListsGroupBy(",
		"AhdStrList[string](AhdStrQuoted)(AhdKeyValueKeys(",
		"AhdStrPair[string, string](AhdStrQuoted, AhdStrQuoted)(AhdKeyValueMapValues(",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated program does not contain %q:\n%s", want, text)
		}
	}

	// The adapted callbacks are concretely typed in both directions: the key
	// Function unboxes to a non-null Int key, and the value transform takes
	// the Pair's own Int value type.
	for _, want := range []string{
		"func(ahdTemporary1 int64) int64 {",
		"func(ahdTemporary2 int64) string {",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated program does not contain the typed adapter %q:\n%s", want, text)
		}
	}

	// No collection call may fall back on an erased Go representation.
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "AhdLists") && !strings.Contains(line, "AhdKeyValue") {
			continue
		}
		for _, erased := range []string{"interface{}", "[any]", "(any)", " any)", "reflect."} {
			if strings.Contains(line, erased) {
				t.Fatalf("a collection call generated the erased form %q:\n%s", erased, strings.TrimSpace(line))
			}
		}
	}
}
