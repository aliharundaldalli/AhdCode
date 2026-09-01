package module

import "testing"

func TestBuiltinExcelCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd":  "bring Excel\nbook := Excel.new()",
		"/Excel.ahd": `new: String := "shadow"`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/Excel.ahd").ID) != 0 {
		t.Fatal("the sibling Excel.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "Excel")
	if !module.Source.Builtin || module.ID != "builtin:Excel" {
		t.Fatalf("Excel did not keep its built-in identity: %#v", module)
	}
}
