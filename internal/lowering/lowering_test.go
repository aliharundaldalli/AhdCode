package lowering

import (
	"strings"
	"testing"

	"ahdcode/internal/ir"
	"ahdcode/internal/module"
)

func lowerSources(t *testing.T, sources map[string]string, entry string) Result {
	t.Helper()
	workspace := module.NewInMemoryWorkspace(sources)
	frontend := module.NewCompiler(workspace, workspace).Compile(entry)
	if frontend.HasErrors() {
		t.Fatalf("frontend diagnostics: %+v", frontend.Diagnostics)
	}
	result := LowerCompilation(frontend)
	if result.HasErrors() {
		t.Fatalf("lowering diagnostics: %+v", result.Diagnostics)
	}
	return result
}

func moduleIR(t *testing.T, compilation *ir.Compilation, name string) *ir.Module {
	t.Helper()
	for _, current := range compilation.Modules {
		if current.Name == name {
			return current
		}
	}
	t.Fatalf("module %s not found", name)
	return nil
}

func globalIR(t *testing.T, module *ir.Module, name string) *ir.Global {
	t.Helper()
	for _, global := range module.Globals {
		if global.Name == name {
			return global
		}
	}
	t.Fatalf("global %s not found", name)
	return nil
}

// globalInitializer returns the module-root initializer of one global. A
// module-root binding declares storage separately from the statement that
// initializes it, so the initializer lives in the module statement stream.
func globalInitializer(t *testing.T, module *ir.Module, name string) ir.Expr {
	t.Helper()
	for _, statement := range module.Init.Statements {
		binding, ok := statement.(*ir.BindingStmt)
		if ok && binding.Storage == ir.ModuleStorage && binding.Name == name {
			return binding.Initializer
		}
	}
	t.Fatalf("module-root initializer for %s not found", name)
	return nil
}

func functionIR(t *testing.T, module *ir.Module, name string) *ir.Function {
	t.Helper()
	for _, function := range module.Functions {
		if function.Name == name && function.Kind != ir.ConstructorFunction {
			return function
		}
	}
	t.Fatalf("function %s not found", name)
	return nil
}

