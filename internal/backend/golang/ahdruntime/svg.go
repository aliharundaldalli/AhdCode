package ahdruntime

// SVG assets for Latex and PDF documents. An SVG is converted, in pure Go,
// into PGF drawing commands that the existing offline Tectonic renderer draws,
// so an SVG logo stays vector in the final PDF. Nothing is rasterized; no
// browser, Inkscape, shell command, or network is involved; and a converted
// drawing is built from the SVG bytes alone -- it never reads another file.
//
// The supported subset is static vector artwork: paths, rectangles, circles,
// ellipses, lines, polylines, and polygons with solid fills and strokes,
// opacity, transforms, a viewBox, same-document <use>, and simple <style>
// rules. Everything else -- text, gradients, clipping, masks, filters,
// images, scripts, foreignObject, animation, and every external reference --
// is rejected with a message naming what is unsupported, never silently
// dropped. This file uses only the Go standard library.

import (
	"bytes"
	"encoding/xml"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
)

const (
	ahdSVGNamespace        = "http://www.w3.org/2000/svg"
	ahdSVGXLinkNamespace   = "http://www.w3.org/1999/xlink"
	ahdSVGMaximumBytes     = 5 << 20
	ahdSVGMaximumElements  = 100000
	ahdSVGMaximumDepth     = 64
	ahdSVGMaximumSegments  = 2000000
	ahdSVGMaximumUseDepth  = 16
	ahdSVGCoordinateLimit  = 16000.0 // bp; TeX dimensions end near 16383pt
	ahdSVGPixelInPoints    = 0.75    // CSS px = 1/96 in; PostScript point = 1/72 in
	ahdSVGMaximumSizePoint = 14000.0
)

// AhdSVGPicture is one converted SVG: PGF drawing source whose bounding box
// is exactly Width by Height PostScript points (the SVG's natural size).
type AhdSVGPicture struct {
	Source string
	Width  float64
	Height float64
}

type ahdSVGNode struct {
	name     string
	attrs    map[string]string
	children []*ahdSVGNode
	text     string
}

type ahdSVGColor struct {
	r, g, b uint8
}

type ahdSVGPaint struct {
	none    bool
	current bool
	color   ahdSVGColor
}

type ahdSVGStyle struct {
	fill, stroke      ahdSVGPaint
	fillOpacity       float64
	strokeOpacity     float64
	opacity           float64
	strokeWidth       float64
	lineCap, lineJoin string
	miterLimit        float64
	dash              []float64
	dashOffset        float64
	evenOdd           bool
	color             ahdSVGColor
	hidden            bool
}

type ahdSVGMatrix [6]float64 // a b c d e f

var ahdSVGIdentity = ahdSVGMatrix{1, 0, 0, 1, 0, 0}

func (m ahdSVGMatrix) multiply(n ahdSVGMatrix) ahdSVGMatrix {
	return ahdSVGMatrix{
		m[0]*n[0] + m[2]*n[1], m[1]*n[0] + m[3]*n[1],
		m[0]*n[2] + m[2]*n[3], m[1]*n[2] + m[3]*n[3],
		m[0]*n[4] + m[2]*n[5] + m[4], m[1]*n[4] + m[3]*n[5] + m[5],
	}
}

func (m ahdSVGMatrix) apply(x, y float64) (float64, float64) {
	return m[0]*x + m[2]*y + m[4], m[1]*x + m[3]*y + m[5]
}

func (m ahdSVGMatrix) scale() float64 {
	return math.Sqrt(math.Abs(m[0]*m[3] - m[1]*m[2]))
}

// ahdSVGSegment is one path step in user space: move, line, cubic curve, or
// close. Quadratic curves, arcs, and shapes are converted to these.
type ahdSVGSegment struct {
	op     byte
	points [3][2]float64
}

type ahdSVGConverter struct {
	root      *ahdSVGNode
	ids       map[string]*ahdSVGNode
	rules     []ahdSVGRule
	out       strings.Builder
	colors    map[ahdSVGColor]string
	colorList []ahdSVGColor
	segments  int
	height    float64
}

type ahdSVGRule struct {
	selector     string
	specificity  int
	order        int
	declarations map[string]string
}

var ahdSVGUnsupportedElements = map[string]string{
	"script":           "scripts are not allowed in document SVG",
	"foreignObject":    "foreignObject is not allowed in document SVG",
	"image":            "embedded or linked raster images are not supported in document SVG",
	"text":             "SVG text is not supported; convert text to outlines (paths) in the SVG editor",
	"tspan":            "SVG text is not supported; convert text to outlines (paths) in the SVG editor",
	"textPath":         "SVG text is not supported; convert text to outlines (paths) in the SVG editor",
	"linearGradient":   "gradients are not supported; use solid fills",
	"radialGradient":   "gradients are not supported; use solid fills",
	"clipPath":         "clipping paths are not supported",
	"mask":             "masks are not supported",
	"pattern":          "pattern fills are not supported",
	"filter":           "filters are not supported",
	"marker":           "markers are not supported",
	"symbol":           "symbol is not supported; use <g> inside <defs>",
	"switch":           "switch is not supported",
	"a":                "links (<a>) are not allowed in document SVG",
	"animate":          "animation is not supported",
	"animateMotion":    "animation is not supported",
	"animateTransform": "animation is not supported",
	"set":              "animation is not supported",
	"iframe":           "iframe is not allowed in document SVG",
	"video":            "video is not allowed in document SVG",
	"audio":            "audio is not allowed in document SVG",
	"svg":              "nested <svg> elements are not supported",
}

var ahdSVGDrawable = map[string]bool{"path": true, "rect": true, "circle": true, "ellipse": true, "line": true, "polyline": true, "polygon": true}

// AhdSVGConvert validates and converts SVG bytes. operation prefixes every
// problem so a failure names the call that received the asset. It never
// raises, so Latex, PDF, and the evaluator share it unchanged.
func AhdSVGConvert(operation string, data []byte) (AhdSVGPicture, string) {
	fail := func(message string) (AhdSVGPicture, string) {
		return AhdSVGPicture{}, operation + ": " + message
	}
	if len(data) > ahdSVGMaximumBytes {
		return fail("SVG is larger than 5 MiB")
	}
	root, problem := ahdSVGParse(data)
	if problem != "" {
		return fail(problem)
	}
	converter := &ahdSVGConverter{root: root, ids: map[string]*ahdSVGNode{}, colors: map[ahdSVGColor]string{}}
	if problem := converter.validate(root, true); problem != "" {
		return fail(problem)
	}
	width, height, viewport, problem := ahdSVGViewport(root)
	if problem != "" {
		return fail(problem)
	}
	converter.height = height
	var body strings.Builder
	style := ahdSVGStyle{
		fill: ahdSVGPaint{color: ahdSVGColor{}}, stroke: ahdSVGPaint{none: true},
		fillOpacity: 1, strokeOpacity: 1, opacity: 1, strokeWidth: 1,
		lineCap: "butt", lineJoin: "miter", miterLimit: 4,
	}
	if problem := converter.render(root, style, viewport, 0, map[*ahdSVGNode]bool{}); problem != "" {
		return fail(problem)
	}
	body.WriteString("\\begin{pgfpicture}%\n")
	body.WriteString("\\pgfpathrectangle{\\pgfqpoint{0bp}{0bp}}{\\pgfqpoint{" + ahdSVGNumber(width) + "bp}{" + ahdSVGNumber(height) + "bp}}%\n")
	body.WriteString("\\pgfusepath{use as bounding box,clip}%\n")
	for index, color := range converter.colorList {
		body.WriteString("\\definecolor{ahdsvg" + strconv.Itoa(index) + "}{RGB}{" + strconv.Itoa(int(color.r)) + "," +
			strconv.Itoa(int(color.g)) + "," + strconv.Itoa(int(color.b)) + "}%\n")
	}
	body.WriteString(converter.out.String())
	body.WriteString("\\end{pgfpicture}%\n")
	return AhdSVGPicture{Source: body.String(), Width: width, Height: height}, ""
}

