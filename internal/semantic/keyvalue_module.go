package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const keyValueModuleID = "builtin:KeyValue"

var keyValueErrorClass = &types.ClassSymbol{
	ModuleID: keyValueModuleID, Name: "KeyValueError",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}},
}

// KeyValueErrorIdentity exposes the standard module's catchable error identity
// to lowering without coupling the public module interface to a backend.
func KeyValueErrorIdentity() *types.ClassSymbol { return keyValueErrorClass }

// keyValueOperationNames is the module's published surface. KeyValue adds no
// container of its own: it transforms the existing ordered, homogeneous
// Pair<K, V> and always hands back a Pair, a List, or nothing at all.
var keyValueOperationNames = map[string]ModuleOperation{
	"keys": KeyValueKeys, "values": KeyValueValues, "combine": KeyValueCombine,
	"with": KeyValueWith, "without": KeyValueWithout, "select": KeyValueSelect,
	"drop": KeyValueDrop, "rename": KeyValueRename, "mapValues": KeyValueMapValues,
	"merge": KeyValueMerge, "overlay": KeyValueOverlay,
}

var keyValueOperationShapes = map[ModuleOperation]moduleOperationShape{
	KeyValueKeys:      {1, "pass one Pair"},
	KeyValueValues:    {1, "pass one Pair"},
	KeyValueCombine:   {2, "pass a key List and a value List of exactly equal length"},
	KeyValueWith:      {3, "pass one Pair, one key, and one value"},
	KeyValueWithout:   {2, "pass one Pair and one existing key"},
	KeyValueSelect:    {2, "pass one Pair and a List of the keys to keep"},
	KeyValueDrop:      {2, "pass one Pair and a List of the keys to remove"},
	KeyValueRename:    {3, "pass one Pair, the existing key, and its new key"},
	KeyValueMapValues: {2, "pass one Pair and a Function taking its value type"},
	KeyValueMerge:     {2, "pass two Pairs of exactly the same type with no shared key"},
	KeyValueOverlay:   {2, "pass a base Pair and a changes Pair of exactly the same type"},
}

func keyValueModuleInterface() *ModuleInterface {
	module := standardInterface(keyValueModuleID, "KeyValue")
	addStandardExport(module, standardErrorClass(keyValueModuleID, "KeyValueError", keyValueErrorClass, module))
	for name, operation := range keyValueOperationNames {
		addStandardExport(module, moduleOperationSymbol(keyValueModuleID, name, operation))
	}
	sort.Strings(module.ExportNames)
	return module
}

// analyzeKeyValueOperation type-checks one KeyValue call, reporting false for
// an operation belonging to another module.
func (a *analyzer) analyzeKeyValueOperation(call *ast.CallExpr, operation ModuleOperation, current *scope, flow flowState) (expressionInfo, bool) {
	switch operation {
	case KeyValueKeys, KeyValueValues:
		return a.analyzeKeyValueProjection(call, operation, current, flow), true
	case KeyValueCombine:
		return a.analyzeKeyValueCombine(call, current, flow), true
	case KeyValueWith:
		return a.analyzeKeyValueWith(call, current, flow), true
	case KeyValueWithout:
		return a.analyzeKeyValueWithout(call, current, flow), true
	case KeyValueSelect, KeyValueDrop:
		return a.analyzeKeyValueKeyList(call, operation, current, flow), true
	case KeyValueRename:
		return a.analyzeKeyValueRename(call, current, flow), true
	case KeyValueMapValues:
		return a.analyzeKeyValueMapValues(call, current, flow), true
	case KeyValueMerge, KeyValueOverlay:
		return a.analyzeKeyValueCombination(call, operation, current, flow), true
	}
	return expressionInfo{}, false
}

// analyzeKeyValueProjection gives keys: Pair<K, V> -> List<K> and values:
// Pair<K, V> -> List<V>. Keys are never null; a value's nullability is the
// Pair's own structural value nullability, and it is preserved exactly.
func (a *analyzer) analyzeKeyValueProjection(call *ast.CallExpr, operation ModuleOperation, current *scope, flow flowState) expressionInfo {
	pair, ok := a.moduleOperandPair(call, 0, operation, current, flow)
	if !ok {
		return moduleOperationFailure()
	}
	result := types.List{Element: pair.Key, ElementNullable: false}
	if operation == KeyValueValues {
		result = types.List{Element: pair.Value, ElementNullable: pair.ValueNullable}
	}
	return a.moduleOperationResult(call, operation, result,
		[]types.Parameter{{Name: "pair", Type: pair}}, []NullState{NonNull})
}

func (a *analyzer) analyzeKeyValueCombine(call *ast.CallExpr, current *scope, flow flowState) expressionInfo {
	keys, keysOK := a.moduleOperandList(call, 0, KeyValueCombine, current, flow)
	values, valuesOK := a.moduleOperandList(call, 1, KeyValueCombine, current, flow)
	if !keysOK || !valuesOK {
		return moduleOperationFailure()
	}
	if !a.requireModulePairKey(keys.Element, keys.ElementNullable, KeyValueCombine, "key element", call.Arguments[0].Span()) {
		return moduleOperationFailure()
	}
	return a.moduleOperationResult(call, KeyValueCombine,
		types.Pair{Key: keys.Element, Value: values.Element, ValueNullable: values.ElementNullable},
		[]types.Parameter{{Name: "keys", Type: keys}, {Name: "values", Type: values}},
		[]NullState{NonNull, NonNull})
}

