package evaluator

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"ahdcode/internal/ir"
)

func (session *Session) eval(expression ir.Expr, current *frame) any {
	if expression == nil {
		return Nothing
	}
	switch value := expression.(type) {
	case *ir.LiteralExpr:
		switch value.Kind {
		case ir.IntLiteral:
			parsed, err := strconv.ParseInt(value.Value, 10, 64)
			if err != nil {
				session.raise("OverflowError", "Int literal is outside signed 64-bit range")
			}
			return parsed
		case ir.RealLiteral:
			parsed, _ := strconv.ParseFloat(value.Value, 64)
			return parsed
		case ir.BoolLiteral:
			return value.Value == "true"
		case ir.StringLiteral:
			return value.Value
		}
	case *ir.NullExpr:
		return nil
	case *ir.LoadExpr:
		return session.cell(current, value.Symbol).Value
	case *ir.ClassRefExpr:
		return ClassValue{Class: value.Class}
	case *ir.FunctionValueExpr:
		if cell := session.optionalCell(current, value.Symbol); cell != nil && cell.Value != nil {
			return cell.Value
		}
		return &FunctionValue{Callable: value.Callable}
	case *ir.UnaryExpr:
		operand := session.eval(value.Operand, current)
		switch value.Op {
		case "IntPositive", "RealPositive":
			return operand
		case "CheckedIntNegate":
			item := operand.(int64)
			if item == math.MinInt64 {
				session.raise("OverflowError", "Int arithmetic overflowed signed 64-bit range")
			}
			return -item
		case "RealNegate":
			return -operand.(float64)
		case "BoolNot":
			return !session.boolean(operand)
		}
	case *ir.BinaryExpr:
		left := session.eval(value.Left, current)
		if value.Op == "BoolAndShortCircuit" && !session.boolean(left) {
			return false
		}
		if value.Op == "BoolOrShortCircuit" && session.boolean(left) {
			return true
		}
		return session.binary(value.Op, left, session.eval(value.Right, current))
	case *ir.ConvertExpr:
		return session.convert(value.From, value.ExprMeta().Type, session.eval(value.Value, current))
	case *ir.CallExpr:
		return session.evalCall(value, current)
	case *ir.ConstructExpr:
		arguments := session.evalArguments(value.Arguments, current)
		return session.construct(value.Class, value.Constructor, arguments)
	case *ir.MemberExpr:
		object := session.eval(value.Object, current)
		if value.Kind == ir.FieldMember {
			instance := session.requireInstance(object)
			return instance.Fields[value.Field]
		}
		return &FunctionValue{Callable: value.Callable, Receiver: session.requireInstance(object), Direct: value.Direct}
	case *ir.IndexExpr:
		return session.index(session.eval(value.Object, current), session.eval(value.Index, current))
	case *ir.SliceExpr:
		var start, end int64
		hasStart, hasEnd := value.Start != nil, value.End != nil
		if hasStart {
			start = session.eval(value.Start, current).(int64)
		}
		if hasEnd {
			end = session.eval(value.End, current).(int64)
		}
		return session.slice(session.eval(value.Object, current), start, hasStart, end, hasEnd)
	case *ir.ListExpr:
		list := &List{Items: make([]any, len(value.Elements))}
		for index, element := range value.Elements {
			list.Items[index] = session.eval(element, current)
		}
		return list
	case *ir.PairExpr:
		pair := &Pair{Values: make(map[any]any)}
		for _, entry := range value.Entries {
			pairSet(pair, session.eval(entry.Key, current), session.eval(entry.Value, current))
		}
		return pair
	case *ir.StringExpr:
		var builder strings.Builder
		for _, part := range value.Parts {
			builder.WriteString(part.Literal)
			if part.ToString != nil {
				builder.WriteString(session.render(session.eval(part.ToString, current), false, make(map[visit]bool)))
			}
		}
		return builder.String()
	case *ir.ToStringExpr:
		return session.textOf(value.Value, current)
	case *ir.IdentityExpr:
		return session.identityOf(session.identityValue(session.eval(value.Value, current)))
	case *ir.TypeNameExpr:
		return session.typeNameOf(value, current)
	}
	session.raise("Error", fmt.Sprintf("unsupported IR expression %T", expression))
	return nil
}

// textOf renders one expression as the text write/str/interpolation use. A
// Class value whose statically declared type resolves CStr dispatches to it
// through the ordinary dynamic-dispatch path (so a more-derived override
// still runs); every other value uses the shared canonical renderer, matching
// the native backend's equivalent choke point in generator.text.
func (session *Session) textOf(expression ir.Expr, current *frame) string {
	if expression.ExprMeta().Type.Kind == ir.ClassType {
		if methodID, found := session.findProtocolMethod(expression.ExprMeta().Type.Class, "CStr"); found {
			instance, ok := session.eval(expression, current).(*Instance)
			if !ok || instance == nil {
				return "null"
			}
			if text, ok := session.invoke(&FunctionValue{Callable: methodID, Receiver: instance}, nil).(string); ok {
				return text
			}
		}
	}
	return session.render(session.eval(expression, current), false, make(map[visit]bool))
}

