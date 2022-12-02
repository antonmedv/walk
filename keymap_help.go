package main

import "github.com/charmbracelet/bubbles/key"

type helpKeymap struct{}

var HelpKeymap = helpKeymap{}

func (h helpKeymap) ShortHelp() []key.Binding {
	return []key.Binding{keyQuit, keyHelp}
}

func (h helpKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{keySearch, keyHelp, keyQuit},
		{keyUp, keyDown, keyLeft, keyRight},
		{keyOpen, keyPreview, keyBack},
		{keyDelete, keyUndo},
	}
}
