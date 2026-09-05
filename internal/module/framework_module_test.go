package module

import (
	"strings"
	"testing"

	"ahdcode/internal/framework"
	"ahdcode/internal/semantic"
)

// compileWorkspace compiles one in-memory workspace and returns the result,
// so a test can assert on resolution without touching the filesystem.
func compileWorkspace(sources map[string]string, entry string) CompilationResult {
	workspace := NewInMemoryWorkspace(sources)
	return NewCompiler(workspace, workspace).Compile(entry)
}

func diagnosticsText(result CompilationResult) string {
	parts := make([]string, 0, len(result.Diagnostics))
	for _, item := range result.Diagnostics {
		parts = append(parts, item.Diagnostic.Code+": "+item.Diagnostic.Message)
	}
	return strings.Join(parts, "\n")
}

// A. `bring Web` resolves from the bundled sources, with no file on disk and
// no entry in the workspace.
func TestBringWebResolvesFromBundledSources(t *testing.T) {
	result := compileWorkspace(map[string]string{"/app.ahd": "bring Web\n"}, "/app.ahd")
	if result.HasErrors() {
		t.Fatalf("bring Web did not resolve:\n%s", diagnosticsText(result))
	}
	web := result.Modules[ModuleID(framework.ModuleID("Web"))]
	if web == nil {
		t.Fatal("the Web module is not in the compilation graph")
	}
	if !web.Source.Framework {
		t.Error("the Web module was not marked as a bundled framework module")
	}
	if web.Interface == nil {
		t.Fatal("the Web module produced no interface")
	}
}

// B. The framework's internal decomposition is invisible to applications:
// only the public name resolves, and the internal modules do not.
func TestFrameworkInternalModulesAreNotReachableFromApplications(t *testing.T) {
	for _, name := range framework.ModuleNames() {
		if framework.IsPublic(name) {
			continue
		}
		result := compileWorkspace(map[string]string{"/app.ahd": "bring " + name + "\n"}, "/app.ahd")
		if !result.HasErrors() {
			t.Errorf("application code was able to bring the internal framework module %s", name)
		}
	}
}

// C. A first-party module name is reserved: a sibling file of the same name
// never shadows the bundled module, exactly as for a built-in module.
func TestApplicationFileDoesNotShadowBundledModule(t *testing.T) {
	result := compileWorkspace(map[string]string{
		"/app.ahd": "bring Web\nfrom Web bring Response\n",
		"/Web.ahd": "impostor: Int := 1\n",
	}, "/app.ahd")
	if result.HasErrors() {
		t.Fatalf("a sibling Web.ahd shadowed the bundled module:\n%s", diagnosticsText(result))
	}
	web := result.Modules[ModuleID(framework.ModuleID("Web"))]
	if web == nil {
		t.Fatal("the bundled Web module was not the one compiled")
	}
}

// D. The bundled source is read from the compiler, never through the caller's
// SourceLoader, so no workspace can substitute its own text for it.
func TestBundledSourceIgnoresTheWorkspaceLoader(t *testing.T) {
	workspace := NewInMemoryWorkspace(map[string]string{
		"/app.ahd": "bring Web\n",
		"/Web.ahd": "broken\n",
	})
	result := NewCompiler(workspace, workspace).Compile("/app.ahd")
	if result.HasErrors() {
		t.Fatalf("compilation used the workspace copy of Web:\n%s", diagnosticsText(result))
	}
}

// E. The facade re-exports the types it imports, and re-exports them with the
// identity the source module already gave them -- not a duplicate.
func TestWebFacadeReExportsUnderlyingTypeIdentities(t *testing.T) {
	result := compileWorkspace(map[string]string{"/app.ahd": "bring Web\n"}, "/app.ahd")
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticsText(result))
	}
	web := result.Modules[ModuleID(framework.ModuleID("Web"))].Interface
	standards := semantic.StandardModuleInterfaces()
	for _, expected := range []struct{ module, name string }{
		{"HTTP", "Request"}, {"HTTP", "Response"}, {"HTTP", "Server"},
		{"HTTP", "Session"}, {"HTTP", "SessionStore"}, {"HTTP", "Cookie"},
		{"HTTP", "UploadedFile"}, {"HTTP", "HTTPError"},
		{"HTML", "HTMLNode"}, {"HTML", "HTMLError"},
	} {
		reExported := web.Exports[expected.name]
		if reExported == nil {
			t.Errorf("Web does not export %s", expected.name)
			continue
		}
		origin := standards[expected.module].Exports[expected.name]
		if origin == nil {
			t.Fatalf("%s.%s is missing from the standard interfaces", expected.module, expected.name)
		}
		if reExported.Class != origin.Class {
			t.Errorf("Web.%s is a different Class identity than %s.%s", expected.name, expected.module, expected.name)
		}
	}
}

// F. Re-export is confined to the bundled facade. An ordinary application
// module still does not republish what it imports, so `bring` stays
// non-transitive for user code.
func TestApplicationModulesDoNotReExportImports(t *testing.T) {
	result := compileWorkspace(map[string]string{
		"/app.ahd":    "bring Helper\nfrom Helper bring HTMLNode\n",
		"/Helper.ahd": "bring HTML\nfrom HTML bring HTMLNode\n",
	}, "/app.ahd")
	if !result.HasErrors() {
		t.Fatal("an ordinary module re-exported a name it merely imported")
	}
	if !strings.Contains(diagnosticsText(result), "HTMLNode") {
		t.Errorf("expected a diagnostic naming HTMLNode; received:\n%s", diagnosticsText(result))
	}
}

func TestWebFacadeExportsV017RouteTypes(t *testing.T) {
	result := compileWorkspace(map[string]string{
		"/app.ahd": "bring Web\nfrom Web bring (RouteSet, RouteGroup, WebRouteError)\n",
	}, "/app.ahd")
	if result.HasErrors() {
		t.Fatalf("v0.17 Web exports did not resolve:\n%s", diagnosticsText(result))
	}
}

// G. A bundled module's own path is virtual, so `ahdcode dev` never adds it to
// a watch set: there is no such file to change.
func TestBundledModulePathsAreVirtual(t *testing.T) {
	result := compileWorkspace(map[string]string{"/app.ahd": "bring Web\n"}, "/app.ahd")
	for id, item := range result.Modules {
		if item == nil || !item.Source.Framework {
			continue
		}
		if !framework.IsVirtualPath(item.File.Path) {
			t.Errorf("bundled module %s has the non-virtual path %q", id, item.File.Path)
		}
	}
}
