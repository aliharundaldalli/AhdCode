package lowering

import (
	"sort"

	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

func (lowerer *moduleLowerer) lowerModule() *ir.Module {
	result := &ir.Module{ID: ir.ModuleID(lowerer.module.ID), Name: lowerer.module.Source.Name, SourcePath: lowerer.module.Source.Path}
	for _, dependency := range lowerer.module.Dependencies {
		result.Dependencies = append(result.Dependencies, ir.ModuleID(dependency))
	}
	result.Init.Span = lowerer.module.Parsed.Program.Span()
	globalOrder := 0
	for _, statement := range lowerer.module.Parsed.Program.Statements {
		switch value := statement.(type) {
		case *ast.VariableDecl:
			if symbol := lowerer.semantic.ResolvedSymbols[value]; symbol != nil && symbol.ModuleRoot && symbol.Alias == nil {
				result.Globals = append(result.Globals, lowerer.lowerGlobal(value, symbol, globalOrder))
				globalOrder++
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

func (lowerer *moduleLowerer) lowerGlobal(declaration *ast.VariableDecl, symbol *semantic.Symbol, order int) *ir.Global {
	typeValue := lowerType(symbol.Type)
	return &ir.Global{
		Span: declaration.Span(), ID: lowerer.compilation.registry.symbolID(lowerer.module, symbol), Name: symbol.Name, Order: order,
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
		function.Overrides = lowerer.overriddenCallable(owner, declaration.Name, callable)
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

// overriddenCallable records the parent method this declaration replaces. The
// dispatch slot is therefore an explicit IR fact rather than a backend guess.
func (lowerer *moduleLowerer) overriddenCallable(owner *semantic.Symbol, name string, callable *semantic.Callable) ir.CallableID {
	if owner.Class == nil || owner.Class.Parent == nil {
		return ""
	}
	for parent := lowerer.classSymbol(owner.Class.Parent); parent != nil; parent = lowerer.classSymbol(parent.Class.Parent) {
		member := parent.Members[name]
		if member != nil && member.Kind == semantic.FunctionSymbol && member.Callable != nil &&
			callableSignaturesMatch(member.Callable, callable) {
			return lowerer.compilation.registry.callableID(lowerer.module, member, member.Callable, false)
		}
		if parent.Class == nil || parent.Class.Parent == nil {
			break
		}
	}
	return ""
}

func callableSignaturesMatch(left, right *semantic.Callable) bool {
	if left == nil || right == nil {
		return false
	}
	return ir.EqualSignature(lowerSignature(left.Signature), lowerSignature(right.Signature))
}

// classSymbol resolves canonical Class identity to its semantic Symbol, in
// this module or in an already-analyzed dependency.
func (lowerer *moduleLowerer) classSymbol(identity *types.ClassSymbol) *semantic.Symbol {
	return lowerer.compilation.registry.classSymbol(identity)
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
	function := lowerer.newConstructor(class, declaration)
	if function == nil {
		return nil
	}
	previousReturn, previousReceiver, previousOwner := lowerer.currentReturn, lowerer.currentReceiver, lowerer.currentOwner
	lowerer.currentReturn, lowerer.currentReceiver, lowerer.currentOwner = function.Signature.Return, function.Receiver, function.Owner
	if declaration != nil {
		for index := range declaration.Parameters {
			parameter := &declaration.Parameters[index]
			if parameter.InheritedAttributes || parameter.Default == nil {
				continue
			}
			for position := range function.Parameters {
				if function.Parameters[position].Name == parameter.Name {
					function.Parameters[position].Default = lowerer.lowerExprExpected(parameter.Default, function.Parameters[position].Type)
					break
				}
			}
		}
	}
	function.Body = lowerer.attributeInitialization(class, function)
	if declaration != nil {
		body := lowerer.lowerBlock(declaration.Body)
		function.Body.Statements = append(function.Body.Statements, body.Statements...)
	}
	lowerer.currentReturn, lowerer.currentReceiver, lowerer.currentOwner = previousReturn, previousReceiver, previousOwner
	return function
}

// newConstructor builds the constructor shell from the frontend's expanded
// construction contract, including the parent slots that SuperClass.attributes
// contributed.
func (lowerer *moduleLowerer) newConstructor(class *semantic.Symbol, declaration *ast.StructureDecl) *ir.Function {
	callable := class.Constructor
	if callable == nil || callable.Signature == nil {
		return nil
	}
	id := lowerer.compilation.registry.callableID(lowerer.module, class, callable, true)
	span := class.Span
	if declaration != nil {
		span = declaration.Span()
	}
	function := &ir.Function{
		Span: span, ID: id, Symbol: lowerer.compilation.registry.symbolID(lowerer.module, class), Name: class.Name,
		Kind: ir.ConstructorFunction, Owner: classID(class.Class), Receiver: ir.SymbolID(string(id) + "::receiver"),
		Signature: *lowerSignature(callable.Signature), ReturnNull: ir.NonNull,
	}
	declared := declaredParameters(declaration)
	for index, parameter := range callable.Signature.Parameters {
		parameterType := lowerType(parameter.Type)
		item := ir.Parameter{ID: "", Name: parameter.Name, Type: parameterType, NullState: ir.NonNull}
		if index < len(callable.ParameterNull) {
			item.NullState = lowerNull(callable.ParameterNull[index])
		}
		if source := declared[parameter.Name]; source != nil {
			item.Span = source.Span()
			item.ID = lowerer.compilation.registry.symbolID(lowerer.module, findSymbol(lowerer.semantic, semantic.ParameterSymbol, parameter.Name, source.Span()))
		}
		if item.ID == "" {
			item.ID = ir.SymbolID(string(function.ID) + "::parameter::" + parameter.Name)
		}
		function.Parameters = append(function.Parameters, item)
	}
	lowerer.linkParentConstructor(class, function, declaration)
	return function
}

func declaredParameters(declaration *ast.StructureDecl) map[string]*ast.Parameter {
	result := make(map[string]*ast.Parameter)
	if declaration == nil {
		return result
	}
	for index := range declaration.Parameters {
		parameter := &declaration.Parameters[index]
		if !parameter.InheritedAttributes {
			result[parameter.Name] = parameter
		}
	}
	return result
}

// linkParentConstructor records which of this constructor's parameters feed
// the parent construction contract, so the backend runs the parent structure
// body instead of re-deriving inherited initialization.
func (lowerer *moduleLowerer) linkParentConstructor(class *semantic.Symbol, function *ir.Function, declaration *ast.StructureDecl) {
	if class.Class == nil || class.Class.Parent == nil {
		return
	}
	parent := lowerer.classSymbol(class.Class.Parent)
	if parent == nil || parent.Constructor == nil || parent.Constructor.Signature == nil {
		return
	}
	inherits := declaration == nil
	if declaration != nil {
		for index := range declaration.Parameters {
			if declaration.Parameters[index].InheritedAttributes {
				inherits = true
			}
		}
	}
	if !inherits {
		return
	}
	count := len(parent.Constructor.Signature.Parameters)
	offset := inheritedOffset(declaration, count, len(function.Parameters))
	if offset < 0 || offset+count > len(function.Parameters) {
		return
	}
	function.ParentConstructor = lowerer.compilation.registry.callableID(lowerer.module, parent, parent.Constructor, true)
	for index := 0; index < count; index++ {
		function.ParentArguments = append(function.ParentArguments, offset+index)
	}
}

// inheritedOffset is the parameter index at which the parent slots were
// spliced into the expanded constructor signature.
func inheritedOffset(declaration *ast.StructureDecl, count, total int) int {
	if declaration == nil {
		if count > total {
			return -1
		}
		return 0
	}
	offset := 0
	for index := range declaration.Parameters {
		if declaration.Parameters[index].InheritedAttributes {
			return offset
		}
		offset++
	}
	return -1
}

// attributeInitialization makes the frontend decision "a non-Local structure
// parameter declares the like-named instance attribute" explicit in the IR, so
// the backend never has to infer field initialization from parameter names.
// Inherited slots are left to the parent constructor.
func (lowerer *moduleLowerer) attributeInitialization(class *semantic.Symbol, function *ir.Function) ir.Block {
	block := ir.Block{Span: function.Span}
	inherited := make(map[int]bool, len(function.ParentArguments))
	for _, index := range function.ParentArguments {
		inherited[index] = true
	}
	for index, parameter := range function.Parameters {
		if inherited[index] || index >= len(class.ConstructorAttributes) {
			continue
		}
		member := class.ConstructorAttributes[index]
		if member == nil {
			continue
		}
		field := fieldID(member)
		if field == "" {
			continue
		}
		receiver := &ir.LoadExpr{ExprBase: ir.ExprBase{Span: parameter.Span, Type: ir.Type{Kind: ir.ClassType, Class: function.Owner}, NullState: ir.NonNull}, Symbol: function.Receiver}
		value := &ir.LoadExpr{ExprBase: ir.ExprBase{Span: parameter.Span, Type: parameter.Type, NullState: parameter.NullState}, Symbol: parameter.ID}
		block.Statements = append(block.Statements, &ir.AssignStmt{
			StmtBase: ir.StmtBase{Span: parameter.Span},
			Target:   ir.Target{Kind: ir.FieldTarget, Type: parameter.Type, Field: field, Receiver: receiver},
			Value:    value,
		})
	}
	return block
}

func (lowerer *moduleLowerer) lowerSyntheticConstructor(class *semantic.Symbol) *ir.Function {
	return lowerer.lowerConstructor(nil, class)
}
