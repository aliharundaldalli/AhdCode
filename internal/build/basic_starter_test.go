package build

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"ahdcode/internal/initweb"
)

func TestBasicStarterServesTurkishWebUIPages(t *testing.T) {
	directory := t.TempDir()
	if err := initweb.Web(directory, io.Discard, io.Discard, initweb.Options{Starter: initweb.StarterBasic, AppName: "Deneme Sitesi"}); err != nil {
		t.Fatal(err)
	}
	port := freeLoopbackPort(t)
	envPath := filepath.Join(directory, ".env")
	env, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(env), "SERVER_PORT=8080", "SERVER_PORT="+strconv.Itoa(port), 1)
	if err := os.WriteFile(envPath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	executable, result := BuildProgram(filepath.Join(directory, "app.ahd"), filepath.Join(directory, "program"))
	if result.HasErrors() {
		t.Fatalf("%s", basicStarterDiagnostics(result))
	}
	base := startBuiltHTTP(t, executable, directory, port)
	home := basicStarterGet(t, base+"/", 200)
	if !strings.Contains(home, "Ana sayfa") || !strings.Contains(home, "Deneme Sitesi") || !strings.Contains(home, "Bileşenler") {
		t.Fatalf("starter home = %s", home)
	}
	for _, markup := range []string{
		`<html lang="tr">`, `<title>Ana sayfa</title>`, `href="/hakkinda"`,
		`/assets/vendor/bootstrap/bootstrap.min.css`,
		`/assets/vendor/bootstrap/bootstrap.bundle.min.js`, `/assets/style.css`, `/assets/main.js`,
	} {
		if !strings.Contains(home, markup) {
			t.Errorf("home is missing %q", markup)
		}
	}
	for _, path := range []string{
		"vendor/bootstrap/bootstrap.min.css", "vendor/bootstrap/bootstrap.bundle.min.js",
		"style.css", "main.js", "ahdcode-logo.png",
	} {
		expected, err := os.ReadFile(filepath.Join(directory, "public", path))
		if err != nil {
			t.Fatal(err)
		}
		if actual := basicStarterGet(t, base+"/assets/"+path, 200); actual != string(expected) {
			t.Errorf("%s differs from the generated asset", path)
		}
	}
	about := basicStarterGet(t, base+"/hakkinda", 200)
	if !strings.Contains(about, "<h1>Hakkında</h1>") || !strings.Contains(about, "Deneme Sitesi") {
		t.Fatalf("starter about = %s", about)
	}
	missing := basicStarterGet(t, base+"/yok", 404)
	if !strings.Contains(missing, "Sayfa bulunamadı") {
		t.Fatalf("starter 404 = %s", missing)
	}
	failed := basicStarterGet(t, base+"/ornek-hata", 500)
	if !strings.Contains(failed, "Sunucu hatası") || !strings.Contains(failed, "Ayrıntı gösterilmez") {
		t.Fatalf("starter 500 = %s", failed)
	}
	if strings.Contains(failed, "DomainError") || strings.Contains(failed, `DomainError("ornek")`) {
		t.Fatalf("starter 500 leaked the failure: %s", failed)
	}
}

func basicStarterGet(t *testing.T, url string, status int) string {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != status {
		t.Fatalf("%s status = %d body = %s", url, response.StatusCode, body)
	}
	return string(body)
}

func basicStarterDiagnostics(result Result) string {
	var text strings.Builder
	for _, item := range result.Diagnostics {
		text.WriteString(item.Message)
		text.WriteString("\n")
	}
	return text.String()
}
