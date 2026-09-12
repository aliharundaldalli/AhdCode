package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The evaluator calls the same codes runtime a compiled program does, so these
// expectations match internal/build's native test exactly.

func TestQRThroughEvaluator(t *testing.T) {
	session := newSecurityTestSession()
	code := session.qrBuiltin("create", []any{"https://ahdcode.org/verify?id=42", nil})
	if value := session.codesOperation("QRCode.value", code, nil); value != "https://ahdcode.org/verify?id=42" {
		t.Fatalf("value = %#v", value)
	}
	if level := session.codesOperation("QRCode.level", code, nil); level != "M" {
		t.Fatalf("default level = %#v", level)
	}
	size := session.codesOperation("QRCode.size", code, nil).(int64)
	matrix := session.codesOperation("QRCode.matrix", code, nil).(*List)
	if int64(len(matrix.Items)) != size || int64(len(matrix.Items[0].(*List).Items)) != size {
		t.Fatalf("matrix is not %dx%d", size, size)
	}
	if dark, ok := matrix.Items[0].(*List).Items[0].(bool); !ok || !dark {
		t.Fatal("the top-left finder module should be dark")
	}
	directory := t.TempDir()
	session.codesOperation("QRCode.savePNG", code, []any{filepath.Join(directory, "qr.png"), nil})
	session.codesOperation("QRCode.saveSVG", code, []any{filepath.Join(directory, "qr.svg"), 4.0})
	for _, name := range []string{"qr.png", "qr.svg"} {
		if info, err := os.Stat(filepath.Join(directory, name)); err != nil || info.Size() == 0 {
			t.Fatalf("%s not written: %v", name, err)
		}
	}
	message := evaluatorRaisedMessage(t, "QRError", func() { session.qrBuiltin("create", []any{"x", "q"}) })
	if message != `QR.create level must be L, M, Q, or H; received "q"` {
		t.Fatalf("level message %q", message)
	}
	message = evaluatorRaisedMessage(t, "QRError", func() {
		session.codesOperation("QRCode.savePNG", code, []any{filepath.Join(directory, "qr.gif"), nil})
	})
	if message != "QRCode.savePNG destination must use the .png extension" {
		t.Fatalf("extension message %q", message)
	}
}

func TestBarcodeThroughEvaluator(t *testing.T) {
	session := newSecurityTestSession()
	ean := session.barcodeBuiltin("ean13", []any{"590123412345"})
	if kind := session.codesOperation("BarcodeCode.kind", ean, nil); kind != "EAN13" {
		t.Fatalf("kind = %#v", kind)
	}
	if value := session.codesOperation("BarcodeCode.value", ean, nil); value != "5901234123457" {
		t.Fatalf("value = %#v", value)
	}
	if pattern := session.codesOperation("BarcodeCode.pattern", ean, nil).(*List); len(pattern.Items) != 95 {
		t.Fatalf("EAN-13 pattern has %d modules, want 95", len(pattern.Items))
	}
	upc := session.barcodeBuiltin("upca", []any{"036000291452"})
	if value := session.codesOperation("BarcodeCode.value", upc, nil); value != "036000291452" {
		t.Fatalf("UPC-A value = %#v", value)
	}
	code128 := session.barcodeBuiltin("code128", []any{"AHD-42"})
	directory := t.TempDir()
	session.codesOperation("BarcodeCode.savePNG", code128, []any{filepath.Join(directory, "bar.png"), int64(900), nil})
	session.codesOperation("BarcodeCode.saveSVG", code128, []any{filepath.Join(directory, "bar.svg"), nil, nil})
	message := evaluatorRaisedMessage(t, "BarcodeError", func() { session.barcodeBuiltin("upca", []any{"036000291453"}) })
	if message != "Barcode.upca check digit 3 is wrong for 03600029145; the correct check digit is 2" {
		t.Fatalf("UPC-A check digit message %q", message)
	}
	message = evaluatorRaisedMessage(t, "BarcodeError", func() { session.barcodeBuiltin("code128", []any{"çay"}) })
	if !strings.Contains(message, "only ASCII characters") {
		t.Fatalf("Code128 message %q", message)
	}
}
