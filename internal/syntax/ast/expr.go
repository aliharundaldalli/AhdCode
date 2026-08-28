package ast

// Expr is a typed expression node.
type Expr interface {
	Node
	expressionNode()
}

type LiteralKind uint8

const (
	IntLiteral LiteralKind = iota
	RealLiteral
	BoolLiteral
	NullLiteral
)

type LiteralExpr struct {
	Base
	Kind  LiteralKind
	Raw   string
	Value string
}

func (*LiteralExpr) expressionNode() {}

type IdentifierExpr struct {
	Base
	Name string
	Raw  string
}

func (*IdentifierExpr) expressionNode() {}

type GroupExpr struct {
	Base
	Expression Expr
}

func (*GroupExpr) expressionNode() {}

type UnaryExpr struct {
	Base
	Operator string
	Operand  Expr
}

func (*UnaryExpr) expressionNode() {}

type BinaryExpr struct {
	Base
	Left     Expr
	Operator string
	Right    Expr
}

func (*BinaryExpr) expressionNode() {}

type CallArgument struct {
	Base
	Name  string
	Value Expr
}

// CallExpr represents every syntactic call, including syntax that semantic
// analysis may later resolve as Class construction.
type CallExpr struct {
	Base
	Callee    Expr
	Arguments []CallArgument
}

func (*CallExpr) expressionNode() {}

type MemberExpr struct {
	Base
	Object Expr
	Name   string
}

func (*MemberExpr) expressionNode() {}

type IndexExpr struct {
	Base
	Object Expr
	Index  Expr
}

func (*IndexExpr) expressionNode() {}

type SliceExpr struct {
	Base
	Object Expr
	Start  Expr
	End    Expr
}

func (*SliceExpr) expressionNode() {}

type ListExpr struct {
	Base
	Elements []Expr
}

func (*ListExpr) expressionNode() {}

type PairEntry struct {
	Base
	Key   Expr
	Value Expr
}

type PairExpr struct {
	Base
	Entries []PairEntry
}

func (*PairExpr) expressionNode() {}

// StringPart contains either Text or Expression.
type StringPart struct {
	Base
	Text       string
	Expression Expr
}

type StringExpr struct {
	Base
	Delimiter string
	Parts     []StringPart
}

func (*StringExpr) expressionNode() {}
