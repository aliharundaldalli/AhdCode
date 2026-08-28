package diagnostics

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"ahdcode/internal/source"
)

// Render produces the stable, source-oriented CLI representation of one
// diagnostic. It consumes existing diagnostic metadata and does not reinterpret
// compiler semantics.
func Render(item Diagnostic, files map[source.FileID]source.File) string {
	severity := "error"
	if item.Severity == SeverityWarning {
		severity = "warning"
	}
	location := ""
	file, known := files[item.Span.FileID]
	if known && item.Span.FileID != 0 {
		location = fmt.Sprintf(" %s:%d:%d", file.Path, item.Span.Start.Line, item.Span.Start.Column)
	}
	var out strings.Builder
	fmt.Fprintf(&out, "%s [%s]%s\n%s", severity, item.Code, location, item.Message)
	if known && item.Span.Start.Line > 0 {
		if line, ok := sourceLine(file.Text, item.Span.Start.Line); ok {
			lineNumber := fmt.Sprintf("%d", item.Span.Start.Line)
			fmt.Fprintf(&out, "\n  %s | %s", lineNumber, line)
			column := item.Span.Start.Column
			if column < 1 {
				column = 1
			}
			width := 1
			if item.Span.End.Line == item.Span.Start.Line && item.Span.End.Column > column {
				width = item.Span.End.Column - column
			}
			lineWidth := utf8.RuneCountInString(line)
			if column > lineWidth+1 {
				column = lineWidth + 1
			}
			if maximum := lineWidth - column + 1; maximum > 0 && width > maximum {
				width = maximum
			}
			fmt.Fprintf(&out, "\n  %s | %s%s", strings.Repeat(" ", len(lineNumber)), strings.Repeat(" ", column-1), strings.Repeat("^", width))
		}
	}
	if item.Hint != "" {
		fmt.Fprintf(&out, "\n  hint: %s", item.Hint)
	}
	return out.String()
}

func sourceLine(text string, wanted int) (string, bool) {
	line := 1
	start := 0
	for index := 0; index <= len(text); index++ {
		if index < len(text) && text[index] != '\n' && text[index] != '\r' {
			continue
		}
		if line == wanted {
			return text[start:index], true
		}
		if index < len(text) && text[index] == '\r' && index+1 < len(text) && text[index+1] == '\n' {
			index++
		}
		line++
		start = index + 1
	}
	return "", false
}
