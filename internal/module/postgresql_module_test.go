package module

import "testing"

func TestBuiltinPostgreSQLCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd":       "bring PostgreSQL\nvalue := PostgreSQL.nullValue()",
		"/PostgreSQL.ahd": `nullValue: Bool := true`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/PostgreSQL.ahd").ID) != 0 {
		t.Fatal("the sibling PostgreSQL.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "PostgreSQL")
	if !module.Source.Builtin || module.ID != "builtin:PostgreSQL" {
		t.Fatalf("PostgreSQL did not keep its built-in identity: %#v", module)
	}
}
