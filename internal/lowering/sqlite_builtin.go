package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// SQLiteModuleID is the synthetic module that carries the SQLite standard
// library's Class declarations into the IR.
const SQLiteModuleID = "builtin:SQLite"

const (
	sqliteDatabaseClassID = ir.ClassID(SQLiteModuleID + "::class::Database")
	sqliteValueClassID    = ir.ClassID(SQLiteModuleID + "::class::SQLiteValue")
	sqliteErrorClassID    = ir.ClassID(SQLiteModuleID + "::class::SQLiteError")
)

// A Database stores one hidden String field: the handle of its logical SQLite
// connection inside the bundled ahdsqlite helper. Assigning a Database to a
// second variable copies the reference to the same instance, so both names
// observe the same connection and the same closed state; no second connection
// is ever opened by assignment.
//
// A SQLiteValue stores one hidden String field: its canonical text encoding
// (one kind byte followed by the payload), the same "hidden String field,
// decoded by helpers" pattern JSONValue uses.
var (
	SQLiteDatabaseHandleFieldID = ir.FieldID(string(sqliteDatabaseClassID) + "::field::handle")
	SQLiteValueDataFieldID      = ir.FieldID(string(sqliteValueClassID) + "::field::data")
)

func sqliteModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	specs := []struct {
		id         ir.ClassID
		name       string
		field      ir.FieldID
		fieldName  string
		operations []string
	}{
		{sqliteDatabaseClassID, "Database", SQLiteDatabaseHandleFieldID, "handle", semantic.SQLiteDatabaseOperations},
		{sqliteValueClassID, "SQLiteValue", SQLiteValueDataFieldID, "data", semantic.SQLiteValueOperations},
	}
	for _, spec := range specs {
		field := ir.Field{ID: spec.field, Name: spec.fieldName, Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true}
		class := &ir.Class{
			ID: spec.id, Symbol: ir.SymbolID(string(spec.id) + "::symbol"), Name: spec.name,
			Operations: spec.operations, Fields: []ir.Field{field},
			Constructor: ir.CallableID(string(spec.id) + "::constructor::(" + spec.fieldName + ":String)->Nothing"),
		}
		module.Classes = append(module.Classes, class)
		module.Functions = append(module.Functions, sqliteValueConstructor(class))
	}

	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: sqliteErrorClassID, Symbol: ir.SymbolID(string(sqliteErrorClassID) + "::symbol"),
		Name: "SQLiteError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(sqliteErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}

// sqliteValueConstructor builds the one-field constructor the backend uses to
// materialize a Database or SQLiteValue. AhdCode source cannot reach it: the
// frontend publishes both Classes without a constructor, so values come only
// from SQLite module functions and Database operations.
func sqliteValueConstructor(class *ir.Class) *ir.Function {
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
