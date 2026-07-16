package ui

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"codex-context-meter-lite/internal/config"
	"codex-context-meter-lite/internal/meter"
	appwindow "codex-context-meter-lite/internal/window"

	"golang.org/x/sys/windows"
)

type HWND uintptr
type HDC uintptr
type HGDIOBJ uintptr
type HMENU uintptr

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type msg struct {
	HWnd     HWND
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}
type wndClassEx struct {
	CbSize     uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSmall  uintptr
}
type paintStruct struct {
	DC        HDC
	Erase     int32
	Paint     rect
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}
type trackMouseEvent struct {
	Size      uint32
	Flags     uint32
	Track     HWND
	HoverTime uint32
}
type notifyIconData struct {
	Size             uint32
	Wnd              HWND
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	VersionOrTimeout uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GUIDItem         windows.GUID
	BalloonIcon      uintptr
}

const (
	className        = "CodexContextMeterLiteWindow"
	panelWidth int32 = 360

	panelCollapsedHeight       int32 = 28
	panelExpandedWaitingHeight int32 = 108
	panelExpandedHeight        int32 = 150

	wmCreate        = 0x0001
	wmDestroy       = 0x0002
	wmPaint         = 0x000F
	wmClose         = 0x0010
	wmCommand       = 0x0111
	wmTimer         = 0x0113
	wmMouseMove     = 0x0200
	wmLButtonDown   = 0x0201
	wmRButtonUp     = 0x0205
	wmMouseLeave    = 0x02A3
	wmNCHitTest     = 0x0084
	wmNCDestroy     = 0x0082
	wmEnterSizeMove = 0x0231
	wmExitSizeMove  = 0x0232
	wmApp           = 0x8000
	wmAppTray       = wmApp + 1
	wmAppUpdate     = wmApp + 2
	wmLButtonDblClk = 0x0203

	wsPopup        = 0x80000000
	wsExToolWindow = 0x00000080
	wsExTopmost    = 0x00000008
	wsExNoActivate = 0x08000000
	wsExLayered    = 0x00080000
	csHRedraw      = 0x0002
	csVRedraw      = 0x0001
	csDblClks      = 0x0008

	swHide           = 0
	swShowNoActivate = 4
	htCaption        = 2

	swpNoActivate = 0x0010
	swpShowWindow = 0x0040

	lwaAlpha      = 0x00000002
	transparent   = 1
	mmAnisotropic = 8
	dtLeft        = 0x0000
	dtRight       = 0x0002
	dtVCenter     = 0x0004
	dtSingleLine  = 0x0020
	dtEndEllipsis = 0x00008000

	mfString       = 0x0000
	mfSeparator    = 0x0800
	mfChecked      = 0x0008
	mfDisabled     = 0x0002
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100

	nimAdd     = 0x00000000
	nimDelete  = 0x00000002
	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	tmeLeave = 0x00000002

	dwmwaWindowCornerPreference = 33
	dwmwcpRound                 = 2

	idToggleVisible   = 1001
	idToggleUsed      = 1002
	idToggleTheme     = 1003
	idToggleAutostart = 1004
	idScale80         = 1010
	idScale100        = 1011
	idScale120        = 1012
	idAutoSession     = 1100
	idSessionBase     = 1200
	idQuit            = 2000
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassEx            = user32.NewProc("RegisterClassExW")
	procCreateWindowEx             = user32.NewProc("CreateWindowExW")
	procDefWindowProc              = user32.NewProc("DefWindowProcW")
	procDestroyWindow              = user32.NewProc("DestroyWindow")
	procShowWindow                 = user32.NewProc("ShowWindow")
	procUpdateWindow               = user32.NewProc("UpdateWindow")
	procGetMessage                 = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessage            = user32.NewProc("DispatchMessageW")
	procPostQuitMessage            = user32.NewProc("PostQuitMessage")
	procPostMessage                = user32.NewProc("PostMessageW")
	procBeginPaint                 = user32.NewProc("BeginPaint")
	procEndPaint                   = user32.NewProc("EndPaint")
	procInvalidateRect             = user32.NewProc("InvalidateRect")
	procSetTimer                   = user32.NewProc("SetTimer")
	procKillTimer                  = user32.NewProc("KillTimer")
	procSetWindowPos               = user32.NewProc("SetWindowPos")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procSetWindowRgn               = user32.NewProc("SetWindowRgn")
	procGetWindowRect              = user32.NewProc("GetWindowRect")
	procIsWindowVisible            = user32.NewProc("IsWindowVisible")
	procTrackMouseEvent            = user32.NewProc("TrackMouseEvent")
	procLoadCursor                 = user32.NewProc("LoadCursorW")
	procLoadIcon                   = user32.NewProc("LoadIconW")
	procCreatePopupMenu            = user32.NewProc("CreatePopupMenu")
	procAppendMenu                 = user32.NewProc("AppendMenuW")
	procTrackPopupMenu             = user32.NewProc("TrackPopupMenu")
	procDestroyMenu                = user32.NewProc("DestroyMenu")
	procGetCursorPos               = user32.NewProc("GetCursorPos")
	procSetForegroundWindow        = user32.NewProc("SetForegroundWindow")
	procSetProcessDPIAwareContext  = user32.NewProc("SetProcessDpiAwarenessContext")
	procReleaseCapture             = user32.NewProc("ReleaseCapture")
	procSendMessage                = user32.NewProc("SendMessageW")

	procCreateSolidBrush      = gdi32.NewProc("CreateSolidBrush")
	procFillRect              = user32.NewProc("FillRect")
	procSetBkMode             = gdi32.NewProc("SetBkMode")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procDrawText              = user32.NewProc("DrawTextW")
	procCreateFont            = gdi32.NewProc("CreateFontW")
	procSelectObject          = gdi32.NewProc("SelectObject")
	procDeleteObject          = gdi32.NewProc("DeleteObject")
	procCreatePen             = gdi32.NewProc("CreatePen")
	procMoveToEx              = gdi32.NewProc("MoveToEx")
	procLineTo                = gdi32.NewProc("LineTo")
	procRoundRect             = gdi32.NewProc("RoundRect")
	procSetMapMode            = gdi32.NewProc("SetMapMode")
	procSetWindowExtEx        = gdi32.NewProc("SetWindowExtEx")
	procSetViewportExtEx      = gdi32.NewProc("SetViewportExtEx")
	procCreateRoundRectRgn    = gdi32.NewProc("CreateRoundRectRgn")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")

	procShellNotifyIcon = shell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandle = kernel32.NewProc("GetModuleHandleW")
)

