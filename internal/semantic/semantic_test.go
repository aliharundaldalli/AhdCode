package semantic

import (
	"strings"
	"testing"

	"ahdcode/internal/lexer"
	"ahdcode/internal/parser"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

func analyzeText(t *testing.T, text string) (parser.Result, Result) {
	t.Helper()
	file := source.NewFile(1, "test.ahd", text)
	lexed := lexer.Lex(file)
	if len(lexed.Diagnostics) != 0 {
		t.Fatalf("lexer diagnostics: %+v", lexed.Diagnostics)
	}
	parsed := parser.Parse(file, lexed.Tokens)
	if parsed.HasErrors() {
		t.Fatalf("parser diagnostics: %+v", parsed.Diagnostics)
	}
	return parsed, Analyze(parsed)
}

func requireSemanticClean(t *testing.T, result Result) {
	t.Helper()
	if result.HasErrors() {
		t.Fatalf("unexpected semantic diagnostics: %+v", result.Diagnostics)
	}
}

func requireSemanticCode(t *testing.T, result Result, code string) {
	t.Helper()
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("expected %s, got %+v", code, result.Diagnostics)
}

func TestAssignmentCompatibility(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"Int receives String", `x: Int := "Ali"`, false},
		{"Int widens to Real", `x: Real := 5`, true},
		{"Real does not narrow", `x: Int := 5.5`, false},
		{"valid reassignment", "x: Int := 5\nx = 8", true},
		{"invalid reassignment", "x: Int := 5\nx = 4.2", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := analyzeText(t, test.text)
			if test.ok {
				requireSemanticClean(t, result)
			} else {
				requireSemanticCode(t, result, codeTypeMismatch)
			}
		})
	}
}

func TestConditionsRequireBool(t *testing.T) {
	_, bad := analyzeText(t, "if 5 {\n}")
	requireSemanticCode(t, bad, codeConditionType)
	_, good := analyzeText(t, "if true {\n}")
	requireSemanticClean(t, good)
}

func TestNullStateAssignmentAndUse(t *testing.T) {
	_, bad := analyzeText(t, "value: Int := null\nvalue + 1")
	requireSemanticCode(t, bad, codeNullableUse)

	parsed, good := analyzeText(t, "value: Int := null\nvalue = 5\nvalue + 1")
	requireSemanticClean(t, good)
	last := parsed.Program.Statements[2].(*ast.ExprStmt).Expression.(*ast.BinaryExpr).Left
	if good.NullStates[last] != NonNull {
		t.Fatalf("value state = %s, want NonNull", good.NullStates[last])
	}
}

func TestBranchAndShortCircuitNullRefinement(t *testing.T) {
	parsed, result := analyzeText(t, `Student: Class<> := {
    structure: Attributes := (
        age: Int
    )
}
student: Student := null
if student != null and student.age >= 18 {
    write(student.age)
}`)
	requireSemanticClean(t, result)
	condition := parsed.Program.Statements[2].(*ast.IfStmt).Branches[0].Condition.(*ast.BinaryExpr)
	rightComparison := condition.Right.(*ast.BinaryExpr)
	member := rightComparison.Left.(*ast.MemberExpr)
	identifier := member.Object.(*ast.IdentifierExpr)
	if result.NullStates[identifier] != NonNull {
		t.Fatalf("short-circuit RHS student state = %s", result.NullStates[identifier])
	}
}

func TestScopeRedeclarationShadowingAndGlobalCapture(t *testing.T) {
	_, redeclared := analyzeText(t, "x: Int := 1\nx: Int := 2")
	requireSemanticCode(t, redeclared, codeRedeclaration)

	_, shadowed := analyzeText(t, `x: Int := 1
if true {
    x: Local Int := 2
    x = 3
}`)
	requireSemanticClean(t, shadowed)

	_, missingLocal := analyzeText(t, `if true {
    nested: Int := 1
}`)
	requireSemanticCode(t, missingLocal, codeMissingLocal)

	_, hidden := analyzeText(t, `counter: Int := 0
increase: Function := () -> Nothing {
    counter++
}`)
	requireSemanticCode(t, hidden, codeHiddenGlobal)

	_, explicit := analyzeText(t, `counter: Int := 0
increase: Function := () -> Nothing {
    counter: Global Int
    counter++
}`)
	requireSemanticClean(t, explicit)
}

