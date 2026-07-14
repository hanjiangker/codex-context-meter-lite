package meter

import (
	"math"
	"time"
)

type Usage struct {
	InputTokens           int64 `json:"input_tokens"`
	CachedInputTokens     int64 `json:"cached_input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
	TotalTokens           int64 `json:"total_tokens"`
}

func (usage Usage) CacheHitPercent() (float64, bool) {
	if usage.InputTokens <= 0 || usage.CachedInputTokens < 0 {
		return 0, false
	}
	percent := float64(usage.CachedInputTokens) / float64(usage.InputTokens) * 100
	return clamp(percent, 0, 100), true
}

type TokenInfo struct {
	TotalUsage         Usage `json:"total_token_usage"`
	LastUsage          Usage `json:"last_token_usage"`
	ModelContextWindow int64 `json:"model_context_window"`
}

type Level string

const (
	LevelNormal   Level = "normal"
	LevelNotice   Level = "notice"
	LevelWarn     Level = "warn"
	LevelDanger   Level = "danger"
	LevelCritical Level = "critical"
)

type Thresholds struct {
	NoticeLeftPercent   float64 `json:"notice_left_percent"`
	WarnLeftPercent     float64 `json:"warn_left_percent"`
	DangerLeftPercent   float64 `json:"danger_left_percent"`
	CriticalLeftPercent float64 `json:"critical_left_percent"`
	CompressionLeft     float64 `json:"compression_left_percent"`
}

func DefaultThresholds() Thresholds {
	return Thresholds{NoticeLeftPercent: 60, WarnLeftPercent: 50, DangerLeftPercent: 40, CriticalLeftPercent: 30, CompressionLeft: 20}
}

type Snapshot struct {
	Known              bool
	UsedTokens         int64
	ContextWindow      int64
	UsedPercent        float64
	LeftPercent        float64
	RemainingTokens    int64
	Level              Level
	CompressionWarning bool
	Turn               Usage
	Session            Usage
}

type HistoryPoint struct {
	Time        time.Time `json:"time"`
	UsedTokens  int64     `json:"used_tokens"`
	UsedPercent float64   `json:"used_percent"`
	Reset       bool      `json:"reset,omitempty"`
}

func Build(info TokenInfo, thresholds Thresholds) Snapshot {
	if info.ModelContextWindow <= 0 || info.LastUsage.TotalTokens < 0 {
		return Snapshot{Turn: info.LastUsage, Session: info.TotalUsage}
	}

	used := min(info.LastUsage.TotalTokens, info.ModelContextWindow)
	usedPercent := clamp(float64(used)/float64(info.ModelContextWindow)*100, 0, 100)
	leftPercent := 100 - usedPercent
	return Snapshot{
		Known:              true,
		UsedTokens:         used,
		ContextWindow:      info.ModelContextWindow,
		UsedPercent:        usedPercent,
		LeftPercent:        leftPercent,
		RemainingTokens:    max(0, info.ModelContextWindow-used),
		Level:              LevelForLeft(leftPercent, thresholds),
		CompressionWarning: leftPercent <= thresholds.CompressionLeft,
		Turn:               info.LastUsage,
		Session:            info.TotalUsage,
	}
}

func LevelForLeft(left float64, thresholds Thresholds) Level {
	switch {
	case left <= thresholds.CriticalLeftPercent:
		return LevelCritical
	case left <= thresholds.DangerLeftPercent:
		return LevelDanger
	case left <= thresholds.WarnLeftPercent:
		return LevelWarn
	case left <= thresholds.NoticeLeftPercent:
		return LevelNotice
	default:
		return LevelNormal
	}
}

func ToHistoryPoint(at time.Time, info TokenInfo, previous *HistoryPoint) HistoryPoint {
	snapshot := Build(info, DefaultThresholds())
	point := HistoryPoint{Time: at, UsedTokens: snapshot.UsedTokens, UsedPercent: snapshot.UsedPercent}
	if previous != nil && point.UsedTokens < previous.UsedTokens {
		point.Reset = true
	}
	return point
}

func clamp(value, low, high float64) float64 {
	return math.Max(low, math.Min(high, value))
}