// ahdSVGParse builds the element tree. It refuses a DOCTYPE (and with it any
// entity definition), processing instructions other than the XML
// declaration, and element nesting beyond a fixed depth. Elements outside the
// SVG namespace are editor metadata and are skipped with their contents.
func ahdSVGParse(data []byte) (*ahdSVGNode, string) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true
	var stack []*ahdSVGNode
	var root *ahdSVGNode
	skipping := 0
	elements := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "SVG is not well-formed XML: " + err.Error()
		}
		switch value := token.(type) {
		case xml.Directive:
			return nil, "SVG must not contain a DOCTYPE or entity declarations"
		case xml.ProcInst:
			if value.Target != "xml" {
				return nil, "SVG must not contain the processing instruction <?" + value.Target + "?>"
			}
		case xml.StartElement:
			if skipping > 0 {
				skipping++
				continue
			}
			if value.Name.Space != "" && value.Name.Space != ahdSVGNamespace {
				skipping = 1
				continue
			}
			elements++
			if elements > ahdSVGMaximumElements {
				return nil, "SVG has more than 100000 elements"
			}
			if len(stack) >= ahdSVGMaximumDepth {
				return nil, "SVG elements are nested more than 64 levels deep"
			}
			node := &ahdSVGNode{name: value.Name.Local, attrs: map[string]string{}}
			for _, attribute := range value.Attr {
				switch attribute.Name.Space {
				case "", ahdSVGNamespace:
					node.attrs[attribute.Name.Local] = attribute.Value
				case ahdSVGXLinkNamespace:
					node.attrs["xlink:"+attribute.Name.Local] = attribute.Value
				case "xmlns":
				default:
					// Attributes in other namespaces (editor metadata) never
					// affect rendering.
				}
			}
			if root == nil {
				if node.name != "svg" {
					return nil, "the root element must be <svg>, not <" + node.name + ">"
				}
				root = node
			} else if len(stack) == 0 {
				return nil, "SVG has more than one root element"
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, node)
			}
			stack = append(stack, node)
		case xml.EndElement:
			if skipping > 0 {
				skipping--
				continue
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if skipping == 0 && len(stack) > 0 {
				stack[len(stack)-1].text += string(value)
			}
		}
	}
	if root == nil {
		return nil, "SVG has no <svg> root element"
	}
	return root, ""
}

// validate enforces the security and support policy before anything is
// drawn, and indexes element ids for <use>.
func (converter *ahdSVGConverter) validate(node *ahdSVGNode, isRoot bool) string {
	if message, rejected := ahdSVGUnsupportedElements[node.name]; rejected && !(isRoot && node.name == "svg") {
		return message
	}
	known := isRoot || ahdSVGDrawable[node.name] || node.name == "g" || node.name == "use" || node.name == "defs" ||
		node.name == "title" || node.name == "desc" || node.name == "metadata" || node.name == "style"
	if !known {
		return "<" + node.name + "> is not supported in document SVG"
	}
	for name, value := range node.attrs {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "on") {
			return "event attribute " + name + " is not allowed in document SVG"
		}
		if strings.Contains(strings.ToLower(value), "url(") {
			return "references such as url(...) are not supported (in " + name + " of <" + node.name + ">)"
		}
		if lower == "href" || lower == "xlink:href" {
			if node.name != "use" || !ahdSVGLocalReference(value) {
				return "<" + node.name + "> " + name + " must reference an element in the same SVG (#id); external and file references are not allowed"
			}
		}
		if lower == "id" {
			converter.ids[value] = node
		}
	}
	if node.name == "style" {
		if kind := node.attrs["type"]; kind != "" && kind != "text/css" {
			return "<style> must contain CSS"
		}
		rules, problem := ahdSVGParseStyleSheet(node.text, len(converter.rules))
		if problem != "" {
			return problem
		}
		converter.rules = append(converter.rules, rules...)
		return ""
	}
	if node.name == "title" || node.name == "desc" || node.name == "metadata" {
		return ""
	}
	if strings.TrimSpace(node.text) != "" {
		return "character data inside <" + node.name + "> is not supported"
	}
	for _, child := range node.children {
		if problem := converter.validate(child, false); problem != "" {
			return problem
		}
	}
	return ""
}

