// Package plotmath is the one math-label authority used by ahdplot and
// ahdplotview. It deliberately has no Gonum dependency: both helpers ask this
// package for the same cached, transparent image asset produced by the
// bundled offline Tectonic runtime.
package plotmath

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/SalvioniDigitalSolutions/gopdf"
	xdraw "golang.org/x/image/draw"
)

const (
	maxFormulaRunes = 512
	maxPDFBytes     = 8 << 20
	rasterDPI       = 240.0
	cacheLimit      = 128
	compileTimeout  = 20 * time.Second
	latexBaseSize   = 10.0

	// Changing this invalidates the in-process cache if the rendering
	// pipeline or its embedded-font handling changes.
	mathRendererGeneration = "tectonic-embedded-cff-v1"
)

// Asset is a transparent, tightly cropped math label. Dimensions are in
// typographic points, matching Gonum's vg coordinate system.
type Asset struct {
	Image  image.Image
	Width  float64
	Height float64
	Depth  float64
}

type cachedAsset struct {
	asset Asset
}

var globalCache = struct {
	sync.Mutex
	items    map[string]cachedAsset
	order    []string
	compiles uint64
}{items: make(map[string]cachedAsset)}

// ResetCache is a test hook and is also useful to a long-lived helper test
// process. It never changes the renderer's on-disk Tectonic resources.
func ResetCache() {
	globalCache.Lock()
	defer globalCache.Unlock()
	globalCache.items = make(map[string]cachedAsset)
	globalCache.order = nil
	globalCache.compiles = 0
}

// CompileCount reports how many cache misses invoked Tectonic since the last
// ResetCache.
func CompileCount() uint64 {
	globalCache.Lock()
	defer globalCache.Unlock()
	return globalCache.compiles
}

// DiscoverLatexRoot returns the packaged latex directory associated with an
// ahdplot helper. Explicit configuration wins, then the helper's sibling and
// the distribution's libexec layout. No PATH or network lookup is performed.
func DiscoverLatexRoot(helper string) string {
	roots := []string{os.Getenv("AHDCODE_LATEX_RUNTIME")}
	if helper != "" {
		dir := filepath.Dir(helper)
		roots = append(roots,
			filepath.Join(dir, "latex"),
			filepath.Join(dir, "..", "libexec", "ahdcode", "latex"),
		)
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		root = filepath.Clean(root)
		if hasRuntime(root) {
			return root
		}
	}
	return ""
}

