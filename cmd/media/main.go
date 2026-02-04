// go-media - Cover media generator for steganography
//
// Usage:
//
//	go-media generate --type mp4|bmp --output <file> [options]
//	go-media template --layout <json> --content <json> -o <file> [options]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xob0t/GoStencil/pkg/generator"
	"github.com/xob0t/GoStencil/pkg/template"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate", "gen":
		if err := runGenerate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "template", "tmpl":
		if err := runTemplate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "init":
		if err := runInit(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)

	var (
		mediaType string
		output    string
		width     int
		height    int
		duration  int
		color     string
	)

	fs.StringVar(&mediaType, "type", "avi", "Media type: avi, bmp, png")
	fs.StringVar(&output, "o", "", "Output file path")
	fs.StringVar(&output, "output", "", "Output file path")
	fs.IntVar(&width, "width", 1280, "Width in pixels")
	fs.IntVar(&height, "height", 720, "Height in pixels")
	fs.IntVar(&duration, "duration", 1, "Duration in seconds (video only)")
	fs.StringVar(&color, "color", "random", "Background color (hex or 'random')")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if output == "" {
		return fmt.Errorf("output file is required (-o)")
	}

	config := generator.Config{
		Width:    width,
		Height:   height,
		Duration: duration,
		Color:    color,
	}

	var gen generator.Generator
	switch strings.ToLower(mediaType) {
	case "avi", "video":
		gen = generator.NewAVIGenerator()
	case "bmp", "image":
		gen = generator.NewBMPGenerator()
	case "png":
		gen = generator.NewPNGGenerator()
	default:
		return fmt.Errorf("invalid media type: %s (use: avi, bmp, png)", mediaType)
	}

	fmt.Printf("Generating %s media: %s\n", mediaType, output)
	if err := gen.Generate(output, config); err != nil {
		return err
	}

	fmt.Printf("Successfully created: %s\n", output)
	return nil
}

func runTemplate(args []string) error {
	fs := flag.NewFlagSet("template", flag.ExitOnError)

	var (
		layoutPath  string
		contentPath string
		output      string
		duration    int
	)

	fs.StringVar(&layoutPath, "layout", "", "Path to layout JSON file")
	fs.StringVar(&contentPath, "content", "", "Path to content JSON file")
	fs.StringVar(&output, "o", "", "Output file path (.png or .avi)")
	fs.StringVar(&output, "output", "", "Output file path (.png or .avi)")
	fs.IntVar(&duration, "duration", 3, "Duration in seconds (video only)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if layoutPath == "" {
		return fmt.Errorf("layout file is required (--layout)")
	}
	if contentPath == "" {
		return fmt.Errorf("content file is required (--content)")
	}
	if output == "" {
		return fmt.Errorf("output file is required (-o)")
	}

	// Parse JSON files
	layout, err := template.ParseLayout(layoutPath)
	if err != nil {
		return fmt.Errorf("failed to parse layout: %w", err)
	}

	content, err := template.ParseContent(contentPath)
	if err != nil {
		return fmt.Errorf("failed to parse content: %w", err)
	}

	// Validate mapping
	if err := template.ValidateMapping(layout, content); err != nil {
		return err
	}

	// Create renderer
	renderer, err := template.NewRenderer(layout.Font.Path)
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}

	// Render image
	fmt.Printf("Rendering template: %s + %s\n", layoutPath, contentPath)
	img, err := renderer.Render(layout, content)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	// Determine output type
	ext := strings.ToLower(filepath.Ext(output))
	switch ext {
	case ".png":
		if err := template.SavePNG(img, output); err != nil {
			return err
		}
	case ".avi":
		gen := generator.NewAVIGenerator()
		config := generator.Config{
			SourceImage: img,
			Duration:    duration,
		}
		if err := gen.Generate(output, config); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s (use .png or .avi)", ext)
	}

	fmt.Printf("Successfully created: %s\n", output)
	return nil
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	var (
		layoutOutput  string
		contentOutput string
	)
	fs.StringVar(&layoutOutput, "layout", "layout.json", "Output path for sample layout JSON")
	fs.StringVar(&contentOutput, "content", "content.json", "Output path for sample content JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	l, c := template.GetExampleJSON()

	if err := os.WriteFile(layoutOutput, []byte(l), 0644); err != nil {
		return fmt.Errorf("failed to write layout: %w", err)
	}
	if err := os.WriteFile(contentOutput, []byte(c), 0644); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	fmt.Printf("Created sample files: %s, %s\n", layoutOutput, contentOutput)
	fmt.Println("You can now run: gostencil template --layout layout.json --content content.json -o output.png")
	return nil
}

func printUsage() {
	fmt.Println(`GoStencil - Programmable Media Generation (Pure Go)

USAGE:
    gostencil generate [options]
    gostencil template [options]
    gostencil init [options]

GENERATE OPTIONS:
    --type <type>        Media type: avi, bmp, png (default: avi)
    -o, --output <path>  Output file path
    --width <pixels>     Width in pixels (default: 1280)
    --height <pixels>    Height in pixels (default: 720)
    --duration <seconds> Duration in seconds, video only (default: 1)
    --color <color>      Background color: hex or 'random' (default: random)

TEMPLATE OPTIONS:
    --layout <path>      Path to layout JSON file
    --content <path>     Path to content JSON file
    -o, --output <path>  Output file path (.png or .avi)
    --duration <seconds> Duration in seconds, video only (default: 3)

INIT OPTIONS:
    --layout <path>      Output path for sample layout (default: layout.json)
    --content <path>     Output path for sample content (default: content.json)

EXAMPLES:
    gostencil init
    gostencil template --layout layout.json --content content.json -o output.png
    gostencil template --layout layout.json --content content.json -o cover.avi
    gostencil generate --type avi -o cover.avi --duration 2 --color "#1a1a2e"`)
}
