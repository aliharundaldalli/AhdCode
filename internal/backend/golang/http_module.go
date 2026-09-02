package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const httpModulePrefix = "builtin:HTTP::"

var (
	httpServerClass       = ir.ClassID("builtin:HTTP::class::Server")
	httpRequestClass      = ir.ClassID("builtin:HTTP::class::Request")
	httpResponseClass     = ir.ClassID("builtin:HTTP::class::Response")
	httpErrorClass        = ir.ClassID("builtin:HTTP::class::HTTPError")
	httpServerHandleField = ir.FieldID("builtin:HTTP::class::Server::field::handle")
	httpRequestDataField  = ir.FieldID("builtin:HTTP::class::Request::field::data")
	httpResponseDataField = ir.FieldID("builtin:HTTP::class::Response::field::data")
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
	case "Response.withHeader":
		return generator.httpValueFrom(httpResponseClass, "AhdHTTPResponseWithHeader("+errorClass+", "+
			generator.httpDataOf(httpResponseClass, httpResponseDataField, value.Callee)+", "+text(0)+", "+text(1)+")", meta)
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
	for _, class := range []ir.ClassID{httpServerClass, httpRequestClass, httpResponseClass} {
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
