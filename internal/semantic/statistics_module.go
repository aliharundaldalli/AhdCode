package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const statisticsModuleID = "builtin:Statistics"

var (
	statisticsErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	statisticsErrorClass = &types.ClassSymbol{ModuleID: statisticsModuleID, Name: "StatisticsError",
		Parent: statisticsErrorParent}
)

// StatisticsErrorIdentity exposes the canonical identity to the lowering layer.
func StatisticsErrorIdentity() *types.ClassSymbol { return statisticsErrorClass }

// The Statistics standard module is descriptive statistics over typed numeric
// Lists. It deliberately does not depend on Data: a Table cell is a String, so
// a program converts explicitly before asking for a statistic, which keeps both
// modules strict instead of introducing a dynamic numeric value.
//
// Every function is declared as an explicit Int/Real overload pair rather than
// one weakly typed entry point, so the static type of a result is always known.
func statisticsModuleInterface() *ModuleInterface {
	module := standardInterface(statisticsModuleID, "Statistics")

	errorSymbol := &Symbol{
		Name: "StatisticsError", Kind: ClassSymbol, Class: statisticsErrorClass,
		Type: types.Class{Symbol: statisticsErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: statisticsModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[statisticsModuleID+"\x00StatisticsError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	values := func(element types.Type) types.Parameter {
		return types.Parameter{Name: "values", Type: types.List{Element: element}}
	}

	// A statistic that stays in its input's type: summing Ints is an Int, and
	// the extremes and their difference are elements of the input.
	for _, name := range []string{"sum", "min", "max", "range"} {
		addStandardExport(module, statisticsFunction(name,
			statisticsSignature(types.Int, values(types.Int)),
			statisticsSignature(types.Real, values(types.Real)),
		))
	}
	// mode returns one of the input's own values, so it keeps the element type.
	addStandardExport(module, statisticsFunction("mode",
		statisticsSignature(types.Int, values(types.Int)),
		statisticsSignature(types.Real, values(types.Real)),
	))
	// A statistic that averages or measures spread is Real for both inputs,
	// because the answer is generally not a whole number even for Int input.
	for _, name := range []string{"mean", "median", "variance", "sampleVariance", "stdDev", "sampleStdDev"} {
		addStandardExport(module, statisticsFunction(name,
			statisticsSignature(types.Real, values(types.Int)),
			statisticsSignature(types.Real, values(types.Real)),
		))
	}
	addStandardExport(module, statisticsFunction("quantile",
		statisticsSignature(types.Real, values(types.Int), types.Parameter{Name: "probability", Type: types.Real}),
		statisticsSignature(types.Real, values(types.Real), types.Parameter{Name: "probability", Type: types.Real}),
	))
	sort.Strings(module.ExportNames)
	return module
}

func statisticsSignature(result types.Type, parameters ...types.Parameter) *types.Signature {
	return &types.Signature{Parameters: parameters, Return: result}
}

// statisticsFunction publishes one Statistics entry point. Two signatures make
// an ordinary overload set, resolved by the existing machinery, so Int and Real
// inputs never share a weakened parameter type.
func statisticsFunction(name string, signatures ...*types.Signature) *Symbol {
	symbol := &Symbol{
		Name: name, Kind: FunctionSymbol, Type: types.Function{},
		ModuleRoot: true, Builtin: true, InitialNull: NonNull,
		OriginModuleID: statisticsModuleID,
	}
	for _, signature := range signatures {
		callable := &Callable{
			Signature: signature, ParameterNull: nonNullParameters(len(signature.Parameters)),
			ReturnNull: NonNull,
		}
		if symbol.Callable == nil {
			symbol.Callable = callable
		}
		if len(signatures) > 1 {
			if symbol.OverloadSet == nil {
				symbol.OverloadSet = &OverloadSet{Name: name}
			}
			symbol.OverloadSet.Candidates = append(symbol.OverloadSet.Candidates, callable)
		}
	}
	if len(signatures) == 1 {
		symbol.Type = types.Function{Signature: signatures[0]}
	}
	return symbol
}
