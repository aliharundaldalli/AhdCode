package build

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildWebProgramOnce compiles one entry to a native executable and caches
// nothing: callers that need several environments reuse the returned path
// rather than paying for a second compile.
func buildWebProgramOnce(t *testing.T, directory string) string {
	t.Helper()
	output := filepath.Join(t.TempDir(), "program")
	path, result := BuildProgram(filepath.Join(directory, "main.ahd"), output)
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	return path
}

// webProgramBinaries memoizes one built executable per source directory, so a
// table-driven environment test compiles once instead of once per case.
var webProgramBinaries = map[string]string{}

func webProgramFor(t *testing.T, directory string) string {
	t.Helper()
	if path, built := webProgramBinaries[directory]; built {
		return path
	}
	path := buildWebProgramOnce(t, directory)
	webProgramBinaries[directory] = path
	return path
}

// runWithEnvironment runs one already-built program with exactly the given
// variables, plus PATH and HOME. Starting from an explicit set rather than
// the test process's own environment keeps a stray APP_ENV on the developer's
// machine from changing what these tests assert.
func runWithEnvironment(t *testing.T, path string, variables map[string]string) (string, string, int) {
	t.Helper()
	command := exec.Command(path)
	command.Dir = filepath.Dir(path)
	environment := []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME")}
	for key, value := range variables {
		environment = append(environment, key+"="+value)
	}
	command.Env = environment
	var out, errorOutput strings.Builder
	command.Stdout = &out
	command.Stderr = &errorOutput
	code := 0
	if runError := command.Run(); runError != nil {
		var exit *exec.ExitError
		if !errors.As(runError, &exit) {
			t.Fatalf("could not run the program: %v", runError)
		}
		code = exit.ExitCode()
	}
	return out.String(), errorOutput.String(), code
}

func runWebEnvProgram(t *testing.T, directory string, variables map[string]string) string {
	t.Helper()
	out, errorOutput, code := runWithEnvironment(t, webProgramFor(t, directory), variables)
	if code != 0 {
		t.Fatalf("program exited with %d\nstderr:\n%s", code, errorOutput)
	}
	return out
}

func runWebEnvProgramExpectingFailure(t *testing.T, directory string, variables map[string]string) (string, string) {
	t.Helper()
	out, errorOutput, code := runWithEnvironment(t, webProgramFor(t, directory), variables)
	if code == 0 {
		t.Fatalf("expected the program to fail; it succeeded with:\n%s", out)
	}
	return out, errorOutput
}
