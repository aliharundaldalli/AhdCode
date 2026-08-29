package semantic

import "testing"

func TestNumericConversionsHaveExactV01Signatures(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"Real to Int", "value: Int := int(3.7)", true},
		{"Int to Real", "value: Real := real(2)", true},
		{"String to Int", `value: Int := int("1")`, true},
		{"String to Real", `value: Real := real("1.5")`, true},
		{"int does not accept Int identity", "value: Int := int(1)", false},
		{"real does not accept Real identity", "value: Real := real(1.0)", false},
		{"bool remains unavailable", "value: Bool := bool(1)", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			if !result.HasErrors() {
				t.Fatal("expected a conversion diagnostic")
			}
		})
	}
}

func TestInvalidExpressionRecoverySuppressesCascades(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		names int
	}{
		{"unknown callee", "write(doesNotExist(2))", 1},
		{"invalid child under power", "write(doesNotExist(2) ^ -3)", 1},
		{"independent siblings", "write(doesNotExist(2) + anotherMissing)", 2},
		{"unknown condition", "if missingCondition {\n}", 1},
		{"unknown member receiver", "write(missingObject.value)", 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			if len(result.Diagnostics) != test.names {
				t.Fatalf("diagnostics = %+v, want exactly %d root diagnostic(s)", result.Diagnostics, test.names)
			}
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Code != codeUnknownName {
					t.Fatalf("diagnostic = %+v, want only %s", diagnostic, codeUnknownName)
				}
			}
		})
	}
}

func TestInvalidConversionHasOneSignatureDiagnostic(t *testing.T) {
	for _, text := range []string{`write(int(true))`, `write(real(false))`, `write(int(1) + "x")`} {
		_, result := analyzeText(t, text)
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != codeCallArguments {
			t.Fatalf("diagnostics for %q = %+v, want one %s", text, result.Diagnostics, codeCallArguments)
		}
	}
}

func TestUnavailableBoolConversionHasOneRootDiagnostic(t *testing.T) {
	_, result := analyzeText(t, "write(bool(5))")
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != codeUnknownName {
		t.Fatalf("diagnostics = %+v, want one %s", result.Diagnostics, codeUnknownName)
	}
}

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
		{"nullable List", "values: List<Int>? := null\nclear(values)", codeNullableUse},
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

// TestTakeFormsAndPromptRules covers the v0.1 terminal input contract:
// take() -> String and take(prompt: String) -> String, with no implicit
// conversion of the text that is read.
func TestTakeFormsAndPromptRules(t *testing.T) {
	tests := []struct {
		name string
		text string
		code string
	}{
		{"take with no prompt", "text: String := take()", ""},
		{"take with a prompt", `text: String := take("Text: ")`, ""},
		{"take with an interpolated prompt", "label: String := \"Name\"\ntext: String := take(\"{label}: \")", ""},
		{"take with a String binding prompt", "label: String := \"Name: \"\ntext: String := take(label)", ""},
		{"int of take", "number: Int := int(take())", ""},
		{"int of a prompted take", `number: Int := int(take("Number: "))`, ""},
		{"real of take", "decimal: Real := real(take())", ""},
		{"take does not yield Int", "number: Int := take()", codeTypeMismatch},
		{"take does not yield Real", "decimal: Real := take()", codeTypeMismatch},
		{"take does not yield Bool", "flag: Bool := take()", codeTypeMismatch},
		{"an Int prompt is rejected", "text: String := take(5)", codeTypeMismatch},
		{"a Bool prompt is rejected", "text: String := take(true)", codeTypeMismatch},
		{"a List prompt is rejected", "text: String := take([1])", codeTypeMismatch},
		{"two prompts are rejected", `text: String := take("A", "B")`, codeCallArguments},
		{"a named prompt is rejected", `text: String := take(prompt: "A")`, codeCallArguments},
		{"a null prompt is rejected", "text: String := take(null)", codeNullableUse},
		{"a nullable prompt binding is rejected", "prompt: String? := null\ntext: String := take(prompt)", codeNullableUse},
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

// TestNullableTakePromptIsReportedOnce keeps the null-state rejection a single
// focused diagnostic.
func TestNullableTakePromptIsReportedOnce(t *testing.T) {
	_, result := analyzeText(t, "prompt: String? := null\ntext: String := take(prompt)")
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected exactly one diagnostic; received %+v", result.Diagnostics)
	}
}
