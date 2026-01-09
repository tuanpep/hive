package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorBg      = lipgloss.Color("#080808")
	ColorFg      = lipgloss.Color("#D1D1D1")
	ColorNeon    = lipgloss.Color("#00FF9C") // Cyber Green
	ColorBlue    = lipgloss.Color("#00E5FF") // Neon Blue
	ColorPink    = lipgloss.Color("#FF007A") // Neon Pink (Errors)
	ColorBorder  = lipgloss.Color("#333333") // Dim Grey
	ColorDimmed  = lipgloss.Color("#666666")
	ColorSuccess = lipgloss.Color("#00B894")

	// Styles
	StyleShell = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorFg)

	StyleHeader = lipgloss.NewStyle().
			Background(ColorBorder).
			Foreground(ColorNeon).
			Bold(true).
			Padding(0, 1)

	StylePaneBorder = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder)

	StylePaneBorderFocus = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorNeon)

	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorNeon).
			Bold(true).
			MarginLeft(1)

	StyleTaskSelected = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(ColorNeon).
				PaddingLeft(1).
				Foreground(ColorNeon)

	StyleTaskDimmed = lipgloss.NewStyle().
			Foreground(ColorDimmed).
			PaddingLeft(2)

	StyleDimmed = lipgloss.NewStyle().
			Foreground(ColorDimmed)

	StyleInputPrefix = lipgloss.NewStyle().
				Foreground(ColorBlue).
				Bold(true)

	StyleNeon = lipgloss.NewStyle().
			Foreground(ColorNeon)

	StyleStatusPending = lipgloss.NewStyle().Foreground(ColorDimmed)
	StyleStatusActive  = lipgloss.NewStyle().Foreground(ColorNeon)
	StyleStatusDone    = lipgloss.NewStyle().Foreground(ColorBlue)
	StyleStatusFailed  = lipgloss.NewStyle().Foreground(ColorPink)

	StyleModal = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorNeon).
			Padding(1, 4).
			Background(ColorBg).
			Align(lipgloss.Center)

	StyleGridLabel = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorNeon).
			Bold(true).
			Padding(0, 1)
)
