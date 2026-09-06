// AhdCode's per-user Windows installer. No system Go, Git, or elevation is needed.
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

//go:embed payload.zip
var payload embed.FS

func main() {
	if err := install(); err != nil {
		fmt.Fprintln(os.Stderr, "AhdCode setup:", err)
		os.Exit(1)
	}
}
func install() error {
	silent := len(os.Args) > 1 && os.Args[1] == "--silent"
	root := filepath.Join(os.Getenv("LOCALAPPDATA"), "AhdCode")
	if !filepath.IsAbs(root) {
		return fmt.Errorf("LOCALAPPDATA must be an absolute path")
	}
	fmt.Println("AhdCode setup installs its private toolchain and runtime into", root)
	fmt.Println("It adds only its own bin directory to your user PATH. User projects and databases are preserved.")
	if !silent {
		fmt.Print("Press Enter to install, or close this window to cancel: ")
		if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
			return err
		}
	}
	if st, err := os.Lstat(root); err == nil {
		if st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("installation root is a link")
		}
		if _, err = os.Stat(filepath.Join(root, ".ahdcode-install")); err != nil {
			return fmt.Errorf("existing directory is not owned by AhdCode")
		}
	}
	data, err := payload.ReadFile("payload.zip")
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	version := ""
	for _, f := range reader.File {
		if f.Name == "VERSION" {
			r, e := f.Open()
			if e != nil {
				return e
			}
			b, e := io.ReadAll(r)
			r.Close()
			if e != nil {
				return e
			}
			version = strings.TrimSpace(string(b))
		}
	}
	if version == "" || strings.ContainsAny(version, "/\\:") {
		return fmt.Errorf("invalid release identity")
	}
	if err = os.MkdirAll(filepath.Join(root, "versions"), 0755); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(root, ".ahdcode-install"), []byte("AhdCode installer v1\n"), 0644); err != nil {
		return err
	}
	target := filepath.Join(root, "versions", version)
	if _, err = os.Stat(target); err == nil {
		return fmt.Errorf("this version is already installed")
	}
	staging, err := os.MkdirTemp(filepath.Join(root, "versions"), ".install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	for _, f := range reader.File {
		name := filepath.FromSlash(f.Name)
		if !filepath.IsLocal(name) || f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe payload entry")
		}
		destination := filepath.Join(staging, name)
		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(destination, 0755); err != nil {
				return err
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return err
		}
		r, e := f.Open()
		if e != nil {
			return e
		}
		w, e := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if e != nil {
			r.Close()
			return e
		}
		_, e = io.Copy(w, r)
		r.Close()
		ce := w.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
	}
	// Verify every payload byte against the release builder's inventory before activation.
	inventory := map[string]string{}
	b, err := os.ReadFile(filepath.Join(staging, "FILES.json"))
	if err != nil {
		return err
	}
	if err = json.Unmarshal(b, &inventory); err != nil {
		return err
	}
	for name, want := range inventory {
		b, e := os.ReadFile(filepath.Join(staging, filepath.FromSlash(name)))
		if e != nil {
			return e
		}
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != want {
			return fmt.Errorf("payload checksum mismatch: %s", name)
		}
	}
	if err = os.Rename(staging, target); err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Join(root, "bin"), 0755); err != nil {
		return err
	}
	// The stable CMD launcher resolves its version from a small pointer file.
	if err = os.WriteFile(filepath.Join(root, "current.txt"), []byte(version+"\r\n"), 0644); err != nil {
		return err
	}
	launcher := "@echo off\r\nsetlocal\r\nset /p AHD_VERSION=<\"%~dp0..\\current.txt\"\r\n\"%~dp0..\\versions\\%AHD_VERSION%\\bin\\ahdcode.exe\" %*\r\nexit /b %errorlevel%\r\n"
	if err = os.WriteFile(filepath.Join(root, "bin", "ahdcode.cmd"), []byte(launcher), 0644); err != nil {
		return err
	}
	script, err := os.ReadFile(filepath.Join(target, "uninstall.ps1"))
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(root, "uninstall.ps1"), script, 0644); err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	path, kind, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return err
	}
	bin := filepath.Join(root, "bin")
	found := false
	for _, part := range strings.Split(path, ";") {
		if strings.EqualFold(strings.TrimRight(part, `\`), bin) {
			found = true
		}
	}
	if !found {
		if path != "" && !strings.HasSuffix(path, ";") {
			path += ";"
		}
		path += bin
		if kind == registry.SZ {
			err = key.SetStringValue("Path", path)
		} else {
			err = key.SetExpandStringValue("Path", path)
		}
		if err != nil {
			return err
		}
	}
	uninstall, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall\AhdCode`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer uninstall.Close()
	for name, value := range map[string]string{"DisplayName": "AhdCode", "DisplayVersion": version, "Publisher": "AhdCode", "InstallLocation": root, "UninstallString": `powershell.exe -NoProfile -File "` + filepath.Join(root, "uninstall.ps1") + `"`} {
		if err = uninstall.SetStringValue(name, value); err != nil {
			return err
		}
	}
	cmd := exec.Command(filepath.Join(target, "bin", "ahdcode.exe"), "--version")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return err
	}
	fmt.Println("Installed. Sign out and back in to refresh PATH, or open a terminal with the updated user environment.")
	return nil
}
