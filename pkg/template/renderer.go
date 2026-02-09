// renderer.go — Rendering engine for presets and legacy templates.
//
// Preset pipeline: background → containers (bg, border, corner radius, image) → text content.
// Supports: backgroundColor with alpha, backgroundImage (PNG/JPG), borderColor/Width,
// cornerRadius, textAlign (left/center/right), bullet/numbered lists, text wrapping.
package template

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // register JPEG decoder
	"image/png"

	"os"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Renderer composites images from presets or legacy templates.
type Renderer struct {
	fontManager *FontManager
	dpi         float64
}

// NewRenderer creates a renderer with the specified font (empty = embedded default).
func NewRenderer(fontPath string) (*Renderer, error) {
	fm, err := NewFontManager(fontPath)
	if err != nil {
		return nil, err
	}
	return &Renderer{fontManager: fm, dpi: 72}, nil
}

// NewRendererFromBytes creates a renderer from raw TTF font data.
// If fontData is nil or empty, the embedded Go Regular font is used.
func NewRendererFromBytes(fontData []byte) (*Renderer, error) {
	fm, err := NewFontManagerFromBytes(fontData)
	if err != nil {
		return nil, err
	}
	return &Renderer{fontManager: fm, dpi: 72}, nil
}

// ── Preset Rendering ──

// RenderPreset creates an image from a preset and its resolved components.
func (r *Renderer) RenderPreset(preset *Preset, components []ResolvedComponent) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, preset.Canvas.Width, preset.Canvas.Height))

	// Draw background.
	if err := r.drawPresetBackground(img, preset); err != nil {
		return nil, err
	}

	// Draw each visible component.
	for _, comp := range components {
		if err := r.drawComponent(img, comp); err != nil {
			return nil, err
		}
	}

	return img, nil
}

// drawPresetBackground fills with an image or solid color.
func (r *Renderer) drawPresetBackground(img *image.RGBA, preset *Preset) error {
	if preset.Background.Type == "image" && preset.Background.Source != "" {
		if bgImg, err := loadImage(preset.Background.Source); err == nil {
			drawScaled(img, bgImg)
			return nil
		}
	}

	c := parseHexColorAlpha(preset.Background.Color)
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return nil
}

// drawComponent paints a component's container and content.
func (r *Renderer) drawComponent(img *image.RGBA, comp ResolvedComponent) error {
	bounds := image.Rect(comp.X, comp.Y, comp.X+comp.Width, comp.Y+comp.Height)

	// 1. Container background.
	if comp.Style.BackgroundColor != "" {
		bgColor := parseHexColorAlpha(comp.Style.BackgroundColor)
		if bgColor.A > 0 {
			if comp.Style.CornerRadius > 0 {
				drawRoundedRect(img, bounds, bgColor, comp.Style.CornerRadius)
			} else {
				drawRect(img, bounds, bgColor)
			}
		}
	}

	// 2. Background image (sticker/logo).
	if comp.Style.BackgroundImage != "" {
		if bgImg, err := loadImage(comp.Style.BackgroundImage); err == nil {
			subImg := img.SubImage(bounds).(*image.RGBA)
			fit := comp.Style.BackgroundFit
			if fit == "" {
				fit = "stretch"
			}
			switch fit {
			case "contain":
				drawContain(subImg, bgImg)
			case "cover":
				drawCover(subImg, bgImg)
			default: // "stretch"
				drawScaled(subImg, bgImg)
			}
		} else {
			fmt.Printf("Warning: could not load background image %q: %v\n", comp.Style.BackgroundImage, err)
		}
	}

	// 3. Border.
	if comp.Style.BorderWidth > 0 && comp.Style.BorderColor != "" {
		borderColor := parseHexColorAlpha(comp.Style.BorderColor)
		if comp.Style.CornerRadius > 0 {
			drawRoundedBorder(img, bounds, borderColor, comp.Style.CornerRadius, comp.Style.BorderWidth)
		} else {
			drawBorder(img, bounds, borderColor, comp.Style.BorderWidth)
		}
	}

	// 4. Text content (title + items).
	return r.drawComponentContent(img, comp)
}

