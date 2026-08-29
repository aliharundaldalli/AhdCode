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
