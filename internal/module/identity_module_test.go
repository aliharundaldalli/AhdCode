package module

import "testing"

func TestBuiltinIdentityCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring Identity\nvalue := Identity.id()",
		"/Identity.ahd": `id: Bool := true`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/Identity.ahd").ID) != 0 {
		t.Fatal("the sibling Identity.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "Identity")
	if !module.Source.Builtin || module.ID != "builtin:Identity" {
		t.Fatalf("Identity did not keep its built-in identity: %#v", module)
	}
}
