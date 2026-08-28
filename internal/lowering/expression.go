package lowering

import (
	"fmt"

	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
	"ahdcode/internal/syntax/ast"
)

func (lowerer *moduleLowerer) lowerExprExpected(expression ast.Expr, expected ir.Type) ir.Expr {
	if expression == nil {
		return nil
	}
	result := lowerer.lowerExprWithExpected(expression, expected)
	if result == nil {
		return nil
	}
	actual := result.ExprMeta().Type
	if expected.Kind == ir.RealType && actual.Kind == ir.IntType {
		return &ir.ConvertExpr{ExprBase: ir.ExprBase{Span: expression.Span(), Type: expected, NullState: result.ExprMeta().NullState}, From: actual, Value: result}
	}
	return result
}

func (lowerer *moduleLowerer) lowerExpr(expression ast.Expr) ir.Expr {
	return lowerer.lowerExprWithExpected(expression, ir.Type{})
}

func (lowerer *moduleLowerer) lowerExprWithExpected(expression ast.Expr, expected ir.Type) ir.Expr {
	if expression == nil {
		return nil
	}
	typeValue := lowerType(lowerer.semantic.ExpressionTypes[expression])
	if literal, ok := expression.(*ast.LiteralExpr); ok && literal.Kind == ast.NullLiteral && expected.Kind != "" {
		typeValue = expected
	}
	base := ir.ExprBase{Span: expression.Span(), Type: typeValue, NullState: lowerNull(lowerer.semantic.NullStates[expression])}
	switch value := expression.(type) {
	case *ast.BadExpr:
		lowerer.compilation.error(CodeUnsupportedNode, "cannot lower recovered BadExpr", value.Span())
		return nil
	case *ast.LiteralExpr:
		switch value.Kind {
		case ast.NullLiteral:
			return &ir.NullExpr{ExprBase: base}
		case ast.IntLiteral:
			return &ir.LiteralExpr{ExprBase: base, Kind: ir.IntLiteral, Value: value.Value}
		case ast.RealLiteral:
			return &ir.LiteralExpr{ExprBase: base, Kind: ir.RealLiteral, Value: value.Value}
		case ast.BoolLiteral:
			return &ir.LiteralExpr{ExprBase: base, Kind: ir.BoolLiteral, Value: value.Value}
		}
	case *ast.StringExpr:
		result := &ir.StringExpr{ExprBase: base}
		for _, part := range value.Parts {
			if part.Expression == nil {
				result.Parts = append(result.Parts, ir.StringPart{Literal: part.Text})
				continue
			}
			inner := lowerer.lowerExpr(part.Expression)
			result.Parts = append(result.Parts, ir.StringPart{ToString: &ir.ToStringExpr{ExprBase: ir.ExprBase{Span: part.Span(), Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull}, Value: inner}})
		}
		return result
	case *ast.IdentifierExpr:
		return lowerer.lowerIdentifier(value, base)
	case *ast.GroupExpr:
		return lowerer.lowerExprExpected(value.Expression, chooseExpected(expected, typeValue))
	case *ast.UnaryExpr:
		operand := lowerer.lowerExpr(value.Operand)
		return &ir.UnaryExpr{ExprBase: base, Op: typedUnaryOp(value.Operator, operand), Operand: operand}
	case *ast.BinaryExpr:
		return lowerer.lowerBinary(value, base)
	case *ast.CallExpr:
		return lowerer.lowerCall(value, base)
	case *ast.MemberExpr:
		return lowerer.lowerMember(value, base)
	case *ast.IndexExpr:
		return &ir.IndexExpr{ExprBase: base, Object: lowerer.lowerExpr(value.Object), Index: lowerer.lowerExprExpected(value.Index, ir.Type{Kind: ir.IntType})}
	case *ast.SliceExpr:
		return &ir.SliceExpr{ExprBase: base, Object: lowerer.lowerExpr(value.Object), Start: lowerer.lowerExprExpected(value.Start, ir.Type{Kind: ir.IntType}), End: lowerer.lowerExprExpected(value.End, ir.Type{Kind: ir.IntType})}
	case *ast.ListExpr:
		result := &ir.ListExpr{ExprBase: base}
		if typeValue.Element != nil {
			result.ElementType = *typeValue.Element
		}
		for _, element := range value.Elements {
			result.Elements = append(result.Elements, lowerer.lowerExprExpected(element, result.ElementType))
		}
		return result
	case *ast.PairExpr:
		result := &ir.PairExpr{ExprBase: base}
		if typeValue.Key != nil {
			result.KeyType = *typeValue.Key
		}
		if typeValue.Value != nil {
			result.ValueType = *typeValue.Value
		}
		for _, entry := range value.Entries {
			result.Entries = append(result.Entries, ir.PairEntry{Key: lowerer.lowerExprExpected(entry.Key, result.KeyType), Value: lowerer.lowerExprExpected(entry.Value, result.ValueType)})
		}
		return result
	}
	lowerer.compilation.error(CodeUnsupportedNode, fmt.Sprintf("unsupported expression %T", expression), expression.Span())
	return nil
}

