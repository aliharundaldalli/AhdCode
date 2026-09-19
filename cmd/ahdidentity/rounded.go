package ahdidentity

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// macOS draws application icons as rounded squares inset in their canvas:
// the body is 824 of 1024 units, with corners of radius 185 units, and the
// system does not round an icon for an application outside a bundle. The
// official AhdCode artwork is a full square, so on macOS it is shown through
// this shape; Windows and Linux show the artwork as it is.

// Rounded is the macOS shape of a square icon, size pixels on a side.
func Rounded(source image.Image, size int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	unit := float64(size) / 1024
	body := 824 * unit
	radius := 185 * unit
	inset := (float64(size) - body) / 2
	scaled := shrinkTo(source, int(math.Round(body)))
	bodySize := scaled.Bounds().Dx()
	for y := range bodySize {
		for x := range bodySize {
			coverage := roundedCoverage(float64(x)+0.5, float64(y)+0.5, float64(bodySize), radius)
			if coverage <= 0 {
				continue
			}
			c := color.NRGBAModel.Convert(scaled.At(x, y)).(color.NRGBA)
			c.A = uint8(math.Round(float64(c.A) * coverage))
			out.SetNRGBA(int(math.Round(inset))+x, int(math.Round(inset))+y, c)
		}
	}
	return out
}

// roundedCoverage is how much of the pixel at (x, y) lies inside a rounded
// square of the given side and corner radius, with a one-pixel soft edge.
func roundedCoverage(x, y, side, radius float64) float64 {
	cx := math.Max(radius-x, math.Max(x-(side-radius), 0))
	cy := math.Max(radius-y, math.Max(y-(side-radius), 0))
	if cx == 0 || cy == 0 {
		return math.Max(0, math.Min(1, math.Min(math.Min(x, side-x), math.Min(y, side-y))+0.5))
	}
	distance := math.Sqrt(cx*cx + cy*cy)
	return math.Max(0, math.Min(1, radius-distance+0.5))
}

// shrinkTo scales a square image down to size pixels by averaging blocks;
// Rounded is only used at sizes up to the artwork's own.
func shrinkTo(source image.Image, size int) image.Image {
	if source.Bounds().Dx() == size {
		return source
	}
	return shrink(source, size)
}

// RoundedPNG is Rounded as PNG bytes.
func RoundedPNG(source image.Image, size int) []byte {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, Rounded(source, size)); err != nil {
		return nil
	}
	return buffer.Bytes()
}
