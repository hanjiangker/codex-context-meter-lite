package ui

import (
	"testing"

	"codex-context-meter-lite/internal/meter"
)

func TestFormatCacheLabel(t *testing.T) {
	tests := []struct {
		name  string
		usage meter.Usage
		want  string
	}{
		{name: "turn", usage: meter.Usage{InputTokens: 141_200, CachedInputTokens: 137_800}, want: "CACHE 97.6%"},
		{name: "session", usage: meter.Usage{InputTokens: 2_840_000, CachedInputTokens: 2_610_000}, want: "CACHE 91.9%"},
		{name: "zero", usage: meter.Usage{InputTokens: 100}, want: "CACHE 0.0%"},
		{name: "unavailable", usage: meter.Usage{}, want: "CACHE --"},
		{name: "negative cache", usage: meter.Usage{InputTokens: 100, CachedInputTokens: -1}, want: "CACHE --"},
		{name: "clamped", usage: meter.Usage{InputTokens: 100, CachedInputTokens: 120}, want: "CACHE 100%"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := formatCacheLabel(test.usage)
			if got != test.want {
				t.Fatalf("formatCacheLabel() = %q, want %q", got, test.want)
			}
			if len(got) > 11 {
				t.Fatalf("cache label %q exceeds the metric cell capacity", got)
			}
		})
	}
}

func TestPanelHeight(t *testing.T) {
	tests := []struct {
		name    string
		hovered bool
		known   bool
		want    int32
	}{
		{name: "collapsed waiting", hovered: false, known: false, want: panelCollapsedHeight},
		{name: "collapsed known", hovered: false, known: true, want: panelCollapsedHeight},
		{name: "expanded waiting", hovered: true, known: false, want: panelExpandedWaitingHeight},
		{name: "expanded known", hovered: true, known: true, want: panelExpandedHeight},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := panelHeight(test.hovered, test.known); got != test.want {
				t.Fatalf("panelHeight(%v, %v) = %d, want %d", test.hovered, test.known, got, test.want)
			}
		})
	}
}

func TestDiagnosticStatusLabels(t *testing.T) {
	windowStatus, sessionStatus := diagnosticStatusLabels(true, false)
	if windowStatus != "Codex window: detected" || sessionStatus != "Session data: not detected" {
		t.Fatalf("unexpected diagnostic status: %q, %q", windowStatus, sessionStatus)
	}

	windowStatus, sessionStatus = diagnosticStatusLabels(false, true)
	if windowStatus != "Codex window: not detected" || sessionStatus != "Session data: detected" {
		t.Fatalf("unexpected diagnostic status: %q, %q", windowStatus, sessionStatus)
	}
}

func TestContextFillWidthIsContinuous(t *testing.T) {
	tests := []struct {
		width   int32
		percent float64
		want    int32
	}{
		{width: 332, percent: -1, want: 0},
		{width: 332, percent: 21.9, want: 73},
		{width: 332, percent: 59.7, want: 198},
		{width: 332, percent: 100, want: 332},
		{width: 332, percent: 120, want: 332},
	}
	for _, test := range tests {
		if got := contextFillWidth(test.width, test.percent); got != test.want {
			t.Fatalf("contextFillWidth(%d, %.1f) = %d, want %d", test.width, test.percent, got, test.want)
		}
	}
}

func TestMetricPaletteUsesDistinctColors(t *testing.T) {
	for _, dark := range []bool{true, false} {
		colors := metricPalette(dark)
		values := []uint32{colors.total, colors.input, colors.cache, colors.output, colors.reason}
		seen := make(map[uint32]bool, len(values))
		for _, value := range values {
			if value == 0 || seen[value] {
				t.Fatalf("metric palette dark=%v contains an empty or duplicate color: %#x", dark, value)
			}
			seen[value] = true
		}
	}
}

func TestContextPressureColorThresholds(t *testing.T) {
	green := rgb(22, 199, 132)
	amber := rgb(255, 170, 32)
	red := rgb(231, 72, 79)
	tests := []struct {
		percent float64
		want    uint32
	}{
		{percent: 0, want: green},
		{percent: 75, want: green},
		{percent: 75.01, want: amber},
		{percent: 85, want: amber},
		{percent: 85.01, want: red},
		{percent: 100, want: red},
	}
	for _, test := range tests {
		if got := contextPressureColor(test.percent); got != test.want {
			t.Fatalf("contextPressureColor(%.2f) = %#x, want %#x", test.percent, got, test.want)
		}
	}
}

