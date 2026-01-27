package tui

import "github.com/charmbracelet/lipgloss"

// Fixed color palette (hacker theme only)
var (
	ColorBg      = lipgloss.Color("#000000") // Pure black
	ColorFg      = lipgloss.Color("#00FF00") // Bright green
	ColorPrimary = lipgloss.Color("#00FF00") // Bright green
	ColorDim     = lipgloss.Color("#006400") // Dark green
	ColorError   = lipgloss.Color("#FF0000") // Red
)

// Essential styles only
var (
	StyleBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDim)

	StyleBorderFocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary)

	StyleTitle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	StyleDimmed = lipgloss.NewStyle().
		Foreground(ColorDim)

	StyleTaskSelected = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	StyleTaskNormal = lipgloss.NewStyle().
		Foreground(ColorFg)

	StyleInput = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	StyleStatus = lipgloss.NewStyle().
		Foreground(ColorDim).
		Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
		Foreground(ColorDim).
		Padding(0, 1)

	StyleError = lipgloss.NewStyle().
		Foreground(ColorError)
)
