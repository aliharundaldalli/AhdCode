package ir

import (
	"fmt"
	"strconv"
	"strings"
)

// Dump returns a deterministic, golden-test-friendly implementation IR.
func Dump(compilation *Compilation) string {
	if compilation == nil {
		return "<nil compilation>\n"
	}
	var out strings.Builder
	fmt.Fprintf(&out, "entry %s\n", compilation.Entry)
	for _, module := range compilation.Modules {
		if module == nil {
			continue
		}
		fmt.Fprintf(&out, "module %s deps=[%s]\n", module.ID, joinModuleIDs(module.Dependencies))
		for _, global := range module.Globals {
			fmt.Fprintf(&out, "  global %s : %s\n", global.ID, global.Type)
		}
		for _, class := range module.Classes {
			builtin := ""
			if class.Builtin {
				builtin = " builtin"
			}
			fmt.Fprintf(&out, "  class %s parent=%s ctor=%s%s\n", class.ID, class.Parent, class.Constructor, builtin)
			for _, field := range class.Fields {
				fmt.Fprintf(&out, "    field %s : %s\n", field.ID, field.Type)
			}
			for _, method := range class.Methods {
				fmt.Fprintf(&out, "    method %s\n", method)
			}
		}
		for _, function := range module.Functions {
			printFunction(&out, function)
		}
		if len(module.Init.Statements) != 0 {
			out.WriteString("  init ")
			printBlock(&out, module.Init, "  ")
		}
	}
	return out.String()
}

func printFunction(out *strings.Builder, function *Function) {
	if function == nil {
		return
	}
	fmt.Fprintf(out, "  fn %s(", function.ID)
	for index, parameter := range function.Parameters {
		if index > 0 {
			out.WriteString(", ")
		}
		fmt.Fprintf(out, "%s:%s", parameter.ID, parameter.Type)
		if parameter.Default != nil {
			fmt.Fprintf(out, "=%s", printExpr(parameter.Default))
		}
	}
	fmt.Fprintf(out, ") -> %s null=%s ", function.Signature.Return, function.ReturnNull)
	if function.Overrides != "" {
		fmt.Fprintf(out, "overrides=%s ", function.Overrides)
	}
	if function.ParentConstructor != "" {
		fmt.Fprintf(out, "parent-ctor=%s%v ", function.ParentConstructor, function.ParentArguments)
	}
	printBlock(out, function.Body, "  ")
}

func printBlock(out *strings.Builder, block Block, indent string) {
	out.WriteString("{\n")
	for _, statement := range block.Statements {
		printStatement(out, statement, indent+"  ")
	}
	fmt.Fprintf(out, "%s}\n", indent)
}

func printStatement(out *strings.Builder, statement Statement, indent string) {
	fmt.Fprint(out, indent)
	switch value := statement.(type) {
	case *ExprStmt:
		fmt.Fprintf(out, "expr %s\n", printExpr(value.Value))
	case *BindingStmt:
		fmt.Fprintf(out, "bind[%s] %s : %s = %s\n", value.Storage, value.Symbol, value.Type, printExpr(value.Initializer))
	case *AssignStmt:
		fmt.Fprintf(out, "assign %s = %s\n", printTarget(value.Target), printExpr(value.Value))
	case *CompoundAssignStmt:
		fmt.Fprintf(out, "compound %s %s %s\n", printTarget(value.Target), value.Op, printExpr(value.Value))
	case *UpdateStmt:
		fmt.Fprintf(out, "update %s delta=%d\n", printTarget(value.Target), value.Delta)
	case *IfStmt:
		out.WriteString("if\n")
		for _, branch := range value.Branches {
			fmt.Fprintf(out, "%s  when %s ", indent, printExpr(branch.Condition))
			printBlock(out, branch.Body, indent+"  ")
		}
		if value.Else != nil {
			fmt.Fprintf(out, "%s  else ", indent)
			printBlock(out, *value.Else, indent+"  ")
		}
	case *WhileStmt:
		fmt.Fprintf(out, "while %s ", printExpr(value.Condition))
		printBlock(out, value.Body, indent)
	case *DoUntilStmt:
		out.WriteString("do ")
		printBlock(out, value.Body, indent)
		fmt.Fprintf(out, "%suntil %s continue-check=true\n", indent, printExpr(value.Condition))
	case *ForStmt:
		fmt.Fprintf(out, "for %s in snapshot[%s](%s) ", value.Iteration, value.Kind, printExpr(value.Iterable))
		printBlock(out, value.Body, indent)
	case *StateStmt:
		fmt.Fprintf(out, "state %s = %s no-fallthrough {\n", value.Temp, printExpr(value.Value))
		for _, item := range value.Cases {
			fmt.Fprintf(out, "%s  case %s ", indent, printExpr(item.Match))
			if item.Default {
				out.WriteString("default ")
			}
			printBlock(out, item.Body, indent+"  ")
		}
		fmt.Fprintf(out, "%s}\n", indent)
	case *AttemptStmt:
		out.WriteString("attempt ")
		printBlock(out, value.Body, indent)
		for _, handler := range value.Handlers {
			fmt.Fprintf(out, "%sexcept %s as %s ", indent, handler.Class, handler.Binding)
			printBlock(out, handler.Body, indent)
		}
		if value.Ultimately != nil {
			fmt.Fprintf(out, "%sultimately(always) ", indent)
			printBlock(out, *value.Ultimately, indent)
		}
	case *TossStmt:
		fmt.Fprintf(out, "toss[%s] %s\n", value.ErrorClass, printExpr(value.Value))
	case *ReturnStmt:
		if value.Value == nil {
			out.WriteString("return\n")
		} else {
			fmt.Fprintf(out, "return %s\n", printExpr(value.Value))
		}
	case *BreakStmt:
		out.WriteString("break\n")
	case *ContinueStmt:
		out.WriteString("continue\n")
	default:
		fmt.Fprintf(out, "<%T>\n", statement)
	}
}

