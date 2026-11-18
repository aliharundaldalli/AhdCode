package build

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestGUIImageNativeAndEvaluatorAgree(t *testing.T) {
	useHeadlessGUI(t, "")
	directory := t.TempDir()
	picture := filepath.Join(directory, "mark.png")
	writeTinyPNG(t, picture)
	missing := filepath.Join(directory, "missing.png")
	notes := filepath.Join(directory, "notes.txt")
	if err := os.WriteFile(notes, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	damaged := filepath.Join(directory, "damaged.png")
	if err := os.WriteFile(damaged, []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	program := `bring GUI
from GUI bring (Window, Container, Image, Label, GUIError)

window: Window := GUI.window("Images", 420, 280)
form: Container := window.column()
photo: Image := form.image(` + strconv.Quote(picture) + `, 40, 20)
photo.setSize(80, 0)
photo.setEnabled(false)
note: Label := form.label("İstanbul 👍")
note.setWrap(true)
note.setPreferredWidth(80)
note.setSize("large")
note.setBold(true)
if photo.isEnabled() {
    write("enabled")
}
attempt {
    form.image(` + strconv.Quote(missing) + `)
    write("accepted missing")
} except GUIError as error {
    write(error.message)
}
attempt {
    form.image(` + strconv.Quote(notes) + `)
    write("accepted text")
} except GUIError as error {
    write(error.message)
}
attempt {
    form.image(` + strconv.Quote(damaged) + `)
    write("accepted damaged")
} except GUIError as error {
    write(error.message)
}
attempt {
    photo.setSize(-1, 0)
    write("accepted size")
} except GUIError as error {
    write(error.message)
}
window.wait()
write("ready")
`
	assertGUI(t, program, "could not read the image: the file does not exist\nGUI images are .png, .jpg, and .jpeg files\nthe file is not a PNG or JPEG image\nimage width and height must be between 0 and 4096\nready\n")

	example := filepath.Join("..", "..", "examples", "v2.9", "gallery", "main.ahd")
	built := filepath.Join(t.TempDir(), "gallery")
	path, result := BuildProgram(example, built)
	if result.HasErrors() {
		t.Fatalf("gallery failed to compile:\n%s", diagnosticText(result.Diagnostics))
	}
	command := exec.Command(path)
	command.Dir = filepath.Dir(example)
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("gallery: %v\n%s", err, out)
	}
}

func writeTinyPNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 40, G: 90, B: 160, A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}
