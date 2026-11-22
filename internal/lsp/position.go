package lsp

import "ahdcode/internal/source"

type Position = source.UTF16Position
type Range = source.UTF16Range
type lineIndex = source.LineIndex

func newLineIndex(text string) *lineIndex {
	return source.NewLineIndex(text)
}