func TestNumericLoweringIsExplicitAndTyped(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `a: Real := 5
explicitReal: Real := real(2)
explicitInt: Int := int(3.7)
parsedReal: Real := real("2.5e1")
parsedInt: Int := int("25")
minimum: Int := -9223372036854775808
b: Int := 2 + 3
c: Real := 2 + 3.5
d: Real := 5 / 2
power: Int := 2 ^ 3
negativePower: Int := 2 ^ -1
runtimePower: Function := (exponent: Int) -> Int {
    return 2 ^ exponent
}`}, "/Main.ahd")
	main := moduleIR(t, result.Compilation, "Main")
	if _, ok := globalInitializer(t, main, "a").(*ir.ConvertExpr); !ok {
		t.Fatalf("Real initializer = %T, want ConvertExpr", globalInitializer(t, main, "a"))
	}
	explicitReal := globalInitializer(t, main, "explicitReal").(*ir.ConvertExpr)
	if explicitReal.From.Kind != ir.IntType || explicitReal.Type.Kind != ir.RealType {
		t.Fatalf("real conversion = %s -> %s", explicitReal.From, explicitReal.Type)
	}
	explicitInt := globalInitializer(t, main, "explicitInt").(*ir.ConvertExpr)
	if explicitInt.From.Kind != ir.RealType || explicitInt.Type.Kind != ir.IntType {
		t.Fatalf("int conversion = %s -> %s", explicitInt.From, explicitInt.Type)
	}
	parsedReal := globalInitializer(t, main, "parsedReal").(*ir.ConvertExpr)
	if parsedReal.From.Kind != ir.StringType || parsedReal.Type.Kind != ir.RealType {
		t.Fatalf("parsed Real conversion = %s -> %s", parsedReal.From, parsedReal.Type)
	}
	parsedInt := globalInitializer(t, main, "parsedInt").(*ir.ConvertExpr)
	if parsedInt.From.Kind != ir.StringType || parsedInt.Type.Kind != ir.IntType {
		t.Fatalf("parsed Int conversion = %s -> %s", parsedInt.From, parsedInt.Type)
	}
	minimum := globalInitializer(t, main, "minimum").(*ir.UnaryExpr)
	if minimum.Op != "CheckedIntNegate" || minimum.Operand.(*ir.LiteralExpr).Value != "9223372036854775808" {
		t.Fatalf("minimum Int lowering = %#v", minimum)
	}
	if add := globalInitializer(t, main, "b").(*ir.BinaryExpr); add.Op != "CheckedIntAdd" {
		t.Fatalf("Int add op = %s", add.Op)
	}
	mixed := globalInitializer(t, main, "c").(*ir.BinaryExpr)
	if mixed.Op != "RealAdd" {
		t.Fatalf("mixed op = %s", mixed.Op)
	}
	if _, ok := mixed.Left.(*ir.ConvertExpr); !ok {
		t.Fatalf("mixed left = %T, want ConvertExpr", mixed.Left)
	}
	division := globalInitializer(t, main, "d").(*ir.BinaryExpr)
	if division.Op != "RealDivide" {
		t.Fatalf("division op = %s", division.Op)
	}
	if _, left := division.Left.(*ir.ConvertExpr); !left {
		t.Fatal("division left was not widened")
	}
	if _, right := division.Right.(*ir.ConvertExpr); !right {
		t.Fatal("division right was not widened")
	}
	if power := globalInitializer(t, main, "power").(*ir.BinaryExpr); power.Op != "CheckedIntPower" {
		t.Fatalf("constant power op = %s", power.Op)
	}
	if power := globalInitializer(t, main, "negativePower").(*ir.BinaryExpr); power.Op != "CheckedIntPower" {
		t.Fatalf("negative power op = %s", power.Op)
	}
	runtime := functionIR(t, main, "runtimePower")
	ret := runtime.Body.Statements[0].(*ir.ReturnStmt)
	if operation := ret.Value.(*ir.BinaryExpr); operation.Op != "CheckedIntPower" {
		t.Fatalf("runtime power op = %s", operation.Op)
	}
}

func TestCallsNormalizeNamesDefaultsOverloadsAndCallbacks(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `create: Function := (name: String, age: Int := 25) -> Int {
    return age
}
calculate: Function := (value: Int) -> Int {
    return value
}
calculate: Overload Function := (value: Real) -> Real {
    return value
}
use: Function := (operation: Function, value: Int) -> Int {
    return operation(value)
}
named: Int := create(age: 30, name: "Ali")
defaulted: Int := create(name: "Veli")
exact: Int := calculate(5)
callback: Int := use(calculate, 5)`}, "/Main.ahd")
	main := moduleIR(t, result.Compilation, "Main")
	named := globalInitializer(t, main, "named").(*ir.CallExpr)
	if named.Arguments[0].ParameterName != "name" || named.Arguments[1].ParameterName != "age" {
		t.Fatalf("named args not canonical: %#v", named.Arguments)
	}
	defaulted := globalInitializer(t, main, "defaulted").(*ir.CallExpr)
	if !defaulted.Arguments[1].UsesDefault || defaulted.Arguments[1].Value != nil {
		t.Fatalf("default arg = %#v", defaulted.Arguments[1])
	}
	if functionIR(t, main, "create").Parameters[1].Default == nil {
		t.Fatal("callee default expression was not preserved")
	}
	exact := globalInitializer(t, main, "exact").(*ir.CallExpr)
	if !strings.Contains(string(exact.Callable), "value:Int") || strings.Contains(string(exact.Callable), "Real") {
		t.Fatalf("selected overload ID = %s", exact.Callable)
	}
	callback := globalInitializer(t, main, "callback").(*ir.CallExpr)
	if _, ok := callback.Arguments[0].Value.(*ir.FunctionValueExpr); !ok {
		t.Fatalf("callback value = %T", callback.Arguments[0].Value)
	}
	use := functionIR(t, main, "use")
	indirect := use.Body.Statements[0].(*ir.ReturnStmt).Value.(*ir.CallExpr)
	if indirect.Callable == "" || indirect.Callee == nil {
		t.Fatalf("indirect call is not concretely typed: %#v", indirect)
	}
}

