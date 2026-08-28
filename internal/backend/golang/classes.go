package golang

import (
	"sort"
	"strings"

	"ahdcode/internal/ir"
)

// slot is one dynamic dispatch position of a Class. Every override of a method
// shares the slot introduced by the ancestor that first declared it.
type slot struct {
	root ir.CallableID
	impl *ir.Function
}

// layout is the deterministic runtime shape of one Class: its ancestry, the
// full field set, and the full dispatch table.
type layout struct {
	class     *ir.Class
	parent    *layout
	ownFields []ir.Field
	allFields []ir.Field
	ownSlots  []slot
	allSlots  []slot
}

// buildLayouts resolves inheritance into concrete object layouts. Classes are
// visited parent-before-child so a subclass sees a complete parent table.
func (generator *generator) buildLayouts() {
	generator.layouts = make(map[ir.ClassID]*layout)
	var ordered []*ir.Class
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		ordered = append(ordered, module.Classes...)
	}
	state := make(map[ir.ClassID]int)
	var visit func(*ir.Class)
	visit = func(class *ir.Class) {
		if class == nil || state[class.ID] != 0 {
			return
		}
		state[class.ID] = 1
		var parent *layout
		if class.Parent != "" {
			visit(generator.classes[class.Parent])
			parent = generator.layouts[class.Parent]
			if parent == nil {
				generator.fail(CodeGenerationFailure, "Class "+class.Name+" has an unknown parent "+string(class.Parent), class.Span, "the IR references a Class with no declaration")
			}
		}
		state[class.ID] = 2
		generator.layouts[class.ID] = generator.newLayout(class, parent)
	}
	for _, class := range ordered {
		visit(class)
	}
	generator.layoutOrder = ordered
}

func (generator *generator) newLayout(class *ir.Class, parent *layout) *layout {
	current := &layout{class: class, parent: parent, ownFields: class.Fields}
	if parent != nil {
		current.allFields = append(current.allFields, parent.allFields...)
		current.allSlots = append(current.allSlots, parent.allSlots...)
	}
	current.allFields = append(current.allFields, class.Fields...)

	methods := append([]ir.CallableID(nil), class.Methods...)
	sort.Slice(methods, func(left, right int) bool { return methods[left] < methods[right] })
	for _, id := range methods {
		function := generator.functions[id]
		if function == nil {
			generator.fail(CodeGenerationFailure, "Class "+class.Name+" lists an unknown method "+string(id), class.Span, "the IR references a callable with no declaration")
			continue
		}
		root := generator.slotRoot(id)
		replaced := false
		for index := range current.allSlots {
			if current.allSlots[index].root == root {
				current.allSlots[index].impl = function
				replaced = true
				break
			}
		}
		if replaced {
			continue
		}
		entry := slot{root: root, impl: function}
		current.allSlots = append(current.allSlots, entry)
		current.ownSlots = append(current.ownSlots, entry)
	}
	return current
}

// emitClassDeclarations writes the descriptor, interface, struct, accessors,
// dispatch table, and freeze traversal of every Class.
func (generator *generator) emitClassDeclarations(writer *emitter) {
	for _, class := range generator.layoutOrder {
		current := generator.layouts[class.ID]
		if current == nil {
			continue
		}
		generator.emitDescriptor(writer, current)
		generator.emitInterface(writer, current)
		generator.emitStruct(writer, current)
		generator.emitAccessors(writer, current)
		generator.emitDispatch(writer, current)
		generator.emitErrorMessage(writer, current)
		generator.emitFreeze(writer, current)
	}
}

// emitDescriptor writes the canonical runtime identity of a Class. A built-in
// Class aliases the descriptor the runtime already uses, so a runtime check and
// an AhdCode except clause agree on identity.
func (generator *generator) emitDescriptor(writer *emitter, current *layout) {
	name := generator.descriptorName(current.class.ID)
	if current.class.Builtin {
		writer.line("// " + current.class.Name + " is a language-supplied Class.")
		writer.line("var " + name + " = AhdClass" + current.class.Name)
		writer.blank()
		return
	}
	parent := "nil"
	if current.parent != nil {
		parent = generator.descriptorName(current.parent.class.ID)
	}
	writer.line("var " + name + " = &AhdClass{Name: " + quote(current.class.Name) + ", Parent: " + parent + "}")
	writer.blank()
}

func (generator *generator) emitInterface(writer *emitter, current *layout) {
	writer.open("type " + generator.interfaceName(current.class.ID) + " interface {")
	if current.parent != nil {
		writer.line(generator.interfaceName(current.parent.class.ID))
	} else {
		writer.line("AhdInstance")
	}
	for _, field := range current.ownFields {
		rendered := generator.fieldType(current, field)
		if rendered == "" {
			continue
		}
		writer.line(generator.fieldName(field.ID) + "_get() " + rendered)
		writer.line(generator.fieldName(field.ID) + "_set(value " + rendered + ")")
	}
	for _, entry := range current.ownSlots {
		writer.line(generator.slotName(entry.root) + generator.slotSignature(entry.impl))
	}
	writer.close("}")
	writer.blank()
}

