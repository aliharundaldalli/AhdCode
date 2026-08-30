package semantic

import "testing"

const regexPreamble = "bring Regex\nfrom Regex bring Pattern\nfrom Regex bring RegexError\n\n"

func TestRegexModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, regexPreamble+`digits: Pattern := Regex.compile("[0-9]+")
found: String? := digits.find("abc123")
everyMatch: List<String> := digits.findAll("abc123")
groups: List<String>? := digits.groups("abc123")
replaced: String := digits.replace("abc123", "#")
parts: List<String> := digits.split("abc123")
matched: Bool := digits.matches("abc123")

attempt {
    Regex.compile("(")
}
except RegexError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

func TestRegexOperationWrongArgumentCount(t *testing.T) {
	result := analyzeWithStandardModules(t, regexPreamble+`digits: Pattern := Regex.compile("[0-9]+")
digits.matches()
`)
	requireSemanticCode(t, result, codeCallArguments)
}

func TestRegexOperationWrongArgumentType(t *testing.T) {
	result := analyzeWithStandardModules(t, regexPreamble+`digits: Pattern := Regex.compile("[0-9]+")
digits.matches(5)
`)
	requireSemanticCode(t, result, codeTypeMismatch)
}

func TestRegexFindResultRequiresNarrowing(t *testing.T) {
	result := analyzeWithStandardModules(t, regexPreamble+`digits: Pattern := Regex.compile("[0-9]+")
found := digits.find("abc123")
write(found.upper())
`)
	requireSemanticCode(t, result, codeNullableUse)
}

func TestRegexIdAndTypeIntegration(t *testing.T) {
	result := analyzeWithStandardModules(t, regexPreamble+`digits: Pattern := Regex.compile("[0-9]+")
identity: Int := id(digits)
name: String := type(digits)
`)
	requireSemanticClean(t, result)
}
