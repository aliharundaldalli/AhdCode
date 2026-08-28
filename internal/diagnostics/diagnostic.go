package diagnostics

import "ahdcode/internal/source"

// Severity describes the effect of a diagnostic.
type Severity uint8

const (
	SeverityError Severity = iota
	SeverityWarning
)

// Diagnostic is a stable, source-addressed compiler message.
type Diagnostic struct {
	Code     string
	Severity Severity
	Message  string
	Span     source.Span
	Hint     string
}

// Bag accumulates diagnostics without exposing mutable internals.
type Bag struct {
	items []Diagnostic
}

// Add appends a diagnostic.
func (b *Bag) Add(d Diagnostic) {
	b.items = append(b.items, d)
}

// Error appends an error diagnostic.
func (b *Bag) Error(code, message string, span source.Span, hint string) {
	b.Add(Diagnostic{Code: code, Severity: SeverityError, Message: message, Span: span, Hint: hint})
}

// Items returns a defensive copy of accumulated diagnostics.
func (b *Bag) Items() []Diagnostic {
	return append([]Diagnostic(nil), b.items...)
}

// HasErrors reports whether at least one error exists.
func (b *Bag) HasErrors() bool {
	for _, item := range b.items {
		if item.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Len returns the number of accumulated diagnostics.
func (b *Bag) Len() int {
	return len(b.items)
}
