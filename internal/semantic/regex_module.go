package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const regexModuleID = "builtin:Regex"

var (
	regexErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	regexErrorClass = &types.ClassSymbol{ModuleID: regexModuleID, Name: "RegexError", Parent: regexErrorParent}
	regexClass      = &types.ClassSymbol{ModuleID: regexModuleID, Name: "Pattern"}
)

// RegexErrorIdentity and RegexClassIdentity expose the canonical identities to
// the lowering layer without coupling the public module interface to a
// backend.
func RegexErrorIdentity() *types.ClassSymbol { return regexErrorClass }
func RegexClassIdentity() *types.ClassSymbol { return regexClass }

// regexOperations names the members the Regex Class publishes through built-in
// type operations, so has/has not reports what a Regex value really offers.
var regexOperations = []string{"matches", "find", "findAll", "groups", "replace", "split"}

func regexModuleInterface() *ModuleInterface {
	module := standardInterface(regexModuleID, "Regex")
	errorSymbol := &Symbol{
		Name: "RegexError", Kind: ClassSymbol, Class: regexErrorClass,
		Type: types.Class{Symbol: regexErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: regexModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[regexModuleID+"\x00RegexError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	patternSymbol := &Symbol{
		Name: "Pattern", Kind: ClassSymbol, Class: regexClass,
		Type: types.Class{Symbol: regexClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: regexModuleID,
		Members: make(map[string]*Symbol),
	}
	module.Classes[regexModuleID+"\x00Pattern"] = patternSymbol
	addStandardExport(module, patternSymbol)

	addStandardExport(module, standardFunction(regexModuleID, "compile",
		types.Class{Symbol: regexClass}, types.Parameter{Name: "pattern", Type: types.String}))

	sort.Strings(module.ExportNames)
	return module
}

// regexOperationShape is the fixed call shape of one Regex Class member.
// resultNullable marks find and groups, whose absence of a match is a
// genuine, statically expected possibility rather than an error.
type regexOperationShape struct {
	parameters     []types.Type
	result         types.Type
	resultNullable bool
	hint           string
}

func regexOperationShapes() map[TypeOperation]regexOperationShape {
	text := []types.Type{types.String}
	return map[TypeOperation]regexOperationShape{
		RegexMatches: {text, types.Bool, false, "pass one String to search"},
		RegexFind:    {text, types.String, true, "pass one String to search"},
		RegexFindAll: {text, types.List{Element: types.String}, false, "pass one String to search"},
		RegexGroups:  {text, types.List{Element: types.String}, true, "pass one String to search"},
		RegexReplace: {[]types.Type{types.String, types.String}, types.String, false, "pass the searched String and its replacement"},
		RegexSplit:   {text, types.List{Element: types.String}, false, "pass one String to split on"},
	}
}

var regexOperationNames = map[string]TypeOperation{
	"matches": RegexMatches, "find": RegexFind, "findAll": RegexFindAll,
	"groups": RegexGroups, "replace": RegexReplace, "split": RegexSplit,
}

// regexOperationFor names the built-in member a Regex Class instance
// publishes. Only the compiler-supplied Regex identity matches, so a user
// Class named Regex (impossible: standard modules cannot be shadowed) never
// collides with it.
func regexOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != regexModuleID || class.Symbol.Name != "Pattern" {
		return "", false
	}
	operation, known := regexOperationNames[name]
	return operation, known
}

// analyzeRegexOperation checks one Regex Class member. Every argument is a
// NonNull String, matching the existing Time operation shape check, except
// the result may itself be a statically expected MaybeNull String or
// List<String> for find and groups.
func (a *analyzer) analyzeRegexOperation(call *ast.CallExpr, operation TypeOperation, shape regexOperationShape, current *scope, flow flowState) expressionInfo {
	nullState := NonNull
	if shape.resultNullable {
		nullState = MaybeNull
	}
	result := expressionInfo{typeValue: shape.result, nullState: nullState}
	if len(call.Arguments) != len(shape.parameters) {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument(s); received %d", operation, len(shape.parameters), len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for index, expected := range shape.parameters {
		argument := a.analyzeExpressionExpected(call.Arguments[index].Value, current, flow, expected)
		if argument.invalid() {
			continue
		}
		if argument.nullState != NonNull {
			a.nullableError(string(operation), call.Arguments[index].Value, argument.nullState)
			continue
		}
		if !types.Assignable(expected, argument.typeValue) {
			a.typeMismatch(call.Arguments[index].Span(), expected, argument.typeValue, string(operation)+" argument")
		}
	}
	return result
}
