package ir

import "ahdcode/internal/source"

type Expr interface {
	IRExpr()
	ExprMeta() ExprBase
}

type ExprBase struct {
	Span      source.Span
	Type      Type
	NullState NullState
}

func (base ExprBase) ExprMeta() ExprBase { return base }

type LiteralKind string

const (
	IntLiteral    LiteralKind = "Int"
	RealLiteral   LiteralKind = "Real"
	BoolLiteral   LiteralKind = "Bool"
	StringLiteral LiteralKind = "String"
)

type LiteralExpr struct {
	ExprBase
	Kind  LiteralKind
	Value string
}

func (*LiteralExpr) IRExpr() {}

type NullExpr struct{ ExprBase }

func (*NullExpr) IRExpr() {}

type LoadExpr struct {
	ExprBase
	Symbol SymbolID
}

func (*LoadExpr) IRExpr() {}

type ClassRefExpr struct {
	ExprBase
	Class ClassID
}

func (*ClassRefExpr) IRExpr() {}

type FunctionValueExpr struct {
	ExprBase
	Symbol   SymbolID
	Callable CallableID
	// Captures are the values bound into a capturing lambda's closure, in the
	// callable's leading-parameter order. They are evaluated once where the
	// lambda value is created, so the closure holds the captured values rather
	// than a live view of the enclosing bindings. Empty for every ordinary
	// Function value.
	Captures []Expr
}

func (*FunctionValueExpr) IRExpr() {}

type UnaryOp string
type BinaryOp string

type UnaryExpr struct {
	ExprBase
	Op      UnaryOp
	Operand Expr
}

func (*UnaryExpr) IRExpr() {}

type BinaryExpr struct {
	ExprBase
	Op          BinaryOp
	Left, Right Expr
}

func (*BinaryExpr) IRExpr() {}

type ConvertExpr struct {
	ExprBase
	From  Type
	Value Expr
}

func (*ConvertExpr) IRExpr() {}

type Argument struct {
	ParameterIndex int
	ParameterName  string
	Value          Expr
	UsesDefault    bool
}

type CallExpr struct {
	ExprBase
	Callable   CallableID
	Callee     Expr // non-nil for indirect/method calls
	Arguments  []Argument
	ReturnNull NullState
}

func (*CallExpr) IRExpr() {}

type ConstructExpr struct {
	ExprBase
	Class       ClassID
	Constructor CallableID
	Arguments   []Argument
}

func (*ConstructExpr) IRExpr() {}

type MemberKind string

const (
	FieldMember  MemberKind = "Field"
	MethodMember MemberKind = "Method"
)

type MemberExpr struct {
	ExprBase
	Kind   MemberKind
	Object Expr
	Field  FieldID
	// Direct suppresses dynamic dispatch: the named Callable is invoked on the
	// receiver exactly as written, which is how SuperClass.member is resolved.
	Direct   bool
	Callable CallableID
}

func (*MemberExpr) IRExpr() {}

type IndexExpr struct {
	ExprBase
	Object Expr
	Index  Expr
}

func (*IndexExpr) IRExpr() {}

type SliceExpr struct {
	ExprBase
	Object Expr
	Start  Expr
	End    Expr
}

func (*SliceExpr) IRExpr() {}

type ListExpr struct {
	ExprBase
	ElementType Type
	Elements    []Expr
}

func (*ListExpr) IRExpr() {}

type PairEntry struct{ Key, Value Expr }
type PairExpr struct {
	ExprBase
	KeyType, ValueType Type
	Entries            []PairEntry
}

func (*PairExpr) IRExpr() {}

type StringPart struct {
	Literal  string
	ToString Expr
}
type StringExpr struct {
	ExprBase
	Parts []StringPart
}

func (*StringExpr) IRExpr() {}

type ToStringExpr struct {
	ExprBase
	Value Expr
}

func (*ToStringExpr) IRExpr() {}

// IdentityExpr is the runtime-managed identity number of a List, Pair, or
// Class instance, produced by the id() Fundamental.
type IdentityExpr struct {
	ExprBase
	Value Expr
}

func (*IdentityExpr) IRExpr() {}

// TypeNameExpr is the canonical AhdCode type name produced by the type()
// Fundamental. StaticName is the compile-time canonical name for every
// non-Class, non-null case; IsClass requests the most-derived runtime Class
// name instead, and a null Value always renders "Null" regardless of either.
type TypeNameExpr struct {
	ExprBase
	Value      Expr
	StaticName string
	IsClass    bool
}

func (*TypeNameExpr) IRExpr() {}
