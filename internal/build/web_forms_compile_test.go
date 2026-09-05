package build

import (
	"os"
	"path/filepath"
	"testing"
)

// Compile only: native HTTP behaviour is covered by tools/qa/v016_forms.py
// against a reused candidate binary, avoiding a Go build for every case.
func TestWebV016ApplicationsCompile(t *testing.T) {
	for _, entry := range []string{
		"testdata/v016/acceptance.ahd",
		"../../examples/v0.16/forms_validation/app.ahd",
		"../../examples/v0.15/ahd_math_portal/app.ahd",
		"../../examples/v0.5/02_session_counter.ahd",
		"../../examples/v0.5/03_session_login.ahd",
	} {
		path, err := filepath.Abs(entry)
		if err != nil {
			t.Fatal(err)
		}
		result := Compile(path)
		if result.HasErrors() {
			t.Fatalf("%s:\n%s", entry, diagnosticText(result.Diagnostics))
		}
	}
}

func TestWebV016ScratchDogfoodCompile(t *testing.T) {
	entry := os.Getenv("AHD_V016_DOGFOOD")
	if entry == "" {
		t.Skip("set AHD_V016_DOGFOOD to the temporary portal app.ahd")
	}
	result := Compile(entry)
	if result.HasErrors() {
		t.Fatal(diagnosticText(result.Diagnostics))
	}
}
