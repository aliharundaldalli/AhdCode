package lowering

import (
	"fmt"

	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

func (lowerer *moduleLowerer) lowerBlock(block *ast.Block) ir.Block {
	if block == nil {
		return ir.Block{}
	}
	result := ir.Block{Span: block.Span()}
	for _, statement := range block.Statements {
		if lowered := lowerer.lowerStatement(statement); lowered != nil {
			result.Statements = append(result.Statements, lowered)
		}
	}
	return result
}

func (lowerer *moduleLowerer) lowerStatement(statement ast.Stmt) ir.Statement {
	if statement == nil {
		return nil
	}
	switch value := statement.(type) {
	case *ast.BadStmt:
		lowerer.compilation.error(CodeUnsupportedNode, "cannot lower recovered BadStmt", value.Span())
	case *ast.ExprStmt:
		return &ir.ExprStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Value: lowerer.lowerExpr(value.Expression)}
	case *ast.VariableDecl:
		return lowerer.lowerVariable(value)
	case *ast.AssignmentStmt:
		target := lowerer.lowerTarget(value.Target)
		if value.Operator == "=" {
			return &ir.AssignStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Target: target, Value: lowerer.lowerExprExpected(value.Value, target.Type)}
		}
		right := lowerer.lowerExprExpected(value.Value, target.Type)
		op := typedBinaryOp(value.Operator[:len(value.Operator)-1], target.Type, right.ExprMeta().Type, target.Type)
		return &ir.CompoundAssignStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Target: target, Op: op, Value: right}
	case *ast.IncDecStmt:
		delta := 1
		if value.Operator == "--" {
			delta = -1
		}
		return &ir.UpdateStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Target: lowerer.lowerTarget(value.Target), Delta: delta}
	case *ast.IfStmt:
		result := &ir.IfStmt{StmtBase: ir.StmtBase{Span: value.Span()}}
		for _, branch := range value.Branches {
			result.Branches = append(result.Branches, ir.ConditionalBlock{Condition: lowerer.lowerExprExpected(branch.Condition, ir.Type{Kind: ir.BoolType}), Body: lowerer.lowerBlock(branch.Body)})
		}
		if value.Else != nil {
			block := lowerer.lowerBlock(value.Else)
			result.Else = &block
		}
		return result
	case *ast.WhileStmt:
		return &ir.WhileStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Condition: lowerer.lowerExprExpected(value.Condition, ir.Type{Kind: ir.BoolType}), Body: lowerer.lowerBlock(value.Body)}
	case *ast.UntilStmt:
		return &ir.DoUntilStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Body: lowerer.lowerBlock(value.Body), Condition: lowerer.lowerExprExpected(value.Condition, ir.Type{Kind: ir.BoolType}), ContinueChecksCondition: true}
	case *ast.ForStmt:
		iterable := lowerer.lowerExpr(value.Iterable)
		iteration := findSymbol(lowerer.semantic, semantic.ForSymbol, value.Name, value.Span())
		iterationType := ir.Type{Kind: ir.InvalidType}
		if iteration != nil {
			iterationType = lowerType(iteration.Type)
		}
		kind := ir.ListElements
		switch lowerer.semantic.ExpressionTypes[value.Iterable].(type) {
		case types.Pair:
			kind = ir.PairKeys
		default:
			if lowerer.semantic.ExpressionTypes[value.Iterable] != nil && lowerer.semantic.ExpressionTypes[value.Iterable].Kind() == types.StringKind {
				kind = ir.StringCharacters
			}
		}
		return &ir.ForStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Iteration: lowerer.compilation.registry.symbolID(lowerer.module, iteration), IterationType: iterationType, Name: value.Name, Kind: kind, Iterable: iterable, Snapshot: true, Body: lowerer.lowerBlock(value.Body)}
	case *ast.StateStmt:
		stateType := lowerType(lowerer.semantic.ExpressionTypes[value.Value])
		result := &ir.StateStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Temp: ir.TempID(fmt.Sprintf("%s::state@%d", lowerer.module.ID, value.Span().Start.Offset)), Value: lowerer.lowerExpr(value.Value), NoFallthrough: true}
		for _, condition := range value.Conditions {
			result.Cases = append(result.Cases, ir.StateCase{Match: lowerer.lowerExprExpected(condition.Match, stateType), Default: condition.Default, Body: lowerer.lowerBlock(condition.Body)})
		}
		return result
	case *ast.AttemptStmt:
		result := &ir.AttemptStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Body: lowerer.lowerBlock(value.Body), FinallyAlways: value.Ultimately != nil}
		for _, clause := range value.Excepts {
			class := lowerer.semantic.ResolvedSymbols[clause.Type]
			binding := findSymbol(lowerer.semantic, semantic.ExceptSymbol, clause.Name, clause.Span())
			handler := ir.ErrorHandler{Body: lowerer.lowerBlock(clause.Body), Binding: lowerer.compilation.registry.symbolID(lowerer.module, binding)}
			if class != nil {
				handler.Class = classID(class.Class)
			}
			result.Handlers = append(result.Handlers, handler)
		}
		if value.Ultimately != nil {
			block := lowerer.lowerBlock(value.Ultimately)
			result.Ultimately = &block
		}
		return result
	case *ast.TossStmt:
		expression := lowerer.lowerExpr(value.Value)
		classType, _ := lowerer.semantic.ExpressionTypes[value.Value].(types.Class)
		return &ir.TossStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Value: expression, ErrorClass: classID(classType.Symbol)}
	case *ast.ReturnStmt:
		return &ir.ReturnStmt{StmtBase: ir.StmtBase{Span: value.Span()}, Value: lowerer.lowerExprExpected(value.Value, lowerer.currentReturn), ReturnType: lowerer.currentReturn}
	case *ast.BreakStmt:
		return &ir.BreakStmt{StmtBase: ir.StmtBase{Span: value.Span()}}
	case *ast.ContinueStmt:
		return &ir.ContinueStmt{StmtBase: ir.StmtBase{Span: value.Span()}}
	case *ast.BringStmt:
		return nil
	case *ast.FunctionDecl, *ast.ClassDecl, *ast.StructureDecl:
		lowerer.compilation.error(CodeUnsupportedNode, fmt.Sprintf("nested declaration %T reached lowering", statement), statement.Span())
	}
	return nil
}

