package semantic

import (
	"testing"

	"ahdcode/internal/lexer"
	"ahdcode/internal/parser"
	"ahdcode/internal/source"
)

const timePreamble = "bring Time\nfrom Time bring DateTime\nfrom Time bring Duration\n\n"

// analyzeText for a module that can see the compiler-supplied standard
// modules, which is how ordinary compilation analyzes a module that imports.
func analyzeWithStandardModules(t *testing.T, text string) Result {
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
	return AnalyzeWithEnvironment(parsed, Environment{
		ModuleID: "test:Main", ModuleName: "Main", Imports: StandardModuleInterfaces(),
	})
}

// TestTimeModuleHasExactSignatures pins the public Time surface: every
// function, its result type, and its argument contract.
func TestTimeModuleHasExactSignatures(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"now produces DateTime", "current: DateTime := Time.now()", true},
		{"monotonic produces Real", "value: Real := Time.monotonic()", true},
		{"sleep produces Nothing", "Time.sleep(0)", true},
		{"duration produces Duration", "wait: Duration := Time.duration(milliseconds: 500)", true},
		{"dateTime with required arguments", "value: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)", true},
		{"dateTime with every argument", "value: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 1, minute: 2, second: 3, millisecond: 4)", true},
		{"between produces Duration", "a: DateTime := Time.now()\ngap: Duration := Time.between(a, a)", true},

		{"now takes no argument", "current: DateTime := Time.now(1)", false},
		{"now is not Real", "value: Real := Time.now()", false},
		{"monotonic is not Int", "value: Int := Time.monotonic()", false},
		{"sleep rejects String", `Time.sleep("500")`, false},
		{"sleep rejects Real", "Time.sleep(1.5)", false},
		{"sleep result is not a value", "value: Int := Time.sleep(0)", false},
		{"duration rejects String", `wait: Duration := Time.duration(milliseconds: "500")`, false},
		{"duration is not DateTime", "wait: DateTime := Time.duration(milliseconds: 5)", false},
		{"dateTime requires day", "value: DateTime := Time.dateTime(year: 2026, month: 1)", false},
		{"dateTime rejects String", `value: DateTime := Time.dateTime(year: "2026", month: 1, day: 1)`, false},
		{"between rejects Int", "gap: Duration := Time.between(10, 20)", false},
		{"between requires two arguments", "a: DateTime := Time.now()\ngap: Duration := Time.between(a)", false},
		{"an unknown Time member is rejected", "value: Int := Time.epoch()", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeWithStandardModules(t, timePreamble+test.text)
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			if !result.HasErrors() {
				t.Fatal("expected a Time diagnostic")
			}
		})
	}
}

// TestTimeClassMembersHaveExactSignatures pins the DateTime attributes and the
// built-in members DateTime and Calendar publish.
func TestTimeClassMembersHaveExactSignatures(t *testing.T) {
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"year attribute is Int", "current: DateTime := Time.now()\nvalue: Int := current.year", true},
		{"weekday attribute is Int", "current: DateTime := Time.now()\nvalue: Int := current.weekday", true},
		{"duration milliseconds is Int", "wait: Duration := Time.duration(milliseconds: 5)\nvalue: Int := wait.milliseconds", true},
		{"duration seconds is Real", "wait: Duration := Time.duration(milliseconds: 5)\nvalue: Real := wait.seconds", true},
		{"before produces Bool", "a: DateTime := Time.now()\nvalue: Bool := a.before(a)", true},
		{"after produces Bool", "a: DateTime := Time.now()\nvalue: Bool := a.after(a)", true},
		{"sameMoment produces Bool", "a: DateTime := Time.now()\nvalue: Bool := a.sameMoment(a)", true},
		{"toString produces String", "a: DateTime := Time.now()\nvalue: String := a.toString()", true},
		{"isLeapYear produces Bool", "value: Bool := Time.Calendar.isLeapYear(2028)", true},
		{"daysInMonth produces Int", "value: Int := Time.Calendar.daysInMonth(2028, 2)", true},
		{"calendar weekday produces Int", "value: Int := Time.Calendar.weekday(2026, 8, 29)", true},

		{"year attribute is not String", "current: DateTime := Time.now()\nvalue: String := current.year", false},
		{"duration seconds is not Int", "wait: Duration := Time.duration(milliseconds: 5)\nvalue: Int := wait.seconds", false},
		{"before requires a DateTime", "a: DateTime := Time.now()\nvalue: Bool := a.before(1)", false},
		{"before requires one argument", "a: DateTime := Time.now()\nvalue: Bool := a.before()", false},
		{"toString takes no argument", "a: DateTime := Time.now()\nvalue: String := a.toString(1)", false},
		{"isLeapYear rejects String", `value: Bool := Time.Calendar.isLeapYear("2028")`, false},
		{"daysInMonth rejects String", `value: Int := Time.Calendar.daysInMonth(2028, "2")`, false},
		{"daysInMonth requires two arguments", "value: Int := Time.Calendar.daysInMonth(2028)", false},
		{"calendar weekday requires three arguments", "value: Int := Time.Calendar.weekday(2026, 8)", false},
		{"an unknown DateTime member is rejected", "a: DateTime := Time.now()\nvalue: Int := a.epoch", false},
		{"an unknown Calendar member is rejected", "value: Int := Time.Calendar.monthName(1)", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeWithStandardModules(t, timePreamble+test.text)
			if test.ok {
				requireSemanticClean(t, result)
				return
			}
			if !result.HasErrors() {
				t.Fatal("expected a Time member diagnostic")
			}
		})
	}
}