func chooseExpected(expected, fallback ir.Type) ir.Type {
	if expected.Kind != "" {
		return expected
	}
	return fallback
}

func (lowerer *moduleLowerer) lowerIdentifier(identifier *ast.IdentifierExpr, base ir.ExprBase) ir.Expr {
	// The implicit attribute receiver is a Class-callable binding rather than a
	// declared Symbol, so it is resolved before the ordinary Symbol lookup.
	if identifier.Name == "attribute" && lowerer.currentReceiver != "" {
		return &ir.LoadExpr{ExprBase: ir.ExprBase{Span: identifier.Span(), Type: ir.Type{Kind: ir.ClassType, Class: lowerer.currentOwner}, NullState: ir.NonNull}, Symbol: lowerer.currentReceiver}
	}
	symbol := lowerer.semantic.ResolvedSymbols[identifier]
	if symbol == nil {
		lowerer.compilation.error(CodeMissingSemantic, "identifier has no resolved Symbol", identifier.Span())
		return nil
	}
	if symbol.Alias != nil {
		symbol = symbol.Alias
	}
	if base.Type.Kind == ir.FunctionType && base.Type.Signature == nil && symbol.Callable != nil {
		base.Type.Signature = lowerSignature(symbol.Callable.Signature)
	}
	if symbol.Kind == semantic.ClassSymbol && symbol.Class != nil {
		return &ir.ClassRefExpr{ExprBase: base, Class: classID(symbol.Class)}
	}
	if base.Type.Kind == ir.FunctionType {
		callable := lowerer.semantic.SelectedFunctionValues[identifier]
		if callable == nil {
			callable = symbol.Callable
		}
		if callable != nil {
			return &ir.FunctionValueExpr{ExprBase: base, Symbol: lowerer.compilation.registry.symbolID(lowerer.module, symbol), Callable: lowerer.compilation.registry.callableID(lowerer.module, symbol, callable, false)}
		}
	}
	return &ir.LoadExpr{ExprBase: base, Symbol: lowerer.compilation.registry.symbolID(lowerer.module, symbol)}
}

func (lowerer *moduleLowerer) lowerBinary(expression *ast.BinaryExpr, base ir.ExprBase) ir.Expr {
	var left, right ir.Expr
	if expression.Operator == "has" || expression.Operator == "has not" {
		return lowerer.lowerMemberDesignator(expression, base)
	}
	if isNullAST(expression.Left) {
		right = lowerer.lowerExpr(expression.Right)
		left = lowerer.lowerExprExpected(expression.Left, right.ExprMeta().Type)
	} else if isNullAST(expression.Right) {
		left = lowerer.lowerExpr(expression.Left)
		right = lowerer.lowerExprExpected(expression.Right, left.ExprMeta().Type)
	} else {
		left = lowerer.lowerExpr(expression.Left)
		right = lowerer.lowerExpr(expression.Right)
	}
	if left == nil || right == nil {
		lowerer.compilation.error(CodeUnsupportedNode, "binary operand did not lower to a typed expression", expression.Span())
		return nil
	}
	leftType, rightType := left.ExprMeta().Type, right.ExprMeta().Type
	op := typedBinaryOp(expression.Operator, leftType, rightType, base.Type)
	if needsRealOperands(expression.Operator, leftType, rightType, base.Type) {
		left = explicitWiden(left)
		right = explicitWiden(right)
	}
	return &ir.BinaryExpr{ExprBase: base, Op: op, Left: left, Right: right}
}

// lowerMemberDesignator keeps the has/has not right operand as the resolved
// member name decided by semantic analysis instead of a lexical binding load.
func (lowerer *moduleLowerer) lowerMemberDesignator(expression *ast.BinaryExpr, base ir.ExprBase) ir.Expr {
	left := lowerer.lowerExpr(expression.Left)
	if left == nil {
		lowerer.compilation.error(CodeUnsupportedNode, "has operand did not lower to a typed expression", expression.Span())
		return nil
	}
	var right ir.Expr
	if identifier, ok := expression.Right.(*ast.IdentifierExpr); ok {
		right = &ir.LiteralExpr{
			ExprBase: ir.ExprBase{Span: identifier.Span(), Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull},
			Kind:     ir.StringLiteral, Value: identifier.Name,
		}
	} else {
		right = lowerer.lowerExpr(expression.Right)
	}
	if right == nil {
		lowerer.compilation.error(CodeUnsupportedNode, "has member designator did not lower", expression.Span())
		return nil
	}
	op := ir.BinaryOp("Has")
	if expression.Operator == "has not" {
		op = "HasNot"
	}
	return &ir.BinaryExpr{ExprBase: base, Op: op, Left: left, Right: right}
}

