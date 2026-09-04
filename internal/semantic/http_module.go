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
	httpErrorClass          = &types.ClassSymbol{ModuleID: httpModuleID, Name: "HTTPError", Parent: httpErrorParent}
	httpServerClass         = &types.ClassSymbol{ModuleID: httpModuleID, Name: "Server"}
	httpRequestClass        = &types.ClassSymbol{ModuleID: httpModuleID, Name: "Request"}
	httpResponseClass       = &types.ClassSymbol{ModuleID: httpModuleID, Name: "Response"}
	httpCookieClass         = &types.ClassSymbol{ModuleID: httpModuleID, Name: "Cookie"}
	httpSessionStoreClass   = &types.ClassSymbol{ModuleID: httpModuleID, Name: "SessionStore"}
	httpSessionClass        = &types.ClassSymbol{ModuleID: httpModuleID, Name: "Session"}
	httpClientClass         = &types.ClassSymbol{ModuleID: httpModuleID, Name: "Client"}
	httpClientRequestClass  = &types.ClassSymbol{ModuleID: httpModuleID, Name: "ClientRequest"}
	httpClientResponseClass = &types.ClassSymbol{ModuleID: httpModuleID, Name: "ClientResponse"}
	httpUploadedFileClass   = &types.ClassSymbol{ModuleID: httpModuleID, Name: "UploadedFile"}
)

// HTTPErrorIdentity, HTTPServerIdentity, HTTPRequestIdentity, and
// HTTPResponseIdentity expose the canonical identities to lowering without
// coupling the public module interface to a backend.
func HTTPErrorIdentity() *types.ClassSymbol        { return httpErrorClass }
func HTTPServerIdentity() *types.ClassSymbol       { return httpServerClass }
func HTTPRequestIdentity() *types.ClassSymbol      { return httpRequestClass }
func HTTPResponseIdentity() *types.ClassSymbol     { return httpResponseClass }
func HTTPCookieIdentity() *types.ClassSymbol       { return httpCookieClass }
func HTTPSessionStoreIdentity() *types.ClassSymbol { return httpSessionStoreClass }
func HTTPSessionIdentity() *types.ClassSymbol      { return httpSessionClass }
func HTTPClientIdentity() *types.ClassSymbol       { return httpClientClass }
func HTTPClientRequestIdentity() *types.ClassSymbol {
	return httpClientRequestClass
}
func HTTPClientResponseIdentity() *types.ClassSymbol {
	return httpClientResponseClass
}
func HTTPUploadedFileIdentity() *types.ClassSymbol {
	return httpUploadedFileClass
}

// HTTPServerOperations, HTTPRequestOperations, and HTTPResponseOperations
// name the members each Class publishes through built-in type operations, so
// has/has not reports the real surface and the IR Class agrees with the
// frontend.
var HTTPServerOperations = []string{"get", "post", "route", "static", "start"}
var HTTPRequestOperations = []string{
	"method", "path", "query", "queryAll", "header", "headerAll", "body", "form", "formAll",
	"cookie", "cookieAll", "file", "files",
}
var HTTPResponseOperations = []string{"withHeader", "withCookie"}
var HTTPCookieOperations = []string{"withPath", "withHttpOnly", "withSecure", "withSameSite", "withMaxAge"}
var HTTPSessionStoreOperations = []string{"open", "commit"}
var HTTPSessionOperations = []string{"get", "has", "set", "remove", "clear", "rotate", "destroy"}
var HTTPClientOperations = []string{"send", "get", "post"}
var HTTPClientRequestOperations = []string{"withHeader", "addHeader", "withBody"}
var HTTPClientResponseOperations = []string{"status", "body", "header", "headerAll", "url"}

// HTTPUploadedFileOperations is the read-only v0.8.0 upload surface. There
// is deliberately no bytes()/raw()/stream()/tempPath(): the payload stays
// opaque and is persisted only through save.
var HTTPUploadedFileOperations = []string{
	"originalName", "declaredContentType", "detectedContentType", "size", "save",
}

