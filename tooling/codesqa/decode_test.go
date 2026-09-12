// Package codesqa decodes the QR and barcode symbols AhdCode generates with
// an independent reader, github.com/makiuchi-d/gozxing (a Go port of ZXing).
//
// It is a separate Go module on purpose: the decoder is a QA tool only. It is
// never a dependency of the ahdcode module, its binaries, or generated
// programs, so it never enters a release payload or its license inventory.
//
// The PNG fixtures it reads live in
// internal/backend/golang/ahdruntime/testdata/codes. The runtime test
// TestCodesDecoderFixturesAreCurrent proves those files show exactly the
// modules the current encoder draws, so decoding them here proves the
// encoder's output carries the intended payload.
//
//	cd tooling/codesqa && go test ./...
package codesqa

import (
	"encoding/json"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
	"github.com/makiuchi-d/gozxing/qrcode"
)

type fixture struct {
	File    string `json:"file"`
	Format  string `json:"format"`
	Payload string `json:"payload"`
}

func TestIndependentDecoderReadsGeneratedSymbols(t *testing.T) {
	directory := filepath.Join("..", "..", "internal", "backend", "golang", "ahdruntime", "testdata", "codes")
	manifest, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []fixture
	if err := json.Unmarshal(manifest, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) < 6 {
		t.Fatalf("expected at least six fixtures, found %d", len(fixtures))
	}
	readers := map[string]gozxing.Reader{
		"QR_CODE":  qrcode.NewQRCodeReader(),
		"EAN_13":   oned.NewEAN13Reader(),
		"UPC_A":    oned.NewUPCAReader(),
		"CODE_128": oned.NewCode128Reader(),
	}
	for _, item := range fixtures {
		file, err := os.Open(filepath.Join(directory, item.File))
		if err != nil {
			t.Fatal(err)
		}
		picture, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			t.Fatalf("%s: %v", item.File, err)
		}
		bitmap, err := gozxing.NewBinaryBitmapFromImage(picture)
		if err != nil {
			t.Fatalf("%s: %v", item.File, err)
		}
		reader := readers[item.Format]
		if reader == nil {
			t.Fatalf("%s: no reader for %s", item.File, item.Format)
		}
		result, err := reader.Decode(bitmap, map[gozxing.DecodeHintType]interface{}{gozxing.DecodeHintType_TRY_HARDER: true})
		if err != nil {
			t.Fatalf("%s: the independent decoder could not read the symbol: %v", item.File, err)
		}
		if result.GetBarcodeFormat().String() != item.Format || result.GetText() != item.Payload {
			t.Fatalf("%s decoded as %s %q, want %s %q", item.File, result.GetBarcodeFormat(), result.GetText(), item.Format, item.Payload)
		}
	}
}