func ahdSVGLocalReference(value string) bool {
	if len(value) < 2 || value[0] != '#' {
		return false
	}
	for _, character := range value[1:] {
		if !(character == '-' || character == '_' || character == '.' || character == ':' ||
			(character >= '0' && character <= '9') || (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')) {
			return false
		}
	}
	return true
}

// ahdSVGParseStyleSheet reads the narrow CSS a <style> element may carry:
// rules whose selectors are an element name, .class, #id, or *, with
// declarations. At-rules and anything that could load a resource are refused.
func ahdSVGParseStyleSheet(text string, order int) ([]ahdSVGRule, string) {
	for {
		start := strings.Index(text, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(text[start+2:], "*/")
		if end < 0 {
			return nil, "unterminated CSS comment in <style>"
		}
		text = text[:start] + " " + text[start+2+end+2:]
	}
	text = strings.ReplaceAll(strings.ReplaceAll(text, "<![CDATA[", ""), "]]>", "")
	if strings.Contains(text, "@") {
		return nil, "CSS at-rules such as @import and @font-face are not supported in <style>"
	}
	var rules []ahdSVGRule
	for _, block := range strings.Split(text, "}") {
		if strings.TrimSpace(block) == "" {
			continue
		}
		open := strings.Index(block, "{")
		if open < 0 {
			return nil, "malformed CSS rule in <style>"
		}
		declarations, problem := ahdSVGDeclarations(block[open+1:])
		if problem != "" {
			return nil, problem
		}
		for _, selector := range strings.Split(block[:open], ",") {
			selector = strings.TrimSpace(selector)
			specificity := -1
			switch {
			case selector == "*":
				specificity = 0
			case strings.HasPrefix(selector, "#") && ahdSVGIdentifier(selector[1:]):
				specificity = 100
			case strings.HasPrefix(selector, ".") && ahdSVGIdentifier(selector[1:]):
				specificity = 10
			case ahdSVGIdentifier(selector):
				specificity = 1
			}
			if specificity < 0 {
				return nil, "CSS selector " + strconv.Quote(selector) + " is not supported; use an element name, .class, #id, or *"
			}
			rules = append(rules, ahdSVGRule{selector: selector, specificity: specificity, order: order, declarations: declarations})
			order++
		}
	}
	return rules, ""
}

func ahdSVGIdentifier(text string) bool {
	if text == "" {
		return false
	}
	for _, character := range text {
		if !(character == '-' || character == '_' || (character >= '0' && character <= '9') ||
			(character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')) {
			return false
		}
	}
	return true
}

func ahdSVGDeclarations(text string) (map[string]string, string) {
	result := map[string]string{}
	for _, declaration := range strings.Split(text, ";") {
		if strings.TrimSpace(declaration) == "" {
			continue
		}
		colon := strings.Index(declaration, ":")
		if colon < 0 {
			return nil, "malformed CSS declaration " + strconv.Quote(strings.TrimSpace(declaration))
		}
		value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(declaration[colon+1:]), "!important"))
		result[strings.ToLower(strings.TrimSpace(declaration[:colon]))] = value
	}
	return result, ""
}

// ahdSVGViewport resolves the natural size in points and the matrix mapping
// user space onto it (y still pointing down).
func ahdSVGViewport(root *ahdSVGNode) (float64, float64, ahdSVGMatrix, string) {
	var viewBox []float64
	if text, present := root.attrs["viewBox"]; present {
		numbers, ok := ahdSVGNumbers(text)
		if !ok || len(numbers) != 4 || numbers[2] <= 0 || numbers[3] <= 0 {
			return 0, 0, ahdSVGIdentity, "viewBox must be four numbers with a positive width and height"
		}
		viewBox = numbers
	}
	width, hasWidth, problem := ahdSVGRootLength(root.attrs["width"])
	if problem != "" {
		return 0, 0, ahdSVGIdentity, problem
	}
	height, hasHeight, problem := ahdSVGRootLength(root.attrs["height"])
	if problem != "" {
		return 0, 0, ahdSVGIdentity, problem
	}
	switch {
	case hasWidth && hasHeight:
	case viewBox == nil:
		return 0, 0, ahdSVGIdentity, "SVG needs width and height or a viewBox to have a size"
	case hasWidth:
		height = width * viewBox[3] / viewBox[2]
	case hasHeight:
		width = height * viewBox[2] / viewBox[3]
	default:
		width, height = viewBox[2]*ahdSVGPixelInPoints, viewBox[3]*ahdSVGPixelInPoints
	}
	if width <= 0 || height <= 0 || width > ahdSVGMaximumSizePoint || height > ahdSVGMaximumSizePoint {
		return 0, 0, ahdSVGIdentity, "SVG size must be positive and at most " + ahdFormatReal(ahdSVGMaximumSizePoint) + " points"
	}
	if viewBox == nil {
		// Without a viewBox one user unit is one CSS pixel.
		return width, height, ahdSVGMatrix{ahdSVGPixelInPoints, 0, 0, ahdSVGPixelInPoints, 0, 0}, ""
	}
	scaleX, scaleY := width/viewBox[2], height/viewBox[3]
	align, slice := "xMidYMid", false
	if text := strings.Fields(root.attrs["preserveAspectRatio"]); len(text) > 0 {
		align = text[0]
		if len(text) > 1 {
			slice = text[1] == "slice"
			if text[1] != "slice" && text[1] != "meet" {
				return 0, 0, ahdSVGIdentity, "preserveAspectRatio " + strconv.Quote(root.attrs["preserveAspectRatio"]) + " is not valid"
			}
		}
	}
	translateX, translateY := -viewBox[0]*scaleX, -viewBox[1]*scaleY
	if align != "none" {
		if len(align) != 8 || !strings.HasPrefix(align, "x") || align[4] != 'Y' {
			return 0, 0, ahdSVGIdentity, "preserveAspectRatio " + strconv.Quote(root.attrs["preserveAspectRatio"]) + " is not valid"
		}
		uniform := math.Min(scaleX, scaleY)
		if slice {
			uniform = math.Max(scaleX, scaleY)
		}
		scaleX, scaleY = uniform, uniform
		offset := func(part string, available, content float64) (float64, bool) {
			switch part {
			case "Min":
				return 0, true
			case "Mid":
				return (available - content) / 2, true
			case "Max":
				return available - content, true
			}
			return 0, false
		}
		dx, okX := offset(align[1:4], width, viewBox[2]*uniform)
		dy, okY := offset(align[5:8], height, viewBox[3]*uniform)
		if !okX || !okY {
			return 0, 0, ahdSVGIdentity, "preserveAspectRatio " + strconv.Quote(root.attrs["preserveAspectRatio"]) + " is not valid"
		}
		translateX, translateY = dx-viewBox[0]*uniform, dy-viewBox[1]*uniform
	}
	return width, height, ahdSVGMatrix{scaleX, 0, 0, scaleY, translateX, translateY}, ""
}

// ahdSVGRootLength converts the root width or height to points. A missing
// value or a percentage means "use the viewBox".
func ahdSVGRootLength(text string) (float64, bool, string) {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasSuffix(text, "%") {
		return 0, false, ""
	}
	value, ok := ahdSVGLength(text)
	if !ok {
		return 0, false, "SVG length " + strconv.Quote(text) + " is not supported; use px, pt, pc, mm, cm, or in"
	}
	return value * ahdSVGPixelInPoints, true, ""
}

// ahdSVGLength parses a length in CSS pixels (user units).
func ahdSVGLength(text string) (float64, bool) {
	text = strings.TrimSpace(text)
	units := map[string]float64{"px": 1, "pt": 1 / ahdSVGPixelInPoints, "pc": 12 / ahdSVGPixelInPoints,
		"mm": 96 / 25.4, "cm": 96 / 2.54, "in": 96}
	factor := 1.0
	for suffix, value := range units {
		if strings.HasSuffix(text, suffix) {
			factor = value
			text = strings.TrimSuffix(text, suffix)
			break
		}
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}
	return number * factor, true
}

func ahdSVGNumbers(text string) ([]float64, bool) {
	var numbers []float64
	scanner := ahdSVGScanner{text: text}
	for {
		scanner.skip()
		if scanner.done() {
			return numbers, true
		}
		number, ok := scanner.number()
		if !ok {
			return nil, false
		}
		numbers = append(numbers, number)
	}
}

// render walks the tree, resolving style and transforms, and emits PGF.
func (converter *ahdSVGConverter) render(node *ahdSVGNode, inherited ahdSVGStyle, matrix ahdSVGMatrix, useDepth int, active map[*ahdSVGNode]bool) string {
	switch node.name {
	case "defs", "title", "desc", "metadata", "style":
		return ""
	}
	properties := converter.properties(node)
	if strings.TrimSpace(properties["display"]) == "none" {
		return ""
	}
	style, problem := ahdSVGResolveStyle(inherited, properties)
	if problem != "" {
		return "<" + node.name + ">: " + problem
	}
	if node.name != "svg" {
		if transform, present := properties["transform"]; present {
			parsed, ok := ahdSVGParseTransform(transform)
			if !ok {
				return "transform " + strconv.Quote(transform) + " is not valid"
			}
			matrix = matrix.multiply(parsed)
		}
	}
	switch node.name {
	case "svg", "g":
		for _, child := range node.children {
			if problem := converter.render(child, style, matrix, useDepth, active); problem != "" {
				return problem
			}
		}
		return ""
	case "use":
		reference := node.attrs["href"]
		if reference == "" {
			reference = node.attrs["xlink:href"]
		}
		target := converter.ids[strings.TrimPrefix(reference, "#")]
		if target == nil {
			return "<use> references " + strconv.Quote(reference) + ", which is not an element in this SVG"
		}
		if active[target] || useDepth >= ahdSVGMaximumUseDepth {
			return "<use> references form a cycle or are nested too deeply"
		}
		x, _ := ahdSVGLength(node.attrs["x"])
		y, _ := ahdSVGLength(node.attrs["y"])
		active[target] = true
		problem := converter.render(target, style, matrix.multiply(ahdSVGMatrix{1, 0, 0, 1, x, y}), useDepth+1, active)
		delete(active, target)
		return problem
	}
	segments, problem := ahdSVGShape(node)
	if problem != "" {
		return "<" + node.name + ">: " + problem
	}
	return converter.draw(node.name, segments, style, matrix)
}

// properties returns an element's presentation attributes overridden by
// matching style-sheet rules (by specificity, then order) and finally by its
// style attribute -- the CSS cascade in the narrow form document SVG uses.
func (converter *ahdSVGConverter) properties(node *ahdSVGNode) map[string]string {
	result := map[string]string{}
	for name, value := range node.attrs {
		if name != "style" {
			result[name] = value
		}
	}
	var matching []ahdSVGRule
	classes := strings.Fields(node.attrs["class"])
	for _, rule := range converter.rules {
		matched := rule.selector == "*" || rule.selector == node.name ||
			(strings.HasPrefix(rule.selector, "#") && rule.selector[1:] == node.attrs["id"])
		if strings.HasPrefix(rule.selector, ".") {
			for _, class := range classes {
				if class == rule.selector[1:] {
					matched = true
				}
			}
		}
		if matched {
			matching = append(matching, rule)
		}
	}
	sort.SliceStable(matching, func(i, j int) bool {
		if matching[i].specificity != matching[j].specificity {
			return matching[i].specificity < matching[j].specificity
		}
		return matching[i].order < matching[j].order
	})
	for _, rule := range matching {
		for name, value := range rule.declarations {
			result[name] = value
		}
	}
	if inline, present := node.attrs["style"]; present {
		declarations, _ := ahdSVGDeclarations(inline)
		for name, value := range declarations {
			result[name] = value
		}
	}
	return result
}

func ahdSVGResolveStyle(parent ahdSVGStyle, properties map[string]string) (ahdSVGStyle, string) {
	style := parent
	style.opacity = parent.opacity
	if value, present := properties["color"]; present {
		color, ok := ahdSVGParseColor(value, parent.color)
		if !ok {
			return style, "color " + strconv.Quote(value) + " is not supported"
		}
		style.color = color
	}
	paint := func(name string, target *ahdSVGPaint) string {
		value, present := properties[name]
		if !present {
			return ""
		}
		value = strings.TrimSpace(value)
		switch value {
		case "none", "transparent":
			*target = ahdSVGPaint{none: true}
		case "currentColor":
			*target = ahdSVGPaint{current: true}
		case "inherit":
		default:
			color, ok := ahdSVGParseColor(value, style.color)
			if !ok {
				return name + " " + strconv.Quote(value) + " is not a supported color"
			}
			*target = ahdSVGPaint{color: color}
		}
		return ""
	}
	if problem := paint("fill", &style.fill); problem != "" {
		return style, problem
	}
	if problem := paint("stroke", &style.stroke); problem != "" {
		return style, problem
	}
	fraction := func(name string, target *float64, multiply float64) string {
		value, present := properties[name]
		if !present {
			return ""
		}
		value = strings.TrimSpace(value)
		scale := 1.0
		if strings.HasSuffix(value, "%") {
			value, scale = strings.TrimSuffix(value, "%"), 0.01
		}
		number, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(number) {
			return name + " " + strconv.Quote(properties[name]) + " is not a number"
		}
		*target = math.Max(0, math.Min(1, number*scale)) * multiply
		return ""
	}
	for _, entry := range []struct {
		name     string
		target   *float64
		multiply float64
	}{
		{"fill-opacity", &style.fillOpacity, 1}, {"stroke-opacity", &style.strokeOpacity, 1}, {"opacity", &style.opacity, parent.opacity},
	} {
		if problem := fraction(entry.name, entry.target, entry.multiply); problem != "" {
			return style, problem
		}
	}
	if value, present := properties["stroke-width"]; present {
		width, ok := ahdSVGLength(value)
		if !ok || width < 0 {
			return style, "stroke-width " + strconv.Quote(value) + " is not supported"
		}
		style.strokeWidth = width
	}
	if value, present := properties["stroke-linecap"]; present {
		if value != "butt" && value != "round" && value != "square" {
			return style, "stroke-linecap " + strconv.Quote(value) + " is not supported"
		}
		style.lineCap = value
	}
	if value, present := properties["stroke-linejoin"]; present {
		if value != "miter" && value != "round" && value != "bevel" && value != "miter-clip" && value != "arcs" {
			return style, "stroke-linejoin " + strconv.Quote(value) + " is not supported"
		}
		if value == "miter-clip" || value == "arcs" {
			value = "miter"
		}
		style.lineJoin = value
	}
	if value, present := properties["stroke-miterlimit"]; present {
		limit, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || limit < 1 {
			return style, "stroke-miterlimit " + strconv.Quote(value) + " is not supported"
		}
		style.miterLimit = limit
	}
	if value, present := properties["stroke-dasharray"]; present {
		value = strings.TrimSpace(value)
		if value == "none" {
			style.dash = nil
		} else {
			var dash []float64
			for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' }) {
				length, ok := ahdSVGLength(part)
				if !ok || length < 0 {
					return style, "stroke-dasharray " + strconv.Quote(value) + " is not supported"
				}
				dash = append(dash, length)
			}
			if len(dash)%2 == 1 {
				dash = append(dash, dash...)
			}
			total := 0.0
			for _, length := range dash {
				total += length
			}
			if total == 0 {
				dash = nil
			}
			style.dash = dash
		}
	}
	if value, present := properties["stroke-dashoffset"]; present {
		offset, ok := ahdSVGLength(value)
		if !ok {
			return style, "stroke-dashoffset " + strconv.Quote(value) + " is not supported"
		}
		style.dashOffset = offset
	}
	if value, present := properties["fill-rule"]; present {
		if value != "nonzero" && value != "evenodd" {
			return style, "fill-rule " + strconv.Quote(value) + " is not supported"
		}
		style.evenOdd = value == "evenodd"
	}
	if value, present := properties["visibility"]; present {
		style.hidden = value == "hidden" || value == "collapse"
	}
	return style, ""
}

