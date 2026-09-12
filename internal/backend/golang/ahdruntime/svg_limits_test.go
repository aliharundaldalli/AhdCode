package ahdruntime

import (
	"strings"
	"testing"
	"time"
)

// Hostile or oversized SVG input is rejected quickly with a bounded amount of
// work, before any drawing source is produced.
func TestSVGResourceLimitsRejectHostileInput(t *testing.T) {
	head := `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10">`
	cases := []struct {
		name    string
		data    string
		problem string
	}{
		{"oversized file", head + `<!--` + strings.Repeat("x", 5<<20) + `--></svg>`, "test: SVG is larger than 5 MiB"},
		{"deep nesting", head + strings.Repeat("<g>", 65) + strings.Repeat("</g>", 65) + `</svg>`, "test: SVG elements are nested more than 64 levels deep"},
		{"element flood", head + strings.Repeat("<g/>", 100001) + `</svg>`, "test: SVG has more than 100000 elements"},
		{"path segment flood", head + `<path d="M0 0` + strings.Repeat("h0", 2000001) + `"/></svg>`, "test: <path>: path has more than 2000000 segments"},
		{"entity expansion", `<?xml version="1.0"?><!DOCTYPE svg [<!ENTITY a "aaaaaaaaaa"><!ENTITY b "&a;&a;&a;&a;&a;&a;&a;&a;&a;&a;">]>` + head + `<title>&b;</title></svg>`, "test: SVG must not contain a DOCTYPE or entity declarations"},
	}
	for _, test := range cases {
		started := time.Now()
		_, problem := AhdSVGConvert("test", []byte(test.data))
		if !strings.HasPrefix(problem, test.problem) {
			t.Fatalf("%s: problem %q, want %q", test.name, problem, test.problem)
		}
		if elapsed := time.Since(started); elapsed > 10*time.Second {
			t.Fatalf("%s took %s", test.name, elapsed)
		}
	}
	// A <use> that refers to itself, directly or through a chain, stops
	// instead of expanding forever.
	for _, cycle := range []string{
		head + `<defs><g id="a"><use href="#a"/></g></defs><use href="#a"/></svg>`,
		head + `<defs><g id="a"><use href="#b"/></g><g id="b"><use href="#a"/></g></defs><use href="#a"/></svg>`,
	} {
		started := time.Now()
		if _, problem := AhdSVGConvert("test", []byte(cycle)); problem == "" {
			t.Fatalf("a recursive <use> was accepted: %s", cycle)
		}
		if elapsed := time.Since(started); elapsed > 2*time.Second {
			t.Fatalf("a recursive <use> took %s", elapsed)
		}
	}
}
