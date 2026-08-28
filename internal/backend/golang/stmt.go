package golang

import (
	"fmt"
	"strconv"

	"ahdcode/internal/ir"
	"ahdcode/internal/source"
)

func (generator *generator) emitBlock(writer *emitter, block ir.Block) {
	for _, statement := range block.Statements {
		generator.emitStatement(writer, statement)
	}
}

func (generator *generator) emitStatement(writer *emitter, statement ir.Statement) {
	if statement == nil {
		return
	}
	switch value := statement.(type) {
	case *ir.ExprStmt:
		generator.emitExprStatement(writer, value)
	case *ir.BindingStmt:
		generator.emitBinding(writer, value)
	case *ir.AssignStmt:
		generator.emitAssign(writer, value)
	case *ir.CompoundAssignStmt:
		generator.emitCompoundAssign(writer, value)
	case *ir.UpdateStmt:
		generator.emitUpdate(writer, value)
	case *ir.IfStmt:
		generator.emitIf(writer, value)
	case *ir.WhileStmt:
		writer.open("for " + generator.value(value.Condition, ir.Type{Kind: ir.BoolType}, false) + " {")
		generator.emitBlock(writer, value.Body)
		writer.close("}")
	case *ir.DoUntilStmt:
		generator.emitDoUntil(writer, value)
	case *ir.ForStmt:
		generator.emitFor(writer, value)
	case *ir.StateStmt:
		generator.emitState(writer, value)
	case *ir.ReturnStmt:
		generator.emitReturn(writer, value)
	case *ir.BreakStmt:
		writer.line("break")
	case *ir.ContinueStmt:
		writer.line("continue")
	case *ir.AttemptStmt:
		generator.unsupported("attempt / except / ultimately error handling", value.Span)
	case *ir.TossStmt:
		generator.unsupported("toss error raising", value.Span)
	default:
		generator.unsupported(fmt.Sprintf("IR statement %T", statement), statement.StatementSpan())
	}
}

// emitExprStatement keeps a Nothing-valued call as a Go statement. Every other
// expression statement is discarded into the blank identifier so that its
// evaluation, and any runtime check it performs, still happens.
func (generator *generator) emitExprStatement(writer *emitter, value *ir.ExprStmt) {
	if value.Value == nil {
		return
	}
	code := generator.expr(value.Value)
	if value.Value.ExprMeta().Type.Kind == ir.NothingType {
		writer.line(code)
		return
	}
	writer.line("_ = " + code)
}

func (generator *generator) emitBinding(writer *emitter, value *ir.BindingStmt) {
	current := generator.slots[value.Symbol]
	rendered := generator.goType(current.typeInfo, current.nullable)
	if rendered == "" {
		generator.fail(CodeInvalidRepresentation, "binding "+value.Name+" has no Go representation", value.Span, "use a v0.1 representable declared type")
		return
	}
	if value.Initializer == nil {
		writer.line("var " + current.name + " " + rendered)
		writer.line("_ = " + current.name)
		return
	}
	writer.line("var " + current.name + " " + rendered + " = " + generator.value(value.Initializer, current.typeInfo, current.nullable))
	writer.line("_ = " + current.name)
}

func (generator *generator) emitAssign(writer *emitter, value *ir.AssignStmt) {
	switch value.Target.Kind {
	case ir.SymbolTarget:
		current := generator.slots[value.Target.Symbol]
		writer.line(current.name + " = " + generator.value(value.Value, current.typeInfo, current.nullable))
	case ir.FieldTarget:
		field, known := generator.fields[value.Target.Field]
		if !known {
			generator.fail(CodeGenerationFailure, "unknown FieldID "+string(value.Target.Field), value.Span, "the IR references a field with no declaration")
			return
		}
		nullable := generator.nullFields[value.Target.Field]
		writer.line(generator.expr(value.Target.Receiver) + "." + generator.fieldName(value.Target.Field) + " = " + generator.value(value.Value, field.Type, nullable))
	case ir.IndexTarget:
		generator.emitIndexAssign(writer, value)
	default:
		generator.unsupported("assignment target kind "+string(value.Target.Kind), value.Span)
	}
}

