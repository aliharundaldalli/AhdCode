package ir

import (
	"fmt"
	"strings"

	"ahdcode/internal/diagnostics"
	"ahdcode/internal/source"
)

const (
	CodeUnresolvedIdentity = "IR001"
	CodeMissingType        = "IR002"
	CodeInvalidConversion  = "IR003"
	CodeInvalidCall        = "IR004"
	CodeInvalidReturn      = "IR005"
	CodeInvalidClass       = "IR006"
	CodeMalformedNode      = "IR007"
)

func Validate(compilation *Compilation) []diagnostics.Diagnostic {
	validator := &validator{
		symbols: make(map[SymbolID]Type), callables: make(map[CallableID]Signature),
		captures: make(map[CallableID]int),
		classes:  make(map[ClassID]bool), parents: make(map[ClassID]ClassID), fields: make(map[FieldID]bool),
	}
	if compilation == nil {
		validator.error(CodeMalformedNode, "nil CompilationIR", source.Span{})
		return validator.diagnostics
	}
	for _, module := range compilation.Modules {
		if module == nil || module.ID == "" {
			validator.error(CodeUnresolvedIdentity, "module has no canonical identity", source.Span{})
			continue
		}
		for _, global := range module.Globals {
			if global != nil {
				validator.symbols[global.ID] = global.Type
			}
		}
		for _, function := range module.Functions {
			if function == nil {
				continue
			}
			validator.callables[function.ID] = function.Signature
			validator.captures[function.ID] = function.Captures
			validator.symbols[function.Symbol] = Type{Kind: FunctionType, Signature: &function.Signature}
			for _, parameter := range function.Parameters {
				validator.symbols[parameter.ID] = parameter.Type
			}
			if function.Receiver != "" {
				validator.symbols[function.Receiver] = Type{Kind: ClassType, Class: function.Owner}
			}
		}
		for _, class := range module.Classes {
			if class == nil {
				continue
			}
			validator.classes[class.ID] = true
			validator.parents[class.ID] = class.Parent
			validator.symbols[class.Symbol] = Type{Kind: ClassType, Class: class.ID, Reference: true}
			for _, field := range class.Fields {
				validator.fields[field.ID] = true
			}
		}
	}
	for _, module := range compilation.Modules {
		validator.validateModule(module)
	}
	return validator.diagnostics
}

type validator struct {
	diagnostics []diagnostics.Diagnostic
	symbols     map[SymbolID]Type
	callables   map[CallableID]Signature
	// captures records how many leading parameters of each callable are
	// closure captures, so a Function value can be checked against it.
	captures map[CallableID]int
	classes  map[ClassID]bool
	parents  map[ClassID]ClassID
	fields   map[FieldID]bool
}

func (v *validator) validateModule(module *Module) {
	if module == nil {
		return
	}
	for _, global := range module.Globals {
		if global == nil || global.ID == "" {
			v.error(CodeUnresolvedIdentity, "global has no SymbolID", spanOfGlobal(global))
			continue
		}
		v.requireType(global.Type, global.Span)
	}
	for _, class := range module.Classes {
		if class == nil {
			continue
		}
		if class.ID == "" || class.Symbol == "" {
			v.error(CodeInvalidClass, "Class has an unresolved identity", class.Span)
		}
		if class.Parent != "" && !v.classes[class.Parent] && !isExternalClass(class.Parent) {
			v.error(CodeInvalidClass, fmt.Sprintf("Class %s has unknown parent %s", class.ID, class.Parent), class.Span)
		}
		for _, field := range class.Fields {
			if field.ID == "" {
				v.error(CodeInvalidClass, fmt.Sprintf("Class %s has a field without FieldID", class.ID), class.Span)
			}
			v.requireType(field.Type, class.Span)
		}
		if class.Constructor == "" {
			v.error(CodeInvalidClass, fmt.Sprintf("Class %s has no constructor CallableID", class.ID), class.Span)
		} else {
			v.requireCallable(class.Constructor, class.Span)
		}
		for _, method := range class.Methods {
			v.requireCallable(method, class.Span)
		}
	}
	for _, function := range module.Functions {
		if function == nil {
			continue
		}
		if function.ID == "" || function.Symbol == "" {
			v.error(CodeUnresolvedIdentity, "Function has no stable identity", function.Span)
		}
		v.requireType(function.Signature.Return, function.Span)
		for _, parameter := range function.Parameters {
			if parameter.ID == "" {
				v.error(CodeUnresolvedIdentity, "Function parameter has no SymbolID", parameter.Span)
			}
			v.requireType(parameter.Type, parameter.Span)
			v.validateExpr(parameter.Default)
		}
		v.validateBlock(function.Body, &function.Signature.Return)
	}
	v.validateBlock(module.Init, nil)
}