var (
	appByHWND             sync.Map
	globalWndProcCallback = syscall.NewCallback(globalWndProc)
)

type postWindowMessageFunc func(HWND, uint32)

func postNativeWindowMessage(hwnd HWND, message uint32) {
	procPostMessage.Call(uintptr(hwnd), uintptr(message), 0, 0)
}

type App struct {
	mu                sync.RWMutex
	hwnd              HWND
	config            config.Config
	view              ViewState
	handler           ActionHandler
	tracker           appwindow.Tracker
	hovered           bool
	tracking          bool
	dragging          bool
	standalone        bool
	standalonePlaced  bool
	quitting          bool
	trayIcon          uintptr
	fontSmall         HGDIOBJ
	font              HGDIOBJ
	fontBold          HGDIOBJ
	fontLarge         HGDIOBJ
	positionValid     bool
	lastVisible       bool
	lastX             int32
	lastY             int32
	lastWidth         int32
	lastHeight        int32
	lastRadius        int32
	dwmRounded        bool
	painted           bool
	positionLogged    bool
	viewLogged        bool
	postWindowMessage postWindowMessageFunc
}

func New(cfg config.Config, handler ActionHandler, standalone bool) *App {
	app := &App{config: cfg, handler: handler, standalone: standalone, postWindowMessage: postNativeWindowMessage}
	if standalone {
		app.hovered = true
	}
	return app
}

func (a *App) SetCodexExecutable(value string) {
	a.tracker.SetExecutableOverride(value)
}

func (a *App) SetView(view ViewState) {
	a.mu.Lock()
	if reflect.DeepEqual(a.view, view) {
		a.mu.Unlock()
		return
	}
	a.view = view
	shouldLog := debugUI() && !a.viewLogged && view.Snapshot.Known
	if shouldLog {
		a.viewLogged = true
	}
	a.mu.Unlock()
	if shouldLog {
		title := ""
		if view.Session.Selected != nil {
			title = view.Session.Selected.Title
		}
		fmt.Fprintf(os.Stderr, "data loaded title=%q used=%.1f left=%.1f\n", title, view.Snapshot.UsedPercent, view.Snapshot.LeftPercent)
	}
	a.postMessage(wmAppUpdate)
}

func (a *App) SetConfig(cfg config.Config) {
	a.mu.Lock()
	a.config = cfg
	a.mu.Unlock()
	a.postMessage(wmAppUpdate)
}

func (a *App) Quit() {
	a.mu.Lock()
	a.quitting = true
	a.mu.Unlock()
	a.postMessage(wmClose)
}

func (a *App) publishWindow(hwnd HWND) {
	a.mu.Lock()
	a.hwnd = hwnd
	a.mu.Unlock()
}

func (a *App) clearWindow(hwnd HWND) {
	a.mu.Lock()
	if a.hwnd == hwnd {
		a.hwnd = 0
	}
	a.mu.Unlock()
}

func (a *App) windowHandle() HWND {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.hwnd
}

func (a *App) postMessage(message uint32) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.hwnd == 0 {
		return
	}
	post := a.postWindowMessage
	if post == nil {
		post = postNativeWindowMessage
	}
	post(a.hwnd, message)
}

