package main

import (
	"fmt"
	"image/color"
	"os"

	"ahdcode/internal/plotproto"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

func renderMath(request plotproto.Request) error {
	if request.Math == nil || request.Math.Formula == "" {
		return fmt.Errorf("math request has no formula")
	}
	if request.OutputPath == "" {
		return fmt.Errorf("math request has no output path")
	}
	handler := plotTextHandler()
	if err := validateMathText(request.Math.Formula, "math label", handler); err != nil {
		return err
	}
	font := plot.DefaultFont
	font.Size = vg.Points(request.Math.FontSize)
	if request.Math.FontSize <= 0 {
		font.Size = vg.Points(12)
	}
	width, height, depth := handler.Box(request.Math.Formula, font)
	padding := vg.Points(4)
	if width <= 0 {
		width = vg.Points(1)
	}
	if height+depth <= 0 {
		height = font.Size
	}
	canvas := vgimg.NewWith(vgimg.UseWH(width+2*padding, height+depth+2*padding), vgimg.UseBackgroundColor(color.Transparent))
	style := draw.TextStyle{Font: font, Handler: handler, Color: color.Black}
	handler.Draw(canvas, request.Math.Formula, style, vg.Point{X: padding, Y: height + padding})
	file, err := os.Create(request.OutputPath)
	if err != nil {
		return fmt.Errorf("creating math output: %w", err)
	}
	_, writeErr := (vgimg.PngCanvas{Canvas: canvas}).WriteTo(file)
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("writing math output: %w", writeErr)
	}
	return closeErr
}
