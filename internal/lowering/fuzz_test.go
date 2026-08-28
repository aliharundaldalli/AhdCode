package lowering

import (
	"testing"

	"ahdcode/internal/ir"
	"ahdcode/internal/module"
)

func FuzzLoweringDeterministicAndPanicFree(f *testing.F) {
	f.Add("x: Int := 1")
	f.Add("f: Function := (x: Int) -> Int { return x + 1 }")
	f.Add("if {")
	f.Fuzz(func(t *testing.T, text string) {
		workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": text})
		frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
		first := LowerCompilation(frontend)
		second := LowerCompilation(frontend)
		if first.Compilation != nil && second.Compilation != nil && ir.Dump(first.Compilation) != ir.Dump(second.Compilation) {
			t.Fatal("nondeterministic lowering")
		}
	})
}
