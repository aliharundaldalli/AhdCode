package semantic

import (
	"fmt"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/parser"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

type callableKind uint8

const (
	functionCallable callableKind = iota
	structureCallable
)

type callableContext struct {
	kind       callableKind
	symbol     *Symbol
	callable   *Callable
	returnType types.Type
	class      *Symbol
	returnNull []NullState
	inferences []*Symbol
}

type analyzer struct {
	bag         diagnostics.Bag
	environment Environment

	result Result
	module *scope
	flow   flowState

	classes       map[string]*Symbol
	classByType   map[*types.ClassSymbol]*Symbol
	classNodes    map[*ast.ClassDecl]*Symbol
	functionNodes map[*ast.FunctionDecl]*Callable
	functionOwner map[*ast.FunctionDecl]*Symbol

	loopDepth        int
	moduleInferences []*Symbol
}

// Analyzer is the reusable parser.Result -> semantic.Result entry point.
// Per-run mutable state remains isolated in an internal engine.
type Analyzer struct{}

func NewAnalyzer() *Analyzer { return &Analyzer{} }

// Analyze is a convenience entry point for a single semantic run.
func Analyze(parsed parser.Result) Result { return NewAnalyzer().Analyze(parsed) }

// Analyze consumes a parser result and produces side-table semantic data. It
// tolerates recovered/Bad AST nodes and does not mutate parser output.
func (*Analyzer) Analyze(parsed parser.Result) Result {
	return analyzeWithEnvironment(parsed, Environment{})
}

func AnalyzeWithEnvironment(parsed parser.Result, environment Environment) Result {
	return NewAnalyzer().AnalyzeWithEnvironment(parsed, environment)
}

func (*Analyzer) AnalyzeWithEnvironment(parsed parser.Result, environment Environment) Result {
	return analyzeWithEnvironment(parsed, environment)
}

func analyzeWithEnvironment(parsed parser.Result, environment Environment) Result {
	a := &analyzer{
		environment:   environment,
		classes:       make(map[string]*Symbol),
		classByType:   make(map[*types.ClassSymbol]*Symbol),
		classNodes:    make(map[*ast.ClassDecl]*Symbol),
		functionNodes: make(map[*ast.FunctionDecl]*Callable),
		functionOwner: make(map[*ast.FunctionDecl]*Symbol),
		flow:          make(flowState),
	}
	a.result.ResolvedSymbols = make(map[ast.Node]*Symbol)
	a.result.ExpressionTypes = make(map[ast.Expr]types.Type)
	a.result.NullStates = make(map[ast.Expr]NullState)
	a.result.SelectedCallables = make(map[*ast.CallExpr]*Callable)
	a.result.SelectedFunctionValues = make(map[ast.Expr]*Callable)
	a.result.OverloadResolutions = make(map[*ast.CallExpr]ResolutionTrace)
	a.module = newScope(nil, moduleScope)
	a.installBuiltins()
	if parsed.Program == nil {
		a.result.Diagnostics = a.bag.Items()
		return a.result
	}

	a.installImports(parsed.Program)
	a.predeclareClasses(parsed.Program)
	a.resolveClassParents(parsed.Program)
	a.predeclareFunctions(parsed.Program)
	a.predeclareModuleConstants(parsed.Program)
	a.predeclareClassMembers(parsed.Program)

	// Module statements are analyzed in source order. Function/Class headers are
	// predeclared for reference identity, while their bodies are checked at the
	// declaration point so inferred null metadata is available to later calls.
	for _, statement := range parsed.Program.Statements {
		outcome := a.analyzeStatement(statement, a.module, a.flow)
		a.flow = outcome.flow
	}
	a.finalizeInferences(a.moduleInferences)

	a.result.Diagnostics = a.bag.Items()
	return a.result
}

func (a *analyzer) predeclareModuleConstants(program *ast.Program) {
	for _, statement := range program.Statements {
		declaration, ok := statement.(*ast.VariableDecl)
		if !ok || !hasModifier(declaration.Modifiers, ast.ModifierConstant) || declaration.Name == "" {
			continue
		}
		if existing, exists := a.module.local(declaration.Name); exists {
			a.error(codeRedeclaration, fmt.Sprintf("%q is already declared in module scope", declaration.Name), declaration.Span(), fmt.Sprintf("previous declaration is %s", types.Display(existing.Type)))
			continue
		}
		typeValue := a.resolveType(declaration.Type)
		symbol := &Symbol{
			Name: declaration.Name, Kind: BindingSymbol, Type: typeValue,
			Span: declaration.Span(), Declaration: declaration, Constant: true,
			Confidential: hasModifier(declaration.Modifiers, ast.ModifierConfidential),
			ModuleRoot:   true, InitialNull: NonNull, OriginModuleID: a.environment.ModuleID,
		}
		a.module.symbols[symbol.Name] = symbol
		a.result.ResolvedSymbols[declaration] = symbol
		a.result.Symbols = append(a.result.Symbols, symbol)
	}
}

func (a *analyzer) installBuiltins() {
	objectType := &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}
	errorType := &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error", Parent: objectType}
	object := &Symbol{Name: "Object", Kind: ClassSymbol, Class: objectType, Type: types.Class{Symbol: objectType, Reference: true}, ModuleRoot: true, Builtin: true, InitialNull: NonNull, Members: make(map[string]*Symbol)}
	errorSymbol := &Symbol{Name: "Error", Kind: ClassSymbol, Class: errorType, Type: types.Class{Symbol: errorType, Reference: true}, ModuleRoot: true, Builtin: true, InitialNull: NonNull, Members: make(map[string]*Symbol)}
	errorSymbol.Members["message"] = &Symbol{Name: "message", Kind: MemberSymbol, Type: types.String, Builtin: true, InitialNull: NonNull}
	a.addBuiltin(object)
	a.addBuiltin(errorSymbol)
	a.classes[object.Name] = object
	a.classes[errorSymbol.Name] = errorSymbol
	a.classByType[objectType] = object
	a.classByType[errorType] = errorSymbol
	for _, name := range []string{"write", "str"} {
		a.addBuiltin(&Symbol{Name: name, Kind: BuiltinSymbol, Type: types.Function{}, ModuleRoot: true, Builtin: true, InitialNull: NonNull})
	}
}

