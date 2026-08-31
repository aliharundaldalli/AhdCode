package module

import "testing"

func TestBuiltinXMLCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring XML\nvalue := XML.text(\"x\")",
		"/XML.ahd":  `text: String := "shadow"`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/XML.ahd").ID) != 0 {
		t.Fatal("the sibling XML.ahd shadowed the standard module")
	}
	module := moduleNamed(t, result, "XML")
	if !module.Source.Builtin || module.ID != "builtin:XML" {
		t.Fatalf("XML did not keep its built-in identity: %#v", module)
	}
}

func TestXMLIsExplicit(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd": `value := XML.text("x")`,
	}, "/Main.ahd")
	requireCode(t, result, "SEM001")
}
