package golang

import (
	"go/format"
	"strings"
	"testing"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/ir"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
)

func lower(t *testing.T, sources map[string]string) *ir.Compilation {
	t.Helper()
	workspace := module.NewInMemoryWorkspace(sources)
	frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
	if frontend.HasErrors() {
		t.Fatalf("frontend diagnostics: %+v", frontend.Diagnostics)
	}
	result := lowering.LowerCompilation(frontend)
	if result.HasErrors() {
		t.Fatalf("lowering diagnostics: %+v", result.Diagnostics)
	}
	return result.Compilation
}

func generate(t *testing.T, source string) *GeneratedProgram {
	t.Helper()
	program, produced := Generate(lower(t, map[string]string{"/Main.ahd": source}))
	for _, item := range produced {
		if item.Severity == diagnostics.SeverityError {
			t.Fatalf("backend diagnostics: %+v", produced)
		}
	}
	if program == nil {
		t.Fatal("expected a generated program")
	}
	return program
}

func programSource(t *testing.T, program *GeneratedProgram) string {
	t.Helper()
	for _, file := range program.Files {
		if file.Name == programFileName {
			return file.Content
		}
	}
	t.Fatal("generated program file is missing")
	return ""
}

func codes(items []diagnostics.Diagnostic) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}

func hasCode(items []diagnostics.Diagnostic, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}

func TestGenerationIsDeterministic(t *testing.T) {
	compilation := lower(t, map[string]string{"/Main.ahd": `Point: Class<> := {
    structure: Attributes := (
        x: Int
        y: Int
    )

    total: Function := (
    ) -> Int {
        return attribute.x + attribute.y
    }
}

scores: Pair<String, Int> := {
    "b": 2
    "a": 1
}

point: Point := Point(x: 1, y: 2)
write(point.total())
write(len(scores))
`})
	first, firstDiagnostics := Generate(compilation)
	second, secondDiagnostics := Generate(compilation)
	if len(firstDiagnostics) != 0 || len(secondDiagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v %v", codes(firstDiagnostics), codes(secondDiagnostics))
	}
	if len(first.Files) != len(second.Files) {
		t.Fatalf("file count differs: %d and %d", len(first.Files), len(second.Files))
	}
	for index := range first.Files {
		if first.Files[index].Name != second.Files[index].Name || first.Files[index].Content != second.Files[index].Content {
			t.Fatalf("generation is not deterministic for %s", first.Files[index].Name)
		}
	}
}

func TestGeneratedSourceIsGofmtStable(t *testing.T) {
	program := generate(t, "write(str([1, 2, 3]))\n")
	for _, file := range program.Files {
		formatted, err := format.Source([]byte(file.Content))
		if err != nil {
			t.Fatalf("%s is not valid Go: %v", file.Name, err)
		}
		if string(formatted) != file.Content {
			t.Fatalf("%s is not gofmt-stable", file.Name)
		}
	}
}

func TestLatexGenerationMarksOnlyLatexPrograms(t *testing.T) {
	ordinary := generate(t, `write("hello")`)
	if ordinary.RequiresLatex {
		t.Fatal("an ordinary program must not acquire a Latex runtime dependency")
	}
	latex := generate(t, `bring Latex as L
write(L.escape("A&B"))
`)
	if !latex.RequiresLatex {
		t.Fatal("a Latex call must record the shared runtime dependency")
	}
	if !strings.Contains(programSource(t, latex), "AhdLatexEscape") {
		t.Fatal("Latex.escape did not lower to the runtime helper")
	}
}

func TestWordGenerationUsesRuntimeHelpersWithoutExternalRequirement(t *testing.T) {
	program := generate(t, `bring Word
from Word bring Document

document: Document := Word.new()
document = document.heading("Report", 1)
document = document.paragraph("Summary", "center", true, false, false)
document.save("report.docx")
`)
	generated := programSource(t, program)
	for _, helper := range []string{"AhdWordNew", "AhdWordHeading", "AhdWordParagraph", "AhdWordSave"} {
		if !strings.Contains(generated, helper) {
			t.Fatalf("Word generation omitted %s:\n%s", helper, generated)
		}
	}
	if program.RequiresLatex || program.RequiresPlot || program.RequiresNumeric {
		t.Fatalf("Word acquired an unrelated external runtime requirement: %+v", program)
	}
}

