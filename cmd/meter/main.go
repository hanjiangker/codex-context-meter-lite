package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"codex-context-meter-lite/internal/config"
	"codex-context-meter-lite/internal/meter"
	"codex-context-meter-lite/internal/session"
	"codex-context-meter-lite/internal/ui"

	"golang.org/x/sys/windows"
)

const version = "0.1.0"

var defaultDemo = "false"

func main() {
	rootFlag := flag.String("sessions", "", "override Codex sessions directory")
	indexFlag := flag.String("session-index", "", "override Codex session index path")
	codexExeFlag := flag.String("codex-exe", "", "override Codex executable path")
	showVersion := flag.Bool("version", false, "print version")
	demo := flag.Bool("demo", defaultDemo == "true", "show synthetic data in a standalone verification window")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	debug.SetGCPercent(50)
	debug.SetMemoryLimit(20 << 20)
	mutexName := `Local\CodexContextMeterLite`
	if *demo {
		mutexName += "Demo"
	}
	mutex, alreadyRunning, err := singleInstance(mutexName)
	if err != nil {
		showError(err)
		return
	}
	if alreadyRunning {
		return
	}
	defer windows.CloseHandle(mutex)

	configPath, err := config.Path()
	if err != nil {
		showError(err)
		return
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.Default()
	}
	cfg.StartWithWindows = config.AutostartEnabled()

	root, indexPath, err := session.ResolvePaths(*rootFlag, *indexFlag)
	if err != nil {
		showError(err)
		return
	}
	manager := session.NewManager(root, indexPath, cfg.PinnedThreadID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var cfgMu sync.Mutex
	var app *ui.App
	handler := func(action ui.Action) {
		cfgMu.Lock()
		defer cfgMu.Unlock()
		switch action.Kind {
		case ui.ActionToggleDisplay:
			cfg.OverlayVisible = !cfg.OverlayVisible
		case ui.ActionToggleUsed:
			cfg.ShowUsed = !cfg.ShowUsed
		case ui.ActionToggleTheme:
			if cfg.Theme == "light" {
				cfg.Theme = "dark"
			} else {
				cfg.Theme = "light"
			}
		case ui.ActionSetScale:
			cfg.Scale = action.Scale
		case ui.ActionSetPinned:
			cfg.PinnedThreadID = action.ThreadID
			manager.SetPinned(action.ThreadID)
		case ui.ActionToggleAutostart:
			executable, exeErr := os.Executable()
			if exeErr == nil {
				next := !cfg.StartWithWindows
				if startErr := config.SetAutostart(next, executable); startErr == nil {
					cfg.StartWithWindows = next
				} else {
					showError(startErr)
				}
			}
		case ui.ActionMove:
			if action.Anchor != "" {
				cfg.Anchor = action.Anchor
			}
			cfg.OffsetX, cfg.OffsetY = action.OffsetX, action.OffsetY
		case ui.ActionQuit:
			cancel()
			app.Quit()
			return
		}
		if saveErr := config.Save(configPath, cfg); saveErr != nil {
			showError(saveErr)
		}
		app.SetConfig(cfg)
	}

	app = ui.New(cfg, handler, *demo)
	app.SetCodexExecutable(*codexExeFlag)
	if *demo {
		app.SetView(demoView())
	} else {
		go refreshLoop(ctx, manager, app)
	}
	if err := app.Run(); err != nil {
		showError(err)
	}
}

func refreshLoop(ctx context.Context, manager *session.Manager, app *ui.App) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	refresh := func() {
		state := manager.Refresh()
		var snapshot meter.Snapshot
		if state.Selected != nil {
			snapshot = meter.Build(state.Selected.Info, meter.DefaultThresholds())
		}
		app.SetView(ui.ViewState{Session: state, Snapshot: snapshot})
	}
	refresh()
	debug.FreeOSMemory()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresh()
		}
	}
}

func singleInstance(mutexName string) (windows.Handle, bool, error) {
	name, _ := windows.UTF16PtrFromString(mutexName)
	handle, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return handle, true, nil
	}
	return handle, false, err
}

func demoView() ui.ViewState {
	now := time.Now()
	info := meter.TokenInfo{
		LastUsage:          meter.Usage{InputTokens: 141_200, CachedInputTokens: 137_800, OutputTokens: 1_300, ReasoningOutputTokens: 420, TotalTokens: 142_500},
		TotalUsage:         meter.Usage{InputTokens: 2_840_000, CachedInputTokens: 2_610_000, OutputTokens: 42_800, ReasoningOutputTokens: 18_400, TotalTokens: 2_882_800},
		ModelContextWindow: 353_400,
	}
	history := make([]meter.HistoryPoint, 0, 12)
	for index := 0; index < 12; index++ {
		history = append(history, meter.HistoryPoint{Time: now.Add(time.Duration(index-11) * 5 * time.Minute), UsedTokens: int64(42_000 + index*9_000), UsedPercent: 12 + float64(index)*2.5})
	}
	summary := &session.Summary{ThreadID: "demo", Title: "Codex context meter visual verification", UpdatedAt: now, Info: info}
	state := session.State{Selected: summary, Candidates: []session.Candidate{{ThreadID: "demo", Title: summary.Title, UpdatedAt: now}}, History: history}
	return ui.ViewState{Session: state, Snapshot: meter.Build(info, meter.DefaultThresholds())}
}

func showError(err error) {
	if err == nil {
		return
	}
	text, _ := windows.UTF16PtrFromString(err.Error())
	title, _ := windows.UTF16PtrFromString("Codex Context Meter Lite")
	windows.MessageBox(0, text, title, 0x10)
}
