package golang

import (
	"fmt"
	"go/format"
	"sort"
	"strings"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/ir"
	"ahdcode/internal/source"

	"ahdcode/internal/backend/golang/ahdruntime"
)

// GeneratedFile is one deterministic Go source file of a generated program.
type GeneratedFile struct {
	Name    string
	Content string
}

// GeneratedProgram is the complete Go source of one AhdCode compilation.
type GeneratedProgram struct {
	Files []GeneratedFile
}

const (
	programFileName = "ahdcode_program.go"
	runtimeFileName = "ahdcode_runtime.go"
)

// slot describes the Go storage representation chosen for one IR symbol.
type slot struct {
	name     string
	typeInfo ir.Type
	nullable bool
}

type generator struct {
	compilation *ir.Compilation
	classes     map[ir.ClassID]*ir.Class
	functions   map[ir.CallableID]*ir.Function
	fields      map[ir.FieldID]ir.Field
	slots       map[ir.SymbolID]slot
	nullSymbols map[ir.SymbolID]bool
	nullFields  map[ir.FieldID]bool
	diagnostics []diagnostics.Diagnostic
	temporary   int
	loop        int
}

// Generate lowers a validated IR compilation into deterministic Go source.
// Generating the same IR twice produces byte-identical output.
func Generate(compilation *ir.Compilation) (*GeneratedProgram, []diagnostics.Diagnostic) {
	if compilation == nil {
		return nil, []diagnostics.Diagnostic{backendError(CodeGenerationFailure, "nil IR compilation reached code generation", source.Span{}, "run the frontend and lowering pipeline first")}
	}
	generator := &generator{
		compilation: compilation,
		classes:     make(map[ir.ClassID]*ir.Class),
		functions:   make(map[ir.CallableID]*ir.Function),
		fields:      make(map[ir.FieldID]ir.Field),
		slots:       make(map[ir.SymbolID]slot),
		nullSymbols: make(map[ir.SymbolID]bool),
		nullFields:  make(map[ir.FieldID]bool),
	}
	generator.buildIndex()
	generator.planRepresentations()
	program := generator.emitProgram()
	if generator.hasErrors() {
		return nil, generator.diagnostics
	}
	formatted, err := format.Source([]byte(program))
	if err != nil {
		return nil, append(generator.diagnostics, backendError(CodeFormatFailure, "generated Go source is not valid Go: "+err.Error(), source.Span{}, "this is a code generation defect; report the failing AhdCode program"))
	}
	runtime, err := format.Source([]byte(runtimeSource()))
	if err != nil {
		return nil, append(generator.diagnostics, backendError(CodeFormatFailure, "embedded runtime source is not valid Go: "+err.Error(), source.Span{}, "the backend runtime must remain gofmt-clean"))
	}
	return &GeneratedProgram{Files: []GeneratedFile{
		{Name: programFileName, Content: string(formatted)},
		{Name: runtimeFileName, Content: string(runtime)},
	}}, generator.diagnostics
}

// runtimeSource re-points the shared runtime package at the generated program.
func runtimeSource() string {
	return strings.Replace(ahdruntime.Source, "package ahdruntime", "package main", 1)
}