func (a *analyzer) addBuiltin(symbol *Symbol) {
	a.module.symbols[symbol.Name] = symbol
	a.result.Symbols = append(a.result.Symbols, symbol)
}

func (a *analyzer) predeclareClasses(program *ast.Program) {
	for _, statement := range program.Statements {
		declaration, ok := statement.(*ast.ClassDecl)
		if !ok {
			continue
		}
		if existing, exists := a.module.local(declaration.Name); exists {
			a.error(codeRedeclaration, fmt.Sprintf("%q is already declared in module scope", declaration.Name), declaration.Span(), fmt.Sprintf("previous declaration is %s", types.Display(existing.Type)))
			continue
		}
		identity := &types.ClassSymbol{ModuleID: a.environment.ModuleID, Name: declaration.Name}
		symbol := &Symbol{
			Name: declaration.Name, Kind: ClassSymbol,
			Type: types.Class{Symbol: identity, Reference: true}, Class: identity,
			Span: declaration.Span(), Declaration: declaration, ModuleRoot: true,
			Confidential:   hasModifier(declaration.Modifiers, ast.ModifierConfidential),
			OriginModuleID: a.environment.ModuleID,
			InitialNull:    NonNull, Members: make(map[string]*Symbol),
		}
		a.module.symbols[symbol.Name] = symbol
		a.classes[symbol.Name] = symbol
		a.classByType[identity] = symbol
		a.classNodes[declaration] = symbol
		a.result.ResolvedSymbols[declaration] = symbol
		a.result.Symbols = append(a.result.Symbols, symbol)
	}
}

func (a *analyzer) resolveClassParents(program *ast.Program) {
	object := a.classes["Object"]
	for _, statement := range program.Statements {
		declaration, ok := statement.(*ast.ClassDecl)
		if !ok {
			continue
		}
		symbol := a.classNodes[declaration]
		if symbol == nil {
			continue
		}
		parent := object
		if declaration.Parent != nil {
			parent = a.classes[declaration.Parent.Name]
			if parent == nil {
				a.error(codeInvalidType, fmt.Sprintf("unknown parent Class %q", declaration.Parent.Name), declaration.Parent.Span(), "declare the parent Class in this module or bring it from another module")
				parent = object
			} else {
				a.result.ResolvedSymbols[declaration.Parent] = parent
			}
		}
		if parent == symbol {
			a.error(codeInvalidType, "a Class cannot inherit from itself", declaration.Span(), "choose a different direct parent")
			parent = object
		}
		if parent != nil {
			symbol.Class.Parent = parent.Class
		}
	}
	for _, symbol := range a.classes {
		if symbol.Builtin || symbol.Class == nil {
			continue
		}
		seen := make(map[*types.ClassSymbol]bool)
		for current := symbol.Class; current != nil; current = current.Parent {
			if seen[current] {
				a.error(codeInvalidType, fmt.Sprintf("inheritance cycle involving Class %q", symbol.Name), symbol.Span, "break the Class parent cycle")
				symbol.Class.Parent = object.Class
				break
			}
			seen[current] = true
		}
	}
}