func TestCrossModuleConstructionAndCallableIdentities(t *testing.T) {
	result := lowerSources(t, map[string]string{
		"/Models.ahd": `Student: Class<> := {
    structure: Attributes := (name: String)
}`,
		"/Helpers.ahd": `identity: Function := (value: Int := 7) -> Int { return value }`,
		"/Main.ahd": `from Models bring Student
from Helpers bring identity
student: Student := Student(name: "Ali")
answer: Int := identity(5)
defaultAnswer: Int := identity()`,
	}, "/Main.ahd")
	// The built-in Class catalog module precedes the user modules, which stay
	// in dependency order with the entry module last.
	if len(result.Compilation.Modules) != 4 || result.Compilation.Modules[0].ID != BuiltinModuleID || result.Compilation.Modules[3].Name != "Main" {
		t.Fatalf("module order = %#v", result.Compilation.Modules)
	}
	main := moduleIR(t, result.Compilation, "Main")
	construction := globalInitializer(t, main, "student").(*ir.ConstructExpr)
	if !strings.Contains(string(construction.Class), "mem:/Models.ahd::class::Student") {
		t.Fatalf("ClassID = %s", construction.Class)
	}
	call := globalInitializer(t, main, "answer").(*ir.CallExpr)
	if !strings.Contains(string(call.Callable), "mem:/Helpers.ahd") {
		t.Fatalf("external CallableID = %s", call.Callable)
	}
	defaultCall := globalInitializer(t, main, "defaultAnswer").(*ir.CallExpr)
	if len(defaultCall.Arguments) != 1 || !defaultCall.Arguments[0].UsesDefault {
		t.Fatalf("imported default call = %#v", defaultCall.Arguments)
	}
	if functionIR(t, moduleIR(t, result.Compilation, "Helpers"), "identity").Parameters[0].Default == nil {
		t.Fatal("dependency default expression missing from CompilationIR")
	}
}

func TestMathStandardModuleLowersStableBuiltinIdentities(t *testing.T) {
	result := lowerSources(t, map[string]string{
		"/Main.ahd": `bring Math
pi: Real := Math.PI
root: Real := Math.sqrt(25)
rounded: Real := Math.round(3.14159, 2)`,
	}, "/Main.ahd")
	if len(result.Compilation.Modules) != 3 || result.Compilation.Modules[0].ID != BuiltinModuleID || result.Compilation.Modules[1].ID != "builtin:Math" {
		t.Fatalf("Math module order = %#v", result.Compilation.Modules)
	}
	main := moduleIR(t, result.Compilation, "Main")
	pi, ok := globalInitializer(t, main, "pi").(*ir.LiteralExpr)
	if !ok || pi.Kind != ir.RealLiteral || pi.Value != "3.141592653589793" {
		t.Fatalf("Math.PI lowering = %#v", pi)
	}
	root := globalInitializer(t, main, "root").(*ir.CallExpr)
	if root.Callable != "builtin:Math::sqrt" || len(root.Arguments) != 1 {
		t.Fatalf("Math.sqrt lowering = %#v", root)
	}
	if _, widened := root.Arguments[0].Value.(*ir.ConvertExpr); !widened {
		t.Fatalf("Math.sqrt Int widening = %T", root.Arguments[0].Value)
	}
	rounded := globalInitializer(t, main, "rounded").(*ir.CallExpr)
	if rounded.Callable != "builtin:Math::round" || len(rounded.Arguments) != 2 {
		t.Fatalf("Math.round lowering = %#v", rounded)
	}
}

