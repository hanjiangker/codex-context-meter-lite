package window

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCodexExecutableMatching(t *testing.T) {
	cases := []struct {
		name          string
		path          string
		packageFamily string
		override      string
		want          bool
	}{
		{
			name:          "official package family ignores drive version and architecture",
			path:          `D:\Apps\arbitrary\Host.exe`,
			packageFamily: `OpenAI.Codex_26.707.3351.0_arm64__2p2nqsd0c76g0`,
			want:          true,
		},
		{
			name:          "official package family works without process path",
			packageFamily: `OpenAI.Codex_2p2nqsd0c76g0`,
			want:          true,
		},
		{
			name:          "ordinary ChatGPT package is rejected",
			path:          `C:\Program Files\OpenAI Codex\ChatGPT.exe`,
			packageFamily: `OpenAI.ChatGPT_1.0_x64__2p2nqsd0c76g0`,
			want:          false,
		},
		{
			name: "legacy official package path",
			path: `C:\Program Files\WindowsApps\OpenAI.Codex_26.707.3351.0_x64__2p2nqsd0c76g0\app\ChatGPT.exe`,
			want: true,
		},
		{
			name: "enterprise ChatGPT host",
			path: `E:\Company Apps\OpenAI Codex\ChatGPT.exe`,
			want: true,
		},
		{
			name: "Codex executable",
			path: `C:\Tools\Codex.exe`,
			want: true,
		},
		{
			name: "generic ChatGPT host",
			path: `C:\Tools\ChatGPT.exe`,
			want: false,
		},
		{
			name:     "exact override",
			path:     `C:\Enterprise\DesktopHost.exe`,
			override: `"c:\enterprise\DESKTOPHOST.exe"`,
			want:     true,
		},
		{
			name:          "exact override supersedes unrelated package family",
			path:          `C:\Enterprise\DesktopHost.exe`,
			packageFamily: `Contoso.CodexDesktop_1234567890abc`,
			override:      `C:\Enterprise\DesktopHost.exe`,
			want:          true,
		},
		{
			name:     "override is exact",
			path:     `C:\Enterprise\Other\DesktopHost.exe`,
			override: `C:\Enterprise\DesktopHost.exe`,
			want:     false,
		},
		{
			name: "unrelated executable under Codex directory",
			path: `C:\Program Files\OpenAI Codex\Other.exe`,
			want: false,
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			if got := isCodexProcess(item.path, item.packageFamily, item.override); got != item.want {
				t.Errorf("isCodexProcess(%q, %q, %q)=%v, want %v", item.path, item.packageFamily, item.override, got, item.want)
			}
		})
	}
}

func TestChooseCodexCandidate(t *testing.T) {
	tests := []struct {
		name       string
		candidates []windowCandidate
		want       HWND
	}{
		{
			name: "largest visible fallback",
			candidates: []windowCandidate{
				{handle: 11, area: 100},
				{handle: 22, area: 200},
			},
			want: 22,
		},
		{
			name: "zero handles ignored",
			candidates: []windowCandidate{
				{handle: 0, area: 500},
				{handle: 33, area: 100},
			},
			want: 33,
		},
		{name: "empty", want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := chooseCodexCandidate(test.candidates); got != test.want {
				t.Fatalf("chooseCodexCandidate()=%v, want %v", got, test.want)
			}
		})
	}
}

func TestFindCodexWindowReusesProcessCallback(t *testing.T) {
	original := enumerateWindows
	defer func() { enumerateWindows = original }()

	var callback uintptr
	calls := 0
	enumerateWindows = func(currentCallback, context uintptr) {
		calls++
		if callback == 0 {
			callback = currentCallback
		}
		if currentCallback != callback || currentCallback != enumWindowsCallback {
			t.Fatalf("EnumWindows callback changed: got %d, want %d", currentCallback, callback)
		}
		state, ok := enumWindowsStateForHandle(context)
		if !ok || state.executableOverride != "missing.exe" {
			t.Fatalf("unexpected enumeration state: %#v", state)
		}
	}

	for index := 0; index < 2500; index++ {
		if got := findCodexWindow("missing.exe"); got != 0 {
			t.Fatalf("findCodexWindow()=%d, want 0", got)
		}
	}
	if calls != 2500 {
		t.Fatalf("enumerateWindows calls=%d, want 2500", calls)
	}
	enumWindowsStates.RLock()
	remainingStates := len(enumWindowsStates.values)
	enumWindowsStates.RUnlock()
	if remainingStates != 0 {
		t.Fatalf("enumeration states retained after synchronous calls: %d", remainingStates)
	}
}

func TestEnumWindowsCallbackRejectsUnknownContext(t *testing.T) {
	if result := enumWindowsProc(0, ^uintptr(0)); result != 0 {
		t.Fatalf("unknown enumeration context result=%d, want 0", result)
	}
	state := &enumWindowsState{seen: make(map[HWND]struct{})}
	handle := registerEnumWindowsState(state)
	unregisterEnumWindowsState(handle)
	if result := enumWindowsProc(0, handle); result != 0 {
		t.Fatalf("deleted enumeration context result=%d, want 0", result)
	}
}

