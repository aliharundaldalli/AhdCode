package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const listsModuleID = "builtin:Lists"

var listsErrorClass = &types.ClassSymbol{
	ModuleID: listsModuleID, Name: "ListsError",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}},
}

// ListsErrorIdentity exposes the standard module's catchable error identity to
// lowering without coupling the public module interface to a backend.
func ListsErrorIdentity() *types.ClassSymbol { return listsErrorClass }

// listsOperationNames is the module's published surface. Lists deliberately
// duplicates nothing the core List type already offers: add, eject, sort,
// reverse, shuffle, count, index, map, filter, slicing, and List + List stay
// where they are. Every operation here is a structural transformation that has
// no natural spelling as a List member.
var listsOperationNames = map[string]ModuleOperation{
	"chunk": ListsChunk, "flatten": ListsFlatten, "transpose": ListsTranspose,
	"unique": ListsUnique, "valueCounts": ListsValueCounts, "groupBy": ListsGroupBy,
}

var listsOperationShapes = map[ModuleOperation]moduleOperationShape{
	ListsChunk:       {2, "pass one List and an Int size greater than zero"},
	ListsFlatten:     {1, "pass one List<List<T>>"},
	ListsTranspose:   {1, "pass one rectangular List<List<T>>"},
	ListsUnique:      {1, "pass one List"},
	ListsValueCounts: {1, "pass one List<String>, List<Int>, or List<Bool>"},
	ListsGroupBy:     {2, "pass one List and a key Function returning String, Int, or Bool"},
}

func listsModuleInterface() *ModuleInterface {
	module := standardInterface(listsModuleID, "Lists")
	addStandardExport(module, standardErrorClass(listsModuleID, "ListsError", listsErrorClass, module))
	for name, operation := range listsOperationNames {
		addStandardExport(module, moduleOperationSymbol(listsModuleID, name, operation))
	}
	sort.Strings(module.ExportNames)
	return module
}

// analyzeListsOperation type-checks one Lists call, reporting false for an
// operation belonging to another module.
func (a *analyzer) analyzeListsOperation(call *ast.CallExpr, operation ModuleOperation, current *scope, flow flowState) (expressionInfo, bool) {
	switch operation {
	case ListsChunk:
		return a.analyzeListsChunk(call, current, flow), true
	case ListsFlatten, ListsTranspose:
		return a.analyzeListsNested(call, operation, current, flow), true
	case ListsUnique:
		return a.analyzeListsUnique(call, current, flow), true
	case ListsValueCounts:
		return a.analyzeListsValueCounts(call, current, flow), true
	case ListsGroupBy:
		return a.analyzeListsGroupBy(call, current, flow), true
	}
	return expressionInfo{}, false
}

// analyzeListsChunk gives List<T> -> List<List<T>>. The element type, with its
// nullability, is carried into the inner Lists unchanged; the outer List's own
// elements are Lists and are never null.
func (a *analyzer) analyzeListsChunk(call *ast.CallExpr, current *scope, flow flowState) expressionInfo {
	list, listOK := a.moduleOperandList(call, 0, ListsChunk, current, flow)
	sizeOK := a.moduleOperandValue(call, 1, ListsChunk, types.Int, "size", current, flow)
	if !listOK || !sizeOK {
		return moduleOperationFailure()
	}
	return a.moduleOperationResult(call, ListsChunk,
		types.List{Element: list, ElementNullable: false},
		[]types.Parameter{{Name: "values", Type: list}, {Name: "size", Type: types.Int}},
		[]NullState{NonNull, NonNull})
}

// analyzeListsNested checks flatten and transpose. Both read List<List<T>>:
// flatten reports the inner List type, transpose the outer one.
func (a *analyzer) analyzeListsNested(call *ast.CallExpr, operation ModuleOperation, current *scope, flow flowState) expressionInfo {
	outer, ok := a.moduleOperandList(call, 0, operation, current, flow)
	if !ok {
		return moduleOperationFailure()
	}
	inner, nested := outer.Element.(types.List)
	if !nested || outer.ElementNullable {
		// A nullable inner List has no defined contribution: skipping it would
		// silently drop data, and keeping it would need a null row.
		a.error(codeTypeMismatch,
			fmt.Sprintf("%s expects List<List<T>>; received %s", operation, types.Display(outer)),
			call.Arguments[0].Span(), moduleOperationShapeOf(operation).hint)
		return moduleOperationFailure()
	}
	result := types.Type(inner)
	if operation == ListsTranspose {
		result = outer
	}
	return a.moduleOperationResult(call, operation, result,
		[]types.Parameter{{Name: "rows", Type: outer}}, []NullState{NonNull})
}

func (a *analyzer) analyzeListsUnique(call *ast.CallExpr, current *scope, flow flowState) expressionInfo {
	list, ok := a.moduleOperandList(call, 0, ListsUnique, current, flow)
	if !ok {
		return moduleOperationFailure()
	}
	if !types.IsInvalid(list.Element) && !equatableElement(list.Element) {
		a.error(codeTypeMismatch,
			fmt.Sprintf("%s does not compare %s elements", ListsUnique, types.Display(list.Element)),
			call.Arguments[0].Span(), "unique keeps distinct elements using ==, which no Function value defines")
		return moduleOperationFailure()
	}
	return a.moduleOperationResult(call, ListsUnique, list,
		[]types.Parameter{{Name: "values", Type: list}}, []NullState{NonNull})
}

func (a *analyzer) analyzeListsValueCounts(call *ast.CallExpr, current *scope, flow flowState) expressionInfo {
	list, ok := a.moduleOperandList(call, 0, ListsValueCounts, current, flow)
	if !ok {
		return moduleOperationFailure()
	}
	if !a.requireModulePairKey(list.Element, list.ElementNullable, ListsValueCounts, "counted element", call.Arguments[0].Span()) {
		return moduleOperationFailure()
	}
	return a.moduleOperationResult(call, ListsValueCounts,
		types.Pair{Key: list.Element, Value: types.Int, ValueNullable: false},
		[]types.Parameter{{Name: "values", Type: list}}, []NullState{NonNull})
}

func (a *analyzer) analyzeListsGroupBy(call *ast.CallExpr, current *scope, flow flowState) expressionInfo {
	list, ok := a.moduleOperandList(call, 0, ListsGroupBy, current, flow)
	if !ok {
		a.analyzeExpression(call.Arguments[1].Value, current, flow)
		return moduleOperationFailure()
	}
	signature, callbackOK, returnsNull := a.analyzeModuleCallback(call, 1, ListsGroupBy, list.Element, list.ElementNullable, current, flow)
	if !callbackOK {
		return moduleOperationFailure()
	}
	if !a.requireModulePairKey(signature.Return, returnsNull, ListsGroupBy, "key Function result", call.Arguments[1].Span()) {
		return moduleOperationFailure()
	}
	callback := types.Function{Signature: signature}
	return a.moduleOperationResult(call, ListsGroupBy,
		types.Pair{Key: signature.Return, Value: list, ValueNullable: false},
		[]types.Parameter{{Name: "values", Type: list}, {Name: "key", Type: callback}},
		[]NullState{NonNull, NonNull})
}

// equatableElement reports whether ordinary AhdCode == compares two values of
// this type, which is exactly what unique needs to keep distinct elements.
func equatableElement(value types.Type) bool {
	if value == nil {
		return false
	}
	switch value.Kind() {
	case types.IntKind, types.RealKind, types.ComplexKind, types.StringKind, types.BoolKind,
		types.ListKind, types.PairKind, types.ClassKind:
		return true
	default:
		return false
	}
}
