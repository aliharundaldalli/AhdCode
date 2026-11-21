package packaging

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"ahdcode/internal/initweb"
)

func TestWebPackageRelocatedPWA(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "Türkçe proje")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := initweb.Web(project, io.Discard, io.Discard, initweb.Options{Starter: initweb.StarterBasic}); err != nil {
		t.Fatal(err)
	}
	if err := initweb.PWA(project, "İstanbul", "İstanbul"); err != nil {
		t.Fatal(err)
	}
	starter := filepath.Join(project, "app.ahd")
	update := func(path, before, after string) {
		t.Helper()
		body, err := os.ReadFile(path)
		if err != nil || !bytes.Contains(body, []byte(before)) {
			t.Fatalf("cannot edit fixture %s: %v", path, err)
		}
		withPWA := append([]byte("require(\"PWA/Integration.ahd\")\n"),
			bytes.Replace(body, []byte(before), []byte(after), 1)...)
		if err := os.WriteFile(path, withPWA, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	update(starter, "site.start()", "ahdPwaMount(site)\nsite.start()")
	update(filepath.Join(project, "Layouts", "Main.ahd"), "Web.UI.stylesheet(\"/assets/style.css\")",
		"ahdPwaManifest()\n            ahdPwaRegistration()\n            Web.UI.stylesheet(\"/assets/style.css\")")
	name := "Türkçe Yayın"
	opts := Options{Entry: starter, Output: filepath.Join(root, "çıktı paket"),
		Name: name, Web: true, PWA: true}
	result := Package(opts)
	if result.Application == "" {
		t.Fatalf("failed to package: %+v", result.Diagnostics)
	}
	if runtime.GOOS == "windows" {
		archive, err := zip.OpenReader(result.Archive)
		if err != nil {
			t.Fatal(err)
		}
		var paths []string
		for _, file := range archive.File {
			paths = append(paths, strings.TrimPrefix(file.Name, name+"/"))
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"public/style.css", "PWA/worker.js", "PWA/app.webmanifest", "PWA/public/icon-512.png"} {
			found := false
			for _, member := range paths {
				found = found || member == path
			}
			if !found {
				t.Errorf("archive missing %s", path)
			}
		}
		for _, member := range paths {
			if forbiddenNames.MatchString(filepath.Base(member)) {
				t.Errorf("private or source file in release: %s", member)
			}
		}
	}
	if again := Package(opts); again.Application != "" || len(again.Diagnostics) == 0 {
		t.Fatal("a Web package could overwrite its deployment")
	}

	deployment := filepath.Join(root, "kopyalanmış paket")
	if err := os.Rename(result.Application, deployment); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(project, filepath.Join(root, "relocated source")); err != nil {
		t.Fatal(err)
	}
	platform := Targets[HostTarget()]
	layout := layoutFor(name, platform)
	dir := filepath.Join(deployment, layout.program)
	for _, excluded := range []string{".env", "app.ahd", "Config", "PWA/Integration.ahd", "database"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(excluded))); !os.IsNotExist(err) {
			t.Fatalf("private path %s: %v", excluded, err)
		}
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + listener.Addr().String()
	port := strings.Split(listener.Addr().String(), ":")[1]
	listener.Close()
	cmd := exec.Command(filepath.Join(dir, layout.executable))
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "SERVER_HOST=127.0.0.1", "SERVER_PORT="+port,
		"APP_ENV=production", "APP_PROTOCOL=https", "APP_NAME=Türkçe Yayın", "APP_HOST=localhost",
		"AHDCODE_DEV_SESSION=", "AHDCODE_DEV_GENERATION=", "AHDCODE_DEV_TLS_CERT=", "AHDCODE_DEV_TLS_KEY=")
	out, err := os.Create(filepath.Join(root, "server.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(9 * time.Second)
	for {
		response, err := client.Get(base + "/")
		if err == nil {
			_ = response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("relocated package did not start: %v", err)
		}
		time.Sleep(30 * time.Millisecond)
	}
	for _, tc := range []struct {
		method, path, mime string
		status             int
	}{
		{"GET", "/", "text/html", 200},
		{"GET", "/assets/style.css", "text/css", 200},
		{"GET", "/assets/vendor/bootstrap/bootstrap.bundle.min.js", "javascript", 200},
		{"GET", "/ahd-pwa.webmanifest", "application/manifest+json", 200},
		{"HEAD", "/ahd-pwa.webmanifest", "application/manifest+json", 200},
		{"HEAD", "/ahdcode-sw.js", "application/javascript", 200},
		{"GET", "/ahd-pwa/icon-512.png", "image/png", 200},
		{"GET", "/assets/%2e%2e/PWA/worker.js", "", 404},
		{"GET", "/assets/.env", "", 404},
		{"GET", "/PWA/Integration.ahd", "", 404},
		{"GET", "/ornek-hata", "text/html", 500},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			request, err := http.NewRequest(tc.method, base+tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != tc.status || !strings.Contains(response.Header.Get("Content-Type"), tc.mime) {
				t.Fatalf("%d %s: %v", response.StatusCode, response.Header.Get("Content-Type"), err)
			}
			if tc.method == "HEAD" && (len(body) != 0 || response.Header.Get("Cache-Control") != "no-cache") {
				t.Fatal("bad HEAD body or cache policy")
			}
			if tc.path == "/" && !bytes.Contains(body, []byte("/ahd-pwa.webmanifest")) {
				t.Fatal("packaged Basic omitted its manifest link")
			}
		})
	}
}

