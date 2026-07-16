package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"codex-context-meter-lite/internal/ui"
)

var iconSizes = []int{16, 20, 24, 32, 40, 48, 64, 128, 256}

type iconFrame struct {
	size int
	png  []byte
}

func main() {
	output := flag.String("output", filepath.FromSlash("assets/codex-context-meter-lite.ico"), "ICO output path")
	flag.Parse()

	data, err := buildICO(iconSizes)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fatal(err)
	}
}

func buildICO(sizes []int) ([]byte, error) {
	if len(sizes) == 0 || len(sizes) > 0xffff {
		return nil, errors.New("ICO must contain between 1 and 65535 images")
	}

	frames := make([]iconFrame, len(sizes))
	seen := make(map[int]struct{}, len(sizes))
	for i, size := range sizes {
		if size < 1 || size > 256 {
			return nil, fmt.Errorf("invalid ICO size %d", size)
		}
		if _, exists := seen[size]; exists {
			return nil, fmt.Errorf("duplicate ICO size %d", size)
		}
		seen[size] = struct{}{}

		frame, err := renderPNG(size)
		if err != nil {
			return nil, err
		}
		frames[i] = iconFrame{size: size, png: frame}
	}

	const directorySize = 6
	const entrySize = 16
	offset := directorySize + entrySize*len(frames)
	var output bytes.Buffer
	output.Grow(offset)
	writeUint16(&output, 0)
	writeUint16(&output, 1)
	writeUint16(&output, uint16(len(frames)))
	for _, frame := range frames {
		dimension := byte(frame.size)
		output.WriteByte(dimension)
		output.WriteByte(dimension)
		output.WriteByte(0)
		output.WriteByte(0)
		writeUint16(&output, 1)
		writeUint16(&output, 32)
		writeUint32(&output, uint32(len(frame.png)))
		writeUint32(&output, uint32(offset))
		offset += len(frame.png)
	}
	for _, frame := range frames {
		output.Write(frame.png)
	}
	return output.Bytes(), nil
}

func renderPNG(size int) ([]byte, error) {
	pixels := ui.MeterIconPixels(size)
	if len(pixels) != size*size {
		return nil, fmt.Errorf("rendered %d pixels for %dx%d icon", len(pixels), size, size)
	}

	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for i, pixel := range pixels {
		offset := i * 4
		img.Pix[offset] = pixel.R
		img.Pix[offset+1] = pixel.G
		img.Pix[offset+2] = pixel.B
		img.Pix[offset+3] = pixel.A
	}

	var output bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&output, img); err != nil {
		return nil, fmt.Errorf("encode %dx%d icon: %w", size, size, err)
	}
	return output.Bytes(), nil
}

func writeUint16(output *bytes.Buffer, value uint16) {
	var encoded [2]byte
	binary.LittleEndian.PutUint16(encoded[:], value)
	output.Write(encoded[:])
}

func writeUint32(output *bytes.Buffer, value uint32) {
	var encoded [4]byte
	binary.LittleEndian.PutUint32(encoded[:], value)
	output.Write(encoded[:])
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
