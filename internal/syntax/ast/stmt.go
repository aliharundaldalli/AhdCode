package ast

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
	Name     string
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
	Module    string
	Namespace bool
	All       bool
	Names     []string
}

func (*BringStmt) statementNode() {}

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
