package golang

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"ahdcode/internal/ir"
	"ahdcode/internal/source"
)

// value renders an expression coerced into the requested storage
// representation.
func (generator *generator) value(expression ir.Expr, target ir.Type, nullable bool) string {
	if expression == nil {
		generator.fail(CodeGenerationFailure, "missing expression value", source.Span{}, "the IR node is incomplete")
		return "nil"
	}
	return generator.coerce(generator.expr(expression), expression.ExprMeta(), target, nullable)
}

// coerce bridges the boxed and unboxed scalar representations. Reference types
// share one representation, so they never need a conversion.
func (generator *generator) coerce(code string, meta ir.ExprBase, target ir.Type, nullable bool) string {
	if !isScalar(target) {
		return code
	}
	from := meta.NullState != ir.NonNull
	switch {
	case from == nullable:
		return code
	case from:
		return "AhdNonNull(" + code + ")"
	default:
		return "AhdBox(" + code + ")"
	}
}

// naturalNullable reports whether an expression renders in the boxed
// representation.
func naturalNullable(expression ir.Expr) bool {
	return expression.ExprMeta().NullState != ir.NonNull
}

func (generator *generator) expr(expression ir.Expr) string {
	if expression == nil {
		generator.fail(CodeGenerationFailure, "nil expression reached code generation", source.Span{}, "the IR node is incomplete")
		return "nil"
	}
	meta := expression.ExprMeta()
	switch value := expression.(type) {
	case *ir.LiteralExpr:
		return generator.literal(value)
	case *ir.NullExpr:
		rendered := generator.goType(meta.Type, true)
		if rendered == "" {
			return generator.unsupported("a null value of this type", meta.Span)
		}
		return "(" + rendered + ")(nil)"
	case *ir.LoadExpr:
		current, known := generator.slots[value.Symbol]
		if !known {
			generator.fail(CodeGenerationFailure, "unknown SymbolID "+string(value.Symbol), meta.Span, "the IR references a symbol with no declaration")
			return "nil"
		}
		return generator.coerce(current.name, ir.ExprBase{Type: current.typeInfo, NullState: nullState(current.nullable)}, meta.Type, naturalNullable(expression))
	case *ir.FunctionValueExpr:
		// A Function-typed binding resolves to its storage slot; a declared
		// callable resolves to the generated Go function.
		if current, known := generator.slots[value.Symbol]; known {
			return current.name
		}
		function := generator.functions[value.Callable]
		if function == nil {
			return generator.unsupported("a Function value with no generated callable", meta.Span)
		}
		return generator.callableName(function)
	case *ir.ConvertExpr:
		if value.From.Kind == ir.IntType && meta.Type.Kind == ir.RealType {
			return "AhdIntToReal(" + generator.value(value.Value, ir.Type{Kind: ir.IntType}, false) + ")"
		}
		return generator.unsupported("conversion "+value.From.String()+" -> "+meta.Type.String(), meta.Span)
	case *ir.UnaryExpr:
		return generator.unary(value)
	case *ir.BinaryExpr:
		return generator.binary(value)
	case *ir.StringExpr:
		return generator.stringParts(value)
	case *ir.ToStringExpr:
		return generator.toString(value)
	case *ir.ListExpr:
		return generator.listLiteral(value)
	case *ir.PairExpr:
		return generator.pairLiteral(value)
	case *ir.IndexExpr:
		return generator.index(value)
	case *ir.SliceExpr:
		return generator.slice(value)
	case *ir.MemberExpr:
		return generator.member(value)
	case *ir.CallExpr:
		return generator.call(value)
	case *ir.ConstructExpr:
		return generator.construct(value)
	case *ir.ClassRefExpr:
		return generator.unsupported("a Class reference used as a value", meta.Span)
	default:
		return generator.unsupported(fmt.Sprintf("IR expression %T", expression), meta.Span)
	}
}

func nullState(nullable bool) ir.NullState {
	if nullable {
		return ir.MaybeNull
	}
	return ir.NonNull
}

// ---------------------------------------------------------------------------
// Literals
// ---------------------------------------------------------------------------

