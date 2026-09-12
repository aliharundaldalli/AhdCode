package semantic

import "testing"

const charactersPreamble = "bring Characters\nfrom Characters bring CharactersError\n\n"

func TestCharactersModuleRegistered(t *testing.T) {
	module, ok := StandardModuleInterfaces()["Characters"]
	if !ok {
		t.Fatal("Characters module not found in StandardModuleInterfaces")
	}
	if module.ModuleID != "builtin:Characters" {
		t.Fatalf("Characters canonical identity = %q, want builtin:Characters", module.ModuleID)
	}
	exports := []string{
		"list", "count", "codePoint", "fromCodePoint",
		"isLetter", "isDigit", "isWhitespace", "isUpper", "isLower",
		"isAlphaNumeric", "isPunctuation", "isSymbol",
		"CharactersError",
	}
	for _, name := range exports {
		if module.Exports[name] == nil {
			t.Fatalf("Characters module missing export %q", name)
		}
	}
	if len(module.Exports) != len(exports) {
		t.Fatalf("Characters exports %d names, want exactly %d: %v", len(module.Exports), len(exports), module.ExportNames)
	}
	// There is no Char type: a character is a String.
	for _, name := range []string{"Char", "Chars", "Character", "Rune"} {
		if module.Exports[name] != nil {
			t.Fatalf("Characters must not export %q", name)
		}
	}
}

func TestCharactersModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, charactersPreamble+`parts: List<String> := Characters.list("Aş😊")
total: Int := Characters.count("Aş😊")
point: Int := Characters.codePoint("A")
back: String := Characters.fromCodePoint(351)
a: Bool := Characters.isLetter("ş")
b: Bool := Characters.isDigit("7")
c: Bool := Characters.isWhitespace(" ")
d: Bool := Characters.isUpper("İ")
e: Bool := Characters.isLower("ı")
f: Bool := Characters.isAlphaNumeric("ğ")
g: Bool := Characters.isPunctuation("!")
h: Bool := Characters.isSymbol("😊")
for piece in Characters.list("abc") {
    write(piece)
}
`)
	requireSemanticClean(t, result)
}

func TestCharactersModuleAliasAndDirectImport(t *testing.T) {
	result := analyzeWithStandardModules(t, `bring Characters as C
from Characters bring (count, isLetter)
write(C.count("abc"))
write(count("abc"))
write(isLetter("a"))
`)
	requireSemanticClean(t, result)
}

// Wrong static argument types and arity are compile-time diagnostics with the
// existing stable codes, never a runtime CharactersError.
func TestCharactersRejectsWrongStaticTypes(t *testing.T) {
	tests := []struct {
		source string
		code   string
	}{
		{`Characters.list(65)`, codeTypeMismatch},
		{`Characters.count(true)`, codeTypeMismatch},
		{`Characters.codePoint(65)`, codeTypeMismatch},
		{`Characters.fromCodePoint("A")`, codeTypeMismatch},
		{`Characters.fromCodePoint(65.0)`, codeTypeMismatch},
		{`Characters.isLetter(1)`, codeTypeMismatch},
		{`Characters.isDigit(["7"])`, codeTypeMismatch},
		{`Characters.list()`, codeCallArguments},
		{`Characters.codePoint("A", "B")`, codeCallArguments},
		{`Characters.isSymbol()`, codeCallArguments},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, charactersPreamble+test.source+"\n")
			requireSemanticFailure(t, result)
			if result.Diagnostics[0].Code != test.code {
				t.Fatalf("%s: first diagnostic %s %q, want %s", test.source,
					result.Diagnostics[0].Code, result.Diagnostics[0].Message, test.code)
			}
		})
	}
}

func TestCharactersResultTypesAreStatic(t *testing.T) {
	for _, source := range []string{
		`wrong: String := Characters.count("a")`,
		`wrong: Int := Characters.fromCodePoint(65)`,
		`wrong: List<Int> := Characters.list("a")`,
		`wrong: String := Characters.isLetter("a")`,
	} {
		t.Run(source, func(t *testing.T) {
			requireSemanticFailure(t, analyzeWithStandardModules(t, charactersPreamble+source+"\n"))
		})
	}
}

func TestCharactersErrorCatchable(t *testing.T) {
	result := analyzeWithStandardModules(t, charactersPreamble+`attempt {
    write(str(Characters.codePoint("AB")))
} except CharactersError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}
