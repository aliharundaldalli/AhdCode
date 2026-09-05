// Package initweb writes an offline AhdCode Web starter into the current
// directory. Templates ship inside the CLI; nothing is fetched at init time.
package initweb

import (
	"embed"
	"errors"
	"fmt"
	"io"
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
	content   []byte
}

// Web initializes root as an AhdCode Web application.
func Web(root string, output, errorOutput io.Writer, options Options) error {
	_ = errorOutput
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("cannot initialize Web project:\n%v", err)
	}
	root = filepath.Clean(root)
	if options.Output == nil {
		options.Output = output
	}

	options, err = resolveOptions(root, options)
	if err != nil {
		return err
	}

	planned, err := preflight(root, options)
	if err != nil {
		return err
	}

	var createdMySQL bool
	if options.isMySQL() {
		createdMySQL, err = bootstrapMySQL(options, nil)
		if err != nil {
			if createdMySQL {
				return fmt.Errorf("%v\n\n%s", err, leftoverMySQLMessage(options.DatabaseName))
			}
			return err
		}
	}

	var stagedSQLite string
	if options.isSQLite() {
		stagedSQLite, err = sqliteStage(options)
		if err != nil {
			if createdMySQL {
				return fmt.Errorf("%v\n\n%s", err, leftoverMySQLMessage(options.DatabaseName))
			}
			return err
		}
		defer func() {
			if stagedSQLite != "" {
				_ = os.Remove(stagedSQLite)
			}
		}()
	}

	for _, dir := range requiredDirsFor(options) {
		path, resolveErr := resolveManaged(root, dir)
		if resolveErr != nil {
			return finishFailure(fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", resolveErr), createdMySQL, options)
		}
		if mkErr := os.MkdirAll(path, 0o755); mkErr != nil {
			return finishFailure(fmt.Errorf("cannot initialize Web project:\n%v", mkErr), createdMySQL, options)
		}
	}

	if err := writePlanned(planned); err != nil {
		return finishFailure(fmt.Errorf("cannot initialize Web project:\n%v", err), createdMySQL, options)
	}

	if stagedSQLite != "" {
		if err := installSQLiteFile(root, options, stagedSQLite); err != nil {
			return finishFailure(err, createdMySQL, options)
		}
		stagedSQLite = ""
	}

	writeSuccess(output, options)
	return nil
}

func finishFailure(err error, createdMySQL bool, options Options) error {
	if createdMySQL {
		return fmt.Errorf("%v\n\n%s", err, leftoverMySQLMessage(options.DatabaseName))
	}
	return err
}

func writeSuccess(output io.Writer, options Options) {
	fmt.Fprint(output, "AhdCode Web application initialized.\n\n")
	switch options.Starter {
	case StarterBasic:
		fmt.Fprintf(output, "Starter: Basic\nApplication: %s\n\n", options.AppName)
		fmt.Fprint(output, "Application configuration ready.\nMail configuration is available in .env.\n\n")
	case StarterAdmin:
		fmt.Fprintf(output, "Starter: Admin\nApplication: %s\n", options.AppName)
		if options.isSQLite() {
			fmt.Fprint(output, "Database: SQLite\n")
		} else {
			fmt.Fprint(output, "Database: MySQL\n")
		}
		fmt.Fprintf(output, "Database name: %s\nAdmin: %s\n\n", options.DatabaseName, options.AdminEmail)
		fmt.Fprint(output, "Database initialized.\nAdministrator created.\n\n")
	default:
		fmt.Fprintf(output, "Starter: Empty\nApplication: %s\n\n", options.AppName)
	}
	fmt.Fprint(output, "Next:\n  ahdcode dev app.ahd\n")
}

type plannedFile struct {
	relPath string
	abs     string
	content []byte
	perm    os.FileMode
}

func preflight(root string, options Options) ([]plannedFile, error) {
	var conflicts []string
	dirs := requiredDirsFor(options)
	specs := managedFor(options)
	planned := make([]plannedFile, 0, len(specs))

	for _, dir := range dirs {
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

	if options.isSQLite() {
		rel := sqliteRelPath(options)
		abs, err := resolveManaged(root, rel)
		if err != nil {
			return nil, fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", err)
		}
		info, err := os.Lstat(abs)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				conflicts = append(conflicts, rel+" is a symlink")
			} else {
				conflicts = append(conflicts, rel+" already exists")
			}
		} else if err != nil && !os.IsNotExist(err) && !isNotDir(err) {
			return nil, fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", err)
		}
	}

	for _, spec := range specs {
		abs, err := resolveManaged(root, spec.relPath)
		if err != nil {
			return nil, fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", err)
		}
		content, err := loadSpecContent(spec)
		if err != nil {
			return nil, fmt.Errorf("cannot initialize Web project:\n%v\n\nNo files were written.", err)
		}

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
