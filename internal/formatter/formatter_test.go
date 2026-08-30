package formatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/lexer"
	"ahdcode/internal/parser"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
)

func formatText(t *testing.T, text string) string {
	t.Helper()
	result := Format(source.NewFile(1, "main.ahd", text))
	if result.HasErrors() {
		t.Fatalf("format diagnostics: %+v", result.Diagnostics)
	}
	return result.Text
}

func TestCanonicalSpacingLayoutAndIdempotence(t *testing.T) {
	input := `calculate:Function:=(values:List<Real>)->Real{return sum(values)/len(values)}
numbers:List<Int>:=[1,2,3]
swap(numbers[0],numbers[1])
createStudent(
name:"Ali"
number:123
active:true
)
`
	formatted := formatText(t, input)
	for _, wanted := range []string{
		"calculate: Function := (values: List<Real>) -> Real {",
		"    return sum(values) / len(values)",
		"numbers: List<Int> := [1, 2, 3]",
		"swap(numbers[0], numbers[1])",
		"createStudent(name: \"Ali\", number: 123, active: true)",
	} {
		if !strings.Contains(formatted, wanted) {
			t.Fatalf("formatted text lacks %q:\n%s", wanted, formatted)
		}
	}
	if twice := formatText(t, formatted); twice != formatted {
		t.Fatalf("formatter is not idempotent:\nfirst:\n%s\nsecond:\n%s", formatted, twice)
	}
}

func TestLambdaFormattingAndIdempotence(t *testing.T) {
	input := "short:=lambda(x:Int)->x>0\nlong:=lambda(first:Int,second:Int,description:String,flag:Bool)->first+second>0 and description!=\"\" and flag\n"
	formatted := formatText(t, input)
	for _, wanted := range []string{
		"short := lambda (x: Int) -> x > 0",
		"long := lambda (",
		"    first: Int",
		") -> first + second > 0 and description != \"\" and flag",
	} {
		if !strings.Contains(formatted, wanted) {
			t.Fatalf("formatted lambda lacks %q:\n%s", wanted, formatted)
		}
	}
	if twice := formatText(t, formatted); twice != formatted {
		t.Fatalf("lambda formatter is not idempotent:\nfirst:\n%s\nsecond:\n%s", formatted, twice)
	}
}

// TestLambdaDependencyListFormatting checks that the formatter normalizes
// spacing around a dependency entry (no space after # or @, one space after
// Local/Global) while preserving whichever spelling -- compact sigil or full
// keyword -- the source actually used, since both are canonical.
func TestLambdaDependencyListFormatting(t *testing.T) {
	compact := formatText(t, "f:=lambda[#minimum,@Maximum](x:Int)->x")
	if want := "f := lambda [#minimum, @Maximum] (x: Int) -> x"; !strings.Contains(compact, want) {
		t.Fatalf("formatted = %q, want %q", compact, want)
	}
	if twice := formatText(t, compact); twice != compact {
		t.Fatalf("compact dependency list is not idempotent:\nfirst:\n%s\nsecond:\n%s", compact, twice)
	}

	full := formatText(t, "f:=lambda[Local minimum,Global Maximum](x:Int)->x")
	if want := "f := lambda [Local minimum, Global Maximum] (x: Int) -> x"; !strings.Contains(full, want) {
		t.Fatalf("formatted = %q, want %q", full, want)
	}
	if twice := formatText(t, full); twice != full {
		t.Fatalf("full dependency list is not idempotent:\nfirst:\n%s\nsecond:\n%s", full, twice)
	}

	mixed := formatText(t, "f:=lambda[#minimum,Global Maximum](x:Int)->x")
	if want := "f := lambda [#minimum, Global Maximum] (x: Int) -> x"; !strings.Contains(mixed, want) {
		t.Fatalf("formatted = %q, want %q", mixed, want)
	}
	if twice := formatText(t, mixed); twice != mixed {
		t.Fatalf("mixed dependency list is not idempotent:\nfirst:\n%s\nsecond:\n%s", mixed, twice)
	}
}

func TestFormatterPreservesUpToOneBlankLineBetweenStatements(t *testing.T) {
	input := "a: Int := 1\n\n\n\nb: Int := 2\nc: Int := 3\n"
	want := "a: Int := 1\n\nb: Int := 2\nc: Int := 3\n"
	formatted := formatText(t, input)
	if formatted != want {
		t.Fatalf("formatted = %q, want %q", formatted, want)
	}
	for _, line := range strings.Split(formatted, "\n") {
		if strings.TrimRight(line, " \t") != line {
			t.Fatalf("blank/line has trailing whitespace: %q in %q", line, formatted)
		}
	}
	if twice := formatText(t, formatted); twice != formatted {
		t.Fatal("blank-line formatting is not idempotent")
	}
}

func TestShortConstructsStayOnOneLine(t *testing.T) {
	input := "check: Function := (\n    x: Int\n) -> Bool {\nreturn x > 0\n}\n"
	want := "check: Function := (x: Int) -> Bool {\n    return x > 0\n}\n"
	formatted := formatText(t, input)
	if formatted != want {
		t.Fatalf("short signatureformatted =\n%q\nwant\n%q", formatted, want)
	}
	if twice := formatText(t, formatted); twice != formatted {
		t.Fatal("short-signature formatting is not idempotent")
	}
}