// ahdSVGShape converts one drawable element to path segments in user space.
func ahdSVGShape(node *ahdSVGNode) ([]ahdSVGSegment, string) {
	number := func(name string) (float64, string) {
		text, present := node.attrs[name]
		if !present || strings.TrimSpace(text) == "" {
			return 0, ""
		}
		value, ok := ahdSVGLength(text)
		if !ok {
			return 0, name + " " + strconv.Quote(text) + " is not a supported length"
		}
		return value, ""
	}
	values := func(names ...string) ([]float64, string) {
		result := make([]float64, len(names))
		for index, name := range names {
			value, problem := number(name)
			if problem != "" {
				return nil, problem
			}
			result[index] = value
		}
		return result, ""
	}
	switch node.name {
	case "path":
		return ahdSVGParsePath(node.attrs["d"])
	case "rect":
		v, problem := values("x", "y", "width", "height", "rx", "ry")
		if problem != "" {
			return nil, problem
		}
		x, y, width, height := v[0], v[1], v[2], v[3]
		if width < 0 || height < 0 {
			return nil, "width and height must not be negative"
		}
		if width == 0 || height == 0 {
			return nil, ""
		}
		_, hasRX := node.attrs["rx"]
		_, hasRY := node.attrs["ry"]
		rx, ry := v[4], v[5]
		if hasRX && !hasRY {
			ry = rx
		} else if hasRY && !hasRX {
			rx = ry
		}
		rx, ry = math.Min(math.Abs(rx), width/2), math.Min(math.Abs(ry), height/2)
		if rx == 0 || ry == 0 {
			return []ahdSVGSegment{
				{op: 'M', points: [3][2]float64{{x, y}}}, {op: 'L', points: [3][2]float64{{x + width, y}}},
				{op: 'L', points: [3][2]float64{{x + width, y + height}}}, {op: 'L', points: [3][2]float64{{x, y + height}}},
				{op: 'Z'},
			}, ""
		}
		k := 0.5522847498307936
		return []ahdSVGSegment{
			{op: 'M', points: [3][2]float64{{x + rx, y}}},
			{op: 'L', points: [3][2]float64{{x + width - rx, y}}},
			{op: 'C', points: [3][2]float64{{x + width - rx + k*rx, y}, {x + width, y + ry - k*ry}, {x + width, y + ry}}},
			{op: 'L', points: [3][2]float64{{x + width, y + height - ry}}},
			{op: 'C', points: [3][2]float64{{x + width, y + height - ry + k*ry}, {x + width - rx + k*rx, y + height}, {x + width - rx, y + height}}},
			{op: 'L', points: [3][2]float64{{x + rx, y + height}}},
			{op: 'C', points: [3][2]float64{{x + rx - k*rx, y + height}, {x, y + height - ry + k*ry}, {x, y + height - ry}}},
			{op: 'L', points: [3][2]float64{{x, y + ry}}},
			{op: 'C', points: [3][2]float64{{x, y + ry - k*ry}, {x + rx - k*rx, y}, {x + rx, y}}},
			{op: 'Z'},
		}, ""
	case "circle", "ellipse":
		var cx, cy, rx, ry float64
		if node.name == "circle" {
			v, problem := values("cx", "cy", "r")
			if problem != "" {
				return nil, problem
			}
			cx, cy, rx, ry = v[0], v[1], v[2], v[2]
		} else {
			v, problem := values("cx", "cy", "rx", "ry")
			if problem != "" {
				return nil, problem
			}
			cx, cy, rx, ry = v[0], v[1], v[2], v[3]
		}
		if rx < 0 || ry < 0 {
			return nil, "radius must not be negative"
		}
		if rx == 0 || ry == 0 {
			return nil, ""
		}
		k := 0.5522847498307936
		return []ahdSVGSegment{
			{op: 'M', points: [3][2]float64{{cx + rx, cy}}},
			{op: 'C', points: [3][2]float64{{cx + rx, cy + k*ry}, {cx + k*rx, cy + ry}, {cx, cy + ry}}},
			{op: 'C', points: [3][2]float64{{cx - k*rx, cy + ry}, {cx - rx, cy + k*ry}, {cx - rx, cy}}},
			{op: 'C', points: [3][2]float64{{cx - rx, cy - k*ry}, {cx - k*rx, cy - ry}, {cx, cy - ry}}},
			{op: 'C', points: [3][2]float64{{cx + k*rx, cy - ry}, {cx + rx, cy - k*ry}, {cx + rx, cy}}},
			{op: 'Z'},
		}, ""
	case "line":
		v, problem := values("x1", "y1", "x2", "y2")
		if problem != "" {
			return nil, problem
		}
		return []ahdSVGSegment{{op: 'M', points: [3][2]float64{{v[0], v[1]}}}, {op: 'L', points: [3][2]float64{{v[2], v[3]}}}}, ""
	case "polyline", "polygon":
		numbers, ok := ahdSVGNumbers(node.attrs["points"])
		if !ok {
			return nil, "points is not a list of numbers"
		}
		if len(numbers)%2 == 1 {
			numbers = numbers[:len(numbers)-1]
		}
		var segments []ahdSVGSegment
		for index := 0; index+1 < len(numbers); index += 2 {
			op := byte('L')
			if index == 0 {
				op = 'M'
			}
			segments = append(segments, ahdSVGSegment{op: op, points: [3][2]float64{{numbers[index], numbers[index+1]}}})
		}
		if node.name == "polygon" && len(segments) > 0 {
			segments = append(segments, ahdSVGSegment{op: 'Z'})
		}
		return segments, ""
	}
	return nil, "is not drawable"
}

