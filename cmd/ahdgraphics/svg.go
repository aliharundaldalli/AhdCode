package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
)

// encodeSVG writes the same command list the rasterizer draws, as vector
// elements. It has no script and no external reference: text is written as
// glyph outlines and an image is embedded as PNG data, so the file looks the
// same wherever it is opened. Its only free text is escaped.
func encodeSVG(value snapshot) []byte {
	var out bytes.Buffer
	width, height := float64(value.width), float64(value.height)
	t := transform{width: width, height: height, scale: 1}

	out.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&out, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n",
		value.width, value.height, value.width, value.height)
	out.WriteString("<title>")
	_ = xml.EscapeText(&out, []byte(value.title))
	out.WriteString("</title>\n")
	fmt.Fprintf(&out, `<rect x="0" y="0" width="%d" height="%d"%s/>`+"\n", value.width, value.height, paint("fill", &value.background))

	for _, drawn := range value.commands {
		switch drawn.kind {
		case lineCommand:
			a, b := t.pixel(drawn.a, drawn.b), t.pixel(drawn.c, drawn.d)
			fmt.Fprintf(&out, `<line x1="%s" y1="%s" x2="%s" y2="%s"%s stroke-width="%s" stroke-linecap="round"/>`+"\n",
				number(a.x), number(a.y), number(b.x), number(b.y), paint("stroke", &drawn.stroke), number(drawn.width))
		case circleCommand:
			if drawn.c <= 0 {
				continue
			}
			center := t.pixel(drawn.a, drawn.b)
			fmt.Fprintf(&out, `<circle cx="%s" cy="%s" r="%s"%s%s stroke-width="%s"/>`+"\n",
				number(center.x), number(center.y), number(drawn.c), fillPaint(drawn), paint("stroke", &drawn.stroke), number(drawn.width))
		case rectangleCommand:
			if drawn.c <= 0 || drawn.d <= 0 {
				continue
			}
			// The Canvas corner is the lower-left one; SVG wants the top-left.
			topLeft := t.pixel(drawn.a, drawn.b+drawn.d)
			fmt.Fprintf(&out, `<rect x="%s" y="%s" width="%s" height="%s"%s%s stroke-width="%s"/>`+"\n",
				number(topLeft.x), number(topLeft.y), number(drawn.c), number(drawn.d), fillPaint(drawn), paint("stroke", &drawn.stroke), number(drawn.width))
		case ellipseCommand:
			if drawn.c <= 0 || drawn.d <= 0 {
				continue
			}
			center := t.pixel(drawn.a, drawn.b)
			fmt.Fprintf(&out, `<ellipse cx="%s" cy="%s" rx="%s" ry="%s"%s%s stroke-width="%s"/>`+"\n",
				number(center.x), number(center.y), number(drawn.c), number(drawn.d), fillPaint(drawn), paint("stroke", &drawn.stroke), number(drawn.width))
		case polygonCommand:
			out.WriteString(`<polygon points="`)
			for index := 0; index+1 < len(drawn.points); index += 2 {
				if index > 0 {
					out.WriteByte(' ')
				}
				vertex := t.pixel(drawn.points[index], drawn.points[index+1])
				out.WriteString(number(vertex.x) + "," + number(vertex.y))
			}
			fmt.Fprintf(&out, `"%s%s stroke-width="%s" stroke-linejoin="round"/>`+"\n",
				fillPaint(drawn), paint("stroke", &drawn.stroke), number(drawn.width))
		case arcCommand:
			if drawn.c <= 0 {
				continue
			}
			center := t.pixel(drawn.a, drawn.b)
			sweep := arcSweep(drawn.start, drawn.end)
			if math.Abs(sweep) >= 360 {
				fmt.Fprintf(&out, `<circle cx="%s" cy="%s" r="%s" fill="none"%s stroke-width="%s"/>`+"\n",
					number(center.x), number(center.y), number(drawn.c), paint("stroke", &drawn.stroke), number(drawn.width))
				continue
			}
			at := func(degrees float64) point {
				angle := degrees * math.Pi / 180
				return point{center.x + drawn.c*math.Cos(angle), center.y - drawn.c*math.Sin(angle)}
			}
			from, to := at(drawn.start), at(drawn.start+sweep)
			large, clockwise := 0, 0
			if math.Abs(sweep) > 180 {
				large = 1
			}
			// SVG's sweep flag is in pixel space, where y points down: a
			// counter-clockwise Canvas arc is drawn with flag 0.
			if sweep < 0 {
				clockwise = 1
			}
			fmt.Fprintf(&out, `<path d="M%s %s A%s %s 0 %d %d %s %s" fill="none"%s stroke-width="%s" stroke-linecap="round"/>`+"\n",
				number(from.x), number(from.y), number(drawn.c), number(drawn.c), large, clockwise, number(to.x), number(to.y),
				paint("stroke", &drawn.stroke), number(drawn.width))
		case textCommand:
			ops := textOutline(drawn.text, drawn.align, t.pixel(drawn.a, drawn.b), drawn.c)
			if len(ops) == 0 {
				continue
			}
			out.WriteString(`<path aria-label="`)
			_ = xml.EscapeText(&out, []byte(drawn.text))
			out.WriteString(`" d="`)
			for _, op := range ops {
				switch op.kind {
				case 0:
					out.WriteString("M" + number(op.p[0].x) + " " + number(op.p[0].y))
				case 1:
					out.WriteString("L" + number(op.p[0].x) + " " + number(op.p[0].y))
				case 2:
					out.WriteString("C" + number(op.p[0].x) + " " + number(op.p[0].y) + " " + number(op.p[1].x) + " " + number(op.p[1].y) + " " + number(op.p[2].x) + " " + number(op.p[2].y))
				case 3:
					out.WriteString("Z")
				case 4:
					out.WriteString("Q" + number(op.p[0].x) + " " + number(op.p[0].y) + " " + number(op.p[1].x) + " " + number(op.p[1].y))
				}
			}
			fmt.Fprintf(&out, `"%s/>`+"\n", paint("fill", &drawn.stroke))
		case imageCommand:
			if drawn.picture == nil || drawn.c <= 0 || drawn.d <= 0 {
				continue
			}
			data := drawn.picture.embedded()
			if data == "" {
				continue
			}
			topLeft := t.pixel(drawn.a, drawn.b+drawn.d)
			fmt.Fprintf(&out, `<image x="%s" y="%s" width="%s" height="%s" preserveAspectRatio="none" href="%s"/>`+"\n",
				number(topLeft.x), number(topLeft.y), number(drawn.c), number(drawn.d), data)
		}
	}
	out.WriteString("</svg>\n")
	return out.Bytes()
}

func fillPaint(drawn command) string {
	if !drawn.hasFill {
		return ` fill="none"`
	}
	return paint("fill", &drawn.fill)
}

// paint renders one color attribute, plus an opacity attribute when the
// color is not fully opaque.
func paint(attribute string, value *rgba) string {
	text := fmt.Sprintf(` %s="#%02x%02x%02x"`, attribute, value[0], value[1], value[2])
	if value[3] != 255 {
		text += fmt.Sprintf(` %s-opacity="%s"`, attribute, number(float64(value[3])/255))
	}
	return text
}

// number prints a coordinate with at most four decimals and no exponent, so
// the same drawing always serializes to the same text.
func number(value float64) string {
	rounded := math.Round(value*1e4) / 1e4
	if rounded == 0 {
		rounded = 0 // normalizes -0
	}
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}
