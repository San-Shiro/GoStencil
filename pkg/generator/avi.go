// avi.go — Pure Go AVI/MJPEG writer.
//
// Creates a valid AVI container with a single MJPEG video stream.
// The input is always an image.Image (the "PNG-first" pipeline).
package generator

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
)

// binaryWriter wraps an io.Writer and accumulates the first error,
// preventing silently-ignored write failures throughout the AVI assembly.
type binaryWriter struct {
	w   io.Writer
	err error
}

func (bw *binaryWriter) fourCC(s string) {
	if bw.err != nil {
		return
	}
	_, bw.err = bw.w.Write([]byte(s))
}

func (bw *binaryWriter) u32(v uint32) {
	if bw.err != nil {
		return
	}
	bw.err = binary.Write(bw.w, binary.LittleEndian, v)
}

func (bw *binaryWriter) u16(v uint16) {
	if bw.err != nil {
		return
	}
	bw.err = binary.Write(bw.w, binary.LittleEndian, v)
}

func (bw *binaryWriter) bytes(data []byte) {
	if bw.err != nil {
		return
	}
	_, bw.err = bw.w.Write(data)
}

// writeAVI creates a valid AVI (MJPEG) file from a single image repeated
// for the given duration at 15 fps.
func writeAVI(output string, img image.Image, durationSec int) error {
	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create %s: %w", output, err)
	}
	defer f.Close()

	if err := writeAVITo(f, img, durationSec); err != nil {
		return err
	}
	return f.Sync()
}

// writeAVITo writes AVI data to any io.Writer.
func writeAVITo(w io.Writer, img image.Image, durationSec int) error {
	// Encode source image to JPEG once.
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return fmt.Errorf("encode JPEG frame: %w", err)
	}
	jpegData := buf.Bytes()
	jpegSize := uint32(len(jpegData))

	// AVI requires even-aligned chunks.
	paddedSize := jpegSize
	if jpegSize%2 != 0 {
		paddedSize++
	}

	// Video parameters.
	imgW := uint32(img.Bounds().Dx())
	imgH := uint32(img.Bounds().Dy())
	const fps = 15
	usPerFrame := uint32(1_000_000 / fps)
	frames := uint32(durationSec) * fps

	// Chunk sizes.
	frameChunk := 8 + paddedSize          // "00dc" + size + data
	moviSize := 4 + (frames * frameChunk) // "movi" + frames
	idx1Size := 8 + (frames * 16)         // "idx1" header + entries
	hdrlSize := uint32(4 + 64 + 124)      // "hdrl" + avih + strl
	fileSize := 4 + (8 + hdrlSize) + (8 + moviSize) + idx1Size

	bw := &binaryWriter{w: w}

	// ── RIFF Header ──
	bw.fourCC("RIFF")
	bw.u32(fileSize)
	bw.fourCC("AVI ")

	// ── hdrl LIST ──
	bw.fourCC("LIST")
	bw.u32(hdrlSize)
	bw.fourCC("hdrl")

	// avih (56 bytes)
	bw.fourCC("avih")
	bw.u32(56)
	bw.u32(usPerFrame)
	bw.u32(uint32(float64(jpegSize) * fps)) // max bytes/sec
	bw.u32(0)                               // padding granularity
	bw.u32(0x10)                            // AVIF_HASINDEX
	bw.u32(frames)
	bw.u32(0)        // initial frames
	bw.u32(1)        // streams
	bw.u32(jpegSize) // suggested buffer
	bw.u32(imgW)
	bw.u32(imgH)
	bw.u32(0) // reserved ×4
	bw.u32(0)
	bw.u32(0)
	bw.u32(0)

	// strl LIST (116 bytes)
	bw.fourCC("LIST")
	bw.u32(116)
	bw.fourCC("strl")

	// strh (56 bytes)
	bw.fourCC("strh")
	bw.u32(56)
	bw.fourCC("vids")
	bw.fourCC("MJPG")
	bw.u32(0) // flags
	bw.u16(0) // priority
	bw.u16(0) // language
	bw.u32(0) // initial frames
	bw.u32(1) // scale
	bw.u32(fps)
	bw.u32(0) // start
	bw.u32(frames)
	bw.u32(jpegSize) // suggested buffer
	bw.u32(0)        // quality
	bw.u32(0)        // sample size
	bw.u16(0)        // rect left
	bw.u16(0)        // rect top
	bw.u16(uint16(imgW))
	bw.u16(uint16(imgH))

	// strf — BITMAPINFOHEADER (40 bytes)
	bw.fourCC("strf")
	bw.u32(40)
	bw.u32(40)
	bw.u32(imgW)
	bw.u32(imgH)
	bw.u16(1)  // planes
	bw.u16(24) // bpp
	bw.fourCC("MJPG")
	bw.u32(imgW * imgH * 3)
	bw.u32(0) // x pels/m
	bw.u32(0) // y pels/m
	bw.u32(0) // clr used
	bw.u32(0) // clr important

	// ── movi LIST ──
	bw.fourCC("LIST")
	bw.u32(moviSize)
	bw.fourCC("movi")

	padByte := []byte{0}
	for range frames {
		bw.fourCC("00dc")
		bw.u32(jpegSize)
		bw.bytes(jpegData)
		if jpegSize%2 != 0 {
			bw.bytes(padByte)
		}
	}

	// ── idx1 ──
	bw.fourCC("idx1")
	bw.u32(frames * 16)

	offset := uint32(4) // from movi start
	for range frames {
		bw.fourCC("00dc")
		bw.u32(0x10) // AVIIF_KEYFRAME
		bw.u32(offset)
		bw.u32(jpegSize)
		offset += frameChunk
	}

	if bw.err != nil {
		return fmt.Errorf("write AVI: %w", bw.err)
	}
	return nil
}