func (a *App) Run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procSetProcessDPIAwareContext.Call(^uintptr(3))
	instance, _, _ := procGetModuleHandle.Call(0)
	name, _ := windows.UTF16PtrFromString(className)
	cursor, _, _ := procLoadCursor.Call(0, 32512)
	icon, _, _ := procLoadIcon.Call(0, 32512)
	if meterIcon := createMeterIcon(instance); meterIcon != 0 {
		icon = meterIcon
		defer procDestroyIcon.Call(icon)
	}
	a.trayIcon = icon
	wc := wndClassEx{CbSize: uint32(unsafe.Sizeof(wndClassEx{})), Style: csHRedraw | csVRedraw | csDblClks, WndProc: globalWndProcCallback, Instance: instance, Icon: icon, Cursor: cursor, ClassName: name, IconSmall: icon}
	if result, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); result == 0 {
		return fmt.Errorf("RegisterClassExW: %w", err)
	}
	title, _ := windows.UTF16PtrFromString("Codex Context Meter Lite")
	extendedStyle := uintptr(wsExTopmost)
	if !a.standalone {
		extendedStyle |= wsExToolWindow | wsExNoActivate
	}
	hwnd, _, err := procCreateWindowEx.Call(extendedStyle, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(title)), wsPopup, 0, 0, uintptr(panelWidth), uintptr(panelCollapsedHeight), 0, 0, instance, 0)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW: %w", err)
	}
	window := HWND(hwnd)
	appByHWND.Store(window, a)
	a.publishWindow(window)
	defer func() {
		a.clearWindow(window)
		appByHWND.Delete(window)
	}()
	a.dwmRounded = enableDwmRoundedCorners(window)
	a.createFonts()
	a.addTrayIcon()
	procSetTimer.Call(hwnd, 1, 250, 0)
	a.syncPosition()
	procInvalidateRect.Call(hwnd, 0, 1)
	procUpdateWindow.Call(hwnd)
	if debugUI() {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		fmt.Fprintf(os.Stderr, "ui created hwnd=%d visible=%d standalone=%v\n", hwnd, visible, a.standalone)
	}

	var message msg
	for {
		result, _, getErr := procGetMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) == -1 {
			return fmt.Errorf("GetMessageW: %w", getErr)
		}
		if result == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
	return nil
}

func globalWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	value, ok := appByHWND.Load(HWND(hwnd))
	if !ok {
		result, _, _ := procDefWindowProc.Call(hwnd, uintptr(message), wParam, lParam)
		return result
	}
	return value.(*App).wndProc(HWND(hwnd), message, wParam, lParam)
}

func (a *App) wndProc(hwnd HWND, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case wmPaint:
		a.paint()
		return 0
	case wmTimer:
		a.syncPosition()
		return 0
	case wmAppUpdate:
		procInvalidateRect.Call(uintptr(a.hwnd), 0, 0)
		a.syncPosition()
		return 0
	case wmMouseMove:
		if !a.hovered {
			a.hovered = true
			a.tracking = true
			track := trackMouseEvent{Size: uint32(unsafe.Sizeof(trackMouseEvent{})), Flags: tmeLeave, Track: a.hwnd}
			procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&track)))
			a.syncPosition()
			procInvalidateRect.Call(uintptr(a.hwnd), 0, 0)
		}
		return 0
	case wmMouseLeave:
		a.hovered = false
		a.tracking = false
		a.syncPosition()
		procInvalidateRect.Call(uintptr(a.hwnd), 0, 0)
		return 0
	case wmRButtonUp:
		a.showMenu()
		return 0
	case wmLButtonDown:
		procReleaseCapture.Call()
		procSendMessage.Call(uintptr(a.hwnd), 0x00A1, htCaption, 0)
		return 0
	case wmEnterSizeMove:
		a.dragging = true
		return 0
	case wmExitSizeMove:
		a.dragging = false
		a.persistOffset()
		return 0
	case wmCommand:
		a.handleMenu(uint16(wParam & 0xffff))
		return 0
	case wmAppTray:
		switch uint32(lParam) {
		case wmRButtonUp:
			a.showMenu()
		case wmLButtonDblClk:
			a.emit(Action{Kind: ActionToggleDisplay})
		}
		return 0
	case wmClose:
		a.mu.RLock()
		quitting := a.quitting
		a.mu.RUnlock()
		if quitting {
			procDestroyWindow.Call(uintptr(a.hwnd))
		} else {
			a.hideOverlay()
		}
		return 0
	case wmDestroy:
		procKillTimer.Call(uintptr(a.hwnd), 1)
		a.removeTrayIcon()
		if a.fontSmall != 0 {
			procDeleteObject.Call(uintptr(a.fontSmall))
		}
		if a.font != 0 {
			procDeleteObject.Call(uintptr(a.font))
		}
		if a.fontBold != 0 {
			procDeleteObject.Call(uintptr(a.fontBold))
		}
		if a.fontLarge != 0 {
			procDeleteObject.Call(uintptr(a.fontLarge))
		}
		procPostQuitMessage.Call(0)
		return 0
	case wmNCDestroy:
		a.clearWindow(hwnd)
		appByHWND.Delete(hwnd)
		result, _, _ := procDefWindowProc.Call(uintptr(hwnd), uintptr(message), wParam, lParam)
		return result
	}
	result, _, _ := procDefWindowProc.Call(uintptr(hwnd), uintptr(message), wParam, lParam)
	return result
}

