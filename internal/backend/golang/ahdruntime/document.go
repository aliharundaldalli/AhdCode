package ahdruntime

// Professional-document building blocks shared by Latex and PDF (v1.3.0):
// page size and four independent margins, running headers and footers with
// page numbers, absolute placement, hyperlinks, PDF bookmarks and metadata,
// and image transforms. Each helper builds ordinary LaTeX for the one offline
// renderer and returns a problem message instead of raising, so the native
// runtime and the interactive evaluator share it unchanged. Fragments that
// need a package or a macro carry a marker comment on a line of its own, the
// same mechanism TikZ fragments use, and document() loads exactly what the
// markers ask for. This file uses only the Go standard library.

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	ahdLatexColorPattern   = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	ahdLatexTheoremPattern = regexp.MustCompile(`\\begin\{(ahdthm[0-9a-f]+)\}`)
)

// ahdLatexAssetMarker names the staged copy of an asset and builds the marker
// line pdf() stages it from. extension is the lower-case file extension.
func ahdLatexAssetMarker(path, extension string) (string, string) {
	sum := sha256.Sum256([]byte(path))
	staged := fmt.Sprintf("ahdasset-%x%s", sum[:8], extension)
	return "% AHDCODE_ASSET " + base64.RawStdEncoding.EncodeToString([]byte(path)) + " " + staged + "\n", staged
}

// ---------------------------------------------------------------------------
// Page layout
// ---------------------------------------------------------------------------

// AhdPaperNames is the closed set of paper names, in documentation order.
var AhdPaperNames = []string{"A3", "A4", "A5", "Letter", "Legal", "Custom"}

// ahdPaperSizes holds each named paper's portrait size in centimeters.
var ahdPaperSizes = map[string][2]float64{
	"A3": {29.7, 42.0}, "A4": {21.0, 29.7}, "A5": {14.8, 21.0},
	"Letter": {21.59, 27.94}, "Legal": {21.59, 35.56},
}

const ahdDocumentMaximumCentimeters = 1000.0

// AhdPageLayout is one validated page layout, in centimeters.
type AhdPageLayout struct {
	Width, Height            float64
	Top, Right, Bottom, Left float64
	Landscape                bool
	// Configured reports that the caller asked for more than the default
	// paper and one margin, so the renderer needs explicit page geometry.
	Configured bool
}

func ahdDocumentCentimeters(value float64) bool {
	return !math.IsNaN(value) && value > 0 && value <= ahdDocumentMaximumCentimeters
}

func ahdPairEntries[V any](pair *AhdPair[string, V]) ([]string, []V) {
	if pair == nil {
		return nil, nil
	}
	pair.require()
	values := make([]V, len(pair.keys))
	for index, key := range pair.keys {
		values[index] = pair.values[key]
	}
	return append([]string(nil), pair.keys...), values
}

// AhdPageLayoutText validates a page layout. defaultPaper is the paper a
// caller receives without asking for one: Letter for Latex.document, whose
// article class has always produced Letter pages, and A4 for PDF. margin is
// the one-value margin, already validated by the caller; margins overrides
// individual sides.
func AhdPageLayoutText(operation, defaultPaper, paper string, landscape bool, margin float64,
	sizeKeys []string, sizeValues []float64, marginKeys []string, marginValues []float64) (AhdPageLayout, string) {
	layout := AhdPageLayout{Landscape: landscape}
	size, named := ahdPaperSizes[paper]
	if !named && paper != "Custom" {
		return layout, operation + " paper must be A3, A4, A5, Letter, Legal, or Custom; received " + strconv.Quote(paper)
	}
	var hasWidth, hasHeight bool
	for index, key := range sizeKeys {
		value := sizeValues[index]
		switch key {
		case "width":
			size[0], hasWidth = value, true
		case "height":
			size[1], hasHeight = value, true
		default:
			return layout, operation + " pageSize supports only width and height; received " + strconv.Quote(key)
		}
		if !ahdDocumentCentimeters(value) {
			return layout, operation + " pageSize " + key + " must be greater than 0 and at most 1000 centimeters"
		}
	}
	if paper == "Custom" && (!hasWidth || !hasHeight) {
		return layout, operation + ` paper "Custom" requires pageSize with both width and height`
	}
	if paper != "Custom" && len(sizeKeys) > 0 {
		return layout, operation + ` pageSize is only used with paper "Custom"; paper ` + strconv.Quote(paper) + " already sets the size"
	}
	layout.Width, layout.Height = size[0], size[1]
	layout.Top, layout.Right, layout.Bottom, layout.Left = margin, margin, margin, margin
	for index, key := range marginKeys {
		value := marginValues[index]
		switch key {
		case "top":
			layout.Top = value
		case "right":
			layout.Right = value
		case "bottom":
			layout.Bottom = value
		case "left":
			layout.Left = value
		default:
			return layout, operation + " margins supports only top, right, bottom, and left; received " + strconv.Quote(key)
		}
		if !ahdDocumentCentimeters(value) {
			return layout, operation + " margins " + key + " must be greater than 0 and at most 1000 centimeters"
		}
	}
	layout.Configured = paper != defaultPaper || len(sizeKeys) > 0 || len(marginKeys) > 0
	if layout.Configured {
		width, height := layout.Width, layout.Height
		if landscape {
			width, height = height, width
		}
		if layout.Left+layout.Right >= width || layout.Top+layout.Bottom >= height {
			return layout, operation + " margins leave no room for content on a " + ahdFormatReal(width) +
				" by " + ahdFormatReal(height) + " cm page"
		}
	}
	return layout, ""
}

