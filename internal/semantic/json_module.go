package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const jsonModuleID = "builtin:JSON"

var (
	jsonErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	jsonErrorClass = &types.ClassSymbol{ModuleID: jsonModuleID, Name: "JSONError", Parent: jsonErrorParent}
	jsonValueClass = &types.ClassSymbol{ModuleID: jsonModuleID, Name: "JSONValue"}
)

// JSONErrorIdentity and JSONValueIdentity expose the canonical identities to
// the lowering layer without coupling the public module interface to a
// backend.
func JSONErrorIdentity() *types.ClassSymbol { return jsonErrorClass }
func JSONValueIdentity() *types.ClassSymbol { return jsonValueClass }

// JSONValueOperations names the members a JSONValue publishes through
// built-in type operations, so has/has not reports what a JSONValue really
// offers, and the lowering layer's IR Class stays in agreement with the
// frontend about the published surface.
var JSONValueOperations = []string{
	"kind", "isNull", "bool", "int", "real", "string", "array", "object", "get", "at",
}

func jsonValueType() types.Type { return types.Class{Symbol: jsonValueClass} }

func jsonModuleInterface() *ModuleInterface {
	module := standardInterface(jsonModuleID, "JSON")

	errorSymbol := &Symbol{
		Name: "JSONError", Kind: ClassSymbol, Class: jsonErrorClass,
		Type: types.Class{Symbol: jsonErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: jsonModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[jsonModuleID+"\x00JSONError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	valueSymbol := &Symbol{
		Name: "JSONValue", Kind: ClassSymbol, Class: jsonValueClass,
		Type: types.Class{Symbol: jsonValueClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: jsonModuleID,
		Members: make(map[string]*Symbol),
	}
	module.Classes[jsonModuleID+"\x00JSONValue"] = valueSymbol
	addStandardExport(module, valueSymbol)

	value := jsonValueType()
	array := types.List{Element: value}
	object := types.Pair{Key: types.String, Value: value}
	stringParameter := func(name string) types.Parameter { return types.Parameter{Name: name, Type: types.String} }
	pretty := types.Parameter{Name: "pretty", Type: types.Bool, HasDefault: true}

	addStandardExport(module, standardFunction(jsonModuleID, "parse", value, stringParameter("source")))
	addStandardExport(module, standardFunction(jsonModuleID, "read", value, stringParameter("path")))
	// nullValue, not null: `null` is a reserved keyword (§2.1) and cannot
	// appear as a member name after `.`, so `JSON.null()` cannot parse.
	addStandardExport(module, standardFunction(jsonModuleID, "nullValue", value))
	addStandardExport(module, standardFunction(jsonModuleID, "fromBool", value, types.Parameter{Name: "value", Type: types.Bool}))
	addStandardExport(module, standardFunction(jsonModuleID, "fromInt", value, types.Parameter{Name: "value", Type: types.Int}))
	addStandardExport(module, standardFunction(jsonModuleID, "fromReal", value, types.Parameter{Name: "value", Type: types.Real}))
	addStandardExport(module, standardFunction(jsonModuleID, "fromString", value, types.Parameter{Name: "value", Type: types.String}))
	addStandardExport(module, standardFunction(jsonModuleID, "array", value, types.Parameter{Name: "values", Type: array}))
	addStandardExport(module, standardFunction(jsonModuleID, "object", value, types.Parameter{Name: "values", Type: object}))
	addStandardExport(module, standardFunction(jsonModuleID, "stringify", types.String,
		types.Parameter{Name: "value", Type: value}, pretty))
	addStandardExport(module, standardFunction(jsonModuleID, "write", types.Nothing,
		types.Parameter{Name: "value", Type: value}, stringParameter("path"), pretty))

	sort.Strings(module.ExportNames)
	return module
}

// jsonConstructionHint names the JSON functions that produce a JSONValue, so
// direct construction has an actionable message instead of a generic missing
// constructor diagnostic.
func jsonConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != jsonModuleID || identity.Name != "JSONValue" {
		return "", false
	}
	return "create a JSONValue with JSON.parse(source), JSON.read(path), JSON.nullValue(), " +
		"JSON.fromBool/fromInt/fromReal/fromString(value), JSON.array(values), or JSON.object(values)", true
}

// jsonOperationShape is the fixed call shape of one JSONValue member.
// resultNullable marks get, whose absence of a key is a genuine, statically
// expected possibility rather than an error.
type jsonOperationShape struct {
	parameters     []types.Type
	result         types.Type
	resultNullable bool
	hint           string
}

func jsonOperationShapes() map[TypeOperation]jsonOperationShape {
	none := []types.Type{}
	value := jsonValueType()
	return map[TypeOperation]jsonOperationShape{
		JSONValueKind:   {none, types.String, false, "call kind with no argument"},
		JSONValueIsNull: {none, types.Bool, false, "call isNull with no argument"},
		JSONValueBool:   {none, types.Bool, false, "call bool with no argument"},
		JSONValueInt:    {none, types.Int, false, "call int with no argument"},
		JSONValueReal:   {none, types.Real, false, "call real with no argument"},
		JSONValueString: {none, types.String, false, "call string with no argument"},
		JSONValueArray:  {none, types.List{Element: value}, false, "call array with no argument"},
		JSONValueObject: {none, types.Pair{Key: types.String, Value: value}, false, "call object with no argument"},
		JSONValueGet:    {[]types.Type{types.String}, value, true, "pass one String key"},
		JSONValueAt:     {[]types.Type{types.Int}, value, false, "pass one Int index"},
	}
}

var jsonOperationNames = map[string]TypeOperation{
	"kind": JSONValueKind, "isNull": JSONValueIsNull, "bool": JSONValueBool,
	"int": JSONValueInt, "real": JSONValueReal, "string": JSONValueString,
	"array": JSONValueArray, "object": JSONValueObject, "get": JSONValueGet, "at": JSONValueAt,
}

// jsonOperationFor names the built-in member a JSONValue instance publishes.
// Only the compiler-supplied JSONValue identity matches, so a user Class
// named JSONValue never collides with it.
func jsonOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil ||
		class.Symbol.ModuleID != jsonModuleID || class.Symbol.Name != "JSONValue" {
		return "", false
	}
	operation, known := jsonOperationNames[name]
	return operation, known
}

// analyzeJSONOperation checks one JSONValue member, following the same shape
// convention as analyzeRegexOperation: every argument is a NonNull value of
// the declared type, and only get's result is statically MaybeNull.
func (a *analyzer) analyzeJSONOperation(call *ast.CallExpr, operation TypeOperation, shape jsonOperationShape, current *scope, flow flowState) expressionInfo {
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