func TestParametersForAndExceptBindingsAreImplicitLocal(t *testing.T) {
	_, result := analyzeText(t, `Failure: Class<Error> := {
    structure: Attributes := (message: String)
}
consume: Function := (items: List<Int>) -> Nothing {
    for item in items {
        current: Local Int := item
    }
    attempt {
        toss Failure(message: "x")
    }
    except Error as error {
        write(error.message)
    }
}`)
	requireSemanticClean(t, result)
}

func TestConstantValidation(t *testing.T) {
	_, reassigned := analyzeText(t, "PI: Constant Real := 3.14\nPI = 5")
	requireSemanticCode(t, reassigned, codeConstantAssignment)

	_, nullConstant := analyzeText(t, "id: Constant Int := null")
	requireSemanticCode(t, nullConstant, codeConstantInitializer)

	_, nonscalar := analyzeText(t, "values: Constant List<Int> := [1, 2]")
	requireSemanticCode(t, nonscalar, codeConstantInitializer)

	_, cyclic := analyzeText(t, "A: Constant Int := B\nB: Constant Int := A")
	requireSemanticCode(t, cyclic, codeConstantInitializer)

	_, scalar := analyzeText(t, "A: Constant Int := 2\nB: Constant Int := A ^ 3")
	requireSemanticClean(t, scalar)
}

func TestCompoundAssignments(t *testing.T) {
	_, realDivide := analyzeText(t, "x: Real := 5\nx /= 2")
	requireSemanticClean(t, realDivide)

	_, intDivide := analyzeText(t, "x: Int := 5\nx /= 2")
	requireSemanticCode(t, intDivide, codeTypeMismatch)

	_, intModulo := analyzeText(t, "x: Int := 5\nx %= 2")
	requireSemanticClean(t, intModulo)

	_, realModulo := analyzeText(t, "x: Real := 5\nx %= 2")
	requireSemanticCode(t, realModulo, codeOperatorType)
}

func TestPowerConstantExponentRules(t *testing.T) {
	parsed, result := analyzeText(t, `base: Int := 2
exponent: Int := 3
known: Int := base ^ (1 + 1)
negative: Real := base ^ -1
unknown: Real := base ^ exponent`)
	requireSemanticClean(t, result)
	knownInitializer := parsed.Program.Statements[2].(*ast.VariableDecl).Initializer
	negativeInitializer := parsed.Program.Statements[3].(*ast.VariableDecl).Initializer
	unknownInitializer := parsed.Program.Statements[4].(*ast.VariableDecl).Initializer
	if result.ExpressionTypes[knownInitializer].Kind() != types.IntKind || result.ExpressionTypes[negativeInitializer].Kind() != types.RealKind || result.ExpressionTypes[unknownInitializer].Kind() != types.RealKind {
		t.Fatalf("power types = %s, %s, %s", result.ExpressionTypes[knownInitializer], result.ExpressionTypes[negativeInitializer], result.ExpressionTypes[unknownInitializer])
	}

	_, badBinding := analyzeText(t, "base: Int := 2\nexponent: Int := 3\nresult: Int := base ^ exponent")
	requireSemanticCode(t, badBinding, codeTypeMismatch)

	_, goodUpdate := analyzeText(t, "base: Int := 2\nbase ^= 3")
	requireSemanticClean(t, goodUpdate)

	_, badUpdate := analyzeText(t, "base: Int := 2\nexponent: Int := 3\nbase ^= exponent")
	requireSemanticCode(t, badUpdate, codeTypeMismatch)
}

