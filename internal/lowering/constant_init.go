package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// moduleInitItem is one statement of a module initializer. constant is set
// only for a module Constant's own initializer.
type moduleInitItem struct {
	statement ir.Statement
	constant  *semantic.Symbol
}

// scheduleModuleInit initializes a pure constant graph in dependency order
// before the module's other statements. An initializer that reads a mutable
// binding or calls a function stays at its source position. A pure constant
// that depends on one of those runs immediately after it. Semantic analysis
// rejects a use that would happen before that point, so this scheduler does
// not invent an order and does not report an internal error.
func (lowerer *moduleLowerer) scheduleModuleInit(items []moduleInitItem) []ir.Statement {
	if len(items) == 0 {
		return nil
	}
	emitted := make([]bool, len(items))
	placed := map[*semantic.Symbol]bool{}
	var statements []ir.Statement
	emit := func(index int) {
		if emitted[index] {
			return
		}
		emitted[index] = true
		statements = append(statements, items[index].statement)
		if items[index].constant != nil {
			placed[items[index].constant] = true
		}
	}
	ready := func(symbol *semantic.Symbol) bool {
		for _, dependency := range symbol.ConstantDependencies {
			if dependency != nil && !placed[dependency] {
				return false
			}
		}
		return true
	}
	drainPure := func() {
		for {
			progress := false
			for index, item := range items {
				if emitted[index] || item.constant == nil || item.constant.ConstantAnchored {
					continue
				}
				if !ready(item.constant) {
					continue
				}
				emit(index)
				progress = true
			}
			if !progress {
				return
			}
		}
	}
	drainPure()
	for index, item := range items {
		if emitted[index] {
			continue
		}
		if item.constant != nil && !ready(item.constant) {
			continue
		}
		emit(index)
		drainPure()
	}
	return statements
}