func (a *App) syncPosition() {
	a.mu.RLock()
	cfg, known := a.config, a.view.Snapshot.Known
	a.mu.RUnlock()
	if a.dragging {
		return
	}
	if !cfg.OverlayVisible {
		a.hideOverlay()
		return
	}
	if a.standalone {
		scale := cfg.Scale
		width := int32(math.Round(float64(panelWidth) * scale))
		height := int32(math.Round(float64(panelHeight(a.hovered, known)) * scale))
		x, y := int32(80), int32(80)
		if a.standalonePlaced {
			var current rect
			if result, _, _ := procGetWindowRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&current))); result != 0 {
				x, y = current.Left, current.Top
			}
		}
		a.standalonePlaced = true
		radius := uintptr(int32(math.Round(8 * scale)))
		a.positionOverlay(x, y, width, height, int32(radius))
		return
	}
	target := a.tracker.Snapshot()
	if debugUI() && !a.positionLogged {
		a.positionLogged = true
		fmt.Fprintf(os.Stderr, "codex target=%d rect=%+v valid=%v foreground=%v\n", target.Target, target.Rect, target.Valid, target.Foreground)
	}
	if !shouldShowTrackedOverlay(target) {
		a.hideOverlay()
		return
	}
	dpiScale := float64(target.DPI) / 96
	scale := cfg.Scale * dpiScale
	width := int32(math.Round(float64(panelWidth) * scale))
	height := int32(math.Round(float64(panelHeight(a.hovered, known)) * scale))
	ox := int32(math.Round(float64(cfg.OffsetX) * dpiScale))
	oy := int32(math.Round(float64(cfg.OffsetY) * dpiScale))
	targetBounds := rect{target.Rect.Left, target.Rect.Top, target.Rect.Right, target.Rect.Bottom}
	x, y := anchoredPosition(cfg.Anchor, targetBounds, width, height, ox, oy)
	a.positionOverlay(x, y, width, height, int32(math.Round(8*scale)))
}

func shouldShowTrackedOverlay(target appwindow.WindowSnapshot) bool {
	return target.Valid && target.Foreground
}

func (a *App) hideOverlay() {
	if !a.lastVisible {
		return
	}
	procShowWindow.Call(uintptr(a.hwnd), swHide)
	a.lastVisible = false
}

func (a *App) positionOverlay(x, y, width, height, radius int32) {
	sizeChanged := !a.positionValid || width != a.lastWidth || height != a.lastHeight || radius != a.lastRadius
	geometryChanged := !a.positionValid || x != a.lastX || y != a.lastY || sizeChanged
	if sizeChanged && !a.dwmRounded {
		region, _, _ := procCreateRoundRectRgn.Call(0, 0, uintptr(width+1), uintptr(height+1), uintptr(radius), uintptr(radius))
		if result, _, _ := procSetWindowRgn.Call(uintptr(a.hwnd), region, 1); result == 0 {
			procDeleteObject.Call(region)
		}
	}
	if geometryChanged || !a.lastVisible {
		flags := uintptr(swpNoActivate)
		if !a.lastVisible {
			flags |= swpShowWindow
		}
		procSetWindowPos.Call(uintptr(a.hwnd), ^uintptr(0), uintptr(x), uintptr(y), uintptr(width), uintptr(height), flags)
	}
	a.positionValid = true
	a.lastVisible = true
	a.lastX, a.lastY = x, y
	a.lastWidth, a.lastHeight, a.lastRadius = width, height, radius
}

func enableDwmRoundedCorners(hwnd HWND) bool {
	preference := uint32(dwmwcpRound)
	result, _, _ := procDwmSetWindowAttribute.Call(
		uintptr(hwnd),
		dwmwaWindowCornerPreference,
		uintptr(unsafe.Pointer(&preference)),
		unsafe.Sizeof(preference),
	)
	return result == 0
}

func panelHeight(hovered, known bool) int32 {
	if !hovered {
		return panelCollapsedHeight
	}
	if !known {
		return panelExpandedWaitingHeight
	}
	return panelExpandedHeight
}

func (a *App) persistOffset() {
	if a.standalone {
		return
	}
	target := a.tracker.Snapshot()
	if !target.Valid {
		return
	}
	var current rect
	if result, _, _ := procGetWindowRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&current))); result == 0 {
		return
	}
	dpiScale := float64(target.DPI) / 96
	targetBounds := rect{target.Rect.Left, target.Rect.Top, target.Rect.Right, target.Rect.Bottom}
	anchor := nearestAnchor(targetBounds, current)
	x, y := anchorOffsets(anchor, targetBounds, current)
	a.emit(Action{Kind: ActionMove, Anchor: anchor, OffsetX: max(0, int(math.Round(float64(x)/dpiScale))), OffsetY: max(0, int(math.Round(float64(y)/dpiScale)))})
}

func nearestAnchor(target, current rect) string {
	horizontal := "right"
	if abs32(current.Left-target.Left) <= abs32(target.Right-current.Right) {
		horizontal = "left"
	}
	vertical := "bottom"
	if abs32(current.Top-target.Top) <= abs32(target.Bottom-current.Bottom) {
		vertical = "top"
	}
	return vertical + "-" + horizontal
}

func anchorOffsets(anchor string, target, current rect) (int32, int32) {
	var x, y int32
	switch anchor {
	case "bottom-left":
		x, y = current.Left-target.Left, target.Bottom-current.Bottom
	case "top-left":
		x, y = current.Left-target.Left, current.Top-target.Top
	case "top-right":
		x, y = target.Right-current.Right, current.Top-target.Top
	default:
		x, y = target.Right-current.Right, target.Bottom-current.Bottom
	}
	return x, y
}

func anchoredPosition(anchor string, target rect, width, height, offsetX, offsetY int32) (int32, int32) {
	x, y := target.Right-width-offsetX, target.Bottom-height-offsetY
	switch anchor {
	case "bottom-left":
		x = target.Left + offsetX
	case "top-left":
		x, y = target.Left+offsetX, target.Top+offsetY
	case "top-right":
		y = target.Top + offsetY
	}
	return x, y
}

func abs32(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}

