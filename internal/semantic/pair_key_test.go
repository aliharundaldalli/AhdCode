package semantic

import "testing"

// TestPairKeyTypesAreRestricted enforces the v0.1 rule that a Pair key is one
// of the stable simple scalar types.
func TestPairKeyTypesAreRestricted(t *testing.T) {
	tests := []struct {
		name string
		text string
		code string
	}{
		{"String key", "scores: Pair<String, Int> := {\n    \"Ali\": 1\n}", ""},
		{"Int key", "scores: Pair<Int, String> := {\n    1: \"A\"\n}", ""},
		{"Bool key", "scores: Pair<Bool, String> := {\n    true: \"A\"\n}", ""},
		{"declared Real key", "scores: Pair<Real, String> := {\n    1.5: \"A\"\n}", codeInvalidPairKey},
		{"inferred Real key", `write({1.5: "A"})`, codeInvalidPairKey},
		{"Real key in a Function signature", "read: Function := (\n    scores: Pair<Real, Int>\n) -> Int {\n    return 1\n}", codeInvalidPairKey},
		{"Real key nested in a List", "rows: List<Pair<Real, Int>> := [\n    {1.5: 1}\n]", codeInvalidPairKey},
		{"List key", "scores: Pair<List<Int>, Int> := {\n    [1]: 1\n}", codeInvalidPairKey},
		{"Pair key", "scores: Pair<Pair<String, Int>, Int> := {\n    {\"a\": 1}: 1\n}", codeInvalidPairKey},
		{"Nothing key", "scores: Pair<Nothing, Int> := {\n    1: 1\n}", codeInvalidPairKey},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			if test.code == "" {
				requireSemanticClean(t, result)
				return
			}
			requireSemanticCode(t, result, test.code)
		})
	}
}

func TestClassInstancePairKeyIsRejected(t *testing.T) {
	_, result := analyzeText(t, `Student: Class<> := {
    structure: Attributes := (
        name: String
    )
}

byStudent: Pair<Student, Int> := {
    Student(name: "Ali"): 1
}`)
	requireSemanticCode(t, result, codeInvalidPairKey)
}

func TestNullPairKeyIsRejected(t *testing.T) {
	_, result := analyzeText(t, "scores: Pair<String, Int> := {\n    null: 1\n}")
	if !result.HasErrors() {
		t.Fatal("a null Pair key must be rejected")
	}
}

// TestDuplicatePairLiteralKeysAreRejected enforces the v0.1 rule that one Pair
// literal may not repeat a key.
func TestDuplicatePairLiteralKeysAreRejected(t *testing.T) {
	tests := []struct {
		name string
		text string
		code string
	}{
		{"duplicate String key", "scores: Pair<String, Int> := {\n    \"Ali\": 1\n    \"Ali\": 2\n}", codeDuplicatePairKey},
		{"duplicate Int key", "scores: Pair<Int, String> := {\n    1: \"A\"\n    1: \"B\"\n}", codeDuplicatePairKey},
		{"duplicate Bool key", "scores: Pair<Bool, String> := {\n    true: \"A\"\n    true: \"B\"\n}", codeDuplicatePairKey},
		{"duplicate negative Int key", "scores: Pair<Int, String> := {\n    -1: \"A\"\n    -1: \"B\"\n}", codeDuplicatePairKey},
		{"distinct String keys", "scores: Pair<String, Int> := {\n    \"Ali\": 1\n    \"Ayse\": 2\n}", ""},
		{"distinct Int keys", "scores: Pair<Int, String> := {\n    1: \"A\"\n    2: \"B\"\n}", ""},
		{"distinct Bool keys", "scores: Pair<Bool, String> := {\n    true: \"A\"\n    false: \"B\"\n}", ""},
		{"single key", "scores: Pair<String, Int> := {\n    \"Ali\": 1\n}", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			if test.code == "" {
				requireSemanticClean(t, result)
				return
			}
			requireSemanticCode(t, result, test.code)
		})
	}
}

// TestDuplicatePairKeysComparePairKeyValues checks that duplicate detection
// follows ordinary Pair key equality rather than raw source spelling.
// TestEmptyPairLiteralHasNoDuplicateKeys keeps duplicate detection out of the
// way of an empty literal.
func TestEmptyPairLiteralHasNoDuplicateKeys(t *testing.T) {
	_, result := analyzeText(t, "write({})")
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == codeDuplicatePairKey || diagnostic.Code == codeInvalidPairKey {
			t.Fatalf("an empty Pair literal must not report a key diagnostic: %s", diagnostic.Message)
		}
	}
}

func TestDuplicatePairKeysComparePairKeyValues(t *testing.T) {
	_, spelling := analyzeText(t, "scores: Pair<Int, String> := {\n    1: \"A\"\n    01: \"B\"\n}")
	requireSemanticCode(t, spelling, codeDuplicatePairKey)

	_, constant := analyzeText(t, "One: Constant Int := 1\nscores: Pair<Int, String> := {\n    1: \"A\"\n    One: \"B\"\n}")
	requireSemanticCode(t, constant, codeDuplicatePairKey)

	// A key that is not known at compile time cannot be compared, so it is not
	// reported as a duplicate.
	_, runtimeKey := analyzeText(t, "first: Int := 1\nscores: Pair<Int, String> := {\n    1: \"A\"\n    first: \"B\"\n}")
	requireSemanticClean(t, runtimeKey)
}

// TestInvalidPairKeyIsReportedOnce keeps one diagnostic per root cause: a
// rejected declared key type must not also cascade an assignability error.
func TestInvalidPairKeyIsReportedOnce(t *testing.T) {
	_, result := analyzeText(t, "scores: Pair<Real, String> := {\n    1.5: \"A\"\n    2.5: \"B\"\n}")
	count := 0
	for _, diagnostic := range result.Diagnostics {
		count++
		if diagnostic.Code != codeInvalidPairKey {
			t.Fatalf("unexpected derivative diagnostic %s: %s", diagnostic.Code, diagnostic.Message)
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one diagnostic; received %d", count)
	}
}
