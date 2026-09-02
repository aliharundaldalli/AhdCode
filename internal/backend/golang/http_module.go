package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const httpModulePrefix = "builtin:HTTP::"

var (
	httpServerClass             = ir.ClassID("builtin:HTTP::class::Server")
	httpRequestClass            = ir.ClassID("builtin:HTTP::class::Request")
	httpResponseClass           = ir.ClassID("builtin:HTTP::class::Response")
	httpCookieClass             = ir.ClassID("builtin:HTTP::class::Cookie")
	httpSessionStoreClass       = ir.ClassID("builtin:HTTP::class::SessionStore")
	httpSessionClass            = ir.ClassID("builtin:HTTP::class::Session")
	httpClientClass             = ir.ClassID("builtin:HTTP::class::Client")
	httpClientRequestClass      = ir.ClassID("builtin:HTTP::class::ClientRequest")
	httpClientResponseClass     = ir.ClassID("builtin:HTTP::class::ClientResponse")
	httpErrorClass              = ir.ClassID("builtin:HTTP::class::HTTPError")
	httpServerHandleField       = ir.FieldID("builtin:HTTP::class::Server::field::handle")
	httpRequestDataField        = ir.FieldID("builtin:HTTP::class::Request::field::data")
	httpResponseDataField       = ir.FieldID("builtin:HTTP::class::Response::field::data")
	httpCookieDataField         = ir.FieldID("builtin:HTTP::class::Cookie::field::data")
	httpSessionStoreHandleField = ir.FieldID("builtin:HTTP::class::SessionStore::field::handle")
	httpSessionDataField        = ir.FieldID("builtin:HTTP::class::Session::field::data")
	httpClientHandleField       = ir.FieldID("builtin:HTTP::class::Client::field::handle")
	httpClientRequestDataField  = ir.FieldID("builtin:HTTP::class::ClientRequest::field::data")
	httpClientResponseDataField = ir.FieldID("builtin:HTTP::class::ClientResponse::field::data")
)

func (generator *generator) httpCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), httpModulePrefix)
	errorClass := generator.descriptorName(httpErrorClass)
	text := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	integer := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	boolean := func(index int, fallback string) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}
	switch name {
	case "server":
		return generator.httpValueFrom(httpServerClass, "AhdHTTPServer("+errorClass+", "+text(0, `""`)+", "+
			integer(1, "int64(0)")+", "+integer(2, "int64(1048576)")+")", meta)
	case "text":
		return generator.httpValueFrom(httpResponseClass, "AhdHTTPText("+errorClass+", "+text(0, `""`)+", "+
			integer(1, "int64(200)")+")", meta)
	case "html":
		return generator.httpValueFrom(httpResponseClass, "AhdHTTPHTML("+errorClass+", "+text(0, `""`)+", "+
			integer(1, "int64(200)")+")", meta)
	case "response":
		return generator.httpValueFrom(httpResponseClass, "AhdHTTPResponse("+errorClass+", "+
			integer(0, "int64(200)")+", "+text(1, `""`)+", "+text(2, `""`)+")", meta)
	case "redirect":
		return generator.httpValueFrom(httpResponseClass, "AhdHTTPRedirect("+errorClass+", "+text(0, `""`)+", "+
			integer(1, "int64(303)")+")", meta)
	case "cookie":
		return generator.httpValueFrom(httpCookieClass, "AhdHTTPCookie("+errorClass+", "+text(0, `""`)+", "+text(1, `""`)+")", meta)
	case "deleteCookie":
		return generator.httpValueFrom(httpCookieClass, "AhdHTTPDeleteCookie("+errorClass+", "+text(0, `""`)+", "+text(1, `"/"`)+")", meta)
	case "sessions":
		return generator.httpValueFrom(httpSessionStoreClass, "AhdHTTPSessions("+errorClass+", "+
			text(0, `"ahd_session"`)+", "+integer(1, "int64(86400)")+", "+boolean(2, "false")+", "+text(3, `"Lax"`)+")", meta)
	case "client":
		return generator.httpValueFrom(httpClientClass, "AhdHTTPClient("+errorClass+", "+
			integer(0, "int64(30)")+", "+integer(1, "int64(8388608)")+", "+boolean(2, "true")+")", meta)
	case "clientRequest":
		return generator.httpValueFrom(httpClientRequestClass, "AhdHTTPClientRequest("+errorClass+", "+
			text(0, `""`)+", "+text(1, `""`)+")", meta)
	default:
		return generator.unsupported("HTTP function "+name, meta.Span)
	}
}

