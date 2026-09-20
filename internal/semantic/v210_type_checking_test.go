package semantic

import (
	"strings"
	"testing"
)

// v2.1's new surface is checked the same way every other built-in member is:
// at compile time, with the ordinary AhdCode diagnostics. These are the
// mistakes a program can make with it, and each one must be a compile error
// rather than a runtime surprise.

const v210Preamble = `bring HTTP
bring SMTP
from HTTP bring (Client, ClientRequest, ClientResponse, ClientFileResponse, WebSocketClient, WebSocketConnection, HTTPError)
from SMTP bring (SMTPClient, SMTPMessage, SMTPError)

client: Client := HTTP.client()
request: ClientRequest := HTTP.clientRequest("POST", "http://127.0.0.1:9/upload")
socket: WebSocketClient := HTTP.webSocketClient("ws://127.0.0.1:9/ws")
message: SMTPMessage := SMTP.message("a@example.com", ["b@example.com"], "Subject")
paths: List<String> := ["report.pdf"]

`

func TestV210SurfaceAcceptsCorrectPrograms(t *testing.T) {
	requireSemanticClean(t, analyzeWithStandardModules(t, v210Preamble+`upload: ClientRequest := request.withMultipartField("title", "report").withMultipartFile("file", "report.pdf")
detailed: ClientRequest := upload.withMultipartFile("scan", "scan.png", "scan.png", "image/png")
saved: ClientFileResponse := client.sendToFile(detailed, "answer.bin")
fetched: ClientFileResponse := client.download("http://127.0.0.1:9/file", "copy.bin")
status: Int := fetched.status()
size: Int := fetched.size()
finalURL: String := fetched.url()
contentType: String? := fetched.header("Content-Type")
cookies: List<String> := fetched.headerAll("Set-Cookie")

live: WebSocketConnection := socket.withHeader("Authorization", "Bearer x").withTimeout(5).withMaxMessageBytes(4096).connect()
delivered: Bool := live.send("hello")
waited: String? := live.receive()
bounded: String? := live.receive(3)
open: Bool := live.isOpen()
code: Int? := live.closeCode()
reason: String := live.closeReason()
live.close()
live.close(1001)
live.close(1001, "going away")

withFile: SMTPMessage := message.withText("see attached").withAttachment("report.pdf")
named: SMTPMessage := withFile.withAttachment("scan.png", "Ölçüm.png", "image/png")

attempt {
    client.download("http://127.0.0.1:9/file", "copy.bin")
} except HTTPError as error {
    write(error.message)
}
attempt {
    SMTP.client("localhost", 25).send(named)
} except SMTPError as error {
    write(error.message)
}
total: Int := status + size + saved.status()
flags: Bool := delivered and open
write(finalURL + reason)
for cookie in cookies {
    write(cookie)
}
if contentType != null {
    write(contentType)
}
if waited != null {
    write(waited)
}
if bounded != null {
    write(bounded)
}
if code != null {
    total = total + code
}
if flags {
    write("closed after {total} bytes of metadata")
}
`))
}