func TestOperatorRulesAndUpdates(t *testing.T) {
	_, valid := analyzeText(t, `i: Int := 2
i++
--i
text: String := "a" + "b"
repeated: String := "a" * 2
flag: Bool := true and not false`)
	requireSemanticClean(t, valid)

	_, realUpdate := analyzeText(t, "value: Real := 2\nvalue++")
	requireSemanticCode(t, realUpdate, codeOperatorType)

	_, coercion := analyzeText(t, `value: String := "age=" + 5`)
	requireSemanticCode(t, coercion, codeOperatorType)

	_, boolean := analyzeText(t, "value: Bool := true and 1")
	requireSemanticCode(t, boolean, codeOperatorType)

	_, unrelatedEquality := analyzeText(t, `value: Bool := "1" == 1`)
	requireSemanticCode(t, unrelatedEquality, codeOperatorType)

	_, numericEquality := analyzeText(t, "value: Bool := 1 == 1.0")
	requireSemanticClean(t, numericEquality)
}

func TestSignedIntBoundsAndConstantOverflow(t *testing.T) {
	_, valid := analyzeText(t, `MAX: Constant Int := 9223372036854775807
MIN: Constant Int := -9223372036854775808`)
	requireSemanticClean(t, valid)

	_, positiveOverflow := analyzeText(t, "BAD: Constant Int := 9223372036854775808")
	requireSemanticCode(t, positiveOverflow, codeConstantRange)

	_, arithmeticOverflow := analyzeText(t, "BAD: Constant Int := 9223372036854775807 + 1")
	requireSemanticCode(t, arithmeticOverflow, codeConstantRange)
}

func TestFunctionSignaturesCallsAndReturns(t *testing.T) {
	parsed, result := analyzeText(t, `square: Function := (
    x: Real
) -> Real {
    return x ^ 2
}
answer: Real := square(2)
named: Real := square(x: 3)`)
	requireSemanticClean(t, result)
	call := parsed.Program.Statements[1].(*ast.VariableDecl).Initializer
	if result.ExpressionTypes[call].Kind() != types.RealKind {
		t.Fatalf("call type = %s", result.ExpressionTypes[call])
	}

	_, wrongArgument := analyzeText(t, `square: Function := (x: Real) -> Real {
    return x
}
answer: Real := square("x")`)
	requireSemanticCode(t, wrongArgument, codeTypeMismatch)

	_, missingReturn := analyzeText(t, `choose: Function := (flag: Bool) -> Int {
    if flag {
        return 1
    }
}`)
	requireSemanticCode(t, missingReturn, codeMissingReturn)

	_, nothingReturn := analyzeText(t, `stop: Function := () -> Nothing {
    return 1
}`)
	requireSemanticCode(t, nothingReturn, codeReturnType)
}

func TestFunctionValueNeverFallsBackToDynamicCall(t *testing.T) {
	_, result := analyzeText(t, `calculate: Function := (
    operation: Function
) -> Int {
    return operation(1)
}`)
	requireSemanticCode(t, result, codePendingFeature)
}

func TestClassConstructionMembersAndTypeOperators(t *testing.T) {
	_, result := analyzeText(t, `Student: Class<> := {
    structure: Attributes := (
        name: String
        age: Int
    )
}
student: Student := Student(name: "Ali", age: 20)
adult: Bool := student.age >= 18
studentType: Bool := student is Student
hasAge: Bool := student has age
names: List<String> := ["Ali", "Ayşe"]
exists: Bool := student.name in names`)
	requireSemanticClean(t, result)

	_, badConstructor := analyzeText(t, `Student: Class<> := {
    structure: Attributes := (age: Int)
}
student: Student := Student(age: "old")`)
	requireSemanticCode(t, badConstructor, codeTypeMismatch)
}

func TestAllReachableIfPathsReturn(t *testing.T) {
	_, result := analyzeText(t, `choose: Function := (flag: Bool) -> Int {
    if flag {
        return 1
    }
    else {
        return 2
    }
}`)
	requireSemanticClean(t, result)
}

func TestStructuredSemanticResult(t *testing.T) {
	parsed, result := analyzeText(t, "x: Int := 1\nx + 2")
	requireSemanticClean(t, result)
	if len(result.Symbols) == 0 || len(result.ResolvedSymbols) == 0 || len(result.ExpressionTypes) == 0 || len(result.NullStates) == 0 {
		t.Fatalf("incomplete semantic result: %#v", result)
	}
	expression := parsed.Program.Statements[1].(*ast.ExprStmt).Expression
	if !types.Equal(result.ExpressionTypes[expression], types.Int) {
		t.Fatalf("expression type = %s", result.ExpressionTypes[expression])
	}
}

