package ui

import (
	appwindow "codex-context-meter-lite/internal/window"
	"image"
	"image/color"
	"image/png"
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

func TestCaptureDemoWindow(t *testing.T) {
	output := os.Getenv("CCML_CAPTURE_PATH")
	if output == "" {
		t.Skip("set CCML_CAPTURE_PATH while a demo window is running")
	}
	procSetProcessDPIAwareContext.Call(^uintptr(3))
	minimumHeight := int32(150)
	captureCollapsed := os.Getenv("CCML_CAPTURE_COLLAPSED") == "1"
	if captureCollapsed {
		minimumHeight = 1
	}
	if os.Getenv("CCML_CAPTURE_HOVER") == "1" {
		var tracker appwindow.Tracker
		target := tracker.Target()
		if target == 0 {
			t.Fatal("Codex window not found")
		}
		if os.Getenv("CCML_CAPTURE_SKIP_FOREGROUND") != "1" {
			forceForeground(HWND(target))
			time.Sleep(500 * time.Millisecond)
		}
		minimumHeight = 1
	}
	hwnd := findVisibleMeterWindow(minimumHeight)
	if hwnd == 0 {
		t.Fatal("visible meter window not found")
	}
	if captureCollapsed {
		var bounds rect
		procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&bounds)))
		procSetCursorPos := user32.NewProc("SetCursorPos")
		procSetCursorPos.Call(uintptr(bounds.Left-12), uintptr(bounds.Top-12))
		procSendMessage.Call(uintptr(hwnd), wmMouseLeave, 0, 0)
		time.Sleep(300 * time.Millisecond)
		hwnd = findVisibleMeterWindow(1)
		if hwnd == 0 {
			t.Fatal("meter window did not collapse")
		}
		procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&bounds)))
		if height, width := bounds.Bottom-bounds.Top, bounds.Right-bounds.Left; height*4 >= width {
			t.Fatalf("meter window remained expanded: %dx%d", width, height)
		}
	}
	if os.Getenv("CCML_CAPTURE_HOVER") == "1" {
		var bounds rect
		procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&bounds)))
		procSetCursorPos := user32.NewProc("SetCursorPos")
		procSetCursorPos.Call(uintptr(bounds.Left-12), uintptr(bounds.Top-12))
		time.Sleep(75 * time.Millisecond)
		procSetCursorPos.Call(uintptr((bounds.Left+bounds.Right)/2), uintptr((bounds.Top+bounds.Bottom)/2))
		time.Sleep(750 * time.Millisecond)
		hwnd = findVisibleMeterWindow(150)
		if hwnd == 0 {
			t.Fatal("meter window did not expand on hover")
		}
	}
	img, err := captureWindow(hwnd)
	if err != nil {
		t.Fatal(err)
	}
	unique := map[color.RGBA]struct{}{}
	for y := 0; y < img.Bounds().Dy(); y += 2 {
		for x := 0; x < img.Bounds().Dx(); x += 2 {
			unique[img.RGBAAt(x, y)] = struct{}{}
		}
	}
	if len(unique) < 16 {
		t.Fatalf("capture has too little visual variation: %d colors", len(unique))
	}
	file, err := os.Create(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

func forceForeground(target HWND) {
	procGetForegroundWindow := user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessID := user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput := user32.NewProc("AttachThreadInput")
	procGetCurrentThreadID := kernel32.NewProc("GetCurrentThreadId")
	foreground, _, _ := procGetForegroundWindow.Call()
	foregroundThread, _, _ := procGetWindowThreadProcessID.Call(foreground, 0)
	targetThread, _, _ := procGetWindowThreadProcessID.Call(uintptr(target), 0)
	currentThread, _, _ := procGetCurrentThreadID.Call()
	if foregroundThread != 0 && foregroundThread != currentThread {
		procAttachThreadInput.Call(currentThread, foregroundThread, 1)
		defer procAttachThreadInput.Call(currentThread, foregroundThread, 0)
	}
	if targetThread != 0 && targetThread != currentThread {
		procAttachThreadInput.Call(currentThread, targetThread, 1)
		defer procAttachThreadInput.Call(currentThread, targetThread, 0)
	}
	procKeybdEvent := user32.NewProc("keybd_event")
	procKeybdEvent.Call(0x12, 0, 0, 0)
	defer procKeybdEvent.Call(0x12, 0, 2, 0)
	procShowWindow.Call(uintptr(target), 9)
	procSetForegroundWindow.Call(uintptr(target))
}

func findVisibleMeterWindow(minimumHeight int32) HWND {
	procEnumWindowsTest := user32.NewProc("EnumWindows")
	procGetClassName := user32.NewProc("GetClassNameW")
	procVisible := user32.NewProc("IsWindowVisible")
	var found HWND
	callback := syscall.NewCallback(func(raw uintptr, _ uintptr) uintptr {
		visible, _, _ := procVisible.Call(raw)
		if visible == 0 {
			return 1
		}
		buffer := make([]uint16, 256)
		length, _, _ := procGetClassName.Call(raw, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
		if length == 0 || windows.UTF16ToString(buffer[:length]) != className {
			return 1
		}
		var bounds rect
		ok, _, _ := procGetWindowRect.Call(raw, uintptr(unsafe.Pointer(&bounds)))
		if ok != 0 && bounds.Bottom-bounds.Top >= minimumHeight {
			found = HWND(raw)
			return 0
		}
		return 1
	})
	procEnumWindowsTest.Call(callback, 0)
	return found
}

func captureWindow(hwnd HWND) (*image.RGBA, error) {
	procGetDC := user32.NewProc("GetDC")
	procReleaseDC := user32.NewProc("ReleaseDC")
	procPrintWindow := user32.NewProc("PrintWindow")
	procCreateCompatibleDC := gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC := gdi32.NewProc("DeleteDC")
	procCreateCompatibleBitmap := gdi32.NewProc("CreateCompatibleBitmap")
	procGetDIBits := gdi32.NewProc("GetDIBits")

	var bounds rect
	if ok, _, err := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&bounds))); ok == 0 {
		return nil, err
	}
	width, height := bounds.Right-bounds.Left, bounds.Bottom-bounds.Top
	screenDC, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screenDC)
	memoryDC, _, err := procCreateCompatibleDC.Call(screenDC)
	if memoryDC == 0 {
		return nil, err
	}
	defer procDeleteDC.Call(memoryDC)
	bitmap, _, err := procCreateCompatibleBitmap.Call(screenDC, uintptr(width), uintptr(height))
	if bitmap == 0 {
		return nil, err
	}
	defer procDeleteObject.Call(bitmap)
	old, _, _ := procSelectObject.Call(memoryDC, bitmap)
	defer procSelectObject.Call(memoryDC, old)
	if ok, _, err := procPrintWindow.Call(uintptr(hwnd), memoryDC, 2); ok == 0 {
		return nil, err
	}

	pixels := make([]byte, int(width*height*4))
	info := bitmapInfo{Header: bitmapInfoHeader{Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: width, Height: -height, Planes: 1, BitCount: 32}}
	if lines, _, err := procGetDIBits.Call(memoryDC, bitmap, 0, uintptr(height), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&info)), 0); lines == 0 {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	for y := int32(0); y < height; y++ {
		for x := int32(0); x < width; x++ {
			offset := int((y*width + x) * 4)
			img.SetRGBA(int(x), int(y), color.RGBA{R: pixels[offset+2], G: pixels[offset+1], B: pixels[offset], A: 255})
		}
	}
	return img, nil
}