// draw emits one element's path with its paint. Coordinates are transformed
// by matrix and flipped so PGF's y axis points up.
func (converter *ahdSVGConverter) draw(name string, segments []ahdSVGSegment, style ahdSVGStyle, matrix ahdSVGMatrix) string {
	if style.hidden || len(segments) == 0 {
		return ""
	}
	fill, stroke := style.fill, style.stroke
	// A line has no interior; SVG never fills it.
	if name == "line" {
		fill = ahdSVGPaint{none: true}
	}
	fillAlpha := style.fillOpacity * style.opacity
	strokeAlpha := style.strokeOpacity * style.opacity
	strokeWidth := style.strokeWidth * matrix.scale()
	doFill := !fill.none && fillAlpha > 0
	doStroke := !stroke.none && strokeAlpha > 0 && strokeWidth > 0
	if !doFill && !doStroke {
		return ""
	}
	converter.segments += len(segments)
	if converter.segments > ahdSVGMaximumSegments {
		return "SVG has more than 2000000 path segments"
	}
	var out strings.Builder
	out.WriteString("\\begin{pgfscope}%\n")
	resolve := func(paint ahdSVGPaint) string {
		color := paint.color
		if paint.current {
			color = style.color
		}
		if existing, known := converter.colors[color]; known {
			return existing
		}
		identifier := "ahdsvg" + strconv.Itoa(len(converter.colorList))
		converter.colors[color] = identifier
		converter.colorList = append(converter.colorList, color)
		return identifier
	}
	if doFill {
		out.WriteString("\\pgfsetfillcolor{" + resolve(fill) + "}%\n")
		if fillAlpha < 1 {
			out.WriteString("\\pgfsetfillopacity{" + ahdSVGNumber(fillAlpha) + "}%\n")
		}
		if style.evenOdd {
			out.WriteString("\\pgfseteorule%\n")
		}
	}
	if doStroke {
		out.WriteString("\\pgfsetstrokecolor{" + resolve(stroke) + "}%\n")
		if strokeAlpha < 1 {
			out.WriteString("\\pgfsetstrokeopacity{" + ahdSVGNumber(strokeAlpha) + "}%\n")
		}
		out.WriteString("\\pgfsetlinewidth{" + ahdSVGNumber(strokeWidth) + "bp}%\n")
		switch style.lineCap {
		case "round":
			out.WriteString("\\pgfsetroundcap%\n")
		case "square":
			out.WriteString("\\pgfsetrectcap%\n")
		}
		switch style.lineJoin {
		case "round":
			out.WriteString("\\pgfsetroundjoin%\n")
		case "bevel":
			out.WriteString("\\pgfsetbeveljoin%\n")
		}
		if style.miterLimit != 4 {
			out.WriteString("\\pgfsetmiterlimit{" + ahdSVGNumber(style.miterLimit) + "}%\n")
		}
		if len(style.dash) > 0 {
			out.WriteString("\\pgfsetdash{")
			scale := matrix.scale()
			for _, length := range style.dash {
				out.WriteString("{" + ahdSVGNumber(length*scale) + "bp}")
			}
			out.WriteString("}{" + ahdSVGNumber(style.dashOffset*scale) + "bp}%\n")
		}
	}
	for _, segment := range segments {
		switch segment.op {
		case 'Z':
			out.WriteString("\\pgfpathclose%\n")
			continue
		}
		points := make([]string, 0, 3)
		count := 1
		if segment.op == 'C' {
			count = 3
		}
		for index := 0; index < count; index++ {
			x, y := matrix.apply(segment.points[index][0], segment.points[index][1])
			y = converter.height - y
			if math.IsNaN(x) || math.IsNaN(y) || math.Abs(x) > ahdSVGCoordinateLimit || math.Abs(y) > ahdSVGCoordinateLimit {
				return "a coordinate of <" + name + "> lies too far outside the drawing"
			}
			points = append(points, "{\\pgfqpoint{"+ahdSVGNumber(x)+"bp}{"+ahdSVGNumber(y)+"bp}}")
		}
		switch segment.op {
		case 'M':
			out.WriteString("\\pgfpathmoveto" + points[0] + "%\n")
		case 'L':
			out.WriteString("\\pgfpathlineto" + points[0] + "%\n")
		case 'C':
			out.WriteString("\\pgfpathcurveto" + points[0] + points[1] + points[2] + "%\n")
		}
	}
	switch {
	case doFill && doStroke:
		out.WriteString("\\pgfusepath{fill,stroke}%\n")
	case doFill:
		out.WriteString("\\pgfusepath{fill}%\n")
	default:
		out.WriteString("\\pgfusepath{stroke}%\n")
	}
	out.WriteString("\\end{pgfscope}%\n")
	converter.out.WriteString(out.String())
	return ""
}

