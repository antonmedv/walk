package main

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	mainColor    = lipgloss.Color("#825DF2")
	barColor     = lipgloss.Color("#5C5C5C")
	searchColor  = lipgloss.Color("#499F1C")
	bold         lipgloss.Style
	warning      lipgloss.Style
	cursor       lipgloss.Style
	bar          lipgloss.Style
	search       lipgloss.Style
	danger       lipgloss.Style
	previewPlain lipgloss.Style
	previewSplit lipgloss.Style
)

func initStyles() {
	bold = lipgloss.NewStyle().Bold(true)
	warning = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).PaddingLeft(1).PaddingRight(1)
	cursor = lipgloss.NewStyle().Background(mainColor).Foreground(lipgloss.Color("#FFFFFF"))
	bar = lipgloss.NewStyle().Background(barColor).Foreground(lipgloss.Color("#FFFFFF"))
	search = lipgloss.NewStyle().Background(searchColor).Foreground(lipgloss.Color("#FFFFFF"))
	danger = lipgloss.NewStyle().Background(lipgloss.Color("#FF0000")).Foreground(lipgloss.Color("#FFFFFF"))
	previewPlain = lipgloss.NewStyle().PaddingLeft(2)
	previewSplit = lipgloss.NewStyle().
		MarginLeft(1).
		PaddingLeft(1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(mainColor).
		BorderLeft(true)
}
