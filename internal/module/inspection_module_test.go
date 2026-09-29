package module

import "testing"

// TestInspectionModulesCannotBeShadowed covers the v2.7.0 standard-module
// names: a sibling Disk.ahd or Service.ahd is not what `bring` loads.
func TestInspectionModulesCannotBeShadowed(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd":    "bring Disk\nbring Service\ndisk := Disk.inspect(\"/\")\nunit := Service.status(\"nginx.service\")\n",
		"/Disk.ahd":    `inspect: Bool := true`,
		"/Service.ahd": `status: Bool := true`,
	}, "/Main.ahd")
	requireClean(t, result)
	for _, name := range []string{"/Disk.ahd", "/Service.ahd"} {
		if workspace.LoadCount(memoryIdentity(name).ID) != 0 {
			t.Fatalf("the sibling %s shadowed the standard module", name)
		}
	}
	for _, name := range []string{"Disk", "Service"} {
		module := moduleNamed(t, result, name)
		if !module.Source.Builtin || string(module.ID) != "builtin:"+name {
			t.Fatalf("%s did not keep its built-in identity: %#v", name, module)
		}
	}
}
