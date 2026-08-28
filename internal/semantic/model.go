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
	InitialNull  NullState
	Alias        *Symbol
	Callable     *Callable
	OverloadSet  *OverloadSet
	Class        *types.ClassSymbol
	Members      map[string]*Symbol
	Constructor  *Callable
	ConstValue   *constantValue
	inference    *functionInference
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
}

func (result Result) HasErrors() bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}
