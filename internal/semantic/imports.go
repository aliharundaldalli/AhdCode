package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/source"
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
			bindingName := bring.Module
			if bring.Alias != "" {
				bindingName = bring.Alias
			}
			symbol := &Symbol{
				Name: bindingName, Kind: NamespaceSymbol, Type: types.Invalid,
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
			a.recordReExport(name, exported)
		}
	}
}

// recordReExport republishes one imported name through this module's own
// interface. Only AhdCode's bundled first-party facade modules enable this;
// for every application module the slice stays empty and `bring` remains
// non-transitive exactly as before.
func (a *analyzer) recordReExport(name string, symbol *Symbol) {
	if !a.environment.ReExportImports || symbol == nil {
		return
	}
	for _, existing := range a.result.ReExports {
		if existing.Name == name {
			return
		}
	}
	a.result.ReExports = append(a.result.ReExports, ReExport{Name: name, Symbol: symbol})
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
	fileID := bring.Span().FileID
	if existing, exists := a.module.local(name); exists {
		// A require(...)-composed program analyzes every required file's
		// statements in one pass, so the same `bring Module` line, written
		// independently by two different required files, is expected and
		// reaches this path twice for the same name. That is not a
		// collision: it is a second file legitimately declaring the
		// dependency it itself uses, and it only extends which files may see
		// the existing symbol -- and only the first time a given file does
		// so; that same file writing the identical bring twice is still an
		// ordinary duplicate. Anything else landing on the same name is a
		// genuine collision, exactly as it already was for a single file.
		visibility := a.importVisibility[existing]
		if visibility != nil && a.importModuleOf[existing] == bring.Module && !visibility[fileID] {
			visibility[fileID] = true
			a.result.ResolvedSymbols[bring] = existing
			return
		}
		a.error(CodeImportCollision, fmt.Sprintf("imported name %q collides with an existing symbol", name), bring.Span(), fmt.Sprintf("existing symbol has type %s", types.Display(existing.Type)))
		return
	}
	// The installed binding is the exact symbol object passed in -- often the
	// same *Symbol the source module's own ModuleInterface holds -- never a
	// private copy: see the importVisibility/importModuleOf fields' comment
	// on the analyzer struct for why a copy is unsafe here.
	a.module.symbols[name] = symbol
	a.result.Symbols = append(a.result.Symbols, symbol)
	a.result.ResolvedSymbols[bring] = symbol
	if symbol.Kind == ClassSymbol && symbol.Class != nil {
		a.classes[name] = symbol
		a.classByType[symbol.Class] = symbol
	}
	if a.importVisibility == nil {
		a.importVisibility = make(map[*Symbol]map[source.FileID]bool)
		a.importModuleOf = make(map[*Symbol]string)
	}
	a.importVisibility[symbol] = map[source.FileID]bool{fileID: true}
	a.importModuleOf[symbol] = bring.Module
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
