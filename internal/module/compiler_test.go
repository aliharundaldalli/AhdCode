package module

import (
	"os"
	"path/filepath"
	"testing"

	"ahdcode/internal/semantic"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

func compileMemory(t *testing.T, sources map[string]string, entry string) (*InMemoryWorkspace, CompilationResult) {
	t.Helper()
	workspace := NewInMemoryWorkspace(sources)
	result := NewCompiler(workspace, workspace).Compile(entry)
	return workspace, result
}

func requireClean(t *testing.T, result CompilationResult) {
	t.Helper()
	if result.HasErrors() {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
}

func requireCode(t *testing.T, result CompilationResult, code string) ModuleDiagnostic {
	t.Helper()
	for _, item := range result.Diagnostics {
		if item.Diagnostic.Code == code {
			return item
		}
	}
	t.Fatalf("expected %s, got %+v", code, result.Diagnostics)
	return ModuleDiagnostic{}
}

func moduleNamed(t *testing.T, result CompilationResult, name string) *Module {
	t.Helper()
	for _, module := range result.Modules {
		if module.Source.Name == name {
			return module
		}
	}
	t.Fatalf("module %s not found in result", name)
	return nil
}

func TestNamespaceDirectSelectiveAndAllImports(t *testing.T) {
	mathematics := `sqrt: Function := (value: Real) -> Real {
    return value
}
sin: Function := (value: Real) -> Real {
    return value
}
secret: Confidential Int := 7`
	tests := []struct {
		name string
		main string
	}{
		{"namespace", "bring Mathematics\nanswer: Real := Mathematics.sqrt(25)"},
		{"direct", "from Mathematics bring sqrt\nanswer: Real := sqrt(25)"},
		{"selective", "from Mathematics bring (\n    sqrt\n    sin\n)\na: Real := sqrt(25)\nb: Real := sin(0)"},
		{"all public only", "from Mathematics bring all\na: Real := sqrt(25)\nb: Real := sin(0)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := compileMemory(t, map[string]string{"/Main.ahd": test.main, "/Mathematics.ahd": mathematics}, "/Main.ahd")
			requireClean(t, result)
		})
	}
}

func TestModuleNamespaceAliasesUseTheGenericImportPath(t *testing.T) {
	t.Run("user module", func(t *testing.T) {
		_, result := compileMemory(t, map[string]string{
			"/Main.ahd":   "bring Engine as E\nanswer: Int := E.tick()",
			"/Engine.ahd": "tick: Function := () -> Int { return 1 }",
		}, "/Main.ahd")
		requireClean(t, result)
		main := moduleNamed(t, result, "Main")
		declaration := main.Parsed.Program.Statements[1].(*ast.VariableDecl)
		member := declaration.Initializer.(*ast.CallExpr).Callee.(*ast.MemberExpr)
		namespace := member.Object.(*ast.IdentifierExpr)
		resolved := main.Semantic.ResolvedSymbols[namespace]
		if resolved == nil || resolved.Kind != semantic.NamespaceSymbol || resolved.Name != "E" || resolved.Namespace.Name != "Engine" {
			t.Fatalf("resolved user alias = %#v", resolved)
		}
	})

	for _, test := range []struct {
		name   string
		source string
	}{
		{"Math", "bring Math as M\nanswer: Real := M.sqrt(25)"},
		{"Time", "bring Time as T\nanswer: Bool := T.Calendar.isLeapYear(2028)"},
		{"Latex", `bring Latex as L
answer: String := L.escape("a&b")`},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, result := compileMemory(t, map[string]string{"/Main.ahd": test.source}, "/Main.ahd")
			requireClean(t, result)
		})
	}
}

func TestAliasOnlyImportDoesNotBindOriginalModuleName(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd":   "bring Engine as E\nanswer: Int := Engine.tick()",
		"/Engine.ahd": "tick: Function := () -> Int { return 1 }",
	}, "/Main.ahd")
	requireCode(t, result, "SEM001")

	_, both := compileMemory(t, map[string]string{
		"/Main.ahd":   "bring Engine as E\nbring Engine\na: Int := E.tick()\nb: Int := Engine.tick()",
		"/Engine.ahd": "tick: Function := () -> Int { return 1 }",
	}, "/Main.ahd")
	requireClean(t, both)
}

