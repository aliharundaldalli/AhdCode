package semantic

import (
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// NullState is flow-sensitive and deliberately orthogonal to types.Type.
type NullState uint8

const (
	MaybeNull NullState = iota
	Null
	NonNull
)

func (state NullState) String() string {
	switch state {
	case Null:
		return "Null"
	case NonNull:
		return "NonNull"
	default:
		return "MaybeNull"
	}
}

type SymbolKind uint8

const (
	BindingSymbol SymbolKind = iota
	ParameterSymbol
	FunctionSymbol
	ClassSymbol
	MemberSymbol
	ForSymbol
	ExceptSymbol
	BuiltinSymbol
	NamespaceSymbol
)

// Callable preserves concrete signatures and null-state metadata separately;
// it can later be serialized as a cross-module contract.
type Callable struct {
	Signature     *types.Signature
	ParameterNull []NullState
	ReturnNull    NullState
	Declaration   *ast.FunctionDecl
	Structure     *ast.StructureDecl
}

// OverloadSet is distinct from a single Function type. Candidates remain
// concrete callables and declaration order has no ranking meaning.
type OverloadSet struct {
	Name       string
	Candidates []*Callable
}

// CandidateDecision records an overload applicability result without exposing
// raw Go/internal type dumps.
type CandidateDecision struct {
	Signature  string
	Applicable bool
	Reason     string
	Widenings  int
	Defaults   int
}

type ResolutionTrace struct {
	Candidates []CandidateDecision
	Selected   string
}

// Symbol is a resolved semantic declaration. Alias points from an explicit
// Global declaration to its module-root binding.
type Symbol struct {
	Name         string
	Kind         SymbolKind
	Type         types.Type
	Span         source.Span
	Declaration  ast.Node
	Constant     bool
	Confidential bool
	ModuleRoot   bool
	Builtin      bool
	// SuperClassBinding marks the implicit SuperClass binding installed inside
	// a Class callable. It designates the parent implementation rather than an
	// ordinary Class reference value.
	SuperClassBinding bool
	InitialNull       NullState
	Alias             *Symbol
	Callable          *Callable
	OverloadSet       *OverloadSet
	Namespace         *ModuleInterface
	Class             *types.ClassSymbol
	OwnerClass        *types.ClassSymbol
	OriginModuleID    string
	Members           map[string]*Symbol
	Constructor       *Callable
	// ConstructorAttributes records, per constructor parameter, the instance
	// attribute that parameter initializes. A Local structure parameter has a
	// nil entry. Inherited entries come from the parent construction contract.
	ConstructorAttributes []*Symbol
	ConstValue            *constantValue
	inference             *functionInference
}

// ModuleInterface is an in-memory, compile-time-only public contract. Identity
// is canonical and never derived from pointer addresses.
type ModuleInterface struct {
	ModuleID    string
	Name        string
	Exports     map[string]*Symbol
	Symbols     map[string]*Symbol // includes Confidential entries for access diagnostics
	Classes     map[string]*Symbol // canonical identity -> Class metadata, including ancestry support
	ExportNames []string
}

// CollectionOperation is one built-in collection mutation. These are typed
// language operations rather than dynamic member calls.
type CollectionOperation string

const (
	// ListAdd appends one element to the end of a List.
	ListAdd CollectionOperation = "List.add"
	// ListEject removes the element at an index, which may be negative.
	ListEject CollectionOperation = "List.eject"
	// PairEject removes one key and its value from a Pair.
	PairEject CollectionOperation = "Pair.eject"
)

// Environment supplies already-resolved dependency interfaces to one semantic
// analysis run. Filesystem/module graph work stays outside this package.
type Environment struct {
	ModuleID      string
	ModuleName    string
	Imports       map[string]*ModuleInterface
	FailedImports map[string]bool
}

// Result is a side-table semantic model; Analyze never mutates the AST.
type Result struct {
	Diagnostics            []diagnostics.Diagnostic
	Symbols                []*Symbol
	ResolvedSymbols        map[ast.Node]*Symbol
	ExpressionTypes        map[ast.Expr]types.Type
	NullStates             map[ast.Expr]NullState
	SelectedCallables      map[*ast.CallExpr]*Callable
	SelectedFunctionValues map[ast.Expr]*Callable
	OverloadResolutions    map[*ast.CallExpr]ResolutionTrace
	// SuperCalls marks member expressions written as SuperClass.member, which
	// bind the current instance but call the parent implementation directly.
	SuperCalls map[ast.Expr]bool
	// CollectionCalls records the built-in List and Pair mutation operations,
	// so lowering never has to rediscover them from member names.
	CollectionCalls map[*ast.CallExpr]CollectionOperation
}

func (result Result) HasErrors() bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}
