package build

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"ahdcode/internal/initweb"
)

func TestPWAIntegrationServesGeneratedBasic(t *testing.T) {
	root := t.TempDir()
	if err := initweb.Web(root, io.Discard, io.Discard, initweb.Options{Starter: initweb.StarterBasic}); err != nil {
		t.Fatal(err)
	}
	if err := initweb.PWA(root, "İstanbul Deneme", "Deneme"); err != nil {
		t.Fatal(err)
	}
	edit := func(path string, transform func(string) string) {
		t.Helper()
		file := filepath.Join(root, path)
		before, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		after := transform(string(before))
		if after == string(before) {
			t.Fatalf("fixture changed: %s", path)
		}
		if err := os.WriteFile(file, []byte(after), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	port := freeLoopbackPort(t)
	edit(".env", func(source string) string {
		return strings.Replace(source, "SERVER_PORT=8080", "SERVER_PORT="+strconv.Itoa(port), 1)
	})
	const include = "require(\"PWA/Integration.ahd\")\n"
	edit("app.ahd", func(source string) string {
		return include + strings.Replace(source, "site.start()", "ahdPwaMount(site)\nsite.start()", 1)
	})
	edit("Layouts/Main.ahd", func(source string) string {
		return include + strings.Replace(source, "Web.UI.stylesheet(\"/assets/style.css\")",
			"ahdPwaManifest()\n            ahdPwaRegistration()\n            Web.UI.stylesheet(\"/assets/style.css\")", 1)
	})
	executable, result := BuildProgram(filepath.Join(root, "app.ahd"), filepath.Join(root, "program"))
	if result.HasErrors() {
		t.Fatal(basicStarterDiagnostics(result))
	}
	base := startBuiltHTTP(t, executable, root, port)
	for _, tc := range []struct {
		url, mime, want string
		status          int
	}{
		{"/", "text/html", "/ahd-pwa.webmanifest", 200},
		{"/hakkinda", "text/html", "Hakkında", 200},
		{"/missing", "text/html", "Sayfa bulunamadı", 404},
		{"/ornek-hata", "text/html", "Sunucu hatası", 500},
		{"/ahd-pwa.webmanifest", "application/manifest+json", "İstanbul Deneme", 200},
		{"/ahdcode-sw.js", "application/javascript", "PUBLIC_ASSETS", 200},
		{"/ahd-pwa/register.js", "javascript", "serviceWorker.register", 200},
		{"/ahd-pwa/icon-192.png", "image/png", "\x89PNG", 200},
		{"/ahd-pwa/icon-512.png", "image/png", "\x89PNG", 200},
		{"/ahd-pwa/Integration.ahd", "text/plain", "", 404},
	} {
		t.Run(tc.url, func(t *testing.T) {
			response, err := http.Get(base + tc.url)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tc.status || !strings.Contains(response.Header.Get("Content-Type"), tc.mime) ||
				!bytes.Contains(body, []byte(tc.want)) {
				t.Fatalf("unexpected response to %s: %d %s", tc.url, response.StatusCode, response.Header.Get("Content-Type"))
			}
			if tc.url == "/ahdcode-sw.js" || tc.url == "/ahd-pwa.webmanifest" {
				if response.Header.Get("Cache-Control") != "no-cache" {
					t.Fatal("worker and manifest must be revalidated")
				}
			}
		})
	}
}
