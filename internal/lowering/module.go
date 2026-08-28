package lowering

import (
	"sort"

	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
	"ahdcode/internal/syntax/ast"
)

func (lowerer *moduleLowerer) lowerModule() *ir.Module {
	result := &ir.Module{ID: ir.ModuleID(lowerer.module.ID), Name: lowerer.module.Source.Name, SourcePath: lowerer.module.Source.Path}
	for _, dependency := range lowerer.module.Dependencies {
		result.Dependencies = append(result.Dependencies, ir.ModuleID(dependency))
	}
	result.Init.Span = lowerer.module.Parsed.Program.Span()
	for _, statement := range lowerer.module.Parsed.Program.Statements {
		switch value := statement.(type) {
		case *ast.VariableDecl:
			if symbol := lowerer.semantic.ResolvedSymbols[value]; symbol != nil && symbol.ModuleRoot && symbol.Alias == nil {
				result.Globals = append(result.Globals, lowerer.lowerGlobal(value, symbol))
			}
		case *ast.FunctionDecl:
			if function := lowerer.lowerFunction(value, nil); function != nil {
				result.Functions = append(result.Functions, function)
			}
		case *ast.ClassDecl:
			class, functions := lowerer.lowerClass(value)
			if class != nil {
				result.Classes = append(result.Classes, class)
			}
			result.Functions = append(result.Functions, functions...)
		case *ast.BringStmt:
		default:
			if lowered := lowerer.lowerStatement(statement); lowered != nil {
				result.Init.Statements = append(result.Init.Statements, lowered)
			}
		}
	}
	sort.Slice(result.Globals, func(i, j int) bool { return result.Globals[i].ID < result.Globals[j].ID })
	sort.Slice(result.Functions, func(i, j int) bool { return result.Functions[i].ID < result.Functions[j].ID })
	sort.Slice(result.Classes, func(i, j int) bool { return result.Classes[i].ID < result.Classes[j].ID })
	return result
}

func (lowerer *moduleLowerer) lowerGlobal(declaration *ast.VariableDecl, symbol *semantic.Symbol) *ir.Global {
	typeValue := lowerType(symbol.Type)
	return &ir.Global{
		Span: declaration.Span(), ID: lowerer.compilation.registry.symbolID(lowerer.module, symbol), Name: symbol.Name,
		Type: typeValue, Constant: symbol.Constant, Confidential: symbol.Confidential,
		NullState: lowerNull(symbol.InitialNull), Initializer: lowerer.lowerExprExpected(declaration.Initializer, typeValue),
	}
}

func (lowerer *moduleLowerer) lowerFunction(declaration *ast.FunctionDecl, owner *semantic.Symbol) *ir.Function {
	symbol := lowerer.semantic.ResolvedSymbols[declaration]
	if symbol == nil {
		lowerer.compilation.error(CodeMissingSemantic, "Function declaration has no resolved Symbol", declaration.Span())
		return nil
	}
	callable := callableForDeclaration(symbol, declaration)
	if callable == nil || callable.Signature == nil {
		lowerer.compilation.error(CodeMissingSemantic, "Function declaration has no concrete Callable", declaration.Span())
		return nil
	}
	callableID := lowerer.compilation.registry.callableID(lowerer.module, symbol, callable, false)
	function := &ir.Function{
		Span: declaration.Span(), ID: callableID, Symbol: lowerer.compilation.registry.symbolID(lowerer.module, symbol), Name: declaration.Name,
		Kind: ir.OrdinaryFunction, Signature: *lowerSignature(callable.Signature), ReturnNull: lowerNull(callable.ReturnNull),
		Confidential: symbol.Confidential, Override: declaration.Flavor == ast.FunctionOverride,
	}
	if owner != nil && owner.Class != nil {
		function.Kind = ir.MethodFunction
		function.Owner = classID(owner.Class)
		function.Receiver = ir.SymbolID(string(callableID) + "::receiver")
	}
	previousReturn, previousReceiver, previousOwner := lowerer.currentReturn, lowerer.currentReceiver, lowerer.currentOwner
	lowerer.currentReturn, lowerer.currentReceiver, lowerer.currentOwner = function.Signature.Return, function.Receiver, function.Owner
	for index := range declaration.Parameters {
		parameter := &declaration.Parameters[index]
		if index >= len(callable.Signature.Parameters) {
			break
		}
		semanticParameter := findSymbol(lowerer.semantic, semantic.ParameterSymbol, parameter.Name, parameter.Span())
		parameterType := lowerType(callable.Signature.Parameters[index].Type)
		id := lowerer.compilation.registry.symbolID(lowerer.module, semanticParameter)
		if id == "" {
			id = ir.SymbolID(string(callableID) + "::parameter::" + parameter.Name)
		}
		item := ir.Parameter{Span: parameter.Span(), ID: id, Name: parameter.Name, Type: parameterType, NullState: ir.NonNull}
		if index < len(callable.ParameterNull) {
			item.NullState = lowerNull(callable.ParameterNull[index])
		}
		item.Default = lowerer.lowerExprExpected(parameter.Default, parameterType)
		function.Parameters = append(function.Parameters, item)
	}
	function.Body = lowerer.lowerBlock(declaration.Body)
	lowerer.currentReturn, lowerer.currentReceiver, lowerer.currentOwner = previousReturn, previousReceiver, previousOwner
	return function
}

