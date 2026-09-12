package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The persistent REPL encodes QR and barcode symbols through the evaluator,
// which calls the same runtime a compiled program does, so its results match
// internal/build's native expectations.
func TestCodesMatchNativeInPersistentREPL(t *testing.T) {
	directory := t.TempDir()
	qrPath := filepath.ToSlash(filepath.Join(directory, "qr.png"))
	barPath := filepath.ToSlash(filepath.Join(directory, "bar.svg"))
	input := `bring QR
bring Barcode
from QR bring (QRCode, QRError)
from Barcode bring (BarcodeCode, BarcodeError)
code: QRCode := QR.create("https://ahdcode.org/verify?id=AHD-2026-0042", "Q")
write(code.level() + " " + str(code.size()))
code.savePNG("` + qrPath + `", 400)
bar: BarcodeCode := Barcode.ean13("590123412345")
write(bar.kind() + " " + bar.value() + " " + str(len(bar.pattern())))
bar.saveSVG("` + barPath + `")
attempt {
    Barcode.ean13("5901234123458")
} except BarcodeError as error {
    write(error.message)
}
QR.create("")
`
	var output, errors bytes.Buffer
	Run(strings.NewReader(input), &output, &errors, "AhdCode v1.3.0")
	want := []string{
		"Q 33\n",
		"EAN13 5901234123457 95\n",
		"Barcode.ean13 check digit 8 is wrong for 590123412345; the correct check digit is 7\n",
	}
	rest := output.String()
	for _, line := range want {
		index := strings.Index(rest, line)
		if index < 0 {
			t.Fatalf("REPL output missing %q in order:\n%s\nerrors:\n%s", line, output.String(), errors.String())
		}
		rest = rest[index+len(line):]
	}
	if !strings.Contains(errors.String(), "QRError") || !strings.Contains(errors.String(), "QR.create value must not be empty") {
		t.Fatalf("uncaught QRError not reported by the REPL: %q", errors.String())
	}
	for _, path := range []string{qrPath, barPath} {
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Fatalf("REPL did not write %s: %v", path, err)
		}
	}
}
