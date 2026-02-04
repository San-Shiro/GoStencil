// bmp.go - Pure Go BMP image generator implementing 24-bit uncompressed bitmap format.
// Manually constructs BMP file headers (BITMAPFILEHEADER + BITMAPINFOHEADER) and
// handles BGR pixel ordering as required by the BMP specification.
package generator

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// BMPGenerator generates BMP image files using pure Go.
type BMPGenerator struct{}

// NewBMPGenerator creates a new BMP generator.
func NewBMPGenerator() *BMPGenerator {
	return &BMPGenerator{}
}

// Generate creates a BMP file with a solid color.
func (g *BMPGenerator) Generate(output string, config Config) error {
	width := config.Width
	height := config.Height
	if width <= 0 {
		width = 320
	}
	if height <= 0 {
		height = 240
	}

	// Parse color
	r, gCol, b, err := parseColor(config.Color)
	if err != nil {
		return err
	}

	// Create file
	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Calculate sizes
	rowSize := ((width*3 + 3) / 4) * 4 // Row size padded to 4 bytes
	pixelDataSize := rowSize * height
	fileSize := 54 + pixelDataSize // 54 = header size

	// BMP File Header (14 bytes)
	fileHeader := make([]byte, 14)
	fileHeader[0] = 'B'
	fileHeader[1] = 'M'
	binary.LittleEndian.PutUint32(fileHeader[2:6], uint32(fileSize))
	binary.LittleEndian.PutUint32(fileHeader[10:14], 54) // Pixel data offset

	// DIB Header (40 bytes) - BITMAPINFOHEADER
	dibHeader := make([]byte, 40)
	binary.LittleEndian.PutUint32(dibHeader[0:4], 40)              // Header size
	binary.LittleEndian.PutUint32(dibHeader[4:8], uint32(width))   // Width
	binary.LittleEndian.PutUint32(dibHeader[8:12], uint32(height)) // Height
	binary.LittleEndian.PutUint16(dibHeader[12:14], 1)             // Color planes
	binary.LittleEndian.PutUint16(dibHeader[14:16], 24)            // Bits per pixel
	binary.LittleEndian.PutUint32(dibHeader[20:24], uint32(pixelDataSize))

	// Write headers
	if _, err := f.Write(fileHeader); err != nil {
		return err
	}
	if _, err := f.Write(dibHeader); err != nil {
		return err
	}

	// Write pixel data (BGR format, bottom-up)
	row := make([]byte, rowSize)
	for x := 0; x < width; x++ {
		row[x*3] = b
		row[x*3+1] = gCol
		row[x*3+2] = r
	}

	for y := 0; y < height; y++ {
		if _, err := f.Write(row); err != nil {
			return err
		}
	}

	return f.Sync()
}

// parseColor parses a color string (hex or "random").
func parseColor(color string) (r, g, b uint8, err error) {
	if color == "random" || color == "" {
		// Generate random color
		buf := make([]byte, 3)
		if _, err := rand.Read(buf); err != nil {
			return 0, 0, 0, err
		}
		return buf[0], buf[1], buf[2], nil
	}

	// Parse hex color
	color = strings.TrimPrefix(color, "#")
	if len(color) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid color format: %s", color)
	}

	rVal, err := strconv.ParseUint(color[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	gVal, err := strconv.ParseUint(color[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	bVal, err := strconv.ParseUint(color[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}

	return uint8(rVal), uint8(gVal), uint8(bVal), nil
}
