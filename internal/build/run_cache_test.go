package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	backend "ahdcode/internal/backend/golang"
)

// isolateCache points the per-user cache directory at a temporary location so
// a test never reads or writes the developer's real cache.
func isolateCache(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	t.Setenv(disableRunCache, "")
	return home
}

func runSource(t *testing.T, directory, entry, stdin string) (string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code, result := RunProgramIO(filepath.Join(directory, entry), nil, strings.NewReader(stdin), &stdout, &stderr)
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	return stdout.String(), code
}

func cacheEntries(t *testing.T) []string {
	t.Helper()
	cache := openRunCache()
	if cache == nil {
		t.Fatal("expected an available run cache")
	}
	entries, err := os.ReadDir(cache.directory)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// TestRunCacheReusesAnIdenticalProgram covers the optimization itself: running
// the same unchanged program twice builds it once.
func TestRunCacheReusesAnIdenticalProgram(t *testing.T) {
	isolateCache(t)
	directory := writeSources(t, map[string]string{"main.ahd": "write(\"hello\")\n"})

	first, code := runSource(t, directory, "main.ahd", "")
	if first != "hello\n" || code != 0 {
		t.Fatalf("first run = %q (exit %d)", first, code)
	}
	after := cacheEntries(t)
	if len(after) != 1 {
		t.Fatalf("cache entries after one run = %v, want exactly one", after)
	}
	cache := openRunCache()
	built, err := os.Stat(filepath.Join(cache.directory, after[0]))
	if err != nil {
		t.Fatal(err)
	}

	second, code := runSource(t, directory, "main.ahd", "")
	if second != first || code != 0 {
		t.Fatalf("second run = %q (exit %d), want %q", second, code, first)
	}
	reused := cacheEntries(t)
	if len(reused) != 1 {
		t.Fatalf("cache entries after two runs = %v, want exactly one", reused)
	}
	// A rebuild publishes a new file by rename, so the entry would be a
	// different file. The same file proves the executable itself was reused.
	again, err := os.Stat(filepath.Join(cache.directory, reused[0]))
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(built, again) {
		t.Fatal("an unchanged program rebuilt instead of reusing its executable")
	}
}

// TestRunCacheNeverRunsAStaleExecutable is the correctness property the cache
// must not break: any change that can affect the program must be observed.
func TestRunCacheNeverRunsAStaleExecutable(t *testing.T) {
	isolateCache(t)
	directory := writeSources(t, map[string]string{
		"Shapes.ahd": "Shape: Class<> := {\n    structure: Attributes := (\n        label: String\n    )\n\n    describe: Function := (\n    ) -> String {\n        return attribute.label\n    }\n}\n",
		"Report.ahd": "from Shapes bring Shape\n\ntitle: Function := (\n    shape: Shape\n) -> String {\n    return \"Shape: {shape.describe()}\"\n}\n",
		"main.ahd":   "from Shapes bring Shape\nfrom Report bring title\n\nwrite(title(Shape(\"box\")))\n",
	})

	if out, _ := runSource(t, directory, "main.ahd", ""); out != "Shape: box\n" {
		t.Fatalf("initial run = %q", out)
	}

	// A directly imported module changes.
	rewrite(t, directory, "Shapes.ahd", "return attribute.label", "return \"[{attribute.label}]\"")
	if out, _ := runSource(t, directory, "main.ahd", ""); out != "Shape: [box]\n" {
		t.Fatalf("an edit to an imported module was not observed: %q", out)
	}

	// A transitively imported module changes.
	rewrite(t, directory, "Report.ahd", "\"Shape: {", "\"SHAPE >> {")
	if out, _ := runSource(t, directory, "main.ahd", ""); out != "SHAPE >> [box]\n" {
		t.Fatalf("an edit to a transitively imported module was not observed: %q", out)
	}

	// The entry module changes.
	rewrite(t, directory, "main.ahd", "Shape(\"box\")", "Shape(\"ring\")")
	if out, _ := runSource(t, directory, "main.ahd", ""); out != "SHAPE >> [ring]\n" {
		t.Fatalf("an edit to the entry module was not observed: %q", out)
	}

	// Reverting to an earlier program is a hit on that program's own entry,
	// which is correct reuse rather than a stale result.
	rewrite(t, directory, "main.ahd", "Shape(\"ring\")", "Shape(\"box\")")
	if out, _ := runSource(t, directory, "main.ahd", ""); out != "SHAPE >> [box]\n" {
		t.Fatalf("reverting an edit did not restore the earlier program: %q", out)
	}
}

func rewrite(t *testing.T, directory, name, from, to string) {
	t.Helper()
	path := filepath.Join(directory, name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(content), from, to, 1)
	if updated == string(content) {
		t.Fatalf("%s does not contain %q", name, from)
	}
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestRunCacheKeyCoversEveryBuildInput checks the key inputs directly, so a
// future change that drops one is caught here rather than in the field.
func TestRunCacheKeyCoversEveryBuildInput(t *testing.T) {
	isolateCache(t)
	cache := openRunCache()
	if cache == nil {
		t.Fatal("expected an available run cache")
	}
	program := &backend.GeneratedProgram{Files: []backend.GeneratedFile{
		{Name: "ahdcode_program.go", Content: "package main\n"},
		{Name: "ahdcode_runtime.go", Content: "// runtime\n"},
	}}
	base := cache.key(program, "/usr/bin/go")
	if base != cache.key(program, "/usr/bin/go") {
		t.Fatal("the key must be stable for one program")
	}

	changed := &backend.GeneratedProgram{Files: []backend.GeneratedFile{
		{Name: "ahdcode_program.go", Content: "package main\n\n// edit\n"},
		{Name: "ahdcode_runtime.go", Content: "// runtime\n"},
	}}
	if cache.key(changed, "/usr/bin/go") == base {
		t.Fatal("generated program content must change the key")
	}

	runtimeChanged := &backend.GeneratedProgram{Files: []backend.GeneratedFile{
		{Name: "ahdcode_program.go", Content: "package main\n"},
		{Name: "ahdcode_runtime.go", Content: "// runtime v2\n"},
	}}
	if cache.key(runtimeChanged, "/usr/bin/go") == base {
		t.Fatal("generated runtime content must change the key")
	}

	renamed := &backend.GeneratedProgram{Files: []backend.GeneratedFile{
		{Name: "other.go", Content: "package main\n"},
		{Name: "ahdcode_runtime.go", Content: "// runtime\n"},
	}}
	if cache.key(renamed, "/usr/bin/go") == base {
		t.Fatal("a generated file name must change the key")
	}

	if cache.key(program, "/opt/other/go") == base {
		t.Fatal("the Go toolchain identity must change the key")
	}

	// Content is length-framed, so no rearrangement of the same bytes across
	// files can collide with a different program.
	split := &backend.GeneratedProgram{Files: []backend.GeneratedFile{
		{Name: "ahdcode_program.go", Content: "package main\n// runtime\n"},
		{Name: "ahdcode_runtime.go", Content: ""},
	}}
	if cache.key(split, "/usr/bin/go") == base {
		t.Fatal("the key must frame each file's content")
	}

	for _, name := range buildEnvironment {
		t.Setenv(name, "cache-test-value")
		if cache.key(program, "/usr/bin/go") == base {
			t.Fatalf("%s must change the key", name)
		}
		t.Setenv(name, "")
	}
}

// TestRunCacheCanBeDisabled keeps an escape hatch that takes the ordinary
// build-and-discard path.
func TestRunCacheCanBeDisabled(t *testing.T) {
	isolateCache(t)
	t.Setenv(disableRunCache, "1")
	if openRunCache() != nil {
		t.Fatal("the cache must be disabled by the environment")
	}
	directory := writeSources(t, map[string]string{"main.ahd": "write(\"hello\")\n"})
	out, code := runSource(t, directory, "main.ahd", "")
	if out != "hello\n" || code != 0 {
		t.Fatalf("a run without the cache = %q (exit %d)", out, code)
	}
}

// TestRunCacheIgnoresAnUnusableEntry treats damage as a miss rather than a
// failure, so a broken cache can never break a run.
func TestRunCacheIgnoresAnUnusableEntry(t *testing.T) {
	isolateCache(t)
	cache := openRunCache()
	if cache == nil {
		t.Fatal("expected an available run cache")
	}
	key := "0000000000000000000000000000000000000000000000000000000000000000"
	if err := os.WriteFile(cache.path(key), []byte("not an executable"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found := cache.lookup(key); found {
		t.Fatal("a non-executable entry must be a miss")
	}
	if _, found := cache.lookup("missing-key"); found {
		t.Fatal("an absent entry must be a miss")
	}
	if err := os.MkdirAll(cache.path("directory-key"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, found := cache.lookup("directory-key"); found {
		t.Fatal("a directory must be a miss")
	}
}

// TestRunCacheStaysBounded keeps the persistent cache from growing without
// end, so a user never has to clean it up by hand.
func TestRunCacheStaysBounded(t *testing.T) {
	isolateCache(t)
	cache := openRunCache()
	if cache == nil {
		t.Fatal("expected an available run cache")
	}
	newest := ""
	for index := 0; index < runCacheLimit+20; index++ {
		newest = filepath.Join(cache.directory, "entry-"+strconv.Itoa(index))
		if err := os.WriteFile(newest, []byte("x"), 0o700); err != nil {
			t.Fatal(err)
		}
		stamp := time.Unix(0, int64(index+1)*int64(time.Second))
		if err := os.Chtimes(newest, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	cache.prune()
	entries, err := os.ReadDir(cache.directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > runCacheLimit {
		t.Fatalf("cache holds %d entries, want at most %d", len(entries), runCacheLimit)
	}
	// Eviction is least-recently-used, so the newest entry has to survive.
	if _, err := os.Stat(newest); err != nil {
		t.Fatalf("the most recently used entry was evicted: %v", err)
	}
}

// TestRunCacheEvictsBySize keeps a few very large executables from filling the
// cache even when the entry count is small.
func TestRunCacheEvictsBySize(t *testing.T) {
	isolateCache(t)
	cache := openRunCache()
	if cache == nil {
		t.Fatal("expected an available run cache")
	}
	chunk := make([]byte, 1<<20)
	for index := 0; index < 4; index++ {
		name := filepath.Join(cache.directory, "big-"+strconv.Itoa(index))
		file, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0o700)
		if err != nil {
			t.Fatal(err)
		}
		// A sparse file reports a large size without writing a large file.
		if _, err := file.WriteAt(chunk, runCacheBytes/2); err != nil {
			t.Fatal(err)
		}
		_ = file.Close()
		stamp := time.Unix(0, int64(index+1)*int64(time.Second))
		if err := os.Chtimes(name, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	cache.prune()
	entries, err := os.ReadDir(cache.directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) >= 4 {
		t.Fatalf("cache kept %d oversized entries, want the total size bounded", len(entries))
	}
}

// TestRunCachePublishesAtomically checks that an entry only ever becomes
// visible complete: publishing renames a fully built file into place, so a
// concurrent run sees either no entry or a usable one, never a partial write.
func TestRunCachePublishesAtomically(t *testing.T) {
	isolateCache(t)
	cache := openRunCache()
	if cache == nil {
		t.Fatal("expected an available run cache")
	}
	key := strings.Repeat("a", 64)

	reserved, ok := cache.reserve()
	if !ok {
		t.Fatal("expected a reserved name")
	}
	if filepath.Dir(reserved) != cache.directory {
		t.Fatalf("a reservation must live in the cache directory: %s", reserved)
	}
	// A reservation that was never built publishes nothing and leaves no entry.
	if _, published := cache.publish(key, reserved); published {
		t.Fatal("an unbuilt reservation must not publish")
	}
	if _, found := cache.lookup(key); found {
		t.Fatal("a failed publish must leave no entry")
	}

	reserved, _ = cache.reserve()
	if err := os.WriteFile(reserved, []byte("#!/bin/sh\nexit 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	published, ok := cache.publish(key, reserved)
	if !ok {
		t.Fatal("a built reservation must publish")
	}
	if published != cache.path(key) {
		t.Fatalf("publish returned %s, want %s", published, cache.path(key))
	}
	// The build output is moved, not copied, so no partial file is left over.
	if _, err := os.Stat(reserved); !os.IsNotExist(err) {
		t.Fatalf("the reservation must not survive publishing: %v", err)
	}
	if _, found := cache.lookup(key); !found {
		t.Fatal("a published entry must be usable")
	}
	if entries, _ := os.ReadDir(cache.directory); len(entries) != 1 {
		t.Fatalf("publishing left %d files, want exactly the entry", len(entries))
	}
}

// TestRunCachePrunesWhenPublishing checks that the bound is enforced as part
// of publishing, so an ordinary run never leaves the cache oversized.
func TestRunCachePrunesWhenPublishing(t *testing.T) {
	isolateCache(t)
	cache := openRunCache()
	if cache == nil {
		t.Fatal("expected an available run cache")
	}
	for index := 0; index < runCacheLimit+8; index++ {
		name := filepath.Join(cache.directory, "old-"+strconv.Itoa(index))
		if err := os.WriteFile(name, []byte("x"), 0o700); err != nil {
			t.Fatal(err)
		}
		stamp := time.Unix(0, int64(index+1)*int64(time.Second))
		if err := os.Chtimes(name, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	reserved, _ := cache.reserve()
	if err := os.WriteFile(reserved, []byte("#!/bin/sh\nexit 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	key := strings.Repeat("b", 64)
	if _, ok := cache.publish(key, reserved); !ok {
		t.Fatal("expected the entry to publish")
	}
	entries, err := os.ReadDir(cache.directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > runCacheLimit {
		t.Fatalf("publishing left %d entries, want at most %d", len(entries), runCacheLimit)
	}
	// The entry just published is the most recently used, so it must survive.
	if _, found := cache.lookup(key); !found {
		t.Fatal("publishing evicted the entry it had just published")
	}
}

// TestRunCachePrunesADirectoryThatIsAlreadyOversized covers a cache left too
// large by an earlier compiler with looser bounds: the next run brings it back
// inside the current bound rather than leaving it oversized forever.
func TestRunCachePrunesADirectoryThatIsAlreadyOversized(t *testing.T) {
	isolateCache(t)
	cache := openRunCache()
	if cache == nil {
		t.Fatal("expected an available run cache")
	}
	for index := 0; index < runCacheLimit*3; index++ {
		name := filepath.Join(cache.directory, "legacy-"+strconv.Itoa(index))
		if err := os.WriteFile(name, []byte("x"), 0o700); err != nil {
			t.Fatal(err)
		}
		stamp := time.Unix(0, int64(index+1)*int64(time.Second))
		if err := os.Chtimes(name, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	directory := writeSources(t, map[string]string{"main.ahd": "write(\"hello\")\n"})
	if out, code := runSource(t, directory, "main.ahd", ""); out != "hello\n" || code != 0 {
		t.Fatalf("run = %q (exit %d)", out, code)
	}
	entries, err := os.ReadDir(cache.directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > runCacheLimit {
		t.Fatalf("an oversized cache stayed at %d entries, want at most %d", len(entries), runCacheLimit)
	}
}

// TestCachedRunsPreserveProgramBehavior checks that everything a program can
// observe still works when its executable came from the cache.
func TestCachedRunsPreserveProgramBehavior(t *testing.T) {
	isolateCache(t)
	directory := writeSources(t, map[string]string{
		"main.ahd": "name: String := take(\"Name: \")\nwrite(\"Hello {name}\")\nvalues: List<Int> := [3, 1]\nvalues.sort()\nwrite(values)\nattempt {\n    write(1 / 0)\n} except DivisionByZeroError as error {\n    write(error.message)\n}\n",
	})
	for round := 0; round < 3; round++ {
		out, code := runSource(t, directory, "main.ahd", "Ali\n")
		expected := "Name: Hello Ali\n[1, 3]\ndivision by zero\n"
		if out != expected || code != 0 {
			t.Fatalf("round %d = %q (exit %d), want %q", round, out, code, expected)
		}
	}

	failing := writeSources(t, map[string]string{"main.ahd": "write(\"before\")\ntoss(ValueError(\"boom\"))\n"})
	for round := 0; round < 2; round++ {
		var stdout, stderr bytes.Buffer
		code, result := RunProgramIO(filepath.Join(failing, "main.ahd"), nil, strings.NewReader(""), &stdout, &stderr)
		if result.HasErrors() {
			t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
		}
		if code != 1 || stdout.String() != "before\n" || !strings.HasPrefix(stderr.String(), "ValueError: ") {
			t.Fatalf("round %d: exit %d stdout %q stderr %q", round, code, stdout.String(), stderr.String())
		}
	}
}

// TestRunCacheLeavesNoTemporaryWorkspaceBehind checks that a cache hit does no
// filesystem work at all beyond running the executable.
func TestRunCacheLeavesNoTemporaryWorkspaceBehind(t *testing.T) {
	isolateCache(t)
	directory := writeSources(t, map[string]string{"main.ahd": "write(\"hello\")\n"})
	runSource(t, directory, "main.ahd", "")

	before := temporaryWorkspaceCount(t)
	runSource(t, directory, "main.ahd", "")
	if after := temporaryWorkspaceCount(t); after != before {
		t.Fatalf("a cache hit created %d temporary workspaces", after-before)
	}
}

func temporaryWorkspaceCount(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "ahdcode-build-") {
			count++
		}
	}
	return count
}
