package window

import (
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type HWND uintptr

type Rect struct {
	Left, Top, Right, Bottom int32
}

func (r Rect) Width() int32  { return r.Right - r.Left }
func (r Rect) Height() int32 { return r.Bottom - r.Top }

const (
	processQueryLimitedInformation = 0x1000
	fallbackSearchCooldown         = time.Second
)

const (
	getAncestorRoot      = 2
	getAncestorRootOwner = 3
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procIsWindow                 = user32.NewProc("IsWindow")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetAncestor              = user32.NewProc("GetAncestor")
	procGetDpiForWindow          = user32.NewProc("GetDpiForWindow")
	procGetPackageFamilyName     = kernel32.NewProc("GetPackageFamilyName")

	enumWindowsCallback = syscall.NewCallback(enumWindowsProc)
	enumerateWindows    = func(callback, context uintptr) {
		procEnumWindows.Call(callback, context)
	}
	enumWindowsStates = struct {
		sync.RWMutex
		next   uintptr
		values map[uintptr]*enumWindowsState
	}{values: make(map[uintptr]*enumWindowsState)}
)

type Tracker struct {
	target             HWND
	executableOverride string
	nextFallbackSearch time.Time
	ops                *trackerOps
}

type WindowSnapshot struct {
	Target     HWND
	Rect       Rect
	DPI        uint32
	Foreground bool
	Valid      bool
}

type trackerOps struct {
	now             func() time.Time
	foregroundCodex func(string) HWND
	isUsable        func(HWND) bool
	find            func(string) HWND
	readRect        func(HWND) (Rect, bool)
	isForeground    func(HWND) bool
	dpi             func(HWND) uint32
}

var defaultTrackerOps = trackerOps{
	now:             time.Now,
	foregroundCodex: foregroundCodexWindow,
	isUsable:        isUsableWindow,
	find:            findCodexWindow,
	readRect:        readWindowRect,
	isForeground:    isForegroundWindow,
	dpi:             windowDPI,
}

// SetExecutableOverride restricts unpackaged-process matching to one exact
// executable path. Official OpenAI.Codex packages continue to be recognized by
// package family name.
func (t *Tracker) SetExecutableOverride(value string) {
	t.executableOverride = normalizeExecutablePath(value)
	t.Invalidate()
}

func (t *Tracker) Target() HWND {
	return t.targetWithOps(t.operations())
}

func (t *Tracker) targetWithOps(ops trackerOps) HWND {
	if foreground := ops.foregroundCodex(t.executableOverride); foreground != 0 {
		t.target = foreground
		return t.target
	}
	if t.target != 0 && ops.isUsable(t.target) {
		return t.target
	}
	t.target = 0
	now := ops.now()
	if now.Before(t.nextFallbackSearch) {
		return 0
	}
	t.nextFallbackSearch = now.Add(fallbackSearchCooldown)
	t.target = ops.find(t.executableOverride)
	return t.target
}

func (t *Tracker) Invalidate() {
	t.target = 0
	t.nextFallbackSearch = time.Time{}
}

func (t *Tracker) Snapshot() WindowSnapshot {
	ops := t.operations()
	target := t.targetWithOps(ops)
	if target == 0 {
		return WindowSnapshot{}
	}
	windowRect, valid := ops.readRect(target)
	if !valid {
		return WindowSnapshot{Target: target}
	}
	return WindowSnapshot{
		Target:     target,
		Rect:       windowRect,
		DPI:        ops.dpi(target),
		Foreground: ops.isForeground(target),
		Valid:      true,
	}
}

func (t *Tracker) operations() trackerOps {
	if t.ops != nil {
		return *t.ops
	}
	return defaultTrackerOps
}

func (t *Tracker) Rect() (Rect, bool) {
	snapshot := t.Snapshot()
	return snapshot.Rect, snapshot.Valid
}

func (t *Tracker) IsForeground() bool {
	return t.Snapshot().Foreground
}

func (t *Tracker) DPI() uint32 {
	dpi := t.Snapshot().DPI
	if dpi == 0 {
		return 96
	}
	return dpi
}

func IsVisible(hwnd HWND) bool {
	result, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return result != 0
}

func IsMinimized(hwnd HWND) bool {
	result, _, _ := procIsIconic.Call(uintptr(hwnd))
	return result != 0
}

func isWindow(hwnd HWND) bool {
	result, _, _ := procIsWindow.Call(uintptr(hwnd))
	return result != 0
}

func isUsableWindow(hwnd HWND) bool {
	return hwnd != 0 && isWindow(hwnd) && IsVisible(hwnd) && !IsMinimized(hwnd)
}

func readWindowRect(hwnd HWND) (Rect, bool) {
	if hwnd == 0 || !IsVisible(hwnd) || IsMinimized(hwnd) {
		return Rect{}, false
	}
	var windowRect Rect
	result, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&windowRect)))
	return windowRect, result != 0 && windowRect.Width() > 0 && windowRect.Height() > 0
}