func TestNullInterpolationAndCollectionsAreConcrete(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `Student: Class<> := {}
student: Student := null
present: Bool := student != null
name: String := "Ali"
greeting: String := "Hello {name}"
items: List<Real> := [1, 2.5]
first: Real := items[0]
tail: List<Real> := items[1:]
scores: Pair<String, Int> := {"Ali": 90}
score: Int := scores["Ali"]`}, "/Main.ahd")
	main := moduleIR(t, result.Compilation, "Main")
	nullValue := globalInitializer(t, main, "student").(*ir.NullExpr)
	if nullValue.Type.Kind != ir.ClassType || nullValue.Type.Class == "" {
		t.Fatalf("typed null = %#v", nullValue.Type)
	}
	comparison := globalInitializer(t, main, "present").(*ir.BinaryExpr)
	if comparison.Right.(*ir.NullExpr).Type.Class != nullValue.Type.Class {
		t.Fatal("null comparison lost typed context")
	}
	text := globalInitializer(t, main, "greeting").(*ir.StringExpr)
	if len(text.Parts) != 2 {
		t.Fatalf("string parts = %#v", text.Parts)
	}
	if _, ok := text.Parts[1].ToString.(*ir.ToStringExpr); !ok {
		t.Fatalf("interpolation part = %T", text.Parts[1].ToString)
	}
	list := globalInitializer(t, main, "items").(*ir.ListExpr)
	if list.ElementType.Kind != ir.RealType {
		t.Fatalf("List type = %s", list.ElementType)
	}
	if _, ok := list.Elements[0].(*ir.ConvertExpr); !ok {
		t.Fatalf("List widening = %T", list.Elements[0])
	}
	if _, ok := globalInitializer(t, main, "first").(*ir.IndexExpr); !ok {
		t.Fatalf("index expression = %T", globalInitializer(t, main, "first"))
	}
	if slice, ok := globalInitializer(t, main, "tail").(*ir.SliceExpr); !ok || slice.End != nil {
		t.Fatalf("slice expression = %#v", globalInitializer(t, main, "tail"))
	}
	pair := globalInitializer(t, main, "scores").(*ir.PairExpr)
	if pair.KeyType.Kind != ir.StringType || pair.ValueType.Kind != ir.IntType {
		t.Fatalf("Pair types = %s/%s", pair.KeyType, pair.ValueType)
	}
	if _, ok := globalInitializer(t, main, "score").(*ir.IndexExpr); !ok {
		t.Fatalf("Pair index expression = %T", globalInitializer(t, main, "score"))
	}
}

func TestStructuredControlAndSingleEvaluationTargets(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `Failure: Class<Error> := {}
flow: Function := (items: List<Int>, done: Bool, text: String, values: Pair<String, Int>) -> Nothing {
    count: Local Int := 0
    count += 1
    count++
    until done {
        continue
    }
    for item in items {
        write(item)
    }
    for character in text {
        write(character)
    }
    for key in values {
        write(key)
    }
    state count {
        condition 0 {
            write("zero")
        }
        condition default {
            write("other")
        }
    }
    attempt {
        write("try")
    }
    except Error as error {
        write(error.message)
    }
    ultimately {
        write("done")
    }
    return
}`}, "/Main.ahd")
	flow := functionIR(t, moduleIR(t, result.Compilation, "Main"), "flow")
	if _, ok := flow.Body.Statements[1].(*ir.CompoundAssignStmt); !ok {
		t.Fatalf("compound = %T", flow.Body.Statements[1])
	}
	if _, ok := flow.Body.Statements[2].(*ir.UpdateStmt); !ok {
		t.Fatalf("update = %T", flow.Body.Statements[2])
	}
	until := flow.Body.Statements[3].(*ir.DoUntilStmt)
	if !until.ContinueChecksCondition {
		t.Fatal("until continue edge is wrong")
	}
	loop := flow.Body.Statements[4].(*ir.ForStmt)
	if !loop.Snapshot || loop.Kind != ir.ListElements || loop.IterationType.Kind != ir.IntType {
		t.Fatalf("for = %#v", loop)
	}
	stringLoop := flow.Body.Statements[5].(*ir.ForStmt)
	if !stringLoop.Snapshot || stringLoop.Kind != ir.StringCharacters || stringLoop.IterationType.Kind != ir.StringType {
		t.Fatalf("String for = %#v", stringLoop)
	}
	pairLoop := flow.Body.Statements[6].(*ir.ForStmt)
	if !pairLoop.Snapshot || pairLoop.Kind != ir.PairKeys || pairLoop.IterationType.Kind != ir.StringType {
		t.Fatalf("Pair for = %#v", pairLoop)
	}
	state := flow.Body.Statements[7].(*ir.StateStmt)
	if state.Temp == "" || !state.NoFallthrough {
		t.Fatalf("state = %#v", state)
	}
	attempt := flow.Body.Statements[8].(*ir.AttemptStmt)
	if attempt.Ultimately == nil || !attempt.FinallyAlways || len(attempt.Handlers) != 1 {
		t.Fatalf("attempt = %#v", attempt)
	}
	if ret := flow.Body.Statements[9].(*ir.ReturnStmt); ret.Value != nil || ret.ReturnType.Kind != ir.NothingType {
		t.Fatalf("return = %#v", ret)
	}
}

