package semantic

import (
	"fmt"

	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
)

// reportConstantCycle records one SEM010 for a module constant. A second
// discovery of the same cycle does not emit another diagnostic.
func (a *analyzer) reportConstantCycle(symbol *Symbol, span source.Span) {
	if symbol == nil {
		return
	}
	if a.constantCycle == nil {
		a.constantCycle = map[*Symbol]bool{}
	}
	if a.constantCycle[symbol] {
		return
	}
	a.constantCycle[symbol] = true
	a.error(codeConstantInitializer, fmt.Sprintf("cyclic Constant dependency involving %q", symbol.Name), span, "break the Constant dependency cycle")
}

// finishConstantDependencies records which module constants each initializer
// reads, and rejects a cycle that scalar evaluation did not already report.
// Reference constants skip the scalar evaluator; their cycles are found here.
// A use that would observe a constant before these rules can initialize it
// is SEM010, not a zero value.
func (a *analyzer) finishConstantDependencies(program *ast.Program) {
	var constants []*Symbol
	for _, symbol := range a.result.Symbols {
		if symbol == nil || !symbol.Constant || !symbol.ModuleRoot || symbol.Builtin || symbol.Alias != nil {
			continue
		}
		if symbol.OriginModuleID != a.environment.ModuleID {
			continue
		}
		dependencies, mutables, effects := a.constantInitializerFacts(symbol)
		symbol.ConstantDependencies = dependencies
		symbol.ConstantMutables = mutables
		symbol.ConstantEffects = effects
		symbol.ConstantAnchored = effects || len(mutables) > 0
		constants = append(constants, symbol)
	}
	color := map[*Symbol]int{}
	var stack []*Symbol
	var visit func(*Symbol)
	visit = func(symbol *Symbol) {
		if symbol == nil || color[symbol] == 2 {
			return
		}
		if color[symbol] == 1 {
			reporting := false
			for _, item := range stack {
				if item == symbol {
					reporting = true
				}
				if reporting {
					span := item.Span
					if declaration, ok := item.Declaration.(*ast.VariableDecl); ok && declaration.Initializer != nil {
						span = declaration.Initializer.Span()
					}
					a.reportConstantCycle(item, span)
				}
			}
			return
		}
		color[symbol] = 1
		stack = append(stack, symbol)
		for _, dependency := range symbol.ConstantDependencies {
			visit(dependency)
		}
		stack = stack[:len(stack)-1]
		color[symbol] = 2
	}
	for _, symbol := range constants {
		visit(symbol)
	}
	if program != nil {
		a.rejectEarlyConstantUses(program, constants)
	}
}

// constantInitializerFacts walks one constant initializer. Same-module
// Constant bindings become dependencies. A function declaration does not.
// A mutable binding is recorded separately. A call sets effects.
func (a *analyzer) constantInitializerFacts(symbol *Symbol) ([]*Symbol, []*Symbol, bool) {
	declaration, ok := symbol.Declaration.(*ast.VariableDecl)
	if !ok || declaration == nil || declaration.Initializer == nil {
		return nil, nil, false
	}
	seen := map[*Symbol]bool{}
	var dependencies []*Symbol
	var mutables []*Symbol
	effects := false
	var walk func(ast.Expr)
	walk = func(expression ast.Expr) {
		if expression == nil {
			return
		}
		switch value := expression.(type) {
		case *ast.LiteralExpr:
		case *ast.GroupExpr:
			walk(value.Expression)
		case *ast.UnaryExpr:
			walk(value.Operand)
		case *ast.BinaryExpr:
			walk(value.Left)
			walk(value.Right)
		case *ast.ListExpr:
			for _, element := range value.Elements {
				walk(element)
			}
		case *ast.PairExpr:
			for index := range value.Entries {
				walk(value.Entries[index].Key)
				walk(value.Entries[index].Value)
			}
		case *ast.StringExpr:
			for _, part := range value.Parts {
				if part.Expression != nil {
					effects = true
					walk(part.Expression)
				}
			}
		case *ast.LambdaExpr:
		case *ast.IdentifierExpr:
			found := a.resolvedConstantTarget(value)
			if found == nil {
				return
			}
			// A function or class declaration is in scope for the whole
			// module. It is not a constant value this initializer waits for.
			// A call is still an effect, recorded by the CallExpr arm.
			if found.Kind == FunctionSymbol || found.Kind == ClassSymbol {
				return
			}
			if found.Constant && found.ModuleRoot && !found.Builtin && found.OriginModuleID == a.environment.ModuleID {
				if !seen[found] {
					seen[found] = true
					dependencies = append(dependencies, found)
				}
				return
			}
			if found.ModuleRoot && !found.Builtin && found.OriginModuleID == a.environment.ModuleID {
				if !seen[found] {
					seen[found] = true
					mutables = append(mutables, found)
				}
			}
		case *ast.CallExpr:
			effects = true
			walk(value.Callee)
			for index := range value.Arguments {
				walk(value.Arguments[index].Value)
			}
		case *ast.MemberExpr:
			walk(value.Object)
		case *ast.IndexExpr:
			walk(value.Object)
			walk(value.Index)
		case *ast.SliceExpr:
			walk(value.Object)
			walk(value.Start)
			walk(value.End)
		default:
			effects = true
		}
	}
	walk(declaration.Initializer)
	return dependencies, mutables, effects
}