func (generator *generator) httpOperation(name string, value *ir.CallExpr) string {
	meta := value.ExprMeta()
	errorClass := generator.descriptorName(httpErrorClass)
	text := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.StringType}, false)
	}
	boolean := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}
	integer := func(index int) string {
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, false)
	}
	switch name {
	case "Server.get":
		return "AhdHTTPServerGet(" + errorClass + ", " + generator.httpDataOf(httpServerClass, httpServerHandleField, value.Callee) + ", " +
			text(0) + ", " + generator.httpHandler(value, 1) + ")"
	case "Server.post":
		return "AhdHTTPServerPost(" + errorClass + ", " + generator.httpDataOf(httpServerClass, httpServerHandleField, value.Callee) + ", " +
			text(0) + ", " + generator.httpHandler(value, 1) + ")"
	case "Server.route":
		return "AhdHTTPServerRoute(" + errorClass + ", " + generator.httpDataOf(httpServerClass, httpServerHandleField, value.Callee) + ", " +
			text(0) + ", " + text(1) + ", " + generator.httpHandler(value, 2) + ")"
	case "Server.start":
		return "AhdHTTPServerStart(" + errorClass + ", " + generator.httpDataOf(httpServerClass, httpServerHandleField, value.Callee) + ")"
	case "Request.method":
		return "AhdHTTPRequestMethod(" + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ")"
	case "Request.path":
		return "AhdHTTPRequestPath(" + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ")"
	case "Request.query":
		return "AhdHTTPRequestQuery(" + errorClass + ", " + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ", " + text(0) + ")"
	case "Request.queryAll":
		return "AhdHTTPRequestQueryAll(" + errorClass + ", " + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ", " + text(0) + ")"
	case "Request.header":
		return "AhdHTTPRequestHeader(" + errorClass + ", " + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ", " + text(0) + ")"
	case "Request.headerAll":
		return "AhdHTTPRequestHeaderAll(" + errorClass + ", " + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ", " + text(0) + ")"
	case "Request.body":
		return "AhdHTTPRequestBody(" + errorClass + ", " + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ")"
	case "Request.form":
		return "AhdHTTPRequestForm(" + errorClass + ", " + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ", " + text(0) + ")"
	case "Request.formAll":
		return "AhdHTTPRequestFormAll(" + errorClass + ", " + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ", " + text(0) + ")"
	case "Request.cookie":
		return "AhdHTTPRequestCookie(" + errorClass + ", " + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ", " + text(0) + ")"
	case "Request.cookieAll":
		return "AhdHTTPRequestCookieAll(" + errorClass + ", " + generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Callee) + ", " + text(0) + ")"
	case "Response.withHeader":
		return generator.httpValueFrom(httpResponseClass, "AhdHTTPResponseWithHeader("+errorClass+", "+
			generator.httpDataOf(httpResponseClass, httpResponseDataField, value.Callee)+", "+text(0)+", "+text(1)+")", meta)
	case "Response.withCookie":
		return generator.httpValueFrom(httpResponseClass, "AhdHTTPResponseWithCookie("+errorClass+", "+
			generator.httpDataOf(httpResponseClass, httpResponseDataField, value.Callee)+", "+
			generator.httpDataOf(httpCookieClass, httpCookieDataField, value.Arguments[0].Value)+")", meta)
	case "Cookie.withPath":
		return generator.httpValueFrom(httpCookieClass, "AhdHTTPCookieWithPath("+errorClass+", "+
			generator.httpDataOf(httpCookieClass, httpCookieDataField, value.Callee)+", "+text(0)+")", meta)
	case "Cookie.withHttpOnly":
		return generator.httpValueFrom(httpCookieClass, "AhdHTTPCookieWithHttpOnly("+errorClass+", "+
			generator.httpDataOf(httpCookieClass, httpCookieDataField, value.Callee)+", "+boolean(0)+")", meta)
	case "Cookie.withSecure":
		return generator.httpValueFrom(httpCookieClass, "AhdHTTPCookieWithSecure("+errorClass+", "+
			generator.httpDataOf(httpCookieClass, httpCookieDataField, value.Callee)+", "+boolean(0)+")", meta)
	case "Cookie.withSameSite":
		return generator.httpValueFrom(httpCookieClass, "AhdHTTPCookieWithSameSite("+errorClass+", "+
			generator.httpDataOf(httpCookieClass, httpCookieDataField, value.Callee)+", "+text(0)+")", meta)
	case "Cookie.withMaxAge":
		return generator.httpValueFrom(httpCookieClass, "AhdHTTPCookieWithMaxAge("+errorClass+", "+
			generator.httpDataOf(httpCookieClass, httpCookieDataField, value.Callee)+", "+integer(0)+")", meta)
	case "SessionStore.open":
		return generator.httpValueFrom(httpSessionClass, "AhdHTTPSessionStoreOpen("+errorClass+", "+
			generator.httpDataOf(httpSessionStoreClass, httpSessionStoreHandleField, value.Callee)+", "+
			generator.httpDataOf(httpRequestClass, httpRequestDataField, value.Arguments[0].Value)+")", meta)
	case "SessionStore.commit":
		return generator.httpValueFrom(httpResponseClass, "AhdHTTPSessionStoreCommit("+errorClass+", "+
			generator.httpDataOf(httpSessionStoreClass, httpSessionStoreHandleField, value.Callee)+", "+
			generator.httpDataOf(httpSessionClass, httpSessionDataField, value.Arguments[0].Value)+", "+
			generator.httpDataOf(httpResponseClass, httpResponseDataField, value.Arguments[1].Value)+")", meta)
	case "Session.get":
		return "AhdHTTPSessionGet(" + errorClass + ", " + generator.httpDataOf(httpSessionClass, httpSessionDataField, value.Callee) + ", " + text(0) + ")"
	case "Session.has":
		return "AhdHTTPSessionHas(" + errorClass + ", " + generator.httpDataOf(httpSessionClass, httpSessionDataField, value.Callee) + ", " + text(0) + ")"
	case "Session.set":
		return generator.httpSessionMutate(value, "AhdHTTPSessionSet("+errorClass+", %s, "+text(0)+", "+text(1)+")")
	case "Session.remove":
		return generator.httpSessionMutate(value, "AhdHTTPSessionRemove("+errorClass+", %s, "+text(0)+")")
	case "Session.clear":
		return generator.httpSessionMutate(value, "AhdHTTPSessionClear("+errorClass+", %s)")
	case "Session.rotate":
		return generator.httpSessionMutate(value, "AhdHTTPSessionRotate("+errorClass+", %s)")
	case "Session.destroy":
		return generator.httpSessionMutate(value, "AhdHTTPSessionDestroy("+errorClass+", %s)")
	case "Client.send":
		return generator.httpValueFrom(httpClientResponseClass, "AhdHTTPClientSend("+errorClass+", "+
			generator.httpDataOf(httpClientClass, httpClientHandleField, value.Callee)+", "+
			generator.httpDataOf(httpClientRequestClass, httpClientRequestDataField, value.Arguments[0].Value)+")", meta)
	case "Client.get":
		return generator.httpValueFrom(httpClientResponseClass, "AhdHTTPClientGet("+errorClass+", "+
			generator.httpDataOf(httpClientClass, httpClientHandleField, value.Callee)+", "+text(0)+")", meta)
	case "Client.post":
		contentType := `"text/plain; charset=utf-8"`
		if len(value.Arguments) > 2 && value.Arguments[2].Value != nil {
			contentType = text(2)
		}
		return generator.httpValueFrom(httpClientResponseClass, "AhdHTTPClientPost("+errorClass+", "+
			generator.httpDataOf(httpClientClass, httpClientHandleField, value.Callee)+", "+
			text(0)+", "+text(1)+", "+contentType+")", meta)
	case "ClientRequest.withHeader":
		return generator.httpValueFrom(httpClientRequestClass, "AhdHTTPClientRequestWithHeader("+errorClass+", "+
			generator.httpDataOf(httpClientRequestClass, httpClientRequestDataField, value.Callee)+", "+text(0)+", "+text(1)+")", meta)
	case "ClientRequest.addHeader":
		return generator.httpValueFrom(httpClientRequestClass, "AhdHTTPClientRequestAddHeader("+errorClass+", "+
			generator.httpDataOf(httpClientRequestClass, httpClientRequestDataField, value.Callee)+", "+text(0)+", "+text(1)+")", meta)
	case "ClientRequest.withBody":
		return generator.httpValueFrom(httpClientRequestClass, "AhdHTTPClientRequestWithBody("+errorClass+", "+
			generator.httpDataOf(httpClientRequestClass, httpClientRequestDataField, value.Callee)+", "+text(0)+")", meta)
	case "ClientResponse.status":
		return "AhdHTTPClientResponseStatus(" + generator.httpDataOf(httpClientResponseClass, httpClientResponseDataField, value.Callee) + ")"
	case "ClientResponse.body":
		return "AhdHTTPClientResponseBody(" + errorClass + ", " + generator.httpDataOf(httpClientResponseClass, httpClientResponseDataField, value.Callee) + ")"
	case "ClientResponse.header":
		return "AhdHTTPClientResponseHeader(" + errorClass + ", " + generator.httpDataOf(httpClientResponseClass, httpClientResponseDataField, value.Callee) + ", " + text(0) + ")"
	case "ClientResponse.headerAll":
		return "AhdHTTPClientResponseHeaderAll(" + errorClass + ", " + generator.httpDataOf(httpClientResponseClass, httpClientResponseDataField, value.Callee) + ", " + text(0) + ")"
	case "ClientResponse.url":
		return "AhdHTTPClientResponseURL(" + errorClass + ", " + generator.httpDataOf(httpClientResponseClass, httpClientResponseDataField, value.Callee) + ")"
	default:
		return generator.unsupported("HTTP operation "+name, meta.Span)
	}
}

