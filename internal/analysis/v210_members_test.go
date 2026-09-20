package analysis

import (
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/semantic"
	"ahdcode/internal/types"
)

// The v2.1 HTTP and SMTP members reach editors from the compiler's own
// operation shapes -- the metadata the compiler checks calls against. There
// is no editor-side list of them, and v2.0's Plot member bug is the reason
// this is checked rather than assumed.

// v210MemberSource calls every v2.1 member once, so hover and signature help
// have a real call site to resolve.
const v210MemberSource = `bring HTTP
bring SMTP
from HTTP bring (Client, ClientRequest, ClientFileResponse, WebSocketClient, WebSocketConnection)
from SMTP bring (SMTPMessage)

client: Client := HTTP.client()
request: ClientRequest := HTTP.clientRequest("POST", "http://127.0.0.1:9/upload")
request = request.withMultipartField("title", "report")
request = request.withMultipartFile("file", "report.pdf", "report.pdf", "application/pdf")
saved: ClientFileResponse := client.sendToFile(request, "answer.bin")
fetched: ClientFileResponse := client.download("http://127.0.0.1:9/file", "copy.bin")
status: Int := fetched.status()
size: Int := fetched.size()
url: String := fetched.url()
one: String? := fetched.header("Content-Type")
many: List<String> := fetched.headerAll("Set-Cookie")

socket: WebSocketClient := HTTP.webSocketClient("ws://127.0.0.1:9/ws")
socket = socket.withHeader("Authorization", "Bearer token")
socket = socket.withTimeout(5)
socket = socket.withMaxMessageBytes(4096)
live: WebSocketConnection := socket.connect()
sent: Bool := live.send("hello")
heard: String? := live.receive(1)
open: Bool := live.isOpen()
code: Int? := live.closeCode()
reason: String := live.closeReason()
live.close(1000, "done")

message: SMTPMessage := SMTP.message("a@example.com", ["b@example.com"], "Subject")
message = message.withAttachment("report.pdf", "report.pdf", "application/pdf")
write(status.toString() + size.toString() + url + many.length().toString())
write(saved.status().toString() + sent.toString() + open.toString() + reason)
write(one ?? "" )
write(heard ?? "")
write(code?.toString() ?? "")
`

// v210Members is the exact surface section 39 of the v2.1 plan requires an
// editor to know, with the hover text each one must render.
var v210Members = []struct {
	receiver string
	member   string
	hover    string
}{
	{"client", "download", "download: (String, String) -> ClientFileResponse"},
	{"client", "sendToFile", "sendToFile: (ClientRequest, String) -> ClientFileResponse"},
	{"fetched", "status", "status: () -> Int"},
	{"fetched", "header", "header: (String) -> String?"},
	{"fetched", "headerAll", "headerAll: (String) -> List<String>"},
	{"fetched", "url", "url: () -> String"},
	{"fetched", "size", "size: () -> Int"},
	{"request", "withMultipartField", "withMultipartField: (String, String) -> ClientRequest"},
	{"request", "withMultipartFile", "withMultipartFile: (String, String, String := default, String := default) -> ClientRequest"},
	{"message", "withAttachment", "withAttachment: (String, String := default, String := default) -> SMTPMessage"},
	{"socket", "withHeader", "withHeader: (String, String) -> WebSocketClient"},
	{"socket", "withTimeout", "withTimeout: (Int) -> WebSocketClient"},
	{"socket", "withMaxMessageBytes", "withMaxMessageBytes: (Int) -> WebSocketClient"},
	{"socket", "connect", "connect: () -> WebSocketConnection"},
	{"live", "send", "send: (String) -> Bool"},
	{"live", "receive", "receive: (Int := default) -> String?"},
	{"live", "close", "close: (Int := default, String := default) -> Nothing"},
	{"live", "isOpen", "isOpen: () -> Bool"},
	{"live", "closeCode", "closeCode: () -> Int?"},
	{"live", "closeReason", "closeReason: () -> String"},
}

func TestHoverAndSignatureHelpForEveryV210Member(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, v210MemberSource)
	for _, member := range v210Members {
		target := member.receiver + "." + member.member
		offset := offsetOf(t, v210MemberSource, target+"(")
		hover, found := store.Hover(path, offset+len(member.receiver)+2)
		if !found {
			t.Fatalf("no hover on %s", target)
		}
		if hover.Text != member.hover {
			t.Fatalf("hover on %s = %q, want %q", target, hover.Text, member.hover)
		}
		help, found := store.SignatureHelp(path, offset+len(target)+1)
		if !found || help.Label == "" {
			t.Fatalf("signature help for %s = %#v (%v)", target, help, found)
		}
		if !strings.HasSuffix(member.hover, help.Label) {
			t.Fatalf("signature help for %s = %q, want the tail of %q", target, help.Label, member.hover)
		}
	}
}

