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