func (generator *generator) httpHandler(value *ir.CallExpr, index int) string {
	handler := generator.expr(value.Arguments[index].Value)
	helper, ok := generator.httpHelper(httpRequestClass)
	if !ok {
		return generator.unsupported("an HTTP Request without its Class declaration", value.ExprMeta().Span)
	}
	reqType := generator.interfaceName(httpRequestClass)
	resType := generator.interfaceName(httpResponseClass)
	getter := generator.fieldName(httpResponseDataField) + "_get()"
	return "func(handler func(" + reqType + ") " + resType + ") AhdHTTPHandler { " +
		"return func(data string) string { return handler(" + helper + "(data))." + getter + " } }(" + handler + ")"
}

func (generator *generator) httpValueFrom(class ir.ClassID, data string, meta ir.ExprBase) string {
	helper, ok := generator.httpHelper(class)
	if !ok {
		return generator.unsupported("an HTTP value without its Class declaration", meta.Span)
	}
	return helper + "(" + data + ")"
}

func (generator *generator) httpDataOf(class ir.ClassID, field ir.FieldID, expression ir.Expr) string {
	rendered := generator.expr(expression)
	getter := generator.fieldName(field) + "_get()"
	return "func(value " + generator.interfaceName(class) + ") string { return value." + getter + " }(" + rendered + ")"
}

