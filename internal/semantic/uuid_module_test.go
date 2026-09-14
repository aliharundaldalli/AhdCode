package semantic

import (
	"strings"
	"testing"
)

const uuidPreamble = "bring UUID\nfrom UUID bring (UUIDValue, UUIDError)\n\n"

func TestUUIDModuleRegistered(t *testing.T) {
	module, ok := StandardModuleInterfaces()["UUID"]
	if !ok {
		t.Fatal("UUID module not registered")
	}
	if module.ModuleID != "builtin:UUID" {
		t.Fatalf("UUID identity = %q", module.ModuleID)
	}
	want := []string{"UUIDError", "UUIDValue", "isValid", "parse", "v4", "v7", "zero"}
	if strings.Join(module.ExportNames, ",") != strings.Join(want, ",") {
		t.Fatalf("UUID exports %v, want exactly %v", module.ExportNames, want)
	}
	// The frozen v1.4.0 surface: no nullable parse, no max UUID, no other
	// versions, and no bridge to Identity.id().
	for _, name := range []string{"tryParse", "max", "v1", "v3", "v5", "v6", "v8", "new", "fromIdentity", "fromString"} {
		if module.Exports[name] != nil {
			t.Fatalf("UUID must not export %q", name)
		}
	}
}

func TestUUIDModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, uuidPreamble+`first: UUIDValue := UUID.v4()
second: UUIDValue := UUID.v7()
parsed: UUIDValue := UUID.parse("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
zero: UUIDValue := UUID.zero()
valid: Bool := UUID.isValid("not a uuid")
text: String := parsed.string()
version: Int := second.version()
empty: Bool := zero.isZero()
equal: Bool := parsed.equals(first)
order: Int := first.compare(second)
identity: Bool := first == second
rendered: String := str(first)
inferred := UUID.v7()
values: List<UUIDValue> := [first, second, inferred]
maybe: UUIDValue? := null
if maybe != null {
    write(maybe.string())
}
attempt {
    UUID.parse("x")
} except UUIDError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

// Static mistakes are compiler diagnostics, never a runtime UUIDError. A
// UUIDValue has no Class protocol methods, so ordering operators stay invalid.
func TestUUIDRejectStaticMistakes(t *testing.T) {
	for _, source := range []string{
		`UUID.parse(42)`,
		`UUID.parse()`,
		`UUID.v4(1)`,
		`UUID.zero("x")`,
		`UUID.isValid()`,
		`UUID.v7().equals("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")`,
		`UUID.v7().equals()`,
		`UUID.v7().compare(UUID.v4(), UUID.v4())`,
		`UUID.v7().string(1)`,
		`text: String := UUID.v7()`,
		`id: UUIDValue := "017f22e2-79b0-7cc3-98c4-dc0c0c07398f"`,
		`count: String := UUID.v7().version()`,
		`UUID.v7().timestamp()`,
		`UUID.tryParse("x")`,
		`UUID.max()`,
		`earlier: Bool := UUID.v4() < UUID.v7()`,
		"maybe: UUIDValue? := null\nmaybe.string()",
		"maybe: UUIDValue? := null\nUUID.v4().equals(maybe)",
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, uuidPreamble+source+"\n"))
	}
}

func TestUUIDValuesAreNotConstructedDirectly(t *testing.T) {
	result := analyzeWithStandardModules(t, uuidPreamble+`UUIDValue("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")`+"\n")
	requireSemanticFailure(t, result)
	if !strings.Contains(semanticHintsOf(result), "create a UUIDValue") {
		t.Fatalf("direct construction of UUIDValue has no creation hint: %q", semanticHintsOf(result))
	}
}
