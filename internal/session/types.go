package session

import (
	"time"

	"codex-context-meter-lite/internal/meter"
)

type Summary struct {
	Path        string
	ThreadID    string
	Title       string
	CWD         string
	Provider    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Info        meter.TokenInfo
	FileSize    int64
	FileModTime time.Time
}

type Candidate struct {
	ThreadID  string
	Title     string
	UpdatedAt time.Time
}

type State struct {
	Selected   *Summary
	Candidates []Candidate
	History    []meter.HistoryPoint
	Pinned     bool
	Error      string
}

type TokenEvent struct {
	Time time.Time
	Info meter.TokenInfo
}
