package module

import (
	"fmt"
	"strconv"
	"strings"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/lexer"
	"ahdcode/internal/parser"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
	"ahdcode/internal/syntax/ast"
)

type Compiler struct {
	Resolver ModuleResolver
	Loader   SourceLoader
	Builtins map[string]*semantic.ModuleInterface

	modules     map[ModuleID]*Module
	order       []ModuleID
	diagnostics []ModuleDiagnostic
	stack       []ModuleID
	nextFileID  source.FileID
}

func NewCompiler(resolver ModuleResolver, loader SourceLoader) *Compiler {
	return &Compiler{Resolver: resolver, Loader: loader, Builtins: semantic.StandardModuleInterfaces()}
}

func (compiler *Compiler) Compile(entryPath string) CompilationResult {
	compiler.modules = make(map[ModuleID]*Module)
	compiler.order = nil
	compiler.diagnostics = nil
	compiler.stack = nil
	compiler.nextFileID = 1

	identity, err := compiler.Resolver.CanonicalEntry(entryPath)
	if err != nil {
		compiler.addDiagnostic(semantic.CodeModuleNotFound, fmt.Sprintf("entry module was not found: %v", err), source.Span{}, "provide a readable .ahd entry file", "", "", "", nil)
		return compiler.result("")
	}
	compiler.analyze(identity, "", source.Span{})
	return compiler.result(identity.ID)
}

func (compiler *Compiler) result(entry ModuleID) CompilationResult {
	return CompilationResult{
		Entry: entry, Modules: compiler.modules,
		Order:       append([]ModuleID(nil), compiler.order...),
		Diagnostics: append([]ModuleDiagnostic(nil), compiler.diagnostics...),
	}
}

func (compiler *Compiler) analyze(identity SourceIdentity, requester ModuleID, importSpan source.Span) *Module {
	if existing := compiler.modules[identity.ID]; existing != nil {
		switch existing.State {
		case Resolving:
			cycle := compiler.cycleFrom(identity.ID)
			compiler.addDiagnostic(semantic.CodeCircularDependency, "circular module dependency: "+formatCycle(cycle), importSpan, "remove one bring edge from the dependency cycle", requester, identity.ID, "", cycle)
			for _, id := range cycle[:len(cycle)-1] {
				if module := compiler.modules[id]; module != nil {
					module.State = Failed
				}
			}
		case Resolved, Failed:
		}
		return existing
	}

	module := &Module{ID: identity.ID, Source: identity, State: Resolving}
	compiler.modules[identity.ID] = module
	compiler.order = append(compiler.order, identity.ID)
	compiler.stack = append(compiler.stack, identity.ID)
	defer func() { compiler.stack = compiler.stack[:len(compiler.stack)-1] }()

	if identity.Builtin {
		module.Interface = compiler.Builtins[identity.Name]
		module.State = Resolved
		return module
	}

	text, err := compiler.Loader.Load(identity)
	if err != nil {
		module.State = Failed
		compiler.addDiagnostic(semantic.CodeModuleNotFound, fmt.Sprintf("module %s was not found: %v", identity.Name, err), importSpan, "provide the sibling .ahd source file", requester, identity.ID, "", nil)
		return module
	}
	module.File = source.NewFile(compiler.nextFileID, identity.Path, text)
	compiler.nextFileID++
	lexed := lexer.Lex(module.File)
	module.Parsed = parser.Parse(module.File, lexed.Tokens)
	compiler.appendFrontendDiagnostics(identity.ID, lexed.Diagnostics)
	compiler.appendFrontendDiagnostics(identity.ID, module.Parsed.Diagnostics)
	frontendFailed := hasErrors(lexed.Diagnostics) || module.Parsed.HasErrors()

	// require(...) is local source composition, resolved before the bring
	// graph below: by the time brings are collected, every required file's
	// own top-level statements (including its own bring statements) are
	// already spliced into module.Parsed.Program, so the existing bring loop
	// picks them up with no change of its own.
	beforeRequire := len(compiler.diagnostics)
	compiler.resolveRequires(identity, module)
	for _, item := range compiler.diagnostics[beforeRequire:] {
		if item.Diagnostic.Severity == diagnostics.SeverityError {
			frontendFailed = true
			break
		}
	}

	environment := semantic.Environment{
		ModuleID: string(identity.ID), ModuleName: identity.Name,
		Imports: make(map[string]*semantic.ModuleInterface), FailedImports: make(map[string]bool),
	}
	dependencyFailed := false
	seenDependencies := make(map[ModuleID]bool)
	dependencyTargets := make(map[string]ModuleID)
	if module.Parsed.Program != nil {
		for _, statement := range module.Parsed.Program.Statements {
			bring, ok := statement.(*ast.BringStmt)
			if !ok {
				continue
			}
			dependencyIdentity, resolveError := compiler.resolveDependency(identity, bring.Module)
			if resolveError != nil {
				compiler.addDiagnostic(semantic.CodeModuleNotFound, fmt.Sprintf("module %s was not found: %v", bring.Module, resolveError), bring.Span(), "provide the sibling .ahd source file", identity.ID, "", "", nil)
				environment.FailedImports[bring.Module] = true
				dependencyFailed = true
				continue
			}
			dependency := compiler.analyze(dependencyIdentity, identity.ID, bring.Span())
			dependencyTargets[bring.Module] = dependency.ID
			if !seenDependencies[dependency.ID] {
				seenDependencies[dependency.ID] = true
				module.Dependencies = append(module.Dependencies, dependency.ID)
			}
			if dependency.State != Resolved || dependency.Interface == nil {
				environment.FailedImports[bring.Module] = true
				dependencyFailed = true
			} else {
				environment.Imports[bring.Module] = dependency.Interface
			}
		}
	}
	if module.State == Failed {
		return module
	}

	module.AnalyzeCount++
	module.Semantic = semantic.AnalyzeWithEnvironment(module.Parsed, environment)
	compiler.appendSemanticDiagnostics(identity.ID, module.Parsed.Program, module.Semantic.Diagnostics, dependencyTargets)
	if frontendFailed || dependencyFailed || module.Semantic.HasErrors() {
		module.State = Failed
		return module
	}
	module.Interface = semantic.BuildModuleInterface(module.Semantic, string(identity.ID), identity.Name)
	module.State = Resolved
	return module
}

