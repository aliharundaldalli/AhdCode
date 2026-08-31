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
	Files         []GeneratedFile
	RequiresLatex bool
}

const (
	programFileName = "ahdcode_program.go"
	runtimeFileName = "ahdcode_runtime.go"
)

// storage describes the Go representation chosen for one IR symbol.
type storage struct {
	name     string
	typeInfo ir.Type
	nullable bool
}

type generator struct {
	compilation *ir.Compilation
	classes     map[ir.ClassID]*ir.Class
	functions   map[ir.CallableID]*ir.Function
	fields      map[ir.FieldID]ir.Field
	slots       map[ir.SymbolID]storage
	nullSymbols map[ir.SymbolID]bool
	nullFields  map[ir.FieldID]bool
	layouts     map[ir.ClassID]*layout
	layoutOrder []*ir.Class
	adapters    map[ir.CallableID]string
	timeHelpers map[ir.ClassID]string
	diagnostics []diagnostics.Diagnostic
	temporary   int
	usesLatex   bool
	// frames tracks the enclosing loop and attempt structure so break,
	// continue, and return transfer through error handling correctly.
	frames []frame
	// result is the function-level variable that carries a return value out of
	// an attempt closure.
	result         string
	resultType     ir.Type
	resultNullable bool
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
		slots:       make(map[ir.SymbolID]storage),
		nullSymbols: make(map[ir.SymbolID]bool),
		nullFields:  make(map[ir.FieldID]bool),
		adapters:    make(map[ir.CallableID]string),
		timeHelpers: make(map[ir.ClassID]string),
	}
	generator.buildIndex()
	generator.planRepresentations()
	generator.buildLayouts()
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
	}, RequiresLatex: generator.usesLatex}, generator.diagnostics
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
			generator.slots[global.ID] = storage{name: mangleNamed(globalPrefix, global.Name, string(global.ID)), typeInfo: global.Type, nullable: generator.nullSymbols[global.ID]}
		}
		for _, function := range module.Functions {
			if function == nil {
				continue
			}
			if function.Receiver != "" {
				generator.slots[function.Receiver] = storage{name: "attribute", typeInfo: ir.Type{Kind: ir.ClassType, Class: function.Owner}}
			}
			for _, parameter := range function.Parameters {
				generator.slots[parameter.ID] = storage{
					name: mangleNamed(localPrefix, parameter.Name, string(parameter.ID)), typeInfo: parameter.Type,
					nullable: parameter.NullState != ir.NonNull,
				}
			}
		}
	}
	generator.walkCompilation(func(statement ir.Statement) {
		switch value := statement.(type) {
		case *ir.BindingStmt:
			// A module-root binding already has its package-level storage.
			if value.Storage != ir.ModuleStorage {
				generator.slots[value.Symbol] = storage{name: mangleNamed(localPrefix, value.Name, string(value.Symbol)), typeInfo: value.Type, nullable: generator.nullSymbols[value.Symbol]}
			}
		case *ir.ForStmt:
			generator.slots[value.Iteration] = storage{name: mangleNamed(localPrefix, value.Name, string(value.Iteration)), typeInfo: value.IterationType}
		case *ir.AttemptStmt:
			for _, handler := range value.Handlers {
				if handler.Binding == "" {
					continue
				}
				generator.slots[handler.Binding] = storage{
					name:     mangleNamed(localPrefix, "error", string(handler.Binding)),
					typeInfo: ir.Type{Kind: ir.ClassType, Class: handler.Class},
				}
			}
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
	generator.emitClassDeclarations(writer)
	for _, module := range generator.compilation.Modules {
		if module != nil {
			generator.emitGlobals(writer, module)
		}
	}
	// Bodies are generated first so every Function value adapter they need is
	// registered before the adapters themselves are written.
	bodies := &emitter{}
	for _, module := range generator.compilation.Modules {
		if module == nil {
			continue
		}
		for _, function := range module.Functions {
			generator.emitFunction(bodies, module.Name, function)
		}
	}
	for _, module := range generator.compilation.Modules {
		if module != nil {
			generator.emitModuleInit(bodies, module)
		}
	}
	generator.emitAdapters(writer)
	// Bodies register the Time helpers they need, so those are written after
	// generation and before the bodies themselves.
	generator.emitTimeHelpers(writer)
	generator.emitDataHelpers(writer)
	generator.emitPlotHelpers(writer)
	writer.raw(bodies.String())
	generator.emitInstaller(writer)
	writer.open("func main() {")
	writer.open("AhdMain(ahdInstall, func() {")
	for _, module := range generator.compilation.Modules {
		if module != nil {
			writer.line(generator.moduleInitName(module) + "()")
		}
	}
	writer.close("})")
	writer.close("}")
	return writer.String()
}

// emitInstaller registers the generated constructors of the language-supplied
// Error classes, so a runtime check raises a real AhdCode Error instance that
// an ordinary except clause can match.
func (generator *generator) emitInstaller(writer *emitter) {
	writer.line("// ahdInstall wires the language-supplied Error catalog into the runtime.")
	writer.open("func ahdInstall() {")
	for _, class := range generator.layoutOrder {
		if !class.Builtin || !generator.descendsFromError(class.ID) {
			continue
		}
		constructor := generator.functions[class.Constructor]
		if constructor == nil {
			continue
		}
		writer.line("AhdRegisterError(" + generator.descriptorName(class.ID) + ", func(message string) AhdInstance { return " +
			generator.callableName(constructor) + "(message) })")
	}
	// CSV file operations preserve FileError even when the program did not
	// separately import File. Its IOError representation is sufficient for
	// ordinary IOError catches; importing File still registers the exact
	// generated FileError constructor above.
	ioError := generator.classes[ir.ClassID("builtin:core::class::IOError")]
	if ioError != nil {
		if constructor := generator.functions[ioError.Constructor]; constructor != nil {
			writer.line("AhdRegisterErrorFallback(AhdClassFileError, func(message string) AhdInstance { return " +
				generator.callableName(constructor) + "(message) })")
		}
	}
	writer.close("}")
	writer.blank()
}

func (generator *generator) descendsFromError(id ir.ClassID) bool {
	for current := generator.layouts[id]; current != nil; current = current.parent {
		if current.class.Builtin && current.class.Name == "Error" {
			return true
		}
	}
	return false
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
	if function.Kind == ir.ConstructorFunction {
		generator.emitConstructor(writer, moduleName, function)
		return
	}
	parameters := make([]string, 0, len(function.Parameters)+1)
	if function.Receiver != "" {
		parameters = append(parameters, generator.slots[function.Receiver].name+" "+generator.interfaceName(function.Owner))
	}
	for _, parameter := range function.Parameters {
		current := generator.slots[parameter.ID]
		rendered := generator.goType(parameter.Type, current.nullable)
		if rendered == "" {
			generator.fail(CodeInvalidRepresentation, fmt.Sprintf("parameter %s has no Go representation", parameter.Name), parameter.Span, "use a v0.1 representable parameter type")
			continue
		}
		parameters = append(parameters, current.name+" "+rendered)
	}
	result := ""
	if function.Signature.Return.Kind != ir.NothingType {
		rendered := generator.goType(function.Signature.Return, function.ReturnNull != ir.NonNull)
		if rendered == "" {
			generator.fail(CodeInvalidRepresentation, "Function return type has no Go representation", function.Span, "use a v0.1 representable return type")
			return
		}
		result = " " + rendered
	}
	writer.line("// " + string(function.Kind) + " " + function.Name + " of module " + moduleName + ".")
	writer.open("func " + generator.callableName(function) + "(" + strings.Join(parameters, ", ") + ")" + result + " {")
	generator.emitFunctionBody(writer, function)
	writer.close("}")
	writer.blank()
}

// emitFunctionBody writes a callable body, reserving the return carrier that
// an attempt closure uses to transfer a pending return out of itself.
func (generator *generator) emitFunctionBody(writer *emitter, function *ir.Function) {
	previousFrames, previousResult, previousType, previousNullable := generator.frames, generator.result, generator.resultType, generator.resultNullable
	generator.frames, generator.result = nil, ""
	generator.resultType, generator.resultNullable = function.Signature.Return, function.ReturnNull != ir.NonNull
	if function.Signature.Return.Kind != ir.NothingType && containsAttempt(function.Body) {
		generator.result = generator.temporaryName()
		writer.line("var " + generator.result + " " + generator.goType(function.Signature.Return, function.ReturnNull != ir.NonNull))
		writer.line("_ = " + generator.result)
	}
	generator.emitBlock(writer, function.Body)
	if function.Signature.Return.Kind != ir.NothingType && !endsWithReturn(function.Body) {
		// Go terminating-statement analysis is narrower than the AhdCode
		// definite-return rule, so an explicit unreachable tail is emitted.
		writer.line("return AhdUnreachable[" + generator.goType(function.Signature.Return, function.ReturnNull != ir.NonNull) + "]()")
	}
	generator.frames, generator.result, generator.resultType, generator.resultNullable = previousFrames, previousResult, previousType, previousNullable
}

// emitConstructor writes the allocating constructor and the initializer that
// runs the inherited construction contract before this Class initializes its
// own attributes.
func (generator *generator) emitConstructor(writer *emitter, moduleName string, function *ir.Function) {
	interfaceType := generator.interfaceName(function.Owner)
	parameters := make([]string, 0, len(function.Parameters))
	arguments := make([]string, 0, len(function.Parameters))
	for _, parameter := range function.Parameters {
		current := generator.slots[parameter.ID]
		rendered := generator.goType(parameter.Type, current.nullable)
		if rendered == "" {
			generator.fail(CodeInvalidRepresentation, fmt.Sprintf("parameter %s has no Go representation", parameter.Name), parameter.Span, "use a v0.1 representable parameter type")
			continue
		}
		parameters = append(parameters, current.name+" "+rendered)
		arguments = append(arguments, current.name)
	}
	receiver := generator.slots[function.Receiver].name
	writer.line("// Initializer of Class " + function.Name + " of module " + moduleName + ".")
	writer.open("func " + generator.initializerName(function) + "(" + strings.Join(append([]string{receiver + " " + interfaceType}, parameters...), ", ") + ") {")
	writer.line("_ = " + receiver)
	if function.ParentConstructor != "" {
		parent := generator.functions[function.ParentConstructor]
		if parent == nil {
			generator.fail(CodeGenerationFailure, "constructor references an unknown parent constructor", function.Span, "the IR references a callable with no declaration")
		} else {
			forwarded := []string{receiver}
			for _, index := range function.ParentArguments {
				if index < 0 || index >= len(arguments) {
					generator.fail(CodeGenerationFailure, "parent constructor argument index is out of range", function.Span, "the IR constructor contract is malformed")
					continue
				}
				forwarded = append(forwarded, arguments[index])
			}
			writer.line(generator.initializerName(parent) + "(" + strings.Join(forwarded, ", ") + ")")
		}
	}
	previousFrames, previousResult := generator.frames, generator.result
	generator.frames, generator.result = nil, ""
	generator.emitBlock(writer, function.Body)
	generator.frames, generator.result = previousFrames, previousResult
	writer.close("}")
	writer.blank()

	writer.line("// Constructor of Class " + function.Name + " of module " + moduleName + ".")
	writer.open("func " + generator.callableName(function) + "(" + strings.Join(parameters, ", ") + ") " + interfaceType + " {")
	writer.line("instance := &" + generator.className(function.Owner) + "{}")
	writer.line("instance.AhdSetClass(" + generator.descriptorName(function.Owner) + ")")
	writer.line(generator.initializerName(function) + "(" + strings.Join(append([]string{"instance"}, arguments...), ", ") + ")")
	writer.line("return instance")
	writer.close("}")
	writer.blank()
}

// emitAdapters writes one uniform Function value per callable taken as a
// value. A Function type carries no null-state, so the adapter boxes and
// unboxes between the uniform shape and the concrete callable signature.
func (generator *generator) emitAdapters(writer *emitter) {
	names := make([]ir.CallableID, 0, len(generator.adapters))
	for id := range generator.adapters {
		names = append(names, id)
	}
	sort.Slice(names, func(left, right int) bool { return names[left] < names[right] })
	for _, id := range names {
		function := generator.functions[id]
		if function == nil {
			continue
		}
		parameters := make([]string, 0, len(function.Parameters))
		arguments := make([]string, 0, len(function.Parameters))
		for index, parameter := range function.Parameters {
			name := "argument" + itoa(index)
			parameters = append(parameters, name+" "+generator.goType(parameter.Type, true))
			arguments = append(arguments, generator.coerce(name,
				ir.ExprBase{Type: parameter.Type, NullState: ir.MaybeNull}, parameter.Type, parameter.NullState != ir.NonNull))
		}
		result := ""
		body := generator.callableName(function) + "(" + strings.Join(arguments, ", ") + ")"
		if function.Signature.Return.Kind != ir.NothingType {
			result = " " + generator.goType(function.Signature.Return, true)
			body = "return " + generator.coerce(body,
				ir.ExprBase{Type: function.Signature.Return, NullState: function.ReturnNull}, function.Signature.Return, true)
		}
		writer.line("// Function value of " + function.Name + ".")
		writer.line("func " + generator.adapters[id] + "(" + strings.Join(parameters, ", ") + ")" + result + " { " + body + " }")
		writer.blank()
	}
}

// adapterName registers the uniform Function value wrapper of a callable.
func (generator *generator) adapterName(function *ir.Function) string {
	if name, known := generator.adapters[function.ID]; known {
		return name
	}
	name := mangleNamed(adapterPrefix, function.Name, string(function.ID))
	generator.adapters[function.ID] = name
	return name
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

func (generator *generator) initializerName(function *ir.Function) string {
	return mangleNamed(initializerPrefix, function.Name, string(function.ID))
}

func (generator *generator) emitModuleInit(writer *emitter, module *ir.Module) {
	writer.line("// Module " + module.Name + " initialization.")
	writer.open("func " + generator.moduleInitName(module) + "() {")
	previousFrames, previousResult := generator.frames, generator.result
	generator.frames, generator.result = nil, ""
	// Module-root initializers are part of the statement stream, so module
	// effects run in source order.
	generator.emitBlock(writer, module.Init)
	generator.frames, generator.result = previousFrames, previousResult
	writer.close("}")
	writer.blank()
}
