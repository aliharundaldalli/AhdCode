package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // registers the JPEG decoder
	_ "image/png"  // registers the PNG decoder
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// A GUI image is a picture file shown in a widget. The helper reads and
// decodes it once, when the Image is created, and keeps the pixels on the
// widget. Later frames scale those pixels only when the destination size
// changes. GIF and every format other than PNG and JPEG are refused.
//
// The path is absolute by the time it is stored: the runtime resolves a
// relative path against the program's working directory. This helper does
// not search its own folder or an application bundle.

const (
	maxImagePath   = 4096
	maxImageBytes  = 64 << 20
	maxImageSide   = 8192
	maxImagePixels = 32 << 20
)

func badImageSize() error {
	return fmt.Errorf("image width and height must be between 0 and %d", maxDimension)
}

// loadGUIImage reads one PNG or JPEG file. The returned image is owned by
// the widget and is not decoded again.
func loadGUIImage(path string) (image.Image, int, int, error) {
	if path == "" || strings.ContainsRune(path, 0) || len(path) > maxImagePath {
		return nil, 0, 0, errors.New("image needs a file path")
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg":
	default:
		return nil, 0, 0, errors.New("GUI images are .png, .jpg, and .jpeg files")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, 0, 0, errors.New("the image path cannot be resolved")
	}
	info, err := os.Stat(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, 0, 0, errors.New("could not read the image: the file does not exist")
		}
		return nil, 0, 0, errors.New("could not read the image")
	}
	if !info.Mode().IsRegular() {
		return nil, 0, 0, errors.New("could not read the image: it is not a file")
	}
	if info.Size() > maxImageBytes {
		return nil, 0, 0, errors.New("the image file is larger than 64 MiB")
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return nil, 0, 0, errors.New("could not read the image")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") {
		return nil, 0, 0, errors.New("the file is not a PNG or JPEG image")
	}
	if config.Width < 1 || config.Height < 1 {
		return nil, 0, 0, errors.New("the image file is damaged and could not be decoded")
	}
	if config.Width > maxImageSide || config.Height > maxImageSide || config.Width*config.Height > maxImagePixels {
		return nil, 0, 0, errors.New("the image is too large; each side may be at most 8192 pixels")
	}
	pixels, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, errors.New("the image file is damaged and could not be decoded")
	}
	return pixels, config.Width, config.Height, nil
}

// measureImage sets the box. Zero fit means the picture's own size, capped
// at maxDimension on each side. One fit side makes the other follow the
// picture, and a derived side past the cap stays at the cap so the picture
// is fitted inside. Both sides are the box exactly; the picture is fitted
// inside it when it is drawn.
func (m *model) measureImage(target *widget) {
	pw, ph := target.pixelW, target.pixelH
	if pw < 1 {
		pw = 1
	}
	if ph < 1 {
		ph = 1
	}
	switch {
	case target.fitW == 0 && target.fitH == 0:
		target.natW, target.natH = capImage(pw, ph, maxDimension)
	case target.fitW > 0 && target.fitH == 0:
		height := max(target.fitW*ph/pw, 1)
		if height > maxDimension {
			height = maxDimension
		}
		target.natW, target.natH = target.fitW, height
	case target.fitH > 0 && target.fitW == 0:
		width := max(target.fitH*pw/ph, 1)
		if width > maxDimension {
			width = maxDimension
		}
		target.natW, target.natH = width, target.fitH
	default:
		target.natW, target.natH = target.fitW, target.fitH
	}
}

func capImage(width, height, limit int) (int, int) {
	if width <= limit && height <= limit {
		return width, height
	}
	if width >= height {
		return limit, max(height*limit/width, 1)
	}
	return max(width*limit/height, 1), limit
}

// fitInside returns the picture's rectangle inside the box, centered, with
// its proportions kept. The coordinates are relative to the box.
func fitInside(boxW, boxH, srcW, srcH int) (x, y, w, h int) {
	if boxW < 1 || boxH < 1 || srcW < 1 || srcH < 1 {
		return
	}
	if srcW*boxH <= srcH*boxW {
		h = boxH
		w = srcW * boxH / srcH
	} else {
		w = boxW
		h = srcH * boxW / srcW
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if w > boxW {
		w = boxW
	}
	if h > boxH {
		h = boxH
	}
	return (boxW - w) / 2, (boxH - h) / 2, w, h
}

// drawImage paints the cached picture. The scaled RGBA is rebuilt only when
// the destination size changes, including after a window resize.
func (m *model) drawImage(img *image.RGBA, w *widget, scale float64) {
	if w.pixels == nil || w.w < 1 || w.h < 1 || w.pixelW < 1 || w.pixelH < 1 {
		return
	}
	ox, oy, fw, fh := fitInside(w.w, w.h, w.pixelW, w.pixelH)
	if fw < 1 || fh < 1 {
		return
	}
	full := scaled(scale, w.x+ox, w.y+oy, fw, fh)
	visible := full.Intersect(img.Bounds())
	if visible.Empty() {
		return
	}
	dw, dh := full.Dx(), full.Dy()
	if dw < 1 || dh < 1 {
		return
	}
	if w.scaled == nil || w.scaledW != dw || w.scaledH != dh {
		dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
		xdraw.BiLinear.Scale(dst, dst.Bounds(), w.pixels, w.pixels.Bounds(), draw.Src, nil)
		w.scaled = dst
		w.scaledW, w.scaledH = dw, dh
	}
	draw.Draw(img, visible, w.scaled, image.Pt(visible.Min.X-full.Min.X, visible.Min.Y-full.Min.Y), draw.Over)
}
