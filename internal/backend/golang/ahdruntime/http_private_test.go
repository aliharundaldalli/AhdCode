package ahdruntime

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestHTTPPublicMountKeepsRegisteredRootAndBlocksHiddenTargets(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "public")
	for path, text := range map[string]string{"index.txt": "public", ".hidden/index.txt": "hidden"} {
		name := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(root, "alias")
	hidden := filepath.Join(root, ".hidden")
	if runtime.GOOS == "windows" {
		if output, err := exec.Command("cmd", "/c", "mklink", "/J", alias, hidden).CombinedOutput(); err != nil {
			t.Fatalf("create fixture junction: %v\n%s", err, output)
		}
	} else if err := os.Symlink(hidden, alias); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(alias) })
	t.Cleanup(ahdHTMLResetExposedAssets)
	AhdHTMLAsset(AhdClassHTMLError, "script", "index.txt")
	AhdHTMLAsset(AhdClassHTMLError, "script", "alias/index.txt")
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/static", root)
		AhdHTTPServerManaged(AhdClassHTMLError, handle, "/managed", root)
	})
	client := &http.Client{Timeout: 2 * time.Second}
	expect := func(path string, status int, content string) {
		t.Helper()
		response, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != status || (content != "" && string(body) != content) {
			t.Fatalf("%s: status %d, body %q, error %v", path, response.StatusCode, body, err)
		}
	}
	for _, prefix := range []string{"/static", "/managed"} {
		expect(prefix+"/index.txt", 200, "public")
		expect(prefix+"/alias/index.txt", 404, "")
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, filepath.Join(parent, "former-public")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.txt"), []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	expect("/static/index.txt", 404, "")
	expect("/managed/index.txt", 404, "")
}

func TestHTTPExplicitFileCachePolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/default", func(string) string {
			return AhdHTTPFile(AhdClassHTTPError, path, "text/plain")
		})
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/public", func(string) string {
			response := AhdHTTPFile(AhdClassHTTPError, path, "text/plain")
			return AhdHTTPResponseWithHeader(AhdClassHTTPError, response, "Cache-Control", "public, max-age=60")
		})
	})
	for route, want := range map[string]string{"/default": "no-store", "/public": "public, max-age=60"} {
		response, err := http.Get(base + route)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != 200 || response.Header.Get("Cache-Control") != want {
			t.Fatalf("%s status=%d cache=%q", route, response.StatusCode, response.Header.Get("Cache-Control"))
		}
	}
}

// Private application files are served only through an explicit HTTP.file
// (or HTTP.download) response after the application has authorized the
// request. They are not mounted, listed, or reachable by guessing a URL
// under a static or managed prefix.
func TestHTTPFileIsExplicitAndDoesNotEnumerate(t *testing.T) {
	root := t.TempDir()
	private := filepath.Join(root, "storage", "private")
	if err := os.MkdirAll(private, 0o755); err != nil {
		t.Fatal(err)
	}
	welcome := filepath.Join(private, "welcome.txt")
	if err := os.WriteFile(welcome, []byte("secret-welcome"), 0o600); err != nil {
		t.Fatal(err)
	}
	if AhdHTTPFile(AhdClassHTTPError, welcome, "text/plain") == "" {
		t.Fatal("HTTP.file produced no response")
	}

	public := filepath.Join(root, "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ahdHTMLResetExposedAssets)
	AhdHTMLAsset(AhdClassHTMLError, "stylesheet", "style.css")
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerManaged(AhdClassHTMLError, handle, "/assets", public)
	})
	client := &http.Client{Timeout: 2 * time.Second}
	for _, path := range []string{
		"/assets/../storage/private/welcome.txt",
		"/storage/private/welcome.txt",
		"/assets/welcome.txt",
	} {
		response, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode == 200 {
			t.Fatalf("%s was publicly reachable", path)
		}
	}
}
