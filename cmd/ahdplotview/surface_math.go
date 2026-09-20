package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxSurfaceMathCache = 64
	surfaceMathTimeout  = 20 * time.Second
)

type surfaceMathEntry struct {
	key string
	img image.Image
}

type surfaceMathCache struct {
	helper  string
	mu      sync.Mutex
	entries map[string]surfaceMathEntry
	order   []string
}

// The viewer is a separate Go module. Keep this tiny wire shape local rather
// than making the GUI helper depend on the compiler's internal packages.
type surfaceMathRequest struct {
	Mode       string           `json:"mode,omitempty"`
	OutputPath string           `json:"output_path,omitempty"`
	Math       *surfaceMathSpec `json:"math,omitempty"`
}

type surfaceMathSpec struct {
	Formula  string  `json:"formula"`
	FontSize float64 `json:"font_size,omitempty"`
}

type surfaceMathResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func newSurfaceMathCache(helper string) *surfaceMathCache {
	return &surfaceMathCache{helper: helper, entries: make(map[string]surfaceMathEntry)}
}

func (cache *surfaceMathCache) image(value string, size float64, _ color.Color) (image.Image, bool) {
	if !isSurfaceMath(value) || cache.helper == "" {
		return nil, false
	}
	key := value + "\x00" + strconv.FormatFloat(size, 'g', 12, 64)
	cache.mu.Lock()
	if entry, ok := cache.entries[key]; ok {
		cache.mu.Unlock()
		return entry.img, true
	}
	cache.mu.Unlock()

	rendered, err := renderSurfaceMath(cache.helper, value, size)
	if err != nil {
		return nil, false
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if existing, ok := cache.entries[key]; ok {
		return existing.img, true
	}
	if len(cache.order) >= maxSurfaceMathCache {
		oldest := cache.order[0]
		cache.order = cache.order[1:]
		delete(cache.entries, oldest)
	}
	cache.entries[key] = surfaceMathEntry{key: key, img: rendered}
	cache.order = append(cache.order, key)
	return rendered, true
}

func isSurfaceMath(value string) bool {
	return len(value) >= 2 && strings.HasPrefix(value, "$") && strings.HasSuffix(value, "$")
}

func (cache *surfaceMathCache) prepare(surface surfaceSpec, size float64) error {
	if cache == nil || cache.helper == "" {
		return nil
	}
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

func renderSurfaceMath(helper, formula string, size float64) (image.Image, error) {
	requestPath, err := os.CreateTemp("", "ahdplot-surface-math-*.json")
	if err != nil {
		return nil, errors.New("could not create math request")
	}
	requestName := requestPath.Name()
	defer os.Remove(requestName)
	outputPath, err := os.CreateTemp("", "ahdplot-surface-math-*.png")
	if err != nil {
		requestPath.Close()
		return nil, errors.New("could not create math output")
	}
	outputName := outputPath.Name()
	outputPath.Close()
	defer os.Remove(outputName)
	request := surfaceMathRequest{Mode: "math", OutputPath: outputName, Math: &surfaceMathSpec{Formula: formula, FontSize: size}}
	encoded, _ := json.Marshal(request)
	if _, err := requestPath.Write(encoded); err != nil {
		requestPath.Close()
		return nil, errors.New("could not write math request")
	}
	if err := requestPath.Close(); err != nil {
		return nil, errors.New("could not close math request")
	}
	ctx, cancel := context.WithTimeout(context.Background(), surfaceMathTimeout)
	defer cancel()
	output, runErr := exec.CommandContext(ctx, helper, requestName).CombinedOutput()
	if runErr != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 512 {
			message = message[:512]
		}
		if message != "" {
			return nil, fmt.Errorf("math renderer stopped: %s", message)
		}
		return nil, fmt.Errorf("math renderer stopped: %w", runErr)
	}
	var response surfaceMathResponse
	if json.Unmarshal(bytes.TrimSpace(output), &response) != nil || !response.OK {
		if response.Message == "" {
			response.Message = "invalid math response"
		}
		return nil, errors.New(response.Message)
	}
	info, err := os.Stat(outputName)
	if err != nil || info.Size() == 0 || info.Size() > 8<<20 {
		return nil, errors.New("math renderer produced no bounded image")
	}
	file, err := os.Open(outputName)
	if err != nil {
		return nil, errors.New("could not read math image")
	}
	decoded, decodeErr := png.Decode(file)
	file.Close()
	if decodeErr != nil || decoded.Bounds().Dx() <= 0 || decoded.Bounds().Dy() <= 0 {
		return nil, errors.New("math renderer produced an invalid image")
	}
	return decoded, nil
}

func newSurfaceRendererWithHelper(helper string) surfaceRenderer {
	renderer := newSurfaceRenderer()
	if helper != "" {
		renderer.math = newSurfaceMathCache(filepath.Clean(helper))
	}
	return renderer
}