func (generator *generator) literal(value *ir.LiteralExpr) string {
	meta := value.ExprMeta()
	switch value.Kind {
	case ir.IntLiteral:
		parsed, err := strconv.ParseInt(value.Value, 10, 64)
		if err != nil {
			generator.fail(CodeGenerationFailure, "Int literal "+value.Value+" is outside signed 64-bit range", meta.Span, "the frontend should reject out-of-range Int constants")
			return "int64(0)"
		}
		return "int64(" + strconv.FormatInt(parsed, 10) + ")"
	case ir.RealLiteral:
		parsed, err := strconv.ParseFloat(value.Value, 64)
		if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
			generator.fail(CodeGenerationFailure, "Real literal "+value.Value+" has no finite float64 value", meta.Span, "the frontend should reject non-finite Real constants")
			return "float64(0)"
		}
		return "float64(" + goRealConstant(parsed) + ")"
	case ir.BoolLiteral:
		if value.Value == "true" {
			return "true"
		}
		return "false"
	case ir.StringLiteral:
		return strconv.Quote(value.Value)
	default:
		return generator.unsupported("literal kind "+string(value.Kind), meta.Span)
	}
}

// goRealConstant renders a Real literal as a round-trip-safe Go constant.
func goRealConstant(value float64) string {
	text := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(text, ".e") {
		text += ".0"
	}
	return text
}

// ---------------------------------------------------------------------------
// Operators
// ---------------------------------------------------------------------------

func (generator *generator) unary(value *ir.UnaryExpr) string {
	meta := value.ExprMeta()
	// A negated Int literal is folded so that the signed 64-bit minimum stays
	// expressible as a Go constant.
	if value.Op == "CheckedIntNegate" {
		if literal, ok := value.Operand.(*ir.LiteralExpr); ok && literal.Kind == ir.IntLiteral {
			if magnitude, err := strconv.ParseUint(literal.Value, 10, 64); err == nil && magnitude <= 1<<63 {
				return "int64(-" + strconv.FormatUint(magnitude, 10) + ")"
			}
		}
	}
	operand := func(kind ir.TypeKind) string {
		return generator.value(value.Operand, ir.Type{Kind: kind}, false)
	}
	switch value.Op {
	case "IntPositive":
		return operand(ir.IntType)
	case "RealPositive":
		return operand(ir.RealType)
	case "CheckedIntNegate":
		return "AhdIntNegate(" + operand(ir.IntType) + ")"
	case "RealNegate":
		return "AhdRealNegate(" + operand(ir.RealType) + ")"
	case "BoolNot":
		return "(!" + operand(ir.BoolType) + ")"
	default:
		return generator.unsupported("unary operation "+string(value.Op), meta.Span)
	}
}

func (generator *generator) binary(value *ir.BinaryExpr) string {
	meta := value.ExprMeta()
	scalar := func(kind ir.TypeKind) (string, string) {
		return generator.value(value.Left, ir.Type{Kind: kind}, false), generator.value(value.Right, ir.Type{Kind: kind}, false)
	}
	switch value.Op {
	case "CheckedIntAdd", "CheckedIntSubtract", "CheckedIntMultiply", "CheckedIntPower", "IntModulo":
		left, right := scalar(ir.IntType)
		return intOperation(value.Op) + "(" + left + ", " + right + ")"
	case "RealAdd", "RealSubtract", "RealMultiply", "RealDivide", "RealPower":
		left, right := scalar(ir.RealType)
		return "Ahd" + string(value.Op) + "(" + left + ", " + right + ")"
	case "StringConcat":
		left, right := scalar(ir.StringType)
		return "(" + left + " + " + right + ")"
	case "StringRepeat":
		return "AhdStringRepeat(" + generator.value(value.Left, ir.Type{Kind: ir.StringType}, false) + ", " + generator.value(value.Right, ir.Type{Kind: ir.IntType}, false) + ")"
	case "ListConcat":
		return "AhdListConcat(" + generator.expr(value.Left) + ", " + generator.expr(value.Right) + ")"
	case "BoolAndShortCircuit":
		left, right := scalar(ir.BoolType)
		return "(" + left + " && " + right + ")"
	case "BoolOrShortCircuit":
		left, right := scalar(ir.BoolType)
		return "(" + left + " || " + right + ")"
	case "IntLess", "IntLessEqual", "IntGreater", "IntGreaterEqual":
		left, right := scalar(ir.IntType)
		return "(" + left + " " + comparisonSymbol(string(value.Op), "Int") + " " + right + ")"
	case "RealLess", "RealLessEqual", "RealGreater", "RealGreaterEqual":
		left, right := scalar(ir.RealType)
		return "(" + left + " " + comparisonSymbol(string(value.Op), "Real") + " " + right + ")"
	case "IdentitySame":
		return generator.same(value)
	case "Contains":
		return generator.membership(value, false)
	case "NotContains":
		return generator.membership(value, true)
	case "Is":
		return generator.classMembership(value, false)
	case "IsNot":
		return generator.classMembership(value, true)
	case "Has":
		return generator.memberExistence(value, false)
	case "HasNot":
		return generator.memberExistence(value, true)
	}
	if strings.HasSuffix(string(value.Op), "Equal") {
		return generator.equality(value, strings.HasSuffix(string(value.Op), "NotEqual"))
	}
	return generator.unsupported("binary operation "+string(value.Op), meta.Span)
}

