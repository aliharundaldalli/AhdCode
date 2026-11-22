package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/framework"
	"ahdcode/internal/module"
	"ahdcode/internal/source"
)

type checkReport struct {
	SchemaVersion int               `json:"schemaVersion"`
	Diagnostics   []checkDiagnostic `json:"diagnostics"`
}

type checkDiagnostic struct {
	Severity string             `json:"severity"`
	Code     string             `json:"code"`
	Message  string             `json:"message"`
	Path     string             `json:"path"`
	Range    *source.UTF16Range `json:"range"`
	Hint     string             `json:"hint,omitempty"`
}

func runCheck(arguments []string, output, errorOutput io.Writer) int {
	entry, format, err := parseCheckArgs(arguments)
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode check: %v\nusage: ahdcode check <entry.ahd> [--format text|json]\n", err)
		return 2
	}

	filesystem := checkFilesystem{}
	result := module.NewCompiler(filesystem, filesystem).Compile(entry)
	err = writeCheckReport(output, format, makeCheckReport(entry, result))
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode check: %v\n", err)
		return 1
	}
	if result.HasErrors() {
		return 1
	}
	return 0
}

func parseCheckArgs(arguments []string) (entry, format string, err error) {
	hasEntry, flags := false, true
	for len(arguments) > 0 {
		argument := arguments[0]
		arguments = arguments[1:]
		if flags && argument == "--" {
			flags = false
			continue
		}
		if flags && strings.HasPrefix(argument, "-") {
			flag, value, inline := strings.Cut(argument, "=")
			if flag != "--format" {
				return "", "", fmt.Errorf("unknown flag %q", flag)
			}
			if format != "" {
				return "", "", errors.New("--format may only be given once")
			}
			if !inline && len(arguments) > 0 && !strings.HasPrefix(arguments[0], "-") {
				value = arguments[0]
				arguments = arguments[1:]
			}
			if value != "text" && value != "json" {
				return "", "", fmt.Errorf("invalid output format %q (expected text or json)", value)
			}
			format = value
			continue
		}
		if hasEntry {
			return "", "", errors.New("exactly one entry file is required")
		}
		entry, hasEntry = argument, true
	}
	if !hasEntry {
		return "", "", errors.New("an entry file is required")
	}
	if format == "" {
		format = "text"
	}
	return entry, format, nil
}

// checkFilesystem uses the normal resolver and loader. Its entry validation
// and path-error presentation apply only to the standalone command.
type checkFilesystem struct {
	module.FileResolver
	module.FileLoader
}

func (filesystem checkFilesystem) CanonicalEntry(entry string) (module.SourceIdentity, error) {
	if !strings.EqualFold(filepath.Ext(entry), ".ahd") {
		return module.SourceIdentity{}, errors.New("entry must be an .ahd source file")
	}
	identity, err := filesystem.FileResolver.CanonicalEntry(entry)
	return identity, checkFilesystemError(err)
}

func (filesystem checkFilesystem) Resolve(importer module.SourceIdentity, name string) (module.SourceIdentity, error) {
	identity, err := filesystem.FileResolver.Resolve(importer, name)
	return identity, checkFilesystemError(err)
}

func (filesystem checkFilesystem) Load(identity module.SourceIdentity) (string, error) {
	text, err := filesystem.FileLoader.Load(identity)
	return text, checkFilesystemError(err)
}

func checkFilesystemError(err error) error {
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		return fmt.Errorf("%s: %w", pathError.Op, pathError.Err)
	}
	return err
}

