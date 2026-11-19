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

func TestTemplateProgramRendersLayoutPartialAndErrors(t *testing.T) {
	directory := t.TempDir()
	port := freeLoopbackPort(t)
	entry := filepath.Join(directory, "app.ahd")
	if err := os.WriteFile(entry, []byte(templateProgram(port)), 0o644); err != nil {
		t.Fatal(err)
	}
	executable, result := BuildProgram(entry, filepath.Join(directory, "program"))
	if result.HasErrors() {
		t.Fatalf("%s", templateDiagnostics(result))
	}
	base := startBuiltHTTP(t, executable, directory, port)
	body := templateGet(t, base+"/", 200)
	if !strings.Contains(body, "Ana sayfa") || !strings.Contains(body, "İstanbul") || !strings.Contains(body, "Şablonlar") {
		t.Fatalf("home = %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") || strings.Contains(body, "<script>alert") {
		t.Fatalf("escaping = %s", body)
	}
	if !strings.Contains(body, "<b>güvenli</b>") {
		t.Fatalf("trusted html = %s", body)
	}
	about := templateGet(t, base+"/hakkinda", 200)
	if !strings.Contains(about, "Hakkında") || strings.Contains(about, "Ana sayfa") {
		t.Fatalf("about = %s", about)
	}
	missing := templateGet(t, base+"/yok", 404)
	if !strings.Contains(missing, "Sayfa bulunamadı") || strings.Contains(missing, directory) {
		t.Fatalf("404 = %s", missing)
	}
	failed := templateGet(t, base+"/hata", 500)
	if strings.TrimSpace(failed) != "Internal Server Error" || strings.Contains(failed, directory) {
		t.Fatalf("500 fallback = %q", failed)
	}
}

func TestBasicStarterServesTurkishTemplates(t *testing.T) {
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
		t.Fatalf("%s", templateDiagnostics(result))
	}
	base := startBuiltHTTP(t, executable, directory, port)
	home := templateGet(t, base+"/", 200)
	if !strings.Contains(home, "Ana sayfa") || !strings.Contains(home, "Deneme Sitesi") || !strings.Contains(home, "Şablonlar") {
		t.Fatalf("starter home = %s", home)
	}
	about := templateGet(t, base+"/hakkinda", 200)
	if !strings.Contains(about, "Hakkında") || !strings.Contains(about, "Deneme Sitesi") {
		t.Fatalf("starter about = %s", about)
	}
	missing := templateGet(t, base+"/yok", 404)
	if !strings.Contains(missing, "Sayfa bulunamadı") {
		t.Fatalf("starter 404 = %s", missing)
	}
	failed := templateGet(t, base+"/ornek-hata", 500)
	if !strings.Contains(failed, "Sunucu hatası") || !strings.Contains(failed, "Ayrıntı gösterilmez") {
		t.Fatalf("starter 500 = %s", failed)
	}
	if strings.Contains(failed, "DomainError") || strings.Contains(failed, `DomainError("ornek")`) {
		t.Fatalf("starter 500 leaked the failure: %s", failed)
	}
}

func templateGet(t *testing.T, url string, status int) string {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != status {
		t.Fatalf("%s status = %d body = %s", url, response.StatusCode, body)
	}
	return string(body)
}

func templateDiagnostics(result Result) string {
	var text strings.Builder
	for _, item := range result.Diagnostics {
		text.WriteString(item.Message)
		text.WriteString("\n")
	}
	return text.String()
}

func templateProgram(port int) string {
	return `bring HTTP
bring Templates
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response
from Templates bring (TemplateSet, TemplateData)

layout: String := r"""
<html><body>{{template "components/nav.html" .}}<main>{{block "content" .}}{{end}}</main></body></html>
"""
nav: String := r"""
<nav>{{.name}}</nav>
"""
homePage: String := r"""
{{define "content"}}<h1>{{.title}}</h1><p>{{.note}}</p><ul>{{range .topics}}<li>{{.}}</li>{{end}}</ul>{{.safe}}{{end}}
"""
aboutPage: String := r"""
{{define "content"}}<h1>Hakkında</h1><p>{{.name}}</p>{{end}}
"""
missingPage: String := r"""
{{define "content"}}<h1>Sayfa bulunamadı</h1>{{end}}
"""

pages: TemplateSet := Templates.sources(
    ["layouts/site.html", "components/nav.html", "pages/home.html", "pages/about.html", "pages/not-found.html"]
    [layout, nav, homePage, aboutPage, missingPage]
)

data: Function := () -> TemplateData {
    return Templates.object({
        "title": Templates.fromString("Ana sayfa")
        "name": Templates.fromString("<script>alert(1)</script>")
        "note": Templates.fromString("İstanbul")
        "topics": Templates.list([Templates.fromString("Şablonlar")])
        "safe": Templates.trusted("<b>güvenli</b>")
    })
}

home: Function := (request: Request) -> Response {
    pages: Global TemplateSet
    return HTTP.html(pages.layout("layouts/site.html", "pages/home.html", data()))
}

about: Function := (request: Request) -> Response {
    pages: Global TemplateSet
    return HTTP.html(pages.layout("layouts/site.html", "pages/about.html", data()))
}

missing: Function := (request: Request) -> Response {
    pages: Global TemplateSet
    return HTTP.html(pages.layout("layouts/site.html", "pages/not-found.html", data()), 200)
}

broken: Function := (request: Request) -> Response {
    toss(DomainError("gizli"))
}

failure: Function := (request: Request) -> Response {
    toss(DomainError("error page"))
}

app: Server := HTTP.server("127.0.0.1", ` + strconv.Itoa(port) + `)
app.get("/", home)
app.get("/hakkinda", about)
app.get("/hata", broken)
app.notFound(missing)
app.internalError(failure)
app.start()
`
}
