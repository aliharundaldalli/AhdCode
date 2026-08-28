package semantic

import "testing"

// TestEmptyCollectionLiteralsUseTheirContext covers the v0.1 rule that an
// empty collection literal takes its type from the surrounding declaration.
func TestEmptyCollectionLiteralsUseTheirContext(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"typed empty List", "numbers: List<Int> := []"},
		{"typed empty Pair", "scores: Pair<String, Int> := {}"},
		{"typed empty List of Class", "Student: Class<> := {\n    structure: Attributes := (\n        name: String\n    )\n}\nstudents: List<Student> := []"},
		{"nested empty List", "rows: List<List<Int>> := [\n    []\n    []\n]"},
		{"nested empty Pair", "maps: List<Pair<String, Int>> := [\n    {}\n    {}\n]"},
		{"deeply nested empty List", "deep: List<List<List<Int>>> := [\n    [\n        []\n    ]\n    []\n]"},
		{"empty List as a Pair value", "rows: Pair<String, List<Int>> := {\n    \"a\": []\n}"},
		{"empty Pair as a Pair value", "rows: Pair<String, Pair<String, Int>> := {\n    \"a\": {}\n}"},
		{"List of only null elements", "names: List<String> := [null, null]"},
		{"Pair with a null value", "scores: Pair<String, Int> := {\n    \"a\": null\n}"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			requireSemanticClean(t, result)
		})
	}
}

func TestEmptyCollectionLiteralsUseCallAndReturnContext(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "empty List argument",
			text: "useList: Function := (\n    values: List<Int>\n) -> Nothing {\n}\n\nuseList([])",
		},
		{
			name: "empty Pair argument",
			text: "usePair: Function := (\n    scores: Pair<String, Int>\n) -> Nothing {\n}\n\nusePair({})",
		},
		{
			name: "empty List argument by name",
			text: "useList: Function := (\n    values: List<Int>\n) -> Nothing {\n}\n\nuseList(values: [])",
		},
		{
			name: "empty List default argument",
			text: "useList: Function := (\n    values: List<Int> := []\n) -> Nothing {\n}\n\nuseList()",
		},
		{
			name: "empty List return",
			text: "makeList: Function := (\n) -> List<Int> {\n    return []\n}",
		},
		{
			name: "empty Pair return",
			text: "makePair: Function := (\n) -> Pair<String, Int> {\n    return {}\n}",
		},
		{
			name: "empty List construction argument",
			text: "Holder: Class<> := {\n    structure: Attributes := (\n        values: List<Int>\n    )\n}\n\nholder: Holder := Holder(values: [])",
		},
		{
			name: "empty List assignment",
			text: "values: List<Int> := [1]\nvalues = []",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			requireSemanticClean(t, result)
		})
	}
}

// TestEmptyCollectionLiteralsWithoutContextAreRejected keeps an uninferable
// literal an error; contextual typing never becomes dynamic inference.
func TestEmptyCollectionLiteralsWithoutContextAreRejected(t *testing.T) {
	tests := []struct {
		name string
		text string
		code string
	}{
		{"bare empty List", "write([])", codeCollectionInference},
		{"bare empty Pair", "write({})", codeCollectionInference},
		{"bare List of only null", "write([null])", codeCollectionInference},
		{"untyped List declaration", "values: List := []", codeInvalidType},
		{"untyped Pair declaration", "scores: Pair := {}", codeInvalidType},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			requireSemanticCode(t, result, test.code)
		})
	}
}

// TestUntypedCollectionDeclarationReportsOneRootCause keeps the missing type
// argument the only diagnostic, without an added inference or mismatch error.
func TestUntypedCollectionDeclarationReportsOneRootCause(t *testing.T) {
	for name, text := range map[string]string{
		"List": "values: List := []",
		"Pair": "scores: Pair := {}",
	} {
		t.Run(name, func(t *testing.T) {
			_, result := analyzeText(t, text)
			if len(result.Diagnostics) != 1 {
				t.Fatalf("expected exactly one diagnostic; received %+v", result.Diagnostics)
			}
			if result.Diagnostics[0].Code != codeInvalidType {
				t.Fatalf("expected %s; received %s", codeInvalidType, result.Diagnostics[0].Code)
			}
		})
	}
}

// TestContextDoesNotOverrideCollectionInference keeps generic invariance and
// ordinary element inference exactly as before.
func TestContextDoesNotOverrideCollectionInference(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"List<Int> is not List<Real>", "integers: List<Int> := [1, 2]\nreals: List<Real> := integers", false},
		{"a Real element does not fit List<Int>", "values: List<Int> := [1, 2.5]", false},
		{"an Int literal list does not fit List<Real>", "values: List<Real> := [1, 2]", false},
		{"a String element does not fit List<Int>", "values: List<Int> := [\"a\"]", false},
		{"mixed element types are rejected", "values: List<Int> := [1, \"a\"]", false},
		{"a Pair value type still has to match", "scores: Pair<String, Int> := {\n    \"a\": \"b\"\n}", false},
		{"non-empty List inference is unchanged", "values: List<Int> := [1, 2, 3]", true},
		{"non-empty Pair inference is unchanged", "scores: Pair<String, Int> := {\n    \"a\": 1\n}", true},
		{"numeric widening inside a literal is unchanged", "values: List<Real> := [1.0, 2]", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			if !result.HasErrors() {
				t.Fatal("expected a diagnostic")
			}
		})
	}
}

// TestPairKeyRulesSurviveContextualTyping keeps the key restrictions from
// being satisfied by the declared type instead of the literal.
func TestPairKeyRulesSurviveContextualTyping(t *testing.T) {
	_, realKey := analyzeText(t, "values: Pair<Real, String> := {}")
	requireSemanticCode(t, realKey, codeInvalidPairKey)

	_, realEntries := analyzeText(t, "values: Pair<Real, String> := {\n    1.5: \"A\"\n}")
	requireSemanticCode(t, realEntries, codeInvalidPairKey)

	_, duplicate := analyzeText(t, "scores: Pair<String, Int> := {\n    \"a\": 1\n    \"a\": 2\n}")
	requireSemanticCode(t, duplicate, codeDuplicatePairKey)

	_, nullKey := analyzeText(t, "scores: Pair<String, Int> := {\n    null: 1\n}")
	requireSemanticCode(t, nullKey, codeInvalidPairKey)
}

// TestNullPairKeyReportsOneRootCause keeps a rejected key from also cascading
// an assignability mismatch against the declared type.
func TestNullPairKeyReportsOneRootCause(t *testing.T) {
	_, result := analyzeText(t, "scores: Pair<String, Int> := {\n    null: 1\n}")
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected exactly one diagnostic; received %+v", result.Diagnostics)
	}
}
