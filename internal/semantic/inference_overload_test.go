package semantic

import (
	"testing"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

func TestDirectFunctionBindingCarriesConcreteSignature(t *testing.T) {
	_, result := analyzeText(t, `add: Function := (a: Int, b: Int) -> Int {
    return a + b
}
operation: Function := add`)
	requireSemanticClean(t, result)
	operation := findSymbol(result, "operation", BindingSymbol)
	function, ok := operation.Type.(types.Function)
	if !ok || function.Signature == nil || len(function.Signature.Parameters) != 2 || function.Signature.Return.Kind() != types.IntKind {
		t.Fatalf("operation type = %#v", operation.Type)
	}
}

func TestFunctionValueConstraintsIgnoreParameterNames(t *testing.T) {
	_, result := analyzeText(t, `first: Function := (x: Int) -> Int {
    return x
}
second: Function := (y: Int) -> Int {
    return y
}
operation: Function := first
operation = second`)
	requireSemanticClean(t, result)
}

func TestFunctionValueConstraintsStillCheckParameterAndReturnTypes(t *testing.T) {
	tests := []string{
		`first: Function := (x: Int) -> Int { return x }
second: Function := (x: Real) -> Int { return int(x) }
operation: Function := first
operation = second`,
		`first: Function := (x: Int) -> Int { return x }
second: Function := (x: Int) -> Real { return x }
operation: Function := first
operation = second`,
	}
	for _, source := range tests {
		_, result := analyzeText(t, source)
		requireSemanticCode(t, result, codeConflictingFunction)
	}
}

func TestDirectNamedCallStillUsesDeclaredParameterName(t *testing.T) {
	_, valid := analyzeText(t, `triple: Function := (value: Int) -> Int {
    return value * 3
}
answer: Int := triple(value: 2)`)
	requireSemanticClean(t, valid)

	_, invalid := analyzeText(t, `triple: Function := (value: Int) -> Int {
    return value * 3
}
answer: Int := triple(x: 2)`)
	requireSemanticCode(t, invalid, codeCallArguments)
}

func TestRepeatedCompatibleCallbackConstraintsMerge(t *testing.T) {
	_, result := analyzeText(t, `use: Function := (operation: Function, x: Int) -> Int {
    a: Local Int := operation(x)
    b: Local Int := operation(x)
    return a + b
}`)
	requireSemanticClean(t, result)
	use := findSymbol(result, "use", FunctionSymbol)
	callback := use.Callable.Signature.Parameters[0].Type.(types.Function)
	if callback.Signature == nil || len(callback.Signature.Parameters) != 1 || callback.Signature.Parameters[0].Type.Kind() != types.IntKind || callback.Signature.Return.Kind() != types.IntKind {
		t.Fatalf("callback signature = %#v", callback.Signature)
	}
}

func TestZeroArgumentAndNamedCallbackInference(t *testing.T) {
	_, zero := analyzeText(t, `use: Function := (operation: Function) -> Int {
    return operation()
}`)
	requireSemanticClean(t, zero)
	use := findSymbol(zero, "use", FunctionSymbol)
	callback := use.Callable.Signature.Parameters[0].Type.(types.Function)
	if callback.Signature == nil || len(callback.Signature.Parameters) != 0 || callback.Signature.Return.Kind() != types.IntKind {
		t.Fatalf("zero-argument callback = %#v", callback.Signature)
	}

	_, named := analyzeText(t, `use: Function := (operation: Function) -> Int {
    first: Local Int := operation(value: 1)
    second: Local Int := operation(value: 2)
    return first + second
}`)
	requireSemanticClean(t, named)
	use = findSymbol(named, "use", FunctionSymbol)
	callback = use.Callable.Signature.Parameters[0].Type.(types.Function)
	if callback.Signature.Parameters[0].Name != "value" {
		t.Fatalf("named callback signature = %#v", callback.Signature)
	}
}

func TestCallbackParameterWideningAndConflict(t *testing.T) {
	_, widening := analyzeText(t, `use: Function := (operation: Function, x: Int) -> Int {
    first: Local Int := operation(x)
    second: Local Int := operation(1.5)
    return first + second
}`)
	requireSemanticClean(t, widening)
	use := findSymbol(widening, "use", FunctionSymbol)
	callback := use.Callable.Signature.Parameters[0].Type.(types.Function)
	if callback.Signature.Parameters[0].Type.Kind() != types.RealKind {
		t.Fatalf("widened callback parameter = %s", callback.Signature.Parameters[0].Type)
	}

	_, conflict := analyzeText(t, `bad: Function := (operation: Function, x: Int) -> Int {
    first: Local Int := operation(x)
    second: Local Int := operation("x")
    return first + second
}`)
	requireSemanticCode(t, conflict, codeConflictingFunction)
}

func TestCallbackReturnConstraintsChooseSingleCompatibleType(t *testing.T) {
	_, result := analyzeText(t, `use: Function := (operation: Function, x: Int) -> Real {
    exact: Local Int := operation(x)
    widened: Local Real := operation(x)
    return exact + widened
}`)
	requireSemanticClean(t, result)
	use := findSymbol(result, "use", FunctionSymbol)
	callback := use.Callable.Signature.Parameters[0].Type.(types.Function)
	if callback.Signature.Return.Kind() != types.IntKind {
		t.Fatalf("inferred return = %s, want Int", callback.Signature.Return)
	}
}

func TestConflictingCallbackReturnConstraints(t *testing.T) {
	_, result := analyzeText(t, `bad: Function := (operation: Function, x: Int) -> Int {
    a: Local Int := operation(x)
    b: Local String := operation(x)
    return a
}`)
	requireSemanticCode(t, result, codeConflictingFunction)
}

func TestCompletelyUnconstrainedFunctionParameter(t *testing.T) {
	_, result := analyzeText(t, `keep: Function := (operation: Function) -> Nothing {
    write("stored")
}`)
	requireSemanticCode(t, result, codeFunctionInference)
}

func TestConcreteFunctionBindingRejectsIncompatibleReassignment(t *testing.T) {
	_, result := analyzeText(t, `intIdentity: Function := (value: Int) -> Int {
    return value
}
stringIdentity: Function := (value: String) -> String {
    return value
}
operation: Function := intIdentity
operation = stringIdentity`)
	requireSemanticCode(t, result, codeConflictingFunction)
}

func TestIntFallsBackToRealOverload(t *testing.T) {
	parsed, result := analyzeText(t, `convert: Function := (value: Real) -> Real {
    return value
}
convert: Overload Function := (value: String) -> String {
    return value
}
answer: Real := convert(1)`)
	requireSemanticClean(t, result)
	call := parsed.Program.Statements[2].(*ast.VariableDecl).Initializer.(*ast.CallExpr)
	selected := result.SelectedCallables[call]
	if selected == nil || selected.Signature.Parameters[0].Type.Kind() != types.RealKind {
		t.Fatalf("selected overload = %#v", selected)
	}
}

func TestFewerDefaultsWins(t *testing.T) {
	parsed, result := analyzeText(t, `f: Function := (x: Int) -> Int {
    return 1
}
f: Overload Function := (x: Int, y: Int := 0) -> Int {
    return 2
}
answer: Int := f(5)`)
	requireSemanticClean(t, result)
	call := parsed.Program.Statements[2].(*ast.VariableDecl).Initializer.(*ast.CallExpr)
	selected := result.SelectedCallables[call]
	if selected == nil || len(selected.Signature.Parameters) != 1 {
		t.Fatalf("selected overload = %#v", selected)
	}
	trace := result.OverloadResolutions[call]
	if trace.Selected == "" {
		t.Fatal("selected overload was not recorded in resolution trace")
	}
}

func TestNamedOverloadMatchingIgnoresArgumentOrder(t *testing.T) {
	parsed, result := analyzeText(t, `create: Function := (name: String, age: Int) -> Int {
    return age
}
create: Overload Function := (id: Int) -> Int {
    return id
}
answer: Int := create(age: 20, name: "Ali")`)
	requireSemanticClean(t, result)
	call := parsed.Program.Statements[2].(*ast.VariableDecl).Initializer.(*ast.CallExpr)
	selected := result.SelectedCallables[call]
	if selected == nil || len(selected.Signature.Parameters) != 2 || selected.Signature.Parameters[0].Name != "name" {
		t.Fatalf("selected overload = %#v", selected)
	}
}

func TestNoMatchingOverloadWrongArityAndType(t *testing.T) {
	_, wrongArity := analyzeText(t, `f: Function := (x: Int) -> Int {
    return x
}
f: Overload Function := (x: Int, y: Int) -> Int {
    return x + y
}
answer: Int := f()`)
	requireSemanticCode(t, wrongArity, codeNoMatchingOverload)

	parsed, wrongType := analyzeText(t, `f: Function := (x: Int) -> Int {
    return x
}
f: Overload Function := (x: Real) -> Real {
    return x
}
answer: Int := f("no")`)
	requireSemanticCode(t, wrongType, codeNoMatchingOverload)
	call := parsed.Program.Statements[2].(*ast.VariableDecl).Initializer.(*ast.CallExpr)
	if len(wrongType.OverloadResolutions[call].Candidates) != 2 {
		t.Fatalf("resolution trace = %#v", wrongType.OverloadResolutions[call])
	}
}

func TestEqualBestOverloadsAreAmbiguous(t *testing.T) {
	_, result := analyzeText(t, `mix: Function := (left: Int, right: Real) -> Real {
    return left + right
}
mix: Overload Function := (left: Real, right: Int) -> Real {
    return left + right
}
answer: Real := mix(1, 1)`)
	requireSemanticCode(t, result, codeAmbiguousOverload)
}

func TestReturnTypeOnlyOverloadRejected(t *testing.T) {
	_, result := analyzeText(t, `f: Function := (x: Int) -> Int {
    return x
}
f: Overload Function := (x: Int) -> Real {
    return x
}`)
	requireSemanticCode(t, result, codeInvalidOverload)
}

func TestOverloadedFunctionValueWithoutContextIsAmbiguous(t *testing.T) {
	_, result := analyzeText(t, `calculate: Function := (x: Int) -> Int {
    return x
}
calculate: Overload Function := (x: Real) -> Real {
    return x
}
operation: Function := calculate`)
	requireSemanticCode(t, result, codeAmbiguousOverload)
}

func TestOverloadedFunctionValueNarrowedByCallbackContext(t *testing.T) {
	parsed, result := analyzeText(t, `calculate: Function := (x: Int) -> Int {
    return x
}
calculate: Overload Function := (x: Real) -> Real {
    return x
}
use: Function := (operation: Function, x: Int) -> Int {
    return operation(x)
}
answer: Int := use(calculate, 5)`)
	requireSemanticClean(t, result)
	outerCall := parsed.Program.Statements[3].(*ast.VariableDecl).Initializer.(*ast.CallExpr)
	functionValue := outerCall.Arguments[0].Value
	selected := result.SelectedFunctionValues[functionValue]
	if selected == nil || selected.Signature.Parameters[0].Type.Kind() != types.IntKind {
		t.Fatalf("selected callback overload = %#v", selected)
	}
}

func TestOrdinaryRecursionUsesPredeclaredSignature(t *testing.T) {
	parsed, result := analyzeText(t, `factorial: Function := (n: Int) -> Int {
    if n <= 1 {
        return 1
    }
    return n * factorial(n - 1)
}
answer: Int := factorial(5)`)
	requireSemanticClean(t, result)
	function := parsed.Program.Statements[0].(*ast.FunctionDecl)
	returnStatement := function.Body.Statements[1].(*ast.ReturnStmt)
	recursiveCall := returnStatement.Value.(*ast.BinaryExpr).Right.(*ast.CallExpr)
	if result.SelectedCallables[recursiveCall] == nil {
		t.Fatal("recursive call was not resolved")
	}
}

func TestSubclassArgumentAcceptedForParentParameter(t *testing.T) {
	_, result := analyzeText(t, `Person: Class<> := {
}
Student: Class<Person> := {
}
accept: Function := (person: Person) -> Nothing {
    write(person)
}
student: Student := Student()
accept(student)`)
	requireSemanticClean(t, result)
}

func findSymbol(result Result, name string, kind SymbolKind) *Symbol {
	for _, symbol := range result.Symbols {
		if symbol.Name == name && symbol.Kind == kind {
			return symbol
		}
	}
	return nil
}
