package golang

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// The WebSocket connection runtime imports the vendored library, so it joins a
// program only where the program creates an endpoint. Bringing HTTP or Web --
// whose App.websocket method always compiles -- is not enough.
func TestWebSocketConnectionRuntimeOnlyJoinsProgramsThatCreateEndpoints(t *testing.T) {
	for _, source := range []string{
		"write(\"hi\")\n",
		"bring HTTP\nfrom HTTP bring Server\napp: Server := HTTP.server(\"127.0.0.1\", 8080)\n",
		"bring Web\nwrite(Web.render(Web.UI.p(\"hi\")))\n",
	} {
		program := generate(t, source)
		if program.RequiresWebSocket {
			t.Fatalf("a program without an endpoint requires the WebSocket tree:\n%s", source)
		}
		for _, file := range program.Files {
			if file.Name == websocketConnRuntimeFileName {
				t.Fatalf("a program without an endpoint received the connection runtime:\n%s", source)
			}
		}
	}
	for _, source := range []string{
		"bring HTTP\nfrom HTTP bring WebSocket\necho: Function := (socket: WebSocket, text: String) -> Nothing {\n    socket.send(text)\n}\nendpoint := HTTP.websocket(echo)\n",
		"bring Web\nfrom Web bring WebSocket\necho: Function := (socket: WebSocket, text: String) -> Nothing {\n    socket.send(text)\n}\nendpoint := Web.websocket(echo)\n",
	} {
		program := generate(t, source)
		if !program.RequiresWebSocket {
			t.Fatalf("an endpoint program does not require the WebSocket tree:\n%s", source)
		}
		seen := 0
		for _, file := range program.Files {
			if file.Name != websocketConnRuntimeFileName {
				continue
			}
			seen++
			parsed, err := parser.ParseFile(token.NewFileSet(), file.Name, file.Content, 0)
			if err != nil {
				t.Fatalf("connection runtime is not valid Go: %v", err)
			}
			if parsed.Name.Name != "main" {
				t.Fatalf("connection runtime package clause not rewritten: %s", parsed.Name.Name)
			}
			for _, spec := range parsed.Imports {
				path, _ := strconv.Unquote(spec.Path.Value)
				if strings.Contains(path, ".") && path != "github.com/coder/websocket" {
					t.Fatalf("connection runtime imports %s", path)
				}
				for _, forbidden := range []string{"os/exec", "syscall", "unsafe", "net/rpc"} {
					if path == forbidden {
						t.Fatalf("connection runtime imports %s", path)
					}
				}
			}
		}
		if seen != 1 {
			t.Fatalf("connection runtime emitted %d times, want 1", seen)
		}
	}
}