func TestIfWhileBreakAndPlainAssignmentRemainStructured(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `flow: Function := (active: Bool) -> Nothing {
    value: Local Int := 0
    value = 1
    if active {
        write("active")
    } else {
        write("inactive")
    }
    while active {
        break
    }
    return
}`}, "/Main.ahd")
	flow := functionIR(t, moduleIR(t, result.Compilation, "Main"), "flow")
	if _, ok := flow.Body.Statements[0].(*ir.BindingStmt); !ok {
		t.Fatalf("local binding = %T", flow.Body.Statements[0])
	}
	if assignment, ok := flow.Body.Statements[1].(*ir.AssignStmt); !ok || assignment.Target.Kind != ir.SymbolTarget {
		t.Fatalf("plain assignment = %#v", flow.Body.Statements[1])
	}
	conditional := flow.Body.Statements[2].(*ir.IfStmt)
	if len(conditional.Branches) != 1 || conditional.Else == nil {
		t.Fatalf("if/else = %#v", conditional)
	}
	loop := flow.Body.Statements[3].(*ir.WhileStmt)
	if len(loop.Body.Statements) != 1 {
		t.Fatalf("while body = %#v", loop.Body)
	}
	if _, ok := loop.Body.Statements[0].(*ir.BreakStmt); !ok {
		t.Fatalf("break = %T", loop.Body.Statements[0])
	}
}

func TestConstantVisibilityAndClassParentIdentityArePreserved(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `LIMIT: Constant Int := 5
Person: Class<> := {}
Student: Confidential Class<Person> := {}
student: Student := Student()
person: Person := student`}, "/Main.ahd")
	main := moduleIR(t, result.Compilation, "Main")
	if !globalIR(t, main, "LIMIT").Constant {
		t.Fatal("Constant flag was lost")
	}
	var person, student *ir.Class
	for _, class := range main.Classes {
		switch class.Name {
		case "Person":
			person = class
		case "Student":
			student = class
		}
	}
	if person == nil || student == nil || student.Parent != person.ID || !student.Confidential {
		t.Fatalf("Class identities/visibility = Person %#v, Student %#v", person, student)
	}
	if load, ok := globalInitializer(t, main, "person").(*ir.LoadExpr); !ok || load.Type.Class != student.ID {
		t.Fatalf("subtype initializer = %#v", globalInitializer(t, main, "person"))
	}
}

func TestClassMemberMethodCompoundTargetAndTossAreResolved(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `Box: Class<> := {
    structure: Attributes := (count: Int, values: List<Int>)
    increment: Function := () -> Nothing {
        attribute.count += 1
        attribute.values[0] = 1
    }
}
Failure: Class<Error> := {}
fail: Function := () -> Nothing {
    toss Failure(message: "x")
}`}, "/Main.ahd")
	main := moduleIR(t, result.Compilation, "Main")
	var box *ir.Class
	for _, class := range main.Classes {
		if class.Name == "Box" {
			box = class
		}
	}
	if box == nil || len(box.Fields) != 2 || len(box.Methods) != 1 {
		t.Fatalf("BoxIR = %#v", box)
	}
	method := functionIR(t, main, "increment")
	if method.Owner != box.ID || method.Receiver == "" {
		t.Fatalf("method receiver metadata = %#v", method)
	}
	compound := method.Body.Statements[0].(*ir.CompoundAssignStmt)
	if compound.Target.Kind != ir.FieldTarget {
		t.Fatalf("compound target = %#v", compound.Target)
	}
	if _, ok := compound.Target.Receiver.(*ir.LoadExpr); !ok {
		t.Fatalf("single receiver target = %#v", compound.Target.Receiver)
	}
	indexed := method.Body.Statements[1].(*ir.AssignStmt)
	if indexed.Target.Kind != ir.IndexTarget {
		t.Fatalf("index target = %#v", indexed.Target)
	}
	fail := functionIR(t, main, "fail")
	toss := fail.Body.Statements[0].(*ir.TossStmt)
	if toss.ErrorClass == "" {
		t.Fatal("toss lost resolved Error Class identity")
	}
	if _, ok := toss.Value.(*ir.ConstructExpr); !ok {
		t.Fatalf("toss value = %T", toss.Value)
	}
}

