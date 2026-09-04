package ast

import "ahdcode/internal/source"

// Stmt is a typed statement/declaration node.
type Stmt interface {
	Node
	statementNode()
}

type Block struct {
	Base
	Statements []Stmt
}

type ExprStmt struct {
	Base
	Expression Expr
}

func (*ExprStmt) statementNode() {}

type VariableDecl struct {
	Base
	Target      Expr
	Name        string
	Modifiers   []Modifier
	Type        *TypeRef
	Initializer Expr
	GlobalOnly  bool
	// Inferred marks a `name := value` declaration written without an
	// explicit `: Type`. Type is nil in that case; the static type is the
	// initializer's complete type and nullability, computed during semantic
	// analysis.
	Inferred bool
}

func (*VariableDecl) statementNode() {}

type AssignmentStmt struct {
	Base
	Target   Expr
	Operator string
	Value    Expr
}

func (*AssignmentStmt) statementNode() {}

type IncDecStmt struct {
	Base
	Target   Expr
	Operator string
	Prefix   bool
}

func (*IncDecStmt) statementNode() {}

type ReturnStmt struct {
	Base
	Value Expr
}

func (*ReturnStmt) statementNode() {}

type TossStmt struct {
	Base
	Value Expr
}

func (*TossStmt) statementNode() {}

type BreakStmt struct{ Base }

func (*BreakStmt) statementNode() {}

type ContinueStmt struct{ Base }

func (*ContinueStmt) statementNode() {}

type ConditionalBlock struct {
	Base
	Condition Expr
	Body      *Block
}

type IfStmt struct {
	Base
	Branches []ConditionalBlock
	Else     *Block
}

func (*IfStmt) statementNode() {}

type WhileStmt struct {
	Base
	Condition Expr
	Body      *Block
}

func (*WhileStmt) statementNode() {}

type UntilStmt struct {
	Base
	Condition Expr
	Body      *Block
}

func (*UntilStmt) statementNode() {}

type ForStmt struct {
	Base
	Name string
	// Type is the optional explicit iteration binding type. The binding stays
	// implicitly Local, so no scope modifier is accepted here.
	Type     *TypeRef
	Iterable Expr
	Body     *Block
}

func (*ForStmt) statementNode() {}

type StateCondition struct {
	Base
	Match   Expr
	Default bool
	Body    *Block
}

type StateStmt struct {
	Base
	Value      Expr
	Conditions []StateCondition
}

func (*StateStmt) statementNode() {}

type ExceptClause struct {
	Base
	Type *TypeRef
	Name string
	Body *Block
}

type AttemptStmt struct {
	Base
	Body       *Block
	Excepts    []ExceptClause
	Ultimately *Block
}

func (*AttemptStmt) statementNode() {}

type BringStmt struct {
	Base
	Module string
	// Alias is the optional namespace binding written by
	// `bring Module as Alias`. It is empty for every existing bring form.
	Alias     string
	Namespace bool
	All       bool
	Names     []string
}

func (*BringStmt) statementNode() {}

// RequireStmt composes another local .ahd source file into this program at
// compile time. Path is the literal string content (never re-parsed as an
// expression); HasLiteralPath is false when the parser could not extract a
// single unparameterized string literal, so later passes can skip resolution
// entirely instead of chasing an empty path.
type RequireStmt struct {
	Base
	Path           string
	PathSpan       source.Span
	HasLiteralPath bool
}

func (*RequireStmt) statementNode() {}

type FunctionFlavor uint8

const (
	FunctionBase FunctionFlavor = iota
	FunctionOverload
	FunctionOverride
)

type FunctionDecl struct {
	Base
	Name       string
	Modifiers  []Modifier
	Flavor     FunctionFlavor
	Parameters []Parameter
	ReturnType *TypeRef
	Body       *Block
}

func (*FunctionDecl) statementNode() {}

type ClassDecl struct {
	Base
	Name         string
	Modifiers    []Modifier
	Parent       *TypeRef
	ExplicitRoot bool
	Members      []Stmt
}

func (*ClassDecl) statementNode() {}

type StructureDecl struct {
	Base
	Parameters []Parameter
	Body       *Block
}

func (*StructureDecl) statementNode() {}
