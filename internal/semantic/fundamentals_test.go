package semantic

import "testing"

func TestLenAcceptsOnlySizedValues(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"String", `write(len("abc"))`, true},
		{"List", "values: List<Int> := [1]\nwrite(len(values))", true},
		{"Pair", "values: Pair<String, Int> := {\"a\": 1}\nwrite(len(values))", true},
		{"Int", "write(len(5))", false},
		{"Bool", "write(len(true))", false},
		{"nullable List", "values: List<Int> := null\nwrite(len(values))", false},
		{"no argument", "write(len())", false},
		{"two arguments", "write(len(\"a\", \"b\"))", false},
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

func TestClearAcceptsOnlyMutableCollections(t *testing.T) {
	tests := []struct {
		name string
		text string
		code string
	}{
		{"List", "values: List<Int> := [1]\nclear(values)", ""},
		{"Pair", "values: Pair<String, Int> := {\"a\": 1}\nclear(values)", ""},
		{"String", "text: String := \"a\"\nclear(text)", codeCallArguments},
		{"Int", "value: Int := 1\nclear(value)", codeCallArguments},
		{"nullable List", "values: List<Int> := null\nclear(values)", codeNullableUse},
		{"no argument", "clear()", codeCallArguments},
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

func TestClearRejectsAConstantTarget(t *testing.T) {
	_, result := analyzeText(t, `Holder: Class<> := {
    structure: Attributes := (
        seed: Int
    ) {
        attribute.values: Constant List<Int> := [1, 2, 3]
    }

    empty: Function := (
    ) -> Nothing {
        clear(attribute.values)
    }
}

write("ready")
`)
	requireSemanticCode(t, result, codeConstantAssignment)
}

func TestClearReturnsNothing(t *testing.T) {
	_, result := analyzeText(t, "values: List<Int> := [1]\nresult: Int := clear(values)")
	if !result.HasErrors() {
		t.Fatal("clear returns Nothing and must not bind to a typed value")
	}
}

func TestWriteAndTakeArity(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"write one value", `write("a")`, true},
		{"write no value", `write()`, false},
		{"write two values", `write("a", "b")`, false},
		{"take with prompt", `name: String := take("Name: ")`, true},
		{"take without prompt", `name: String := take()`, true},
		{"take with a non-String prompt", `name: String := take(5)`, false},
		{"take with two arguments", `name: String := take("a", "b")`, false},
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
