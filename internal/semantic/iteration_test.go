package semantic

import "testing"

// TestIterationBindingTypes covers the optional explicit type on a for binding
// against every iterable kind.
func TestIterationBindingTypes(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"inferred List element", "values: List<Int> := [1]\nfor value in values {\n    write(value)\n}", true},
		{"explicit List element", "values: List<Int> := [1]\nfor value: Int in values {\n    write(value)\n}", true},
		{"explicit List element of Class", "S: Class<> := {\n    structure: Attributes := (\n        n: Int\n    )\n}\nvalues: List<S> := []\nfor value: S in values {\n    write(value.n)\n}", true},
		{"wrong explicit List element", "values: List<String> := [\"A\"]\nfor value: Int in values {\n}", false},
		{"inferred String character", "for character in \"ab\" {\n    write(character)\n}", true},
		{"explicit String character", "for character: String in \"ab\" {\n    write(character)\n}", true},
		{"wrong explicit String character", "for character: Int in \"ab\" {\n}", false},
		{"inferred Pair key", "scores: Pair<String, Int> := {}\nfor key in scores {\n    write(key)\n}", true},
		{"explicit Pair key", "scores: Pair<String, Int> := {}\nfor key: String in scores {\n    write(key)\n}", true},
		{"wrong explicit Pair key", "scores: Pair<String, Int> := {}\nfor key: Int in scores {\n}", false},
		{"explicit Pair value type is not the key type", "scores: Pair<Int, String> := {}\nfor key: String in scores {\n}", false},
		{"inferred between value", "for value in between(3) {\n    write(value)\n}", true},
		{"explicit between value", "for value: Int in between(3) {\n    write(value)\n}", true},
		{"wrong explicit between value", "for value: String in between(10) {\n}", false},
		{"a numeric widening is not an iteration conversion", "values: List<Int> := [1]\nfor value: Real in values {\n}", false},
		{"an unknown binding type is rejected", "values: List<Int> := [1]\nfor value: Nonsense in values {\n}", false},
		{"a non-iterable value is rejected", "value: Int := 1\nfor item in value {\n}", false},
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

func TestWrongIterationBindingTypeIsReportedOnce(t *testing.T) {
	_, result := analyzeText(t, "values: List<String> := [\"A\"]\nfor value: Int in values {\n}")
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected exactly one diagnostic; received %+v", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != codeTypeMismatch {
		t.Fatalf("expected %s; received %s", codeTypeMismatch, result.Diagnostics[0].Code)
	}
}

// TestBetweenArguments covers the arity and Int argument rules of the lazy
// integer iteration builtin.
func TestBetweenArguments(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"stop", "for value in between(5) {\n}", true},
		{"start and stop", "for value in between(1, 5) {\n}", true},
		{"start, stop, and step", "for value in between(0, 10, 2) {\n}", true},
		{"a negative step", "for value in between(5, 0, -1) {\n}", true},
		{"Int bindings", "start: Int := 1\nstop: Int := 5\nfor value in between(start, stop) {\n}", true},
		{"no arguments", "for value in between() {\n}", false},
		{"four arguments", "for value in between(1, 2, 3, 4) {\n}", false},
		{"a Real start", "for value in between(1.5, 10) {\n}", false},
		{"a String stop", "for value in between(1, \"10\") {\n}", false},
		{"a Real step", "for value in between(1, 10, 1.5) {\n}", false},
		{"a null argument", "for value in between(null) {\n}", false},
		{"a named argument", "for value in between(stop: 5) {\n}", false},
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

// TestBetweenHasNoValueSurface keeps the lazy iteration out of every position
// that would require a public type or a rendered value.
func TestBetweenHasNoValueSurface(t *testing.T) {
	for name, text := range map[string]string{
		"written":       "write(between(3))",
		"stringified":   "write(str(between(3)))",
		"bound to Int":  "value: Int := between(3)",
		"bound to List": "values: List<Int> := between(3)",
		"indexed":       "write(between(3)[0])",
		"measured":      "write(len(between(3)))",
		"cleared":       "clear(between(3))",
	} {
		t.Run(name, func(t *testing.T) {
			_, result := analyzeText(t, text)
			if !result.HasErrors() {
				t.Fatal("a lazy range has no value surface in v0.1")
			}
		})
	}
}
