// Package ahdidentity gives AhdCode's own window helpers -- ahdgui,
// ahdgraphics, and ahdplotview -- one application identity: the name AhdCode
// and the official AhdCode icon (editors/vscode/images/ahdcode-icon.png,
// embedded here byte for byte). It is private infrastructure shared by those
// helper modules through a local replace directive; it is not part of any
// AhdCode language module and publishes no API to AhdCode programs.
//
// What each platform shows:
//
//   - Windows and Linux: the window icon, through Ebitengine's supported
//     SetWindowIcon (window managers that ignore window icons show their
//     own default).
//   - macOS: the Dock icon and the application name in the menu bar. An
//     executable outside an application bundle has no Info.plist, so the
//     name is set at run time through LaunchServices (see identity_darwin.go).
//
// None of this builds user programs into applications: it only names
// AhdCode's own helper windows.
package ahdidentity

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Name is the application name AhdCode's windows show.
const Name = "AhdCode"

//go:embed ahdcode-icon.png
var iconPNG []byte

var (
	decodeOnce sync.Once
	decoded    image.Image
)

// IconPNG is the embedded official icon, as PNG bytes.
func IconPNG() []byte { return iconPNG }

// Icon is the decoded official icon.
func Icon() image.Image {
	decodeOnce.Do(func() {
		img, err := png.Decode(bytes.NewReader(iconPNG))
		if err != nil {
			panic("ahdidentity: the embedded AhdCode icon does not decode")
		}
		decoded = img
	})
	return decoded
}

// Prepare sets the window icon before the window opens. On macOS the window
// has no icon of its own; Apply sets the Dock icon there instead.
func Prepare() {
	source := Icon()
	ebiten.SetWindowIcon([]image.Image{shrink(source, 16), shrink(source, 32), shrink(source, 48), shrink(source, 256), source})
}

var applyOnce sync.Once

// Apply finishes the identity once the window's event loop is running; call
// it from the first Update. It is safe to call more than once and never
// fails: a platform that cannot take part of the identity keeps its default.
func Apply() {
	applyOnce.Do(applyPlatform)
}

// shrink scales the square icon down to size by averaging each block of
// source pixels, so the small window icons stay smooth without another
// dependency.
func shrink(source image.Image, size int) image.Image {
	bounds := source.Bounds()
	if bounds.Dx() <= size {
		return source
	}
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		y0, y1 := bounds.Min.Y+y*bounds.Dy()/size, bounds.Min.Y+(y+1)*bounds.Dy()/size
		for x := 0; x < size; x++ {
			x0, x1 := bounds.Min.X+x*bounds.Dx()/size, bounds.Min.X+(x+1)*bounds.Dx()/size
			var r, g, b, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := source.At(sx, sy).RGBA()
					r, g, b, a, n = r+uint64(cr), g+uint64(cg), b+uint64(cb), a+uint64(ca), n+1
				}
			}
			out.Set(x, y, color.RGBA64{uint16(r / n), uint16(g / n), uint16(b / n), uint16(a / n)})
		}
	}
	return out
}
