package winsetup

import (
	"path/filepath"
	"strings"
	"testing"
)

const bin = `C:\Users\someone\AppData\Local\AhdCode\bin`

func expandLocalAppData(value string) string {
	return strings.ReplaceAll(value, `%LOCALAPPDATA%`, `C:\Users\someone\AppData\Local`)
}

func TestCheckEntryNameAcceptsOrdinaryPayloadPaths(t *testing.T) {
	for _, name := range []string{
		"VERSION",
		"bin/ahdcode.exe",
		"libexec/go/bin/go.exe",
		"libexec/ahdcode/latex/ahdcode-latex.ttb",
		"licenses/modules/golang.org_x_text@v0.41.0/LICENSE",
	} {
		if err := CheckEntryName(name); err != nil {
			t.Errorf("CheckEntryName(%q) = %v, want nil", name, err)
		}
	}
}

func TestCheckEntryNameRejectsEscapes(t *testing.T) {
	for _, name := range []string{
		"",
		"../outside.txt",
		"bin/../../outside.txt",
		`..\outside.txt`,
		`bin\..\..\outside.txt`,
		"/absolute/path",
		`\rooted\path`,
		`C:\Windows\System32\evil.dll`,
		"bin//ahdcode.exe",
		"bin/./ahdcode.exe",
	} {
		if err := CheckEntryName(name); err == nil {
			t.Errorf("CheckEntryName(%q) = nil, want an error", name)
		}
	}
}

func TestCheckVersion(t *testing.T) {
	if err := CheckVersion("1.0.0-rc.1"); err != nil {
		t.Fatalf("CheckVersion(release) = %v, want nil", err)
	}
	for _, version := range []string{"", "  ", ".", "..", `1.0\..\..`, "1.0/2.0", "C:1.0"} {
		if err := CheckVersion(version); err == nil {
			t.Errorf("CheckVersion(%q) = nil, want an error", version)
		}
	}
}

func TestReadPointerTrimsWindowsLineEndings(t *testing.T) {
	version, err := ReadPointer([]byte("1.0.0-rc.1\r\n"))
	if err != nil {
		t.Fatalf("ReadPointer = %v", err)
	}
	if version != "1.0.0-rc.1" {
		t.Fatalf("ReadPointer = %q, want %q", version, "1.0.0-rc.1")
	}
	if _, err := ReadPointer([]byte("  \r\n")); err == nil {
		t.Fatal("ReadPointer(blank) = nil, want an error")
	}
}

// Scenario 4: a PATH that does not mention AhdCode gains exactly one entry.
func TestAddPathEntryAppendsOnce(t *testing.T) {
	raw := `C:\Windows\system32;C:\Windows;C:\Program Files\Git\cmd`
	updated, changed := AddPathEntry(raw, bin, expandLocalAppData)
	if !changed {
		t.Fatal("AddPathEntry reported no change on a PATH without AhdCode")
	}
	if updated != raw+";"+bin {
		t.Fatalf("AddPathEntry = %q", updated)
	}
	if strings.Count(updated, bin) != 1 {
		t.Fatalf("AhdCode appears %d times", strings.Count(updated, bin))
	}
}