func (generator *generator) hasErrors() bool {
	for _, item := range generator.diagnostics {
		if item.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}

func (generator *generator) fail(code, message string, span source.Span, hint string) {
	generator.diagnostics = append(generator.diagnostics, backendError(code, message, span, hint))
}

func (generator *generator) unsupported(what string, span source.Span) string {
	generator.fail(CodeUnsupportedNode, "the Go backend does not yet lower "+what, span, "this construct is deferred; it is reported instead of generating incorrect Go")
	return "nil"
}

// ---------------------------------------------------------------------------
// Indexing
// ---------------------------------------------------------------------------

func (generator *generator) buildIndex() {
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		for _, class := range module.Classes {
			if class == nil {
				continue
			}
			generator.classes[class.ID] = class
			for _, field := range class.Fields {
				generator.fields[field.ID] = field
			}
		}
		for _, function := range module.Functions {
			if function != nil {
				generator.functions[function.ID] = function
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Representation planning
// ---------------------------------------------------------------------------

// planRepresentations chooses one Go storage representation per symbol and
// field. A scalar slot is boxed when its declaration allows null or when any
// assignment in the compilation stores a possibly-null value into it. This is
// representation selection over already-resolved IR, not semantic analysis.
func (generator *generator) planRepresentations() {
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		for _, global := range module.Globals {
			if global != nil && global.NullState != ir.NonNull {
				generator.nullSymbols[global.ID] = true
			}
		}
		for _, class := range module.Classes {
			if class == nil {
				continue
			}
			for _, field := range class.Fields {
				if field.NullState != ir.NonNull {
					generator.nullFields[field.ID] = true
				}
			}
		}
	}
	generator.walkCompilation(func(statement ir.Statement) {
		switch value := statement.(type) {
		case *ir.BindingStmt:
			if value.NullState != ir.NonNull || nullableValue(value.Initializer) {
				generator.nullSymbols[value.Symbol] = true
			}
		case *ir.AssignStmt:
			if !nullableValue(value.Value) {
				return
			}
			switch value.Target.Kind {
			case ir.SymbolTarget:
				generator.nullSymbols[value.Target.Symbol] = true
			case ir.FieldTarget:
				generator.nullFields[value.Target.Field] = true
			}
		}
	})
	generator.bindSlots()
}

func nullableValue(expression ir.Expr) bool {
	return expression != nil && expression.ExprMeta().NullState != ir.NonNull
}

func (generator *generator) bindSlots() {
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		for _, global := range module.Globals {
			if global == nil {
				continue
			}
			generator.slots[global.ID] = slot{name: mangle(globalPrefix, string(global.ID)), typeInfo: global.Type, nullable: generator.nullSymbols[global.ID]}
		}
		for _, function := range module.Functions {
			if function == nil {
				continue
			}
			if function.Receiver != "" {
				generator.slots[function.Receiver] = slot{name: mangle(localPrefix, string(function.Receiver)), typeInfo: ir.Type{Kind: ir.ClassType, Class: function.Owner}}
			}
			for _, parameter := range function.Parameters {
				generator.slots[parameter.ID] = slot{name: mangle(localPrefix, string(parameter.ID)), typeInfo: parameter.Type}
				if parameter.NullState != ir.NonNull || generator.nullSymbols[parameter.ID] {
					generator.fail(CodeUnsupportedNode, fmt.Sprintf("nullable Function parameter %q has no Go representation in this milestone", parameter.Name), parameter.Span, "a Function signature carries no per-parameter null-state in the IR contract; keep parameters NonNull")
				}
			}
		}
	}
	generator.walkCompilation(func(statement ir.Statement) {
		switch value := statement.(type) {
		case *ir.BindingStmt:
			generator.slots[value.Symbol] = slot{name: mangle(localPrefix, string(value.Symbol)), typeInfo: value.Type, nullable: generator.nullSymbols[value.Symbol]}
		case *ir.ForStmt:
			generator.slots[value.Iteration] = slot{name: mangle(localPrefix, string(value.Iteration)), typeInfo: value.IterationType}
		}
	})
}

func (generator *generator) walkCompilation(visit func(ir.Statement)) {
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		for _, function := range module.Functions {
			if function != nil {
				walkBlock(function.Body, visit)
			}
		}
		walkBlock(module.Init, visit)
	}
}

func walkBlock(block ir.Block, visit func(ir.Statement)) {
	for _, statement := range block.Statements {
		if statement == nil {
			continue
		}
		visit(statement)
		switch value := statement.(type) {
		case *ir.IfStmt:
			for _, branch := range value.Branches {
				walkBlock(branch.Body, visit)
			}
			if value.Else != nil {
				walkBlock(*value.Else, visit)
			}
		case *ir.WhileStmt:
			walkBlock(value.Body, visit)
		case *ir.DoUntilStmt:
			walkBlock(value.Body, visit)
		case *ir.ForStmt:
			walkBlock(value.Body, visit)
		case *ir.StateStmt:
			for _, item := range value.Cases {
				walkBlock(item.Body, visit)
			}
		case *ir.AttemptStmt:
			walkBlock(value.Body, visit)
			for _, handler := range value.Handlers {
				walkBlock(handler.Body, visit)
			}
			if value.Ultimately != nil {
				walkBlock(*value.Ultimately, visit)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Program emission
// ---------------------------------------------------------------------------

func (generator *generator) emitProgram() string {
	writer := &emitter{}
	writer.line("// Code generated by the AhdCode compiler. DO NOT EDIT.")
	writer.blank()
	writer.line("package main")
	writer.blank()
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		generator.emitClasses(writer, module)
	}
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		generator.emitGlobals(writer, module)
	}
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		for _, function := range module.Functions {
			generator.emitFunction(writer, module.Name, function)
		}
	}
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		generator.emitModuleInit(writer, module)
	}
	writer.open("func main() {")
	writer.open("AhdMain(func() {")
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		writer.line(generator.moduleInitName(module) + "()")
	}
	writer.close("})")
	writer.close("}")
	return writer.String()
}

func (generator *generator) emitClasses(writer *emitter, module *ir.Module) {
	for _, class := range module.Classes {
		if class == nil {
			continue
		}
		if class.Parent != "" && !isBuiltinClass(class.Parent) {
			generator.fail(CodeUnsupportedNode, fmt.Sprintf("Class %s declares inheritance, which the Go backend defers", class.Name), class.Span, "Class inheritance and its runtime dispatch are deferred to a later milestone")
			continue
		}
		writer.line("// Class " + class.Name + " of module " + module.Name + ".")
		writer.open("type " + generator.className(class.ID) + " struct {")
		for _, field := range class.Fields {
			rendered := generator.goType(field.Type, generator.nullFields[field.ID])
			if rendered == "" {
				generator.fail(CodeInvalidRepresentation, fmt.Sprintf("Class field %s has no Go representation", field.Name), class.Span, "use a v0.1 representable field type")
				continue
			}
			writer.line(generator.fieldName(field.ID) + " " + rendered)
		}
		writer.close("}")
		writer.blank()
	}
}

