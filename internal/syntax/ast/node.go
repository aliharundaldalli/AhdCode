package ast

import "ahdcode/internal/source"

// Node is implemented by every typed AST node.
type Node interface {
	Span() source.Span
}

// Base stores the source span shared by all nodes.
type Base struct {
	Range source.Span
}

// Span returns the node's half-open source range.
func (b Base) Span() source.Span { return b.Range }

// Program is one parsed AhdCode source file.
type Program struct {
	Base
	Statements []Stmt
}

// BadExpr preserves parser progress after malformed expression syntax.
type BadExpr struct{ Base }

func (*BadExpr) expressionNode() {}

// BadStmt preserves parser progress after malformed statement syntax.
type BadStmt struct{ Base }

func (*BadStmt) statementNode() {}
