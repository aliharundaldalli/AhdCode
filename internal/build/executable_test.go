package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 1 and 6: Windows generated executables carry .exe; Unix naming is unchanged.
func TestExecutablePathPerPlatform(t *testing.T) {
	cases := []struct {
		goos, in, want string
	}{
		{"windows", `C:\cache\run-v1\partial-1383540727`, `C:\cache\run-v1\partial-1383540727.exe`},
		{"windows", `C:\cache\run-v1\partial-1383540727.exe`, `C:\cache\run-v1\partial-1383540727.exe`},
		{"windows", `C:\cache\run-v1\partial-1383540727.EXE`, `C:\cache\run-v1\partial-1383540727.EXE`},
		{"windows", "hello", "hello.exe"},
		{"darwin", "/tmp/run-v1/partial-123", "/tmp/run-v1/partial-123"},
		{"darwin", "hello", "hello"},
		{"linux", "/tmp/ahdcode-program", "/tmp/ahdcode-program"},
		{"windows", "", ""},
		{"darwin", "", ""},
	}
	for _, c := range cases {
		if got := executablePathFor(c.goos, c.in); got != c.want {
			t.Errorf("executablePathFor(%q, %q) = %q, want %q", c.goos, c.in, got, c.want)
		}
	}
}

// Applying the policy twice must never produce partial-1.exe.exe.
func TestExecutablePathIsIdempotent(t *testing.T) {
	for _, goos := range []string{"windows", "darwin", "linux"} {
		once := executablePathFor(goos, "build/out")
		if twice := executablePathFor(goos, once); twice != once {
			t.Errorf("%s: applying twice gave %q, want %q", goos, twice, once)
		}
	}
}

// 5: a Windows project directory containing spaces must survive naming. The
// build and launch APIs take argument slices, so no quoting is involved.
func TestExecutablePathKeepsSpacesAndDotsIntact(t *testing.T) {
	windows := `C:\Users\aliha\Downloads\zrd hbv\OneDrive\Belgeler\PROJE\deneme`
	got := executablePathFor("windows", windows)
	if got != windows+".exe" {
		t.Fatalf("path with spaces = %q", got)
	}
	if !strings.Contains(got, "zrd hbv") || !strings.Contains(got, "OneDrive") {
		t.Fatalf("path with spaces was rewritten: %q", got)
	}
	// A dotted directory must not be mistaken for an extension on the file.
	dotted := executablePathFor("windows", `C:\src\v1.0\app`)
	if dotted != `C:\src\v1.0\app.exe` {
		t.Fatalf("dotted directory = %q", dotted)
	}
	// A file that genuinely ends in another extension still gains .exe, so the
	// launcher and the artifact continue to agree.
	if other := executablePathFor("windows", `C:\src\app.bin`); other != `C:\src\app.bin.exe` {
		t.Fatalf("app.bin = %q", other)
	}
}

// 3: os.CreateTemp must keep the suffix last so each reservation is a distinct,
// launchable name.
func TestExecutableTempPattern(t *testing.T) {
	if got := executableTempPatternFor("windows", "partial-"); got != "partial-*.exe" {
		t.Fatalf("windows pattern = %q", got)
	}
	if got := executableTempPatternFor("darwin", "partial-"); got != "partial-" {
		t.Fatalf("darwin pattern = %q", got)
	}
	directory := t.TempDir()
	seen := map[string]bool{}
	for i := 0; i < 5; i++ {
		file, err := os.CreateTemp(directory, executableTempPatternFor("windows", "partial-"))
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(file.Name())
		file.Close()
		if !strings.HasPrefix(name, "partial-") || !strings.HasSuffix(name, ".exe") {
			t.Fatalf("reserved name %q does not match partial-*.exe", name)
		}
		if seen[name] {
			t.Fatalf("reservation %q was handed out twice", name)
		}
		seen[name] = true
	}
}

// Windows reports no execute bit for any regular file, so a Unix-style 0111
// test rejects every freshly built executable there.
func TestRunnableModePerPlatform(t *testing.T) {
	cases := []struct {
		goos string
		mode fs.FileMode
		want bool
	}{
		{"windows", 0o666, true},
		{"windows", 0o444, true},
		{"windows", fs.ModeDir | 0o777, false},
		{"windows", fs.ModeSymlink | 0o777, false},
		{"darwin", 0o666, false},
		{"darwin", 0o755, true},
		{"linux", 0o644, false},
		{"linux", 0o700, true},
		{"linux", fs.ModeDir | 0o755, false},
	}
	for _, c := range cases {
		if got := runnableModeFor(c.goos, c.mode); got != c.want {
			t.Errorf("runnableModeFor(%q, %v) = %v, want %v", c.goos, c.mode, got, c.want)
		}
	}
}

