package ui

import (
	"codex-context-meter-lite/internal/config"
	"codex-context-meter-lite/internal/meter"
	"codex-context-meter-lite/internal/session"
)

type ViewState struct {
	Session  session.State
	Snapshot meter.Snapshot
}

type ActionKind int

const (
	ActionToggleDisplay ActionKind = iota
	ActionToggleUsed
	ActionToggleTheme
	ActionSetScale
	ActionSetPinned
	ActionToggleAutostart
	ActionMove
	ActionQuit
)

type Action struct {
	Kind     ActionKind
	Scale    float64
	ThreadID string
	Anchor   string
	OffsetX  int
	OffsetY  int
}

type ActionHandler func(Action)

type appConfig struct {
	config.Config
}