func TestDumpIsDeterministicAndFailedFrontendIsControlled(t *testing.T) {
	sources := map[string]string{"/Main.ahd": "x: Int := 2 + 3"}
	first := lowerSources(t, sources, "/Main.ahd")
	second := lowerSources(t, sources, "/Main.ahd")
	if ir.Dump(first.Compilation) != ir.Dump(second.Compilation) {
		t.Fatal("IR dump is nondeterministic")
	}

	workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": `x: Int := "bad"`})
	frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
	failed := LowerCompilation(frontend)
	if !failed.HasErrors() || failed.Compilation != nil {
		t.Fatalf("failed lowering = %#v", failed)
	}
}

func TestConstructorInitializesAttributesExplicitly(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `Point: Class<> := {
    structure: Attributes := (
        x: Int
        label: Local String
    ) {
        attribute.tag: String := label
    }
}

point: Point := Point(x: 1, label: "p")
write(point.tag)
`}, "/Main.ahd")
	current := moduleIR(t, result.Compilation, "Main")
	var constructor *ir.Function
	for _, function := range current.Functions {
		if function.Kind == ir.ConstructorFunction {
			constructor = function
		}
	}
	if constructor == nil {
		t.Fatal("expected a constructor FunctionIR")
	}
	if len(constructor.Body.Statements) < 2 {
		t.Fatalf("expected explicit attribute initialization; received %s", ir.Dump(result.Compilation))
	}
	assignment, ok := constructor.Body.Statements[0].(*ir.AssignStmt)
	if !ok || assignment.Target.Kind != ir.FieldTarget {
		t.Fatalf("expected a field assignment first; received %T", constructor.Body.Statements[0])
	}
	if !strings.HasSuffix(string(assignment.Target.Field), "::field::x") {
		t.Fatalf("the structure parameter did not initialize its attribute: %s", assignment.Target.Field)
	}
	// A Local structure parameter declares no attribute, so it must not be
	// assigned as one.
	for _, statement := range constructor.Body.Statements {
		if item, isAssign := statement.(*ir.AssignStmt); isAssign && strings.HasSuffix(string(item.Target.Field), "::field::label") {
			t.Fatal("a Local structure parameter must not become an attribute")
		}
	}
}

func TestStructureAttributeModifiersReachTypedIR(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `Example: Class<> := {
    structure: Attributes := (
        id: Constant Int
        name: String
        temporary: Local Int
        secret: Confidential Int
    )
}
`}, "/Main.ahd")
	current := moduleIR(t, result.Compilation, "Main")
	if len(current.Classes) != 1 {
		t.Fatalf("Classes = %#v", current.Classes)
	}
	fields := make(map[string]ir.Field)
	for _, field := range current.Classes[0].Fields {
		fields[field.Name] = field
	}
	if id, ok := fields["id"]; !ok || !id.Constant {
		t.Fatalf("id field = %#v, want Constant", id)
	}
	if name, ok := fields["name"]; !ok || name.Constant {
		t.Fatalf("name field = %#v, want mutable", name)
	}
	if _, exists := fields["temporary"]; exists {
		t.Fatal("Local structure parameter reached IR as a field")
	}
	if secret, ok := fields["secret"]; !ok || !secret.Confidential {
		t.Fatalf("secret field = %#v, want Confidential", secret)
	}
}