func httpServerType() types.Type       { return types.Class{Symbol: httpServerClass} }
func httpRequestType() types.Type      { return types.Class{Symbol: httpRequestClass} }
func httpResponseType() types.Type     { return types.Class{Symbol: httpResponseClass} }
func httpCookieType() types.Type       { return types.Class{Symbol: httpCookieClass} }
func httpSessionStoreType() types.Type { return types.Class{Symbol: httpSessionStoreClass} }
func httpSessionType() types.Type      { return types.Class{Symbol: httpSessionClass} }
func httpClientType() types.Type       { return types.Class{Symbol: httpClientClass} }
func httpClientRequestType() types.Type {
	return types.Class{Symbol: httpClientRequestClass}
}
func httpClientResponseType() types.Type {
	return types.Class{Symbol: httpClientResponseClass}
}
func httpUploadedFileType() types.Type {
	return types.Class{Symbol: httpUploadedFileClass}
}

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
		{"Cookie", httpCookieClass}, {"SessionStore", httpSessionStoreClass},
		{"Session", httpSessionClass}, {"Client", httpClientClass},
		{"ClientRequest", httpClientRequestClass}, {"ClientResponse", httpClientResponseClass},
		{"UploadedFile", httpUploadedFileClass},
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
	filePath := types.Parameter{Name: "path", Type: types.String}
	fileContentType := types.Parameter{Name: "contentType", Type: types.String}
	fileName := types.Parameter{Name: "fileName", Type: types.String}
	addStandardExport(module, standardFunction(httpModuleID, "file", httpResponseType(), filePath, fileContentType))
	addStandardExport(module, standardFunction(httpModuleID, "download", httpResponseType(), filePath, fileContentType, fileName))
	addStandardExport(module, standardFunction(httpModuleID, "cookie", httpCookieType(),
		types.Parameter{Name: "name", Type: types.String}, types.Parameter{Name: "value", Type: types.String}))
	addStandardExport(module, standardFunction(httpModuleID, "deleteCookie", httpCookieType(),
		types.Parameter{Name: "name", Type: types.String},
		types.Parameter{Name: "path", Type: types.String, HasDefault: true}))
	addStandardExport(module, standardFunction(httpModuleID, "sessions", httpSessionStoreType(),
		types.Parameter{Name: "cookieName", Type: types.String, HasDefault: true},
		types.Parameter{Name: "maxAgeSeconds", Type: types.Int, HasDefault: true},
		types.Parameter{Name: "secure", Type: types.Bool, HasDefault: true},
		types.Parameter{Name: "sameSite", Type: types.String, HasDefault: true}))
	addStandardExport(module, standardFunction(httpModuleID, "client", httpClientType(),
		types.Parameter{Name: "timeoutSeconds", Type: types.Int, HasDefault: true},
		types.Parameter{Name: "maxResponseBytes", Type: types.Int, HasDefault: true},
		types.Parameter{Name: "followRedirects", Type: types.Bool, HasDefault: true}))
	addStandardExport(module, standardFunction(httpModuleID, "clientRequest", httpClientRequestType(),
		types.Parameter{Name: "method", Type: types.String},
		types.Parameter{Name: "url", Type: types.String}))
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
	case "Cookie":
		return "create a Cookie with HTTP.cookie or HTTP.deleteCookie", true
	case "SessionStore":
		return "create a SessionStore with HTTP.sessions", true
	case "Session":
		return "Session values are produced by SessionStore.open", true
	case "Client":
		return "create a Client with HTTP.client", true
	case "ClientRequest":
		return "create a ClientRequest with HTTP.clientRequest", true
	case "ClientResponse":
		return "ClientResponse values are produced by Client.send, Client.get, or Client.post", true
	case "UploadedFile":
		return "UploadedFile values are produced by Request.file or Request.files", true
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
		HTTPServerGet:    {[]types.Type{types.String, handler}, types.Nothing, false, "pass a path String and a (request: Request) -> Response Function"},
		HTTPServerPost:   {[]types.Type{types.String, handler}, types.Nothing, false, "pass a path String and a (request: Request) -> Response Function"},
		HTTPServerRoute:  {[]types.Type{types.String, types.String, handler}, types.Nothing, false, "pass a method String, a path String, and a (request: Request) -> Response Function"},
		HTTPServerStatic: {[]types.Type{types.String, types.String}, types.Nothing, false, "pass a URL path prefix String and a filesystem root directory String"},
		HTTPServerStart:  {none, types.Nothing, false, "call start with no argument"},

		HTTPRequestMethod:    {none, types.String, false, "call method with no argument"},
		HTTPRequestPath:      {none, types.String, false, "call path with no argument"},
		HTTPRequestQuery:     {[]types.Type{types.String}, types.String, true, "pass one String query name"},
		HTTPRequestQueryAll:  {[]types.Type{types.String}, strings, false, "pass one String query name"},
		HTTPRequestHeader:    {[]types.Type{types.String}, types.String, true, "pass one String header name"},
		HTTPRequestHeaderAll: {[]types.Type{types.String}, strings, false, "pass one String header name"},
		HTTPRequestBody:      {none, types.String, false, "call body with no argument"},
		HTTPRequestForm:      {[]types.Type{types.String}, types.String, true, "pass one String form field name"},
		HTTPRequestFormAll:   {[]types.Type{types.String}, strings, false, "pass one String form field name"},
		HTTPRequestCookie:    {[]types.Type{types.String}, types.String, true, "pass one String cookie name"},
		HTTPRequestCookieAll: {[]types.Type{types.String}, strings, false, "pass one String cookie name"},

		HTTPResponseWithHeader: {[]types.Type{types.String, types.String}, httpResponseType(), false, "pass a header name String and a header value String"},
		HTTPResponseWithCookie: {[]types.Type{httpCookieType()}, httpResponseType(), false, "pass one Cookie"},

		HTTPCookieWithPath:     {[]types.Type{types.String}, httpCookieType(), false, "pass one String path"},
		HTTPCookieWithHttpOnly: {[]types.Type{types.Bool}, httpCookieType(), false, "pass one Bool"},
		HTTPCookieWithSecure:   {[]types.Type{types.Bool}, httpCookieType(), false, "pass one Bool"},
		HTTPCookieWithSameSite: {[]types.Type{types.String}, httpCookieType(), false, "pass Lax, Strict, or None"},
		HTTPCookieWithMaxAge:   {[]types.Type{types.Int}, httpCookieType(), false, "pass Max-Age in seconds as Int"},

		HTTPSessionStoreOpen:   {[]types.Type{httpRequestType()}, httpSessionType(), false, "pass the current Request"},
		HTTPSessionStoreCommit: {[]types.Type{httpSessionType(), httpResponseType()}, httpResponseType(), false, "pass the Session and a Response"},

		HTTPSessionGet:     {[]types.Type{types.String}, types.String, true, "pass one String session key"},
		HTTPSessionHas:     {[]types.Type{types.String}, types.Bool, false, "pass one String session key"},
		HTTPSessionSet:     {[]types.Type{types.String, types.String}, types.Nothing, false, "pass a String key and a String value"},
		HTTPSessionRemove:  {[]types.Type{types.String}, types.Nothing, false, "pass one String session key"},
		HTTPSessionClear:   {none, types.Nothing, false, "call clear with no argument"},
		HTTPSessionRotate:  {none, types.Nothing, false, "call rotate with no argument"},
		HTTPSessionDestroy: {none, types.Nothing, false, "call destroy with no argument"},

		HTTPClientSend: {[]types.Type{httpClientRequestType()}, httpClientResponseType(), false, "pass one ClientRequest"},
		HTTPClientGet:  {[]types.Type{types.String}, httpClientResponseType(), false, "pass one String URL"},
		HTTPClientPost: {[]types.Type{types.String, types.String, types.String}, httpClientResponseType(), false, "pass a URL String, a body String, and optionally a Content-Type String"},

		HTTPClientRequestWithHeader: {[]types.Type{types.String, types.String}, httpClientRequestType(), false, "pass a header name String and a header value String"},
		HTTPClientRequestAddHeader:  {[]types.Type{types.String, types.String}, httpClientRequestType(), false, "pass a header name String and a header value String"},
		HTTPClientRequestWithBody:   {[]types.Type{types.String}, httpClientRequestType(), false, "pass one String body"},

		HTTPClientResponseStatus:    {none, types.Int, false, "call status with no argument"},
		HTTPClientResponseBody:      {none, types.String, false, "call body with no argument"},
		HTTPClientResponseHeader:    {[]types.Type{types.String}, types.String, true, "pass one String header name"},
		HTTPClientResponseHeaderAll: {[]types.Type{types.String}, strings, false, "pass one String header name"},
		HTTPClientResponseURL:       {none, types.String, false, "call url with no argument"},

		HTTPRequestFile:  {[]types.Type{types.String}, httpUploadedFileType(), true, "pass one String file field name"},
		HTTPRequestFiles: {[]types.Type{types.String}, types.List{Element: httpUploadedFileType()}, false, "pass one String file field name"},

		HTTPUploadedFileOriginalName:        {none, types.String, false, "call originalName with no argument"},
		HTTPUploadedFileDeclaredContentType: {none, types.String, true, "call declaredContentType with no argument"},
		HTTPUploadedFileDetectedContentType: {none, types.String, false, "call detectedContentType with no argument"},
		HTTPUploadedFileSize:                {none, types.Int, false, "call size with no argument"},
		HTTPUploadedFileSave:                {[]types.Type{types.String}, types.String, false, "pass one String directory path"},
	}
}

