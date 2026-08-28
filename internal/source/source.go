package source

// FileID identifies a source file within one compiler invocation.
type FileID uint32

// File contains immutable AhdCode source text.
type File struct {
	ID   FileID
	Path string
	Text string
}

// NewFile constructs a source file.
func NewFile(id FileID, path, text string) File {
	return File{ID: id, Path: path, Text: text}
}

// Position identifies a byte boundary in a source file. Line and Column are
// one-based. Column counts Unicode code points, not bytes.
type Position struct {
	Offset int
	Line   int
	Column int
}

// Span is a half-open source range [Start, End).
type Span struct {
	FileID FileID
	Start  Position
	End    Position
}

// NewSpan constructs a source span.
func NewSpan(fileID FileID, start, end Position) Span {
	return Span{FileID: fileID, Start: start, End: end}
}

// Empty reports whether the span contains no source bytes.
func (s Span) Empty() bool {
	return s.Start.Offset == s.End.Offset
}

// Text returns the source covered by the span. Invalid cross-file or
// out-of-range spans return an empty string instead of panicking.
func (s Span) Text(file File) string {
	if s.FileID != file.ID || s.Start.Offset < 0 || s.End.Offset < s.Start.Offset || s.End.Offset > len(file.Text) {
		return ""
	}
	return file.Text[s.Start.Offset:s.End.Offset]
}
