package parser

import (
	"strings"
	"testing"

	"ahdcode/internal/lexer"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/syntax/token"
)

func parseText(t *testing.T, text string) Result {
	t.Helper()
	file := source.NewFile(1, "test.ahd", text)
	lexed := lexer.Lex(file)
	if len(lexed.Diagnostics) != 0 {
		t.Fatalf("unexpected lexer diagnostics: %+v", lexed.Diagnostics)
	}
	return Parse(file, lexed.Tokens)
}

func requireClean(t *testing.T, result Result) {
	t.Helper()
	if result.HasErrors() {
		t.Fatalf("unexpected parser diagnostics: %+v", result.Diagnostics)
	}
}

func requireCode(t *testing.T, result Result, code string) {
	t.Helper()
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("expected diagnostic %s, got %+v", code, result.Diagnostics)
}

func expressionAt(t *testing.T, result Result, index int) ast.Expr {
	t.Helper()
	statement, ok := result.Program.Statements[index].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("statement %d is %T, want ExprStmt", index, result.Program.Statements[index])
	}
	return statement.Expression
}

func TestPowerIsRightAssociative(t *testing.T) {
	result := parseText(t, "2^3^2")
	requireClean(t, result)
	root, ok := expressionAt(t, result, 0).(*ast.BinaryExpr)
	if !ok || root.Operator != "^" {
		t.Fatalf("root = %#v, want power", root)
	}
	if _, ok := root.Right.(*ast.BinaryExpr); !ok {
		t.Fatalf("right operand = %T, want nested BinaryExpr", root.Right)
	}
	if _, ok := root.Left.(*ast.LiteralExpr); !ok {
		t.Fatalf("left operand = %T, want literal", root.Left)
	}
}

func TestNotBindsOutsideEquality(t *testing.T) {
	result := parseText(t, "not x == 5")
	requireClean(t, result)
	unary, ok := expressionAt(t, result, 0).(*ast.UnaryExpr)
	if !ok || unary.Operator != "not" {
		t.Fatalf("expression = %T %#v, want unary not", expressionAt(t, result, 0), expressionAt(t, result, 0))
	}
	if binary, ok := unary.Operand.(*ast.BinaryExpr); !ok || binary.Operator != "==" {
		t.Fatalf("not operand = %T %#v, want equality", unary.Operand, unary.Operand)
	}
}

func TestOperatorPrecedenceAndMultiwordOperators(t *testing.T) {
	tests := []struct {
		text     string
		operator string
	}{
		{"1 + 2 * 3", "+"},
		{"1 < 2 == true", "=="},
		{"true or false and true", "or"},
		{"item is not null", "is not"},
		{"item not in items", "not in"},
		{"object has not field", "has not"},
		{"a same b", "same"},
	}
	for _, test := range tests {
		t.Run(test.text, func(t *testing.T) {
			result := parseText(t, test.text)
			requireClean(t, result)
			binary, ok := expressionAt(t, result, 0).(*ast.BinaryExpr)
			if !ok || binary.Operator != test.operator {
				t.Fatalf("expression = %T %#v, want root %q", expressionAt(t, result, 0), expressionAt(t, result, 0), test.operator)
			}
		})
	}

	leftAssociative := parseText(t, "8 - 4 - 2")
	requireClean(t, leftAssociative)
	root := expressionAt(t, leftAssociative, 0).(*ast.BinaryExpr)
	if _, ok := root.Left.(*ast.BinaryExpr); !ok {
		t.Fatalf("subtraction left operand = %T, want nested BinaryExpr", root.Left)
	}

	unaryPower := parseText(t, "-2^2")
	requireClean(t, unaryPower)
	if unary, ok := expressionAt(t, unaryPower, 0).(*ast.UnaryExpr); !ok {
		t.Fatalf("expression = %T, want UnaryExpr", expressionAt(t, unaryPower, 0))
	} else if _, ok := unary.Operand.(*ast.BinaryExpr); !ok {
		t.Fatalf("unary operand = %T, want power expression", unary.Operand)
	}
}