func intOperation(op ir.BinaryOp) string {
	switch op {
	case "CheckedIntAdd":
		return "AhdIntAdd"
	case "CheckedIntSubtract":
		return "AhdIntSubtract"
	case "CheckedIntMultiply":
		return "AhdIntMultiply"
	case "CheckedIntPower":
		return "AhdIntPower"
	default:
		return "AhdIntModulo"
	}
}

func comparisonSymbol(op, prefix string) string {
	switch strings.TrimPrefix(op, prefix) {
	case "Less":
		return "<"
	case "LessEqual":
		return "<="
	case "Greater":
		return ">"
	default:
		return ">="
	}
}

// equality renders == and != for every representable operand type, including
// the null comparisons that refine null state in AhdCode source.
func (generator *generator) equality(value *ir.BinaryExpr, negated bool) string {
	meta := value.ExprMeta()
	_, leftNull := value.Left.(*ir.NullExpr)
	_, rightNull := value.Right.(*ir.NullExpr)
	if leftNull || rightNull {
		other := value.Right
		if rightNull {
			other = value.Left
		}
		if !naturalNullable(other) {
			return "AhdConstBool(" + generator.expr(other) + ", " + strconv.FormatBool(negated) + ")"
		}
		operator := " == nil)"
		if negated {
			operator = " != nil)"
		}
		return "(" + generator.expr(other) + operator
	}
	operandType := value.Left.ExprMeta().Type
	nullable := naturalNullable(value.Left) || naturalNullable(value.Right)
	comparer := generator.equalFunc(operandType, nullable, meta.Span)
	code := comparer + "(" + generator.value(value.Left, operandType, nullable) + ", " + generator.value(value.Right, operandType, nullable) + ")"
	if negated {
		return "(!" + code + ")"
	}
	return code
}

// same is strict type plus value or instance identity. Statically distinct
// operand types can never be the same value, and both operands still evaluate.
func (generator *generator) same(value *ir.BinaryExpr) string {
	meta := value.ExprMeta()
	left, right := value.Left.ExprMeta().Type, value.Right.ExprMeta().Type
	if !ir.EqualType(left, right) {
		return "AhdSameDifferent(" + generator.expr(value.Left) + ", " + generator.expr(value.Right) + ")"
	}
	if left.Kind == ir.ListType || left.Kind == ir.PairType || left.Kind == ir.ClassType {
		return "(" + generator.expr(value.Left) + " == " + generator.expr(value.Right) + ")"
	}
	nullable := naturalNullable(value.Left) || naturalNullable(value.Right)
	comparer := generator.equalFunc(left, nullable, meta.Span)
	return comparer + "(" + generator.value(value.Left, left, nullable) + ", " + generator.value(value.Right, left, nullable) + ")"
}