// lowerSuperMember keeps SuperClass.member bound to the current instance while
// naming the parent implementation directly.
func (lowerer *moduleLowerer) lowerSuperMember(member *ast.MemberExpr, resolved *semantic.Symbol, base ir.ExprBase) ir.Expr {
	if lowerer.currentReceiver == "" || lowerer.currentOwner == "" {
		lowerer.compilation.error(CodeMissingSemantic, "SuperClass is valid only inside a Class callable", member.Span())
		return nil
	}
	callable := resolved.Callable
	if selected := lowerer.semantic.SelectedFunctionValues[member]; selected != nil {
		callable = selected
	}
	object := &ir.LoadExpr{
		ExprBase: ir.ExprBase{Span: member.Span(), Type: ir.Type{Kind: ir.ClassType, Class: lowerer.currentOwner}, NullState: ir.NonNull},
		Symbol:   lowerer.currentReceiver,
	}
	return &ir.MemberExpr{
		ExprBase: base, Kind: ir.MethodMember, Object: object, Direct: true,
		Callable: lowerer.compilation.registry.callableID(lowerer.module, resolved, callable, false),
	}
}

func isNullAST(expression ast.Expr) bool {
	literal, ok := expression.(*ast.LiteralExpr)
	return ok && literal.Kind == ast.NullLiteral
}

func explicitWiden(expression ir.Expr) ir.Expr {
	if expression == nil || expression.ExprMeta().Type.Kind != ir.IntType {
		return expression
	}
	base := expression.ExprMeta()
	return &ir.ConvertExpr{ExprBase: ir.ExprBase{Span: base.Span, Type: ir.Type{Kind: ir.RealType}, NullState: base.NullState}, From: base.Type, Value: expression}
}

func needsRealOperands(operator string, left, right, result ir.Type) bool {
	if left.Kind != ir.IntType && right.Kind != ir.IntType {
		return false
	}
	if operator == "/" {
		return true
	}
	if operator == "^" && result.Kind == ir.RealType {
		return true
	}
	if result.Kind == ir.RealType && (operator == "+" || operator == "-" || operator == "*") {
		return true
	}
	if result.Kind == ir.BoolType && isComparison(operator) && (left.Kind == ir.RealType || right.Kind == ir.RealType) {
		return true
	}
	return false
}

// isComparison excludes same on purpose: same is strict type plus value, so
// widening its operands would change the result.
func isComparison(operator string) bool {
	switch operator {
	case "<", "<=", ">", ">=", "==", "!=":
		return true
	}
	return false
}

func typedUnaryOp(operator string, operand ir.Expr) ir.UnaryOp {
	kind := ir.InvalidType
	if operand != nil {
		kind = operand.ExprMeta().Type.Kind
	}
	switch operator {
	case "+":
		if kind == ir.IntType {
			return "IntPositive"
		}
		return "RealPositive"
	case "-":
		if kind == ir.IntType {
			return "CheckedIntNegate"
		}
		return "RealNegate"
	case "not":
		return "BoolNot"
	}
	return ir.UnaryOp("InvalidUnary(" + operator + ")")
}

