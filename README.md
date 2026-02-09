# GoStencil

Programmable media generation in pure Go. JSON-driven templates, native AVI encoding, built-in web editor.

**No CGo. No FFmpeg. No external dependencies.**

**🌐 [Try it in your browser](https://san-shiro.github.io/GoStencil/)** — 100% client-side via WebAssembly, no installation needed.

---

## Quick Links

| | |
|---|---|
| [Documentation](docs/DOCUMENTATION.md) | Full usage guide, web editor reference, CLI options |
| [Codebase Guide](docs/CODEBASE.md) | Architecture, package deep-dive, algorithms |

---

## Features

- **Preset System** — JSON-defined templates with components, styling, and canvas presets
- **Pure Go Rendering** — Text layout, background images, rounded corners, border, alpha blending
- **Native AVI Encoding** — MJPEG video output, no external tools
- **Web Editor** — Three-panel UI with live preview, asset manager, import/export
- **WASM Client** — 100% client-side rendering via WebAssembly (no server needed)
- **Go Library** — Import `pkg/generator` and `pkg/template` directly in your Go apps
- **Zero Dependencies** — Only stdlib + `golang.org/x/image` for font rendering

---

## Project Structure

```
GoStencil/
├── cmd/gostencil/main.go          # CLI entry point
├── pkg/
│   ├── generator/                 # PNG/AVI generation (importable)
│   └── template/                  # Preset rendering engine (importable)
├── clients/
│   ├── server/                    # HTTP server + embedded web UI
│   │   ├── server.go              # Exported RunServe()
│   │   └── web/                   # Frontend assets
│   └── wasm/                      # WebAssembly client
│       ├── main.go                # WASM entry point (syscall/js)
│       └── web/                   # Deployable static frontend
├── scripts/build_wasm.bat         # WASM build script
├── presets/                       # Example .gspresets bundles
└── docs/                          # Documentation
```

---

## Quick Start

### Build & Run

```bash
# Build CLI
go build -o gostencil ./cmd/gostencil

# Start web editor
gostencil serve --port 8080

# Generate from preset
gostencil -o output.png --preset theme.gspresets --data data.json

# Initialize sample files
gostencil init
```

### Use as a Go Library

Since packages live in `pkg/`, you can import them directly:

```bash
go get github.com/xob0t/GoStencil@latest
```

```go
import (
    "github.com/xob0t/GoStencil/pkg/generator"
    "github.com/xob0t/GoStencil/pkg/template"
)

// Generate a solid-color AVI
cfg := generator.Config{Width: 1920, Height: 1080, Duration: 5, Color: "#ff6600"}
generator.Generate("output.avi", cfg)

// Render a preset
preset, cleanup, _ := template.LoadPreset("theme.gspresets")
defer cleanup()
data, _, _ := template.LoadData("data.json")
components := template.MergeData(preset, data)
renderer, _ := template.NewRenderer("")
img, _ := renderer.RenderPreset(preset, components)
template.SavePNG(img, "output.png")
```

---

## WASM Client (Client-Side Only)

The WASM client runs GoStencil entirely in the browser — no server, no backend. After the initial ~5 MB download, everything is cached and works offline.

### Build WASM

```bash
# Windows
scripts\build_wasm.bat

# Linux/macOS
GOOS=js GOARCH=wasm go build -o clients/wasm/web/gostencil.wasm ./clients/wasm/
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" clients/wasm/web/
```

### Deploy

The `clients/wasm/web/` folder is fully self-contained. Deploy it to:
- **GitHub Pages** — push the folder, enable Pages in repo settings
- **Any static host** — Netlify, Cloudflare Pages, S3, etc.
- **Local** — `python -m http.server 8080` in the `web/` folder

### How It Works

```
Browser                          WASM (Go)
┌──────────┐    JSON strings    ┌──────────────┐
│  app.js  │ ──────────────────▶│  main.go     │
│  (UI)    │◀────────────────── │  (renderer)  │
│          │    base64 PNG/AVI  │              │
└──────────┘                    └──────────────┘

No HTTP. No server. Just function calls via syscall/js.
```

---

## Distribution

### Pre-built Binaries (Recommended)

Cross-compile for all platforms from any OS:

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o gostencil-windows-amd64.exe ./cmd/gostencil

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o gostencil-darwin-arm64 ./cmd/gostencil

# Linux
GOOS=linux GOARCH=amd64 go build -o gostencil-linux-amd64 ./cmd/gostencil
```

Upload to **GitHub Releases** — users download once, run locally. Zero hosting cost.

### Why Not GitHub Pages for the Server?

GoStencil's server mode needs a Go backend for rendering. The **WASM client** solves this — it runs the same Go renderer in the browser via WebAssembly, making it deployable on any static host including GitHub Pages.

---

## Web Editor Workflow

```
┌────────────────┐    ┌────────────────┐    ┌────────────────┐
│  preset.json   │    │   data.json    │    │    Preview     │
│  (template)    │───▶│  (overrides)   │───▶│  (live render) │
│                │    │                │    │                │
│  Components,   │    │  Uncomment     │    │  Auto-updates  │
│  styling,      │    │  keys to       │    │  as you type   │
│  canvas size   │    │  override      │    │                │
└────────────────┘    └────────────────┘    └────────────────┘

Toolbar: Import | Font | Image | Assets | Help | Export ▾
```

---

## CLI Reference

```
gostencil -o <file> --preset <path> [--data <path>] [--duration N]
gostencil -o <file> --color <hex> [-w N] [-h N] [--duration N]
gostencil serve [--port 8080]
gostencil schema --preset <path>
gostencil init
```

---

## License

See [LICENSE](LICENSE).