func (a *App) paint() {
	var ps paintStruct
	dc, _, _ := procBeginPaint.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&ps)))
	if dc == 0 {
		return
	}
	defer procEndPaint.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&ps)))
	var bounds rect
	procGetWindowRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&bounds)))
	physicalWidth, physicalHeight := bounds.Right-bounds.Left, bounds.Bottom-bounds.Top
	a.mu.RLock()
	cfg, view := a.config, a.view
	a.mu.RUnlock()
	width, height := panelWidth, panelHeight(a.hovered, view.Snapshot.Known)
	procSetMapMode.Call(dc, mmAnisotropic)
	procSetWindowExtEx.Call(dc, uintptr(width), uintptr(height), 0)
	procSetViewportExtEx.Call(dc, uintptr(physicalWidth), uintptr(physicalHeight), 0)

	dark := cfg.Theme != "light"
	bg := rgb(8, 12, 17)
	fg, muted, faint := rgb(225, 235, 241), rgb(105, 126, 139), rgb(43, 58, 67)
	track, accent := rgb(26, 39, 47), rgb(55, 214, 237)
	if !dark {
		bg = rgb(243, 247, 249)
		fg, muted, faint = rgb(18, 31, 38), rgb(82, 104, 115), rgb(184, 199, 205)
		track, accent = rgb(207, 220, 225), rgb(0, 137, 168)
	}
	fillRect(HDC(dc), rect{0, 0, width, height}, bg)
	if debugUI() && !a.painted {
		a.painted = true
		fmt.Fprintf(os.Stderr, "ui painted hwnd=%d physical=%dx%d logical=%dx%d background=%06x\n", a.hwnd, physicalWidth, physicalHeight, width, height, bg)
	}
	procSetBkMode.Call(dc, transparent)

	margin := int32(14)

	if !view.Snapshot.Known {
		if !a.hovered {
			drawText(HDC(dc), a.fontSmall, muted, "AWAITING DATA", rect{margin, 0, width - margin, height}, dtLeft|dtVCenter|dtSingleLine)
			return
		}
		drawText(HDC(dc), a.fontBold, fg, "AWAITING DATA", rect{margin, 3, width - margin, 29}, dtLeft|dtVCenter|dtSingleLine)
		detail := "READ-ONLY SESSION LINK"
		if view.Session.Error != "" {
			detail = "SESSION LINK UNAVAILABLE"
		}
		drawText(HDC(dc), a.fontSmall, muted, detail, rect{margin, 27, width - margin, 48}, dtLeft|dtVCenter|dtSingleLine)
		if a.hovered {
			drawLine(HDC(dc), margin, 54, width-margin, 54, faint, 1)
			drawText(HDC(dc), a.fontSmall, accent, "SOURCE", rect{margin, 61, 170, 76}, dtLeft|dtVCenter|dtSingleLine)
			drawText(HDC(dc), a.font, fg, "CODEX JSONL", rect{margin, 76, 170, 96}, dtLeft|dtVCenter|dtSingleLine)
			drawText(HDC(dc), a.fontSmall, accent, "ACCESS", rect{190, 61, width - margin, 76}, dtLeft|dtVCenter|dtSingleLine)
			drawText(HDC(dc), a.font, fg, "READ ONLY", rect{190, 76, width - margin, 96}, dtLeft|dtVCenter|dtSingleLine)
		}
		return
	}
	percent := view.Snapshot.LeftPercent
	tokens := view.Snapshot.RemainingTokens
	if cfg.ShowUsed {
		percent, tokens = view.Snapshot.UsedPercent, view.Snapshot.UsedTokens
	}
	color := contextPressureColor(view.Snapshot.UsedPercent)
	if !a.hovered {
		drawText(HDC(dc), a.fontBold, color, fmt.Sprintf("%.1f%%", percent), rect{margin, 0, 112, 20}, dtLeft|dtVCenter|dtSingleLine)
		drawText(HDC(dc), a.fontSmall, muted, fmt.Sprintf("%s / %s", formatHeaderTokens(tokens), formatHeaderTokens(view.Snapshot.ContextWindow)), rect{105, 0, width - margin, 20}, dtRight|dtVCenter|dtSingleLine)
		drawContextBar(HDC(dc), rect{margin, 22, width - margin, 26}, percent, color, track, bg)
		return
	}
	drawText(HDC(dc), a.fontLarge, color, fmt.Sprintf("%.1f%%", percent), rect{margin, 1, 190, 34}, dtLeft|dtVCenter|dtSingleLine)
	drawText(HDC(dc), a.font, muted, fmt.Sprintf("%s / %s TOKENS", formatHeaderTokens(tokens), formatHeaderTokens(view.Snapshot.ContextWindow)), rect{170, 6, width - margin, 33}, dtRight|dtVCenter|dtSingleLine)
	drawContextBar(HDC(dc), rect{margin, 38, width - margin, 47}, percent, color, track, bg)
	drawLine(HDC(dc), margin, 54, width-margin, 54, faint, 1)
	colors := metricPalette(dark)
	drawMetricColumn(HDC(dc), a.fontSmall, a.font, a.fontBold, fg, colors, rect{margin, 60, 171, 142}, "TURN LOAD", view.Snapshot.Turn)
	drawMetricColumn(HDC(dc), a.fontSmall, a.font, a.fontBold, fg, colors, rect{189, 60, width - margin, 142}, "SESSION", view.Snapshot.Session)
	drawLine(HDC(dc), 180, 62, 180, 141, faint, 1)
}