// TestModuleEffectsKeepSourceOrder checks that a module-root initializer is a
// statement at its own source position, so module-level effects are not
// reordered around it.
func TestModuleEffectsKeepSourceOrder(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `write("A")
zebra: String := "z"
write("B")
alpha: String := zebra
write("C")
`}, "/Main.ahd")
	current := moduleIR(t, result.Compilation, "Main")
	// Storage is declared for every module-root binding.
	if len(current.Globals) != 2 {
		t.Fatalf("module declares %d globals", len(current.Globals))
	}
	var order []string
	for _, statement := range current.Init.Statements {
		switch value := statement.(type) {
		case *ir.BindingStmt:
			if value.Storage != ir.ModuleStorage {
				t.Fatalf("module-root binding %s has %s storage", value.Name, value.Storage)
			}
			order = append(order, "init:"+value.Name)
		case *ir.ExprStmt:
			order = append(order, "statement")
		default:
			t.Fatalf("unexpected module statement %T", statement)
		}
	}
	expected := []string{"statement", "init:zebra", "statement", "init:alpha", "statement"}
	if strings.Join(order, ",") != strings.Join(expected, ",") {
		t.Fatalf("module effects run as %v; want %v", order, expected)
	}
}

func TestHasLowersToAResolvedMemberDesignator(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `Box: Class<> := {
    structure: Attributes := (
        value: Int
    )
}

box: Box := Box(value: 1)
value: Int := 7
write(str(box has value))
write(str(box has not missing))
`}, "/Main.ahd")
	dump := ir.Dump(result.Compilation)
	if !strings.Contains(dump, `Has(`) || !strings.Contains(dump, `string("value")`) {
		t.Fatalf("has did not lower to a member designator: %s", dump)
	}
	// A module binding of the same name must not turn the designator into a
	// value load, and has not lowers through the same designator path.
	if !strings.Contains(dump, `HasNot(`) || !strings.Contains(dump, `string("missing")`) {
		t.Fatalf("has not did not lower to a member designator: %s", dump)
	}
}

func TestSameDoesNotWidenItsOperands(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": "write(str(5 same 5.0))\n"}, "/Main.ahd")
	dump := ir.Dump(result.Compilation)
	if strings.Contains(dump, "IdentitySame(convert") {
		t.Fatalf("same widened its operands: %s", dump)
	}
}

func TestListConcatenationIsTypedAsAListOperation(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": "values: List<Int> := [1] + [2]\nwrite(len(values))\n"}, "/Main.ahd")
	dump := ir.Dump(result.Compilation)
	if !strings.Contains(dump, "ListConcat(") {
		t.Fatalf("List concatenation was not typed as a List operation: %s", dump)
	}
}

func TestCollectionMutationsLowerToBuiltinOperations(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `values: List<Int> := []
scores: Pair<String, Int> := {}
values.add(1)
values.eject(0)
scores.eject("a")
`}, "/Main.ahd")
	dump := ir.Dump(result.Compilation)
	for _, expected := range []string{"builtin:core::List.add", "builtin:core::List.eject", "builtin:core::Pair.eject"} {
		if !strings.Contains(dump, expected) {
			t.Fatalf("collection mutation did not lower to %s:\n%s", expected, dump)
		}
	}
}

func TestEmptyCollectionLiteralsLowerWithTheirDeclaredType(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `values: List<Int> := []
scores: Pair<String, Int> := {}
rows: List<List<Int>> := [
    []
]
`}, "/Main.ahd")
	main := moduleIR(t, result.Compilation, "Main")
	if got := globalIR(t, main, "values").Type.String(); got != "List<Int>" {
		t.Fatalf("empty List global type = %s", got)
	}
	if got := globalIR(t, main, "scores").Type.String(); got != "Pair<String, Int>" {
		t.Fatalf("empty Pair global type = %s", got)
	}
	dump := ir.Dump(result.Compilation)
	if strings.Contains(dump, "Invalid") {
		t.Fatalf("a contextually typed empty literal left an Invalid type in the IR:\n%s", dump)
	}
}

