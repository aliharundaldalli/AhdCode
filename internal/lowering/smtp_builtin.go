package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

const SMTPModuleID = "builtin:SMTP"

const (
	smtpClientClassID  = ir.ClassID(SMTPModuleID + "::class::SMTPClient")
	smtpMessageClassID = ir.ClassID(SMTPModuleID + "::class::SMTPMessage")
	smtpErrorClassID   = ir.ClassID(SMTPModuleID + "::class::SMTPError")
)

var (
	SMTPClientHandleFieldID = ir.FieldID(string(smtpClientClassID) + "::field::handle")
	SMTPMessageDataFieldID  = ir.FieldID(string(smtpMessageClassID) + "::field::data")
)

func smtpModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	specs := []struct {
		id         ir.ClassID
		name       string
		field      ir.FieldID
		fieldName  string
		operations []string
	}{
		{smtpClientClassID, "SMTPClient", SMTPClientHandleFieldID, "handle", semantic.SMTPClientOperations},
		{smtpMessageClassID, "SMTPMessage", SMTPMessageDataFieldID, "data", semantic.SMTPMessageOperations},
	}
	for _, spec := range specs {
		field := ir.Field{ID: spec.field, Name: spec.fieldName, Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true}
		class := &ir.Class{
			ID: spec.id, Symbol: ir.SymbolID(string(spec.id) + "::symbol"), Name: spec.name,
			Operations: spec.operations, Fields: []ir.Field{field},
			Constructor: ir.CallableID(string(spec.id) + "::constructor::(" + spec.fieldName + ":String)->Nothing"),
		}
		module.Classes = append(module.Classes, class)
		module.Functions = append(module.Functions, smtpValueConstructor(class))
	}
	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: smtpErrorClassID, Symbol: ir.SymbolID(string(smtpErrorClassID) + "::symbol"),
		Name: "SMTPError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(smtpErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}

func smtpValueConstructor(class *ir.Class) *ir.Function {
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
		Signature: ir.Signature{
			Parameters: []ir.ParameterType{{Name: field.Name, Type: field.Type}},
			Return:     ir.Type{Kind: ir.NothingType},
		},
		Parameters: []ir.Parameter{parameter}, ReturnNull: ir.NonNull,
		Body: ir.Block{Statements: []ir.Statement{&ir.AssignStmt{
			Target: ir.Target{
				Kind: ir.FieldTarget, Type: field.Type, Field: field.ID,
				Receiver: &ir.LoadExpr{
					ExprBase: ir.ExprBase{Type: ir.Type{Kind: ir.ClassType, Class: class.ID}, NullState: ir.NonNull},
					Symbol:   receiver,
				},
			},
			Value: &ir.LoadExpr{ExprBase: ir.ExprBase{Type: field.Type, NullState: ir.NonNull}, Symbol: parameter.ID},
		}}},
	}
}
