package ahdruntime

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
