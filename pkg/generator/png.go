// png.go — PNG file writer.
package generator

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

// writePNG encodes img to a PNG file at the given path.
func writePNG(output string, img image.Image) error {
	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create %s: %w", output, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encode PNG: %w", err)
	}
	return nil
}

// toRGBA is a convenience to construct color.RGBA with full alpha.
func toRGBA(r, g, b uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
