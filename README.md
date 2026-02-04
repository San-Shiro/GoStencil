# GoStencil: Programmable Media Generation

**GoStencil** is a zero-dependency Go library for generating programmable media assets. It features a **JSON-driven template engine** to automate complex layouts and styling, paired with a **native AVI encoder** to generate widely compatible video files purely in Go—no external binaries required.

---

## 🚀 Features

- **Pure Go Implementation**: No external dependencies (like FFmpeg or ImageMagick) required.
- **Native AVI Generation**: Creates widely compatible AVI files using a built-in MJPEG encoder.
- **JSON Template Engine**:
  - **Layouts**: Define regions, margins, presets, and base styles.
  - **Content**: Populate text, lists, and headers dynamically.
  - **Auto-Formatting**: Automatic text wrapping, bullet points, and numbered lists.
  - **Styling**: Support for custom fonts (TTF), colors, and per-item style overrides.
- **Zero Configuration**: Comes with an embedded font (Go Regular) for immediate use.

---

## 📦 Installation

```powershell
# Clone and build
git clone https://github.com/xob0t/GoStencil
cd GoStencil
go build -o gostencil.exe ./cmd/media
```

---

## 🛠️ Quick Start

### 1. Initialize a Project
Create sample layout and content files to get started quickly.
```powershell
.\gostencil.exe init
```
This creates `layout.json` and `content.json` in your current directory.

### 2. Generate from Template
```powershell
# Generate a static PNG image
.\gostencil.exe template --layout layout.json --content content.json -o output.png

# Generate a 5-second AVI video (Pure Go)
.\gostencil.exe template --layout layout.json --content content.json -o output.avi --duration 5
```

### 3. Simple Image/Video Generation
Generate simple solid-color media without templates:
```powershell
# Generate a random color background video
.\gostencil.exe generate --type avi -o background.avi --duration 10

# Generate a specific color image
.\gostencil.exe generate --type png -o red.png --color "#ff0000" --width 1920 --height 1080
```

---

## 🎨 Layout Presets & Examples

The project includes a collection of professional presets in the `examples/` folder.

### included Presets (`examples/presets/`)
- **`cyberpunk.json`**: Dark purple (#0d0221) with cyan neon text.
- **`forest_green.json`**: Three-column layout with sidebar in nature tones.
- **`gradient_sunset.json`**: Warm orange (#ff6b35) modern design.
- **`minimal_light.json`**: Clean, professional light gray (#f5f5f5) theme.
- **`ocean_blue.json`**: Deep blue (#023e8a) with centered hero text.

### Example Usage
```powershell
# Use the cyberpunk preset
.\gostencil.exe template --layout examples/presets/cyberpunk.json --content examples/content_templates/welcome.json -o my_cyberpunk_video.avi
```

### Platform Presets
The `canvas.preset` field in your layout JSON automatically sets the resolution:
- `1080p`: 1920x1080
- `720p`: 1280x720
- `instagram_square`: 1080x1080
- `instagram_story`: 1080x1920
- `youtube_thumb`: 1280x720

---

## 📚 Library Usage

You can import `GoStencil` packages directly into your own Go applications used as a library.

### Installation
```bash
go get github.com/xob0t/GoStencil
```

### 1. Generating a Simple Video (AVI)
```go
package main

import (
    "log"
    "github.com/xob0t/GoStencil/pkg/generator"
)

func main() {
    // Create new AVI generator
    gen := generator.NewAVIGenerator()
    
    config := generator.Config{
        Width:    1920,
        Height:   1080,
        Duration: 5,         // seconds
        Color:    "#00ff00", // Green screen hex
    }

    if err := gen.Generate("output.avi", config); err != nil {
        log.Fatal(err)
    }
}
```

### 2. Rendering a Template
```go
package main

import (
    "log"
    "github.com/xob0t/GoStencil/pkg/generator"
    "github.com/xob0t/GoStencil/pkg/template"
)

func main() {
    // 1. Parse Layout and Content
    layout, err := template.ParseLayout("layout.json")
    if err != nil { panic(err) }
    
    content, err := template.ParseContent("content.json")
    if err != nil { panic(err) }

    // 2. Render to Image
    renderer, err := template.NewRenderer(layout.Font.Path)
    if err != nil { panic(err) }
    
    img, err := renderer.Render(layout, content)
    if err != nil { panic(err) }

    // 3. Save as AVI Video
    gen := generator.NewAVIGenerator()
    config := generator.Config{
        SourceImage: img,
        Duration:    10, // 10 seconds duration
    }
    
    if err := gen.Generate("render.avi", config); err != nil {
        log.Fatal(err)
    }
}
```

---

## 📄 Schema Reference

### Layout JSON (`layout.json`)
| Field | Description |
|-------|-------------|
| `canvas` | Defines `width`, `height`, or `preset`. |
| `background`| `type` ("color"/"image"), `color` (hex), `source` (file path). |
| `margin` | `top`, `bottom`, `left`, `right` padding for the safe area. |
| `font` | `path` to a `.ttf` file. Falls back to embedded font if empty. |
| `textboxes` | Array of regions. Coordinates `x`, `y`, `width`, `height` are 0.0 to 1.0. |

### Content JSON (`content.json`)
| Field | Description |
|-------|-------------|
| `textboxes` | Map of ID to content. |
| `title` | Optional header text for the region. |
| `items` | Array of objects with `type` ("text", "bullet", "numbered") and `text`. |

---

## ⌨️ Full CLI Options

### `template`
- `--layout <path>`: Layout specification (required).
- `--content <path>`: Content data (required).
- `-o, --output <path>`: Output file (`.png` or `.avi`).
- `--duration <sec>`: Video length (default: 3).

### `generate` (Simple Mode)
- `--type <avi|png|bmp>`: Output format (default: avi).
- `--color <hex>`: Solid background color.
- `--width/--height`: Manual dimensions (default: 1280x720).
- `--duration <sec>`: Video length (default: 1).
