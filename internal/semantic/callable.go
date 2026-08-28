package semantic

import (
	"fmt"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

func (a *analyzer) analyzeClass(declaration *ast.ClassDecl) {
	class := a.classNodes[declaration]
	if class == nil {
		return
	}
	for _, member := range declaration.Members {
		if structure, ok := member.(*ast.StructureDecl); ok {
			a.analyzeStructure(structure, class)
		}
	}
	for _, member := range declaration.Members {
		if function, ok := member.(*ast.FunctionDecl); ok {
			a.validateOverride(function, class)
			a.analyzeFunction(function, class)
		}
	}
}

func (a *analyzer) validateOverride(declaration *ast.FunctionDecl, class *Symbol) {
	if class == nil || class.Class == nil || class.Class.Parent == nil {
		if declaration.Flavor == ast.FunctionOverride {
			a.error(codeInvalidMember, fmt.Sprintf("Override Function %q has no inherited target", declaration.Name), declaration.Span(), "override a compatible method from a parent Class")
		}
		return
	}
	parent := a.classSymbolFor(class.Class.Parent)
	inherited := a.lookupMember(parent, declaration.Name)
	if declaration.Flavor == ast.FunctionOverride {
		if inherited == nil || inherited.Callable == nil {
			a.error(codeInvalidMember, fmt.Sprintf("Override Function %q has no inherited target", declaration.Name), declaration.Span(), "override a compatible method from a parent Class")
			return
		}
		callable := a.functionNodes[declaration]
		if callable != nil && !callableSignaturesEqual(callable, inherited.Callable) {
			a.error(codeTypeMismatch, fmt.Sprintf("Override Function %q has an incompatible signature", declaration.Name), declaration.Span(), "match the inherited parameter and return types")
		}
	} else if inherited != nil && inherited.Kind == FunctionSymbol {
		a.error(codeInvalidMember, fmt.Sprintf("method %q collides with an inherited Function", declaration.Name), declaration.Span(), "write Override Function for intentional replacement")
	}
}

func callableSignaturesEqual(left, right *Callable) bool {
	if left == nil || right == nil || left.Signature == nil || right.Signature == nil {
		return false
	}
	return types.Equal(types.Function{Signature: left.Signature}, types.Function{Signature: right.Signature})
}

func (a *analyzer) analyzeFunction(declaration *ast.FunctionDecl, class *Symbol) {
	callable := a.functionNodes[declaration]
	owner := a.functionOwner[declaration]
	if callable == nil || owner == nil {
		return
	}
	functionScope := newScope(a.module, callableScope)
	context := &callableContext{kind: functionCallable, symbol: owner, callable: callable, returnType: callable.Signature.Return, class: class}
	functionScope.callable = context
	if class != nil {
		a.installClassImplicitBindings(functionScope, class)
	}
	flow := a.flow.clone()
	for index := range declaration.Parameters {
		parameter := &declaration.Parameters[index]
		if hasModifier(parameter.Modifiers, ast.ModifierLocal) || hasModifier(parameter.Modifiers, ast.ModifierGlobal) || hasModifier(parameter.Modifiers, ast.ModifierConstant) {
			a.error(codeScopeModifier, fmt.Sprintf("Function parameter %q is implicitly Local", parameter.Name), parameter.Span(), "remove Local, Global, or Constant from the Function parameter")
		}
		if _, exists := functionScope.local(parameter.Name); exists {
			a.error(codeRedeclaration, fmt.Sprintf("duplicate parameter %q", parameter.Name), parameter.Span(), "use a unique parameter name")
			continue
		}
		typeValue := callable.Signature.Parameters[index].Type
		symbol := &Symbol{Name: parameter.Name, Kind: ParameterSymbol, Type: typeValue, Span: parameter.Span(), InitialNull: callable.ParameterNull[index]}
		if function, ok := typeValue.(types.Function); ok && function.Signature == nil {
			symbol.inference = newFunctionInference(callable.Signature, index)
		}
		functionScope.symbols[symbol.Name] = symbol
		flow[symbol] = symbol.InitialNull
		a.result.Symbols = append(a.result.Symbols, symbol)
		a.trackInference(symbol, functionScope)
		if parameter.Default != nil {
			value := a.analyzeExpressionExpected(parameter.Default, functionScope, flow, typeValue)
			if function, ok := value.typeValue.(types.Function); ok && function.Signature != nil && symbol.inference != nil {
				a.constrainConcreteFunction(symbol, function.Signature, concreteCallable(value), parameter.Default.Span())
			}
			if value.nullState != Null && !types.Assignable(typeValue, value.typeValue) {
				a.typeMismatch(parameter.Default.Span(), typeValue, value.typeValue, fmt.Sprintf("default value of %s", parameter.Name))
			}
			if value.nullState != NonNull {
				callable.ParameterNull[index] = MaybeNull
				symbol.InitialNull = MaybeNull
				flow[symbol] = MaybeNull
			}
		}
	}
	outcome := a.analyzeBlock(declaration.Body, functionScope, flow, nil)
	a.finalizeInferences(context.inferences)
	if callable.Signature.Return.Kind() != types.NothingKind && !outcome.returns {
		a.error(codeMissingReturn, fmt.Sprintf("Function %q does not return %s on every reachable path", declaration.Name, types.Display(callable.Signature.Return)), declaration.Span(), "add a compatible return to every reachable path")
	}
	if callable.Signature.Return.Kind() == types.NothingKind {
		callable.ReturnNull = NonNull
	} else if len(context.returnNull) > 0 {
		state := context.returnNull[0]
		for _, next := range context.returnNull[1:] {
			state = mergeNull(state, next)
		}
		callable.ReturnNull = state
	}
}

func (a *analyzer) analyzeStructure(declaration *ast.StructureDecl, class *Symbol) {
	if declaration == nil || class == nil {
		return
	}
	callable := class.Constructor
	if callable == nil || callable.Signature == nil {
		return
	}
	structureScope := newScope(a.module, callableScope)
	context := &callableContext{kind: structureCallable, callable: callable, returnType: types.Nothing, class: class}
	structureScope.callable = context
	a.installClassImplicitBindings(structureScope, class)
	flow := a.flow.clone()
	parameterIndex := 0
	inherited := a.inheritedParameterCount(class)
	for index := range declaration.Parameters {
		parameter := &declaration.Parameters[index]
		if parameter.InheritedAttributes {
			// SuperClass.attributes occupies the parent parameter slots that
			// buildConstructor already spliced into this signature.
			parameterIndex += inherited
			continue
		}
		if parameterIndex >= len(callable.Signature.Parameters) {
			break
		}
		if hasModifier(parameter.Modifiers, ast.ModifierGlobal) {
			a.error(codeScopeModifier, fmt.Sprintf("structure parameter %q is implicitly Local", parameter.Name), parameter.Span(), "use Local only to exclude a structure parameter from instance attributes")
		}
		if _, exists := structureScope.local(parameter.Name); exists {
			a.error(codeRedeclaration, fmt.Sprintf("duplicate structure parameter %q", parameter.Name), parameter.Span(), "use a unique parameter name")
			continue
		}
		typeValue := callable.Signature.Parameters[parameterIndex].Type
		nullState := callable.ParameterNull[parameterIndex]
		callableParameterIndex := parameterIndex
		parameterIndex++
		symbol := &Symbol{Name: parameter.Name, Kind: ParameterSymbol, Type: typeValue, Span: parameter.Span(), InitialNull: nullState}
		if function, ok := typeValue.(types.Function); ok && function.Signature == nil {
			symbol.inference = newFunctionInference(callable.Signature, callableParameterIndex)
		}
		structureScope.symbols[symbol.Name] = symbol
		flow[symbol] = nullState
		a.result.Symbols = append(a.result.Symbols, symbol)
		a.trackInference(symbol, structureScope)
		if parameter.Default != nil {
			value := a.analyzeExpressionExpected(parameter.Default, structureScope, flow, typeValue)
			if value.nullState != Null && !types.Assignable(typeValue, value.typeValue) {
				a.typeMismatch(parameter.Default.Span(), typeValue, value.typeValue, fmt.Sprintf("default value of %s", parameter.Name))
			}
			if value.nullState != NonNull {
				callable.ParameterNull[parameterIndex-1] = MaybeNull
			}
		}
	}
	if declaration.Body != nil {
		a.analyzeBlock(declaration.Body, structureScope, flow, nil)
	}
	a.finalizeInferences(context.inferences)
}

// inheritedParameterCount is the number of parent construction parameters that
// one SuperClass.attributes marker contributes.
func (a *analyzer) inheritedParameterCount(class *Symbol) int {
	if class == nil || class.Class == nil || class.Class.Parent == nil {
		return 0
	}
	parent := a.classSymbolFor(class.Class.Parent)
	if parent == nil || parent.Constructor == nil || parent.Constructor.Signature == nil {
		return 0
	}
	return len(parent.Constructor.Signature.Parameters)
}

func (a *analyzer) installClassImplicitBindings(current *scope, class *Symbol) {
	attribute := &Symbol{Name: "attribute", Kind: ParameterSymbol, Type: types.Class{Symbol: class.Class}, Builtin: true, InitialNull: NonNull}
	current.symbols[attribute.Name] = attribute
	if class.Class.Parent != nil {
		superClass := &Symbol{Name: "SuperClass", Kind: ClassSymbol, Type: types.Class{Symbol: class.Class.Parent, Reference: true}, Class: class.Class.Parent, Builtin: true, InitialNull: NonNull, SuperClassBinding: true}
		current.symbols[superClass.Name] = superClass
	}
}

func (a *analyzer) analyzeReturn(statement *ast.ReturnStmt, current *scope, flow flowState) {
	if current.callable == nil {
		a.error(codeReturnType, "return is valid only inside a Function", statement.Span(), "remove return from module/class scope")
		if statement.Value != nil {
			a.analyzeExpression(statement.Value, current, flow)
		}
		return
	}
	context := current.callable
	if context.kind == structureCallable {
		a.error(codeReturnType, "return is not allowed inside structure", statement.Span(), "structure completes implicitly")
		if statement.Value != nil {
			a.analyzeExpression(statement.Value, current, flow)
		}
		return
	}
	if context.returnType.Kind() == types.NothingKind {
		if statement.Value != nil {
			value := a.analyzeExpression(statement.Value, current, flow)
			a.error(codeReturnType, fmt.Sprintf("Nothing Function cannot return %s", types.Display(value.typeValue)), statement.Value.Span(), "use a bare return")
		}
		return
	}
	if statement.Value == nil {
		a.error(codeReturnType, fmt.Sprintf("Function must return %s", types.Display(context.returnType)), statement.Span(), "return a compatible value")
		return
	}
	value := a.analyzeExpressionExpected(statement.Value, current, flow, context.returnType)
	if value.nullState != Null && !types.Assignable(context.returnType, value.typeValue) {
		a.typeMismatch(statement.Value.Span(), context.returnType, value.typeValue, "return value")
	}
	context.returnNull = append(context.returnNull, value.nullState)
}
