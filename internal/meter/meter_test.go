package meter

import (
	"math"
	"testing"
	"time"
)

func TestUsageCacheHitPercent(t *testing.T) {
	tests := []struct {
		name  string
		usage Usage
		want  float64
		ok    bool
	}{
		{name: "turn", usage: Usage{InputTokens: 141_200, CachedInputTokens: 137_800}, want: 97.59206798866856, ok: true},
		{name: "session", usage: Usage{InputTokens: 2_840_000, CachedInputTokens: 2_610_000}, want: 91.90140845070422, ok: true},
		{name: "no cache", usage: Usage{InputTokens: 100}, want: 0, ok: true},
		{name: "no input", usage: Usage{}, ok: false},
		{name: "negative cache", usage: Usage{InputTokens: 100, CachedInputTokens: -1}, ok: false},
		{name: "cache exceeds input", usage: Usage{InputTokens: 100, CachedInputTokens: 120}, want: 100, ok: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := test.usage.CacheHitPercent()
			if ok != test.ok || math.Abs(got-test.want) > 1e-9 {
				t.Fatalf("CacheHitPercent() = (%v, %v), want (%v, %v)", got, ok, test.want, test.ok)
			}
		})
	}
}

func TestBuildUsesTotalTokensForContextPressure(t *testing.T) {
	snapshot := Build(TokenInfo{
		LastUsage:          Usage{InputTokens: 550, OutputTokens: 50, TotalTokens: 600},
		TotalUsage:         Usage{TotalTokens: 1600},
		ModelContextWindow: 1000,
	}, DefaultThresholds())

	if !snapshot.Known || snapshot.UsedPercent != 60 || snapshot.LeftPercent != 40 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if snapshot.UsedTokens != 600 || snapshot.RemainingTokens != 400 {
		t.Fatalf("unexpected token counts: %+v", snapshot)
	}
	if snapshot.Level != LevelDanger {
		t.Fatalf("expected danger, got %s", snapshot.Level)
	}
}

func TestBuildClampsUsageToWindow(t *testing.T) {
	snapshot := Build(TokenInfo{LastUsage: Usage{TotalTokens: 1200}, ModelContextWindow: 1000}, DefaultThresholds())
	if snapshot.UsedPercent != 100 || snapshot.RemainingTokens != 0 || !snapshot.CompressionWarning {
		t.Fatalf("unexpected clamped snapshot: %+v", snapshot)
	}
}

func TestHistoryMarksCompaction(t *testing.T) {
	first := ToHistoryPoint(time.Now(), TokenInfo{LastUsage: Usage{TotalTokens: 800}, ModelContextWindow: 1000}, nil)
	second := ToHistoryPoint(time.Now(), TokenInfo{LastUsage: Usage{TotalTokens: 300}, ModelContextWindow: 1000}, &first)
	if !second.Reset {
		t.Fatal("expected compaction reset")
	}
}