func (v *validator) validateBlock(block Block, returnType *Type) {
	for _, statement := range block.Statements {
		if statement == nil {
			v.error(CodeMalformedNode, "nil statement", block.Span)
			continue
		}
		switch value := statement.(type) {
		case *ExprStmt:
			v.validateExpr(value.Value)
		case *BindingStmt:
			v.requireType(value.Type, value.Span)
			if value.Symbol == "" {
				v.error(CodeUnresolvedIdentity, "binding has no SymbolID", value.Span)
			} else {
				v.symbols[value.Symbol] = value.Type
			}
			v.validateExpr(value.Initializer)
		case *AssignStmt:
			v.validateTarget(value.Target, value.Span)
			v.validateExpr(value.Value)
		case *CompoundAssignStmt:
			v.validateTarget(value.Target, value.Span)
			if value.Op == "" && value.Protocol == "" {
				v.error(CodeMalformedNode, "compound assignment has no typed operation", value.Span)
			}
			if value.Protocol != "" {
				v.requireCallable(value.Protocol, value.Span)
			}
			v.validateExpr(value.Value)
		case *UpdateStmt:
			v.validateTarget(value.Target, value.Span)
			if value.Delta != 1 && value.Delta != -1 {
				v.error(CodeMalformedNode, "update delta must be +1 or -1", value.Span)
			}
		case *IfStmt:
			for _, branch := range value.Branches {
				v.validateExpr(branch.Condition)
				v.validateBlock(branch.Body, returnType)
			}
			if value.Else != nil {
				v.validateBlock(*value.Else, returnType)
			}
		case *WhileStmt:
			v.validateExpr(value.Condition)
			v.validateBlock(value.Body, returnType)
		case *DoUntilStmt:
			v.validateBlock(value.Body, returnType)
			v.validateExpr(value.Condition)
			if !value.ContinueChecksCondition {
				v.error(CodeMalformedNode, "DoUntil continue edge skips its condition", value.Span)
			}
		case *ForStmt:
			v.requireType(value.IterationType, value.Span)
			if value.Iteration == "" {
				v.error(CodeUnresolvedIdentity, "for iteration has no SymbolID", value.Span)
			} else {
				v.symbols[value.Iteration] = value.IterationType
			}
			v.validateExpr(value.Iterable)
			v.validateBlock(value.Body, returnType)
			// A lazy range carries no collection, so only collection iteration
			// requires the shallow snapshot marker.
			if !value.Snapshot && value.Kind != IntRange {
				v.error(CodeMalformedNode, "for loop lacks snapshot semantics", value.Span)
			}
			if value.Snapshot && value.Kind == IntRange {
				v.error(CodeMalformedNode, "a lazy range must not be snapshotted", value.Span)
			}
		case *StateStmt:
			if value.Temp == "" || !value.NoFallthrough {
				v.error(CodeMalformedNode, "state is missing single-evaluation/no-fallthrough metadata", value.Span)
			}
			v.validateExpr(value.Value)
			for _, item := range value.Cases {
				v.validateExpr(item.Match)
				v.validateBlock(item.Body, returnType)
			}
		case *AttemptStmt:
			v.validateBlock(value.Body, returnType)
			for _, handler := range value.Handlers {
				if handler.Class == "" {
					v.error(CodeInvalidClass, "error handler has no ClassID", value.Span)
				}
				if handler.Binding != "" {
					v.symbols[handler.Binding] = Type{Kind: ClassType, Class: handler.Class}
				}
				v.validateBlock(handler.Body, returnType)
			}
			if value.Ultimately != nil {
				v.validateBlock(*value.Ultimately, returnType)
			}
			if !value.FinallyAlways && value.Ultimately != nil {
				v.error(CodeMalformedNode, "ultimately block is not marked always-run", value.Span)
			}
		case *TossStmt:
			v.validateExpr(value.Value)
			if value.ErrorClass == "" {
				v.error(CodeInvalidClass, "toss has no resolved Error ClassID", value.Span)
			}
		case *ReturnStmt:
			if returnType == nil {
				v.error(CodeInvalidReturn, "return outside FunctionIR", value.Span)
				break
			}
			if !EqualType(value.ReturnType, *returnType) {
				v.error(CodeInvalidReturn, "return metadata does not match FunctionIR return type", value.Span)
			}
			if returnType.Kind == NothingType && value.Value != nil {
				v.error(CodeInvalidReturn, "Nothing return carries a value", value.Span)
			}
			if returnType.Kind != NothingType && value.Value == nil {
				v.error(CodeInvalidReturn, "typed return has no value", value.Span)
			}
			v.validateExpr(value.Value)
			if value.Value != nil && !v.typeAssignable(value.Value.ExprMeta().Type, *returnType) {
				v.error(CodeInvalidReturn, "return value type does not match FunctionIR return type", value.Span)
			}
		case *BreakStmt, *ContinueStmt:
		default:
			v.error(CodeMalformedNode, fmt.Sprintf("unknown statement %T", statement), statement.StatementSpan())
		}
	}
}

