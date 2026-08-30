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

// Global is the storage declaration of one module-root binding. It carries no
// initializer: the initializer executes as a ModuleStorage BindingStmt inside
// Module.Init, at its source position among the other module-level statements.
type Global struct {
	Span         source.Span
	ID           SymbolID
	Name         string
	Type         Type
	Constant     bool
	Confidential bool
	NullState    NullState
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
	// Captures is how many leading Parameters are closure captures rather than
	// declared parameters. A lambda's explicit capture list is passed as
	// ordinary typed parameters, so no second environment mechanism exists;
	// Signature still describes only the declared parameters, which is what a
	// caller supplies. It is zero for every non-capturing callable.
	Captures int
	Body     Block
}

type Field struct {
	ID           FieldID
	Name         string
	Type         Type
	NullState    NullState
	Constant     bool
	Confidential bool
	// Hidden marks storage that a compiler-supplied Class needs at runtime but
	// never publishes. The frontend does not declare such a field as an
	// attribute, so reading it is already an unknown-member error; Hidden keeps
	// member existence agreeing with that by excluding it from has / has not.
	// It is not an access-control flag: Confidential covers restricted but
	// genuinely published members.
	Hidden bool
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
	// Operations names the additional members a compiler-supplied Class
	// publishes through built-in type operations rather than through declared
	// fields or methods, so member existence stays truthful for them.
	Operations []string
}
