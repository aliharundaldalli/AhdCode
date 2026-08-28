package ir

import "ahdcode/internal/source"

type Compilation struct {
	Entry   ModuleID
	Modules []*Module
}

type Module struct {
	ID           ModuleID
	Name         string
	SourcePath   string
	Dependencies []ModuleID
	Globals      []*Global
	Functions    []*Function
	Classes      []*Class
	Init         Block
}

type Global struct {
	Span source.Span
	ID   SymbolID
	Name string
	// Order is the module-source declaration index. Globals are stored sorted
	// by ID for deterministic output, so initialization order lives here
	// instead of in slice position.
	Order        int
	Type         Type
	Constant     bool
	Confidential bool
	NullState    NullState
	Initializer  Expr
}

type Parameter struct {
	Span      source.Span
	ID        SymbolID
	Name      string
	Type      Type
	NullState NullState
	Default   Expr
}

type FunctionKind string

const (
	OrdinaryFunction    FunctionKind = "Function"
	MethodFunction      FunctionKind = "Method"
	ConstructorFunction FunctionKind = "Constructor"
)

type Function struct {
	Span         source.Span
	ID           CallableID
	Symbol       SymbolID
	Name         string
	Kind         FunctionKind
	Owner        ClassID
	Receiver     SymbolID
	Signature    Signature
	ReturnNull   NullState
	Confidential bool
	Override     bool
	Parameters   []Parameter
	Body         Block
}

type Field struct {
	ID           FieldID
	Name         string
	Type         Type
	NullState    NullState
	Constant     bool
	Confidential bool
}

type Class struct {
	Span         source.Span
	ID           ClassID
	Symbol       SymbolID
	Name         string
	Parent       ClassID
	Confidential bool
	Fields       []Field
	Constructor  CallableID
	Methods      []CallableID
}
