package ahdidentity

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// The embedded icon is the repository's official artwork, byte for byte.
func TestIconIsTheOfficialArtwork(t *testing.T) {
	official, err := os.ReadFile(filepath.Join("..", "..", "editors", "vscode", "images", "ahdcode-icon.png"))
	if err != nil {
		t.Skip("official icon not available outside the repository")
	}
	if !bytes.Equal(official, OfficialIconPNG()) {
		t.Fatal("cmd/ahdidentity/ahdcode-icon.png differs from editors/vscode/images/ahdcode-icon.png")
	}
	if officialIcon().Bounds().Dx() != 512 || officialIcon().Bounds().Dy() != 512 {
		t.Fatalf("icon size %v", Icon().Bounds())
	}
}

func TestShrinkMakesTheWindowIconSizes(t *testing.T) {
	for _, size := range []int{16, 32, 48, 256} {
		small := shrink(Icon(), size)
		if small.Bounds().Dx() != size || small.Bounds().Dy() != size {
			t.Fatalf("shrink to %d gave %v", size, small.Bounds())
		}
	}
	if shrink(Icon(), 1024) != Icon() {
		t.Fatal("shrinking to a larger size must keep the source")
	}
}

func TestPackagedIdentityFile(t *testing.T) {
	directory := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(directory, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if err := os.WriteFile(filepath.Join(directory, "icon.png"), OfficialIconPNG(), 0o600); err != nil {
		t.Fatal(err)
	}
	name, icon, raw, ok := readApp(write("good.json", `{"name":"Ledger","icon":"icon.png"}`))
	if !ok || name != "Ledger" || icon == nil || !bytes.Equal(raw, OfficialIconPNG()) {
		t.Fatalf("good identity: %q %v %v", name, icon != nil, ok)
	}
	if name, icon, _, ok := readApp(write("plain.json", `{"name":"Kasa Defteri"}`)); !ok || name != "Kasa Defteri" || icon != nil {
		t.Fatal("an identity without an icon")
	}
	for _, bad := range []string{`{"name":""}`, `{"name":"a/b"}`, `not json`, `{"name":"x","icon":"../icon.png"}`} {
		if _, _, _, ok := readApp(write("bad.json", bad)); ok {
			t.Fatalf("%s accepted", bad)
		}
	}
	if _, _, _, ok := readApp(filepath.Join(directory, "missing.json")); ok {
		t.Fatal("a missing identity file")
	}
	// The test binary is not packaged.
	if Packaged() || Name() != DefaultName {
		t.Fatal("a helper outside an application claimed a packaged identity")
	}
}

func TestRoundedIconHasTheMacOSShape(t *testing.T) {
	rounded := Rounded(officialIcon(), 512)
	alpha := func(x, y int) uint8 { return rounded.NRGBAAt(x, y).A }
	// The margin and the corners are transparent; the middle and the body's
	// straight edges are opaque.
	if alpha(25, 256) != 0 || alpha(55, 55) != 0 || alpha(256, 256) != 255 || alpha(60, 256) != 255 {
		t.Fatalf("rounded icon alpha: margin %d corner %d center %d edge %d", alpha(25, 256), alpha(55, 55), alpha(256, 256), alpha(60, 256))
	}
	if RoundedPNG(officialIcon(), 256) == nil {
		t.Fatal("RoundedPNG")
	}
}
