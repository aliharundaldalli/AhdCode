// Package studio materializes the exact-version AhdDataStudio bundled in the
// AhdCode CLI into a per-user cache directory.
package studio

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ahdcode/internal/ahdversion"
	studioembed "ahdcode/tools"
)

const stampName = "AHDSTUDIO_VERSION"

// CacheDir is the exact-version directory that should contain this CLI's
// AhdDataStudio sources.
func CacheDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("AHDCODE_STUDIO_CACHE")); override != "" {
		if !filepath.IsAbs(override) {
			return "", fmt.Errorf("AHDCODE_STUDIO_CACHE must be an absolute path")
		}
		return filepath.Clean(override), nil
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate a per-user cache directory: %v", err)
	}
	return filepath.Join(root, "ahdcode", "studio", "v"+ahdversion.Number), nil
}

// Materialize writes the bundled Studio into the per-user cache if needed
// and returns the directory that contains app.ahd.
func Materialize() (string, error) {
	dest, err := CacheDir()
	if err != nil {
		return "", err
	}
	if studioReady(dest) {
		return dest, nil
	}
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", fmt.Errorf("cannot create AhdDataStudio cache: %v", err)
	}
	lock, err := lockDir(parent)
	if err != nil {
		return "", err
	}
	defer unlockDir(lock)
	if studioReady(dest) {
		return dest, nil
	}

	staging, err := os.MkdirTemp(parent, "v"+ahdversion.Number+".tmp-*")
	if err != nil {
		return "", fmt.Errorf("cannot stage AhdDataStudio: %v", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err := extractBundle(staging); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(staging, stampName), []byte(ahdversion.Number+"\n"), 0o644); err != nil {
		return "", err
	}
	_ = os.RemoveAll(dest)
	if err := os.Rename(staging, dest); err != nil {
		return "", fmt.Errorf("cannot publish AhdDataStudio: %v", err)
	}
	return dest, nil
}

func studioReady(dir string) bool {
	app := filepath.Join(dir, "app.ahd")
	info, err := os.Stat(app)
	if err != nil || info.IsDir() {
		return false
	}
	stamp, err := os.ReadFile(filepath.Join(dir, stampName))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(stamp)) == ahdversion.Number
}

func extractBundle(dest string) error {
	return fs.WalkDir(studioembed.Files, "AhdDataStudio", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel("AhdDataStudio", path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if shouldSkipStudioPath(relative) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		target := filepath.Join(dest, filepath.FromSlash(relative))
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) && filepath.Clean(target) != filepath.Clean(dest) {
			return fmt.Errorf("invalid AhdDataStudio path")
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		source, err := studioembed.Files.Open(path)
		if err != nil {
			return err
		}
		defer source.Close()
		file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(file, source); err != nil {
			_ = file.Close()
			return err
		}
		return file.Close()
	})
}

func shouldSkipStudioPath(relative string) bool {
	base := filepath.Base(relative)
	switch {
	case base == ".env" || strings.HasSuffix(base, ".run") || strings.HasSuffix(base, ".dev"):
		return true
	case base == ".DS_Store":
		return true
	default:
		return false
	}
}

func lockDir(dir string) (*os.File, error) {
	lockPath := filepath.Join(dir, ".materialize.lock")
	deadline := time.Now().Add(10 * time.Second)
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			return file, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("cannot lock AhdDataStudio cache: %v", err)
		}
		if time.Now().After(deadline) {
			_ = os.Remove(lockPath)
			continue
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func unlockDir(file *os.File) {
	if file == nil {
		return
	}
	path := file.Name()
	_ = file.Close()
	_ = os.Remove(path)
}
