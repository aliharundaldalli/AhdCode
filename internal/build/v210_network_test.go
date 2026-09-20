package build

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"ahdcode/internal/backend/golang/ahdruntime"
)

// v2.1's new APIs must behave the same through `ahdcode run` and a native
// executable, and a program that only talks to a WebSocket service must
// still build offline.

// v210BinaryFixture writes a file no String could carry: a NUL byte, bytes
// that are not valid UTF-8, and a PNG signature.
func v210BinaryFixture(t *testing.T, directory, name string) (string, []byte) {
	t.Helper()
	payload := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0xFF, 0xFE},
		bytes.Repeat([]byte{0x00, 0x7F, 0x80, 0xFF}, 700)...)
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, payload
}

func v210Digest(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// v210TransferServer is the service both halves of the parity test talk to:
// it accepts a multipart upload and serves the stored file back.
func v210TransferServer(port int) string {
	return `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring UploadedFile

stored: Pair<String, String> := {}

receive: Function := (request: Request) -> Response {
    stored: Global Pair<String, String>
    title: Local String? := request.form("title")
    upload: Local UploadedFile? := request.file("file")
    if upload == null {
        return HTTP.text("no file", 400)
    }
    path: Local String := upload.save("received")
    stored["path"] = path
    heading: Local String := "none"
    if title != null {
        heading = title
    }
    return HTTP.text(heading + "|" + upload.originalName() + "|" + str(upload.size()))
}

send: Function := (request: Request) -> Response {
    stored: Global Pair<String, String>
    if "path" not in stored {
        return HTTP.text("nothing yet", 404)
    }
    return HTTP.file(stored["path"], "application/octet-stream")
}

probe: Function := (request: Request) -> Response {
    return HTTP.text("ready")
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/__probe__", probe)
app.post("/upload", receive)
app.get("/file", send)
app.start()
`
}

// v210TransferClient uploads one file and downloads it back, printing only
// facts both execution paths must agree on.
func v210TransferClient(port int, source, destination string) string {
	return `bring HTTP
from HTTP bring Client
from HTTP bring ClientRequest
from HTTP bring ClientResponse
from HTTP bring ClientFileResponse

client: Client := HTTP.client(10)
base: String := "http://127.0.0.1:` + strconv.Itoa(port) + `"
request: ClientRequest := HTTP.clientRequest("POST", base + "/upload")
titled: ClientRequest := request.withMultipartField("title", "Ölçüm")
withFile: ClientRequest := titled.withMultipartFile("file", ` + strconv.Quote(source) + `, "sent.bin", "image/png")
answer: ClientResponse := client.send(withFile)
write("upload " + str(answer.status()) + " " + answer.body())

fetched: ClientFileResponse := client.download(base + "/file", ` + strconv.Quote(destination) + `)
write("download " + str(fetched.status()) + " " + str(fetched.size()))
contentType: String? := fetched.header("Content-Type")
if contentType != null {
    write("type " + contentType)
}
write("url " + fetched.url())
`
}

// TestV210FileTransferIsIdenticalInTheEvaluatorAndNatively is the parity
// check for the whole outbound file path: the same program, the same
// service, the same bytes on disk.
func TestV210FileTransferIsIdenticalInTheEvaluatorAndNatively(t *testing.T) {
	port := freeLoopbackPort(t)
	serverDirectory := writeSources(t, map[string]string{"main.ahd": v210TransferServer(port)})
	serverEntry := filepath.Join(serverDirectory, "main.ahd")
	serverExecutable := filepath.Join(t.TempDir(), "service")
	if _, result := BuildProgram(serverEntry, serverExecutable); result.HasErrors() {
		t.Fatalf("service build failed:\n%s", diagnosticText(result.Diagnostics))
	}
	startBuiltHTTP(t, serverExecutable, serverDirectory, port)

	workingDirectory := t.TempDir()
	source, payload := v210BinaryFixture(t, workingDirectory, "source.bin")
	expected := sha256.Sum256(payload)
	expectedDigest := hex.EncodeToString(expected[:])

	// The evaluator first.
	evaluated := filepath.Join(workingDirectory, "from-evaluator.bin")
	var output, errorOutput bytes.Buffer
	runEvaluatorIn(t, workingDirectory, v210TransferClient(port, source, evaluated), &output, &errorOutput)
	evaluatorLines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if digest := v210Digest(t, evaluated); digest != expectedDigest {
		t.Fatalf("the evaluator's download differs from the source file")
	}

	// Then the same program, compiled.
	native := filepath.Join(workingDirectory, "from-native.bin")
	clientDirectory := writeSources(t, map[string]string{"main.ahd": v210TransferClient(port, source, native)})
	clientEntry := filepath.Join(clientDirectory, "main.ahd")
	clientExecutable := filepath.Join(t.TempDir(), "client")
	if _, result := BuildProgram(clientEntry, clientExecutable); result.HasErrors() {
		t.Fatalf("client build failed:\n%s", diagnosticText(result.Diagnostics))
	}
	nativeOutput, nativeErrors, code := runIn(t, clientExecutable, workingDirectory)
	if code != 0 {
		t.Fatalf("the native client exited with %d:\n%s", code, nativeErrors)
	}
	nativeLines := strings.Split(strings.TrimSpace(nativeOutput), "\n")
	if digest := v210Digest(t, native); digest != expectedDigest {
		t.Fatalf("the native download differs from the source file")
	}

	if len(evaluatorLines) != len(nativeLines) {
		t.Fatalf("evaluator printed %d lines, native %d:\n%v\n%v",
			len(evaluatorLines), len(nativeLines), evaluatorLines, nativeLines)
	}
	for index := range evaluatorLines {
		if evaluatorLines[index] != nativeLines[index] {
			t.Fatalf("line %d: evaluator %q, native %q", index, evaluatorLines[index], nativeLines[index])
		}
	}
	// The facts the program printed are the ones the transfer actually had.
	joined := strings.Join(nativeLines, "\n")
	for _, want := range []string{
		"upload 200 Ölçüm|sent.bin|" + strconv.Itoa(len(payload)),
		"download 200 " + strconv.Itoa(len(payload)),
		"type application/octet-stream",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("output has no %q:\n%s", want, joined)
		}
	}
}

