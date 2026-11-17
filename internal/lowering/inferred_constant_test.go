package lowering

import (
	"strings"
	"testing"

	"ahdcode/internal/ir"
	"ahdcode/internal/module"
)

// The compiler must reject an inferred Constant that is not a constant
// expression, and it must lower a forward reference. Neither path may surface
// the internal "IR node has no concrete type" diagnostic.
func TestInferredConstantDoesNotLeakIR002(t *testing.T) {
	rejected := []string{
		"x: Int := 5\ny: Constant := x\n",
		"y: Constant := x\nx: Constant := y\n",
	}
	for _, text := range rejected {
		frontend := compileFrontend(text)
		if !frontend.HasErrors() {
			lowered := LowerCompilation(frontend)
			t.Fatalf("expected a semantic error, lowering = %+v", lowered.Diagnostics)
		}
		saw := false
		for _, item := range frontend.Diagnostics {
			if item.Diagnostic.Code == "IR002" {
				t.Fatalf("IR002 leaked: %+v", frontend.Diagnostics)
			}
			if item.Diagnostic.Code == "SEM010" {
				saw = true
			}
		}
		if !saw {
			t.Fatalf("expected SEM010, got %+v", frontend.Diagnostics)
		}
	}

	lowerSources(t, map[string]string{"/Main.ahd": "y: Constant := x\nx: Constant := 5\n"}, "/Main.ahd")
	lowerSources(t, map[string]string{"/Main.ahd": "b: Constant := a\na: Constant := [1, 2]\n"}, "/Main.ahd")
}

// A pure constant graph is initialized before any use, dependency first.
// A constant that reads a mutable binding stays after that binding.
func TestPureConstantsInitializeInDependencyOrder(t *testing.T) {
	forward := lowerSources(t, map[string]string{"/Main.ahd": "y: Constant := x + 1\nwrite(y)\nx: Constant := 5\n"}, "/Main.ahd")
	if got := initOrder(t, forward); got != "x,y,stmt" {
		t.Fatalf("forward init = %s", got)
	}
	list := lowerSources(t, map[string]string{"/Main.ahd": "values: Constant := [x]\nwrite(values[0])\nx: Constant := 5\n"}, "/Main.ahd")
	if got := initOrder(t, list); got != "x,values,stmt" {
		t.Fatalf("list init = %s", got)
	}
	uses := lowerSources(t, map[string]string{"/Main.ahd": "area: Function := () -> Int uses [@radius] {\n    return radius * radius\n}\nwrite(area())\nradius: Constant := 20\n"}, "/Main.ahd")
	if got := initOrder(t, uses); got != "radius,stmt" {
		t.Fatalf("uses init = %s", got)
	}
	global := lowerSources(t, map[string]string{"/Main.ahd": "scale: Function := () -> Int {\n    radius: Global\n    return radius + 1\n}\nwrite(scale())\nradius: Constant := 20\n"}, "/Main.ahd")
	if got := initOrder(t, global); got != "radius,stmt" {
		t.Fatalf("global init = %s", got)
	}
	anchored := lowerSources(t, map[string]string{"/Main.ahd": "source: List<Int> := [1, 2, 3]\nwrite(\"A\")\nfrozen: Constant List<Int> := source\nwrite(\"B\")\n"}, "/Main.ahd")
	if got := initOrder(t, anchored); got != "source,stmt,frozen,stmt" {
		t.Fatalf("anchored init = %s", got)
	}
	mutable := lowerSources(t, map[string]string{"/Main.ahd": "v: Int := 1\nb: Constant := [a]\na: Constant := [v]\nwrite(b[0][0])\n"}, "/Main.ahd")
	if got := initOrder(t, mutable); got != "v,a,b,stmt" {
		t.Fatalf("mutable forward init = %s", got)
	}
	called := lowerSources(t, map[string]string{"/Main.ahd": "write(\"before\")\nitems: Constant := answer()\nwrite(\"after\")\nanswer: Function := () -> List<Int> {\n    write(\"call\")\n    return [7]\n}\n"}, "/Main.ahd")
	if got := initOrder(t, called); got != "stmt,items,stmt" {
		t.Fatalf("call init = %s", got)
	}
}

func initOrder(t *testing.T, result Result) string {
	t.Helper()
	main := moduleIR(t, result.Compilation, "Main")
	names := make([]string, 0, len(main.Init.Statements))
	for _, statement := range main.Init.Statements {
		binding, ok := statement.(*ir.BindingStmt)
		if !ok {
			names = append(names, "stmt")
			continue
		}
		names = append(names, binding.Name)
	}
	return strings.Join(names, ",")
}

func compileFrontend(text string) module.CompilationResult {
	workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": text})
	return module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
}