func TestCompletionOffersEveryV210Member(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	prelude := `bring HTTP
bring SMTP
from HTTP bring (Client, ClientRequest, ClientFileResponse, WebSocketClient, WebSocketConnection)
from SMTP bring (SMTPMessage)
client: Client := HTTP.client()
request: ClientRequest := HTTP.clientRequest("POST", "http://127.0.0.1:9/u")
fetched: ClientFileResponse := client.download("http://127.0.0.1:9/f", "copy.bin")
socket: WebSocketClient := HTTP.webSocketClient("ws://127.0.0.1:9/ws")
live: WebSocketConnection := socket.connect()
message: SMTPMessage := SMTP.message("a@example.com", ["b@example.com"], "S")
`
	for _, testCase := range []struct {
		receiver string
		want     []string
		absent   []string
	}{
		{"client", semantic.HTTPClientOperations, []string{"body", "receive"}},
		{"request", semantic.HTTPClientRequestOperations, []string{"send", "status"}},
		// A ClientFileResponse never offers body(): its payload is in the file.
		{"fetched", semantic.HTTPClientFileResponseOperations, []string{"body"}},
		{"socket", semantic.HTTPWebSocketClientOperations, []string{"send", "receive", "close"}},
		{"live", semantic.HTTPWebSocketConnectionOperations, []string{"connect", "withHeader"}},
		{"message", semantic.SMTPMessageOperations, []string{"send", "withPlainAuth"}},
	} {
		source := prelude + testCase.receiver + "."
		store.Open(path, source+"\n")
		items := store.Completion(path, len(source))
		if len(testCase.want) == 0 {
			t.Fatalf("%s publishes no members", testCase.receiver)
		}
		for _, name := range testCase.want {
			if !hasLabel(items, name) {
				t.Fatalf("%s. offers no %q; got %#v", testCase.receiver, name, items)
			}
		}
		for _, name := range testCase.absent {
			if hasLabel(items, name) {
				t.Fatalf("%s. unexpectedly offers %q", testCase.receiver, name)
			}
		}
	}
}

// TestV210EditorMembersMatchCompilerMembers keeps the editor aligned with
// the compiler: every name an HTTP or SMTP Class publishes resolves to a
// member Symbol carrying the signature the compiler checks calls against.
func TestV210EditorMembersMatchCompilerMembers(t *testing.T) {
	identities := map[string]*types.ClassSymbol{
		"Client":              semantic.HTTPClientIdentity(),
		"ClientRequest":       semantic.HTTPClientRequestIdentity(),
		"ClientResponse":      semantic.HTTPClientResponseIdentity(),
		"ClientFileResponse":  semantic.HTTPClientFileResponseIdentity(),
		"WebSocketClient":     semantic.HTTPWebSocketClientIdentity(),
		"WebSocketConnection": semantic.HTTPWebSocketConnectionIdentity(),
		"WebSocket":           semantic.HTTPWebSocketIdentity(),
		"WebSocketEndpoint":   semantic.HTTPWebSocketEndpointIdentity(),
		"SMTPMessage":         semantic.SMTPMessageIdentity(),
		"SMTPClient":          semantic.SMTPClientIdentity(),
	}
	names := map[string][]string{
		"Client":              semantic.HTTPClientOperations,
		"ClientRequest":       semantic.HTTPClientRequestOperations,
		"ClientResponse":      semantic.HTTPClientResponseOperations,
		"ClientFileResponse":  semantic.HTTPClientFileResponseOperations,
		"WebSocketClient":     semantic.HTTPWebSocketClientOperations,
		"WebSocketConnection": semantic.HTTPWebSocketConnectionOperations,
		"WebSocket":           semantic.HTTPWebSocketOperations,
		"WebSocketEndpoint":   semantic.HTTPWebSocketEndpointOperations,
		"SMTPMessage":         semantic.SMTPMessageOperations,
		"SMTPClient":          semantic.SMTPClientOperations,
	}
	for class, identity := range identities {
		members := semantic.BuiltinClassMembers(identity)
		if len(members) != len(names[class]) {
			t.Fatalf("%s publishes %d members for %d names", class, len(members), len(names[class]))
		}
		for index, member := range members {
			if member == nil || member.Name != names[class][index] || member.Callable == nil || member.Callable.Signature == nil {
				t.Fatalf("%s member %d = %#v", class, index, member)
			}
		}
	}
}

// The v2.0 Plot and GUI member metadata must still be there: this is the
// regression the v2.0 release fixed, rerun beside the v2.1 surface.
func TestV200PlotAndGUIMembersStillPublished(t *testing.T) {
	for class, names := range map[string][]string{
		"Chart":   semantic.PlotChartOperations,
		"Figure":  semantic.PlotFigureOperations,
		"Surface": semantic.PlotSurfaceOperations,
	} {
		if len(names) == 0 {
			t.Fatalf("%s publishes no members", class)
		}
		for _, name := range names {
			if !semantic.TypeOperationBindsArguments(semantic.TypeOperation(class + "." + name)) {
				t.Fatalf("%s.%s lost its published Symbol", class, name)
			}
		}
	}
}