func TestModuleAliasConflictsUseOrdinaryImportDiagnostics(t *testing.T) {
	for name, sources := range map[string]map[string]string{
		"two modules": {
			"/Main.ahd": "bring Time as T\nbring Math as T",
		},
		"local declaration": {
			"/Main.ahd":   "E: Int := 1\nbring Engine as E",
			"/Engine.ahd": "tick: Function := () -> Int { return 1 }",
		},
		"duplicate alias": {
			"/Main.ahd": "bring Time as T\nbring Time as T",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, result := compileMemory(t, sources, "/Main.ahd")
			requireCode(t, result, semantic.CodeImportCollision)
		})
	}
}

func TestUnknownModuleWithAliasKeepsModuleDiagnostic(t *testing.T) {
	_, result := compileMemory(t, map[string]string{"/Main.ahd": "bring Missing as M"}, "/Main.ahd")
	diagnostic := requireCode(t, result, semantic.CodeModuleNotFound)
	if diagnostic.Diagnostic.Span.FileID == 0 {
		t.Fatal("missing aliased module diagnostic has no source span")
	}
}

func TestNamespaceAndMemberResolutionAreRecordedInSideTables(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd":        "bring Mathematics\nanswer: Real := Mathematics.sqrt(25)",
		"/Mathematics.ahd": "sqrt: Function := (value: Real) -> Real { return value }",
	}, "/Main.ahd")
	requireClean(t, result)
	main := moduleNamed(t, result, "Main")
	declaration := main.Parsed.Program.Statements[1].(*ast.VariableDecl)
	call := declaration.Initializer.(*ast.CallExpr)
	member := call.Callee.(*ast.MemberExpr)
	namespace := member.Object.(*ast.IdentifierExpr)
	if main.Semantic.ResolvedSymbols[namespace].Kind != semantic.NamespaceSymbol {
		t.Fatal("namespace identifier was not resolved as NamespaceSymbol")
	}
	if main.Semantic.ResolvedSymbols[member] == nil || main.Semantic.SelectedCallables[call] == nil {
		t.Fatal("namespace member/callable resolution was not recorded")
	}
}

func TestMissingAndConfidentialDiagnosticsAreDistinct(t *testing.T) {
	_, missingModule := compileMemory(t, map[string]string{"/Main.ahd": "bring Missing"}, "/Main.ahd")
	requireCode(t, missingModule, semantic.CodeModuleNotFound)

	_, missingExport := compileMemory(t, map[string]string{
		"/Main.ahd": "from Other bring absent", "/Other.ahd": "present: Int := 1",
	}, "/Main.ahd")
	missing := requireCode(t, missingExport, semantic.CodeExportNotFound)
	if missing.RequestedSymbol != "absent" || missing.TargetModule == "" {
		t.Fatalf("missing export context = %+v", missing)
	}

	for _, main := range []string{
		"from Secrets bring secret",
		"bring Secrets\nwrite(Secrets.secret)",
	} {
		_, result := compileMemory(t, map[string]string{
			"/Main.ahd": main, "/Secrets.ahd": `secret: Confidential String := "hidden"`,
		}, "/Main.ahd")
		requireCode(t, result, semantic.CodeConfidentialAccess)
	}

	_, missingMember := compileMemory(t, map[string]string{
		"/Main.ahd": "bring Other\nwrite(Other.absent)", "/Other.ahd": "present: Int := 1",
	}, "/Main.ahd")
	requireCode(t, missingMember, semantic.CodeNamespaceMember)
}

func TestImportCollisionsNeverOverwrite(t *testing.T) {
	_, local := compileMemory(t, map[string]string{
		"/Main.ahd": "x: Int := 5\nfrom Other bring x", "/Other.ahd": "x: Int := 1",
	}, "/Main.ahd")
	requireCode(t, local, semantic.CodeImportCollision)

	_, twoImports := compileMemory(t, map[string]string{
		"/Main.ahd": "from A bring x\nfrom B bring x",
		"/A.ahd":    "x: Int := 1",
		"/B.ahd":    "x: Int := 2",
	}, "/Main.ahd")
	requireCode(t, twoImports, semantic.CodeImportCollision)

	_, namespace := compileMemory(t, map[string]string{
		"/Main.ahd": "A: Int := 1\nbring A", "/A.ahd": "x: Int := 1",
	}, "/Main.ahd")
	requireCode(t, namespace, semantic.CodeImportCollision)
}

func TestCircularDependenciesReportStructuredChain(t *testing.T) {
	for _, sources := range []map[string]string{
		{"/A.ahd": "bring B", "/B.ahd": "bring A"},
		{"/A.ahd": "bring B", "/B.ahd": "bring C", "/C.ahd": "bring A"},
	} {
		_, result := compileMemory(t, sources, "/A.ahd")
		diagnostic := requireCode(t, result, semantic.CodeCircularDependency)
		if len(diagnostic.Cycle) < 3 || diagnostic.Cycle[0] != diagnostic.Cycle[len(diagnostic.Cycle)-1] {
			t.Fatalf("invalid cycle chain: %v", diagnostic.Cycle)
		}
	}
}