func (a *App) createFonts() {
	face, _ := windows.UTF16PtrFromString("Cascadia Mono")
	small, _, _ := procCreateFont.Call(negativeFontHeight(10), 0, 0, 0, 500, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(face)))
	regular, _, _ := procCreateFont.Call(negativeFontHeight(11), 0, 0, 0, 500, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(face)))
	bold, _, _ := procCreateFont.Call(negativeFontHeight(14), 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(face)))
	large, _, _ := procCreateFont.Call(negativeFontHeight(22), 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(face)))
	a.fontSmall, a.font, a.fontBold, a.fontLarge = HGDIOBJ(small), HGDIOBJ(regular), HGDIOBJ(bold), HGDIOBJ(large)
}

func (a *App) addTrayIcon() {
	data := notifyIconData{Size: uint32(unsafe.Sizeof(notifyIconData{})), Wnd: a.hwnd, ID: 1, Flags: nifMessage | nifIcon | nifTip, CallbackMessage: wmAppTray, Icon: a.trayIcon}
	copy(data.Tip[:], windows.StringToUTF16("Codex Context Meter Lite"))
	procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&data)))
}

func (a *App) removeTrayIcon() {
	data := notifyIconData{Size: uint32(unsafe.Sizeof(notifyIconData{})), Wnd: a.hwnd, ID: 1}
	procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
}

func (a *App) showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)
	a.mu.RLock()
	cfg, view := a.config, a.view
	a.mu.RUnlock()
	windowStatus, sessionStatus := diagnosticStatusLabels(a.tracker.Target() != 0, view.Snapshot.Known)
	appendMenu(HMENU(menu), 0, mfDisabled, windowStatus)
	appendMenu(HMENU(menu), 0, mfDisabled, sessionStatus)
	procAppendMenu.Call(menu, mfSeparator, 0, 0)
	appendMenu(HMENU(menu), idToggleVisible, checked(cfg.OverlayVisible), "Show overlay")
	appendMenu(HMENU(menu), idToggleUsed, checked(cfg.ShowUsed), "Show context used")
	appendMenu(HMENU(menu), idToggleTheme, checked(cfg.Theme == "light"), "Light theme")
	procAppendMenu.Call(menu, mfSeparator, 0, 0)
	appendMenu(HMENU(menu), idScale80, checked(almost(cfg.Scale, .8)), "Scale 80%")
	appendMenu(HMENU(menu), idScale100, checked(almost(cfg.Scale, 1)), "Scale 100%")
	appendMenu(HMENU(menu), idScale120, checked(almost(cfg.Scale, 1.2)), "Scale 120%")
	procAppendMenu.Call(menu, mfSeparator, 0, 0)
	appendMenu(HMENU(menu), idAutoSession, checked(cfg.PinnedThreadID == ""), "Session: Auto (latest active)")
	for i, candidate := range view.Session.Candidates {
		flags := uint32(0)
		if candidate.ThreadID == cfg.PinnedThreadID {
			flags = mfChecked
		}
		label := candidate.Title
		if len([]rune(label)) > 36 {
			label = string([]rune(label)[:36]) + "..."
		}
		appendMenu(HMENU(menu), uint16(idSessionBase+i), flags, label)
	}
	procAppendMenu.Call(menu, mfSeparator, 0, 0)
	appendMenu(HMENU(menu), idToggleAutostart, checked(cfg.StartWithWindows), "Start with Windows")
	procAppendMenu.Call(menu, mfSeparator, 0, 0)
	appendMenu(HMENU(menu), idQuit, 0, "Quit")
	var cursor point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	procSetForegroundWindow.Call(uintptr(a.hwnd))
	command, _, _ := procTrackPopupMenu.Call(menu, tpmRightButton|tpmReturnCmd, uintptr(cursor.X), uintptr(cursor.Y), 0, uintptr(a.hwnd), 0)
	if command != 0 {
		a.handleMenu(uint16(command))
	}
}

func (a *App) handleMenu(id uint16) {
	switch id {
	case idToggleVisible:
		a.emit(Action{Kind: ActionToggleDisplay})
	case idToggleUsed:
		a.emit(Action{Kind: ActionToggleUsed})
	case idToggleTheme:
		a.emit(Action{Kind: ActionToggleTheme})
	case idToggleAutostart:
		a.emit(Action{Kind: ActionToggleAutostart})
	case idScale80:
		a.emit(Action{Kind: ActionSetScale, Scale: .8})
	case idScale100:
		a.emit(Action{Kind: ActionSetScale, Scale: 1})
	case idScale120:
		a.emit(Action{Kind: ActionSetScale, Scale: 1.2})
	case idAutoSession:
		a.emit(Action{Kind: ActionSetPinned})
	case idQuit:
		a.emit(Action{Kind: ActionQuit})
	default:
		if id >= idSessionBase {
			a.mu.RLock()
			index := int(id - idSessionBase)
			threadID := ""
			if index < len(a.view.Session.Candidates) {
				threadID = a.view.Session.Candidates[index].ThreadID
			}
			a.mu.RUnlock()
			if threadID != "" {
				a.emit(Action{Kind: ActionSetPinned, ThreadID: threadID})
			}
		}
	}
}

func (a *App) emit(action Action) {
	if a.handler != nil {
		a.handler(action)
	}
}

func fillRect(dc HDC, area rect, color uint32) {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(color))
	procFillRect.Call(uintptr(dc), uintptr(unsafe.Pointer(&area)), brush)
	procDeleteObject.Call(brush)
}

