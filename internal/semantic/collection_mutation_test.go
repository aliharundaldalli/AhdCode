package semantic

import "testing"

// TestCollectionMutationsAreStaticallyChecked covers the v0.1 built-in List and
// Pair mutation surface: add, eject, and index assignment.
func TestCollectionMutationsAreStaticallyChecked(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"List add", "values: List<Int> := [1]\nvalues.add(2)", true},
		{"List add widens Int to Real", "values: List<Real> := [1.0]\nvalues.add(1)", true},
		{"List add a null element", "values: List<String> := [\"a\"]\nvalues.add(null)", true},
		{"List eject", "values: List<Int> := [1]\nvalues.eject(0)", true},
		{"List eject a negative index", "values: List<Int> := [1]\nvalues.eject(-1)", true},
		{"Pair eject", "scores: Pair<String, Int> := {\n    \"a\": 1\n}\nscores.eject(\"a\")", true},
		{"Pair insert", "scores: Pair<String, Int> := {}\nscores[\"a\"] = 1", true},
		{"empty List add and eject", "values: List<Int> := []\nvalues.add(1)\nvalues.eject(0)", true},
		{"empty Pair insert and eject", "scores: Pair<String, Int> := {}\nscores[\"a\"] = 1\nscores.eject(\"a\")", true},
		{"List add of the wrong type", "values: List<Int> := [1]\nvalues.add(\"wrong\")", false},
		{"List add does not narrow Real to Int", "values: List<Int> := [1]\nvalues.add(1.5)", false},
		{"List eject with a Real index", "values: List<Int> := [1]\nvalues.eject(1.5)", false},
		{"List eject with a String index", "values: List<Int> := [1]\nvalues.eject(\"a\")", false},
		{"Pair eject with the wrong key type", "scores: Pair<String, Int> := {}\nscores.eject(10)", false},
		{"Pair has no add", "scores: Pair<String, Int> := {}\nscores.add(\"a\", 1)", false},
		{"List add without an argument", "values: List<Int> := [1]\nvalues.add()", false},
		{"List add with two arguments", "values: List<Int> := [1]\nvalues.add(1, 2)", false},
		{"List eject without an argument", "values: List<Int> := [1]\nvalues.eject()", false},
		{"List add with a named argument", "values: List<Int> := [1]\nvalues.add(value: 1)", false},
		{"List has no unknown method", "values: List<Int> := [1]\nvalues.push(1)", false},
		{"List has no pop", "values: List<Int> := [1]\nvalues.pop()", false},
		{"List has no remove", "values: List<Int> := [1]\nvalues.remove(0)", false},
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

// TestCollectionMutationsReturnNothing keeps add and eject out of value
// position.
func TestCollectionMutationsReturnNothing(t *testing.T) {
	for name, text := range map[string]string{
		"bound List eject":   "values: List<Int> := [1]\nx: Int := values.eject(0)",
		"written List add":   "values: List<Int> := [1]\nwrite(values.add(5))",
		"written Pair eject": "scores: Pair<String, Int> := {\n    \"a\": 1\n}\nwrite(scores.eject(\"a\"))",
		"interpolated add":   "values: List<Int> := [1]\nwrite(\"{values.add(5)}\")",
	} {
		t.Run(name, func(t *testing.T) {
			_, result := analyzeText(t, text)
			if !result.HasErrors() {
				t.Fatal("a Nothing result must not be usable as a value")
			}
		})
	}
}

// TestCollectionMutationsRequireNonNullReceivers keeps the receiver null-state
// rule the same as every other collection operation.
func TestCollectionMutationsRequireNonNullReceivers(t *testing.T) {
	for name, text := range map[string]string{
		"null List add":    "values: List<Int> := null\nvalues.add(1)",
		"null List eject":  "values: List<Int> := null\nvalues.eject(0)",
		"null Pair eject":  "scores: Pair<String, Int> := null\nscores.eject(\"a\")",
		"null Pair insert": "scores: Pair<String, Int> := null\nscores[\"a\"] = 1",
	} {
		t.Run(name, func(t *testing.T) {
			_, result := analyzeText(t, text)
			requireSemanticCode(t, result, codeNullableUse)
		})
	}
}

// TestClassMethodsMayStillBeNamedAddOrEject keeps the built-in operations
// receiver-typed rather than reserved names.
func TestClassMethodsMayStillBeNamedAddOrEject(t *testing.T) {
	_, result := analyzeText(t, `Counter: Class<> := {
    structure: Attributes := (
        value: Int
    )

    add: Function := (
        amount: Int
    ) -> Nothing {
        attribute.value = attribute.value + amount
    }

    eject: Function := (
    ) -> Int {
        return attribute.value
    }
}

counter: Counter := Counter(value: 1)
counter.add(2)
write(counter.eject())`)
	requireSemanticClean(t, result)
}

// TestCollectionMutationsPreservePairKeyRules keeps the key restrictions in
// force on the mutation surface.
func TestCollectionMutationsPreservePairKeyRules(t *testing.T) {
	_, realKey := analyzeText(t, "scores: Pair<Real, Int> := {}\nscores.eject(1.5)")
	requireSemanticCode(t, realKey, codeInvalidPairKey)

	_, nullKey := analyzeText(t, "scores: Pair<String, Int> := {}\nscores.eject(null)")
	requireSemanticCode(t, nullKey, codeNullableUse)
}
