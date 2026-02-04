// fonts.go - Font management with custom TTF support and embedded fallback font.
// Uses golang.org/x/image/font for OpenType rendering. Defaults to Go Regular
// font when no custom font is specified or when custom font loading fails.
package template

import (
	"fmt"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// FontManager handles font loading with fallback.
type FontManager struct {
	fontData []byte
	parsed   *opentype.Font
}

// NewFontManager creates a font manager with the specified font.
// If customPath is empty or invalid, uses embedded Go font.
func NewFontManager(customPath string) (*FontManager, error) {
	var fontData []byte
	var err error

	// Try custom font first
	if customPath != "" {
		fontData, err = os.ReadFile(customPath)
		if err != nil {
			fmt.Printf("Warning: could not load custom font '%s', using default\n", customPath)
			fontData = nil
		}
	}

	// Fallback to embedded Go font
	if fontData == nil {
		fontData = goregular.TTF
	}

	parsed, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	return &FontManager{
		fontData: fontData,
		parsed:   parsed,
	}, nil
}

// GetFace returns a font.Face at the specified size.
func (fm *FontManager) GetFace(size float64, dpi float64) (font.Face, error) {
	if dpi <= 0 {
		dpi = 72
	}

	face, err := opentype.NewFace(fm.parsed, &opentype.FaceOptions{
		Size:    size,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create font face: %w", err)
	}

	return face, nil
}