func (generator *generator) membership(value *ir.BinaryExpr, negated bool) string {
	meta := value.ExprMeta()
	container := value.Right.ExprMeta().Type
	var code string
	switch container.Kind {
	case ir.ListType:
		if container.Element == nil {
			return generator.unsupported("membership in an untyped List", meta.Span)
		}
		element := *container.Element
		code = "AhdListContains(" + generator.expr(value.Right) + ", " + generator.value(value.Left, element, true) + ", " + generator.equalFunc(element, true, meta.Span) + ")"
	case ir.StringType:
		code = "AhdStringContains(" + generator.value(value.Right, ir.Type{Kind: ir.StringType}, false) + ", " + generator.value(value.Left, ir.Type{Kind: ir.StringType}, false) + ")"
	case ir.PairType:
		if container.Key == nil {
			return generator.unsupported("membership in an untyped Pair", meta.Span)
		}
		code = generator.expr(value.Right) + ".Has(" + generator.value(value.Left, *container.Key, false) + ")"
	default:
		return generator.unsupported("membership in "+container.String(), meta.Span)
	}
	if negated {
		return "(!" + code + ")"
	}
	return code
}

// classMembership resolves is / is not statically. Class inheritance is
// deferred, so exact Class identity decides membership for a non-null value.
func (generator *generator) classMembership(value *ir.BinaryExpr, negated bool) string {
	meta := value.ExprMeta()
	instance := value.Left.ExprMeta().Type
	reference, ok := value.Right.(*ir.ClassRefExpr)
	if !ok || instance.Kind != ir.ClassType {
		return generator.unsupported("a Class membership test with these operands", meta.Span)
	}
	code := "AhdIsType(" + generator.expr(value.Left) + ", " + strconv.FormatBool(instance.Class == reference.Class) + ")"
	if negated {
		return "(!" + code + ")"
	}
	return code
}

// memberExistence resolves has / has not from the Class declaration recorded
// in the IR rather than from any runtime reflection.
func (generator *generator) memberExistence(value *ir.BinaryExpr, negated bool) string {
	meta := value.ExprMeta()
	literal, ok := value.Right.(*ir.LiteralExpr)
	instance := value.Left.ExprMeta().Type
	if !ok || literal.Kind != ir.StringLiteral || instance.Kind != ir.ClassType {
		return generator.unsupported("a Class member existence test with these operands", meta.Span)
	}
	exists := false
	if class := generator.classes[instance.Class]; class != nil {
		for _, field := range class.Fields {
			if field.Name == literal.Value {
				exists = true
			}
		}
		for _, method := range class.Methods {
			if function := generator.functions[method]; function != nil && function.Name == literal.Value {
				exists = true
			}
		}
	}
	if negated {
		exists = !exists
	}
	return "AhdConstBool(" + generator.expr(value.Left) + ", " + strconv.FormatBool(exists) + ")"
}

// ---------------------------------------------------------------------------
// Canonical rendering and equality helper selection
// ---------------------------------------------------------------------------

// renderFunc returns a Go expression whose value renders one AhdCode value as
// canonical str text. nested selects the collection-internal String form.
func (generator *generator) renderFunc(value ir.Type, nullable, nested bool, span source.Span) string {
	base := ""
	switch value.Kind {
	case ir.IntType:
		base = "AhdStrInt"
	case ir.RealType:
		base = "AhdStrReal"
	case ir.BoolType:
		base = "AhdStrBool"
	case ir.StringType:
		base = "AhdStrString"
		if nested {
			base = "AhdStrQuoted"
		}
	case ir.ListType:
		if value.Element == nil {
			return generator.unsupported("canonical text for an untyped List", span)
		}
		element := generator.goType(*value.Element, true)
		return "AhdStrList[" + element + "](" + generator.renderFunc(*value.Element, true, true, span) + ")"
	case ir.PairType:
		if value.Key == nil || value.Value == nil {
			return generator.unsupported("canonical text for an untyped Pair", span)
		}
		key, item := generator.goType(*value.Key, false), generator.goType(*value.Value, true)
		return "AhdStrPair[" + key + ", " + item + "](" + generator.renderFunc(*value.Key, false, true, span) + ", " + generator.renderFunc(*value.Value, true, true, span) + ")"
	case ir.ClassType:
		class := generator.classes[value.Class]
		name := string(value.Class)
		if class != nil {
			name = class.Name
		}
		return "AhdStrRef[" + generator.className(value.Class) + "](" + strconv.Quote(name) + ")"
	default:
		return generator.unsupported("canonical text for "+value.String(), span)
	}
	if !nullable {
		return base
	}
	return "AhdStrNull[" + generator.plainType(value) + "](" + base + ")"
}