func callableForDeclaration(symbol *semantic.Symbol, declaration *ast.FunctionDecl) *semantic.Callable {
	if symbol == nil {
		return nil
	}
	if symbol.Callable != nil && symbol.Callable.Declaration == declaration {
		return symbol.Callable
	}
	if symbol.OverloadSet != nil {
		for _, callable := range symbol.OverloadSet.Candidates {
			if callable != nil && callable.Declaration == declaration {
				return callable
			}
		}
	}
	return nil
}

func (lowerer *moduleLowerer) lowerClass(declaration *ast.ClassDecl) (*ir.Class, []*ir.Function) {
	symbol := lowerer.semantic.ResolvedSymbols[declaration]
	if symbol == nil || symbol.Class == nil {
		lowerer.compilation.error(CodeMissingSemantic, "Class declaration has no canonical Class identity", declaration.Span())
		return nil, nil
	}
	result := &ir.Class{
		Span: declaration.Span(), ID: classID(symbol.Class), Symbol: lowerer.compilation.registry.symbolID(lowerer.module, symbol),
		Name: symbol.Name, Confidential: symbol.Confidential,
	}
	if symbol.Class.Parent != nil {
		result.Parent = classID(symbol.Class.Parent)
	}
	memberNames := make([]string, 0, len(symbol.Members))
	for name := range symbol.Members {
		memberNames = append(memberNames, name)
	}
	sort.Strings(memberNames)
	for _, name := range memberNames {
		member := symbol.Members[name]
		if member == nil || member.Kind == semantic.FunctionSymbol {
			continue
		}
		result.Fields = append(result.Fields, ir.Field{
			ID: fieldID(member), Name: member.Name, Type: lowerType(member.Type), NullState: lowerNull(member.InitialNull),
			Constant: member.Constant, Confidential: member.Confidential,
		})
	}
	var functions []*ir.Function
	foundStructure := false
	for _, member := range declaration.Members {
		switch value := member.(type) {
		case *ast.StructureDecl:
			foundStructure = true
			constructor := lowerer.lowerConstructor(value, symbol)
			if constructor != nil {
				result.Constructor = constructor.ID
				functions = append(functions, constructor)
			}
		case *ast.FunctionDecl:
			function := lowerer.lowerFunction(value, symbol)
			if function != nil {
				result.Methods = append(result.Methods, function.ID)
				functions = append(functions, function)
			}
		}
	}
	if !foundStructure && symbol.Constructor != nil {
		constructor := lowerer.lowerSyntheticConstructor(symbol)
		result.Constructor = constructor.ID
		functions = append(functions, constructor)
	}
	sort.Slice(result.Fields, func(i, j int) bool { return result.Fields[i].ID < result.Fields[j].ID })
	sort.Slice(result.Methods, func(i, j int) bool { return result.Methods[i] < result.Methods[j] })
	return result, functions
}

func (lowerer *moduleLowerer) lowerConstructor(declaration *ast.StructureDecl, class *semantic.Symbol) *ir.Function {
	callable := class.Constructor
	if callable == nil || callable.Signature == nil {
		return nil
	}
	id := lowerer.compilation.registry.callableID(lowerer.module, class, callable, true)
	function := &ir.Function{
		Span: declaration.Span(), ID: id, Symbol: lowerer.compilation.registry.symbolID(lowerer.module, class), Name: class.Name,
		Kind: ir.ConstructorFunction, Owner: classID(class.Class), Receiver: ir.SymbolID(string(id) + "::receiver"),
		Signature: *lowerSignature(callable.Signature), ReturnNull: ir.NonNull,
	}
	previousReturn, previousReceiver, previousOwner := lowerer.currentReturn, lowerer.currentReceiver, lowerer.currentOwner
	lowerer.currentReturn, lowerer.currentReceiver, lowerer.currentOwner = function.Signature.Return, function.Receiver, function.Owner
	parameterIndex := 0
	for index := range declaration.Parameters {
		parameter := &declaration.Parameters[index]
		if parameter.InheritedAttributes {
			continue
		}
		if parameterIndex >= len(callable.Signature.Parameters) {
			break
		}
		semanticParameter := findSymbol(lowerer.semantic, semantic.ParameterSymbol, parameter.Name, parameter.Span())
		parameterType := lowerType(callable.Signature.Parameters[parameterIndex].Type)
		id := lowerer.compilation.registry.symbolID(lowerer.module, semanticParameter)
		if id == "" {
			id = ir.SymbolID(string(function.ID) + "::parameter::" + parameter.Name)
		}
		item := ir.Parameter{Span: parameter.Span(), ID: id, Name: parameter.Name, Type: parameterType, NullState: ir.NonNull}
		if parameterIndex < len(callable.ParameterNull) {
			item.NullState = lowerNull(callable.ParameterNull[parameterIndex])
		}
		item.Default = lowerer.lowerExprExpected(parameter.Default, parameterType)
		function.Parameters = append(function.Parameters, item)
		parameterIndex++
	}
	function.Body = lowerer.lowerBlock(declaration.Body)
	lowerer.currentReturn, lowerer.currentReceiver, lowerer.currentOwner = previousReturn, previousReceiver, previousOwner
	return function
}

func (lowerer *moduleLowerer) lowerSyntheticConstructor(class *semantic.Symbol) *ir.Function {
	callable := class.Constructor
	id := lowerer.compilation.registry.callableID(lowerer.module, class, callable, true)
	return &ir.Function{ID: id, Symbol: lowerer.compilation.registry.symbolID(lowerer.module, class), Name: class.Name, Kind: ir.ConstructorFunction, Owner: classID(class.Class), Receiver: ir.SymbolID(string(id) + "::receiver"), Signature: *lowerSignature(callable.Signature), ReturnNull: ir.NonNull}
}