// TestTimeValuesAreReadOnly records that DateTime and Duration attributes are
// Constant, so the existing Constant rule rejects assignment through a value
// without any Time-specific immutability concept.
func TestTimeValuesAreReadOnly(t *testing.T) {
	for _, text := range []string{
		"current: DateTime := Time.now()\ncurrent.year = 2030",
		"current: DateTime := Time.now()\ncurrent.weekday = 1",
		"wait: Duration := Time.duration(milliseconds: 5)\nwait.milliseconds = 9",
		"wait: Duration := Time.duration(milliseconds: 5)\nwait.seconds = 9.0",
	} {
		result := analyzeWithStandardModules(t, timePreamble+text)
		requireSemanticCode(t, result, codeConstantAssignment)
	}
}

// TestTimeValuesAreNotConstructedDirectly keeps validation from being bypassed
// by construction, and points at the function that produces each value.
func TestTimeValuesAreNotConstructedDirectly(t *testing.T) {
	for _, text := range []string{
		"value: DateTime := DateTime(year: 2026, month: 1, day: 1)",
		"value: Duration := Duration(milliseconds: 5)",
	} {
		result := analyzeWithStandardModules(t, timePreamble+text)
		requireSemanticCode(t, result, codeCallArguments)
	}
}

// TestTimeImportsFollowTheOrdinaryModuleRules checks that Time needs no import
// syntax of its own.
func TestTimeImportsFollowTheOrdinaryModuleRules(t *testing.T) {
	namespaced := analyzeWithStandardModules(t, "bring Time\n\nvalue: Bool := Time.Calendar.isLeapYear(2028)\n")
	requireSemanticClean(t, namespaced)

	selective := analyzeWithStandardModules(t, "bring Time\nfrom Time bring DateTime\n\ncurrent: DateTime := Time.now()\n")
	requireSemanticClean(t, selective)

	// A Time type still needs its import, exactly like any module Class.
	missing := analyzeWithStandardModules(t, "bring Time\n\ncurrent: DateTime := Time.now()\n")
	requireSemanticCode(t, missing, codeInvalidType)
}

// TestTimeDoesNotShadowUserDeclarations records that the Time Classes are
// ordinary module exports rather than globally reserved names.
func TestTimeDoesNotShadowUserDeclarations(t *testing.T) {
	_, result := analyzeText(t, `DateTime: Class<> := {
    structure: Attributes := (
        label: String
    )
}

value: DateTime := DateTime(label: "own")
write(value.label)
`)
	requireSemanticClean(t, result)
}

// TestUserClassKeepsItsOwnTimeMemberNames records that the Time members are
// selected by the compiler-supplied Class identity, not by member name, so a
// user Class may declare before, after, or isLeapYear itself.
func TestUserClassKeepsItsOwnTimeMemberNames(t *testing.T) {
	_, result := analyzeText(t, `Marker: Class<> := {
    structure: Attributes := (
        year: Int
    )

    before: Function := (
        other: Int
    ) -> Bool {
        return other > attribute.year
    }
}

value: Marker := Marker(year: 2026)
write(value.before(2030))
write(value.year)
`)
	requireSemanticClean(t, result)
}