func (v *validator) validateTarget(target Target, span source.Span) {
	v.requireType(target.Type, span)
	switch target.Kind {
	case SymbolTarget:
		v.requireSymbol(target.Symbol, span)
	case FieldTarget:
		if target.Field == "" || !v.fields[target.Field] {
			v.error(CodeUnresolvedIdentity, "field target is unresolved", span)
		}
		v.validateExpr(target.Receiver)
	case IndexTarget:
		v.validateExpr(target.Receiver)
		v.validateExpr(target.Index)
	default:
		v.error(CodeMalformedNode, "invalid assignment target", span)
	}
}

func (v *validator) validateExpr(expression Expr) {
	if expression == nil {
		return
	}
	meta := expression.ExprMeta()
	v.requireType(meta.Type, meta.Span)
	switch value := expression.(type) {
	case *LiteralExpr, *NullExpr:
	case *ClassRefExpr:
		if !v.classes[value.Class] && !isExternalClass(value.Class) {
			v.error(CodeInvalidClass, fmt.Sprintf("unknown Class reference %s", value.Class), meta.Span)
		}
	case *LoadExpr:
		v.requireSymbol(value.Symbol, meta.Span)
	case *FunctionValueExpr:
		v.requireSymbol(value.Symbol, meta.Span)
		v.requireCallable(value.Callable, meta.Span)
		// A capturing lambda's value must bind exactly the leading capture
		// parameters its lifted callable declares, or the closure would call
		// that callable with the wrong arity.
		if declared, known := v.captures[value.Callable]; known && len(value.Captures) != declared {
			v.error(CodeMalformedNode, fmt.Sprintf("Function value binds %d capture(s) for a callable declaring %d",
				len(value.Captures), declared), meta.Span)
		}
		for _, capture := range value.Captures {
			v.validateExpr(capture)
		}
	case *UnaryExpr:
		if value.Op == "" {
			v.error(CodeMalformedNode, "unary expression has no typed operation", meta.Span)
		}
		v.validateExpr(value.Operand)
	case *BinaryExpr:
		if value.Op == "" {
			v.error(CodeMalformedNode, "binary expression has no typed operation", meta.Span)
		}
		v.validateExpr(value.Left)
		v.validateExpr(value.Right)
	case *ConvertExpr:
		v.validateExpr(value.Value)
		widening := value.From.Kind == IntType && value.Type.Kind == RealType
		complexWidening := value.Type.Kind == ComplexType && (value.From.Kind == IntType || value.From.Kind == RealType)
		narrowing := value.From.Kind == RealType && value.Type.Kind == IntType
		stringConversion := value.From.Kind == StringType && (value.Type.Kind == IntType || value.Type.Kind == RealType)
		if !widening && !complexWidening && !narrowing && !stringConversion {
			v.error(CodeInvalidConversion, fmt.Sprintf("invalid numeric conversion %s -> %s", value.From, value.Type), meta.Span)
		}
	case *CallExpr:
		v.requireCallable(value.Callable, meta.Span)
		v.validateExpr(value.Callee)
		v.validateArguments(value.Callable, value.Arguments, meta.Span)
	case *ConstructExpr:
		if !v.classes[value.Class] && !isExternalClass(value.Class) {
			v.error(CodeInvalidClass, fmt.Sprintf("construction references unknown Class %s", value.Class), meta.Span)
		}
		v.requireCallable(value.Constructor, meta.Span)
		v.validateArguments(value.Constructor, value.Arguments, meta.Span)
	case *MemberExpr:
		v.validateExpr(value.Object)
		if value.Kind == FieldMember && (value.Field == "" || (!v.fields[value.Field] && !isExternalID(string(value.Field)))) {
			v.error(CodeUnresolvedIdentity, "member has unresolved FieldID", meta.Span)
		}
		if value.Kind == MethodMember {
			v.requireCallable(value.Callable, meta.Span)
		}
	case *IndexExpr:
		v.validateExpr(value.Object)
		v.validateExpr(value.Index)
	case *SliceExpr:
		v.validateExpr(value.Object)
		v.validateExpr(value.Start)
		v.validateExpr(value.End)
	case *ListExpr:
		for _, element := range value.Elements {
			v.validateExpr(element)
		}
	case *PairExpr:
		for _, entry := range value.Entries {
			v.validateExpr(entry.Key)
			v.validateExpr(entry.Value)
		}
	case *StringExpr:
		for _, part := range value.Parts {
			v.validateExpr(part.ToString)
		}
	case *ToStringExpr:
		v.validateExpr(value.Value)
	case *IdentityExpr:
		v.validateExpr(value.Value)
	case *TypeNameExpr:
		v.validateExpr(value.Value)
	default:
		v.error(CodeMalformedNode, fmt.Sprintf("unknown expression %T", expression), meta.Span)
	}
}

