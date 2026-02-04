// parser.go - JSON parsing and validation for template layout and content specifications.
// Handles preset resolution, default value application, and textbox coordinate conversion.
package template

import (
	"encoding/json"
	"fmt"
	"os"
)

// ParseLayout loads and validates a layout JSON file from the given path.
// It also applies presets and sets default values for dimensions and styles.
func ParseLayout(path string) (*LayoutSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read layout file: %w", err)
	}

	var layout LayoutSpec
	if err := json.Unmarshal(data, &layout); err != nil {
		return nil, fmt.Errorf("failed to parse layout JSON: %w", err)
	}

	// Apply preset if specified
	if layout.Canvas.Preset != "" {
		if dims, ok := Presets[layout.Canvas.Preset]; ok {
			layout.Canvas.Width = dims[0]
			layout.Canvas.Height = dims[1]
		}
	}

	// Set defaults
	if layout.Canvas.Width <= 0 {
		layout.Canvas.Width = 1920
	}
	if layout.Canvas.Height <= 0 {
		layout.Canvas.Height = 1080
	}

	// Default background
	if layout.Background.Color == "" {
		layout.Background.Color = "#1a1a2e"
	}

	// Set default styles for textboxes
	for i := range layout.Textboxes {
		if layout.Textboxes[i].Style.FontSize <= 0 {
			layout.Textboxes[i].Style.FontSize = 24
		}
		if layout.Textboxes[i].Style.Color == "" {
			layout.Textboxes[i].Style.Color = "#ffffff"
		}
		if layout.Textboxes[i].Style.LineHeight <= 0 {
			layout.Textboxes[i].Style.LineHeight = 1.5
		}
	}

	return &layout, nil
}

// ParseContent loads and validates a content JSON file from the given path.
// It maps text content and styles to the textbox IDs defined in the layout.
func ParseContent(path string) (*ContentSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read content file: %w", err)
	}

	var content ContentSpec
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("failed to parse content JSON: %w", err)
	}

	return &content, nil
}

// ValidateMapping ensures that all textbox IDs referenced in the content
// actually exist in the provided layout specification.
func ValidateMapping(layout *LayoutSpec, content *ContentSpec) error {
	layoutIDs := make(map[string]bool)
	for _, tb := range layout.Textboxes {
		layoutIDs[tb.ID] = true
	}

	for id := range content.Textboxes {
		if !layoutIDs[id] {
			return fmt.Errorf("content references unknown textbox ID: %s", id)
		}
	}

	return nil
}

// ResolveTextboxes converts relative (0.0 to 1.0) coordinates and dimensions
// from the LayoutSpec into absolute pixel values based on the canvas size and margins.
func ResolveTextboxes(layout *LayoutSpec) []ResolvedTextbox {
	// Calculate content area (inside margins)
	contentX := layout.Margin.Left
	contentY := layout.Margin.Top
	contentW := layout.Canvas.Width - layout.Margin.Left - layout.Margin.Right
	contentH := layout.Canvas.Height - layout.Margin.Top - layout.Margin.Bottom

	resolved := make([]ResolvedTextbox, len(layout.Textboxes))
	for i, tb := range layout.Textboxes {
		resolved[i] = ResolvedTextbox{
			ID:      tb.ID,
			X:       contentX + int(tb.X*float64(contentW)),
			Y:       contentY + int(tb.Y*float64(contentH)),
			Width:   int(tb.Width * float64(contentW)),
			Height:  int(tb.Height * float64(contentH)),
			Padding: tb.Padding,
			Style:   tb.Style,
		}
	}

	return resolved
}

// GetExampleJSON returns sample JSON structures for layout and content.
func GetExampleJSON() (layout string, content string) {
	layout = `{
  "canvas": {
    "preset": "1080p"
  },
  "background": {
    "type": "color",
    "color": "#1a1a2e"
  },
  "margin": {
    "top": 80,
    "right": 100,
    "bottom": 80,
    "left": 100
  },
  "textboxes": [
    {
      "id": "main",
      "x": 0,
      "y": 0,
      "width": 1.0,
      "height": 1.0,
      "padding": 20,
      "style": {
        "fontSize": 32,
        "color": "#ffffff",
        "lineHeight": 1.5
      }
    }
  ]
}`

	content = `{
  "textboxes": {
    "main": {
      "title": "Example Title",
      "items": [
        { "type": "bullet", "text": "First bullet point" },
        { "type": "numbered", "text": "First numbered item" },
        { "type": "text", "text": "Plain paragraph text." }
      ]
    }
  }
}`
	return
}
