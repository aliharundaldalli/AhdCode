package semantic

type scopeKind uint8

const (
	moduleScope scopeKind = iota
	classScope
	callableScope
	blockScope
)

type scope struct {
	parent   *scope
	kind     scopeKind
	symbols  map[string]*Symbol
	callable *callableContext
}

func newScope(parent *scope, kind scopeKind) *scope {
	created := &scope{parent: parent, kind: kind, symbols: make(map[string]*Symbol)}
	if parent != nil {
		created.callable = parent.callable
	}
	return created
}

func (current *scope) local(name string) (*Symbol, bool) {
	symbol, ok := current.symbols[name]
	return symbol, ok
}

func (current *scope) lookup(name string) (*Symbol, *scope) {
	for candidate := current; candidate != nil; candidate = candidate.parent {
		if symbol, ok := candidate.symbols[name]; ok {
			return symbol, candidate
		}
	}
	return nil, nil
}

type flowState map[*Symbol]NullState

func (flow flowState) clone() flowState {
	cloned := make(flowState, len(flow))
	for symbol, state := range flow {
		cloned[symbol] = state
	}
	return cloned
}

func (flow flowState) state(symbol *Symbol) NullState {
	if state, ok := flow[symbol]; ok {
		return state
	}
	if symbol.Alias != nil {
		return flow.state(symbol.Alias)
	}
	return symbol.InitialNull
}

func mergeNull(left, right NullState) NullState {
	if left == right {
		return left
	}
	return MaybeNull
}

func mergeFlows(flows ...flowState) flowState {
	if len(flows) == 0 {
		return make(flowState)
	}
	merged := flows[0].clone()
	for _, flow := range flows[1:] {
		for symbol, state := range merged {
			state = mergeNull(state, flow.state(symbol))
			merged[symbol] = state
		}
		for symbol, state := range flow {
			if _, ok := merged[symbol]; !ok {
				merged[symbol] = mergeNull(symbol.InitialNull, state)
			}
		}
	}
	return merged
}
