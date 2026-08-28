package semantic

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

type constantValue struct {
	typeValue types.Type
	integer   *big.Int
	real      float64
	text      string
	boolean   bool
	overflow  bool
}

type constFailure uint8

const (
	constOK constFailure = iota
	constNotExpression
	constCycle
	constInvalid
)

var (
	minInt = new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 63))
	maxInt = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 63), big.NewInt(1))
)

func (value *constantValue) fitsInt() bool {
	return value != nil && value.integer != nil && !value.overflow && value.integer.Cmp(minInt) >= 0 && value.integer.Cmp(maxInt) <= 0
}

func (a *analyzer) evaluateConstant(expression ast.Expr) (*constantValue, constFailure) {
	return a.evaluateConstantWithStack(expression, make(map[*Symbol]bool))
}

func (a *analyzer) evaluateConstantWithStack(expression ast.Expr, stack map[*Symbol]bool) (*constantValue, constFailure) {
	switch value := expression.(type) {
	case *ast.LiteralExpr:
		switch value.Kind {
		case ast.IntLiteral:
			integer, ok := new(big.Int).SetString(strings.ReplaceAll(value.Value, "_", ""), 10)
			if !ok {
				integer, ok = new(big.Int).SetString(strings.ReplaceAll(value.Raw, "_", ""), 10)
			}
			if !ok {
				return nil, constInvalid
			}
			return &constantValue{typeValue: types.Int, integer: integer}, constOK
		case ast.RealLiteral:
			real, err := strconv.ParseFloat(strings.ReplaceAll(value.Value, "_", ""), 64)
			if err != nil {
				real, err = strconv.ParseFloat(strings.ReplaceAll(value.Raw, "_", ""), 64)
			}
			if err != nil || math.IsInf(real, 0) {
				return nil, constInvalid
			}
			return &constantValue{typeValue: types.Real, real: real}, constOK
		case ast.BoolLiteral:
			return &constantValue{typeValue: types.Bool, boolean: value.Value == "true"}, constOK
		case ast.NullLiteral:
			return nil, constNotExpression
		}
	case *ast.StringExpr:
		var builder strings.Builder
		for _, part := range value.Parts {
			if part.Expression != nil {
				return nil, constNotExpression
			}
			builder.WriteString(part.Text)
		}
		return &constantValue{typeValue: types.String, text: builder.String()}, constOK
	case *ast.GroupExpr:
		return a.evaluateConstantWithStack(value.Expression, stack)
	case *ast.IdentifierExpr:
		symbol := a.result.ResolvedSymbols[value]
		if symbol == nil {
			symbol, _ = a.module.local(value.Name)
		}
		if symbol == nil || !symbol.Constant {
			return nil, constNotExpression
		}
		if symbol.ConstValue != nil {
			return cloneConstant(symbol.ConstValue), constOK
		}
		if stack[symbol] {
			return nil, constCycle
		}
		declaration, ok := symbol.Declaration.(*ast.VariableDecl)
		if !ok || declaration.Initializer == nil {
			return nil, constNotExpression
		}
		stack[symbol] = true
		resolved, failure := a.evaluateConstantWithStack(declaration.Initializer, stack)
		delete(stack, symbol)
		if failure == constOK {
			symbol.ConstValue = cloneConstant(resolved)
		}
		return resolved, failure
	case *ast.UnaryExpr:
		operand, failure := a.evaluateConstantWithStack(value.Operand, stack)
		if failure != constOK {
			return nil, failure
		}
		switch value.Operator {
		case "+":
			if !types.IsNumeric(operand.typeValue) {
				return nil, constInvalid
			}
			return operand, constOK
		case "-":
			if operand.typeValue.Kind() == types.IntKind {
				operand.integer.Neg(operand.integer)
				return operand, constOK
			}
			if operand.typeValue.Kind() == types.RealKind {
				operand.real = -operand.real
				return operand, constOK
			}
			return nil, constInvalid
		case "not":
			if operand.typeValue.Kind() != types.BoolKind {
				return nil, constInvalid
			}
			operand.boolean = !operand.boolean
			return operand, constOK
		}
	case *ast.BinaryExpr:
		left, leftFailure := a.evaluateConstantWithStack(value.Left, stack)
		if leftFailure != constOK {
			return nil, leftFailure
		}
		right, rightFailure := a.evaluateConstantWithStack(value.Right, stack)
		if rightFailure != constOK {
			return nil, rightFailure
		}
		return evaluateConstantBinary(value.Operator, left, right)
	}
	return nil, constNotExpression
}

