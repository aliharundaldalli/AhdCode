package main

import (
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Canvas text is drawn from glyph outlines, never from a system font: the
// bundled Go Regular face is the only font, so the same text is the same
// shape in the window, in a PNG, and in an SVG on every machine. Each glyph
// becomes filled path segments, which is also what the SVG export writes, so
// an SVG needs no font to be installed where it is opened.

var canvasFont = struct {
	once   sync.Once
	face   *sfnt.Font
	mu     sync.Mutex
	buffer sfnt.Buffer
}{}

func loadCanvasFont() *sfnt.Font {
	canvasFont.once.Do(func() {
		canvasFont.face, _ = sfnt.Parse(goregular.TTF)
	})
	return canvasFont.face
}

func fixedToFloat(value fixed.Int26_6) float64 { return float64(value) / 64 }

// textOutline returns the outline of one line of text as closed sub-paths in
// pixel coordinates. origin is the pixel position of the text's anchor on
// its baseline; size is the font size in pixels.
func textOutline(text, align string, origin point, size float64) []pathOp {
	face := loadCanvasFont()
	if face == nil || size <= 0 || text == "" {
		return nil
	}
	canvasFont.mu.Lock()
	defer canvasFont.mu.Unlock()
	buffer := &canvasFont.buffer
	ppem := fixed.Int26_6(size*64 + 0.5)
	if ppem < 1 {
		return nil
	}

	// First pass: glyph indexes and pen positions, to know the full width.
	type placed struct {
		index sfnt.GlyphIndex
		x     fixed.Int26_6
	}
	glyphs := make([]placed, 0, len(text))
	var pen fixed.Int26_6
	var previous sfnt.GlyphIndex
	for position, character := range text {
		index, err := face.GlyphIndex(buffer, character)
		if err != nil {
			index = 0
		}
		if position > 0 {
			if kern, err := face.Kern(buffer, previous, index, ppem, font.HintingNone); err == nil {
				pen += kern
			}
		}
		glyphs = append(glyphs, placed{index: index, x: pen})
		if advance, err := face.GlyphAdvance(buffer, index, ppem, font.HintingNone); err == nil {
			pen += advance
		}
		previous = index
	}
	left := origin.x
	switch align {
	case "center":
		left -= fixedToFloat(pen) / 2
	case "right":
		left -= fixedToFloat(pen)
	}

	var ops []pathOp
	open := false
	at := func(offset fixed.Int26_6, p fixed.Point26_6) point {
		// Glyph coordinates already have +y pointing down, like pixels.
		return point{left + fixedToFloat(offset+p.X), origin.y + fixedToFloat(p.Y)}
	}
	for _, glyph := range glyphs {
		segments, err := face.LoadGlyph(buffer, glyph.index, ppem, nil)
		if err != nil {
			continue
		}
		for _, segment := range segments {
			switch segment.Op {
			case sfnt.SegmentOpMoveTo:
				if open {
					ops = append(ops, pathOp{kind: 3})
				}
				ops = append(ops, pathOp{kind: 0, p: [3]point{at(glyph.x, segment.Args[0])}})
				open = true
			case sfnt.SegmentOpLineTo:
				ops = append(ops, pathOp{kind: 1, p: [3]point{at(glyph.x, segment.Args[0])}})
			case sfnt.SegmentOpQuadTo:
				ops = append(ops, pathOp{kind: 4, p: [3]point{at(glyph.x, segment.Args[0]), at(glyph.x, segment.Args[1])}})
			case sfnt.SegmentOpCubeTo:
				ops = append(ops, pathOp{kind: 2, p: [3]point{at(glyph.x, segment.Args[0]), at(glyph.x, segment.Args[1]), at(glyph.x, segment.Args[2])}})
			}
		}
		if open {
			ops = append(ops, pathOp{kind: 3})
			open = false
		}
	}
	return ops
}