// Geometry is the geometry package option list for a configured layout.
// geometry's landscape option turns the stated portrait paper sideways.
func (layout AhdPageLayout) Geometry() string {
	prefix := ""
	if layout.Landscape {
		prefix = "landscape,"
	}
	return prefix + "paperwidth=" + ahdFormatReal(layout.Width) + "cm,paperheight=" + ahdFormatReal(layout.Height) +
		"cm,top=" + ahdFormatReal(layout.Top) + "cm,right=" + ahdFormatReal(layout.Right) +
		"cm,bottom=" + ahdFormatReal(layout.Bottom) + "cm,left=" + ahdFormatReal(layout.Left) + "cm"
}

// ---------------------------------------------------------------------------
// Links, bookmarks, and metadata
// ---------------------------------------------------------------------------

// AhdDocumentURL validates a hyperlink target and returns it in the form
// \href reads. Only https, http, and mailto links are accepted. Bytes that
// are not printable ASCII, and the characters URLs never carry literally, are
// percent-encoded; the characters TeX treats specially are escaped, so a URL
// can never end the \href argument or inject a command. The link a PDF
// reader follows is the URL exactly as given, apart from that encoding.
func AhdDocumentURL(operation, url string) (string, string) {
	if url == "" {
		return "", operation + " url must not be empty"
	}
	if len(url) > 8192 {
		return "", operation + " url must be at most 8192 bytes"
	}
	lower := strings.ToLower(url)
	switch {
	case strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://"):
		if len(url) == strings.Index(url, "://")+3 {
			return "", operation + " url must name a host after " + strconv.Quote(url)
		}
	case strings.HasPrefix(lower, "mailto:"):
		if len(url) == len("mailto:") {
			return "", operation + " url must name an address after mailto:"
		}
	default:
		shown := url
		if len(shown) > 60 {
			shown = shown[:60] + "..."
		}
		return "", operation + " url must start with https://, http://, or mailto:; received " + strconv.Quote(shown)
	}
	const hexDigits = "0123456789ABCDEF"
	var result strings.Builder
	for index := 0; index < len(url); index++ {
		character := url[index]
		switch {
		case character < 0x20 || character == 0x7f:
			return "", operation + " url must not contain control characters"
		case character == ' ' || character >= 0x80 || strings.IndexByte("\"<>\\^`{|}", character) >= 0:
			result.WriteByte('%')
			result.WriteByte(hexDigits[character>>4])
			result.WriteByte(hexDigits[character&0x0f])
		case strings.IndexByte("#%&_$~", character) >= 0:
			result.WriteByte('\\')
			result.WriteByte(character)
		default:
			result.WriteByte(character)
		}
	}
	return result.String(), ""
}

