package semantic

import "testing"

const latexPreamble = "bring Latex\n\n"

func TestLatexStandardModuleHasExactSignatures(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"pdf", `Latex.pdf(source: "document", output: "out.pdf")`, true},
		{"pdf positional", `Latex.pdf("document", "out.pdf")`, true},
		{"pdfFile", `Latex.pdfFile(input: "document.tex", output: "out.pdf")`, true},
		{"escape", `value: String := Latex.escape("a&b")`, true},
		{"section", `value: String := Latex.section("Title")`, true},
		{"subsection", `value: String := Latex.subsection("Title")`, true},
		{"equation", `value: String := Latex.equation("x^2")`, true},
		{"document required only", `value: String := Latex.document("body")`, true},
		{"document all", `value: String := Latex.document(body: "body", title: "Title", author: "AhdCode")`, true},
		{"table", `value: String := Latex.table(["Name"], [["Ali"]])`, true},

		{"pdf rejects numeric source", `Latex.pdf(source: 123, output: "out.pdf")`, false},
		{"pdf rejects numeric output", `Latex.pdf(source: "document", output: 123)`, false},
		{"pdf wrong arity", `Latex.pdf("document")`, false},
		{"escape rejects Int", `value: String := Latex.escape(42)`, false},
		{"equation rejects Bool", `value: String := Latex.equation(true)`, false},
		{"document rejects extra argument", `value: String := Latex.document("b", "t", "a", "extra")`, false},
		{"table requires nested strings", `value: String := Latex.table(["Name"], [[1]])`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeWithStandardModules(t, latexPreamble+test.text)
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			if !result.HasErrors() {
				t.Fatal("expected a Latex semantic diagnostic")
			}
		})
	}
}

func TestLatexImportsAndErrorUseOrdinaryModuleRules(t *testing.T) {
	direct := analyzeWithStandardModules(t, `from Latex bring LatexError

failure: LatexError := LatexError("failed")
`)
	requireSemanticClean(t, direct)

	missing := analyzeWithStandardModules(t, `value: String := Latex.escape("text")`)
	if !missing.HasErrors() {
		t.Fatal("Latex must not be implicitly in scope")
	}

	userDeclaration := analyzeWithStandardModules(t, `Latex: Class<> := {}`)
	requireSemanticClean(t, userDeclaration)
}

// TestLatexTableMathColumnsAreOptionalAndTyped pins the new argument: it is
// optional, it is a List<Int>, and the existing two-argument call is unchanged.
func TestLatexTableMathColumnsAreOptionalAndTyped(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"two arguments still compile", `value: String := Latex.table(["A"], [["x"]])`, true},
		{"positional math columns", `value: String := Latex.table(["A"], [["x"]], [0])`, true},
		{"named math columns", `value: String := Latex.table(headers: ["A"], rows: [["x"]], mathColumns: [0])`, true},
		{"an empty column list is allowed", `value: String := Latex.table(["A"], [["x"]], [])`, true},

		{"math columns must be Int", `value: String := Latex.table(["A"], [["x"]], ["0"])`, false},
		{"math columns must be a List", `value: String := Latex.table(["A"], [["x"]], 0)`, false},
		{"a fourth argument is rejected", `value: String := Latex.table(["A"], [["x"]], [0], [1])`, false},
		{"table still produces String", `value: Int := Latex.table(["A"], [["x"]], [0])`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeWithStandardModules(t, "bring Latex\n\n"+test.text)
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			if !result.HasErrors() {
				t.Fatal("expected a Latex.table diagnostic")
			}
		})
	}
}
