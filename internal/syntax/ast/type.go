package ast

// TypeRef is a syntactic type reference. It deliberately carries no resolved
// symbol or semantic type information.
type TypeRef struct {
	Base
	Name          string
	Arguments     []*TypeRef
	ExplicitEmpty bool
	// Nullable is true when this exact type reference was written with a
	// trailing `?` (e.g. Int?, List<Int?>?). It only describes this node's
	// own position; a generic argument's nullability is its own TypeRef.
	Nullable bool
}

// Modifier is a declaration modifier as written in source.
type Modifier string

const (
	ModifierLocal        Modifier = "Local"
	ModifierGlobal       Modifier = "Global"
	ModifierConstant     Modifier = "Constant"
	ModifierConfidential Modifier = "Confidential"
)

// Parameter is Function or structure parameter syntax.
type Parameter struct {
	Base
	Name                string
	Modifiers           []Modifier
	Type                *TypeRef
	Default             Expr
	InheritedAttributes bool
}