func TestCallSeparatorsAndArgumentStyles(t *testing.T) {
	tests := []struct {
		name string
		text string
		code string
	}{
		{"same-line whitespace rejected", "swap(a b)", codeExpectedSeparator},
		{"same-line comma", "swap(a, b)", ""},
		{"newline separator", "swap(\n a\n b\n)", ""},
		{"mixed arguments rejected", `createUser("Ali", age: 25)`, codeMixedCallArguments},
		{"all named", `createUser(name: "Ali", age: 25)`, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parseText(t, test.text)
			if test.code == "" {
				requireClean(t, result)
			} else {
				requireCode(t, result, test.code)
			}
			if _, ok := expressionAt(t, result, 0).(*ast.CallExpr); !ok {
				t.Fatalf("expression = %T, want CallExpr", expressionAt(t, result, 0))
			}
		})
	}
}

func TestClassAndFunctionCallsShareCallExpr(t *testing.T) {
	result := parseText(t, "Student(name: \"Ali\")\ncalculate(1)")
	requireClean(t, result)
	for index := range 2 {
		call, ok := expressionAt(t, result, index).(*ast.CallExpr)
		if !ok {
			t.Fatalf("statement %d expression = %T, want CallExpr", index, expressionAt(t, result, index))
		}
		if _, ok := call.Callee.(*ast.IdentifierExpr); !ok {
			t.Fatalf("callee = %T, want unresolved IdentifierExpr", call.Callee)
		}
	}
}

func TestGenericTypeSeparators(t *testing.T) {
	bad := parseText(t, "values: Pair<String Int> := null")
	requireCode(t, bad, codeExpectedSeparator)

	good := parseText(t, "values: Pair<String, Int> := null\nother: Pair<\n String\n Int\n> := null")
	requireClean(t, good)
	declaration := good.Program.Statements[0].(*ast.VariableDecl)
	if declaration.Type.Name != "Pair" || len(declaration.Type.Arguments) != 2 {
		t.Fatalf("type = %#v, want Pair with two arguments", declaration.Type)
	}
}

func TestFunctionOnlyDeclarationModifiersAreDiagnosed(t *testing.T) {
	result := parseText(t, "value: Override Int := 1")
	requireCode(t, result, codeInvalidTypeSyntax)
}

func TestDeclarationsAssignmentsAndUpdates(t *testing.T) {
	result := parseText(t, strings.Join([]string{
		"count: Local Int := 0",
		"count = 1",
		"count += 2",
		"count ^= 3",
		"count++",
		"--count",
		"shared: Global Int",
	}, "\n"))
	requireClean(t, result)
	if len(result.Program.Statements) != 7 {
		t.Fatalf("got %d statements", len(result.Program.Statements))
	}
	if _, ok := result.Program.Statements[0].(*ast.VariableDecl); !ok {
		t.Fatalf("statement 0 = %T", result.Program.Statements[0])
	}
	for _, index := range []int{1, 2, 3} {
		if _, ok := result.Program.Statements[index].(*ast.AssignmentStmt); !ok {
			t.Fatalf("statement %d = %T", index, result.Program.Statements[index])
		}
	}
	for _, index := range []int{4, 5} {
		if _, ok := result.Program.Statements[index].(*ast.IncDecStmt); !ok {
			t.Fatalf("statement %d = %T", index, result.Program.Statements[index])
		}
	}
	if declaration := result.Program.Statements[6].(*ast.VariableDecl); !declaration.GlobalOnly {
		t.Fatal("Global declaration without initializer was not retained")
	}
}

