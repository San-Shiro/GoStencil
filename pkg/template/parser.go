// parser.go — Legacy JSON parsing and example generation.
package template

import (
	"encoding/json"
	"fmt"
	"os"
)

// GetExampleJSON returns a sample preset.json and data.json for gostencil init.
func GetExampleJSON() (presetJSON, dataJSON string) {
	presetJSON = `{
  "meta": {
    "name": "Sample Preset",
    "version": "1.0",
    "author": "GoStencil",
    "description": "A simple starter preset"
  },
  "canvas": { "preset": "1080p" },
  "background": {
    "type": "color",
    "color": "#1a1a2e"
  },
  "font": {},
  "components": [
    {
      "id": "header",
      "x": 0.0, "y": 0.0, "width": 1.0, "height": 0.2,
      "padding": 40,
      "style": {
        "backgroundColor": "#16213e",
        "fontSize": 48,
        "color": "#00ffcc",
        "lineHeight": 1.4,
        "textAlign": "center"
      },
      "defaults": {
        "visible": true,
        "title": "Welcome to GoStencil"
      }
    },
    {
      "id": "body",
      "x": 0.05, "y": 0.25, "width": 0.9, "height": 0.5,
      "padding": 30,
      "style": {
        "backgroundColor": "#0f3460",
        "cornerRadius": 16,
        "fontSize": 28,
        "color": "#e0e0e0",
        "lineHeight": 1.6,
        "textAlign": "left"
      },
      "defaults": {
        "visible": true,
        "items": [
          { "type": "bullet", "text": "JSON-driven template engine" },
          { "type": "bullet", "text": "Component visibility control" },
          { "type": "bullet", "text": "Pure Go — no dependencies" },
          { "type": "numbered", "text": "Create a .gspresets bundle" },
          { "type": "numbered", "text": "Provide data.json to customize" }
        ]
      }
    },
    {
      "id": "footer",
      "x": 0.0, "y": 0.85, "width": 1.0, "height": 0.15,
      "padding": 30,
      "style": {
        "backgroundColor": "#16213e80",
        "fontSize": 20,
        "color": "#888888",
        "lineHeight": 1.3,
        "textAlign": "center"
      },
      "defaults": {
        "visible": true,
        "items": [
          { "type": "text", "text": "Generated with GoStencil" }
        ]
      }
    }
  ],
  "schema": {
    "description": "Override text and toggle components via data.json",
    "components": {
      "header": {
        "description": "Top banner with title",
        "fields": {
          "visible": "boolean — show/hide",
          "title": "string — heading text"
        }
      },
      "body": {
        "description": "Main content area",
        "fields": {
          "visible": "boolean",
          "items": "array of {type, text}"
        }
      },
      "footer": {
        "description": "Bottom bar",
        "fields": {
          "visible": "boolean",
          "items": "array of {type, text}"
        }
      }
    }
  }
}`

	dataJSON = `{
  "components": {
    "header": {
      "title": "My Custom Title"
    },
    "body": {
      "items": [
        { "type": "bullet", "text": "Your first point" },
        { "type": "bullet", "text": "Your second point" },
        { "type": "text", "text": "Add more items as needed." }
      ]
    }
  }
}`
	return
}

// ParsePresetFile loads a standalone preset JSON file (for testing without ZIP).
func ParsePresetFile(path string) (*Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read preset: %w", err)
	}

	var preset Preset
	if err := json.Unmarshal(data, &preset); err != nil {
		return nil, fmt.Errorf("parse preset JSON: %w", err)
	}

	// Apply canvas preset.
	if dims, ok := Presets[preset.Canvas.Preset]; ok {
		preset.Canvas.Width = dims[0]
		preset.Canvas.Height = dims[1]
	}
	preset.Canvas.Width = max(preset.Canvas.Width, 1280)
	preset.Canvas.Height = max(preset.Canvas.Height, 720)

	if preset.Background.Color == "" {
		preset.Background.Color = "#1a1a2e"
	}

	for i := range preset.Components {
		applyComponentDefaults(&preset.Components[i])
	}

	return &preset, nil
}