func TestLongConstructsBreakToOneItemPerLineWithNoTrailingComma(t *testing.T) {
	input := "calculate: Function := (first: Int, second: Int, description: String, flag: Bool) -> Real {\nreturn first\n}\n"
	want := "calculate: Function := (\n" +
		"    first: Int\n" +
		"    second: Int\n" +
		"    description: String\n" +
		"    flag: Bool\n" +
		") -> Real {\n" +
		"    return first\n" +
		"}\n"
	formatted := formatText(t, input)
	if formatted != want {
		t.Fatalf("long signature formatted =\n%q\nwant\n%q", formatted, want)
	}
	if strings.Contains(formatted, ",") {
		t.Fatalf("broken multi-line construct must not use commas:\n%s", formatted)
	}
	if twice := formatText(t, formatted); twice != formatted {
		t.Fatal("long-signature formatting is not idempotent")
	}
}

func TestCommentsTripleStringsInterpolationAndUnicodeSurvive(t *testing.T) {
	input := `// başlık
ad : String := "Ayşe" // kişi
metin:String:="""first
  second {ad}
last"""
write("Merhaba { ad }") /* son */
if true {
/* çok
   satır */
write(ad)
}
`
	formatted := formatText(t, input)
	for _, exact := range []string{"// başlık", "// kişi", "/* son */", "/* çok\n   satır */", "Ayşe", `"""first
  second {ad}
last"""`, `"Merhaba { ad }"`} {
		if !strings.Contains(formatted, exact) {
			t.Fatalf("formatted text lost %q:\n%s", exact, formatted)
		}
	}
	if twice := formatText(t, formatted); twice != formatted {
		t.Fatal("comment/string formatting is not idempotent")
	}
}

func TestFunctionClassAndSemanticValidityArePreserved(t *testing.T) {
	input := `Person:Class<>:={
structure:Attributes:=(
name:String
age:Int
)
describe:Function:=()->String{
return "{attribute.name} - {attribute.age}"
}
}
person:Person:=Person(name:"Ali",age:25)
write(person.describe())
`
	formatted := formatText(t, input)
	if !strings.Contains(formatted, "Person: Class<> := {") || !strings.Contains(formatted, "    structure: Attributes := (name: String, age: Int)") {
		t.Fatalf("Class layout:\n%s", formatted)
	}
	for _, text := range []string{input, formatted} {
		file := source.NewFile(1, "main.ahd", text)
		lexed := lexer.Lex(file)
		parsed := parser.Parse(file, lexed.Tokens)
		analyzed := semantic.Analyze(parsed)
		if lexed.HasErrors() || parsed.HasErrors() || analyzed.HasErrors() {
			t.Fatalf("semantic validity changed for:\n%s\nlex=%+v\nparse=%+v\nsemantic=%+v", text, lexed.Diagnostics, parsed.Diagnostics, analyzed.Diagnostics)
		}
	}
}

func TestInvalidSourceIsNotFormatted(t *testing.T) {
	result := Format(source.NewFile(1, "bad.ahd", "x: Int := )\n"))
	if !result.HasErrors() || result.Text != "" {
		t.Fatalf("invalid format result = %#v", result)
	}
}

func TestValidExampleCorpusRemainsSemanticAndIdempotent(t *testing.T) {
	for _, name := range []string{"tester.ahd", "class_test.ahd", "errors.ahd", "inheritance.ahd", "deep_freeze.ahd", "nullable_function.ahd"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "examples", name)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			formatted := formatText(t, string(content))
			if second := formatText(t, formatted); second != formatted {
				t.Fatal("example formatting is not idempotent")
			}
			file := source.NewFile(1, path, formatted)
			lexed := lexer.Lex(file)
			parsed := parser.Parse(file, lexed.Tokens)
			analyzed := semantic.Analyze(parsed)
			if lexed.HasErrors() || parsed.HasErrors() || analyzed.HasErrors() {
				t.Fatalf("formatted example is invalid: lex=%+v parse=%+v semantic=%+v\n%s", lexed.Diagnostics, parsed.Diagnostics, analyzed.Diagnostics, formatted)
			}
		})
	}
}

func TestFormatterPreservesInferredDeclarations(t *testing.T) {
	sourceText := "age:=25\nname:Local:=\"Ali\"\nuser:Local User?:=null\n"
	want := "age := 25\nname: Local := \"Ali\"\nuser: Local User? := null\n"
	result := Format(source.NewFile(1, "inferred.ahd", sourceText))
	if result.HasErrors() {
		t.Fatalf("format diagnostics: %#v", result.Diagnostics)
	}
	if result.Text != want {
		t.Fatalf("formatted = %q, want %q", result.Text, want)
	}
}

func TestFormatterRendersIterationBindingTypes(t *testing.T) {
	cases := map[string]string{
		"typed for":   "for j: Int in between(1,100) {\nwrite(j)\n}\n",
		"untyped for": "for j in between(1,100) {\nwrite(j)\n}\n",
	}
	expected := map[string]string{
		"typed for":   "for j: Int in between(1, 100) {\n    write(j)\n}\n",
		"untyped for": "for j in between(1, 100) {\n    write(j)\n}\n",
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			first := formatText(t, source)
			if first != expected[name] {
				t.Fatalf("formatted output\n want %q\n have %q", expected[name], first)
			}
			if second := formatText(t, first); second != first {
				t.Fatalf("formatting is not idempotent\n first %q\n again %q", first, second)
			}
		})
	}
}

func TestFormatterRendersModuleAliasCanonically(t *testing.T) {
	first := formatText(t, "bring   Time   as   T\n")
	if first != "bring Time as T\n" {
		t.Fatalf("formatted alias = %q", first)
	}
	if second := formatText(t, first); second != first {
		t.Fatalf("module alias formatting is not idempotent: %q", second)
	}
}