func printTarget(target Target) string {
	switch target.Kind {
	case SymbolTarget:
		return "%" + string(target.Symbol)
	case FieldTarget:
		return "field(" + printExpr(target.Receiver) + "," + string(target.Field) + ")"
	case IndexTarget:
		return "index(" + printExpr(target.Receiver) + "," + printExpr(target.Index) + ")"
	default:
		return "<target>"
	}
}

func printExpr(expression Expr) string {
	if expression == nil {
		return "<none>"
	}
	switch value := expression.(type) {
	case *LiteralExpr:
		return strings.ToLower(string(value.Kind)) + "(" + strconv.Quote(value.Value) + ")"
	case *NullExpr:
		return "null<" + value.Type.String() + ">"
	case *LoadExpr:
		return "load(" + string(value.Symbol) + ")"
	case *ClassRefExpr:
		return "classref(" + string(value.Class) + ")"
	case *FunctionValueExpr:
		return "fnvalue(" + string(value.Callable) + ")"
	case *UnaryExpr:
		return string(value.Op) + "(" + printExpr(value.Operand) + ")"
	case *BinaryExpr:
		return string(value.Op) + "(" + printExpr(value.Left) + "," + printExpr(value.Right) + ")"
	case *ConvertExpr:
		return "convert[" + value.From.String() + "->" + value.Type.String() + "](" + printExpr(value.Value) + ")"
	case *CallExpr:
		return "call(" + string(value.Callable) + "," + printArguments(value.Arguments) + ")"
	case *ConstructExpr:
		return "construct(" + string(value.Class) + "," + printArguments(value.Arguments) + ")"
	case *MemberExpr:
		if value.Kind == FieldMember {
			return "field(" + printExpr(value.Object) + "," + string(value.Field) + ")"
		}
		if value.Direct {
			return "direct-method(" + printExpr(value.Object) + "," + string(value.Callable) + ")"
		}
		return "method(" + printExpr(value.Object) + "," + string(value.Callable) + ")"
	case *IndexExpr:
		return "index(" + printExpr(value.Object) + "," + printExpr(value.Index) + ")"
	case *SliceExpr:
		return "slice(" + printExpr(value.Object) + "," + printExpr(value.Start) + "," + printExpr(value.End) + ")"
	case *ListExpr:
		parts := make([]string, len(value.Elements))
		for i, item := range value.Elements {
			parts[i] = printExpr(item)
		}
		return "list<" + value.ElementType.String() + ">[" + strings.Join(parts, ",") + "]"
	case *PairExpr:
		parts := make([]string, len(value.Entries))
		for i, item := range value.Entries {
			parts[i] = printExpr(item.Key) + ":" + printExpr(item.Value)
		}
		return "pair<" + value.KeyType.String() + "," + value.ValueType.String() + ">{" + strings.Join(parts, ",") + "}"
	case *StringExpr:
		parts := make([]string, len(value.Parts))
		for i, part := range value.Parts {
			if part.ToString != nil {
				parts[i] = printExpr(part.ToString)
			} else {
				parts[i] = strconv.Quote(part.Literal)
			}
		}
		return "stringparts[" + strings.Join(parts, ",") + "]"
	case *ToStringExpr:
		return "tostring(" + printExpr(value.Value) + ")"
	case *IdentityExpr:
		return "id(" + printExpr(value.Value) + ")"
	case *TypeNameExpr:
		return "type(" + printExpr(value.Value) + ")"
	default:
		return fmt.Sprintf("<%T>", expression)
	}
}

func printArguments(arguments []Argument) string {
	parts := make([]string, len(arguments))
	for i, argument := range arguments {
		if argument.UsesDefault {
			parts[i] = fmt.Sprintf("%d:%s=<default>", argument.ParameterIndex, argument.ParameterName)
		} else {
			parts[i] = fmt.Sprintf("%d:%s=%s", argument.ParameterIndex, argument.ParameterName, printExpr(argument.Value))
		}
	}
	return strings.Join(parts, ",")
}
func joinModuleIDs(ids []ModuleID) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = string(id)
	}
	return strings.Join(parts, ",")
}
