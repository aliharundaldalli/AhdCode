package semantic

import (
	"fmt"

	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

type statementOutcome struct {
	flow    flowState
	returns bool
}

func (a *analyzer) analyzeStatement(statement ast.Stmt, current *scope, flow flowState) statementOutcome {
	if statement == nil {
		return statementOutcome{flow: flow}
	}
	switch value := statement.(type) {
	case *ast.BadStmt:
		return statementOutcome{flow: flow}
	case *ast.ExprStmt:
		a.analyzeExpression(value.Expression, current, flow)
	case *ast.VariableDecl:
		return statementOutcome{flow: a.analyzeVariableDeclaration(value, current, flow)}
	case *ast.AssignmentStmt:
		return statementOutcome{flow: a.analyzeAssignment(value, current, flow)}
	case *ast.IncDecStmt:
		return statementOutcome{flow: a.analyzeIncDec(value, current, flow)}
	case *ast.ReturnStmt:
		a.analyzeReturn(value, current, flow)
		return statementOutcome{flow: flow, returns: true}
	case *ast.TossStmt:
		a.analyzeToss(value, current, flow)
		return statementOutcome{flow: flow, returns: true}
	case *ast.BreakStmt:
		if a.loopDepth == 0 {
			a.error(codeControlContext, "break is valid only inside a loop", value.Span(), "move break into for, while, or until")
		}
		return statementOutcome{flow: flow, returns: true}
	case *ast.ContinueStmt:
		if a.loopDepth == 0 {
			a.error(codeControlContext, "continue is valid only inside a loop", value.Span(), "move continue into for, while, or until")
		}
		return statementOutcome{flow: flow, returns: true}
	case *ast.IfStmt:
		return a.analyzeIf(value, current, flow)
	case *ast.WhileStmt:
		return a.analyzeWhile(value, current, flow)
	case *ast.UntilStmt:
		return a.analyzeUntil(value, current, flow)
	case *ast.ForStmt:
		return a.analyzeFor(value, current, flow)
	case *ast.StateStmt:
		return a.analyzeState(value, current, flow)
	case *ast.AttemptStmt:
		return a.analyzeAttempt(value, current, flow)
	case *ast.BringStmt:
		// Imports are installed before declaration/type predeclaration so later
		// statements use the same ordinary scope and overload machinery.
	case *ast.FunctionDecl:
		a.analyzeFunction(value, current.callableClass())
	case *ast.ClassDecl:
		a.analyzeClass(value)
	case *ast.StructureDecl:
		a.analyzeStructure(value, current.callableClass())
	}
	return statementOutcome{flow: flow}
}

func (current *scope) callableClass() *Symbol {
	if current != nil && current.callable != nil {
		return current.callable.class
	}
	return nil
}

func (a *analyzer) analyzeVariableDeclaration(declaration *ast.VariableDecl, current *scope, flow flowState) flowState {
	if hasModifier(declaration.Modifiers, ast.ModifierGlobal) || declaration.GlobalOnly {
		return a.analyzeGlobalDeclaration(declaration, current, flow)
	}
	if declaration.Inferred {
		return a.analyzeInferredDeclaration(declaration, current, flow)
	}
	identifier, isIdentifier := declaration.Target.(*ast.IdentifierExpr)
	if current.kind == moduleScope {
		if hasModifier(declaration.Modifiers, ast.ModifierLocal) {
			a.error(codeScopeModifier, "module-root declarations must not use Local", declaration.Span(), "remove Local at module root")
		}
	} else if isIdentifier && !hasModifier(declaration.Modifiers, ast.ModifierLocal) {
		a.error(codeMissingLocal, fmt.Sprintf("nested declaration %q requires Local", declaration.Name), declaration.Span(), fmt.Sprintf("write %s: Local %s := ...", declaration.Name, declaration.Type.Name))
	}

	typeValue := a.resolveType(declaration.Type)
	if typeValue.Kind() == types.NothingKind {
		a.error(codeInvalidType, "Nothing cannot be used as a value binding type", declaration.Type.Span(), "use Nothing only as a callable return type")
	}

	if !isIdentifier {
		return a.analyzeMemberDeclaration(declaration, current, flow, typeValue)
	}

	predeclared, exists := current.local(declaration.Name)
	var symbol *Symbol
	if exists && predeclared.Declaration == declaration {
		symbol = predeclared
	} else if exists {
		a.error(codeRedeclaration, fmt.Sprintf("%q is already declared in this lexical scope", declaration.Name), declaration.Span(), "use = for reassignment or choose a shadowing declaration in a nested scope")
		symbol = predeclared
	} else {
		symbol = &Symbol{
			Name: declaration.Name, Kind: BindingSymbol, Type: typeValue,
			Span: declaration.Span(), Declaration: declaration,
			Constant:         hasModifier(declaration.Modifiers, ast.ModifierConstant),
			Confidential:     hasModifier(declaration.Modifiers, ast.ModifierConfidential),
			ModuleRoot:       current.kind == moduleScope,
			InitialNull:      MaybeNull,
			DeclaredNullable: declaration.Type.Nullable,
			OriginModuleID:   a.environment.ModuleID,
		}
		current.symbols[symbol.Name] = symbol
		a.result.Symbols = append(a.result.Symbols, symbol)
	}
	a.result.ResolvedSymbols[declaration] = symbol
	a.result.ResolvedSymbols[identifier] = symbol
	if function, ok := typeValue.(types.Function); ok && function.Signature == nil && symbol.inference == nil {
		symbol.inference = newFunctionInference(nil, -1)
		a.trackInference(symbol, current)
	}

	if declaration.Initializer == nil {
		a.error(codeTypeMismatch, fmt.Sprintf("declaration %q requires an initializer", declaration.Name), declaration.Span(), "use := followed by a compatible value")
		flow[symbol] = MaybeNull
		return flow
	}
	initializer := a.analyzeExpressionExpected(declaration.Initializer, current, flow, typeValue)
	if initializer.nullState != Null && !types.Assignable(typeValue, initializer.typeValue) {
		a.typeMismatch(declaration.Initializer.Span(), typeValue, initializer.typeValue, fmt.Sprintf("initializer of %s", declaration.Name))
	}
	if targetFunction, ok := typeValue.(types.Function); ok && targetFunction.Signature == nil {
		if resolvedFunction, ok := initializer.typeValue.(types.Function); ok && resolvedFunction.Signature != nil {
			a.constrainConcreteFunction(symbol, resolvedFunction.Signature, concreteCallable(initializer), declaration.Initializer.Span())
		}
	}
	nullOK := a.requireCompatibleNull(declaration.Type.Nullable, initializer, declaration.Initializer.Span(), fmt.Sprintf("declaration %q", symbol.Name))
	resultNull := initializer.nullState
	if !nullOK {
		resultNull = NonNull
	}
	symbol.InitialNull = resultNull
	flow[symbol] = resultNull

	if symbol.Constant {
		if nullOK && initializer.nullState != NonNull {
			a.error(codeConstantInitializer, fmt.Sprintf("Constant %q cannot be null", symbol.Name), declaration.Initializer.Span(), "initialize Constant with a NonNull value")
			return flow
		}
		// A Constant reference binding deep-freezes the referenced object graph
		// instead of requiring a compile-time scalar constant expression. The
		// constant-expression rule stays scalar-only, as specified.
		if isReferenceType(typeValue) {
			return flow
		}
		constant, failure := a.evaluateConstant(declaration.Initializer)
		switch failure {
		case constCycle:
			a.error(codeConstantInitializer, fmt.Sprintf("cyclic Constant dependency involving %q", symbol.Name), declaration.Initializer.Span(), "break the Constant dependency cycle")
		case constNotExpression, constInvalid:
			a.error(codeConstantInitializer, fmt.Sprintf("initializer of Constant %q is not a scalar constant expression", symbol.Name), declaration.Initializer.Span(), "use only literals, parentheses, pure scalar operators, and scalar Constant references")
		case constOK:
			if constant.typeValue.Kind() == types.IntKind && !constant.fitsInt() {
				a.error(codeConstantRange, fmt.Sprintf("Constant %q overflows signed 64-bit Int", symbol.Name), declaration.Initializer.Span(), "keep the constant result within Int range")
			} else {
				symbol.ConstValue = cloneConstant(constant)
			}
		}
	}
	return flow
}

// isReferenceType reports the v0.1 reference-semantics types, for which
// Constant means deep freeze rather than a compile-time scalar value.
func isReferenceType(value types.Type) bool {
	switch value.Kind() {
	case types.ListKind, types.PairKind, types.ClassKind, types.FunctionKind:
		return true
	default:
		return false
	}
}

// analyzeInferredDeclaration handles `name := value` (optionally `name:
// Local := value`, scope modifier explicit): the declaration has no
// explicit `: Type`, so the static type -- including its nullability -- is
// exactly the initializer's own type. A definitely-null or maybe-null
// initializer therefore infers a nullable type (e.g. a call returning `T?`
// infers `T?`); only a bare `null` literal, which carries no type at all,
// is rejected with its own dedicated diagnostic. Scope intent is never
// inferred: the same Local/module-root rule the explicit-type path enforces
// applies here unchanged, using whatever modifiers were actually written.
func (a *analyzer) analyzeInferredDeclaration(declaration *ast.VariableDecl, current *scope, flow flowState) flowState {
	identifier, isIdentifier := declaration.Target.(*ast.IdentifierExpr)
	if !isIdentifier {
		if declaration.Initializer != nil {
			a.analyzeExpression(declaration.Initializer, current, flow)
		}
		return flow
	}
	if current.kind == moduleScope {
		if hasModifier(declaration.Modifiers, ast.ModifierLocal) {
			a.error(codeScopeModifier, "module-root declarations must not use Local", declaration.Span(), "remove Local at module root")
		}
	} else if !hasModifier(declaration.Modifiers, ast.ModifierLocal) {
		a.error(codeMissingLocal, fmt.Sprintf("nested declaration %q requires Local", declaration.Name), declaration.Span(), fmt.Sprintf("write %s: Local := ...", declaration.Name))
	}
	if declaration.Initializer == nil {
		a.error(codeTypeMismatch, fmt.Sprintf("declaration %q requires an initializer", declaration.Name), declaration.Span(), "use := followed by a value")
		return flow
	}
	initializer := a.analyzeExpression(declaration.Initializer, current, flow)
	if types.IsInvalid(initializer.typeValue) {
		if initializer.nullState == Null {
			a.error(codeCannotInferType, "cannot infer a type from null", declaration.Initializer.Span(), "declare an explicit nullable type, e.g. name: T? := null")
		}
		return flow
	}
	typeValue := initializer.typeValue
	if typeValue.Kind() == types.NothingKind {
		a.error(codeInvalidType, "Nothing cannot be used as a value binding type", declaration.Initializer.Span(), "use Nothing only as a callable return type")
	}
	// The inferred type takes on exactly the initializer's own nullability --
	// never forced to non-null, never widened to nullable beyond what the
	// initializer already is.
	declaredNullable := initializer.nullState != NonNull
	resultNull := initializer.nullState
	predeclared, exists := current.local(declaration.Name)
	var symbol *Symbol
	if exists && predeclared.Declaration == declaration {
		symbol = predeclared
	} else if exists {
		a.error(codeRedeclaration, fmt.Sprintf("%q is already declared in this lexical scope", declaration.Name), declaration.Span(), "use = for reassignment or choose a shadowing declaration in a nested scope")
		symbol = predeclared
	} else {
		symbol = &Symbol{
			Name: declaration.Name, Kind: BindingSymbol, Type: typeValue,
			Span: declaration.Span(), Declaration: declaration,
			ModuleRoot:       current.kind == moduleScope,
			InitialNull:      resultNull,
			DeclaredNullable: declaredNullable,
			OriginModuleID:   a.environment.ModuleID,
		}
		current.symbols[symbol.Name] = symbol
		a.result.Symbols = append(a.result.Symbols, symbol)
	}
	a.result.ResolvedSymbols[declaration] = symbol
	a.result.ResolvedSymbols[identifier] = symbol
	flow[symbol] = resultNull
	return flow
}

func (a *analyzer) analyzeGlobalDeclaration(declaration *ast.VariableDecl, current *scope, flow flowState) flowState {
	if current.callable == nil {
		a.error(codeInvalidGlobal, "Global declarations are valid only inside a function, method, or structure", declaration.Span(), "access module bindings directly from module-level control flow")
		return flow
	}
	identifier, ok := declaration.Target.(*ast.IdentifierExpr)
	if !ok {
		a.error(codeInvalidGlobal, "Global declaration target must be an identifier", declaration.Target.Span(), "declare the module binding name directly")
		return flow
	}
	if declaration.Initializer != nil {
		a.error(codeInvalidGlobal, "Global declarations cannot have an initializer", declaration.Initializer.Span(), "declare the module binding alias without :=")
	}
	moduleSymbol, ok := a.module.local(declaration.Name)
	if !ok || moduleSymbol.Kind == ClassSymbol {
		a.error(codeInvalidGlobal, fmt.Sprintf("no module-root value binding named %q", declaration.Name), declaration.Span(), "declare the module binding before this callable")
		return flow
	}
	if _, exists := current.local(declaration.Name); exists {
		a.error(codeRedeclaration, fmt.Sprintf("%q is already declared in this lexical scope", declaration.Name), declaration.Span(), "remove the duplicate Global declaration")
		return flow
	}
	// An inferred Global (`x: Global`, no explicit type) has no declared type
	// to validate: it adopts the module binding's type and nullability
	// directly, exactly like an inferred ordinary declaration adopts its
	// initializer's type. Scope intent (Global) stays explicit; only the
	// value type is inferred.
	if !declaration.Inferred {
		declaredType := a.resolveType(declaration.Type)
		if !types.Equal(declaredType, moduleSymbol.Type) && !(declaredType.Kind() == types.FunctionKind && moduleSymbol.Type.Kind() == types.FunctionKind) {
			a.typeMismatch(declaration.Type.Span(), moduleSymbol.Type, declaredType, fmt.Sprintf("Global declaration %s", declaration.Name))
		} else if declaration.Type.Nullable != moduleSymbol.DeclaredNullable {
			a.error(codeTypeMismatch, fmt.Sprintf("Global declaration %s must match the module binding's nullability", declaration.Name), declaration.Type.Span(), fmt.Sprintf("declare %s: %s", declaration.Name, types.Display(moduleSymbol.Type)))
		}
	}
	alias := &Symbol{
		Name: declaration.Name, Kind: BindingSymbol, Type: moduleSymbol.Type,
		Span: declaration.Span(), Declaration: declaration, Constant: moduleSymbol.Constant,
		InitialNull: flow.state(moduleSymbol), Alias: moduleSymbol, DeclaredNullable: moduleSymbol.DeclaredNullable,
	}
	current.symbols[alias.Name] = alias
	flow[alias] = flow.state(moduleSymbol)
	a.result.Symbols = append(a.result.Symbols, alias)
	a.result.ResolvedSymbols[declaration] = alias
	a.result.ResolvedSymbols[identifier] = alias
	return flow
}

func (a *analyzer) analyzeMemberDeclaration(declaration *ast.VariableDecl, current *scope, flow flowState, typeValue types.Type) flowState {
	member, ok := declaration.Target.(*ast.MemberExpr)
	if !ok || current.callable == nil || current.callable.class == nil {
		a.error(codeInvalidTarget, "declaration target must be an identifier or attribute member in Class callable scope", declaration.Target.Span(), "declare a Local identifier or attribute.name")
		if declaration.Initializer != nil {
			a.analyzeExpression(declaration.Initializer, current, flow)
		}
		return flow
	}
	object, ok := member.Object.(*ast.IdentifierExpr)
	if !ok || object.Name != "attribute" {
		a.error(codeInvalidTarget, "Class member declarations must target attribute.name", declaration.Target.Span(), "use attribute followed by the member name")
		return flow
	}
	class := current.callable.class
	if _, exists := class.Members[member.Name]; exists {
		a.error(codeRedeclaration, fmt.Sprintf("Class member %q is already declared", member.Name), member.Span(), "use = to assign an existing member")
	}
	memberSymbol := &Symbol{
		Name: member.Name, Kind: MemberSymbol, Type: typeValue, Span: member.Span(), Declaration: declaration,
		Constant: hasModifier(declaration.Modifiers, ast.ModifierConstant), Confidential: hasModifier(declaration.Modifiers, ast.ModifierConfidential), InitialNull: MaybeNull,
		OwnerClass: class.Class, OriginModuleID: a.environment.ModuleID, DeclaredNullable: declaration.Type.Nullable,
	}
	class.Members[member.Name] = memberSymbol
	a.result.Symbols = append(a.result.Symbols, memberSymbol)
	a.result.ResolvedSymbols[declaration] = memberSymbol
	a.result.ResolvedSymbols[member] = memberSymbol
	if function, ok := typeValue.(types.Function); ok && function.Signature == nil {
		memberSymbol.inference = newFunctionInference(nil, -1)
		a.trackInference(memberSymbol, current)
	}
	if declaration.Initializer != nil {
		initializer := a.analyzeExpressionExpected(declaration.Initializer, current, flow, typeValue)
		if initializer.nullState != Null && !types.Assignable(typeValue, initializer.typeValue) {
			a.typeMismatch(declaration.Initializer.Span(), typeValue, initializer.typeValue, fmt.Sprintf("initializer of member %s", member.Name))
		}
		nullOK := a.requireCompatibleNull(declaration.Type.Nullable, initializer, declaration.Initializer.Span(), fmt.Sprintf("member %q", member.Name))
		memberSymbol.InitialNull = initializer.nullState
		if !nullOK {
			memberSymbol.InitialNull = NonNull
		}
		if function, ok := initializer.typeValue.(types.Function); ok && function.Signature != nil {
			a.constrainConcreteFunction(memberSymbol, function.Signature, concreteCallable(initializer), declaration.Initializer.Span())
		}
	}
	return flow
}

func (a *analyzer) analyzeAssignment(statement *ast.AssignmentStmt, current *scope, flow flowState) flowState {
	target := a.analyzeExpression(statement.Target, current, flow)
	if target.symbol == nil {
		if !target.invalid() {
			if _, ok := statement.Target.(*ast.IndexExpr); !ok {
				a.error(codeInvalidTarget, "assignment target does not resolve to a mutable binding or member", statement.Target.Span(), "assign an identifier, member, or index")
			}
		}
	} else if target.symbol.Constant {
		a.error(codeConstantAssignment, fmt.Sprintf("Constant %q cannot be reassigned", target.symbol.Name), statement.Target.Span(), "remove the assignment or declare a mutable binding")
	}
	expected := target.typeValue
	if target.symbol != nil && target.symbol.inference != nil && target.symbol.inference.fixed != nil {
		expected = types.Function{Signature: target.symbol.inference.fixed}
	}
	value := a.analyzeExpressionExpected(statement.Value, current, flow, expected)
	if target.invalid() || value.invalid() {
		return flow
	}
	if target.symbol != nil && target.symbol.inference != nil {
		if function, ok := value.typeValue.(types.Function); ok && function.Signature != nil {
			a.constrainConcreteFunction(target.symbol, function.Signature, concreteCallable(value), statement.Value.Span())
		}
	}
	resultType := value.typeValue
	resultNull := value.nullState
	if statement.Operator != "=" {
		if target.nullState != NonNull {
			a.nullableError(statement.Operator, statement.Target, target.nullState)
		}
		if value.nullState != NonNull {
			a.nullableError(statement.Operator, statement.Value, value.nullState)
		}
		operator := statement.Operator[:len(statement.Operator)-1]
		resultType = a.binaryOperatorType(operator, target.typeValue, value.typeValue)
		resultNull = NonNull
		if types.IsInvalid(resultType) {
			a.operatorError(statement.Operator, target.typeValue, value.typeValue, statement.Span())
		}
	} else if !a.requireCompatibleNull(targetNullable(target), value, statement.Value.Span(), "assignment target") {
		resultNull = NonNull
	}
	if resultNull != Null && !types.Assignable(target.typeValue, resultType) {
		a.typeMismatch(statement.Value.Span(), target.typeValue, resultType, fmt.Sprintf("assignment with %s", statement.Operator))
	}
	if target.symbol != nil && !target.symbol.Constant {
		flow[target.symbol] = resultNull
		if target.symbol.Alias != nil {
			flow[target.symbol.Alias] = resultNull
		}
	}
	return flow
}

func (a *analyzer) analyzeIncDec(statement *ast.IncDecStmt, current *scope, flow flowState) flowState {
	target := a.analyzeExpression(statement.Target, current, flow)
	if target.invalid() {
		return flow
	}
	if target.symbol != nil && target.symbol.Constant {
		a.error(codeConstantAssignment, fmt.Sprintf("Constant %q cannot be updated", target.symbol.Name), statement.Target.Span(), "declare a mutable Int binding")
	}
	if target.nullState != NonNull {
		a.nullableError(statement.Operator, statement.Target, target.nullState)
	}
	if target.typeValue.Kind() != types.IntKind {
		a.error(codeOperatorType, fmt.Sprintf("%s requires Int; received %s", statement.Operator, types.Display(target.typeValue)), statement.Span(), "use ++ or -- only on an Int target")
	}
	return flow
}

func (a *analyzer) analyzeIf(statement *ast.IfStmt, current *scope, flow flowState) statementOutcome {
	remaining := flow.clone()
	var exits []flowState
	allReturn := statement.Else != nil
	for _, branch := range statement.Branches {
		condition := a.analyzeExpression(branch.Condition, current, remaining)
		a.requireBoolCondition(branch.Condition, condition)
		branchFlow := remaining.clone()
		a.applyRefinements(branch.Condition, true, branchFlow)
		outcome := a.analyzeBlock(branch.Body, current, branchFlow, nil)
		if !outcome.returns {
			exits = append(exits, outcome.flow)
		}
		allReturn = allReturn && outcome.returns
		a.applyRefinements(branch.Condition, false, remaining)
	}
	if statement.Else != nil {
		outcome := a.analyzeBlock(statement.Else, current, remaining, nil)
		if !outcome.returns {
			exits = append(exits, outcome.flow)
		}
		allReturn = allReturn && outcome.returns
	} else {
		exits = append(exits, remaining)
	}
	return statementOutcome{flow: mergeFlows(exits...), returns: allReturn}
}

func (a *analyzer) analyzeWhile(statement *ast.WhileStmt, current *scope, flow flowState) statementOutcome {
	condition := a.analyzeExpression(statement.Condition, current, flow)
	a.requireBoolCondition(statement.Condition, condition)
	bodyFlow := flow.clone()
	a.applyRefinements(statement.Condition, true, bodyFlow)
	a.loopDepth++
	outcome := a.analyzeBlock(statement.Body, current, bodyFlow, nil)
	a.loopDepth--
	return statementOutcome{flow: mergeFlows(flow, outcome.flow)}
}

func (a *analyzer) analyzeUntil(statement *ast.UntilStmt, current *scope, flow flowState) statementOutcome {
	a.loopDepth++
	body := a.analyzeBlock(statement.Body, current, flow.clone(), nil)
	a.loopDepth--
	condition := a.analyzeExpression(statement.Condition, current, body.flow)
	a.requireBoolCondition(statement.Condition, condition)
	return statementOutcome{flow: body.flow}
}

func (a *analyzer) analyzeFor(statement *ast.ForStmt, current *scope, flow flowState) statementOutcome {
	iterable := a.analyzeExpression(statement.Iterable, current, flow)
	if !iterable.invalid() && iterable.nullState != NonNull {
		a.nullableError("for iteration", statement.Iterable, iterable.nullState)
	}
	elementType, elementNullable, iterableType := types.IterationYield(iterable.typeValue)
	if !iterableType && !iterable.invalid() {
		a.error(codeOperatorType, fmt.Sprintf("type %s is not iterable", types.Display(iterable.typeValue)), statement.Iterable.Span(), "iterate a List, Pair, String, or between(...)")
	}
	// An explicit binding type constrains the yielded type; the binding itself
	// stays implicitly Local.
	if statement.Type != nil {
		declared := a.resolveType(statement.Type)
		if !types.IsInvalid(declared) && !types.IsInvalid(elementType) && !types.Equal(declared, elementType) {
			a.error(codeTypeMismatch, fmt.Sprintf("iteration variable expects %s; iterable yields %s", types.Display(declared), types.Display(elementType)), statement.Type.Span(), "declare the type the iterable yields, or omit the type")
		}
		if !types.IsInvalid(declared) {
			elementType = declared
		}
		if elementNullable && !statement.Type.Nullable {
			a.error(codeNullNotAllowed, fmt.Sprintf("iteration variable %q is not nullable; the iterable's elements may be null", statement.Name), statement.Type.Span(), "declare the type with a trailing ?")
		}
		elementNullable = statement.Type.Nullable
	}
	iteration := &Symbol{
		Name: statement.Name, Kind: ForSymbol, Type: elementType, Span: statement.Span(),
		InitialNull: nullStateFor(elementNullable), DeclaredNullable: elementNullable,
	}
	a.result.Symbols = append(a.result.Symbols, iteration)
	a.loopDepth++
	body := a.analyzeBlock(statement.Body, current, flow.clone(), map[string]*Symbol{statement.Name: iteration})
	a.loopDepth--
	return statementOutcome{flow: mergeFlows(flow, body.flow)}
}

func (a *analyzer) analyzeState(statement *ast.StateStmt, current *scope, flow flowState) statementOutcome {
	value := a.analyzeExpression(statement.Value, current, flow)
	var exits []flowState
	hasDefault, allReturn := false, true
	for _, condition := range statement.Conditions {
		if condition.Default {
			hasDefault = true
		} else {
			match := a.analyzeExpression(condition.Match, current, flow)
			compatible := types.Equal(value.typeValue, match.typeValue) || (types.IsNumeric(value.typeValue) && types.IsNumeric(match.typeValue)) || value.nullState == Null || match.nullState == Null
			if !compatible {
				a.typeMismatch(condition.Match.Span(), value.typeValue, match.typeValue, "state condition")
			}
		}
		outcome := a.analyzeBlock(condition.Body, current, flow.clone(), nil)
		if !outcome.returns {
			exits = append(exits, outcome.flow)
		}
		allReturn = allReturn && outcome.returns
	}
	if !hasDefault {
		exits = append(exits, flow)
		allReturn = false
	}
	return statementOutcome{flow: mergeFlows(exits...), returns: allReturn && hasDefault}
}

func (a *analyzer) analyzeAttempt(statement *ast.AttemptStmt, current *scope, flow flowState) statementOutcome {
	body := a.analyzeBlock(statement.Body, current, flow.clone(), nil)
	exits := []flowState{body.flow}
	// An attempt exits definitely when its body returns and every handler
	// returns. With no handler, a raised error propagates, which is an exit
	// rather than a fall-through.
	allReturn := body.returns
	for _, clause := range statement.Excepts {
		typeValue := a.resolveType(clause.Type)
		class, ok := typeValue.(types.Class)
		errorClass := a.classes["Error"].Class
		if !ok || !classAssignableTo(class.Symbol, errorClass) {
			a.error(codeTypeMismatch, fmt.Sprintf("except type must inherit Error; received %s", types.Display(typeValue)), clause.Type.Span(), "use Error or a derived Class")
		}
		bindings := make(map[string]*Symbol)
		if clause.Name != "" {
			symbol := &Symbol{Name: clause.Name, Kind: ExceptSymbol, Type: typeValue, Span: clause.Span(), InitialNull: NonNull}
			bindings[clause.Name] = symbol
			a.result.Symbols = append(a.result.Symbols, symbol)
		}
		outcome := a.analyzeBlock(clause.Body, current, flow.clone(), bindings)
		exits = append(exits, outcome.flow)
		allReturn = allReturn && outcome.returns
	}
	merged := mergeFlows(exits...)
	if statement.Ultimately != nil {
		final := a.analyzeBlock(statement.Ultimately, current, merged, nil)
		if final.returns {
			return final
		}
		merged = final.flow
	}
	return statementOutcome{flow: merged, returns: allReturn}
}

func classAssignableTo(value, target *types.ClassSymbol) bool {
	visited := make(map[*types.ClassSymbol]bool)
	for current := value; current != nil && !visited[current]; current = current.Parent {
		visited[current] = true
		if types.SameClassIdentity(current, target) {
			return true
		}
	}
	return false
}

func (a *analyzer) analyzeToss(statement *ast.TossStmt, current *scope, flow flowState) {
	value := a.analyzeExpression(statement.Value, current, flow)
	if value.invalid() {
		return
	}
	class, ok := value.typeValue.(types.Class)
	if !ok || class.Reference || !classAssignableTo(class.Symbol, a.classes["Error"].Class) {
		a.error(codeTypeMismatch, fmt.Sprintf("toss requires Error; received %s", types.Display(value.typeValue)), statement.Value.Span(), "toss an Error instance")
	}
	if value.nullState != NonNull {
		a.nullableError("toss", statement.Value, value.nullState)
	}
}

func (a *analyzer) requireBoolCondition(expression ast.Expr, info expressionInfo) {
	if info.invalid() {
		return
	}
	if info.nullState != NonNull || info.typeValue.Kind() != types.BoolKind {
		a.error(codeConditionType, fmt.Sprintf("condition requires NonNull Bool; received %s (%s)", types.Display(info.typeValue), info.nullState), expression.Span(), "use a Bool expression as the condition")
	}
}

func (a *analyzer) analyzeBlock(block *ast.Block, parent *scope, flow flowState, bindings map[string]*Symbol) statementOutcome {
	if block == nil {
		return statementOutcome{flow: flow}
	}
	current := newScope(parent, blockScope)
	for name, symbol := range bindings {
		current.symbols[name] = symbol
		flow[symbol] = symbol.InitialNull
	}
	returns := false
	for _, statement := range block.Statements {
		outcome := a.analyzeStatement(statement, current, flow)
		flow = outcome.flow
		returns = returns || outcome.returns
	}
	return statementOutcome{flow: flow, returns: returns}
}

func (a *analyzer) typeMismatch(span source.Span, expected, received types.Type, subject string) {
	a.error(codeTypeMismatch, fmt.Sprintf("%s expects %s; received %s", subject, types.Display(expected), types.Display(received)), span, "use a value assignable to the declared type")
}
