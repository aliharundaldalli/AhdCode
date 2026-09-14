package semantic

import (
	"strings"
	"testing"
)

const webSocketPreamble = `bring HTTP
from HTTP bring (Server, Request, Response, WebSocket, WebSocketEndpoint, HTTPError)

liveMessage: Function := (socket: WebSocket, text: String) -> Nothing {
    write(text)
}

liveOpen: Function := (socket: WebSocket, request: Request) -> Nothing {
    write(request.path())
}

liveClose: Function := (socket: WebSocket, code: Int, reason: String) -> Nothing {
    write(reason)
}

liveAccept: Function := (request: Request) -> Response? {
    if request.header("X-Token") == null {
        return HTTP.text("Unauthorized", 401)
    }
    return null
}

wrongMessage: Function := (text: String) -> Nothing {
    write(text)
}

app: Server := HTTP.server("127.0.0.1", 8080)

`

func TestWebSocketValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, webSocketPreamble+`clients: Pair<String, WebSocket> := {}

echo: Function := (socket: WebSocket, text: String) -> Nothing {
    clients: Global Pair<String, WebSocket>
    clients[socket.id()] = socket
    delivered: Local Bool := socket.send("echo " + text)
    if not delivered {
        socket.close(1011, "echo failed")
    }
    if socket.isOpen() {
        socket.close()
        socket.close(4000)
    }
}

endpoint: WebSocketEndpoint := HTTP.websocket(echo)
endpoint = endpoint.withOpen(liveOpen)
endpoint = endpoint.withClose(liveClose)
endpoint = endpoint.withAccept(liveAccept)
endpoint = endpoint.withAllowedOrigins(["https://example.com", "http://localhost:3000"])
endpoint = endpoint.withMaxMessageBytes(4096)
endpoint = endpoint.withMaxQueuedMessages(16)
endpoint = endpoint.withMaxConnections(100)
app.websocket("/live", endpoint)
app.websocket("/rooms/*", HTTP.websocket(liveMessage))
attempt {
    app.websocket("/live", endpoint)
} except HTTPError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

// Wrong callback shapes and argument types are compile-time diagnostics.
func TestWebSocketRejectsStaticMistakes(t *testing.T) {
	for _, source := range []string{
		`HTTP.websocket()`,
		`HTTP.websocket("x")`,
		`HTTP.websocket(wrongMessage)`,
		`HTTP.websocket(liveOpen)`,
		`HTTP.websocket(liveMessage, liveOpen)`,
		`HTTP.websocket(liveMessage).withOpen(liveMessage)`,
		`HTTP.websocket(liveMessage).withClose(liveOpen)`,
		`HTTP.websocket(liveMessage).withAccept(liveMessage)`,
		`HTTP.websocket(liveMessage).withAllowedOrigins("https://example.com")`,
		`HTTP.websocket(liveMessage).withMaxMessageBytes("big")`,
		`HTTP.websocket(liveMessage).withMaxConnections()`,
		`HTTP.websocket(liveMessage).withCompression(true)`,
		`HTTP.websocket(liveMessage).start()`,
		`app.websocket("/live")`,
		`app.websocket("/live", liveMessage)`,
		`app.websocket(1, HTTP.websocket(liveMessage))`,
		`endpoint: Server := HTTP.websocket(liveMessage)`,
		"bad: Function := (socket: WebSocket, text: String) -> Nothing {\n    socket.close(\"normal\")\n}",
		"bad: Function := (socket: WebSocket, text: String) -> Nothing {\n    socket.close(1000, \"bye\", 1)\n}",
		"bad: Function := (socket: WebSocket, text: String) -> Nothing {\n    socket.send(1)\n}",
		"bad: Function := (socket: WebSocket, text: String) -> Nothing {\n    sent: Local String := socket.send(text)\n}",
		"bad: Function := (socket: WebSocket, text: String) -> Nothing {\n    socket.sendBinary(text)\n}",
		"bad: Function := (socket: WebSocket, text: String) -> Nothing {\n    socket.receive()\n}",
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, webSocketPreamble+source+"\n"))
	}
}

func TestWebSocketValuesAreNotConstructedDirectly(t *testing.T) {
	for source, hint := range map[string]string{
		`WebSocket("id")`:        "WebSocket values are produced",
		`WebSocketEndpoint("1")`: "create a WebSocketEndpoint with HTTP.websocket",
	} {
		result := analyzeWithStandardModules(t, webSocketPreamble+source+"\n")
		requireSemanticFailure(t, result)
		if !strings.Contains(semanticHintsOf(result), hint) {
			t.Fatalf("%s has no construction hint: %q", source, semanticHintsOf(result))
		}
	}
}

// The frozen v1.4.0 surface: no client, no binary, no hub.
func TestWebSocketSurfaceIsExactlyTheFrozenOne(t *testing.T) {
	if strings.Join(HTTPWebSocketOperations, ",") != "id,send,close,isOpen" {
		t.Fatalf("WebSocket operations = %v", HTTPWebSocketOperations)
	}
	want := "withOpen,withClose,withAccept,withAllowedOrigins,withMaxMessageBytes,withMaxQueuedMessages,withMaxConnections"
	if strings.Join(HTTPWebSocketEndpointOperations, ",") != want {
		t.Fatalf("WebSocketEndpoint operations = %v", HTTPWebSocketEndpointOperations)
	}
	module := StandardModuleInterfaces()["HTTP"]
	for _, name := range []string{"websocketClient", "dial", "websocketHub", "broadcast"} {
		if module.Exports[name] != nil {
			t.Fatalf("HTTP must not export %q", name)
		}
	}
}
