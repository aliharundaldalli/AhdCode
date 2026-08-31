package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

const NumericModuleID = "builtin:Numeric"
const (
	numericVectorClassID = ir.ClassID(NumericModuleID + "::class::Vector")
	numericMatrixClassID = ir.ClassID(NumericModuleID + "::class::Matrix")
	numericErrorClassID  = ir.ClassID(NumericModuleID + "::class::NumericError")
)

var NumericVectorValuesFieldID = ir.FieldID(string(numericVectorClassID) + "::field::values")
var NumericMatrixRowsFieldID = ir.FieldID(string(numericMatrixClassID) + "::field::rows")

func numericRealList() ir.Type {
	e := ir.Type{Kind: ir.RealType}
	return ir.Type{Kind: ir.ListType, Element: &e}
}
func numericGrid() ir.Type { e := numericRealList(); return ir.Type{Kind: ir.ListType, Element: &e} }

func numericModule(id ir.ModuleID, name, path string) *ir.Module {
	m := &ir.Module{ID: id, Name: name, SourcePath: path}
	v := &ir.Class{ID: numericVectorClassID, Symbol: ir.SymbolID(string(numericVectorClassID) + "::symbol"), Name: "Vector", Operations: semantic.NumericVectorOperations, Fields: []ir.Field{{ID: NumericVectorValuesFieldID, Name: "values", Type: numericRealList(), NullState: ir.NonNull, Hidden: true}}}
	v.Constructor = numericConstructorID(v)
	m.Classes = append(m.Classes, v)
	m.Functions = append(m.Functions, numericConstructor(v))
	matrix := &ir.Class{ID: numericMatrixClassID, Symbol: ir.SymbolID(string(numericMatrixClassID) + "::symbol"), Name: "Matrix", Operations: semantic.NumericMatrixOperations, Fields: []ir.Field{{ID: NumericMatrixRowsFieldID, Name: "rows", Type: numericGrid(), NullState: ir.NonNull, Hidden: true}}}
	matrix.Constructor = numericConstructorID(matrix)
	m.Classes = append(m.Classes, matrix)
	m.Functions = append(m.Functions, numericConstructor(matrix))
	parentID := builtinErrorClass
	errClass := &ir.Class{ID: numericErrorClassID, Symbol: ir.SymbolID(string(numericErrorClassID) + "::symbol"), Name: "NumericError", Parent: parentID, Builtin: true, Constructor: builtinConstructorID(numericErrorClassID)}
	m.Classes = append(m.Classes, errClass)
	m.Functions = append(m.Functions, builtinConstructor(errClass, &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}))
	return m
}

func numericConstructorID(class *ir.Class) ir.CallableID {
	f := class.Fields[0]
	return ir.CallableID(string(class.ID) + "::constructor::(" + f.Name + ":" + f.Type.String() + ")->Nothing")
}
func numericConstructor(class *ir.Class) *ir.Function {
	f := class.Fields[0]
	id := class.Constructor
	receiver := ir.SymbolID(string(id) + "::receiver")
	p := ir.Parameter{ID: ir.SymbolID(string(id) + "::parameter::" + f.Name), Name: f.Name, Type: f.Type, NullState: ir.NonNull}
	return &ir.Function{ID: id, Symbol: class.Symbol, Name: class.Name, Kind: ir.ConstructorFunction, Owner: class.ID, Receiver: receiver, Signature: ir.Signature{Parameters: []ir.ParameterType{{Name: f.Name, Type: f.Type}}, Return: ir.Type{Kind: ir.NothingType}}, ReturnNull: ir.NonNull, Parameters: []ir.Parameter{p}, Body: ir.Block{Statements: []ir.Statement{&ir.AssignStmt{Target: ir.Target{Kind: ir.FieldTarget, Type: f.Type, Field: f.ID, Receiver: &ir.LoadExpr{ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.ClassType, Class: class.ID}, NullState: ir.NonNull}, Symbol: receiver}}, Value: &ir.LoadExpr{ExprBase: ir.ExprBase{Type: f.Type, NullState: ir.NonNull}, Symbol: p.ID}}}}}
}