func typedBinaryOp(operator string, left, right, result ir.Type) ir.BinaryOp {
	numeric := "Int"
	if left.Kind == ir.RealType || right.Kind == ir.RealType || result.Kind == ir.RealType {
		numeric = "Real"
	}
	switch operator {
	case "+":
		if result.Kind == ir.StringType {
			return "StringConcat"
		}
		if result.Kind == ir.ListType {
			return "ListConcat"
		}
		if numeric == "Int" {
			return "CheckedIntAdd"
		}
		return "RealAdd"
	case "-":
		if numeric == "Int" {
			return "CheckedIntSubtract"
		}
		return "RealSubtract"
	case "*":
		if left.Kind == ir.StringType {
			return "StringRepeat"
		}
		if numeric == "Int" {
			return "CheckedIntMultiply"
		}
		return "RealMultiply"
	case "/":
		return "RealDivide"
	case "%":
		return "IntModulo"
	case "^":
		if result.Kind == ir.IntType {
			return "CheckedIntPower"
		}
		return "RealPower"
	case "and":
		return "BoolAndShortCircuit"
	case "or":
		return "BoolOrShortCircuit"
	case "==":
		return ir.BinaryOp(comparisonPrefix(left, right) + "Equal")
	case "!=":
		return ir.BinaryOp(comparisonPrefix(left, right) + "NotEqual")
	case "same":
		return "IdentitySame"
	case "<":
		return ir.BinaryOp(numeric + "Less")
	case "<=":
		return ir.BinaryOp(numeric + "LessEqual")
	case ">":
		return ir.BinaryOp(numeric + "Greater")
	case ">=":
		return ir.BinaryOp(numeric + "GreaterEqual")
	case "is":
		return "Is"
	case "is not":
		return "IsNot"
	case "in":
		return "Contains"
	case "not in":
		return "NotContains"
	case "has":
		return "Has"
	case "has not":
		return "HasNot"
	}
	return ir.BinaryOp("Typed(" + operator + ")")
}

func comparisonPrefix(left, right ir.Type) string {
	if left.Kind == ir.RealType || right.Kind == ir.RealType {
		return "Real"
	}
	switch left.Kind {
	case ir.IntType:
		return "Int"
	case ir.StringType:
		return "String"
	case ir.BoolType:
		return "Bool"
	case ir.ClassType:
		return "Class"
	case ir.ListType:
		return "List"
	case ir.PairType:
		return "Pair"
	default:
		return "Value"
	}
}

// lowerCollectionCall lowers a built-in List or Pair mutation. The receiver is
// carried as the callee so the operation keeps one evaluated target.
func (lowerer *moduleLowerer) lowerCollectionCall(call *ast.CallExpr, operation semantic.CollectionOperation, base ir.ExprBase) ir.Expr {
	member, ok := call.Callee.(*ast.MemberExpr)
	if !ok {
		lowerer.compilation.error(CodeMissingSemantic, "collection mutation has no receiver", call.Span())
		return nil
	}
	receiver := lowerer.lowerExpr(member.Object)
	if receiver == nil {
		return nil
	}
	argumentType := ir.Type{Kind: ir.IntType}
	receiverType := receiver.ExprMeta().Type
	switch operation {
	case semantic.ListAdd:
		if receiverType.Element != nil {
			argumentType = *receiverType.Element
		}
	case semantic.PairEject:
		if receiverType.Key != nil {
			argumentType = *receiverType.Key
		}
	}
	result := &ir.CallExpr{
		ExprBase: base, Callable: ir.CallableID("builtin:core::" + string(operation)),
		Callee: receiver, ReturnNull: ir.NonNull,
	}
	for index, argument := range call.Arguments {
		result.Arguments = append(result.Arguments, ir.Argument{
			ParameterIndex: index, Value: lowerer.lowerExprExpected(argument.Value, argumentType),
		})
	}
	return result
}

func (lowerer *moduleLowerer) lowerCall(call *ast.CallExpr, base ir.ExprBase) ir.Expr {
	if operation, known := lowerer.semantic.CollectionCalls[call]; known {
		return lowerer.lowerCollectionCall(call, operation, base)
	}
	symbol := lowerer.semantic.ResolvedSymbols[call.Callee]
	if symbol != nil && symbol.Alias != nil {
		symbol = symbol.Alias
	}
	if symbol != nil && symbol.Builtin && symbol.Name == "str" && len(call.Arguments) == 1 {
		// str(null) is normative and has no declared type context to lower, so
		// the specified canonical text is folded here rather than guessed by a
		// backend.
		if isNullAST(call.Arguments[0].Value) {
			return &ir.LiteralExpr{ExprBase: base, Kind: ir.StringLiteral, Value: "null"}
		}
		return &ir.ToStringExpr{ExprBase: base, Value: lowerer.lowerExpr(call.Arguments[0].Value)}
	}
	if symbol != nil && symbol.Builtin && len(call.Arguments) == 1 && (symbol.Name == "int" || symbol.Name == "real") {
		argument := lowerer.lowerExpr(call.Arguments[0].Value)
		if argument == nil {
			return nil
		}
		return &ir.ConvertExpr{ExprBase: base, From: argument.ExprMeta().Type, Value: argument}
	}
	selected := lowerer.semantic.SelectedCallables[call]
	if symbol != nil && symbol.Kind == semantic.ClassSymbol && symbol.Class != nil {
		if selected == nil {
			selected = symbol.Constructor
		}
		id := lowerer.compilation.registry.callableID(lowerer.module, symbol, selected, true)
		return &ir.ConstructExpr{ExprBase: base, Class: classID(symbol.Class), Constructor: id, Arguments: lowerer.lowerArguments(call, selected)}
	}
	callableID := lowerer.compilation.registry.callableID(lowerer.module, symbol, selected, false)
	if callableID == "" && symbol != nil && symbol.Builtin {
		callableID = ir.CallableID("builtin:core::" + symbol.Name)
	}
	var callee ir.Expr
	if symbol == nil || (symbol.Kind != semantic.FunctionSymbol && symbol.Kind != semantic.BuiltinSymbol) {
		callee = lowerer.lowerExpr(call.Callee)
	} else if _, method := call.Callee.(*ast.MemberExpr); method && symbol.OwnerClass != nil {
		callee = lowerer.lowerExpr(call.Callee)
	}
	return &ir.CallExpr{ExprBase: base, Callable: callableID, Callee: callee, Arguments: lowerer.lowerArguments(call, selected), ReturnNull: lowerNullState(selected)}
}

