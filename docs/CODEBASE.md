# GoStencil -- Codebase Guide

> Technical deep-dive into the GoStencil implementation. Covers architecture, package structure, data flow, and key algorithms.

---

## Table of Contents

- [Project Structure](#project-structure)
- [Architecture Overview](#architecture-overview)
- [Package: generator](#package-generator)
- [Package: template](#package-template)
  - [models.go -- Type System](#modelsgo----type-system)
  - [loader.go -- Preset Loading](#loadergo----preset-loading)
  - [merge.go -- Data Merging](#mergego----data-merging)
  - [validator.go -- Validation](#validatorgo----validation)
  - [renderer.go -- Rendering Engine](#renderergo----rendering-engine)
  - [fonts.go -- Font Management](#fontsgo----font-management)
- [Web UI: server.go](#web-ui-servergo)
- [CLI: main.go](#cli-maingo)
- [Library Usage](#library-usage)
- [Key Algorithms](#key-algorithms)

---

## Project Structure

```
GoStencil/
+-- cmd/gostencil/
|   +-- main.go              <- CLI entrypoint
|   +-- server.go            <- Web UI server + API endpoints
|   +-- web/                 <- Frontend (HTML/CSS/JS, embedded via go:embed)
|       +-- index.html       <- 3-panel editor + modals
|       +-- style.css        <- UI styles
|       +-- app.js           <- Editor logic, asset manager, exports
+-- internal/
|   +-- generator/           <- Media output (PNG, AVI)
|   |   +-- generator.go     <- Config + Generate() dispatcher
|   |   +-- color.go         <- ParseColor, ParseHexRGBA, NewSolidImage
|   |   +-- png.go           <- PNG encoder
|   |   +-- avi.go           <- MJPEG AVI writer
|   +-- template/            <- Preset system + rendering
|       +-- models.go        <- All type definitions
|       +-- loader.go        <- .gspresets ZIP extraction
|       +-- merge.go         <- Data overlay + z-index sorting
|       +-- validator.go     <- Validation + schema printer
|       +-- renderer.go      <- Image composition engine
|       +-- parser.go        <- Example JSON + standalone parser
|       +-- fonts.go         <- Font loading (TTF + embedded fallback)
+-- presets/                  <- Example .gspresets bundles
+-- docs/                    <- Documentation
```

> **Note:** Packages are in `internal/` which restricts import to the same module. See [Library Usage](#library-usage) for how to use GoStencil as a dependency in other Go apps.

---

## Architecture Overview

```
+------------------------------------------------------------+
|                      CLI (main.go)                          |
|   Parses flags -> mode: preset / solid / init / serve       |
+----------+----------------------+--------------------------+
           |                      |
    +------v------+        +------v------+
    |   template  |        |  generator  |
    |   package   |        |   package   |
    |             |        |             |
    | LoadPreset  |        | Generate()  |
    | LoadData    |        |   |- PNG    |
    | MergeData   |        |   +- AVI   |
    | RenderPreset|------->|             |
    |             | image  |             |
    +-------------+        +-------------+
           |
    +------v------+
    |  server.go  |   (serve mode only)
    |  Web UI     |
    |  HTTP API   |
    |  Assets     |
    +-------------+
```

**Data flow (CLI):**
1. Load `.gspresets` (ZIP) -> extract preset.json + assets
2. Optional data.json loaded and validated
3. Merge defaults + overrides -> `[]ResolvedComponent` (visible-only, z-sorted)
4. Render: background -> containers -> images -> text -> `*image.RGBA`
5. Output as PNG or AVI

**Data flow (Web UI):**
1. Frontend sends preset + data JSON to `/api/render`
2. Server resolves asset IDs to temp files
3. Same merge + render pipeline
4. Returns PNG bytes for live preview

---

## Package: generator

Media output. Format inferred from file extension.

```go
type Config struct {
    Width    int         // default: 1280
    Height   int         // default: 720
    Duration int         // seconds, AVI only
    Color    string      // "#rrggbb" or "random"
    Image    image.Image // pre-rendered; overrides Width/Height/Color
}

func Generate(output string, cfg Config) error
```

| File | Purpose |
|------|---------|
| `generator.go` | Config + `Generate()` dispatcher |
| `color.go` | `ParseColor`, `ParseHexRGBA`, `NewSolidImage` (uses `draw.Draw` for fast fill) |
| `png.go` | PNG encoder + `toRGBA()` conversion |
| `avi.go` | MJPEG AVI writer with `binaryWriter` error-capture pattern |

**AVI structure:** RIFF container with `hdrl` (headers), `movi` (JPEG frames at 15fps), `idx1` (frame index). Single JPEG encoded once, replicated for all frames.

---

## Package: template

### models.go -- Type System

| Type | Purpose |
|------|---------|
| `Preset` | Top-level: meta, canvas, background, font, components, schema |
| `Component` | Region: position (0.0--1.0), style, defaults, `zIndex` |
| `ComponentStyle` | Visual: bg color/image, `backgroundFit`, border, `fontPath`, text styling |
| `ComponentData` | Content: `*bool` Visible, Title, Items, Style override |
| `ResolvedComponent` | Final merged state with absolute pixel positions |

Key design decisions:
- `Visible` is `*bool` to distinguish "not set" (nil = inherit true) from "explicitly false"
- `backgroundFit`: `"stretch"` | `"contain"` | `"cover"`
- `fontPath`: per-component font override (asset ID or path)
- `zIndex`: render order (higher = on top)

### loader.go -- Preset Loading

`LoadPreset()` opens `.gspresets` ZIP, extracts to temp dir (with zip-slip protection), resolves paths, applies defaults, returns cleanup function.

`LoadData()` returns warnings (not errors) for malformed JSON -- graceful degradation.

### merge.go -- Data Merging

`MergeData()`:
1. Iterates preset components
2. Applies data overrides (visibility, title, items, style)
3. Filters invisible components
4. Resolves relative -> absolute pixel coordinates
5. **Sorts by zIndex** (ascending, stable sort)

Style merge is shallow: each non-zero override field replaces the preset value.

### validator.go -- Validation

Warns about unknown component IDs. Provides `FormatSchema()` for self-documenting presets.

### renderer.go -- Rendering Engine

```
RenderPreset()
  +-- drawPresetBackground()     <- solid color or image
  +-- for each component (z-sorted):
      +-- drawComponent()
          +-- drawContainer       <- bg color (alpha supported)
          |   +-- drawRect() or drawRoundedRect()
          +-- backgroundImage     <- based on backgroundFit:
          |   +-- drawScaled()    <- "stretch"
          |   +-- drawContain()   <- "contain" (letterbox)
          |   +-- drawCover()     <- "cover" (crop)
          +-- per-component font  <- fontPath -> global -> embedded
          +-- drawBorder()
          +-- drawComponentContent()
              +-- title (1.4x fontSize)
              +-- items (text/bullet/numbered, wrapped, aligned)
```

Key drawing primitives:
- **Rounded corners**: pixel-level distance check from corner centers
- **Alpha blending**: per-pixel `blendPixel()` for translucent containers
- **Text wrapping**: font-metric-based line breaking (not character count)
- **Text alignment**: `left`/`center`/`right` via measured string width
- **Image fit**: `contain` uses `min(scaleX, scaleY)`, `cover` uses `max(scaleX, scaleY)`

### fonts.go -- Font Management

`FontManager` wraps `opentype.Font`. Loads custom TTF via `opentype.Parse()`, falls back to embedded Go Regular. `GetFace(size, dpi)` returns a `font.Face`.

Used both globally (preset-level) and per-component (via `fontPath`).

---

## Web UI: server.go

### Server

```go
type server struct {
    assets *assetManager  // in-memory asset storage
    tmpDir string         // temp directory for resolved assets
}
```

Web frontend is embedded via `go:embed web/*` -- no external files needed.

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/render` | Render preset+data to PNG bytes |
| POST | `/api/export/png` | Download rendered PNG |
| POST | `/api/export/avi` | Download rendered AVI |
| POST | `/api/export/json` | Download preset or data JSON |
| POST | `/api/export/gspresets` | Download .gspresets bundle (no data.json) |
| POST | `/api/import/gspresets` | Import .gspresets bundle |
| POST | `/api/upload/font` | Upload font asset |
| POST | `/api/upload/image` | Upload image asset |
| GET | `/api/assets` | List all assets |
| GET | `/api/assets/{id}` | Serve asset by ID |
| DELETE | `/api/assets/{id}` | Remove asset |

### Asset Manager

In-memory storage keyed by random 16-char hex ID. Assets are resolved to temp files at render time, bundled into `.gspresets` on export.

### Frontend (web/)

| File | Key functions |
|------|---------------|
| `app.js` | `render()`, `handleExport()`, `buildDataTemplate()`, `handleAssetAction()` |
| `index.html` | 3-panel layout, toolbar, AVI modal, Help modal |
| `style.css` | Dark theme, asset panel, modal styles |

`buildDataTemplate()` generates data.json with `// ` prefixed keys for every preset component. Make Component adds entries under `d.components['// id']`.

---

## CLI: main.go

| Command | Function | Purpose |
|---------|----------|---------|
| (default) | `run()` | Generate from preset or solid color |
| `init` | `runInit()` | Create sample files |
| `schema` | `runSchema()` | Print preset schema |
| `serve` | `runServe()` | Launch web editor |

---

## Library Usage

### Current State

Packages are in `internal/`, restricting imports to this module only. To use GoStencil as a library in another Go app, the packages need to be moved to a public path.

**Option 1: Move to `pkg/`** (recommended for this project):
```
internal/generator/ -> pkg/generator/
internal/template/  -> pkg/template/
```

**Option 2: Top-level packages**:
```
internal/generator/ -> generator/
internal/template/  -> template/
```

After moving, external Go apps can import:

```go
import (
    "github.com/San-Shiro/GoStencil/pkg/generator"
    "github.com/San-Shiro/GoStencil/pkg/template"
)

// Generate a solid-color image
cfg := generator.Config{Width: 1920, Height: 1080, Color: "#ff0000"}
generator.Generate("output.png", cfg)

// Render a preset
preset, cleanup, _ := template.LoadPreset("theme.gspresets")
defer cleanup()
data, _, _ := template.LoadData("data.json")
components := template.MergeData(preset, data)
renderer, _ := template.NewRenderer(preset.Font.Path)
img, _ := renderer.RenderPreset(preset, components)
template.SavePNG(img, "output.png")

// Render to AVI
cfg = generator.Config{Image: img, Duration: 5}
generator.Generate("output.avi", cfg)
```

### In-Module Usage (Current)

Within the GoStencil module itself:

```go
import (
    "github.com/San-Shiro/GoStencil/internal/generator"
    "github.com/San-Shiro/GoStencil/internal/template"
)
```

---

## Key Algorithms

| Algorithm | Location | Description |
|-----------|----------|-------------|
| Z-index sort | `merge.go` | `sort.SliceStable` by zIndex (preserves order for equal values) |
| Contain fit | `renderer.go` | `scale = min(scaleX, scaleY)`, center in bounds |
| Cover fit | `renderer.go` | `scale = max(scaleX, scaleY)`, crop excess |
| Relative coords | `merge.go` | `int(comp.X * float64(canvasWidth))` |
| Visibility gate | `merge.go` | `visible=false` excluded before rendering |
| Font fallback | `renderer.go` | fontPath -> global -> embedded Go Regular |
| Text wrap | `renderer.go` | Font metric width check per word |
| Alpha blend | `renderer.go` | Per-pixel `(src*a + dst*(255-a))/255` |
| Rounded corners | `renderer.go` | Distance from corner center vs radius |