func TestPostfixAndCollectionExpressions(t *testing.T) {
	result := parseText(t, strings.Join([]string{
		"object.member[1]",
		"items[1:3]",
		"items[:3]",
		"items[1:]",
		"[1, 2, 3]",
		"{\"a\": 1, \"b\": 2}",
		`"value={object.member}"`,
	}, "\n"))
	requireClean(t, result)
	wants := []any{(*ast.IndexExpr)(nil), (*ast.SliceExpr)(nil), (*ast.SliceExpr)(nil), (*ast.SliceExpr)(nil), (*ast.ListExpr)(nil), (*ast.PairExpr)(nil), (*ast.StringExpr)(nil)}
	for index, want := range wants {
		got := expressionAt(t, result, index)
		switch want.(type) {
		case *ast.IndexExpr:
			_, _ = got.(*ast.IndexExpr)
			if _, ok := got.(*ast.IndexExpr); !ok {
				t.Fatalf("%d: got %T", index, got)
			}
		case *ast.SliceExpr:
			if _, ok := got.(*ast.SliceExpr); !ok {
				t.Fatalf("%d: got %T", index, got)
			}
		case *ast.ListExpr:
			if _, ok := got.(*ast.ListExpr); !ok {
				t.Fatalf("%d: got %T", index, got)
			}
		case *ast.PairExpr:
			if _, ok := got.(*ast.PairExpr); !ok {
				t.Fatalf("%d: got %T", index, got)
			}
		case *ast.StringExpr:
			if _, ok := got.(*ast.StringExpr); !ok {
				t.Fatalf("%d: got %T", index, got)
			}
		}
	}
}

func TestFunctionDeclarationsAndFunctionValue(t *testing.T) {
	result := parseText(t, `square: Function := (
    number: Int
) -> Int {
    return number ^ 2
}
calculate: Overload Function := (value: Real) -> Real {
    return value
}
operation: Local Function := square`)
	requireClean(t, result)
	first := result.Program.Statements[0].(*ast.FunctionDecl)
	if first.Name != "square" || first.Flavor != ast.FunctionBase || len(first.Parameters) != 1 {
		t.Fatalf("first Function = %#v", first)
	}
	second := result.Program.Statements[1].(*ast.FunctionDecl)
	if second.Flavor != ast.FunctionOverload {
		t.Fatalf("flavor = %v, want Overload", second.Flavor)
	}
	if _, ok := result.Program.Statements[2].(*ast.VariableDecl); !ok {
		t.Fatalf("Function value binding = %T, want VariableDecl", result.Program.Statements[2])
	}
}

func TestClassesStructureAndMethods(t *testing.T) {
	result := parseText(t, `Person: Confidential Class<
> := {
    structure: Attributes := (
        name: String
    )
    describe: Function := () -> String {
        return attribute.name
    }
}
Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        age: Constant Int
        password: Local String
        secret: Confidential String
    )
    describe: Override Function := () -> String {
        return attribute.name
    }
}`)
	requireClean(t, result)
	person := result.Program.Statements[0].(*ast.ClassDecl)
	if !person.ExplicitRoot || person.Parent != nil || len(person.Members) != 2 || len(person.Modifiers) != 1 || person.Modifiers[0] != ast.ModifierConfidential {
		t.Fatalf("Person Class = %#v", person)
	}
	student := result.Program.Statements[1].(*ast.ClassDecl)
	if student.Parent == nil || student.Parent.Name != "Person" || len(student.Members) != 2 {
		t.Fatalf("Student Class = %#v", student)
	}
	structure := student.Members[0].(*ast.StructureDecl)
	if len(structure.Parameters) != 4 || !structure.Parameters[0].InheritedAttributes {
		t.Fatalf("structure parameters = %#v", structure.Parameters)
	}
	if !hasModifier(structure.Parameters[1].Modifiers, ast.ModifierConstant) ||
		!hasModifier(structure.Parameters[2].Modifiers, ast.ModifierLocal) ||
		!hasModifier(structure.Parameters[3].Modifiers, ast.ModifierConfidential) {
		t.Fatalf("structure modifiers were not preserved: %#v", structure.Parameters)
	}
	method := student.Members[1].(*ast.FunctionDecl)
	if method.Flavor != ast.FunctionOverride {
		t.Fatalf("method flavor = %v", method.Flavor)
	}
}