func ahdSVGNumber(value float64) string {
	rounded := math.Round(value*10000) / 10000
	if rounded == 0 {
		return "0"
	}
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

// ---------------------------------------------------------------------------
// Transforms, colors, and path data
// ---------------------------------------------------------------------------

func ahdSVGParseTransform(text string) (ahdSVGMatrix, bool) {
	result := ahdSVGIdentity
	scanner := ahdSVGScanner{text: text}
	for {
		scanner.skip()
		if scanner.done() {
			return result, true
		}
		start := scanner.position
		for !scanner.done() && (scanner.peek() >= 'a' && scanner.peek() <= 'z' || scanner.peek() >= 'A' && scanner.peek() <= 'Z') {
			scanner.position++
		}
		name := text[start:scanner.position]
		scanner.skipSpace()
		if scanner.done() || scanner.peek() != '(' {
			return result, false
		}
		scanner.position++
		var arguments []float64
		for {
			scanner.skip()
			if scanner.done() {
				return result, false
			}
			if scanner.peek() == ')' {
				scanner.position++
				break
			}
			number, ok := scanner.number()
			if !ok {
				return result, false
			}
			arguments = append(arguments, number)
		}
		var step ahdSVGMatrix
		switch {
		case name == "matrix" && len(arguments) == 6:
			step = ahdSVGMatrix{arguments[0], arguments[1], arguments[2], arguments[3], arguments[4], arguments[5]}
		case name == "translate" && (len(arguments) == 1 || len(arguments) == 2):
			ty := 0.0
			if len(arguments) == 2 {
				ty = arguments[1]
			}
			step = ahdSVGMatrix{1, 0, 0, 1, arguments[0], ty}
		case name == "scale" && (len(arguments) == 1 || len(arguments) == 2):
			sy := arguments[0]
			if len(arguments) == 2 {
				sy = arguments[1]
			}
			step = ahdSVGMatrix{arguments[0], 0, 0, sy, 0, 0}
		case name == "rotate" && (len(arguments) == 1 || len(arguments) == 3):
			angle := arguments[0] * math.Pi / 180
			cos, sin := math.Cos(angle), math.Sin(angle)
			step = ahdSVGMatrix{cos, sin, -sin, cos, 0, 0}
			if len(arguments) == 3 {
				cx, cy := arguments[1], arguments[2]
				step = ahdSVGMatrix{1, 0, 0, 1, cx, cy}.multiply(step).multiply(ahdSVGMatrix{1, 0, 0, 1, -cx, -cy})
			}
		case name == "skewX" && len(arguments) == 1:
			step = ahdSVGMatrix{1, 0, math.Tan(arguments[0] * math.Pi / 180), 1, 0, 0}
		case name == "skewY" && len(arguments) == 1:
			step = ahdSVGMatrix{1, math.Tan(arguments[0] * math.Pi / 180), 0, 1, 0, 0}
		default:
			return result, false
		}
		result = result.multiply(step)
	}
}

func ahdSVGParseColor(text string, current ahdSVGColor) (ahdSVGColor, bool) {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "currentcolor" {
		return current, true
	}
	if strings.HasPrefix(text, "#") {
		hex := text[1:]
		if len(hex) == 3 || len(hex) == 4 {
			expanded := ""
			for _, digit := range hex[:3] {
				expanded += string(digit) + string(digit)
			}
			hex = expanded
		} else if len(hex) == 8 {
			hex = hex[:6]
		}
		if len(hex) != 6 {
			return ahdSVGColor{}, false
		}
		value, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return ahdSVGColor{}, false
		}
		return ahdSVGColor{uint8(value >> 16), uint8(value >> 8), uint8(value)}, true
	}
	if strings.HasPrefix(text, "rgb(") || strings.HasPrefix(text, "rgba(") {
		open, closing := strings.Index(text, "("), strings.LastIndex(text, ")")
		if closing < open {
			return ahdSVGColor{}, false
		}
		parts := strings.FieldsFunc(text[open+1:closing], func(r rune) bool { return r == ',' || r == ' ' || r == '/' })
		if len(parts) < 3 {
			return ahdSVGColor{}, false
		}
		var channels [3]uint8
		for index := 0; index < 3; index++ {
			part := parts[index]
			scale := 1.0
			if strings.HasSuffix(part, "%") {
				part, scale = strings.TrimSuffix(part, "%"), 2.55
			}
			value, err := strconv.ParseFloat(part, 64)
			if err != nil {
				return ahdSVGColor{}, false
			}
			channels[index] = uint8(math.Round(math.Max(0, math.Min(255, value*scale))))
		}
		return ahdSVGColor{channels[0], channels[1], channels[2]}, true
	}
	if value, known := ahdSVGNamedColors[text]; known {
		return ahdSVGColor{uint8(value >> 16), uint8(value >> 8), uint8(value)}, true
	}
	return ahdSVGColor{}, false
}

// ahdSVGScanner reads SVG number lists and path data.
type ahdSVGScanner struct {
	text     string
	position int
}

func (scanner *ahdSVGScanner) done() bool { return scanner.position >= len(scanner.text) }
func (scanner *ahdSVGScanner) peek() byte { return scanner.text[scanner.position] }

func (scanner *ahdSVGScanner) skipSpace() {
	for !scanner.done() && strings.IndexByte(" \t\r\n", scanner.peek()) >= 0 {
		scanner.position++
	}
}

// skip passes whitespace and at most one comma.
func (scanner *ahdSVGScanner) skip() {
	scanner.skipSpace()
	if !scanner.done() && scanner.peek() == ',' {
		scanner.position++
		scanner.skipSpace()
	}
}

func (scanner *ahdSVGScanner) number() (float64, bool) {
	start := scanner.position
	if !scanner.done() && (scanner.peek() == '+' || scanner.peek() == '-') {
		scanner.position++
	}
	digits := false
	for !scanner.done() && scanner.peek() >= '0' && scanner.peek() <= '9' {
		scanner.position++
		digits = true
	}
	if !scanner.done() && scanner.peek() == '.' {
		scanner.position++
		for !scanner.done() && scanner.peek() >= '0' && scanner.peek() <= '9' {
			scanner.position++
			digits = true
		}
	}
	if !digits {
		scanner.position = start
		return 0, false
	}
	if !scanner.done() && (scanner.peek() == 'e' || scanner.peek() == 'E') {
		mark := scanner.position
		scanner.position++
		if !scanner.done() && (scanner.peek() == '+' || scanner.peek() == '-') {
			scanner.position++
		}
		exponent := false
		for !scanner.done() && scanner.peek() >= '0' && scanner.peek() <= '9' {
			scanner.position++
			exponent = true
		}
		if !exponent {
			scanner.position = mark
		}
	}
	value, err := strconv.ParseFloat(scanner.text[start:scanner.position], 64)
	if err != nil || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}

