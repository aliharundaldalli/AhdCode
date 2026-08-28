package lowering

import (
	"fmt"
	"sort"
	"strings"

	"ahdcode/internal/ir"
	"ahdcode/internal/module"
	"ahdcode/internal/semantic"
	"ahdcode/internal/source"
	"ahdcode/internal/types"
)

type registry struct {
	symbols   map[*semantic.Symbol]ir.SymbolID
	callables map[*semantic.Callable]ir.CallableID
	// classes indexes every Class declaration in the compilation by canonical
	// identity so cross-module inheritance keeps one Class Symbol.
	classes map[string]*semantic.Symbol
}

func newRegistry(modules []*module.Module) *registry {
	result := &registry{
		symbols: make(map[*semantic.Symbol]ir.SymbolID), callables: make(map[*semantic.Callable]ir.CallableID),
		classes: make(map[string]*semantic.Symbol),
	}
	for _, current := range modules {
		if current == nil {
			continue
		}
		for _, symbol := range current.Semantic.Symbols {
			result.registerSymbol(current, symbol)
			result.registerClass(symbol)
		}
	}
	return result
}

func (r *registry) registerClass(symbol *semantic.Symbol) {
	if symbol == nil || symbol.Kind != semantic.ClassSymbol || symbol.Class == nil || symbol.SuperClassBinding {
		return
	}
	target := symbol
	if target.Alias != nil {
		target = target.Alias
	}
	key := classIdentityKey(target.Class)
	if existing := r.classes[key]; existing == nil || len(existing.Members) < len(target.Members) {
		r.classes[key] = target
	}
}

func (r *registry) classSymbol(identity *types.ClassSymbol) *semantic.Symbol {
	if identity == nil {
		return nil
	}
	return r.classes[classIdentityKey(identity)]
}

func classIdentityKey(identity *types.ClassSymbol) string {
	if identity == nil {
		return ""
	}
	return identity.ModuleID + "::" + identity.Name
}

func (r *registry) registerSymbol(current *module.Module, symbol *semantic.Symbol) {
	if symbol == nil {
		return
	}
	if symbol.Alias != nil {
		r.registerSymbol(current, symbol.Alias)
		r.symbols[symbol] = r.symbols[symbol.Alias]
		return
	}
	id := stableSymbolID(current, symbol)
	r.symbols[symbol] = id
	registerCallable := func(callable *semantic.Callable, constructor bool) {
		if callable == nil || callable.Signature == nil {
			return
		}
		callableID := stableCallableID(current, symbol, callable, constructor)
		r.callables[callable] = callableID
	}
	registerCallable(symbol.Callable, false)
	if symbol.OverloadSet != nil {
		for _, callable := range symbol.OverloadSet.Candidates {
			registerCallable(callable, false)
		}
	}
	registerCallable(symbol.Constructor, true)
	if len(symbol.Members) != 0 {
		names := make([]string, 0, len(symbol.Members))
		for name := range symbol.Members {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			r.registerSymbol(current, symbol.Members[name])
		}
	}
}

func stableSymbolID(current *module.Module, symbol *semantic.Symbol) ir.SymbolID {
	if symbol == nil {
		return ""
	}
	if symbol.Builtin {
		if symbol.Kind == semantic.ClassSymbol && symbol.Class != nil {
			return ir.SymbolID(classID(symbol.Class) + "::symbol")
		}
		return ir.SymbolID("builtin:core::symbol::" + symbol.Name)
	}
	if symbol.OwnerClass != nil {
		return ir.SymbolID(string(classID(symbol.OwnerClass)) + "::member::" + symbol.Name)
	}
	origin := symbol.OriginModuleID
	if origin == "" && current != nil {
		origin = string(current.ID)
	}
	if symbol.ModuleRoot {
		return ir.SymbolID(origin + "::symbol::" + symbolKindName(symbol.Kind) + "::" + symbol.Name)
	}
	return ir.SymbolID(fmt.Sprintf("%s::local::%s::%s@%d", origin, symbolKindName(symbol.Kind), symbol.Name, symbol.Span.Start.Offset))
}

