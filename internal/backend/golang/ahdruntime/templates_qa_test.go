package ahdruntime

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestTemplatesRenderDoesNotInvalidateLayout(t *testing.T) {
	class := AhdClassTemplateError
	set := AhdTemplatesSources(class,
		[]string{"layouts/site.html", "components/nav.html", "pages/home.html"},
		[]string{
			`<html>{{template "components/nav.html" .}}<main>{{block "content" .}}{{end}}</main></html>`,
			`<nav>{{.name}}</nav>`,
			`{{define "content"}}<h1>{{.title}}</h1>{{end}}`,
		},
	)
	data := AhdTemplatesObject(class,
		[]string{"name", "title"},
		[]string{AhdTemplatesFromString("Nav"), AhdTemplatesFromString("Başlık")},
	)
	partial := AhdTemplatesRender(class, set, "components/nav.html", data)
	if partial != "<nav>Nav</nav>" {
		t.Fatalf("render = %q", partial)
	}
	page := AhdTemplatesLayout(class, set, "layouts/site.html", "pages/home.html", data)
	if !strings.Contains(page, "<h1>Başlık</h1>") || !strings.Contains(page, "<nav>Nav</nav>") {
		t.Fatalf("layout after render = %s", page)
	}
	again := AhdTemplatesRender(class, set, "components/nav.html", data)
	if again != partial {
		t.Fatalf("render after layout = %q", again)
	}
	second := AhdTemplatesLayout(class, set, "layouts/site.html", "pages/home.html", data)
	if second != page {
		t.Fatalf("second layout = %s", second)
	}
}

func TestTemplatesMixedRenderLayoutConcurrently(t *testing.T) {
	class := AhdClassTemplateError
	set := AhdTemplatesSources(class,
		[]string{"layouts/site.html", "components/nav.html", "pages/public.html", "pages/admin.html"},
		[]string{
			`<main>{{template "components/nav.html" .}}{{block "content" .}}default{{end}}</main>`,
			`<nav>x</nav>`,
			`{{template "content" .}}{{define "content"}}public-{{.n}}{{end}}`,
			`{{template "content" .}}{{define "content"}}ADMIN-SECRET-{{.n}}{{end}}`,
		},
	)
	render := func(name, data string) string {
		return AhdTemplatesRender(class, set, name, data)
	}
	layout := func(name, data string) string {
		return AhdTemplatesLayout(class, set, "layouts/site.html", name, data)
	}
	cases := []struct {
		call func(string, string) string
		name string
		want string
	}{
		{render, "components/nav.html", "<nav>x</nav>"},
		{render, "layouts/site.html", "<main><nav>x</nav>default</main>"},
		{render, "pages/public.html", "public-99"},
		{render, "pages/admin.html", "ADMIN-SECRET-99"},
		{layout, "pages/public.html", "<main><nav>x</nav>public-99</main>"},
		{layout, "pages/admin.html", "<main><nav>x</nav>ADMIN-SECRET-99</main>"},
	}
	data := AhdTemplatesObject(class, []string{"n"}, []string{AhdTemplatesFromInt(99)})
	var group sync.WaitGroup
	for range 8 {
		for _, check := range cases {
			group.Go(func() {
				if got := check.call(check.name, data); got != check.want {
					t.Errorf("%s rendered %q, want %q", check.name, got, check.want)
				}
			})
		}
	}
	group.Wait()
}

func TestTemplatesPageDefinitionsDoNotLeak(t *testing.T) {
	class := AhdClassTemplateError
	set := AhdTemplatesSources(class,
		[]string{"layouts/site.html", "pages/public.html", "pages/admin.html"},
		[]string{
			`<html>{{block "sidebar" .}}public-side{{end}}{{block "content" .}}{{end}}</html>`,
			`{{define "sidebar"}}public-side{{end}}{{define "content"}}public-body{{end}}`,
			`{{define "sidebar"}}ADMIN-SECRET{{end}}{{define "content"}}admin-body{{end}}`,
		},
	)
	public := AhdTemplatesLayout(class, set, "layouts/site.html", "pages/public.html", AhdTemplatesNull())
	if !strings.Contains(public, "public-body") || strings.Contains(public, "ADMIN-SECRET") {
		t.Fatalf("public leaked admin: %s", public)
	}
	admin := AhdTemplatesLayout(class, set, "layouts/site.html", "pages/admin.html", AhdTemplatesNull())
	if !strings.Contains(admin, "ADMIN-SECRET") || !strings.Contains(admin, "admin-body") {
		t.Fatalf("admin = %s", admin)
	}
	publicAgain := AhdTemplatesLayout(class, set, "layouts/site.html", "pages/public.html", AhdTemplatesNull())
	if strings.Contains(publicAgain, "ADMIN-SECRET") {
		t.Fatalf("admin definition stuck on public: %s", publicAgain)
	}
}

