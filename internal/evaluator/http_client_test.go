package evaluator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestHTTPEvaluatorClientTalksToLocalServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/echo":
			body, _ := io.ReadAll(request.Body)
			writer.Header().Set("X-Seen", request.Header.Get("X-App"))
			_, _ = writer.Write(body)
		case "/missing":
			writer.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(writer, "no")
		default:
			_, _ = io.WriteString(writer, "hello")
		}
	}))
	defer server.Close()
	source := `bring HTTP
from HTTP bring Client
from HTTP bring ClientRequest
from HTTP bring ClientResponse

client: Client := HTTP.client()
page: ClientResponse := client.get("` + server.URL + `/ok")
write(page.body())
write(str(page.status()))
request: ClientRequest := HTTP.clientRequest("POST", "` + server.URL + `/echo")
request = request.withHeader("X-App", "AhdCode")
request = request.withHeader("Content-Type", "text/plain")
request = request.withBody("ping")
echo: ClientResponse := client.send(request)
write(echo.body())
missing: ClientResponse := client.get("` + server.URL + `/missing")
write(str(missing.status()))
write(missing.body())
`
	var output bytes.Buffer
	session := New(bufio.NewReader(strings.NewReader("")), &output, "")
	if result := session.Execute(compileAhd(t, source), 0); result.Failure != nil {
		t.Fatalf("execute: %v", result.Failure)
	}
	got := output.String()
	for _, want := range []string{"hello", "200", "ping", "404", "no"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestHTTPEvaluatorJSONAPIMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/chat" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if request.Header.Get("Authorization") != "Bearer test-token" {
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(writer, `{"error":"auth"}`)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		if payload["question"] != "2+2" {
			t.Errorf("question = %#v", payload["question"])
		}
		writer.Header().Set("Content-Type", "application/json")
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
token: String := "test-token"
payload: JSONValue := JSON.object({"question": JSON.fromString("2+2")})
body: String := JSON.stringify(payload)
request: ClientRequest := HTTP.clientRequest("POST", "` + server.URL + `/v1/chat")
request = request.withHeader("Authorization", "Bearer {token}")
request = request.withHeader("Content-Type", "application/json")
request = request.withBody(body)
response: ClientResponse := client.send(request)
parsed: JSONValue := JSON.parse(response.body())
answer: JSONValue? := parsed.get("answer")
if answer == null {
    write("missing")
} else {
    write(answer.string())
}
`
	var output bytes.Buffer
	session := New(bufio.NewReader(strings.NewReader("")), &output, "")
	if result := session.Execute(compileAhd(t, source), 0); result.Failure != nil {
		t.Fatalf("execute: %v", result.Failure)
	}
	if strings.TrimSpace(output.String()) != "4" {
		t.Fatalf("answer = %q", output.String())
	}
}

func TestHTTPEvaluatorClientSecretIsRedacted(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(writer, "no")
	}))
	defer server.Close()
	source := `bring HTTP
from HTTP bring Client
from HTTP bring ClientRequest
from HTTP bring HTTPError

client: Client := HTTP.client(2)
request: ClientRequest := HTTP.clientRequest("GET", "` + server.URL + `")
request = request.withHeader("Authorization", "Bearer SUPER_SECRET_TEST_VALUE")
attempt {
    client.send(request)
} except HTTPError as error {
    write(error.message)
}
`
	var output bytes.Buffer
	session := New(bufio.NewReader(strings.NewReader("")), &output, "")
	result := session.Execute(compileAhd(t, source), 0)
	if result.Failure != nil {
		t.Fatalf("execute: %v", result.Failure)
	}
	got := output.String()
	if strings.Contains(got, "SUPER_SECRET_TEST_VALUE") {
		t.Fatalf("secret leaked:\n%s", got)
	}
	if !strings.Contains(got, "TLS") && !strings.Contains(got, "failed") && !strings.Contains(got, "HTTP") {
		t.Fatalf("expected a transport error, got %q", got)
	}
}

func TestHTTPEvaluatorServerAndSessionSmoke(t *testing.T) {
	port := freeEvaluatorPort(t)
	source := `bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from HTTP bring SessionStore
from HTTP bring Session

sessions: SessionStore := HTTP.sessions()

ok: Function := (request: Request) -> Response {
    return HTTP.text("ok")
}

echo: Function := (request: Request) -> Response {
    return HTTP.text(request.body())
}

count: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    raw: Local String? := session.get("n")
    value: Local Int := 0
    if raw != null {
        value = int(raw)
    }
    value = value + 1
    session.set("n", str(value))
    return sessions.commit(session, HTTP.text(str(value)))
}

boom: Function := (request: Request) -> Response {
    toss (DomainError("boom"))
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/ok", ok)
app.post("/echo", echo)
app.get("/count", count)
app.get("/boom", boom)
app.start()
`
	base := startEvaluatorHTTP(t, source, port)
	get := func(path string) (int, string) {
		t.Helper()
		response, err := http.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		return response.StatusCode, string(body)
	}
	if status, body := get("/ok"); status != 200 || body != "ok" {
		t.Fatalf("GET /ok = %d %q", status, body)
	}
	response, err := http.Post(base+"/echo", "text/plain", strings.NewReader("hi"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 || string(body) != "hi" {
		t.Fatalf("POST /echo = %d %q", response.StatusCode, body)
	}
	jar, _ := cookiejar.New(nil)
	browser := &http.Client{Jar: jar}
	first, err := browser.Get(base + "/count")
	if err != nil {
		t.Fatal(err)
	}
	firstBody, _ := io.ReadAll(first.Body)
	_ = first.Body.Close()
	second, err := browser.Get(base + "/count")
	if err != nil {
		t.Fatal(err)
	}
	secondBody, _ := io.ReadAll(second.Body)
	_ = second.Body.Close()
	if string(firstBody) != "1" || string(secondBody) != "2" {
		t.Fatalf("session = %q then %q", firstBody, secondBody)
	}
	if status, body := get("/ok?q=%ZZ"); status != 400 || strings.Contains(body, "\uFFFD") {
		t.Fatalf("malformed query = %d %q", status, body)
	}
	if status, _ := get("/boom"); status != 500 {
		t.Fatalf("handler error = %d", status)
	}
	if status, body := get("/ok"); status != 200 || body != "ok" {
		t.Fatalf("server died: %d %q", status, body)
	}
}
