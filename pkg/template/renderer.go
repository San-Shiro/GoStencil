// renderer.go - Template rendering engine for compositing images from JSON specifications.
// Handles background rendering, textbox positioning, text wrapping, and font styling.
// Uses a layered approach: background -> textboxes -> content items.
package template

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Renderer handles image composition from templates.
type Renderer struct {
	fontManager *FontManager
	dpi         float64
}

// NewRenderer creates a new template renderer.
func NewRenderer(fontPath string) (*Renderer, error) {
	fm, err := NewFontManager(fontPath)
	if err != nil {
		return nil, err
	}

	return &Renderer{
		fontManager: fm,
		dpi:         72,
	}, nil
}

// Render creates an image from layout and content specs. It follows a layered approach:
// 1. Draws the background (image or solid color)
// 2. Resolves relative textbox positions to absolute pixels
// 3. Renders the text content within each textbox with auto-wrapping and styling.
func (r *Renderer) Render(layout *LayoutSpec, content *ContentSpec) (*image.RGBA, error) {
	// Create base image
	img := image.NewRGBA(image.Rect(0, 0, layout.Canvas.Width, layout.Canvas.Height))

	// Draw background
	if err := r.drawBackground(img, layout); err != nil {
		return nil, err
	}

	// Resolve textbox positions
	textboxes := ResolveTextboxes(layout)

	// Draw each textbox content
	for _, tb := range textboxes {
		if tbContent, ok := content.Textboxes[tb.ID]; ok {
			if err := r.drawTextbox(img, tb, tbContent); err != nil {
				return nil, err
			}
		}
	}

	return img, nil
}

// drawBackground fills the image with color or loads a background image.
func (r *Renderer) drawBackground(img *image.RGBA, layout *LayoutSpec) error {
	// Try to load background image
	if layout.Background.Type == "image" && layout.Background.Source != "" {
		bgFile, err := os.Open(layout.Background.Source)
		if err == nil {
			defer bgFile.Close()
			bgImg, _, err := image.Decode(bgFile)
			if err == nil {
				// Scale/draw background image
				draw.Draw(img, img.Bounds(), bgImg, image.Point{}, draw.Src)
				return nil
			}
			fmt.Printf("Warning: could not decode background image: %v, using color fallback\n", err)
		} else {
			fmt.Printf("Warning: could not open background image: %v, using color fallback\n", err)
		}
	}

	// Fallback to solid color
	bgColor := parseHexColor(layout.Background.Color)
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	return nil
}

// drawTextbox renders text content (title and items) within a resolved textbox region.
// It handles font scaling, line height, and different list types (bullet, numbered).
func (r *Renderer) drawTextbox(img *image.RGBA, tb ResolvedTextbox, content TextboxContent) error {
	// Calculate drawing area
	drawX := tb.X + tb.Padding
	drawY := tb.Y + tb.Padding
	drawW := tb.Width - (2 * tb.Padding)

	currentY := drawY

	// Draw title if present
	if content.Title != "" {
		titleStyle := mergeStyles(tb.Style, content.TitleStyle)
		if titleStyle.FontSize <= 0 {
			titleStyle.FontSize = tb.Style.FontSize * 1.4 // Title is 40% larger by default
		}

		face, err := r.fontManager.GetFace(titleStyle.FontSize, r.dpi)
		if err != nil {
			return err
		}

		textColor := parseHexColor(titleStyle.Color)
		lineHeight := int(titleStyle.FontSize * titleStyle.LineHeight)
		if lineHeight == 0 {
			lineHeight = int(titleStyle.FontSize * 1.5)
		}

		lines := r.wrapText(content.Title, drawW, face)
		for _, line := range lines {
			currentY += lineHeight
			r.drawString(img, line, drawX, currentY, textColor, face)
		}

		// Add spacing after title
		currentY += int(titleStyle.FontSize * 0.5)
	}

	// Draw items
	numberedIndex := 1
	for _, item := range content.Items {
		itemStyle := mergeStyles(tb.Style, item.Style)

		face, err := r.fontManager.GetFace(itemStyle.FontSize, r.dpi)
		if err != nil {
			return err
		}

		textColor := parseHexColor(itemStyle.Color)
		lineHeight := int(itemStyle.FontSize * itemStyle.LineHeight)
		if lineHeight == 0 {
			lineHeight = int(itemStyle.FontSize * 1.5)
		}

		// Prepare text with prefix
		var text string
		var indent int
		switch item.Type {
		case "bullet":
			text = "• " + item.Text
			indent = int(itemStyle.FontSize * 1.2)
		case "numbered":
			text = fmt.Sprintf("%d. %s", numberedIndex, item.Text)
			numberedIndex++
			indent = int(itemStyle.FontSize * 1.5)
		default:
			text = item.Text
			indent = 0
		}

		// Wrap and draw text
		lines := r.wrapText(text, drawW-indent, face)
		for i, line := range lines {
			currentY += lineHeight
			x := drawX
			if i > 0 && indent > 0 {
				x += indent // Indent continuation lines
			}
			r.drawString(img, line, x, currentY, textColor, face)
		}
	}

	return nil
}

// wrapText breaks a single string of text into multiple lines that each fit
// within the specified maxWidth in pixels, using the metrics of the provided font face.
func (r *Renderer) wrapText(text string, maxWidth int, face font.Face) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return lines
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		testLine := currentLine + " " + word
		advance := font.MeasureString(face, testLine)
		if advance.Ceil() > maxWidth {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = testLine
		}
	}
	lines = append(lines, currentLine)

	return lines
}

// drawString draws text at the specified position.
func (r *Renderer) drawString(img *image.RGBA, text string, x, y int, col color.Color, face font.Face) {
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	drawer.DrawString(text)
}

// mergeStyles combines base style with override style.
func mergeStyles(base, override Style) Style {
	result := base
	if override.FontSize > 0 {
		result.FontSize = override.FontSize
	}
	if override.Color != "" {
		result.Color = override.Color
	}
	if override.LineHeight > 0 {
		result.LineHeight = override.LineHeight
	}
	return result
}

// parseHexColor converts a hex color string to color.RGBA.
func parseHexColor(hex string) color.RGBA {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return color.RGBA{255, 255, 255, 255} // Default white
	}

	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)

	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// SavePNG saves an image to a PNG file.
func SavePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}
