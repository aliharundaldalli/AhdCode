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
	// Overrides is the parent method this method replaces. It carries the
	// frontend's Override decision so a backend never has to rediscover the
	// dispatch slot from names.
	Overrides CallableID
	// ParentConstructor and ParentArguments describe how a constructor
	// initializes its inherited attribute slots. ParentArguments holds this
	// constructor's parameter indices, in parent parameter order.
	ParentConstructor CallableID
	ParentArguments   []int
	Parameters        []Parameter
	Body              Block
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
	// Builtin marks a Class supplied by the language rather than declared in
	// AhdCode source, such as Object, Error, and the runtime Error subclasses.
	Builtin     bool
	Fields      []Field
	Constructor CallableID
	Methods     []CallableID
}