// 2 and 4: the cache names the entry it reserves, publishes and later finds
// with one policy, so the launcher path always matches a file that exists and
// cleanup targets that same file.
func TestRunCacheNamesOneConsistentArtifact(t *testing.T) {
	directory := t.TempDir()
	cache := &runCache{directory: directory}

	reserved, ok := cache.reserve()
	if !ok {
		t.Fatal("reserve failed")
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(reserved, ".exe") {
		t.Fatalf("reserved %q has no .exe", reserved)
	}
	if got := ExecutablePath(reserved); got != reserved {
		t.Fatalf("reserved path is not already canonical: %q vs %q", reserved, got)
	}
	if _, err := os.Stat(reserved); err == nil {
		t.Fatal("reserve left a placeholder file behind")
	}

	// Stand in for `go build -o reserved`.
	if err := os.WriteFile(reserved, []byte("native program"), 0o755); err != nil {
		t.Fatal(err)
	}
	key := strings.Repeat("a", 64)
	published, ok := cache.publish(key, reserved)
	if !ok {
		t.Fatal("publish failed for a freshly built executable")
	}
	if published != cache.path(key) {
		t.Fatalf("published %q, want %q", published, cache.path(key))
	}
	if !runnableFile(published) {
		t.Fatalf("published entry %q is not runnable", published)
	}
	if _, err := os.Stat(reserved); err == nil {
		t.Fatal("the reserved name still exists after publishing")
	}
	found, hit := cache.lookup(key)
	if !hit {
		t.Fatal("lookup missed an entry that was just published")
	}
	if found != published {
		t.Fatalf("lookup returned %q, want %q", found, published)
	}
}

// A refused publish must leave the built executable exactly where the caller
// still expects to find it: this is what turned a cache problem into an
// unrunnable path on Windows.
func TestPublishFailureLeavesTheBuiltExecutableInPlace(t *testing.T) {
	directory := t.TempDir()
	cache := &runCache{directory: filepath.Join(directory, "cache")}
	if err := os.MkdirAll(cache.directory, 0o700); err != nil {
		t.Fatal(err)
	}
	built := ExecutablePath(filepath.Join(directory, "built"))
	if err := os.WriteFile(built, []byte("native program"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A key that cannot be a filename makes the rename fail.
	if _, ok := cache.publish(strings.Repeat("b", 64)+string(filepath.Separator)+"nope", built); ok {
		t.Fatal("publish reported success for an impossible destination")
	}
	if !runnableFile(built) {
		t.Fatal("a refused publish moved or destroyed the built executable")
	}
}

// 8 and 9: a launch failure must not be reported as a disk-space problem, and
// a missing artifact must read differently from one the system refused to run.
func TestLaunchFailureDiagnosticsAreAccurate(t *testing.T) {
	directory := t.TempDir()
	present := filepath.Join(directory, "present")
	if err := os.WriteFile(present, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	refused := launchFailure(present, os.ErrPermission)
	if strings.Contains(refused.Hint, "disk space") {
		t.Errorf("a refused launch still mentions disk space: %q", refused.Hint)
	}
	if !strings.Contains(refused.Message, present) {
		t.Errorf("the message does not name the executable: %q", refused.Message)
	}
	if !strings.Contains(refused.Hint, "refused to start") {
		t.Errorf("unexpected hint for a present executable: %q", refused.Hint)
	}

	missing := launchFailure(filepath.Join(directory, "absent"), os.ErrNotExist)
	if strings.Contains(missing.Hint, "disk space") {
		t.Errorf("a missing artifact still mentions disk space: %q", missing.Hint)
	}
	if !strings.Contains(missing.Hint, "left no executable") {
		t.Errorf("unexpected hint for a missing executable: %q", missing.Hint)
	}
	if refused.Hint == missing.Hint {
		t.Error("a refused launch and a missing artifact report the same hint")
	}
	// The workspace diagnostic keeps its own hint, which is right for the
	// problems it actually reports.
	if !strings.Contains(workspaceFailure("x").Hint, "disk space") {
		t.Error("workspaceFailure lost its hint")
	}
}

// 7: run, dev, databases and build all name their executable through the same
// helper, so the policy cannot drift between commands.
func TestDefaultOutputPathFollowsThePolicy(t *testing.T) {
	got := DefaultOutputPath(filepath.Join("some", "where", "deneme.ahd"))
	if got != ExecutablePath(got) {
		t.Fatalf("DefaultOutputPath returned a non-canonical path: %q", got)
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(got, "deneme.exe") {
		t.Fatalf("windows default output = %q", got)
	}
	if runtime.GOOS != "windows" && !strings.HasSuffix(got, "deneme") {
		t.Fatalf("unix default output = %q", got)
	}
}
