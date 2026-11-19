package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

// markReachableSQLite sets usesSQLite only when a call that needs the helper
// can actually run. Defining Web.persistentSessions is not a use. Calling it,
// or calling HTTP.persistentSessions or any SQLite operation through a
// reachable wrapper, is. str, string interpolation, write, and Terminal.pretty
// of a class value dispatch to that class's CStr method, so a reachable
// conversion also reaches that method and its overrides. A CStr method that
// nothing converts is left unreached.
func (generator *generator) markReachableSQLite() {
	if generator.compilation == nil {
		return
	}
	functions := map[ir.CallableID]*ir.Function{}
	overrides := map[ir.CallableID][]ir.CallableID{}
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		for _, function := range module.Functions {
			if function == nil {
				continue
			}
			functions[function.ID] = function
			if function.Overrides != "" {
				overrides[function.Overrides] = append(overrides[function.Overrides], function.ID)
			}
		}
	}
	seen := map[ir.CallableID]bool{}
	var visitBlock func(ir.Block)
	var visitExpr func(ir.Expr)
	var visitFunction func(ir.CallableID)
	note := func(id ir.CallableID) {
		text := string(id)
		if strings.HasPrefix(text, "builtin:SQLite::") || text == "builtin:HTTP::persistentSessions" || strings.HasPrefix(text, "builtin:HTTP::persistentSessions::") {
			generator.usesSQLite = true
		}
	}
	visitFunction = func(id ir.CallableID) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		note(id)
		for _, child := range overrides[id] {
			visitFunction(child)
		}
		function := functions[id]
		if function == nil {
			return
		}
		if function.ParentConstructor != "" {
			visitFunction(function.ParentConstructor)
		}
		for _, parameter := range function.Parameters {
			visitExpr(parameter.Default)
		}
		visitBlock(function.Body)
	}
	// followText walks a value that code generation stringifies. A class
	// value calls its CStr method, and that method's overrides can run
	// through dynamic dispatch. Other types have no protocol call.
	var followText func(ir.Expr)
	followText = func(expression ir.Expr) {
		visitExpr(expression)
		if expression == nil || expression.ExprMeta().Type.Kind != ir.ClassType {
			return
		}
		if method := generator.findProtocolMethod(expression.ExprMeta().Type.Class, "CStr"); method != nil {
			visitFunction(method.ID)
		}
	}
	visitExpr = func(expression ir.Expr) {
		if expression == nil {
			return
		}
		switch value := expression.(type) {
		case *ir.LoadExpr, *ir.LiteralExpr, *ir.NullExpr, *ir.ClassRefExpr:
		case *ir.FunctionValueExpr:
			for _, capture := range value.Captures {
				visitExpr(capture)
			}
			visitFunction(value.Callable)
		case *ir.UnaryExpr:
			visitExpr(value.Operand)
		case *ir.BinaryExpr:
			visitExpr(value.Left)
			visitExpr(value.Right)
		case *ir.ConvertExpr:
			visitExpr(value.Value)
		case *ir.CallExpr:
			visitFunction(value.Callable)
			visitExpr(value.Callee)
			if argument, stringify := stringifiedArgument(value); stringify {
				followText(argument)
				break
			}
			for _, argument := range value.Arguments {
				visitExpr(argument.Value)
			}
		case *ir.ConstructExpr:
			visitFunction(value.Constructor)
			for _, argument := range value.Arguments {
				visitExpr(argument.Value)
			}
		case *ir.MemberExpr:
			visitExpr(value.Object)
			if value.Callable != "" {
				visitFunction(value.Callable)
			}
		case *ir.IndexExpr:
			visitExpr(value.Object)
			visitExpr(value.Index)
		case *ir.SliceExpr:
			visitExpr(value.Object)
			visitExpr(value.Start)
			visitExpr(value.End)
		case *ir.ListExpr:
			for _, element := range value.Elements {
				visitExpr(element)
			}
		case *ir.PairExpr:
			for _, entry := range value.Entries {
				visitExpr(entry.Key)
				visitExpr(entry.Value)
			}
		case *ir.StringExpr:
			for _, part := range value.Parts {
				visitExpr(part.ToString)
			}
		case *ir.ToStringExpr:
			followText(value.Value)
		case *ir.IdentityExpr:
			visitExpr(value.Value)
		case *ir.TypeNameExpr:
			visitExpr(value.Value)
		}
	}
	visitTarget := func(target ir.Target) {
		visitExpr(target.Receiver)
		visitExpr(target.Index)
	}
	visitBlock = func(block ir.Block) {
		for _, statement := range block.Statements {
			if statement == nil {
				continue
			}
			switch value := statement.(type) {
			case *ir.ExprStmt:
				visitExpr(value.Value)
			case *ir.BindingStmt:
				if function, ok := value.Initializer.(*ir.FunctionValueExpr); ok && function.Symbol == value.Symbol && !function.UseStorage {
					for _, capture := range function.Captures {
						visitExpr(capture)
					}
					continue
				}
				visitExpr(value.Initializer)
			case *ir.AssignStmt:
				visitTarget(value.Target)
				visitExpr(value.Value)
			case *ir.CompoundAssignStmt:
				visitTarget(value.Target)
				visitExpr(value.Value)
				visitFunction(value.Protocol)
			case *ir.UpdateStmt:
				visitTarget(value.Target)
			case *ir.IfStmt:
				for _, branch := range value.Branches {
					visitExpr(branch.Condition)
					visitBlock(branch.Body)
				}
				if value.Else != nil {
					visitBlock(*value.Else)
				}
			case *ir.WhileStmt:
				visitExpr(value.Condition)
				visitBlock(value.Body)
			case *ir.DoUntilStmt:
				visitBlock(value.Body)
				visitExpr(value.Condition)
			case *ir.ForStmt:
				visitExpr(value.Iterable)
				visitBlock(value.Body)
			case *ir.StateStmt:
				visitExpr(value.Value)
				for _, item := range value.Cases {
					visitExpr(item.Match)
					visitBlock(item.Body)
				}
			case *ir.AttemptStmt:
				visitBlock(value.Body)
				for _, handler := range value.Handlers {
					visitBlock(handler.Body)
				}
				if value.Ultimately != nil {
					visitBlock(*value.Ultimately)
				}
			case *ir.TossStmt:
				visitExpr(value.Value)
			case *ir.ReturnStmt:
				visitExpr(value.Value)
			}
		}
	}
	for _, module := range generator.compilation.Modules {
		if module != nil {
			visitBlock(module.Init)
		}
	}
}

// stringifiedArgument reports the argument that write or Terminal.pretty
// passes through the same text conversion as str. pretty of a List or Pair
// lays the collection out without calling CStr on the collection itself.
func stringifiedArgument(call *ir.CallExpr) (ir.Expr, bool) {
	if call == nil || len(call.Arguments) != 1 || call.Arguments[0].Value == nil {
		return nil, false
	}
	id := string(call.Callable)
	argument := call.Arguments[0].Value
	switch {
	case id == "builtin:core::write":
		return argument, true
	case id == "builtin:Terminal::pretty" || strings.HasPrefix(id, "builtin:Terminal::pretty::"):
		kind := argument.ExprMeta().Type.Kind
		if kind == ir.ListType || kind == ir.PairType {
			return nil, false
		}
		return argument, true
	default:
		return nil, false
	}
}
