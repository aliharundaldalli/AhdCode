package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePNG(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	path := filepath.Join(t.TempDir(), "chart-test.png")
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func handshake(t *testing.T, out *bytes.Buffer) reply {
	t.Helper()
	var value reply
	line := strings.TrimSpace(out.String())
	if strings.Count(line, "\n") != 0 || json.Unmarshal([]byte(line), &value) != nil {
		t.Fatalf("handshake %q is not one JSON line", out.String())
	}
	return value
}

func TestArgumentsAreBounded(t *testing.T) {
	for _, args := range [][]string{nil, {"only"}, {"a", "b", "c"}, {"", "t"}, {strings.Repeat("p", maxPathBytes+1), "t"},
		{"x.png", strings.Repeat("t", maxTitleRunes+1)}, {"x.png", "bad\x00title"}, {"x.png", string([]byte{0xff})}} {
		if run(args, &bytes.Buffer{}) == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}

func TestPreviewIsReadCompletelyAndDeleted(t *testing.T) {
	path := writePNG(t, 300, 200)
	img, err := loadPreview(path)
	if err != nil || img.Bounds().Dx() != 300 || img.Bounds().Dy() != 200 {
		t.Fatalf("load: %v %v", err, img)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("the preview file was not deleted after loading")
	}
}

func TestBadPreviewsAreErrorsNotPanics(t *testing.T) {
	if _, err := loadPreview(filepath.Join(t.TempDir(), "missing.png")); err == nil {
		t.Fatal("a missing preview")
	}
	broken := filepath.Join(t.TempDir(), "broken.png")
	_ = os.WriteFile(broken, []byte("not a png"), 0o600)
	if _, err := loadPreview(broken); err == nil || !strings.Contains(err.Error(), "not a valid PNG") {
		t.Fatalf("a malformed preview: %v", err)
	}
	if _, err := os.Stat(broken); !os.IsNotExist(err) {
		t.Fatal("a malformed preview was left behind")
	}
	if _, err := loadPreview(t.TempDir()); err == nil {
		t.Fatal("a directory as preview")
	}
	huge := writePNG(t, maxSide+1, 1)
	if _, err := loadPreview(huge); err == nil || !strings.Contains(err.Error(), "at most 8192") {
		t.Fatalf("an oversized preview: %v", err)
	}
}

func TestHeadlessReplayReportsEveryStep(t *testing.T) {
	report := filepath.Join(t.TempDir(), "report.json")
	t.Setenv("AHDCODE_PLOTVIEW_HEADLESS", "1")
	t.Setenv("AHDCODE_PLOTVIEW_HEADLESS_REPORT", report)
	t.Setenv("AHDCODE_PLOTVIEW_HEADLESS_SCRIPT", `[{"action":"zoom","factor":2,"x":600,"y":300},{"action":"pan","x":-50,"y":0},`+
		`{"action":"rotateLeft"},{"action":"rotateRight"},{"action":"rotateRight"},{"action":"reset"},{"action":"close"},{"action":"zoom","factor":2}]`)
	var out bytes.Buffer
	if err := run([]string{writePNG(t, 1600, 1200), "Test"}, &out); err != nil {
		t.Fatal(err)
	}
	if !handshake(t, &out).Ready {
		t.Fatal("not ready")
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var states []state
	if json.Unmarshal(data, &states) != nil || len(states) != 8 {
		t.Fatalf("report %s", data)
	}
	if states[0].Zoom != 0.5 || states[1].Zoom != 1 || states[3].Rotation != 90 || states[5].Rotation != 270 {
		t.Fatalf("states %+v", states)
	}
	if states[6].Rotation != 0 || states[6].Zoom != 0.5 || !states[7].Closed {
		t.Fatalf("reset and close %+v", states[6:])
	}
	t.Setenv("AHDCODE_PLOTVIEW_HEADLESS_SCRIPT", `[{"action":"explode"}]`)
	if run([]string{writePNG(t, 10, 10), ""}, &bytes.Buffer{}) == nil {
		t.Fatal("an unknown headless action")
	}
}

func TestInitialWindowSize(t *testing.T) {
	for _, c := range []struct{ w, h, bw, bh, ww, wh int }{
		{800, 600, 1400, 1000, 800, 600},
		{100, 50, 1400, 1000, 480, 360},
		{3000, 2400, 1400, 1000, 1400, 1000},
		{3000, 2400, 100, 100, 480, 360},
	} {
		if w, h := initialWindow(c.w, c.h, c.bw, c.bh); w != c.ww || h != c.wh {
			t.Fatalf("initialWindow(%d,%d,%d,%d) = %d,%d", c.w, c.h, c.bw, c.bh, w, h)
		}
	}
}