func (a *analyzer) analyzeKeyValueWith(call *ast.CallExpr, current *scope, flow flowState) expressionInfo {
	pair, ok := a.moduleOperandPair(call, 0, KeyValueWith, current, flow)
	if !ok {
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return moduleOperationFailure()
	}
	keyOK := a.moduleOperandValue(call, 1, KeyValueWith, pair.Key, "key", current, flow)
	valueOK := a.moduleOperandNullableValue(call, 2, KeyValueWith, pair.Value, pair.ValueNullable, "value", current, flow)
	if !keyOK || !valueOK {
		return moduleOperationFailure()
	}
	valueNull := NonNull
	if pair.ValueNullable {
		valueNull = MaybeNull
	}
	return a.moduleOperationResult(call, KeyValueWith, pair,
		[]types.Parameter{{Name: "pair", Type: pair}, {Name: "key", Type: pair.Key}, {Name: "value", Type: pair.Value}},
		[]NullState{NonNull, NonNull, valueNull})
}

func (a *analyzer) analyzeKeyValueWithout(call *ast.CallExpr, current *scope, flow flowState) expressionInfo {
	pair, ok := a.moduleOperandPair(call, 0, KeyValueWithout, current, flow)
	if !ok {
		a.analyzeExpression(call.Arguments[1].Value, current, flow)
		return moduleOperationFailure()
	}
	if !a.moduleOperandValue(call, 1, KeyValueWithout, pair.Key, "key", current, flow) {
		return moduleOperationFailure()
	}
	return a.moduleOperationResult(call, KeyValueWithout, pair,
		[]types.Parameter{{Name: "pair", Type: pair}, {Name: "key", Type: pair.Key}},
		[]NullState{NonNull, NonNull})
}

// analyzeKeyValueKeyList checks select and drop. Both take a List of the
// Pair's own key type, which List invariance pins exactly.
func (a *analyzer) analyzeKeyValueKeyList(call *ast.CallExpr, operation ModuleOperation, current *scope, flow flowState) expressionInfo {
	pair, ok := a.moduleOperandPair(call, 0, operation, current, flow)
	if !ok {
		a.analyzeExpression(call.Arguments[1].Value, current, flow)
		return moduleOperationFailure()
	}
	expected := types.List{Element: pair.Key, ElementNullable: false}
	if !a.moduleOperandValue(call, 1, operation, expected, "key List", current, flow) {
		return moduleOperationFailure()
	}
	return a.moduleOperationResult(call, operation, pair,
		[]types.Parameter{{Name: "pair", Type: pair}, {Name: "keys", Type: expected}},
		[]NullState{NonNull, NonNull})
}

func (a *analyzer) analyzeKeyValueRename(call *ast.CallExpr, current *scope, flow flowState) expressionInfo {
	pair, ok := a.moduleOperandPair(call, 0, KeyValueRename, current, flow)
	if !ok {
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return moduleOperationFailure()
	}
	oldOK := a.moduleOperandValue(call, 1, KeyValueRename, pair.Key, "oldKey", current, flow)
	newOK := a.moduleOperandValue(call, 2, KeyValueRename, pair.Key, "newKey", current, flow)
	if !oldOK || !newOK {
		return moduleOperationFailure()
	}
	return a.moduleOperationResult(call, KeyValueRename, pair,
		[]types.Parameter{{Name: "pair", Type: pair}, {Name: "oldKey", Type: pair.Key}, {Name: "newKey", Type: pair.Key}},
		[]NullState{NonNull, NonNull, NonNull})
}

// analyzeKeyValueMapValues gives Pair<K, V> -> Pair<K, U>, where U is the
// callback's own return type. A callback that may return null produces
// Pair<K, U?>, so nullability is transformed rather than erased.
func (a *analyzer) analyzeKeyValueMapValues(call *ast.CallExpr, current *scope, flow flowState) expressionInfo {
	pair, ok := a.moduleOperandPair(call, 0, KeyValueMapValues, current, flow)
	if !ok {
		a.analyzeExpression(call.Arguments[1].Value, current, flow)
		return moduleOperationFailure()
	}
	signature, callbackOK, returnsNull := a.analyzeModuleCallback(call, 1, KeyValueMapValues, pair.Value, pair.ValueNullable, current, flow)
	if !callbackOK {
		return moduleOperationFailure()
	}
	if signature.Return.Kind() == types.NothingKind {
		a.error(codeCallArguments, fmt.Sprintf("%s requires a Function that returns a value", KeyValueMapValues),
			call.Arguments[1].Span(), "return a value from the transform Function")
		return moduleOperationFailure()
	}
	callback := types.Function{Signature: signature}
	return a.moduleOperationResult(call, KeyValueMapValues,
		types.Pair{Key: pair.Key, Value: signature.Return, ValueNullable: returnsNull},
		[]types.Parameter{{Name: "pair", Type: pair}, {Name: "transform", Type: callback}},
		[]NullState{NonNull, NonNull})
}

// analyzeKeyValueCombination checks merge and overlay. Pair is invariant, so
// both operands must have exactly the same type, including value nullability:
// a Pair<String, Int> never silently becomes a Pair<String, Real> because it
// met one.
func (a *analyzer) analyzeKeyValueCombination(call *ast.CallExpr, operation ModuleOperation, current *scope, flow flowState) expressionInfo {
	left, ok := a.moduleOperandPair(call, 0, operation, current, flow)
	if !ok {
		a.analyzeExpression(call.Arguments[1].Value, current, flow)
		return moduleOperationFailure()
	}
	first, second := "left", "right"
	if operation == KeyValueOverlay {
		first, second = "base", "changes"
	}
	if !a.moduleOperandValue(call, 1, operation, left, second, current, flow) {
		return moduleOperationFailure()
	}
	return a.moduleOperationResult(call, operation, left,
		[]types.Parameter{{Name: first, Type: left}, {Name: second, Type: left}},
		[]NullState{NonNull, NonNull})
}
