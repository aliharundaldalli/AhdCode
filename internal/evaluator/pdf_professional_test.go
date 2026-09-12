package evaluator

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"ahdcode/internal/backend/golang/ahdruntime"
)

func pdfEvaluatorBlocks(t *testing.T, value any) []string {
	t.Helper()
	items := value.(*Instance).Fields[pdfBlocksField].(*List).Items
	blocks := make([]string, len(items))
	for index, item := range items {
		blocks[index] = item.(string)
	}
	return blocks
}

func pdfEvaluatorPair(keys []string, values []float64) *Pair {
	pair := &Pair{Values: map[any]any{}}
	for index, key := range keys {
		pair.Keys = append(pair.Keys, key)
		pair.Values[key] = values[index]
	}
	return pair
}

// The evaluator and a compiled program store byte-identical PDFDocument
// blocks for every v1.3.0 operation, and a later v1.2.0 operation keeps them.
func TestPDFProfessionalBlocksMatchNativeRuntime(t *testing.T) {
	directory := t.TempDir()
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 8, 4))); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(directory, "a.png")
	svgPath := filepath.Join(directory, "a.svg")
	if os.WriteFile(pngPath, encoded.Bytes(), 0o600) != nil ||
		os.WriteFile(svgPath, []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="5"><rect width="10" height="5"/></svg>`), 0o600) != nil {
		t.Fatal("could not write fixtures")
	}

	session := newLatexTestSession()
	doc := session.pdfBuiltin("new", nil)
	step := func(name string, args ...any) {
		doc = session.pdfOperation("PDFDocument."+name, doc, args)
	}
	step("layout", "Custom", true, pdfEvaluatorPair([]string{"width", "height"}, []float64{10, 6}), nil)
	step("header", "L", nil, nil)
	step("footer", "L", "C", "R")
	step("pageNumbers", nil, nil)
	step("qr", "https://ahdcode.org", nil, nil, nil)
	step("qr", "x", int64(2), "H", "right")
	step("barcode", "UPCA", "03600029145", nil, nil, nil)
	step("link", "Site", "mailto:info@ahdcode.org", nil)
	step("bookmark", "Intro", nil)
	step("metadata", "Title", nil, nil, &List{Items: []any{"a", "b"}}, "AhdCode")
	step("metadata", "Title", nil, nil, nil, nil)
	step("image", pngPath, nil, nil)
	step("image", svgPath, pdfEvaluatorPair([]string{"height"}, []float64{2}), pdfEvaluatorPair([]string{"trimLeft", "opacity"}, []float64{0.5, 0.25}))
	step("heading", "After", int64(1))

	native := ahdruntime.AhdPDFLayout(ahdruntime.AhdPDFNew(), "Custom", true,
		ahdruntime.AhdBuildPair([]string{"width", "height"}, []float64{10, 6}), nil)
	native = ahdruntime.AhdPDFRunning(native, "header", "L", "", "")
	native = ahdruntime.AhdPDFRunning(native, "footer", "L", "C", "R")
	native = ahdruntime.AhdPDFPageNumbers(native, "center", false)
	native = ahdruntime.AhdPDFQR(native, "https://ahdcode.org", 3, "M", "center")
	native = ahdruntime.AhdPDFQR(native, "x", 2, "H", "right")
	native = ahdruntime.AhdPDFBarcode(native, "UPCA", "03600029145", 8, 2, "center")
	native = ahdruntime.AhdPDFLink(native, "Site", "mailto:info@ahdcode.org", "left")
	native = ahdruntime.AhdPDFBookmark(native, "Intro", 1)
	native = ahdruntime.AhdPDFMetadata(native, "Title", "", "", ahdruntime.AhdNewList("a", "b"), "AhdCode")
	native = ahdruntime.AhdPDFMetadata(native, "Title", "", "", ahdruntime.AhdNewList[string](), "")
	native = ahdruntime.AhdPDFImageComplete(native, pngPath, nil, nil)
	native = ahdruntime.AhdPDFImageComplete(native, svgPath, ahdruntime.AhdBuildPair([]string{"height"}, []float64{2}),
		ahdruntime.AhdBuildPair([]string{"trimLeft", "opacity"}, []float64{0.5, 0.25}))
	native = ahdruntime.AhdPDFHeading(native, "After", 1)

	got := pdfEvaluatorBlocks(t, doc)
	if len(got) != len(native.Blocks) {
		t.Fatalf("evaluator stored %d blocks; native stored %d", len(got), len(native.Blocks))
	}
	for index := range got {
		if got[index] != native.Blocks[index] {
			t.Fatalf("block %d differs\nevaluator %s\nnative    %s", index, got[index], native.Blocks[index])
		}
	}
}

func TestPDFProfessionalErrorsMatchNativeRuntime(t *testing.T) {
	session := newLatexTestSession()
	doc := session.pdfBuiltin("new", nil)
	cases := []struct {
		name    string
		args    []any
		problem string
	}{
		{"layout", []any{"a4", nil, nil, nil}, second(ahdruntime.AhdPDFLayoutBlock("a4", false, nil, nil, nil, nil))},
		{"header", []any{"a\nb", nil, nil}, second(ahdruntime.AhdPDFRunningBlock("header", "a\nb", "", ""))},
		{"pageNumbers", []any{"middle", nil}, second(ahdruntime.AhdPDFPageNumbersBlock("middle", false))},
		{"qr", []any{"x", 3.0, "Z", nil}, second(ahdruntime.AhdPDFQRBlock("x", 3, "Z", "center"))},
		{"barcode", []any{"EAN13", "12", nil, nil, nil}, second(ahdruntime.AhdPDFBarcodeBlock("EAN13", "12", 8, 2, "center"))},
		{"link", []any{"x", "javascript:alert(1)", nil}, second(ahdruntime.AhdPDFLinkBlock("x", "javascript:alert(1)", "left"))},
		{"bookmark", []any{"x", int64(0)}, second(ahdruntime.AhdPDFBookmarkBlock("x", 0))},
		{"metadata", []any{"t", nil, nil, &List{Items: []any{" "}}, nil}, second(ahdruntime.AhdPDFMetadataBlock("t", "", "", []string{" "}, ""))},
		{"image", []any{"missing-image.svg", nil, nil}, second(ahdruntime.AhdPDFImageBlock("missing-image.svg", nil, nil, nil, nil))},
	}
	for _, test := range cases {
		if test.problem == "" {
			t.Fatalf("%s: the runtime accepted the invalid case", test.name)
		}
		message := evaluatorRaisedMessage(t, "PDFError", func() { session.pdfOperation("PDFDocument."+test.name, doc, test.args) })
		if message != test.problem {
			t.Fatalf("%s\n got %q\nwant %q", test.name, message, test.problem)
		}
	}
}

func second(_ string, problem string) string { return problem }
