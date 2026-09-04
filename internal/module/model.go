// Package module coordinates deterministic multi-file frontend compilation.
package module

import (
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/parser"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
)

// ModuleID is a canonical identity, independent of the source spelling used
// by a bring statement.
type ModuleID string

type AnalysisState uint8

const (
	Unseen AnalysisState = iota
	Resolving
	Resolved
	Failed
)

type SourceIdentity struct {
	ID      ModuleID
	Name    string
	Path    string
	Builtin bool
	// Framework marks one of AhdCode's bundled first-party AhdCode source
	// modules (see internal/framework). Such a module is compiled from bytes
	// embedded in the compiler rather than loaded through SourceLoader, and
	// its own imports resolve only against other bundled modules and the
	// built-in modules -- never against the application's source tree.
	Framework bool
}

type Module struct {
	ID           ModuleID
	Source       SourceIdentity
	File         source.File
	Parsed       parser.Result
	Semantic     semantic.Result
	Interface    *semantic.ModuleInterface
	Dependencies []ModuleID
	State        AnalysisState
	AnalyzeCount int
	// RequiredFiles holds the parsed source.File for every distinct local
	// file a require(...) statement merged into this module, in first-
	// encountered (deterministic) order. Empty when the module uses no
	// require(...). Populated before Parsed.Program's statements are
	// analyzed, since the required files' statements are already spliced
	// into Parsed.Program by then.
	RequiredFiles []source.File
	// UnresolvedRequires lists the canonical absolute paths this module's
	// require(...) statements named but could not load on this attempt, so a
	// caller such as `ahdcode dev` can watch for the file appearing later
	// without needing another edit to the requesting file to notice it.
	UnresolvedRequires []string
}

// ModuleDiagnostic adds graph/import context without changing the stable
// frontend Diagnostic contract.
type ModuleDiagnostic struct {
	Diagnostic       diagnostics.Diagnostic
	RequestingModule ModuleID
	TargetModule     ModuleID
	RequestedSymbol  string
	Cycle            []ModuleID
}

type CompilationResult struct {
	Entry       ModuleID
	Modules     map[ModuleID]*Module
	Order       []ModuleID
	Diagnostics []ModuleDiagnostic
}

func (result CompilationResult) HasErrors() bool {
	for _, item := range result.Diagnostics {
		if item.Diagnostic.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}
