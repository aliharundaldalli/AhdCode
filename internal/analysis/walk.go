package analysis

import (
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
)

// findNodeAtOffset returns the innermost AST node whose source span covers
// the given byte offset, or nil if the offset falls outside the program
// entirely. This is a purely structural tree walk -- span containment only
// -- with no type-checking or name-resolution knowledge of its own; it
// exists only to turn a cursor position into a lookup key for the semantic
// analyzer's own ResolvedSymbols/ExpressionTypes facts.
//
// Containment is inclusive on both ends (unlike source.Span's own
// half-open [Start, End) convention) because a hover request commonly lands
// exactly on the boundary between two tokens -- right after the last
// character of an identifier is a normal cursor position, and it should
// still resolve to that identifier.
func findNodeAtOffset(program *ast.Program, offset int) ast.Node {
	if program == nil {
		return nil
	}
	return descend(program, offset)
}

func descend(node ast.Node, offset int) ast.Node {
	if node == nil || !containsOffset(node.Span(), offset) {
		return nil
	}
	for _, child := range children(node) {
		if found := descend(child, offset); found != nil {
			return found
		}
	}
	return node
}

func containsOffset(span source.Span, offset int) bool {
	return offset >= span.Start.Offset && offset <= span.End.Offset
}

// children enumerates one node's immediate structural children, in source
// order, skipping absent (nil) ones. It is a mechanical reflection of the
// grammar already defined by internal/syntax/ast -- adding a case here
// costs nothing semantically; it only decides which sub-span to search
// next.
func children(node ast.Node) []ast.Node {
	switch n := node.(type) {
	case *ast.Program:
		return stmtNodes(n.Statements)
	case *ast.Block:
		return stmtNodes(n.Statements)
	case *ast.ExprStmt:
		return exprNodes(n.Expression)
	case *ast.VariableDecl:
		out := exprNodes(n.Target)
		out = append(out, typeRefNodes(n.Type)...)
		out = append(out, exprNodes(n.Initializer)...)
		return out
	case *ast.AssignmentStmt:
		out := exprNodes(n.Target)
		return append(out, exprNodes(n.Value)...)
	case *ast.IncDecStmt:
		return exprNodes(n.Target)
	case *ast.ReturnStmt:
		return exprNodes(n.Value)
	case *ast.TossStmt:
		return exprNodes(n.Value)
	case *ast.IfStmt:
		var out []ast.Node
		for index := range n.Branches {
			out = append(out, exprNodes(n.Branches[index].Condition)...)
			out = append(out, blockNodes(n.Branches[index].Body)...)
		}
		return append(out, blockNodes(n.Else)...)
	case *ast.WhileStmt:
		return append(exprNodes(n.Condition), blockNodes(n.Body)...)
	case *ast.UntilStmt:
		return append(exprNodes(n.Condition), blockNodes(n.Body)...)
	case *ast.ForStmt:
		out := typeRefNodes(n.Type)
		out = append(out, exprNodes(n.Iterable)...)
		return append(out, blockNodes(n.Body)...)
	case *ast.StateStmt:
		out := exprNodes(n.Value)
		for index := range n.Conditions {
			out = append(out, exprNodes(n.Conditions[index].Match)...)
			out = append(out, blockNodes(n.Conditions[index].Body)...)
		}
		return out
	case *ast.AttemptStmt:
		out := blockNodes(n.Body)
		for index := range n.Excepts {
			out = append(out, typeRefNodes(n.Excepts[index].Type)...)
			out = append(out, blockNodes(n.Excepts[index].Body)...)
		}
		return append(out, blockNodes(n.Ultimately)...)
	case *ast.BringStmt:
		return nil
	case *ast.FunctionDecl:
		out := parameterNodes(n.Parameters)
		out = append(out, typeRefNodes(n.ReturnType)...)
		return append(out, blockNodes(n.Body)...)
	case *ast.ClassDecl:
		out := typeRefNodes(n.Parent)
		return append(out, stmtNodes(n.Members)...)
	case *ast.StructureDecl:
		return append(parameterNodes(n.Parameters), blockNodes(n.Body)...)
	case *ast.GroupExpr:
		return exprNodes(n.Expression)
	case *ast.LambdaExpr:
		out := make([]ast.Node, 0, len(n.Captures)+len(n.Parameters)+1)
		for index := range n.Captures {
			out = append(out, &n.Captures[index])
		}
		out = append(out, parameterNodes(n.Parameters)...)
		return append(out, exprNodes(n.Body)...)
	case *ast.UnaryExpr:
		return exprNodes(n.Operand)
	case *ast.BinaryExpr:
		return append(exprNodes(n.Left), exprNodes(n.Right)...)
	case *ast.CallExpr:
		out := exprNodes(n.Callee)
		for index := range n.Arguments {
			out = append(out, exprNodes(n.Arguments[index].Value)...)
		}
		return out
	case *ast.MemberExpr:
		return exprNodes(n.Object)
	case *ast.IndexExpr:
		return append(exprNodes(n.Object), exprNodes(n.Index)...)
	case *ast.SliceExpr:
		out := exprNodes(n.Object)
		out = append(out, exprNodes(n.Start)...)
		return append(out, exprNodes(n.End)...)
	case *ast.ListExpr:
		var out []ast.Node
		for _, element := range n.Elements {
			out = append(out, exprNodes(element)...)
		}
		return out
	case *ast.PairExpr:
		var out []ast.Node
		for index := range n.Entries {
			out = append(out, exprNodes(n.Entries[index].Key)...)
			out = append(out, exprNodes(n.Entries[index].Value)...)
		}
		return out
	case *ast.StringExpr:
		var out []ast.Node
		for index := range n.Parts {
			out = append(out, exprNodes(n.Parts[index].Expression)...)
		}
		return out
	case *ast.TypeRef:
		out := make([]ast.Node, 0, len(n.Arguments))
		for _, argument := range n.Arguments {
			if argument != nil {
				out = append(out, argument)
			}
		}
		return out
	case *ast.Parameter:
		return append(typeRefNodes(n.Type), exprNodes(n.Default)...)
	default:
		// LiteralExpr, IdentifierExpr, CaptureRef, BadExpr, BadStmt,
		// BreakStmt, ContinueStmt: leaves with no children to search.
		return nil
	}
}

func exprNodes(expression ast.Expr) []ast.Node {
	if expression == nil {
		return nil
	}
	return []ast.Node{expression}
}

func blockNodes(block *ast.Block) []ast.Node {
	if block == nil {
		return nil
	}
	return []ast.Node{block}
}

func typeRefNodes(typeRef *ast.TypeRef) []ast.Node {
	if typeRef == nil {
		return nil
	}
	return []ast.Node{typeRef}
}

func stmtNodes(statements []ast.Stmt) []ast.Node {
	out := make([]ast.Node, 0, len(statements))
	for _, statement := range statements {
		if statement != nil {
			out = append(out, statement)
		}
	}
	return out
}

func parameterNodes(parameters []ast.Parameter) []ast.Node {
	out := make([]ast.Node, 0, len(parameters))
	for index := range parameters {
		out = append(out, &parameters[index])
	}
	return out
}
