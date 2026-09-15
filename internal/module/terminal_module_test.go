package module

import "testing"

func TestBuiltinTerminalCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd":     "bring Terminal\nTerminal.flush()",
		"/Terminal.ahd": `flush: Bool := true`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/Terminal.ahd").ID) != 0 {
		t.Fatal("the sibling Terminal.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "Terminal")
	if !module.Source.Builtin || module.ID != "builtin:Terminal" {
		t.Fatalf("Terminal did not keep its built-in identity: %#v", module)
	}
}
