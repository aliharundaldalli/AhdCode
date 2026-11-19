package build

import (
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
)

// A .env file is application configuration loaded after the process starts.
// It must not be able to turn on the development reload hook, including when
// Web.start reads it and when a normal HTTP program calls Env.load.
func TestDevReloadEnvFileCannotArmAStandaloneBinary(t *testing.T) {
	env := "AHDCODE_DEV_SESSION=0123456789abcdef0123456789abcdef\nAHDCODE_DEV_GENERATION=4\nAPP_NAME=Isolation\nAPP_ENV=production\nAPP_HOST=localhost\nAPP_PROTOCOL=http\nSERVER_HOST=127.0.0.1\n"
	httpSource := func(port int) string {
		return `bring HTTP
bring Env
from HTTP bring (Server, Request, Response)

Env.load(".env", true)

home: Function := (request: Request) -> Response {
    page: Local String := "<!doctype html><html><body><h1>İstanbul İletişim</h1></body></html>"
    return HTTP.html(page, 200)
}

server: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
server.get("/", home)
server.start()
`
	}
	webSource := func(port int) string {
		return `bring Web
from Web bring (App, Request, Response)

home: Function := (request: Request) -> Response {
    return Web.html(Web.UI.p("İstanbul İletişim"), 200)
}

site: App := Web.start()
site.get("/", home)
site.start()
`
	}
	configureSource := func(port int) string {
		return `bring Web
from Web bring (App, Request, Response)

home: Function := (request: Request) -> Response {
    return Web.html(Web.UI.p("İstanbul İletişim"), 200)
}

site: App := Web.app(Web.configure())
site.get("/", home)
site.start()
`
	}
	assertBarePage := func(t *testing.T, body string) {
		t.Helper()
		if !strings.Contains(body, "İstanbul") || !strings.Contains(body, "İletişim") {
			t.Fatalf("page lost Turkish text: %s", body)
		}
		if strings.Contains(body, "/__ahdcode/dev/reload") {
			t.Fatalf(".env armed the development hook: %s", body)
		}
	}
	t.Run("http Env.load", func(t *testing.T) {
		body := runIsolatedPage(t, env, "app.ahd", httpSource)
		assertBarePage(t, body)
	})
	t.Run("Web.start", func(t *testing.T) {
		body := runIsolatedPage(t, env, "app.ahd", webSource)
		assertBarePage(t, body)
	})
	t.Run("Web.configure", func(t *testing.T) {
		body := runIsolatedPage(t, env, "app.ahd", configureSource)
		assertBarePage(t, body)
	})
}

func runIsolatedPage(t *testing.T, envText, name string, source func(int) string) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	envText += "SERVER_PORT=" + strconv.Itoa(port) + "\n"
	if err := os.WriteFile(filepath.Join(directory, ".env"), []byte(envText), 0o600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(directory, name)
	if err := os.WriteFile(entry, []byte(source(port)), 0o600); err != nil {
		t.Fatal(err)
	}
	path, result := BuildProgram(entry, filepath.Join(directory, "program"))
	if result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	command := exec.Command(path)
	command.Dir = directory
	command.Env = withoutApplicationEnvironment(os.Environ())
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
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
	url := "http://127.0.0.1:" + strconv.Itoa(port) + "/"
	deadline := time.Now().Add(20 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		response, err := http.Get(url)
		if err != nil {
			last = err
			select {
			case <-exited:
				t.Fatalf("server exited before answering\nstdout:\n%s\nstderr:\n%s\nlast: %v", stdout.String(), stderr.String(), err)
			default:
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d body = %s", response.StatusCode, body)
		}
		probe, err := http.Get(url + "__ahdcode/dev/reload")
		if err != nil {
			t.Fatal(err)
		}
		_ = probe.Body.Close()
		if probe.StatusCode != http.StatusNotFound {
			t.Fatalf("development probe status = %d", probe.StatusCode)
		}
		return string(body)
	}
	t.Fatalf("server did not answer: %v\nstdout:\n%s\nstderr:\n%s", last, stdout.String(), stderr.String())
	return ""
}

func withoutApplicationEnvironment(environ []string) []string {
	filtered := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(key) {
		case "AHDCODE_DEV_SESSION", "AHDCODE_DEV_GENERATION", "AHDCODE_DEV_RELOAD",
			"SERVER_PORT", "SERVER_HOST", "APP_NAME", "APP_ENV", "APP_HOST", "APP_PROTOCOL":
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
