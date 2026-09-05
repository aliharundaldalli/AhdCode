// Package initweb writes a minimal offline AhdCode Web application into
// the current directory. Templates ship inside the CLI; nothing is fetched.
package initweb

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

//go:embed templates
var templates embed.FS

type fileSpec struct {
	embedPath string
	relPath   string
	perm      os.FileMode
	mergeGI   bool
}

var managedFiles = []fileSpec{
	{embedPath: "templates/app.ahd", relPath: "app.ahd", perm: 0o644},
	{embedPath: "templates/env", relPath: ".env", perm: 0o600},
	{embedPath: "templates/env.example", relPath: ".env.example", perm: 0o644},
	{embedPath: "templates/gitignore", relPath: ".gitignore", perm: 0o644, mergeGI: true},
	{embedPath: "templates/Config/App.ahd", relPath: "Config/App.ahd", perm: 0o644},
	{embedPath: "templates/Components/Navbar.ahd", relPath: "Components/Navbar.ahd", perm: 0o644},
	{embedPath: "templates/Components/Footer.ahd", relPath: "Components/Footer.ahd", perm: 0o644},
	{embedPath: "templates/Layouts/Main.ahd", relPath: "Layouts/Main.ahd", perm: 0o644},
	{embedPath: "templates/Pages/Home.ahd", relPath: "Pages/Home.ahd", perm: 0o644},
	{embedPath: "templates/public/style.css", relPath: "public/style.css", perm: 0o644},
	{embedPath: "templates/public/main.js", relPath: "public/main.js", perm: 0o644},
}

var requiredDirs = []string{
	"Config",
	"Components",
	"Layouts",
	"Pages",
	"public",
}

// Web initializes root as a minimal AhdCode Web application.
func Web(root string, output, errorOutput io.Writer) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("cannot initialize Web project:\n%v", err)
	}
	root = filepath.Clean(root)

	planned, err := preflight(root)
	if err != nil {
		return err
	}

	for _, dir := range requiredDirs {
		path, resolveErr := resolveManaged(root, dir)
		if resolveErr != nil {
			return fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", resolveErr)
		}
		if mkErr := os.MkdirAll(path, 0o755); mkErr != nil {
			return fmt.Errorf("cannot initialize Web project:\n%v", mkErr)
		}
	}

	created := make([]string, 0, len(planned))
	for _, item := range planned {
		if wrErr := os.WriteFile(item.abs, item.content, item.perm); wrErr != nil {
			return fmt.Errorf("cannot initialize Web project:\n%v", wrErr)
		}
		created = append(created, item.relPath)
	}

	fmt.Fprintf(output, "Initialized AhdCode Web project in %s\n\nCreated:\n", root)
	for _, rel := range created {
		fmt.Fprintf(output, "  %s\n", rel)
	}
	fmt.Fprint(output, "\nNext:\n  ahdcode dev app.ahd\n")
	return nil
}

type plannedFile struct {
	relPath string
	abs     string
	content []byte
	perm    os.FileMode
}

func preflight(root string) ([]plannedFile, error) {
	var conflicts []string
	planned := make([]plannedFile, 0, len(managedFiles))

	for _, dir := range requiredDirs {
		path, err := resolveManaged(root, dir)
		if err != nil {
			return nil, fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", err)
		}
		info, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			conflicts = append(conflicts, dir+" is a symlink")
			continue
		}
		if !info.IsDir() {
			conflicts = append(conflicts, dir+" is a file; a directory is required")
		}
	}

	for _, spec := range managedFiles {
		abs, err := resolveManaged(root, spec.relPath)
		if err != nil {
			return nil, fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", err)
		}
		content, err := fs.ReadFile(templates, spec.embedPath)
		if err != nil {
			return nil, fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", err)
		}
		content = normalizeNewlines(content)

		info, err := os.Lstat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				planned = append(planned, plannedFile{relPath: spec.relPath, abs: abs, content: content, perm: spec.perm})
				continue
			}
			if isNotDir(err) {
				conflicts = append(conflicts, spec.relPath+" cannot be created because a parent path is a file")
				continue
			}
			return nil, fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			conflicts = append(conflicts, spec.relPath+" is a symlink")
			continue
		}
		if spec.mergeGI && !info.IsDir() {
			existing, readErr := os.ReadFile(abs)
			if readErr != nil {
				return nil, fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", readErr)
			}
			merged, changed := mergeGitignore(string(existing), string(content))
			if changed {
				planned = append(planned, plannedFile{
					relPath: spec.relPath,
					abs:     abs,
					content: []byte(merged),
					perm:    spec.perm,
				})
			}
			continue
		}
		conflicts = append(conflicts, spec.relPath+" already exists")
	}

	if len(conflicts) > 0 {
		return nil, fmt.Errorf("cannot initialize Web project:\n%s\n\nNo files were written.", strings.Join(conflicts, "\n"))
	}
	return planned, nil
}

func resolveManaged(root, rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) || strings.Contains(rel, "..") {
		return "", fmt.Errorf("invalid scaffold path")
	}
	slash := filepath.ToSlash(rel)
	if strings.HasPrefix(slash, "/") || strings.Contains(slash, ":") {
		return "", fmt.Errorf("invalid scaffold path")
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	clean := filepath.Clean(full)
	sep := string(os.PathSeparator)
	if clean != root && !strings.HasPrefix(clean, root+sep) {
		return "", fmt.Errorf("invalid scaffold path")
	}
	return clean, nil
}

func normalizeNewlines(content []byte) []byte {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return []byte(text)
}

func mergeGitignore(existing, required string) (string, bool) {
	have := map[string]bool{}
	for _, line := range strings.Split(existing, "\n") {
		have[strings.TrimSpace(line)] = true
	}
	var missing []string
	for _, line := range strings.Split(required, "\n") {
		entry := strings.TrimSpace(line)
		if entry == "" || have[entry] {
			continue
		}
		missing = append(missing, entry)
	}
	if len(missing) == 0 {
		return existing, false
	}
	out := existing
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	for _, entry := range missing {
		out += entry + "\n"
	}
	return out, true
}

func isNotDir(err error) bool {
	return errors.Is(err, syscall.ENOTDIR) || strings.Contains(err.Error(), "not a directory")
}
