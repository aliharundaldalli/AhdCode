package lowering

import "ahdcode/internal/ir"

// RegexModuleID is the synthetic module that carries the Regex standard
// library's Class declarations into the IR.
const RegexModuleID = "builtin:Regex"

const regexClassID = ir.ClassID(RegexModuleID + "::class::Pattern")
const regexErrorClassID = ir.ClassID(RegexModuleID + "::class::RegexError")

// RegexPatternFieldID is the one field a Pattern instance carries: its source
// pattern text. Every Pattern operation recompiles (and the runtime caches)
// the pattern from this field rather than storing an opaque compiled value,
// so a Pattern instance stays representable by AhdCode's ordinary Class field
// model.
var RegexPatternFieldID = ir.FieldID(string(regexClassID) + "::field::pattern")

var regexOperations = []string{"matches", "find", "findAll", "groups", "replace", "split"}

// regexModule emits the Pattern Class (one Constant String field) and the
// RegexError Class as ordinary IR classes, mirroring the File/Latex error
// pattern and the Time Class-with-fields pattern.
func regexModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}

	regexClass := &ir.Class{
		ID: regexClassID, Symbol: ir.SymbolID(string(regexClassID) + "::symbol"),
		Name: "Pattern", Operations: regexOperations,
		Fields: []ir.Field{{
			ID: RegexPatternFieldID, Name: "pattern",
			Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull,
		}},
		Constructor: regexConstructorID(),
	}
	module.Classes = append(module.Classes, regexClass)
	module.Functions = append(module.Functions, regexConstructor(regexClass))

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: regexErrorClassID, Symbol: ir.SymbolID(string(regexErrorClassID) + "::symbol"),
		Name: "RegexError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(regexErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))

	return module
}

func regexConstructorID() ir.CallableID {
	return ir.CallableID(string(regexClassID) + "::constructor::(pattern:String)->Nothing")
}

// regexConstructor builds the constructor the backend uses to materialize one
// Regex value. AhdCode source cannot reach it directly: values come only from
// Regex.compile, which validates the pattern first.
func regexConstructor(class *ir.Class) *ir.Function {
	id := class.Constructor
	receiver := ir.SymbolID(string(id) + "::receiver")
	parameter := ir.SymbolID(string(id) + "::parameter::pattern")
	function := &ir.Function{
		ID: id, Symbol: class.Symbol, Name: class.Name, Kind: ir.ConstructorFunction,
		Owner: class.ID, Receiver: receiver,
		Signature: ir.Signature{
			Parameters: []ir.ParameterType{{Name: "pattern", Type: ir.Type{Kind: ir.StringType}}},
			Return:     ir.Type{Kind: ir.NothingType},
		},
		Parameters: []ir.Parameter{{ID: parameter, Name: "pattern", Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull}},
		ReturnNull: ir.NonNull,
	}
	function.Body = ir.Block{Statements: []ir.Statement{&ir.AssignStmt{
		Target: ir.Target{
			Kind: ir.FieldTarget, Type: ir.Type{Kind: ir.StringType}, Field: RegexPatternFieldID,
			Receiver: &ir.LoadExpr{
				ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.ClassType, Class: class.ID}, NullState: ir.NonNull},
				Symbol:   receiver,
			},
		},
		Value: &ir.LoadExpr{ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull}, Symbol: parameter},
	}}}
	return function
}