func TestEnumWindowsStateRegistryConcurrent(t *testing.T) {
	original := enumerateWindows
	defer func() { enumerateWindows = original }()

	errors := make(chan error, 1600)
	enumerateWindows = func(callback, context uintptr) {
		if callback != enumWindowsCallback {
			errors <- fmt.Errorf("unexpected callback: %d", callback)
			return
		}
		state, ok := enumWindowsStateForHandle(context)
		if !ok || state.executableOverride != "concurrent.exe" {
			errors <- fmt.Errorf("invalid context %d", context)
		}
	}

	var workers sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for iteration := 0; iteration < 100; iteration++ {
				findCodexWindow("concurrent.exe")
			}
		}()
	}
	workers.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}

	enumWindowsStates.RLock()
	remainingStates := len(enumWindowsStates.values)
	enumWindowsStates.RUnlock()
	if remainingStates != 0 {
		t.Fatalf("enumeration states retained after concurrent calls: %d", remainingStates)
	}
}

func TestTrackerFallbackSearchCooldown(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	findCalls := 0
	ops := testTrackerOps(&now)
	ops.find = func(string) HWND {
		findCalls++
		return 0
	}
	tracker := Tracker{ops: &ops}

	tracker.Snapshot()
	now = now.Add(500 * time.Millisecond)
	tracker.Snapshot()
	if findCalls != 1 {
		t.Fatalf("find calls within cooldown=%d, want 1", findCalls)
	}

	now = now.Add(500 * time.Millisecond)
	tracker.Snapshot()
	if findCalls != 2 {
		t.Fatalf("find calls after cooldown=%d, want 2", findCalls)
	}
}

func TestTrackerInvalidateAndOverrideClearCooldown(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	findCalls := 0
	var overrides []string
	ops := testTrackerOps(&now)
	ops.find = func(override string) HWND {
		findCalls++
		overrides = append(overrides, override)
		return 0
	}
	tracker := Tracker{ops: &ops}

	tracker.Target()
	tracker.Target()
	tracker.Invalidate()
	tracker.Target()
	tracker.SetExecutableOverride(`C:\Tools\Codex.exe`)
	tracker.Target()

	if findCalls != 3 {
		t.Fatalf("find calls=%d, want 3", findCalls)
	}
	if got := overrides[len(overrides)-1]; got != `c:\tools\codex.exe` {
		t.Fatalf("normalized override=%q", got)
	}
}

func TestTrackerForegroundRecoveryBypassesCooldown(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	foreground := HWND(0)
	findCalls := 0
	ops := testTrackerOps(&now)
	ops.foregroundCodex = func(string) HWND { return foreground }
	ops.find = func(string) HWND {
		findCalls++
		return 0
	}
	tracker := Tracker{ops: &ops}

	if snapshot := tracker.Snapshot(); snapshot.Valid {
		t.Fatal("missing target unexpectedly valid")
	}
	foreground = 42
	snapshot := tracker.Snapshot()
	if !snapshot.Valid || !snapshot.Foreground || snapshot.Target != 42 {
		t.Fatalf("foreground recovery snapshot=%+v", snapshot)
	}
	if findCalls != 1 {
		t.Fatalf("foreground recovery used fallback search: calls=%d", findCalls)
	}
}

func TestTrackerSnapshotReadsWindowStateOnce(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	foregroundCalls, rectCalls, dpiCalls, activeCalls := 0, 0, 0, 0
	ops := testTrackerOps(&now)
	ops.foregroundCodex = func(string) HWND {
		foregroundCalls++
		return 77
	}
	ops.readRect = func(hwnd HWND) (Rect, bool) {
		rectCalls++
		return Rect{Left: 1, Top: 2, Right: 301, Bottom: 202}, hwnd == 77
	}
	ops.dpi = func(hwnd HWND) uint32 {
		dpiCalls++
		return 144
	}
	ops.isForeground = func(hwnd HWND) bool {
		activeCalls++
		return hwnd == 77
	}
	tracker := Tracker{ops: &ops}

	snapshot := tracker.Snapshot()
	if !snapshot.Valid || snapshot.DPI != 144 || snapshot.Rect.Width() != 300 || !snapshot.Foreground {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if foregroundCalls != 1 || rectCalls != 1 || dpiCalls != 1 || activeCalls != 1 {
		t.Fatalf("snapshot calls foreground=%d rect=%d dpi=%d active=%d", foregroundCalls, rectCalls, dpiCalls, activeCalls)
	}
}

func testTrackerOps(now *time.Time) trackerOps {
	return trackerOps{
		now:             func() time.Time { return *now },
		foregroundCodex: func(string) HWND { return 0 },
		isUsable:        func(HWND) bool { return false },
		find:            func(string) HWND { return 0 },
		readRect: func(hwnd HWND) (Rect, bool) {
			return Rect{Left: 10, Top: 20, Right: 210, Bottom: 120}, hwnd != 0
		},
		isForeground: func(hwnd HWND) bool { return hwnd != 0 },
		dpi:          func(HWND) uint32 { return 96 },
	}
}