func TestRecoveredASTDoesNotPanic(t *testing.T) {
	inputs := []string{"(", "if {", "x: Pair<String Int :=", "attempt {}", "Student(name:)", "x[::]", "else {}"}
	for _, input := range inputs {
		t.Run(strings.ReplaceAll(input, "/", "_"), func(t *testing.T) {
			file := source.NewFile(1, "bad.ahd", input)
			parsed := parser.Parse(file, lexer.Lex(file).Tokens)
			result := Analyze(parsed)
			if result.ExpressionTypes == nil || result.NullStates == nil || result.ResolvedSymbols == nil {
				t.Fatal("semantic analyzer returned nil side tables")
			}
		})
	}
}

func TestLoopsStateAndControlContext(t *testing.T) {
	_, valid := analyzeText(t, `count: Int := 0
while count < 3 {
    count += 1
    continue
}
until count == 5 {
    count++
    if count == 4 {
        break
    }
}
label: String := "ready"
state label {
    condition "ready" {
        count = 5
    }
    condition default {
        count = 0
    }
}`)
	requireSemanticClean(t, valid)

	_, invalid := analyzeText(t, "break\ncontinue")
	requireSemanticCode(t, invalid, codeControlContext)
}

func TestStateAndAttemptCanProveAllReturns(t *testing.T) {
	_, stateResult := analyzeText(t, `choose: Function := (status: String) -> Int {
    state status {
        condition "yes" {
            return 1
        }
        condition default {
            return 0
        }
    }
}`)
	requireSemanticClean(t, stateResult)

	_, attemptResult := analyzeText(t, `Failure: Class<Error> := {
}
choose: Function := () -> Int {
    attempt {
        return 1
    }
    except Failure as error {
        return 2
    }
    ultimately {
        write("done")
    }
}`)
	requireSemanticClean(t, attemptResult)
}

func TestCollectionsIndexSliceAndMembership(t *testing.T) {
	_, result := analyzeText(t, `items: List<Int> := [1, 2, 3]
first: Int := items[0]
tail: List<Int> := items[1:]
scores: Pair<String, Int> := {"Ali": 90, "Ayşe": 95}
score: Int := scores["Ali"]
word: String := "AhdCode"
letter: String := word[0]
part: String := word[1:3]
hasAli: Bool := "Ali" in scores
hasTwo: Bool := 2 in items
hasCode: Bool := "Code" in word`)
	requireSemanticClean(t, result)

	_, badIndex := analyzeText(t, `items: List<Int> := [1]
value: Int := items["zero"]`)
	requireSemanticCode(t, badIndex, codeTypeMismatch)

	_, badMembership := analyzeText(t, `items: List<Int> := [1]
value: Bool := "one" in items`)
	requireSemanticCode(t, badMembership, codeOperatorType)
}

func TestStructureBodyMemberDeclaration(t *testing.T) {
	_, result := analyzeText(t, `Secret: Class<> := {
    structure: Attributes := (password: Local String) {
        attribute.code: Confidential String := password
    }
    reveal: Function := () -> String {
        return attribute.code
    }
}`)
	requireSemanticClean(t, result)
}

func TestOverrideValidation(t *testing.T) {
	_, valid := analyzeText(t, `Person: Class<> := {
    describe: Function := () -> String {
        return "person"
    }
}
Student: Class<Person> := {
    describe: Override Function := () -> String {
        return "student"
    }
}`)
	requireSemanticClean(t, valid)

	_, missing := analyzeText(t, `Person: Class<> := {
}
Student: Class<Person> := {
    describe: Override Function := () -> String {
        return "student"
    }
}`)
	requireSemanticCode(t, missing, codeInvalidMember)

	_, incompatible := analyzeText(t, `Person: Class<> := {
    describe: Function := (value: Int) -> String {
        return "person"
    }
}
Student: Class<Person> := {
    describe: Override Function := (value: Real) -> String {
        return "student"
    }
}`)
	requireSemanticCode(t, incompatible, codeTypeMismatch)
}

