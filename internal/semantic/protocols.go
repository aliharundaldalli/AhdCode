package semantic

import (
	"fmt"

	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// protocolShape is the required call shape of one Class Protocol Method.
// hasFixedReturn is false for the arithmetic protocols and CNegate, whose
// result type is simply the containing overload's declared return type.
type protocolShape struct {
	arity          int
	fixedReturn    types.Kind
	hasFixedReturn bool
}

// classProtocolShapes are the exact ten compiler-recognized Class Protocol
// Method names. Only these names carry protocol meaning inside a Class body;
// every other identifier, including one that merely starts with C, remains an
// ordinary member.
var classProtocolShapes = map[string]protocolShape{
	"CEqual":     {arity: 1, fixedReturn: types.BoolKind, hasFixedReturn: true},
	"CCompare":   {arity: 1, fixedReturn: types.IntKind, hasFixedReturn: true},
	"CAdd":       {arity: 1},
	"CSubtract":  {arity: 1},
	"CMultiply":  {arity: 1},
	"CDivide":    {arity: 1},
	"CRemainder": {arity: 1},
	"CPower":     {arity: 1},
	"CNegate":    {arity: 0},
	"CStr":       {arity: 0, fixedReturn: types.StringKind, hasFixedReturn: true},
}

// arithmeticProtocolNames maps each binary arithmetic operator to the Class
// Protocol Method that implements it for a Class receiver.
var arithmeticProtocolNames = map[string]string{
	"+": "CAdd", "-": "CSubtract", "*": "CMultiply",
	"/": "CDivide", "%": "CRemainder", "^": "CPower",
}

func isClassProtocolName(name string) bool {
	_, ok := classProtocolShapes[name]
	return ok
}

// validateProtocolSlot rejects a non-Function Class member that reuses one of
// the ten reserved protocol names, such as CAdd: Int := 5. It does not affect
// any other member name.
func (a *analyzer) validateProtocolSlot(name string, span source.Span) {
	if !isClassProtocolName(name) {
		return
	}
	a.error(codeProtocolSlot, fmt.Sprintf("%q is a reserved Class Protocol Method slot", name), span, fmt.Sprintf("declare %s as a Function with the required Class Protocol Method signature", name))
}

// validateProtocolSignature checks one Function declaration that occupies a
// reserved protocol slot against its required arity and return type. It runs
// once per declared overload, so every overload of an arithmetic protocol is
// checked independently.
func (a *analyzer) validateProtocolSignature(declaration *ast.FunctionDecl, callable *Callable) {
	shape, ok := classProtocolShapes[declaration.Name]
	if !ok || callable == nil || callable.Signature == nil {
		return
	}
	if len(callable.Signature.Parameters) != shape.arity {
		hint := fmt.Sprintf("declare %s with exactly %d explicit parameter(s)", declaration.Name, shape.arity)
		a.error(codeProtocolSignature, fmt.Sprintf("Class Protocol Method %q expects %d parameter(s); declared %d", declaration.Name, shape.arity, len(callable.Signature.Parameters)), declaration.Span(), hint)
	}
	if shape.hasFixedReturn && callable.Signature.Return.Kind() != shape.fixedReturn {
		hint := fmt.Sprintf("declare %s to return %s", declaration.Name, shape.fixedReturn)
		a.error(codeProtocolSignature, fmt.Sprintf("Class Protocol Method %q must return %s; declared %s", declaration.Name, shape.fixedReturn, types.Display(callable.Signature.Return)), declaration.Span(), hint)
	}
}

// resolveClassProtocol looks up one protocol name on a Class, including
// inherited members, and reports it only when it resolves to a Function. A
// malformed slot (already diagnosed at declaration time) is treated as
// unavailable rather than raised again here.
func (a *analyzer) resolveClassProtocol(classSymbol *Symbol, name string) *Symbol {
	resolved := a.lookupMember(classSymbol, name)
	if resolved == nil || resolved.Kind != FunctionSymbol {
		return nil
	}
	return resolved
}

// protocolCandidates lists the concrete callables of a resolved protocol
// Symbol, whether or not it was declared with Overload Function.
func protocolCandidates(resolved *Symbol) []*Callable {
	if resolved.OverloadSet != nil && len(resolved.OverloadSet.Candidates) > 0 {
		return resolved.OverloadSet.Candidates
	}
	if resolved.Callable != nil {
		return []*Callable{resolved.Callable}
	}
	return nil
}

// resolveProtocolNoArgs selects the zero-argument candidate of a resolved
// CNegate/CStr protocol. A valid declaration always has exactly one, since a
// second zero-argument overload would already be rejected as a duplicate
// overload signature at declaration time.
func resolveProtocolNoArgs(resolved *Symbol) *Callable {
	for _, candidate := range protocolCandidates(resolved) {
		if candidate != nil && candidate.Signature != nil && len(candidate.Signature.Parameters) == 0 {
			return candidate
		}
	}
	return nil
}

// resolveProtocolOneArg performs static overload resolution for a one-argument
// protocol method (CEqual, CCompare, or one of the arithmetic protocols)
// against a single already-analyzed right-hand operand. It reuses the same
// scoring rules as an ordinary call so operator resolution stays consistent
// with explicit method calls.
func (a *analyzer) resolveProtocolOneArg(name string, resolved *Symbol, argument expressionInfo, span source.Span) *Callable {
	var applicable []rankedCallable
	for _, candidate := range protocolCandidates(resolved) {
		if candidate == nil || candidate.Signature == nil || len(candidate.Signature.Parameters) != 1 {
			continue
		}
		parameter := candidate.Signature.Parameters[0]
		parameterNull := parameterNullAt(candidate.ParameterNull, 0)
		if argument.nullState != NonNull && parameterNull == NonNull {
			continue
		}
		quality, ok := conversionQuality(parameter.Type, argument.typeValue)
		if !ok {
			continue
		}
		score := overloadScore{widenings: quality}
		if argument.nullState == NonNull && parameterNull != NonNull {
			score.widenings++
		}
		applicable = append(applicable, rankedCallable{callable: candidate, score: score})
	}
	if len(applicable) == 0 {
		a.error(codeNoMatchingOverload, fmt.Sprintf("no overload of Class Protocol Method %q matches this operand", name), span, fmt.Sprintf("declare %s to accept %s, or convert the right operand", name, types.Display(argument.typeValue)))
		return nil
	}
	best := applicable[0].score
	for _, candidate := range applicable[1:] {
		if candidate.score.betterThan(best) {
			best = candidate.score
		}
	}
	var finalists []*Callable
	for _, candidate := range applicable {
		if candidate.score.equal(best) {
			finalists = append(finalists, candidate.callable)
		}
	}
	if len(finalists) != 1 {
		a.error(codeAmbiguousOverload, fmt.Sprintf("call to overloaded Class Protocol Method %q is ambiguous", name), span, "equally ranked candidates match this operand; add an explicit conversion or narrow the operand type")
		return nil
	}
	return finalists[0]
}

// classInstanceSymbol returns the Class Symbol for a NonNull Class instance
// type, or nil for every other type (including a bare Class reference).
func (a *analyzer) classInstanceSymbol(value types.Type) *Symbol {
	class, ok := value.(types.Class)
	if !ok || class.Reference {
		return nil
	}
	return a.classSymbolFor(class.Symbol)
}

// analyzeClassOperator resolves +, -, *, /, %, ^, <, <=, >, and >= against a
// NonNull Class left operand. It reports handled=false when the Class
// declares no matching protocol, so the caller falls back to the ordinary
// primitive operator diagnostic. CCompare is evaluated at most once: both
// ordering and arithmetic share this one resolution path, and the selected
// Callable is recorded for lowering to reuse verbatim.
func (a *analyzer) analyzeClassOperator(expression *ast.BinaryExpr, classSymbol *Symbol, operator string, right expressionInfo) (expressionInfo, bool) {
	switch operator {
	case "<", "<=", ">", ">=":
		resolved := a.resolveClassProtocol(classSymbol, "CCompare")
		if resolved == nil {
			return expressionInfo{}, false
		}
		callable := a.resolveProtocolOneArg("CCompare", resolved, right, expression.Span())
		if callable == nil {
			return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}, true
		}
		a.result.ResolvedSymbols[expression] = resolved
		a.result.SelectedFunctionValues[expression] = callable
		return expressionInfo{typeValue: types.Bool, nullState: NonNull}, true
	}
	name, ok := arithmeticProtocolNames[operator]
	if !ok {
		return expressionInfo{}, false
	}
	resolved := a.resolveClassProtocol(classSymbol, name)
	if resolved == nil {
		return expressionInfo{}, false
	}
	callable := a.resolveProtocolOneArg(name, resolved, right, expression.Span())
	if callable == nil {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}, true
	}
	a.result.ResolvedSymbols[expression] = resolved
	a.result.SelectedFunctionValues[expression] = callable
	return expressionInfo{typeValue: callable.Signature.Return, nullState: callable.ReturnNull}, true
}

