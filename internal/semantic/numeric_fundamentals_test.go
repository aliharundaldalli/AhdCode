package semantic

import "testing"

// TestNumericFundamentalsHaveExactOverloads pins the v0.1 public surface of
// abs, sum, min, and max: two numeric overloads each, with no implicit
// conversion, no truthiness, and no dynamic element typing.
func TestNumericFundamentalsHaveExactOverloads(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"abs of a positive Int", "value: Int := abs(5)", true},
		{"abs of a negative Int", "value: Int := abs(-5)", true},
		{"abs of a positive Real", "value: Real := abs(2.5)", true},
		{"abs of a negative Real", "value: Real := abs(-2.5)", true},
		{"abs of the minimum Int is statically valid", "value: Int := abs(-9223372036854775808)", true},
		{"abs does not accept String", `value: Int := abs("5")`, false},
		{"abs does not accept Bool", "value: Int := abs(true)", false},
		{"abs does not accept a List", "values: List<Int> := [1]\nvalue: Int := abs(values)", false},
		{"abs does not accept a Pair", `value: Int := abs({"a": 1})`, false},
		{"abs does not widen Int to Real", "value: Int := abs(2.5)", false},

		{"sum of List<Int>", "values: List<Int> := [1, 2, 3]\ntotal: Int := sum(values)", true},
		{"sum of List<Real>", "values: List<Real> := [1.5, 2.0]\ntotal: Real := sum(values)", true},
		{"sum of an empty List<Int>", "values: List<Int> := []\ntotal: Int := sum(values)", true},
		{"sum of an empty List<Real>", "values: List<Real> := []\ntotal: Real := sum(values)", true},
		{"sum does not accept List<Bool>", "total: Int := sum([true, false])", false},
		{"sum does not accept List<String>", `total: Int := sum(["a"])`, false},
		{"sum does not accept a String", `total: Int := sum("abc")`, false},
		{"sum does not accept a scalar", "total: Int := sum(5)", false},
		{"sum of List<Int> is not Real", "values: List<Int> := [1]\ntotal: Int := sum(values)", true},

		{"min of List<Int>", "values: List<Int> := [8, 3]\nvalue: Int := min(values)", true},
		{"max of List<Int>", "values: List<Int> := [8, 3]\nvalue: Int := max(values)", true},
		{"min of List<Real>", "values: List<Real> := [3.5, -2.0]\nvalue: Real := min(values)", true},
		{"max of List<Real>", "values: List<Real> := [3.5, -2.0]\nvalue: Real := max(values)", true},
		{"min does not accept a scalar", "value: Int := min(5)", false},
		{"max does not accept a Pair", `value: Int := max({"a": 1})`, false},
		{"min of List<Int> does not produce Real", "values: List<Int> := [1]\nvalue: Int := min(values)", true},
		{"max of List<Real> does not produce Int", "values: List<Real> := [1.0]\nvalue: Int := max(values)", false},

		{"a nested call keeps its exact type", "values: List<Int> := [-4, 2]\nvalue: Int := abs(min(values))", true},
		{"a nested Real call keeps Real", "values: List<Real> := [-4.5]\nvalue: Real := abs(min(values))", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			if !result.HasErrors() {
				t.Fatal("expected a Fundamentals diagnostic")
			}
		})
	}
}

// TestNumericFundamentalsRequireNonNullArguments covers the null-state rule:
// a MaybeNull or Null argument is rejected until it is refined NonNull.
func TestNumericFundamentalsRequireNonNullArguments(t *testing.T) {
	rejected := map[string]string{
		"null List receiver for sum": "values: List<Int> := null\ntotal: Int := sum(values)",
		"null List receiver for min": "values: List<Int> := null\nvalue: Int := min(values)",
		"null List receiver for max": "values: List<Int> := null\nvalue: Int := max(values)",
		"null argument for abs":      "value: Int := 5\nmagnitude: Int := abs(null)",
	}
	for name, text := range rejected {
		t.Run(name, func(t *testing.T) {
			_, result := analyzeText(t, text)
			requireSemanticCode(t, result, codeNullableUse)
		})
	}
	refined := "values: List<Int> := null\nif values != null {\n    write(sum(values))\n    write(min(values))\n    write(max(values))\n}\n"
	_, result := analyzeText(t, refined)
	requireSemanticClean(t, result)
}

// TestNumericFundamentalsAcceptConstantCollections records that the three
// reductions are pure reads, so a Constant List is a valid argument.
func TestNumericFundamentalsAcceptConstantCollections(t *testing.T) {
	_, result := analyzeText(t, "values: Constant List<Int> := [4, 1, 9]\nwrite(sum(values))\nwrite(min(values))\nwrite(max(values))\n")
	requireSemanticClean(t, result)
}

// TestNumericFundamentalsCallShape covers arity and the built-in rule that a
// Fundamentals entry point publishes no parameter names.
func TestNumericFundamentalsCallShape(t *testing.T) {
	for _, text := range []string{
		"write(abs())", "write(abs(1, 2))", "write(abs(value: 3))",
		"write(sum())", "write(sum([1], [2]))", "write(sum(values: [1, 2]))",
		"write(min(values: [1, 2]))", "write(max(values: [1, 2]))",
	} {
		_, result := analyzeText(t, text)
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != codeCallArguments {
			t.Fatalf("diagnostics for %q = %+v, want one %s", text, result.Diagnostics, codeCallArguments)
		}
	}
}

// TestRejectedNumericFundamentalsReportOneRootDiagnostic keeps an invalid call
// from cascading into derivative type or null-state diagnostics.
func TestRejectedNumericFundamentalsReportOneRootDiagnostic(t *testing.T) {
	for _, text := range []string{
		`total: Int := sum("abc") + 1`,
		`value: Real := abs("5") * 2.0`,
		`value: Int := min(5) + max(5)`,
	} {
		_, result := analyzeText(t, text)
		for _, diagnostic := range result.Diagnostics {
			if diagnostic.Code != codeCallArguments {
				t.Fatalf("diagnostics for %q = %+v, want only %s", text, result.Diagnostics, codeCallArguments)
			}
		}
		if len(result.Diagnostics) == 0 {
			t.Fatalf("expected a diagnostic for %q", text)
		}
	}
}

// TestPlannedFundamentalsRemainUnavailable keeps the still-unspecified
// Fundamentals names out of the predeclared surface.
func TestPlannedFundamentalsRemainUnavailable(t *testing.T) {
	for _, text := range []string{
		"write(round(1.5))", "write(bool(1))", "write(copy([1]))", "write(deepCopy([1]))",
		"write(swap(1, 2))", "write(combine([1], [2]))", "write(merge([1], [2]))", "write(jump([1], 2))",
	} {
		_, result := analyzeText(t, text)
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != codeUnknownName {
			t.Fatalf("diagnostics for %q = %+v, want one %s", text, result.Diagnostics, codeUnknownName)
		}
	}
}
