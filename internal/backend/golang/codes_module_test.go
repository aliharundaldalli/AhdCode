package golang

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// The codes runtime imports the vendored encoder, so it reaches a generated
// program only when the program encodes a symbol.
func TestCodesRuntimeOnlyJoinsProgramsThatUseCodes(t *testing.T) {
	plain := generate(t, "write(\"hi\")\n")
	for _, file := range plain.Files {
		if file.Name == codesRuntimeFileName {
			t.Fatal("a program without codes received the codes runtime")
		}
	}
	if plain.RequiresCodes {
		t.Fatal("a program without codes requires the vendored encoder")
	}
	for _, source := range []string{
		"bring QR\nwrite(QR.create(\"x\").value())\n",
		"bring Barcode\nwrite(Barcode.ean13(\"590123412345\").value())\n",
	} {
		program := generate(t, source)
		if !program.RequiresCodes {
			t.Fatalf("program does not require codes:\n%s", source)
		}
		seen := 0
		for _, file := range program.Files {
			if file.Name != codesRuntimeFileName {
				continue
			}
			seen++
			parsed, err := parser.ParseFile(token.NewFileSet(), file.Name, file.Content, 0)
			if err != nil {
				t.Fatalf("codes runtime is not valid Go: %v", err)
			}
			if parsed.Name.Name != "main" {
				t.Fatalf("codes runtime package clause not rewritten: %s", parsed.Name.Name)
			}
			// No process execution or network access can hide in the encoder
			// path: the only non-standard imports are the vendored packages.
			for _, spec := range parsed.Imports {
				path, _ := strconv.Unquote(spec.Path.Value)
				if strings.Contains(path, ".") && !strings.HasPrefix(path, "github.com/boombuler/barcode") {
					t.Fatalf("codes runtime imports %s", path)
				}
				for _, forbidden := range []string{"os/exec", "net", "net/http", "syscall", "unsafe"} {
					if path == forbidden {
						t.Fatalf("codes runtime imports %s", path)
					}
				}
			}
		}
		if seen != 1 {
			t.Fatalf("codes runtime emitted %d times, want 1", seen)
		}
	}
}

func TestCodesLoweringEmitsRuntimeCalls(t *testing.T) {
	source := programSource(t, generate(t, `bring QR
bring Barcode
from QR bring QRCode
from Barcode bring BarcodeCode

code: QRCode := QR.create("https://ahdcode.org", "H")
write(code.value() + code.level())
write(str(code.size()))
rows: List<List<Bool>> := code.matrix()
code.savePNG("qr.png")
code.saveSVG("qr.svg", 4.0)
bar: BarcodeCode := Barcode.upca("03600029145")
write(bar.kind() + bar.value())
bars: List<Bool> := bar.pattern()
bar.savePNG("bar.png", 900)
bar.saveSVG("bar.svg")
write(Barcode.code128("A").value() + Barcode.ean13("590123412345").value())
`))
	for _, call := range []string{
		"AhdQRCreate(", `"H")`, "AhdQRValue(", "AhdQRLevel(", "AhdQRSize(", "AhdQRMatrix(", "AhdQRSavePNG(", "int64(512)",
		"AhdQRSaveSVG(", "AhdBarcodeCreate(", `"UPCA"`, `"Code128"`, `"EAN13"`, "AhdBarcodeKind(", "AhdBarcodeValue(",
		"AhdBarcodePattern(", "AhdBarcodeSavePNG(", "int64(240)", "AhdBarcodeSaveSVG(", "8.0, 2.4)",
	} {
		if !strings.Contains(source, call) {
			t.Fatalf("generated program does not contain %s", call)
		}
	}
}