func TestControlFlowErrorHandlingAndBring(t *testing.T) {
	result := parseText(t, `if ready {
    write("yes")
}
else if waiting {
    continue
}
else {
    break
}
while active {
    count++
}
until done {
    count++
}
for item in items {
    write(item)
}
state status {
    condition "active" {
        write(status)
    }
    condition default {
        return
    }
}
attempt {
    toss Failure
}
except Error as error {
    write(error)
}
ultimately {
    write("done")
}
bring Mathematics
bring Mathematics as M
from Fundamentals bring all
from Mathematics bring (
    sqrt
    sin
)`)
	requireClean(t, result)
	wants := []any{
		(*ast.IfStmt)(nil), (*ast.WhileStmt)(nil), (*ast.UntilStmt)(nil), (*ast.ForStmt)(nil),
		(*ast.StateStmt)(nil), (*ast.AttemptStmt)(nil), (*ast.BringStmt)(nil), (*ast.BringStmt)(nil),
		(*ast.BringStmt)(nil), (*ast.BringStmt)(nil),
	}
	if len(result.Program.Statements) != len(wants) {
		t.Fatalf("statements = %d, want %d; diagnostics=%+v", len(result.Program.Statements), len(wants), result.Diagnostics)
	}
	if statement := result.Program.Statements[5].(*ast.AttemptStmt); len(statement.Excepts) != 1 || statement.Ultimately == nil {
		t.Fatalf("attempt = %#v", statement)
	}
	alias := result.Program.Statements[7].(*ast.BringStmt)
	if alias.Module != "Mathematics" || alias.Alias != "M" || !alias.Namespace {
		t.Fatalf("module alias = %#v", alias)
	}
	imports := result.Program.Statements[9].(*ast.BringStmt)
	if strings.Join(imports.Names, ",") != "sqrt,sin" {
		t.Fatalf("import names = %#v", imports.Names)
	}
}

func TestModuleAliasRequiresIdentifier(t *testing.T) {
	result := parseText(t, "bring Time as\n")
	if !result.HasErrors() {
		t.Fatal("missing module alias identifier was accepted")
	}
}

func TestDeclarationScopeDiagnosticsAreSyntactic(t *testing.T) {
	result := parseText(t, `if true {
    nested: Function := () -> Nothing {
        return
    }
    Inner: Class<> := {
    }
}`)
	requireCode(t, result, codeInvalidDeclarationScope)
	ifStatement := result.Program.Statements[0].(*ast.IfStmt)
	body := ifStatement.Branches[0].Body.Statements
	if _, ok := body[0].(*ast.FunctionDecl); !ok {
		t.Fatalf("nested declaration = %T", body[0])
	}
	if _, ok := body[1].(*ast.ClassDecl); !ok {
		t.Fatalf("nested declaration = %T", body[1])
	}
}

func TestNewlineContinuationRules(t *testing.T) {
	continued := parseText(t, "x = 5 +\n2")
	requireClean(t, continued)
	if len(continued.Program.Statements) != 1 {
		t.Fatalf("continued statements = %d", len(continued.Program.Statements))
	}

	notContinued := parseText(t, "x = 5\n+ 2")
	requireClean(t, notContinued)
	if len(notContinued.Program.Statements) != 2 {
		t.Fatalf("separate statements = %d", len(notContinued.Program.Statements))
	}
}

func TestContextualWordsRemainIdentifiers(t *testing.T) {
	result := parseText(t, "structure: Int := 1\nattribute: Int := 2")
	requireClean(t, result)
	if result.Program.Statements[0].(*ast.VariableDecl).Name != "structure" || result.Program.Statements[1].(*ast.VariableDecl).Name != "attribute" {
		t.Fatal("contextual words were not retained as identifiers")
	}
}

func TestTokensAndCommentTriviaRemainAvailable(t *testing.T) {
	text := "// heading\nvalue /* note */ = 1"
	result := parseText(t, text)
	requireClean(t, result)
	foundLine, foundBlock := false, false
	for _, item := range result.Tokens {
		for _, trivia := range item.LeadingTrivia {
			foundLine = foundLine || trivia.Kind == token.LineCommentTrivia
			foundBlock = foundBlock || trivia.Kind == token.BlockCommentTrivia
		}
	}
	if !foundLine || !foundBlock {
		t.Fatalf("comment trivia missing: line=%v block=%v", foundLine, foundBlock)
	}
}

