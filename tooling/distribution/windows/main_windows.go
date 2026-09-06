// AhdCode's per-user Windows setup program. No system Go, Git, or elevation is
// needed, and no console interaction is required: double-clicking the file in
// Explorer shows a confirmation dialog, a progress window, and a result dialog.
package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"ahdcode/tooling/distribution/winsetup"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed payload.zip
var payload embed.FS

const setupCaption = "AhdCode Setup"

// reporter receives progress. The graphical and console front ends both
// implement it so the installation itself has no idea which one is in use.
type reporter interface {
	Update(status string, percent int)
}

type consoleReporter struct {
	out  io.Writer
	last string
}

func (c *consoleReporter) Update(status string, percent int) {
	if c.out == nil || status == c.last {
		return
	}
	c.last = status
	fmt.Fprintf(c.out, "  %3d%%  %s\n", percent, status)
}

type nopReporter struct{}

func (nopReporter) Update(string, int) {}

func main() {
	silent := false
	for _, argument := range os.Args[1:] {
		switch argument {
		case "--silent", "/S", "/silent":
			silent = true
		case "--help", "/?", "-h":
			console := attachParentConsole()
			if console == nil {
				infoDialog("Usage:\n\n  AhdCode Setup            install with a graphical prompt\n  AhdCode Setup --silent   install without any prompt", setupCaption)
				return
			}
			fmt.Fprintln(console, "usage: AhdCode setup [--silent]")
			return
		}
	}

	console := attachParentConsole()
	if console != nil || silent {
		// Started from a shell, or asked to be quiet: no dialogs, no prompts,
		// and above all no waiting on stdin.
		var report reporter = nopReporter{}
		if console != nil && !silent {
			fmt.Fprintf(console, "%s\nInstalling into %s\n", setupCaption, installationRoot())
			report = &consoleReporter{out: console}
		}
		if err := install(report); err != nil {
			if console != nil {
				fmt.Fprintln(console, "AhdCode setup failed:", friendly(err))
			}
			os.Exit(1)
		}
		if console != nil && !silent {
			fmt.Fprintln(console, successMessage())
		}
		return
	}

	runGraphical()
}

// runGraphical is the Explorer double-click experience: one confirmation
// click, a progress window, then one dismissal click.
func runGraphical() {
	// Every window is created and pumped on one OS thread, because Windows
	// delivers messages only to the thread that owns the window.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	root := installationRoot()
	prompt := "AhdCode will be installed for your account only.\n\n" +
		"Location:\n" + root + "\n\n" +
		"Setup adds only AhdCode's own bin folder to your user PATH.\n" +
		"Your projects, databases and settings are not touched.\n" +
		"Administrator rights are not required.\n\n" +
		"Click OK to install."
	if !confirmDialog(prompt, setupCaption) {
		return
	}

	window := newProgressWindow(setupCaption)
	result := make(chan error, 1)
	go func() {
		defer window.Done()
		result <- install(window)
	}()
	window.Pump()

	if err := <-result; err != nil {
		errorDialog("AhdCode was not installed.\n\n"+friendly(err)+"\n\nNothing was left half-installed; you can run this setup again.", setupCaption)
		os.Exit(1)
	}
	infoDialog(successMessage(), setupCaption)
}

func successMessage() string {
	return "Installation complete.\n\nOpen a new terminal and run:\n\n    ahdcode --version"
}

// friendly turns the failures a user can actually hit into plain language and
// keeps everything else short, rather than showing a Go error chain.
func friendly(err error) string {
	text := err.Error()
	switch {
	case errors.Is(err, os.ErrPermission):
		return "Windows refused access to a file in your AppData folder.\n" +
			"Close any running AhdCode program or terminal and try again.\n\nDetails: " + text
	case errors.Is(err, winsetup.ErrUnsafeEntry):
		return "This setup file is damaged or incomplete. Download it again."
	case strings.Contains(text, "no space"):
		return "There is not enough free disk space to install AhdCode."
	}
	return text
}

func installationRoot() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		if directory, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0); err == nil {
			local = directory
		}
	}
	return filepath.Join(local, "AhdCode")
}