func hasRuntime(root string) bool {
	engine := "tectonic"
	if runtime.GOOS == "windows" {
		engine += ".exe"
	}
	for _, name := range []string{engine, "ahdcode-latex.ttb"} {
		info, err := os.Stat(filepath.Join(root, name))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

// Render validates a whole-string $...$ label, compiles it with the bundled
// offline Tectonic engine, rasterises the resulting PDF, removes the white
// page background, and returns a bounded cached asset. An empty latexRoot is
// an error: math labels never fall back to Gonum or Unicode approximation.
func Render(formula string, fontSize float64, latexRoot string) (Asset, error) {
	if err := validate(formula); err != nil {
		return Asset{}, err
	}
	if fontSize <= 0 || math.IsNaN(fontSize) || math.IsInf(fontSize, 0) || fontSize > 128 {
		return Asset{}, errors.New("math font size must be finite and between 0 and 128")
	}
	if !hasRuntime(latexRoot) {
		return Asset{}, errors.New("bundled offline Tectonic runtime is missing")
	}
	key := mathRendererGeneration + "\x00" + formula + "\x00" + latexRoot

	// The lock covers compile-once as well as lookup. Surface interaction and
	// chart sizing can ask for the same formula at different sizes; only one
	// Tectonic process should be created for a unique formula.
	globalCache.Lock()
	defer globalCache.Unlock()
	if entry, ok := globalCache.items[key]; ok {
		return scaleAsset(entry.asset, fontSize/latexBaseSize), nil
	}
	baseAsset, err := compileAndRasterize(formula, latexRoot)
	if err != nil {
		return Asset{}, err
	}
	globalCache.compiles++
	if len(globalCache.order) >= cacheLimit {
		oldest := globalCache.order[0]
		globalCache.order = globalCache.order[1:]
		delete(globalCache.items, oldest)
	}
	globalCache.items[key] = cachedAsset{asset: baseAsset}
	globalCache.order = append(globalCache.order, key)
	return scaleAsset(baseAsset, fontSize/latexBaseSize), nil
}

func validate(formula string) error {
	if len(formula) < 2 || !strings.HasPrefix(formula, "$") || !strings.HasSuffix(formula, "$") {
		return errors.New("math label must be a whole-string $...$ formula")
	}
	if !utf8.ValidString(formula) {
		return errors.New("math label is not valid UTF-8")
	}
	if utf8.RuneCountInString(formula) > maxFormulaRunes {
		return fmt.Errorf("math label is too long (maximum %d characters)", maxFormulaRunes)
	}
	content := strings.TrimSuffix(strings.TrimPrefix(formula, "$"), "$")
	if strings.TrimSpace(content) == "" {
		return errors.New("math label is empty")
	}
	if strings.ContainsAny(content, "\x00\r\n%#$&") {
		return errors.New("math label contains a forbidden control character")
	}
	if strings.Contains(content, "$") {
		return errors.New("math label contains an inner dollar delimiter")
	}
	if !balanced(formula) {
		return errors.New("math label contains unmatched braces")
	}
	if err := validateCommands(content); err != nil {
		return err
	}
	return nil
}

// The whitelist keeps the fragment inside ordinary math notation. In
// particular, adding a new TeX package or command cannot accidentally turn a
// label into a file, shell, network, or document-level operation.
var allowedCommands = map[string]struct{}{
	"alpha": {}, "beta": {}, "gamma": {}, "delta": {}, "epsilon": {}, "varepsilon": {},
	"zeta": {}, "eta": {}, "theta": {}, "vartheta": {}, "iota": {}, "kappa": {},
	"lambda": {}, "mu": {}, "nu": {}, "xi": {}, "pi": {}, "varpi": {}, "rho": {},
	"sigma": {}, "varsigma": {}, "tau": {}, "upsilon": {}, "phi": {}, "varphi": {},
	"chi": {}, "psi": {}, "omega": {}, "Gamma": {}, "Delta": {}, "Theta": {},
	"Lambda": {}, "Xi": {}, "Pi": {}, "Sigma": {}, "Phi": {}, "Psi": {}, "Omega": {},
	"sum": {}, "prod": {}, "int": {}, "oint": {}, "partial": {}, "nabla": {},
	"infty": {}, "frac": {}, "dfrac": {}, "tfrac": {}, "sqrt": {}, "binom": {},
	"displaystyle": {}, "limits": {}, "nolimits": {},
	"left": {}, "right": {}, "big": {}, "Big": {}, "bigg": {}, "Bigg": {},
	"cdot": {}, "times": {}, "pm": {}, "mp": {}, "div": {}, "le": {}, "leq": {},
	"ge": {}, "geq": {}, "neq": {}, "approx": {}, "sim": {}, "equiv": {}, "in": {},
	"notin": {}, "subset": {}, "subseteq": {}, "supset": {}, "supseteq": {},
	"log": {}, "ln": {}, "exp": {}, "sin": {}, "cos": {}, "tan": {}, "lim": {},
	"min": {}, "max": {}, "mathrm": {}, "mathbf": {}, "mathit": {}, "mathcal": {},
	"overline": {}, "underline": {}, "vec": {}, "text": {}, "quad": {}, "qquad": {},
	",": {}, ";": {}, ":": {}, "!": {}, " ": {},
}

func validateCommands(content string) error {
	for index := 0; index < len(content); index++ {
		if content[index] != '\\' {
			continue
		}
		index++
		if index >= len(content) {
			return errors.New("math label ends with an incomplete TeX command")
		}
		start := index
		if (content[index] >= 'A' && content[index] <= 'Z') || (content[index] >= 'a' && content[index] <= 'z') {
			for index < len(content) && ((content[index] >= 'A' && content[index] <= 'Z') || (content[index] >= 'a' && content[index] <= 'z')) {
				index++
			}
		} else {
			index++
		}
		command := content[start:index]
		if _, ok := allowedCommands[command]; !ok {
			return fmt.Errorf("math label uses unsupported or forbidden TeX command \\%s", command)
		}
		index--
	}
	return nil
}

func balanced(value string) bool {
	depth := 0
	for index := 0; index < len(value); index++ {
		if value[index] == '\\' {
			index++
			continue
		}
		switch value[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

func compileAndRasterize(formula string, latexRoot string) (Asset, error) {
	working, err := os.MkdirTemp("", "ahdcode-plotmath-")
	if err != nil {
		return Asset{}, fmt.Errorf("creating isolated math directory: %w", err)
	}
	defer os.RemoveAll(working)
	output := filepath.Join(working, "output")
	cache := filepath.Join(working, "cache")
	if err := os.Mkdir(output, 0o700); err != nil {
		return Asset{}, err
	}
	if err := os.Mkdir(cache, 0o700); err != nil {
		return Asset{}, err
	}
	source := filepath.Join(working, "formula.tex")
	text := texDocument(formula)
	if err := os.WriteFile(source, []byte(text), 0o600); err != nil {
		return Asset{}, fmt.Errorf("writing math source: %w", err)
	}
	engine := filepath.Join(latexRoot, "tectonic")
	if runtime.GOOS == "windows" {
		engine += ".exe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), compileTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, engine, "--untrusted", "--color", "never", "--bundle", bundleURL(filepath.Join(latexRoot, "ahdcode-latex.ttb")), "--only-cached", "--outdir", output, source)
	command.Dir = working
	command.Env = append(os.Environ(), "TECTONIC_CACHE_DIR="+cache)
	log := &boundedLog{}
	command.Stdout, command.Stderr = log, log
	if err := command.Run(); ctx.Err() == context.DeadlineExceeded {
		return Asset{}, fmt.Errorf("math compilation timed out after %s", compileTimeout)
	} else if err != nil {
		message := strings.TrimSpace(log.String())
		if message == "" {
			message = err.Error()
		}
		return Asset{}, fmt.Errorf("Tectonic math compilation failed: %s", message)
	}
	pdfPath := filepath.Join(output, "formula.pdf")
	info, err := os.Stat(pdfPath)
	if err != nil {
		return Asset{}, fmt.Errorf("Tectonic did not produce a PDF: %w", err)
	}
	if info.Size() == 0 || info.Size() > maxPDFBytes {
		return Asset{}, errors.New("Tectonic PDF output exceeded the math-label bound")
	}
	reader, err := gopdf.Open(pdfPath)
	if err != nil {
		return Asset{}, fmt.Errorf("opening Tectonic PDF: %w", err)
	}
	img, report, err := reader.RenderPageDetail(0, gopdf.RenderOpts{
		DPI:           rasterDPI,
		IncludeText:   true,
		IncludeVector: true,
		Transparent:   true,
	})
	if err != nil {
		return Asset{}, fmt.Errorf("rasterising Tectonic PDF: %w", err)
	}
	if report.Missing > 0 {
		return Asset{}, fmt.Errorf("rasterising Tectonic PDF faithfully failed: %d glyphs were unavailable from embedded PDF font programs", report.Missing)
	}
	asset, err := cropTransparent(img, rasterDPI)
	if err != nil {
		return Asset{}, err
	}
	return asset, nil
}

func texDocument(formula string) string {
	content := strings.TrimSuffix(strings.TrimPrefix(formula, "$"), "$")
	leading := strconv.FormatFloat(latexBaseSize, 'g', 8, 64)
	line := strconv.FormatFloat(latexBaseSize*1.35, 'g', 8, 64)
	// The pinned offline bundle intentionally carries size10/size11 only;
	// fontSize is applied explicitly below, so the class default is immaterial.
	return `\documentclass[10pt]{article}
\usepackage{amsmath,amssymb}
\usepackage[paperwidth=12in,paperheight=4in,margin=0.25in]{geometry}
\renewcommand{\rmdefault}{cmr}
\pagestyle{empty}
\begin{document}
\thispagestyle{empty}
\noindent\fontsize{` + leading + `pt}{` + line + `pt}\selectfont
\[` + content + `\]
\end{document}
`
}

func cropTransparent(input image.Image, dpi float64) (Asset, error) {
	bounds := input.Bounds()
	alpha := image.NewNRGBA(bounds)
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := color.NRGBAModel.Convert(input.At(x, y)).RGBA()
			var level uint8
			if a == 0 {
				level = 0
			} else if a < 0xffff {
				level = uint8(a >> 8)
			} else {
				level = uint8(255 - max8(uint8(r>>8), uint8(g>>8), uint8(b>>8)))
			}
			if level < 3 {
				level = 0
			}
			alpha.SetNRGBA(x, y, color.NRGBA{R: 0, G: 0, B: 0, A: level})
			if level > 0 {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x >= maxX {
					maxX = x + 1
				}
				if y >= maxY {
					maxY = y + 1
				}
			}
		}
	}
	if minX >= maxX || minY >= maxY {
		return Asset{}, errors.New("Tectonic rendered an empty math label")
	}
	padding := 2
	minX = maxInt(bounds.Min.X, minX-padding)
	minY = maxInt(bounds.Min.Y, minY-padding)
	maxX = minInt(bounds.Max.X, maxX+padding)
	maxY = minInt(bounds.Max.Y, maxY+padding)
	result := image.NewNRGBA(image.Rect(0, 0, maxX-minX, maxY-minY))
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			result.Set(x-minX, y-minY, alpha.At(x, y))
		}
	}
	points := 72 / dpi
	return Asset{Image: result, Width: float64(result.Bounds().Dx()) * points, Height: float64(result.Bounds().Dy()) * points, Depth: 0}, nil
}

func scaleAsset(asset Asset, factor float64) Asset {
	if math.Abs(factor-1) < 0.0001 {
		return asset
	}
	width := maxInt(1, int(math.Round(float64(asset.Image.Bounds().Dx())*factor)))
	height := maxInt(1, int(math.Round(float64(asset.Image.Bounds().Dy())*factor)))
	resized := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), asset.Image, asset.Image.Bounds(), xdraw.Over, nil)
	return Asset{Image: resized, Width: asset.Width * factor, Height: asset.Height * factor, Depth: asset.Depth * factor}
}

func max8(a, b, c uint8) uint8 {
	if a < b {
		a = b
	}
	if a < c {
		a = c
	}
	return a
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func bundleURL(path string) string {
	path = filepath.ToSlash(path)
	return (&url.URL{Scheme: "file", Path: path}).String()
}

type boundedLog struct{ data []byte }

func (log *boundedLog) Write(value []byte) (int, error) {
	count := len(value)
	if len(log.data) < 32<<10 {
		remaining := (32 << 10) - len(log.data)
		if len(value) > remaining {
			value = value[:remaining]
		}
		log.data = append(log.data, value...)
	}
	return count, nil
}
func (log *boundedLog) String() string { return string(log.data) }