func TestGeneratedProgramCarriesRuntime(t *testing.T) {
	program := generate(t, "write(\"hi\")\n")
	names := make([]string, 0, len(program.Files))
	for _, file := range program.Files {
		names = append(names, file.Name)
	}
	if strings.Join(names, ",") != programFileName+","+runtimeFileName+","+excelRuntimeFileName+","+pdfRuntimeFileName+","+archiveRuntimeFileName {
		t.Fatalf("unexpected generated files %v", names)
	}
	for _, file := range program.Files {
		if !strings.HasPrefix(file.Content, "// Code generated") && !strings.Contains(file.Content, "package main") {
			t.Fatalf("%s does not declare package main", file.Name)
		}
	}
	if strings.Contains(programSource(t, program), "package ahdruntime") {
		t.Fatal("the runtime package clause was not rewritten")
	}
}

func TestGoKeywordAndUnicodeIdentifiersStayValid(t *testing.T) {
	program := generate(t, `range: Int := 1
func: Int := 2
kaçıncı: Int := 3
select: Function := (
    type: Int
) -> Int {
    return type + 1
}

write(select(4) + range + func + kaçıncı)
`)
	generated := programSource(t, program)
	for _, reserved := range []string{" range ", " func(", " select(", " type "} {
		if strings.Contains(generated, "var "+reserved) {
			t.Fatalf("a raw AhdCode name leaked into generated Go: %q", reserved)
		}
	}
	if strings.Contains(generated, "kaçıncı") {
		t.Fatal("a non-ASCII identifier leaked into generated Go")
	}
}

func TestNegatedIntMinimumIsRepresentable(t *testing.T) {
	program := generate(t, "minimum: Int := -9223372036854775808\nwrite(minimum)\n")
	if !strings.Contains(programSource(t, program), "int64(-9223372036854775808)") {
		t.Fatal("the signed 64-bit minimum was not folded into a Go constant")
	}
}

func TestLeadingZeroIntLiteralStaysDecimal(t *testing.T) {
	program := generate(t, "value: Int := 0012\nwrite(value)\n")
	generated := programSource(t, program)
	if strings.Contains(generated, "int64(0012)") || !strings.Contains(generated, "int64(12)") {
		t.Fatal("a leading-zero Int literal was not normalized to decimal")
	}
}

func TestUntilKeepsPostCheckAndConditionOnContinue(t *testing.T) {
	program := generate(t, `i: Int := 0
until i >= 2 {
    i++
    continue
}

write(i)
`)
	generated := programSource(t, program)
	if !strings.Contains(generated, "for {") || !strings.Contains(generated, "= true") {
		t.Fatal("until was not lowered to a post-check loop")
	}
	if strings.Contains(generated, "for AhdConstBool") {
		t.Fatal("until was lowered to a pre-check loop")
	}
}

func TestListElementsUseTheNullableRepresentation(t *testing.T) {
	program := generate(t, "values: List<Int?> := [1, null, 3]\nwrite(len(values))\n")
	if !strings.Contains(programSource(t, program), "*AhdList[*int64]") {
		t.Fatal("List elements were not boxed")
	}
}

// TestNonNullableListElementsAreUnboxed locks in v0.1.7's stricter default:
// a List<Int> (no ?) stores unboxed int64 elements, since the type itself
// now statically forbids a null element.
func TestNonNullableListElementsAreUnboxed(t *testing.T) {
	program := generate(t, "values: List<Int> := [1, 2, 3]\nwrite(len(values))\n")
	source := programSource(t, program)
	if !strings.Contains(source, "*AhdList[int64]") {
		t.Fatal("non-nullable List elements were unexpectedly boxed")
	}
	if strings.Contains(source, "*AhdList[*int64]") {
		t.Fatal("non-nullable List elements were boxed")
	}
}

func TestClearKeepsListIdentity(t *testing.T) {
	program := generate(t, "a: List<Int> := [1]\nb: List<Int> := a\nclear(a)\nwrite(len(b))\n")
	generated := programSource(t, program)
	if !strings.Contains(generated, ".Clear()") {
		t.Fatal("clear was not lowered to an in-place runtime call")
	}
	if strings.Contains(generated, "AhdNewList[*int64]()") {
		t.Fatal("clear rebound a fresh List instead of emptying in place")
	}
}