func drawLine(dc HDC, x1, y1, x2, y2 int32, color uint32, width int32) {
	pen, _, _ := procCreatePen.Call(0, uintptr(width), uintptr(color))
	old, _, _ := procSelectObject.Call(uintptr(dc), pen)
	procMoveToEx.Call(uintptr(dc), uintptr(x1), uintptr(y1), 0)
	procLineTo.Call(uintptr(dc), uintptr(x2), uintptr(y2))
	procSelectObject.Call(uintptr(dc), old)
	procDeleteObject.Call(pen)
}

func drawContextBar(dc HDC, area rect, percent float64, color, track, background uint32) {
	fillAntialiasedRoundedRect(dc, area, track, background)
	fillWidth := contextFillWidth(area.Right-area.Left, percent)
	if fillWidth > 0 {
		fillBackground := track
		if fillWidth >= area.Right-area.Left {
			fillBackground = background
		}
		fillAntialiasedRoundedRect(dc, rect{area.Left, area.Top, area.Left + fillWidth, area.Bottom}, color, fillBackground)
	}
}

func fillAntialiasedRoundedRect(dc HDC, area rect, color, background uint32) {
	width, height := area.Right-area.Left, area.Bottom-area.Top
	if width <= 0 || height <= 0 {
		return
	}
	for y := int32(0); y < height; y++ {
		runStart := int32(0)
		runCoverage := roundedRectCoverage(0, y, width, height)
		for x := int32(1); x <= width; x++ {
			coverage := -1
			if x < width {
				coverage = roundedRectCoverage(x, y, width, height)
			}
			if coverage == runCoverage {
				continue
			}
			if runCoverage > 0 {
				fillRect(dc, rect{area.Left + runStart, area.Top + y, area.Left + x, area.Top + y + 1}, blendColor(color, background, runCoverage, 16))
			}
			runStart, runCoverage = x, coverage
		}
	}
}

func roundedRectCoverage(x, y, width, height int32) int {
	const samples = 4
	radius := float64(min(width, height)) / 2
	left, right := radius, float64(width)-radius
	top, bottom := radius, float64(height)-radius
	coverage := 0
	for sampleY := 0; sampleY < samples; sampleY++ {
		for sampleX := 0; sampleX < samples; sampleX++ {
			px := float64(x) + (float64(sampleX)+0.5)/samples
			py := float64(y) + (float64(sampleY)+0.5)/samples
			nearestX := math.Max(left, math.Min(right, px))
			nearestY := math.Max(top, math.Min(bottom, py))
			if math.Hypot(px-nearestX, py-nearestY) <= radius {
				coverage++
			}
		}
	}
	return coverage
}

func blendColor(foreground, background uint32, coverage, samples int) uint32 {
	if coverage <= 0 {
		return background
	}
	if coverage >= samples {
		return foreground
	}
	blend := func(foregroundComponent, backgroundComponent uint32) uint32 {
		return (foregroundComponent*uint32(coverage) + backgroundComponent*uint32(samples-coverage) + uint32(samples/2)) / uint32(samples)
	}
	r := blend(foreground&0xff, background&0xff)
	g := blend((foreground>>8)&0xff, (background>>8)&0xff)
	b := blend((foreground>>16)&0xff, (background>>16)&0xff)
	return r | g<<8 | b<<16
}

func fillRoundedRect(dc HDC, area rect, color uint32) {
	width, height := area.Right-area.Left, area.Bottom-area.Top
	if width <= 0 || height <= 0 {
		return
	}
	roundness := min(width, height)
	brush, _, _ := procCreateSolidBrush.Call(uintptr(color))
	pen, _, _ := procCreatePen.Call(0, 1, uintptr(color))
	oldBrush, _, _ := procSelectObject.Call(uintptr(dc), brush)
	oldPen, _, _ := procSelectObject.Call(uintptr(dc), pen)
	procRoundRect.Call(uintptr(dc), uintptr(area.Left), uintptr(area.Top), uintptr(area.Right), uintptr(area.Bottom), uintptr(roundness), uintptr(roundness))
	procSelectObject.Call(uintptr(dc), oldPen)
	procSelectObject.Call(uintptr(dc), oldBrush)
	procDeleteObject.Call(pen)
	procDeleteObject.Call(brush)
}

func contextFillWidth(width int32, percent float64) int32 {
	clamped := math.Max(0, math.Min(100, percent))
	return int32(math.Round(float64(width) * clamped / 100))
}

func drawText(dc HDC, font HGDIOBJ, color uint32, text string, area rect, flags uint32) {
	value, _ := windows.UTF16PtrFromString(text)
	old, _, _ := procSelectObject.Call(uintptr(dc), uintptr(font))
	procSetTextColor.Call(uintptr(dc), uintptr(color))
	procDrawText.Call(uintptr(dc), uintptr(unsafe.Pointer(value)), ^uintptr(0), uintptr(unsafe.Pointer(&area)), uintptr(flags))
	procSelectObject.Call(uintptr(dc), old)
}

type metricColors struct {
	total  uint32
	input  uint32
	cache  uint32
	output uint32
	reason uint32
}