func isForegroundWindow(hwnd HWND) bool {
	foreground, _, _ := procGetForegroundWindow.Call()
	return hwnd != 0 && normalizeWindow(HWND(foreground)) == normalizeWindow(hwnd)
}

func windowDPI(hwnd HWND) uint32 {
	if hwnd == 0 {
		return 96
	}
	result, _, _ := procGetDpiForWindow.Call(uintptr(hwnd))
	if result == 0 {
		return 96
	}
	return uint32(result)
}

func normalizeWindow(hwnd HWND) HWND {
	if hwnd == 0 {
		return 0
	}
	if owner, _, _ := procGetAncestor.Call(uintptr(hwnd), getAncestorRootOwner); owner != 0 {
		hwnd = HWND(owner)
	}
	if root, _, _ := procGetAncestor.Call(uintptr(hwnd), getAncestorRoot); root != 0 {
		return HWND(root)
	}
	return hwnd
}

func foregroundCodexWindow(executableOverride string) HWND {
	foreground, _, _ := procGetForegroundWindow.Call()
	raw := HWND(foreground)
	if raw == 0 {
		return 0
	}
	normalized := normalizeWindow(raw)
	if !isUsableWindow(normalized) {
		return 0
	}
	if isCodexWindow(normalized, executableOverride) || (normalized != raw && isCodexWindow(raw, executableOverride)) {
		return normalized
	}
	return 0
}

type windowCandidate struct {
	handle HWND
	area   int64
}

type enumWindowsState struct {
	executableOverride string
	candidates         []windowCandidate
	seen               map[HWND]struct{}
}

func chooseCodexCandidate(candidates []windowCandidate) HWND {
	var best HWND
	var bestArea int64
	for _, candidate := range candidates {
		if candidate.handle == 0 {
			continue
		}
		if candidate.area > bestArea {
			best = candidate.handle
			bestArea = candidate.area
		}
	}
	return best
}

func findCodexWindow(executableOverride string) HWND {
	state := &enumWindowsState{executableOverride: executableOverride, seen: make(map[HWND]struct{})}
	context := registerEnumWindowsState(state)
	defer unregisterEnumWindowsState(context)
	enumerateWindows(enumWindowsCallback, context)
	return chooseCodexCandidate(state.candidates)
}

func enumWindowsProc(hwnd, context uintptr) uintptr {
	state, ok := enumWindowsStateForHandle(context)
	if !ok {
		return 0
	}
	handle := normalizeWindow(HWND(hwnd))
	if !isUsableWindow(handle) {
		return 1
	}
	if _, ok := state.seen[handle]; ok {
		return 1
	}
	if !isCodexWindow(handle, state.executableOverride) {
		return 1
	}
	state.seen[handle] = struct{}{}
	var windowRect Rect
	result, _, _ := procGetWindowRect.Call(uintptr(handle), uintptr(unsafe.Pointer(&windowRect)))
	if result == 0 {
		return 1
	}
	area := int64(windowRect.Width()) * int64(windowRect.Height())
	if area > 0 {
		state.candidates = append(state.candidates, windowCandidate{handle: handle, area: area})
	}
	return 1
}

