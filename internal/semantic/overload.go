package semantic

import (
	"fmt"
	"strings"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

type overloadScore struct {
	widenings int
	defaults  int
}

func (score overloadScore) betterThan(other overloadScore) bool {
	if score.widenings != other.widenings {
		return score.widenings < other.widenings
	}
	return score.defaults < other.defaults
}

func (score overloadScore) equal(other overloadScore) bool {
	return score.widenings == other.widenings && score.defaults == other.defaults
}

type rankedCallable struct {
	callable *Callable
	score    overloadScore
}

func (a *analyzer) resolveOverloadCall(call *ast.CallExpr, set *OverloadSet, arguments []expressionInfo) *Callable {
	trace := ResolutionTrace{}
	var applicable []rankedCallable
	for _, candidate := range set.Candidates {
		score, reason, ok := assessCallCandidate(call, candidate, arguments)
		decision := CandidateDecision{Signature: formatSignature(candidate.Signature), Applicable: ok, Reason: reason, Widenings: score.widenings, Defaults: score.defaults}
		trace.Candidates = append(trace.Candidates, decision)
		if ok {
			applicable = append(applicable, rankedCallable{callable: candidate, score: score})
		}
	}
	if len(applicable) == 0 {
		a.result.OverloadResolutions[call] = trace
		a.error(codeNoMatchingOverload, fmt.Sprintf("no overload of %q matches this call", set.Name), call.Span(), summarizeRejections(trace.Candidates))
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
		signatures := make([]string, len(finalists))
		for index, candidate := range finalists {
			signatures[index] = formatSignature(candidate.Signature)
		}
		a.result.OverloadResolutions[call] = trace
		a.error(codeAmbiguousOverload, fmt.Sprintf("call to overloaded Function %q is ambiguous", set.Name), call.Span(), "equally ranked candidates: "+strings.Join(signatures, "; "))
		return nil
	}
	trace.Selected = formatSignature(finalists[0].Signature)
	a.result.OverloadResolutions[call] = trace
	return finalists[0]
}

func assessCallCandidate(call *ast.CallExpr, callable *Callable, arguments []expressionInfo) (overloadScore, string, bool) {
	if callable == nil || callable.Signature == nil {
		return overloadScore{}, "candidate has no concrete signature", false
	}
	parameters := callable.Signature.Parameters
	assigned := make([]bool, len(parameters))
	score := overloadScore{}
	for argumentIndex, argument := range call.Arguments {
		parameterIndex := argumentIndex
		if argument.Name != "" {
			parameterIndex = -1
			for index, parameter := range parameters {
				if parameter.Name == argument.Name {
					parameterIndex = index
					break
				}
			}
		}
		if parameterIndex < 0 || parameterIndex >= len(parameters) {
			if argument.Name != "" {
				return score, fmt.Sprintf("no parameter named %q", argument.Name), false
			}
			return score, fmt.Sprintf("accepts %d parameter(s), received %d", len(parameters), len(arguments)), false
		}
		if assigned[parameterIndex] {
			return score, fmt.Sprintf("parameter %q is supplied more than once", parameters[parameterIndex].Name), false
		}
		assigned[parameterIndex] = true
		if arguments[argumentIndex].nullState != NonNull && (parameterIndex >= len(callable.ParameterNull) || callable.ParameterNull[parameterIndex] == NonNull) {
			return score, fmt.Sprintf("argument for %q may be null", parameters[parameterIndex].Name), false
		}
		quality, ok := conversionQuality(parameters[parameterIndex].Type, arguments[argumentIndex].typeValue)
		if !ok {
			return score, fmt.Sprintf("argument %q expects %s, received %s", parameters[parameterIndex].Name, types.Display(parameters[parameterIndex].Type), types.Display(arguments[argumentIndex].typeValue)), false
		}
		score.widenings += quality
		// A proven-NonNull argument matching a nullable parameter is a wider
		// (less exact) match than matching a non-nullable parameter of the same
		// type, so a same-type non-nullable overload is preferred whenever the
		// argument is already known non-null. A null/MaybeNull argument does not
		// get this penalty: it genuinely needs the wider, nullable parameter.
		if arguments[argumentIndex].nullState == NonNull && parameterIndex < len(callable.ParameterNull) && callable.ParameterNull[parameterIndex] != NonNull {
			score.widenings++
		}
	}
	for index, parameter := range parameters {
		if assigned[index] {
			continue
		}
		if !parameter.HasDefault {
			return score, fmt.Sprintf("required parameter %q is missing", parameter.Name), false
		}
		score.defaults++
	}
	return score, "applicable", true
}

func conversionQuality(target, value types.Type) (int, bool) {
	if types.Equal(target, value) {
		return 0, true
	}
	if targetFunction, targetOK := target.(types.Function); targetOK && targetFunction.Signature != nil {
		if valueFunction, valueOK := value.(types.Function); valueOK && valueFunction.Signature != nil {
			score, ok := functionValueScore(valueFunction.Signature, targetFunction.Signature)
			if !ok {
				return 0, false
			}
			return score.widenings + score.defaults + 1, true
		}
	}
	if types.Assignable(target, value) {
		return 1, true
	}
	return 0, false
}

func summarizeRejections(decisions []CandidateDecision) string {
	parts := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		parts = append(parts, decision.Signature+": "+decision.Reason)
	}
	return strings.Join(parts, "; ")
}

func (a *analyzer) selectFunctionValue(set *OverloadSet, expected *types.Signature, expression ast.Expr) *Callable {
	var best []*Callable
	bestScore := overloadScore{widenings: int(^uint(0) >> 1), defaults: int(^uint(0) >> 1)}
	for _, candidate := range set.Candidates {
		score, ok := functionValueScore(candidate.Signature, expected)
		if !ok {
			continue
		}
		if score.betterThan(bestScore) {
			bestScore = score
			best = []*Callable{candidate}
		} else if score.equal(bestScore) {
			best = append(best, candidate)
		}
	}
	if len(best) == 0 {
		a.error(codeNoMatchingOverload, fmt.Sprintf("no overload of %q matches callback type %s", set.Name, formatSignature(expected)), expression.Span(), "use a Function whose parameters and return type satisfy the callback context")
		return nil
	}
	if len(best) > 1 {
		a.error(codeAmbiguousOverload, fmt.Sprintf("overloaded Function value %q remains ambiguous", set.Name), expression.Span(), "callback context leaves multiple equally ranked candidates")
		return nil
	}
	return best[0]
}

func functionValueScore(candidate, expected *types.Signature) (overloadScore, bool) {
	if candidate == nil || expected == nil || len(expected.Parameters) > len(candidate.Parameters) {
		return overloadScore{}, false
	}
	score := overloadScore{}
	for index, parameter := range expected.Parameters {
		quality, ok := conversionQuality(candidate.Parameters[index].Type, parameter.Type)
		if !ok {
			return score, false
		}
		score.widenings += quality
	}
	for index := len(expected.Parameters); index < len(candidate.Parameters); index++ {
		if !candidate.Parameters[index].HasDefault {
			return score, false
		}
		score.defaults++
	}
	if !types.Assignable(expected.Return, candidate.Return) {
		return score, false
	}
	return score, true
}
