package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ahdcode/internal/framework"
	"ahdcode/internal/module"
	"ahdcode/internal/semantic"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

// CompletionItem is one completion candidate.
type CompletionItem struct {
	Label  string
	Detail string
	// Import, when set, tells the LSP layer to add an import for this symbol
	// via additionalTextEdits when the item is selected.
	Import *ImportEdit
}

// ImportEdit describes one import to add when an auto-import completion is
// accepted.
type ImportEdit struct {
	ModuleName string
	SymbolName string
}

// statementKeywords is a deliberately small, restrained set of keywords
// offered at an ordinary expression/statement position -- the control-flow
// and literal vocabulary a user is actually likely to be typing next, not
// the language's full reserved-word list (type names and modifiers belong
// to a type or modifier position, not this one, and are left out rather
// than offered indiscriminately everywhere).
var statementKeywords = []string{
	"if", "else", "while", "until", "for", "break", "continue",
	"attempt", "except", "ultimately", "toss", "return", "state",
	"true", "false", "null", "bring", "from", "require",
}

// Completion answers a completion request at the given byte offset. It
// covers, in order of precedence: a `bring`/`from` module or export name,
// a member access after `.` (module/alias members and Class members), and
// otherwise an in-scope identifier (locals, parameters, module-root
// functions/classes/constants) plus a restrained keyword list -- every
// category reusing compiler-computed facts (ResolvedSymbols, ExpressionTypes,
// ModuleInterface, StandardModuleInterfaces) rather than a hand-maintained
// catalog of names.
func (store *Store) Completion(path string, offset int) []CompletionItem {
	canonical := canonicalPath(path)
	store.mutex.Lock()
	cached := store.entries[canonical]
	store.mutex.Unlock()

	if cached == nil {
		return nil
	}
	entryPath := cached.result.EntryPath
	entryModule := cached.entryModule()
	if entryModule == nil || entryModule.Parsed.Program == nil {
		return nil
	}
	ancestors := ancestorsAtOffset(entryModule.Parsed.Program, offset, cached.fileIDFor(canonical))
	if len(ancestors) == 0 {
		return nil
	}
	innermost := ancestors[len(ancestors)-1]

	if bring, ok := innermost.(*ast.BringStmt); ok {
		return bringCompletions(cached, entryModule, bring, offset)
	}
	if member, ok := innermost.(*ast.MemberExpr); ok {
		return store.memberCompletions(cached, entryModule, member, ancestors)
	}
	return store.scopeCompletions(cached, entryPath, entryModule, ancestors, offset)
}

// bringCompletions completes a `bring <module>`, `from <module>`, or
// `from <module> bring <export>` statement. BringStmt carries no per-token
// spans for its Module/Names text (only the final parsed strings), so the
// three positions are told apart with a small, tightly bounded look at the
// statement's own raw source text up to the cursor, rather than by adding a
// second parser.
func bringCompletions(cached *entry, entryModule *module.Module, bring *ast.BringStmt, offset int) []CompletionItem {
	start := bring.Span().Start.Offset
	if offset < start || offset > len(entryModule.File.Text) {
		return nil
	}
	prefix := entryModule.File.Text[start:offset]
	switch {
	case strings.HasPrefix(prefix, "bring"):
		// A plain `bring <module>` is always a Namespace import: the only
		// thing left to type is the module name itself.
		return moduleNameCompletions(cached, lastWord(prefix))
	case strings.HasPrefix(prefix, "from") && !strings.Contains(prefix, "bring"):
		return moduleNameCompletions(cached, lastWord(prefix))
	case strings.HasPrefix(prefix, "from") && strings.Contains(prefix, "bring"):
		afterBring := prefix[strings.LastIndex(prefix, "bring")+len("bring"):]
		if strings.Contains(afterBring, "as") {
			return nil
		}
		word := lastWord(afterBring)
		moduleInterface := moduleInterfaceNamed(cached, bring.Module)
		if moduleInterface == nil {
			return nil
		}
		var items []CompletionItem
		for _, name := range moduleInterface.ExportNames {
			if strings.HasPrefix(name, word) {
				items = append(items, CompletionItem{Label: name, Detail: renderHover(moduleInterface.Exports[name])})
			}
		}
		return items
	default:
		return nil
	}
}

// lastWord returns the run of identifier characters at the end of text, the
// partial word the user is still typing.
func lastWord(text string) string {
	index := len(text)
	for index > 0 {
		r := text[index-1]
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			index--
			continue
		}
		break
	}
	return text[index:]
}