func makeCheckReport(entry string, result module.CompilationResult) checkReport {
	report := checkReport{SchemaVersion: 1, Diagnostics: make([]checkDiagnostic, 0, len(result.Diagnostics))}
	if len(result.Diagnostics) == 0 {
		return report
	}
	entryPath := filepath.Base(entry)
	var root string
	if compiled := result.Modules[result.Entry]; compiled != nil {
		root = filepath.Dir(compiled.Source.Path)
		entryPath = filepath.Base(compiled.Source.Path)
	}
	files := make(map[source.FileID]source.File)
	for _, compiled := range result.Modules {
		if compiled.File.ID != 0 {
			files[compiled.File.ID] = compiled.File
		}
		for _, file := range compiled.RequiredFiles {
			files[file.ID] = file
		}
	}
	indexes := make(map[source.FileID]*source.LineIndex)
	for _, item := range result.Diagnostics {
		diagnostic := item.Diagnostic
		display := checkDiagnostic{
			Severity: "error", Code: diagnostic.Code, Message: diagnostic.Message,
			Path: entryPath, Hint: diagnostic.Hint,
		}
		if diagnostic.Severity == diagnostics.SeverityWarning {
			display.Severity = "warning"
		}
		if file, ok := files[diagnostic.Span.FileID]; ok {
			display.Path = checkSourcePath(root, file.Path)
			index := indexes[file.ID]
			if index == nil {
				index = source.NewLineIndex(file.Text)
				indexes[file.ID] = index
			}
			display.Range = &source.UTF16Range{
				Start: index.OffsetToPosition(diagnostic.Span.Start.Offset),
				End:   index.OffsetToPosition(diagnostic.Span.End.Offset),
			}
		}
		if len(item.Cycle) != 0 {
			// The compiler's cycle message contains absolute module IDs.
			// Use the graph metadata to replace only those generated IDs.
			replacements := make([]string, 0, len(item.Cycle)*2)
			for _, id := range item.Cycle {
				if compiled := result.Modules[id]; compiled != nil && !compiled.Source.Builtin && !compiled.Source.Framework {
					replacements = append(replacements, string(id), checkSourcePath(root, compiled.Source.Path))
				}
			}
			display.Message = strings.NewReplacer(replacements...).Replace(display.Message)
		}
		report.Diagnostics = append(report.Diagnostics, display)
	}
	slices.SortFunc(report.Diagnostics, compareCheckDiagnostics)
	return report
}

func checkSourcePath(root, path string) string {
	if framework.IsVirtualPath(path) {
		return path
	}
	if relative, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(relative)
	}
	return filepath.Base(path)
}

func compareCheckDiagnostics(left, right checkDiagnostic) int {
	if order := cmp.Compare(left.Path, right.Path); order != 0 {
		return order
	}
	switch {
	case left.Range == nil && right.Range != nil:
		return -1
	case left.Range != nil && right.Range == nil:
		return 1
	case left.Range != nil && right.Range != nil:
		if order := cmp.Or(
			cmp.Compare(left.Range.Start.Line, right.Range.Start.Line),
			cmp.Compare(left.Range.Start.Character, right.Range.Start.Character),
			cmp.Compare(left.Range.End.Line, right.Range.End.Line),
			cmp.Compare(left.Range.End.Character, right.Range.End.Character),
		); order != 0 {
			return order
		}
	}
	return cmp.Or(
		cmp.Compare(left.Severity, right.Severity),
		cmp.Compare(left.Code, right.Code),
		cmp.Compare(left.Message, right.Message),
		cmp.Compare(left.Hint, right.Hint),
	)
}

func writeCheckReport(output io.Writer, format string, report checkReport) error {
	if format == "json" {
		return json.NewEncoder(output).Encode(report)
	}
	var text strings.Builder
	for _, item := range report.Diagnostics {
		fmt.Fprint(&text, item.Path)
		if item.Range != nil {
			fmt.Fprintf(&text, ":%d:%d", item.Range.Start.Line+1, item.Range.Start.Character+1)
		}
		fmt.Fprintf(&text, ": %s", item.Severity)
		if item.Code != "" {
			fmt.Fprintf(&text, " %s", item.Code)
		}
		fmt.Fprintf(&text, ": %s\n", item.Message)
		if item.Hint != "" {
			fmt.Fprintf(&text, "  hint: %s\n", item.Hint)
		}
	}
	_, err := io.WriteString(output, text.String())
	return err
}