// AhdLatexLinkText is Latex.link: escaped display text linked to a URL.
func AhdLatexLinkText(operation, text, url string) (string, string) {
	if text == "" {
		return "", operation + " text must not be empty"
	}
	target, problem := AhdDocumentURL(operation, url)
	if problem != "" {
		return "", problem
	}
	return "\\href{" + target + "}{" + AhdLatexEscape(text) + "}", ""
}

const (
	ahdPageStyleMarker = "% AHDCODE_PAGESTYLE"
	ahdLastPageMarker  = "% AHDCODE_LASTPAGE"
	ahdBookmarkMarker  = "% AHDCODE_BOOKMARK"
	ahdImageMarker     = "% AHDCODE_IMAGE"
)

// ahdDocumentHasMarker reports whether marker appears on a line of its own.
func ahdDocumentHasMarker(marker string, texts ...string) bool {
	for _, text := range texts {
		if strings.Contains("\n"+text+"\n", "\n"+marker+"\n") {
			return true
		}
	}
	return false
}

// AhdLatexBookmarkText is Latex.bookmark: a PDF outline entry pointing at
// the current position. Level 1 is a top-level entry; 2 to 4 nest below the
// nearest preceding entry of a smaller level.
func AhdLatexBookmarkText(operation, title string, level int64) (string, string) {
	if strings.TrimSpace(title) == "" {
		return "", operation + " title must not be empty"
	}
	if level < 1 || level > 4 {
		return "", operation + " level must be between 1 and 4; received " + strconv.FormatInt(level, 10)
	}
	return "%\n" + ahdBookmarkMarker + "\n\\ahdbookmark{" + strconv.FormatInt(level-1, 10) + "}{" + AhdLatexEscape(title) + "}%\n", ""
}

// ahdDocumentMetadata builds the PDF document properties. It returns "" when
// none of the v1.3.0 properties is set, so an existing document stays
// byte-for-byte what it was; once subject, keywords, or creator is set, the
// document's title and author are recorded as properties too.
func ahdDocumentMetadata(operation, title, author, subject string, keywords []string, creator string) (string, string) {
	for _, keyword := range keywords {
		if strings.TrimSpace(keyword) == "" || strings.Contains(keyword, ",") {
			return "", operation + " keywords must be non-empty and must not contain commas"
		}
	}
	if subject == "" && len(keywords) == 0 && creator == "" {
		return "", ""
	}
	var fields []string
	add := func(name, value string) {
		if value != "" {
			fields = append(fields, name+"={"+AhdLatexEscape(value)+"}")
		}
	}
	add("pdftitle", title)
	add("pdfauthor", author)
	add("pdfsubject", subject)
	add("pdfkeywords", strings.Join(keywords, ", "))
	add("pdfcreator", creator)
	return "\\hypersetup{" + strings.Join(fields, ",") + "}\n", ""
}

// ---------------------------------------------------------------------------
// Headers, footers, page numbers, and placement
// ---------------------------------------------------------------------------

// AhdLatexRunningText builds Latex.header (part "head") or Latex.footer (part
// "foot"). The three regions hold generated Latex -- text, a page number, an
// image, or a QR symbol -- so ordinary text should pass through
// Latex.escape. The fragment takes effect from the page it is reached on.
func AhdLatexRunningText(part, left, center, right string) string {
	// Each region starts on a line of its own, so a marker at the start of
	// region content (an image's asset marker, say) stays at a line start.
	return "%\n" + ahdPageStyleMarker + "\n\\fancy" + part + "[L]{%\n" + left + "}%\n\\fancy" + part + "[C]{%\n" + center +
		"}%\n\\fancy" + part + "[R]{%\n" + right + "}%\n"
}

// AhdLatexPageNumberText and AhdLatexPageCountText are the current page and
// the document's total page count, for headers, footers, and body text.
func AhdLatexPageNumberText() string { return "\\thepage{}" }

func AhdLatexPageCountText() string {
	return "%\n" + ahdLastPageMarker + "\n\\pageref*{LastPage}"
}

var ahdLatexPlaceAnchors = map[string]string{
	"north west": "[tl]", "north": "[t]", "north east": "[tr]",
	"west": "[l]", "center": "", "east": "[r]",
	"south west": "[bl]", "south": "[b]", "south east": "[br]",
}

