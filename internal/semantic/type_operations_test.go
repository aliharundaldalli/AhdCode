package semantic

import (
	"testing"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const callbackPreamble = `double: Function := (
    x: Int
) -> Int {
    return x * 2
}

describe: Function := (
    x: Int
) -> String {
    return "n {x}"
}

isEven: Function := (
    x: Int
) -> Bool {
    return x % 2 == 0
}

toInt: Function := (
    text: String
) -> Int {
    return len(text)
}

`

// TestStringOperationsHaveExactSignatures pins the immutable String API: each
// operation takes NonNull String arguments and produces a new value.
func TestStringOperationsHaveExactSignatures(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"trim", `value: String := "  a  ".trim()`, true},
		{"lower", `value: String := "A".lower()`, true},
		{"upper", `value: String := "a".upper()`, true},
		{"capitalize", `value: String := "a".capitalize()`, true},
		{"split produces List<String>", `value: List<String> := "a,b".split(",")`, true},
		{"replace", `value: String := "a".replace("a", "b")`, true},
		{"contains produces Bool", `value: Bool := "a".contains("a")`, true},
		{"startsWith produces Bool", `value: Bool := "a".startsWith("a")`, true},
		{"endsWith produces Bool", `value: Bool := "a".endsWith("a")`, true},
		{"count produces Int", `value: Int := "a".count("a")`, true},
		{"index produces Int", `value: Int := "a".index("a")`, true},

		{"trim takes no argument", `value: String := "a".trim("b")`, false},
		{"replace takes two arguments", `value: String := "a".replace("a")`, false},
		{"split does not accept Int", `value: List<String> := "a".split(1)`, false},
		{"contains does not accept Int", `value: Bool := "a".contains(1)`, false},
		{"split does not produce String", `value: String := "a,b".split(",")`, false},
		{"count does not produce String", `value: String := "a".count("a")`, false},
		{"an unknown String member is not an operation", `value: String := "a".shout()`, false},
		{"String operations reject a named argument", `value: Bool := "a".contains(text: "a")`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			if !result.HasErrors() {
				t.Fatal("expected a String operation diagnostic")
			}
		})
	}
}

// TestListOperationsHaveExactSignatures pins the List API, including the two
// sort forms and the callback-typed operations.
func TestListOperationsHaveExactSignatures(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"reverse", "values: List<Int> := [1]\nvalues.reverse()", true},
		{"shuffle", "values: List<Int> := [1]\nvalues.shuffle()", true},
		{"sort Int", "values: List<Int> := [1]\nvalues.sort()", true},
		{"sort Real", "values: List<Real> := [1.0]\nvalues.sort()", true},
		{"sort String", `values: List<String> := ["a"]` + "\nvalues.sort()", true},
		{"count produces Int", "values: List<Int> := [1]\ntotal: Int := values.count(1)", true},
		{"index produces Int", "values: List<Int> := [1]\nposition: Int := values.index(1)", true},

		{"sort rejects List<Bool>", "values: List<Bool> := [true]\nvalues.sort()", false},
		{"sort rejects a nested List", "values: List<List<Int>> := [[1]]\nvalues.sort()", false},
		{"sort rejects a Pair element", `values: List<Pair<String, Int>> := [{"a": 1}]` + "\nvalues.sort()", false},
		{"reverse takes no argument", "values: List<Int> := [1]\nvalues.reverse(1)", false},
		{"shuffle takes no argument", "values: List<Int> := [1]\nvalues.shuffle(1)", false},
		{"shuffle rejects a named argument", "values: List<Int> := [1]\nvalues.shuffle(value: 1)", false},
		{"shuffle returns Nothing", "values: List<Int> := [1]\nwrite(values.shuffle())", false},
		{"count needs the element type", `values: List<Int> := [1]` + "\ntotal: Int := values.count(\"a\")", false},
		{"index needs the element type", `values: List<Int> := [1]` + "\nposition: Int := values.index(\"a\")", false},
		{"count rejects a named argument", "values: List<Int> := [1]\ntotal: Int := values.count(value: 1)", false},
		{"an unknown List member is not an operation", "values: List<Int> := [1]\nvalues.rotate()", false},

		{"map keeps the callback result type", callbackPreamble + "values: List<Int> := [1]\nresult: List<Int> := values.map(double)", true},
		{"map may change the element type", callbackPreamble + "values: List<Int> := [1]\nresult: List<String> := values.map(describe)", true},
		{"filter keeps the receiver type", callbackPreamble + "values: List<Int> := [1]\nresult: List<Int> := values.filter(isEven)", true},
		{"sort by an Int key", callbackPreamble + "values: List<Int> := [1]\nvalues.sort(double)", true},
		{"sort by a String key", callbackPreamble + "values: List<Int> := [1]\nvalues.sort(describe)", true},

		{"map result type is not the receiver type", callbackPreamble + "values: List<Int> := [1]\nresult: List<Int> := values.map(describe)", false},
		{"filter requires a Bool predicate", callbackPreamble + "values: List<Int> := [1]\nresult: List<Int> := values.filter(double)", false},
		{"filter requires the element parameter", callbackPreamble + "values: List<Int> := [1]\nresult: List<Int> := values.filter(toInt)", false},
		{"map requires the element parameter", callbackPreamble + "values: List<Int> := [1]\nresult: List<Int> := values.map(toInt)", false},
		{"sort rejects a Bool key", callbackPreamble + "values: List<Int> := [1]\nvalues.sort(isEven)", false},
		{"sort rejects a non-Function argument", "values: List<Int> := [1]\nvalues.sort(1)", false},
		{"map requires a Function", "values: List<Int> := [1]\nresult: List<Int> := values.map(1)", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			if !result.HasErrors() {
				t.Fatal("expected a List operation diagnostic")
			}
		})
	}
}

