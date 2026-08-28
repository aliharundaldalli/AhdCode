package golang

import (
	"go/format"
	"testing"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
)

// FuzzGenerationIsPanicFreeAndDeterministic asserts the backend contract on
// arbitrary source: it never panics, it never returns a program together with
// an error diagnostic, every returned program is valid gofmt-stable Go, and
// generating twice produces identical output.
func FuzzGenerationIsPanicFreeAndDeterministic(f *testing.F) {
	f.Add("write(\"hello\")")
	f.Add("x: Int := 1\nwrite(x + 1)")
	f.Add("a: List<Int> := [1]\nclear(a)\nwrite(len(a))")
	f.Add("f: Function := (x: Int) -> Int { return x }\nwrite(f(1))")
	f.Add("attempt {\n}\n")
	f.Add("if {")
	f.Add("x: Int := 99999999999999999999")
	f.Fuzz(func(t *testing.T, text string) {
		workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": text})
		frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
		if frontend.HasErrors() {
			return
		}
		lowered := lowering.LowerCompilation(frontend)
		if lowered.HasErrors() || lowered.Compilation == nil {
			return
		}
		first, firstDiagnostics := Generate(lowered.Compilation)
		second, secondDiagnostics := Generate(lowered.Compilation)
		if first == nil {
			if !anyError(firstDiagnostics) {
				t.Fatal("a refused generation reported no error diagnostic")
			}
			if second != nil {
				t.Fatal("generation refusal is not deterministic")
			}
			return
		}
		if anyError(firstDiagnostics) || anyError(secondDiagnostics) {
			t.Fatalf("a generated program carries error diagnostics: %v", codes(firstDiagnostics))
		}
		if second == nil || len(first.Files) != len(second.Files) {
			t.Fatal("generation is not deterministic")
		}
		for index := range first.Files {
			if first.Files[index].Content != second.Files[index].Content {
				t.Fatalf("generation is not deterministic for %s", first.Files[index].Name)
			}
			formatted, err := format.Source([]byte(first.Files[index].Content))
			if err != nil {
				t.Fatalf("%s is not valid Go: %v", first.Files[index].Name, err)
			}
			if string(formatted) != first.Files[index].Content {
				t.Fatalf("%s is not gofmt-stable", first.Files[index].Name)
			}
		}
	})
}

func anyError(items []diagnostics.Diagnostic) bool {
	for _, item := range items {
		if item.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}

// FuzzMangledIdentifiersAreValidGo asserts that any identity string yields a
// usable Go identifier.
func FuzzMangledIdentifiersAreValidGo(f *testing.F) {
	f.Add("mem:/Main.ahd::symbol::binding::value")
	f.Add("kaçıncı")
	f.Add("range")
	f.Add("")
	f.Add("::::")
	f.Add("\x00\x01")
	f.Fuzz(func(t *testing.T, identity string) {
		for _, prefix := range []string{globalPrefix, localPrefix, functionPrefix, classPrefix, fieldPrefix} {
			name := mangle(prefix, identity)
			if name != mangle(prefix, identity) {
				t.Fatal("mangling is not stable")
			}
			source := "package main\n\nvar " + name + " int\n"
			if _, err := format.Source([]byte(source)); err != nil {
				t.Fatalf("mangled identifier %q is not valid Go: %v", name, err)
			}
		}
	})
}
