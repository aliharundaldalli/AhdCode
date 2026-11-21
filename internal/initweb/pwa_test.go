package initweb

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPWAProjectFiles(t *testing.T) {
	root := t.TempDir()
	const entry = "existing app source"
	app := filepath.Join(root, "app.ahd")
	if err := os.WriteFile(app, []byte(entry), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := PWA(root, "İstanbul <&> Site", "İstanbul"); err != nil {
		t.Fatal(err)
	}
	read := func(path string) []byte {
		t.Helper()
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := string(read("app.ahd")); got != entry {
		t.Fatalf("entry was changed: %q", got)
	}
	var manifest pwaManifest
	if err := json.Unmarshal(read("PWA/app.webmanifest"), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "İstanbul <&> Site" || manifest.ShortName != "İstanbul" ||
		manifest.Display != "standalone" || manifest.Scope != "/" || manifest.StartURL != "/" ||
		manifest.ThemeColor == "" || manifest.BackgroundColor == "" || len(manifest.Icons) != 2 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	worker := string(read("PWA/worker.js"))
	for i, size := range []int{192, 512} {
		icon := manifest.Icons[i]
		path, ok := strings.CutPrefix(icon.Source, "/ahd-pwa/")
		if !ok || icon.Type != "image/png" {
			t.Fatalf("unexpected icon: %+v", icon)
		}
		data := read("PWA/public/" + path)
		config, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil || config.Width != size || config.Height != size {
			t.Fatalf("icon %s: %+v, %v", path, config, err)
		}
		hash := sha256.Sum256(data)
		if !strings.Contains(worker, "sha256-"+base64.StdEncoding.EncodeToString(hash[:])) {
			t.Fatalf("missing integrity hash for %s", path)
		}
	}
	if err := PWA(root, "replacement", ""); err == nil {
		t.Fatal("second initialization unexpectedly succeeded")
	}
	if !strings.Contains(string(read("PWA/app.webmanifest")), "İstanbul") {
		t.Fatal("second initialization modified the manifest")
	}
}

func TestPWAInitRejectsInvalidNamesAndConcurrentOverwrite(t *testing.T) {
	root := t.TempDir()
	if err := PWA(root, "contains\ncontrol", ""); err == nil {
		t.Fatal("control character accepted")
	}
	if _, err := os.Stat(filepath.Join(root, "PWA")); !os.IsNotExist(err) {
		t.Fatalf("invalid input touched the project: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { results <- PWA(root, "", "") })
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected one exclusive initialization, got %d", successes)
	}
}
