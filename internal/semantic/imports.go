package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// installImports materializes already-resolved module interfaces in module
// scope. Resolution and source loading deliberately remain outside semantic.
func (a *analyzer) installImports(program *ast.Program) {
	localNames := moduleDeclarationNames(program)
	for _, statement := range program.Statements {
		bring, ok := statement.(*ast.BringStmt)
		if !ok {
			continue
		}
		if a.environment.FailedImports[bring.Module] {
			a.error(CodeFailedDependency, fmt.Sprintf("module %s could not be analyzed", bring.Module), bring.Span(), "fix the dependency diagnostics before importing it")
			continue
		}
		module := a.environment.Imports[bring.Module]
		if module == nil {
			a.error(CodeModuleNotFound, fmt.Sprintf("module %s was not found", bring.Module), bring.Span(), "provide the sibling .ahd module or a registered built-in module")
			continue
		}
		a.registerInterfaceClasses(module)

		if bring.Namespace {
			symbol := &Symbol{
				Name: bring.Module, Kind: NamespaceSymbol, Type: types.Invalid,
				Span: bring.Span(), ModuleRoot: true, Constant: true,
				InitialNull: NonNull, Namespace: module, OriginModuleID: a.environment.ModuleID,
			}
			a.installImportedName(bring, symbol.Name, symbol, localNames)
			continue
		}

		names := append([]string(nil), bring.Names...)
		if bring.All {
			names = append([]string(nil), module.ExportNames...)
			sort.Strings(names)
		}
		for _, name := range names {
			symbol, exists := module.Symbols[name]
			if !exists {
				a.error(CodeExportNotFound, fmt.Sprintf("module %s has no symbol %q", bring.Module, name), bring.Span(), "import a symbol declared by the target module")
				continue
			}
			if symbol.Confidential {
				a.error(CodeConfidentialAccess, fmt.Sprintf("symbol %q in module %s is Confidential", name, bring.Module), bring.Span(), "import only public module exports")
				continue
			}
			exported := module.Exports[name]
			if exported == nil {
				a.error(CodeExportNotFound, fmt.Sprintf("symbol %q in module %s is not exported", name, bring.Module), bring.Span(), "import a public module export")
				continue
			}
			a.installImportedName(bring, name, exported, localNames)
		}
	}
}

func moduleDeclarationNames(program *ast.Program) map[string]bool {
	names := make(map[string]bool)
	for _, statement := range program.Statements {
		switch declaration := statement.(type) {
		case *ast.VariableDecl:
			if declaration.Name != "" {
				names[declaration.Name] = true
			}
		case *ast.FunctionDecl:
			names[declaration.Name] = true
		case *ast.ClassDecl:
			names[declaration.Name] = true
		}
	}
	return names
}

func (a *analyzer) installImportedName(bring *ast.BringStmt, name string, symbol *Symbol, localNames map[string]bool) {
	if localNames[name] {
		a.error(CodeImportCollision, fmt.Sprintf("imported name %q collides with a module declaration", name), bring.Span(), "rename one declaration or use a namespace import")
		return
	}
	if existing, exists := a.module.local(name); exists {
		a.error(CodeImportCollision, fmt.Sprintf("imported name %q collides with an existing symbol", name), bring.Span(), fmt.Sprintf("existing symbol has type %s", types.Display(existing.Type)))
		return
	}
	a.module.symbols[name] = symbol
	a.result.Symbols = append(a.result.Symbols, symbol)
	a.result.ResolvedSymbols[bring] = symbol
	if symbol.Kind == ClassSymbol && symbol.Class != nil {
		a.classes[name] = symbol
		a.classByType[symbol.Class] = symbol
	}
}

func (a *analyzer) registerInterfaceClasses(module *ModuleInterface) {
	if module == nil {
		return
	}
	keys := make([]string, 0, len(module.Classes))
	for key := range module.Classes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		symbol := module.Classes[key]
		if symbol != nil && symbol.Class != nil {
			existing := a.classSymbolFor(symbol.Class)
			if existing == nil {
				a.result.Symbols = append(a.result.Symbols, symbol)
				existing = symbol
			}
			a.classByType[symbol.Class] = existing
		}
	}
}
