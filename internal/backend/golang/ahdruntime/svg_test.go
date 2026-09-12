package ahdruntime

import (
	"strings"
	"testing"
)

func svgConvert(t *testing.T, svg string) AhdSVGPicture {
	t.Helper()
	picture, problem := AhdSVGConvert("test", []byte(svg))
	if problem != "" {
		t.Fatalf("conversion failed: %s\n%s", problem, svg)
	}
	return picture
}

func TestSVGSizeComesFromWidthHeightOrViewBox(t *testing.T) {
	cases := []struct {
		svg           string
		width, height float64
	}{
		{`<svg xmlns="http://www.w3.org/2000/svg" width="96" height="48"><rect width="10" height="10"/></svg>`, 72, 36},
		{`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 100"><rect width="10" height="10"/></svg>`, 150, 75},
		{`<svg xmlns="http://www.w3.org/2000/svg" width="2cm" viewBox="0 0 200 100"><rect width="10" height="10"/></svg>`, 72 / 2.54 * 2, 72 / 2.54},
		{`<svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" viewBox="0 0 40 40"><rect width="10" height="10"/></svg>`, 30, 30},
		{`<svg xmlns="http://www.w3.org/2000/svg" width="1in" height="36pt"><rect width="10" height="10"/></svg>`, 72, 36},
	}
	for _, test := range cases {
		picture := svgConvert(t, test.svg)
		if abs(picture.Width-test.width) > 1e-9 || abs(picture.Height-test.height) > 1e-9 {
			t.Fatalf("size %.4fx%.4f, want %.4fx%.4f for %s", picture.Width, picture.Height, test.width, test.height, test.svg)
		}
		if !strings.Contains(picture.Source, "\\pgfusepath{use as bounding box,clip}") {
			t.Fatal("picture does not fix its bounding box")
		}
	}
	if _, problem := AhdSVGConvert("test", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`)); problem != "test: SVG needs width and height or a viewBox to have a size" {
		t.Fatalf("sizeless SVG problem %q", problem)
	}
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func TestSVGShapesBecomeVectorPaths(t *testing.T) {
	picture := svgConvert(t, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="100" height="100">
  <title>logo</title>
  <rect x="10" y="10" width="30" height="20" rx="4" fill="#1F4E79"/>
  <circle cx="70" cy="30" r="15" fill="rgb(176, 141, 87)" stroke="black" stroke-width="2"/>
  <ellipse cx="50" cy="70" rx="20" ry="10" fill="none" stroke="red" stroke-dasharray="4 2" stroke-linecap="round"/>
  <line x1="0" y1="100" x2="100" y2="0" stroke="navy"/>
  <polyline points="10,90 20,80 30,90" fill="none" stroke="green"/>
  <polygon points="80,80 95,95 65,95" fill="gold" fill-rule="evenodd" opacity="0.5"/>
  <path d="M10 50 h10 v10 H10 z m30 0 q5 -10 10 0 t10 0 c2 3 4 3 6 0 s4 -3 6 0 a5 5 0 1 0 10 0 l1e1 0"/>
</svg>`)
	for _, want := range []string{
		"\\definecolor{ahdsvg0}{RGB}{31,78,121}", "\\pgfpathcurveto", "\\pgfpathclose", "\\pgfusepath{fill,stroke}",
		"\\pgfsetdash{{3bp}{1.5bp}}{0bp}", "\\pgfsetroundcap", "\\pgfseteorule", "\\pgfsetfillopacity{0.5}", "\\pgfusepath{stroke}",
	} {
		if !strings.Contains(picture.Source, want) {
			t.Fatalf("converted SVG lacks %q:\n%s", want, picture.Source)
		}
	}
	for _, forbidden := range []string{"includegraphics", "\\input", "\\write", "\\openin", "\\special"} {
		if strings.Contains(picture.Source, forbidden) {
			t.Fatalf("converted SVG contains %q", forbidden)
		}
	}
	// SVG y points down; PGF y points up. The rectangle's top edge at y=10
	// in a 100 px (75 pt) drawing lands at 75 - 7.5 = 67.5 pt.
	if !strings.Contains(picture.Source, "{\\pgfqpoint{10.5bp}{67.5bp}}") {
		t.Fatalf("rounded rectangle start point not flipped as expected:\n%s", picture.Source[:600])
	}
	again := svgConvert(t, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="100" height="100"><rect x="10" y="10" width="30" height="20" rx="4" fill="#1F4E79"/></svg>`)
	third := svgConvert(t, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="100" height="100"><rect x="10" y="10" width="30" height="20" rx="4" fill="#1F4E79"/></svg>`)
	if again.Source != third.Source {
		t.Fatal("SVG conversion is not deterministic")
	}
}