func (generator *generator) emitStruct(writer *emitter, current *layout) {
	writer.open("type " + generator.className(current.class.ID) + " struct {")
	if current.parent != nil {
		writer.line(generator.className(current.parent.class.ID))
	} else {
		writer.line("AhdBase")
	}
	for _, field := range current.ownFields {
		rendered := generator.fieldType(current, field)
		if rendered == "" {
			generator.fail(CodeInvalidRepresentation, "Class field "+field.Name+" has no Go representation", current.class.Span, "use a v0.1 representable field type")
			continue
		}
		writer.line(generator.fieldName(field.ID) + " " + rendered)
	}
	writer.close("}")
	writer.blank()
}

func (generator *generator) emitAccessors(writer *emitter, current *layout) {
	receiver := "(object *" + generator.className(current.class.ID) + ")"
	for _, field := range current.ownFields {
		rendered := generator.fieldType(current, field)
		if rendered == "" {
			continue
		}
		name := generator.fieldName(field.ID)
		writer.line("func " + receiver + " " + name + "_get() " + rendered + " { return object." + name + " }")
		writer.open("func " + receiver + " " + name + "_set(value " + rendered + ") {")
		writer.line("object.AhdRequireMutable()")
		writer.line("object." + name + " = value")
		writer.close("}")
	}
	if len(current.ownFields) != 0 {
		writer.blank()
	}
}

// emitDispatch writes one forwarder per slot of the full method table. Go
// method promotion is deliberately not relied upon: an inherited method must
// still receive the derived instance so a further override dispatches.
func (generator *generator) emitDispatch(writer *emitter, current *layout) {
	receiver := "(object *" + generator.className(current.class.ID) + ")"
	for _, entry := range current.allSlots {
		if entry.impl == nil {
			continue
		}
		parameters, arguments := generator.slotParameters(entry.impl)
		result, prefix := "", ""
		if entry.impl.Signature.Return.Kind != ir.NothingType {
			result = " " + generator.goType(entry.impl.Signature.Return, entry.impl.ReturnNull != ir.NonNull)
			prefix = "return "
		}
		call := generator.callableName(entry.impl) + "(" + strings.Join(append([]string{"object"}, arguments...), ", ") + ")"
		writer.line("func " + receiver + " " + generator.slotName(entry.root) + "(" + strings.Join(parameters, ", ") + ")" + result + " { " + prefix + call + " }")
	}
	if len(current.allSlots) != 0 {
		writer.blank()
	}
}

// emitFreeze writes the Constant deep-freeze traversal of a Class. It marks the
// instance and continues into every reachable attribute, including inherited
// ones, and terminates on cyclic object graphs.
func (generator *generator) emitFreeze(writer *emitter, current *layout) {
	writer.open("func (object *" + generator.className(current.class.ID) + ") AhdFreezeGraph(visited map[AhdFreezable]bool) {")
	writer.open("if !AhdEnterFreeze(object, visited) {")
	writer.line("return")
	writer.close("}")
	writer.line("object.AhdMarkFrozen()")
	for _, field := range current.allFields {
		if generator.fieldType(current, field) == "" {
			continue
		}
		writer.line("AhdFreezeChild(object." + generator.fieldName(field.ID) + ", visited)")
	}
	writer.close("}")
	writer.blank()
}

// emitErrorMessage exposes the built-in Error message attribute to the
// runtime, so an uncaught error reports its own text. Every Error subclass
// inherits the accessor through struct embedding.
func (generator *generator) emitErrorMessage(writer *emitter, current *layout) {
	if !current.class.Builtin || current.class.Name != "Error" {
		return
	}
	for _, field := range current.ownFields {
		if field.Name != "message" {
			continue
		}
		writer.line("func (object *" + generator.className(current.class.ID) + ") AhdErrorMessage() string { return object." + generator.fieldName(field.ID) + " }")
		writer.blank()
		return
	}
}

func (generator *generator) fieldType(current *layout, field ir.Field) string {
	return generator.goType(field.Type, generator.nullFields[field.ID])
}

// slotParameters renders the dispatch forwarder parameter list and the
// arguments passed on to the implementation.
func (generator *generator) slotParameters(function *ir.Function) ([]string, []string) {
	parameters := make([]string, 0, len(function.Parameters))
	arguments := make([]string, 0, len(function.Parameters))
	for index, parameter := range function.Parameters {
		name := "argument" + itoa(index)
		parameters = append(parameters, name+" "+generator.goType(parameter.Type, parameter.NullState != ir.NonNull))
		arguments = append(arguments, name)
	}
	return parameters, arguments
}

func (generator *generator) slotSignature(function *ir.Function) string {
	if function == nil {
		return "()"
	}
	parts := make([]string, 0, len(function.Parameters))
	for _, parameter := range function.Parameters {
		parts = append(parts, generator.goType(parameter.Type, parameter.NullState != ir.NonNull))
	}
	result := "(" + strings.Join(parts, ", ") + ")"
	if function.Signature.Return.Kind == ir.NothingType {
		return result
	}
	return result + " " + generator.goType(function.Signature.Return, function.ReturnNull != ir.NonNull)
}
