// GoStencil — Programmable media generation.
//
// Usage:
//
//	gostencil -o <file> --preset <path> [--data <path>] [options]
//	gostencil schema --preset <path>
//	gostencil serve [--port 8080]
//	gostencil init
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xob0t/GoStencil/clients/server"
	"github.com/xob0t/GoStencil/pkg/generator"
	"github.com/xob0t/GoStencil/pkg/template"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		if err := runInit(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "schema":
		if err := runSchema(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "serve":
		if err := server.RunServe(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		// Default: generate mode (all flags on root).
		if err := run(os.Args[1:]); err != nil {
			fatal(err)
		}
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("gostencil", flag.ExitOnError)

	var (
		output     string
		presetPath string
		dataPath   string
		width      int
		height     int
		duration   int
		color      string
	)

	fs.StringVar(&output, "o", "", "Output file path (.png or .avi)")
	fs.StringVar(&output, "output", "", "Output file path (.png or .avi)")
	fs.StringVar(&presetPath, "preset", "", "Path to .gspresets bundle or preset JSON")
	fs.StringVar(&dataPath, "data", "", "Path to data.json (optional)")
	fs.IntVar(&width, "w", 1280, "Width in pixels")
	fs.IntVar(&width, "width", 1280, "Width in pixels")
	fs.IntVar(&height, "h", 720, "Height in pixels")
	fs.IntVar(&height, "height", 720, "Height in pixels")
	fs.IntVar(&duration, "duration", 3, "Duration in seconds (AVI only)")
	fs.StringVar(&color, "color", "random", "Background color: hex or 'random'")

	fs.Usage = printUsage
	if err := fs.Parse(args); err != nil {
		return err
	}

	if output == "" {
		printUsage()
		return fmt.Errorf("output file is required (-o)")
	}

	// Preset mode.
	if presetPath != "" {
		return runPreset(presetPath, dataPath, output, duration)
	}

	// Simple solid-color mode.
	cfg := generator.Config{
		Width:    width,
		Height:   height,
		Duration: duration,
		Color:    color,
	}

	fmt.Printf("Generating: %s\n", output)
	if err := generator.Generate(output, cfg); err != nil {
		return err
	}
	fmt.Printf("Done: %s\n", output)
	return nil
}

func runPreset(presetPath, dataPath, output string, duration int) error {
	// Load preset.
	var preset *template.Preset
	var cleanup func()
	var err error

	ext := strings.ToLower(filepath.Ext(presetPath))
	switch ext {
	case ".gspresets":
		preset, cleanup, err = template.LoadPreset(presetPath)
		if err != nil {
			return fmt.Errorf("load preset: %w", err)
		}
		defer cleanup()
	default:
		// Treat as standalone JSON.
		preset, err = template.ParsePresetFile(presetPath)
		if err != nil {
			return fmt.Errorf("load preset: %w", err)
		}
	}

	// Load data (optional).
	var data *template.DataSpec
	if dataPath != "" {
		var warnings []string
		data, warnings, err = template.LoadData(dataPath)
		if err != nil {
			return fmt.Errorf("load data: %w", err)
		}
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
		}

		// Validate.
		for _, w := range template.ValidateData(data, preset) {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
		}
	}

	// Merge defaults + data → resolved components.
	components := template.MergeData(preset, data)

	// Render.
	renderer, err := template.NewRenderer(preset.Font.Path)
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	fmt.Printf("Rendering preset: %s\n", preset.Meta.Name)
	img, err := renderer.RenderPreset(preset, components)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	// Output.
	cfg := generator.Config{
		Image:    img,
		Duration: duration,
	}

	if err := generator.Generate(output, cfg); err != nil {
		return err
	}
	fmt.Printf("Done: %s\n", output)
	return nil
}

func runSchema(args []string) error {
	fs := flag.NewFlagSet("schema", flag.ExitOnError)
	var presetPath string
	fs.StringVar(&presetPath, "preset", "", "Path to .gspresets or preset JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if presetPath == "" {
		return fmt.Errorf("--preset is required for schema command")
	}

	var preset *template.Preset
	var err error

	ext := strings.ToLower(filepath.Ext(presetPath))
	switch ext {
	case ".gspresets":
		var cleanup func()
		preset, cleanup, err = template.LoadPreset(presetPath)
		if err != nil {
			return err
		}
		defer cleanup()
	default:
		preset, err = template.ParsePresetFile(presetPath)
		if err != nil {
			return err
		}
	}

	fmt.Print(template.FormatSchema(preset))
	return nil
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	var presetOut, dataOut string
	fs.StringVar(&presetOut, "preset", "preset.json", "Output path for sample preset")
	fs.StringVar(&dataOut, "data", "data.json", "Output path for sample data")
	if err := fs.Parse(args); err != nil {
		return err
	}

	p, d := template.GetExampleJSON()

	if err := os.WriteFile(presetOut, []byte(p), 0644); err != nil {
		return fmt.Errorf("write preset: %w", err)
	}
	if err := os.WriteFile(dataOut, []byte(d), 0644); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	fmt.Printf("Created: %s, %s\n", presetOut, dataOut)
	fmt.Println("Run: gostencil -o output.png --preset preset.json --data data.json")
	return nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

func printUsage() {
	fmt.Print(`GoStencil — Programmable Media Generation (Pure Go)

USAGE:
    gostencil -o <file> --preset <path> [--data <path>] [options]
    gostencil -o <file> --color <hex> [options]
    gostencil schema --preset <path>
    gostencil serve [--port 8080]
    gostencil init [options]

PRESET MODE:
    --preset <path>        .gspresets bundle or standalone preset JSON
    --data <path>          Data JSON with overrides (optional)
    -o, --output <path>    Output file (.png or .avi)
    --duration <sec>       Video duration in seconds (default: 3)

SIMPLE MODE:
    -o, --output <path>    Output file (.png or .avi)
    --color <hex>          Background color or 'random' (default: random)
    -w, --width <px>       Width in pixels (default: 1280)
    -h, --height <px>      Height in pixels (default: 720)
    --duration <sec>       Video duration (default: 3)

UI SERVER:
    gostencil serve [--port 8080]       Start the web UI editor

SCHEMA:
    gostencil schema --preset <path>    Print preset's data.json format

EXAMPLES:
    gostencil init
    gostencil serve
    gostencil -o card.png --preset theme.gspresets
    gostencil -o card.png --preset theme.gspresets --data data.json
    gostencil -o video.avi --preset theme.gspresets --duration 5
    gostencil schema --preset theme.gspresets
    gostencil -o solid.png --color "#ff0000" -w 1920 -h 1080
`)
}
