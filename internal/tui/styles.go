package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	colorPrimary   = lipgloss.Color("#9aa3ad")
	colorSecondary = lipgloss.Color("#b4bcc5")
	colorSuccess   = lipgloss.Color("#7fb3a1")
	colorWarning   = lipgloss.Color("#c6a36e")
	colorError     = lipgloss.Color("#c07a6b")
	colorMuted     = lipgloss.Color("#7b7f85")
	colorFg        = lipgloss.Color("#e4e6e8")
	colorBg        = lipgloss.Color("#1e2023")
	colorBgLight   = lipgloss.Color("#26292e")
	colorBorder    = lipgloss.Color("#353a40")

	// Base styles
	baseStyle = lipgloss.NewStyle().Foreground(colorFg).Background(colorBg)

	// List styles
	listStyle         = lipgloss.NewStyle().Background(colorBg).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorBorder)
	listItemStyle     = lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
	listSelectedStyle = lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1).Background(colorBgLight).Foreground(colorPrimary).Bold(true)
	listConflictStyle = lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1).Foreground(colorError)
	listModifiedStyle = lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1).Foreground(colorWarning)
	listSyncedStyle   = lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1).Foreground(colorMuted)
	listHelpStyle     = lipgloss.NewStyle().Foreground(colorMuted).PaddingLeft(1)

	// Detail styles
	detailStyle      = lipgloss.NewStyle().Background(colorBg).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorBorder).Padding(1)
	detailTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).MarginBottom(1)
	detailMetaStyle  = lipgloss.NewStyle().Foreground(colorMuted).MarginBottom(1)
	detailLabelStyle = lipgloss.NewStyle().Foreground(colorMuted)
	detailValueStyle = lipgloss.NewStyle().Foreground(colorFg)
	detailTagStyle   = lipgloss.NewStyle().Foreground(colorSecondary).Background(colorBgLight).Padding(0, 1)
	commentSectionTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	commentMetaStyle = lipgloss.NewStyle().Foreground(colorMuted)
	commentBodyStyle = lipgloss.NewStyle().Foreground(colorFg)
	commentEmptyStyle = lipgloss.NewStyle().Foreground(colorMuted)
	commentDividerStyle = lipgloss.NewStyle().Foreground(colorBorder)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().Background(colorBgLight).Foreground(colorFg).Padding(0, 1)
	statusKeyStyle = lipgloss.NewStyle().Background(colorPrimary).Foreground(colorBg).Bold(true).Padding(0, 1)
	statusValStyle = lipgloss.NewStyle().Background(colorBgLight).Foreground(colorFg).Padding(0, 1)
	statusErrStyle = lipgloss.NewStyle().Background(colorError).Foreground(colorBg).Bold(true).Padding(0, 1)
	statusWarnStyle = lipgloss.NewStyle().Background(colorWarning).Foreground(colorBg).Bold(true).Padding(0, 1)
	statusOkStyle   = lipgloss.NewStyle().Background(colorSuccess).Foreground(colorBg).Bold(true).Padding(0, 1)

	// Input/filter
	inputStyle    = lipgloss.NewStyle().Background(colorBgLight).Foreground(colorFg).Padding(0, 1)
	inputPrompt   = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)

	// Board styles
	boardColumnStyle   = lipgloss.NewStyle().Background(colorBg).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorBorder).Padding(0, 1)
	boardColumnHeader  = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Align(lipgloss.Center)
	boardItemStyle     = lipgloss.NewStyle().Background(colorBgLight).Padding(0, 1).MarginTop(1)
	boardSelectedStyle = lipgloss.NewStyle().Background(colorPrimary).Foreground(colorBg).Padding(0, 1).MarginTop(1)

	// Palette
	paletteStyle     = lipgloss.NewStyle().Background(colorBgLight).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorPrimary).Padding(1)
	paletteMatchStyle = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)

	// Help
	helpStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

// width/height helpers for responsive layout
func listWidth(total int) int {
	// Left panel takes ~40% of width
	return total * 2 / 5
}

func detailWidth(total int) int {
	return total - listWidth(total) - 2 // account for borders
}

func boardColumnWidth(total int, cols int) int {
	if cols <= 0 {
		return total
	}
	gap := cols - 1
	available := total - gap*2
	if available < cols {
		return 1
	}
	return available / cols
}