// AhdLatexPlaceText is Latex.place: content positioned against the physical
// page, x centimeters from its left edge and y centimeters from its top edge,
// by the chosen point of the content. It is drawn in the page foreground the
// way Latex.overlay is, so it never moves the page's text.
func AhdLatexPlaceText(content string, x, y float64, anchor string) (string, string) {
	position, known := ahdLatexPlaceAnchors[anchor]
	if !known {
		return "", "Latex.place anchor must be north west, north, north east, west, center, east, south west, south, or south east; received " + strconv.Quote(anchor)
	}
	for _, coordinate := range []struct {
		name  string
		value float64
	}{{"x", x}, {"y", y}} {
		if math.IsNaN(coordinate.value) || coordinate.value < 0 || coordinate.value > ahdDocumentMaximumCentimeters {
			return "", "Latex.place " + coordinate.name + " must be between 0 and 1000 centimeters; received " + ahdFormatReal(coordinate.value)
		}
	}
	return "%\n\\AddToHookNext{shipout/foreground}{{\\setlength{\\unitlength}{1cm}\\put(" + ahdFormatReal(x) + ",-" + ahdFormatReal(y) +
		"){\\makebox(0,0)" + position + "{%\n" + content + "}}}}%\n", ""
}

// AhdLatexFeaturePreamble returns the preamble lines the v1.3.0 fragments in
// texts need, in a fixed order, or "" when there are none.
func AhdLatexFeaturePreamble(texts ...string) string {
	var result strings.Builder
	if ahdDocumentHasMarker(ahdPageStyleMarker, texts...) {
		// Headers and footers start empty apart from the page number LaTeX
		// already shows, draw no rules, and apply to every page, including
		// title and chapter pages.
		result.WriteString("\\usepackage{fancyhdr}\n\\fancyhf{}\n\\fancyfoot[C]{\\thepage}\n" +
			"\\renewcommand{\\headrulewidth}{0pt}\n\\renewcommand{\\footrulewidth}{0pt}\n\\pagestyle{fancy}\n" +
			"\\makeatletter\n\\let\\ps@plain\\ps@fancy\n\\makeatother\n")
	}
	if ahdDocumentHasMarker(ahdLastPageMarker, texts...) {
		result.WriteString("\\usepackage{lastpage}\n")
	}
	if ahdDocumentHasMarker(ahdBookmarkMarker, texts...) {
		result.WriteString("\\newcounter{ahdbookmark}\n" +
			"\\newcommand{\\ahdbookmark}[2]{\\stepcounter{ahdbookmark}\\pdfbookmark[#1]{#2}{ahdbookmark.\\theahdbookmark}}\n")
	}
	if ahdDocumentHasMarker(ahdImageMarker, texts...) {
		result.WriteString(ahdImageMacros)
	}
	return result.String()
}

// ahdImageMacros trims a box to a clipped PGF picture and draws a box at a
// fixed opacity. Both keep the result vector.
const ahdImageMacros = "\\newsavebox{\\ahdimagebox}\n" +
	"\\newcommand{\\ahdimagetrim}[5]{\\sbox{\\ahdimagebox}{#5}\\begin{pgfpicture}" +
	"\\pgfpathrectangle{\\pgfqpoint{0pt}{0pt}}{\\pgfqpoint{\\dimexpr\\wd\\ahdimagebox-#1cm-#3cm\\relax}{\\dimexpr\\ht\\ahdimagebox+\\dp\\ahdimagebox-#2cm-#4cm\\relax}}" +
	"\\pgfusepath{use as bounding box,clip}\\pgftext[left,bottom,at={\\pgfqpoint{-#1cm}{-#4cm}}]{\\usebox{\\ahdimagebox}}\\end{pgfpicture}}\n" +
	"\\newcommand{\\ahdimageopacity}[2]{\\begin{pgfpicture}\\pgfsetfillopacity{#1}\\pgfsetstrokeopacity{#1}\\pgftext[left,bottom]{#2}\\end{pgfpicture}}\n"

// ---------------------------------------------------------------------------
// Images
// ---------------------------------------------------------------------------

// AhdImageTransform is a validated image transform: rotation in degrees
// counterclockwise, opacity from 0 to 1, and trims in centimeters removed
// from each edge of the sized image.
type AhdImageTransform struct {
	Rotation, Opacity                        float64
	TrimLeft, TrimTop, TrimRight, TrimBottom float64
	rotate, fade, trim                       bool
}