func isBuiltinClass(id ir.ClassID) bool {
	return strings.HasPrefix(string(id), "builtin:")
}

func (generator *generator) emitGlobals(writer *emitter, module *ir.Module) {
	if len(module.Globals) == 0 {
		return
	}
	writer.open("var (")
	for _, global := range module.Globals {
		if global == nil {
			continue
		}
		current := generator.slots[global.ID]
		rendered := generator.goType(current.typeInfo, current.nullable)
		if rendered == "" {
			generator.fail(CodeInvalidRepresentation, fmt.Sprintf("global %s has no Go representation", global.Name), global.Span, "use a v0.1 representable declared type")
			continue
		}
		writer.line(current.name + " " + rendered)
	}
	writer.close(")")
	writer.blank()
}

func (generator *generator) emitFunction(writer *emitter, moduleName string, function *ir.Function) {
	if function == nil {
		return
	}
	if function.Owner != "" {
		if class := generator.classes[function.Owner]; class != nil && class.Parent != "" && !isBuiltinClass(class.Parent) {
			return
		}
	}
	parameters := make([]string, 0, len(function.Parameters)+1)
	// A constructor allocates its own receiver; every other method receives it.
	if function.Receiver != "" && function.Kind != ir.ConstructorFunction {
		receiver := generator.slots[function.Receiver]
		parameters = append(parameters, receiver.name+" *"+generator.className(function.Owner))
	}
	for _, parameter := range function.Parameters {
		current := generator.slots[parameter.ID]
		rendered := generator.goType(parameter.Type, false)
		if rendered == "" {
			generator.fail(CodeInvalidRepresentation, fmt.Sprintf("parameter %s has no Go representation", parameter.Name), parameter.Span, "use a v0.1 representable parameter type")
			continue
		}
		parameters = append(parameters, current.name+" "+rendered)
	}
	name := generator.callableName(function)
	result := ""
	switch {
	case function.Kind == ir.ConstructorFunction:
		result = " *" + generator.className(function.Owner)
	case function.Signature.Return.Kind != ir.NothingType:
		rendered := generator.goType(function.Signature.Return, false)
		if rendered == "" {
			generator.fail(CodeInvalidRepresentation, "Function return type has no Go representation", function.Span, "use a v0.1 representable return type")
			return
		}
		result = " " + rendered
	}
	if function.ReturnNull != ir.NonNull && function.Signature.Return.Kind != ir.NothingType {
		generator.fail(CodeUnsupportedNode, fmt.Sprintf("Function %q may return null, which has no Go representation in this milestone", function.Name), function.Span, "a Function signature carries no return null-state in the IR contract; return a NonNull value")
		return
	}
	writer.line("// " + string(function.Kind) + " " + function.Name + " of module " + moduleName + ".")
	writer.open("func " + name + "(" + strings.Join(parameters, ", ") + ")" + result + " {")
	if function.Kind == ir.ConstructorFunction {
		receiver := generator.slots[function.Receiver]
		writer.line(receiver.name + " := &" + generator.className(function.Owner) + "{}")
	}
	generator.emitBlock(writer, function.Body)
	switch {
	case function.Kind == ir.ConstructorFunction:
		writer.line("return " + generator.slots[function.Receiver].name)
	case function.Signature.Return.Kind != ir.NothingType && !endsWithReturn(function.Body):
		// Go terminating-statement analysis is narrower than the AhdCode
		// definite-return rule, so an explicit unreachable tail is emitted.
		writer.line("return AhdUnreachable[" + generator.goType(function.Signature.Return, false) + "]()")
	}
	writer.close("}")
	writer.blank()
}

func endsWithReturn(block ir.Block) bool {
	if len(block.Statements) == 0 {
		return false
	}
	_, ok := block.Statements[len(block.Statements)-1].(*ir.ReturnStmt)
	return ok
}

func (generator *generator) moduleInitName(module *ir.Module) string {
	return mangleNamed(initPrefix, module.Name, string(module.ID))
}

func (generator *generator) callableName(function *ir.Function) string {
	if function.Kind == ir.ConstructorFunction {
		return mangleNamed(constructorPrefix, function.Name, string(function.ID))
	}
	return mangleNamed(functionPrefix, function.Name, string(function.ID))
}

func (generator *generator) emitModuleInit(writer *emitter, module *ir.Module) {
	writer.line("// Module " + module.Name + " initialization.")
	writer.open("func " + generator.moduleInitName(module) + "() {")
	ordered := append([]*ir.Global(nil), module.Globals...)
	sort.SliceStable(ordered, func(left, right int) bool { return ordered[left].Order < ordered[right].Order })
	for _, global := range ordered {
		if global == nil {
			continue
		}
		current := generator.slots[global.ID]
		if global.Initializer == nil {
			continue
		}
		writer.line(current.name + " = " + generator.value(global.Initializer, current.typeInfo, current.nullable))
	}
	generator.emitBlock(writer, module.Init)
	writer.close("}")
	writer.blank()
}
