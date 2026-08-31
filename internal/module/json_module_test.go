package module

import "testing"

func TestBuiltinJSONCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring JSON\nvalue := JSON.nullValue()",
		"/JSON.ahd": `nullValue: String := "shadow"`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/JSON.ahd").ID) != 0 {
		t.Fatal("the sibling JSON.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "JSON")
	if !module.Source.Builtin || module.ID != "builtin:JSON" {
		t.Fatalf("JSON did not keep its built-in identity: %#v", module)
	}
}

func TestJSONIsExplicit(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd": `value := JSON.nullValue()`,
	}, "/Main.ahd")
	requireCode(t, result, "SEM001")
}
