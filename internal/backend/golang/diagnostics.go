// Package golang lowers validated AhdCode IR into deterministic Go source.
//
// The generator never re-runs name resolution, overload selection, Function
// inference, implicit conversion discovery, or Class identity decisions. Every
// such decision is read from the IR contract produced by earlier milestones.
package golang

import (
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/source"
)

// Backend diagnostic family. These codes are stable and independent of the
// frontend LEX/PAR/SEM/IR/LWR families.
const (
	// CodeUnsupportedNode reports valid IR that this milestone's backend does
	// not yet lower. It never produces silently wrong Go.
	CodeUnsupportedNode = "BCK001"
	// CodeInvalidRepresentation reports IR that has no valid Go runtime
	// representation.
	CodeInvalidRepresentation = "BCK002"
	// CodeGenerationFailure reports a malformed or incomplete IR node reaching
	// code generation.
	CodeGenerationFailure = "BCK003"
	// CodeFormatFailure reports generated source that gofmt could not parse.
	CodeFormatFailure = "BCK004"
	// CodeBuildFailure reports a failing go build of the generated program.
	CodeBuildFailure = "BCK005"
	// CodeMissingToolchain reports an unavailable Go toolchain.
	CodeMissingToolchain = "BCK006"
	// CodeWorkspaceFailure reports a failing temporary build workspace.
	CodeWorkspaceFailure = "BCK007"
)

func backendError(code, message string, span source.Span, hint string) diagnostics.Diagnostic {
	return diagnostics.Diagnostic{Code: code, Severity: diagnostics.SeverityError, Message: message, Span: span, Hint: hint}
}