func TestRecoveryKeepsFollowingStatement(t *testing.T) {
	result := parseText(t, "swap(a b)\nvalid = 1")
	requireCode(t, result, codeExpectedSeparator)
	if len(result.Program.Statements) != 2 {
		t.Fatalf("recovered statements = %d", len(result.Program.Statements))
	}
	if assignment, ok := result.Program.Statements[1].(*ast.AssignmentStmt); !ok {
		t.Fatalf("following statement = %T", result.Program.Statements[1])
	} else if identifier := assignment.Target.(*ast.IdentifierExpr); identifier.Name != "valid" {
		t.Fatalf("following target = %q", identifier.Name)
	}
}

func TestMalformedInputsDoNotPanic(t *testing.T) {
	inputs := []string{
		"(", "[1 2", "{a:}", "if {", "state x { condition", "attempt {}", "from bring",
		"name: Pair<String Int :=", `"unterminated {`, "Student(name:)", "x[::]", "else {}",
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			result := parseTextAllowLexErrors(input)
			if result.Program == nil || len(result.Tokens) == 0 {
				t.Fatalf("parser returned incomplete result for %q", input)
			}
		})
	}
}

func parseTextAllowLexErrors(text string) Result {
	file := source.NewFile(1, "malformed.ahd", text)
	return Parse(file, lexer.Lex(file).Tokens)
}

func TestForBindingTypeIsOptional(t *testing.T) {
	result := parseText(t, "for value in values {\n}\nfor value: Int in values {\n}\n")
	if result.HasErrors() {
		t.Fatalf("parser diagnostics: %+v", result.Diagnostics)
	}
	inferred, ok := result.Program.Statements[0].(*ast.ForStmt)
	if !ok || inferred.Type != nil || inferred.Name != "value" {
		t.Fatalf("untyped for = %#v", result.Program.Statements[0])
	}
	typed, ok := result.Program.Statements[1].(*ast.ForStmt)
	if !ok || typed.Type == nil || typed.Type.Name != "Int" {
		t.Fatalf("typed for = %#v", result.Program.Statements[1])
	}
}

func TestForBindingRejectsAScopeModifier(t *testing.T) {
	result := parseText(t, "for value: Local Int in values {\n}\n")
	if !result.HasErrors() {
		t.Fatal("a for binding is implicitly Local and must reject a scope modifier")
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected one focused diagnostic; received %+v", result.Diagnostics)
	}
	// Recovery must still produce the declared type rather than cascading.
	statement, ok := result.Program.Statements[0].(*ast.ForStmt)
	if !ok || statement.Type == nil || statement.Type.Name != "Int" {
		t.Fatalf("recovered for = %#v", result.Program.Statements[0])
	}
}