func moduleNameCompletions(cached *entry, prefix string) []CompletionItem {
	var items []CompletionItem
	for name := range semantic.StandardModuleInterfaces() {
		if strings.HasPrefix(name, prefix) {
			items = append(items, CompletionItem{Label: name, Detail: "module " + name})
		}
	}
	// The bundled first-party modules are offered by name for the same
	// reason the standard modules are: they are importable and reserved.
	// Only the *names* come from here -- every export, signature, and hover
	// for them is read from the interface the compiler builds by actually
	// compiling the module, exactly as for a module the user wrote.
	for _, name := range framework.PublicModuleNames() {
		if strings.HasPrefix(name, prefix) {
			items = append(items, CompletionItem{Label: name, Detail: "module " + name})
		}
	}
	entryModule := cached.entryModule()
	if entryModule != nil {
		directory := filepath.Dir(entryModule.File.Path)
		ownName := strings.TrimSuffix(filepath.Base(entryModule.File.Path), ".ahd")
		if files, err := os.ReadDir(directory); err == nil {
			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".ahd") {
					continue
				}
				name := strings.TrimSuffix(file.Name(), ".ahd")
				if name != "" && name != ownName && strings.HasPrefix(name, prefix) {
					items = append(items, CompletionItem{Label: name, Detail: "module " + name})
				}
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items
}

// moduleInterfaceNamed resolves a `from`/`bring` module name to its
// interface, checking the fixed standard-module catalog first and then the
// sibling modules already part of this same compile graph (a module a
// BringStmt names is compiled as a dependency as soon as its name parses,
// independent of which export names are requested afterward).
func moduleInterfaceNamed(cached *entry, name string) *semantic.ModuleInterface {
	if standard, ok := semantic.StandardModuleInterfaces()[name]; ok {
		return standard
	}
	for _, candidate := range cached.modules {
		if candidate != nil && candidate.Source.Name == name {
			return candidate.Interface
		}
	}
	return nil
}

// memberCompletions completes a `<namespace>.<partial>` or
// `<instance>.<partial>` member access, using whichever compiler fact
// already describes the receiver: ResolvedSymbols for a namespace, or
// ExpressionTypes for a Class-typed value.
func (store *Store) memberCompletions(cached *entry, entryModule *module.Module, member *ast.MemberExpr, ancestors []ast.Node) []CompletionItem {
	if receiver, ok := entryModule.Semantic.ResolvedSymbols[member.Object]; ok && receiver != nil && receiver.Kind == semantic.NamespaceSymbol && receiver.Namespace != nil {
		var items []CompletionItem
		for _, name := range receiver.Namespace.ExportNames {
			if strings.HasPrefix(name, member.Name) {
				items = append(items, CompletionItem{Label: name, Detail: renderHover(receiver.Namespace.Exports[name])})
			}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
		return items
	}
	if identifier, ok := member.Object.(*ast.IdentifierExpr); ok {
		if items := store.qualifiedModuleMemberCompletions(cached, identifier.Name, member.Name); len(items) > 0 {
			return items
		}
	}
	receiverType, ok := entryModule.Semantic.ExpressionTypes[member.Object]
	if !ok {
		return nil
	}
	class, ok := receiverType.(types.Class)
	if !ok {
		return nil
	}
	enclosingClass := enclosingClassAt(entryModule, ancestors)
	seen := make(map[string]bool)
	var items []CompletionItem
	for identity := class.Symbol; identity != nil; identity = identity.Parent {
		classSymbol := classSymbolNamed(entryModule, identity)
		if classSymbol == nil {
			break
		}
		names := make([]string, 0, len(classSymbol.Members))
		for name := range classSymbol.Members {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			candidate := classSymbol.Members[name]
			if seen[name] || candidate == nil || !strings.HasPrefix(name, member.Name) {
				continue
			}
			if candidate.Confidential && !semantic.CanAccessConfidentialMember(enclosingClass, candidate.OwnerClass) {
				continue
			}
			seen[name] = true
			items = append(items, CompletionItem{Label: name, Detail: renderHover(candidate)})
		}
	}
	return items
}

// qualifiedModuleMemberCompletions completes ModuleName.partial for a module
// not yet in the compile graph by on-demand discovery.
func (store *Store) qualifiedModuleMemberCompletions(cached *entry, moduleName, prefix string) []CompletionItem {
	if moduleInterfaceNamed(cached, moduleName) != nil {
		return nil
	}
	for _, discovered := range store.discoverUserModules(cached.result.EntryPath) {
		if discovered.Name != moduleName {
			continue
		}
		iface := store.moduleInterfaceAt(discovered.Path)
		if iface == nil {
			return nil
		}
		var items []CompletionItem
		for _, name := range iface.ExportNames {
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			exported := iface.Exports[name]
			if exported != nil && !exported.Confidential {
				items = append(items, CompletionItem{Label: name, Detail: renderHover(exported)})
			}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
		return items
	}
	return nil
}

// classSymbolNamed finds the *semantic.Symbol describing a Class identity
// among every symbol this module's analysis ever touched -- its own
// declared classes plus every imported one registered by
// registerInterfaceClasses -- so a member completion can walk up an
// inheritance chain without a second copy of class-registry logic.
func classSymbolNamed(entryModule *module.Module, identity *types.ClassSymbol) *semantic.Symbol {
	for _, candidate := range entryModule.Semantic.Symbols {
		if candidate != nil && candidate.Kind == semantic.ClassSymbol && candidate.Class == identity {
			return candidate
		}
	}
	return nil
}

// scopeCompletions completes a bare identifier prefix: every module-root
// declaration (visible everywhere in the module), every parameter and
// local binding structurally enclosing the cursor, and the restrained
// keyword list.
func (store *Store) scopeCompletions(cached *entry, entryPath string, entryModule *module.Module, ancestors []ast.Node, offset int) []CompletionItem {
	prefix := ""
	if identifier, ok := ancestors[len(ancestors)-1].(*ast.IdentifierExpr); ok {
		prefix = identifier.Name
	}

	seen := make(map[string]bool)
	var items []CompletionItem
	add := func(name, detail string) {
		if name == "" || seen[name] || !strings.HasPrefix(name, prefix) {
			return
		}
		seen[name] = true
		items = append(items, CompletionItem{Label: name, Detail: detail})
	}
	for _, symbol := range entryModule.Semantic.Symbols {
		if symbol != nil && symbol.ModuleRoot && !symbol.Builtin {
			add(symbol.Name, renderHover(symbol))
		}
	}
	for _, local := range enclosingScopeNodes(ancestors, offset, entryModule) {
		add(local.Label, local.Detail)
	}
	for _, keyword := range statementKeywords {
		add(keyword, "keyword")
	}
	store.autoImportCompletions(cached, entryPath, entryModule, prefix, seen, &items)

	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items
}

// autoImportCompletions adds completion entries for exported symbols from
// discoverable user modules that are not already in scope.
func (store *Store) autoImportCompletions(cached *entry, entryPath string, entryModule *module.Module, prefix string, seen map[string]bool, items *[]CompletionItem) {
	inScope := make(map[string]bool)
	for name := range seen {
		inScope[name] = true
	}
	for _, symbol := range entryModule.Semantic.Symbols {
		if symbol != nil && !symbol.Builtin {
			inScope[symbol.Name] = true
		}
	}
	for _, discovered := range store.discoverUserModules(entryPath) {
		if discovered.Path == entryPath {
			continue
		}
		iface := store.moduleInterfaceAt(discovered.Path)
		if iface == nil {
			continue
		}
		for _, name := range iface.ExportNames {
			if inScope[name] || !strings.HasPrefix(name, prefix) {
				continue
			}
			exported := iface.Exports[name]
			if exported == nil || exported.Confidential {
				continue
			}
			inScope[name] = true
			*items = append(*items, CompletionItem{
				Label:  name,
				Detail: "from " + discovered.Name,
				Import: &ImportEdit{ModuleName: discovered.Name, SymbolName: name},
			})
		}
	}
}

// enclosingScopeNodes approximates the locals visible at offset by walking
// the structural ancestor chain: a Function/Structure declaration
// contributes its parameters, a Block contributes the bindings its own
// statement list declares before offset, and a for-loop contributes its
// iteration variable. This is a structural approximation, not a full
// re-implementation of the analyzer's lexical scoping -- it does not
// attempt except-clause bindings or lambda captures.
func enclosingScopeNodes(ancestors []ast.Node, offset int, entryModule *module.Module) []CompletionItem {
	resolved := entryModule.Semantic.ResolvedSymbols
	var items []CompletionItem
	for _, ancestor := range ancestors {
		switch node := ancestor.(type) {
		case *ast.FunctionDecl:
			for index := range node.Parameters {
				if symbol, ok := resolved[&node.Parameters[index]]; ok && symbol != nil {
					items = append(items, CompletionItem{Label: symbol.Name, Detail: renderHover(symbol)})
				}
			}
		case *ast.StructureDecl:
			for index := range node.Parameters {
				if symbol, ok := resolved[&node.Parameters[index]]; ok && symbol != nil {
					items = append(items, CompletionItem{Label: symbol.Name, Detail: renderHover(symbol)})
				}
			}
		case *ast.LambdaExpr:
			for index := range node.Parameters {
				if symbol, ok := resolved[&node.Parameters[index]]; ok && symbol != nil {
					items = append(items, CompletionItem{Label: symbol.Name, Detail: renderHover(symbol)})
				}
			}
		case *ast.ForStmt:
			for _, symbol := range entryModule.Semantic.Symbols {
				if symbol != nil && symbol.Kind == semantic.ForSymbol && symbol.Span == node.Span() {
					items = append(items, CompletionItem{Label: symbol.Name, Detail: renderHover(symbol)})
				}
			}
		case *ast.Block:
			for _, statement := range node.Statements {
				if statement.Span().Start.Offset >= offset {
					break
				}
				declaration, ok := statement.(*ast.VariableDecl)
				if !ok {
					continue
				}
				if symbol, ok := resolved[declaration]; ok && symbol != nil {
					items = append(items, CompletionItem{Label: symbol.Name, Detail: renderHover(symbol)})
				}
			}
		case *ast.AttemptStmt:
			for _, clause := range node.Excepts {
				if clause.Name == "" || clause.Body == nil {
					continue
				}
				if !containsOffset(clause.Body.Span(), offset) {
					continue
				}
				for _, symbol := range entryModule.Semantic.Symbols {
					if symbol != nil && symbol.Kind == semantic.ExceptSymbol && symbol.Name == clause.Name && symbol.Span == clause.Span() {
						items = append(items, CompletionItem{Label: symbol.Name, Detail: renderHover(symbol)})
					}
				}
			}
		}
	}
	return items
}
