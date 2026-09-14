package module

import (
	"testing"

	"ahdcode/internal/framework"
	"ahdcode/internal/semantic"
)

// Web publishes HTTP's WebSocket surface itself rather than wrapping it:
// Web.websocket is the HTTP builtin, so only a program that actually calls it
// creates an endpoint, and the types keep HTTP's identities.
func TestWebFacadeReExportsWebSocketSurface(t *testing.T) {
	result := compileWorkspace(map[string]string{"/app.ahd": `bring Web
from Web bring (App, Request, WebSocket, WebSocketEndpoint)

onMessage: Function := (socket: WebSocket, text: String) -> Nothing {
    socket.send(text)
}

onOpen: Function := (socket: WebSocket, request: Request) -> Nothing {
    write(request.path())
}

endpoint: WebSocketEndpoint := Web.websocket(onMessage)
endpoint = endpoint.withOpen(onOpen)
site: App := Web.app(Web.configure())
site.websocket("/live", endpoint)
`}, "/app.ahd")
	if result.HasErrors() {
		t.Fatalf("Web WebSocket usage did not compile:\n%s", diagnosticsText(result))
	}
	web := result.Modules[ModuleID(framework.ModuleID("Web"))].Interface
	http := semantic.StandardModuleInterfaces()["HTTP"]
	for _, name := range []string{"WebSocket", "WebSocketEndpoint"} {
		exported := web.Exports[name]
		if exported == nil || exported.Class != http.Exports[name].Class {
			t.Fatalf("Web.%s is not HTTP's %s", name, name)
		}
	}
	function := web.Exports["websocket"]
	if function == nil || function.OriginModuleID != "builtin:HTTP" {
		t.Fatalf("Web.websocket is not the HTTP builtin: %#v", function)
	}
}
