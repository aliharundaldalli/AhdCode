package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const latexModuleID = "builtin:Latex"

var (
	latexErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	latexErrorClass = &types.ClassSymbol{ModuleID: latexModuleID, Name: "LatexError", Parent: latexErrorParent}
)

// LatexErrorIdentity exposes the canonical identity to lowering without
// coupling the public module interface to backend details.
func LatexErrorIdentity() *types.ClassSymbol { return latexErrorClass }

func latexModuleInterface() *ModuleInterface {
	module := &ModuleInterface{
		ModuleID: latexModuleID,
		Name:     "Latex",
		Exports:  make(map[string]*Symbol),
		Symbols:  make(map[string]*Symbol),
		Classes:  make(map[string]*Symbol),
	}
	add := func(symbol *Symbol) {
		module.Symbols[symbol.Name] = symbol
		module.Exports[symbol.Name] = symbol
		module.ExportNames = append(module.ExportNames, symbol.Name)
	}

	errorSymbol := &Symbol{
		Name: "LatexError", Kind: ClassSymbol, Class: latexErrorClass,
		Type: types.Class{Symbol: latexErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: latexModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[latexModuleID+"\x00LatexError"] = errorSymbol
	add(errorSymbol)

	stringParameter := func(name string) types.Parameter { return types.Parameter{Name: name, Type: types.String} }
	add(latexFunction("pdf", latexSignature(types.Nothing,
		stringParameter("source"), stringParameter("output"))))
	add(latexFunction("pdfFile", latexSignature(types.Nothing,
		stringParameter("input"), stringParameter("output"))))
	for _, name := range []string{"escape", "section", "subsection", "equation"} {
		parameter := "text"
		if name == "section" || name == "subsection" {
			parameter = "title"
		} else if name == "equation" {
			parameter = "source"
		}
		add(latexFunction(name, latexSignature(types.String, stringParameter(parameter))))
	}
	add(latexFunction("document", latexSignature(types.String,
		stringParameter("body"),
		types.Parameter{Name: "title", Type: types.String, HasDefault: true},
		types.Parameter{Name: "author", Type: types.String, HasDefault: true},
	)))
	add(latexFunction("table", latexSignature(types.String,
		types.Parameter{Name: "headers", Type: types.List{Element: types.String}},
		types.Parameter{Name: "rows", Type: types.List{Element: types.List{Element: types.String}}},
		types.Parameter{Name: "mathColumns", Type: types.List{Element: types.Int}, HasDefault: true},
	)))

	sort.Strings(module.ExportNames)
	return module
}

func latexFunction(name string, signature *types.Signature) *Symbol {
	return &Symbol{
		Name: name, Kind: FunctionSymbol, Type: types.Function{Signature: signature},
		ModuleRoot: true, Builtin: true, InitialNull: NonNull,
		OriginModuleID: latexModuleID,
		Callable: &Callable{
			Signature: signature, ParameterNull: nonNullParameters(len(signature.Parameters)),
			ReturnNull: NonNull,
		},
	}
}

func latexSignature(result types.Type, parameters ...types.Parameter) *types.Signature {
	return &types.Signature{Parameters: parameters, Return: result}
}
