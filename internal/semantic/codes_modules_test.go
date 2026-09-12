package semantic

import (
	"strings"
	"testing"
)

const codesPreamble = "bring QR\nbring Barcode\nfrom QR bring (QRCode, QRError)\nfrom Barcode bring (BarcodeCode, BarcodeError)\n\n"

func TestCodesModulesRegistered(t *testing.T) {
	cases := map[string][]string{
		"QR":      {"create", "QRCode", "QRError"},
		"Barcode": {"code128", "ean13", "upca", "BarcodeCode", "BarcodeError"},
	}
	for name, exports := range cases {
		module, ok := StandardModuleInterfaces()[name]
		if !ok {
			t.Fatalf("%s module not registered", name)
		}
		if module.ModuleID != "builtin:"+name {
			t.Fatalf("%s identity = %q", name, module.ModuleID)
		}
		for _, export := range exports {
			if module.Exports[export] == nil {
				t.Fatalf("%s missing export %q", name, export)
			}
		}
		if len(module.Exports) != len(exports) {
			t.Fatalf("%s exports %v, want exactly %v", name, module.ExportNames, exports)
		}
	}
	// v1.3.0 decodes nothing and supports exactly three barcode symbologies.
	for _, name := range []string{"decode", "read", "scan"} {
		if StandardModuleInterfaces()["QR"].Exports[name] != nil {
			t.Fatalf("QR must not export %q", name)
		}
	}
	for _, name := range []string{"code39", "ean8", "upce", "pdf417", "datamatrix"} {
		if StandardModuleInterfaces()["Barcode"].Exports[name] != nil {
			t.Fatalf("Barcode must not export %q", name)
		}
	}
}

func TestCodesModulesValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, codesPreamble+`code: QRCode := QR.create("https://ahdcode.org")
strong: QRCode := QR.create("https://ahdcode.org", "H")
text: String := code.value() + strong.level()
side: Int := code.size()
matrix: List<List<Bool>> := code.matrix()
first: Bool := matrix[0][0]
code.savePNG("qr.png")
code.savePNG("qr.png", 1024)
code.saveSVG("qr.svg")
code.saveSVG("qr.svg", 3.5)
bar: BarcodeCode := Barcode.code128("AHD-42")
ean: BarcodeCode := Barcode.ean13("590123412345")
upc: BarcodeCode := Barcode.upca("03600029145")
kind: String := bar.kind() + ean.value() + upc.value()
bars: List<Bool> := upc.pattern()
bar.savePNG("bar.png")
bar.savePNG("bar.png", 900, 300)
bar.saveSVG("bar.svg", 6.0, 2.0)
attempt {
    QR.create("")
} except QRError as error {
    write(error.message)
}
attempt {
    Barcode.ean13("1")
} except BarcodeError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}

// Static mistakes are compiler diagnostics, never a runtime QRError or
// BarcodeError.
func TestCodesRejectStaticMistakes(t *testing.T) {
	for _, source := range []string{
		`QR.create(42)`,
		`QR.create("x", 1)`,
		`QR.create()`,
		`Barcode.ean13(590123412345)`,
		`Barcode.code128()`,
		`QR.create("x").savePNG()`,
		`QR.create("x").savePNG("a.png", 1.5)`,
		`QR.create("x").saveSVG("a.svg", "big")`,
		`QR.create("x").value(1)`,
		`Barcode.upca("03600029145").savePNG("a.png", 1, 2, 3)`,
		`Barcode.upca("03600029145").saveSVG(1)`,
		`size: String := QR.create("x").size()`,
		`bars: List<Int> := Barcode.code128("x").pattern()`,
		`QR.create("x").decode()`,
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, codesPreamble+source+"\n"))
	}
}

func semanticHintsOf(result Result) string {
	var hints []string
	for _, diagnostic := range result.Diagnostics {
		hints = append(hints, diagnostic.Hint)
	}
	return strings.Join(hints, "\n")
}

func TestCodesValuesAreNotConstructedDirectly(t *testing.T) {
	for _, source := range []string{`QRCode("x")`, `BarcodeCode("EAN13:5901234123457")`} {
		result := analyzeWithStandardModules(t, codesPreamble+source+"\n")
		requireSemanticFailure(t, result)
		if !strings.Contains(semanticHintsOf(result), "create a") {
			t.Fatalf("direct construction of %s has no creation hint", source)
		}
	}
}