func registerEnumWindowsState(state *enumWindowsState) uintptr {
	enumWindowsStates.Lock()
	defer enumWindowsStates.Unlock()
	enumWindowsStates.next++
	if enumWindowsStates.next == 0 {
		enumWindowsStates.next++
	}
	handle := enumWindowsStates.next
	enumWindowsStates.values[handle] = state
	return handle
}

func enumWindowsStateForHandle(handle uintptr) (*enumWindowsState, bool) {
	enumWindowsStates.RLock()
	defer enumWindowsStates.RUnlock()
	state, ok := enumWindowsStates.values[handle]
	return state, ok
}

func unregisterEnumWindowsState(handle uintptr) {
	enumWindowsStates.Lock()
	delete(enumWindowsStates.values, handle)
	enumWindowsStates.Unlock()
}

func isCodexWindow(hwnd HWND, executableOverride string) bool {
	var pid uint32
	procGetWindowThreadProcessID.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return false
	}
	handle, err := windows.OpenProcess(processQueryLimitedInformation, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)
	packageFamily, _ := getPackageFamilyName(handle)
	var processPath string
	if executableOverride != "" || packageFamily == "" {
		buffer := make([]uint16, 32768)
		size := uint32(len(buffer))
		if windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size) == nil {
			processPath = windows.UTF16ToString(buffer[:size])
		}
	}
	return isCodexProcess(processPath, packageFamily, executableOverride)
}

func isCodexExecutable(path string) bool {
	return isCodexProcess(path, "", "")
}

func isCodexProcess(path, packageFamily, executableOverride string) bool {
	normalizedPath := normalizeExecutablePath(path)
	if executableOverride != "" && normalizedPath != "" && normalizedPath == normalizeExecutablePath(executableOverride) {
		return true
	}

	packageFamily = strings.TrimSpace(packageFamily)
	if packageFamily != "" {
		return strings.HasPrefix(strings.ToLower(packageFamily), "openai.codex_")
	}

	if normalizedPath == "" {
		return false
	}

	base := strings.ToLower(filepath.Base(normalizedPath))
	if base != "codex.exe" && base != "chatgpt.exe" {
		return false
	}
	return hasCodexProductComponent(normalizedPath)
}

func normalizeExecutablePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"`)
	if path == "" {
		return ""
	}
	return strings.ToLower(filepath.Clean(path))
}

func hasCodexProductComponent(path string) bool {
	components := strings.FieldsFunc(strings.ToLower(path), func(r rune) bool {
		return r == '\\' || r == '/'
	})
	for _, component := range components {
		name := strings.TrimSuffix(component, filepath.Ext(component))
		switch name {
		case "codex", "openai codex", "openai.codex", "codex desktop", "openai codex desktop":
			return true
		}
		if strings.HasPrefix(component, "openai.codex_") {
			return true
		}
	}
	return false
}

func getPackageFamilyName(process windows.Handle) (string, error) {
	var length uint32
	result, _, _ := procGetPackageFamilyName.Call(
		uintptr(process),
		uintptr(unsafe.Pointer(&length)),
		0,
	)
	code := syscall.Errno(uint32(result))
	if code == windows.APPMODEL_ERROR_NO_PACKAGE {
		return "", nil
	}
	if code != windows.ERROR_INSUFFICIENT_BUFFER {
		if code == windows.ERROR_SUCCESS {
			return "", nil
		}
		return "", code
	}
	if length == 0 {
		return "", nil
	}

	buffer := make([]uint16, length)
	result, _, _ = procGetPackageFamilyName.Call(
		uintptr(process),
		uintptr(unsafe.Pointer(&length)),
		uintptr(unsafe.Pointer(&buffer[0])),
	)
	code = syscall.Errno(uint32(result))
	if code != windows.ERROR_SUCCESS {
		return "", code
	}
	return windows.UTF16ToString(buffer), nil
}