func TestStringAndListOperationsLowerToBuiltinIdentities(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `double: Function := (
    x: Int
) -> Int {
    return x * 2
}

isEven: Function := (
    x: Int
) -> Bool {
    return x % 2 == 0
}

text: String := "a,b"
values: List<Int> := [3, 1]

write(text.trim())
write(text.lower())
write(text.upper())
write(text.capitalize())
write(text.split(","))
write(text.replace("a", "b"))
write(text.contains("a"))
write(text.startsWith("a"))
write(text.endsWith("b"))
write(text.count("a"))
write(text.index("a"))
values.sort()
values.sort(double)
values.reverse()
values.shuffle()
write(values.count(1))
write(values.index(1))
write(values.map(double))
write(values.filter(isEven))
`}, "/Main.ahd")
	dump := ir.Dump(result.Compilation)
	for _, expected := range []string{
		"builtin:core::String.trim", "builtin:core::String.lower", "builtin:core::String.upper",
		"builtin:core::String.capitalize", "builtin:core::String.split", "builtin:core::String.replace",
		"builtin:core::String.contains", "builtin:core::String.startsWith", "builtin:core::String.endsWith",
		"builtin:core::String.count", "builtin:core::String.index",
		"builtin:core::List.sort", "builtin:core::List.reverse", "builtin:core::List.shuffle", "builtin:core::List.count",
		"builtin:core::List.index", "builtin:core::List.map", "builtin:core::List.filter",
	} {
		if !strings.Contains(dump, expected) {
			t.Fatalf("type operation did not lower to %s:\n%s", expected, dump)
		}
	}
}

func TestUserClassMethodsShadowNoBuiltinOperation(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `Report: Class<> := {
    structure: Attributes := (
        title: String
    )

    count: Function := (
    ) -> Int {
        return 7
    }
}

report: Report := Report("t")
write(report.count())
`}, "/Main.ahd")
	dump := ir.Dump(result.Compilation)
	if strings.Contains(dump, "builtin:core::List.count") || strings.Contains(dump, "builtin:core::String.count") {
		t.Fatalf("a Class method lowered to a built-in type operation:\n%s", dump)
	}
}

func TestTimeModuleLowersItsClassDeclarations(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": `bring Time
from Time bring DateTime
from Time bring Duration

value: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)
wait: Duration := Time.duration(milliseconds: 5)

write(value.year)
write(value.before(value))
write(wait.seconds)
write(Time.Calendar.isLeapYear(2028))
`}, "/Main.ahd")
	dump := ir.Dump(result.Compilation)
	for _, expected := range []string{
		"builtin:Time::class::DateTime", "builtin:Time::class::Duration",
		"builtin:Time::class::Calendar", "builtin:Time::dateTime",
		"builtin:Time::duration", "builtin:core::DateTime.before",
		"builtin:core::Calendar.isLeapYear",
	} {
		if !strings.Contains(dump, expected) {
			t.Fatalf("Time did not lower to %s:\n%s", expected, dump)
		}
	}
	if strings.Contains(dump, "Invalid") {
		t.Fatalf("a Time declaration left an Invalid type in the IR:\n%s", dump)
	}
}

// TestTimeClassesPublishTheirBuiltInMembers keeps member existence truthful for
// the members reached through built-in type operations.
func TestTimeClassesPublishTheirBuiltInMembers(t *testing.T) {
	result := lowerSources(t, map[string]string{"/Main.ahd": "bring Time\n\nwrite(Time.Calendar.isLeapYear(2028))\n"}, "/Main.ahd")
	for _, module := range result.Compilation.Modules {
		if module == nil || string(module.ID) != "builtin:Time" {
			continue
		}
		found := map[string][]string{}
		for _, class := range module.Classes {
			found[class.Name] = class.Operations
		}
		if len(found["DateTime"]) != 4 || len(found["Calendar"]) != 3 {
			t.Fatalf("published operations = %v", found)
		}
		if len(found["Duration"]) != 0 {
			t.Fatalf("Duration reaches its members as fields, not operations: %v", found["Duration"])
		}
		return
	}
	t.Fatal("the Time module was not lowered")
}
