package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writePNG(t *testing.T, path string, width, height int, c color.NRGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

func TestImagesAndWrappingLabels(t *testing.T) {
	dir := t.TempDir()
	redPath := filepath.Join(dir, "red.png")
	writePNG(t, redPath, 20, 10, color.NRGBA{R: 220, A: 255})
	jpegPath := filepath.Join(dir, "red.jpg")
	writeJPEG(t, jpegPath)
	tallPath := filepath.Join(dir, "tall.png")
	writePNG(t, tallPath, 8193, 1, color.NRGBA{R: 1, A: 255})
	hugePath := filepath.Join(dir, "huge.png")
	huge, err := os.Create(hugePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := huge.Truncate(maxImageBytes + 1); err != nil {
		t.Fatal(err)
	}
	huge.Close()
	os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "bad.png"), []byte("not a png"), 0o644)
	os.Mkdir(filepath.Join(dir, "folder.png"), 0o755)

	m := newModel("t", 200, 160, fixedMeasure)
	root := mustAdd(t, m, 0, kindColumn, "")
	photo := addImage(t, m, root, redPath, 40, 40)
	if err := m.setColor(photo, partBackground, color.NRGBA{B: 200, A: 255}); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	box := widgetBox(m, photo)
	if box != [4]int{0, 0, 40, 40} {
		t.Fatalf("fitted image box %v", box)
	}
	img := render(frame{model: m, scale: 1})
	if img.RGBAAt(2, 2).B < 150 {
		t.Fatalf("letterbox pixel %#v", img.RGBAAt(2, 2))
	}
	if img.RGBAAt(20, 20).R < 150 {
		t.Fatalf("picture pixel %#v", img.RGBAAt(20, 20))
	}
	os.Remove(redPath)
	img = render(frame{model: m, scale: 1})
	if img.RGBAAt(20, 20).R < 150 {
		t.Fatal("image was decoded again after the file disappeared")
	}

	wide := addImage(t, m, root, jpegPath, 40, 0)
	onlyHeight := addImage(t, m, root, jpegPath, 0, 10)
	natural := addImage(t, m, root, jpegPath, 0, 0)
	layout(t, m)
	if sizeOf(m, wide) != [2]int{40, 20} {
		t.Fatalf("width-only box %v", widgetBox(m, wide))
	}
	if sizeOf(m, onlyHeight) != [2]int{20, 10} {
		t.Fatalf("height-only box %v", widgetBox(m, onlyHeight))
	}
	if sizeOf(m, natural) != [2]int{20, 10} {
		t.Fatalf("natural box %v", widgetBox(m, natural))
	}
	if err := m.setBox("fit", wide, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, wide) != [2]int{20, 10} {
		t.Fatalf("setSize reset %v", widgetBox(m, wide))
	}

	hidden := addImage(t, m, root, jpegPath, 0, 0)
	below := mustAdd(t, m, root, kindLabel, "L")
	layout(t, m)
	shown := widgetBox(m, below)[1]
	if err := m.setVisible(hidden, false); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if widgetBox(m, hidden)[3] != 0 || widgetBox(m, below)[1] >= shown {
		t.Fatalf("hidden image box %v label %v, label was at %d", widgetBox(m, hidden), widgetBox(m, below), shown)
	}
	if err := m.setEnabled(photo, false); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, photo) != [2]int{40, 40} {
		t.Fatalf("disabled image changed size %v", widgetBox(m, photo))
	}
	m.press(20, 20)
	if m.focus != 0 {
		t.Fatalf("disabled image took focus %d", m.focus)
	}

	for _, testCase := range []struct {
		path string
		fitW int
		want string
	}{
		{"", 0, "image needs a file path"},
		{filepath.Join(dir, "missing.png"), 0, "could not read the image: the file does not exist"},
		{filepath.Join(dir, "note.txt"), 0, "GUI images are .png, .jpg, and .jpeg files"},
		{filepath.Join(dir, "bad.png"), 0, "the file is not a PNG or JPEG image"},
		{filepath.Join(dir, "folder.png"), 0, "could not read the image: it is not a file"},
		{hugePath, 0, "the image file is larger than 64 MiB"},
		{tallPath, 0, "the image is too large; each side may be at most 8192 pixels"},
		{jpegPath, -1, "image width and height must be between 0 and 4096"},
	} {
		_, err := m.addSpec(root, spec{kind: kindImage, text: testCase.path, fitW: testCase.fitW, selected: -1})
		if err == nil || err.Error() != testCase.want {
			t.Fatalf("image %q fit %d: %v, want %s", testCase.path, testCase.fitW, err, testCase.want)
		}
	}
	if err := m.setBox("fit", photo, 4097, 0, "", ""); err == nil {
		t.Fatal("accepted an image size past 4096")
	}

	note := mustAdd(t, m, root, kindLabel, "hello world")
	if err := m.setBox("wrap", note, 1, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := m.setBox("preferred", note, 50, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, note) != [2]int{50, 64} {
		t.Fatalf("wrapped hello world %v", widgetBox(m, note))
	}
	m.widgets[note].text = []rune("one\ntwo")
	if err := m.setBox("preferred", note, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, note) != [2]int{30, 64} {
		t.Fatalf("explicit breaks %v", widgetBox(m, note))
	}
	m.widgets[note].text = []rune("a\r\nb")
	layout(t, m)
	if sizeOf(m, note) != [2]int{10, 64} {
		t.Fatalf("CRLF %v", widgetBox(m, note))
	}
	m.widgets[note].wrap = false
	m.widgets[note].text = []rune("a\nb")
	layout(t, m)
	if sizeOf(m, note) != [2]int{30, 32} {
		t.Fatalf("unwrapped newline %v", widgetBox(m, note))
	}
	m.widgets[note].wrap = true
	m.widgets[note].text = []rune("aaaaabbbbb")
	if err := m.setBox("preferred", note, 50, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, note) != [2]int{50, 64} {
		t.Fatalf("long word %v", widgetBox(m, note))
	}
	m.widgets[note].text = []rune("İstanbul")
	if err := m.setBox("preferred", note, 40, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, note) != [2]int{40, 64} {
		t.Fatalf("Turkish %v", widgetBox(m, note))
	}
	m.widgets[note].text = []rune("👍👍")
	if err := m.setBox("preferred", note, 10, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, note) != [2]int{10, 64} {
		t.Fatalf("emoji %v", widgetBox(m, note))
	}
	family := []rune("👨\u200d👩\u200d👧")
	lines := labelLines(fixedMeasure, family, "", false, 10)
	if len(lines) != 1 || lines[0] != string(family) {
		t.Fatalf("family emoji split into %#v", lines)
	}
	m.widgets[note].text = []rune("ab")
	m.widgets[note].labelSize = "small"
	m.widgets[note].bold = true
	if err := m.setBox("preferred", note, 80, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, note) != [2]int{80, 26} {
		t.Fatalf("small bold wrap %v", widgetBox(m, note))
	}
	plain := mustAdd(t, m, root, kindLabel, "aaaaa")
	if err := m.setLabelStyle(plain, "", boolPtr(true)); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	if sizeOf(m, plain) != [2]int{52, 32} {
		t.Fatalf("unwrapped bold %v", widgetBox(m, plain))
	}

	m.widgets[note].text = []rune("abcdefghij")
	m.widgets[note].labelSize = ""
	m.widgets[note].bold = false
	if err := m.setBox("preferred", note, 20, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := m.setColor(note, partBackground, color.NRGBA{R: 180, A: 255}); err != nil {
		t.Fatal(err)
	}
	layout(t, m)
	noteBox := widgetBox(m, note)
	painted := render(frame{model: m, scale: 1})
	outside := painted.RGBAAt(noteBox[0]+noteBox[2]+2, noteBox[1]+4)
	if outside != colorBackground {
		t.Fatalf("label painted outside its box: %#v at %v", outside, noteBox)
	}

	nested := newModel("t", 120, 80, fixedMeasure)
	pane := mustSpec(t, nested, 0, spec{kind: kindScroll, spacing: 0, padding: 0, selected: -1})
	grid := mustSpec(t, nested, pane, spec{kind: kindGrid, gridColumns: 2, spacing: 0, rowGap: 0, selected: -1})
	mark := addImage(t, nested, grid, jpegPath, 30, 20)
	if err := nested.setBox("span", mark, 2, 1, "", ""); err != nil {
		t.Fatal(err)
	}
	caption := mustAdd(t, nested, grid, kindLabel, "aaaa bbbb")
	if err := nested.setBox("wrap", caption, 1, 0, "", ""); err != nil || nested.setBox("preferred", caption, 40, 0, "", "") != nil {
		t.Fatal("wrap in grid")
	}
	if err := nested.setBox("span", caption, 2, 1, "", ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		mustAdd(t, nested, pane, kindLabel, "line")
	}
	layout(t, nested)
	if sizeOf(nested, caption) != [2]int{40, 64} {
		t.Fatalf("grid caption %v", widgetBox(nested, caption))
	}
	origin := widgetBox(nested, mark)
	nested.press(origin[0]+2, origin[1]+2)
	if nested.focus != 0 {
		t.Fatal("image accepted focus")
	}
	nested.wheel(10, 10, 0, -3)
	layout(t, nested)
	if nested.widgets[pane].scrollY == 0 {
		t.Fatal("wheel over the image did not scroll")
	}
	if nested.hit(origin[0]+2, origin[1]+2) == mark {
		t.Fatal("hit an image that had scrolled out of view")
	}
	nested.resize(120, 400)
	layout(t, nested)
	if sizeOf(nested, mark) != [2]int{30, 20} {
		t.Fatalf("resize changed the image %v", widgetBox(nested, mark))
	}
}

func addImage(t *testing.T, m *model, parent int64, path string, width, height int) int64 {
	t.Helper()
	id, err := m.addSpec(parent, spec{kind: kindImage, text: path, fitW: width, fitH: height, selected: -1})
	if err != nil {
		t.Fatalf("image: %v", err)
	}
	return id
}

func writeJPEG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{R: 220, A: 255}), image.Point{}, draw.Src)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := jpeg.Encode(file, img, nil); err != nil {
		t.Fatal(err)
	}
}

func boolPtr(value bool) *bool { return &value }

func sizeOf(m *model, id int64) [2]int {
	box := widgetBox(m, id)
	return [2]int{box[2], box[3]}
}
