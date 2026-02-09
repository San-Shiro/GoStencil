// Package template provides JSON-driven image generation via presets and components.
package template

// ── Preset types ──

// Preset is the top-level structure of a preset.json file.
type Preset struct {
	Meta       Meta        `json:"meta"`
	Canvas     Canvas      `json:"canvas"`
	Background Background  `json:"background"`
	Font       FontConfig  `json:"font"`
	Components []Component `json:"components"`
	Schema     Schema      `json:"schema"`
}

// Meta holds preset metadata.
type Meta struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
}

// Canvas defines output dimensions. Preset overrides explicit Width/Height.
type Canvas struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Preset string `json:"preset"`
}

// Background defines the canvas fill.
type Background struct {
	Type   string `json:"type"`   // "image" or "color"
	Source string `json:"source"` // path to image file (resolved from assets)
	Color  string `json:"color"`  // hex fallback
}

// FontConfig specifies the font source.
type FontConfig struct {
	Path     string `json:"path"`     // custom TTF path (resolved from assets)
	Fallback string `json:"fallback"` // "embedded" for default
}

// ── Component types ──

// Component defines a renderable div-like region in the preset.
// Position (X/Y/Width/Height) is immutable — data.json cannot override it.
type Component struct {
	ID       string         `json:"id"`
	X        float64        `json:"x"`      // relative 0.0–1.0
	Y        float64        `json:"y"`      // relative 0.0–1.0
	Width    float64        `json:"width"`  // relative 0.0–1.0
	Height   float64        `json:"height"` // relative 0.0–1.0
	ZIndex   int            `json:"zIndex"` // rendering order (higher = on top)
	Padding  int            `json:"padding"`
	Style    ComponentStyle `json:"style"`
	Defaults ComponentData  `json:"defaults"`
}

// ComponentStyle defines the visual appearance of a component container.
type ComponentStyle struct {
	BackgroundColor string  `json:"backgroundColor"` // "#rrggbb" or "#rrggbbaa"
	BackgroundImage string  `json:"backgroundImage"` // path to PNG/JPG sticker
	BackgroundFit   string  `json:"backgroundFit"`   // "stretch" (default), "contain", "cover"
	BorderColor     string  `json:"borderColor"`
	BorderWidth     int     `json:"borderWidth"`
	CornerRadius    int     `json:"cornerRadius"`
	FontPath        string  `json:"fontPath"` // per-component custom font (asset ID or path)
	FontSize        float64 `json:"fontSize"`
	Color           string  `json:"color"`      // text color
	LineHeight      float64 `json:"lineHeight"` // multiplier
	TextAlign       string  `json:"textAlign"`  // "left", "center", "right"
}

// ComponentData holds the content and visibility for a component.
// Used both as defaults in preset.json and as overrides in data.json.
type ComponentData struct {
	Visible *bool           `json:"visible,omitempty"` // nil = inherit default (true)
	Title   string          `json:"title,omitempty"`
	Items   []TextItem      `json:"items,omitempty"`
	Style   *ComponentStyle `json:"style,omitempty"` // per-component style override
}

// TextItem defines a single text entry within a component.
type TextItem struct {
	Type string `json:"type"` // "text", "bullet", "numbered"
	Text string `json:"text"`
}

// ── Data types ──

// DataSpec is the top-level structure of data.json.
type DataSpec struct {
	Components map[string]ComponentData `json:"components"`
}

// ── Schema types (self-documenting presets) ──

// Schema documents the expected data.json format for this preset.
type Schema struct {
	Description string                     `json:"description"`
	Components  map[string]SchemaComponent `json:"components"`
}

// SchemaComponent documents one component's editable fields.
type SchemaComponent struct {
	Description string            `json:"description"`
	Fields      map[string]string `json:"fields"` // field name → description
}

// ── Resolved types (after merging defaults + data) ──

// ResolvedComponent is a component ready for rendering with final values.
type ResolvedComponent struct {
	ID      string
	X, Y    int // absolute pixels
	Width   int
	Height  int
	ZIndex  int
	Padding int
	Style   ComponentStyle
	Data    ComponentData
}

// ── Presets for common resolutions ──

// Presets maps preset names to [width, height].
var Presets = map[string][2]int{
	"720p":             {1280, 720},
	"1080p":            {1920, 1080},
	"4k":               {3840, 2160},
	"instagram_square": {1080, 1080},
	"instagram_story":  {1080, 1920},
	"youtube_thumb":    {1280, 720},
}

// ── Legacy support ──

// Margin defines spacing around the content area (used by legacy layout mode).
type Margin struct {
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
}

// Style is the legacy text style (used internally for font face creation).
type Style struct {
	FontSize   float64 `json:"fontSize"`
	Color      string  `json:"color"`
	LineHeight float64 `json:"lineHeight"`
}