func TestWebAssetsRejectUnsafeFiles(t *testing.T) {
	for _, tc := range []struct{ file, content, want string }{
		{"public/.env", "test", "approved public path"},
		{"public/sessions.db", "test", "approved public path"},
		{"public/private/styles.css", "test", "private directory"},
		{"public/config.json", "test", "supported public asset type"},
		{"public/Sécurité.js", "-----BEGIN PRIVATE KEY-----", "PEM private key"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			project := t.TempDir()
			name := filepath.Join(project, filepath.FromSlash(tc.file))
			if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(name, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := copyWebAssets(filepath.Join(project, "app.ahd"), t.TempDir(), ".", false)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %q: %v", tc.want, err)
			}
		})
	}
	if validWebComponent("CON.css") || validWebComponent("CON .css") || validWebComponent("file.js:secret") {
		t.Fatal("a Windows device or alternate stream name was accepted")
	}
}

func TestWebAssetsRejectCaseCollisions(t *testing.T) {
	project := t.TempDir()
	public := filepath.Join(project, "public")
	if err := os.Mkdir(public, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "Site.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	lowercase := filepath.Join(public, "site.css")
	if _, err := os.Stat(lowercase); err == nil {
		t.Skip("fixture requires a case-sensitive filesystem")
	}
	if err := os.WriteFile(lowercase, []byte("main{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := copyWebAssets(filepath.Join(project, "app.ahd"), t.TempDir(), ".", false); err == nil ||
		!strings.Contains(err.Error(), "collide on case-insensitive systems") {
		t.Fatalf("wanted a case collision: %v", err)
	}
}

func TestWebPWARejectsMissingMalformedAndExternalAssets(t *testing.T) {
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(project, "app.ahd")
	if _, err := copyWebAssets(entry, t.TempDir(), ".", true); err == nil {
		t.Fatal("missing PWA was accepted")
	}
	if err := initweb.PWA(project, "QA", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "PWA", "public", "empty.png"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		"{", `{"name":"QA","icons":[{"src":"https://example.test/icon.png"}]}`,
		`{"name":"QA","icons":[{"src":"/ahd-pwa/../icon.png"}]}`,
		`{"name":"QA","icons":[{"src":"/ahd-pwa/missing.png"}]}`,
		`{"name":"QA","icons":[{"src":"/ahd-pwa/register.js"}]}`,
		`{"name":"QA","icons":[{"src":"/ahd-pwa/empty.png"}]}`,
	} {
		if err := os.WriteFile(filepath.Join(project, "PWA", "app.webmanifest"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := copyWebAssets(entry, t.TempDir(), ".", true); err == nil {
			t.Errorf("bad manifest accepted: %s", body)
		}
	}
}

func TestWebPackagePublishDoesNotOverwriteArchiveOrDeployment(t *testing.T) {
	root := t.TempDir()
	final := filepath.Join(root, "existing")
	staged := filepath.Join(root, "stage", "app")
	for _, dir := range []string{staged, filepath.Join(final, "runtime")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	id, _ := json.Marshal(appIdentity{Name: "existing", Web: true})
	if err := os.WriteFile(filepath.Join(final, "runtime", identityFile), id, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := replace(final, staged); err == nil {
		t.Fatal("desktop package could erase an existing Web deployment")
	}
	const archive = "hold.zip"
	for _, path := range []string{filepath.Join(root, archive), filepath.Join(filepath.Dir(staged), archive),
		filepath.Join(staged, "test.exe")} {
		if err := os.WriteFile(path, []byte(path), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	destination := filepath.Join(root, "reserved")
	if err := publishWeb(staged, destination, archive); err == nil {
		t.Fatal("existing archive was overwritten")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("an incomplete deployment remains: %v", err)
	}
	existing, err := os.ReadFile(filepath.Join(root, archive))
	if err != nil || string(existing) != filepath.Join(root, archive) {
		t.Fatal("existing archive was changed")
	}
}

func TestWebAssetsRejectJunctionOrSymlink(t *testing.T) {
	project, private := t.TempDir(), t.TempDir()
	public := filepath.Join(project, "public")
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", public, private).CombinedOutput(); err != nil {
			t.Skipf("junction unavailable: %v: %s", err, out)
		}
	} else if err := os.Symlink(private, public); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(public) })
	if _, err := copyWebAssets(filepath.Join(project, "app.ahd"), t.TempDir(), ".", false); err == nil {
		t.Fatal("linked public root was accepted")
	}
}
