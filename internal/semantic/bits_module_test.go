package semantic

import "testing"

const bitsPreamble = "bring Bits\nfrom Bits bring BitsError\n\n"

func TestBitsModuleRegistered(t *testing.T) {
	modules := StandardModuleInterfaces()
	module, ok := modules["Bits"]
	if !ok {
		t.Fatal("Bits module not found in StandardModuleInterfaces")
	}
	exports := []string{
		"bitAnd", "bitOr", "bitXor", "bitNot",
		"shiftLeft", "shiftRight", "shiftRightUnsigned",
		"rotateLeft", "rotateRight",
		"count", "leadingZeros", "trailingZeros",
		"BitsError",
	}
	for _, name := range exports {
		if module.Exports[name] == nil {
			t.Fatalf("Bits module missing export %q", name)
		}
	}
}

func TestBitsModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, bitsPreamble+`a: Int := Bits.bitAnd(12, 10)
b: Int := Bits.bitOr(12, 10)
c: Int := Bits.bitXor(12, 10)
d: Int := Bits.bitNot(0)
e: Int := Bits.shiftLeft(1, 10)
f: Int := Bits.shiftRight(-8, 1)
g: Int := Bits.shiftRightUnsigned(-1, 60)
h: Int := Bits.rotateLeft(1, 63)
i: Int := Bits.rotateRight(1, 1)
j: Int := Bits.count(255)
k: Int := Bits.leadingZeros(1)
l: Int := Bits.trailingZeros(8)
`)
	requireSemanticClean(t, result)
}

// Every Bits function returns Int; binding a result to String must fail.
func TestBitsResultsAreInt(t *testing.T) {
	result := analyzeWithStandardModules(t, bitsPreamble+`wrong: String := Bits.bitAnd(1, 2)
`)
	requireSemanticFailure(t, result)
}

func TestBitsFunctionsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`Bits.bitAnd(1)`,
		`Bits.bitAnd(1, 2, 3)`,
		`Bits.bitAnd("1", 2)`,
		`Bits.bitOr(1)`,
		`Bits.bitXor(1, "2")`,
		`Bits.bitNot()`,
		`Bits.bitNot(1, 2)`,
		`Bits.bitNot("x")`,
		`Bits.shiftLeft(1)`,
		`Bits.shiftRight(1, "2")`,
		`Bits.shiftRightUnsigned(1)`,
		`Bits.rotateLeft(1)`,
		`Bits.rotateRight("1", 2)`,
		`Bits.count()`,
		`Bits.count(1, 2)`,
		`Bits.leadingZeros("x")`,
		`Bits.trailingZeros()`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, bitsPreamble+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

// Bits shift distances are validated at run time, so BitsError must be a
// catchable Class like every other module error.
func TestBitsErrorCatchable(t *testing.T) {
	result := analyzeWithStandardModules(t, bitsPreamble+`attempt {
    write(str(Bits.shiftLeft(1, 64)))
} except BitsError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

// Real numbers are not Ints: the module is deliberately integer-only.
func TestBitsRejectsRealArguments(t *testing.T) {
	result := analyzeWithStandardModules(t, bitsPreamble+`value: Int := Bits.bitAnd(1.5, 2)
`)
	requireSemanticFailure(t, result)
}
