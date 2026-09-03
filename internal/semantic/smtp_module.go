package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const smtpModuleID = "builtin:SMTP"

var (
	smtpErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	smtpErrorClass   = &types.ClassSymbol{ModuleID: smtpModuleID, Name: "SMTPError", Parent: smtpErrorParent}
	smtpClientClass  = &types.ClassSymbol{ModuleID: smtpModuleID, Name: "SMTPClient"}
	smtpMessageClass = &types.ClassSymbol{ModuleID: smtpModuleID, Name: "SMTPMessage"}
)

// SMTPErrorIdentity, SMTPClientIdentity, and SMTPMessageIdentity expose the
// canonical identities to lowering without coupling the public module
// interface to a backend.
func SMTPErrorIdentity() *types.ClassSymbol   { return smtpErrorClass }
func SMTPClientIdentity() *types.ClassSymbol  { return smtpClientClass }
func SMTPMessageIdentity() *types.ClassSymbol { return smtpMessageClass }

// SMTPClientOperations and SMTPMessageOperations name the members each Class
// publishes through built-in type operations, so has/has not reports the real
// surface and the IR Class agrees with the frontend.
var SMTPClientOperations = []string{"withPlainAuth", "send"}
var SMTPMessageOperations = []string{"withCc", "withBcc", "withReplyTo", "withText", "withHtml"}

func smtpClientType() types.Type  { return types.Class{Symbol: smtpClientClass} }
func smtpMessageType() types.Type { return types.Class{Symbol: smtpMessageClass} }

func smtpModuleInterface() *ModuleInterface {
	module := standardInterface(smtpModuleID, "SMTP")
	classes := []struct {
		name     string
		identity *types.ClassSymbol
	}{
		{"SMTPError", smtpErrorClass}, {"SMTPClient", smtpClientClass},
		{"SMTPMessage", smtpMessageClass},
	}
	for _, entry := range classes {
		symbol := &Symbol{
			Name: entry.name, Kind: ClassSymbol, Class: entry.identity,
			Type: types.Class{Symbol: entry.identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: smtpModuleID,
			Members: make(map[string]*Symbol),
		}
		if entry.name == "SMTPError" {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[smtpModuleID+"\x00"+entry.name] = symbol
		addStandardExport(module, symbol)
	}
	addresses := types.List{Element: types.String}
	addStandardExport(module, standardFunction(smtpModuleID, "client", smtpClientType(),
		types.Parameter{Name: "host", Type: types.String},
		types.Parameter{Name: "port", Type: types.Int},
		types.Parameter{Name: "security", Type: types.String, HasDefault: true},
		types.Parameter{Name: "timeoutSeconds", Type: types.Int, HasDefault: true}))
	addStandardExport(module, standardFunction(smtpModuleID, "message", smtpMessageType(),
		types.Parameter{Name: "from", Type: types.String},
		types.Parameter{Name: "to", Type: addresses},
		types.Parameter{Name: "subject", Type: types.String}))
	sort.Strings(module.ExportNames)
	return module
}

func smtpConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != smtpModuleID {
		return "", false
	}
	switch identity.Name {
	case "SMTPClient":
		return "create an SMTPClient with SMTP.client(host, port)", true
	case "SMTPMessage":
		return "create an SMTPMessage with SMTP.message(from, to, subject)", true
	}
	return "", false
}

type smtpOperationShape struct {
	parameters []types.Type
	result     types.Type
	hint       string
}

func smtpOperationShapes() map[TypeOperation]smtpOperationShape {
	addresses := types.List{Element: types.String}
	return map[TypeOperation]smtpOperationShape{
		SMTPClientWithPlainAuth: {[]types.Type{types.String, types.String}, smtpClientType(), "pass a username String and a password String"},
		SMTPClientSend:          {[]types.Type{smtpMessageType()}, types.Nothing, "pass one SMTPMessage"},
		SMTPMessageWithCc:       {[]types.Type{addresses}, smtpMessageType(), "pass a List of recipient Strings"},
		SMTPMessageWithBcc:      {[]types.Type{addresses}, smtpMessageType(), "pass a List of recipient Strings"},
		SMTPMessageWithReplyTo:  {[]types.Type{types.String}, smtpMessageType(), "pass one mailbox String"},
		SMTPMessageWithText:     {[]types.Type{types.String}, smtpMessageType(), "pass one String text body"},
		SMTPMessageWithHtml:     {[]types.Type{types.String}, smtpMessageType(), "pass one String HTML body"},
	}
}

var smtpOperationNames = map[string]map[string]TypeOperation{
	"SMTPClient": {
		"withPlainAuth": SMTPClientWithPlainAuth, "send": SMTPClientSend,
	},
	"SMTPMessage": {
		"withCc": SMTPMessageWithCc, "withBcc": SMTPMessageWithBcc,
		"withReplyTo": SMTPMessageWithReplyTo, "withText": SMTPMessageWithText,
		"withHtml": SMTPMessageWithHtml,
	},
}

func smtpOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != smtpModuleID {
		return "", false
	}
	operation, known := smtpOperationNames[class.Symbol.Name][name]
	return operation, known
}

func (a *analyzer) analyzeSMTPOperation(call *ast.CallExpr, operation TypeOperation, shape smtpOperationShape, current *scope, flow flowState) expressionInfo {
	result := expressionInfo{typeValue: shape.result, nullState: NonNull}
	if len(call.Arguments) != len(shape.parameters) {
		a.error(codeCallArguments, fmt.Sprintf("%s expects %d argument(s); received %d", operation, len(shape.parameters), len(call.Arguments)), call.Span(), shape.hint)
		a.analyzeTypeOperationArguments(call, current, flow, nil)
		return result
	}
	for index, argument := range call.Arguments {
		expected := shape.parameters[index]
		info := a.analyzeExpressionExpected(argument.Value, current, flow, expected)
		if info.invalid() {
			continue
		}
		if info.nullState != NonNull {
			a.nullableError(string(operation), argument.Value, info.nullState)
			continue
		}
		if !types.Assignable(expected, info.typeValue) {
			a.typeMismatch(argument.Span(), expected, info.typeValue, string(operation)+" argument")
		}
	}
	parameters := make([]types.Parameter, len(shape.parameters))
	for index, expected := range shape.parameters {
		parameters[index] = types.Parameter{Type: expected}
	}
	a.result.SelectedCallables[call] = &Callable{
		Signature:  &types.Signature{Parameters: parameters, Return: shape.result},
		ReturnNull: NonNull,
	}
	return result
}
