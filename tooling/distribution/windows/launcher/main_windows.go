// Command ahdcode is the stable Windows launcher installed at
// %LOCALAPPDATA%\AhdCode\bin\ahdcode.exe.
//
// It is a real executable rather than a batch shim so `where.exe ahdcode`
// resolves it, and so editors and build tools that call CreateProcess directly
// can start it. It reads the active version from current.txt and runs that
// version's ahdcode.exe with the same arguments and the same standard handles,
// which keeps `ahdcode lsp --stdio` transparent. Because the real program runs
// from its own version folder, it still finds libexec\go and the bundled
// helpers next to itself.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	"ahdcode/tooling/distribution/winsetup"
)

func main() {
	target, err := resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ahdcode:", err)
		fmt.Fprintln(os.Stderr, "Run the AhdCode setup program again to repair the installation.")
		os.Exit(1)
	}
	command := exec.Command(target, os.Args[1:]...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	// Console signals reach every process attached to the console. The child
	// owns the interaction, so the launcher waits instead of exiting first.
	signal.Ignore(os.Interrupt)
	if err := command.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "ahdcode:", err)
		os.Exit(1)
	}
}

func resolve() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	root := filepath.Dir(filepath.Dir(executable))
	pointer, err := os.ReadFile(filepath.Join(root, winsetup.LauncherPointerName))
	if err != nil {
		return "", errors.New("no active AhdCode version is recorded")
	}
	version, err := winsetup.ReadPointer(pointer)
	if err != nil {
		return "", err
	}
	target := winsetup.ActiveExecutable(root, version)
	if information, err := os.Stat(target); err != nil || !information.Mode().IsRegular() {
		return "", fmt.Errorf("AhdCode %s is recorded as active but is not installed", version)
	}
	return target, nil
}
