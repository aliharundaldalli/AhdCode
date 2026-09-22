package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"ahdcode/cmd/ahdplotmath"
)

// surfaceMathCache is intentionally a thin viewer adapter. Compilation and
// bounded caching belong to plotmath, so Plot and Surface cannot diverge or
// start a Tectonic process once per Surface frame.
type surfaceMathCache struct {
	latexRoot string
}

func newSurfaceMathCache(helper string) *surfaceMathCache {
	return &surfaceMathCache{latexRoot: plotmath.DiscoverLatexRoot(helper)}
}

func (cache *surfaceMathCache) image(value string, size float64, _ color.Color) (image.Image, bool) {
	if cache == nil || !isSurfaceMath(value) {
		return nil, false
	}
	asset, err := plotmath.Render(value, size, cache.latexRoot)
	if err != nil {
		return nil, false
	}
	return asset.Image, true
}

func isSurfaceMath(value string) bool {
	return len(value) >= 2 && strings.HasPrefix(value, "$") && strings.HasSuffix(value, "$")
}

func (cache *surfaceMathCache) prepare(surface surfaceSpec, size float64) error {
	texts := []string{surface.Title, surface.XLabel, surface.YLabel, surface.ZLabel}
	texts = append(texts, surface.XCategories...)
	texts = append(texts, surface.YCategories...)
	for _, value := range texts {
		if !isSurfaceMath(value) {
			continue
		}
		if _, ok := cache.image(value, size, color.Black); !ok {
			return fmt.Errorf("could not render Surface math text %q", value)
		}
	}
	return nil
}

func (r surfaceRenderer) prepare(surface surfaceSpec, size float64) error {
	if r.math == nil {
		return nil
	}
	return r.math.prepare(surface, size)
}

func newSurfaceRendererWithHelper(helper string) surfaceRenderer {
	renderer := newSurfaceRenderer()
	// Keep an adapter even when discovery fails. A math label must then fail
	// Surface preparation instead of silently becoming raw TeX on the canvas.
	renderer.math = newSurfaceMathCache(helper)
	return renderer
}
