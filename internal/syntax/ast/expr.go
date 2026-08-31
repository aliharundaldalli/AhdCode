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
	ImaginaryLiteral
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

// LambdaExpr is the expression-only shorthand for an anonymous value of the
// existing Function type. Its return type is inferred from Body.

// CaptureKind distinguishes the two kinds of external dependency a lambda's
// dependency list may name.
type CaptureKind uint8

const (
	// LocalCapture reads an enclosing lexical binding by value, spelled either
	// `#name` or `Local name`.
	LocalCapture CaptureKind = iota
	// GlobalCapture is an explicit declaration that the lambda intentionally
	// accesses a module/global binding, spelled either `@name` or `Global
	// name`. It is not a capture: the lambda reads the real binding, exactly
	// like an ordinary Function's Global declaration.
	GlobalCapture
)

// CaptureRef is one entry in a lambda's explicit dependency list. Dependency
// is always written out: a lambda never reads an enclosing local or a module
// binding implicitly.
type CaptureRef struct {
	Base
	Kind CaptureKind
	Name string
}

type LambdaExpr struct {
	Base
	// Captures is the explicit capture list. It is empty both for a lambda
	// written without brackets and for one written `lambda [] (...)`, because
	// the two mean the same thing: no enclosing local is captured.
	Captures   []CaptureRef
	Parameters []Parameter
	Body       Expr
}

func (*LambdaExpr) expressionNode() {}

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