// TestListCallbacksNarrowOverloadedFunctionValues preserves the existing
// callback-context rule: the List element type is enough to choose among
// overloads that differ by parameter type. map infers the selected return type
// and keyed sort validates it after selection.
func TestListCallbacksNarrowOverloadedFunctionValues(t *testing.T) {
	parsed, result := analyzeText(t, `calculate: Function := (
    x: Int
) -> Int {
    return x * 2
}

calculate: Overload Function := (
    x: Real
) -> Real {
    return x * 2.0
}

values: List<Int> := [1, 2]
mapped: List<Int> := values.map(calculate)
values.sort(calculate)
`)
	requireSemanticClean(t, result)

	mapDecl := parsed.Program.Statements[3].(*ast.VariableDecl)
	mapCall := mapDecl.Initializer.(*ast.CallExpr)
	mapCallback := mapCall.Arguments[0].Value
	mapSelected := result.SelectedFunctionValues[mapCallback]
	if mapSelected == nil || mapSelected.Signature.Parameters[0].Type.Kind() != types.IntKind {
		t.Fatalf("selected map callback overload = %#v", mapSelected)
	}

	sortStatement := parsed.Program.Statements[4].(*ast.ExprStmt)
	sortCall := sortStatement.Expression.(*ast.CallExpr)
	sortCallback := sortCall.Arguments[0].Value
	sortSelected := result.SelectedFunctionValues[sortCallback]
	if sortSelected == nil || sortSelected.Signature.Parameters[0].Type.Kind() != types.IntKind {
		t.Fatalf("selected sort callback overload = %#v", sortSelected)
	}
}

// TestMapCallbackReturningNothingIsRejected keeps map from producing a
// List<Nothing>.
func TestMapCallbackReturningNothingIsRejected(t *testing.T) {
	_, result := analyzeText(t, `report: Function := (
    x: Int
) -> Nothing {
    write(x)
}

values: List<Int> := [1]
result: List<Int> := values.map(report)
`)
	requireSemanticCode(t, result, codeCallArguments)
}

// TestTypeOperationReceiversRequireNonNull covers the receiver null-state rule
// for both String and List operations.
func TestTypeOperationReceiversRequireNonNull(t *testing.T) {
	rejected := map[string]string{
		"null String receiver":  "text: String? := null\nvalue: String := text.trim()",
		"null List receiver":    "values: List<Int>? := null\ntotal: Int := values.count(5)",
		"null sort receiver":    "values: List<Int>? := null\nvalues.sort()",
		"null shuffle receiver": "values: List<Int>? := null\nvalues.shuffle()",
		"null map receiver":     callbackPreamble + "values: List<Int>? := null\nresult: List<Int> := values.map(double)",
		"null String argument":  "needle: String? := null\nvalue: Bool := \"abc\".contains(needle)",
		"null List argument":    "wanted: Int? := null\nvalues: List<Int> := [1]\ntotal: Int := values.count(wanted)",
	}
	for name, text := range rejected {
		t.Run(name, func(t *testing.T) {
			_, result := analyzeText(t, text)
			requireSemanticCode(t, result, codeNullableUse)
		})
	}
	refined := "text: String? := null\nif text != null {\n    write(text.trim())\n}\nvalues: List<Int>? := null\nif values != null {\n    values.sort()\n    values.shuffle()\n    write(values.count(1))\n}\n"
	_, result := analyzeText(t, refined)
	requireSemanticClean(t, result)
}

