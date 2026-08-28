package semantic

import (
	"fmt"
	"math/big"

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
	case *ast.CallExpr:
		info := a.analyzeCallExpected(value, current, flow, expected)
		a.result.ExpressionTypes[expression] = info.typeValue
		a.result.NullStates[expression] = info.nullState
		return info
	case *ast.IdentifierExpr:
		return a.analyzeFunctionValueIdentifier(value, current, flow, expected)
	default:
		return a.analyzeExpression(expression, current, flow)
	}
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
	if owner == a.module && current.callable != nil && !symbol.Builtin && symbol.Kind != ClassSymbol && symbol.Kind != NamespaceSymbol && current.callable.symbol != symbol {
		a.error(codeHiddenGlobal, fmt.Sprintf("module binding %q requires an explicit Global declaration", identifier.Name), identifier.Span(), fmt.Sprintf("add %s: Global %s in this callable", identifier.Name, types.Display(symbol.Type)))
	}
	typeValue := symbol.Type
	if symbol.Kind == ClassSymbol {
		typeValue = types.Class{Symbol: symbol.Class, Reference: true}
	}
	constant := symbol.ConstValue
	return expressionInfo{typeValue: typeValue, nullState: flow.state(symbol), symbol: symbol, constant: constant}
}

func (a *analyzer) analyzeUnary(expression *ast.UnaryExpr, current *scope, flow flowState) expressionInfo {
	allowMagnitude := expression.Operator == "-"
	operand := a.analyzeExpressionWithMagnitude(expression.Operand, current, flow, allowMagnitude)
	if operand.nullState != NonNull {
		a.nullableError(expression.Operator, expression.Operand, operand.nullState)
	}
	switch expression.Operator {
	case "+", "-":
		if !types.IsNumeric(operand.typeValue) {
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
	}
	if right.nullState != NonNull {
		a.nullableError(operator, expression.Right, right.nullState)
	}
	resultType := a.binaryOperatorType(operator, left.typeValue, right.typeValue, expression.Right)
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
	compatible := types.Equal(left.typeValue, right.typeValue) || (types.IsNumeric(left.typeValue) && types.IsNumeric(right.typeValue))
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

func (a *analyzer) binaryOperatorType(operator string, left, right types.Type, rightExpression ast.Expr) types.Type {
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
		if left.Kind() == types.IntKind && right.Kind() == types.IntKind {
			constant, failure := a.evaluateConstant(rightExpression)
			if failure == constOK && constant != nil && constant.integer != nil && constant.integer.Sign() >= 0 {
				return types.Int
			}
			return types.Real
		}
		if types.IsNumeric(left) && types.IsNumeric(right) {
			return types.Real
		}
		return types.Invalid
	}
	if operator == "/" {
		if types.IsNumeric(left) && types.IsNumeric(right) {
			return types.Real
		}
		return types.Invalid
	}
	if operator == "+" || operator == "-" || operator == "*" {
		if types.IsNumeric(left) && types.IsNumeric(right) {
			if left.Kind() == types.RealKind || right.Kind() == types.RealKind {
				return types.Real
			}
			return types.Int
		}
		return types.Invalid
	}
	if operator == "<" || operator == "<=" || operator == ">" || operator == ">=" {
		if types.IsNumeric(left) && types.IsNumeric(right) {
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
	if symbol.Alias != nil {
		symbol = symbol.Alias
	}
	expectedFunction, hasExpectedFunction := expected.(types.Function)
	if symbol.inference != nil && hasExpectedFunction && expectedFunction.Signature != nil {
		a.constrainConcreteFunction(symbol, expectedFunction.Signature, identifier.Span())
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
	callee := a.analyzeExpression(call.Callee, current, flow)
	if callee.symbol != nil && callee.symbol.Builtin {
		arguments := a.analyzeCallArguments(call, current, flow, nil)
		return a.analyzeBuiltinCall(call, callee.symbol, arguments)
	}
	if class, ok := callee.typeValue.(types.Class); ok && class.Reference {
		symbol := a.classSymbolFor(class.Symbol)
		if symbol == nil {
			return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
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
		return expressionInfo{typeValue: types.Nothing, nullState: NonNull}
	case "str":
		if len(arguments) != 1 {
			a.error(codeCallArguments, fmt.Sprintf("str expects 1 argument; received %d", len(arguments)), call.Span(), "pass exactly one non-Nothing value")
		} else if arguments[0].typeValue.Kind() == types.NothingKind {
			a.error(codeCallArguments, "str does not accept Nothing", call.Arguments[0].Span(), "pass a value with a textual representation")
		}
		return expressionInfo{typeValue: types.String, nullState: NonNull}
	}
	return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
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

func (a *analyzer) analyzeMember(member *ast.MemberExpr, current *scope, flow flowState) expressionInfo {
	object := a.analyzeExpression(member.Object, current, flow)
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
		return expressionInfo{typeValue: collection.Element, nullState: MaybeNull}
	case types.Pair:
		if !types.Assignable(collection.Key, position.typeValue) {
			a.typeMismatch(index.Index.Span(), collection.Key, position.typeValue, "Pair key")
		}
		return expressionInfo{typeValue: collection.Value, nullState: MaybeNull}
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
	if object.nullState != NonNull {
		a.nullableError("slice", slice.Object, object.nullState)
	}
	for _, bound := range []ast.Expr{slice.Start, slice.End} {
		if bound == nil {
			continue
		}
		info := a.analyzeExpression(bound, current, flow)
		if info.nullState != NonNull || info.typeValue.Kind() != types.IntKind {
			a.typeMismatch(bound.Span(), types.Int, info.typeValue, "slice bound")
		}
	}
	if _, ok := object.typeValue.(types.List); ok || object.typeValue.Kind() == types.StringKind {
		return expressionInfo{typeValue: object.typeValue, nullState: NonNull}
	}
	a.error(codeOperatorType, fmt.Sprintf("type %s is not sliceable", types.Display(object.typeValue)), slice.Object.Span(), "slice a List or String")
	return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
}

func (a *analyzer) analyzeList(list *ast.ListExpr, current *scope, flow flowState) expressionInfo {
	elementType := types.Invalid
	for _, element := range list.Elements {
		info := a.analyzeExpression(element, current, flow)
		if info.nullState == Null {
			continue
		}
		if types.IsInvalid(elementType) {
			elementType = info.typeValue
			continue
		}
		if types.IsNumeric(elementType) && types.IsNumeric(info.typeValue) {
			if elementType.Kind() == types.RealKind || info.typeValue.Kind() == types.RealKind {
				elementType = types.Real
			}
			continue
		}
		if !types.Equal(elementType, info.typeValue) {
			a.typeMismatch(element.Span(), elementType, info.typeValue, "List element")
		}
	}
	return expressionInfo{typeValue: types.List{Element: elementType}, nullState: NonNull}
}

func (a *analyzer) analyzePair(pair *ast.PairExpr, current *scope, flow flowState) expressionInfo {
	keyType, valueType := types.Invalid, types.Invalid
	for _, entry := range pair.Entries {
		key := a.analyzeExpression(entry.Key, current, flow)
		value := a.analyzeExpression(entry.Value, current, flow)
		keyType = a.mergeLiteralType(keyType, key.typeValue, entry.Key)
		valueType = a.mergeLiteralType(valueType, value.typeValue, entry.Value)
	}
	return expressionInfo{typeValue: types.Pair{Key: keyType, Value: valueType}, nullState: NonNull}
}

func (a *analyzer) mergeLiteralType(current, next types.Type, expression ast.Expr) types.Type {
	if types.IsInvalid(current) {
		return next
	}
	if types.IsNumeric(current) && types.IsNumeric(next) {
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