// TestFlexibleCommaExhaustiveMatrix exercises every separator style the
// v0.1.6 grammar must accept -- comma+newline, newline-only, trailing comma,
// the zero/one-item edges, and nesting -- across every construct the rule
// applies to: call arguments, List literals, Pair literals, and Function/
// structure parameter lists.
func TestFlexibleCommaExhaustiveMatrix(t *testing.T) {
	styles := []struct {
		name string
		item func(a, b string) string
	}{
		{"comma same line", func(a, b string) string { return a + ", " + b }},
		{"comma then newline", func(a, b string) string { return a + ",\n" + b }},
		{"newline only", func(a, b string) string { return a + "\n" + b }},
		{"trailing comma", func(a, b string) string { return a + ",\n" + b + ",\n" }},
	}

	t.Run("call arguments", func(t *testing.T) {
		for _, style := range styles {
			t.Run(style.name, func(t *testing.T) {
				text := "swap(" + style.item("a", "b") + ")"
				result := parseText(t, text)
				requireClean(t, result)
				call, ok := expressionAt(t, result, 0).(*ast.CallExpr)
				if !ok || len(call.Arguments) != 2 {
					t.Fatalf("expression = %#v, want a 2-argument CallExpr", expressionAt(t, result, 0))
				}
			})
		}
	})

	t.Run("List literal", func(t *testing.T) {
		for _, style := range styles {
			t.Run(style.name, func(t *testing.T) {
				text := "values: List<Int> := [" + style.item("1", "2") + "]"
				result := parseText(t, text)
				requireClean(t, result)
				decl := result.Program.Statements[0].(*ast.VariableDecl)
				list, ok := decl.Initializer.(*ast.ListExpr)
				if !ok || len(list.Elements) != 2 {
					t.Fatalf("initializer = %#v, want a 2-element ListExpr", decl.Initializer)
				}
			})
		}
	})

	t.Run("Pair literal", func(t *testing.T) {
		for _, style := range styles {
			t.Run(style.name, func(t *testing.T) {
				text := `values: Pair<String, Int> := {` + style.item(`"a": 1`, `"b": 2`) + `}`
				result := parseText(t, text)
				requireClean(t, result)
				decl := result.Program.Statements[0].(*ast.VariableDecl)
				pair, ok := decl.Initializer.(*ast.PairExpr)
				if !ok || len(pair.Entries) != 2 {
					t.Fatalf("initializer = %#v, want a 2-entry PairExpr", decl.Initializer)
				}
			})
		}
	})

	t.Run("Function parameters", func(t *testing.T) {
		for _, style := range styles {
			t.Run(style.name, func(t *testing.T) {
				text := "calculate: Function := (" + style.item("x: Int", "y: Int") + ") -> Int {\nreturn x + y\n}"
				result := parseText(t, text)
				requireClean(t, result)
				decl := result.Program.Statements[0].(*ast.FunctionDecl)
				if len(decl.Parameters) != 2 {
					t.Fatalf("parameters = %#v, want 2 parameters", decl.Parameters)
				}
			})
		}
	})

	t.Run("structure parameters", func(t *testing.T) {
		for _, style := range styles {
			t.Run(style.name, func(t *testing.T) {
				text := "Person: Class<> := {\nstructure: Attributes := (" + style.item("name: String", "age: Int") + ")\n}"
				result := parseText(t, text)
				requireClean(t, result)
				class := result.Program.Statements[0].(*ast.ClassDecl)
				structure := class.Members[0].(*ast.StructureDecl)
				if len(structure.Parameters) != 2 {
					t.Fatalf("parameters = %#v, want 2 parameters", structure.Parameters)
				}
			})
		}
	})
}

func TestFlexibleCommaSingleAndZeroItemEdges(t *testing.T) {
	result := parseText(t, strings.Join([]string{
		"noArgs()",
		"one(1)",
		"singleNoTrailing(1)",
		"singleTrailing(1,)",
		"empty: List<Int> := []",
		"singleList: List<Int> := [1]",
		"singleListTrailing: List<Int> := [1,]",
		"emptyPair: Pair<String, Int> := {}",
		`singlePair: Pair<String, Int> := {"a": 1}`,
	}, "\n"))
	requireClean(t, result)

	noArgs := result.Program.Statements[0].(*ast.ExprStmt).Expression.(*ast.CallExpr)
	if len(noArgs.Arguments) != 0 {
		t.Fatalf("noArgs() = %#v, want zero arguments", noArgs.Arguments)
	}
	for _, index := range []int{1, 2, 3} {
		call := result.Program.Statements[index].(*ast.ExprStmt).Expression.(*ast.CallExpr)
		if len(call.Arguments) != 1 {
			t.Fatalf("statement %d call = %#v, want one argument", index, call.Arguments)
		}
	}
	emptyList := result.Program.Statements[4].(*ast.VariableDecl).Initializer.(*ast.ListExpr)
	if len(emptyList.Elements) != 0 {
		t.Fatalf("empty List = %#v, want zero elements", emptyList.Elements)
	}
	for _, index := range []int{5, 6} {
		list := result.Program.Statements[index].(*ast.VariableDecl).Initializer.(*ast.ListExpr)
		if len(list.Elements) != 1 {
			t.Fatalf("statement %d List = %#v, want one element", index, list.Elements)
		}
	}
	emptyPair := result.Program.Statements[7].(*ast.VariableDecl).Initializer.(*ast.PairExpr)
	if len(emptyPair.Entries) != 0 {
		t.Fatalf("empty Pair = %#v, want zero entries", emptyPair.Entries)
	}
	singlePair := result.Program.Statements[8].(*ast.VariableDecl).Initializer.(*ast.PairExpr)
	if len(singlePair.Entries) != 1 {
		t.Fatalf("single Pair = %#v, want one entry", singlePair.Entries)
	}

	zeroParams := parseText(t, "noop: Function := () -> Nothing {\n}")
	requireClean(t, zeroParams)
	if fn := zeroParams.Program.Statements[0].(*ast.FunctionDecl); len(fn.Parameters) != 0 {
		t.Fatalf("zero-parameter Function = %#v", fn.Parameters)
	}

	oneParam := parseText(t, "identity: Function := (x: Int,) -> Int {\nreturn x\n}")
	requireClean(t, oneParam)
	if fn := oneParam.Program.Statements[0].(*ast.FunctionDecl); len(fn.Parameters) != 1 {
		t.Fatalf("one-parameter Function with trailing comma = %#v", fn.Parameters)
	}
}