func TestUnsupportedNodesProduceBackendDiagnostics(t *testing.T) {
	cases := map[string]string{
		// A Function value reaching str without a declared name has no
		// canonical text, so it is reported rather than approximated.
		"anonymous Function text": `describe: Function := (
    action: Function
    value: Int
) -> String {
    result: Local Int := action(value)
    return "{str(action)}={result}"
}

double: Function := (
    n: Int
) -> Int {
    return n * 2
}

write(describe(double, 2))
`,
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			program, produced := Generate(lower(t, map[string]string{"/Main.ahd": text}))
			if program != nil {
				t.Fatal("expected no generated program for an unsupported node")
			}
			if !hasCode(produced, CodeUnsupportedNode) {
				t.Fatalf("expected %s; received %v", CodeUnsupportedNode, codes(produced))
			}
		})
	}
}

func TestErrorHandlingAndInheritanceAreGenerated(t *testing.T) {
	program := generate(t, `Failure: Class<Error> := {
    structure: Attributes := (
        SuperClass.attributes
    )
}

Animal: Class<> := {
    structure: Attributes := (
        name: String
    )

    speak: Function := (
    ) -> String {
        return "..."
    }
}

Dog: Class<Animal> := {
    structure: Attributes := (
        SuperClass.attributes
    )

    speak: Override Function := (
    ) -> String {
        return "{SuperClass.speak()} woof"
    }
}

attempt {
    toss Failure(message: "boom")
}
except Error as error {
    write(error.message)
}
ultimately {
    write(Dog(name: "Rex").speak())
}
`)
	generated := programSource(t, program)
	for _, expected := range []string{"AhdSignalOf(recover())", "AhdMatches(signal,", "AhdToss(", "defer func()"} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("error handling did not generate %q", expected)
		}
	}
	if !strings.Contains(generated, "AhdRegisterError(") {
		t.Fatal("the built-in Error catalog was not installed")
	}
	if !strings.Contains(generated, "AhdInstance") {
		t.Fatal("Class instances do not carry runtime identity")
	}
}

func TestNilCompilationIsReportedNotPanicked(t *testing.T) {
	program, produced := Generate(nil)
	if program != nil || !hasCode(produced, CodeGenerationFailure) {
		t.Fatalf("expected %s; received %v", CodeGenerationFailure, codes(produced))
	}
}

func TestMalformedIRIsReportedNotPanicked(t *testing.T) {
	compilation := &ir.Compilation{
		Entry: "mem:/Main.ahd",
		Modules: []*ir.Module{{
			ID: "mem:/Main.ahd", Name: "Main",
			Init: ir.Block{Statements: []ir.Statement{
				&ir.ExprStmt{Value: &ir.LoadExpr{ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.IntType}, NullState: ir.NonNull}, Symbol: "missing"}},
			}},
		}},
	}
	program, produced := Generate(compilation)
	if program != nil || !hasCode(produced, CodeGenerationFailure) {
		t.Fatalf("expected %s; received %v", CodeGenerationFailure, codes(produced))
	}
}

func TestModuleGlobalsInitializeInDeclarationOrder(t *testing.T) {
	program := generate(t, "zebra: String := \"z\"\nalpha: String := zebra\nwrite(alpha)\n")
	generated := programSource(t, program)
	zebra := strings.Index(generated, "gv_zebra")
	alpha := strings.Index(generated, "gv_alpha")
	assignment := strings.Index(generated, "func md_")
	if zebra < 0 || alpha < 0 || assignment < 0 {
		t.Fatal("expected both globals and a module initializer")
	}
	body := generated[assignment:]
	if strings.Index(body, "gv_zebra") > strings.Index(body, "gv_alpha") {
		t.Fatal("globals do not initialize in declaration order")
	}
}

func TestMangledIdentifiersAreStableAndDistinct(t *testing.T) {
	first := mangle(globalPrefix, "mem:/Main.ahd::symbol::binding::value")
	second := mangle(globalPrefix, "mem:/Other.ahd::symbol::binding::value")
	if first == second {
		t.Fatal("identities from different modules collided")
	}
	if first != mangle(globalPrefix, "mem:/Main.ahd::symbol::binding::value") {
		t.Fatal("mangling is not stable")
	}
	if mangle(globalPrefix, "") == "" {
		t.Fatal("an empty identity must still produce a valid identifier")
	}
}

