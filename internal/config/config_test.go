package config

import (
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := Default()
	want.ShowUsed = true
	want.PinnedThreadID = "thread-1"
	want.Scale = 1.2
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestNormalizeRejectsUnsafeValues(t *testing.T) {
	value := Config{Theme: "unknown", Scale: 9, Anchor: "middle", OffsetX: -1, OffsetY: 9000}
	value.Normalize()
	if value.Theme != "dark" || value.Scale != 1 || value.Anchor != "bottom-right" || value.OffsetX != 12 || value.OffsetY != 12 {
		t.Fatalf("unexpected normalized config: %+v", value)
	}
}
