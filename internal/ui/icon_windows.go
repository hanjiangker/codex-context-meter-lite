package ui

import (
	"image/color"
	"math"
	"unsafe"
)

const (
	meterIconDesignSize = 32.0
	smCXSmallIcon       = 49
)

var (
	procCreateIcon       = user32.NewProc("CreateIcon")
	procDestroyIcon      = user32.NewProc("DestroyIcon")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
)

func createMeterIcon(instance uintptr) uintptr {
	size := systemTrayIconSize()
	andBits, xorBits := meterIconBitmaps(size)
	icon, _, _ := procCreateIcon.Call(
		instance,
		uintptr(size),
		uintptr(size),
		1,
		32,
		uintptr(unsafe.Pointer(&andBits[0])),
		uintptr(unsafe.Pointer(&xorBits[0])),
	)
	return icon
}

func systemTrayIconSize() int {
	size, _, _ := procGetSystemMetrics.Call(smCXSmallIcon)
	if size < 16 || size > 64 {
		return 32
	}
	return int(size)
}

func meterIconBitmaps(size int) ([]byte, []byte) {
	pixels := meterIconPixels(size)
	andStride := ((size + 15) / 16) * 2
	andBits := make([]byte, andStride*size)
	xorBits := make([]byte, size*size*4)
	for y := 0; y < size; y++ {
		bottomUpY := size - 1 - y
		for x := 0; x < size; x++ {
			pixel := pixels[y*size+x]
			if pixel.A == 0 {
				andBits[bottomUpY*andStride+x/8] |= 0x80 >> (x % 8)
			}
			offset := (bottomUpY*size + x) * 4
			xorBits[offset] = pixel.B
			xorBits[offset+1] = pixel.G
			xorBits[offset+2] = pixel.R
			xorBits[offset+3] = pixel.A
		}
	}
	return andBits, xorBits
}

func meterIconPixels(size int) []color.RGBA {
	const samples = 4
	pixels := make([]color.RGBA, size*size)
	scale := meterIconDesignSize / float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var sumR, sumG, sumB, sumA int
			for sampleY := 0; sampleY < samples; sampleY++ {
				for sampleX := 0; sampleX < samples; sampleX++ {
					px := (float64(x) + (float64(sampleX)+0.5)/float64(samples)) * scale
					py := (float64(y) + (float64(sampleY)+0.5)/float64(samples)) * scale
					pixel := meterIconSample(px, py)
					alpha := int(pixel.A)
					sumR += int(pixel.R) * alpha
					sumG += int(pixel.G) * alpha
					sumB += int(pixel.B) * alpha
					sumA += alpha
				}
			}
			if sumA == 0 {
				continue
			}
			pixels[y*size+x] = color.RGBA{
				R: uint8(sumR / sumA),
				G: uint8(sumG / sumA),
				B: uint8(sumB / sumA),
				A: uint8(sumA / (samples * samples)),
			}
		}
	}
	return pixels
}

// MeterIconPixels renders the icon shared by the tray and executable resources.
func MeterIconPixels(size int) []color.RGBA {
	return meterIconPixels(size)
}

func meterIconSample(px, py float64) color.RGBA {
	if !insideRoundedBox(px, py, 1, 31, 7.2) {
		return color.RGBA{}
	}
	pixel := color.RGBA{R: 7, G: 15, B: 21, A: 255}

	dx, dy := px-16, py-16
	distance := math.Hypot(dx, dy)
	if distance >= 6.5 && distance <= 10.8 {
		pixel = color.RGBA{R: 29, G: 58, B: 67, A: 255}
		angle := math.Atan2(dy, dx) * 180 / math.Pi
		if math.Abs(angle) >= 42 {
			pixel = color.RGBA{R: 55, G: 214, B: 237, A: 255}
		}
	}
	if distance <= 2.4 {
		pixel = color.RGBA{R: 225, G: 235, B: 241, A: 255}
	}
	return pixel
}

func insideRoundedBox(x, y, low, high, radius float64) bool {
	nearestX := math.Max(low+radius, math.Min(high-radius, x))
	nearestY := math.Max(low+radius, math.Min(high-radius, y))
	return math.Hypot(x-nearestX, y-nearestY) <= radius
}