func TestFlexibleCommaNestedCallsAndLists(t *testing.T) {
	result := parseText(t, "outer(\n inner(\n  1\n  2\n )\n [\n  3\n  4\n ]\n)")
	requireClean(t, result)
	outer := expressionAt(t, result, 0).(*ast.CallExpr)
	if len(outer.Arguments) != 2 {
		t.Fatalf("outer arguments = %#v, want 2", outer.Arguments)
	}
	inner, ok := outer.Arguments[0].Value.(*ast.CallExpr)
	if !ok || len(inner.Arguments) != 2 {
		t.Fatalf("nested call = %#v, want a 2-argument CallExpr", outer.Arguments[0].Value)
	}
	nestedList, ok := outer.Arguments[1].Value.(*ast.ListExpr)
	if !ok || len(nestedList.Elements) != 2 {
		t.Fatalf("nested List = %#v, want a 2-element ListExpr", outer.Arguments[1].Value)
	}
}

// TestFlexibleCommaCallbackFunctionValueArguments checks that a Function-
// typed value (declared once, then referenced by name, which is how
// AhdCode passes callbacks -- see examples/release_qa/17_function_callback.ahd)
// can be passed alongside another argument using every flexible separator
// style, including inside a call that itself spans multiple lines.
func TestFlexibleCommaCallbackFunctionValueArguments(t *testing.T) {
	result := parseText(t, strings.Join([]string{
		"addOne: Function := (",
		"    x: Int",
		") -> Int {",
		"    return x + 1",
		"}",
		"runWith(",
		" addOne",
		" 5",
		")",
	}, "\n"))
	requireClean(t, result)
	call := expressionAt(t, result, 1).(*ast.CallExpr)
	if len(call.Arguments) != 2 {
		t.Fatalf("arguments = %#v, want 2", call.Arguments)
	}
	if _, ok := call.Arguments[0].Value.(*ast.IdentifierExpr); !ok {
		t.Fatalf("first argument = %T, want the callback identifier", call.Arguments[0].Value)
	}
}

