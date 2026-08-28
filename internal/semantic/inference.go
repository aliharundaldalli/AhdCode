package semantic

import (
	"fmt"
	"strings"

	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

type inferredParameter struct {
	name      string
	typeValue types.Type
}

type functionInference struct {
	parameters []inferredParameter
	arityKnown bool
	returnType types.Type
	returnSet  bool
	conflict   bool
	reported   bool
	fixed      *types.Signature
	// fixedNull is the concrete callable that fixed the signature. Its
	// null-state contract must survive inference so a callback call keeps the
	// null-state of the callable actually assigned to it.
	fixedNull  *Callable
	calls      []*ast.CallExpr
	owner      *types.Signature
	ownerIndex int
}

func newFunctionInference(owner *types.Signature, ownerIndex int) *functionInference {
	return &functionInference{owner: owner, ownerIndex: ownerIndex}
}

func (a *analyzer) trackInference(symbol *Symbol, current *scope) {
	if symbol == nil || symbol.inference == nil {
		return
	}
	if current != nil && current.callable != nil {
		current.callable.inferences = append(current.callable.inferences, symbol)
	} else {
		a.moduleInferences = append(a.moduleInferences, symbol)
	}
}

func (a *analyzer) constrainFunctionCall(symbol *Symbol, call *ast.CallExpr, arguments []expressionInfo, expected types.Type) expressionInfo {
	inference := symbol.inference
	if inference == nil {
		inference = newFunctionInference(nil, -1)
		symbol.inference = inference
	}
	if inference.fixed != nil {
		callable := inferredCallable(inference)
		a.validateCallArguments(call, callable, arguments)
		a.result.SelectedCallables[call] = callable
		return expressionInfo{typeValue: inference.fixed.Return, nullState: callable.ReturnNull, symbol: symbol}
	}
	if !inference.arityKnown {
		inference.parameters = make([]inferredParameter, len(arguments))
		inference.arityKnown = true
		for index, argument := range arguments {
			inference.parameters[index] = inferredParameter{name: call.Arguments[index].Name, typeValue: argument.typeValue}
		}
	} else if len(inference.parameters) != len(arguments) {
		a.inferenceConflict(symbol, call.Span(), fmt.Sprintf("calls use both %d and %d arguments", len(inference.parameters), len(arguments)))
	} else {
		for index, argument := range arguments {
			parameterIndex := index
			if call.Arguments[index].Name != "" {
				parameterIndex = inferredParameterIndex(inference.parameters, call.Arguments[index].Name)
				if parameterIndex < 0 {
					a.inferenceConflict(symbol, call.Arguments[index].Span(), fmt.Sprintf("named argument %q conflicts with earlier calls", call.Arguments[index].Name))
					continue
				}
			}
			merged, ok := mergeParameterConstraint(inference.parameters[parameterIndex].typeValue, argument.typeValue)
			if !ok {
				a.inferenceConflict(symbol, call.Arguments[index].Span(), fmt.Sprintf("parameter %d receives incompatible %s and %s values", parameterIndex+1, types.Display(inference.parameters[parameterIndex].typeValue), types.Display(argument.typeValue)))
				continue
			}
			inference.parameters[parameterIndex].typeValue = merged
		}
	}
	if expected != nil && !types.IsInvalid(expected) {
		if !inference.returnSet {
			inference.returnType = expected
			inference.returnSet = true
		} else if merged, ok := mergeReturnConstraint(inference.returnType, expected); ok {
			inference.returnType = merged
		} else {
			a.inferenceConflict(symbol, call.Span(), fmt.Sprintf("return value is required as both %s and %s", types.Display(inference.returnType), types.Display(expected)))
		}
	}
	inference.calls = append(inference.calls, call)
	resultType := expected
	if resultType == nil {
		resultType = types.Invalid
	}
	return expressionInfo{typeValue: resultType, nullState: NonNull, symbol: symbol}
}

func inferredParameterIndex(parameters []inferredParameter, name string) int {
	for index, parameter := range parameters {
		if parameter.name == name {
			return index
		}
	}
	return -1
}

func mergeParameterConstraint(current, next types.Type) (types.Type, bool) {
	if types.IsInvalid(current) || types.IsInvalid(next) {
		return types.Invalid, false
	}
	if types.Equal(current, next) {
		return current, true
	}
	if types.Assignable(current, next) {
		return current, true
	}
	if types.Assignable(next, current) {
		return next, true
	}
	return types.Invalid, false
}

func mergeReturnConstraint(current, next types.Type) (types.Type, bool) {
	if types.IsInvalid(current) || types.IsInvalid(next) {
		return types.Invalid, false
	}
	if types.Equal(current, next) {
		return current, true
	}
	if types.Assignable(current, next) {
		return next, true
	}
	if types.Assignable(next, current) {
		return current, true
	}
	return types.Invalid, false
}

// inferredCallable rebuilds the callable contract of an inferred Function
// binding, preserving the null-state of the concrete callable assigned to it.
func inferredCallable(inference *functionInference) *Callable {
	callable := &Callable{
		Signature: inference.fixed, ReturnNull: NonNull,
		ParameterNull: nonNullParameters(len(inference.fixed.Parameters)),
	}
	if inference.fixedNull == nil {
		return callable
	}
	callable.ReturnNull = inference.fixedNull.ReturnNull
	for index := range callable.ParameterNull {
		if index < len(inference.fixedNull.ParameterNull) {
			callable.ParameterNull[index] = inference.fixedNull.ParameterNull[index]
		}
	}
	return callable
}

func (a *analyzer) constrainConcreteFunction(symbol *Symbol, signature *types.Signature, concrete *Callable, span source.Span) {
	if symbol == nil || signature == nil {
		return
	}
	if symbol.inference == nil {
		symbol.inference = newFunctionInference(nil, -1)
	}
	inference := symbol.inference
	if inference.fixed != nil {
		if !types.Equal(types.Function{Signature: inference.fixed}, types.Function{Signature: signature}) {
			a.inferenceConflict(symbol, span, fmt.Sprintf("assigned signatures %s and %s", formatSignature(inference.fixed), formatSignature(signature)))
		}
		return
	}
	if inference.arityKnown {
		if len(inference.parameters) != len(signature.Parameters) {
			a.inferenceConflict(symbol, span, "assigned Function has a different parameter count")
			return
		}
		for index, parameter := range signature.Parameters {
			if !types.Assignable(parameter.Type, inference.parameters[index].typeValue) {
				a.inferenceConflict(symbol, span, fmt.Sprintf("assigned parameter %d cannot accept inferred %s", index+1, types.Display(inference.parameters[index].typeValue)))
				return
			}
		}
	}
	if inference.returnSet && !types.Assignable(inference.returnType, signature.Return) {
		a.inferenceConflict(symbol, span, fmt.Sprintf("assigned return %s cannot satisfy expected %s", types.Display(signature.Return), types.Display(inference.returnType)))
		return
	}
	inference.fixed = signature
	inference.fixedNull = concrete
}

// concreteCallable extracts the callable contract behind a Function-typed
// expression, so its null-state survives assignment into an inferred binding.
func concreteCallable(info expressionInfo) *Callable {
	if info.symbol == nil {
		return nil
	}
	symbol := info.symbol
	if symbol.Alias != nil {
		symbol = symbol.Alias
	}
	return symbol.Callable
}

func (a *analyzer) inferenceConflict(symbol *Symbol, span source.Span, detail string) {
	if symbol.inference != nil {
		symbol.inference.conflict = true
		if symbol.inference.reported {
			return
		}
		symbol.inference.reported = true
	}
	a.error(codeConflictingFunction, fmt.Sprintf("conflicting Function constraints for %q: %s", symbol.Name, detail), span, "make every use resolve to one concrete callable signature")
}

func (a *analyzer) finalizeInferences(symbols []*Symbol) {
	seen := make(map[*Symbol]bool)
	for _, symbol := range symbols {
		if symbol == nil || seen[symbol] || symbol.inference == nil {
			continue
		}
		seen[symbol] = true
		inference := symbol.inference
		if inference.conflict {
			continue
		}
		var signature *types.Signature
		if inference.fixed != nil {
			signature = inference.fixed
		} else {
			complete := inference.arityKnown && inference.returnSet
			parameters := make([]types.Parameter, len(inference.parameters))
			for index, parameter := range inference.parameters {
				if types.IsInvalid(parameter.typeValue) || !functionTypeComplete(parameter.typeValue) {
					complete = false
				}
				parameters[index] = types.Parameter{Name: parameter.name, Type: parameter.typeValue}
			}
			if !complete || !functionTypeComplete(inference.returnType) {
				a.error(codeFunctionInference, fmt.Sprintf("Function signature for %q cannot be inferred", symbol.Name), symbol.Span, "use the Function in calls with fully known argument and expected result types, or assign one concrete Function")
				continue
			}
			signature = &types.Signature{Parameters: parameters, Return: inference.returnType}
		}
		symbol.Type = types.Function{Signature: signature}
		concrete := &Callable{Signature: signature, ParameterNull: nonNullParameters(len(signature.Parameters)), ReturnNull: NonNull}
		if inference.fixed != nil {
			concrete = inferredCallable(inference)
		}
		if symbol.Callable == nil {
			symbol.Callable = concrete
		} else {
			symbol.Callable.Signature = signature
			symbol.Callable.ParameterNull = concrete.ParameterNull
			symbol.Callable.ReturnNull = concrete.ReturnNull
		}
		if inference.owner != nil && inference.ownerIndex >= 0 && inference.ownerIndex < len(inference.owner.Parameters) {
			inference.owner.Parameters[inference.ownerIndex].Type = symbol.Type
		}
		for _, call := range inference.calls {
			a.result.ExpressionTypes[call] = signature.Return
			a.result.NullStates[call] = symbol.Callable.ReturnNull
			a.result.SelectedCallables[call] = symbol.Callable
		}
	}
}

func functionTypeComplete(value types.Type) bool {
	if value == nil || types.IsInvalid(value) {
		return false
	}
	function, ok := value.(types.Function)
	return !ok || function.Signature != nil
}

func nonNullParameters(count int) []NullState {
	states := make([]NullState, count)
	for index := range states {
		states[index] = NonNull
	}
	return states
}

func formatSignature(signature *types.Signature) string {
	if signature == nil {
		return "Function<?>"
	}
	parameters := make([]string, len(signature.Parameters))
	for index, parameter := range signature.Parameters {
		prefix := ""
		if parameter.Name != "" {
			prefix = parameter.Name + ": "
		}
		parameters[index] = prefix + types.Display(parameter.Type)
		if parameter.HasDefault {
			parameters[index] += " := default"
		}
	}
	return fmt.Sprintf("(%s) -> %s", strings.Join(parameters, ", "), types.Display(signature.Return))
}
