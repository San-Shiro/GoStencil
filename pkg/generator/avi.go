// avi.go - Pure Go AVI generator using Motion JPEG (MJPEG) video codec.
// AVI container has better native MJPEG support on Windows than MP4.
// This generator creates valid AVI files without requiring any external dependencies.
package generator

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
)

// AVIGenerator generates AVI files with MJPEG video track using pure Go.
// AVI format has better native Windows support for MJPEG than MP4.
type AVIGenerator struct{}

// NewAVIGenerator creates a new AVI generator.
func NewAVIGenerator() *AVIGenerator {
	return &AVIGenerator{}
}

// Generate creates a valid AVI file containing an MJPEG stream.
func (g *AVIGenerator) Generate(output string, config Config) error {
	// 1. Prepare source image
	var img image.Image
	if config.SourceImage != nil {
		img = config.SourceImage
	} else {
		// Create solid color image
		width := config.Width
		height := config.Height
		if width <= 0 {
			width = 1280
		}
		if height <= 0 {
			height = 720
		}
		r, gCol, b, err := parseColor(config.Color)
		if err != nil {
			return err
		}
		rgba := image.NewRGBA(image.Rect(0, 0, width, height))
		fillColor := color.RGBA{r, gCol, b, 255}
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				rgba.Set(x, y, fillColor)
			}
		}
		img = rgba
	}

	// 2. Encode to JPEG
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return fmt.Errorf("failed to encode JPEG: %w", err)
	}
	jpegData := buf.Bytes()
	jpegSize := uint32(len(jpegData))

	// Pad to even size (AVI requirement)
	paddedJPEGSize := jpegSize
	if jpegSize%2 != 0 {
		paddedJPEGSize = jpegSize + 1
	}

	// 3. Configure Video
	width := uint32(img.Bounds().Dx())
	height := uint32(img.Bounds().Dy())
	fps := uint32(15)
	microSecPerFrame := uint32(1000000 / fps)
	durationSec := uint32(config.Duration)
	if durationSec < 1 {
		durationSec = 1
	}
	totalFrames := durationSec * fps

	// Calculate sizes
	frameChunkSize := 8 + paddedJPEGSize // "00dc" + size + data
	moviSize := 4 + (totalFrames * frameChunkSize)
	idx1Size := 8 + (totalFrames * 16) // idx1 header + entries

	// 4. Create file
	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	// Helper to write FourCC
	writeFourCC := func(s string) {
		f.Write([]byte(s))
	}

	// Helper to write uint32 little-endian
	writeUint32 := func(v uint32) {
		binary.Write(f, binary.LittleEndian, v)
	}

	// Helper to write uint16 little-endian
	writeUint16 := func(v uint16) {
		binary.Write(f, binary.LittleEndian, v)
	}

	// === RIFF Header ===
	// Total file size = RIFF header (12) + hdrl list + movi list + idx1
	hdrlSize := uint32(4 + 64 + 124) // LIST + avih + strl
	fileSize := 4 + (8 + hdrlSize) + (8 + moviSize) + idx1Size

	writeFourCC("RIFF")
	writeUint32(fileSize)
	writeFourCC("AVI ")

	// === hdrl LIST ===
	writeFourCC("LIST")
	writeUint32(hdrlSize)
	writeFourCC("hdrl")

	// === avih (Main AVI Header) - 56 bytes + 8 header ===
	writeFourCC("avih")
	writeUint32(56) // chunk size
	writeUint32(microSecPerFrame)
	writeUint32(uint32(float64(jpegSize) * float64(fps))) // max bytes per sec
	writeUint32(0)                                        // padding granularity
	writeUint32(0x10)                                     // flags: AVIF_HASINDEX
	writeUint32(totalFrames)
	writeUint32(0)        // initial frames
	writeUint32(1)        // number of streams
	writeUint32(jpegSize) // suggested buffer size
	writeUint32(width)    // width
	writeUint32(height)   // height
	writeUint32(0)        // reserved
	writeUint32(0)        // reserved
	writeUint32(0)        // reserved
	writeUint32(0)        // reserved

	// === strl LIST (Stream List) ===
	writeFourCC("LIST")
	writeUint32(116) // strl size: strh(64) + strf(48) + 4
	writeFourCC("strl")

	// === strh (Stream Header) - 56 bytes + 8 header ===
	writeFourCC("strh")
	writeUint32(56)
	writeFourCC("vids") // fccType
	writeFourCC("MJPG") // fccHandler - MJPEG codec
	writeUint32(0)      // flags
	writeUint16(0)      // priority
	writeUint16(0)      // language
	writeUint32(0)      // initial frames
	writeUint32(1)      // scale
	writeUint32(fps)    // rate
	writeUint32(0)      // start
	writeUint32(totalFrames)
	writeUint32(jpegSize) // suggested buffer size
	writeUint32(0)        // quality
	writeUint32(0)        // sample size
	writeUint16(0)        // left
	writeUint16(0)        // top
	writeUint16(uint16(width))
	writeUint16(uint16(height))

	// === strf (Stream Format - BITMAPINFOHEADER) - 40 bytes + 8 header ===
	writeFourCC("strf")
	writeUint32(40)
	writeUint32(40)     // biSize
	writeUint32(width)  // biWidth
	writeUint32(height) // biHeight
	writeUint16(1)      // biPlanes
	writeUint16(24)     // biBitCount
	writeFourCC("MJPG") // biCompression
	writeUint32(width * height * 3)
	writeUint32(0) // biXPelsPerMeter
	writeUint32(0) // biYPelsPerMeter
	writeUint32(0) // biClrUsed
	writeUint32(0) // biClrImportant

	// === movi LIST ===
	writeFourCC("LIST")
	writeUint32(moviSize)
	writeFourCC("movi")

	// Write video frames
	for i := uint32(0); i < totalFrames; i++ {
		writeFourCC("00dc") // video chunk
		writeUint32(jpegSize)
		f.Write(jpegData)
		// Pad to even boundary
		if jpegSize%2 != 0 {
			f.Write([]byte{0})
		}
	}

	// === idx1 (Index) ===
	writeFourCC("idx1")
	writeUint32(totalFrames * 16)

	moviOffset := uint32(4) // offset from movi start
	for i := uint32(0); i < totalFrames; i++ {
		writeFourCC("00dc")
		writeUint32(0x10) // flags: AVIIF_KEYFRAME
		writeUint32(moviOffset)
		writeUint32(jpegSize)
		moviOffset += frameChunkSize
	}

	return f.Sync()
}