// AhdImageTransformText validates a transform Pair's entries.
func AhdImageTransformText(operation string, keys []string, values []float64) (AhdImageTransform, string) {
	transform := AhdImageTransform{Opacity: 1}
	for index, key := range keys {
		value := values[index]
		switch key {
		case "rotation":
			if math.IsNaN(value) || value < -360 || value > 360 {
				return transform, operation + " transform rotation must be between -360 and 360 degrees; received " + ahdFormatReal(value)
			}
			transform.Rotation, transform.rotate = value, value != 0
		case "opacity":
			if math.IsNaN(value) || value < 0 || value > 1 {
				return transform, operation + " transform opacity must be between 0.0 and 1.0; received " + ahdFormatReal(value)
			}
			transform.Opacity, transform.fade = value, value < 1
		case "trimLeft", "trimTop", "trimRight", "trimBottom":
			if math.IsNaN(value) || value < 0 || value > ahdDocumentMaximumCentimeters {
				return transform, operation + " transform " + key + " must be between 0 and 1000 centimeters; received " + ahdFormatReal(value)
			}
			switch key {
			case "trimLeft":
				transform.TrimLeft = value
			case "trimTop":
				transform.TrimTop = value
			case "trimRight":
				transform.TrimRight = value
			default:
				transform.TrimBottom = value
			}
			transform.trim = transform.trim || value > 0
		default:
			return transform, operation + " transform supports only rotation, opacity, trimLeft, trimTop, trimRight, and trimBottom; received " + strconv.Quote(key)
		}
	}
	return transform, ""
}

// checkTrim rejects trims that would remove the whole image when its size is
// known in advance.
func (transform AhdImageTransform) checkTrim(operation string, width, height float64) string {
	if width > 0 && transform.TrimLeft+transform.TrimRight >= width {
		return operation + " transform trimLeft and trimRight remove the whole " + ahdFormatReal(width) + " cm width"
	}
	if height > 0 && transform.TrimTop+transform.TrimBottom >= height {
		return operation + " transform trimTop and trimBottom remove the whole " + ahdFormatReal(height) + " cm height"
	}
	return ""
}

// NeedsPGF reports whether the transform draws through PGF.
func (transform AhdImageTransform) NeedsPGF() bool { return transform.trim || transform.fade }

// Wrap applies trim, then rotation, then opacity to a sized image box.
func (transform AhdImageTransform) Wrap(content string) string {
	if transform.trim {
		content = "\\ahdimagetrim{" + ahdFormatReal(transform.TrimLeft) + "}{" + ahdFormatReal(transform.TrimTop) + "}{" +
			ahdFormatReal(transform.TrimRight) + "}{" + ahdFormatReal(transform.TrimBottom) + "}{" + content + "}"
	}
	if transform.rotate {
		content = "\\rotatebox{" + ahdFormatReal(transform.Rotation) + "}{" + content + "}"
	}
	if transform.fade {
		content = "\\ahdimageopacity{" + ahdFormatReal(transform.Opacity) + "}{" + content + "}"
	}
	return content
}

// ahdImageSize validates a size Pair's entries: width and height in
// centimeters, both optional.
func ahdImageSize(operation string, keys []string, values []float64) (width, height float64, problem string) {
	seen := map[string]bool{}
	for index, key := range keys {
		if key != "width" && key != "height" {
			return 0, 0, operation + " size supports only width and height"
		}
		if seen[key] {
			return 0, 0, "duplicate Latex image size option"
		}
		seen[key] = true
		if values[index] <= 0 {
			return 0, 0, operation + " dimensions must be positive"
		}
		if key == "width" {
			width = values[index]
		} else {
			height = values[index]
		}
	}
	return width, height, ""
}

