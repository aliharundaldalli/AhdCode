package ir

import "ahdcode/internal/source"

type Statement interface {
	IRStatement()
	StatementSpan() source.Span
}

type StmtBase struct{ Span source.Span }

func (base StmtBase) StatementSpan() source.Span { return base.Span }

type Block struct {
	Span       source.Span
	Statements []Statement
}

type ExprStmt struct {
	StmtBase
	Value Expr
}

func (*ExprStmt) IRStatement() {}

type Storage string

const (
	ModuleStorage    Storage = "Module"
	LocalStorage     Storage = "Local"
	ParameterStorage Storage = "Parameter"
	IterationStorage Storage = "Iteration"
	ErrorStorage     Storage = "Error"
)

type BindingStmt struct {
	StmtBase
	Symbol SymbolID
	Name   string
	Type   Type
	// NullState is the declared storage null-state of the binding slot. It is
	// declaration-level metadata, not the flow-sensitive state of a use site.
	NullState   NullState
	Constant    bool
	Storage     Storage
	Initializer Expr
}

func (*BindingStmt) IRStatement() {}

type TargetKind string

const (
	SymbolTarget TargetKind = "Symbol"
	FieldTarget  TargetKind = "Field"
	IndexTarget  TargetKind = "Index"
)

// Target owns its receiver/index expressions exactly once. Compound/update
// statements therefore preserve single-evaluation lvalue semantics.
type Target struct {
	Kind     TargetKind
	Type     Type
	Symbol   SymbolID
	Field    FieldID
	Receiver Expr
	Index    Expr
}

type AssignStmt struct {
	StmtBase
	Target Target
	Value  Expr
}

func (*AssignStmt) IRStatement() {}

// CompoundAssignStmt evaluates its Target's receiver/index exactly once, then
// combines the read value with Value and writes the result back. Exactly one
// of Op or Protocol is set: Op selects a built-in binary operation, while
// Protocol selects a Class Protocol Method call on the read value (with Value
// as its sole argument) for a compound assignment resolved to a Class
// instance target -- there is no separate in-place protocol name.
type CompoundAssignStmt struct {
	StmtBase
	Target   Target
	Op       BinaryOp
	Protocol CallableID
	Value    Expr
}

func (*CompoundAssignStmt) IRStatement() {}

type UpdateStmt struct {
	StmtBase
	Target Target
	Delta  int
}

func (*UpdateStmt) IRStatement() {}

type ConditionalBlock struct {
	Condition Expr
	Body      Block
}
type IfStmt struct {
	StmtBase
	Branches []ConditionalBlock
	Else     *Block
}

func (*IfStmt) IRStatement() {}

type WhileStmt struct {
	StmtBase
	Condition Expr
	Body      Block
}

func (*WhileStmt) IRStatement() {}

// DoUntilStmt is explicitly post-check; ContinueChecksCondition documents the
// required continue edge for backend lowering.
type DoUntilStmt struct {
	StmtBase
	Body                    Block
	Condition               Expr
	ContinueChecksCondition bool
}

func (*DoUntilStmt) IRStatement() {}

type IterationKind string

const (
	ListElements     IterationKind = "ListElements"
	StringCharacters IterationKind = "StringCharacters"
	PairKeys         IterationKind = "PairKeys"
	// IntRange iterates a lazy integer range. It has no backing collection, so
	// it needs no shallow snapshot.
	IntRange IterationKind = "IntRange"
)

type ForStmt struct {
	StmtBase
	Iteration     SymbolID
	IterationType Type
	Name          string
	Kind          IterationKind
	Iterable      Expr
	// Snapshot marks the shallow copy taken at loop entry. Collection
	// iteration always snapshots; a lazy range has no collection to copy.
	Snapshot bool
	Body     Block
}

func (*ForStmt) IRStatement() {}

type StateCase struct {
	Match   Expr
	Default bool
	Body    Block
}
type StateStmt struct {
	StmtBase
	Temp          TempID
	Value         Expr
	Cases         []StateCase
	NoFallthrough bool
}

func (*StateStmt) IRStatement() {}

type ErrorHandler struct {
	Class   ClassID
	Binding SymbolID
	Body    Block
}
type AttemptStmt struct {
	StmtBase
	Body          Block
	Handlers      []ErrorHandler
	Ultimately    *Block
	FinallyAlways bool
}

func (*AttemptStmt) IRStatement() {}

type TossStmt struct {
	StmtBase
	Value      Expr
	ErrorClass ClassID
}

func (*TossStmt) IRStatement() {}

type ReturnStmt struct {
	StmtBase
	Value      Expr
	ReturnType Type
}

func (*ReturnStmt) IRStatement() {}

type BreakStmt struct{ StmtBase }

func (*BreakStmt) IRStatement() {}

type ContinueStmt struct{ StmtBase }

func (*ContinueStmt) IRStatement() {}