func TestSVGStyleTransformsAndUse(t *testing.T) {
	picture := svgConvert(t, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="40" height="40">
  <style><![CDATA[ /* brand */ .brand { fill: #B08D57 } rect { stroke: blue } #accent { fill: purple } ]]></style>
  <defs><path id="dot" d="M0 0 h4 v4 h-4 z" class="brand"/></defs>
  <g transform="translate(10 10) rotate(90) scale(2)" color="teal">
    <use xlink:href="#dot" x="1" y="1"/>
    <rect id="accent" width="2" height="2" style="stroke-width:0.5"/>
    <rect width="2" height="2" fill="currentColor" style="fill: currentColor"/>
  </g>
  <sodipodi:namedview xmlns:sodipodi="http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd" pagecolor="#ffffff"/>
</svg>`)
	for _, want := range []string{"{RGB}{176,141,87}", "{RGB}{128,0,128}", "{RGB}{0,128,128}", "{RGB}{0,0,255}", "\\pgfsetlinewidth{0.75bp}"} {
		if !strings.Contains(picture.Source, want) {
			t.Fatalf("styled SVG lacks %q:\n%s", want, picture.Source)
		}
	}
}

// Every external or active construct is refused with a message naming it.
func TestSVGSecurityPolicyRejectsActiveAndExternalContent(t *testing.T) {
	head := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="10" height="10">`
	cases := []struct{ svg, problem string }{
		{`<?xml version="1.0"?><!DOCTYPE svg [<!ENTITY lol "lol"><!ENTITY lol2 "&lol;&lol;&lol;">]><svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"><title>&lol2;</title></svg>`,
			"SVG must not contain a DOCTYPE or entity declarations"},
		{`<?xml-stylesheet href="https://evil.example/x.css"?>` + head + `</svg>`, "SVG must not contain the processing instruction <?xml-stylesheet?>"},
		{head + `<script>alert(1)</script></svg>`, "scripts are not allowed in document SVG"},
		{head + `<foreignObject><div xmlns="http://www.w3.org/1999/xhtml">x</div></foreignObject></svg>`, "foreignObject is not allowed in document SVG"},
		{head + `<image href="https://evil.example/tracker.png" width="1" height="1"/></svg>`, "embedded or linked raster images are not supported in document SVG"},
		{head + `<image xlink:href="file:///etc/passwd" width="1" height="1"/></svg>`, "embedded or linked raster images are not supported in document SVG"},
		{head + `<use href="https://evil.example/sprite.svg#icon"/></svg>`, "<use> href must reference an element in the same SVG (#id); external and file references are not allowed"},
		{head + `<use xlink:href="file:///tmp/other.svg#a"/></svg>`, "<use> xlink:href must reference an element in the same SVG (#id); external and file references are not allowed"},
		{head + `<rect width="1" height="1" onclick="steal()"/></svg>`, "event attribute onclick is not allowed in document SVG"},
		{`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1" onload="x()"></svg>`, "event attribute onload is not allowed in document SVG"},
		{head + `<style>@import url("https://evil.example/a.css");</style></svg>`, "CSS at-rules such as @import and @font-face are not supported in <style>"},
		{head + `<rect width="1" height="1" fill="url(#grad)"/></svg>`, "references such as url(...) are not supported (in fill of <rect>)"},
		{head + `<rect width="1" height="1" style="fill:url('https://evil.example/p')"/></svg>`, "references such as url(...) are not supported (in style of <rect>)"},
		{head + `<text x="0" y="5">Hi</text></svg>`, "SVG text is not supported; convert text to outlines (paths) in the SVG editor"},
		{head + `<linearGradient id="g"/></svg>`, "gradients are not supported; use solid fills"},
		{head + `<svg width="1" height="1"/></svg>`, "nested <svg> elements are not supported"},
		{head + `<a href="https://evil.example"><rect width="1" height="1"/></a></svg>`, "links (<a>) are not allowed in document SVG"},
		{head + `<animate attributeName="x"/></svg>`, "animation is not supported"},
		{head + `<g id="a"><use href="#a"/></g></svg>`, "<use> references form a cycle or are nested too deeply"},
		{head + `<rect width="1" height="1" transform="translate(1,2"/></svg>`, `transform "translate(1,2" is not valid`},
		{head + `<path d="M0 0 L 1"/></svg>`, `<path>: path data "M0 0 L 1" is not valid`},
		{head + `<rect width="1" height="1" fill="hsl(0,100%,50%)"/></svg>`, `<rect>: fill "hsl(0,100%,50%)" is not a supported color`},
		{head + `<path d="M0 0 L 100000 0" stroke="black"/></svg>`, "a coordinate of <path> lies too far outside the drawing"},
		{`<html><body/></html>`, "the root element must be <svg>, not <html>"},
		{`not xml`, "SVG has no <svg> root element"},
		{`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"><rect></svg>`, "SVG is not well-formed XML: XML syntax error on line 1: element <rect> closed by </svg>"},
	}
	for _, test := range cases {
		_, problem := AhdSVGConvert("Latex.image", []byte(test.svg))
		if problem != "Latex.image: "+test.problem {
			t.Fatalf("problem\n got %q\nwant %q\nfor %s", problem, "Latex.image: "+test.problem, test.svg)
		}
	}
	deep := strings.Repeat("<g>", 70) + strings.Repeat("</g>", 70)
	if _, problem := AhdSVGConvert("test", []byte(head+deep+"</svg>")); problem != "test: SVG elements are nested more than 64 levels deep" {
		t.Fatalf("deep nesting problem %q", problem)
	}
	huge := make([]byte, ahdSVGMaximumBytes+1)
	if _, problem := AhdSVGConvert("test", huge); problem != "test: SVG is larger than 5 MiB" {
		t.Fatalf("huge SVG problem %q", problem)
	}
}

// QR.saveSVG and Barcode.saveSVG output is a first-class document asset.
func TestSVGRoundTripsGeneratedCodes(t *testing.T) {
	matrix, _ := AhdQRMatrixText("test", "https://ahdcode.org", "M")
	qrSVG, _ := AhdQRSVGBytes("test", matrix, 3)
	picture, problem := AhdSVGConvert("test", qrSVG)
	if problem != "" {
		t.Fatal(problem)
	}
	if abs(picture.Width-72/2.54*3) > 1e-9 || picture.Width != picture.Height {
		t.Fatalf("QR SVG size %.4fx%.4f", picture.Width, picture.Height)
	}
	pattern, _, _ := AhdBarcodePatternText("test", "Code128", "AHD-42")
	barSVG, _ := AhdBarcodeSVGBytes("test", "Code128", pattern, 8, 2.4)
	picture, problem = AhdSVGConvert("test", barSVG)
	if problem != "" {
		t.Fatal(problem)
	}
	if abs(picture.Width-72/2.54*8) > 1e-9 || abs(picture.Height-72/2.54*2.4) > 1e-9 {
		t.Fatalf("barcode SVG size %.4fx%.4f", picture.Width, picture.Height)
	}
	// preserveAspectRatio="none" stretches the one-unit-tall viewBox to the
	// full height, so every bar spans the whole drawing.
	if !strings.Contains(picture.Source, "bp}{0bp}}") || !strings.Contains(picture.Source, "bp}{"+ahdSVGNumber(72/2.54*2.4)+"bp}}") {
		t.Fatalf("barcode bars do not span the drawing height:\n%s", picture.Source[:500])
	}
}