// TestIndexedCompoundAssignmentEvaluatesItsTargetOnce drives the IR contract
// directly: the frontend currently rejects this source form, but the IR node is
// valid and the backend must still preserve single-evaluation lvalue
// semantics.
func TestIndexedCompoundAssignmentEvaluatesItsTargetOnce(t *testing.T) {
	intType := ir.Type{Kind: ir.IntType}
	listType := ir.Type{Kind: ir.ListType, Element: &intType}
	load := func() ir.Expr {
		return &ir.LoadExpr{ExprBase: ir.ExprBase{Type: listType, NullState: ir.NonNull}, Symbol: "values"}
	}
	literal := func(text string) ir.Expr {
		return &ir.LiteralExpr{ExprBase: ir.ExprBase{Type: intType, NullState: ir.NonNull}, Kind: ir.IntLiteral, Value: text}
	}
	compilation := &ir.Compilation{
		Entry: "mem:/Main.ahd",
		Modules: []*ir.Module{{
			ID: "mem:/Main.ahd", Name: "Main",
			Globals: []*ir.Global{{ID: "values", Name: "values", Type: listType, NullState: ir.NonNull}},
			Init: ir.Block{Statements: []ir.Statement{
				&ir.BindingStmt{
					Symbol: "values", Name: "values", Type: listType, NullState: ir.NonNull, Storage: ir.ModuleStorage,
					Initializer: &ir.ListExpr{ExprBase: ir.ExprBase{Type: listType, NullState: ir.NonNull}, ElementType: intType, Elements: []ir.Expr{literal("1")}},
				},
				&ir.CompoundAssignStmt{
					Target: ir.Target{Kind: ir.IndexTarget, Type: intType, Receiver: load(), Index: literal("0")},
					Op:     "CheckedIntAdd", Value: literal("10"),
				},
				&ir.UpdateStmt{
					Target: ir.Target{Kind: ir.IndexTarget, Type: intType, Receiver: load(), Index: literal("0")},
					Delta:  1,
				},
			}},
		}},
	}
	program, produced := Generate(compilation)
	if program == nil {
		t.Fatalf("expected a generated program; received %v", codes(produced))
	}
	generated := programSource(t, program)
	if strings.Count(generated, "ahdTemporary") < 4 {
		t.Fatalf("indexed updates did not bind their receiver and index to temporaries:\n%s", generated)
	}
	if !strings.Contains(generated, ".Set(ahdTemporary") || !strings.Contains(generated, ".At(ahdTemporary") {
		t.Fatalf("indexed updates did not read and write through one bound target:\n%s", generated)
	}
}

func TestClassGenerationIsDeterministic(t *testing.T) {
	compilation := lower(t, map[string]string{"/Main.ahd": `Person: Class<> := {
    structure: Attributes := (
        name: String
    )

    describe: Function := (
    ) -> String {
        return "Person {attribute.name}"
    }
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )

    describe: Override Function := (
    ) -> String {
        return "{SuperClass.describe()} #{attribute.number}"
    }
}

Failure: Class<Error> := {
    structure: Attributes := (
        SuperClass.attributes
    )
}

attempt {
    write(Student(name: "Ada", number: 1).describe())
    toss Failure(message: "x")
}
except Error as error {
    write(error.message)
}
ultimately {
    write("done")
}
`})
	first, firstDiagnostics := Generate(compilation)
	second, secondDiagnostics := Generate(compilation)
	if len(firstDiagnostics) != 0 || len(secondDiagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v %v", codes(firstDiagnostics), codes(secondDiagnostics))
	}
	for index := range first.Files {
		if first.Files[index].Content != second.Files[index].Content {
			t.Fatalf("Class and error generation is not deterministic for %s", first.Files[index].Name)
		}
	}
}

func TestOverrideSharesOneDispatchSlot(t *testing.T) {
	program := generate(t, `Animal: Class<> := {
    structure: Attributes := (
        name: String
    )

    speak: Function := (
    ) -> String {
        return "..."
    }
}

Dog: Class<Animal> := {
    structure: Attributes := (
        SuperClass.attributes
    )

    speak: Override Function := (
    ) -> String {
        return "woof"
    }
}

animal: Animal := Dog(name: "Rex")
write(animal.speak())
`)
	generated := programSource(t, program)
	slots := make(map[string]bool)
	for _, line := range strings.Split(generated, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "func (object *Cl_") {
			continue
		}
		if cut := strings.Index(trimmed, ") Mt_"); cut >= 0 {
			rest := trimmed[cut+2:]
			slots[rest[:strings.Index(rest, "(")]] = true
		}
	}
	if len(slots) != 1 {
		t.Fatalf("an override must share the parent dispatch slot; found %v", slots)
	}
}