func lowerNullState(callable *semantic.Callable) ir.NullState {
	if callable == nil {
		return ir.MaybeNull
	}
	return lowerNull(callable.ReturnNull)
}

func (lowerer *moduleLowerer) lowerArguments(call *ast.CallExpr, callable *semantic.Callable) []ir.Argument {
	if callable == nil || callable.Signature == nil {
		result := make([]ir.Argument, len(call.Arguments))
		for index, argument := range call.Arguments {
			result[index] = ir.Argument{ParameterIndex: index, ParameterName: argument.Name, Value: lowerer.lowerExpr(argument.Value)}
		}
		return result
	}
	parameters := callable.Signature.Parameters
	result := make([]ir.Argument, len(parameters))
	assigned := make([]bool, len(parameters))
	for sourceIndex, argument := range call.Arguments {
		parameterIndex := sourceIndex
		if argument.Name != "" {
			parameterIndex = -1
			for index, parameter := range parameters {
				if parameter.Name == argument.Name {
					parameterIndex = index
					break
				}
			}
		}
		if parameterIndex < 0 || parameterIndex >= len(parameters) {
			continue
		}
		result[parameterIndex] = ir.Argument{ParameterIndex: parameterIndex, ParameterName: parameters[parameterIndex].Name, Value: lowerer.lowerExprExpected(argument.Value, lowerType(parameters[parameterIndex].Type))}
		assigned[parameterIndex] = true
	}
	for index, parameter := range parameters {
		if !assigned[index] {
			result[index] = ir.Argument{ParameterIndex: index, ParameterName: parameter.Name, UsesDefault: true}
		}
	}
	return result
}

func (lowerer *moduleLowerer) lowerMember(member *ast.MemberExpr, base ir.ExprBase) ir.Expr {
	resolved := lowerer.semantic.ResolvedSymbols[member]
	if resolved == nil {
		lowerer.compilation.error(CodeMissingSemantic, "member has no resolved Symbol", member.Span())
		return nil
	}
	if lowerer.semantic.SuperCalls[member] {
		return lowerer.lowerSuperMember(member, resolved, base)
	}
	if objectSymbol := lowerer.semantic.ResolvedSymbols[member.Object]; objectSymbol != nil && objectSymbol.Kind == semantic.NamespaceSymbol {
		if resolved.Kind == semantic.ClassSymbol && resolved.Class != nil {
			return &ir.ClassRefExpr{ExprBase: base, Class: classID(resolved.Class)}
		}
		if base.Type.Kind == ir.FunctionType {
			callable := resolved.Callable
			return &ir.FunctionValueExpr{ExprBase: base, Symbol: lowerer.compilation.registry.symbolID(lowerer.module, resolved), Callable: lowerer.compilation.registry.callableID(lowerer.module, resolved, callable, false)}
		}
		return &ir.LoadExpr{ExprBase: base, Symbol: lowerer.compilation.registry.symbolID(lowerer.module, resolved)}
	}
	object := lowerer.lowerExpr(member.Object)
	if resolved.Kind == semantic.FunctionSymbol {
		callable := resolved.Callable
		if selected := lowerer.semantic.SelectedFunctionValues[member]; selected != nil {
			callable = selected
		}
		return &ir.MemberExpr{ExprBase: base, Kind: ir.MethodMember, Object: object, Callable: lowerer.compilation.registry.callableID(lowerer.module, resolved, callable, false)}
	}
	return &ir.MemberExpr{ExprBase: base, Kind: ir.FieldMember, Object: object, Field: fieldID(resolved)}
}
