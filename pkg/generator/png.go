// png.go - PNG image generator for simple solid-color images and template output.
// Uses the standard library image/png encoder for maximum compatibility.
package generator

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

// PNGGenerator generates PNG image files.
type PNGGenerator struct{}

// NewPNGGenerator creates a new PNG generator.
func NewPNGGenerator() *PNGGenerator {
	return &PNGGenerator{}
}

// Generate creates a PNG file from a source image or solid color.
func (g *PNGGenerator) Generate(output string, config Config) error {
	var img image.Image

	if config.SourceImage != nil {
		img = config.SourceImage
	} else {
		// Create solid color image
		width := config.Width
		height := config.Height
		if width <= 0 {
			width = 320
		}
		if height <= 0 {
			height = 240
		}

		r, gCol, b, err := parseColor(config.Color)
		if err != nil {
			return err
		}

		rgba := image.NewRGBA(image.Rect(0, 0, width, height))
		fillColor := color.RGBA{r, gCol, b, 255}
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				rgba.Set(x, y, fillColor)
			}
		}
		img = rgba
	}

	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}
