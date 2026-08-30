package evaluator

import "ahdcode/internal/ir"

type frame struct {
	parent       *frame
	locals       map[ir.SymbolID]*Cell
	constructing *Instance
}

func newFrame(parent *frame) *frame {
	return &frame{parent: parent, locals: make(map[ir.SymbolID]*Cell)}
}

func (session *Session) cell(current *frame, symbol ir.SymbolID) *Cell {
	for scope := current; scope != nil; scope = scope.parent {
		if cell := scope.locals[symbol]; cell != nil {
			return cell
		}
	}
	if cell := session.globals[symbol]; cell != nil {
		return cell
	}
	session.raise("Error", "missing runtime binding "+string(symbol))
	return nil
}

type flowKind uint8

const (
	flowNormal flowKind = iota
	flowReturn
	flowBreak
	flowContinue
)

type flowResult struct {
	kind  flowKind
	value any
}

func (session *Session) execBlock(block ir.Block, current *frame) (any, bool, flowResult) {
	return session.execBlockFiltered(block, current, -1)
}

func (session *Session) execBlockFiltered(block ir.Block, current *frame, offset int) (any, bool, flowResult) {
	var last any
	hasValue := false
	for _, statement := range block.Statements {
		if statement == nil || offset >= 0 && statement.StatementSpan().Start.Offset < offset {
			continue
		}
		value, expression, outcome := session.execStatement(statement, current)
		if expression {
			last, hasValue = value, true
		}
		if outcome.kind != flowNormal {
			return last, hasValue, outcome
		}
	}
	return last, hasValue, flowResult{}
}

func (session *Session) execStatement(statement ir.Statement, current *frame) (any, bool, flowResult) {
	switch value := statement.(type) {
	case *ir.ExprStmt:
		result := session.eval(value.Value, current)
		return result, value.Value != nil && value.Value.ExprMeta().Type.Kind != ir.NothingType, flowResult{}
	case *ir.BindingStmt:
		cell := &Cell{Constant: value.Constant}
		if value.Initializer != nil {
			cell.Value = session.eval(value.Initializer, current)
			if value.Constant {
				freeze(cell.Value)
			}
		}
		if value.Storage == ir.ModuleStorage {
			session.globals[value.Symbol] = cell
		} else {
			current.locals[value.Symbol] = cell
		}
	case *ir.AssignStmt:
		session.assignTarget(value.Target, session.eval(value.Value, current), current)
	case *ir.CompoundAssignStmt:
		target := session.resolveTarget(value.Target, current)
		right := session.eval(value.Value, current)
		if value.Protocol != "" {
			receiver := session.requireInstance(target.get())
			target.set(session.invoke(&FunctionValue{Callable: value.Protocol, Receiver: receiver}, []argumentValue{{value: right}}))
		} else {
			target.set(session.binary(value.Op, target.get(), right))
		}
	case *ir.UpdateStmt:
		target := session.resolveTarget(value.Target, current)
		target.set(session.binary("CheckedIntAdd", target.get(), int64(value.Delta)))
	case *ir.IfStmt:
		for _, branch := range value.Branches {
			if session.boolean(session.eval(branch.Condition, current)) {
				_, _, outcome := session.execBlock(branch.Body, newFrame(current))
				return nil, false, outcome
			}
		}
		if value.Else != nil {
			_, _, outcome := session.execBlock(*value.Else, newFrame(current))
			return nil, false, outcome
		}
	case *ir.WhileStmt:
		for session.boolean(session.eval(value.Condition, current)) {
			_, _, outcome := session.execBlock(value.Body, newFrame(current))
			switch outcome.kind {
			case flowReturn:
				return nil, false, outcome
			case flowBreak:
				return nil, false, flowResult{}
			}
		}
	case *ir.DoUntilStmt:
		for {
			_, _, outcome := session.execBlock(value.Body, newFrame(current))
			if outcome.kind == flowReturn {
				return nil, false, outcome
			}
			if outcome.kind == flowBreak || session.boolean(session.eval(value.Condition, current)) {
				break
			}
		}
	case *ir.ForStmt:
		if value.Kind == ir.IntRange {
			iteration, ok := session.eval(value.Iterable, current).(*Range)
			if !ok || iteration == nil {
				session.raise("NullError", "range value is null")
			}
			for {
				item, more := nextRange(iteration)
				if !more {
					break
				}
				scope := newFrame(current)
				scope.locals[value.Iteration] = &Cell{Value: item}
				_, _, outcome := session.execBlock(value.Body, scope)
				if outcome.kind == flowReturn {
					return nil, false, outcome
				}
				if outcome.kind == flowBreak {
					break
				}
			}
			break
		}
		items := session.iterationItems(value, current)
		for _, item := range items {
			scope := newFrame(current)
			scope.locals[value.Iteration] = &Cell{Value: item}
			_, _, outcome := session.execBlock(value.Body, scope)
			switch outcome.kind {
			case flowReturn:
				return nil, false, outcome
			case flowBreak:
				return nil, false, flowResult{}
			}
		}
	case *ir.StateStmt:
		subject := session.eval(value.Value, current)
		var fallback *ir.StateCase
		for index := range value.Cases {
			item := &value.Cases[index]
			if item.Default {
				fallback = item
				continue
			}
			if session.equal(subject, session.eval(item.Match, current)) {
				_, _, outcome := session.execBlock(item.Body, newFrame(current))
				return nil, false, outcome
			}
		}
		if fallback != nil {
			_, _, outcome := session.execBlock(fallback.Body, newFrame(current))
			return nil, false, outcome
		}
	case *ir.AttemptStmt:
		return nil, false, session.execAttempt(value, current)
	case *ir.TossStmt:
		instance := session.requireInstance(session.eval(value.Value, current))
		message := ""
		if field := session.fieldNamed(instance.Class, "message"); field != "" {
			message, _ = instance.Fields[field].(string)
		}
		panic(raised{failure: &RuntimeError{Class: instance.Class, Name: className(instance.Class), Message: message, Instance: instance}})
	case *ir.ReturnStmt:
		result := any(Nothing)
		if value.Value != nil {
			result = session.eval(value.Value, current)
		}
		return nil, false, flowResult{kind: flowReturn, value: result}
	case *ir.BreakStmt:
		return nil, false, flowResult{kind: flowBreak}
	case *ir.ContinueStmt:
		return nil, false, flowResult{kind: flowContinue}
	default:
		session.raise("Error", "unsupported IR statement")
	}
	return nil, false, flowResult{}
}