// identityValue requires the value id() accepts: a non-null List, Pair, or
// Class instance. Semantic analysis already rejects every other case at
// compile time; this is defense in depth for the runtime value actually
// reached.
func (session *Session) identityValue(value any) any {
	switch item := value.(type) {
	case *List:
		if item == nil {
			session.raise("NullError", "id requires a non-null List")
		}
		return item
	case *Pair:
		if item == nil {
			session.raise("NullError", "id requires a non-null Pair")
		}
		return item
	case *Instance:
		if item == nil {
			session.raise("NullError", "id requires a non-null Class instance")
		}
		return item
	}
	session.raise("Error", "id requires a List, Pair, or Class instance")
	return nil
}

// typeNameOf implements the type() Fundamental. A Class value reports its
// most-derived runtime Class name; every other value reports the canonical
// static name computed once during lowering, unless it is actually null right
// now, which always reports "Null" regardless of the declared type.
func (session *Session) typeNameOf(node *ir.TypeNameExpr, current *frame) string {
	result := session.eval(node.Value, current)
	if node.IsClass {
		instance, ok := result.(*Instance)
		if !ok || instance == nil {
			return "Null"
		}
		return className(instance.Class)
	}
	if result == nil {
		return "Null"
	}
	return node.StaticName
}

func (session *Session) optionalCell(current *frame, symbol ir.SymbolID) *Cell {
	for scope := current; scope != nil; scope = scope.parent {
		if cell := scope.locals[symbol]; cell != nil {
			return cell
		}
	}
	return session.globals[symbol]
}

func (session *Session) boolean(value any) bool {
	result, ok := value.(bool)
	if !ok {
		session.raise("NullError", "Bool value is null")
	}
	return result
}

func (session *Session) convert(from, to ir.Type, value any) any {
	if value == nil {
		session.raise("NullError", "conversion requires a NonNull value")
	}
	switch {
	case from.Kind == ir.IntType && to.Kind == ir.RealType:
		return float64(value.(int64))
	case from.Kind == ir.RealType && to.Kind == ir.IntType:
		number := value.(float64)
		if math.IsNaN(number) {
			session.raise("DomainError", "cannot convert a non-number Real to Int")
		}
		if math.IsInf(number, 0) || number < -9223372036854775808.0 || number >= 9223372036854775808.0 {
			session.raise("OverflowError", "Real value is outside signed 64-bit Int range")
		}
		return int64(math.Trunc(number))
	case from.Kind == ir.StringType && to.Kind == ir.IntType:
		return session.parseInt(value.(string))
	case from.Kind == ir.StringType && to.Kind == ir.RealType:
		return session.parseReal(value.(string))
	}
	session.raise("Error", "unsupported conversion "+from.String()+" -> "+to.String())
	return nil
}

func (session *Session) binary(operation ir.BinaryOp, left, right any) any {
	switch operation {
	case "CheckedIntAdd":
		return session.intAdd(left.(int64), right.(int64))
	case "CheckedIntSubtract":
		return session.intSubtract(left.(int64), right.(int64))
	case "CheckedIntMultiply":
		return session.intMultiply(left.(int64), right.(int64))
	case "CheckedIntPower":
		return session.intPower(left.(int64), right.(int64))
	case "IntModulo":
		if right.(int64) == 0 {
			session.raise("DivisionByZeroError", "Int modulo by zero")
		}
		return left.(int64) % right.(int64)
	case "RealAdd":
		return session.realCheck(left.(float64)+right.(float64), "addition")
	case "RealSubtract":
		return session.realCheck(left.(float64)-right.(float64), "subtraction")
	case "RealMultiply":
		return session.realCheck(left.(float64)*right.(float64), "multiplication")
	case "RealDivide":
		if right.(float64) == 0 {
			session.raise("DivisionByZeroError", "division by zero")
		}
		return session.realCheck(left.(float64)/right.(float64), "division")
	case "RealPower":
		return session.realCheck(math.Pow(left.(float64), right.(float64)), "power")
	case "StringConcat":
		return left.(string) + right.(string)
	case "StringRepeat":
		count := right.(int64)
		if count < 0 || count > math.MaxInt32 {
			session.raise("ValueError", "String repeat count is invalid")
		}
		return strings.Repeat(left.(string), int(count))
	case "ListConcat":
		first, second := session.requireList(left), session.requireList(right)
		return &List{Items: append(append([]any(nil), first.Items...), second.Items...)}
	case "BoolAndShortCircuit":
		return session.boolean(left) && session.boolean(right)
	case "BoolOrShortCircuit":
		return session.boolean(left) || session.boolean(right)
	case "IntLess":
		return left.(int64) < right.(int64)
	case "IntLessEqual":
		return left.(int64) <= right.(int64)
	case "IntGreater":
		return left.(int64) > right.(int64)
	case "IntGreaterEqual":
		return left.(int64) >= right.(int64)
	case "RealLess":
		return left.(float64) < right.(float64)
	case "RealLessEqual":
		return left.(float64) <= right.(float64)
	case "RealGreater":
		return left.(float64) > right.(float64)
	case "RealGreaterEqual":
		return left.(float64) >= right.(float64)
	case "IdentitySame":
		return session.same(left, right)
	case "Contains", "NotContains":
		result := session.contains(right, left)
		if operation == "NotContains" {
			result = !result
		}
		return result
	case "Is", "IsNot":
		instance, ok := left.(*Instance)
		class, classOK := right.(ClassValue)
		result := ok && instance != nil && classOK && session.isClass(instance.Class, class.Class)
		if operation == "IsNot" {
			result = !result
		}
		return result
	case "Has", "HasNot":
		result := session.hasMember(left, right.(string))
		if operation == "HasNot" {
			result = !result
		}
		return result
	}
	if strings.HasSuffix(string(operation), "NotEqual") {
		return !session.equal(left, right)
	}
	if strings.HasSuffix(string(operation), "Equal") {
		return session.equal(left, right)
	}
	session.raise("Error", "unsupported binary operation "+string(operation))
	return nil
}

