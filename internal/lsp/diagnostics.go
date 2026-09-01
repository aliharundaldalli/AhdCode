package lsp

import "ahdcode/internal/diagnostics"

// LSP DiagnosticSeverity values (textDocument/publishDiagnostics).
const (
	diagnosticSeverityError   = 1
	diagnosticSeverityWarning = 2
)

// lspDiagnostic is one LSP Diagnostic. Source is always "ahdcode": every
// diagnostic originates from the real AhdCode compiler frontend, never a
// second, LSP-invented validator.
type lspDiagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}

type publishDiagnosticsParams struct {
	URI         string          `json:"uri"`
	Diagnostics []lspDiagnostic `json:"diagnostics"`
}

// convertDiagnostic translates one compiler diagnostics.Diagnostic into its
// LSP wire form using index (built from the exact text that diagnostic was
// produced against) for the byte-offset-to-UTF-16-Position range
// conversion. The compiler's own Code, Message, and Severity are carried
// through unchanged -- this never rewrites a compiler message for the
// editor. A non-empty Hint is appended as a second line, since LSP's
// Diagnostic has no dedicated hint field but editors render a multi-line
// message fine.
func convertDiagnostic(item diagnostics.Diagnostic, index *lineIndex) lspDiagnostic {
	message := item.Message
	if item.Hint != "" {
		message = message + "\nhint: " + item.Hint
	}
	severity := diagnosticSeverityError
	if item.Severity == diagnostics.SeverityWarning {
		severity = diagnosticSeverityWarning
	}
	return lspDiagnostic{
		Range: Range{
			Start: index.OffsetToPosition(item.Span.Start.Offset),
			End:   index.OffsetToPosition(item.Span.End.Offset),
		},
		Severity: severity,
		Code:     item.Code,
		Source:   "ahdcode",
		Message:  message,
	}
}
