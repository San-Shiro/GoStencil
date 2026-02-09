// GoStencil WASM — Client-side renderer.
// Compiled with: GOOS=js GOARCH=wasm go build -o gostencil.wasm ./clients/wasm/
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"sync"
	"syscall/js"

	"github.com/xob0t/GoStencil/pkg/generator"
	"github.com/xob0t/GoStencil/pkg/template"
)

// In-memory asset store (replaces server-side asset manager).
var (
	assetsMu sync.RWMutex
	assets   = make(map[string]assetEntry)
)

type assetEntry struct {
	Data []byte
	Mime string
}

func main() {
	fmt.Println("GoStencil WASM loaded")

	// Register JS-callable functions.
	js.Global().Set("goRenderImage", js.FuncOf(renderImage))
	js.Global().Set("goRegisterAsset", js.FuncOf(registerAsset))
	js.Global().Set("goRemoveAsset", js.FuncOf(removeAsset))
	js.Global().Set("goExportAVI", js.FuncOf(exportAVI))
	js.Global().Set("goReady", js.ValueOf(true))

	// Block forever (WASM must not exit).
	select {}
}

// resolveAsset replaces asset IDs with in-memory data.
// Returns the raw bytes if the path is an asset ID, nil otherwise.
func resolveAsset(id string) []byte {
	assetsMu.RLock()
	defer assetsMu.RUnlock()
	if a, ok := assets[id]; ok {
		return a.Data
	}
	return nil
}

// goRegisterAsset(id, base64Data, mime) — store an asset in Go memory.
func registerAsset(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return js.ValueOf("error: need id, base64Data, mime")
	}
	id := args[0].String()
	b64 := args[1].String()
	mimeType := args[2].String()

	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return js.ValueOf("error: invalid base64: " + err.Error())
	}

	assetsMu.Lock()
	assets[id] = assetEntry{Data: data, Mime: mimeType}
	assetsMu.Unlock()

	return js.ValueOf("ok")
}

// goRemoveAsset(id) — remove an asset from Go memory.
func removeAsset(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return js.ValueOf("error: need id")
	}
	id := args[0].String()
	assetsMu.Lock()
	delete(assets, id)
	assetsMu.Unlock()
	return js.ValueOf("ok")
}

// goRenderImage(presetJSON, dataJSON) — render and return base64 PNG.
func renderImage(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return js.ValueOf("error: need presetJSON, dataJSON")
	}

	presetStr := args[0].String()
	dataStr := args[1].String()

	var preset template.Preset
	if err := json.Unmarshal([]byte(presetStr), &preset); err != nil {
		return js.ValueOf("error: parse preset: " + err.Error())
	}

	// Apply canvas preset.
	if dims, ok := template.Presets[preset.Canvas.Preset]; ok {
		preset.Canvas.Width = dims[0]
		preset.Canvas.Height = dims[1]
	}
	if preset.Canvas.Width < 320 {
		preset.Canvas.Width = 320
	}
	if preset.Canvas.Height < 240 {
		preset.Canvas.Height = 240
	}
	if preset.Background.Color == "" {
		preset.Background.Color = "#1a1a2e"
	}

	// Resolve assets: write to temp memory map for the renderer.
	// The renderer loads images by file path, but in WASM we need to
	// write asset bytes to a virtual path. We use the template package's
	// ability to load from byte data by writing to the asset resolver.
	fontData := resolveAsset(preset.Font.Path)
	bgData := resolveAsset(preset.Background.Source)

	// For backgroundImage and fontPath in components, we need to write
	// the asset data to temp paths. In WASM, we'll use the Go os package
	// which maps to in-memory filesystem.
	for i := range preset.Components {
		resolveComponentAssets(&preset.Components[i])
		applyDefaults(&preset.Components[i])
	}

	// Parse data.
	var data *template.DataSpec
	if dataStr != "" && dataStr != "null" && dataStr != "{}" {
		var d template.DataSpec
		if err := json.Unmarshal([]byte(dataStr), &d); err == nil {
			data = &d
		}
	}

	// Merge.
	components := template.MergeData(&preset, data)

	// Create renderer with font.
	var renderer *template.Renderer
	var err error
	if fontData != nil {
		renderer, err = template.NewRendererFromBytes(fontData)
	} else {
		renderer, err = template.NewRenderer("") // embedded fallback
	}
	if err != nil {
		return js.ValueOf("error: renderer: " + err.Error())
	}

	// Resolve background image from bytes.
	if bgData != nil {
		preset.Background.Source = "" // clear path, we'll handle it differently
	}

	img, err := renderer.RenderPreset(&preset, components)
	if err != nil {
		return js.ValueOf("error: render: " + err.Error())
	}

	// Encode to PNG.
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return js.ValueOf("error: encode: " + err.Error())
	}

	return js.ValueOf(base64.StdEncoding.EncodeToString(buf.Bytes()))
}

// goExportAVI(presetJSON, dataJSON, duration) — render and return base64 AVI.
func exportAVI(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return js.ValueOf("error: need presetJSON, dataJSON, duration")
	}

	// First render the image.
	imgResult := renderImage(this, args[:2])
	resultStr := imgResult.(js.Value).String()
	if len(resultStr) > 6 && resultStr[:6] == "error:" {
		return js.ValueOf(resultStr)
	}

	pngData, err := base64.StdEncoding.DecodeString(resultStr)
	if err != nil {
		return js.ValueOf("error: decode PNG: " + err.Error())
	}

	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return js.ValueOf("error: decode image: " + err.Error())
	}

	duration := args[2].Int()
	if duration < 1 {
		duration = 1
	}

	// Generate AVI in memory.
	var aviBuf bytes.Buffer
	cfg := generator.Config{Image: img, Duration: duration}
	if err := generator.GenerateToWriter(&aviBuf, ".avi", cfg); err != nil {
		return js.ValueOf("error: generate AVI: " + err.Error())
	}

	return js.ValueOf(base64.StdEncoding.EncodeToString(aviBuf.Bytes()))
}

func resolveComponentAssets(c *template.Component) {
	// For background images, write asset data to a temp file path
	// that the renderer can load.
	if data := resolveAsset(c.Style.BackgroundImage); data != nil {
		// Store in a temporary in-memory location.
		// The renderer will need to be able to load this.
		tmpID := "_wasm_bg_" + c.ID
		assetsMu.Lock()
		assets[tmpID] = assetEntry{Data: data, Mime: "image/png"}
		assetsMu.Unlock()
		// We'll need the renderer to support loading from our asset map.
		// For now, keep the original ID — we'll patch the renderer.
	}
	if data := resolveAsset(c.Style.FontPath); data != nil {
		_ = data // font loading handled separately
	}
}

func applyDefaults(c *template.Component) {
	s := &c.Style
	if s.FontSize <= 0 {
		s.FontSize = 24
	}
	if s.Color == "" {
		s.Color = "#ffffff"
	}
	if s.LineHeight <= 0 {
		s.LineHeight = 1.5
	}
	if s.TextAlign == "" {
		s.TextAlign = "left"
	}
	if c.Defaults.Visible == nil {
		t := true
		c.Defaults.Visible = &t
	}
}