// v210WebSocketService is an AhdCode WebSocket server for the client to
// talk to.
func v210WebSocketService(port int) string {
	return `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring WebSocket
from HTTP bring WebSocketEndpoint

accept: Function := (request: Request) -> Response? {
    if request.header("X-Token") != "open-sesame" {
        return HTTP.text("no", 401)
    }
    return null
}

opened: Function := (socket: WebSocket, request: Request) -> Nothing {
    socket.send("welcome")
}

answer: Function := (socket: WebSocket, text: String) -> Nothing {
    if text == "bye" {
        socket.close(1000, "asked")
        return
    }
    socket.send("echo " + text)
}

probe: Function := (request: Request) -> Response {
    return HTTP.text("ready")
}

endpoint: WebSocketEndpoint := HTTP.websocket(answer)
endpoint = endpoint.withAccept(accept)
endpoint = endpoint.withOpen(opened)

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/__probe__", probe)
app.websocket("/live", endpoint)
app.start()
`
}

// v210WebSocketClient uses ONLY the client: no endpoint, no Server. That is
// what makes it the offline-build regression.
func v210WebSocketClient(port int) string {
	return `bring HTTP
from HTTP bring WebSocketClient
from HTTP bring WebSocketConnection

socket: WebSocketClient := HTTP.webSocketClient("ws://127.0.0.1:` + strconv.Itoa(port) + `/live")
authorized: WebSocketClient := socket.withHeader("X-Token", "open-sesame")
live: WebSocketConnection := authorized.withTimeout(5).withMaxMessageBytes(4096).connect()

greeting: String? := live.receive(5)
if greeting != null {
    write("greeting " + greeting)
}
for question in ["bir", "iki", "üç"] {
    live.send(question)
    reply: Local String? := live.receive(5)
    if reply != null {
        write("reply " + reply)
    }
}
live.send("bye")
ending: String? := live.receive(5)
if ending == null {
    code: Local Int? := live.closeCode()
    if code != null {
        write("closed " + str(code) + " " + live.closeReason())
    }
}
write("open " + str(live.isOpen()))
`
}

