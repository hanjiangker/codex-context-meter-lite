package meter

import (
	"testing"
	"time"
)

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