func install(report reporter) error {
	root := installationRoot()
	if !filepath.IsAbs(root) || strings.TrimSpace(root) == "" {
		return errors.New("your Windows user profile location could not be determined")
	}
	if information, err := os.Lstat(root); err == nil {
		if information.Mode()&os.ModeSymlink != 0 {
			return errors.New("the AhdCode folder is a link; remove it and run setup again")
		}
		if _, err := os.Stat(filepath.Join(root, ".ahdcode-install")); err != nil {
			return errors.New("a folder named AhdCode already exists in your AppData and was not created by this setup; rename it and run setup again")
		}
	}

	report.Update("Reading the setup payload...", 2)
	data, err := payload.ReadFile("payload.zip")
	if err != nil {
		return err
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	version, err := winsetup.PayloadVersion(archive)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, ".ahdcode-install"), []byte("AhdCode installer v1\r\n"), 0o644); err != nil {
		return err
	}

	staging, err := os.MkdirTemp(filepath.Join(root, "versions"), ".setup-")
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(staging)
		}
	}()

	// The inventory is checked while each file is written, so the payload is
	// read once instead of twice. On a machine with real-time virus scanning a
	// second full pass over ~380 MB was most of the wait.
	inventory, err := winsetup.ReadInventory(archive)
	if err != nil {
		return err
	}
	err = winsetup.Extract(archive, staging, inventory, func(written, total int) {
		percent := 90
		if total > 0 {
			percent = 5 + written*85/total
		}
		report.Update(fmt.Sprintf("Installing files... (%d of %d)", written, total), percent)
	})
	if err != nil {
		return err
	}

	report.Update("Activating this version...", 92)
	target := filepath.Join(root, "versions", version)
	previous := ""
	if _, err := os.Stat(target); err == nil {
		previous = target + ".replaced-" + fmt.Sprint(time.Now().UnixNano())
		if err := os.Rename(target, previous); err != nil {
			return fmt.Errorf("the installed copy of %s is in use; close AhdCode and run setup again", version)
		}
	}
	if err := os.Rename(staging, target); err != nil {
		if previous != "" {
			os.Rename(previous, target)
		}
		return err
	}
	committed = true
	if previous != "" {
		os.RemoveAll(previous)
	}

	report.Update("Registering the ahdcode command...", 95)
	if err := installLauncher(root, target, version); err != nil {
		return err
	}
	if err := registerUninstall(root, version); err != nil {
		return err
	}

	report.Update("Updating your PATH...", 97)
	if err := addToUserPath(winsetup.LauncherDirectory(root)); err != nil {
		return err
	}
	broadcastEnvironmentChange()

	report.Update("Checking the installation...", 99)
	if err := verify(root); err != nil {
		return err
	}
	report.Update("Done.", 100)
	return nil
}

// installLauncher puts a real ahdcode.exe in a fixed folder so PATH is written
// once and never has to change again. The launcher reads current.txt and runs
// the active version, which is how the macOS and Linux packages use their
// `current` symlink.
func installLauncher(root, target, version string) error {
	bin := winsetup.LauncherDirectory(root)
	if err := os.MkdirAll(bin, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, winsetup.LauncherPointerName), []byte(version+"\r\n"), 0o644); err != nil {
		return err
	}
	content, err := os.ReadFile(filepath.Join(target, "launcher", "ahdcode.exe"))
	if err != nil {
		return errors.New("this setup file is damaged: the ahdcode launcher is missing")
	}
	destination := filepath.Join(bin, "ahdcode.exe")
	temporary := destination + ".new"
	if err := os.WriteFile(temporary, content, 0o755); err != nil {
		return err
	}
	os.Remove(destination)
	if err := os.Rename(temporary, destination); err != nil {
		os.Remove(temporary)
		return errors.New("ahdcode.exe is running; close it and run setup again")
	}
	// Earlier releases installed a batch shim next to it. It is AhdCode's own
	// file, and leaving two launchers behind would only be confusing.
	os.Remove(filepath.Join(bin, "ahdcode.cmd"))

	if script, err := os.ReadFile(filepath.Join(target, "uninstall.ps1")); err == nil {
		os.WriteFile(filepath.Join(root, "uninstall.ps1"), script, 0o644)
	}
	return nil
}

func registerUninstall(root, version string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall\AhdCode`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	values := map[string]string{
		"DisplayName":     "AhdCode",
		"DisplayVersion":  version,
		"Publisher":       "AhdCode",
		"InstallLocation": root,
		"UninstallString": `powershell.exe -NoProfile -ExecutionPolicy Bypass -File "` + filepath.Join(root, "uninstall.ps1") + `"`,
	}
	for name, value := range values {
		if err := key.SetStringValue(name, value); err != nil {
			return err
		}
	}
	return nil
}

// addToUserPath appends AhdCode's bin folder to the per-user PATH exactly once.
// The existing value is edited, never rebuilt, and its registry type is kept,
// so unrelated entries survive untouched.
func addToUserPath(bin string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("your user environment could not be opened for writing: %w", err)
	}
	defer key.Close()

	current, kind, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("your PATH could not be read: %w", err)
	}
	updated, changed := winsetup.AddPathEntry(current, bin, expandEnvironment)
	if changed {
		if kind == registry.SZ {
			err = key.SetStringValue("Path", updated)
		} else {
			err = key.SetExpandStringValue("Path", updated)
		}
		if err != nil {
			return fmt.Errorf("your PATH could not be saved: %w", err)
		}
	}
	// Read the value back: a silent PATH failure is exactly the outcome this
	// setup must never report as success.
	stored, _, err := key.GetStringValue("Path")
	if err != nil {
		return fmt.Errorf("your PATH could not be confirmed: %w", err)
	}
	if !winsetup.HasPathEntry(stored, bin, expandEnvironment) {
		return errors.New("AhdCode could not be added to your PATH")
	}
	return nil
}

func expandEnvironment(value string) string {
	expanded, err := registry.ExpandString(value)
	if err != nil {
		return value
	}
	return expanded
}

// verify refuses to report success unless the command the user was promised
// actually exists and runs.
func verify(root string) error {
	launcher := filepath.Join(winsetup.LauncherDirectory(root), "ahdcode.exe")
	information, err := os.Stat(launcher)
	if err != nil || !information.Mode().IsRegular() {
		return errors.New("ahdcode.exe was not installed")
	}
	output, err := exec.Command(launcher, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("the installed ahdcode command did not run: %v", err)
	}
	if !strings.Contains(string(output), "AhdCode") {
		return errors.New("the installed ahdcode command did not report a version")
	}
	return nil
}
