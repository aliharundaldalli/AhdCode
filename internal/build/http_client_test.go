package build

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPExamplesV06Compile(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"01_https_get.ahd", "02_custom_request.ahd", "03_json_api.ahd"} {
		entry := filepath.Join(root, "examples", "v0.6", name)
		result := Compile(entry)
		if result.HasErrors() {
			t.Fatalf("%s failed:\n%s", name, diagnosticText(result.Diagnostics))
		}
	}
}

func TestHTTPClientNativeRelocatesWithoutHelpers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-token" {
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(writer, `{"error":"auth"}`)
			return
		}
		_, _ = io.WriteString(writer, `{"answer":"4"}`)
	}))
	defer server.Close()
	source := `bring HTTP
from HTTP bring Client
from HTTP bring ClientRequest
from HTTP bring ClientResponse
bring JSON
from JSON bring JSONValue

client: Client := HTTP.client()
payload: JSONValue := JSON.object({"question": JSON.fromString("2+2")})
request: ClientRequest := HTTP.clientRequest("POST", "` + server.URL + `/v1/chat")
request = request.withHeader("Authorization", "Bearer test-token")
request = request.withHeader("Content-Type", "application/json")
request = request.withBody(JSON.stringify(payload))
response: ClientResponse := client.send(request)
parsed: JSONValue := JSON.parse(response.body())
answer: JSONValue? := parsed.get("answer")
if answer != null {
    write(answer.string())
}
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	result := Compile(filepath.Join(directory, "main.ahd"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if result.Program == nil || result.Program.RequiresSQLite {
		t.Fatal("an HTTP client program must not require the SQLite helper")
	}
	for _, file := range result.Program.Files {
		for _, banned := range []string{"ahdhttp", "ahdclient", "ahdhttps", "curl", "wget"} {
			if strings.Contains(file.Content, banned) {
				t.Fatalf("generated file %s mentions %s", file.Name, banned)
			}
		}
	}
	executable := buildSQLiteProgram(t, source)
	relocated := filepath.Join(t.TempDir(), "httpclient")
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(relocated, data, 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(relocated)
	command.Dir = filepath.Dir(relocated)
	command.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + t.TempDir()}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("relocated client failed: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "4" {
		t.Fatalf("relocated output = %q", output)
	}
}
