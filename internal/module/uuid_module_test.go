package module

import "testing"

func TestBuiltinUUIDCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring UUID\nvalue := UUID.v4()",
		"/UUID.ahd": `v4: Bool := true`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/UUID.ahd").ID) != 0 {
		t.Fatal("the sibling UUID.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "UUID")
	if !module.Source.Builtin || module.ID != "builtin:UUID" {
		t.Fatalf("UUID did not keep its built-in identity: %#v", module)
	}
}
