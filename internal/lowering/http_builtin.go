package lowering

import (
	"ahdcode/internal/ir"
	"ahdcode/internal/semantic"
)

const HTTPModuleID = "builtin:HTTP"

const (
	httpServerClassID         = ir.ClassID(HTTPModuleID + "::class::Server")
	httpRequestClassID        = ir.ClassID(HTTPModuleID + "::class::Request")
	httpResponseClassID       = ir.ClassID(HTTPModuleID + "::class::Response")
	httpCookieClassID         = ir.ClassID(HTTPModuleID + "::class::Cookie")
	httpSessionStoreClassID   = ir.ClassID(HTTPModuleID + "::class::SessionStore")
	httpSessionClassID        = ir.ClassID(HTTPModuleID + "::class::Session")
	httpClientClassID         = ir.ClassID(HTTPModuleID + "::class::Client")
	httpClientRequestClassID  = ir.ClassID(HTTPModuleID + "::class::ClientRequest")
	httpClientResponseClassID = ir.ClassID(HTTPModuleID + "::class::ClientResponse")
	httpUploadedFileClassID   = ir.ClassID(HTTPModuleID + "::class::UploadedFile")
	httpErrorClassID          = ir.ClassID(HTTPModuleID + "::class::HTTPError")
)

var (
	HTTPServerHandleFieldID       = ir.FieldID(string(httpServerClassID) + "::field::handle")
	HTTPRequestDataFieldID        = ir.FieldID(string(httpRequestClassID) + "::field::data")
	HTTPResponseDataFieldID       = ir.FieldID(string(httpResponseClassID) + "::field::data")
	HTTPCookieDataFieldID         = ir.FieldID(string(httpCookieClassID) + "::field::data")
	HTTPSessionStoreHandleFieldID = ir.FieldID(string(httpSessionStoreClassID) + "::field::handle")
	HTTPSessionDataFieldID        = ir.FieldID(string(httpSessionClassID) + "::field::data")
	HTTPClientHandleFieldID       = ir.FieldID(string(httpClientClassID) + "::field::handle")
	HTTPClientRequestDataFieldID  = ir.FieldID(string(httpClientRequestClassID) + "::field::data")
	HTTPClientResponseDataFieldID = ir.FieldID(string(httpClientResponseClassID) + "::field::data")
	HTTPUploadedFileDataFieldID   = ir.FieldID(string(httpUploadedFileClassID) + "::field::data")
)

func httpModule(id ir.ModuleID, name, path string) *ir.Module {
	module := &ir.Module{ID: id, Name: name, SourcePath: path}
	specs := []struct {
		id         ir.ClassID
		name       string
		field      ir.FieldID
		fieldName  string
		operations []string
	}{
		{httpServerClassID, "Server", HTTPServerHandleFieldID, "handle", semantic.HTTPServerOperations},
		{httpRequestClassID, "Request", HTTPRequestDataFieldID, "data", semantic.HTTPRequestOperations},
		{httpResponseClassID, "Response", HTTPResponseDataFieldID, "data", semantic.HTTPResponseOperations},
		{httpCookieClassID, "Cookie", HTTPCookieDataFieldID, "data", semantic.HTTPCookieOperations},
		{httpSessionStoreClassID, "SessionStore", HTTPSessionStoreHandleFieldID, "handle", semantic.HTTPSessionStoreOperations},
		{httpSessionClassID, "Session", HTTPSessionDataFieldID, "data", semantic.HTTPSessionOperations},
		{httpClientClassID, "Client", HTTPClientHandleFieldID, "handle", semantic.HTTPClientOperations},
		{httpClientRequestClassID, "ClientRequest", HTTPClientRequestDataFieldID, "data", semantic.HTTPClientRequestOperations},
		{httpClientResponseClassID, "ClientResponse", HTTPClientResponseDataFieldID, "data", semantic.HTTPClientResponseOperations},
		{httpUploadedFileClassID, "UploadedFile", HTTPUploadedFileDataFieldID, "data", semantic.HTTPUploadedFileOperations},
	}
	for _, spec := range specs {
		field := ir.Field{ID: spec.field, Name: spec.fieldName, Type: ir.Type{Kind: ir.StringType}, NullState: ir.NonNull, Hidden: true}
		class := &ir.Class{
			ID: spec.id, Symbol: ir.SymbolID(string(spec.id) + "::symbol"), Name: spec.name,
			Operations: spec.operations, Fields: []ir.Field{field},
			Constructor: ir.CallableID(string(spec.id) + "::constructor::(" + spec.fieldName + ":String)->Nothing"),
		}
		module.Classes = append(module.Classes, class)
		module.Functions = append(module.Functions, httpValueConstructor(class))
	}
	parentID := ir.ClassID("builtin:core::class::Error")
	errorClass := &ir.Class{
		ID: httpErrorClassID, Symbol: ir.SymbolID(string(httpErrorClassID) + "::symbol"),
		Name: "HTTPError", Parent: parentID, Builtin: true,
		Constructor: builtinConstructorID(httpErrorClassID),
	}
	parent := &ir.Class{ID: parentID, Constructor: builtinConstructorID(parentID)}
	module.Classes = append(module.Classes, errorClass)
	module.Functions = append(module.Functions, builtinConstructor(errorClass, parent))
	return module
}

func httpValueConstructor(class *ir.Class) *ir.Function {
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