func (generator *generator) emitIndexAssign(writer *emitter, value *ir.AssignStmt) {
	container := value.Target.Receiver.ExprMeta().Type
	receiver, index := generator.temporaryName(), generator.temporaryName()
	writer.open("{")
	writer.line(receiver + " := " + generator.expr(value.Target.Receiver))
	switch container.Kind {
	case ir.ListType:
		if container.Element == nil {
			generator.unsupported("indexed assignment into an untyped List", value.Span)
			break
		}
		writer.line(index + " := " + generator.value(value.Target.Index, ir.Type{Kind: ir.IntType}, false))
		writer.line(receiver + ".Set(" + index + ", " + generator.value(value.Value, *container.Element, true) + ")")
	case ir.PairType:
		if container.Key == nil || container.Value == nil {
			generator.unsupported("indexed assignment into an untyped Pair", value.Span)
			break
		}
		writer.line(index + " := " + generator.value(value.Target.Index, *container.Key, false))
		writer.line(receiver + ".Set(" + index + ", " + generator.value(value.Value, *container.Value, true) + ")")
	default:
		generator.unsupported("indexed assignment into "+container.String(), value.Span)
	}
	writer.close("}")
}

// emitCompoundAssign evaluates the receiver and index exactly once, so a
// compound assignment never re-evaluates its target expression.
func (generator *generator) emitCompoundAssign(writer *emitter, value *ir.CompoundAssignStmt) {
	generator.emitTargetUpdate(writer, value.Target, value.Span, func(read string) string {
		return generator.applyOperation(value.Op, read, generator.value(value.Value, value.Target.Type, false), value)
	})
}

func (generator *generator) emitUpdate(writer *emitter, value *ir.UpdateStmt) {
	operation := ir.BinaryOp("CheckedIntAdd")
	if value.Delta < 0 {
		operation = "CheckedIntSubtract"
	}
	generator.emitTargetUpdate(writer, value.Target, value.Span, func(read string) string {
		if value.Target.Type.Kind == ir.RealType {
			if value.Delta < 0 {
				return "AhdRealSubtract(" + read + ", float64(1.0))"
			}
			return "AhdRealAdd(" + read + ", float64(1.0))"
		}
		return intOperation(operation) + "(" + read + ", int64(1))"
	})
}

// applyOperation renders a compound arithmetic step by reusing the ordinary
// binary operator lowering table.
func (generator *generator) applyOperation(operation ir.BinaryOp, left, right string, node ir.Statement) string {
	switch operation {
	case "CheckedIntAdd", "CheckedIntSubtract", "CheckedIntMultiply", "CheckedIntPower", "IntModulo":
		return intOperation(operation) + "(" + left + ", " + right + ")"
	case "RealAdd", "RealSubtract", "RealMultiply", "RealDivide", "RealPower":
		return "Ahd" + string(operation) + "(" + left + ", " + right + ")"
	case "StringConcat":
		return "(" + left + " + " + right + ")"
	case "StringRepeat":
		return "AhdStringRepeat(" + left + ", " + right + ")"
	case "ListConcat":
		return "AhdListConcat(" + left + ", " + right + ")"
	default:
		return generator.unsupported("compound operation "+string(operation), node.StatementSpan())
	}
}

