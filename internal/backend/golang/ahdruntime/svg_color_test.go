package ahdruntime

import (
	"strings"
	"testing"
)

// A color's own alpha is never silently dropped: a translucent color is
// rejected, and a fully opaque one converts like any other color.
func TestSVGTranslucentColorsAreRejectedNotDrawnOpaque(t *testing.T) {
	document := func(fill string) []byte {
		return []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="` + fill + `"/></svg>`)
	}
	for _, fill := range []string{"#0008", "#00000080", "rgba(0, 0, 0, 0.5)", "rgb(0 0 0 / 50%)", "rgba(10%, 20%, 30%, 0)"} {
		_, problem := AhdSVGConvert("test", document(fill))
		if !strings.Contains(problem, "is not a supported color") {
			t.Fatalf("translucent fill %q: problem %q", fill, problem)
		}
	}
	for _, fill := range []string{"#123", "#000F", "#000000FF", "rgba(0, 0, 0, 1)", "rgb(0 0 0 / 100%)", "rgb(10%, 20%, 30%)", "teal"} {
		if _, problem := AhdSVGConvert("test", document(fill)); problem != "" {
			t.Fatalf("opaque fill %q was rejected: %s", fill, problem)
		}
	}
}