// TestV210WebSocketClientIsIdenticalInTheEvaluatorAndNatively also proves
// the offline build: a client-only program must pull in the vendored
// WebSocket tree and compile with an empty module cache and no network.
func TestV210WebSocketClientIsIdenticalInTheEvaluatorAndNatively(t *testing.T) {
	port := freeLoopbackPort(t)
	serviceDirectory := writeSources(t, map[string]string{"main.ahd": v210WebSocketService(port)})
	serviceExecutable := filepath.Join(t.TempDir(), "service")
	if _, result := BuildProgram(filepath.Join(serviceDirectory, "main.ahd"), serviceExecutable); result.HasErrors() {
		t.Fatalf("service build failed:\n%s", diagnosticText(result.Diagnostics))
	}
	startBuiltHTTP(t, serviceExecutable, serviceDirectory, port)

	source := v210WebSocketClient(port)

	// A client-only program must still say it needs the vendored WebSocket
	// tree, or its native build would not compile.
	clientDirectory := writeSources(t, map[string]string{"main.ahd": source})
	clientEntry := filepath.Join(clientDirectory, "main.ahd")
	compiled := Compile(clientEntry)
	if compiled.HasErrors() {
		t.Fatalf("client compilation failed:\n%s", diagnosticText(compiled.Diagnostics))
	}
	if !compiled.Program.RequiresWebSocket {
		t.Fatal("a program that uses only the WebSocket client does not request the vendored WebSocket tree")
	}
	if compiled.Program.RequiresMySQL || compiled.Program.RequiresPostgreSQL || compiled.Program.RequiresCodes {
		t.Fatalf("the client pulled in an unrelated vendor tree: %+v", compiled.Program)
	}

	var output, errorOutput bytes.Buffer
	runEvaluatorIn(t, t.TempDir(), source, &output, &errorOutput)
	evaluated := strings.TrimSpace(output.String())

	// An empty module cache is what makes this an offline build: nothing may
	// be fetched, so everything must come from the embedded vendor tree.
	t.Setenv("GOMODCACHE", t.TempDir())
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOPROXY", "off")
	clientExecutable := filepath.Join(t.TempDir(), "client")
	if _, result := BuildProgram(clientEntry, clientExecutable); result.HasErrors() {
		t.Fatalf("offline client build failed:\n%s", diagnosticText(result.Diagnostics))
	}
	nativeOutput, nativeErrors, code := runIn(t, clientExecutable, t.TempDir())
	if code != 0 {
		t.Fatalf("the native client exited with %d:\n%s", code, nativeErrors)
	}
	native := strings.TrimSpace(nativeOutput)

	if evaluated != native {
		t.Fatalf("evaluator and native disagree:\n--- evaluator ---\n%s\n--- native ---\n%s", evaluated, native)
	}
	for _, want := range []string{
		"greeting welcome", "reply echo bir", "reply echo iki", "reply echo üç",
		"closed 1000 asked", "open false",
	} {
		if !strings.Contains(native, want) {
			t.Fatalf("output has no %q:\n%s", want, native)
		}
	}
}

// A program that uses neither an endpoint nor a client must not carry the
// vendored WebSocket tree: the client must not have made it unconditional.
func TestV210ProgramWithoutWebSocketStillNeedsNoVendorTree(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "bring HTTP\nfrom HTTP bring Client\nclient: Client := HTTP.client()\nwrite(\"no socket\")\n"})
	compiled := Compile(filepath.Join(directory, "main.ahd"))
	if compiled.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(compiled.Diagnostics))
	}
	if compiled.Program.RequiresWebSocket {
		t.Fatal("a program with no WebSocket requests the vendored WebSocket tree")
	}
}

