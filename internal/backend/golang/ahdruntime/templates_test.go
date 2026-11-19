package ahdruntime

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestTemplatesEscapeLayoutPartialAndTurkish(t *testing.T) {
	class := AhdClassTemplateError
	layout := `<!DOCTYPE html><html lang="tr"><body>{{template "components/nav.html" .}}<main>{{block "content" .}}{{end}}</main></body></html>`
	nav := `<nav>{{.name}}</nav>`
	home := `{{define "content"}}<h1>{{.title}}</h1><p>{{.note}}</p><ul>{{range .topics}}<li>{{.}}</li>{{end}}</ul>{{.safe}}{{end}}`
	about := `{{define "content"}}<p>Hakkında: {{.name}}</p>{{end}}`
	set := AhdTemplatesSources(class,
		[]string{"layouts/site.html", "components/nav.html", "pages/home.html", "pages/about.html"},
		[]string{layout, nav, home, about},
	)
	data := AhdTemplatesObject(class,
		[]string{"title", "name", "note", "topics", "safe"},
		[]string{
			AhdTemplatesFromString("Ana sayfa"),
			AhdTemplatesFromString(`<script>alert("x")</script>`),
			AhdTemplatesFromString("İstanbul'da şablon"),
			AhdTemplatesList([]string{AhdTemplatesFromString("Şablonlar"), AhdTemplatesFromString("Yerleşim")}),
			AhdTemplatesTrusted("<b>güvenli</b>"),
		},
	)
	page := AhdTemplatesLayout(class, set, "layouts/site.html", "pages/home.html", data)
	for _, want := range []string{"Ana sayfa", "İstanbul&#39;da şablon", "Şablonlar", "Yerleşim", "<b>güvenli</b>", "&lt;script&gt;"} {
		if !strings.Contains(page, want) {
			t.Fatalf("page missing %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "<script>alert") {
		t.Fatalf("unescaped script:\n%s", page)
	}
	other := AhdTemplatesLayout(class, set, "layouts/site.html", "pages/about.html", data)
	if !strings.Contains(other, "Hakkında:") || strings.Contains(other, "Ana sayfa") {
		t.Fatalf("about page = %s", other)
	}
	literal := AhdTemplatesRender(class, set, "components/nav.html", AhdTemplatesObject(class,
		[]string{"name"}, []string{AhdTemplatesFromString("{{.secret}}")},
	))
	if literal != "<nav>{{.secret}}</nav>" {
		t.Fatalf("partial = %q", literal)
	}
}

func TestTemplatesFailClosed(t *testing.T) {
	class := AhdClassTemplateError
	assertPanic := func(name string, call func()) {
		t.Helper()
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatalf("%s was accepted", name)
			}
		}()
		call()
	}
	assertPanic("missing template", func() {
		set := AhdTemplatesParse(class, "pages/home.html", "<p>{{.title}}</p>")
		AhdTemplatesRender(class, set, "missing.html", AhdTemplatesObject(class, []string{"title"}, []string{AhdTemplatesFromString("x")}))
	})
	assertPanic("missing value", func() {
		set := AhdTemplatesParse(class, "pages/home.html", "<p>{{.title}}</p>")
		AhdTemplatesRender(class, set, "pages/home.html", AhdTemplatesObject(class, nil, nil))
	})
	assertPanic("malformed", func() {
		AhdTemplatesParse(class, "pages/home.html", "<p>{{</p>")
	})
	assertPanic("traversal", func() {
		AhdTemplatesParse(class, "../secret.html", "<p>x</p>")
	})
	assertPanic("non finite", func() {
		AhdTemplatesFromReal(class, math.NaN())
	})
	directory := t.TempDir()
	root := filepath.Join(directory, "templates")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "page.html"), []byte("<p>İç</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "secret.html"), []byte("<p>secret</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	set := AhdTemplatesFiles(class, root)
	got := AhdTemplatesRender(class, set, "page.html", AhdTemplatesNull())
	if got != "<p>İç</p>" {
		t.Fatalf("file render = %q", got)
	}
	assertPanic("outside name", func() {
		AhdTemplatesRender(class, set, "../secret.html", AhdTemplatesNull())
	})
	if err := os.Symlink(filepath.Join(directory, "secret.html"), filepath.Join(root, "leak.html")); err == nil {
		assertPanic("symlink", func() {
			AhdTemplatesFiles(class, root)
		})
	}
}

func TestTemplatesRenderConcurrently(t *testing.T) {
	class := AhdClassTemplateError
	set := AhdTemplatesParse(class, "page.html", "<p>{{.name}}</p>")
	left := AhdTemplatesObject(class, []string{"name"}, []string{AhdTemplatesFromString("Ayşe")})
	right := AhdTemplatesObject(class, []string{"name"}, []string{AhdTemplatesFromString("Mehmet")})
	var group sync.WaitGroup
	errors := make(chan string, 8)
	for index := 0; index < 8; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			data := left
			want := "Ayşe"
			if index%2 == 1 {
				data = right
				want = "Mehmet"
			}
			got := AhdTemplatesRender(class, set, "page.html", data)
			if got != "<p>"+want+"</p>" {
				errors <- got
			}
		}(index)
	}
	group.Wait()
	close(errors)
	for err := range errors {
		t.Fatalf("concurrent render = %s", err)
	}
}
