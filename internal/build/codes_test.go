package build

import (
	"bytes"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// codesProgram exercises every QR and Barcode operation natively, including
// the documented runtime failures.
const codesProgram = `bring QR
bring Barcode
from QR bring (QRCode, QRError)
from Barcode bring (BarcodeCode, BarcodeError)

code: QRCode := QR.create("https://ahdcode.org/verify?id=AHD-2026-0042&lang=tr#top")
write(code.value())
write(code.level() + " " + str(code.size()))
matrix: List<List<Bool>> := code.matrix()
write(str(len(matrix)) + " " + str(matrix[0][0]) + " " + str(matrix[7][7]))
code.savePNG("qr.png")
code.saveSVG("qr.svg", 4.0)
strong: QRCode := QR.create("Doğrulama: Ayşe Yılmaz 😊", "H")
strong.savePNG("qr-h.png", 600)
write(strong.level() + " " + str(strong.size()))

ean: BarcodeCode := Barcode.ean13("590123412345")
write(ean.kind() + " " + ean.value() + " " + str(len(ean.pattern())))
ean.savePNG("ean.png")
ean.saveSVG("ean.svg")
upc: BarcodeCode := Barcode.upca("036000291452")
write(upc.kind() + " " + upc.value())
asset: BarcodeCode := Barcode.code128("AHD-Asset 00042/b")
write(asset.kind() + " " + asset.value())
asset.savePNG("code128.png", 900, 260)

attempt {
    Barcode.ean13("5901234123458")
} except BarcodeError as error {
    write(error.message)
}
attempt {
    Barcode.upca("0360002914")
} except BarcodeError as error {
    write(error.message)
}
attempt {
    Barcode.code128("çay")
} except BarcodeError as error {
    write(error.message)
}
attempt {
    QR.create("x", "m")
} except QRError as error {
    write(error.message)
}
attempt {
    code.savePNG("small.png", 10)
} except QRError as error {
    write(error.message)
}
attempt {
    code.saveSVG("wrong.png")
} except QRError as error {
    write(error.message)
}
`

const codesExpected = `https://ahdcode.org/verify?id=AHD-2026-0042&lang=tr#top
M 33
33 true false
H 33
EAN13 5901234123457 95
UPCA 036000291452
Code128 AHD-Asset 00042/b
Barcode.ean13 check digit 8 is wrong for 590123412345; the correct check digit is 7
Barcode.upca value must have 11 digits (the check digit is computed) or 12 digits (the check digit is verified); received 10
Barcode.code128 value must contain only ASCII characters; received 'ç' at position 1
QR.create level must be L, M, Q, or H; received "m"
QRCode.savePNG pixels must be between 41 and 10000 for this symbol; received 10
QRCode.saveSVG destination must use the .svg extension
`

func TestCodesRunThroughNativeBackend(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": codesProgram})
	output := filepath.Join(t.TempDir(), "codes")
	if _, result := BuildProgram(filepath.Join(directory, "main.ahd"), output); result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	work := t.TempDir()
	command := exec.Command(output)
	command.Dir = work
	var stdout, stderr strings.Builder
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil || stderr.String() != "" {
		t.Fatalf("codes program failed: %v %q", err, stderr.String())
	}
	if stdout.String() != codesExpected {
		t.Fatalf("codes native output mismatch:\n got: %q\nwant: %q", stdout.String(), codesExpected)
	}
	for _, name := range []string{"qr.png", "qr-h.png", "ean.png", "code128.png"} {
		data, err := os.ReadFile(filepath.Join(work, name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
			t.Fatalf("%s is not a PNG: %v", name, err)
		}
	}
	for _, name := range []string{"qr.svg", "ean.svg"} {
		data, err := os.ReadFile(filepath.Join(work, name))
		if err != nil || !bytes.Contains(data, []byte("<svg ")) {
			t.Fatalf("%s is not SVG: %v", name, err)
		}
	}
	for _, name := range []string{"small.png", "wrong.png"} {
		if _, err := os.Stat(filepath.Join(work, name)); err == nil {
			t.Fatalf("a failed save left %s behind", name)
		}
	}
}

func TestCodesUncaughtErrorsAreAhdCodeLevel(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "bring Barcode\nwrite(Barcode.ean13(\"12345\").value())\n"})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code == 0 {
		t.Fatalf("expected a failing exit code (stdout=%q)", stdout)
	}
	assertNoGoInternals(t, stderr)
	if !containsAll(stderr, "BarcodeError", "Barcode.ean13 value must have 12 digits") {
		t.Fatalf("uncaught BarcodeError not reported at AhdCode level: %q", stderr)
	}
}