func TestFunctionDefaultsNamedCallsAndGlobalFunctionAlias(t *testing.T) {
	_, result := analyzeText(t, `greet: Function := (
    name: String
    title: String := "Student"
) -> String {
    return title + name
}
message: String := greet(name: "Ali")
copy: Function := greet
again: String := copy("Ali")
caller: Function := () -> String {
    greet: Global Function
    return greet("Ayşe")
}`)
	requireSemanticClean(t, result)

	_, missing := analyzeText(t, `greet: Function := (name: String) -> String {
    return name
}
message: String := greet()`)
	requireSemanticCode(t, missing, codeCallArguments)

	_, duplicate := analyzeText(t, `greet: Function := (name: String) -> String {
    return name
}
message: String := greet(name: "A", name: "B")`)
	requireSemanticCode(t, duplicate, codeCallArguments)
}

func TestOverloadHasNoDynamicFallback(t *testing.T) {
	_, result := analyzeText(t, `convert: Function := (value: Int) -> Int {
    return value
}
convert: Overload Function := (value: Real) -> Real {
    return value
}
answer: Int := convert(1)`)
	requireSemanticCode(t, result, codePendingFeature)
}

func TestReturnNullMetadataAndNullableCallResult(t *testing.T) {
	_, result := analyzeText(t, `Student: Class<> := {
    structure: Attributes := (age: Int)
}
find: Function := () -> Student {
    return null
}
student: Student := find()
student.age`)
	requireSemanticCode(t, result, codeNullableUse)
	var found *Symbol
	for _, symbol := range result.Symbols {
		if symbol.Name == "find" && symbol.Kind == FunctionSymbol {
			found = symbol
			break
		}
	}
	if found == nil || found.Callable.ReturnNull != Null {
		t.Fatalf("find return metadata = %#v", found)
	}
}

func TestTypeAndClassHierarchyDiagnostics(t *testing.T) {
	_, unknown := analyzeText(t, "value: Missing := null")
	requireSemanticCode(t, unknown, codeInvalidType)

	_, genericArity := analyzeText(t, "values: List<Int, Real> := null")
	requireSemanticCode(t, genericArity, codeInvalidType)

	_, cycle := analyzeText(t, `A: Class<B> := {
}
B: Class<A> := {
}
value: A := null`)
	requireSemanticCode(t, cycle, codeInvalidType)
}

func TestTypeOperatorsAndStrictSame(t *testing.T) {
	_, valid := analyzeText(t, `Student: Class<> := {
}
student: Student := Student()
isStudent: Bool := student is Student
hasName: Bool := student has name
strictNumeric: Bool := 1 same 1.0`)
	// has is reserved for Class/Object semantics and does not require the member
	// to exist at compile time; it asks about runtime member existence.
	requireSemanticClean(t, valid)

	_, invalidIs := analyzeText(t, "value: Bool := 1 is 2")
	requireSemanticCode(t, invalidIs, codeOperatorType)

	_, invalidHas := analyzeText(t, "value: Bool := 1 has member")
	requireSemanticCode(t, invalidHas, codeOperatorType)
}

func TestConstantScalarOperators(t *testing.T) {
	_, result := analyzeText(t, `TEXT: Constant String := "a" * 3
SUM: Constant Int := (2 + 3) * 4
RATIO: Constant Real := 5 / 2
ORDER: Constant Bool := 2 < 3
LOGIC: Constant Bool := not false and true`)
	requireSemanticClean(t, result)
}

func TestNullableBoolAndCallableUse(t *testing.T) {
	_, boolResult := analyzeText(t, "flag: Bool := null\nvalue: Bool := flag and true")
	requireSemanticCode(t, boolResult, codeNullableUse)

	_, callResult := analyzeText(t, "operation: Function := null\noperation()")
	requireSemanticCode(t, callResult, codeNullableUse)
}