// AhdLatexImageText builds Latex.image and Latex.figure fragments. A PNG,
// PDF, or JPEG asset is placed with \includegraphics; an SVG asset is
// converted to vector PGF when the document compiles and scaled as a box.
// With no transform, a PNG, PDF, or JPEG fragment is exactly what v1.2.0
// produced.
func AhdLatexImageText(path string, sizeKeys []string, sizeValues []float64, transformKeys []string, transformValues []float64,
	figure bool, caption, label string) (string, string) {
	if path == "" {
		return "", "Latex image path must not be empty"
	}
	extension := strings.ToLower(ahdPathExtension(path))
	svg := extension == ".svg"
	if !svg && extension != ".png" && extension != ".pdf" && extension != ".jpg" && extension != ".jpeg" {
		return "", "Latex image supports PNG, PDF, JPEG, and SVG assets"
	}
	width, height, problem := ahdImageSize("Latex image", sizeKeys, sizeValues)
	if problem != "" {
		return "", problem
	}
	transform, problem := AhdImageTransformText("Latex image", transformKeys, transformValues)
	if problem != "" {
		return "", problem
	}
	if problem := transform.checkTrim("Latex image", width, height); problem != "" {
		return "", problem
	}
	marker, staged := ahdLatexAssetMarker(path, extension)
	var content string
	if svg {
		content = ahdLatexScaledBox(width, height, "\\input{"+AhdLatexSVGInputName(staged)+"}")
	} else {
		options := []string{}
		for index, key := range sizeKeys {
			options = append(options, key+"="+ahdFormatReal(sizeValues[index])+"cm")
		}
		content = "\\includegraphics"
		if len(options) > 0 {
			content += "[" + strings.Join(options, ",") + "]"
		}
		content += "{" + staged + "}"
	}
	if svg || transform.NeedsPGF() {
		marker += "%\n% AHDCODE_TIKZ\n"
	}
	if transform.NeedsPGF() {
		marker += ahdImageMarker + "\n"
	}
	content = transform.Wrap(content)
	if figure {
		return marker + "\\begin{figure}[!ht]\n\\centering\n" + content + "\n\\caption{" + AhdLatexEscape(caption) + "}\n" +
			ahdLatexLabel(label) + "\\end{figure}\n", ""
	}
	return marker + content + "\n", ""
}

// ahdLatexScaledBox scales a box to width and/or height centimeters,
// preserving its aspect ratio when only one is given.
func ahdLatexScaledBox(width, height float64, box string) string {
	if width <= 0 && height <= 0 {
		return "\\mbox{" + box + "}"
	}
	scale := func(value float64) string {
		if value <= 0 {
			return "!"
		}
		return ahdFormatReal(value) + "cm"
	}
	return "\\resizebox{" + scale(width) + "}{" + scale(height) + "}{" + box + "}"
}

// AhdLatexSVGInputName is the staged file holding a converted SVG asset.
func AhdLatexSVGInputName(staged string) string {
	return strings.TrimSuffix(staged, ".svg") + "-svg.tex"
}

