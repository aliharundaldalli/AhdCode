package semantic

import "testing"

const vectorClass = `Vector2: Class<> := {
    structure: Attributes := (
        x: Real
        y: Real
    )

    CEqual: Function := (
        other: Vector2
    ) -> Bool {
        return attribute.x == other.x and attribute.y == other.y
    }

    CAdd: Function := (
        other: Vector2
    ) -> Vector2 {
        return Vector2(x: attribute.x + other.x, y: attribute.y + other.y)
    }

    CAdd: Overload Function := (
        scalar: Real
    ) -> Vector2 {
        return Vector2(x: attribute.x + scalar, y: attribute.y + scalar)
    }

    CNegate: Function := (
    ) -> Vector2 {
        return Vector2(x: -attribute.x, y: -attribute.y)
    }

    CStr: Function := (
    ) -> String {
        return "Vector2"
    }
}
`

func TestClassProtocolValidDeclarations(t *testing.T) {
	_, result := analyzeText(t, vectorClass+"a: Vector2 := Vector2(x: 1.0, y: 2.0)\nb: Vector2 := Vector2(x: 3.0, y: 4.0)\nc: Vector2 := a + b\nd: Vector2 := a + 5.0\ne: Bool := a == b\nf: Bool := a != b\ng: Vector2 := -a\nh: String := str(a)\n")
	requireSemanticClean(t, result)
}

func TestClassProtocolOrderingUsesCCompare(t *testing.T) {
	score := `Score: Class<> := {
    structure: Attributes := (value: Int)
    CCompare: Function := (other: Score) -> Int {
        return attribute.value - other.value
    }
}
`
	_, result := analyzeText(t, score+"a: Score := Score(value: 1)\nb: Score := Score(value: 2)\nw: Bool := a < b\nx: Bool := a <= b\ny: Bool := a > b\nz: Bool := a >= b\n")
	requireSemanticClean(t, result)
}

func TestClassProtocolMissingProtocolIsOrdinaryOperatorError(t *testing.T) {
	class := `Plain: Class<> := { structure: Attributes := (x: Int) }
a: Plain := Plain(x: 1)
b: Plain := Plain(x: 2)
c: Plain := a + b
`
	_, result := analyzeText(t, class)
	requireSemanticCode(t, result, codeOperatorType)
}

func TestClassProtocolNoReverseDispatch(t *testing.T) {
	money := `Money: Class<> := {
    structure: Attributes := (amount: Real)
    CAdd: Function := (extra: Real) -> Money {
        return Money(amount: attribute.amount + extra)
    }
}
`
	_, forward := analyzeText(t, money+"m: Money := Money(amount: 1.0)\nn: Money := m + 2.0\n")
	requireSemanticClean(t, forward)

	_, reverse := analyzeText(t, money+"m: Money := Money(amount: 1.0)\nn: Real := 2.0 + m\n")
	requireSemanticCode(t, reverse, codeOperatorType)
}

func TestClassProtocolCompoundAssignment(t *testing.T) {
	money := `Money: Class<> := {
    structure: Attributes := (amount: Real)
    CAdd: Function := (extra: Real) -> Money {
        return Money(amount: attribute.amount + extra)
    }
}
`
	_, result := analyzeText(t, money+"m: Money := Money(amount: 1.0)\nm += 2.0\n")
	requireSemanticClean(t, result)
}

func TestClassProtocolReservedSlotRejectsNonFunction(t *testing.T) {
	_, result := analyzeText(t, "Bad: Class<> := {\n    structure: Attributes := (x: Int)\n    CAdd: Int := 5\n}\n")
	requireSemanticCode(t, result, codeProtocolSlot)
}

func TestClassProtocolOrdinaryCPrefixedMemberIsUnaffected(t *testing.T) {
	class := `Fine: Class<> := {
    structure: Attributes := (x: Int)
    CWhatever: Function := () -> Int { return attribute.x }
    Calculate: Function := () -> Int { return attribute.x * 2 }
}
`
	_, result := analyzeText(t, class+"f: Fine := Fine(x: 5)\nn: Int := f.CWhatever()\nm: Int := f.Calculate()\n")
	requireSemanticClean(t, result)
}

func TestClassProtocolMalformedSignatures(t *testing.T) {
	cases := map[string]string{
		"CEqual missing argument": "CEqual: Function := () -> Bool { return true }",
		"CEqual wrong return":     "CEqual: Function := (other: Bad) -> Int { return 1 }",
		"CCompare wrong return":   "CCompare: Function := (other: Bad) -> Bool { return true }",
		"CStr unexpected arg":     "CStr: Function := (x: Int) -> String { return \"x\" }",
		"CStr wrong return":       "CStr: Function := () -> Int { return 1 }",
		"CNegate unexpected arg":  "CNegate: Function := (x: Int) -> Bad { return attribute }",
		"CAdd wrong arity":        "CAdd: Function := () -> Bad { return attribute }",
	}
	for name, method := range cases {
		t.Run(name, func(t *testing.T) {
			source := "Bad: Class<> := {\n    structure: Attributes := (x: Int)\n    " + method + "\n}\n"
			_, result := analyzeText(t, source)
			requireSemanticCode(t, result, codeProtocolSignature)
		})
	}
}

func TestClassProtocolInheritanceAndOverride(t *testing.T) {
	source := `Animal: Class<> := {
    structure: Attributes := (name: String)
    CStr: Function := () -> String { return "Animal" }
}
Dog: Class<Animal> := {
    structure: Attributes := (SuperClass.attributes)
    CStr: Override Function := () -> String { return "Dog" }
}
a: Animal := Dog(name: "Rex")
text: String := str(a)
kind: String := type(a)
`
	_, result := analyzeText(t, source)
	requireSemanticClean(t, result)
}

func TestTypeFundamental(t *testing.T) {
	cases := []string{
		"n := type(5)",
		"n := type(5.0)",
		"n := type(\"x\")",
		"n := type(true)",
		"n := type(null)",
		"values: List<Int> := [1, 2]\nn := type(values)",
		"scores: Pair<String, Int> := {\"a\": 1}\nn := type(scores)",
		"x: Int? := 5\nn := type(x)",
	}
	for _, source := range cases {
		_, result := analyzeText(t, source)
		requireSemanticClean(t, result)
	}
}

func TestTypeFundamentalArity(t *testing.T) {
	_, result := analyzeText(t, "n := type(1, 2)")
	requireSemanticCode(t, result, codeCallArguments)
}

func TestIdFundamentalAcceptsReferenceTypes(t *testing.T) {
	source := `Foo: Class<> := { structure: Attributes := (x: Int) }
f: Foo := Foo(x: 1)
values: List<Int> := [1, 2]
scores: Pair<String, Int> := {"a": 1}
a := id(f)
b := id(values)
c := id(scores)
`
	_, result := analyzeText(t, source)
	requireSemanticClean(t, result)
}

func TestIdFundamentalRejectsPrimitives(t *testing.T) {
	cases := []string{"id(5)", "id(3.14)", "id(true)", "id(\"Ali\")"}
	for _, source := range cases {
		_, result := analyzeText(t, source)
		requireSemanticCode(t, result, codeCallArguments)
	}
}

func TestIdFundamentalRejectsUnnarrowedNullable(t *testing.T) {
	source := `Foo: Class<> := { structure: Attributes := (x: Int) }
f: Foo? := null
n := id(f)
`
	_, result := analyzeText(t, source)
	requireSemanticCode(t, result, codeNullableUse)
}
