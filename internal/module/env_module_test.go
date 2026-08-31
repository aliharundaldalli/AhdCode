package module

import "testing"

func TestBuiltinEnvCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring Env\nvalue := Env.exists(\"PATH\")",
		"/Env.ahd":  `exists: Bool := true`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/Env.ahd").ID) != 0 {
		t.Fatal("the sibling Env.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "Env")
	if !module.Source.Builtin || module.ID != "builtin:Env" {
		t.Fatalf("Env did not keep its built-in identity: %#v", module)
	}
}

func TestEnvIsExplicit(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd": `value := Env.exists("PATH")`,
	}, "/Main.ahd")
	requireCode(t, result, "SEM001")
}
