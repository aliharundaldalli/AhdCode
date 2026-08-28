package module

import (
	"testing"

	"ahdcode/internal/semantic"
)

func TestMathUsesTheBuiltinModuleInterface(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": `bring Math
pi: Real := Math.PI
root: Real := Math.sqrt(25)
from Math bring cos
one: Real := cos(0)`,
	}, "/Main.ahd")
	requireClean(t, result)
	mathModule := moduleNamed(t, result, "Math")
	if !mathModule.Source.Builtin || mathModule.ID != "builtin:Math" || mathModule.Interface == nil {
		t.Fatalf("Math module = %#v", mathModule)
	}
	if workspace.LoadCount(mathModule.ID) != 0 {
		t.Fatal("built-in Math must not be loaded from the filesystem")
	}
}

func TestBuiltinMathCannotBeShadowedByASiblingFile(t *testing.T) {
	workspace, result := compileMemory(t, map[string]string{
		"/Main.ahd": "bring Math\nvalue: Real := Math.PI",
		"/Math.ahd": `PI: String := "shadow"`,
	}, "/Main.ahd")
	requireClean(t, result)
	if workspace.LoadCount(memoryIdentity("/Math.ahd").ID) != 0 {
		t.Fatal("the sibling Math.ahd shadowed the standard module")
	}
	if moduleNamed(t, result, "Math").Source.Builtin != true {
		t.Fatal("Math did not keep its built-in identity")
	}
}

func TestMathIsExplicitAndItsConstantsAreImmutable(t *testing.T) {
	_, missing := compileMemory(t, map[string]string{"/Main.ahd": "write(Math.PI)"}, "/Main.ahd")
	requireCode(t, missing, "SEM001")

	_, mutation := compileMemory(t, map[string]string{"/Main.ahd": "bring Math\nMath.PI = 3.0"}, "/Main.ahd")
	requireCode(t, mutation, "SEM009")

	_, absentAlias := compileMemory(t, map[string]string{"/Main.ahd": "bring Math\nwrite(Math.abs(1.0))"}, "/Main.ahd")
	requireCode(t, absentAlias, semantic.CodeNamespaceMember)
}

func TestOrdinaryModulesStillResolveBesideBuiltinMath(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd":    "bring Math\nfrom Helpers bring answer\nvalue: Real := Math.sqrt(answer)",
		"/Helpers.ahd": "answer: Int := 25",
	}, "/Main.ahd")
	requireClean(t, result)
}

func TestMathBringAllImportsOnlyItsExactPublicSurface(t *testing.T) {
	_, result := compileMemory(t, map[string]string{
		"/Main.ahd": `from Math bring all
value: Real := sqrt(PI)
roll: Int := randomInt(1, 1)`,
	}, "/Main.ahd")
	requireClean(t, result)
}
