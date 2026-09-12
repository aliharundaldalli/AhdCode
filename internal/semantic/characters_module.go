package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const charactersModuleID = "builtin:Characters"

var charactersErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
var charactersErrorClass = &types.ClassSymbol{
	ModuleID: charactersModuleID, Name: "CharactersError",
	Parent: charactersErrorParent,
}

// CharactersErrorIdentity exposes the canonical identity to the lowering layer.
func CharactersErrorIdentity() *types.ClassSymbol { return charactersErrorClass }

// CharactersClassifiers are the one-character Bool predicates the module
// publishes, in documentation order.
var CharactersClassifiers = []string{
	"isLetter", "isDigit", "isWhitespace", "isUpper", "isLower",
	"isAlphaNumeric", "isPunctuation", "isSymbol",
}

// charactersModuleInterface declares the Characters standard module.
//
// The module works in Unicode code points. A character is an ordinary String
// holding exactly one code point; there is no Char type. Static argument types
// are checked here like every other standard function, while the
// exactly-one-code-point rule is a value-domain check that raises
// CharactersError at run time.
func charactersModuleInterface() *ModuleInterface {
	module := standardInterface(charactersModuleID, "Characters")
	errorSymbol := &Symbol{
		Name: "CharactersError", Kind: ClassSymbol, Class: charactersErrorClass,
		Type: types.Class{Symbol: charactersErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: charactersModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[charactersModuleID+"\x00CharactersError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	text := types.Parameter{Name: "text", Type: types.String}
	character := types.Parameter{Name: "character", Type: types.String}

	addStandardExport(module, standardFunction(charactersModuleID, "list", types.List{Element: types.String}, text))
	addStandardExport(module, standardFunction(charactersModuleID, "count", types.Int, text))
	addStandardExport(module, standardFunction(charactersModuleID, "codePoint", types.Int, character))
	addStandardExport(module, standardFunction(charactersModuleID, "fromCodePoint", types.String,
		types.Parameter{Name: "value", Type: types.Int}))
	for _, name := range CharactersClassifiers {
		addStandardExport(module, standardFunction(charactersModuleID, name, types.Bool, character))
	}

	sort.Strings(module.ExportNames)
	return module
}