func (v *validator) validateArguments(callable CallableID, arguments []Argument, span source.Span) {
	signature, known := v.callables[callable]
	if known && len(arguments) != len(signature.Parameters) {
		v.error(CodeInvalidCall, fmt.Sprintf("call %s has %d arguments for %d parameters", callable, len(arguments), len(signature.Parameters)), span)
	}
	for index, argument := range arguments {
		if argument.ParameterIndex != index {
			v.error(CodeInvalidCall, "arguments are not in canonical parameter order", span)
		}
		if argument.UsesDefault && argument.Value != nil {
			v.error(CodeInvalidCall, "default argument also carries a value", span)
		}
		if known && index < len(signature.Parameters) {
			parameter := signature.Parameters[index]
			if argument.UsesDefault && !parameter.HasDefault {
				v.error(CodeInvalidCall, fmt.Sprintf("parameter %s has no default", parameter.Name), span)
			}
			if !argument.UsesDefault && argument.Value == nil {
				v.error(CodeInvalidCall, fmt.Sprintf("parameter %s has no argument value", parameter.Name), span)
			}
			if argument.Value != nil && !v.typeAssignable(argument.Value.ExprMeta().Type, parameter.Type) {
				v.error(CodeInvalidCall, fmt.Sprintf("argument for %s has type %s, want %s", parameter.Name, argument.Value.ExprMeta().Type, parameter.Type), span)
			}
		}
		v.validateExpr(argument.Value)
	}
}