func (a *analyzer) predeclareFunctions(program *ast.Program) {
	for _, statement := range program.Statements {
		declaration, ok := statement.(*ast.FunctionDecl)
		if !ok {
			continue
		}
		a.registerFunction(declaration, a.module, nil)
	}
}

func (a *analyzer) predeclareClassMembers(program *ast.Program) {
	for _, statement := range program.Statements {
		declaration, ok := statement.(*ast.ClassDecl)
		if !ok {
			continue
		}
		class := a.classNodes[declaration]
		if class == nil {
			continue
		}
		for _, member := range declaration.Members {
			switch value := member.(type) {
			case *ast.StructureDecl:
				class.Constructor = a.callableFromStructure(value)
				for index := range value.Parameters {
					parameter := &value.Parameters[index]
					if parameter.InheritedAttributes || hasModifier(parameter.Modifiers, ast.ModifierLocal) {
						continue
					}
					typeValue := a.resolveType(parameter.Type)
					nullState := NonNull
					if parameter.Default != nil {
						nullState = a.initializerNullState(parameter.Default)
					}
					memberSymbol := &Symbol{Name: parameter.Name, Kind: MemberSymbol, Type: typeValue, Span: parameter.Span(), InitialNull: nullState, Confidential: hasModifier(parameter.Modifiers, ast.ModifierConfidential), OwnerClass: class.Class, OriginModuleID: a.environment.ModuleID}
					class.Members[parameter.Name] = memberSymbol
					a.result.Symbols = append(a.result.Symbols, memberSymbol)
				}
			case *ast.FunctionDecl:
				a.registerFunction(value, nil, class)
			}
		}
		if class.Constructor == nil {
			class.Constructor = &Callable{Signature: &types.Signature{Return: types.Class{Symbol: class.Class}}, ReturnNull: NonNull}
		}
	}
}

func (a *analyzer) registerFunction(declaration *ast.FunctionDecl, targetScope *scope, owner *Symbol) {
	callable := a.callableFromFunction(declaration)
	var existing *Symbol
	if owner != nil {
		existing = owner.Members[declaration.Name]
	} else if targetScope != nil {
		existing, _ = targetScope.local(declaration.Name)
	}
	if existing != nil {
		if existing.Kind == FunctionSymbol && declaration.Flavor == ast.FunctionOverload {
			if existing.OverloadSet == nil {
				existing.OverloadSet = &OverloadSet{Name: existing.Name, Candidates: []*Callable{existing.Callable}}
			}
			for _, candidate := range existing.OverloadSet.Candidates {
				if sameOverloadKey(candidate.Signature, callable.Signature) {
					a.error(codeInvalidOverload, fmt.Sprintf("overload %q duplicates parameter signature %s", declaration.Name, formatSignature(callable.Signature)), declaration.Span(), "overloads must differ by parameter count and/or parameter type; return type alone cannot distinguish them")
					a.functionNodes[declaration] = callable
					a.functionOwner[declaration] = existing
					a.result.ResolvedSymbols[declaration] = existing
					return
				}
			}
			existing.OverloadSet.Candidates = append(existing.OverloadSet.Candidates, callable)
			a.functionNodes[declaration] = callable
			a.functionOwner[declaration] = existing
			a.result.ResolvedSymbols[declaration] = existing
			return
		}
		a.error(codeRedeclaration, fmt.Sprintf("%q is already declared in this scope", declaration.Name), declaration.Span(), "use Overload Function only for an intentional overload")
		return
	}
	symbol := &Symbol{
		Name: declaration.Name, Kind: FunctionSymbol, Type: types.Function{Signature: callable.Signature},
		Span: declaration.Span(), Declaration: declaration, Callable: callable,
		ModuleRoot: owner == nil, InitialNull: NonNull,
		Constant: true, Confidential: hasModifier(declaration.Modifiers, ast.ModifierConfidential),
		OriginModuleID: a.environment.ModuleID,
	}
	if owner != nil {
		symbol.OwnerClass = owner.Class
	}
	symbol.OverloadSet = &OverloadSet{Name: symbol.Name, Candidates: []*Callable{callable}}
	if owner != nil {
		owner.Members[symbol.Name] = symbol
	} else {
		targetScope.symbols[symbol.Name] = symbol
	}
	a.functionNodes[declaration] = callable
	a.functionOwner[declaration] = symbol
	a.result.ResolvedSymbols[declaration] = symbol
	a.result.Symbols = append(a.result.Symbols, symbol)
}

