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

func TestGeneratedProgramCarriesRuntime(t *testing.T) {
	program := generate(t, "write(\"hi\")\n")
	names := make([]string, 0, len(program.Files))
	for _, file := range program.Files {
		names = append(names, file.Name)
	}
	if strings.Join(names, ",") != programFileName+","+runtimeFileName {
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
	program := generate(t, "values: List<Int> := [1, null, 3]\nwrite(len(values))\n")
	if !strings.Contains(programSource(t, program), "*AhdList[*int64]") {
		t.Fatal("List elements were not boxed")
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
		"error handling": `attempt {
    write("body")
}
except Error as error {
    write(error.message)
}
`,
		"inheritance": `Animal: Class<> := {
    structure: Attributes := (
        name: String
    )
}

Dog: Class<Animal> := {
    structure: Attributes := (
        tag: String
    )
}

write("ready")
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
			Globals: []*ir.Global{{
				ID: "values", Name: "values", Type: listType, NullState: ir.NonNull,
				Initializer: &ir.ListExpr{ExprBase: ir.ExprBase{Type: listType, NullState: ir.NonNull}, ElementType: intType, Elements: []ir.Expr{literal("1")}},
			}},
			Init: ir.Block{Statements: []ir.Statement{
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