func (v *validator) typeAssignable(actual, target Type) bool {
	if EqualType(actual, target) {
		return true
	}
	if actual.Kind == ClassType && target.Kind == ClassType {
		for current := actual.Class; current != ""; current = v.parents[current] {
			if current == target.Class {
				return true
			}
		}
		return false
	}
	if actual.Kind != FunctionType || target.Kind != FunctionType || actual.Signature == nil || target.Signature == nil {
		return false
	}
	if len(actual.Signature.Parameters) != len(target.Signature.Parameters) || !v.typeAssignable(actual.Signature.Return, target.Signature.Return) {
		return false
	}
	for index := range actual.Signature.Parameters {
		left := actual.Signature.Parameters[index].Type
		right := target.Signature.Parameters[index].Type
		if !v.typeAssignable(left, right) || !v.typeAssignable(right, left) {
			return false
		}
	}
	return true
}

func (v *validator) requireType(value Type, span source.Span) {
	if !IsValidType(value) {
		v.error(CodeMissingType, "IR node has no concrete type", span)
		return
	}
	v.requirePairKeys(value, span)
}

// requirePairKeys defensively rejects a Pair type whose key has no v0.1 key
// representation. The frontend already enforces this, so reaching it means the
// IR was built without that check.
func (v *validator) requirePairKeys(value Type, span source.Span) {
	switch value.Kind {
	case PairType:
		if value.Key != nil && !IsPairKeyType(*value.Key) {
			v.error(CodeMalformedNode, fmt.Sprintf("Pair key type %s is not a valid v0.1 key type", value.Key), span)
			return
		}
		if value.Key != nil {
			v.requirePairKeys(*value.Key, span)
		}
		if value.Value != nil {
			v.requirePairKeys(*value.Value, span)
		}
	case ListType:
		if value.Element != nil {
			v.requirePairKeys(*value.Element, span)
		}
	case FunctionType:
		if value.Signature == nil {
			return
		}
		for _, parameter := range value.Signature.Parameters {
			v.requirePairKeys(parameter.Type, span)
		}
		v.requirePairKeys(value.Signature.Return, span)
	}
}
func (v *validator) requireSymbol(id SymbolID, span source.Span) {
	if id == "" || (v.symbols[id].Kind == "" && !isExternalID(string(id))) {
		v.error(CodeUnresolvedIdentity, fmt.Sprintf("unknown SymbolID %s", id), span)
	}
}
func (v *validator) requireCallable(id CallableID, span source.Span) {
	if id == "" || (v.callables[id].Return.Kind == "" && !isExternalID(string(id))) {
		v.error(CodeUnresolvedIdentity, fmt.Sprintf("unknown CallableID %s", id), span)
	}
}
func (v *validator) error(code, message string, span source.Span) {
	v.diagnostics = append(v.diagnostics, diagnostics.Diagnostic{Code: code, Severity: diagnostics.SeverityError, Message: message, Span: span})
}
func isExternalID(id string) bool {
	return strings.HasPrefix(id, "builtin:") || strings.HasPrefix(id, "signature:")
}
func isExternalClass(id ClassID) bool { return strings.HasPrefix(string(id), "builtin:") }
func spanOfGlobal(global *Global) source.Span {
	if global == nil {
		return source.Span{}
	}
	return global.Span
}
