package module

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type ModuleResolver interface {
	CanonicalEntry(entryPath string) (SourceIdentity, error)
	Resolve(importer SourceIdentity, moduleName string) (SourceIdentity, error)
}

type SourceLoader interface {
	Load(identity SourceIdentity) (string, error)
}

// FileResolver implements the v0.1 sibling ModuleName.ahd rule.
type FileResolver struct{}

func (FileResolver) CanonicalEntry(entryPath string) (SourceIdentity, error) {
	canonical, err := canonicalFilePath(entryPath)
	if err != nil {
		return SourceIdentity{}, err
	}
	return fileIdentity(canonical), nil
}

func (FileResolver) Resolve(importer SourceIdentity, moduleName string) (SourceIdentity, error) {
	if importer.Builtin {
		return SourceIdentity{}, fmt.Errorf("built-in module %s has no filesystem-relative imports", importer.Name)
	}
	directory := filepath.Dir(importer.Path)
	filename := moduleName + ".ahd"
	if entries, err := os.ReadDir(directory); err == nil {
		exact := false
		for _, entry := range entries {
			if entry.Name() == filename {
				exact = true
				break
			}
		}
		if !exact {
			return SourceIdentity{}, fmt.Errorf("no case-sensitive sibling file %s", filename)
		}
	}
	candidate := filepath.Join(directory, filename)
	canonical, err := canonicalFilePath(candidate)
	if err != nil {
		return SourceIdentity{}, err
	}
	return fileIdentity(canonical), nil
}

func canonicalFilePath(value string) (string, error) {
	absolute, err := filepath.Abs(filepath.Clean(value))
	if err != nil {
		return "", err
	}
	if evaluated, evaluateError := filepath.EvalSymlinks(absolute); evaluateError == nil {
		absolute = evaluated
	}
	return filepath.Clean(absolute), nil
}

func fileIdentity(canonical string) SourceIdentity {
	base := filepath.Base(canonical)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return SourceIdentity{ID: ModuleID("file:" + filepath.ToSlash(canonical)), Name: name, Path: canonical}
}

type FileLoader struct{}

func (FileLoader) Load(identity SourceIdentity) (string, error) {
	bytes, err := os.ReadFile(identity.Path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// InMemoryWorkspace is both a resolver and loader for deterministic tests.
type InMemoryWorkspace struct {
	sources   map[string]string
	loadCount map[ModuleID]int
}

func NewInMemoryWorkspace(sources map[string]string) *InMemoryWorkspace {
	workspace := &InMemoryWorkspace{sources: make(map[string]string), loadCount: make(map[ModuleID]int)}
	for filePath, text := range sources {
		workspace.sources[canonicalMemoryPath(filePath)] = text
	}
	return workspace
}

func (workspace *InMemoryWorkspace) CanonicalEntry(entryPath string) (SourceIdentity, error) {
	return memoryIdentity(canonicalMemoryPath(entryPath)), nil
}

func (workspace *InMemoryWorkspace) Resolve(importer SourceIdentity, moduleName string) (SourceIdentity, error) {
	return memoryIdentity(canonicalMemoryPath(path.Join(path.Dir(importer.Path), moduleName+".ahd"))), nil
}

func (workspace *InMemoryWorkspace) Load(identity SourceIdentity) (string, error) {
	workspace.loadCount[identity.ID]++
	text, exists := workspace.sources[identity.Path]
	if !exists {
		return "", fmt.Errorf("module source %s does not exist", identity.Path)
	}
	return text, nil
}

func (workspace *InMemoryWorkspace) LoadCount(id ModuleID) int { return workspace.loadCount[id] }

func canonicalMemoryPath(value string) string {
	cleaned := path.Clean("/" + strings.TrimPrefix(filepath.ToSlash(value), "/"))
	return cleaned
}

func memoryIdentity(canonical string) SourceIdentity {
	base := path.Base(canonical)
	name := strings.TrimSuffix(base, path.Ext(base))
	return SourceIdentity{ID: ModuleID("mem:" + canonical), Name: name, Path: canonical}
}