func (generator *generator) httpSessionMutate(value *ir.CallExpr, call string) string {
	receiver := generator.expr(value.Callee)
	sessionType := generator.interfaceName(httpSessionClass)
	getter := generator.fieldName(httpSessionDataField) + "_get()"
	setter := generator.fieldName(httpSessionDataField) + "_set"
	updated := strings.Replace(call, "%s", "session."+getter, 1)
	return "func(session " + sessionType + ") { session." + setter + "(" + updated + ") }(" + receiver + ")"
}

func (generator *generator) httpHelper(class ir.ClassID) (string, bool) {
	if generator.layouts[class] == nil {
		return "", false
	}
	if name, known := generator.timeHelpers[class]; known {
		return name, true
	}
	name := mangleNamed("http_", generator.classDisplayName(class), string(class))
	generator.timeHelpers[class] = name
	return name, true
}

func (generator *generator) emitHTTPHelpers(writer *emitter) {
	for _, class := range []ir.ClassID{
		httpServerClass, httpRequestClass, httpResponseClass,
		httpCookieClass, httpSessionStoreClass, httpSessionClass,
		httpClientClass, httpClientRequestClass, httpClientResponseClass,
	} {
		name, known := generator.timeHelpers[class]
		if !known {
			continue
		}
		layout := generator.layouts[class]
		if layout == nil {
			continue
		}
		constructor := generator.functions[layout.class.Constructor]
		if constructor == nil {
			continue
		}
		writer.open("func " + name + "(data string) " + generator.interfaceName(class) + " {")
		writer.line("return " + generator.callableName(constructor) + "(data)")
		writer.close("}")
		writer.blank()
	}
}