func TestDiamondDependencyAnalyzedOnce(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/A.ahd": "bring B\nbring C",
		"/B.ahd": "bring D",
		"/C.ahd": "bring D",
		"/D.ahd": "value: Int := 1",
	}, "/A.ahd")
	requireClean(t, result)
	d := moduleNamed(t, result, "D")
	if d.AnalyzeCount != 1 || workspace.LoadCount(d.ID) != 1 {
		t.Fatalf("D analyze/load = %d/%d, want 1/1", d.AnalyzeCount, workspace.LoadCount(d.ID))
	}
}

func TestImportedFunctionOverloadAndCallbackUseSameResolver(t *testing.T) {
	library := `calculate: Function := (x: Int) -> Int {
    return x
}
calculate: Overload Function := (x: Real) -> Real {
    return x
}`
	_, result := compileMemory(t, map[string]string{
		"/Library.ahd": library,
		"/Main.ahd": `from Library bring calculate
use: Function := (operation: Function, x: Int) -> Int {
    return operation(x)
}
exact: Int := calculate(5)
callback: Int := use(calculate, 5)`,
	}, "/Main.ahd")
	requireClean(t, result)
	main := moduleNamed(t, result, "Main")
	exactDecl := main.Parsed.Program.Statements[2].(*ast.VariableDecl)
	exactCall := exactDecl.Initializer.(*ast.CallExpr)
	selected := main.Semantic.SelectedCallables[exactCall]
	if selected == nil || selected.Signature.Parameters[0].Type.Kind() != types.IntKind {
		t.Fatalf("selected imported overload = %#v", selected)
	}
	callbackDecl := main.Parsed.Program.Statements[3].(*ast.VariableDecl)
	callbackCall := callbackDecl.Initializer.(*ast.CallExpr)
	if main.Semantic.SelectedFunctionValues[callbackCall.Arguments[0].Value] == nil {
		t.Fatal("imported overloaded callback was not context-selected")
	}
}

func TestCrossModuleCallbackMetadataIgnoresParameterSpelling(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Numbers.ahd": `triple: Function := (
    value: Int
) -> Int {
    return value * 3
}`,
		"/Engine.ahd": `applyTwice: Function := (
    operation: Function
    value: Int
) -> Int {
    first: Local Int := operation(value)
    return operation(first)
}`,
		"/Main.ahd": `bring Engine
from Numbers bring triple
operation: Function := triple
first: Int := Engine.applyTwice(operation, 2)
second: Int := Engine.applyTwice(triple, 3)`,
	}, "/Main.ahd")
	requireClean(t, result)

	triple := moduleNamed(t, result, "Numbers").Interface.Exports["triple"].Callable
	applyTwice := moduleNamed(t, result, "Engine").Interface.Exports["applyTwice"].Callable
	callback, ok := applyTwice.Signature.Parameters[0].Type.(types.Function)
	if triple == nil || !ok || callback.Signature == nil {
		t.Fatalf("cross-module callback metadata is incomplete: triple=%#v callback=%#v", triple, callback)
	}
	if triple.Signature.Parameters[0].Name != "value" || callback.Signature.Parameters[0].Name != "" {
		t.Fatalf("test did not retain distinct declaration metadata: triple=%#v callback=%#v", triple.Signature, callback.Signature)
	}
	if !types.Equal(types.Function{Signature: triple.Signature}, callback) {
		t.Fatal("equivalent cross-module Function types conflict on parameter spelling")
	}
}

func TestCrossModuleClassIdentitySubtypeAndMembers(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Models.ahd": `Person: Class<> := {
    structure: Attributes := (name: String)
    describe: Function := () -> String {
        return "person"
    }
}`,
		"/Main.ahd": `from Models bring Person