// drawComponentContent renders title and items within a component.
func (r *Renderer) drawComponentContent(img *image.RGBA, comp ResolvedComponent) error {
	if comp.Data.Title == "" && len(comp.Data.Items) == 0 {
		return nil // image-only component
	}

	pad := comp.Padding
	drawX := comp.X + pad
	drawY := comp.Y + pad
	drawW := comp.Width - 2*pad
	if drawW <= 0 {
		return nil
	}

	currentY := drawY
	align := comp.Style.TextAlign

	// Resolve per-component font (with fallback to global).
	fontMgr := r.fontManager
	if comp.Style.FontPath != "" {
		if compFM, err := NewFontManager(comp.Style.FontPath); err == nil {
			fontMgr = compFM
		} else {
			fmt.Printf("Warning: component %q font %q unavailable, using global: %v\n", comp.ID, comp.Style.FontPath, err)
		}
	}

	// Title.
	if comp.Data.Title != "" {
		titleSize := comp.Style.FontSize * 1.4
		face, err := fontMgr.GetFace(titleSize, r.dpi)
		if err != nil {
			return err
		}

		titleColor := parseHexColorAlpha(comp.Style.Color)
		lh := int(titleSize * comp.Style.LineHeight)

		for _, line := range r.wrapText(comp.Data.Title, drawW, face) {
			currentY += lh
			x := alignX(drawX, drawW, line, face, align)
			r.drawString(img, line, x, currentY, titleColor, face)
		}
		currentY += int(titleSize * 0.5)
	}

	// Items.
	face, err := fontMgr.GetFace(comp.Style.FontSize, r.dpi)
	if err != nil {
		return err
	}

	textColor := parseHexColorAlpha(comp.Style.Color)
	lh := int(comp.Style.FontSize * comp.Style.LineHeight)
	num := 1

	for _, item := range comp.Data.Items {
		var text string
		var indent int

		switch item.Type {
		case "bullet":
			text = "• " + item.Text
			indent = int(comp.Style.FontSize * 1.2)
		case "numbered":
			text = fmt.Sprintf("%d. %s", num, item.Text)
			num++
			indent = int(comp.Style.FontSize * 1.5)
		default:
			text = item.Text
		}

		for i, line := range r.wrapText(text, drawW-indent, face) {
			currentY += lh
			dx := drawX
			if i > 0 && indent > 0 {
				dx += indent
			}
			x := alignX(dx, drawW, line, face, align)
			r.drawString(img, line, x, currentY, textColor, face)
		}
	}

	return nil
}

// ── Drawing Primitives ──

// drawRect fills a rectangle with alpha blending.
func drawRect(img *image.RGBA, bounds image.Rectangle, c color.RGBA) {
	if c.A == 255 {
		draw.Draw(img, bounds, &image.Uniform{c}, image.Point{}, draw.Src)
	} else {
		draw.Draw(img, bounds, &image.Uniform{c}, image.Point{}, draw.Over)
	}
}

// drawRoundedRect fills a rectangle with rounded corners.
func drawRoundedRect(img *image.RGBA, bounds image.Rectangle, c color.RGBA, radius int) {
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if insideRoundedRect(x, y, bounds, radius) {
				blendPixel(img, x, y, c)
			}
		}
	}
}

// drawBorder draws a rectangular border of given width.
func drawBorder(img *image.RGBA, bounds image.Rectangle, c color.RGBA, w int) {
	// Top
	drawRect(img, image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+w), c)
	// Bottom
	drawRect(img, image.Rect(bounds.Min.X, bounds.Max.Y-w, bounds.Max.X, bounds.Max.Y), c)
	// Left
	drawRect(img, image.Rect(bounds.Min.X, bounds.Min.Y+w, bounds.Min.X+w, bounds.Max.Y-w), c)
	// Right
	drawRect(img, image.Rect(bounds.Max.X-w, bounds.Min.Y+w, bounds.Max.X, bounds.Max.Y-w), c)
}

