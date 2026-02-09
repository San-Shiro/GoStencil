// Package generator provides PNG and AVI media generation.
//
// All output follows a unified pipeline: create an image.Image first,
// then write it as PNG or containerize it as an MJPEG AVI.
package generator

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"path/filepath"
	"strings"
)

// Config holds parameters for media generation.
type Config struct {
	Width    int         // Pixel width (default: 1280)
	Height   int         // Pixel height (default: 720)
	Duration int         // Seconds, AVI only (default: 1)
	Color    string      // Hex "#rrggbb" or "random"
	Image    image.Image // Pre-rendered image; overrides Width/Height/Color
}

// Generate creates an output file. The format is inferred from the file extension:
//   - ".png" → PNG image
//   - ".avi" → MJPEG AVI video
//
// If cfg.Image is nil, a solid-color image is created from cfg.Color/Width/Height.
func Generate(output string, cfg Config) error {
	img, err := resolveImage(cfg)
	if err != nil {
		return err
	}

	switch ext := strings.ToLower(filepath.Ext(output)); ext {
	case ".png":
		return writePNG(output, img)
	case ".avi":
		dur := max(cfg.Duration, 1)
		return writeAVI(output, img, dur)
	default:
		return fmt.Errorf("unsupported format %q: use .png or .avi", ext)
	}
}

// GenerateToWriter writes media to an io.Writer. The format is specified by ext (".png" or ".avi").
// This is useful for in-memory generation (e.g., WASM).
func GenerateToWriter(w io.Writer, ext string, cfg Config) error {
	img, err := resolveImage(cfg)
	if err != nil {
		return err
	}

	switch strings.ToLower(ext) {
	case ".png":
		return png.Encode(w, img)
	case ".avi":
		dur := max(cfg.Duration, 1)
		return writeAVITo(w, img, dur)
	default:
		return fmt.Errorf("unsupported format %q: use .png or .avi", ext)
	}
}

// resolveImage returns the source image from config, creating a solid-color
// image if none is provided.
func resolveImage(cfg Config) (image.Image, error) {
	if cfg.Image != nil {
		return cfg.Image, nil
	}

	w := max(cfg.Width, 1280)
	h := max(cfg.Height, 720)

	r, g, b, err := ParseColor(cfg.Color)
	if err != nil {
		return nil, err
	}

	return NewSolidImage(w, h, toRGBA(r, g, b)), nil
}
