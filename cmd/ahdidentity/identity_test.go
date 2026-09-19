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
	if !bytes.Equal(official, IconPNG()) {
		t.Fatal("cmd/ahdidentity/ahdcode-icon.png differs from editors/vscode/images/ahdcode-icon.png")
	}
	if Icon().Bounds().Dx() != 512 || Icon().Bounds().Dy() != 512 {
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