func (scanner *ahdSVGScanner) flag() (bool, bool) {
	scanner.skip()
	if scanner.done() || (scanner.peek() != '0' && scanner.peek() != '1') {
		return false, false
	}
	value := scanner.peek() == '1'
	scanner.position++
	return value, true
}

// ahdSVGParsePath converts path data to absolute moves, lines, cubic curves,
// and closes. Quadratic curves and elliptical arcs become cubic curves.
func ahdSVGParsePath(data string) ([]ahdSVGSegment, string) {
	scanner := ahdSVGScanner{text: data}
	var segments []ahdSVGSegment
	var command byte
	var x, y, startX, startY, controlX, controlY float64
	var previous byte
	numbers := func(count int) ([]float64, bool) {
		result := make([]float64, count)
		for index := range result {
			scanner.skip()
			value, ok := scanner.number()
			if !ok {
				return nil, false
			}
			result[index] = value
		}
		return result, true
	}
	invalid := func() ([]ahdSVGSegment, string) {
		shown := data
		if len(shown) > 40 {
			shown = shown[:40] + "..."
		}
		return nil, "path data " + strconv.Quote(shown) + " is not valid"
	}
	for {
		scanner.skip()
		if scanner.done() {
			break
		}
		character := scanner.peek()
		if strings.IndexByte("MmLlHhVvCcSsQqTtAaZz", character) >= 0 {
			command = character
			scanner.position++
		} else if command == 0 {
			return invalid()
		} else if command == 'M' {
			command = 'L'
		} else if command == 'm' {
			command = 'l'
		}
		relative := command >= 'a' && command <= 'z'
		offsetX, offsetY := 0.0, 0.0
		if relative {
			offsetX, offsetY = x, y
		}
		switch command | 0x20 {
		case 'z':
			segments = append(segments, ahdSVGSegment{op: 'Z'})
			x, y = startX, startY
			previous = 'z'
			if !scanner.done() {
				scanner.skip()
				if !scanner.done() && strings.IndexByte("MmLlHhVvCcSsQqTtAaZz", scanner.peek()) < 0 {
					return invalid()
				}
			}
			continue
		case 'm', 'l':
			values, ok := numbers(2)
			if !ok {
				return invalid()
			}
			x, y = values[0]+offsetX, values[1]+offsetY
			if command|0x20 == 'm' {
				startX, startY = x, y
				segments = append(segments, ahdSVGSegment{op: 'M', points: [3][2]float64{{x, y}}})
			} else {
				segments = append(segments, ahdSVGSegment{op: 'L', points: [3][2]float64{{x, y}}})
			}
		case 'h':
			values, ok := numbers(1)
			if !ok {
				return invalid()
			}
			x = values[0] + offsetX
			segments = append(segments, ahdSVGSegment{op: 'L', points: [3][2]float64{{x, y}}})
		case 'v':
			values, ok := numbers(1)
			if !ok {
				return invalid()
			}
			y = values[0] + offsetY
			segments = append(segments, ahdSVGSegment{op: 'L', points: [3][2]float64{{x, y}}})
		case 'c':
			values, ok := numbers(6)
			if !ok {
				return invalid()
			}
			controlX, controlY = values[2]+offsetX, values[3]+offsetY
			segments = append(segments, ahdSVGSegment{op: 'C', points: [3][2]float64{
				{values[0] + offsetX, values[1] + offsetY}, {controlX, controlY}, {values[4] + offsetX, values[5] + offsetY}}})
			x, y = values[4]+offsetX, values[5]+offsetY
		case 's':
			values, ok := numbers(4)
			if !ok {
				return invalid()
			}
			firstX, firstY := x, y
			if previous == 'c' || previous == 's' {
				firstX, firstY = 2*x-controlX, 2*y-controlY
			}
			controlX, controlY = values[0]+offsetX, values[1]+offsetY
			segments = append(segments, ahdSVGSegment{op: 'C', points: [3][2]float64{{firstX, firstY}, {controlX, controlY}, {values[2] + offsetX, values[3] + offsetY}}})
			x, y = values[2]+offsetX, values[3]+offsetY
		case 'q', 't':
			var qx, qy, endX, endY float64
			if command|0x20 == 'q' {
				values, ok := numbers(4)
				if !ok {
					return invalid()
				}
				qx, qy, endX, endY = values[0]+offsetX, values[1]+offsetY, values[2]+offsetX, values[3]+offsetY
			} else {
				values, ok := numbers(2)
				if !ok {
					return invalid()
				}
				qx, qy = x, y
				if previous == 'q' || previous == 't' {
					qx, qy = 2*x-controlX, 2*y-controlY
				}
				endX, endY = values[0]+offsetX, values[1]+offsetY
			}
			segments = append(segments, ahdSVGSegment{op: 'C', points: [3][2]float64{
				{x + 2.0/3.0*(qx-x), y + 2.0/3.0*(qy-y)}, {endX + 2.0/3.0*(qx-endX), endY + 2.0/3.0*(qy-endY)}, {endX, endY}}})
			controlX, controlY = qx, qy
			x, y = endX, endY
		case 'a':
			radii, ok := numbers(3)
			if !ok {
				return invalid()
			}
			large, okLarge := scanner.flag()
			sweep, okSweep := scanner.flag()
			end, okEnd := numbers(2)
			if !okLarge || !okSweep || !okEnd {
				return invalid()
			}
			endX, endY := end[0]+offsetX, end[1]+offsetY
			segments = append(segments, ahdSVGArc(x, y, radii[0], radii[1], radii[2], large, sweep, endX, endY)...)
			x, y = endX, endY
		}
		previous = command | 0x20
		if len(segments) > ahdSVGMaximumSegments {
			return nil, "path has more than 2000000 segments"
		}
	}
	if len(segments) > 0 && segments[0].op != 'M' {
		return invalid()
	}
	return segments, ""
}

