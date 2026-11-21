package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWebPackageCLIRequiresExplicitWebMode(t *testing.T) {
	var output, diagnostics bytes.Buffer
	code := runWithIO([]string{"package", "app.ahd", "--pwa"}, strings.NewReader(""), &output, &diagnostics)
	if code != 2 || !strings.Contains(diagnostics.String(), "--pwa requires --web") {
		t.Fatalf("exit %d, diagnostics %q", code, diagnostics.String())
	}
}
