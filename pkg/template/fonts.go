// fonts.go — Font loading with embedded Go Regular fallback.
package template

import (
	"fmt"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// FontManager loads and caches a parsed OpenType font.
type FontManager struct {
	parsed *opentype.Font
}

// NewFontManager creates a font manager. If customPath is empty or unreadable,
// the embedded Go Regular font is used as fallback.
func NewFontManager(customPath string) (*FontManager, error) {
	data := goregular.TTF // default

	if customPath != "" {
		if custom, err := os.ReadFile(customPath); err != nil {
			fmt.Printf("Warning: font %q unavailable, using default: %v\n", customPath, err)
		} else {
			data = custom
		}
	}

	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}

	return &FontManager{parsed: parsed}, nil
}

// NewFontManagerFromBytes creates a font manager from raw TTF data.
// If data is nil or empty, the embedded Go Regular font is used.
func NewFontManagerFromBytes(data []byte) (*FontManager, error) {
	if len(data) == 0 {
		data = goregular.TTF
	}

	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}

	return &FontManager{parsed: parsed}, nil
}

// GetFace returns a font.Face at the given size. DPI defaults to 72 if ≤ 0.
func (fm *FontManager) GetFace(size, dpi float64) (font.Face, error) {
	dpi = max(dpi, 72)

	face, err := opentype.NewFace(fm.parsed, &opentype.FaceOptions{
		Size:    size,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("create font face at %.1fpt: %w", size, err)
	}
	return face, nil
}