func (generator *generator) equalFunc(value ir.Type, nullable bool, span source.Span) string {
	base := ""
	switch value.Kind {
	case ir.IntType:
		base = "AhdEqInt"
	case ir.RealType:
		base = "AhdEqReal"
	case ir.BoolType:
		base = "AhdEqBool"
	case ir.StringType:
		base = "AhdEqString"
	case ir.ListType:
		if value.Element == nil {
			return generator.unsupported("equality for an untyped List", span)
		}
		element := generator.goType(*value.Element, true)
		return "AhdEqList[" + element + "](" + generator.equalFunc(*value.Element, true, span) + ")"
	case ir.PairType:
		if value.Key == nil || value.Value == nil {
			return generator.unsupported("equality for an untyped Pair", span)
		}
		key, item := generator.goType(*value.Key, false), generator.goType(*value.Value, true)
		return "AhdEqPair[" + key + ", " + item + "](" + generator.equalFunc(*value.Value, true, span) + ")"
	case ir.ClassType:
		return "AhdEqRef[" + generator.className(value.Class) + "]()"
	default:
		return generator.unsupported("equality for "+value.String(), span)
	}
	if !nullable {
		return base
	}
	return "AhdEqNull[" + generator.plainType(value) + "](" + base + ")"
}

// text renders one expression as canonical str text.
func (generator *generator) text(expression ir.Expr) string {
	meta := expression.ExprMeta()
	if function, ok := expression.(*ir.FunctionValueExpr); ok {
		name := string(function.Callable)
		if declared := generator.functions[function.Callable]; declared != nil {
			name = declared.Name
		}
		return "AhdStrFunction(" + strconv.Quote(name) + ")"
	}
	nullable := naturalNullable(expression)
	return generator.renderFunc(meta.Type, nullable, false, meta.Span) + "(" + generator.expr(expression) + ")"
}

func (generator *generator) toString(value *ir.ToStringExpr) string {
	if value.Value == nil {
		generator.fail(CodeGenerationFailure, "str has no operand", value.ExprMeta().Span, "the IR node is incomplete")
		return `""`
	}
	return generator.text(value.Value)
}