func (session *Session) intAdd(left, right int64) int64 {
	result := left + right
	if right > 0 && result < left || right < 0 && result > left {
		session.raise("OverflowError", "Int arithmetic overflowed signed 64-bit range")
	}
	return result
}

func (session *Session) intSubtract(left, right int64) int64 {
	if right == math.MinInt64 {
		if left >= 0 {
			session.raise("OverflowError", "Int arithmetic overflowed signed 64-bit range")
		}
		return left - right
	}
	return session.intAdd(left, -right)
}

func (session *Session) intMultiply(left, right int64) int64 {
	if left == 0 || right == 0 {
		return 0
	}
	if left == math.MinInt64 && right == -1 || right == math.MinInt64 && left == -1 {
		session.raise("OverflowError", "Int arithmetic overflowed signed 64-bit range")
	}
	result := left * right
	if result/right != left {
		session.raise("OverflowError", "Int arithmetic overflowed signed 64-bit range")
	}
	return result
}

func (session *Session) intPower(base, exponent int64) int64 {
	if exponent < 0 {
		session.raise("DomainError", "Int power requires a non-negative exponent")
	}
	result := int64(1)
	for exponent > 0 {
		if exponent&1 != 0 {
			result = session.intMultiply(result, base)
		}
		exponent >>= 1
		if exponent > 0 {
			base = session.intMultiply(base, base)
		}
	}
	return result
}

func (session *Session) realCheck(value float64, operation string) float64 {
	if math.IsInf(value, 0) {
		session.raise("OverflowError", "Real "+operation+" produced a non-finite result")
	}
	if math.IsNaN(value) {
		session.raise("DomainError", "Real "+operation+" is not defined for these operands")
	}
	return value
}

func (session *Session) index(object, index any) any {
	switch value := object.(type) {
	case *List:
		return value.Items[resolveIndex(session, index.(int64), len(value.Items))]
	case *Pair:
		result, exists := session.requirePair(value).Values[index]
		if !exists {
			session.raise("KeyError", "Pair key was not found")
		}
		return result
	case string:
		runes := []rune(value)
		return string(runes[resolveIndex(session, index.(int64), len(runes))])
	}
	session.raise("NullError", "indexed value is null")
	return nil
}

func (session *Session) slice(object any, start int64, hasStart bool, end int64, hasEnd bool) any {
	switch value := object.(type) {
	case *List:
		low, high := resolveRange(start, hasStart, end, hasEnd, len(value.Items))
		return &List{Items: append([]any(nil), value.Items[low:high]...)}
	case string:
		runes := []rune(value)
		low, high := resolveRange(start, hasStart, end, hasEnd, len(runes))
		return string(runes[low:high])
	}
	session.raise("NullError", "sliced value is null")
	return nil
}

func (session *Session) contains(container, needle any) bool {
	switch value := container.(type) {
	case string:
		return strings.Contains(value, needle.(string))
	case *List:
		for _, item := range session.requireList(value).Items {
			if session.equal(item, needle) {
				return true
			}
		}
	case *Pair:
		_, found := session.requirePair(value).Values[needle]
		return found
	}
	return false
}

func (session *Session) hasMember(value any, name string) bool {
	instance, ok := value.(*Instance)
	if !ok || instance == nil {
		return false
	}
	for current := instance.Class; current != ""; {
		class := session.classes[current]
		if class == nil {
			break
		}
		for _, field := range class.Fields {
			if field.Name == name {
				return true
			}
		}
		for _, method := range class.Methods {
			if function := session.functions[method]; function != nil && function.Name == name {
				return true
			}
		}
		for _, operation := range class.Operations {
			if operation == name {
				return true
			}
		}
		current = class.Parent
	}
	return false
}