// drawRoundedBorder draws a border with rounded corners.
func drawRoundedBorder(img *image.RGBA, bounds image.Rectangle, c color.RGBA, radius, width int) {
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			outer := insideRoundedRect(x, y, bounds, radius)
			inner := insideRoundedRect(x, y, bounds.Inset(width), max(radius-width, 0))
			if outer && !inner {
				blendPixel(img, x, y, c)
			}
		}
	}
}

// insideRoundedRect checks if (x,y) is inside a rounded rectangle.
func insideRoundedRect(x, y int, r image.Rectangle, radius int) bool {
	// Check corners.
	corners := [][2]int{
		{r.Min.X + radius, r.Min.Y + radius}, // top-left
		{r.Max.X - radius, r.Min.Y + radius}, // top-right
		{r.Min.X + radius, r.Max.Y - radius}, // bottom-left
		{r.Max.X - radius, r.Max.Y - radius}, // bottom-right
	}

	for _, c := range corners {
		dx := x - c[0]
		dy := y - c[1]
		// Only check if we're in the corner quadrant.
		inCornerX := (c[0] == r.Min.X+radius && x < c[0]) || (c[0] == r.Max.X-radius && x >= c[0])
		inCornerY := (c[1] == r.Min.Y+radius && y < c[1]) || (c[1] == r.Max.Y-radius && y >= c[1])
		if inCornerX && inCornerY {
			if dx*dx+dy*dy > radius*radius {
				return false
			}
		}
	}

	return x >= r.Min.X && x < r.Max.X && y >= r.Min.Y && y < r.Max.Y
}

// blendPixel alpha-blends a color onto a pixel.
func blendPixel(img *image.RGBA, x, y int, c color.RGBA) {
	if c.A == 255 {
		img.SetRGBA(x, y, c)
		return
	}
	if c.A == 0 {
		return
	}
	existing := img.RGBAAt(x, y)
	a := uint32(c.A)
	inv := 255 - a
	img.SetRGBA(x, y, color.RGBA{
		R: uint8((uint32(c.R)*a + uint32(existing.R)*inv) / 255),
		G: uint8((uint32(c.G)*a + uint32(existing.G)*inv) / 255),
		B: uint8((uint32(c.B)*a + uint32(existing.B)*inv) / 255),
		A: uint8(min(uint32(existing.A)+a, 255)),
	})
}

// drawScaled draws src into dst, stretching to fit.
func drawScaled(dst *image.RGBA, src image.Image) {
	dstB := dst.Bounds()
	srcB := src.Bounds()

	scaleX := float64(srcB.Dx()) / float64(dstB.Dx())
	scaleY := float64(srcB.Dy()) / float64(dstB.Dy())

	for y := dstB.Min.Y; y < dstB.Max.Y; y++ {
		for x := dstB.Min.X; x < dstB.Max.X; x++ {
			srcX := srcB.Min.X + int(float64(x-dstB.Min.X)*scaleX)
			srcY := srcB.Min.Y + int(float64(y-dstB.Min.Y)*scaleY)
			r, g, b, a := src.At(srcX, srcY).RGBA()
			px := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
			blendPixel(dst, x, y, px)
		}
	}
}

