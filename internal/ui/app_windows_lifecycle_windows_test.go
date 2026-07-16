package ui

import (
	"sync"
	"sync/atomic"
	"testing"

	"codex-context-meter-lite/internal/config"
	appwindow "codex-context-meter-lite/internal/window"
)

func TestAppPostsOnlyWhileWindowIsPublished(t *testing.T) {
	app := New(configForLifecycleTest(), nil, false)
	var posted []struct {
		hwnd    HWND
		message uint32
	}
	app.postWindowMessage = func(hwnd HWND, message uint32) {
		posted = append(posted, struct {
			hwnd    HWND
			message uint32
		}{hwnd: hwnd, message: message})
	}

	app.postMessage(wmAppUpdate)
	app.publishWindow(41)
	app.postMessage(wmAppUpdate)
	app.clearWindow(99)
	app.postMessage(wmClose)
	app.clearWindow(41)
	app.postMessage(wmAppUpdate)

	if len(posted) != 2 {
		t.Fatalf("posted messages=%d, want 2", len(posted))
	}
	if posted[0].hwnd != 41 || posted[0].message != wmAppUpdate {
		t.Fatalf("first post=%+v", posted[0])
	}
	if posted[1].hwnd != 41 || posted[1].message != wmClose {
		t.Fatalf("second post=%+v", posted[1])
	}
	if hwnd := app.windowHandle(); hwnd != 0 {
		t.Fatalf("window handle after clear=%d", hwnd)
	}
}

func TestAppWindowLifecycleConcurrentAccess(t *testing.T) {
	app := New(configForLifecycleTest(), nil, false)
	var posts atomic.Int64
	app.postWindowMessage = func(HWND, uint32) { posts.Add(1) }
	app.publishWindow(51)

	const workerCount = 16
	ready := make(chan struct{}, workerCount)
	stop := make(chan struct{})
	var workers sync.WaitGroup
	for index := 0; index < workerCount; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			ready <- struct{}{}
			for {
				select {
				case <-stop:
					return
				default:
				}
				app.postMessage(wmAppUpdate)
				_ = app.windowHandle()
			}
		}()
	}
	for index := 0; index < workerCount; index++ {
		<-ready
	}

	lifecycleDone := make(chan struct{})
	go func() {
		defer close(lifecycleDone)
		for iteration := 0; iteration < 1000; iteration++ {
			app.clearWindow(51)
			app.publishWindow(51)
		}
		app.clearWindow(51)
	}()
	<-lifecycleDone
	close(stop)
	workers.Wait()

	before := posts.Load()
	app.postMessage(wmAppUpdate)
	if posts.Load() != before {
		t.Fatal("message posted after window was cleared")
	}
}

func TestTrackedOverlayVisibilityRequiresForeground(t *testing.T) {
	visible := appwindow.WindowSnapshot{Target: 42, Valid: true, Foreground: true}
	if !shouldShowTrackedOverlay(visible) {
		t.Fatal("valid foreground Codex window should show the overlay")
	}
	visible.Foreground = false
	if shouldShowTrackedOverlay(visible) {
		t.Fatal("background Codex window should hide the overlay")
	}
	visible.Foreground = true
	visible.Valid = false
	if shouldShowTrackedOverlay(visible) {
		t.Fatal("invalid Codex window should hide the overlay")
	}
}

func configForLifecycleTest() config.Config {
	return config.Default()
}