func (compiler *Compiler) resolveDependency(importer SourceIdentity, name string) (SourceIdentity, error) {
	if interfaceValue := compiler.Builtins[name]; interfaceValue != nil {
		identity := SourceIdentity{ID: ModuleID(interfaceValue.ModuleID), Name: name, Path: interfaceValue.ModuleID, Builtin: true}
		if identity.ID == "" {
			identity.ID = ModuleID("builtin:" + name)
		}
		return identity, nil
	}
	return compiler.Resolver.Resolve(importer, name)
}

func (compiler *Compiler) cycleFrom(target ModuleID) []ModuleID {
	start := 0
	for index, id := range compiler.stack {
		if id == target {
			start = index
			break
		}
	}
	cycle := append([]ModuleID(nil), compiler.stack[start:]...)
	return append(cycle, target)
}

func formatCycle(cycle []ModuleID) string {
	if len(cycle) == 0 {
		return ""
	}
	result := string(cycle[0])
	for _, id := range cycle[1:] {
		result += " -> " + string(id)
	}
	return result
}

func (compiler *Compiler) appendFrontendDiagnostics(moduleID ModuleID, items []diagnostics.Diagnostic) {
	for _, item := range items {
		compiler.diagnostics = append(compiler.diagnostics, ModuleDiagnostic{Diagnostic: item, RequestingModule: moduleID})
	}
}

func (compiler *Compiler) appendSemanticDiagnostics(moduleID ModuleID, program *ast.Program, items []diagnostics.Diagnostic, targets map[string]ModuleID) {
	for _, item := range items {
		context := ModuleDiagnostic{Diagnostic: item, RequestingModule: moduleID}
		if program != nil {
			for _, statement := range program.Statements {
				bring, ok := statement.(*ast.BringStmt)
				if !ok || !spanContains(bring.Span(), item.Span) {
					continue
				}
				context.TargetModule = targets[bring.Module]
				for _, name := range bring.Names {
					if strings.Contains(item.Message, strconv.Quote(name)) || len(bring.Names) == 1 {
						context.RequestedSymbol = name
						break
					}
				}
				break
			}
		}
		compiler.diagnostics = append(compiler.diagnostics, context)
	}
}

func spanContains(outer, inner source.Span) bool {
	return outer.FileID == inner.FileID && outer.Start.Offset <= inner.Start.Offset && outer.End.Offset >= inner.End.Offset
}

func (compiler *Compiler) addDiagnostic(code, message string, span source.Span, hint string, requester, target ModuleID, symbol string, cycle []ModuleID) {
	compiler.diagnostics = append(compiler.diagnostics, ModuleDiagnostic{
		Diagnostic:       diagnostics.Diagnostic{Code: code, Severity: diagnostics.SeverityError, Message: message, Span: span, Hint: hint},
		RequestingModule: requester, TargetModule: target, RequestedSymbol: symbol,
		Cycle: append([]ModuleID(nil), cycle...),
	})
}

func hasErrors(items []diagnostics.Diagnostic) bool {
	for _, item := range items {
		if item.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}