Student: Class<Person> := {
}
person: Person := Student(name: "Ada")
name: String := Person(name: "Ali").name`,
	}, "/Main.ahd")
	requireClean(t, result)
	main := moduleNamed(t, result, "Main")
	student := main.Interface.Symbols["Student"]
	person := moduleNamed(t, result, "Models").Interface.Symbols["Person"]
	if student == nil || person == nil || !types.SameClassIdentity(student.Class.Parent, person.Class) {
		t.Fatalf("cross-module parent identity was not preserved")
	}
	studentDecl := main.Parsed.Program.Statements[1].(*ast.ClassDecl)
	if main.Semantic.ResolvedSymbols[studentDecl.Parent] != person {
		t.Fatal("imported parent TypeRef was not recorded in semantic side tables")
	}
	if person.Constructor == nil || len(person.Constructor.Signature.Parameters) != 1 || person.Members["name"] == nil {
		t.Fatalf("constructor/member interface metadata is incomplete: %#v", person)
	}
}

func TestSameClassNameInDifferentModulesHasDifferentIdentity(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Root.ahd": "bring A\nbring B",
		"/A.ahd":    "Person: Class<> := {}",
		"/B.ahd":    "Person: Class<> := {}",
	}, "/Root.ahd")
	requireClean(t, result)
	a := moduleNamed(t, result, "A").Interface.Symbols["Person"].Class
	b := moduleNamed(t, result, "B").Interface.Symbols["Person"].Class
	if types.SameClassIdentity(a, b) {
		t.Fatal("same-named Classes from different modules share identity")
	}
}

func TestCrossModuleNullableReturnMetadataIsPreserved(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Students.ahd": `Student: Class<> := {
    structure: Attributes := (name: String)
}
findStudent: Function := (id: Int) -> Student {
    return null
}`,
		"/Main.ahd": `from Students bring (
    Student
    findStudent
)
student: Student := findStudent(5)
write(student.name)`,
	}, "/Main.ahd")
	requireCode(t, result, "SEM011")
	callable := moduleNamed(t, result, "Students").Interface.Symbols["findStudent"].Callable
	if callable == nil || callable.ReturnNull != semantic.Null {
		t.Fatalf("return null metadata = %#v", callable)
	}
}

func TestCrossModuleBindingNullMetadataIsPreserved(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Values.ahd": "maybe: String := null",
		"/Main.ahd":   "from Values bring maybe\nwrite(maybe + \"x\")",
	}, "/Main.ahd")
	requireCode(t, result, "SEM011")
	if moduleNamed(t, result, "Values").Interface.Symbols["maybe"].InitialNull != semantic.Null {
		t.Fatal("binding null-state was not exported")
	}
}

func TestDependencyFailuresPropagateWithoutFakeInterface(t *testing.T) {
	_, semanticFailure := compileMemory(t, map[string]string{
		"/Main.ahd": "bring Broken", "/Broken.ahd": `value: Int := "wrong"`,
	}, "/Main.ahd")
	requireCode(t, semanticFailure, semantic.CodeFailedDependency)
	if moduleNamed(t, semanticFailure, "Broken").Interface != nil {
		t.Fatal("failed dependency exposed a successful interface")
	}

	_, malformed := compileMemory(t, map[string]string{
		"/Main.ahd": "bring Broken", "/Broken.ahd": "if {",
	}, "/Main.ahd")
	if !malformed.HasErrors() || moduleNamed(t, malformed, "Broken").State != Failed {
		t.Fatalf("malformed dependency did not fail safely: %+v", malformed.Diagnostics)
	}
}

func TestBuiltinRegistryUsesSameModuleInterface(t *testing.T) {
	workspace := NewInMemoryWorkspace(map[string]string{"/Main.ahd": "from Fundamentals bring answer\nvalue: Int := answer"})
	compiler := NewCompiler(workspace, workspace)
	compiler.Builtins["Fundamentals"] = &semantic.ModuleInterface{
		ModuleID: "builtin:Fundamentals", Name: "Fundamentals",
		Symbols: map[string]*semantic.Symbol{"answer": {Name: "answer", Kind: semantic.BindingSymbol, Type: types.Int, Constant: true, InitialNull: semantic.NonNull, OriginModuleID: "builtin:Fundamentals"}},
		Exports: map[string]*semantic.Symbol{}, ExportNames: []string{"answer"}, Classes: map[string]*semantic.Symbol{},
	}
	compiler.Builtins["Fundamentals"].Exports["answer"] = compiler.Builtins["Fundamentals"].Symbols["answer"]
	result := compiler.Compile("/Main.ahd")
	requireClean(t, result)
	if moduleNamed(t, result, "Fundamentals").Source.Builtin != true {
		t.Fatal("built-in module did not use module interface path")
	}
}

func TestImportedConstantMetadataRemainsCompileTimeUsable(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Constants.ahd": "power: Constant Int := 3",
		"/Main.ahd":      "from Constants bring power\nvalue: Int := 2 ^ power",
	}, "/Main.ahd")
	requireClean(t, result)
}

func TestCrossModuleConfidentialMemberVisibility(t *testing.T) {
	models := `Secret: Class<> := {
    structure: Attributes := (code: Confidential String)
}`
	_, external := compileMemory(t, map[string]string{
		"/Models.ahd": models,
		"/Main.ahd":   "from Models bring Secret\nsecret: Secret := Secret(code: \"x\")\nwrite(secret.code)",
	}, "/Main.ahd")
	requireCode(t, external, semantic.CodeConfidentialAccess)

	_, subclass := compileMemory(t, map[string]string{
		"/Models.ahd": models,
		"/Main.ahd": `from Models bring Secret