func TestFlexibleCommaInsideIfForAndClassBodies(t *testing.T) {
	result := parseText(t, strings.Join([]string{
		"if ready(\n a\n b\n) {",
		"    process(\n     a\n     b\n    )",
		"}",
		"else {",
		"    process(a, b)",
		"}",
		"for item in listOf(\n 1\n 2\n) {",
		"    write(item)",
		"}",
		"Worker: Class<> := {",
		"    run: Function := (\n     x: Int\n     y: Int\n    ) -> Int {",
		"        return x + y",
		"    }",
		"}",
	}, "\n"))
	requireClean(t, result)

	ifStatement, ok := result.Program.Statements[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("statement 0 = %T, want IfStmt", result.Program.Statements[0])
	}
	if _, ok := ifStatement.Branches[0].Condition.(*ast.CallExpr); !ok {
		t.Fatalf("if condition = %T, want CallExpr", ifStatement.Branches[0].Condition)
	}

	forStatement, ok := result.Program.Statements[1].(*ast.ForStmt)
	if !ok {
		t.Fatalf("statement 1 = %T, want ForStmt", result.Program.Statements[1])
	}
	if _, ok := forStatement.Iterable.(*ast.CallExpr); !ok {
		t.Fatalf("for iterable = %T, want CallExpr", forStatement.Iterable)
	}

	class, ok := result.Program.Statements[2].(*ast.ClassDecl)
	if !ok {
		t.Fatalf("statement 2 = %T, want ClassDecl", result.Program.Statements[2])
	}
	method, ok := class.Members[0].(*ast.FunctionDecl)
	if !ok || len(method.Parameters) != 2 {
		t.Fatalf("class method = %#v, want a 2-parameter FunctionDecl", class.Members[0])
	}
}

func TestFlexibleCommaModuleRootDeclarations(t *testing.T) {
	result := parseText(t, strings.Join([]string{
		"first: Int := 1",
		"greeting: List<String> := [",
		"    \"hi\"",
		"    \"there\"",
		"]",
		"describe: Function := (",
		"    name: String",
		"    times: Int",
		") -> Nothing {",
		"    write(name)",
		"}",
	}, "\n"))
	requireClean(t, result)
	if len(result.Program.Statements) != 3 {
		t.Fatalf("module-root statements = %d, want 3", len(result.Program.Statements))
	}
	greeting := result.Program.Statements[1].(*ast.VariableDecl).Initializer.(*ast.ListExpr)
	if len(greeting.Elements) != 2 {
		t.Fatalf("module-root List = %#v, want 2 elements", greeting.Elements)
	}
	describe := result.Program.Statements[2].(*ast.FunctionDecl)
	if len(describe.Parameters) != 2 {
		t.Fatalf("module-root Function parameters = %#v, want 2", describe.Parameters)
	}
}

// TestRHSMustStartOnSameLineAsOperator locks in the one strict structural
// rule that survives the rest of the grammar's flexibility: the first token
// of a := or = right-hand side must begin on the operator's own physical
// line. A violation must produce exactly the documented diagnostic message,
// not a generic parse error.
func TestRHSMustStartOnSameLineAsOperator(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		message string
	}{
		{
			name:    "declaration RHS on next line",
			text:    "values: List<Int> :=\n[\n    1\n    2\n]",
			message: "expected the assigned expression to begin after ':=' on the same line",
		},
		{
			name:    "declaration RHS call on next line",
			text:    "result: Int :=\ncalculate(x, y)",
			message: "expected the assigned expression to begin after ':=' on the same line",
		},
		{
			name:    "mutation RHS on next line",
			text:    "x: Int := 1\nx =\n2",
			message: "expected the assigned expression to begin after '=' on the same line",
		},
		{
			name:    "compound mutation RHS on next line",
			text:    "x: Int := 1\nx +=\n2",
			message: "expected the assigned expression to begin after '+=' on the same line",
		},
		{
			name:    "parameter default RHS on next line",
			text:    "f: Function := (x: Int :=\n1) -> Int {\nreturn x\n}",
			message: "expected the assigned expression to begin after ':=' on the same line",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parseText(t, test.text)
			requireCode(t, result, codeExpectedSameLineRHS)
			found := false
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Code == codeExpectedSameLineRHS && diagnostic.Message == test.message {
					found = true
				}
			}
			if !found {
				t.Fatalf("diagnostics = %+v, want message %q", result.Diagnostics, test.message)
			}
		})
	}
}

func TestRHSSameLineIsValidForEveryDeclarationShape(t *testing.T) {
	result := parseText(t, strings.Join([]string{
		"values: List<Int> := [",
		"    1",
		"    2",
		"]",
		"test: Function := () -> Bool {",
		"    return true",
		"}",
		"x: Int := 1",
		"x = 2",
		"x +=\t3",
	}, "\n"))
	requireClean(t, result)
}