func (generator *generator) emitTargetUpdate(writer *emitter, target ir.Target, span source.Span, combine func(read string) string) {
	switch target.Kind {
	case ir.SymbolTarget:
		current := generator.slots[target.Symbol]
		read := generator.coerce(current.name, ir.ExprBase{Type: current.typeInfo, NullState: nullState(current.nullable)}, target.Type, false)
		updated := combine(read)
		writer.line(current.name + " = " + generator.coerce(updated, ir.ExprBase{Type: target.Type, NullState: ir.NonNull}, current.typeInfo, current.nullable))
	case ir.FieldTarget:
		field, known := generator.fields[target.Field]
		if !known {
			generator.fail(CodeGenerationFailure, "unknown FieldID "+string(target.Field), span, "the IR references a field with no declaration")
			return
		}
		receiver := generator.temporaryName()
		nullable := generator.nullFields[target.Field]
		writer.open("{")
		writer.line(receiver + " := " + generator.expr(target.Receiver))
		access := receiver + "." + generator.fieldName(target.Field)
		read := generator.coerce(access, ir.ExprBase{Type: field.Type, NullState: nullState(nullable)}, target.Type, false)
		updated := combine(read)
		writer.line(access + " = " + generator.coerce(updated, ir.ExprBase{Type: target.Type, NullState: ir.NonNull}, field.Type, nullable))
		writer.close("}")
	case ir.IndexTarget:
		container := target.Receiver.ExprMeta().Type
		element, key := elementAndKey(container)
		if element == nil {
			generator.unsupported("compound assignment into "+container.String(), span)
			return
		}
		receiver, index := generator.temporaryName(), generator.temporaryName()
		writer.open("{")
		writer.line(receiver + " := " + generator.expr(target.Receiver))
		if key != nil {
			writer.line(index + " := " + generator.value(target.Index, *key, false))
		} else {
			writer.line(index + " := " + generator.value(target.Index, ir.Type{Kind: ir.IntType}, false))
		}
		read := generator.coerce(readElement(receiver, index, container), ir.ExprBase{Type: *element, NullState: ir.MaybeNull}, target.Type, false)
		updated := combine(read)
		writer.line(receiver + ".Set(" + index + ", " + generator.coerce(updated, ir.ExprBase{Type: target.Type, NullState: ir.NonNull}, *element, true) + ")")
		writer.close("}")
	default:
		generator.unsupported("assignment target kind "+string(target.Kind), span)
	}
}

func readElement(receiver, index string, container ir.Type) string {
	if container.Kind == ir.PairType {
		return receiver + ".Get(" + index + ")"
	}
	return receiver + ".At(" + index + ")"
}

func elementAndKey(container ir.Type) (*ir.Type, *ir.Type) {
	switch container.Kind {
	case ir.ListType:
		return container.Element, nil
	case ir.PairType:
		return container.Value, container.Key
	default:
		return nil, nil
	}
}

func (generator *generator) emitIf(writer *emitter, value *ir.IfStmt) {
	for index, branch := range value.Branches {
		keyword := "if "
		if index > 0 {
			keyword = "} else if "
			writer.close("")
		}
		writer.open(keyword + generator.value(branch.Condition, ir.Type{Kind: ir.BoolType}, false) + " {")
		generator.emitBlock(writer, branch.Body)
	}
	if len(value.Branches) == 0 {
		return
	}
	if value.Else != nil {
		writer.close("} else {")
		writer.indent++
		generator.emitBlock(writer, *value.Else)
	}
	writer.close("}")
}

// emitDoUntil preserves post-check semantics and keeps continue on the edge
// that still evaluates the until condition.
func (generator *generator) emitDoUntil(writer *emitter, value *ir.DoUntilStmt) {
	first := generator.temporaryName()
	writer.line(first + " := true")
	writer.open("for {")
	writer.open("if !" + first + " {")
	writer.open("if " + generator.value(value.Condition, ir.Type{Kind: ir.BoolType}, false) + " {")
	writer.line("break")
	writer.close("}")
	writer.close("}")
	writer.line(first + " = false")
	generator.emitBlock(writer, value.Body)
	writer.close("}")
}

