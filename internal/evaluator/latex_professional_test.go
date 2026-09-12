package evaluator

import (
	"testing"

	"ahdcode/internal/backend/golang/ahdruntime"
)

// Every v1.3.0 Latex helper is built by the shared runtime, so the evaluator
// returns exactly the fragment a compiled program returns.
func TestLatexProfessionalFragmentsMatchNativeRuntime(t *testing.T) {
	session := newLatexTestSession()
	pair := func(keys []string, values []float64) *Pair {
		result := &Pair{Values: map[any]any{}}
		for index, key := range keys {
			result.Keys = append(result.Keys, key)
			result.Values[key] = values[index]
		}
		return result
	}
	expect := func(text, problem string) string {
		if problem != "" {
			t.Fatal(problem)
		}
		return text
	}
	cases := []struct {
		name string
		args []any
		want string
	}{
		{"qr", []any{"https://ahdcode.org", nil, nil}, expect(ahdruntime.AhdLatexQRText("Latex.qr", "https://ahdcode.org", 3, "M"))},
		{"qr", []any{"x", 2.5, "H"}, expect(ahdruntime.AhdLatexQRText("Latex.qr", "x", 2.5, "H"))},
		{"barcode", []any{"EAN13", "590123412345", nil, nil}, expect(ahdruntime.AhdLatexBarcodeText("Latex.barcode", "EAN13", "590123412345", 8, 2))},
		{"place", []any{"X", 2.0, int64(3), "south east"}, expect(ahdruntime.AhdLatexPlaceText("X", 2, 3, "south east"))},
		{"place", []any{"X", 1.0, 1.0, nil}, expect(ahdruntime.AhdLatexPlaceText("X", 1, 1, "north west"))},
		{"header", []any{"L", nil, "R"}, ahdruntime.AhdLatexRunningText("head", "L", "", "R")},
		{"footer", []any{nil, "C", nil}, ahdruntime.AhdLatexRunningText("foot", "", "C", "")},
		{"pageNumber", nil, ahdruntime.AhdLatexPageNumberText()},
		{"pageCount", nil, ahdruntime.AhdLatexPageCountText()},
		{"link", []any{"site", "https://ahdcode.org/?a=1&b=2"}, expect(ahdruntime.AhdLatexLinkText("Latex.link", "site", "https://ahdcode.org/?a=1&b=2"))},
		{"bookmark", []any{"Giriş", nil}, expect(ahdruntime.AhdLatexBookmarkText("Latex.bookmark", "Giriş", 1))},
		{"bookmark", []any{"Alt", int64(3)}, expect(ahdruntime.AhdLatexBookmarkText("Latex.bookmark", "Alt", 3))},
		{"image", []any{"logo.svg", pair([]string{"width"}, []float64{4}), pair([]string{"opacity", "rotation"}, []float64{0.5, 30})},
			expect(ahdruntime.AhdLatexImageText("logo.svg", []string{"width"}, []float64{4}, []string{"opacity", "rotation"}, []float64{0.5, 30}, false, "", ""))},
		{"image", []any{"logo.png", nil, nil}, expect(ahdruntime.AhdLatexImageText("logo.png", nil, nil, nil, nil, false, "", ""))},
		{"figure", []any{"chart.png", "Chart", "fig:c", pair([]string{"width"}, []float64{12}), pair([]string{"trimTop"}, []float64{1})},
			expect(ahdruntime.AhdLatexImageText("chart.png", []string{"width"}, []float64{12}, []string{"trimTop"}, []float64{1}, true, "Chart", "fig:c"))},
	}
	for _, test := range cases {
		if got := session.latexBuiltin(test.name, test.args).(string); got != test.want {
			t.Fatalf("%s(%v) differs from the native runtime\n got %q\nwant %q", test.name, test.args, got, test.want)
		}
	}
	document := session.latexBuiltin("document", []any{"Body", "Title", "Author", nil, nil, nil, nil, nil, nil, nil, true,
		"A3", nil, pair([]string{"left"}, []float64{3}), "Subject", &List{Items: []any{"a", "b"}}, "AhdCode"}).(string)
	want := ahdruntime.AhdLatexDocumentComplete("Body", "Title", "Author", "", "Article", 2.54, "", "", nil, "Default", true, "A3", nil,
		ahdruntime.AhdBuildPair([]string{"left"}, []float64{3}), "Subject", ahdruntime.AhdNewList("a", "b"), "AhdCode")
	if document != want {
		t.Fatalf("document differs from the native runtime\n got %q\nwant %q", document, want)
	}
}

func TestLatexProfessionalValidationMatchesNativeRuntime(t *testing.T) {
	session := newLatexTestSession()
	cases := []struct {
		name    string
		args    []any
		message string
	}{
		{"qr", []any{"", nil, nil}, "Latex.qr value must not be empty"},
		{"qr", []any{"x", 0.0, nil}, "Latex.qr size must be greater than 0 and at most 1000.0 centimeters; received 0.0"},
		{"barcode", []any{"Code39", "X", nil, nil}, `Latex.barcode kind must be Code128, EAN13, or UPCA; received "Code39"`},
		{"place", []any{"X", 1.0, 1.0, "middle"}, `Latex.place anchor must be north west, north, north east, west, center, east, south west, south, or south east; received "middle"`},
		{"link", []any{"x", "javascript:alert(1)"}, `Latex.link url must start with https://, http://, or mailto:; received "javascript:alert(1)"`},
		{"bookmark", []any{"x", int64(9)}, "Latex.bookmark level must be between 1 and 4; received 9"},
		{"document", []any{"b", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "Tabloid"}, `Latex.document paper must be A3, A4, A5, Letter, Legal, or Custom; received "Tabloid"`},
	}
	for _, test := range cases {
		message := evaluatorRaisedMessage(t, "ValueError", func() { session.latexBuiltin(test.name, test.args) })
		if message != test.message {
			t.Fatalf("%s message\n got %q\nwant %q", test.name, message, test.message)
		}
	}
}