// Scenarios 5 and 6: an existing entry is recognized and never duplicated.
func TestAddPathEntryIsIdempotent(t *testing.T) {
	for _, existing := range []string{
		bin,
		bin + `\`,
		strings.ToUpper(bin),
		`"` + bin + `"`,
		`%LOCALAPPDATA%\AhdCode\bin`,
		strings.ReplaceAll(bin, `\`, "/"),
	} {
		raw := `C:\Windows\system32;` + existing + `;C:\Program Files\Git\cmd`
		updated, changed := AddPathEntry(raw, bin, expandLocalAppData)
		if changed {
			t.Errorf("AddPathEntry duplicated an existing entry %q", existing)
		}
		if updated != raw {
			t.Errorf("AddPathEntry rewrote PATH for %q:\n got %q\nwant %q", existing, updated, raw)
		}
	}
}

// Scenario 7: nothing else about the user's PATH may change.
func TestAddPathEntryPreservesUnrelatedEntries(t *testing.T) {
	raw := `C:\Windows\system32;;C:\Program Files (x86)\Odd; C:\spaced ;%JAVA_HOME%\bin`
	updated, changed := AddPathEntry(raw, bin, expandLocalAppData)
	if !changed {
		t.Fatal("expected a change")
	}
	if !strings.HasPrefix(updated, raw+";") {
		t.Fatalf("unrelated entries were rewritten:\n got %q\nwant prefix %q", updated, raw+";")
	}
	before := SplitPath(raw)
	after := SplitPath(updated)
	if len(after) != len(before)+1 {
		t.Fatalf("entry count %d, want %d", len(after), len(before)+1)
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("entry %d changed from %q to %q", i, before[i], after[i])
		}
	}
}

func TestAddPathEntryHandlesEmptyAndTrailingSemicolon(t *testing.T) {
	if updated, changed := AddPathEntry("", bin, nil); !changed || updated != bin {
		t.Fatalf("empty PATH: got %q changed=%v", updated, changed)
	}
	raw := `C:\Windows\system32;`
	if updated, _ := AddPathEntry(raw, bin, nil); updated != raw+bin {
		t.Fatalf("trailing semicolon: got %q", updated)
	}
}

func TestRemovePathEntryKeepsEverythingElse(t *testing.T) {
	raw := `C:\Windows\system32;` + bin + `\;C:\Program Files\Git\cmd`
	updated, changed := RemovePathEntry(raw, bin, expandLocalAppData)
	if !changed {
		t.Fatal("expected removal")
	}
	if updated != `C:\Windows\system32;C:\Program Files\Git\cmd` {
		t.Fatalf("RemovePathEntry = %q", updated)
	}
	if _, changed := RemovePathEntry(`C:\Windows\system32`, bin, nil); changed {
		t.Fatal("RemovePathEntry changed a PATH that never had AhdCode")
	}
}

func TestPathHelpersNeverTruncate(t *testing.T) {
	raw := strings.Join([]string{`C:\a`, `C:\b`, `C:\c`, bin, `C:\d`}, ";")
	updated, changed := AddPathEntry(raw, bin, nil)
	if changed || updated != raw {
		t.Fatalf("AddPathEntry mutated a satisfied PATH: %q", updated)
	}
	removed, _ := RemovePathEntry(raw, bin, nil)
	restored, _ := AddPathEntry(removed, bin, nil)
	if len(SplitPath(restored)) != len(SplitPath(raw)) {
		t.Fatalf("round trip lost entries: %q", restored)
	}
}

// Scenario 8: the launcher must resolve a real executable inside the version
// folder, which is also where libexec sits relative to it.
func TestActiveExecutableIsInsideTheVersionFolder(t *testing.T) {
	root := filepath.Join("C:", "Users", "someone", "AppData", "Local", "AhdCode")
	got := ActiveExecutable(root, "1.0.0-rc.1")
	want := filepath.Join(root, "versions", "1.0.0-rc.1", "bin", "ahdcode.exe")
	if got != want {
		t.Fatalf("ActiveExecutable = %q, want %q", got, want)
	}
	if filepath.Base(got) != "ahdcode.exe" {
		t.Fatalf("the launcher must run a real executable, got %q", filepath.Base(got))
	}
	// libexec must be the sibling of the resolved bin folder, which is what
	// makes the private Go toolchain and the bundled helpers discoverable.
	libexec := filepath.Join(filepath.Dir(filepath.Dir(got)), "libexec")
	if libexec != filepath.Join(root, "versions", "1.0.0-rc.1", "libexec") {
		t.Fatalf("libexec resolves to %q", libexec)
	}
}

// The folder placed on PATH must not depend on the version, so an upgrade
// never has to edit PATH again.
func TestLauncherDirectoryIsVersionIndependent(t *testing.T) {
	root := filepath.Join("C:", "Users", "someone", "AppData", "Local", "AhdCode")
	directory := LauncherDirectory(root)
	if directory != filepath.Join(root, "bin") {
		t.Fatalf("LauncherDirectory = %q", directory)
	}
	raw := ""
	for _, version := range []string{"1.0.0-rc.1", "1.0.0", "1.1.0"} {
		if ActiveExecutable(root, version) == "" {
			t.Fatal("empty target")
		}
		updated, _ := AddPathEntry(raw, LauncherDirectory(root), nil)
		raw = updated
	}
	if strings.Count(raw, directory) != 1 {
		t.Fatalf("three upgrades produced %d PATH entries: %q", strings.Count(raw, directory), raw)
	}
}
