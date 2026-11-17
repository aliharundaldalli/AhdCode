package parser

import (
	"strings"
	"testing"

	"ahdcode/internal/syntax/ast"
)

// codesOf lists the diagnostic codes of a parse, in order.
func codesOf(result Result) []string {
	codes := make([]string, 0, len(result.Diagnostics))
	for _, item := range result.Diagnostics {
		codes = append(codes, item.Code)
	}
	return codes
}

func requireOnly(t *testing.T, text string, codes ...string) Result {
	t.Helper()
	result := parseText(t, text)
	if strings.Join(codesOf(result), ",") != strings.Join(codes, ",") {
		t.Fatalf("%q\ndiagnostics = %+v\nwant codes %v", text, result.Diagnostics, codes)
	}
	return result
}

func TestReservedKeywordAsDeclaredNameIsOneDiagnostic(t *testing.T) {
	result := requireOnly(t, "not: Int := 70\nwrite(not)\nnot = not + 5\nif not > 50 {\n    write(\"{not}\")\n}\n", codeReservedKeywordName)
	only := result.Diagnostics[0]
	if only.Message != "'not' is a reserved keyword in AhdCode" ||
		only.Hint != "choose a different identifier, such as 'score'" ||
		only.Span.Start.Line != 1 || only.Span.Start.Column != 1 || only.Span.End.Column != 4 {
		t.Fatalf("diagnostic = %+v", only)
	}
	// The declaration still parses as the declaration the author meant.
	declaration, ok := result.Program.Statements[0].(*ast.VariableDecl)
	if !ok || declaration.Name != "not" || declaration.Type == nil || declaration.Type.Name != "Int" {
		t.Fatalf("first statement = %#v", result.Program.Statements[0])
	}
}

func TestReservedKeywordIsReportedOncePerKeyword(t *testing.T) {
	requireOnly(t, "state := 3\nstate = 4\nstate += 1\nstate++\n", codeReservedKeywordName)
	requireOnly(t, "not := 1\nstate := 2\n", codeReservedKeywordName, codeReservedKeywordName)
}

func TestReservedKeywordInOtherNamePositions(t *testing.T) {
	for _, text := range []string{
		"f: Function := (default: Int) -> Int {\n    return default\n}\nwrite(f(default: 1))\n",
		"for from in [1, 2] {\n    write(from)\n}\n",
		"attempt {\n    write(1)\n}\nexcept Error as all {\n    write(all)\n}\n",
		"bring Math as in\n",
		"write(value.default)\n",
	} {
		requireOnly(t, text, codeReservedKeywordName)
	}
}

func TestReservedKeywordSuggestionFallsBackToAGenericName(t *testing.T) {
	result := requireOnly(t, "Overload := 1\n", codeReservedKeywordName)
	if result.Diagnostics[0].Hint != "choose a different identifier, such as 'myOverload'" {
		t.Fatalf("hint = %q", result.Diagnostics[0].Hint)
	}
}

func TestKeywordsKeepTheirMeaningWithoutAMisuse(t *testing.T) {
	requireOnly(t, "ready := true\nif not ready {\n    write(1)\n}\nstate ready {\n    condition default {\n        write(2)\n    }\n}\nfor x in [1] {\n    write(x)\n}\n")
}

func TestAnIndependentErrorSurvivesBesideAReservedName(t *testing.T) {
	requireOnly(t, "not := 1\nvalues := [1 2]\n", codeReservedKeywordName, codeExpectedSeparator)
	// An error on the declaring line itself is not an echo either.
	requireOnly(t, "values := [1 2]\nnot := 1\n", codeExpectedSeparator, codeReservedKeywordName)
}

func TestOneSyntaxErrorIsOneDiagnostic(t *testing.T) {
	for text, code := range map[string]string{
		"x: Int = 5\nwrite(x)\n":                                   codeExpectedToken,
		"if x > 3\n    write(1)\n":                                 codeExpectedToken,
		"for i in [1, 2]\n    write(i)\n":                          codeExpectedToken,
		"if x == 3 then {\n    write(1)\n}\n":                      codeExpectedToken,
		"x := (3 + 4\nwrite(x)\n":                                  codeExpectedToken,
		"x = = 3\n":                                                codeUnexpectedToken,
		"x := 5\nif x > 3 {\n    write(1)\nwrite(2)\n":             codeExpectedToken,
		"function f(a: Int) -> Int {\n    return a\n}\nwrite(1)\n": codeExpectedSeparator,
		"def f(a):\n    return a\n":                                codeExpectedSeparator,
	} {
		requireOnly(t, text, code)
	}
}

func TestIndependentSyntaxErrorsAreAllReported(t *testing.T) {
	requireOnly(t, "a := [1 2]\nb := 3\nc := (4\n", codeExpectedSeparator, codeExpectedToken)
	requireOnly(t, "if a {\n    x := [1 2]\n}\nif b {\n    y := [3 4]\n}\n", codeExpectedSeparator, codeExpectedSeparator)
}

func TestAssignInATypedDeclarationNamesTheFix(t *testing.T) {
	result := requireOnly(t, "x: Int = 5\n", codeExpectedToken)
	if !strings.Contains(result.Diagnostics[0].Hint, "introduces its name with :=") {
		t.Fatalf("hint = %q", result.Diagnostics[0].Hint)
	}
	declaration, ok := result.Program.Statements[0].(*ast.VariableDecl)
	if !ok || declaration.Name != "x" || declaration.Initializer == nil {
		t.Fatalf("statement = %#v", result.Program.Statements[0])
	}
}

func TestUnclosedBlockNamesTheLineThatOpenedIt(t *testing.T) {
	result := requireOnly(t, "x := 5\nif x > 3 {\n    write(1)\n", codeExpectedToken)
	if !strings.Contains(result.Diagnostics[0].Hint, "on line 2") {
		t.Fatalf("hint = %q", result.Diagnostics[0].Hint)
	}
}

func TestForeignStatementKeywordsGetASpecificHint(t *testing.T) {
	for text, fragment := range map[string]string{
		"function f() {\n}\n": "name: Function := (parameter: Type) -> Type { ... }",
		"var x = 3\n":         "name := value",
		"print \"hello\"\n":   "write(value)",
		"class Point {\n}\n":  "Name: Class<> := { ... }",
		"import math\n":       "bring Module",
		"two words\n":         "place each independent statement on its own line",
	} {
		result := parseText(t, text)
		if len(result.Diagnostics) == 0 || !strings.Contains(result.Diagnostics[0].Hint, fragment) {
			t.Fatalf("%q\ndiagnostics = %+v\nwant a hint containing %q", text, result.Diagnostics, fragment)
		}
	}
}

// A brace on its own line is one mistake at each block, and the block is
// still read, so nothing inside or after it is reported.
func TestBraceOnItsOwnLineIsOneDiagnosticPerBlock(t *testing.T) {
	result := requireOnly(t, "if 2 > 1\n{\n    write(\"a\")\n}\nelse\n{\n    write(\"b\")\n}\nwhile false\n{\n    write(\"n\")\n}\nwrite(\"end\")\n",
		codeExpectedToken, codeExpectedToken, codeExpectedToken)
	for index, line := range []int{1, 5, 9} {
		item := result.Diagnostics[index]
		if item.Span.Start.Line != line || !strings.Contains(item.Hint, "move the { up to the end of this line") {
			t.Fatalf("diagnostic %d = %+v", index, item)
		}
	}
	if len(result.Program.Statements) != 3 {
		t.Fatalf("statements = %d, want the if, the while, and the write", len(result.Program.Statements))
	}
}
