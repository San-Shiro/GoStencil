// Package generator provides cover media generation for steganography.
package generator

import "image"

// Config holds configuration for media generation.
type Config struct {
	Width       int
	Height      int
	Duration    int         // Seconds (for video)
	Color       string      // Hex color or "random"
	Text        string      // Optional text overlay
	SourceImage image.Image // Pre-rendered image for templates
}

// Generator is the interface for media generators.
type Generator interface {
	Generate(output string, config Config) error
}
