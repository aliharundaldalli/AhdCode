package main

import (
	"fmt"
	"image/png"
	"os"

	"ahdcode/cmd/ahdplotmath"
	"ahdcode/internal/plotproto"
)

// renderMath is retained for the narrow helper protocol used by older
// Surface callers. It delegates to the same shared Tectonic renderer as Plot
// labels; it is not a second math implementation.
func renderMath(request plotproto.Request) error {
	if request.Math == nil || request.Math.Formula == "" {
		return fmt.Errorf("math request has no formula")
	}
	if request.OutputPath == "" {
		return fmt.Errorf("math request has no output path")
	}
	root := plotmath.DiscoverLatexRoot(os.Args[0])
	fontSize := request.Math.FontSize
	if fontSize <= 0 {
		fontSize = 12
	}
	asset, err := plotmath.Render(request.Math.Formula, fontSize, root)
	if err != nil {
		return err
	}
	file, err := os.Create(request.OutputPath)
	if err != nil {
		return fmt.Errorf("creating math output: %w", err)
	}
	defer file.Close()
	// The compatibility protocol's output is PNG. The shared asset is already
	// transparent and tightly cropped.
	if err := png.Encode(file, asset.Image); err != nil {
		return fmt.Errorf("writing math output: %w", err)
	}
	return nil
}
