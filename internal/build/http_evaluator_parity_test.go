package build

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/evaluator"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
)

// A compiled program and the evaluator serve the same GET. The compiled
// process is stopped by ending it; the evaluator returns from Server.start
// after the host shuts the listener down.
func TestHTTPNativeAndEvaluatorAgreeOnAGET(t *testing.T) {
	nativePort := freeHTTPParityPort(t)
	evaluatorPort := freeHTTPParityPort(t)
	program := func(port int) string {
		return `bring HTTP
from HTTP bring (Server, Request, Response)

home: Function := (request: Request) -> Response {
    return HTTP.text("parity İstanbul")
}

server: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
server.get("/parity", home)
server.start()
`
	}
	nativeStatus, nativeBody, nativeType := runNativeHTTPGET(t, program(nativePort), nativePort)
	evalStatus, evalBody, evalType := runEvaluatorHTTPGET(t, program(evaluatorPort), evaluatorPort)
	if nativeStatus != 200 || evalStatus != 200 || nativeBody != "parity İstanbul" || evalBody != nativeBody || evalType != nativeType {
		t.Fatalf("native %d %q %q; evaluator %d %q %q", nativeStatus, nativeBody, nativeType, evalStatus, evalBody, evalType)
	}
}

func runEvaluatorHTTPGET(t *testing.T, source string, port int) (int, string, string) {
	t.Helper()
	workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": source})
	frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
	if frontend.HasErrors() {
		t.Fatalf("frontend diagnostics: %+v", frontend.Diagnostics)
	}
	lowered := lowering.LowerCompilation(frontend)
	if lowered.HasErrors() {
		t.Fatalf("lowering diagnostics: %+v", lowered.Diagnostics)
	}
	session := evaluator.New(bufio.NewReader(strings.NewReader("")), io.Discard, t.TempDir())
	done := make(chan evaluator.ExecutionResult, 1)
	go func() {
		done <- session.Execute(lowered.Compilation, 0)
	}()
	t.Cleanup(func() {
		_ = ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", port)
	})
	status, body, contentType := waitHTTPGET(t, "http://127.0.0.1:"+strconv.Itoa(port)+"/parity")
	if !ahdruntime.AhdHTTPShutdownAddress("127.0.0.1", port) {
		t.Fatal("evaluator server did not shut down")
	}
	select {
	case result := <-done:
		if result.Failure != nil {
			t.Fatalf("evaluator failed after shutdown: %s", result.Failure.Message)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("evaluator Server.start did not return")
	}
	return status, body, contentType
}

func runNativeHTTPGET(t *testing.T, source string, port int) (int, string, string) {
	t.Helper()
	directory := t.TempDir()
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	path, result := BuildProgram(entry, filepath.Join(directory, "program"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	command := exec.Command(path)
	command.Dir = directory
	command.Env = withoutApplicationEnvironment(os.Environ())
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() {
		_ = command.Wait()
		close(exited)
	}()
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		<-exited
	})
	status, body, contentType := waitHTTPGET(t, "http://127.0.0.1:"+strconv.Itoa(port)+"/parity")
	if command.Process != nil {
		_ = command.Process.Kill()
	}
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("native server did not exit")
	}
	return status, body, contentType
}

func waitHTTPGET(t *testing.T, url string) (int, string, string) {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		response, err := client.Get(url)
		if err != nil {
			last = err
			time.Sleep(20 * time.Millisecond)
			continue
		}
		payload, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, string(payload), response.Header.Get("Content-Type")
	}
	t.Fatalf("GET %s did not answer: %v", url, last)
	return 0, "", ""
}

func freeHTTPParityPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}