// drawContain scales src to fit inside dst without stretching (letterbox).
func drawContain(dst *image.RGBA, src image.Image) {
	dstB := dst.Bounds()
	srcB := src.Bounds()

	scale := min(
		float64(dstB.Dx())/float64(srcB.Dx()),
		float64(dstB.Dy())/float64(srcB.Dy()),
	)

	newW := int(float64(srcB.Dx()) * scale)
	newH := int(float64(srcB.Dy()) * scale)
	offX := dstB.Min.X + (dstB.Dx()-newW)/2
	offY := dstB.Min.Y + (dstB.Dy()-newH)/2

	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := srcB.Min.X + int(float64(x)/scale)
			srcY := srcB.Min.Y + int(float64(y)/scale)
			if srcX >= srcB.Max.X {
				srcX = srcB.Max.X - 1
			}
			if srcY >= srcB.Max.Y {
				srcY = srcB.Max.Y - 1
			}
			r, g, b, a := src.At(srcX, srcY).RGBA()
			px := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
			blendPixel(dst, offX+x, offY+y, px)
		}
	}
}

// drawCover scales src to fill dst, cropping excess.
func drawCover(dst *image.RGBA, src image.Image) {
	dstB := dst.Bounds()
	srcB := src.Bounds()

	scale := max(
		float64(dstB.Dx())/float64(srcB.Dx()),
		float64(dstB.Dy())/float64(srcB.Dy()),
	)

	newW := int(float64(srcB.Dx()) * scale)
	newH := int(float64(srcB.Dy()) * scale)
	// Center the crop.
	offX := (newW - dstB.Dx()) / 2
	offY := (newH - dstB.Dy()) / 2

	for y := dstB.Min.Y; y < dstB.Max.Y; y++ {
		for x := dstB.Min.X; x < dstB.Max.X; x++ {
			srcX := srcB.Min.X + int(float64(x-dstB.Min.X+offX)/scale)
			srcY := srcB.Min.Y + int(float64(y-dstB.Min.Y+offY)/scale)
			if srcX < srcB.Min.X {
				srcX = srcB.Min.X
			}
			if srcY < srcB.Min.Y {
				srcY = srcB.Min.Y
			}
			if srcX >= srcB.Max.X {
				srcX = srcB.Max.X - 1
			}
			if srcY >= srcB.Max.Y {
				srcY = srcB.Max.Y - 1
			}
			r, g, b, a := src.At(srcX, srcY).RGBA()
			px := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
			blendPixel(dst, x, y, px)
		}
	}
}

// loadImage reads and decodes an image file (PNG or JPEG).
func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

// ── Text Helpers ──

// wrapText splits text into lines fitting within maxWidth pixels.
func (r *Renderer) wrapText(text string, maxWidth int, face font.Face) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		test := cur + " " + w
		if font.MeasureString(face, test).Ceil() > maxWidth {
			lines = append(lines, cur)
			cur = w
		} else {
			cur = test
		}
	}
	return append(lines, cur)
}

// drawString renders text at (x, y).
func (r *Renderer) drawString(img *image.RGBA, text string, x, y int, c color.Color, face font.Face) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

// alignX computes the x position based on text alignment.
func alignX(baseX, areaWidth int, text string, face font.Face, align string) int {
	switch align {
	case "center":
		tw := font.MeasureString(face, text).Ceil()
		return baseX + (areaWidth-tw)/2
	case "right":
		tw := font.MeasureString(face, text).Ceil()
		return baseX + areaWidth - tw
	default: // "left"
		return baseX
	}
}

// ── Color Parsing ──

// parseHexColorAlpha converts "#rrggbb" or "#rrggbbaa" to color.RGBA.
// Returns white on error.
func parseHexColorAlpha(hex string) color.RGBA {
	hex = strings.TrimPrefix(hex, "#")

	switch len(hex) {
	case 6:
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	case 8:
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		a, _ := strconv.ParseUint(hex[6:8], 16, 8)
		return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
	default:
		return color.RGBA{255, 255, 255, 255}
	}
}

// ── Legacy PNG save ──

// savePNGInline is used by SavePNG to save without import cycles.
func savePNGInline(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	return png.Encode(f, img)
}

// SavePNG saves an image to a PNG file.
func SavePNG(img image.Image, path string) error {
	return savePNGInline(img, path)
}
