package lowering

import (
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/ir"
	"ahdcode/internal/module"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
)

const (
	CodeFrontendFailed  = "LWR001"
	CodeMissingSemantic = "LWR002"
	CodeUnsupportedNode = "LWR003"
)

type Result struct {
	Compilation *ir.Compilation
	Diagnostics []diagnostics.Diagnostic
}

func (result Result) HasErrors() bool {
	for _, item := range result.Diagnostics {
		if item.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}

// LowerCompilation consumes semantic/module decisions without re-running name,
// overload, type, or import resolution.
func LowerCompilation(compilation module.CompilationResult) Result {
	engine := &compilationLowerer{}
	if compilation.HasErrors() {
		engine.error(CodeFrontendFailed, "cannot lower a compilation with frontend errors", source.Span{})
		return Result{Diagnostics: engine.diagnostics}
	}
	ordered := dependencyOrder(compilation)
	for _, current := range ordered {
		if current == nil || current.State != module.Resolved || current.Semantic.HasErrors() {
			engine.error(CodeFrontendFailed, "cannot lower unresolved or failed module", source.Span{})
			return Result{Diagnostics: engine.diagnostics}
		}
	}
	engine.registry = newRegistry(ordered)
	result := &ir.Compilation{Entry: ir.ModuleID(compilation.Entry)}
	// The language-supplied Class catalog is emitted first so every module can
	// depend on it without an implicit backend contract.
	result.Modules = append(result.Modules, builtinModule())
	for _, current := range ordered {
		if current.Source.Builtin {
			result.Modules = append(result.Modules, &ir.Module{
				ID: ir.ModuleID(current.ID), Name: current.Source.Name, SourcePath: current.Source.Path,
			})
			continue
		}
		lowerer := &moduleLowerer{compilation: engine, module: current, semantic: current.Semantic}
		result.Modules = append(result.Modules, lowerer.lowerModule())
	}
	if len(engine.diagnostics) == 0 {
		engine.diagnostics = append(engine.diagnostics, ir.Validate(result)...)
	}
	if len(engine.diagnostics) != 0 {
		return Result{Diagnostics: engine.diagnostics}
	}
	return Result{Compilation: result}
}

type compilationLowerer struct {
	registry    *registry
	diagnostics []diagnostics.Diagnostic
}

func (lowerer *compilationLowerer) error(code, message string, span source.Span) {
	lowerer.diagnostics = append(lowerer.diagnostics, diagnostics.Diagnostic{Code: code, Severity: diagnostics.SeverityError, Message: message, Span: span})
}

func dependencyOrder(compilation module.CompilationResult) []*module.Module {
	visited := make(map[module.ModuleID]bool)
	var result []*module.Module
	var visit func(module.ModuleID)
	visit = func(id module.ModuleID) {
		if visited[id] {
			return
		}
		visited[id] = true
		current := compilation.Modules[id]
		if current == nil {
			return
		}
		for _, dependency := range current.Dependencies {
			visit(dependency)
		}
		result = append(result, current)
	}
	visit(compilation.Entry)
	for _, id := range compilation.Order {
		visit(id)
	}
	return result
}

type moduleLowerer struct {
	compilation     *compilationLowerer
	module          *module.Module
	semantic        semantic.Result
	currentReturn   ir.Type
	currentReceiver ir.SymbolID
	currentOwner    ir.ClassID
}
