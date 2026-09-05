package build

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestWebV017ContextRoutesGroupsAndGuardsCompile(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `bring Web
from Web bring (App, Request, RequestContext, Response, SessionStore, RouteSet, RouteGroup)

legacy: Function := (request: Request) -> Response {
    return Web.text("legacy", 200)
}

home: Function := (context: RequestContext) -> Response {
    return context.respond(Web.text("home", 200))
}

allow: Function := (context: RequestContext) -> Response? {
    return null
}

deny: Function := (context: RequestContext) -> Response? {
    return context.respond(Web.text("no", 403))
}

users: Function := (context: RequestContext) -> Response {
    return context.respond(Web.text("users", 200))
}

application: App := Web.start()
sessions: SessionStore := Web.sessions("t", 60, false, "Lax")
routes: RouteSet := Web.routes(application, sessions)
routes.get("/", home)
application.get("/legacy", legacy)
admin: RouteGroup := routes.group("/admin")
admin.get("/users", users, allow, deny)
admin.get("/*", users, allow, deny)
write("ok")
`})
	out := runWebEnvProgram(t, directory, map[string]string{
		"APP_NAME": "Example", "APP_ENV": "development",
		"APP_HOST": "localhost", "APP_PROTOCOL": "http",
		"SERVER_HOST": "127.0.0.1", "SERVER_PORT": "8080",
	})
	if strings.TrimSpace(out) != "ok" {
		t.Fatalf("registration program: %q", out)
	}
}

func TestWebV017RejectsNonCanonicalGroupPrefix(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `bring Web
from Web bring (App, SessionStore, WebRouteError)

application: App := Web.start()
sessions: SessionStore := Web.sessions("t", 60, false, "Lax")
routes := Web.routes(application, sessions)
attempt {
    routes.group("admin")
    write("grouped")
} except WebRouteError as error {
    write(error.message)
}
`})
	out := runWebEnvProgram(t, directory, map[string]string{
		"APP_NAME": "Example", "APP_ENV": "development",
		"APP_HOST": "localhost", "APP_PROTOCOL": "http",
		"SERVER_HOST": "127.0.0.1", "SERVER_PORT": "8080",
	})
	if !strings.Contains(out, "canonical path fragment") {
		t.Fatalf("expected WebRouteError; received %q", out)
	}
}

func TestWebV017ExampleCompiles(t *testing.T) {
	for _, entry := range []string{
		"../../examples/v0.17/routes_guards/app.ahd",
		"../../docs/qa/v0.17/portal_routes.ahd",
	} {
		path, err := filepath.Abs(entry)
		if err != nil {
			t.Fatal(err)
		}
		result := Compile(path)
		if result.HasErrors() {
			t.Fatalf("%s:\n%s", path, diagnosticText(result.Diagnostics))
		}
	}
}

func TestWebV017OldHandlersRemainValid(t *testing.T) {
	out := runWebProgram(t, `bring Web
from Web bring (Request, Response)
bring HTTP
from HTTP bring Server

hello: Function := (request: Request) -> Response {
    return Web.text("hello", 200)
}

server: Server := HTTP.server("127.0.0.1", 8197)
server.get("/hello", hello)
write("compat")
`)
	if strings.TrimSpace(out) != "compat" {
		t.Fatalf("old handler: %q", out)
	}
}
