package config

import (
	"fmt"
	"strconv"

	"golang.org/x/sys/windows/registry"
)

const (
	autostartKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
	autostartName = "CodexContextMeterLite"
)

func SetAutostart(enabled bool, executable string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(autostartName); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}
	if executable == "" {
		return fmt.Errorf("empty executable path")
	}
	return key.SetStringValue(autostartName, strconv.Quote(executable))
}

func AutostartEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetStringValue(autostartName)
	return err == nil
}