// resolvedConstantTarget is the module binding an identifier names, following
// a uses/Global alias. Nil means the name is not a module value binding.
func (a *analyzer) resolvedConstantTarget(identifier *ast.IdentifierExpr) *Symbol {
	found := a.result.ResolvedSymbols[identifier]
	if found == nil {
		if local, known := a.module.local(identifier.Name); known {
			found = local
		}
	}
	followed := map[*Symbol]bool{}
	for found != nil && found.Alias != nil && !followed[found] {
		followed[found] = true
		found = found.Alias
	}
	return found
}

// rejectEarlyConstantUses applies the initialization rules to the module
// statement order. A pure constant graph is ready before the other
// statements. A constant that reads a mutable or calls a function becomes
// ready at its own statement, after the constants it depends on. A read
// before that point is SEM010.
func (a *analyzer) rejectEarlyConstantUses(program *ast.Program, constants []*Symbol) {
	ready := map[*Symbol]bool{}
	var mark func(*Symbol)
	mark = func(symbol *Symbol) {
		if symbol == nil || ready[symbol] || a.constantCycle[symbol] {
			return
		}
		ready[symbol] = true
		for _, candidate := range constants {
			if ready[candidate] || candidate.ConstantAnchored || a.constantCycle[candidate] {
				continue
			}
			if dependenciesReady(candidate, ready) {
				mark(candidate)
			}
		}
	}
	for _, symbol := range constants {
		if symbol.ConstantAnchored || a.constantCycle[symbol] {
			continue
		}
		if dependenciesReady(symbol, ready) {
			mark(symbol)
		}
	}
	reported := map[*Symbol]bool{}
	for _, statement := range program.Statements {
		a.scheduleConstantStatement(statement, ready, reported, mark)
	}
}

func dependenciesReady(symbol *Symbol, ready map[*Symbol]bool) bool {
	for _, dependency := range symbol.ConstantDependencies {
		if dependency != nil && !ready[dependency] {
			return false
		}
	}
	return true
}

func (a *analyzer) scheduleConstantStatement(statement ast.Stmt, ready map[*Symbol]bool, reported map[*Symbol]bool, mark func(*Symbol)) {
	declaration, isDeclaration := statement.(*ast.VariableDecl)
	if isDeclaration {
		symbol := a.result.ResolvedSymbols[declaration]
		if a.isScheduledConstant(symbol) {
			if a.constantCycle[symbol] {
				return
			}
			if !dependenciesReady(symbol, ready) {
				if symbol.ConstantAnchored {
					span := declaration.Span()
					if declaration.Initializer != nil {
						span = declaration.Initializer.Span()
					}
					a.reportEarlyConstant(symbol, span, ready, reported)
				}
				return
			}
			if declaration.Initializer != nil && a.reportUnreadyReads(&ast.ExprStmt{Expression: declaration.Initializer}, ready, reported) {
				return
			}
			mark(symbol)
			return
		}
	}
	switch statement.(type) {
	case *ast.FunctionDecl, *ast.ClassDecl, *ast.BringStmt, *ast.RequireStmt, *ast.StructureDecl:
		return
	}
	a.reportUnreadyReads(statement, ready, reported)
}

// reportUnreadyReads reports module constants this statement would observe
// before they are initialized. It returns whether any such read was found.
func (a *analyzer) reportUnreadyReads(statement ast.Stmt, ready map[*Symbol]bool, reported map[*Symbol]bool) bool {
	blocked := false
	for _, identifier := range a.runningConstantReads(statement, map[*Symbol]bool{}) {
		symbol := a.resolvedConstantTarget(identifier)
		if !a.isScheduledConstant(symbol) || ready[symbol] || a.constantCycle[symbol] {
			continue
		}
		a.reportEarlyConstant(symbol, identifier.Span(), ready, reported)
		blocked = true
	}
	return blocked
}

func (a *analyzer) isScheduledConstant(symbol *Symbol) bool {
	return symbol != nil && symbol.Kind == BindingSymbol && symbol.Constant && symbol.ModuleRoot && !symbol.Builtin && symbol.OriginModuleID == a.environment.ModuleID
}

func (a *analyzer) reportEarlyConstant(symbol *Symbol, span source.Span, ready map[*Symbol]bool, reported map[*Symbol]bool) {
	if symbol == nil || reported[symbol] || a.constantCycle[symbol] {
		return
	}
	reported[symbol] = true
	a.error(codeConstantInitializer, fmt.Sprintf("Constant %q is used before it can be initialized", symbol.Name), span, a.earlyConstantHint(symbol, ready))
}

