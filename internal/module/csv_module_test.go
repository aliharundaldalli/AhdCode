package module

import "testing"

func TestBuiltinCSVCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring CSV\nrows: List<List<String>> := CSV.parse(\"a,b\")",
		"/CSV.ahd":  `parse: String := "shadow"`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/CSV.ahd").ID) != 0 {
		t.Fatal("the sibling CSV.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "CSV")
	if !module.Source.Builtin || module.ID != "builtin:CSV" {
		t.Fatalf("CSV did not keep its built-in identity: %#v", module)
	}
}

func TestCSVIsExplicit(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd": `rows: List<List<String>> := CSV.parse("a,b")`,
	}, "/Main.ahd")
	requireCode(t, result, "SEM001")
}