// The SMTP attachment surface must configure and fail identically on both
// paths. The message is never sent: a port nothing listens on is enough to
// show that the same configuration produces the same SMTPError.
func TestV210SMTPAttachmentBehavesIdenticallyOnBothPaths(t *testing.T) {
	directory := t.TempDir()
	attachment, _ := v210BinaryFixture(t, directory, "chart.png")
	port := freeLoopbackPort(t)
	source := `bring SMTP
from SMTP bring SMTPClient
from SMTP bring SMTPMessage
from SMTP bring SMTPError

client: SMTPClient := SMTP.client("127.0.0.1", ` + strconv.Itoa(port) + `, "none", 2)
message: SMTPMessage := SMTP.message("a@example.com", ["b@example.com"], "Rapor")
withText: SMTPMessage := message.withText("Ekte.")
withFile: SMTPMessage := withText.withAttachment(` + strconv.Quote(attachment) + `, "Grafik.png", "image/png")
attempt {
    client.send(withFile)
    write("sent")
} except SMTPError as error {
    write("failed: " + error.message)
}
missing: SMTPMessage := withText.withAttachment(` + strconv.Quote(filepath.Join(directory, "absent.png")) + `)
attempt {
    client.send(missing)
    write("sent")
} except SMTPError as error {
    write("missing: " + error.message)
}
`
	var output, errorOutput bytes.Buffer
	runEvaluatorIn(t, directory, source, &output, &errorOutput)
	evaluated := strings.TrimSpace(output.String())

	built := writeSources(t, map[string]string{"main.ahd": source})
	executable := filepath.Join(t.TempDir(), "mail")
	if _, result := BuildProgram(filepath.Join(built, "main.ahd"), executable); result.HasErrors() {
		t.Fatalf("build failed:\n%s", diagnosticText(result.Diagnostics))
	}
	nativeOutput, nativeErrors, code := runIn(t, executable, directory)
	if code != 0 {
		t.Fatalf("the native program exited with %d:\n%s", code, nativeErrors)
	}
	native := strings.TrimSpace(nativeOutput)

	if evaluated != native {
		t.Fatalf("evaluator and native disagree:\n--- evaluator ---\n%s\n--- native ---\n%s", evaluated, native)
	}
	if !strings.Contains(native, "failed: SMTP connection failed") {
		t.Fatalf("unexpected send failure:\n%s", native)
	}
	// A missing attachment is found before the connection is attempted, so
	// it reports the attachment, not the connection.
	if !strings.Contains(native, "missing: SMTP attachment file does not exist") {
		t.Fatalf("unexpected missing-attachment failure:\n%s", native)
	}
}

// The v2.1 runtime files must be in every generated program, and the
// WebSocket connection runtime must still be conditional.
func TestV210RuntimeFilesAreEmittedAsExpected(t *testing.T) {
	plain := writeSources(t, map[string]string{"main.ahd": "write(\"hi\")\n"})
	compiled := Compile(filepath.Join(plain, "main.ahd"))
	if compiled.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(compiled.Diagnostics))
	}
	names := map[string]bool{}
	for _, file := range compiled.Program.Files {
		names[file.Name] = true
	}
	for _, always := range []string{"ahdcode_http_files_runtime.go", "ahdcode_websocket_client_runtime.go"} {
		if !names[always] {
			t.Fatalf("%s is missing from a plain program", always)
		}
	}
	if names["ahdcode_websocket_conn_runtime.go"] {
		t.Fatal("a plain program received the vendored WebSocket connection runtime")
	}
	// Both always-emitted files are standard library only, so neither may
	// import a third-party module. Prose may name one; an import line may
	// not, which is why this reads the import block rather than the text.
	for name, source := range map[string]string{
		"http_files.go":       ahdruntime.HTTPFilesSource,
		"websocket_client.go": ahdruntime.WebSocketClientSource,
	} {
		for _, line := range importedPackages(t, name, source) {
			if strings.Contains(line, ".") && strings.Contains(line, "/") {
				t.Fatalf("%s imports the third-party module %s", name, line)
			}
		}
	}
}

// importedPackages lists the import paths of one runtime source file.
func importedPackages(t *testing.T, name, source string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), name, source, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("%s is not valid Go: %v", name, err)
	}
	paths := make([]string, 0, len(parsed.Imports))
	for _, entry := range parsed.Imports {
		path, err := strconv.Unquote(entry.Path.Value)
		if err != nil {
			t.Fatalf("%s has an unreadable import %s", name, entry.Path.Value)
		}
		paths = append(paths, path)
	}
	return paths
}
