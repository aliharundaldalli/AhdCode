package source

import "testing"

func TestSpanText(t *testing.T) {
	file := NewFile(7, "sample.ahd", "öğrenci")
	span := NewSpan(file.ID, Position{Offset: 0, Line: 1, Column: 1}, Position{Offset: len(file.Text), Line: 1, Column: 8})
	if got := span.Text(file); got != "öğrenci" {
		t.Fatalf("Span.Text() = %q", got)
	}
}

func TestSpanTextRejectsInvalidRange(t *testing.T) {
	file := NewFile(1, "sample.ahd", "abc")
	span := NewSpan(file.ID, Position{Offset: 2}, Position{Offset: 9})
	if got := span.Text(file); got != "" {
		t.Fatalf("invalid Span.Text() = %q", got)
	}
}