func (session *Session) execAttempt(statement *ir.AttemptStmt, current *frame) (outcome flowResult) {
	func() {
		if statement.Ultimately != nil {
			defer func() {
				_, _, final := session.execBlock(*statement.Ultimately, newFrame(current))
				if final.kind != flowNormal {
					outcome = final
				}
			}()
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				var failure *RuntimeError
				switch value := recovered.(type) {
				case raised:
					failure = value.failure
				case *raised:
					failure = value.failure
				default:
					panic(recovered)
				}
				for _, handler := range statement.Handlers {
					if session.isClass(failure.Class, handler.Class) {
						scope := newFrame(current)
						if handler.Binding != "" {
							scope.locals[handler.Binding] = &Cell{Value: failure.Instance}
						}
						_, _, outcome = session.execBlock(handler.Body, scope)
						return
					}
				}
				panic(recovered)
			}
		}()
		_, _, outcome = session.execBlock(statement.Body, newFrame(current))
	}()
	return outcome
}

func (session *Session) iterationItems(statement *ir.ForStmt, current *frame) []any {
	value := session.eval(statement.Iterable, current)
	switch statement.Kind {
	case ir.ListElements:
		list := session.requireList(value)
		return append([]any(nil), list.Items...)
	case ir.PairKeys:
		pair := session.requirePair(value)
		return append([]any(nil), pair.Keys...)
	case ir.StringCharacters:
		text, _ := value.(string)
		runes := []rune(text)
		result := make([]any, len(runes))
		for index, item := range runes {
			result[index] = string(item)
		}
		return result
	}
	return nil
}

func nextRange(value *Range) (int64, bool) {
	if value == nil || value.Finished || value.Step > 0 && value.Current >= value.Stop || value.Step < 0 && value.Current <= value.Stop {
		return 0, false
	}
	current := value.Current
	next := current + value.Step
	if value.Step > 0 && next < current || value.Step < 0 && next > current {
		value.Finished = true
	} else {
		value.Current = next
	}
	return current, true
}

type resolvedTarget struct {
	get func() any
	set func(any)
}

func (session *Session) assignTarget(target ir.Target, value any, current *frame) {
	session.resolveTarget(target, current).set(value)
}

func (session *Session) resolveTarget(target ir.Target, current *frame) resolvedTarget {
	switch target.Kind {
	case ir.SymbolTarget:
		cell := session.cell(current, target.Symbol)
		return resolvedTarget{get: func() any { return cell.Value }, set: func(value any) {
			if cell.Constant {
				session.raise("ConstantError", "cannot mutate a Constant binding")
			}
			cell.Value = value
		}}
	case ir.FieldTarget:
		instance := session.requireInstance(session.eval(target.Receiver, current))
		field := session.field(target.Field)
		return resolvedTarget{get: func() any { return instance.Fields[target.Field] }, set: func(value any) {
			if instance.Frozen || field.Constant && current.constructing != instance {
				session.raise("ConstantError", "cannot mutate a Constant attribute")
			}
			if field.Constant {
				freeze(value)
			}
			instance.Fields[target.Field] = value
		}}
	case ir.IndexTarget:
		receiver := session.eval(target.Receiver, current)
		index := session.eval(target.Index, current)
		switch collection := receiver.(type) {
		case *List:
			position := resolveIndex(session, index.(int64), len(collection.Items))
			return resolvedTarget{get: func() any { return collection.Items[position] }, set: func(value any) {
				session.requireMutable(collection)
				collection.Items[position] = value
			}}
		case *Pair:
			pair := session.requirePair(collection)
			return resolvedTarget{get: func() any {
				value, exists := pair.Values[index]
				if !exists {
					session.raise("KeyError", "Pair key was not found")
				}
				return value
			}, set: func(value any) {
				session.requireMutable(pair)
				pairSet(pair, index, value)
			}}
		}
	}
	session.raise("Error", "invalid assignment target")
	return resolvedTarget{}
}

func (session *Session) field(identity ir.FieldID) ir.Field {
	for _, class := range session.classes {
		for _, field := range class.Fields {
			if field.ID == identity {
				return field
			}
		}
	}
	return ir.Field{ID: identity}
}