func evaluateConstantBinary(operator string, left, right *constantValue) (*constantValue, constFailure) {
	if operator == "and" || operator == "or" {
		if left.typeValue.Kind() != types.BoolKind || right.typeValue.Kind() != types.BoolKind {
			return nil, constInvalid
		}
		result := left.boolean && right.boolean
		if operator == "or" {
			result = left.boolean || right.boolean
		}
		return &constantValue{typeValue: types.Bool, boolean: result}, constOK
	}
	if operator == "+" && left.typeValue.Kind() == types.StringKind && right.typeValue.Kind() == types.StringKind {
		return &constantValue{typeValue: types.String, text: left.text + right.text}, constOK
	}
	if operator == "*" && left.typeValue.Kind() == types.StringKind && right.typeValue.Kind() == types.IntKind {
		if !right.integer.IsInt64() || right.integer.Sign() < 0 || right.integer.Int64() > 1_000_000 {
			return nil, constInvalid
		}
		return &constantValue{typeValue: types.String, text: strings.Repeat(left.text, int(right.integer.Int64()))}, constOK
	}
	if types.IsNumeric(left.typeValue) && types.IsNumeric(right.typeValue) {
		return evaluateNumericConstant(operator, left, right)
	}
	if operator == "==" || operator == "!=" || operator == "same" {
		if !types.Equal(left.typeValue, right.typeValue) {
			return nil, constInvalid
		}
		equal := constantsEqual(left, right)
		if operator == "!=" {
			equal = !equal
		}
		return &constantValue{typeValue: types.Bool, boolean: equal}, constOK
	}
	return nil, constInvalid
}

func evaluateNumericConstant(operator string, left, right *constantValue) (*constantValue, constFailure) {
	bothInt := left.typeValue.Kind() == types.IntKind && right.typeValue.Kind() == types.IntKind
	if bothInt {
		result := new(big.Int)
		switch operator {
		case "+":
			result.Add(left.integer, right.integer)
		case "-":
			result.Sub(left.integer, right.integer)
		case "*":
			result.Mul(left.integer, right.integer)
		case "%":
			if right.integer.Sign() == 0 {
				return nil, constInvalid
			}
			result.Rem(left.integer, right.integer)
		case "^":
			if right.integer.Sign() < 0 {
				// Int power remains Int regardless of exponent sign. A negative
				// exponent is a runtime DomainError, never a folded Real value.
				return nil, constInvalid
			}
			if !right.integer.IsInt64() || right.integer.Int64() > 10000 {
				return &constantValue{typeValue: types.Int, integer: new(big.Int), overflow: true}, constOK
			}
			result.Exp(left.integer, right.integer, nil)
		case "/":
			if right.integer.Sign() == 0 {
				return nil, constInvalid
			}
			return &constantValue{typeValue: types.Real, real: toFloat(left) / toFloat(right)}, constOK
		case "<", "<=", ">", ">=", "==", "!=", "same":
			return numericComparison(operator, left, right), constOK
		default:
			return nil, constInvalid
		}
		return &constantValue{typeValue: types.Int, integer: result}, constOK
	}
	leftReal, rightReal := toFloat(left), toFloat(right)
	switch operator {
	case "+":
		return &constantValue{typeValue: types.Real, real: leftReal + rightReal}, constOK
	case "-":
		return &constantValue{typeValue: types.Real, real: leftReal - rightReal}, constOK
	case "*":
		return &constantValue{typeValue: types.Real, real: leftReal * rightReal}, constOK
	case "/":
		if rightReal == 0 {
			return nil, constInvalid
		}
		return &constantValue{typeValue: types.Real, real: leftReal / rightReal}, constOK
	case "^":
		return &constantValue{typeValue: types.Real, real: math.Pow(leftReal, rightReal)}, constOK
	case "<", "<=", ">", ">=", "==", "!=", "same":
		return numericComparison(operator, left, right), constOK
	}
	return nil, constInvalid
}

func numericComparison(operator string, left, right *constantValue) *constantValue {
	leftValue, rightValue := toFloat(left), toFloat(right)
	var result bool
	switch operator {
	case "<":
		result = leftValue < rightValue
	case "<=":
		result = leftValue <= rightValue
	case ">":
		result = leftValue > rightValue
	case ">=":
		result = leftValue >= rightValue
	case "==":
		result = leftValue == rightValue
	case "!=":
		result = leftValue != rightValue
	case "same":
		result = types.Equal(left.typeValue, right.typeValue) && leftValue == rightValue
	}
	return &constantValue{typeValue: types.Bool, boolean: result}
}

func constantsEqual(left, right *constantValue) bool {
	switch left.typeValue.Kind() {
	case types.IntKind:
		return left.integer.Cmp(right.integer) == 0
	case types.RealKind:
		return left.real == right.real
	case types.StringKind:
		return left.text == right.text
	case types.BoolKind:
		return left.boolean == right.boolean
	}
	return false
}

func toFloat(value *constantValue) float64 {
	if value.typeValue.Kind() == types.RealKind {
		return value.real
	}
	converted, _ := new(big.Float).SetInt(value.integer).Float64()
	return converted
}

func cloneConstant(value *constantValue) *constantValue {
	if value == nil {
		return nil
	}
	cloned := *value
	if value.integer != nil {
		cloned.integer = new(big.Int).Set(value.integer)
	}
	return &cloned
}