func stableCallableID(current *module.Module, symbol *semantic.Symbol, callable *semantic.Callable, constructor bool) ir.CallableID {
	if callable == nil || callable.Signature == nil {
		return ""
	}
	if constructor && symbol != nil && symbol.Class != nil {
		return ir.CallableID(string(classID(symbol.Class)) + "::constructor::" + signatureKey(callable.Signature))
	}
	if callable.Declaration == nil && callable.Structure == nil && (symbol == nil || symbol.Kind != semantic.FunctionSymbol) {
		return ir.CallableID("signature:" + signatureKey(callable.Signature))
	}
	base := string(stableSymbolID(current, symbol))
	if base == "" {
		base = "signature"
	}
	return ir.CallableID(base + "::callable::" + signatureKey(callable.Signature))
}

func (r *registry) symbolID(current *module.Module, symbol *semantic.Symbol) ir.SymbolID {
	if symbol == nil {
		return ""
	}
	if symbol.Alias != nil {
		symbol = symbol.Alias
	}
	if id := r.symbols[symbol]; id != "" {
		return id
	}
	id := stableSymbolID(current, symbol)
	r.symbols[symbol] = id
	return id
}

func (r *registry) callableID(current *module.Module, symbol *semantic.Symbol, callable *semantic.Callable, constructor bool) ir.CallableID {
	if callable == nil || callable.Signature == nil {
		if symbol != nil && symbol.Builtin {
			return ir.CallableID("builtin:core::" + symbol.Name)
		}
		return ""
	}
	if id := r.callables[callable]; id != "" {
		return id
	}
	if symbol != nil {
		id := stableCallableID(current, symbol, callable, constructor)
		r.callables[callable] = id
		return id
	}
	return ir.CallableID("signature:" + signatureKey(callable.Signature))
}

func classID(symbol *types.ClassSymbol) ir.ClassID {
	if symbol == nil {
		return ""
	}
	return ir.ClassID(symbol.ModuleID + "::class::" + symbol.Name)
}

func fieldID(symbol *semantic.Symbol) ir.FieldID {
	if symbol == nil || symbol.OwnerClass == nil {
		return ""
	}
	return ir.FieldID(string(classID(symbol.OwnerClass)) + "::field::" + symbol.Name)
}

func symbolKindName(kind semantic.SymbolKind) string {
	switch kind {
	case semantic.BindingSymbol:
		return "binding"
	case semantic.ParameterSymbol:
		return "parameter"
	case semantic.FunctionSymbol:
		return "function"
	case semantic.ClassSymbol:
		return "class"
	case semantic.MemberSymbol:
		return "member"
	case semantic.ForSymbol:
		return "iteration"
	case semantic.ExceptSymbol:
		return "error"
	case semantic.BuiltinSymbol:
		return "builtin"
	case semantic.NamespaceSymbol:
		return "namespace"
	default:
		return "unknown"
	}
}

func signatureKey(signature *types.Signature) string {
	if signature == nil {
		return "unknown"
	}
	parts := make([]string, len(signature.Parameters))
	for index, parameter := range signature.Parameters {
		defaultMarker := ""
		if parameter.HasDefault {
			defaultMarker = "?"
		}
		parts[index] = parameter.Name + ":" + typeKey(parameter.Type) + defaultMarker
	}
	return "(" + strings.Join(parts, ",") + ")->" + typeKey(signature.Return)
}

func typeKey(value types.Type) string {
	switch typed := value.(type) {
	case types.List:
		return "List<" + typeKey(typed.Element) + ">"
	case types.Pair:
		return "Pair<" + typeKey(typed.Key) + "," + typeKey(typed.Value) + ">"
	case types.Function:
		return "Function" + signatureKey(typed.Signature)
	case types.Class:
		suffix := ""
		if typed.Reference {
			suffix = "&"
		}
		return string(classID(typed.Symbol)) + suffix
	default:
		return types.Display(value)
	}
}

func findSymbol(result semantic.Result, kind semantic.SymbolKind, name string, span source.Span) *semantic.Symbol {
	for _, symbol := range result.Symbols {
		if symbol != nil && symbol.Kind == kind && symbol.Name == name && symbol.Span.FileID == span.FileID && symbol.Span.Start.Offset == span.Start.Offset {
			return symbol
		}
	}
	return nil
}