func (lowerer *moduleLowerer) lowerVariable(declaration *ast.VariableDecl) ir.Statement {
	symbol := lowerer.semantic.ResolvedSymbols[declaration]
	if symbol == nil {
		lowerer.compilation.error(CodeMissingSemantic, "variable declaration has no resolved Symbol", declaration.Span())
		return nil
	}
	if symbol.Alias != nil || hasModifier(declaration.Modifiers, ast.ModifierGlobal) || declaration.GlobalOnly {
		return nil
	}
	if _, identifier := declaration.Target.(*ast.IdentifierExpr); !identifier {
		target := lowerer.lowerTarget(declaration.Target)
		return &ir.AssignStmt{StmtBase: ir.StmtBase{Span: declaration.Span()}, Target: target, Value: lowerer.lowerExprExpected(declaration.Initializer, target.Type)}
	}
	typeValue := lowerType(symbol.Type)
	return &ir.BindingStmt{
		StmtBase: ir.StmtBase{Span: declaration.Span()}, Symbol: lowerer.compilation.registry.symbolID(lowerer.module, symbol), Name: symbol.Name,
		Type: typeValue, Constant: symbol.Constant, Storage: ir.LocalStorage, Initializer: lowerer.lowerExprExpected(declaration.Initializer, typeValue),
	}
}

func (lowerer *moduleLowerer) lowerTarget(expression ast.Expr) ir.Target {
	typeValue := lowerType(lowerer.semantic.ExpressionTypes[expression])
	switch value := expression.(type) {
	case *ast.IdentifierExpr:
		symbol := lowerer.semantic.ResolvedSymbols[value]
		return ir.Target{Kind: ir.SymbolTarget, Type: typeValue, Symbol: lowerer.compilation.registry.symbolID(lowerer.module, symbol)}
	case *ast.MemberExpr:
		symbol := lowerer.semantic.ResolvedSymbols[value]
		return ir.Target{Kind: ir.FieldTarget, Type: typeValue, Field: fieldID(symbol), Receiver: lowerer.lowerExpr(value.Object)}
	case *ast.IndexExpr:
		return ir.Target{Kind: ir.IndexTarget, Type: typeValue, Receiver: lowerer.lowerExpr(value.Object), Index: lowerer.lowerExprExpected(value.Index, ir.Type{Kind: ir.IntType})}
	default:
		lowerer.compilation.error(CodeUnsupportedNode, fmt.Sprintf("unsupported lvalue %T", expression), expression.Span())
		return ir.Target{Type: typeValue}
	}
}

func hasModifier(modifiers []ast.Modifier, wanted ast.Modifier) bool {
	for _, modifier := range modifiers {
		if modifier == wanted {
			return true
		}
	}
	return false
}
