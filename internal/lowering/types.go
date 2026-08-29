package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
	"ahdcode/internal/types"
)

func lowerType(value types.Type) ir.Type {
	switch typed := value.(type) {
	case types.Basic:
		switch typed.Kind() {
		case types.IntKind:
			return ir.Type{Kind: ir.IntType}
		case types.RealKind:
			return ir.Type{Kind: ir.RealType}
		case types.StringKind:
			return ir.Type{Kind: ir.StringType}
		case types.BoolKind:
			return ir.Type{Kind: ir.BoolType}
		case types.NothingKind:
			return ir.Type{Kind: ir.NothingType}
		default:
			return ir.Type{Kind: ir.InvalidType}
		}
	case types.List:
		element := lowerType(typed.Element)
		return ir.Type{Kind: ir.ListType, Element: &element, ElementNullable: typed.ElementNullable}
	case types.Pair:
		key, item := lowerType(typed.Key), lowerType(typed.Value)
		return ir.Type{Kind: ir.PairType, Key: &key, Value: &item, ValueNullable: typed.ValueNullable}
	case types.Function:
		return ir.Type{Kind: ir.FunctionType, Signature: lowerSignature(typed.Signature)}
	case types.Class:
		return ir.Type{Kind: ir.ClassType, Class: classID(typed.Symbol), Reference: typed.Reference}
	case types.Range:
		return ir.Type{Kind: ir.RangeType}
	default:
		return ir.Type{Kind: ir.InvalidType}
	}
}

func lowerSignature(value *types.Signature) *ir.Signature {
	if value == nil {
		return nil
	}
	result := &ir.Signature{Return: lowerType(value.Return)}
	for _, parameter := range value.Parameters {
		result.Parameters = append(result.Parameters, ir.ParameterType{Name: parameter.Name, Type: lowerType(parameter.Type), HasDefault: parameter.HasDefault})
	}
	return result
}

func lowerNull(value semantic.NullState) ir.NullState {
	switch value {
	case semantic.Null:
		return ir.Null
	case semantic.NonNull:
		return ir.NonNull
	default:
		return ir.MaybeNull
	}
}
