package module

import (
	"os"
	"path/filepath"
	"testing"
)

// writeApp materializes files (relative-path -> source text) under a fresh
// temp directory and returns the absolute entry path. require(...) needs a
// real filesystem (canonicalization and symlink containment checks are not
// meaningfully expressible over InMemoryWorkspace's virtual paths), so these
// tests use scratch directories instead of the in-memory harness the bring
// graph tests use.
func writeApp(t *testing.T, files map[string]string, entry string) string {
	t.Helper()
	root := t.TempDir()
	for relative, text := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return filepath.Join(root, filepath.FromSlash(entry))
}

func compileFS(t *testing.T, entry string) CompilationResult {
	t.Helper()
	return NewCompiler(FileResolver{}, FileLoader{}).Compile(entry)
}

func TestRequireSimpleComposition(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd":     "require(\"Message.ahd\")\nresult: String := Greeting()\n",
		"Message.ahd": "Greeting: Function := () -> String {\n    return \"hi\"\n}\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireClean(t, result)
}

func TestRequireNestedComposition(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd": "require(\"A.ahd\")\nresult: String := Outer()\n",
		"A.ahd":   "require(\"B.ahd\")\nOuter: Function := () -> String {\n    return Inner()\n}\n",
		"B.ahd":   "Inner: Function := () -> String {\n    return \"deep\"\n}\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireClean(t, result)
}

func TestRequireDeduplicatesDiamondDependency(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd": "require(\"A.ahd\")\nrequire(\"B.ahd\")\nresult: String := UseA() + UseB()\n",
		"A.ahd":   "require(\"Shared.ahd\")\nUseA: Function := () -> String {\n    return SharedValue()\n}\n",
		// A "./" segment must canonicalize to the exact same file as A.ahd's
		// plain spelling, so this is also the deterministic-identity check.
		"B.ahd":             "require(\"Shared/./Values.ahd\")\nUseB: Function := () -> String {\n    return SharedValue()\n}\n",
		"Shared.ahd":        "require(\"Shared/Values.ahd\")\nUnused: Function := () -> Nothing {\n}\n",
		"Shared/Values.ahd": "SharedValue: Function := () -> String {\n    return \"shared\"\n}\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireClean(t, result)
	entryModule := moduleNamed(t, result, "app")
	// A, B, Shared, and Shared/Values are four genuinely distinct files; the
	// dedup being tested is that B's "Shared/./Values.ahd" spelling reuses
	// the exact same Shared/Values.ahd unit Shared.ahd already required,
	// rather than compiling a fifth, separately-identified copy of it.
	if len(entryModule.RequiredFiles) != 4 {
		t.Fatalf("expected exactly 4 distinct required files (A, B, Shared, Shared/Values -- merged once despite two spellings), got %d: %+v",
			len(entryModule.RequiredFiles), entryModule.RequiredFiles)
	}
}

func TestRequireCycleIsDiagnosed(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd": "require(\"A.ahd\")\n",
		"A.ahd":   "require(\"B.ahd\")\n",
		"B.ahd":   "require(\"A.ahd\")\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	item := requireCode(t, result, "SEM047")
	if item.Diagnostic.Message == "" {
		t.Fatalf("expected a non-empty cycle diagnostic message")
	}
}

func TestRequireMissingFileIsDiagnosed(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd": "require(\"Pages/Missing.ahd\")\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	item := requireCode(t, result, "SEM046")
	if !containsSubstring(item.Diagnostic.Message, "Pages/Missing.ahd") {
		t.Fatalf("expected the missing-file diagnostic to name the requested path, got %q", item.Diagnostic.Message)
	}
}

func TestRequireAbsolutePathRejected(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd": "require(\"/etc/Secret.ahd\")\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireCode(t, result, "SEM048")
}

func TestRequireParentTraversalRejected(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd": "require(\"../Secret.ahd\")\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireCode(t, result, "SEM048")
}

func TestRequireSymlinkEscapeRejected(t *testing.T) {
	root := filepath.Dir(writeApp(t, map[string]string{
		"app.ahd": "require(\"External.ahd\")\n",
	}, "app.ahd"))
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.ahd")
	if err := os.WriteFile(secret, []byte("Leak: Function := () -> Nothing {\n}\n"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(root, "External.ahd")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks unavailable in this environment: %v", err)
	}
	result := compileFS(t, filepath.Join(root, "app.ahd"))
	requireCode(t, result, "SEM048")
}

func TestRequireNonAhdExtensionRejected(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd":   "require(\"data.json\")\n",
		"data.json": "{}",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireCode(t, result, "SEM048")
}

func TestRequireNonLiteralArgumentRejected(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd": "path: String := \"Message.ahd\"\nrequire(path)\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireCode(t, result, "PAR014")
}

func TestRequireNestedScopeRejected(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd":     "check: Function := () -> Nothing {\n    require(\"Message.ahd\")\n}\n",
		"Message.ahd": "Greeting: Function := () -> String {\n    return \"hi\"\n}\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireCode(t, result, "PAR005")
}

func TestRequireFileLocalBringVisibility(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd": "bring HTTP\nrequire(\"Card.ahd\")\n",
		// Card.ahd never brings HTTP itself, so it must not see the name
		// merely because the requiring file did.
		"Card.ahd": "HTTP.text(\"hi\")\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireCode(t, result, "SEM049")
}

func TestRequireFileLocalBringOwnDeclarationWorks(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd":  "require(\"Card.ahd\")\nCard()\n",
		"Card.ahd": "bring HTTP\nCard: Function := () -> Nothing {\n    HTTP.text(\"hi\")\n}\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireClean(t, result)
}

func TestRequireDuplicateDeclarationAcrossFiles(t *testing.T) {
	entry := writeApp(t, map[string]string{
		"app.ahd": "require(\"A.ahd\")\nrequire(\"B.ahd\")\n",
		"A.ahd":   "Foo: Function := () -> Nothing {\n}\n",
		"B.ahd":   "Foo: Function := () -> Nothing {\n}\n",
	}, "app.ahd")
	result := compileFS(t, entry)
	requireCode(t, result, "SEM002")
}

func containsSubstring(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
