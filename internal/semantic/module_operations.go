package semantic

import (
	"fmt"

	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// Type-directed standard module operations.
//
// A standard module function normally publishes one fixed types.Signature
// through standardFunction, and the ordinary call path checks it. A few
// operations cannot be described that way: Lists.chunk(List<Int>, Int) and
// Lists.chunk(List<String>, Int) have different, equally exact result types,
// and neither an erased element type nor a family of per-element overloads
// would preserve them.
//
// This file is the smallest mechanism that closes that gap. A ModuleOperation
// symbol carries no signature at all; instead analyzeModuleOperation reads the
// statically known argument types at one call site, checks them, and records
// the exact concrete signature that call selected in SelectedCallables. Every
// later layer -- lowering, the native backend, the evaluator -- reads that one
// specialized signature, so no stage ever has to guess a type, and no Any,
// dynamic, or erased Object representation exists anywhere in the pipeline.
//
// The mechanism is deliberately module-agnostic: a future standard module adds
// its operations by registering shapes and one analyze...Operation dispatcher,
// exactly the way the receiver-directed TypeOperation modules already do.

// moduleOperationShape is the fixed part of one type-directed operation: how
// many arguments it takes, and what a caller should have written instead. The
// argument *types* are not fixed, so they are absent here on purpose.
type moduleOperationShape struct {
	arity int
	hint  string
}

// moduleOperationShapeOf finds one operation's call shape. Each standard
// module owns its own table, the same way each TypeOperation module owns its
// own operation shapes.
func moduleOperationShapeOf(operation ModuleOperation) moduleOperationShape {
	return listsOperationShapes[operation]
}

// moduleOperationSymbol builds the module-root symbol of one type-directed
// standard module function. It deliberately carries no types.Signature: the
// operation has no single one, and analyzeModuleOperation computes the exact
// concrete signature of every individual call site instead.
func moduleOperationSymbol(moduleID, name string, operation ModuleOperation) *Symbol {
	return &Symbol{
		Name: name, Kind: FunctionSymbol, Type: types.Function{},
		ModuleRoot: true, Builtin: true, InitialNull: NonNull,
		OriginModuleID: moduleID, ModuleOperation: operation,
	}
}

// standardErrorClass builds one standard module's catchable Error subclass and
// registers it under the module's canonical Class identity.
func standardErrorClass(moduleID, name string, identity *types.ClassSymbol, module *ModuleInterface) *Symbol {
	symbol := &Symbol{
		Name: name, Kind: ClassSymbol, Class: identity,
		Type: types.Class{Symbol: identity, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: moduleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[moduleID+"\x00"+name] = symbol
	return symbol
}

// moduleOperationOf names the type-directed operation a resolved symbol
// denotes, following an explicit Global alias to its module-root binding.
func moduleOperationOf(symbol *Symbol) (ModuleOperation, bool) {
	if symbol == nil {
		return "", false
	}
	if symbol.Alias != nil {
		symbol = symbol.Alias
	}
	if symbol.ModuleOperation == "" {
		return "", false
	}
	return symbol.ModuleOperation, true
}

// rejectModuleOperationValue reports the one deliberate boundary of this
// design: a type-directed operation is specialized against the arguments at
// its call site, so it has no single concrete Function type and cannot be
// stored as an unspecialized Function value. Supporting that would require
// either a dynamic polymorphic Function value or user-facing generic Function
// syntax; AhdCode has neither, so this stays a compile-time diagnostic.
func (a *analyzer) rejectModuleOperationValue(operation ModuleOperation, span source.Span) {
	a.error(codeInvalidType,
		fmt.Sprintf("%s is a type-directed module operation and has no Function value", operation),
		span, fmt.Sprintf("call %s(...) directly; its result type is computed from its arguments, "+
			"so it cannot be bound to a Function", operation))
}

// analyzeModuleOperation type-checks one call of a type-directed standard
// module function and reports its exact concrete result type. Every branch
// derives that result from the statically known argument types, so the
// compiler never falls back on an erased element or value type.
func (a *analyzer) analyzeModuleOperation(call *ast.CallExpr, operation ModuleOperation, current *scope, flow flowState) expressionInfo {
	shape := moduleOperationShapeOf(operation)
	for _, argument := range call.Arguments {
		if argument.Name != "" {
			// A compiler-supplied operation publishes no parameter names, the
			// same call shape the built-in type operations use.
			a.error(codeCallArguments, fmt.Sprintf("%s does not accept a named argument", operation), argument.Span(), shape.hint)
			a.analyzeTypeOperationArguments(call, current, flow, nil)
			return moduleOperationFailure()
		}
	}
	if len(call.Arguments) != shape.arity {
		a.error(codeCallArguments,
			fmt.Sprintf("%s expects %d argument(s); received %d", operation, shape.arity, len(call.Arguments)),
			call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return moduleOperationFailure()
	}
	if info, handled := a.analyzeListsOperation(call, operation, current, flow); handled {
		return info
	}
	return moduleOperationFailure()
}

// moduleOperationFailure is the recovery result of a rejected call. It has no
// statically known type: unlike a fixed-signature function, a type-directed
// operation's result exists only once its arguments type-check.
func moduleOperationFailure() expressionInfo {
	return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}
}

// moduleOperationResult records the exact specialized signature this call site
// selected and reports its result. Lowering reads that signature to give each
// argument its expected type, so the two layers agree on one concrete shape.
func (a *analyzer) moduleOperationResult(call *ast.CallExpr, operation ModuleOperation, result types.Type, parameters []types.Parameter, parameterNull []NullState) expressionInfo {
	a.result.SelectedCallables[call] = &Callable{
		Signature:     &types.Signature{Parameters: parameters, Return: result},
		ParameterNull: parameterNull, ReturnNull: NonNull,
	}
	a.result.ModuleOperations[call] = operation
	return expressionInfo{typeValue: result, nullState: NonNull}
}

// moduleOperandList analyzes one argument and requires a NonNull List value.
func (a *analyzer) moduleOperandList(call *ast.CallExpr, index int, operation ModuleOperation, current *scope, flow flowState) (types.List, bool) {
	info := a.analyzeExpression(call.Arguments[index].Value, current, flow)
	if info.invalid() {
		return types.List{}, false
	}
	if info.nullState != NonNull {
		a.nullableError(string(operation), call.Arguments[index].Value, info.nullState)
		return types.List{}, false
	}
	list, ok := info.typeValue.(types.List)
	if !ok {
		a.error(codeTypeMismatch,
			fmt.Sprintf("%s expects a List; received %s", operation, types.Display(info.typeValue)),
			call.Arguments[index].Span(), moduleOperationShapeOf(operation).hint)
		return types.List{}, false
	}
	return list, true
}

// moduleOperandPair analyzes one argument and requires a NonNull Pair value.
func (a *analyzer) moduleOperandPair(call *ast.CallExpr, index int, operation ModuleOperation, current *scope, flow flowState) (types.Pair, bool) {
	info := a.analyzeExpression(call.Arguments[index].Value, current, flow)
	if info.invalid() {
		return types.Pair{}, false
	}
	if info.nullState != NonNull {
		a.nullableError(string(operation), call.Arguments[index].Value, info.nullState)
		return types.Pair{}, false
	}
	pair, ok := info.typeValue.(types.Pair)
	if !ok {
		a.error(codeTypeMismatch,
			fmt.Sprintf("%s expects a Pair; received %s", operation, types.Display(info.typeValue)),
			call.Arguments[index].Span(), moduleOperationShapeOf(operation).hint)
		return types.Pair{}, false
	}
	return pair, true
}

// moduleOperandValue checks one argument against an already-specialized
// expected type that does not accept null.
func (a *analyzer) moduleOperandValue(call *ast.CallExpr, index int, operation ModuleOperation, expected types.Type, subject string, current *scope, flow flowState) bool {
	return a.moduleOperandNullableValue(call, index, operation, expected, false, subject, current, flow)
}

// moduleOperandNullableValue checks one argument against an already-specialized
// expected type. nullable states whether the target slot legally holds null,
// which is exactly the collection's own structural nullability.
func (a *analyzer) moduleOperandNullableValue(call *ast.CallExpr, index int, operation ModuleOperation, expected types.Type, nullable bool, subject string, current *scope, flow flowState) bool {
	info := a.analyzeExpressionExpected(call.Arguments[index].Value, current, flow, expected)
	if info.invalid() {
		return false
	}
	if !a.requireCompatibleNull(nullable, info, call.Arguments[index].Span(), string(operation)+" "+subject) {
		return false
	}
	if info.nullState == Null {
		return true
	}
	if !types.Assignable(expected, info.typeValue) {
		a.typeMismatch(call.Arguments[index].Span(), expected, info.typeValue, string(operation)+" "+subject)
		return false
	}
	return true
}

// analyzeModuleCallback checks one Function argument of a type-directed
// operation. The callback must take exactly the collection's element or value
// type -- including its nullability -- because List and Pair are invariant and
// no value is converted or null-checked on the way into the call. That last
// part is stricter than the older List.map contract on purpose: these
// operations turn callback results into Pair keys and Pair values, where a
// null that the callback never declared would have no sound representation.
//
// It reads the callback from any argument position and leaves the return type
// free, so each operation can decide what its result type means.
func (a *analyzer) analyzeModuleCallback(call *ast.CallExpr, index int, operation ModuleOperation, parameter types.Type, parameterNullable bool, current *scope, flow flowState) (*types.Signature, bool, bool) {
	var expected types.Type
	if !types.IsInvalid(parameter) {
		expected = types.Function{Signature: &types.Signature{
			Parameters: []types.Parameter{{Name: "value", Type: parameter}},
		}}
	}
	reported := a.bag.Len()
	info := a.analyzeExpressionExpected(call.Arguments[index].Value, current, flow, expected)
	if info.invalid() || a.bag.Len() != reported {
		// The argument already reported its own incompatibility; a second
		// derivative diagnostic about the same callback adds nothing.
		return nil, false, false
	}
	if info.nullState != NonNull {
		a.nullableError(string(operation), call.Arguments[index].Value, info.nullState)
		return nil, false, false
	}
	function, isFunction := info.typeValue.(types.Function)
	if !isFunction || function.Signature == nil {
		a.typeMismatch(call.Arguments[index].Span(), types.Function{}, info.typeValue, string(operation)+" argument")
		return nil, false, false
	}
	signature := function.Signature
	provided := a.result.SelectedFunctionValues[call.Arguments[index].Value]
	if provided == nil {
		provided = concreteCallable(info)
	}
	if len(signature.Parameters) != 1 || !types.Equal(signature.Parameters[0].Type, parameter) ||
		!declaresParameterNull(provided, parameterNullable) {
		a.error(codeCallArguments,
			fmt.Sprintf("%s requires a Function taking exactly one %s", operation,
				nullableDisplay(parameter, parameterNullable)),
			call.Arguments[index].Span(), moduleOperationShapeOf(operation).hint)
		return nil, false, false
	}
	return signature, true, a.callbackReturnsNull(call.Arguments[index].Value, info)
}

// declaresParameterNull reports whether the selected callback's single
// parameter is declared with exactly the nullability the collection slot has.
// A callable with no recorded null metadata is accepted, because that only
// happens at an external interface boundary where the type check already
// matched.
func declaresParameterNull(callable *Callable, nullable bool) bool {
	if callable == nil || len(callable.ParameterNull) != 1 {
		return true
	}
	return (callable.ParameterNull[0] != NonNull) == nullable
}

// nullableDisplay renders one type with the structural nullability marker
// AhdCode source writes, which types.Display deliberately omits.
func nullableDisplay(value types.Type, nullable bool) string {
	if nullable {
		return types.Display(value) + "?"
	}
	return types.Display(value)
}

// requireModulePairKey enforces the existing v0.1 Pair key rules where an
// operation turns values into keys. Keys stay restricted to the stable scalar
// types and are never null, so nothing is stringified to make it fit.
func (a *analyzer) requireModulePairKey(value types.Type, nullable bool, operation ModuleOperation, subject string, span source.Span) bool {
	if types.IsInvalid(value) {
		return false
	}
	if !types.IsPairKey(value) {
		a.error(codeInvalidPairKey,
			fmt.Sprintf("%s requires a Pair key type; the %s is %s", operation, subject, types.Display(value)),
			span, "a Pair key is String, Int, or Bool")
		return false
	}
	if nullable {
		a.error(codeInvalidPairKey,
			fmt.Sprintf("%s requires a non-null Pair key; the %s is %s?", operation, subject, types.Display(value)),
			span, "a Pair key is never null; remove the trailing ? or refine the value first")
		return false
	}
	return true
}