func (a *analyzer) earlyConstantHint(symbol *Symbol, ready map[*Symbol]bool) string {
	for _, dependency := range symbol.ConstantDependencies {
		if dependency == nil || ready[dependency] || a.constantCycle[dependency] {
			continue
		}
		if dependency.ConstantEffects {
			return fmt.Sprintf("it depends on Constant %q, which calls a function and runs where it is written", dependency.Name)
		}
		if len(dependency.ConstantMutables) > 0 {
			return fmt.Sprintf("it depends on Constant %q, which reads a mutable binding and stays where it is written", dependency.Name)
		}
		return fmt.Sprintf("it depends on Constant %q, which is initialized later", dependency.Name)
	}
	if symbol.ConstantEffects {
		return "this initializer calls a function and runs where it is written"
	}
	if len(symbol.ConstantMutables) > 0 {
		return "this initializer reads a mutable binding and stays where it is written"
	}
	return "move this use below the Constant"
}

// runningConstantReads lists identifiers in one module statement that read a
// module constant while the statement runs. A direct call of a module
// function also reads what that function's body reads.
func (a *analyzer) runningConstantReads(statement ast.Stmt, visiting map[*Symbol]bool) []*ast.IdentifierExpr {
	var found []*ast.IdentifierExpr
	var walkExpr func(ast.Expr)
	var walkStmt func(ast.Stmt)
	var walkBlock func(*ast.Block)
	walkBlock = func(block *ast.Block) {
		if block == nil {
			return
		}
		for _, inner := range block.Statements {
			walkStmt(inner)
		}
	}
	walkExpr = func(expression ast.Expr) {
		if expression == nil {
			return
		}
		switch value := expression.(type) {
		case *ast.IdentifierExpr:
			found = append(found, value)
		case *ast.GroupExpr:
			walkExpr(value.Expression)
		case *ast.UnaryExpr:
			walkExpr(value.Operand)
		case *ast.BinaryExpr:
			walkExpr(value.Left)
			walkExpr(value.Right)
		case *ast.ListExpr:
			for _, element := range value.Elements {
				walkExpr(element)
			}
		case *ast.PairExpr:
			for index := range value.Entries {
				walkExpr(value.Entries[index].Key)
				walkExpr(value.Entries[index].Value)
			}
		case *ast.StringExpr:
			for _, part := range value.Parts {
				walkExpr(part.Expression)
			}
		case *ast.CallExpr:
			if identifier, ok := value.Callee.(*ast.IdentifierExpr); ok {
				target := a.resolvedConstantTarget(identifier)
				if target != nil && target.Kind == FunctionSymbol && !visiting[target] {
					if declaration, ok := target.Declaration.(*ast.FunctionDecl); ok && declaration.Body != nil {
						visiting[target] = true
						walkBlock(declaration.Body)
						delete(visiting, target)
					}
				}
			}
			walkExpr(value.Callee)
			for index := range value.Arguments {
				walkExpr(value.Arguments[index].Value)
			}
		case *ast.MemberExpr:
			walkExpr(value.Object)
		case *ast.IndexExpr:
			walkExpr(value.Object)
			walkExpr(value.Index)
		case *ast.SliceExpr:
			walkExpr(value.Object)
			walkExpr(value.Start)
			walkExpr(value.End)
		case *ast.LambdaExpr:
			walkExpr(value.Body)
		}
	}
	walkStmt = func(statement ast.Stmt) {
		if statement == nil {
			return
		}
		switch value := statement.(type) {
		case *ast.ExprStmt:
			walkExpr(value.Expression)
		case *ast.VariableDecl:
			walkExpr(value.Initializer)
		case *ast.AssignmentStmt:
			walkExpr(value.Target)
			walkExpr(value.Value)
		case *ast.IncDecStmt:
			walkExpr(value.Target)
		case *ast.ReturnStmt:
			walkExpr(value.Value)
		case *ast.TossStmt:
			walkExpr(value.Value)
		case *ast.IfStmt:
			for index := range value.Branches {
				walkExpr(value.Branches[index].Condition)
				walkBlock(value.Branches[index].Body)
			}
			walkBlock(value.Else)
		case *ast.WhileStmt:
			walkExpr(value.Condition)
			walkBlock(value.Body)
		case *ast.UntilStmt:
			walkExpr(value.Condition)
			walkBlock(value.Body)
		case *ast.ForStmt:
			walkExpr(value.Iterable)
			walkBlock(value.Body)
		case *ast.AttemptStmt:
			walkBlock(value.Body)
			for index := range value.Excepts {
				walkBlock(value.Excepts[index].Body)
			}
			walkBlock(value.Ultimately)
		case *ast.StateStmt:
			walkExpr(value.Value)
			for index := range value.Conditions {
				walkExpr(value.Conditions[index].Match)
				walkBlock(value.Conditions[index].Body)
			}
		}
	}
	walkStmt(statement)
	return found
}
