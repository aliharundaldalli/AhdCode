package semantic

import (
	"fmt"
	"sort"

	"ahdcode/internal/syntax/ast"
	"ahdcode/internal/types"
)

const httpModuleID = "builtin:HTTP"

var (
	httpErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	httpErrorClass    = &types.ClassSymbol{ModuleID: httpModuleID, Name: "HTTPError", Parent: httpErrorParent}
	httpServerClass   = &types.ClassSymbol{ModuleID: httpModuleID, Name: "Server"}
	httpRequestClass  = &types.ClassSymbol{ModuleID: httpModuleID, Name: "Request"}
	httpResponseClass = &types.ClassSymbol{ModuleID: httpModuleID, Name: "Response"}
)

// HTTPErrorIdentity, HTTPServerIdentity, HTTPRequestIdentity, and
// HTTPResponseIdentity expose the canonical identities to lowering without
// coupling the public module interface to a backend.
func HTTPErrorIdentity() *types.ClassSymbol    { return httpErrorClass }
func HTTPServerIdentity() *types.ClassSymbol   { return httpServerClass }
func HTTPRequestIdentity() *types.ClassSymbol  { return httpRequestClass }
func HTTPResponseIdentity() *types.ClassSymbol { return httpResponseClass }

// HTTPServerOperations, HTTPRequestOperations, and HTTPResponseOperations
// name the members each Class publishes through built-in type operations, so
// has/has not reports the real surface and the IR Class agrees with the
// frontend.
var HTTPServerOperations = []string{"get", "post", "route", "start"}
var HTTPRequestOperations = []string{
	"method", "path", "query", "queryAll", "header", "headerAll", "body", "form", "formAll",
}
var HTTPResponseOperations = []string{"withHeader"}

func httpServerType() types.Type   { return types.Class{Symbol: httpServerClass} }
func httpRequestType() types.Type  { return types.Class{Symbol: httpRequestClass} }
func httpResponseType() types.Type { return types.Class{Symbol: httpResponseClass} }

func httpHandlerType() types.Type {
	return types.Function{Signature: &types.Signature{
		Parameters: []types.Parameter{{Name: "request", Type: httpRequestType()}},
		Return:     httpResponseType(),
	}}
}