func TestInheritedFieldsAreOneStorageLocation(t *testing.T) {
	program := generate(t, `Person: Class<> := {
    structure: Attributes := (
        name: String
    )
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )
}

	student: Student := Student(name: "Ada", number: 1)
write(student.name)
`)
	generated := programSource(t, program)
	storageLocations := 0
	for _, line := range strings.Split(generated, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Fd_name_") && !strings.Contains(trimmed, "(") && strings.HasSuffix(trimmed, " string") {
			storageLocations++
		}
	}
	if storageLocations != 1 {
		t.Fatalf("an inherited attribute was duplicated into the subclass:\n%s", generated)
	}
	if !strings.Contains(generated, "Cl_Person_") {
		t.Fatal("the subclass struct does not embed its parent")
	}
}

func TestConstantReferenceBindingsFreezeTheirGraph(t *testing.T) {
	program := generate(t, "source: List<Int> := [1]\nfrozen: Constant List<Int> := source\nwrite(len(frozen))\n")
	if !strings.Contains(programSource(t, program), "AhdFreeze(") {
		t.Fatal("a Constant reference binding did not deep-freeze its graph")
	}
}

// TestBetweenDoesNotMaterializeAList asserts the non-allocation contract at the
// code-generation level: a range loop must drive the lazy runtime value and
// must never build a List.
func TestBetweenDoesNotMaterializeAList(t *testing.T) {
	program := generate(t, `for value in between(1, 10000000) {
    write(value)
}
`)
	generated := programSource(t, program)
	if !strings.Contains(generated, "AhdBetween(") || !strings.Contains(generated, ".Next()") {
		t.Fatalf("a range loop did not drive the lazy iteration:\n%s", generated)
	}
	if strings.Contains(generated, "AhdNewList") || strings.Contains(generated, "AhdList[") {
		t.Fatalf("a range loop materialized a List:\n%s", generated)
	}
	if strings.Contains(generated, ".Snapshot()") {
		t.Fatalf("a lazy range must not be snapshotted:\n%s", generated)
	}
}

func TestBetweenDefaultsAreExplicitInGeneratedSource(t *testing.T) {
	program := generate(t, "for value in between(5) {\n    write(value)\n}\n")
	if !strings.Contains(programSource(t, program), "AhdBetween(int64(0), int64(5), int64(1))") {
		t.Fatalf("between defaults were not applied:\n%s", programSource(t, program))
	}
}

func TestCollectionIterationStillSnapshots(t *testing.T) {
	program := generate(t, `values: List<Int> := [1]
for value in values {
    write(value)
}
`)
	if !strings.Contains(programSource(t, program), ".Snapshot()") {
		t.Fatal("collection iteration lost its shallow snapshot")
	}
}

func TestClassDescriptorsPublishTheirOwnMembers(t *testing.T) {
	program := generate(t, `Person: Class<> := {
    structure: Attributes := (
        name: String
    )

    describe: Function := (
    ) -> String {
        return attribute.name
    }
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )

    describe: Override Function := (
    ) -> String {
        return attribute.name
    }

    study: Function := (
    ) -> Nothing {
        return
    }
}

person: Person := Student(name: "Ali", number: 1)
write(person has number)
write(person has not nickname)
`)
	generated := programSource(t, program)
	if !strings.Contains(generated, `Members: []string{"describe", "name"}`) {
		t.Fatalf("the Person descriptor does not publish its own members:\n%s", generated)
	}
	// An override reuses the slot its parent introduced, so the child does not
	// restate the inherited name; the Parent chain already publishes it.
	if !strings.Contains(generated, `Members: []string{"number", "study"}`) {
		t.Fatalf("the Student descriptor does not publish exactly its own members:\n%s", generated)
	}
	if !strings.Contains(generated, "AhdHasMember(") {
		t.Fatalf("has did not lower to the runtime member lookup:\n%s", generated)
	}
	if strings.Contains(generated, "AhdConstBool(gv_person") {
		t.Fatalf("has was still folded to a compile-time constant:\n%s", generated)
	}
	if !strings.Contains(generated, `(!AhdHasMember(`) {
		t.Fatalf("has not did not negate the same lookup:\n%s", generated)
	}
}