func TestTemplatesRejectPageIdentityHijackAndSharedConflict(t *testing.T) {
	class := AhdClassTemplateError
	assertPanic := func(name string, call func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s was accepted", name)
			}
		}()
		call()
	}
	assertPanic("hijack", func() {
		AhdTemplatesSources(class,
			[]string{"layouts/site.html", "pages/home.html"},
			[]string{`<html>{{block "content" .}}{{end}}</html>`, `{{define "layouts/site.html"}}stolen{{end}}`},
		)
	})
	assertPanic("shared conflict", func() {
		AhdTemplatesSources(class,
			[]string{"components/a.html", "components/b.html"},
			[]string{`{{define "shared"}}a{{end}}`, `{{define "shared"}}b{{end}}`},
		)
	})
}

func TestTemplatesRejectSharedIdentityHijack(t *testing.T) {
	class := AhdClassTemplateError
	for _, test := range []struct {
		name, target, source string
	}{
		{"page define", "pages/home.html", `{{define "pages/home.html"}}wrong{{end}}`},
		{"page block", "pages/home.html", `{{block "pages/home.html" .}}wrong{{end}}`},
		{"raw quoted page", "pages/home.html", "{{define \x60pages/home.html\x60}}wrong{{end}}"},
		{"escaped quoted page", "pages/home.html", `{{define "pages/h\x6fme.html"}}wrong{{end}}`},
		{"file name containing quotes", `pages/"quoted".html`, `{{define "pages/\"quoted\".html"}}wrong{{end}}`},
		{"shared sibling", "layouts/site.html", `{{define "layouts/site.html"}}wrong{{end}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, index := range []int{0, 1} {
				names := []string{"components/evil.html", test.target}
				bodies := []string{test.source, ""}
				names[0], names[index] = names[index], names[0]
				bodies[0], bodies[index] = bodies[index], bodies[0]
				requireTemplateError(t, "must not redefine", func() {
					AhdTemplatesSources(class, names, bodies)
				})
			}
		})
	}
	t.Run("reserved page namespace", func(t *testing.T) {
		requireTemplateError(t, "must not redefine", func() {
			AhdTemplatesParse(class, "components/evil.html",
				`{{define "pages/admin/private"}}wrong{{end}}`)
		})
	})
	t.Run("page owns its own identity", func(t *testing.T) {
		set := AhdTemplatesParse(class, `pages/"quoted".html`,
			`{{define "pages/\"quoted\".html"}}own page{{end}}`)
		if got := AhdTemplatesRender(class, set, `pages/"quoted".html`, AhdTemplatesNull()); got != "own page" {
			t.Fatalf("render = %q", got)
		}
	})
}

func TestTemplatesLayoutRequiresPageNamespace(t *testing.T) {
	class := AhdClassTemplateError
	set := AhdTemplatesSources(class, []string{"layouts/site.html", "components/nav.html", "pages/home.html"},
		[]string{`{{block "content" .}}default{{end}}`, "<nav>Nav</nav>", `{{define "content"}}home{{end}}`})
	data := AhdTemplatesNull()
	requireTemplateError(t, "must be under pages/", func() {
		AhdTemplatesLayout(class, set, "layouts/site.html", "components/nav.html", data)
	})
	requireTemplateError(t, "is not in this set", func() {
		AhdTemplatesLayout(class, set, "layouts/site.html", "pages/missing.html", data)
	})
	requireTemplateError(t, "cannot be used as a layout", func() {
		AhdTemplatesLayout(class, set, "pages/home.html", "pages/home.html", data)
	})
	if got := AhdTemplatesLayout(class, set, "layouts/site.html", "pages/home.html", data); got != "home" {
		t.Fatalf("layout = %q", got)
	}
}

func TestTemplatesConstructorsEnforceCumulativeDataLimits(t *testing.T) {
	class := AhdClassTemplateError
	wrappers := []struct {
		name  string
		build func([]string) string
	}{
		{"list", AhdTemplatesList},
		{"object", func(children []string) string {
			keys := make([]string, len(children))
			for index := range keys {
				keys[index] = strconv.Itoa(index)
			}
			return AhdTemplatesObject(class, keys, children)
		}},
	}
	leaves := make([]string, (ahdTemplateMaxNodes-2)/2)
	for index := range leaves {
		leaves[index] = AhdTemplatesFromString(`{["\}"]}`)
	}
	branch := AhdTemplatesList(leaves)
	for _, wrapper := range wrappers {
		t.Run(wrapper.name, func(t *testing.T) {
			// Exactly 10000 nodes including the new root, followed by a
			// rejected extra child. Repeated branches count independently.
			exact := wrapper.build([]string{branch, AhdTemplatesList(leaves[:len(leaves)-1])})
			nodes := 0
			if _, err := ahdTemplateDecode(exact, 0, &nodes); err != nil || nodes != ahdTemplateMaxNodes {
				t.Fatalf("boundary decode counted %d nodes: %v", nodes, err)
			}
			requireTemplateError(t, "too large", func() {
				wrapper.build([]string{branch, branch})
			})
			requireTemplateError(t, "too large", func() {
				wrapper.build(make([]string, ahdTemplateMaxNodes))
			})

			deep := AhdTemplatesNull()
			for range ahdTemplateMaxDepth {
				deep = wrapper.build([]string{deep})
			}
			if _, err := ahdTemplateDecode(deep, 0, new(int)); err != nil {
				t.Fatalf("depth boundary decode: %v", err)
			}
			requireTemplateError(t, "too deep", func() {
				wrapper.build([]string{deep})
			})
		})
	}
}

func requireTemplateError(t *testing.T, message string, call func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		signal, ok := recovered.(*AhdSignal)
		if !ok || signal.Instance.AhdClassOf() != AhdClassTemplateError || !strings.Contains(signal.Message, message) {
			t.Errorf("expected TemplateError containing %q, got %v", message, recovered)
		}
	}()
	call()
}

func TestTemplatesDataEncodingIsLinear(t *testing.T) {
	class := AhdClassTemplateError
	nested := AhdTemplatesFromString(`quote " \ İstanbul`)
	sizes := map[int]int{}
	for _, depth := range []int{1, 10, 20, 32} {
		value := nested
		for level := 0; level < depth; level++ {
			value = AhdTemplatesList([]string{value})
		}
		sizes[depth] = len(value)
		if _, err := ahdTemplateDecode(value, 0, new(int)); err != nil {
			t.Fatalf("depth %d decode: %v", depth, err)
		}
	}
	if sizes[32] > 64*1024 {
		t.Fatalf("depth 32 encoding grew too large: %d bytes (depth1=%d depth10=%d depth20=%d)", sizes[32], sizes[1], sizes[10], sizes[20])
	}
	if sizes[32] > sizes[20]*4 || sizes[20] > sizes[10]*4 {
		t.Fatalf("encoding did not stay roughly linear: %v", sizes)
	}
	object := AhdTemplatesFromString("x")
	for level := 0; level < 20; level++ {
		object = AhdTemplatesObject(class, []string{"c"}, []string{object})
	}
	if len(object) > 64*1024 {
		t.Fatalf("nested object encoding = %d", len(object))
	}
}

func TestTemplatesRejectsUnsafeNamesAndLimits(t *testing.T) {
	class := AhdClassTemplateError
	assertPanic := func(name string, call func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s was accepted", name)
			}
		}()
		call()
	}
	assertPanic("windows path", func() {
		AhdTemplatesParse(class, "C:/a.html", "<p>x</p>")
	})
	assertPanic("absolute", func() {
		AhdTemplatesParse(class, "/tmp/a.html", "<p>x</p>")
	})
	assertPanic("oversized parse", func() {
		AhdTemplatesParse(class, "pages/home.html", strings.Repeat("a", ahdTemplateMaxBytes+1))
	})
	assertPanic("bad utf8", func() {
		AhdTemplatesParse(class, "pages/home.html", string([]byte{0xff, 0xfe, 0xfd}))
	})
	names := make([]string, ahdTemplateMaxFiles+1)
	bodies := make([]string, ahdTemplateMaxFiles+1)
	for index := range names {
		names[index] = "components/item" + strconv.Itoa(index) + ".html"
		bodies[index] = "<p>" + strconv.Itoa(index) + "</p>"
	}
	assertPanic("too many sources", func() {
		AhdTemplatesSources(class, names, bodies)
	})
	directory := t.TempDir()
	outside := filepath.Join(directory, "outside")
	root := filepath.Join(directory, "templates")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.html"), []byte("<p>secret</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(root, "leak")
	if err := makeTemplateJunction(t, outside, junction); err != nil {
		t.Skip(err.Error())
	}
	assertPanic("junction child", func() {
		AhdTemplatesFiles(class, root)
	})
}

func TestTemplatesOutputBudget(t *testing.T) {
	class := AhdClassTemplateError
	set := AhdTemplatesParse(class, "boom.html", `{{range .N}}{{.}}{{end}}`)
	chunk := strings.Repeat("a", 20000)
	items := make([]string, 500)
	for index := range items {
		items[index] = AhdTemplatesFromString(chunk)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("oversized output was accepted")
		}
	}()
	AhdTemplatesRender(class, set, "boom.html", AhdTemplatesObject(class, []string{"N"}, []string{AhdTemplatesList(items)}))
}

func makeTemplateJunction(t *testing.T, target, link string) error {
	t.Helper()
	command := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	output, err := command.CombinedOutput()
	if err != nil {
		return err
	}
	if _, err := os.Lstat(link); err != nil {
		return err
	}
	_ = output
	return nil
}
