package ui

import (
	"image"
	"image/png"
	"os"
	"testing"
)

func TestMeterIcon(t *testing.T) {
	for _, size := range []int{16, 20, 24, 32, 64} {
		andBits, xorBits := meterIconBitmaps(size)
		wantAND := ((size + 15) / 16) * 2 * size
		wantXOR := size * size * 4
		if len(andBits) != wantAND || len(xorBits) != wantXOR {
			t.Fatalf("size %d bitmap lengths: AND=%d XOR=%d, want %d and %d", size, len(andBits), len(xorBits), wantAND, wantXOR)
		}
	}

	pixels := meterIconPixels(systemTrayIconSize())
	hasAntialiasing := false
	for _, pixel := range pixels {
		if pixel.A > 0 && pixel.A < 255 {
			hasAntialiasing = true
			break
		}
	}
	if !hasAntialiasing {
		t.Fatal("icon has no antialiased edge pixels")
	}

	icon := createMeterIcon(0)
	if icon == 0 {
		t.Fatal("CreateIcon returned a null handle")
	}
	procDestroyIcon.Call(icon)

	if output := os.Getenv("CCML_ICON_CAPTURE_PATH"); output != "" {
		writeMeterIconPreview(t, output)
	}
}

func writeMeterIconPreview(t *testing.T, output string) {
	t.Helper()
	const size = 64
	const scale = 2
	pixels := meterIconPixels(size)
	preview := image.NewRGBA(image.Rect(0, 0, size*scale, size*scale))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			pixel := pixels[y*size+x]
			for py := 0; py < scale; py++ {
				for px := 0; px < scale; px++ {
					preview.SetRGBA(x*scale+px, y*scale+py, pixel)
				}
			}
		}
	}
	file, err := os.Create(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, preview); err != nil {
		t.Fatal(err)
	}
}