Vault: Class<Secret> := {
    reveal: Function := () -> String {
        return attribute.code
    }
}`,
	}, "/Main.ahd")
	requireClean(t, subclass)
}

func TestCrossModuleStructureAttributeModifiersArePreserved(t *testing.T) {
	models := `Example: Class<> := {
    structure: Attributes := (
        id: Constant Int
        temporary: Local Int
    )
}`
	_, constant := compileMemory(t, map[string]string{
		"/Models.ahd": models,
		"/Main.ahd":   "from Models bring Example\nexample: Example := Example(id: 1, temporary: 9)\nexample.id = 2",
	}, "/Main.ahd")
	requireCode(t, constant, "SEM009")

	_, local := compileMemory(t, map[string]string{
		"/Models.ahd": models,
		"/Main.ahd":   "from Models bring Example\nexample: Example := Example(id: 1, temporary: 9)\nwrite(example.temporary)",
	}, "/Main.ahd")
	requireCode(t, local, "SEM019")
}

func TestFileResolverCanonicalizesAliases(t *testing.T) {
	directory := t.TempDir()
	realPath := filepath.Join(directory, "Main.ahd")
	if err := os.WriteFile(realPath, []byte("value: Int := 1"), 0o600); err != nil {
		t.Fatal(err)
	}
	aliasPath := filepath.Join(directory, "Alias.ahd")
	if err := os.Symlink(realPath, aliasPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	resolver := FileResolver{}
	realIdentity, err := resolver.CanonicalEntry(realPath)
	if err != nil {
		t.Fatal(err)
	}
	aliasIdentity, err := resolver.CanonicalEntry(aliasPath)
	if err != nil {
		t.Fatal(err)
	}
	if realIdentity.ID != aliasIdentity.ID {
		t.Fatalf("canonical IDs differ: %s and %s", realIdentity.ID, aliasIdentity.ID)
	}
}

func TestFileResolverEnforcesCaseSensitiveModuleNames(t *testing.T) {
	directory := t.TempDir()
	mainPath := filepath.Join(directory, "Main.ahd")
	otherPath := filepath.Join(directory, "Mathematics.ahd")
	for filePath, text := range map[string]string{mainPath: "bring mathematics", otherPath: "value: Int := 1"} {
		if err := os.WriteFile(filePath, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result := NewCompiler(FileResolver{}, FileLoader{}).Compile(mainPath)
	requireCode(t, result, semantic.CodeModuleNotFound)
}

func TestProductionFileResolverAndLoaderCompileSibling(t *testing.T) {
	directory := t.TempDir()
	mainPath := filepath.Join(directory, "Main.ahd")
	otherPath := filepath.Join(directory, "Other.ahd")
	for filePath, text := range map[string]string{mainPath: "from Other bring value\ncopy: Int := value", otherPath: "value: Int := 1"} {
		if err := os.WriteFile(filePath, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result := NewCompiler(FileResolver{}, FileLoader{}).Compile(mainPath)
	requireClean(t, result)
}

func TestPhysicalModuleAliasIsAnalyzedOnce(t *testing.T) {
	directory := t.TempDir()
	mainPath := filepath.Join(directory, "Main.ahd")
	libraryPath := filepath.Join(directory, "Library.ahd")
	aliasPath := filepath.Join(directory, "Alias.ahd")
	for filePath, text := range map[string]string{
		mainPath:    "bring Library\nbring Alias\na: Int := Library.value\nb: Int := Alias.value",
		libraryPath: "value: Int := 1",
	} {
		if err := os.WriteFile(filePath, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(libraryPath, aliasPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	result := NewCompiler(FileResolver{}, FileLoader{}).Compile(mainPath)
	requireClean(t, result)
	library := moduleNamed(t, result, "Library")
	if library.AnalyzeCount != 1 || len(result.Modules) != 2 {
		t.Fatalf("alias produced duplicate analysis: count=%d modules=%d", library.AnalyzeCount, len(result.Modules))
	}
}
