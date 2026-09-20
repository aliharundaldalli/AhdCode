package lsp

import (
	"strings"
	"testing"
)

// The v2.1 surface over the real LSP wire: an editor asks the server the
// same way a person's editor does, so the protocol layer is exercised and
// not only the analysis store behind it.

const v210LSPSource = `bring HTTP
bring SMTP
from HTTP bring (Client, ClientRequest, ClientFileResponse, WebSocketClient, WebSocketConnection)
from SMTP bring (SMTPMessage)

client: Client := HTTP.client()
request: ClientRequest := HTTP.clientRequest("POST", "http://127.0.0.1:9/upload")
request = request.withMultipartField("title", "report")
request = request.withMultipartFile("file", "report.pdf")
fetched: ClientFileResponse := client.download("http://127.0.0.1:9/file", "copy.bin")
socket: WebSocketClient := HTTP.webSocketClient("ws://127.0.0.1:9/ws")
live: WebSocketConnection := socket.connect()
message: SMTPMessage := SMTP.message("a@example.com", ["b@example.com"], "Subject")
write(fetched.url() + live.closeReason())
`

func TestCompletionOffersTheV210HTTPSurface(t *testing.T) {
	text := "bring HTTP\nx := HTTP.\n"
	items := completionAt(t, text, "file:///main.ahd", len(text)-1)
	for _, label := range []string{"webSocketClient", "ClientFileResponse", "WebSocketClient", "WebSocketConnection"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("expected %s among HTTP completions, got %#v", label, items)
		}
	}
	text = "from HTTP bring WebSocket\n"
	items = completionAt(t, text, "file:///main.ahd", len(text)-1)
	for _, label := range []string{"WebSocket", "WebSocketEndpoint", "WebSocketClient", "WebSocketConnection"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("expected %s among WebSocket exports, got %#v", label, items)
		}
	}
}

func TestCompletionOffersTheV210MembersOverTheWire(t *testing.T) {
	for receiver, wanted := range map[string][]string{
		"client":  {"download", "sendToFile", "send", "get", "post"},
		"request": {"withMultipartField", "withMultipartFile", "withHeader", "withBody"},
		"fetched": {"status", "header", "headerAll", "url", "size"},
		"socket":  {"withHeader", "withTimeout", "withMaxMessageBytes", "connect"},
		"live":    {"send", "receive", "close", "isOpen", "closeCode", "closeReason"},
		"message": {"withAttachment", "withText", "withHtml"},
	} {
		text := v210LSPSource + receiver + "."
		items := completionAt(t, text, "file:///main.ahd", len(text))
		for _, label := range wanted {
			if !hasCompletionLabel(items, label) {
				t.Fatalf("%s. offers no %q over the wire; got %#v", receiver, label, items)
			}
		}
	}
	// A ClientFileResponse has no body(): the payload went to the file.
	text := v210LSPSource + "fetched."
	if hasCompletionLabel(completionAt(t, text, "file:///main.ahd", len(text)), "body") {
		t.Fatal("ClientFileResponse offers body()")
	}
}

func TestHoverAndSignatureHelpForTheV210MembersOverTheWire(t *testing.T) {
	for needle, wanted := range map[string]string{
		"client.download(":           "download: (String, String) -> ClientFileResponse",
		"request.withMultipartFile(": "withMultipartFile: (String, String, String := default, String := default) -> ClientRequest",
		"socket.connect(":            "connect: () -> WebSocketConnection",
		"live.closeReason(":          "closeReason: () -> String",
		"fetched.url(":               "url: () -> String",
	} {
		receiver := needle[:strings.Index(needle, ".")]
		hover := hoverAt(t, v210LSPSource, "file:///main.ahd", needle, len(receiver)+1)
		if !strings.Contains(hover.Contents.Value, wanted) {
			t.Fatalf("hover on %s = %q, want %q", needle, hover.Contents.Value, wanted)
		}
		help, found := signatureHelpAt(t, v210LSPSource, "file:///main.ahd", needle, len(needle))
		if !found || len(help.Signatures) == 0 {
			t.Fatalf("no signature help for %s", needle)
		}
		if !strings.HasSuffix(wanted, help.Signatures[0].Label) {
			t.Fatalf("signature help for %s = %q, want the tail of %q", needle, help.Signatures[0].Label, wanted)
		}
	}
}

// The v2.0 Plot members must still answer over the wire: this is the exact
// metadata bug v2.0 fixed, rechecked beside the v2.1 surface.
func TestPlotMembersStillAnswerOverTheWire(t *testing.T) {
	text := "bring Plot\nchart := Plot.new()\nchart."
	items := completionAt(t, text, "file:///main.ahd", len(text))
	for _, label := range []string{"title", "xLabel", "yLabel", "legend", "size", "line", "scatter", "save", "show"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("Plot Chart offers no %q; got %#v", label, items)
		}
	}
	figure := "bring Plot\nfigure := Plot.subplots(1, 1, [Plot.new()])\nfigure."
	items = completionAt(t, figure, "file:///main.ahd", len(figure))
	for _, label := range []string{"save", "show"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("Plot Figure offers no %q; got %#v", label, items)
		}
	}
}
