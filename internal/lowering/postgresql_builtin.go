package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

// PostgreSQLModuleID is the synthetic module that carries the PostgreSQL
// standard library's Class declarations into the IR.
const PostgreSQLModuleID = "builtin:PostgreSQL"

const (
	postgresqlDatabaseClassID    = ir.ClassID(PostgreSQLModuleID + "::class::PostgreSQLDatabase")
	postgresqlTransactionClassID = ir.ClassID(PostgreSQLModuleID + "::class::PostgreSQLTransaction")
	postgresqlResultClassID      = ir.ClassID(PostgreSQLModuleID + "::class::PostgreSQLResult")
	postgresqlValueClassID       = ir.ClassID(PostgreSQLModuleID + "::class::PostgreSQLValue")
	postgresqlErrorClassID       = ir.ClassID(PostgreSQLModuleID + "::class::PostgreSQLError")
)

// Every PostgreSQL value is one hidden String field, exactly as in MySQL: the
// database and transaction store their runtime handle, the result and value
// their canonical text encoding.
var (
	PostgreSQLDatabaseHandleFieldID    = ir.FieldID(string(postgresqlDatabaseClassID) + "::field::handle")
	PostgreSQLTransactionHandleFieldID = ir.FieldID(string(postgresqlTransactionClassID) + "::field::handle")
	PostgreSQLResultDataFieldID        = ir.FieldID(string(postgresqlResultClassID) + "::field::data")
	PostgreSQLValueDataFieldID         = ir.FieldID(string(postgresqlValueClassID) + "::field::data")
)

func postgresqlModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	specs := []struct {
		id         ir.ClassID
		name       string
		field      ir.FieldID
		fieldName  string
		operations []string
	}{
		{postgresqlDatabaseClassID, "PostgreSQLDatabase", PostgreSQLDatabaseHandleFieldID, "handle", semantic.PostgreSQLDatabaseOperations},
		{postgresqlTransactionClassID, "PostgreSQLTransaction", PostgreSQLTransactionHandleFieldID, "handle", semantic.PostgreSQLTransactionOperations},
		{postgresqlResultClassID, "PostgreSQLResult", PostgreSQLResultDataFieldID, "data", semantic.PostgreSQLResultOperations},
		{postgresqlValueClassID, "PostgreSQLValue", PostgreSQLValueDataFieldID, "data", semantic.PostgreSQLValueOperations},
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
		ID: postgresqlErrorClassID, Symbol: ir.SymbolID(string(postgresqlErrorClassID) + "::symbol"),
		Name: "PostgreSQLError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(postgresqlErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}
