package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

// BuildModuleInterface extracts deterministic compile-time metadata from a
// successful semantic result. AST declarations and inference state never
// cross the module boundary.
func BuildModuleInterface(result Result, moduleID, name string) *ModuleInterface {
	interfaceValue := &ModuleInterface{
		ModuleID: moduleID,
		Name:     name,
		Exports:  make(map[string]*Symbol),
		Symbols:  make(map[string]*Symbol),
		Classes:  make(map[string]*Symbol),
	}
	memo := make(map[*Symbol]*Symbol)
	for _, symbol := range result.Symbols {
		if symbol == nil || symbol.Builtin {
			continue
		}
		if symbol.Kind == ClassSymbol && symbol.Class != nil {
			interfaceValue.Classes[classIdentityKey(symbol.Class)] = cloneInterfaceSymbol(symbol, memo)
		}
		if !symbol.ModuleRoot || symbol.OriginModuleID != moduleID || symbol.Kind == NamespaceSymbol {
			continue
		}
		if _, exists := interfaceValue.Symbols[symbol.Name]; exists {
			continue
		}
		cloned := cloneInterfaceSymbol(symbol, memo)
		interfaceValue.Symbols[symbol.Name] = cloned
		if !symbol.Confidential {
			interfaceValue.Exports[symbol.Name] = cloned
			interfaceValue.ExportNames = append(interfaceValue.ExportNames, symbol.Name)
		}
	}
	addReExports(interfaceValue, result, memo)
	sort.Strings(interfaceValue.ExportNames)
	return interfaceValue
}

// addReExports publishes a facade module's imported names as its own exports.
// The symbol is cloned exactly like a locally declared one, which keeps every
// consumer on the same code path -- but the clone deliberately keeps the
// original Class pointer and OriginModuleID, so the re-exported name resolves
// to the identical type the source module exports rather than to a duplicate.
func addReExports(interfaceValue *ModuleInterface, result Result, memo map[*Symbol]*Symbol) {
	for _, item := range result.ReExports {
		if item.Symbol == nil || item.Symbol.Confidential {
			continue
		}
		if _, exists := interfaceValue.Symbols[item.Name]; exists {
			continue
		}
		cloned := cloneInterfaceSymbol(item.Symbol, memo)
		interfaceValue.Symbols[item.Name] = cloned
		interfaceValue.Exports[item.Name] = cloned
		interfaceValue.ExportNames = append(interfaceValue.ExportNames, item.Name)
		if cloned.Kind == ClassSymbol && cloned.Class != nil {
			interfaceValue.Classes[classIdentityKey(cloned.Class)] = cloned
		}
	}
}

func classIdentityKey(class *types.ClassSymbol) string {
	if class == nil {
		return ""
	}
	return class.ModuleID + "\x00" + class.Name
}

func cloneInterfaceSymbol(symbol *Symbol, memo map[*Symbol]*Symbol) *Symbol {
	if symbol == nil {
		return nil
	}
	if cloned := memo[symbol]; cloned != nil {
		return cloned
	}
	cloned := &Symbol{
		Name: symbol.Name, Kind: symbol.Kind, Type: cloneType(symbol.Type),
		Constant: symbol.Constant, Confidential: symbol.Confidential,
		ModuleRoot: symbol.ModuleRoot, Builtin: symbol.Builtin,
		InitialNull: symbol.InitialNull, Class: symbol.Class,
		OwnerClass: symbol.OwnerClass, OriginModuleID: symbol.OriginModuleID,
		ConstValue: cloneConstant(symbol.ConstValue), BuiltinLiteral: symbol.BuiltinLiteral,
		ModuleOperation: symbol.ModuleOperation,
	}
	memo[symbol] = cloned
	cloned.Callable = cloneCallable(symbol.Callable)
	cloned.Constructor = cloneCallable(symbol.Constructor)
	for _, attribute := range symbol.ConstructorAttributes {
		cloned.ConstructorAttributes = append(cloned.ConstructorAttributes, cloneInterfaceSymbol(attribute, memo))
	}
	if symbol.OverloadSet != nil {
		cloned.OverloadSet = &OverloadSet{Name: symbol.OverloadSet.Name}
		for _, callable := range symbol.OverloadSet.Candidates {
			cloned.OverloadSet.Candidates = append(cloned.OverloadSet.Candidates, cloneCallable(callable))
		}
	}
	if len(symbol.Members) != 0 {
		cloned.Members = make(map[string]*Symbol, len(symbol.Members))
		for name, member := range symbol.Members {
			cloned.Members[name] = cloneInterfaceSymbol(member, memo)
		}
	}
	return cloned
}

func cloneCallable(callable *Callable) *Callable {
	if callable == nil {
		return nil
	}
	return &Callable{
		Signature:     cloneSignature(callable.Signature),
		ParameterNull: append([]NullState(nil), callable.ParameterNull...),
		ReturnNull:    callable.ReturnNull,
	}
}

func cloneSignature(signature *types.Signature) *types.Signature {
	if signature == nil {
		return nil
	}
	cloned := &types.Signature{Return: cloneType(signature.Return)}
	for _, parameter := range signature.Parameters {
		cloned.Parameters = append(cloned.Parameters, types.Parameter{
			Name: parameter.Name, Type: cloneType(parameter.Type), HasDefault: parameter.HasDefault,
		})
	}
	return cloned
}

func cloneType(value types.Type) types.Type {
	switch typed := value.(type) {
	case types.List:
		return types.List{Element: cloneType(typed.Element)}
	case types.Pair:
		return types.Pair{Key: cloneType(typed.Key), Value: cloneType(typed.Value)}
	case types.Function:
		return types.Function{Signature: cloneSignature(typed.Signature)}
	case types.Class:
		return types.Class{Symbol: typed.Symbol, Reference: typed.Reference}
	default:
		return value
	}
}