func (generator *generator) stringParts(value *ir.StringExpr) string {
	if len(value.Parts) == 0 {
		return `""`
	}
	parts := make([]string, 0, len(value.Parts))
	for _, part := range value.Parts {
		if part.ToString == nil {
			parts = append(parts, strconv.Quote(part.Literal))
			continue
		}
		parts = append(parts, generator.expr(part.ToString))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "(" + strings.Join(parts, " + ") + ")"
}

// ---------------------------------------------------------------------------
// Collections, members, and calls
// ---------------------------------------------------------------------------

func (generator *generator) listLiteral(value *ir.ListExpr) string {
	element := generator.goType(value.ElementType, true)
	if element == "" {
		return generator.unsupported("a List literal of "+value.ElementType.String(), value.ExprMeta().Span)
	}
	parts := make([]string, 0, len(value.Elements))
	for _, item := range value.Elements {
		parts = append(parts, generator.value(item, value.ElementType, true))
	}
	return "AhdNewList[" + element + "](" + strings.Join(parts, ", ") + ")"
}

func (generator *generator) pairLiteral(value *ir.PairExpr) string {
	key, item := generator.goType(value.KeyType, false), generator.goType(value.ValueType, true)
	if key == "" || item == "" {
		return generator.unsupported("a Pair literal of "+value.ExprMeta().Type.String(), value.ExprMeta().Span)
	}
	keys := make([]string, 0, len(value.Entries))
	values := make([]string, 0, len(value.Entries))
	for _, entry := range value.Entries {
		keys = append(keys, generator.value(entry.Key, value.KeyType, false))
		values = append(values, generator.value(entry.Value, value.ValueType, true))
	}
	return "AhdBuildPair([]" + key + "{" + strings.Join(keys, ", ") + "}, []" + item + "{" + strings.Join(values, ", ") + "})"
}

func (generator *generator) index(value *ir.IndexExpr) string {
	meta := value.ExprMeta()
	container := value.Object.ExprMeta().Type
	switch container.Kind {
	case ir.ListType:
		return generator.expr(value.Object) + ".At(" + generator.value(value.Index, ir.Type{Kind: ir.IntType}, false) + ")"
	case ir.PairType:
		if container.Key == nil {
			return generator.unsupported("indexing an untyped Pair", meta.Span)
		}
		return generator.expr(value.Object) + ".Get(" + generator.value(value.Index, *container.Key, false) + ")"
	case ir.StringType:
		return "AhdStringAt(" + generator.value(value.Object, ir.Type{Kind: ir.StringType}, false) + ", " + generator.value(value.Index, ir.Type{Kind: ir.IntType}, false) + ")"
	default:
		return generator.unsupported("indexing "+container.String(), meta.Span)
	}
}

func (generator *generator) slice(value *ir.SliceExpr) string {
	meta := value.ExprMeta()
	bound := func(expression ir.Expr) (string, string) {
		if expression == nil {
			return "0", "false"
		}
		return generator.value(expression, ir.Type{Kind: ir.IntType}, false), "true"
	}
	start, hasStart := bound(value.Start)
	end, hasEnd := bound(value.End)
	container := value.Object.ExprMeta().Type
	switch container.Kind {
	case ir.ListType:
		return generator.expr(value.Object) + ".Slice(" + start + ", " + hasStart + ", " + end + ", " + hasEnd + ")"
	case ir.StringType:
		return "AhdStringSlice(" + generator.value(value.Object, ir.Type{Kind: ir.StringType}, false) + ", " + start + ", " + hasStart + ", " + end + ", " + hasEnd + ")"
	default:
		return generator.unsupported("slicing "+container.String(), meta.Span)
	}
}

func (generator *generator) member(value *ir.MemberExpr) string {
	meta := value.ExprMeta()
	if value.Kind == ir.FieldMember {
		field, known := generator.fields[value.Field]
		if !known {
			generator.fail(CodeGenerationFailure, "unknown FieldID "+string(value.Field), meta.Span, "the IR references a field with no declaration")
			return "nil"
		}
		access := generator.expr(value.Object) + "." + generator.fieldName(value.Field)
		return generator.coerce(access, ir.ExprBase{Type: field.Type, NullState: nullState(generator.nullFields[value.Field])}, meta.Type, naturalNullable(value))
	}
	function := generator.functions[value.Callable]
	if function == nil {
		return generator.unsupported("a method value with no generated callable", meta.Span)
	}
	// A method used as a Function value binds its receiver exactly once.
	parameters := make([]string, 0, len(function.Parameters))
	arguments := make([]string, 0, len(function.Parameters)+1)
	arguments = append(arguments, "bound")
	for index, parameter := range function.Parameters {
		name := "argument" + strconv.Itoa(index)
		parameters = append(parameters, name+" "+generator.goType(parameter.Type, false))
		arguments = append(arguments, name)
	}
	result := ""
	body := generator.callableName(function) + "(" + strings.Join(arguments, ", ") + ")"
	if function.Signature.Return.Kind != ir.NothingType {
		result = " " + generator.goType(function.Signature.Return, false)
		body = "return " + body
	}
	closure := "func(" + strings.Join(parameters, ", ") + ")" + result + " { " + body + " }"
	return "func(bound *" + generator.className(function.Owner) + ") " + generator.functionType(&function.Signature) + " { return " + closure + " }(" + generator.expr(value.Object) + ")"
}

func (generator *generator) construct(value *ir.ConstructExpr) string {
	meta := value.ExprMeta()
	constructor := generator.functions[value.Constructor]
	if constructor == nil {
		return generator.unsupported("construction of "+string(value.Class), meta.Span)
	}
	return generator.callableName(constructor) + "(" + strings.Join(generator.arguments(constructor, value.Arguments, meta.Span), ", ") + ")"
}

func (generator *generator) call(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	if strings.HasPrefix(string(value.Callable), "builtin:core::") {
		return generator.builtinCall(value)
	}
	if method, ok := value.Callee.(*ir.MemberExpr); ok && method.Kind == ir.MethodMember {
		function := generator.functions[method.Callable]
		if function == nil {
			return generator.unsupported("a method call with no generated callable", meta.Span)
		}
		arguments := append([]string{generator.expr(method.Object)}, generator.arguments(function, value.Arguments, meta.Span)...)
		return generator.callableName(function) + "(" + strings.Join(arguments, ", ") + ")"
	}
	if function := generator.functions[value.Callable]; function != nil {
		return generator.callableName(function) + "(" + strings.Join(generator.arguments(function, value.Arguments, meta.Span), ", ") + ")"
	}
	if value.Callee == nil {
		return generator.unsupported("a call to unknown callable "+string(value.Callable), meta.Span)
	}
	// Indirect call through a concrete Function value.
	signature := value.Callee.ExprMeta().Type.Signature
	if signature == nil {
		return generator.unsupported("an indirect call without a concrete signature", meta.Span)
	}
	parts := make([]string, 0, len(value.Arguments))
	for index, argument := range value.Arguments {
		if argument.UsesDefault || argument.Value == nil {
			return generator.unsupported("a default argument in an indirect call", meta.Span)
		}
		target := ir.Type{Kind: ir.InvalidType}
		if index < len(signature.Parameters) {
			target = signature.Parameters[index].Type
		}
		parts = append(parts, generator.value(argument.Value, target, false))
	}
	return generator.expr(value.Callee) + "(" + strings.Join(parts, ", ") + ")"
}

func (generator *generator) arguments(function *ir.Function, arguments []ir.Argument, span source.Span) []string {
	parts := make([]string, 0, len(arguments))
	for index, argument := range arguments {
		if index >= len(function.Parameters) {
			generator.fail(CodeGenerationFailure, "call has more arguments than the callable declares", span, "the IR call is malformed")
			break
		}
		parameter := function.Parameters[index]
		source := argument.Value
		if argument.UsesDefault {
			source = parameter.Default
		}
		if source == nil {
			generator.fail(CodeGenerationFailure, "argument for parameter "+parameter.Name+" has no value", span, "the IR call is malformed")
			continue
		}
		parts = append(parts, generator.value(source, parameter.Type, false))
	}
	return parts
}

// builtinCall lowers the Fundamentals entry points that the backend provides
// directly. Every other builtin is reported rather than guessed.
func (generator *generator) builtinCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), "builtin:core::")
	argument := func(index int) ir.Expr {
		if index < len(value.Arguments) {
			return value.Arguments[index].Value
		}
		return nil
	}
	switch name {
	case "write":
		if argument(0) == nil {
			generator.fail(CodeGenerationFailure, "write has no argument", meta.Span, "the IR call is malformed")
			return "nil"
		}
		return "AhdWrite(" + generator.text(argument(0)) + ")"
	case "take":
		if argument(0) == nil {
			return `AhdTake("", false)`
		}
		return "AhdTake(" + generator.value(argument(0), ir.Type{Kind: ir.StringType}, false) + ", true)"
	case "len":
		return generator.length(argument(0), meta.Span)
	case "clear":
		if argument(0) == nil {
			generator.fail(CodeGenerationFailure, "clear has no argument", meta.Span, "the IR call is malformed")
			return "nil"
		}
		return generator.expr(argument(0)) + ".Clear()"
	default:
		return generator.unsupported("Fundamentals function "+name, meta.Span)
	}
}

func (generator *generator) length(expression ir.Expr, span source.Span) string {
	if expression == nil {
		generator.fail(CodeGenerationFailure, "len has no argument", span, "the IR call is malformed")
		return "int64(0)"
	}
	switch expression.ExprMeta().Type.Kind {
	case ir.StringType:
		return "AhdStringLen(" + generator.value(expression, ir.Type{Kind: ir.StringType}, false) + ")"
	case ir.ListType, ir.PairType:
		return generator.expr(expression) + ".Len()"
	default:
		return generator.unsupported("len of "+expression.ExprMeta().Type.String(), span)
	}
}