var httpOperationNames = map[string]map[string]TypeOperation{
	"Server": {
		"get": HTTPServerGet, "post": HTTPServerPost, "route": HTTPServerRoute,
		"static": HTTPServerStatic, "start": HTTPServerStart,
	},
	"Request": {
		"method": HTTPRequestMethod, "path": HTTPRequestPath,
		"query": HTTPRequestQuery, "queryAll": HTTPRequestQueryAll,
		"header": HTTPRequestHeader, "headerAll": HTTPRequestHeaderAll,
		"body": HTTPRequestBody, "form": HTTPRequestForm, "formAll": HTTPRequestFormAll,
		"file": HTTPRequestFile, "files": HTTPRequestFiles,
		"cookie": HTTPRequestCookie, "cookieAll": HTTPRequestCookieAll,
	},
	"Response": {"withHeader": HTTPResponseWithHeader, "withCookie": HTTPResponseWithCookie},
	"Cookie": {
		"withPath": HTTPCookieWithPath, "withHttpOnly": HTTPCookieWithHttpOnly,
		"withSecure": HTTPCookieWithSecure, "withSameSite": HTTPCookieWithSameSite,
		"withMaxAge": HTTPCookieWithMaxAge,
	},
	"SessionStore": {"open": HTTPSessionStoreOpen, "commit": HTTPSessionStoreCommit},
	"Session": {
		"get": HTTPSessionGet, "has": HTTPSessionHas, "set": HTTPSessionSet,
		"remove": HTTPSessionRemove, "clear": HTTPSessionClear,
		"rotate": HTTPSessionRotate, "destroy": HTTPSessionDestroy,
	},
	"Client": {"send": HTTPClientSend, "get": HTTPClientGet, "post": HTTPClientPost},
	"ClientRequest": {
		"withHeader": HTTPClientRequestWithHeader, "addHeader": HTTPClientRequestAddHeader,
		"withBody": HTTPClientRequestWithBody,
	},
	"ClientResponse": {
		"status": HTTPClientResponseStatus, "body": HTTPClientResponseBody,
		"header": HTTPClientResponseHeader, "headerAll": HTTPClientResponseHeaderAll,
		"url": HTTPClientResponseURL,
	},
	"UploadedFile": {
		"originalName":        HTTPUploadedFileOriginalName,
		"declaredContentType": HTTPUploadedFileDeclaredContentType,
		"detectedContentType": HTTPUploadedFileDetectedContentType,
		"size":                HTTPUploadedFileSize, "save": HTTPUploadedFileSave,
	},
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
	optional := 0
	if operation == HTTPClientPost {
		optional = 1
	}
	minimum := len(shape.parameters) - optional
	if len(call.Arguments) < minimum || len(call.Arguments) > len(shape.parameters) {
		expected := fmt.Sprintf("%d", len(shape.parameters))
		if optional > 0 {
			expected = fmt.Sprintf("%d to %d", minimum, len(shape.parameters))
		}
		a.error(codeCallArguments, fmt.Sprintf("%s expects %s argument(s); received %d", operation, expected, len(call.Arguments)), call.Span(), shape.hint)
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
		parameters[index] = types.Parameter{Type: expected, HasDefault: optional > 0 && index >= len(shape.parameters)-optional}
	}
	a.result.SelectedCallables[call] = &Callable{
		Signature:  &types.Signature{Parameters: parameters, Return: shape.result},
		ReturnNull: nullState,
	}
	return result
}
