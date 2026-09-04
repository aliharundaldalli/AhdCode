package ahdruntime

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeStaticFixture(t *testing.T, root string, relative string, content []byte) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestHTTPStaticServesFilesWithCorrectTypeAndBytes(t *testing.T) {
	root := t.TempDir()
	writeStaticFixture(t, root, "app.css", []byte("body { color: red; }"))
	// A tiny valid PNG (1x1 transparent pixel) as the binary fixture.
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x62, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
	writeStaticFixture(t, root, "logo.png", png)

	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets", root)
	})
	client := &http.Client{Timeout: 2 * time.Second}

	response, err := client.Get(base + "/assets/app.css")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("app.css status = %d", response.StatusCode)
	}
	if ct := response.Header.Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Fatalf("app.css content-type = %q", ct)
	}
	if string(body) != "body { color: red; }" {
		t.Fatalf("app.css body = %q", body)
	}

	response, err = client.Get(base + "/assets/logo.png")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("logo.png status = %d", response.StatusCode)
	}
	if ct := response.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("logo.png content-type = %q", ct)
	}
	if !bytes.Equal(body, png) {
		t.Fatalf("logo.png bytes were not preserved exactly: got %d bytes, want %d", len(body), len(png))
	}
}

func TestHTTPStaticMissingFileIs404(t *testing.T) {
	root := t.TempDir()
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets", root)
	})
	response, err := http.Get(base + "/assets/missing.css")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.StatusCode)
	}
}

func TestHTTPStaticTraversalRejected(t *testing.T) {
	root := t.TempDir()
	staticSubdir := filepath.Join(root, "public")
	if err := os.MkdirAll(staticSubdir, 0o755); err != nil {
		t.Fatal(err)
	}
	// secret.txt lives OUTSIDE the configured static root (root/public),
	// directly under root, so a successful traversal would read it.
	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets", staticSubdir)
	})
	client := &http.Client{
		Timeout: 2 * time.Second,
		// The traversal must be rejected at the server, not merely by a
		// client-side URL normalizer -- disable redirect-following and send
		// the raw path so a genuine escape would actually be exercised.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	for _, path := range []string{
		"/assets/../secret.txt",
		"/assets/%2e%2e/secret.txt",
		"/assets/..%2fsecret.txt",
	} {
		request, err := http.NewRequest(http.MethodGet, base+path, nil)
		if err != nil {
			t.Fatalf("%s: build request: %v", path, err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode == http.StatusOK && bytes.Contains(body, []byte("top secret")) {
			t.Fatalf("%s: traversal served the outside-root file (status %d, body %q)", path, response.StatusCode, body)
		}
	}
}

func TestHTTPStaticSymlinkEscapeRejected(t *testing.T) {
	root := t.TempDir()
	staticRoot := filepath.Join(root, "public")
	if err := os.MkdirAll(staticRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(staticRoot, "escape.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks unavailable in this environment: %v", err)
	}
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets", staticRoot)
	})
	response, err := http.Get(base + "/assets/escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode == http.StatusOK && bytes.Contains(body, []byte("top secret")) {
		t.Fatalf("symlink escape served the outside-root file (status %d, body %q)", response.StatusCode, body)
	}
}

func TestHTTPStaticDotfileRejected(t *testing.T) {
	root := t.TempDir()
	writeStaticFixture(t, root, ".env", []byte("SECRET=1"))
	writeStaticFixture(t, root, ".git/config", []byte("[core]"))
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets", root)
	})
	for _, path := range []string{"/assets/.env", "/assets/.git/config"} {
		response, err := http.Get(base + path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode == http.StatusOK {
			t.Fatalf("%s: dotfile was served (status %d, body %q)", path, response.StatusCode, body)
		}
	}
}

func TestHTTPStaticDirectoryRequestHasNoListing(t *testing.T) {
	root := t.TempDir()
	writeStaticFixture(t, root, "sub/inside.txt", []byte("inside"))
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets", root)
	})
	for _, path := range []string{"/assets/", "/assets/sub", "/assets/sub/"} {
		response, err := http.Get(base + path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode == http.StatusOK {
			t.Fatalf("%s: directory request was served instead of rejected (status %d, body %q)", path, response.StatusCode, body)
		}
		if bytes.Contains(body, []byte("inside.txt")) {
			t.Fatalf("%s: response leaked a directory listing: %q", path, body)
		}
	}
}

func TestHTTPStaticExactRouteTakesPrecedenceOverPrefix(t *testing.T) {
	root := t.TempDir()
	writeStaticFixture(t, root, "special.txt", []byte("from disk"))
	base, _ := startHTTP(t, ahdHTTPDefaultMaxBody, func(handle string) {
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets", root)
		AhdHTTPServerGet(AhdClassHTTPError, handle, "/assets/special.txt", func(string) string {
			return AhdHTTPText(AhdClassHTTPError, "from handler", 200)
		})
	})
	response, err := http.Get(base + "/assets/special.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if string(body) != "from handler" {
		t.Fatalf("exact route did not win over static prefix: body = %q", body)
	}
}

func TestHTTPStaticRegistrationValidation(t *testing.T) {
	t.Run("prefix must start with slash", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic for an invalid prefix")
			}
		}()
		handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(freeLoopbackPort(t)), ahdHTTPDefaultMaxBody)
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "assets", t.TempDir())
	})
	t.Run("root must exist", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic for a missing root")
			}
		}()
		handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(freeLoopbackPort(t)), ahdHTTPDefaultMaxBody)
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets", filepath.Join(t.TempDir(), "does-not-exist"))
	})
	t.Run("overlapping prefixes are rejected", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic for an overlapping prefix")
			}
		}()
		handle := AhdHTTPServer(AhdClassHTTPError, "127.0.0.1", int64(freeLoopbackPort(t)), ahdHTTPDefaultMaxBody)
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets", t.TempDir())
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets/nested", t.TempDir())
	})
	t.Run("cannot register after start", func(t *testing.T) {
		root := t.TempDir()
		_, handle := startHTTP(t, ahdHTTPDefaultMaxBody, func(string) {})
		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic for registering static after start")
			}
		}()
		AhdHTTPServerStatic(AhdClassHTTPError, handle, "/assets", root)
	})
}
