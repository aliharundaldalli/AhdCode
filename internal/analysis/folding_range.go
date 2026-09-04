package analysis

import (
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
)

// FoldingRange is one foldable source region.
type FoldingRange struct {
	StartLine      int
	EndLine        int
	Kind           string
	StartCharacter int
	EndCharacter   int
}

// FoldingRanges returns structural folding ranges for path.
func (store *Store) FoldingRanges(path string) []FoldingRange {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()
	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return nil
	}
	index := newLineIndexForFolding(cached.result.Text[canonical])
	fileID := cached.fileIDFor(canonical)
	var ranges []FoldingRange
	for _, statement := range entryModule.Parsed.Program.Statements {
		if !stmtInFile(statement, fileID) {
			continue
		}
		collectFoldingStmt(statement, index, &ranges)
	}
	return ranges
}

func collectFoldingStmt(statement ast.Stmt, index *foldLineIndex, ranges *[]FoldingRange) {
	if statement == nil {
		return
	}
	switch node := statement.(type) {
	case *ast.FunctionDecl:
		if node.Body != nil {
			addBlockFold(node.Body, index, ranges)
		}
	case *ast.ClassDecl:
		if span := node.Span(); !span.Empty() {
			*ranges = append(*ranges, index.blockFold(span))
		}
		for _, member := range node.Members {
			collectFoldingStmt(member, index, ranges)
		}
	case *ast.StructureDecl:
		if node.Body != nil {
			addBlockFold(node.Body, index, ranges)
		}
	case *ast.IfStmt:
		for _, branch := range node.Branches {
			if branch.Body != nil {
				addBlockFold(branch.Body, index, ranges)
			}
		}
		if node.Else != nil {
			addBlockFold(node.Else, index, ranges)
		}
	case *ast.WhileStmt, *ast.UntilStmt, *ast.ForStmt:
		if body := stmtBody(node); body != nil {
			addBlockFold(body, index, ranges)
		}
	case *ast.AttemptStmt:
		if node.Body != nil {
			addBlockFold(node.Body, index, ranges)
		}
		for _, clause := range node.Excepts {
			if clause.Body != nil {
				addBlockFold(clause.Body, index, ranges)
			}
		}
		if node.Ultimately != nil {
			addBlockFold(node.Ultimately, index, ranges)
		}
	case *ast.StateStmt:
		for _, condition := range node.Conditions {
			if condition.Body != nil {
				addBlockFold(condition.Body, index, ranges)
			}
		}
	}
}

func stmtBody(statement ast.Stmt) *ast.Block {
	switch node := statement.(type) {
	case *ast.WhileStmt:
		return node.Body
	case *ast.UntilStmt:
		return node.Body
	case *ast.ForStmt:
		return node.Body
	default:
		return nil
	}
}

func addBlockFold(block *ast.Block, index *foldLineIndex, ranges *[]FoldingRange) {
	if block == nil || block.Span().Empty() {
		return
	}
	*ranges = append(*ranges, index.blockFold(block.Span()))
}

type foldLineIndex struct {
	text   string
	starts []int
}

func newLineIndexForFolding(text string) *foldLineIndex {
	starts := []int{0}
	for index := 0; index < len(text); index++ {
		if text[index] == '\n' {
			starts = append(starts, index+1)
		}
	}
	return &foldLineIndex{text: text, starts: starts}
}

func (index *foldLineIndex) lineOf(offset int) int {
	line := 0
	for line+1 < len(index.starts) && index.starts[line+1] <= offset {
		line++
	}
	return line
}

func (index *foldLineIndex) blockFold(span source.Span) FoldingRange {
	startLine := index.lineOf(span.Start.Offset)
	endLine := index.lineOf(span.End.Offset)
	if endLine <= startLine {
		endLine = startLine + 1
	}
	return FoldingRange{StartLine: startLine, EndLine: endLine, Kind: "region"}
}