// analyzeClassEquality resolves == and != against a NonNull Class left
// operand through CEqual. != is always the logical negation of the same
// CEqual call, so the two operators can never disagree.
func (a *analyzer) analyzeClassEquality(expression *ast.BinaryExpr, classSymbol *Symbol, right expressionInfo) (expressionInfo, bool) {
	resolved := a.resolveClassProtocol(classSymbol, "CEqual")
	if resolved == nil {
		return expressionInfo{}, false
	}
	callable := a.resolveProtocolOneArg("CEqual", resolved, right, expression.Span())
	if callable == nil {
		return expressionInfo{typeValue: types.Invalid, nullState: MaybeNull}, true
	}
	a.result.ResolvedSymbols[expression] = resolved
	a.result.SelectedFunctionValues[expression] = callable
	return expressionInfo{typeValue: types.Bool, nullState: NonNull}, true
}

// analyzeClassNegate resolves unary - against a NonNull Class operand through
// CNegate.
func (a *analyzer) analyzeClassNegate(expression ast.Expr, classSymbol *Symbol) (expressionInfo, bool) {
	resolved := a.resolveClassProtocol(classSymbol, "CNegate")
	if resolved == nil {
		return expressionInfo{}, false
	}
	callable := resolveProtocolNoArgs(resolved)
	if callable == nil {
		return expressionInfo{}, false
	}
	a.result.ResolvedSymbols[expression] = resolved
	a.result.SelectedFunctionValues[expression] = callable
	return expressionInfo{typeValue: callable.Signature.Return, nullState: callable.ReturnNull}, true
}
