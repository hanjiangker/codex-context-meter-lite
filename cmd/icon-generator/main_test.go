package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
)

type parsedIconFrame struct {
	size int
	data []byte
}

func TestIconAsset(t *testing.T) {
	want, err := buildICO(iconSizes)
	if err != nil {
		t.Fatal(err)
	}
	again, err := buildICO(iconSizes)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, again) {
		t.Fatal("ICO encoding is not deterministic")
	}

	assetPath := filepath.Join("..", "..", "assets", "codex-context-meter-lite.ico")
	asset, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(asset, want) {
		t.Fatalf("%s is stale; run scripts/generate-icon.ps1", assetPath)
	}

	frames, err := parseICO(asset)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != len(iconSizes) {
		t.Fatalf("ICO has %d frames, want %d", len(frames), len(iconSizes))
	}

	seen := make(map[int]bool, len(frames))
	for i, frame := range frames {
		if frame.size != iconSizes[i] {
			t.Errorf("frame %d size = %d, want %d", i, frame.size, iconSizes[i])
		}
		if seen[frame.size] {
			t.Errorf("duplicate %dx%d frame", frame.size, frame.size)
		}
		seen[frame.size] = true

		img, _, err := image.Decode(bytes.NewReader(frame.data))
		if err != nil {
			t.Errorf("decode %dx%d frame: %v", frame.size, frame.size, err)
			continue
		}
		bounds := img.Bounds()
		if bounds.Dx() != frame.size || bounds.Dy() != frame.size {
			t.Errorf("decoded frame is %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), frame.size, frame.size)
		}
		if frame.size == 16 || frame.size == 32 || frame.size == 256 {
			checkAlphaCoverage(t, img, frame.size)
		}
	}
}

func parseICO(data []byte) ([]parsedIconFrame, error) {
	if len(data) < 6 {
		return nil, errors.New("ICO header is truncated")
	}
	if binary.LittleEndian.Uint16(data[0:2]) != 0 {
		return nil, errors.New("ICO reserved field is not zero")
	}
	if binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return nil, errors.New("ICO type is not icon")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count == 0 || len(data) < 6+count*16 {
		return nil, errors.New("ICO directory is truncated")
	}

	frames := make([]parsedIconFrame, count)
	for i := 0; i < count; i++ {
		entry := data[6+i*16 : 6+(i+1)*16]
		size := int(entry[0])
		if size == 0 {
			size = 256
		}
		height := int(entry[1])
		if height == 0 {
			height = 256
		}
		if height != size || entry[2] != 0 || entry[3] != 0 {
			return nil, fmt.Errorf("invalid ICO directory entry %d", i)
		}
		if binary.LittleEndian.Uint16(entry[4:6]) != 1 || binary.LittleEndian.Uint16(entry[6:8]) != 32 {
			return nil, fmt.Errorf("invalid ICO format for frame %d", i)
		}
		length := uint64(binary.LittleEndian.Uint32(entry[8:12]))
		offset := uint64(binary.LittleEndian.Uint32(entry[12:16]))
		end := offset + length
		if length == 0 || end > uint64(len(data)) || end < offset {
			return nil, fmt.Errorf("invalid ICO frame bounds for entry %d", i)
		}
		frames[i] = parsedIconFrame{size: size, data: data[int(offset):int(end)]}
	}
	return frames, nil
}

func checkAlphaCoverage(t *testing.T, img image.Image, size int) {
	t.Helper()
	hasTransparent := false
	hasVisible := false
	hasAntialiasing := false
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			hasTransparent = hasTransparent || alpha == 0
			hasVisible = hasVisible || alpha > 0
			hasAntialiasing = hasAntialiasing || (alpha > 0 && alpha < 0xffff)
		}
	}
	if !hasTransparent || !hasVisible || !hasAntialiasing {
		t.Errorf("%dx%d alpha coverage: transparent=%t visible=%t antialiased=%t", size, size, hasTransparent, hasVisible, hasAntialiasing)
	}
}
