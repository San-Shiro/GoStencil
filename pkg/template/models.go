// Package template provides JSON-driven image/video generation.
package template

import "image"

// LayoutSpec defines the visual structure of the output, including canvas size,
// background configuration, margins, font settings, and textbox regions.
type LayoutSpec struct {
	Canvas     Canvas     `json:"canvas"`     // Output dimensions and presets
	Background Background `json:"background"` // Color or image background
	Margin     Margin     `json:"margin"`     // Spacing around the safe content area
	Font       FontConfig `json:"font"`       // Font configuration and fallbacks
	Textboxes  []Textbox  `json:"textboxes"`  // List of regions where text can be rendered
}

// Canvas defines the logical dimensions of the output media.
// Presets like "1080p" can be used to automatically set Width and Height.
type Canvas struct {
	Width  int    `json:"width"`  // Width in pixels
	Height int    `json:"height"` // Height in pixels
	Preset string `json:"preset"` // Optional preset name (e.g., "1080p", "instagram_story")
}

// Background defines the image background.
type Background struct {
	Type   string `json:"type"`   // "image" or "color"
	Source string `json:"source"` // Path to image file
	Color  string `json:"color"`  // Hex color fallback
}

// Margin defines spacing around content area.
type Margin struct {
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
}

// FontConfig defines the font to use.
type FontConfig struct {
	Path     string `json:"path"`     // Custom TTF path
	Fallback string `json:"fallback"` // "embedded" for default
}

// Textbox defines a text region.
type Textbox struct {
	ID      string  `json:"id"`
	X       float64 `json:"x"`      // Relative X (0.0-1.0)
	Y       float64 `json:"y"`      // Relative Y (0.0-1.0)
	Width   float64 `json:"width"`  // Relative width (0.0-1.0)
	Height  float64 `json:"height"` // Relative height (0.0-1.0)
	Padding int     `json:"padding"`
	Style   Style   `json:"style"`
}

// Style defines text appearance.
type Style struct {
	FontSize   float64 `json:"fontSize"`
	Color      string  `json:"color"`
	LineHeight float64 `json:"lineHeight"`
}

// ContentSpec defines text content for textboxes.
type ContentSpec struct {
	Textboxes map[string]TextboxContent `json:"textboxes"`
}

// TextboxContent defines content for a single textbox.
type TextboxContent struct {
	Title      string     `json:"title"`
	TitleStyle Style      `json:"titleStyle"`
	Items      []TextItem `json:"items"`
}

// TextItem defines a single text entry.
type TextItem struct {
	Type  string `json:"type"` // "text", "bullet", "numbered"
	Text  string `json:"text"`
	Style Style  `json:"style"` // Override style
}

// Resolved dimensions for a textbox after applying margins.
type ResolvedTextbox struct {
	ID      string
	X       int
	Y       int
	Width   int
	Height  int
	Padding int
	Style   Style
}

// TemplateResult holds the rendered output.
type TemplateResult struct {
	Image  image.Image
	Width  int
	Height int
}

// Presets for common resolutions.
var Presets = map[string][2]int{
	"720p":             {1280, 720},
	"1080p":            {1920, 1080},
	"4k":               {3840, 2160},
	"instagram_square": {1080, 1080},
	"instagram_story":  {1080, 1920},
	"youtube_thumb":    {1280, 720},
}
