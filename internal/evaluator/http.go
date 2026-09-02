package evaluator

import (
	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

const (
	evaluatorHTTPServerClass       = ir.ClassID("builtin:HTTP::class::Server")
	evaluatorHTTPRequestClass      = ir.ClassID("builtin:HTTP::class::Request")
	evaluatorHTTPResponseClass     = ir.ClassID("builtin:HTTP::class::Response")
	evaluatorHTTPCookieClass       = ir.ClassID("builtin:HTTP::class::Cookie")
	evaluatorHTTPSessionStoreClass = ir.ClassID("builtin:HTTP::class::SessionStore")
	evaluatorHTTPSessionClass      = ir.ClassID("builtin:HTTP::class::Session")
)

var (
	evaluatorHTTPServerField       = ir.FieldID("builtin:HTTP::class::Server::field::handle")
	evaluatorHTTPRequestField      = ir.FieldID("builtin:HTTP::class::Request::field::data")
	evaluatorHTTPResponseField     = ir.FieldID("builtin:HTTP::class::Response::field::data")
	evaluatorHTTPCookieField       = ir.FieldID("builtin:HTTP::class::Cookie::field::data")
	evaluatorHTTPSessionStoreField = ir.FieldID("builtin:HTTP::class::SessionStore::field::handle")
	evaluatorHTTPSessionField      = ir.FieldID("builtin:HTTP::class::Session::field::data")
)

func (session *Session) httpServer(handle string) *Instance {
	return &Instance{Class: evaluatorHTTPServerClass, Fields: map[ir.FieldID]any{evaluatorHTTPServerField: handle}}
}

func (session *Session) httpRequest(data string) *Instance {
	return &Instance{Class: evaluatorHTTPRequestClass, Fields: map[ir.FieldID]any{evaluatorHTTPRequestField: data}}
}

func (session *Session) httpResponse(data string) *Instance {
	return &Instance{Class: evaluatorHTTPResponseClass, Fields: map[ir.FieldID]any{evaluatorHTTPResponseField: data}}
}

func (session *Session) httpCookie(data string) *Instance {
	return &Instance{Class: evaluatorHTTPCookieClass, Fields: map[ir.FieldID]any{evaluatorHTTPCookieField: data}}
}

func (session *Session) httpSessionStore(handle string) *Instance {
	return &Instance{Class: evaluatorHTTPSessionStoreClass, Fields: map[ir.FieldID]any{evaluatorHTTPSessionStoreField: handle}}
}

func (session *Session) httpSession(data string) *Instance {
	return &Instance{Class: evaluatorHTTPSessionClass, Fields: map[ir.FieldID]any{evaluatorHTTPSessionField: data}}
}

func (session *Session) httpHandleOf(value any) string {
	instance := session.requireInstance(value)
	handle, ok := instance.Fields[evaluatorHTTPServerField].(string)
	if !ok || instance.Class != evaluatorHTTPServerClass {
		session.raise("HTTPError", "Server storage is corrupted")
	}
	return handle
}

func (session *Session) httpDataOf(value any, class ir.ClassID, field ir.FieldID, name string) string {
	instance := session.requireInstance(value)
	data, ok := instance.Fields[field].(string)
	if !ok || instance.Class != class {
		session.raise("HTTPError", name+" storage is corrupted")
	}
	return data
}

func (session *Session) httpRecover(name string) {
	recovered := recover()
	if recovered == nil {
		return
	}
	if signal, ok := recovered.(*ahdruntime.AhdSignal); ok {
		session.raise(name, signal.Message)
	}
	panic(recovered)
}

func (session *Session) httpIntArg(args []any, index int, fallback int64) int64 {
	if index >= len(args) || args[index] == nil {
		return fallback
	}
	return args[index].(int64)
}

func (session *Session) httpStringArg(args []any, index int, fallback string) string {
	if index >= len(args) || args[index] == nil {
		return fallback
	}
	return args[index].(string)
}

func (session *Session) httpBoolArg(args []any, index int, fallback bool) bool {
	if index >= len(args) || args[index] == nil {
		return fallback
	}
	return args[index].(bool)
}

func (session *Session) httpMutateSession(receiver any, data string) {
	instance := session.requireInstance(receiver)
	if instance.Class != evaluatorHTTPSessionClass {
		session.raise("HTTPError", "Session storage is corrupted")
	}
	instance.Fields[evaluatorHTTPSessionField] = data
}

func (session *Session) httpHandler(value any) ahdruntime.AhdHTTPHandler {
	function, ok := value.(*FunctionValue)
	if !ok || function == nil {
		session.raise("HTTPError", "HTTP route handler is missing")
	}
	return func(data string) string {
		result := session.invoke(function, []argumentValue{{value: session.httpRequest(data)}})
		return session.httpDataOf(result, evaluatorHTTPResponseClass, evaluatorHTTPResponseField, "Response")
	}
}

func (session *Session) httpBuiltin(name string, args []any) any {
	defer session.httpRecover("HTTPError")
	class := ahdruntime.AhdClassHTTPError
	switch name {
	case "server":
		return session.httpServer(ahdruntime.AhdHTTPServer(class, args[0].(string), args[1].(int64), session.httpIntArg(args, 2, 1048576)))
	case "text":
		return session.httpResponse(ahdruntime.AhdHTTPText(class, args[0].(string), session.httpIntArg(args, 1, 200)))
	case "html":
		return session.httpResponse(ahdruntime.AhdHTTPHTML(class, args[0].(string), session.httpIntArg(args, 1, 200)))
	case "response":
		return session.httpResponse(ahdruntime.AhdHTTPResponse(class, args[0].(int64), args[1].(string), args[2].(string)))
	case "redirect":
		return session.httpResponse(ahdruntime.AhdHTTPRedirect(class, args[0].(string), session.httpIntArg(args, 1, 303)))
	case "cookie":
		return session.httpCookie(ahdruntime.AhdHTTPCookie(class, args[0].(string), args[1].(string)))
	case "deleteCookie":
		return session.httpCookie(ahdruntime.AhdHTTPDeleteCookie(class, args[0].(string), session.httpStringArg(args, 1, "/")))
	case "sessions":
		return session.httpSessionStore(ahdruntime.AhdHTTPSessions(class,
			session.httpStringArg(args, 0, "ahd_session"),
			session.httpIntArg(args, 1, 86400),
			session.httpBoolArg(args, 2, false),
			session.httpStringArg(args, 3, "Lax")))
	}
	session.raise("Error", "unsupported HTTP function "+name)
	return nil
}

func (session *Session) httpOperation(name string, receiver any, args []any) any {
	defer session.httpRecover("HTTPError")
	class := ahdruntime.AhdClassHTTPError
	arg := func(index int) string { return args[index].(string) }
	switch name {
	case "Server.get":
		ahdruntime.AhdHTTPServerGet(class, session.httpHandleOf(receiver), arg(0), session.httpHandler(args[1]))
		return Nothing
	case "Server.post":
		ahdruntime.AhdHTTPServerPost(class, session.httpHandleOf(receiver), arg(0), session.httpHandler(args[1]))
		return Nothing
	case "Server.route":
		ahdruntime.AhdHTTPServerRoute(class, session.httpHandleOf(receiver), arg(0), arg(1), session.httpHandler(args[2]))
		return Nothing
	case "Server.start":
		ahdruntime.AhdHTTPServerStart(class, session.httpHandleOf(receiver))
		return Nothing
	case "Request.method":
		return ahdruntime.AhdHTTPRequestMethod(session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"))
	case "Request.path":
		return ahdruntime.AhdHTTPRequestPath(session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"))
	case "Request.query":
		return session.httpOptional(ahdruntime.AhdHTTPRequestQuery(class, session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"), arg(0)))
	case "Request.queryAll":
		return session.httpStringList(ahdruntime.AhdHTTPRequestQueryAll(class, session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"), arg(0)))
	case "Request.header":
		return session.httpOptional(ahdruntime.AhdHTTPRequestHeader(class, session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"), arg(0)))
	case "Request.headerAll":
		return session.httpStringList(ahdruntime.AhdHTTPRequestHeaderAll(class, session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"), arg(0)))
	case "Request.body":
		return ahdruntime.AhdHTTPRequestBody(class, session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"))
	case "Request.form":
		return session.httpOptional(ahdruntime.AhdHTTPRequestForm(class, session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"), arg(0)))
	case "Request.formAll":
		return session.httpStringList(ahdruntime.AhdHTTPRequestFormAll(class, session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"), arg(0)))
	case "Request.cookie":
		return session.httpOptional(ahdruntime.AhdHTTPRequestCookie(class, session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"), arg(0)))
	case "Request.cookieAll":
		return session.httpStringList(ahdruntime.AhdHTTPRequestCookieAll(class, session.httpDataOf(receiver, evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request"), arg(0)))
	case "Response.withHeader":
		return session.httpResponse(ahdruntime.AhdHTTPResponseWithHeader(class,
			session.httpDataOf(receiver, evaluatorHTTPResponseClass, evaluatorHTTPResponseField, "Response"), arg(0), arg(1)))
	case "Response.withCookie":
		return session.httpResponse(ahdruntime.AhdHTTPResponseWithCookie(class,
			session.httpDataOf(receiver, evaluatorHTTPResponseClass, evaluatorHTTPResponseField, "Response"),
			session.httpDataOf(args[0], evaluatorHTTPCookieClass, evaluatorHTTPCookieField, "Cookie")))
	case "Cookie.withPath":
		return session.httpCookie(ahdruntime.AhdHTTPCookieWithPath(class,
			session.httpDataOf(receiver, evaluatorHTTPCookieClass, evaluatorHTTPCookieField, "Cookie"), arg(0)))
	case "Cookie.withHttpOnly":
		return session.httpCookie(ahdruntime.AhdHTTPCookieWithHttpOnly(class,
			session.httpDataOf(receiver, evaluatorHTTPCookieClass, evaluatorHTTPCookieField, "Cookie"), args[0].(bool)))
	case "Cookie.withSecure":
		return session.httpCookie(ahdruntime.AhdHTTPCookieWithSecure(class,
			session.httpDataOf(receiver, evaluatorHTTPCookieClass, evaluatorHTTPCookieField, "Cookie"), args[0].(bool)))
	case "Cookie.withSameSite":
		return session.httpCookie(ahdruntime.AhdHTTPCookieWithSameSite(class,
			session.httpDataOf(receiver, evaluatorHTTPCookieClass, evaluatorHTTPCookieField, "Cookie"), arg(0)))
	case "Cookie.withMaxAge":
		return session.httpCookie(ahdruntime.AhdHTTPCookieWithMaxAge(class,
			session.httpDataOf(receiver, evaluatorHTTPCookieClass, evaluatorHTTPCookieField, "Cookie"), args[0].(int64)))
	case "SessionStore.open":
		return session.httpSession(ahdruntime.AhdHTTPSessionStoreOpen(class,
			session.httpDataOf(receiver, evaluatorHTTPSessionStoreClass, evaluatorHTTPSessionStoreField, "SessionStore"),
			session.httpDataOf(args[0], evaluatorHTTPRequestClass, evaluatorHTTPRequestField, "Request")))
	case "SessionStore.commit":
		return session.httpResponse(ahdruntime.AhdHTTPSessionStoreCommit(class,
			session.httpDataOf(receiver, evaluatorHTTPSessionStoreClass, evaluatorHTTPSessionStoreField, "SessionStore"),
			session.httpDataOf(args[0], evaluatorHTTPSessionClass, evaluatorHTTPSessionField, "Session"),
			session.httpDataOf(args[1], evaluatorHTTPResponseClass, evaluatorHTTPResponseField, "Response")))
	case "Session.get":
		return session.httpOptional(ahdruntime.AhdHTTPSessionGet(class,
			session.httpDataOf(receiver, evaluatorHTTPSessionClass, evaluatorHTTPSessionField, "Session"), arg(0)))
	case "Session.has":
		return ahdruntime.AhdHTTPSessionHas(class,
			session.httpDataOf(receiver, evaluatorHTTPSessionClass, evaluatorHTTPSessionField, "Session"), arg(0))
	case "Session.set":
		session.httpMutateSession(receiver, ahdruntime.AhdHTTPSessionSet(class,
			session.httpDataOf(receiver, evaluatorHTTPSessionClass, evaluatorHTTPSessionField, "Session"), arg(0), arg(1)))
		return Nothing
	case "Session.remove":
		session.httpMutateSession(receiver, ahdruntime.AhdHTTPSessionRemove(class,
			session.httpDataOf(receiver, evaluatorHTTPSessionClass, evaluatorHTTPSessionField, "Session"), arg(0)))
		return Nothing
	case "Session.clear":
		session.httpMutateSession(receiver, ahdruntime.AhdHTTPSessionClear(class,
			session.httpDataOf(receiver, evaluatorHTTPSessionClass, evaluatorHTTPSessionField, "Session")))
		return Nothing
	case "Session.rotate":
		session.httpMutateSession(receiver, ahdruntime.AhdHTTPSessionRotate(class,
			session.httpDataOf(receiver, evaluatorHTTPSessionClass, evaluatorHTTPSessionField, "Session")))
		return Nothing
	case "Session.destroy":
		session.httpMutateSession(receiver, ahdruntime.AhdHTTPSessionDestroy(class,
			session.httpDataOf(receiver, evaluatorHTTPSessionClass, evaluatorHTTPSessionField, "Session")))
		return Nothing
	}
	session.raise("Error", "unsupported HTTP operation "+name)
	return nil
}

func (session *Session) httpOptional(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func (session *Session) httpStringList(list *ahdruntime.AhdList[string]) *List {
	items := list.Snapshot()
	result := make([]any, len(items))
	for index, item := range items {
		result[index] = item
	}
	return &List{Items: result}
}
