package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	ShowUsed         bool    `json:"show_used"`
	Theme            string  `json:"theme"`
	Scale            float64 `json:"scale"`
	Anchor           string  `json:"anchor"`
	OffsetX          int     `json:"offset_x"`
	OffsetY          int     `json:"offset_y"`
	PinnedThreadID   string  `json:"pinned_thread_id,omitempty"`
	StartWithWindows bool    `json:"start_with_windows"`
	OverlayVisible   bool    `json:"overlay_visible"`
}

func Default() Config {
	return Config{Theme: "dark", Scale: 1, Anchor: "bottom-right", OffsetX: 12, OffsetY: 12, OverlayVisible: true}
}

func Dir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}
	return filepath.Join(base, "CodexContextMeterLite"), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load(path string) (Config, error) {
	result := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return Default(), err
	}
	result.Normalize()
	return result, nil
}

func Save(path string, value Config) error {
	value.Normalize()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (c *Config) Normalize() {
	if c.Theme != "light" {
		c.Theme = "dark"
	}
	if c.Scale < 0.8 || c.Scale > 1.4 {
		c.Scale = 1
	}
	if c.Anchor != "bottom-left" && c.Anchor != "top-right" && c.Anchor != "top-left" {
		c.Anchor = "bottom-right"
	}
	if c.OffsetX < 0 || c.OffsetX > 2000 {
		c.OffsetX = 12
	}
	if c.OffsetY < 0 || c.OffsetY > 2000 {
		c.OffsetY = 12
	}
}