// Sibling QR.ahd and Barcode.ahd files cannot hijack the builtin modules.
func TestCodesBuiltinsWinOverSiblingFiles(t *testing.T) {
	directory := writeSources(t, map[string]string{
		"QR.ahd":      "create: Function := (value: String) -> String {\n    return \"shadow\"\n}\n",
		"Barcode.ahd": "ean13: Function := (value: String) -> String {\n    return \"shadow\"\n}\n",
		"main.ahd":    "bring QR\nbring Barcode\nwrite(QR.create(\"x\").level())\nwrite(Barcode.ean13(\"590123412345\").value())\n",
	})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stdout != "M\n5901234123457\n" {
		t.Fatalf("sibling files shadowed the builtins: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

// A program using both MySQL and codes vendors both pinned trees in one
// workspace and still builds without the network.
func TestCodesAndMySQLComposeOneVendoredWorkspace(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `bring MySQL
bring QR
from MySQL bring MySQLValue

value: MySQLValue := MySQL.fromInt(7)
write(QR.create("mysql and codes").level())
`})
	program := Compile(filepath.Join(directory, "main.ahd"))
	if program.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(program.Diagnostics))
	}
	if !program.Program.RequiresMySQL || !program.Program.RequiresCodes {
		t.Fatalf("program flags: mysql=%v codes=%v", program.Program.RequiresMySQL, program.Program.RequiresCodes)
	}
	workspace, diagnostics := NewWorkspace(program.Program)
	if len(diagnostics) != 0 {
		t.Fatalf("workspace failed: %s", diagnosticText(diagnostics))
	}
	defer workspace.Close()
	goMod, _ := os.ReadFile(filepath.Join(workspace.Directory, "go.mod"))
	modules, _ := os.ReadFile(filepath.Join(workspace.Directory, "vendor", "modules.txt"))
	for _, want := range []string{"github.com/boombuler/barcode v1.1.0", "github.com/go-sql-driver/mysql v1.10.1", "filippo.io/edwards25519 v1.2.0 // indirect"} {
		if !strings.Contains(string(goMod), want) {
			t.Fatalf("composed go.mod lacks %q:\n%s", want, goMod)
		}
	}
	if !strings.HasPrefix(string(modules), "# filippo.io/edwards25519") || !strings.Contains(string(modules), "# github.com/boombuler/barcode v1.1.0\n## explicit\n") {
		t.Fatalf("composed modules.txt:\n%s", modules)
	}
	output := filepath.Join(t.TempDir(), "composed")
	if diagnostics := workspace.BuildExecutable(output); len(diagnostics) != 0 {
		t.Fatalf("composed vendored build failed: %s", diagnosticText(diagnostics))
	}
}

// A built codes program needs neither its source nor the repository: moved to
// an unrelated directory, it still writes decodable symbols there.
func TestCodesNativeExecutableRelocates(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "bring QR\nbring Barcode\nQR.create(\"relocated\").savePNG(\"moved.png\")\nBarcode.code128(\"MOVED-1\").saveSVG(\"moved.svg\")\nwrite(\"ok\")\n"})
	built := filepath.Join(t.TempDir(), "codes")
	if _, result := BuildProgram(filepath.Join(directory, "main.ahd"), built); result.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(result.Diagnostics))
	}
	if err := os.RemoveAll(directory); err != nil {
		t.Fatal(err)
	}
	elsewhere := t.TempDir()
	moved := filepath.Join(elsewhere, "relocated-codes")
	if err := os.Rename(built, moved); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(moved)
	command.Dir = elsewhere
	command.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + elsewhere}
	output, err := command.CombinedOutput()
	if err != nil || string(output) != "ok\n" {
		t.Fatalf("relocated program failed: %v %q", err, output)
	}
	for _, name := range []string{"moved.png", "moved.svg"} {
		if info, err := os.Stat(filepath.Join(elsewhere, name)); err != nil || info.Size() == 0 {
			t.Fatalf("relocated program did not write %s", name)
		}
	}
}
