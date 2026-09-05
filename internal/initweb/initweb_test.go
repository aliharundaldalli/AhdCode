package initweb

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebFreshDirectory(t *testing.T) {
	root := t.TempDir()
	var out, errBuf bytes.Buffer
	if err := Web(root, &out, &errBuf); err != nil {
		t.Fatalf("Web: %v\nstderr=%s", err, errBuf.String())
	}
	want := []string{
		"app.ahd",
		".env",
		".env.example",
		".gitignore",
		"Config/App.ahd",
		"Components/Navbar.ahd",
		"Components/Footer.ahd",
		"Layouts/Main.ahd",
		"Pages/Home.ahd",
		"public/style.css",
		"public/main.js",
	}
	for _, rel := range want {
		info, err := os.Lstat(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("%s is a symlink", rel)
		}
	}
	env, err := os.ReadFile(filepath.Join(root, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(env)
	for _, key := range []string{
		"APP_NAME=AhdCode Web",
		"APP_ENV=development",
		"APP_HOST=localhost",
		"APP_PROTOCOL=http",
		"SERVER_HOST=127.0.0.1",
		"SERVER_PORT=8080",
	} {
		if !strings.Contains(text, key) {
			t.Fatalf(".env missing %s:\n%s", key, text)
		}
	}
	if strings.Contains(strings.ToLower(text), "password") || strings.Contains(text, "TOKEN") {
		t.Fatalf(".env looks secret-bearing:\n%s", text)
	}
	ignore, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []string{".env", "*.dev", "*.run"} {
		if !gitignoreHas(string(ignore), entry) {
			t.Fatalf(".gitignore missing %s:\n%s", entry, ignore)
		}
	}
	example, err := os.ReadFile(filepath.Join(root, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(example), "APP_HOST=localhost") {
		t.Fatalf(".env.example missing APP_HOST:\n%s", example)
	}
	home, err := os.ReadFile(filepath.Join(root, "Pages/Home.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(home), "home: Function") {
		t.Fatalf("home handler not named home:\n%s", home)
	}
	if strings.Contains(string(home), "homePage") {
		t.Fatalf("scaffold used homePage")
	}
	if strings.Contains(string(home), "http") && strings.Contains(string(home), "cdn") {
		t.Fatalf("home loaded a CDN")
	}
	layout, err := os.ReadFile(filepath.Join(root, "Layouts/Main.ahd"))
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"navbar(config.name)",
		"footer(config.name)",
		`Web.UI.stylesheet("/assets/style.css")`,
		`Web.UI.element("script", {"src": "/assets/main.js"}, [])`,
	} {
		if !strings.Contains(string(layout), needle) {
			t.Fatalf("layout missing %s:\n%s", needle, layout)
		}
	}
	if strings.Contains(string(layout), "app.css") {
		t.Fatal("layout still references app.css")
	}
	css, err := os.ReadFile(filepath.Join(root, "public/style.css"))
	if err != nil {
		t.Fatal(err)
	}
	if len(css) != 0 {
		t.Fatalf("public/style.css should be empty, got %q", css)
	}
	script, err := os.ReadFile(filepath.Join(root, "public/main.js"))
	if err != nil {
		t.Fatal(err)
	}
	if len(script) != 0 {
		t.Fatalf("public/main.js should be empty, got %q", script)
	}
	if _, err := os.Stat(filepath.Join(root, "public/app.css")); !os.IsNotExist(err) {
		t.Fatal("public/app.css should not be generated")
	}
	if _, err := os.Stat(filepath.Join(root, "public/css")); !os.IsNotExist(err) {
		t.Fatal("public/css/ should not be generated")
	}
	if _, err := os.Stat(filepath.Join(root, "public/js")); !os.IsNotExist(err) {
		t.Fatal("public/js/ should not be generated")
	}
	if !strings.Contains(out.String(), "ahdcode dev app.ahd") {
		t.Fatalf("success output missing next step:\n%s", out.String())
	}
}

func TestWebConflictWritesNothing(t *testing.T) {
	root := t.TempDir()
	sentinel := []byte("keep-me\n")
	if err := os.WriteFile(filepath.Join(root, "app.ahd"), sentinel, 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errBuf bytes.Buffer
	err := Web(root, &out, &errBuf)
	if err == nil {
		t.Fatal("expected conflict")
	}
	if !strings.Contains(err.Error(), "app.ahd already exists") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "No files were written.") {
		t.Fatalf("error missing no-write note: %v", err)
	}
	got, readErr := os.ReadFile(filepath.Join(root, "app.ahd"))
	if readErr != nil || string(got) != string(sentinel) {
		t.Fatalf("sentinel changed: %q %v", got, readErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, "Config")); !os.IsNotExist(statErr) {
		t.Fatalf("Config/ was created on conflict")
	}
	if _, statErr := os.Stat(filepath.Join(root, ".env")); !os.IsNotExist(statErr) {
		t.Fatalf(".env was created on conflict")
	}
	if _, statErr := os.Stat(filepath.Join(root, "Pages")); !os.IsNotExist(statErr) {
		t.Fatalf("Pages/ was created on conflict")
	}
}

func TestWebDirectoryPathConflict(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Pages"), []byte("not-a-dir\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Web(root, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("expected directory conflict")
	}
	if !strings.Contains(err.Error(), "Pages is a file") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "app.ahd")); !os.IsNotExist(statErr) {
		t.Fatalf("partial scaffold after directory conflict")
	}
}

func TestWebGitignoreMerge(t *testing.T) {
	root := t.TempDir()
	existing := "# keep\nbuild/\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Web(root, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "# keep") || !strings.Contains(text, "build/") {
		t.Fatalf("lost existing rules:\n%s", text)
	}
	for _, entry := range []string{".env", "*.dev", "*.run"} {
		if !gitignoreHas(text, entry) {
			t.Fatalf("missing %s after merge:\n%s", entry, text)
		}
	}
}

func TestWebSymlinkConflict(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "elsewhere.ahd")
	if err := os.WriteFile(target, []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "app.ahd")); err != nil {
		t.Fatal(err)
	}
	err := Web(root, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("expected symlink conflict")
	}
	if !strings.Contains(err.Error(), "app.ahd is a symlink") {
		t.Fatalf("error = %v", err)
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil || string(got) != "secret\n" {
		t.Fatalf("symlink destination changed: %q %v", got, readErr)
	}
}

func TestWebSecondInitDoesNotOverwrite(t *testing.T) {
	root := t.TempDir()
	if err := Web(root, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "Pages/Home.ahd")
	original, err := os.ReadFile(home)
	if err != nil {
		t.Fatal(err)
	}
	edited := append([]byte("// edited\n"), original...)
	if err := os.WriteFile(home, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	err = Web(root, ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("second init should fail")
	}
	got, readErr := os.ReadFile(home)
	if readErr != nil || string(got) != string(edited) {
		t.Fatalf("user edit overwritten")
	}
}

func gitignoreHas(text, entry string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == entry {
			return true
		}
	}
	return false
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
