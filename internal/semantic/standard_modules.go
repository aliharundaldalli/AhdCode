package semantic

import (
	"math"
	"sort"
	"strconv"

	"ahdcode/internal/types"
)

const mathModuleID = "builtin:Math"

// StandardModuleInterfaces returns a fresh registry of the explicit standard
// modules supplied by the compiler. They use the same ModuleInterface model as
// source modules, but have canonical builtin identities and no source file.
func StandardModuleInterfaces() map[string]*ModuleInterface {
	return map[string]*ModuleInterface{
		"Archive":    archiveModuleInterface(),
		"CSV":        csvModuleInterface(),
		"Data":       dataModuleInterface(),
		"Env":        envModuleInterface(),
		"Excel":      excelModuleInterface(),
		"File":       fileModuleInterface(),
		"HTML":       htmlModuleInterface(),
		"HTTP":       httpModuleInterface(),
		"JSON":       jsonModuleInterface(),
		"KeyValue":   keyValueModuleInterface(),
		"Latex":      latexModuleInterface(),
		"Lists":      listsModuleInterface(),
		"Math":       mathModuleInterface(),
		"Numeric":    numericModuleInterface(),
		"Path":       pathModuleInterface(),
		"PDF":        pdfModuleInterface(),
		"Plot":       plotModuleInterface(),
		"Regex":      regexModuleInterface(),
		"SQLite":     sqliteModuleInterface(),
		"Statistics": statisticsModuleInterface(),
		"Time":       timeModuleInterface(),
		"Word":       wordModuleInterface(),
		"XML":        xmlModuleInterface(),
	}
}

func mathModuleInterface() *ModuleInterface {
	module := &ModuleInterface{
		ModuleID: mathModuleID,
		Name:     "Math",
		Exports:  make(map[string]*Symbol),
		Symbols:  make(map[string]*Symbol),
		Classes:  make(map[string]*Symbol),
	}
	add := func(symbol *Symbol) {
		module.Symbols[symbol.Name] = symbol
		module.Exports[symbol.Name] = symbol
		module.ExportNames = append(module.ExportNames, symbol.Name)
	}
	add(mathRealConstant("PI", math.Pi))
	add(mathRealConstant("E", math.E))
	add(mathFunction("round",
		mathSignature(types.Real, mathParameter("value", types.Real)),
		mathSignature(types.Real, mathParameter("value", types.Real), mathParameter("digits", types.Int)),
	))
	add(mathFunction("floor", mathSignature(types.Int, mathParameter("value", types.Real))))
	add(mathFunction("ceil", mathSignature(types.Int, mathParameter("value", types.Real))))
	for _, name := range []string{"sqrt", "sin", "cos", "tan", "log", "log10", "exp"} {
		add(mathFunction(name, mathSignature(types.Real, mathParameter("value", types.Real))))
	}
	add(mathFunction("seed", mathSignature(types.Nothing, mathParameter("value", types.Int))))
	add(mathFunction("random", mathSignature(types.Real)))
	add(mathFunction("randomInt", mathSignature(types.Int,
		mathParameter("min", types.Int), mathParameter("max", types.Int))))
	sort.Strings(module.ExportNames)
	return module
}

func mathRealConstant(name string, value float64) *Symbol {
	return &Symbol{
		Name: name, Kind: BindingSymbol, Type: types.Real,
		Constant: true, ModuleRoot: true, Builtin: true,
		InitialNull: NonNull, OriginModuleID: mathModuleID,
		BuiltinLiteral: strconv.FormatFloat(value, 'g', -1, 64),
		ConstValue:     &constantValue{typeValue: types.Real, real: value},
	}
}

func mathFunction(name string, signatures ...*types.Signature) *Symbol {
	symbol := &Symbol{
		Name: name, Kind: FunctionSymbol, Type: types.Function{},
		ModuleRoot: true, Builtin: true, InitialNull: NonNull,
		OriginModuleID: mathModuleID,
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

func mathSignature(result types.Type, parameters ...types.Parameter) *types.Signature {
	return &types.Signature{Parameters: parameters, Return: result}
}

func mathParameter(name string, value types.Type) types.Parameter {
	return types.Parameter{Name: name, Type: value}
}
