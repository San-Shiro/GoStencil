# GoStencil -- Documentation

> Programmable media generation in pure Go. No CGo, no FFmpeg, no external dependencies.

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [CLI Reference](#cli-reference)
- [Web Editor](#web-editor)
  - [Launching](#launching)
  - [Editor Layout](#editor-layout)
  - [Toolbar](#toolbar)
  - [Asset Manager](#asset-manager)
  - [Commented Data Overrides](#commented-data-overrides)
  - [Help Modal](#help-modal)
  - [Exporting](#exporting)
  - [Typical Workflow](#typical-workflow)
- [Preset System](#preset-system)
  - [What is a Preset?](#what-is-a-preset)
  - [Using Presets](#using-presets)
  - [Creating Presets](#creating-presets)
  - [.gspresets Bundle Format](#gspresets-bundle-format)
  - [Component Reference](#component-reference)
  - [data.json Override Rules](#datajson-override-rules)
  - [Self-Documenting Schema](#self-documenting-schema)
- [Distribution](#distribution)
- [Simple Mode](#simple-mode)
- [Library Usage](#library-usage)
- [Canvas Presets](#canvas-presets)
- [Error Handling](#error-handling)

---

## Installation

```bash
git clone https://github.com/San-Shiro/GoStencil.git
cd GoStencil
go build -o gostencil ./cmd/gostencil
```

Or download a pre-built binary from [Releases](https://github.com/San-Shiro/GoStencil/releases).

Requires **Go 1.24+** (only for building from source).

---

## Quick Start

```bash
# Launch the web editor
gostencil serve --port 8080

# Or use the CLI
gostencil init
gostencil -o output.png --preset preset.json --data data.json
gostencil -o video.avi --preset preset.json --duration 5
```

---

## CLI Reference

### Generate from Preset

```
gostencil -o <file> --preset <path> [--data <path>] [--duration <sec>]
```

| Flag | Description | Default |
|------|-------------|---------|
| `-o`, `--output` | Output file path (`.png` or `.avi`) | required |
| `--preset` | Path to `.gspresets` bundle or standalone JSON | required |
| `--data` | Path to `data.json` for overrides | none |
| `--duration` | Video duration in seconds (AVI only) | `3` |

### Generate Solid Color

```
gostencil -o <file> --color <hex> [-w <px>] [-h <px>] [--duration <sec>]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--color` | Hex color `"#rrggbb"` or `"random"` | `random` |
| `-w`, `--width` | Width in pixels | `1280` |
| `-h`, `--height` | Height in pixels | `720` |

### Other Commands

```bash
gostencil init                          # Create sample preset.json + data.json
gostencil schema --preset theme.gspresets  # Print expected data.json format
gostencil serve --port 8080             # Launch web editor
```

---

## Web Editor

### Launching

```bash
gostencil serve --port 8080
```

The browser opens automatically. The entire web UI is embedded in the binary -- no internet connection needed.

### Editor Layout

The editor has 3 resizable panels:

```
+------------------+------------------+------------------+
|  preset.json     |  data.json       |  Live Preview    |
|  (template)      |  (overrides)     |                  |
|                  |                  |  [Fit] [1:1]     |
|  Define your     |  Override text,  |                  |
|  components,     |  visibility,     |  Updates on      |
|  styles, and     |  and styles      |  every edit      |
|  layout here     |  per component   |  (350ms delay)   |
+------------------+------------------+------------------+
```

- **Left panel** -- `preset.json`: The template definition (components, styles, canvas, background)
- **Center panel** -- `data.json`: Per-component overrides (text, visibility, style tweaks)
- **Right panel** -- Live preview that re-renders on every edit

Panels are resizable by dragging the dividers between them. Preview supports Fit-to-panel and 1:1 zoom.

### Toolbar

| Button | Action |
|--------|--------|
| **Import** | Load a `.gspresets` bundle. Extracts preset, imports assets, rebuilds data.json |
| **Font** | Upload a `.ttf` font file. Sets it as the global font in the preset |
| **Image** | Upload a PNG/JPG image. Makes it available in the assets panel |
| **Assets** | Opens the asset manager sidebar (see below) |
| **Help** | Opens a JSON reference modal with all fields, examples, and syntax |
| **Export** | Dropdown menu with PNG, AVI, preset.json, data.json, .gspresets options |

### Asset Manager

Click **Assets** in the toolbar to open the sidebar. Each uploaded asset shows:

- Preview thumbnail (images)
- Asset ID, file name, size
- Action buttons:

| Button | What it does |
|--------|-------------|
| **Copy ID** | Copies the asset ID to clipboard |
| **Copy BG Pair** | Copies `"backgroundImage": "ID", "backgroundFit": "contain"` -- paste into a component's `"style"` |
| **Copy fontPath** | Copies `"fontPath": "ID"` -- paste into a component's `"style"` for per-component font |
| **Global Font** | Sets the font as the global preset font (affects all components without a `fontPath`) |
| **Make Component** | Creates a new image component in preset.json with automatic unique ID, z-index, contain fit, and adds a commented entry in data.json |
| **Remove** | Deletes the asset from memory |

### Commented Data Overrides

The data.json panel auto-generates with `// ` prefixed keys for every component:

```json
{
  "components": {
    "// title": {
      "visible": true,
      "title": "Hello, GoStencil!",
      "items": [{ "type": "text", "text": "Edit this preset to get started" }],
      "style": { "fontSize": 48, "color": "#ffffff", "textAlign": "center" }
    },
    "// content": {
      "visible": true,
      "title": "Features",
      "items": [ ... ],
      "style": { "fontSize": 24, "color": "#cccccc", "textAlign": "left" }
    }
  }
}
```

**How it works:**
- Keys with `// ` prefix are **ignored** by the renderer -- they act as comments
- To **activate** an override: remove the `// ` prefix (`"// title"` becomes `"title"`)
- To **hide** a component: activate it and set `"visible": false`
- To **change text**: activate it and modify `title` or `items`
- To **tweak style**: activate it and change `fontSize`, `color`, `textAlign`, etc.

When you use **Make Component**, a matching commented entry is added to data.json automatically.

When you **Import** a `.gspresets` bundle, data.json is always rebuilt from the imported preset (data.json is never bundled inside `.gspresets`).

### Help Modal

Click **Help** in the toolbar to see a reference card with:
- Component position and size (x, y, width, height, zIndex, padding)
- Style properties (backgroundFit, fontPath, cornerRadius, etc.)
- Content structure (defaults, items, text types)
- Font setup (global vs per-component, fallback chain)
- Canvas presets (1080p, 4k, instagram, etc.)
- data.json override examples

### Exporting

The **Export** dropdown offers:

| Format | Description |
|--------|-------------|
| **PNG** | Rendered image at canvas resolution |
| **AVI** | MJPEG video (prompts for duration) |
| **preset.json** | The current preset definition (client-side download) |
| **data.json** | The current data overrides (client-side download) |
| **.gspresets** | ZIP bundle with preset.json + all uploaded assets (no data.json) |

JSON exports happen client-side (instant). PNG, AVI, and .gspresets exports go through the server.

### Typical Workflow

1. **Start fresh**: `gostencil serve` -- opens with a default preset
2. **Upload assets**: Click **Image** to upload backgrounds/logos, **Font** for custom fonts
3. **Design layout**: Edit `preset.json` -- add components, set positions (0.0--1.0), style them
4. **Use assets**: Open **Assets** panel, click **Copy BG Pair** or **Make Component**
5. **Preview**: Watch the right panel update in real-time as you edit
6. **Override content**: In `data.json`, remove `// ` prefix from components you want to customize
7. **Export**: Click **Export** -- choose PNG for images, AVI for video, .gspresets to share

**For teams/reuse**: Export as `.gspresets`, share the file. Recipients import it, get the same preset + assets, and customize via data.json.

---

## Preset System

### What is a Preset?

A preset is a complete visual template that defines:

- **Canvas** -- output dimensions or a named preset
- **Background** -- solid color or image
- **Font** -- custom TTF or embedded default
- **Components** -- positioned regions with style and default content
- **Schema** -- self-documentation for the expected data.json

### Using Presets

```bash
# Defaults only
gostencil -o output.png --preset theme.gspresets

# With overrides
gostencil -o output.png --preset theme.gspresets --data my_data.json

# As video
gostencil -o output.avi --preset theme.gspresets --duration 5
```

### Creating Presets

```json
{
  "meta": { "name": "My Theme", "version": "1.0", "author": "You" },
  "canvas": { "preset": "1080p" },
  "background": { "type": "color", "color": "#1a1a2e" },
  "font": { "path": "assets/fonts/MyFont.ttf" },
  "components": [
    {
      "id": "header",
      "x": 0.05, "y": 0.03, "width": 0.9, "height": 0.15,
      "zIndex": 10, "padding": 30,
      "style": {
        "backgroundColor": "#1a053380",
        "fontSize": 48, "color": "#00ffff", "textAlign": "center",
        "cornerRadius": 12, "borderColor": "#00ffff", "borderWidth": 2
      },
      "defaults": {
        "visible": true,
        "title": "WELCOME",
        "items": [{ "type": "text", "text": "Subtitle" }]
      }
    }
  ]
}
```

**Canvas options**: `{ "preset": "1080p" }` or `{ "width": 1920, "height": 1080 }`

**Background options**: `{ "type": "color", "color": "#0d0221" }` or `{ "type": "image", "source": "assets/bg.png", "color": "#0d0221" }`

### .gspresets Bundle Format

A `.gspresets` file is a ZIP archive:

```
mytheme.gspresets
+-- preset.json
+-- assets/
    +-- font_abc123.ttf
    +-- image_def456.png
```

- All asset paths in `preset.json` use asset IDs (resolved at runtime)
- **data.json is never included** -- it's always rebuilt from the preset on import
- Create manually: `zip -r mytheme.gspresets preset.json assets/`

### Component Reference

#### Position

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | Unique identifier (used as key in data.json) |
| `x`, `y` | `float` | Position as fraction of canvas (0.0--1.0) |
| `width`, `height` | `float` | Size as fraction of canvas (0.0--1.0) |
| `zIndex` | `int` | Render order: higher = on top |
| `padding` | `int` | Inner padding in pixels |

#### Style

| Property | Type | Description |
|----------|------|-------------|
| `backgroundColor` | `string` | `#rrggbb` or `#rrggbbaa` |
| `backgroundImage` | `string` | Asset ID or file path |
| `backgroundFit` | `string` | `stretch` (default), `contain`, `cover` |
| `fontPath` | `string` | Per-component font (overrides global) |
| `borderColor` | `string` | Border hex color |
| `borderWidth` | `int` | Border thickness (px) |
| `cornerRadius` | `int` | Rounded corners (px) |
| `fontSize` | `float` | Text size (points) |
| `color` | `string` | Text color hex |
| `lineHeight` | `float` | Line height multiplier |
| `textAlign` | `string` | `left`, `center`, `right` |

#### Background Fit Modes

| Mode | Behavior |
|------|----------|
| `stretch` | Fills component, may distort |
| `contain` | Fits inside without distortion (letterboxed) |
| `cover` | Fills component, crops excess |

#### Font Fallback Chain

1. `style.fontPath` (per-component)
2. `font.path` (global preset)
3. Embedded Go Regular (always available)

#### Text Item Types

| Type | Rendering |
|------|-----------|
| `text` | Plain paragraph |
| `bullet` | Prefixed with bullet |
| `numbered` | Prefixed with 1., 2., etc. |

### data.json Override Rules

| Field | Behavior |
|-------|----------|
| `visible` | `false` hides the component entirely |
| `title` | Replaces default title |
| `items` | **Replaces** (not appends) default items |
| `style.*` | Shallow merge onto preset style |

**Cannot override**: position (`x`, `y`, `width`, `height`) -- locked by preset.

**Merge behavior**: Omitted = use defaults. `visible: false` = skip entirely. Style = shallow merge. Items = replace.

### Self-Documenting Schema

```json
{
  "schema": {
    "description": "Data format for My Theme",
    "components": {
      "header": {
        "description": "Top banner",
        "fields": {
          "visible": "boolean",
          "title": "string",
          "items": "array of {type, text}"
        }
      }
    }
  }
}
```

```bash
gostencil schema --preset theme.gspresets
```

---

## Distribution

GoStencil compiles to a **single self-contained binary**. The web UI is embedded via `go:embed` -- no external files, no internet needed.

### GitHub Releases (Recommended -- Free)

The cheapest way to distribute:

1. **Cross-compile** for all platforms:
   ```bash
   GOOS=windows GOARCH=amd64 go build -o gostencil.exe ./cmd/gostencil
   GOOS=darwin  GOARCH=amd64 go build -o gostencil-mac ./cmd/gostencil
   GOOS=darwin  GOARCH=arm64 go build -o gostencil-mac-arm ./cmd/gostencil
   GOOS=linux   GOARCH=amd64 go build -o gostencil-linux ./cmd/gostencil
   ```

2. **Create a GitHub Release** and upload the binaries

3. Users **download once** and run:
   ```bash
   gostencil serve
   ```

**Cost: $0.** GitHub provides unlimited free releases for public repos.

### Why Not GitHub Pages / Static Hosting?

GoStencil requires a Go backend for:
- Server-side rendering (pixel-perfect output)
- Font loading and text metrics
- Image composition and scaling
- AVI video generation

A pure client-side version would require porting the entire renderer to WebAssembly -- significant effort for minimal benefit since the binary is already self-contained.

### For Teams

Export presets as `.gspresets` bundles and share them. Everyone runs the same binary locally with their own data overrides.

---

## Simple Mode

```bash
gostencil -o red.png --color "#ff0000" -w 1920 -h 1080
gostencil -o blue.avi --color "#0000ff" --duration 5
gostencil -o random.png --color random
```

---

## Library Usage

```go
import "github.com/San-Shiro/GoStencil/internal/template"

preset, cleanup, _ := template.LoadPreset("theme.gspresets")
defer cleanup()

data, _, _ := template.LoadData("data.json")
components := template.MergeData(preset, data)
renderer, _ := template.NewRenderer(preset.Font.Path)
img, _ := renderer.RenderPreset(preset, components)
template.SavePNG(img, "output.png")
```

---

## Canvas Presets

| Name | Dimensions |
|------|------------|
| `720p` | 1280 x 720 |
| `1080p` | 1920 x 1080 |
| `4k` | 3840 x 2160 |
| `instagram_square` | 1080 x 1080 |
| `instagram_story` | 1080 x 1920 |
| `youtube_thumb` | 1280 x 720 |

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| No data.json | Uses all preset defaults |
| Malformed data.json | Warning, uses defaults |
| Unknown component ID | Warning, ignored |
| Missing asset | Warning, skips image / fallback font |
| Invalid fontPath | Falls back to global font, then embedded |
| Corrupt ZIP | Fatal error |
| Invalid extension | Fatal error |