// ahdSVGArc converts an SVG elliptical arc (endpoint parameterization, SVG 1.1
// implementation notes F.6) to cubic Bezier segments of at most 90 degrees.
func ahdSVGArc(x1, y1, rx, ry, rotation float64, large, sweep bool, x2, y2 float64) []ahdSVGSegment {
	if x1 == x2 && y1 == y2 {
		return nil
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		return []ahdSVGSegment{{op: 'L', points: [3][2]float64{{x2, y2}}}}
	}
	phi := rotation * math.Pi / 180
	cos, sin := math.Cos(phi), math.Sin(phi)
	dx, dy := (x1-x2)/2, (y1-y2)/2
	x1p, y1p := cos*dx+sin*dy, -sin*dx+cos*dy
	lambda := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry)
	if lambda > 1 {
		scale := math.Sqrt(lambda)
		rx, ry = rx*scale, ry*scale
	}
	numerator := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	denominator := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	factor := 0.0
	if denominator != 0 {
		factor = math.Sqrt(math.Max(0, numerator/denominator))
	}
	if large == sweep {
		factor = -factor
	}
	cxp, cyp := factor*rx*y1p/ry, -factor*ry*x1p/rx
	cx, cy := cos*cxp-sin*cyp+(x1+x2)/2, sin*cxp+cos*cyp+(y1+y2)/2
	angle := func(ux, uy, vx, vy float64) float64 {
		return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
	}
	theta := angle(1, 0, (x1p-cxp)/rx, (y1p-cyp)/ry)
	delta := angle((x1p-cxp)/rx, (y1p-cyp)/ry, (-x1p-cxp)/rx, (-y1p-cyp)/ry)
	if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	} else if sweep && delta < 0 {
		delta += 2 * math.Pi
	}
	count := int(math.Ceil(math.Abs(delta) / (math.Pi / 2)))
	if count < 1 {
		count = 1
	}
	step := delta / float64(count)
	k := 4.0 / 3.0 * math.Tan(step/4)
	point := func(t float64) (float64, float64) {
		return cx + rx*math.Cos(t)*cos - ry*math.Sin(t)*sin, cy + rx*math.Cos(t)*sin + ry*math.Sin(t)*cos
	}
	derivative := func(t float64) (float64, float64) {
		return -rx*math.Sin(t)*cos - ry*math.Cos(t)*sin, -rx*math.Sin(t)*sin + ry*math.Cos(t)*cos
	}
	segments := make([]ahdSVGSegment, 0, count)
	for index := 0; index < count; index++ {
		t1 := theta + float64(index)*step
		t2 := t1 + step
		px1, py1 := point(t1)
		px2, py2 := point(t2)
		dx1, dy1 := derivative(t1)
		dx2, dy2 := derivative(t2)
		if index == count-1 {
			px2, py2 = x2, y2
		}
		segments = append(segments, ahdSVGSegment{op: 'C', points: [3][2]float64{
			{px1 + k*dx1, py1 + k*dy1}, {px2 - k*dx2, py2 - k*dy2}, {px2, py2}}})
	}
	return segments
}

// ahdSVGNamedColors is the CSS Color Module Level 4 named-color table.
var ahdSVGNamedColors = map[string]uint32{
	"aliceblue": 0xF0F8FF, "antiquewhite": 0xFAEBD7, "aqua": 0x00FFFF, "aquamarine": 0x7FFFD4, "azure": 0xF0FFFF,
	"beige": 0xF5F5DC, "bisque": 0xFFE4C4, "black": 0x000000, "blanchedalmond": 0xFFEBCD, "blue": 0x0000FF,
	"blueviolet": 0x8A2BE2, "brown": 0xA52A2A, "burlywood": 0xDEB887, "cadetblue": 0x5F9EA0, "chartreuse": 0x7FFF00,
	"chocolate": 0xD2691E, "coral": 0xFF7F50, "cornflowerblue": 0x6495ED, "cornsilk": 0xFFF8DC, "crimson": 0xDC143C,
	"cyan": 0x00FFFF, "darkblue": 0x00008B, "darkcyan": 0x008B8B, "darkgoldenrod": 0xB8860B, "darkgray": 0xA9A9A9,
	"darkgreen": 0x006400, "darkgrey": 0xA9A9A9, "darkkhaki": 0xBDB76B, "darkmagenta": 0x8B008B, "darkolivegreen": 0x556B2F,
	"darkorange": 0xFF8C00, "darkorchid": 0x9932CC, "darkred": 0x8B0000, "darksalmon": 0xE9967A, "darkseagreen": 0x8FBC8F,
	"darkslateblue": 0x483D8B, "darkslategray": 0x2F4F4F, "darkslategrey": 0x2F4F4F, "darkturquoise": 0x00CED1,
	"darkviolet": 0x9400D3, "deeppink": 0xFF1493, "deepskyblue": 0x00BFFF, "dimgray": 0x696969, "dimgrey": 0x696969,
	"dodgerblue": 0x1E90FF, "firebrick": 0xB22222, "floralwhite": 0xFFFAF0, "forestgreen": 0x228B22, "fuchsia": 0xFF00FF,
	"gainsboro": 0xDCDCDC, "ghostwhite": 0xF8F8FF, "gold": 0xFFD700, "goldenrod": 0xDAA520, "gray": 0x808080,
	"green": 0x008000, "greenyellow": 0xADFF2F, "grey": 0x808080, "honeydew": 0xF0FFF0, "hotpink": 0xFF69B4,
	"indianred": 0xCD5C5C, "indigo": 0x4B0082, "ivory": 0xFFFFF0, "khaki": 0xF0E68C, "lavender": 0xE6E6FA,
	"lavenderblush": 0xFFF0F5, "lawngreen": 0x7CFC00, "lemonchiffon": 0xFFFACD, "lightblue": 0xADD8E6, "lightcoral": 0xF08080,
	"lightcyan": 0xE0FFFF, "lightgoldenrodyellow": 0xFAFAD2, "lightgray": 0xD3D3D3, "lightgreen": 0x90EE90, "lightgrey": 0xD3D3D3,
	"lightpink": 0xFFB6C1, "lightsalmon": 0xFFA07A, "lightseagreen": 0x20B2AA, "lightskyblue": 0x87CEFA,
	"lightslategray": 0x778899, "lightslategrey": 0x778899, "lightsteelblue": 0xB0C4DE, "lightyellow": 0xFFFFE0,
	"lime": 0x00FF00, "limegreen": 0x32CD32, "linen": 0xFAF0E6, "magenta": 0xFF00FF, "maroon": 0x800000,
	"mediumaquamarine": 0x66CDAA, "mediumblue": 0x0000CD, "mediumorchid": 0xBA55D3, "mediumpurple": 0x9370DB,
	"mediumseagreen": 0x3CB371, "mediumslateblue": 0x7B68EE, "mediumspringgreen": 0x00FA9A, "mediumturquoise": 0x48D1CC,
	"mediumvioletred": 0xC71585, "midnightblue": 0x191970, "mintcream": 0xF5FFFA, "mistyrose": 0xFFE4E1, "moccasin": 0xFFE4B5,
	"navajowhite": 0xFFDEAD, "navy": 0x000080, "oldlace": 0xFDF5E6, "olive": 0x808000, "olivedrab": 0x6B8E23,
	"orange": 0xFFA500, "orangered": 0xFF4500, "orchid": 0xDA70D6, "palegoldenrod": 0xEEE8AA, "palegreen": 0x98FB98,
	"paleturquoise": 0xAFEEEE, "palevioletred": 0xDB7093, "papayawhip": 0xFFEFD5, "peachpuff": 0xFFDAB9, "peru": 0xCD853F,
	"pink": 0xFFC0CB, "plum": 0xDDA0DD, "powderblue": 0xB0E0E6, "purple": 0x800080, "rebeccapurple": 0x663399,
	"red": 0xFF0000, "rosybrown": 0xBC8F8F, "royalblue": 0x4169E1, "saddlebrown": 0x8B4513, "salmon": 0xFA8072,
	"sandybrown": 0xF4A460, "seagreen": 0x2E8B57, "seashell": 0xFFF5EE, "sienna": 0xA0522D, "silver": 0xC0C0C0,
	"skyblue": 0x87CEEB, "slateblue": 0x6A5ACD, "slategray": 0x708090, "slategrey": 0x708090, "snow": 0xFFFAFA,
	"springgreen": 0x00FF7F, "steelblue": 0x4682B4, "tan": 0xD2B48C, "teal": 0x008080, "thistle": 0xD8BFD8,
	"tomato": 0xFF6347, "turquoise": 0x40E0D0, "violet": 0xEE82EE, "wheat": 0xF5DEB3, "white": 0xFFFFFF,
	"whitesmoke": 0xF5F5F5, "yellow": 0xFFFF00, "yellowgreen": 0x9ACD32,
}
