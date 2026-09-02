package analysis

import (
	"ahdcode/internal/syntax/ast"
)

// SelectionRange is one expandable selection level.
type SelectionRange struct {
	StartOffset int
	EndOffset   int
	Parent      *SelectionRange
}

// SelectionRanges returns progressively larger AST ancestor ranges for offset.
func (store *Store) SelectionRanges(path string, offset int) *SelectionRange {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()
	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return nil
	}
	ancestors := ancestorsAtOffset(entryModule.Parsed.Program, offset)
	if len(ancestors) == 0 {
		return nil
	}
	var head *SelectionRange
	var tail *SelectionRange
	for index := len(ancestors) - 1; index >= 0; index-- {
		span := ancestors[index].Span()
		if span.Empty() {
			continue
		}
		node := &SelectionRange{
			StartOffset: span.Start.Offset,
			EndOffset:   span.End.Offset,
		}
		if tail != nil {
			tail.Parent = node
		} else {
			head = node
		}
		tail = node
	}
	return head
}

// selectionRangeNode filters ancestors to meaningful selection levels.
func selectionRangeNode(node ast.Node) bool {
	switch node.(type) {
	case *ast.Program:
		return false
	default:
		return true
	}
}
