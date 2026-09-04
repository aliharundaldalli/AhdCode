package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequiredFileIsAnalyzedThroughAppEntry(t *testing.T) {
	directory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, "Shared"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(directory, "Pages"), 0o700); err != nil {
		t.Fatal(err)
	}
	helperPath := filepath.Join(directory, "Shared", "Helpers.ahd")
	if err := os.WriteFile(helperPath, []byte("greet: Function := () -> String {\n    return \"hi\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	homePath := filepath.Join(directory, "Pages", "Home.ahd")
	home := "require(\"Shared/Helpers.ahd\")\nwrite(greet())\n"
	if err := os.WriteFile(homePath, []byte(home), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "app.ahd"), []byte("require(\"Pages/Home.ahd\")\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewStore()
	result := store.Open(homePath, home)
	for _, item := range result.Diagnostics[canonicalPath(homePath)] {
		t.Fatalf("nested require file should compile through app.ahd, got %#v", item)
	}
	offset := strings.Index(home, "greet()")
	hover, ok := store.Hover(homePath, offset)
	if !ok {
		t.Fatal("expected hover on greet from the required helper")
	}
	if !strings.Contains(hover.Text, "greet") {
		t.Fatalf("hover = %q", hover.Text)
	}
}

func TestRequiredFileErrorIsNotPaintedOnEntryRequire(t *testing.T) {
	directory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, "Pages"), 0o700); err != nil {
		t.Fatal(err)
	}
	homePath := filepath.Join(directory, "Pages", "Home.ahd")
	if err := os.WriteFile(homePath, []byte("x: Int := \"not an int\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	appPath := filepath.Join(directory, "app.ahd")
	app := "require(\"Pages/Home.ahd\")\nwrite(\"entry\")\n"
	store := NewStore()
	result := store.Open(appPath, app)
	if items := result.Diagnostics[canonicalPath(appPath)]; len(items) != 0 {
		t.Fatalf("entry should not inherit required-file diagnostics: %#v", items)
	}
	if items := result.Diagnostics[canonicalPath(homePath)]; len(items) == 0 {
		t.Fatal("expected Home.ahd's type error to stay on Home.ahd")
	}
}
