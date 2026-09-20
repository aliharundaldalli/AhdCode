package build

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The v2.0.0 Plot Save defect only appeared in an *installed* AhdCode: the
// compiler recorded the ahdgui location only for a program that used GUI
// itself, so a Plot-only program compiled into a temporary directory had no
// way to find the helper and the viewer's Save reported it as not
// installed. The unit test beside this one checks which programs carry the
// hint; this one checks the layout the defect actually appeared in.
//
// It builds a real ahdcode into an install-shaped tree
// (<root>/bin/ahdcode, <root>/libexec/ahdcode/ahdgui), then compiles a
// Plot-only program from a working directory that is not the repository,
// with no AHDCODE_GUI_RUNTIME and a PATH that holds nothing, and checks
// that the built program carries the installed helper's directory.

const plotOnlyInstallProgram = `bring Plot
from Plot bring Chart

chart: Chart := Plot.line([1.0, 2.0, 3.0], [2.0, 4.0, 9.0]).title("Save")
chart.save("chart.png")
`

// stageInstallTree builds ahdcode into <root>/bin and puts stub helpers
// around it in the two places an installation keeps them.
//
// ahdgui goes in <root>/libexec/ahdcode **alone**, and every other helper in
// <root>/bin, on purpose: the Plot viewer's own hint (ahdplotview) would
// otherwise record the same directory and the test could not tell which hint
// put it in the program. With them apart, the libexec path can only come
// from the ahdgui hint -- the one the v2.0.0 defect was missing.
func stageInstallTree(t *testing.T) (root, ahdcode, dialogHelperDirectory string) {
	t.Helper()
	if testing.Short() {
		t.Skip("staging an install tree builds the compiler")
	}
	root = t.TempDir()
	bin := filepath.Join(root, "bin")
	dialogHelperDirectory = filepath.Join(root, "libexec", "ahdcode")
	for _, directory := range []string{bin, dialogHelperDirectory} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ahdcode = filepath.Join(bin, "ahdcode")
	if runtime.GOOS == "windows" {
		ahdcode += ".exe"
	}
	build := exec.Command("go", "build", "-o", ahdcode, "ahdcode/cmd/ahdcode")
	build.Dir = repositoryRoot(t)
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("could not build ahdcode: %v\n%s", err, output)
	}
	// The helpers are stubs: only their presence and location matter to the
	// compiler, which records where they are rather than running them.
	stub := func(directory, name string) {
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if err := os.WriteFile(filepath.Join(directory, name), []byte("stub"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	stub(dialogHelperDirectory, "ahdgui")
	stub(bin, "ahdplotview")
	stub(bin, "ahdplot")
	return root, ahdcode, dialogHelperDirectory
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// internal/build -> the repository root.
	return filepath.Dir(filepath.Dir(working))
}

// TestPlotOnlyProgramFindsTheInstalledDialogHelperAfterRelocation is the
// acceptance shape of the v2.0.0 Save defect: an installed, relocated
// AhdCode, a Plot-only program, no environment override, and a working
// directory nowhere near the source checkout.
func TestPlotOnlyProgramFindsTheInstalledDialogHelperAfterRelocation(t *testing.T) {
	_, ahdcode, dialogHelperDirectory := stageInstallTree(t)

	working := t.TempDir()
	entry := filepath.Join(working, "chart.ahd")
	if err := os.WriteFile(entry, []byte(plotOnlyInstallProgram), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(working, "chart")

	command := exec.Command(ahdcode, "build", entry, "-o", output)
	command.Dir = working
	// A hostile environment: no override, and nothing useful on PATH. The
	// compiler must find the helper beside itself, not through either.
	command.Env = append(environmentWithout(os.Environ(),
		"AHDCODE_GUI_RUNTIME", "AHDCODE_PLOTVIEW_RUNTIME", "AHDCODE_PLOT_RUNTIME", "AHDCODE_ROOT"),
		"PATH="+t.TempDir(), "HOME="+t.TempDir())
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("installed ahdcode could not build a Plot program: %v\n%s", err, combined)
	}

	built, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	// The hint is a Go string constant compiled into the program, so the
	// installed helper's directory is in the executable. Before the fix a
	// Plot-only program carried no hint at all and this was absent.
	if !bytes.Contains(built, []byte(dialogHelperDirectory)) {
		t.Fatalf("a Plot-only program built by an installed ahdcode does not name the dialog helper's directory %s;"+
			" the viewer's Save would report the GUI helper as not installed", dialogHelperDirectory)
	}
}

// A program that uses neither Plot nor GUI must still carry no hint, so the
// fix did not make the helper unconditional.
func TestPlainProgramStillCarriesNoDialogHelperHintWhenInstalled(t *testing.T) {
	_, ahdcode, dialogHelperDirectory := stageInstallTree(t)

	working := t.TempDir()
	entry := filepath.Join(working, "plain.ahd")
	if err := os.WriteFile(entry, []byte("write(\"no helper\")\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(working, "plain")
	command := exec.Command(ahdcode, "build", entry, "-o", output)
	command.Dir = working
	command.Env = append(environmentWithout(os.Environ(),
		"AHDCODE_GUI_RUNTIME", "AHDCODE_PLOTVIEW_RUNTIME", "AHDCODE_PLOT_RUNTIME", "AHDCODE_ROOT"),
		"PATH="+t.TempDir(), "HOME="+t.TempDir())
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("installed ahdcode could not build a plain program: %v\n%s", err, combined)
	}
	built, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(built, []byte(dialogHelperDirectory)) {
		t.Fatal("a program that uses neither Plot nor GUI names the dialog helper's directory")
	}
}

// environmentWithout returns environment with the named variables removed.
func environmentWithout(environment []string, names ...string) []string {
	remaining := make([]string, 0, len(environment))
	for _, entry := range environment {
		drop := false
		for _, name := range names {
			if strings.HasPrefix(entry, name+"=") {
				drop = true
				break
			}
		}
		if !drop {
			remaining = append(remaining, entry)
		}
	}
	return remaining
}