func TestNearestAnchorAndOffsets(t *testing.T) {
	target := rect{Left: 100, Top: 100, Right: 1100, Bottom: 900}
	tests := []struct {
		name    string
		current rect
		anchor  string
		x       int32
		y       int32
	}{
		{name: "top right", current: rect{Left: 728, Top: 112, Right: 1088, Bottom: 262}, anchor: "top-right", x: 12, y: 12},
		{name: "top left", current: rect{Left: 114, Top: 110, Right: 474, Bottom: 260}, anchor: "top-left", x: 14, y: 10},
		{name: "bottom right", current: rect{Left: 728, Top: 736, Right: 1088, Bottom: 886}, anchor: "bottom-right", x: 12, y: 14},
		{name: "bottom left", current: rect{Left: 109, Top: 738, Right: 469, Bottom: 888}, anchor: "bottom-left", x: 9, y: 12},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			anchor := nearestAnchor(target, test.current)
			if anchor != test.anchor {
				t.Fatalf("nearestAnchor() = %q, want %q", anchor, test.anchor)
			}
			x, y := anchorOffsets(anchor, target, test.current)
			if x != test.x || y != test.y {
				t.Fatalf("anchorOffsets() = (%d, %d), want (%d, %d)", x, y, test.x, test.y)
			}
		})
	}
}

func TestTopAnchorKeepsTopEdgeWhileHeightChanges(t *testing.T) {
	target := rect{Left: 100, Top: 100, Right: 1100, Bottom: 900}
	_, collapsedY := anchoredPosition("top-right", target, 360, panelCollapsedHeight, 12, 14)
	_, expandedY := anchoredPosition("top-right", target, 360, panelExpandedHeight, 12, 14)
	if collapsedY != 114 || expandedY != collapsedY {
		t.Fatalf("top anchor moved during expansion: collapsed=%d expanded=%d", collapsedY, expandedY)
	}

	_, collapsedBottomY := anchoredPosition("bottom-right", target, 360, panelCollapsedHeight, 12, 14)
	_, expandedBottomY := anchoredPosition("bottom-right", target, 360, panelExpandedHeight, 12, 14)
	if collapsedBottomY+panelCollapsedHeight != expandedBottomY+panelExpandedHeight {
		t.Fatalf("bottom anchor did not preserve bottom edge: collapsed=%d expanded=%d", collapsedBottomY, expandedBottomY)
	}
}

func TestRoundedRectCoverageAntialiasesBothBarHeights(t *testing.T) {
	for _, height := range []int32{4, 9} {
		center := roundedRectCoverage(20, height/2, 332, height)
		if center != 16 {
			t.Fatalf("height %d center coverage = %d, want 16", height, center)
		}
		partialFound := false
		for y := int32(0); y < height; y++ {
			for x := int32(0); x < height; x++ {
				coverage := roundedRectCoverage(x, y, 332, height)
				if coverage > 0 && coverage < 16 {
					partialFound = true
					opposite := roundedRectCoverage(331-x, height-1-y, 332, height)
					if opposite != coverage {
						t.Fatalf("height %d asymmetric coverage at (%d,%d): %d and %d", height, x, y, coverage, opposite)
					}
				}
			}
		}
		if !partialFound {
			t.Fatalf("height %d has no partially covered edge pixels", height)
		}
	}
}

func TestBlendColorCoverageEndpoints(t *testing.T) {
	foreground := rgb(22, 199, 132)
	background := rgb(8, 12, 17)
	if got := blendColor(foreground, background, 0, 16); got != background {
		t.Fatalf("zero coverage = %#x, want background %#x", got, background)
	}
	if got := blendColor(foreground, background, 16, 16); got != foreground {
		t.Fatalf("full coverage = %#x, want foreground %#x", got, foreground)
	}
	partial := blendColor(foreground, background, 8, 16)
	if partial == foreground || partial == background {
		t.Fatalf("partial coverage did not blend: %#x", partial)
	}
}
