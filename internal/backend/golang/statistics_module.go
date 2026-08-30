package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const statisticsModulePrefix = "builtin:Statistics::"

// The runtime Class a Statistics helper raises through. It is named directly
// rather than through a generated descriptor for the same reason Data does:
// Statistics can be imported without the Class being separately declared.
const statisticsErrorRuntime = "AhdClassStatisticsError"

// statisticsCall lowers the Statistics standard module. The frontend has
// already selected the Int or Real overload, so this layer only picks the
// matching runtime helper from the argument's static element type.
func (generator *generator) statisticsCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), statisticsModulePrefix)
	if len(value.Arguments) == 0 || value.Arguments[0].Value == nil {
		generator.fail(CodeGenerationFailure, "Statistics."+name+" has no argument", meta.Span, "the IR call is malformed")
		return "nil"
	}
	element := ir.Type{Kind: ir.InvalidType}
	listType := value.Arguments[0].Value.ExprMeta().Type
	if listType.Kind == ir.ListType && listType.Element != nil {
		element = *listType.Element
	}
	suffix := ""
	switch element.Kind {
	case ir.IntType:
		suffix = "Int"
	case ir.RealType:
		suffix = "Real"
	default:
		return generator.unsupported("Statistics."+name+" over List<"+element.String()+">", meta.Span)
	}
	values := generator.value(value.Arguments[0].Value, listType, false)
	// sum needs no error Class: the empty sum is defined, so it cannot fail the
	// way an undefined statistic does.
	if name == "sum" {
		return "AhdStatisticsSum" + suffix + "(" + values + ")"
	}
	if name == "quantile" {
		if len(value.Arguments) != 2 || value.Arguments[1].Value == nil {
			generator.fail(CodeGenerationFailure, "Statistics.quantile has no probability", meta.Span, "the IR call is malformed")
			return "nil"
		}
		probability := generator.value(value.Arguments[1].Value, ir.Type{Kind: ir.RealType}, false)
		return "AhdStatisticsQuantile" + suffix + "(" + statisticsErrorRuntime + ", " + values + ", " + probability + ")"
	}
	switch name {
	case "mean", "median", "min", "max", "range", "variance", "sampleVariance", "stdDev", "sampleStdDev", "mode":
		return "AhdStatistics" + statisticsHelper(name) + suffix + "(" + statisticsErrorRuntime + ", " + values + ")"
	default:
		return generator.unsupported("Statistics function "+name, meta.Span)
	}
}

// statisticsHelper maps an AhdCode name onto its runtime helper's spelling.
func statisticsHelper(name string) string {
	return strings.ToUpper(name[:1]) + name[1:]
}