func httpModuleInterface() *ModuleInterface {
	module := standardInterface(httpModuleID, "HTTP")
	classes := []struct {
		name     string
		identity *types.ClassSymbol
	}{
		{"HTTPError", httpErrorClass}, {"Server", httpServerClass},
		{"Request", httpRequestClass}, {"Response", httpResponseClass},
	}
	for _, entry := range classes {
		symbol := &Symbol{
			Name: entry.name, Kind: ClassSymbol, Class: entry.identity,
			Type: types.Class{Symbol: entry.identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: httpModuleID,
			Members: make(map[string]*Symbol),
		}
		if entry.name == "HTTPError" {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[httpModuleID+"\x00"+entry.name] = symbol
		addStandardExport(module, symbol)
	}
	host := types.Parameter{Name: "host", Type: types.String}
	port := types.Parameter{Name: "port", Type: types.Int}
	maxBody := types.Parameter{Name: "maxBodyBytes", Type: types.Int, HasDefault: true}
	body := types.Parameter{Name: "body", Type: types.String}
	status := types.Parameter{Name: "status", Type: types.Int, HasDefault: true}
	contentType := types.Parameter{Name: "contentType", Type: types.String}
	location := types.Parameter{Name: "location", Type: types.String}
	addStandardExport(module, standardFunction(httpModuleID, "server", httpServerType(), host, port, maxBody))
	addStandardExport(module, standardFunction(httpModuleID, "response", httpResponseType(),
		types.Parameter{Name: "status", Type: types.Int}, body, contentType))
	addStandardExport(module, standardFunction(httpModuleID, "text", httpResponseType(), body, status))
	addStandardExport(module, standardFunction(httpModuleID, "html", httpResponseType(), body, status))
	addStandardExport(module, standardFunction(httpModuleID, "redirect", httpResponseType(), location, status))
	sort.Strings(module.ExportNames)
	return module
}

func httpConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != httpModuleID {
		return "", false
	}
	switch identity.Name {
	case "Server":
		return "create a Server with HTTP.server(host, port)", true
	case "Request":
		return "Request values are produced by the HTTP server for each incoming request", true
	case "Response":
		return "create a Response with HTTP.text, HTTP.html, HTTP.response, or HTTP.redirect", true
	}
	return "", false
}

type httpOperationShape struct {
	parameters     []types.Type
	result         types.Type
	resultNullable bool
	hint           string
}

func httpOperationShapes() map[TypeOperation]httpOperationShape {
	none := []types.Type{}
	handler := httpHandlerType()
	strings := types.List{Element: types.String}
	return map[TypeOperation]httpOperationShape{
		HTTPServerGet:   {[]types.Type{types.String, handler}, types.Nothing, false, "pass a path String and a (request: Request) -> Response Function"},
		HTTPServerPost:  {[]types.Type{types.String, handler}, types.Nothing, false, "pass a path String and a (request: Request) -> Response Function"},
		HTTPServerRoute: {[]types.Type{types.String, types.String, handler}, types.Nothing, false, "pass a method String, a path String, and a (request: Request) -> Response Function"},
		HTTPServerStart: {none, types.Nothing, false, "call start with no argument"},

		HTTPRequestMethod:    {none, types.String, false, "call method with no argument"},
		HTTPRequestPath:      {none, types.String, false, "call path with no argument"},
		HTTPRequestQuery:     {[]types.Type{types.String}, types.String, true, "pass one String query name"},
		HTTPRequestQueryAll:  {[]types.Type{types.String}, strings, false, "pass one String query name"},
		HTTPRequestHeader:    {[]types.Type{types.String}, types.String, true, "pass one String header name"},
		HTTPRequestHeaderAll: {[]types.Type{types.String}, strings, false, "pass one String header name"},
		HTTPRequestBody:      {none, types.String, false, "call body with no argument"},
		HTTPRequestForm:      {[]types.Type{types.String}, types.String, true, "pass one String form field name"},
		HTTPRequestFormAll:   {[]types.Type{types.String}, strings, false, "pass one String form field name"},

		HTTPResponseWithHeader: {[]types.Type{types.String, types.String}, httpResponseType(), false, "pass a header name String and a header value String"},
	}
}

var httpOperationNames = map[string]map[string]TypeOperation{
	"Server": {
		"get": HTTPServerGet, "post": HTTPServerPost, "route": HTTPServerRoute, "start": HTTPServerStart,
	},
	"Request": {
		"method": HTTPRequestMethod, "path": HTTPRequestPath,
		"query": HTTPRequestQuery, "queryAll": HTTPRequestQueryAll,
		"header": HTTPRequestHeader, "headerAll": HTTPRequestHeaderAll,
		"body": HTTPRequestBody, "form": HTTPRequestForm, "formAll": HTTPRequestFormAll,
	},
	"Response": {"withHeader": HTTPResponseWithHeader},
}

func httpOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != httpModuleID {
		return "", false
	}
	operation, known := httpOperationNames[class.Symbol.Name][name]
	return operation, known
}

func (a *analyzer) analyzeHTTPOperation(call *ast.CallExpr, operation TypeOperation, shape httpOperationShape, current *scope, flow flowState) expressionInfo {
	nullState := NonNull
	if shape.resultNullable {
		nullState = MaybeNull
	}
	result := expressionInfo{typeValue: shape.result, nullState: nullState}
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
		if expectedFunction, isFunction := expected.(types.Function); isFunction && expectedFunction.Signature != nil {
			got, ok := info.typeValue.(types.Function)
			if !ok || got.Signature == nil {
				a.typeMismatch(argument.Span(), expected, info.typeValue, string(operation)+" argument")
				continue
			}
			if _, compatible := functionValueScore(got.Signature, expectedFunction.Signature); !compatible {
				a.typeMismatch(argument.Span(), expected, info.typeValue, string(operation)+" handler")
			}
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
		ReturnNull: nullState,
	}
	return result
}