func ahdPathExtension(path string) string {
	for index := len(path) - 1; index >= 0 && path[index] != '/' && path[index] != '\\'; index-- {
		if path[index] == '.' {
			return path[index:]
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Latex.document
// ---------------------------------------------------------------------------

// AhdLatexDocumentOptions carries every Latex.document parameter.
type AhdLatexDocumentOptions struct {
	Body, Title, Author, Date, Type string
	Margin                          float64
	Color, Cover                    string
	TheoremNames, TheoremRules      []string
	Theme                           string
	Landscape                       bool
	Paper                           string
	PageSizeKeys                    []string
	PageSizeValues                  []float64
	MarginKeys                      []string
	MarginValues                    []float64
	Subject                         string
	Keywords                        []string
	Creator                         string
}

var ahdLatexDocumentClasses = map[string]string{"Article": "article", "Report": "report", "Beamer": "beamer"}

// AhdLatexDocumentText builds one complete, stable document. Font files are
// named explicitly so the supported baseline never depends on a system font.
// A call using only the v1.2.0 parameters produces exactly the v1.2.0 source.
func AhdLatexDocumentText(options AhdLatexDocumentOptions) (string, string) {
	documentClass := ahdLatexDocumentClasses[options.Type]
	if documentClass == "" {
		return "", "Latex.document type must be Article, Report, or Beamer"
	}
	if options.Margin <= 0 {
		return "", "Latex.document margin must be positive"
	}
	if options.Color != "" && !ahdLatexColorPattern.MatchString(options.Color) {
		return "", "Latex.document color must use #RRGGBB"
	}
	if !ahdLatexBeamerThemes[options.Theme] {
		return "", "Latex.document theme must be Default, Madrid, or Warsaw"
	}
	if options.Theme != "Default" && options.Type != "Beamer" {
		return "", "Latex.document theme requires a Beamer document"
	}
	if options.Landscape && options.Type == "Beamer" {
		return "", "Latex.document landscape requires an Article or Report document"
	}
	paper := options.Paper
	if paper == "" {
		paper = "Letter"
	}
	layout, problem := AhdPageLayoutText("Latex.document", "Letter", paper, options.Landscape, options.Margin,
		options.PageSizeKeys, options.PageSizeValues, options.MarginKeys, options.MarginValues)
	if problem != "" {
		return "", problem
	}
	if layout.Configured && options.Type == "Beamer" {
		return "", "Latex.document paper, pageSize, and margins require an Article or Report document"
	}
	if options.Type == "Beamer" && ahdDocumentHasMarker(ahdPageStyleMarker, options.Cover, options.Body) {
		return "", "Latex.document header and footer require an Article or Report document"
	}
	metadata, problem := ahdDocumentMetadata("Latex.document", options.Title, options.Author, options.Subject, options.Keywords, options.Creator)
	if problem != "" {
		return "", problem
	}
	tikz, problem := AhdLatexTikZPreamble(options.Cover, options.Body)
	if problem != "" {
		return "", problem
	}
	var result strings.Builder
	result.WriteString("\\documentclass{" + documentClass + "}\n")
	if options.Theme != "Default" {
		// A non-Default theme already implies a Beamer document, checked above.
		result.WriteString("\\usetheme{" + options.Theme + "}\n")
	}
	result.WriteString("\\usepackage{fontspec}\n")
	result.WriteString("\\setmainfont{lmroman10-regular.otf}[BoldFont=lmroman10-bold.otf,ItalicFont=lmroman10-italic.otf,BoldItalicFont=lmroman10-bolditalic.otf]\n")
	result.WriteString("\\usepackage{amsmath,amssymb,mathtools}\n")
	result.WriteString("\\usepackage{geometry,graphicx,booktabs,array,xcolor,hyperref}\n")
	geometry := "margin=" + ahdFormatReal(options.Margin) + "cm"
	if options.Landscape {
		geometry = "landscape," + geometry
	}
	if layout.Configured {
		geometry = layout.Geometry()
	}
	result.WriteString("\\geometry{" + geometry + "}\n")
	result.WriteString("\\hypersetup{hidelinks}\n")
	result.WriteString(metadata)
	if options.Color != "" {
		result.WriteString("\\definecolor{ahdaccent}{HTML}{" + strings.ToUpper(strings.TrimPrefix(options.Color, "#")) + "}\n")
		if options.Type == "Beamer" {
			result.WriteString("\\setbeamercolor{structure}{fg=ahdaccent}\n")
		}
	}
	result.WriteString(tikz)
	result.WriteString(AhdLatexFeaturePreamble(options.Cover, options.Body))
	declared := map[string]string{}
	for index, display := range options.TheoremNames {
		if display == "" {
			return "", "theorem type name must not be empty"
		}
		id := ahdLatexTheoremID(display)
		rule := options.TheoremRules[index]
		switch rule {
		case "":
			result.WriteString("\\newtheorem{" + id + "}{" + AhdLatexEscape(display) + "}\n")
		case "section", "subsection":
			result.WriteString("\\newtheorem{" + id + "}{" + AhdLatexEscape(display) + "}[" + rule + "]\n")
		case "chapter":
			if options.Type != "Report" {
				return "", "chapter theorem counters require a Report document"
			}
			result.WriteString("\\newtheorem{" + id + "}{" + AhdLatexEscape(display) + "}[chapter]\n")
		default:
			shared := declared[rule]
			if shared == "" {
				return "", "theorem counter references an unknown or later type: " + rule
			}
			result.WriteString("\\newtheorem{" + id + "}[" + shared + "]{" + AhdLatexEscape(display) + "}\n")
		}
		declared[display] = id
	}
	knownTheorems := map[string]bool{}
	for _, id := range declared {
		knownTheorems[id] = true
	}
	for _, match := range ahdLatexTheoremPattern.FindAllStringSubmatch(options.Body, -1) {
		if !knownTheorems[match[1]] {
			return "", "document body uses an undeclared theorem type"
		}
	}
	if options.Title != "" {
		result.WriteString("\\title{" + AhdLatexEscape(options.Title) + "}\n")
	}
	if options.Author != "" {
		result.WriteString("\\author{" + AhdLatexEscape(options.Author) + "}\n")
	}
	result.WriteString("\\date{" + AhdLatexEscape(options.Date) + "}\n\\begin{document}\n")
	if options.Cover != "" {
		result.WriteString(options.Cover)
		if !strings.HasSuffix(options.Cover, "\n") {
			result.WriteByte('\n')
		}
		result.WriteString("\\clearpage\n")
	}
	if options.Title != "" {
		if options.Type == "Beamer" {
			result.WriteString("\\begin{frame}\n\\titlepage\n\\end{frame}\n")
		} else {
			result.WriteString("\\maketitle\n")
		}
	}
	result.WriteString(options.Body)
	if options.Body != "" && !strings.HasSuffix(options.Body, "\n") {
		result.WriteByte('\n')
	}
	result.WriteString("\\end{document}\n")
	return result.String(), ""
}

// ---------------------------------------------------------------------------
// Native entry points
// ---------------------------------------------------------------------------

func ahdLatexValue(text, problem string) string {
	if problem != "" {
		AhdRaiseClass(AhdClassValueError, problem)
	}
	return text
}

// AhdLatexDocumentComplete is the native entry point of Latex.document.
func AhdLatexDocumentComplete(body, title, author, date, documentType string, margin float64, color, cover string,
	theorems *AhdPair[string, string], theme string, landscape bool, paper string, pageSize, margins *AhdPair[string, float64],
	subject string, keywords *AhdList[string], creator string) string {
	names, rules := ahdPairEntries(theorems)
	sizeKeys, sizeValues := ahdPairEntries(pageSize)
	marginKeys, marginValues := ahdPairEntries(margins)
	var keywordValues []string
	if keywords != nil {
		keywordValues = keywords.Snapshot()
	}
	return ahdLatexValue(AhdLatexDocumentText(AhdLatexDocumentOptions{
		Body: body, Title: title, Author: author, Date: date, Type: documentType, Margin: margin, Color: color, Cover: cover,
		TheoremNames: names, TheoremRules: rules, Theme: theme, Landscape: landscape, Paper: paper,
		PageSizeKeys: sizeKeys, PageSizeValues: sizeValues, MarginKeys: marginKeys, MarginValues: marginValues,
		Subject: subject, Keywords: keywordValues, Creator: creator,
	}))
}

// AhdLatexImageComplete is the native entry point of Latex.image.
func AhdLatexImageComplete(path string, size, transform *AhdPair[string, float64]) string {
	sizeKeys, sizeValues := ahdPairEntries(size)
	transformKeys, transformValues := ahdPairEntries(transform)
	return ahdLatexValue(AhdLatexImageText(path, sizeKeys, sizeValues, transformKeys, transformValues, false, "", ""))
}

// AhdLatexFigureComplete is the native entry point of Latex.figure.
func AhdLatexFigureComplete(path, caption, label string, size, transform *AhdPair[string, float64]) string {
	sizeKeys, sizeValues := ahdPairEntries(size)
	transformKeys, transformValues := ahdPairEntries(transform)
	return ahdLatexValue(AhdLatexImageText(path, sizeKeys, sizeValues, transformKeys, transformValues, true, caption, label))
}

func AhdLatexPlace(content string, x, y float64, anchor string) string {
	return ahdLatexValue(AhdLatexPlaceText(content, x, y, anchor))
}

func AhdLatexHeader(left, center, right string) string {
	return AhdLatexRunningText("head", left, center, right)
}
func AhdLatexFooter(left, center, right string) string {
	return AhdLatexRunningText("foot", left, center, right)
}

func AhdLatexLink(text, url string) string {
	return ahdLatexValue(AhdLatexLinkText("Latex.link", text, url))
}

func AhdLatexBookmark(title string, level int64) string {
	return ahdLatexValue(AhdLatexBookmarkText("Latex.bookmark", title, level))
}