func TestV210SurfaceRejectsStaticMistakes(t *testing.T) {
	for _, source := range []string{
		// A List where a String path belongs.
		`client.download("http://127.0.0.1:9/f", paths)`,
		`client.download(paths, "copy.bin")`,
		`request.withMultipartFile("file", paths)`,
		`message.withAttachment(paths)`,
		// Wrong arity.
		`client.download("http://127.0.0.1:9/f")`,
		`client.download("http://127.0.0.1:9/f", "copy.bin", "extra")`,
		`client.sendToFile(request)`,
		`request.withMultipartField("title")`,
		`request.withMultipartField("title", "a", "b")`,
		`request.withMultipartFile("file", "report.pdf", "r.pdf", "application/pdf", "extra")`,
		`message.withAttachment("report.pdf", "r.pdf", "application/pdf", "extra")`,
		// Wrong argument types.
		`request.withMultipartField("title", 1)`,
		`request.withMultipartFile(1, "report.pdf")`,
		`request.withMultipartFile("file", "report.pdf", "r.pdf", 1)`,
		`message.withAttachment("report.pdf", 1)`,
		`socket.withTimeout("5")`,
		`socket.withTimeout(5.0)`,
		`socket.withMaxMessageBytes("4096")`,
		`socket.withHeader("Authorization")`,
		`live: WebSocketConnection := socket.connect()
live.receive("1")`,
		`live: WebSocketConnection := socket.connect()
live.send(1)`,
		`live: WebSocketConnection := socket.connect()
live.send(paths)`,
		`live: WebSocketConnection := socket.connect()
live.close("1000")`,
		// A ClientFileResponse has no body: the payload is in the file.
		`fetched: ClientFileResponse := client.download("http://127.0.0.1:9/f", "copy.bin")
text: String := fetched.body()`,
		// Configuration is not a connection, and a connection is not
		// configuration.
		`socket.receive()`,
		`socket.send("hi")`,
		`socket.close()`,
		`live: WebSocketConnection := socket.connect()
live.connect()`,
		`live: WebSocketConnection := socket.connect()
live.withTimeout(5)`,
		// A ClientResponse is not a ClientFileResponse.
		`plain: ClientResponse := client.download("http://127.0.0.1:9/f", "copy.bin")`,
		`fileResponse: ClientFileResponse := client.get("http://127.0.0.1:9/f")`,
		// A ClientRequest is not a WebSocketClient.
		`wrong: WebSocketClient := HTTP.clientRequest("GET", "http://127.0.0.1:9/f")`,
		`wrong: ClientRequest := HTTP.webSocketClient("ws://127.0.0.1:9/ws")`,
		// A member call still refuses a named argument, exactly like every
		// other built-in type operation.
		`request.withMultipartField(name: "title", value: "report")`,
		`client.download(url: "http://127.0.0.1:9/f", path: "copy.bin")`,
		// Results have the types they have.
		`sent: String := socket.connect().send("hi")`,
		`heard: String := socket.connect().receive()`,
		`code: Int := socket.connect().closeCode()`,
		`size: String := client.download("http://127.0.0.1:9/f", "copy.bin").size()`,
		// No member AhdCode does not have.
		`client.downloadBytes("http://127.0.0.1:9/f")`,
		`request.withMultipartBytes("file", "data")`,
		`socket.connect().sendBinary("hi")`,
		`socket.connect().onMessage("hi")`,
		`socket.withInsecureTLS(true)`,
		`message.withAttachmentBytes("data")`,
	} {
		program := source
		result := analyzeWithStandardModules(t, v210Preamble+program+"\n")
		requireSemanticFailure(t, result)
	}
}

// The three opaque v2.1 types are produced by the runtime, never built by a
// program, and each says how to obtain one.
func TestV210OpaqueValuesAreNotConstructedDirectly(t *testing.T) {
	for source, hint := range map[string]string{
		`ClientFileResponse("x")`:  "ClientFileResponse values are produced by Client.download or Client.sendToFile",
		`WebSocketClient("1")`:     "create a WebSocketClient with HTTP.webSocketClient(url)",
		`WebSocketConnection("1")`: "WebSocketConnection values are produced by WebSocketClient.connect",
	} {
		result := analyzeWithStandardModules(t, v210Preamble+source+"\n")
		requireSemanticFailure(t, result)
		if !strings.Contains(semanticHintsOf(result), hint) {
			t.Fatalf("%s has no construction hint: %q", source, semanticHintsOf(result))
		}
	}
}

// The v2.1 surface is exactly this and no more: no Bytes, no stream, no
// binary WebSocket, no reconnect, no second error class.
func TestV210SurfaceIsExactlyTheIntendedOne(t *testing.T) {
	if got := strings.Join(HTTPClientOperations, ","); got != "send,get,post,download,sendToFile" {
		t.Fatalf("Client operations = %s", got)
	}
	if got := strings.Join(HTTPClientRequestOperations, ","); got != "withHeader,addHeader,withBody,withMultipartField,withMultipartFile" {
		t.Fatalf("ClientRequest operations = %s", got)
	}
	if got := strings.Join(HTTPClientFileResponseOperations, ","); got != "status,header,headerAll,url,size" {
		t.Fatalf("ClientFileResponse operations = %s", got)
	}
	if got := strings.Join(HTTPWebSocketClientOperations, ","); got != "withHeader,withTimeout,withMaxMessageBytes,connect" {
		t.Fatalf("WebSocketClient operations = %s", got)
	}
	if got := strings.Join(HTTPWebSocketConnectionOperations, ","); got != "send,receive,close,isOpen,closeCode,closeReason" {
		t.Fatalf("WebSocketConnection operations = %s", got)
	}
	if got := strings.Join(SMTPMessageOperations, ","); got != "withCc,withBcc,withReplyTo,withText,withHtml,withAttachment" {
		t.Fatalf("SMTPMessage operations = %s", got)
	}
	// No new error class: HTTP failures stay HTTPError and mail failures
	// stay SMTPError.
	httpModule := StandardModuleInterfaces()["HTTP"]
	for _, name := range []string{"DownloadError", "MultipartError", "WebSocketError", "Bytes", "Buffer", "Stream"} {
		if httpModule.Exports[name] != nil {
			t.Fatalf("HTTP must not export %q", name)
		}
	}
	smtpModule := StandardModuleInterfaces()["SMTP"]
	for _, name := range []string{"AttachmentError", "Bytes", "inbox", "receive"} {
		if smtpModule.Exports[name] != nil {
			t.Fatalf("SMTP must not export %q", name)
		}
	}
}