func sameOverloadKey(left, right *types.Signature) bool {
	if left == nil || right == nil || len(left.Parameters) != len(right.Parameters) {
		return false
	}
	for index := range left.Parameters {
		if !types.Equal(left.Parameters[index].Type, right.Parameters[index].Type) {
			return false
		}
	}
	return true
}

func (a *analyzer) callableFromFunction(declaration *ast.FunctionDecl) *Callable {
	parameters := make([]types.Parameter, 0, len(declaration.Parameters))
	parameterNull := make([]NullState, 0, len(declaration.Parameters))
	for index := range declaration.Parameters {
		parameter := &declaration.Parameters[index]
		parameters = append(parameters, types.Parameter{Name: parameter.Name, Type: a.resolveType(parameter.Type), HasDefault: parameter.Default != nil})
		parameterNull = append(parameterNull, NonNull)
	}
	returnType := a.resolveType(declaration.ReturnType)
	return &Callable{Signature: &types.Signature{Parameters: parameters, Return: returnType}, ParameterNull: parameterNull, ReturnNull: NonNull, Declaration: declaration}
}

func (a *analyzer) callableFromStructure(declaration *ast.StructureDecl) *Callable {
	var parameters []types.Parameter
	var parameterNull []NullState
	for index := range declaration.Parameters {
		parameter := &declaration.Parameters[index]
		if parameter.InheritedAttributes {
			continue
		}
		parameters = append(parameters, types.Parameter{Name: parameter.Name, Type: a.resolveType(parameter.Type), HasDefault: parameter.Default != nil})
		parameterNull = append(parameterNull, NonNull)
	}
	return &Callable{Signature: &types.Signature{Parameters: parameters, Return: types.Nothing}, ParameterNull: parameterNull, ReturnNull: NonNull, Structure: declaration}
}

func (a *analyzer) resolveType(reference *ast.TypeRef) types.Type {
	if reference == nil {
		return types.Invalid
	}
	noArguments := func(result types.Type) types.Type {
		if len(reference.Arguments) != 0 {
			a.error(codeInvalidType, fmt.Sprintf("%s does not accept generic arguments", reference.Name), reference.Span(), fmt.Sprintf("received %d type argument(s)", len(reference.Arguments)))
		}
		return result
	}
	switch reference.Name {
	case "Int":
		return noArguments(types.Int)
	case "Real":
		return noArguments(types.Real)
	case "String":
		return noArguments(types.String)
	case "Bool":
		return noArguments(types.Bool)
	case "Nothing":
		return noArguments(types.Nothing)
	case "Function":
		return noArguments(types.Function{})
	case "List":
		if len(reference.Arguments) != 1 {
			a.error(codeInvalidType, "List requires exactly one type argument", reference.Span(), fmt.Sprintf("received %d", len(reference.Arguments)))
			return types.List{Element: types.Invalid}
		}
		return types.List{Element: a.resolveType(reference.Arguments[0])}
	case "Pair":
		if len(reference.Arguments) != 2 {
			a.error(codeInvalidType, "Pair requires exactly two type arguments", reference.Span(), fmt.Sprintf("received %d", len(reference.Arguments)))
			return types.Pair{Key: types.Invalid, Value: types.Invalid}
		}
		return types.Pair{Key: a.resolveType(reference.Arguments[0]), Value: a.resolveType(reference.Arguments[1])}
	default:
		if class := a.classes[reference.Name]; class != nil {
			if len(reference.Arguments) != 0 {
				a.error(codeInvalidType, fmt.Sprintf("Class %s does not accept type arguments", reference.Name), reference.Span(), "remove generic arguments")
			}
			a.result.ResolvedSymbols[reference] = class
			return types.Class{Symbol: class.Class}
		}
		a.error(codeInvalidType, fmt.Sprintf("unknown type %q", reference.Name), reference.Span(), "declare the Class or use an AhdCode built-in type")
		return types.Invalid
	}
}

func (a *analyzer) classSymbolFor(identity *types.ClassSymbol) *Symbol {
	if identity == nil {
		return nil
	}
	if symbol := a.classByType[identity]; symbol != nil {
		return symbol
	}
	for candidate, symbol := range a.classByType {
		if types.SameClassIdentity(candidate, identity) {
			a.classByType[identity] = symbol
			return symbol
		}
	}
	return nil
}

func (a *analyzer) error(code, message string, span source.Span, hint string) {
	a.bag.Error(code, message, span, hint)
}

func hasModifier(modifiers []ast.Modifier, wanted ast.Modifier) bool {
	for _, modifier := range modifiers {
		if modifier == wanted {
			return true
		}
	}
	return false
}
