package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// MySQLModuleID is the synthetic module that carries the MySQL standard
// library's Class declarations into the IR.
const MySQLModuleID = "builtin:MySQL"

const (
	mysqlDatabaseClassID    = ir.ClassID(MySQLModuleID + "::class::MySQLDatabase")
	mysqlTransactionClassID = ir.ClassID(MySQLModuleID + "::class::MySQLTransaction")
	mysqlResultClassID      = ir.ClassID(MySQLModuleID + "::class::MySQLResult")
	mysqlValueClassID       = ir.ClassID(MySQLModuleID + "::class::MySQLValue")
	mysqlErrorClassID       = ir.ClassID(MySQLModuleID + "::class::MySQLError")
)

// Every MySQL value is one hidden String field: MySQLDatabase and
// MySQLTransaction store their runtime handle, MySQLResult and MySQLValue
// store their canonical text encoding -- the same "hidden String field,
// decoded by helpers" pattern SQLite and HTTP already use.
var (
	MySQLDatabaseHandleFieldID    = ir.FieldID(string(mysqlDatabaseClassID) + "::field::handle")
	MySQLTransactionHandleFieldID = ir.FieldID(string(mysqlTransactionClassID) + "::field::handle")
	MySQLResultDataFieldID        = ir.FieldID(string(mysqlResultClassID) + "::field::data")
	MySQLValueDataFieldID         = ir.FieldID(string(mysqlValueClassID) + "::field::data")
)

func mysqlModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	specs := []struct {
		id         ir.ClassID
		name       string
		field      ir.FieldID
		fieldName  string
		operations []string
	}{
		{mysqlDatabaseClassID, "MySQLDatabase", MySQLDatabaseHandleFieldID, "handle", semantic.MySQLDatabaseOperations},
		{mysqlTransactionClassID, "MySQLTransaction", MySQLTransactionHandleFieldID, "handle", semantic.MySQLTransactionOperations},
		{mysqlResultClassID, "MySQLResult", MySQLResultDataFieldID, "data", semantic.MySQLResultOperations},
		{mysqlValueClassID, "MySQLValue", MySQLValueDataFieldID, "data", semantic.MySQLValueOperations},
	}
	for _, spec := range specs {
		field := ir.Field{ID: spec.field, Name: spec.fieldName, Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true}
		class := &ir.Class{
			ID: spec.id, Symbol: ir.SymbolID(string(spec.id) + "::symbol"), Name: spec.name,
			Operations: spec.operations, Fields: []ir.Field{field},
			Constructor: ir.CallableID(string(spec.id) + "::constructor::(" + spec.fieldName + ":String)->Nothing"),
		}
		module.Classes = append(module.Classes, class)
		module.Functions = append(module.Functions, mysqlValueConstructor(class))
	}

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: mysqlErrorClassID, Symbol: ir.SymbolID(string(mysqlErrorClassID) + "::symbol"),
		Name: "MySQLError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(mysqlErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}

// mysqlValueConstructor builds the one-field constructor the backend uses to
// materialize a MySQLDatabase, MySQLTransaction, MySQLResult, or MySQLValue.
// AhdCode source cannot reach it: the frontend publishes every one of these
// Classes without a constructor, so values come only from MySQL module
// functions and MySQLDatabase/MySQLTransaction/MySQLResult operations.
func mysqlValueConstructor(class *ir.Class) *ir.Function {
	id := class.Constructor
	receiver := ir.SymbolID(string(id) + "::receiver")
	field := class.Fields[0]
	parameter := ir.Parameter{
		ID: ir.SymbolID(string(id) + "::parameter::" + field.Name), Name: field.Name,
		Type: field.Type, NullState: ir.NonNull,
	}
	return &ir.Function{
		ID: id, Symbol: class.Symbol, Name: class.Name, Kind: ir.ConstructorFunction,
		Owner: class.ID, Receiver: receiver,
		Signature:  ir.Signature{Parameters: []ir.ParameterType{{Name: field.Name, Type: field.Type}}, Return: ir.Type{Kind: ir.NothingType}},
		Parameters: []ir.Parameter{parameter}, ReturnNull: ir.NonNull,
		Body: ir.Block{Statements: []ir.Statement{&ir.AssignStmt{
			Target: ir.Target{Kind: ir.FieldTarget, Type: field.Type, Field: field.ID,
				Receiver: &ir.LoadExpr{ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.ClassType, Class: class.ID}, NullState: ir.NonNull}, Symbol: receiver}},
			Value: &ir.LoadExpr{ExprBase: ir.ExprBase{Type: field.Type, NullState: ir.NonNull}, Symbol: parameter.ID},
		}}},
	}
}