// emitFor materializes the shallow iteration snapshot demanded by the IR
// instead of relying on Go range semantics over live storage.
func (generator *generator) emitFor(writer *emitter, value *ir.ForStmt) {
	current := generator.slots[value.Iteration]
	rendered := generator.goType(current.typeInfo, current.nullable)
	if rendered == "" {
		generator.fail(CodeInvalidRepresentation, "for iteration variable has no Go representation", value.Span, "iterate a v0.1 representable List, Pair, or String")
		return
	}
	snapshot := generator.temporaryName()
	item := generator.temporaryName()
	var source string
	var element ir.Type
	container := value.Iterable.ExprMeta().Type
	switch value.Kind {
	case ir.ListElements:
		if container.Element == nil {
			generator.unsupported("iteration over an untyped List", value.Span)
			return
		}
		source = generator.expr(value.Iterable) + ".Snapshot()"
		element = *container.Element
		writer.line(snapshot + " := " + source)
		writer.open("for _, " + item + " := range " + snapshot + " {")
		writer.line(current.name + " := " + generator.coerce(item, ir.ExprBase{Type: element, NullState: ir.MaybeNull}, current.typeInfo, current.nullable))
	case ir.PairKeys:
		if container.Key == nil {
			generator.unsupported("iteration over an untyped Pair", value.Span)
			return
		}
		writer.line(snapshot + " := " + generator.expr(value.Iterable) + ".Keys()")
		writer.open("for _, " + item + " := range " + snapshot + " {")
		writer.line(current.name + " := " + generator.coerce(item, ir.ExprBase{Type: *container.Key, NullState: ir.NonNull}, current.typeInfo, current.nullable))
	case ir.StringCharacters:
		writer.line(snapshot + " := AhdStringChars(" + generator.value(value.Iterable, ir.Type{Kind: ir.StringType}, false) + ")")
		writer.open("for _, " + item + " := range " + snapshot + " {")
		writer.line(current.name + " := " + generator.coerce(item, ir.ExprBase{Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull}, current.typeInfo, current.nullable))
	default:
		generator.unsupported("iteration kind "+string(value.Kind), value.Span)
		return
	}
	writer.line("_ = " + current.name)
	generator.emitBlock(writer, value.Body)
	writer.close("}")
}

// emitState evaluates the state expression exactly once and never falls
// through between conditions.
func (generator *generator) emitState(writer *emitter, value *ir.StateStmt) {
	subject := generator.temporaryName()
	stateType := value.Value.ExprMeta().Type
	nullable := naturalNullable(value.Value)
	writer.open("{")
	writer.line(subject + " := " + generator.value(value.Value, stateType, nullable))
	writer.line("_ = " + subject)
	writer.open("switch {")
	var fallback *ir.StateCase
	for index := range value.Cases {
		item := value.Cases[index]
		if item.Default {
			fallback = &value.Cases[index]
			continue
		}
		if item.Match == nil {
			generator.fail(CodeGenerationFailure, "state condition has no match value", value.Span, "the IR node is incomplete")
			continue
		}
		matchNullable := nullable || naturalNullable(item.Match)
		comparer := generator.equalFunc(stateType, matchNullable, value.Span)
		left := generator.coerce(subject, ir.ExprBase{Type: stateType, NullState: nullState(nullable)}, stateType, matchNullable)
		writer.open("case " + comparer + "(" + left + ", " + generator.value(item.Match, stateType, matchNullable) + "):")
		generator.emitBlock(writer, item.Body)
		writer.indent--
	}
	if fallback != nil {
		writer.open("default:")
		generator.emitBlock(writer, fallback.Body)
		writer.indent--
	}
	writer.close("}")
	writer.close("}")
}

func (generator *generator) emitReturn(writer *emitter, value *ir.ReturnStmt) {
	if value.Value == nil {
		writer.line("return")
		return
	}
	writer.line("return " + generator.value(value.Value, value.ReturnType, false))
}

func (generator *generator) temporaryName() string {
	generator.temporary++
	return "ahdTemporary" + strconv.Itoa(generator.temporary)
}
