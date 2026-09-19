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
// Inside an application made by `ahdcode package`, the helpers belong to
// that application instead: the packager writes ahdcode-app.json beside them
// (or in the macOS bundle's Resources folder), naming the application and its
// icon, and the helpers show that name and icon. On macOS the application
// bundle itself then provides the name and Dock icon, so no LaunchServices
// call is made at all.
package ahdidentity

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

// DefaultName is the application name AhdCode's own windows show.
const DefaultName = "AhdCode"

// AppFile is the packaged application's identity file.
const AppFile = "ahdcode-app.json"

// Bounds on a packaged identity.
const (
	maxNameRunes = 128
	maxIconBytes = 8 << 20
	maxIconSide  = 2048
)

// App is a packaged application's identity file: its name and the file name
// of its PNG icon, beside the identity file.
type App struct {
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

var (
	appOnce  sync.Once
	appName  string
	appIcon  image.Image
	appBytes []byte
	packaged bool
)

// loadApp reads the identity of a packaged application, if the running
// helper is part of one. Anything malformed is ignored: the helper then
// keeps AhdCode's identity.
func loadApp() {
	appOnce.Do(func() {
		executable, err := os.Executable()
		if err != nil {
			return
		}
		directory := filepath.Dir(executable)
		for _, candidate := range []string{filepath.Join(directory, AppFile), filepath.Join(directory, "..", "Resources", AppFile)} {
			name, icon, raw, ok := readApp(candidate)
			if ok {
				appName, appIcon, appBytes, packaged = name, icon, raw, true
				return
			}
		}
	})
}

func readApp(path string) (string, image.Image, []byte, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 64<<10 {
		return "", nil, nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, nil, false
	}
	var app App
	if json.Unmarshal(data, &app) != nil || !ValidName(app.Name) {
		return "", nil, nil, false
	}
	if app.Icon == "" {
		return app.Name, nil, nil, true
	}
	if strings.ContainsAny(app.Icon, `/\`) || app.Icon == "." || app.Icon == ".." {
		return "", nil, nil, false
	}
	iconPath := filepath.Join(filepath.Dir(path), app.Icon)
	iconInfo, err := os.Stat(iconPath)
	if err != nil || !iconInfo.Mode().IsRegular() || iconInfo.Size() > maxIconBytes {
		return app.Name, nil, nil, true
	}
	raw, err := os.ReadFile(iconPath)
	if err != nil {
		return app.Name, nil, nil, true
	}
	config, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width > maxIconSide || config.Height > maxIconSide {
		return app.Name, nil, nil, true
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return app.Name, nil, nil, true
	}
	return app.Name, img, raw, true
}

// ValidName reports whether a text can name a packaged application.
func ValidName(name string) bool {
	return strings.TrimSpace(name) != "" && utf8.ValidString(name) && utf8.RuneCountInString(name) <= maxNameRunes &&
		!strings.ContainsAny(name, "\x00\n\r\t/\\:")
}

// Packaged reports whether the running helper is part of a packaged
// application.
func Packaged() bool {
	loadApp()
	return packaged
}

// Name is the application name the helper's windows show: the packaged
// application's, or AhdCode.
func Name() string {
	loadApp()
	if packaged {
		return appName
	}
	return DefaultName
}

//go:embed ahdcode-icon.png
var iconPNG []byte

var (
	decodeOnce sync.Once
	decoded    image.Image
)

// IconPNG is the application icon as PNG bytes: the packaged
// application's, or the embedded official AhdCode icon.
func IconPNG() []byte {
	loadApp()
	if appBytes != nil {
		return appBytes
	}
	return iconPNG
}

// OfficialIconPNG is the embedded official AhdCode icon.
func OfficialIconPNG() []byte { return iconPNG }

// Icon is the decoded application icon.
func Icon() image.Image {
	loadApp()
	if appIcon != nil {
		return appIcon
	}
	return officialIcon()
}

func officialIcon() image.Image {
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
