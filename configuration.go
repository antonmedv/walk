package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
)

type LlamaConfiguration struct {
	Bindings []BindingConfiguration
}

type BindingConfiguration struct {
	Action   string
	Keys     []string
	Disabled bool
	Help     key.Help
}

var (
	keyMap = map[string]*key.Binding{
		"keyForceQuit": &keyForceQuit,
		"keyQuit":      &keyQuit,
		"keyOpen":      &keyOpen,
		"keyBack":      &keyBack,
		"keyUp":        &keyUp,
		"keyDown":      &keyDown,
		"keyLeft":      &keyLeft,
		"keyRight":     &keyRight,
		"keyTop":       &keyTop,
		"keyBottom":    &keyBottom,
		"keyLeftmost":  &keyLeftmost,
		"keyRightmost": &keyRightmost,
		"keyVimUp":     &keyVimUp,
		"keyVimDown":   &keyVimDown,
		"keyVimLeft":   &keyVimLeft,
		"keyVimRight":  &keyVimRight,
		"keyVimTop":    &keyVimTop,
		"keyVimBottom": &keyVimBottom,
		"keySearch":    &keySearch,
		"keyPreview":   &keyPreview,
		"keyDelete":    &keyDelete,
		"keyUndo":      &keyUndo,
	}
)

func processConfig() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	configDir := filepath.Join(homeDir, ".config", "llama", "config.json")

	file, err := os.Open(configDir)
	if err != nil {
		return // Don't load configuration
	}
	defer file.Close()

	configBytes, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	var llamaConfig LlamaConfiguration
	err = json.Unmarshal(configBytes, &llamaConfig)
	if err != nil {
		panic(err)
	}

	for _, bindingConfig := range llamaConfig.Bindings {
		if binding, exists := keyMap[bindingConfig.Action]; exists {
			binding.SetKeys(bindingConfig.Keys...)
			binding.SetHelp(bindingConfig.Help.Key, bindingConfig.Help.Desc)
			binding.SetEnabled(!bindingConfig.Disabled)
		}
	}
}
