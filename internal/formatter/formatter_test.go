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
		"calculate: Function := (",
		"    values: List<Real>",
		") -> Real {",
		"    return sum(values) / len(values)",
		"numbers: List<Int> := [1, 2, 3]",
		"swap(numbers[0], numbers[1])",
		"    name: \"Ali\"",
	} {
		if !strings.Contains(formatted, wanted) {
			t.Fatalf("formatted text lacks %q:\n%s", wanted, formatted)
		}
	}
	if twice := formatText(t, formatted); twice != formatted {
		t.Fatalf("formatter is not idempotent:\nfirst:\n%s\nsecond:\n%s", formatted, twice)
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
	if !strings.Contains(formatted, "Person: Class<> := {") || !strings.Contains(formatted, "        name: String") {
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