func metricPalette(dark bool) metricColors {
	if dark {
		return metricColors{
			total:  rgb(47, 140, 255),
			input:  rgb(22, 199, 132),
			cache:  rgb(28, 183, 202),
			output: rgb(139, 92, 246),
			reason: rgb(255, 170, 32),
		}
	}
	return metricColors{
		total:  rgb(0, 102, 204),
		input:  rgb(0, 142, 89),
		cache:  rgb(0, 139, 160),
		output: rgb(108, 58, 198),
		reason: rgb(190, 108, 0),
	}
}

func drawMetricColumn(dc HDC, small, regular, bold HGDIOBJ, fg uint32, colors metricColors, area rect, title string, usage meter.Usage) {
	drawMetricLabel(dc, small, colors.total, title, rect{area.Left, area.Top, area.Right, area.Top + 15})
	drawText(dc, bold, fg, formatTokens(usage.TotalTokens), rect{area.Left, area.Top + 14, area.Right, area.Top + 34}, dtLeft|dtVCenter|dtSingleLine)
	middle := area.Left + (area.Right-area.Left)/2
	drawMetricLabel(dc, small, colors.input, "INPUT", rect{area.Left, area.Top + 34, middle, area.Top + 46})
	drawMetricLabel(dc, small, colors.cache, formatCacheLabel(usage), rect{middle, area.Top + 34, area.Right, area.Top + 46})
	drawText(dc, regular, fg, formatTokens(usage.InputTokens), rect{area.Left, area.Top + 45, middle, area.Top + 59}, dtLeft|dtVCenter|dtSingleLine)
	drawText(dc, regular, fg, formatTokens(usage.CachedInputTokens), rect{middle, area.Top + 45, area.Right, area.Top + 59}, dtLeft|dtVCenter|dtSingleLine)
	drawMetricLabel(dc, small, colors.output, "OUTPUT", rect{area.Left, area.Top + 59, middle, area.Top + 71})
	drawMetricLabel(dc, small, colors.reason, "REASON", rect{middle, area.Top + 59, area.Right, area.Top + 71})
	drawText(dc, regular, fg, formatTokens(usage.OutputTokens), rect{area.Left, area.Top + 70, middle, area.Bottom}, dtLeft|dtVCenter|dtSingleLine)
	drawText(dc, regular, fg, formatTokens(usage.ReasoningOutputTokens), rect{middle, area.Top + 70, area.Right, area.Bottom}, dtLeft|dtVCenter|dtSingleLine)
}

func drawMetricLabel(dc HDC, font HGDIOBJ, color uint32, text string, area rect) {
	dotSize := int32(4)
	dotTop := area.Top + (area.Bottom-area.Top-dotSize)/2
	fillRoundedRect(dc, rect{area.Left, dotTop, area.Left + dotSize, dotTop + dotSize}, color)
	drawText(dc, font, color, text, rect{area.Left + 8, area.Top, area.Right, area.Bottom}, dtLeft|dtVCenter|dtSingleLine)
}

func drawMetricRow(dc HDC, font HGDIOBJ, leftColor, rightColor uint32, left, right, top int32, leftText, rightText string) {
	middle := left + (right-left)*43/100
	drawText(dc, font, leftColor, leftText, rect{left, top, middle, top + 20}, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	drawText(dc, font, rightColor, rightText, rect{middle, top, right, top + 20}, dtRight|dtVCenter|dtSingleLine|dtEndEllipsis)
}

func appendMenu(menu HMENU, id uint16, flags uint32, label string) {
	value, _ := windows.UTF16PtrFromString(label)
	procAppendMenu.Call(uintptr(menu), uintptr(mfString|flags), uintptr(id), uintptr(unsafe.Pointer(value)))
}

func checked(value bool) uint32 {
	if value {
		return mfChecked
	}
	return 0
}

func diagnosticStatusLabels(windowDetected, sessionDetected bool) (windowStatus, sessionStatus string) {
	windowStatus = "Codex window: not detected"
	if windowDetected {
		windowStatus = "Codex window: detected"
	}
	sessionStatus = "Session data: not detected"
	if sessionDetected {
		sessionStatus = "Session data: detected"
	}
	return windowStatus, sessionStatus
}
func almost(a, b float64) bool { return math.Abs(a-b) < .01 }
func rgb(r, g, b byte) uint32  { return uint32(r) | uint32(g)<<8 | uint32(b)<<16 }
func negativeFontHeight(size uintptr) uintptr {
	return ^uintptr(size - 1)
}
func contextPressureColor(usedPercent float64) uint32 {
	switch {
	case usedPercent > 85:
		return rgb(231, 72, 79)
	case usedPercent > 75:
		return rgb(255, 170, 32)
	default:
		return rgb(22, 199, 132)
	}
}
func formatTokens(value int64) string {
	switch n := math.Abs(float64(value)); {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(value)/1_000)
	default:
		return fmt.Sprintf("%d", value)
	}
}

func formatCacheLabel(usage meter.Usage) string {
	percent, ok := usage.CacheHitPercent()
	if !ok {
		return "CACHE --"
	}
	if percent >= 99.95 {
		return "CACHE 100%"
	}
	return fmt.Sprintf("CACHE %.1f%%", percent)
}

func formatHeaderTokens(value int64) string {
	switch n := math.Abs(float64(value)); {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.0fK", float64(value)/1_000)
	default:
		return fmt.Sprintf("%d", value)
	}
}

func sanitize(value string) string { return strings.ReplaceAll(value, "\n", " ") }

var _ = sanitize
var _ = mfDisabled

func debugUI() bool { return os.Getenv("CCML_DEBUG_UI") == "1" }
