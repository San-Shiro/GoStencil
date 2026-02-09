// color.go — Unified color parsing and solid image creation.
package generator

import (
	"crypto/rand"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
)

// ParseColor parses a color string. Accepts "#rrggbb", "random", or "".
// Empty string is treated as "random".
func ParseColor(s string) (r, g, b uint8, err error) {
	if s == "" || s == "random" {
		buf := make([]byte, 3)
		if _, err := rand.Read(buf); err != nil {
			return 0, 0, 0, fmt.Errorf("random color: %w", err)
		}
		return buf[0], buf[1], buf[2], nil
	}

	hex := strings.TrimPrefix(s, "#")
	if len(hex) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid color %q: expected 6-char hex", s)
	}

	rv, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid red channel in %q: %w", s, err)
	}
	gv, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid green channel in %q: %w", s, err)
	}
	bv, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid blue channel in %q: %w", s, err)
	}

	return uint8(rv), uint8(gv), uint8(bv), nil
}

// ParseHexRGBA converts a "#rrggbb" string to color.RGBA.
// Returns white on any parse error (safe default for rendering).
func ParseHexRGBA(hex string) color.RGBA {
	r, g, b, err := ParseColor(hex)
	if err != nil {
		return color.RGBA{255, 255, 255, 255}
	}
	return color.RGBA{r, g, b, 255}
}

// NewSolidImage creates a uniform solid-color image using draw.Draw (O(1) fill).
func NewSolidImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return img
}
