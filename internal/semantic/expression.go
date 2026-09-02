package semantic

import (
	"fmt"
	"math/big"
	"strconv"

	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

type expressionInfo struct {
	typeValue types.Type
	nullState NullState
	symbol    *Symbol
	constant  *constantValue
}

// invalid reports the recovery state produced after a primary semantic error.
// A null literal deliberately has Invalid as its placeholder type, but Null as
// its state, and is therefore not an error value.
func (info expressionInfo) invalid() bool {
	// A namespace is a valid semantic designator whose members carry types; it
	// intentionally has no ordinary value type of its own.
	if info.symbol != nil && info.symbol.Kind == NamespaceSymbol {
		return false
	}
	return types.IsInvalid(info.typeValue) && info.nullState != Null
}

func (a *analyzer) analyzeExpression(expression ast.Expr, current *scope, flow flowState) expressionInfo {
	return a.analyzeExpressionWithMagnitude(expression, current, flow, false)
}

func (a *analyzer) analyzeExpressionExpected(expression ast.Expr, current *scope, flow flowState, expected types.Type) expressionInfo {
	if expression == nil {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	switch value := expression.(type) {
	case *ast.GroupExpr:
		info := a.analyzeExpressionExpected(value.Expression, current, flow, expected)
		a.result.ExpressionTypes[expression] = info.typeValue
		a.result.NullStates[expression] = info.nullState
		return info
	case *ast.LambdaExpr:
		return a.recordExpression(expression, a.analyzeLambda(value, current, flow))
	case *ast.CallExpr:
		info := a.analyzeCallExpected(value, current, flow, expected)
		a.result.ExpressionTypes[expression] = info.typeValue
		a.result.NullStates[expression] = info.nullState
		return info
	case *ast.IdentifierExpr:
		return a.analyzeFunctionValueIdentifier(value, current, flow, expected)
	case *ast.MemberExpr:
		return a.analyzeFunctionValueMember(value, current, flow, expected)
	case *ast.ListExpr:
		info := a.analyzeListExpected(value, current, flow, expected)
		a.result.ExpressionTypes[expression] = info.typeValue
		a.result.NullStates[expression] = info.nullState
		return info
	case *ast.PairExpr:
		info := a.analyzePairExpected(value, current, flow, expected)
		a.result.ExpressionTypes[expression] = info.typeValue
		a.result.NullStates[expression] = info.nullState
		return info
	default:
		return a.analyzeExpression(expression, current, flow)
	}
}

// analyzeFunctionValueMember applies the ordinary callback context to an
// overloaded Function reached through a namespace or Class member. Direct
// calls still use call overload resolution; this path is only for taking the
// member itself as a Function value.
func (a *analyzer) analyzeFunctionValueMember(member *ast.MemberExpr, current *scope, flow flowState, expected types.Type) expressionInfo {
	info := a.analyzeMember(member, current, flow)
	symbol := info.symbol
	if symbol == nil {
		a.result.ExpressionTypes[member] = info.typeValue
		a.result.NullStates[member] = info.nullState
		return info
	}
	if symbol.Alias != nil {
		symbol = symbol.Alias
	}
	if symbol.OverloadSet == nil {
		a.result.ExpressionTypes[member] = info.typeValue
		a.result.NullStates[member] = info.nullState
		return info
	}
	expectedFunction, hasExpectedFunction := expected.(types.Function)
	if !hasExpectedFunction || expectedFunction.Signature == nil {
		a.error(codeAmbiguousOverload, fmt.Sprintf("overloaded Function value %q has no selecting context", symbol.Name), member.Span(), "provide a concrete callback context that selects exactly one overload")
		info.typeValue = types.Invalid
	} else if selected := a.selectFunctionValue(symbol.OverloadSet, expectedFunction.Signature, member); selected != nil {
		info.typeValue = types.Function{Signature: selected.Signature}
		a.result.SelectedFunctionValues[member] = selected
	} else {
		info.typeValue = types.Invalid
	}
	a.result.ExpressionTypes[member] = info.typeValue
	a.result.NullStates[member] = info.nullState
	return info
}

func (a *analyzer) analyzeExpressionWithMagnitude(expression ast.Expr, current *scope, flow flowState, allowMinMagnitude bool) expressionInfo {
	if expression == nil {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	var info expressionInfo
	switch value := expression.(type) {
	case *ast.BadExpr:
		info = expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	case *ast.LiteralExpr:
		info = a.analyzeLiteral(value, allowMinMagnitude)
	case *ast.StringExpr:
		for _, part := range value.Parts {
			if part.Expression == nil {
				continue
			}
			partInfo := a.analyzeExpression(part.Expression, current, flow)
			if partInfo.typeValue.Kind() == types.NothingKind {
				a.error(codeTypeMismatch, "String interpolation cannot contain Nothing", part.Expression.Span(), "interpolate a value accepted by str")
			}
		}
		info = expressionInfo{typeValue: types.String, nullState: NonNull}
	case *ast.IdentifierExpr:
		info = a.analyzeIdentifier(value, current, flow)
	case *ast.GroupExpr:
		info = a.analyzeExpressionWithMagnitude(value.Expression, current, flow, allowMinMagnitude)
	case *ast.LambdaExpr:
		info = a.analyzeLambda(value, current, flow)
	case *ast.UnaryExpr:
		info = a.analyzeUnary(value, current, flow)
	case *ast.BinaryExpr:
		info = a.analyzeBinary(value, current, flow)
	case *ast.CallExpr:
		info = a.analyzeCallExpected(value, current, flow, nil)
	case *ast.MemberExpr:
		info = a.analyzeMember(value, current, flow)
	case *ast.IndexExpr:
		info = a.analyzeIndex(value, current, flow)
	case *ast.SliceExpr:
		info = a.analyzeSlice(value, current, flow)
	case *ast.ListExpr:
		info = a.analyzeList(value, current, flow)
	case *ast.PairExpr:
		info = a.analyzePair(value, current, flow)
	default:
		info = expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	a.result.ExpressionTypes[expression] = info.typeValue
	a.result.NullStates[expression] = info.nullState
	if info.symbol != nil {
		a.result.ResolvedSymbols[expression] = info.symbol
	}
	return info
}

func (a *analyzer) recordExpression(expression ast.Expr, info expressionInfo) expressionInfo {
	a.result.ExpressionTypes[expression] = info.typeValue
	a.result.NullStates[expression] = info.nullState
	if info.symbol != nil {
		a.result.ResolvedSymbols[expression] = info.symbol
	}
	return info
}

// analyzeLambda builds one concrete callable signature from explicit
// parameter types and the statically inferred type of the single body
// expression. The scope is linked for name lookup, but crossing an enclosing
// callable boundary is diagnosed as unsupported capture by analyzeIdentifier.
func (a *analyzer) analyzeLambda(lambda *ast.LambdaExpr, current *scope, flow flowState) expressionInfo {
	parameters := make([]types.Parameter, len(lambda.Parameters))
	parameterNull := make([]NullState, len(lambda.Parameters))
	for index := range lambda.Parameters {
		parameter := &lambda.Parameters[index]
		parameters[index] = types.Parameter{Name: parameter.Name, Type: a.resolveType(parameter.Type)}
		parameterNull[index] = nullStateFor(parameter.Type != nil && parameter.Type.Nullable)
	}
	callable := &Callable{
		Signature:     &types.Signature{Parameters: parameters, Return: types.Invalid},
		ParameterNull: parameterNull, ReturnNull: NonNull, Lambda: lambda,
	}
	symbol := &Symbol{
		Name: "lambda", Kind: FunctionSymbol, Type: types.Function{Signature: callable.Signature},
		Span: lambda.Span(), Declaration: lambda, Callable: callable, InitialNull: NonNull,
		OriginModuleID: a.environment.ModuleID,
	}
	a.result.ResolvedSymbols[lambda] = symbol
	a.result.Symbols = append(a.result.Symbols, symbol)
	a.result.LambdaExpressions = append(a.result.LambdaExpressions, lambda)

	lambdaScope := newScope(current, callableScope)
	context := &callableContext{kind: lambdaCallable, symbol: symbol, callable: callable, returnType: types.Invalid}
	lambdaScope.callable = context
	lambdaFlow := flow.clone()
	a.analyzeLambdaCaptures(lambda, current, lambdaScope, flow, lambdaFlow, callable)
	for index := range lambda.Parameters {
		parameter := &lambda.Parameters[index]
		if len(parameter.Modifiers) != 0 {
			a.error(codeScopeModifier, fmt.Sprintf("lambda parameter %q is implicitly Local", parameter.Name), parameter.Span(), "remove declaration modifiers from the lambda parameter")
		}
		if _, exists := lambdaScope.local(parameter.Name); exists {
			a.error(codeRedeclaration, fmt.Sprintf("duplicate lambda parameter %q", parameter.Name), parameter.Span(), "use a unique parameter name")
			continue
		}
		parameterSymbol := &Symbol{
			Name: parameter.Name, Kind: ParameterSymbol, Type: parameters[index].Type,
			Span: parameter.Span(), InitialNull: parameterNull[index],
			DeclaredNullable: parameter.Type != nil && parameter.Type.Nullable,
		}
		if function, ok := parameters[index].Type.(types.Function); ok && function.Signature == nil {
			parameterSymbol.inference = newFunctionInference(callable.Signature, index)
		}
		lambdaScope.symbols[parameter.Name] = parameterSymbol
		lambdaFlow[parameterSymbol] = parameterSymbol.InitialNull
		a.result.Symbols = append(a.result.Symbols, parameterSymbol)
		a.trackInference(parameterSymbol, lambdaScope)
		if parameter.Default != nil {
			a.error(codeInvalidLambda, fmt.Sprintf("lambda parameter %q cannot have a default value in v0.1.11", parameter.Name), parameter.Default.Span(), "use a required lambda parameter or a named Function declaration")
		}
	}
	body := a.analyzeExpression(lambda.Body, lambdaScope, lambdaFlow)
	a.finalizeInferences(context.inferences)
	if resolved := a.result.ExpressionTypes[lambda.Body]; resolved != nil {
		body.typeValue = resolved
		body.nullState = a.result.NullStates[lambda.Body]
	}
	if types.IsInvalid(body.typeValue) {
		if body.nullState == Null {
			a.error(codeCannotInferType, "cannot infer a lambda return type from null", lambda.Body.Span(), "return an expression with a concrete static type")
		}
		callable.Signature.Return = types.Invalid
	} else {
		callable.Signature.Return = body.typeValue
		callable.ReturnNull = body.nullState
	}
	symbol.Type = types.Function{Signature: callable.Signature}
	return expressionInfo{typeValue: symbol.Type, nullState: NonNull, symbol: symbol}
}

func (a *analyzer) analyzeLiteral(literal *ast.LiteralExpr, allowMinMagnitude bool) expressionInfo {
	switch literal.Kind {
	case ast.IntLiteral:
		value, failure := a.evaluateConstant(literal)
		if failure == constOK && value.integer != nil {
			limit := maxInt
			if allowMinMagnitude {
				limit = new(big.Int).Add(maxInt, big.NewInt(1))
			}
			if value.integer.Sign() < 0 || value.integer.Cmp(limit) > 0 {
				a.error(codeConstantRange, fmt.Sprintf("Int literal %s is outside signed 64-bit range", literal.Raw), literal.Span(), "valid Int values range from -9223372036854775808 to 9223372036854775807")
			}
			return expressionInfo{typeValue: types.Int, nullState: NonNull, constant: value}
		}
		return expressionInfo{typeValue: types.Int, nullState: NonNull}
	case ast.RealLiteral:
		value, failure := a.evaluateConstant(literal)
		if failure != constOK {
			a.error(codeConstantRange, fmt.Sprintf("Real literal %s is not representable", literal.Raw), literal.Span(), "use a finite Real literal")
		}
		return expressionInfo{typeValue: types.Real, nullState: NonNull, constant: value}
	case ast.ImaginaryLiteral:
		value, err := strconv.ParseFloat(literal.Value, 64)
		if err != nil || value != value || value > 1.7976931348623157e308 || value < -1.7976931348623157e308 {
			a.error(codeConstantRange, fmt.Sprintf("imaginary literal %s is not representable", literal.Raw), literal.Span(), "use a finite imaginary literal")
		}
		return expressionInfo{typeValue: types.Complex, nullState: NonNull}
	case ast.BoolLiteral:
		value, _ := a.evaluateConstant(literal)
		return expressionInfo{typeValue: types.Bool, nullState: NonNull, constant: value}
	case ast.NullLiteral:
		return expressionInfo{typeValue: types.Invalid, nullState: Null}
	default:
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
}

func (a *analyzer) analyzeIdentifier(identifier *ast.IdentifierExpr, current *scope, flow flowState) expressionInfo {
	symbol, owner := current.lookup(identifier.Name)
	if symbol == nil {
		a.error(codeUnknownName, fmt.Sprintf("unknown name %q", identifier.Name), identifier.Span(), "declare the binding in a visible lexical scope")
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	// A lambda reads an enclosing binding only when it lists that name. The
	// listed names already resolve inside the lambda's own scope, so reaching
	// this with a lexical binding from an outer callable means the dependency
	// list is missing it -- including a binding that is itself an enclosing
	// Function's explicit Global alias: from the lambda's own point of view
	// that is still an ordinary enclosing lexical binding, and a lambda gains
	// no privilege a Function lacks just because a Global declaration sits
	// somewhere between it and the module.
	if current.callable != nil && current.callable.kind == lambdaCallable && owner != a.module && owner.callable != current.callable && isLexicalCapture(symbol.Kind) {
		a.error(codeMissingCapture, fmt.Sprintf("local %q is not a lambda dependency", identifier.Name), identifier.Span(),
			fmt.Sprintf("add #%s (or Local %s) to the lambda dependency list, or pass the %s as a parameter",
				identifier.Name, identifier.Name, captureTypeText(symbol)))
	}
	// Global governs module state. A module-root Function, Class, or namespace
	// declaration is a callable or type declaration rather than a binding, so
	// it needs no Global declaration to be used inside a callable.
	if owner == a.module && current.callable != nil && !symbol.Builtin &&
		symbol.Kind != ClassSymbol && symbol.Kind != NamespaceSymbol && symbol.Kind != FunctionSymbol &&
		current.callable.symbol != symbol {
		if current.callable.kind == lambdaCallable {
			a.error(codeHiddenGlobal, fmt.Sprintf("module binding %q requires an explicit Global dependency", identifier.Name), identifier.Span(),
				fmt.Sprintf("add @%s (or Global %s) to the lambda dependency list", identifier.Name, identifier.Name))
		} else {
			a.error(codeHiddenGlobal, fmt.Sprintf("module binding %q requires an explicit Global declaration", identifier.Name), identifier.Span(), fmt.Sprintf("add %s: Global %s in this callable", identifier.Name, types.Display(symbol.Type)))
		}
	}
	typeValue := symbol.Type
	if symbol.Kind == ClassSymbol {
		typeValue = types.Class{Symbol: symbol.Class, Reference: true}
	}
	if operation, known := moduleOperationOf(symbol); known && !a.calleeExpressions[identifier] {
		a.rejectModuleOperationValue(operation, identifier.Span())
		typeValue = types.Invalid
	}
	constant := symbol.ConstValue
	return expressionInfo{typeValue: typeValue, nullState: flow.state(symbol), symbol: symbol, constant: constant}
}

func isLexicalCapture(kind SymbolKind) bool {
	switch kind {
	case BindingSymbol, ParameterSymbol, ForSymbol, ExceptSymbol:
		return true
	default:
		return false
	}
}

func (a *analyzer) analyzeUnary(expression *ast.UnaryExpr, current *scope, flow flowState) expressionInfo {
	allowMagnitude := expression.Operator == "-"
	operand := a.analyzeExpressionWithMagnitude(expression.Operand, current, flow, allowMagnitude)
	if operand.invalid() {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	if operand.nullState != NonNull {
		a.nullableError(expression.Operator, expression.Operand, operand.nullState)
	}
	switch expression.Operator {
	case "+", "-":
		if !types.IsNumeric(operand.typeValue) {
			if expression.Operator == "-" {
				if classSymbol := a.classInstanceSymbol(operand.typeValue); classSymbol != nil {
					if info, handled := a.analyzeClassNegate(expression, classSymbol); handled {
						return info
					}
				}
			}
			a.operatorError(expression.Operator, operand.typeValue, nil, expression.Span())
			return expressionInfo{typeValue: types.Invalid, nullState: NonNull}
		}
	case "not":
		if operand.typeValue.Kind() != types.BoolKind {
			a.operatorError("not", operand.typeValue, nil, expression.Span())
			return expressionInfo{typeValue: types.Invalid, nullState: NonNull}
		}
	}
	constant, _ := a.evaluateConstant(expression)
	if constant != nil && constant.typeValue.Kind() == types.IntKind && !constant.fitsInt() {
		a.error(codeConstantRange, "constant Int expression overflows signed 64-bit range", expression.Span(), "use a Real expression or keep the result within Int range")
	}
	return expressionInfo{typeValue: operand.typeValue, nullState: NonNull, constant: constant}
}

func (a *analyzer) analyzeBinary(expression *ast.BinaryExpr, current *scope, flow flowState) expressionInfo {
	left := a.analyzeExpression(expression.Left, current, flow)
	rightFlow := flow
	if expression.Operator == "and" {
		rightFlow = flow.clone()
		a.applyRefinements(expression.Left, true, rightFlow)
	} else if expression.Operator == "or" {
		rightFlow = flow.clone()
		a.applyRefinements(expression.Left, false, rightFlow)
	}

	// The right operand of has is a member designator, not a lexical binding.
	var right expressionInfo
	if expression.Operator == "has" || expression.Operator == "has not" {
		if _, ok := expression.Right.(*ast.IdentifierExpr); !ok {
			right = a.analyzeExpression(expression.Right, current, rightFlow)
		} else {
			right = expressionInfo{typeValue: types.String, nullState: NonNull}
		}
	} else {
		right = a.analyzeExpression(expression.Right, current, rightFlow)
	}
	// Both sides are analyzed before recovery so independent sibling errors are
	// retained. Parent checks must not diagnose consequences of an invalid child.
	if left.invalid() || right.invalid() {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}

	operator := expression.Operator
	if operator == "==" || operator == "!=" || operator == "same" || operator == "is" || operator == "is not" {
		return a.analyzeEqualityLike(expression, left, right)
	}
	if operator == "and" || operator == "or" {
		if left.nullState != NonNull {
			a.nullableError(operator, expression.Left, left.nullState)
		}
		if right.nullState != NonNull {
			a.nullableError(operator, expression.Right, right.nullState)
		}
		if left.typeValue.Kind() != types.BoolKind || right.typeValue.Kind() != types.BoolKind {
			a.operatorError(operator, left.typeValue, right.typeValue, expression.Span())
		}
		constant, _ := a.evaluateConstant(expression)
		return expressionInfo{typeValue: types.Bool, nullState: NonNull, constant: constant}
	}
	if operator == "in" || operator == "not in" {
		return a.analyzeMembership(expression, left, right)
	}
	if operator == "has" || operator == "has not" {
		if left.nullState != NonNull {
			a.nullableError(operator, expression.Left, left.nullState)
		}
		class, ok := left.typeValue.(types.Class)
		if !ok || class.Reference {
			a.operatorError(operator, left.typeValue, right.typeValue, expression.Span())
		}
		return expressionInfo{typeValue: types.Bool, nullState: NonNull}
	}
	if left.nullState != NonNull {
		a.nullableError(operator, expression.Left, left.nullState)
	} else if classSymbol := a.classInstanceSymbol(left.typeValue); classSymbol != nil {
		if info, handled := a.analyzeClassOperator(expression, classSymbol, operator, right); handled {
			return info
		}
	}
	if right.nullState != NonNull {
		a.nullableError(operator, expression.Right, right.nullState)
	}
	resultType := a.binaryOperatorType(operator, left.typeValue, right.typeValue)
	if types.IsInvalid(resultType) {
		a.operatorError(operator, left.typeValue, right.typeValue, expression.Span())
	}
	constant, failure := a.evaluateConstant(expression)
	if failure == constOK && constant != nil && constant.typeValue.Kind() == types.IntKind && !constant.fitsInt() {
		a.error(codeConstantRange, "constant Int expression overflows signed 64-bit range", expression.Span(), "use a Real expression or keep the result within Int range")
	}
	return expressionInfo{typeValue: resultType, nullState: NonNull, constant: constant}
}

func (a *analyzer) analyzeEqualityLike(expression *ast.BinaryExpr, left, right expressionInfo) expressionInfo {
	operator := expression.Operator
	if operator == "is" || operator == "is not" {
		class, ok := right.typeValue.(types.Class)
		if !ok || !class.Reference {
			a.error(codeOperatorType, fmt.Sprintf("right operand of %s must be a Class reference", operator), expression.Right.Span(), fmt.Sprintf("received %s", types.Display(right.typeValue)))
		}
		return expressionInfo{typeValue: types.Bool, nullState: NonNull}
	}
	if left.nullState == Null || right.nullState == Null {
		return expressionInfo{typeValue: types.Bool, nullState: NonNull}
	}
	if operator == "same" {
		constant, _ := a.evaluateConstant(expression)
		return expressionInfo{typeValue: types.Bool, nullState: NonNull, constant: constant}
	}
	if operator == "==" || operator == "!=" {
		if left.nullState == NonNull {
			if classSymbol := a.classInstanceSymbol(left.typeValue); classSymbol != nil {
				if info, handled := a.analyzeClassEquality(expression, classSymbol, right); handled {
					return info
				}
			}
		}
	}
	compatible := types.Equal(left.typeValue, right.typeValue) ||
		(types.IsNumeric(left.typeValue) && types.IsNumeric(right.typeValue)) ||
		relatedClasses(left.typeValue, right.typeValue)
	if !compatible {
		a.operatorError(operator, left.typeValue, right.typeValue, expression.Span())
	}
	constant, _ := a.evaluateConstant(expression)
	return expressionInfo{typeValue: types.Bool, nullState: NonNull, constant: constant}
}

func (a *analyzer) analyzeMembership(expression *ast.BinaryExpr, left, right expressionInfo) expressionInfo {
	if left.nullState != NonNull {
		a.nullableError(expression.Operator, expression.Left, left.nullState)
	}
	if right.nullState != NonNull {
		a.nullableError(expression.Operator, expression.Right, right.nullState)
	}
	compatible := false
	switch collection := right.typeValue.(type) {
	case types.List:
		compatible = types.Assignable(collection.Element, left.typeValue)
	case types.Pair:
		compatible = types.Assignable(collection.Key, left.typeValue)
	default:
		compatible = right.typeValue.Kind() == types.StringKind && left.typeValue.Kind() == types.StringKind
	}
	if !compatible {
		a.operatorError(expression.Operator, left.typeValue, right.typeValue, expression.Span())
	}
	return expressionInfo{typeValue: types.Bool, nullState: NonNull}
}

func (a *analyzer) binaryOperatorType(operator string, left, right types.Type) types.Type {
	if operator == "+" && left.Kind() == types.StringKind && right.Kind() == types.StringKind {
		return types.String
	}
	if operator == "*" && left.Kind() == types.StringKind && right.Kind() == types.IntKind {
		return types.String
	}
	if operator == "+" {
		leftList, leftOK := left.(types.List)
		rightList, rightOK := right.(types.List)
		if leftOK && rightOK && types.Equal(leftList, rightList) {
			return leftList
		}
	}
	if operator == "%" {
		if left.Kind() == types.IntKind && right.Kind() == types.IntKind {
			return types.Int
		}
		return types.Invalid
	}
	if operator == "^" {
		if left.Kind() == types.ComplexKind && right.Kind() == types.IntKind {
			return types.Complex
		}
		if left.Kind() == types.IntKind && right.Kind() == types.IntKind {
			return types.Int
		}
		if (left.Kind() == types.IntKind || left.Kind() == types.RealKind) &&
			(right.Kind() == types.IntKind || right.Kind() == types.RealKind) {
			return types.Real
		}
		return types.Invalid
	}
	if operator == "/" {
		if types.IsNumeric(left) && types.IsNumeric(right) {
			if left.Kind() == types.ComplexKind || right.Kind() == types.ComplexKind {
				return types.Complex
			}
			return types.Real
		}
		return types.Invalid
	}
	if operator == "+" || operator == "-" || operator == "*" {
		if types.IsNumeric(left) && types.IsNumeric(right) {
			if left.Kind() == types.ComplexKind || right.Kind() == types.ComplexKind {
				return types.Complex
			}
			if left.Kind() == types.RealKind || right.Kind() == types.RealKind {
				return types.Real
			}
			return types.Int
		}
		return types.Invalid
	}
	if operator == "<" || operator == "<=" || operator == ">" || operator == ">=" {
		if (left.Kind() == types.IntKind || left.Kind() == types.RealKind) && (right.Kind() == types.IntKind || right.Kind() == types.RealKind) {
			return types.Bool
		}
	}
	return types.Invalid
}

func (a *analyzer) analyzeFunctionValueIdentifier(identifier *ast.IdentifierExpr, current *scope, flow flowState, expected types.Type) expressionInfo {
	info := a.analyzeIdentifier(identifier, current, flow)
	symbol := info.symbol
	if symbol == nil {
		a.result.ExpressionTypes[identifier] = info.typeValue
		a.result.NullStates[identifier] = info.nullState
		return info
	}
	a.result.ResolvedSymbols[identifier] = symbol
	if symbol.Alias != nil {
		symbol = symbol.Alias
	}
	expectedFunction, hasExpectedFunction := expected.(types.Function)
	if symbol.inference != nil && hasExpectedFunction && expectedFunction.Signature != nil {
		a.constrainConcreteFunction(symbol, expectedFunction.Signature, nil, identifier.Span())
		info.typeValue = expectedFunction
	}
	if symbol.OverloadSet == nil {
		a.result.ExpressionTypes[identifier] = info.typeValue
		a.result.NullStates[identifier] = info.nullState
		return info
	}
	candidates := symbol.OverloadSet.Candidates
	if len(candidates) == 1 {
		selected := candidates[0]
		if hasExpectedFunction && expectedFunction.Signature != nil {
			if _, ok := functionValueScore(selected.Signature, expectedFunction.Signature); !ok {
				a.typeMismatch(identifier.Span(), expectedFunction, types.Function{Signature: selected.Signature}, fmt.Sprintf("Function value %s", identifier.Name))
			}
		}
		info.typeValue = types.Function{Signature: selected.Signature}
		info.symbol = symbol
		a.result.SelectedFunctionValues[identifier] = selected
		a.result.ExpressionTypes[identifier] = info.typeValue
		a.result.NullStates[identifier] = info.nullState
		return info
	}
	if !hasExpectedFunction || expectedFunction.Signature == nil {
		a.error(codeAmbiguousOverload, fmt.Sprintf("overloaded Function value %q has no selecting context", identifier.Name), identifier.Span(), "provide a concrete callback context that selects exactly one overload")
		info.typeValue = types.Invalid
	} else if selected := a.selectFunctionValue(symbol.OverloadSet, expectedFunction.Signature, identifier); selected != nil {
		info.typeValue = types.Function{Signature: selected.Signature}
		a.result.SelectedFunctionValues[identifier] = selected
	} else {
		info.typeValue = types.Invalid
	}
	a.result.ExpressionTypes[identifier] = info.typeValue
	a.result.NullStates[identifier] = info.nullState
	return info
}

func (a *analyzer) analyzeCallExpected(call *ast.CallExpr, current *scope, flow flowState, expected types.Type) expressionInfo {
	// A type-directed module operation is specialized against the arguments
	// written at this call site, so it is legal here and nowhere else.
	a.calleeExpressions[call.Callee] = true
	if member, ok := call.Callee.(*ast.MemberExpr); ok {
		// A member receiver is analyzed exactly once. Its statically known type
		// decides whether the call is a built-in type operation or an ordinary
		// member call, so neither path re-analyzes the receiver expression.
		receiver := a.analyzeExpression(member.Object, current, flow)
		if info, handled := a.analyzeTypeOperation(call, member, receiver, current, flow); handled {
			return info
		}
		return a.analyzeCallWithCallee(call, a.recordMember(member, a.analyzeMemberOf(member, receiver, current, flow)), current, flow, expected)
	}
	return a.analyzeCallWithCallee(call, a.analyzeExpression(call.Callee, current, flow), current, flow, expected)
}

// recordMember stores the side-table entries the expression dispatcher would
// have written for a member analyzed directly by the call path.
func (a *analyzer) recordMember(member *ast.MemberExpr, info expressionInfo) expressionInfo {
	a.result.ExpressionTypes[member] = info.typeValue
	a.result.NullStates[member] = info.nullState
	if info.symbol != nil {
		a.result.ResolvedSymbols[member] = info.symbol
	}
	return info
}

func (a *analyzer) analyzeCallWithCallee(call *ast.CallExpr, callee expressionInfo, current *scope, flow flowState, expected types.Type) expressionInfo {
	if callee.invalid() {
		// Analyze arguments for independent diagnostics, but do not report that
		// an already-invalid callee is also non-callable.
		a.analyzeCallArguments(call, current, flow, nil)
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	// A type-directed standard module function has no single declared
	// signature; its exact concrete one is computed from this call's argument
	// types. Both Lists.chunk(...) and an imported chunk(...) arrive here with
	// the same resolved module symbol, so one dispatch covers both spellings.
	if operation, known := moduleOperationOf(callee.symbol); known {
		return a.analyzeModuleOperation(call, operation, current, flow)
	}
	// Only the built-in Fundamentals functions use the builtin call path; a
	// built-in Class is constructed like any other Class.
	if callee.symbol != nil && callee.symbol.Builtin && callee.symbol.Kind == BuiltinSymbol {
		arguments := a.analyzeCallArguments(call, current, flow, nil)
		return a.analyzeBuiltinCall(call, callee.symbol, arguments)
	}
	if class, ok := callee.typeValue.(types.Class); ok && class.Reference {
		symbol := a.classSymbolFor(class.Symbol)
		if symbol == nil {
			return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
		}
		hint, supplied := timeConstructionHint(class.Symbol)
		if !supplied {
			hint, supplied = dataConstructionHint(class.Symbol)
		}
		if !supplied {
			hint, supplied = plotConstructionHint(class.Symbol)
		}
		if !supplied {
			hint, supplied = numericConstructionHint(class.Symbol)
		}
		if !supplied {
			hint, supplied = wordConstructionHint(class.Symbol)
		}
		if !supplied {
			hint, supplied = pdfConstructionHint(class.Symbol)
		}
		if !supplied {
			hint, supplied = excelConstructionHint(class.Symbol)
		}
		if !supplied {
			hint, supplied = jsonConstructionHint(class.Symbol)
		}
		if !supplied {
			hint, supplied = xmlConstructionHint(class.Symbol)
		}
		if !supplied {
			hint, supplied = sqliteConstructionHint(class.Symbol)
		}
		if supplied {
			// A compiler-supplied value is produced by a standard-module
			// function that validates its arguments, never by direct
			// construction.
			a.error(codeCallArguments, fmt.Sprintf("%s values are not constructed directly", class.Symbol.Name), call.Span(), hint)
			a.analyzeCallArguments(call, current, flow, nil)
			return expressionInfo{typeValue: types.Class{Symbol: class.Symbol}, nullState: NonNull}
		}
		arguments := a.analyzeCallArguments(call, current, flow, symbol.Constructor)
		a.validateCallArguments(call, symbol.Constructor, arguments)
		a.result.SelectedCallables[call] = symbol.Constructor
		return expressionInfo{typeValue: types.Class{Symbol: class.Symbol}, nullState: NonNull, symbol: symbol}
	}
	function, ok := callee.typeValue.(types.Function)
	if !ok {
		a.error(codeNotCallable, fmt.Sprintf("value of type %s is not callable", types.Display(callee.typeValue)), call.Callee.Span(), "call a Function binding or Class reference")
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	if callee.nullState != NonNull {
		a.nullableError("call", call.Callee, callee.nullState)
	}
	resolvedSymbol := callee.symbol
	if resolvedSymbol != nil && resolvedSymbol.Alias != nil {
		resolvedSymbol = resolvedSymbol.Alias
	}
	if resolvedSymbol != nil && resolvedSymbol.OverloadSet != nil && len(resolvedSymbol.OverloadSet.Candidates) > 1 {
		arguments := a.analyzeCallArguments(call, current, flow, nil)
		selected := a.resolveOverloadCall(call, resolvedSymbol.OverloadSet, arguments)
		if selected == nil {
			return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
		}
		a.result.SelectedCallables[call] = selected
		return expressionInfo{typeValue: selected.Signature.Return, nullState: selected.ReturnNull}
	}
	callable := callee.symbolCallable()
	if callable == nil && function.Signature != nil {
		// A concrete type without callable metadata can occur at an external
		// interface boundary. Missing return-state information is conservative.
		callable = &Callable{Signature: function.Signature, ParameterNull: nonNullParameters(len(function.Signature.Parameters)), ReturnNull: MaybeNull}
	}
	if callable == nil || callable.Signature == nil {
		arguments := a.analyzeCallArguments(call, current, flow, nil)
		if resolvedSymbol != nil {
			return a.constrainFunctionCall(resolvedSymbol, call, arguments, expected)
		}
		a.error(codeFunctionInference, "Function call has no inferable binding", call.Callee.Span(), "bind the callable to one concrete Function signature")
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	arguments := a.analyzeCallArguments(call, current, flow, callable)
	a.validateCallArguments(call, callable, arguments)
	a.result.SelectedCallables[call] = callable
	return expressionInfo{typeValue: callable.Signature.Return, nullState: callable.ReturnNull}
}

// typeOperationFor names the built-in operation a member call selects, if any.
// The statically known receiver type decides, so a Class may still declare its
// own add, map, or index method.
func typeOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	switch receiver.Kind() {
	case types.ComplexKind:
		operation, known := complexOperationNames[name]
		return operation, known
	case types.StringKind:
		operation, known := stringOperationNames[name]
		return operation, known
	case types.ListKind:
		operation, known := listOperationNames[name]
		return operation, known
	case types.PairKind:
		if name == "eject" {
			return PairEject, true
		}
	case types.ClassKind:
		if operation, ok := numericOperationFor(receiver, name); ok {
			return operation, true
		}
		if operation, ok := timeOperationFor(receiver, name); ok {
			return operation, true
		}
		if operation, ok := regexOperationFor(receiver, name); ok {
			return operation, true
		}
		if operation, ok := plotOperationFor(receiver, name); ok {
			return operation, true
		}
		if operation, ok := wordOperationFor(receiver, name); ok {
			return operation, true
		}
		if operation, ok := pdfOperationFor(receiver, name); ok {
			return operation, true
		}
		if operation, ok := excelOperationFor(receiver, name); ok {
			return operation, true
		}
		if operation, ok := jsonOperationFor(receiver, name); ok {
			return operation, true
		}
		if operation, ok := xmlOperationFor(receiver, name); ok {
			return operation, true
		}
		if operation, ok := sqliteOperationFor(receiver, name); ok {
			return operation, true
		}
		return dataOperationFor(receiver, name)
	}
	return "", false
}

// timeConstructionHint names the Time function that produces one
// compiler-supplied value, so direct construction has an actionable message.
func timeConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != timeModuleID {
		return "", false
	}
	switch identity.Name {
	case "DateTime":
		return "create a DateTime with Time.now(), Time.utc(), or a Time.dateTime... function", true
	case "Duration":
		return "create a Duration with Time.duration(...) or Time.between(...)", true
	case "Calendar":
		return "use Calendar members directly, as in Time.Calendar.isLeapYear(2028)", true
	}
	return "", false
}

// timeOperationFor names the built-in member a Time Class publishes. Only the
// compiler-supplied Time identities match, so a user Class is never affected.
func timeOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Symbol == nil || class.Symbol.ModuleID != timeModuleID {
		return "", false
	}
	// Calendar is used through its Class reference; DateTime through a value.
	switch {
	case class.Symbol.Name == "Calendar" && class.Reference:
		operation, known := calendarOperationNames[name]
		return operation, known
	case class.Symbol.Name == "DateTime" && !class.Reference:
		operation, known := dateTimeOperationNames[name]
		return operation, known
	}
	return "", false
}

var dateTimeOperationNames = map[string]TypeOperation{
	"before": DateTimeBefore, "after": DateTimeAfter,
	"sameMoment": DateTimeSameMoment, "timestamp": DateTimeTimestamp,
	"toUTC": DateTimeToUTC, "toLocal": DateTimeToLocal,
	"toOffset": DateTimeToOffset, "toString": DateTimeToString,
}

var calendarOperationNames = map[string]TypeOperation{
	"isLeapYear": CalendarIsLeapYear, "daysInMonth": CalendarDaysInMonth,
	"weekday": CalendarWeekday,
}

// timeOperationShape is the fixed call shape of one Time Class member.
type timeOperationShape struct {
	parameters []types.Type
	result     types.Type
	hint       string
}

func timeOperationShapes() map[TypeOperation]timeOperationShape {
	instant := types.Class{Symbol: timeDateTimeClass}
	return map[TypeOperation]timeOperationShape{
		DateTimeBefore:      {[]types.Type{instant}, types.Bool, "pass one DateTime to compare against"},
		DateTimeAfter:       {[]types.Type{instant}, types.Bool, "pass one DateTime to compare against"},
		DateTimeSameMoment:  {[]types.Type{instant}, types.Bool, "pass one DateTime to compare against"},
		DateTimeTimestamp:   {nil, types.Int, "call timestamp with no argument"},
		DateTimeToUTC:       {nil, instant, "call toUTC with no argument"},
		DateTimeToLocal:     {nil, instant, "call toLocal with no argument"},
		DateTimeToOffset:    {[]types.Type{types.Int}, instant, "pass one Int offset in minutes"},
		DateTimeToString:    {nil, types.String, "call toString with no argument"},
		CalendarIsLeapYear:  {[]types.Type{types.Int}, types.Bool, "pass one Int year"},
		CalendarDaysInMonth: {[]types.Type{types.Int, types.Int}, types.Int, "pass an Int year and an Int month"},
		CalendarWeekday:     {[]types.Type{types.Int, types.Int, types.Int}, types.Int, "pass an Int year, month, and day"},
	}
}

// analyzeTimeOperation checks one Time Class member. Every argument type is
// fixed, so this reuses the ordinary assignability and null-state rules.
func (a *analyzer) analyzeTimeOperation(call *ast.CallExpr, operation TypeOperation, shape timeOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	if len(call.Arguments) != len(shape.parameters) {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument(s); received %d", operation, len(shape.parameters), len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for index, expected := range shape.parameters {
		argument := a.analyzeExpressionExpected(call.Arguments[index].Value, current, flow, expected)
		if argument.invalid() {
			continue
		}
		if argument.nullState != NonNull {
			a.nullableError(string(operation), call.Arguments[index].Value, argument.nullState)
			continue
		}
		if !types.Assignable(expected, argument.typeValue) {
			a.typeMismatch(call.Arguments[index].Span(), expected, argument.typeValue, string(operation)+" argument")
		}
	}
	return result
}

var stringOperationNames = map[string]TypeOperation{
	"trim": StringTrim, "lower": StringLower, "upper": StringUpper,
	"capitalize": StringCapitalize, "split": StringSplit, "replace": StringReplace,
	"contains": StringContains, "startsWith": StringStartsWith, "endsWith": StringEndsWith,
	"count": StringCount, "index": StringIndex,
}

var complexOperationNames = map[string]TypeOperation{
	"real": ComplexReal, "imag": ComplexImag, "conjugate": ComplexConjugate,
	"magnitude": ComplexMagnitude, "phase": ComplexPhase,
}

var complexOperationResults = map[TypeOperation]types.Type{
	ComplexReal: types.Real, ComplexImag: types.Real, ComplexConjugate: types.Complex,
	ComplexMagnitude: types.Real, ComplexPhase: types.Real,
}

var listOperationNames = map[string]TypeOperation{
	"add": ListAdd, "eject": ListEject, "sort": ListSort, "reverse": ListReverse, "shuffle": ListShuffle,
	"count": ListCount, "index": ListIndex, "map": ListMap, "filter": ListFilter,
}

// stringOperationShape is the fixed call shape of one String operation. Every
// argument is a NonNull String, and String itself is never mutated.
type stringOperationShape struct {
	arguments int
	result    types.Type
	hint      string
}

var stringOperationShapes = map[TypeOperation]stringOperationShape{
	StringTrim:       {0, types.String, "call trim with no argument"},
	StringLower:      {0, types.String, "call lower with no argument"},
	StringUpper:      {0, types.String, "call upper with no argument"},
	StringCapitalize: {0, types.String, "call capitalize with no argument"},
	StringSplit:      {1, types.List{Element: types.String}, "pass one non-empty String separator"},
	StringReplace:    {2, types.String, "pass the searched String and its replacement"},
	StringContains:   {1, types.Bool, "pass one String to search for"},
	StringStartsWith: {1, types.Bool, "pass one String prefix"},
	StringEndsWith:   {1, types.Bool, "pass one String suffix"},
	StringCount:      {1, types.Int, "pass one non-empty String to count"},
	StringIndex:      {1, types.Int, "pass one non-empty String to locate"},
}

// analyzeTypeOperation type-checks one built-in String, List, or Pair
// operation. The receiver is already analyzed, so it is evaluated and
// diagnosed exactly once.
func (a *analyzer) analyzeTypeOperation(call *ast.CallExpr, member *ast.MemberExpr, receiver expressionInfo, current *scope, flow flowState) (expressionInfo, bool) {
	operation, known := typeOperationFor(receiver.typeValue, member.Name)
	if !known {
		return expressionInfo{}, false
	}
	a.result.TypeOperations[call] = operation
	a.result.ExpressionTypes[call.Callee] = types.Nothing
	a.result.NullStates[call.Callee] = NonNull
	if receiver.nullState != NonNull {
		a.nullableError(string(operation), member.Object, receiver.nullState)
	}
	if listOperationMutates(operation) {
		if target := receiver.symbol; target != nil && constantTarget(target) {
			a.error(codeConstantAssignment, fmt.Sprintf("cannot %s Constant %q", member.Name, target.Name), member.Object.Span(), fmt.Sprintf("%s mutates the List in place; declare it without Constant", member.Name))
		}
	}
	for _, argument := range call.Arguments {
		if argument.Name != "" {
			// A built-in type operation publishes no parameter names, which is
			// the same call shape the other built-in operations use.
			a.error(codeCallArguments, fmt.Sprintf("%s does not accept a named argument", operation), argument.Span(), typeOperationHint(operation, receiver.typeValue))
			a.analyzeTypeOperationArguments(call, current, flow, nil)
			return typeOperationFailure(operation, receiver.typeValue), true
		}
	}
	if shape, isString := stringOperationShapes[operation]; isString {
		return a.analyzeStringOperation(call, operation, shape, current, flow), true
	}
	if shape, isNumeric := numericShapes()[operation]; isNumeric {
		return a.analyzeNumericOperation(call, operation, shape, current, flow), true
	}
	if result, isComplex := complexOperationResults[operation]; isComplex {
		a.requireTypeOperationArity(call, operation, types.Complex, 0, current, flow)
		return expressionInfo{typeValue: result, nullState: NonNull}, true
	}
	if shape, isTime := timeOperationShapes()[operation]; isTime {
		return a.analyzeTimeOperation(call, operation, shape, current, flow), true
	}
	if shape, isRegex := regexOperationShapes()[operation]; isRegex {
		return a.analyzeRegexOperation(call, operation, shape, current, flow), true
	}
	if shape, isData := dataOperationShapes()[operation]; isData {
		return a.analyzeDataOperation(call, operation, shape, current, flow), true
	}
	if dataCallbackOperation(operation) {
		return a.analyzeDataCallbackOperation(call, operation, current, flow), true
	}
	if shape, isPlot := plotOperationShapes()[operation]; isPlot {
		return a.analyzePlotOperation(call, operation, shape, current, flow), true
	}
	if plotSeriesOperation(operation) {
		return a.analyzePlotSeriesOperation(call, operation, current, flow), true
	}
	if shape, isWord := wordOperationShapes()[operation]; isWord {
		return a.analyzeWordOperation(call, operation, shape, current, flow), true
	}
	if shape, isPDF := pdfOperationShapes()[operation]; isPDF {
		return a.analyzePDFOperation(call, operation, shape, current, flow), true
	}
	if shape, isExcel := excelOperationShapes()[operation]; isExcel {
		return a.analyzeExcelOperation(call, operation, shape, current, flow), true
	}
	if shape, isJSON := jsonOperationShapes()[operation]; isJSON {
		return a.analyzeJSONOperation(call, operation, shape, current, flow), true
	}
	if shape, isXML := xmlOperationShapes()[operation]; isXML {
		return a.analyzeXMLOperation(call, operation, shape, current, flow), true
	}
	if shape, isSQLite := sqliteOperationShapes()[operation]; isSQLite {
		return a.analyzeSQLiteOperation(call, operation, shape, current, flow), true
	}
	switch operation {
	case ListAdd, ListEject, PairEject:
		return a.analyzeCollectionMutation(call, operation, receiver, current, flow), true
	case ListReverse, ListShuffle:
		a.requireTypeOperationArity(call, operation, receiver.typeValue, 0, current, flow)
		return expressionInfo{typeValue: types.Nothing, nullState: NonNull}, true
	case ListSort:
		return a.analyzeListSort(call, receiver, current, flow), true
	case ListCount, ListIndex:
		return a.analyzeListSearch(call, operation, receiver, current, flow), true
	default:
		return a.analyzeListTransform(call, operation, receiver, current, flow), true
	}
}

// typeOperationFailure is the result an already-rejected operation reports. It
// keeps the statically known result type where the operation has one, so a
// rejection does not cascade into derivative diagnostics.
func typeOperationFailure(operation TypeOperation, receiver types.Type) expressionInfo {
	if shape, known := stringOperationShapes[operation]; known {
		return expressionInfo{typeValue: shape.result, nullState: NonNull}
	}
	if shape, known := numericShapes()[operation]; known {
		return expressionInfo{typeValue: shape.result, nullState: NonNull}
	}
	if result, known := complexOperationResults[operation]; known {
		return expressionInfo{typeValue: result, nullState: NonNull}
	}
	if shape, known := timeOperationShapes()[operation]; known {
		return expressionInfo{typeValue: shape.result, nullState: NonNull}
	}
	if shape, known := regexOperationShapes()[operation]; known {
		nullState := NonNull
		if shape.resultNullable {
			nullState = MaybeNull
		}
		return expressionInfo{typeValue: shape.result, nullState: nullState}
	}
	if result, known := dataOperationResult(operation); known {
		return expressionInfo{typeValue: result, nullState: NonNull}
	}
	if shape, known := plotOperationShapes()[operation]; known {
		return expressionInfo{typeValue: shape.result, nullState: NonNull}
	}
	if plotSeriesOperation(operation) {
		return expressionInfo{typeValue: plotChartType(), nullState: NonNull}
	}
	if shape, known := wordOperationShapes()[operation]; known {
		return expressionInfo{typeValue: shape.result, nullState: NonNull}
	}
	if shape, known := pdfOperationShapes()[operation]; known {
		return expressionInfo{typeValue: shape.result, nullState: NonNull}
	}
	if shape, known := excelOperationShapes()[operation]; known {
		nullState := NonNull
		if shape.resultNullable {
			nullState = MaybeNull
		}
		return expressionInfo{typeValue: shape.result, nullState: nullState}
	}
	if shape, known := jsonOperationShapes()[operation]; known {
		nullState := NonNull
		if shape.resultNullable {
			nullState = MaybeNull
		}
		return expressionInfo{typeValue: shape.result, nullState: nullState}
	}
	if shape, known := xmlOperationShapes()[operation]; known {
		nullState := NonNull
		if shape.nullable {
			nullState = MaybeNull
		}
		return expressionInfo{typeValue: shape.result, nullState: nullState}
	}
	if shape, known := sqliteOperationShapes()[operation]; known {
		return expressionInfo{typeValue: shape.result, nullState: NonNull}
	}
	switch operation {
	case ListAdd, ListEject, PairEject, ListSort, ListReverse, ListShuffle:
		return expressionInfo{typeValue: types.Nothing, nullState: NonNull}
	case ListCount, ListIndex:
		return expressionInfo{typeValue: types.Int, nullState: NonNull}
	case ListFilter:
		return expressionInfo{typeValue: receiver, nullState: NonNull}
	default:
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
}

func typeOperationHint(operation TypeOperation, receiver types.Type) string {
	if shape, known := stringOperationShapes[operation]; known {
		return shape.hint
	}
	if shape, known := numericShapes()[operation]; known {
		return shape.hint
	}
	if _, known := complexOperationResults[operation]; known {
		return "call " + string(operation) + " with no argument"
	}
	if shape, known := timeOperationShapes()[operation]; known {
		return shape.hint
	}
	if shape, known := regexOperationShapes()[operation]; known {
		return shape.hint
	}
	if shape, known := dataOperationShapes()[operation]; known {
		return shape.hint
	}
	if dataCallbackOperation(operation) {
		return dataCallbackHint(operation)
	}
	if shape, known := plotOperationShapes()[operation]; known {
		return shape.hint
	}
	if plotSeriesOperation(operation) {
		return "pass an x List, a y List, and a String label"
	}
	if shape, known := wordOperationShapes()[operation]; known {
		return shape.hint
	}
	if shape, known := pdfOperationShapes()[operation]; known {
		return shape.hint
	}
	if shape, known := excelOperationShapes()[operation]; known {
		return shape.hint
	}
	if shape, known := sqliteOperationShapes()[operation]; known {
		return shape.hint
	}
	element := types.Invalid
	if list, ok := receiver.(types.List); ok {
		element = list.Element
	}
	switch operation {
	case ListAdd:
		return "pass one element of the List element type"
	case ListEject:
		return "pass one Int index, which may be negative"
	case PairEject:
		return "pass one key of the Pair key type"
	case ListReverse:
		return "call reverse with no argument"
	case ListShuffle:
		return "call shuffle with no argument"
	case ListSort:
		return "call sort with no argument, or with one key Function"
	case ListCount, ListIndex:
		return "pass one " + types.Display(element) + " value"
	case ListMap:
		return "pass one Function taking " + types.Display(element)
	default:
		return "pass one Function taking " + types.Display(element) + " and returning Bool"
	}
}

// requireTypeOperationArity reports an argument-count mismatch and still
// analyzes the written arguments, so their own diagnostics are independent.
func (a *analyzer) requireTypeOperationArity(call *ast.CallExpr, operation TypeOperation, receiver types.Type, count int, current *scope, flow flowState) bool {
	if len(call.Arguments) == count {
		return true
	}
	a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument(s); received %d", operation, count, len(call.Arguments)), call.Span(), typeOperationHint(operation, receiver))
	a.analyzeTypeOperationArguments(call, current, flow, nil)
	return false
}

// analyzeStringOperation checks the immutable String operations. Every
// argument is a NonNull String and the receiver is never modified.
func (a *analyzer) analyzeStringOperation(call *ast.CallExpr, operation TypeOperation, shape stringOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	if !a.requireTypeOperationArity(call, operation, types.String, shape.arguments, current, flow) {
		return result
	}
	arguments := a.analyzeTypeOperationArguments(call, current, flow, types.String)
	for index, argument := range arguments {
		if argument.invalid() {
			continue
		}
		if argument.nullState != NonNull {
			a.nullableError(string(operation), call.Arguments[index].Value, argument.nullState)
			continue
		}
		if argument.typeValue.Kind() != types.StringKind {
			a.typeMismatch(call.Arguments[index].Span(), types.String, argument.typeValue, string(operation)+" argument")
		}
	}
	return result
}

// analyzeCollectionMutation checks the in-place List and Pair mutations.
func (a *analyzer) analyzeCollectionMutation(call *ast.CallExpr, operation TypeOperation, receiver expressionInfo, current *scope, flow flowState) expressionInfo {
	nothing := expressionInfo{typeValue: types.Nothing, nullState: NonNull}
	expectedArgument := collectionArgumentType(operation, receiver.typeValue)
	if !a.requireTypeOperationArity(call, operation, receiver.typeValue, 1, current, flow) {
		return nothing
	}
	argument := a.analyzeTypeOperationArguments(call, current, flow, expectedArgument)[0]
	if operation == ListAdd {
		elementNullable := false
		if list, ok := receiver.typeValue.(types.List); ok {
			elementNullable = list.ElementNullable
		}
		if !a.requireCompatibleNull(elementNullable, argument, call.Arguments[0].Span(), "List element") {
			return nothing
		}
	} else if argument.nullState != NonNull {
		a.nullableError(string(operation), call.Arguments[0].Value, argument.nullState)
		return nothing
	}
	if argument.nullState == Null || argument.invalid() {
		return nothing
	}
	if !types.Assignable(expectedArgument, argument.typeValue) {
		a.typeMismatch(call.Arguments[0].Span(), expectedArgument, argument.typeValue, string(operation)+" argument")
	}
	return nothing
}

// analyzeListSearch checks count and index. Both read the List with the
// ordinary == semantics and never mutate it.
func (a *analyzer) analyzeListSearch(call *ast.CallExpr, operation TypeOperation, receiver expressionInfo, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: types.Int, nullState: NonNull}
	element := listElement(receiver.typeValue)
	if !a.requireTypeOperationArity(call, operation, receiver.typeValue, 1, current, flow) {
		return result
	}
	argument := a.analyzeTypeOperationArguments(call, current, flow, element)[0]
	if argument.invalid() {
		return result
	}
	if argument.nullState != NonNull {
		a.nullableError(string(operation), call.Arguments[0].Value, argument.nullState)
		return result
	}
	if !types.Assignable(element, argument.typeValue) {
		a.typeMismatch(call.Arguments[0].Span(), element, argument.typeValue, string(operation)+" argument")
	}
	return result
}

// analyzeListSort checks both sort forms. The natural form orders comparable
// scalar elements; the key form orders by an Int, Real, or String key.
func (a *analyzer) analyzeListSort(call *ast.CallExpr, receiver expressionInfo, current *scope, flow flowState) expressionInfo {
	nothing := expressionInfo{typeValue: types.Nothing, nullState: NonNull}
	element := listElement(receiver.typeValue)
	if len(call.Arguments) == 0 {
		if element.Kind() != types.InvalidKind && !sortableKey(element) {
			a.error(codeCallArguments, fmt.Sprintf("sort does not order %s", types.Display(receiver.typeValue)), call.Span(), "sort List<Int>, List<Real>, or List<String>, or pass a key Function")
		}
		return nothing
	}
	if !a.requireTypeOperationArity(call, ListSort, receiver.typeValue, 1, current, flow) {
		return nothing
	}
	signature, ok, _ := a.analyzeListCallback(call, ListSort, element, nil, current, flow)
	if !ok {
		return nothing
	}
	if !sortableKey(signature.Return) {
		a.error(codeCallArguments, fmt.Sprintf("sort key Function returns %s", types.Display(signature.Return)), call.Arguments[0].Span(), "return Int, Real, or String from the key Function")
	}
	return nothing
}

// analyzeListTransform checks map and filter. Both read a snapshot of the
// receiver and build a new List, so a Constant receiver is valid.
func (a *analyzer) analyzeListTransform(call *ast.CallExpr, operation TypeOperation, receiver expressionInfo, current *scope, flow flowState) expressionInfo {
	element := listElement(receiver.typeValue)
	failure := typeOperationFailure(operation, receiver.typeValue)
	if !a.requireTypeOperationArity(call, operation, receiver.typeValue, 1, current, flow) {
		return failure
	}
	expectedReturn := types.Type(nil)
	if operation == ListFilter {
		expectedReturn = types.Bool
	}
	signature, ok, returnsNull := a.analyzeListCallback(call, operation, element, expectedReturn, current, flow)
	if !ok {
		return failure
	}
	if operation == ListFilter {
		if signature.Return.Kind() != types.BoolKind {
			a.typeMismatch(call.Arguments[0].Span(), types.Bool, signature.Return, "filter predicate result")
		}
		return expressionInfo{typeValue: receiver.typeValue, nullState: NonNull}
	}
	if signature.Return.Kind() == types.NothingKind {
		a.error(codeCallArguments, "map requires a Function that returns a value", call.Arguments[0].Span(), "return a value from the mapped Function")
		return failure
	}
	return expressionInfo{typeValue: types.List{Element: signature.Return, ElementNullable: returnsNull}, nullState: NonNull}
}

// analyzeListCallback checks one Function argument of a List operation. The
// callback must take exactly the element type, because List is invariant and
// no element is converted on the way into the call.
func (a *analyzer) analyzeListCallback(call *ast.CallExpr, operation TypeOperation, element, expectedReturn types.Type, current *scope, flow flowState) (*types.Signature, bool, bool) {
	var expected types.Type
	if element.Kind() != types.InvalidKind {
		expected = types.Function{Signature: &types.Signature{
			Parameters: []types.Parameter{{Name: "value", Type: element}}, Return: expectedReturn,
		}}
	}
	reported := a.bag.Len()
	info := a.analyzeExpressionExpected(call.Arguments[0].Value, current, flow, expected)
	if info.invalid() || a.bag.Len() != reported {
		// The argument already reported its own incompatibility; a second
		// derivative diagnostic about the same callback adds nothing.
		return nil, false, false
	}
	if info.nullState != NonNull {
		a.nullableError(string(operation), call.Arguments[0].Value, info.nullState)
		return nil, false, false
	}
	function, isFunction := info.typeValue.(types.Function)
	if !isFunction || function.Signature == nil {
		a.typeMismatch(call.Arguments[0].Span(), types.Function{}, info.typeValue, string(operation)+" argument")
		return nil, false, false
	}
	signature := function.Signature
	if len(signature.Parameters) != 1 || !types.Equal(signature.Parameters[0].Type, element) {
		a.error(codeCallArguments, fmt.Sprintf("%s requires a Function taking exactly one %s", operation, types.Display(element)), call.Arguments[0].Span(), typeOperationHint(operation, types.List{Element: element}))
		return nil, false, false
	}
	returnsNull := a.callbackReturnsNull(call.Arguments[0].Value, info)
	if operation != ListMap && returnsNull {
		a.error(codeNullableUse, fmt.Sprintf("Function argument for %s may return null", operation), call.Arguments[0].Span(), "return a NonNull value from the callback, or refine the result before returning it")
		return nil, false, false
	}
	return signature, true, returnsNull
}

// callbackReturnsNull reports whether the selected callback is known to return
// a possibly-null result.
func (a *analyzer) callbackReturnsNull(expression ast.Expr, info expressionInfo) bool {
	provided := a.result.SelectedFunctionValues[expression]
	if provided == nil {
		provided = concreteCallable(info)
	}
	return provided != nil && provided.ReturnNull != NonNull
}

// sortableKey names the element and key types the natural ordering supports.
func sortableKey(value types.Type) bool {
	switch value.Kind() {
	case types.IntKind, types.RealKind, types.StringKind:
		return true
	default:
		return false
	}
}

func listElement(receiver types.Type) types.Type {
	if list, ok := receiver.(types.List); ok {
		return list.Element
	}
	return types.Invalid
}

func (a *analyzer) analyzeTypeOperationArguments(call *ast.CallExpr, current *scope, flow flowState, expected types.Type) []expressionInfo {
	arguments := make([]expressionInfo, 0, len(call.Arguments))
	for _, argument := range call.Arguments {
		arguments = append(arguments, a.analyzeExpressionExpected(argument.Value, current, flow, expected))
	}
	return arguments
}

// collectionArgumentType is the statically required argument type of one
// built-in mutation.
func collectionArgumentType(operation TypeOperation, receiver types.Type) types.Type {
	switch operation {
	case ListAdd:
		if list, ok := receiver.(types.List); ok {
			return list.Element
		}
	case ListEject:
		return types.Int
	case PairEject:
		if pair, ok := receiver.(types.Pair); ok {
			return pair.Key
		}
	}
	return types.Invalid
}

func (a *analyzer) analyzeCallArguments(call *ast.CallExpr, current *scope, flow flowState, callable *Callable) []expressionInfo {
	arguments := make([]expressionInfo, 0, len(call.Arguments))
	for index, argument := range call.Arguments {
		var expected types.Type
		if callable != nil && callable.Signature != nil {
			if parameterIndex := callParameterIndex(call, callable.Signature, index); parameterIndex >= 0 {
				expected = callable.Signature.Parameters[parameterIndex].Type
			}
		}
		arguments = append(arguments, a.analyzeExpressionExpected(argument.Value, current, flow, expected))
	}
	return arguments
}

func callParameterIndex(call *ast.CallExpr, signature *types.Signature, argumentIndex int) int {
	if argumentIndex < 0 || argumentIndex >= len(call.Arguments) || signature == nil {
		return -1
	}
	name := call.Arguments[argumentIndex].Name
	if name == "" {
		if argumentIndex < len(signature.Parameters) {
			return argumentIndex
		}
		return -1
	}
	for index, parameter := range signature.Parameters {
		if parameter.Name == name {
			return index
		}
	}
	return -1
}

func (info expressionInfo) symbolCallable() *Callable {
	if info.symbol == nil {
		return nil
	}
	symbol := info.symbol
	if symbol.Alias != nil {
		symbol = symbol.Alias
	}
	return symbol.Callable
}

func (a *analyzer) analyzeBuiltinCall(call *ast.CallExpr, symbol *Symbol, arguments []expressionInfo) expressionInfo {
	switch symbol.Name {
	case "write":
		if len(arguments) != 1 {
			a.error(codeCallArguments, fmt.Sprintf("write expects 1 argument; received %d", len(arguments)), call.Span(), "pass exactly one non-Nothing value")
		} else if !renderable(arguments[0].typeValue) {
			a.error(codeCallArguments, fmt.Sprintf("write does not accept %s", types.Display(arguments[0].typeValue)), call.Arguments[0].Span(), "pass a value with a textual representation")
		}
		return expressionInfo{typeValue: types.Nothing, nullState: NonNull}
	case "take":
		return a.analyzeTakeCall(call, arguments)
	case "str":
		if len(arguments) != 1 {
			a.error(codeCallArguments, fmt.Sprintf("str expects 1 argument; received %d", len(arguments)), call.Span(), "pass exactly one non-Nothing value")
		} else if !renderable(arguments[0].typeValue) {
			a.error(codeCallArguments, fmt.Sprintf("str does not accept %s", types.Display(arguments[0].typeValue)), call.Arguments[0].Span(), "pass a value with a textual representation")
		}
		return expressionInfo{typeValue: types.String, nullState: NonNull}
	case "int":
		return a.analyzeNumericConversion(call, arguments, "int", types.Int, types.Real, types.String)
	case "real":
		return a.analyzeNumericConversion(call, arguments, "real", types.Real, types.Int, types.String)
	case "len":
		return a.analyzeLenCall(call, arguments)
	case "clear":
		return a.analyzeClearCall(call, arguments)
	case "between":
		return a.analyzeBetweenCall(call, arguments)
	case "abs":
		return a.analyzeAbsCall(call, arguments)
	case "sum", "min", "max":
		return a.analyzeNumericListCall(call, symbol.Name, arguments)
	case "type":
		return a.analyzeTypeCall(call, arguments)
	case "id":
		return a.analyzeIdCall(call, arguments)
	}
	return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
}

// analyzeTypeCall checks the runtime/introspection built-in type(value). It
// accepts any value, including one that is currently null or only possibly
// null: reporting the exact runtime type, including "Null" itself, is the
// entire point, so no narrowing is required.
func (a *analyzer) analyzeTypeCall(call *ast.CallExpr, arguments []expressionInfo) expressionInfo {
	if !a.checkBuiltinArity(call, "type", arguments, 1, "call type with exactly one value") {
		return expressionInfo{typeValue: types.String, nullState: NonNull}
	}
	if arguments[0].invalid() && arguments[0].nullState != Null {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	return expressionInfo{typeValue: types.String, nullState: NonNull}
}

// analyzeIdCall checks the runtime identity built-in id(reference). Only
// Class instances, List, and Pair carry a meaningful AhdCode reference
// identity in v0.1.8; every other type, and every not-yet-narrowed nullable
// reference, is a compile-time error.
func (a *analyzer) analyzeIdCall(call *ast.CallExpr, arguments []expressionInfo) expressionInfo {
	result := expressionInfo{typeValue: types.Int, nullState: NonNull}
	if !a.checkBuiltinArity(call, "id", arguments, 1, "call id with exactly one Class instance, List, or Pair value") {
		return result
	}
	argument := arguments[0]
	if argument.invalid() {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	if argument.nullState != NonNull {
		a.nullableError("id", call.Arguments[0].Value, argument.nullState)
		return result
	}
	switch value := argument.typeValue.(type) {
	case types.List, types.Pair:
	case types.Class:
		if value.Reference {
			a.error(codeCallArguments, "id does not accept a Class reference", call.Arguments[0].Span(), "pass a constructed Class instance")
		}
	default:
		a.error(codeCallArguments, fmt.Sprintf("id does not accept %s", types.Display(argument.typeValue)), call.Arguments[0].Span(), "pass a Class instance, List, or Pair value")
	}
	return result
}

// analyzeNumericConversion implements only the two explicit numeric
// conversions fixed by the v0.1 contract. It intentionally does not parse
// Strings or introduce truthiness through bool-like coercion.
func (a *analyzer) analyzeNumericConversion(call *ast.CallExpr, arguments []expressionInfo, name string, output types.Type, inputs ...types.Type) expressionInfo {
	expected := types.Display(inputs[0])
	if len(inputs) == 2 {
		expected += " or " + types.Display(inputs[1])
	}
	if len(arguments) != 1 {
		a.error(codeCallArguments, fmt.Sprintf("%s expects 1 %s argument; received %d argument(s)", name, expected, len(arguments)), call.Span(), fmt.Sprintf("call %s with exactly one %s value", name, expected))
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	argument := arguments[0]
	if argument.invalid() {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	if argument.nullState != NonNull {
		a.nullableError(name, call.Arguments[0].Value, argument.nullState)
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	accepted := false
	for _, input := range inputs {
		accepted = accepted || types.Equal(argument.typeValue, input)
	}
	if !accepted {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %s; received %s", name, expected, types.Display(argument.typeValue)), call.Arguments[0].Span(), "use one of the exact numeric conversion input types; AhdCode does not apply truthiness")
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	return expressionInfo{typeValue: output, nullState: NonNull}
}

// analyzeLenCall accepts only the v0.1 sized Fundamentals types.
func (a *analyzer) analyzeLenCall(call *ast.CallExpr, arguments []expressionInfo) expressionInfo {
	if len(arguments) != 1 {
		a.error(codeCallArguments, fmt.Sprintf("len expects 1 argument; received %d", len(arguments)), call.Span(), "pass exactly one String, List, or Pair value")
		return expressionInfo{typeValue: types.Int, nullState: NonNull}
	}
	argument := arguments[0]
	if argument.invalid() {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	switch argument.typeValue.Kind() {
	case types.StringKind, types.ListKind, types.PairKind:
	case types.InvalidKind:
	default:
		a.error(codeCallArguments, fmt.Sprintf("len does not accept %s", types.Display(argument.typeValue)), call.Arguments[0].Span(), "pass a String, List, or Pair value")
	}
	if argument.nullState != NonNull {
		a.nullableError("len", call.Arguments[0].Value, argument.nullState)
	}
	return expressionInfo{typeValue: types.Int, nullState: NonNull}
}

// analyzeClearCall enforces the v0.1 in-place collection-emptying contract.
func (a *analyzer) analyzeClearCall(call *ast.CallExpr, arguments []expressionInfo) expressionInfo {
	if len(arguments) != 1 {
		a.error(codeCallArguments, fmt.Sprintf("clear expects 1 argument; received %d", len(arguments)), call.Span(), "pass exactly one List or Pair value")
		return expressionInfo{typeValue: types.Nothing, nullState: NonNull}
	}
	argument := arguments[0]
	if argument.invalid() {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	switch argument.typeValue.Kind() {
	case types.ListKind, types.PairKind, types.InvalidKind:
	default:
		a.error(codeCallArguments, fmt.Sprintf("clear does not accept %s", types.Display(argument.typeValue)), call.Arguments[0].Span(), "pass a List or Pair value")
		return expressionInfo{typeValue: types.Nothing, nullState: NonNull}
	}
	if argument.nullState != NonNull {
		a.nullableError("clear", call.Arguments[0].Value, argument.nullState)
	}
	if target := argument.symbol; target != nil && constantTarget(target) {
		a.error(codeConstantAssignment, fmt.Sprintf("cannot clear Constant %q", target.Name), call.Arguments[0].Span(), "clear mutates the collection in place; declare it without Constant")
	}
	return expressionInfo{typeValue: types.Nothing, nullState: NonNull}
}

// analyzeTakeCall checks the terminal input built-in. Its two forms are
// take() -> String and take(prompt: String) -> String; the returned text is
// never parsed or coerced into another type.
func (a *analyzer) analyzeTakeCall(call *ast.CallExpr, arguments []expressionInfo) expressionInfo {
	result := expressionInfo{typeValue: types.String, nullState: NonNull}
	if len(arguments) > 1 {
		a.error(codeCallArguments, fmt.Sprintf("take expects at most 1 prompt argument; received %d", len(arguments)), call.Span(), "call take() or take(prompt)")
		return result
	}
	if len(arguments) == 0 {
		return result
	}
	if call.Arguments[0].Name != "" {
		a.error(codeCallArguments, "take does not accept a named argument", call.Arguments[0].Span(), "pass the prompt positionally")
		return result
	}
	prompt := arguments[0]
	if prompt.invalid() {
		return result
	}
	if prompt.nullState != NonNull {
		a.nullableError("take prompt", call.Arguments[0].Value, prompt.nullState)
		return result
	}
	if prompt.typeValue.Kind() != types.StringKind {
		a.typeMismatch(call.Arguments[0].Span(), types.String, prompt.typeValue, "take prompt")
	}
	return result
}

// analyzeAbsCall checks the numeric magnitude built-in. Its two overloads are
// abs(Int) -> Int and abs(Real) -> Real; the result type is exactly the
// argument type, so no numeric widening is introduced by the call itself.
func (a *analyzer) analyzeAbsCall(call *ast.CallExpr, arguments []expressionInfo) expressionInfo {
	invalid := expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	if !a.checkBuiltinArity(call, "abs", arguments, 1, "call abs with exactly one Int or Real value") {
		return invalid
	}
	argument := arguments[0]
	if argument.invalid() {
		return invalid
	}
	if argument.nullState != NonNull {
		a.nullableError("abs", call.Arguments[0].Value, argument.nullState)
		return invalid
	}
	switch argument.typeValue.Kind() {
	case types.IntKind:
		return expressionInfo{typeValue: types.Int, nullState: NonNull}
	case types.RealKind:
		return expressionInfo{typeValue: types.Real, nullState: NonNull}
	}
	a.error(codeCallArguments, fmt.Sprintf("abs does not accept %s", types.Display(argument.typeValue)), call.Arguments[0].Span(), "pass an Int or Real value; AhdCode applies no implicit conversion")
	return invalid
}

// analyzeNumericListCall checks the numeric List reductions. Each one has the
// two overloads List<Int> -> Int and List<Real> -> Real, reads the List
// without mutating it, and never treats List<Int> as List<Real>.
func (a *analyzer) analyzeNumericListCall(call *ast.CallExpr, name string, arguments []expressionInfo) expressionInfo {
	invalid := expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	hint := "call " + name + " with exactly one List<Int> or List<Real> value"
	if !a.checkBuiltinArity(call, name, arguments, 1, hint) {
		return invalid
	}
	argument := arguments[0]
	if argument.invalid() {
		return invalid
	}
	if argument.nullState != NonNull {
		a.nullableError(name, call.Arguments[0].Value, argument.nullState)
		return invalid
	}
	if list, ok := argument.typeValue.(types.List); ok {
		switch list.Element.Kind() {
		case types.IntKind:
			return expressionInfo{typeValue: types.Int, nullState: NonNull}
		case types.RealKind:
			return expressionInfo{typeValue: types.Real, nullState: NonNull}
		case types.InvalidKind:
			return invalid
		}
	}
	a.error(codeCallArguments, fmt.Sprintf("%s does not accept %s", name, types.Display(argument.typeValue)), call.Arguments[0].Span(), hint)
	return invalid
}

// checkBuiltinArity applies the shared call shape of the Fundamentals
// entry points: an exact positional argument count and no named argument,
// because a built-in operation publishes no parameter names.
func (a *analyzer) checkBuiltinArity(call *ast.CallExpr, name string, arguments []expressionInfo, count int, hint string) bool {
	if len(arguments) != count {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument; received %d", name, count, len(arguments)), call.Span(), hint)
		return false
	}
	for _, argument := range call.Arguments {
		if argument.Name != "" {
			a.error(codeCallArguments, fmt.Sprintf("%s does not accept a named argument", name), argument.Span(), hint)
			return false
		}
	}
	return true
}

// renderable reports whether a value has canonical str text. Nothing is not a
// value, and a lazy range is iteration state rather than a rendered value.
func renderable(value types.Type) bool {
	switch value.Kind() {
	case types.NothingKind, types.RangeKind:
		return false
	default:
		return true
	}
}

// analyzeBetweenCall checks the lazy integer iteration built-in. It takes one
// to three Int arguments and yields Int values when iterated; it never
// produces a List.
func (a *analyzer) analyzeBetweenCall(call *ast.CallExpr, arguments []expressionInfo) expressionInfo {
	result := expressionInfo{typeValue: types.IntRange, nullState: NonNull}
	if len(arguments) < 1 || len(arguments) > 3 {
		a.error(codeCallArguments, fmt.Sprintf("between expects 1 to 3 arguments; received %d", len(arguments)), call.Span(), "call between(stop), between(start, stop), or between(start, stop, step)")
		return result
	}
	for index, argument := range arguments {
		if index >= len(call.Arguments) {
			break
		}
		if call.Arguments[index].Name != "" {
			a.error(codeCallArguments, "between does not accept named arguments", call.Arguments[index].Span(), "pass between arguments positionally")
			continue
		}
		if argument.nullState != NonNull {
			a.nullableError("between", call.Arguments[index].Value, argument.nullState)
			continue
		}
		if argument.typeValue.Kind() != types.IntKind {
			a.typeMismatch(call.Arguments[index].Span(), types.Int, argument.typeValue, betweenArgumentName(index, len(arguments)))
		}
	}
	return result
}

func betweenArgumentName(index, count int) string {
	if count == 1 {
		return "between stop"
	}
	switch index {
	case 0:
		return "between start"
	case 1:
		return "between stop"
	default:
		return "between step"
	}
}

func constantTarget(symbol *Symbol) bool {
	if symbol.Alias != nil {
		return symbol.Constant || constantTarget(symbol.Alias)
	}
	return symbol.Constant
}

func (a *analyzer) validateCallArguments(call *ast.CallExpr, callable *Callable, arguments []expressionInfo) {
	if callable == nil || callable.Signature == nil {
		if len(arguments) != 0 {
			a.error(codeCallArguments, "Class has no declared construction parameters", call.Span(), fmt.Sprintf("received %d argument(s)", len(arguments)))
		}
		return
	}
	a.callArgumentsCompatible(call, callable, arguments, true)
}

func (a *analyzer) callArgumentsCompatible(call *ast.CallExpr, callable *Callable, arguments []expressionInfo, diagnose bool) bool {
	parameters := callable.Signature.Parameters
	valid := true
	assigned := make([]bool, len(parameters))
	named := len(call.Arguments) > 0 && call.Arguments[0].Name != ""
	for index, argument := range call.Arguments {
		parameterIndex := index
		if named {
			parameterIndex = -1
			for candidate := range parameters {
				if parameters[candidate].Name == argument.Name {
					parameterIndex = candidate
					break
				}
			}
		}
		if parameterIndex < 0 || parameterIndex >= len(parameters) || assigned[parameterIndex] {
			valid = false
			if diagnose {
				a.error(codeCallArguments, fmt.Sprintf("unexpected or duplicate call argument %q", argument.Name), argument.Span(), "match the callable parameter names and provide each once")
			}
			continue
		}
		assigned[parameterIndex] = true
		if arguments[index].invalid() {
			continue
		}
		if arguments[index].nullState != NonNull && parameterIndex < len(callable.ParameterNull) && callable.ParameterNull[parameterIndex] == NonNull {
			valid = false
			if diagnose {
				a.error(codeNullableUse, fmt.Sprintf("argument for %s may be null", parameters[parameterIndex].Name), argument.Span(), "refine or assign the value to NonNull before this call")
			}
		} else if _, compatible := conversionQuality(parameters[parameterIndex].Type, arguments[index].typeValue); arguments[index].nullState != Null && !compatible {
			valid = false
			if diagnose {
				a.typeMismatch(argument.Span(), parameters[parameterIndex].Type, arguments[index].typeValue, fmt.Sprintf("argument %s", parameters[parameterIndex].Name))
			}
		} else if !a.callableNullContractSatisfied(callable, parameterIndex, argument.Value, arguments[index]) {
			valid = false
			if diagnose {
				a.error(codeNullableUse, fmt.Sprintf("Function argument for %s may return null, but the callback contract is NonNull", parameters[parameterIndex].Name), argument.Span(), "return a NonNull value from the callback, or refine the result before returning it")
			}
		}
	}
	for index, parameter := range parameters {
		if !assigned[index] && !parameter.HasDefault {
			valid = false
			if diagnose {
				a.error(codeCallArguments, fmt.Sprintf("missing required argument %q", parameter.Name), call.Span(), "provide all required arguments")
			}
		}
	}
	return valid
}

// callableNullContractSatisfied keeps a callback's null-state contract intact
// across a call boundary: a Function argument may not return null where the
// parameter contract promises a NonNull result.
func (a *analyzer) callableNullContractSatisfied(callable *Callable, index int, expression ast.Expr, info expressionInfo) bool {
	if callable == nil || index >= len(callable.Signature.Parameters) {
		return true
	}
	if _, ok := callable.Signature.Parameters[index].Type.(types.Function); !ok {
		return true
	}
	expected := parameterCallableNull(callable, index)
	if expected != NonNull {
		return true
	}
	provided := a.result.SelectedFunctionValues[expression]
	if provided == nil {
		provided = concreteCallable(info)
	}
	return provided == nil || provided.ReturnNull == NonNull
}

// parameterCallableNull is the return null-state the parameter's own callable
// contract promises, when that contract is known.
func parameterCallableNull(callable *Callable, index int) NullState {
	if callable.Structure == nil && callable.Declaration == nil {
		return MaybeNull
	}
	return NonNull
}

func (a *analyzer) analyzeMember(member *ast.MemberExpr, current *scope, flow flowState) expressionInfo {
	return a.analyzeMemberOf(member, a.analyzeExpression(member.Object, current, flow), current, flow)
}

// analyzeMemberOf resolves one member against an already-analyzed receiver, so
// a call site can decide between a built-in type operation and an ordinary
// member without analyzing the receiver expression twice.
func (a *analyzer) analyzeMemberOf(member *ast.MemberExpr, object expressionInfo, current *scope, flow flowState) expressionInfo {
	if object.invalid() {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	if object.symbol != nil && object.symbol.SuperClassBinding {
		return a.analyzeSuperMember(member, object)
	}
	if object.symbol != nil && object.symbol.Kind == NamespaceSymbol && object.symbol.Namespace != nil {
		resolved, exists := object.symbol.Namespace.Symbols[member.Name]
		if !exists {
			a.error(CodeNamespaceMember, fmt.Sprintf("module %s has no symbol %q", object.symbol.Namespace.Name, member.Name), member.Span(), "use a symbol exported by the module")
			return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
		}
		if resolved.Confidential {
			a.error(CodeConfidentialAccess, fmt.Sprintf("symbol %q in module %s is Confidential", member.Name, object.symbol.Namespace.Name), member.Span(), "access only public module exports")
			return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
		}
		resolved = object.symbol.Namespace.Exports[member.Name]
		if resolved == nil {
			a.error(CodeNamespaceMember, fmt.Sprintf("module %s does not export symbol %q", object.symbol.Namespace.Name, member.Name), member.Span(), "use a public symbol exported by the module")
			return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
		}
		a.result.ResolvedSymbols[member] = resolved
		typeValue := resolved.Type
		if resolved.Kind == ClassSymbol {
			typeValue = types.Class{Symbol: resolved.Class, Reference: true}
		}
		if operation, known := moduleOperationOf(resolved); known && !a.calleeExpressions[member] {
			a.rejectModuleOperationValue(operation, member.Span())
			typeValue = types.Invalid
		}
		return expressionInfo{typeValue: typeValue, nullState: resolved.InitialNull, symbol: resolved}
	}
	if object.nullState != NonNull {
		a.nullableError("member access", member.Object, object.nullState)
	}
	class, ok := object.typeValue.(types.Class)
	if !ok || class.Reference {
		a.error(codeInvalidMember, fmt.Sprintf("type %s has no Class members", types.Display(object.typeValue)), member.Span(), fmt.Sprintf("cannot access member %q", member.Name))
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	classSymbol := a.classSymbolFor(class.Symbol)
	resolved := a.lookupMember(classSymbol, member.Name)
	if resolved == nil {
		a.error(codeInvalidMember, fmt.Sprintf("Class %s has no member %q", class.Symbol.Name, member.Name), member.Span(), "declare the attribute or method in the Class hierarchy")
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	if resolved.Confidential && !a.canAccessConfidentialMember(current, resolved.OwnerClass) {
		a.error(CodeConfidentialAccess, fmt.Sprintf("Class member %q is Confidential", member.Name), member.Span(), "access Confidential members only from their defining Class or a subclass")
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	a.result.ResolvedSymbols[member] = resolved
	return expressionInfo{typeValue: resolved.Type, nullState: resolved.InitialNull, symbol: resolved}
}

func (a *analyzer) canAccessConfidentialMember(current *scope, owner *types.ClassSymbol) bool {
	return owner != nil && current != nil && current.callable != nil && current.callable.class != nil &&
		classAssignableTo(current.callable.class.Class, owner)
}

// relatedClasses reports whether two Class instance types share an ancestry,
// so a parent-typed and a child-typed reference to one object compare.
func relatedClasses(left, right types.Type) bool {
	leftClass, leftOK := left.(types.Class)
	rightClass, rightOK := right.(types.Class)
	if !leftOK || !rightOK || leftClass.Reference || rightClass.Reference {
		return false
	}
	return classAssignableTo(leftClass.Symbol, rightClass.Symbol) || classAssignableTo(rightClass.Symbol, leftClass.Symbol)
}

// analyzeSuperMember resolves SuperClass.member to the parent implementation
// while keeping the current instance as the receiver.
func (a *analyzer) analyzeSuperMember(member *ast.MemberExpr, object expressionInfo) expressionInfo {
	parent := a.classSymbolFor(object.symbol.Class)
	resolved := a.lookupMember(parent, member.Name)
	if resolved == nil {
		a.error(codeInvalidMember, fmt.Sprintf("parent Class has no member %q", member.Name), member.Span(), "call a member declared by the direct superclass or one of its ancestors")
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	if resolved.Kind != FunctionSymbol {
		a.error(codeInvalidMember, fmt.Sprintf("SuperClass.%s is not a parent Function", member.Name), member.Span(), "use attribute.%s for inherited attribute access")
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	a.result.ResolvedSymbols[member] = resolved
	a.result.SuperCalls[member] = true
	return expressionInfo{typeValue: resolved.Type, nullState: resolved.InitialNull, symbol: resolved}
}

func (a *analyzer) lookupMember(class *Symbol, name string) *Symbol {
	visited := make(map[*types.ClassSymbol]bool)
	for current := class; current != nil && current.Class != nil && !visited[current.Class]; {
		visited[current.Class] = true
		if member := current.Members[name]; member != nil {
			return member
		}
		if current.Class == nil || current.Class.Parent == nil {
			break
		}
		current = a.classSymbolFor(current.Class.Parent)
	}
	return nil
}

func (a *analyzer) analyzeIndex(index *ast.IndexExpr, current *scope, flow flowState) expressionInfo {
	object := a.analyzeExpression(index.Object, current, flow)
	position := a.analyzeExpression(index.Index, current, flow)
	if object.invalid() || position.invalid() {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	if object.nullState != NonNull {
		a.nullableError("index", index.Object, object.nullState)
	}
	if position.nullState != NonNull {
		a.nullableError("index", index.Index, position.nullState)
	}
	switch collection := object.typeValue.(type) {
	case types.List:
		if position.typeValue.Kind() != types.IntKind {
			a.typeMismatch(index.Index.Span(), types.Int, position.typeValue, "List index")
		}
		return expressionInfo{typeValue: collection.Element, nullState: nullStateFor(collection.ElementNullable)}
	case types.Pair:
		if !types.Assignable(collection.Key, position.typeValue) {
			a.typeMismatch(index.Index.Span(), collection.Key, position.typeValue, "Pair key")
		}
		return expressionInfo{typeValue: collection.Value, nullState: nullStateFor(collection.ValueNullable)}
	default:
		if object.typeValue.Kind() == types.StringKind {
			if position.typeValue.Kind() != types.IntKind {
				a.typeMismatch(index.Index.Span(), types.Int, position.typeValue, "String index")
			}
			return expressionInfo{typeValue: types.String, nullState: NonNull}
		}
	}
	a.error(codeOperatorType, fmt.Sprintf("type %s is not indexable", types.Display(object.typeValue)), index.Object.Span(), "index a List, Pair, or String")
	return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
}

func (a *analyzer) analyzeSlice(slice *ast.SliceExpr, current *scope, flow flowState) expressionInfo {
	object := a.analyzeExpression(slice.Object, current, flow)
	invalid := object.invalid()
	for _, bound := range []ast.Expr{slice.Start, slice.End} {
		if bound == nil {
			continue
		}
		info := a.analyzeExpression(bound, current, flow)
		if info.invalid() {
			invalid = true
			continue
		}
		if info.nullState != NonNull || info.typeValue.Kind() != types.IntKind {
			a.typeMismatch(bound.Span(), types.Int, info.typeValue, "slice bound")
		}
	}
	if invalid {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
	}
	if object.nullState != NonNull {
		a.nullableError("slice", slice.Object, object.nullState)
	}
	if _, ok := object.typeValue.(types.List); ok || object.typeValue.Kind() == types.StringKind {
		return expressionInfo{typeValue: object.typeValue, nullState: NonNull}
	}
	a.error(codeOperatorType, fmt.Sprintf("type %s is not sliceable", types.Display(object.typeValue)), slice.Object.Span(), "slice a List or String")
	return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
}

func (a *analyzer) analyzeList(list *ast.ListExpr, current *scope, flow flowState) expressionInfo {
	return a.analyzeListExpected(list, current, flow, nil)
}

// analyzeListExpected infers a List literal from its elements, and falls back
// to the surrounding expected type only when the literal carries no element
// type of its own. Successful inference is never overridden, so generic
// invariance is unaffected.
func (a *analyzer) analyzeListExpected(list *ast.ListExpr, current *scope, flow flowState, expected types.Type) expressionInfo {
	expectedElement, expectedElementNullable := expectedListElement(expected)
	_, hasExpectedList := expected.(types.List)
	elementType := types.Invalid
	sawNull := false
	for _, element := range list.Elements {
		info := a.analyzeCollectionEntry(element, current, flow, expectedElement)
		if info.nullState != NonNull {
			sawNull = true
			if hasExpectedList && !expectedElementNullable {
				a.error(codeNullNotAllowed, "List element is not nullable; received a value that may be null", element.Span(), "declare the List with a nullable element type, e.g. List<T?>")
			}
		}
		if info.nullState == Null {
			continue
		}
		if types.IsInvalid(elementType) {
			elementType = info.typeValue
			continue
		}
		if types.IsNumeric(elementType) && types.IsNumeric(info.typeValue) {
			if elementType.Kind() == types.ComplexKind || info.typeValue.Kind() == types.ComplexKind {
				elementType = types.Complex
			} else if elementType.Kind() == types.RealKind || info.typeValue.Kind() == types.RealKind {
				elementType = types.Real
			}
			continue
		}
		if !types.Equal(elementType, info.typeValue) {
			a.typeMismatch(element.Span(), elementType, info.typeValue, "List element")
		}
	}
	if types.IsInvalid(elementType) {
		elementType = expectedElement
		if types.IsInvalid(elementType) && !declaredCollectionRejected(expected) {
			a.error(codeCollectionInference, "List element type cannot be inferred", list.Span(), "declare the collection type, as in values: List<Int> := []")
		}
	}
	elementNullable := sawNull
	if hasExpectedList {
		elementNullable = expectedElementNullable
	}
	return expressionInfo{typeValue: types.List{Element: elementType, ElementNullable: elementNullable}, nullState: NonNull}
}

// analyzeCollectionEntry analyzes one literal entry. A collection context is
// propagated so a nested empty literal is contextually typed; every other
// expression is analyzed exactly as before.
func (a *analyzer) analyzeCollectionEntry(expression ast.Expr, current *scope, flow flowState, expected types.Type) expressionInfo {
	switch expected.(type) {
	case types.List, types.Pair:
		return a.analyzeExpressionExpected(expression, current, flow, expected)
	default:
		return a.analyzeExpression(expression, current, flow)
	}
}

// expectedListElement is the element type (and its nullability) a List
// literal may adopt from its surrounding context.
func expectedListElement(expected types.Type) (types.Type, bool) {
	if declared, ok := expected.(types.List); ok {
		return declared.Element, declared.ElementNullable
	}
	return types.Invalid, false
}

// declaredCollectionRejected reports whether the surrounding collection type
// is itself already in error, so an uninferable literal stays quiet instead of
// restating the same root cause.
func declaredCollectionRejected(expected types.Type) bool {
	switch declared := expected.(type) {
	case types.List:
		return types.IsInvalid(declared.Element)
	case types.Pair:
		return types.IsInvalid(declared.Key) || types.IsInvalid(declared.Value)
	default:
		return false
	}
}

func (a *analyzer) analyzePair(pair *ast.PairExpr, current *scope, flow flowState) expressionInfo {
	return a.analyzePairExpected(pair, current, flow, nil)
}

// analyzePairExpected checks a Pair literal against the v0.1 key rules: keys
// use only the stable simple scalar types, and one literal may not repeat a
// key. When the declared type already reported an invalid key type, the
// literal degrades quietly instead of restating the same root cause.
func (a *analyzer) analyzePairExpected(pair *ast.PairExpr, current *scope, flow flowState, expected types.Type) expressionInfo {
	expectedKey, expectedValue, expectedValueNullable := expectedPairTypes(expected)
	_, hasExpectedPair := expected.(types.Pair)
	keyType, valueType := types.Invalid, types.Invalid
	rejectedKey := false
	sawNullValue := false
	seen := make(map[string]source.Span)
	for _, entry := range pair.Entries {
		// A Pair key is always a simple scalar, so only the value position
		// carries a collection context.
		key := a.analyzeExpression(entry.Key, current, flow)
		value := a.analyzeCollectionEntry(entry.Value, current, flow, expectedValue)
		if key.nullState == Null {
			a.error(codeInvalidPairKey, "a Pair key must not be null", entry.Key.Span(), "use a NonNull String, Int, or Bool key")
			rejectedKey = true
			continue
		}
		if value.nullState != NonNull {
			sawNullValue = true
			if hasExpectedPair && !expectedValueNullable {
				a.error(codeNullNotAllowed, "Pair value is not nullable; received a value that may be null", entry.Value.Span(), "declare the Pair with a nullable value type, e.g. Pair<K, V?>")
			}
		}
		keyType = a.mergeLiteralType(keyType, key.typeValue, entry.Key)
		// A bare null value carries no type of its own (Invalid), so it must not
		// be merged into the inferred value type -- mirroring how a List literal
		// skips a null element for the same reason.
		if value.nullState != Null {
			valueType = a.mergeLiteralType(valueType, value.typeValue, entry.Value)
		}
		a.checkDuplicatePairKey(entry.Key, seen)
	}
	if !types.IsInvalid(keyType) && !types.IsPairKey(keyType) {
		if !declaredPairKeyRejected(expected) {
			a.error(codeInvalidPairKey, fmt.Sprintf("Pair key type must be String, Int, or Bool; received %s", types.Display(keyType)), pair.Span(), "use String, Int, or Bool keys")
		}
		keyType = types.Invalid
	}
	valueNullable := sawNullValue
	if hasExpectedPair {
		valueNullable = expectedValueNullable
	}
	// An empty literal adopts the surrounding expected type. A literal that has
	// entries keeps whatever its keys resolved to, so a rejected key is not
	// masked by the declared type.
	if len(pair.Entries) == 0 {
		keyType, valueType = expectedKey, expectedValue
		if (types.IsInvalid(keyType) || types.IsInvalid(valueType)) && !declaredCollectionRejected(expected) {
			a.error(codeCollectionInference, "Pair key and value types cannot be inferred", pair.Span(), "declare the collection type, as in scores: Pair<String, Int> := {}")
		}
		return expressionInfo{typeValue: types.Pair{Key: keyType, Value: valueType, ValueNullable: valueNullable}, nullState: NonNull}
	}
	// A value position may hold only null values, which carry no type of their
	// own; the declared value type then applies.
	if types.IsInvalid(valueType) {
		valueType = expectedValue
	}
	// An already-reported key keeps the declared key type so the rejection is
	// not restated as an assignability mismatch.
	if rejectedKey && types.IsInvalid(keyType) {
		keyType = expectedKey
	}
	return expressionInfo{typeValue: types.Pair{Key: keyType, Value: valueType, ValueNullable: valueNullable}, nullState: NonNull}
}

// expectedPairTypes are the key type, value type, and value nullability a
// Pair literal may adopt from its surrounding context.
func expectedPairTypes(expected types.Type) (types.Type, types.Type, bool) {
	if declared, ok := expected.(types.Pair); ok {
		return declared.Key, declared.Value, declared.ValueNullable
	}
	return types.Invalid, types.Invalid, false
}

// declaredPairKeyRejected reports whether the declared Pair type already
// rejected its key type, which degrades that key to Invalid.
func declaredPairKeyRejected(expected types.Type) bool {
	declared, ok := expected.(types.Pair)
	return ok && types.IsInvalid(declared.Key)
}

// checkDuplicatePairKey rejects a key that repeats an earlier key in the same
// literal. Keys are compared by their compile-time value, so equivalent
// spellings of one Int are the same key.
func (a *analyzer) checkDuplicatePairKey(expression ast.Expr, seen map[string]source.Span) {
	constant, failure := a.evaluateConstant(expression)
	if failure != constOK || constant == nil {
		return
	}
	identity, ok := pairKeyIdentity(constant)
	if !ok {
		return
	}
	if previous, exists := seen[identity]; exists {
		a.error(codeDuplicatePairKey, fmt.Sprintf("duplicate Pair key %s in one Pair literal", pairKeyText(constant)), expression.Span(), fmt.Sprintf("the same key is already given at line %d, column %d", previous.Start.Line, previous.Start.Column))
		return
	}
	seen[identity] = expression.Span()
}

// pairKeyIdentity is the canonical compile-time identity of a Pair key. Only
// the v0.1 key types have one.
func pairKeyIdentity(constant *constantValue) (string, bool) {
	switch constant.typeValue.Kind() {
	case types.StringKind:
		return "String\x00" + constant.text, true
	case types.IntKind:
		if constant.integer == nil {
			return "", false
		}
		return "Int\x00" + constant.integer.String(), true
	case types.BoolKind:
		return "Bool\x00" + strconv.FormatBool(constant.boolean), true
	default:
		return "", false
	}
}

func pairKeyText(constant *constantValue) string {
	switch constant.typeValue.Kind() {
	case types.StringKind:
		return strconv.Quote(constant.text)
	case types.IntKind:
		return constant.integer.String()
	default:
		return strconv.FormatBool(constant.boolean)
	}
}

func (a *analyzer) mergeLiteralType(current, next types.Type, expression ast.Expr) types.Type {
	if types.IsInvalid(current) {
		return next
	}
	if types.IsNumeric(current) && types.IsNumeric(next) {
		if current.Kind() == types.ComplexKind || next.Kind() == types.ComplexKind {
			return types.Complex
		}
		if current.Kind() == types.RealKind || next.Kind() == types.RealKind {
			return types.Real
		}
		return types.Int
	}
	if !types.Equal(current, next) {
		a.typeMismatch(expression.Span(), current, next, "collection element")
	}
	return current
}

func (a *analyzer) operatorError(operator string, left, right types.Type, span source.Span) {
	received := types.Display(left)
	if right != nil {
		received += " and " + types.Display(right)
	}
	a.error(codeOperatorType, fmt.Sprintf("operator %s does not accept %s", operator, received), span, "use operands allowed by AhdCode's built-in operator table")
}

func (a *analyzer) nullableError(operation string, expression ast.Expr, state NullState) {
	a.error(codeNullableUse, fmt.Sprintf("%s requires a NonNull value; received %s", operation, state), expression.Span(), "guard with != null or assign a non-null value before use")
}

func (a *analyzer) initializerNullState(expression ast.Expr) NullState {
	if literal, ok := expression.(*ast.LiteralExpr); ok && literal.Kind == ast.NullLiteral {
		return Null
	}
	return NonNull
}

func (a *analyzer) applyRefinements(expression ast.Expr, truth bool, flow flowState) {
	binary, ok := expression.(*ast.BinaryExpr)
	if !ok {
		if group, grouped := expression.(*ast.GroupExpr); grouped {
			a.applyRefinements(group.Expression, truth, flow)
		}
		return
	}
	if binary.Operator == "and" && truth {
		a.applyRefinements(binary.Left, true, flow)
		a.applyRefinements(binary.Right, true, flow)
		return
	}
	if binary.Operator == "or" && !truth {
		a.applyRefinements(binary.Left, false, flow)
		a.applyRefinements(binary.Right, false, flow)
		return
	}
	var identifier *ast.IdentifierExpr
	if left, ok := binary.Left.(*ast.IdentifierExpr); ok && isNullLiteral(binary.Right) {
		identifier = left
	} else if right, ok := binary.Right.(*ast.IdentifierExpr); ok && isNullLiteral(binary.Left) {
		identifier = right
	}
	if identifier == nil {
		return
	}
	symbol := a.result.ResolvedSymbols[identifier]
	if symbol == nil {
		return
	}
	nonNullWhenTrue := binary.Operator == "!=" || binary.Operator == "is not"
	nullWhenTrue := binary.Operator == "==" || binary.Operator == "is"
	if !nonNullWhenTrue && !nullWhenTrue {
		return
	}
	state := Null
	if truth == nonNullWhenTrue {
		state = NonNull
	}
	flow[symbol] = state
}

func isNullLiteral(expression ast.Expr) bool {
	literal, ok := expression.(*ast.LiteralExpr)
	return ok && literal.Kind == ast.NullLiteral
}
