package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors - Opencode Theme
	ColorBg      = lipgloss.Color("#0a0a0a") // Very dark gray
	ColorFg      = lipgloss.Color("#ECEFF4") // Whiteish
	ColorPrimary = lipgloss.Color("#61AFEF") // Cyan/Blue
	ColorSecondary = lipgloss.Color("#5C6370") // Slate Gray
	ColorSuccess = lipgloss.Color("#98C379") // Green
	ColorError   = lipgloss.Color("#E06C75") // Red
	ColorWarning = lipgloss.Color("#E5C07B") // Yellow
	ColorDimmed  = lipgloss.Color("#4B5263")

	// Styles
	StyleShell = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorFg)

	StyleHeader = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	StylePaneBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary)

	StylePaneBorderFocus = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary)

	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			MarginLeft(1)

	StyleTaskSelected = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(ColorPrimary).
				PaddingLeft(1).
				Foreground(ColorPrimary)

	StyleTaskDimmed = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			PaddingLeft(2)

	StyleDimmed = lipgloss.NewStyle().
			Foreground(ColorDimmed)

	StyleInputPrefix = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	StylePrimary = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	StyleStatusPending = lipgloss.NewStyle().Foreground(ColorDimmed)
	StyleStatusActive  = lipgloss.NewStyle().Foreground(ColorPrimary)
	StyleStatusDone    = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleStatusFailed  = lipgloss.NewStyle().Foreground(ColorError)

	StyleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 4).
			Background(ColorBg).
			Align(lipgloss.Center)

	StyleGridLabel = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorPrimary).
			Bold(true).
			Padding(0, 1)
	
	StyleLogo = lipgloss.NewStyle().
			Foreground(ColorSecondary)
)
