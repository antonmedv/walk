package main

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	keyForceQuit      = key.NewBinding(key.WithKeys(tea.KeyCtrlC.String()))
	keyQuit           = key.NewBinding(key.WithKeys(tea.KeyEscape.String()))
	keyOpen           = key.NewBinding(key.WithKeys(tea.KeyEnter.String()))
	keyBack           = key.NewBinding(key.WithKeys(tea.KeyBackspace.String()))
	keyFnDelete       = key.NewBinding(key.WithKeys(tea.KeyDelete.String()))
	keyTop            = key.NewBinding(key.WithKeys(tea.KeyShiftUp.String()))
	keyBottom         = key.NewBinding(key.WithKeys(tea.KeyShiftDown.String()))
	keyLeftmost       = key.NewBinding(key.WithKeys(tea.KeyShiftLeft.String()))
	keyRightmost      = key.NewBinding(key.WithKeys(tea.KeyShiftRight.String()))
	keyPageUp         = key.NewBinding(key.WithKeys(tea.KeyPgUp.String()))
	keyPageDown       = key.NewBinding(key.WithKeys(tea.KeyPgDown.String()))
	keyHome           = key.NewBinding(key.WithKeys(tea.KeyHome.String()))
	keyEnd            = key.NewBinding(key.WithKeys(tea.KeyEnd.String()))
	keyPreview        = key.NewBinding(key.WithKeys(tea.KeySpace.String()))
	keyUp             = key.NewBinding(key.WithKeys(tea.KeyUp.String(), "k"))
	keyDown           = key.NewBinding(key.WithKeys(tea.KeyDown.String(), "j"))
	keyLeft           = key.NewBinding(key.WithKeys(tea.KeyLeft.String(), "h"))
	keyRight          = key.NewBinding(key.WithKeys(tea.KeyRight.String(), "l"))
	keyQuitQ          = key.NewBinding(key.WithKeys("q"))
	keyVimTop         = key.NewBinding(key.WithKeys("g"))
	keyVimBottom      = key.NewBinding(key.WithKeys("G"))
	keySearch         = key.NewBinding(key.WithKeys("/"))
	keyDelete         = key.NewBinding(key.WithKeys("d"))
	keyUndo           = key.NewBinding(key.WithKeys("u"))
	keyYank           = key.NewBinding(key.WithKeys("y"))
	keyHidden         = key.NewBinding(key.WithKeys("."))
	keyHelp           = key.NewBinding(key.WithKeys("?"))
	keySelect         = key.NewBinding(key.WithKeys("v"))
	keyClearSelection = key.NewBinding(key.WithKeys("c"))
)