// TestConstantListsAllowReadsAndRejectMutations records that sort and reverse
// rewrite the receiver while the other operations only read it.
func TestConstantListsAllowReadsAndRejectMutations(t *testing.T) {
	for _, text := range []string{
		"values: Constant List<Int> := [3, 1]\nvalues.sort()",
		"values: Constant List<Int> := [3, 1]\nvalues.reverse()",
		"values: Constant List<Int> := [3, 1]\nvalues.shuffle()",
		callbackPreamble + "values: Constant List<Int> := [3, 1]\nvalues.sort(double)",
	} {
		_, result := analyzeText(t, text)
		requireSemanticCode(t, result, codeConstantAssignment)
	}
	_, result := analyzeText(t, callbackPreamble+`values: Constant List<Int> := [3, 1]
write(values.count(1))
write(values.index(1))
write(values.map(double))
write(values.filter(isEven))
`)
	requireSemanticClean(t, result)
}

// TestCallbackReturningNullIsRejected keeps a possibly-null key or predicate
// out of sort and filter, where no null result has a defined meaning.
func TestCallbackReturningNullIsRejected(t *testing.T) {
	preamble := `maybeKey: Function := (
    x: Int
) -> Int? {
    if x > 0 {
        return null
    }

    return x
}

maybeBool: Function := (
    x: Int
) -> Bool? {
    return null
}

`
	for _, text := range []string{
		preamble + "values: List<Int> := [1]\nvalues.sort(maybeKey)",
		preamble + "values: List<Int> := [1]\nresult: List<Int> := values.filter(maybeBool)",
	} {
		_, result := analyzeText(t, text)
		requireSemanticCode(t, result, codeNullableUse)
	}
	// map has no such restriction on the callback itself: its result List
	// simply picks up the callback's own return nullability.
	_, result := analyzeText(t, preamble+"values: List<Int> := [1]\nresult: List<Int?> := values.map(maybeKey)")
	requireSemanticClean(t, result)
}

// TestUserClassMethodsShadowBuiltinOperationNames records that the receiver
// type, not the member name, selects a built-in operation.
func TestUserClassMethodsShadowBuiltinOperationNames(t *testing.T) {
	_, result := analyzeText(t, `Report: Class<> := {
    structure: Attributes := (
        title: String
    )

    count: Function := (
    ) -> Int {
        return 7
    }

    map: Function := (
        factor: Int
    ) -> Int {
        return factor
    }
}

report: Report := Report("t")
total: Int := report.count()
scaled: Int := report.map(2)
`)
	requireSemanticClean(t, result)
}

// TestRejectedTypeOperationsReportOneRootDiagnostic keeps an invalid receiver
// or callback from cascading into derivative diagnostics.
func TestRejectedTypeOperationsReportOneRootDiagnostic(t *testing.T) {
	for _, text := range []string{
		"missing.add(1)",
		"missing.trim()",
		`write(unknownText.split(","))`,
		"values: List<Int> := [1]\nwrite(values.count(missingValue))",
	} {
		_, result := analyzeText(t, text)
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != codeUnknownName {
			t.Fatalf("diagnostics for %q = %+v, want one %s", text, result.Diagnostics, codeUnknownName)
		}
	}
	for _, text := range []string{
		callbackPreamble + "values: List<Int> := [1]\nwrite(values.filter(double))",
		callbackPreamble + "values: List<Int> := [1]\nwrite(values.map(toInt))",
		"values: List<Bool> := [true]\nvalues.sort()",
	} {
		_, result := analyzeText(t, text)
		if len(result.Diagnostics) != 1 {
			t.Fatalf("diagnostics for %q = %+v, want exactly one root diagnostic", text, result.Diagnostics)
		}
	}
}

// TestNoAliasNamesAreIntroduced keeps the frozen v0.1 surface free of the
// alternative spellings other languages use.
func TestNoAliasNamesAreIntroduced(t *testing.T) {
	for _, text := range []string{
		"values: List<Int> := [1]\nvalues.append(2)",
		"values: List<Int> := [1]\nvalues.push(2)",
		"values: List<Int> := [1]\nvalues.remove(1)",
		"values: List<Int> := [1]\nvalues.findIndex(1)",
		"values: List<Int> := [1]\nvalues.foreach(1)",
		"values: List<Int> := [1]\nvalues.select(1)",
		"values: List<Int> := [1]\nvalues.where(1)",
		"values: List<Int> := [1]\nvalues.transform(1)",
		"values: List<Int> := [1]\nvalues.reduce(1)",
		`write("a".strip())`,
		`write("a".toLowerCase())`,
	} {
		_, result := analyzeText(t, text)
		if !result.HasErrors() {
			t.Fatalf("%q must not resolve to a built-in operation", text)
		}
	}
}
