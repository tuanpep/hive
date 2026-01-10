package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ASCII Logo for Home Screen
var asciiLogo = []string{
	"██╗  ██╗██╗██╗   ██╗███████╗",
	"██║  ██║██║██║   ██║██╔════╝",
	"███████║██║██║   ██║█████╗  ",
	"██╔══██║██║╚██╗ ██╔╝██╔══╝  ",
	"██║  ██║██║ ╚████╔╝ ███████╗",
	"╚═╝  ╚═╝╚═╝  ╚═══╝  ╚══════╝",
}

func (m Model) View() string {
	if m.Width == 0 || !m.Ready {
		return "Initialising system..."
	}

	runningTasks := m.GetRunningTasks()
	activeCount := len(runningTasks)
	totalTasks := len(m.TaskList.Items())

	var content string

	if totalTasks == 0 {
		content = m.viewHome()
	} else {
		content = m.viewDashboard(activeCount, runningTasks)
	}

	if m.ShowModal {
		modal := StyleModal.Render(m.ModalContent)
		return m.overlay(content, modal)
	}

	return content
}

func (m Model) viewHome() string {
	// Logo
	logoBlock := lipgloss.JoinVertical(lipgloss.Center, asciiLogo...)
	logoStyled := StyleLogo.Render(logoBlock)

	// Helper text
	helpText := StyleDimmed.Render("Type a task title to start. Press [?] for help. [q] to quit.")

	// Input
	// Ensure input view width matches our box if possible, or leave as is.
	inputView := m.Input.View()

	// Box the input to look centered and prominent
	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(60).
		Render(inputView)

	// Suggestions Overlay
	suggestions := m.viewSuggestions()

	// Center Layout: Logo -> Suggestions -> Space -> Input -> Space -> Help
	verticalElements := []string{logoStyled, "\n"}
	if suggestions != "" {
		verticalElements = append(verticalElements, suggestions)
	} else {
		verticalElements = append(verticalElements, "\n")
	}
	verticalElements = append(verticalElements, inputBox, "\n", helpText)

	centerContent := lipgloss.JoinVertical(lipgloss.Center, verticalElements...)

	// Footer
	footer := m.viewStatusFooter()

	// Calculate main content height
	contentHeight := m.Height - lipgloss.Height(footer)
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Place center content in the middle of the available space
	centerPlaced := lipgloss.Place(m.Width, contentHeight,
		lipgloss.Center, lipgloss.Center,
		centerContent,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(ColorSecondary),
	)

	return lipgloss.JoinVertical(lipgloss.Left, centerPlaced, footer)
}

func (m Model) viewDashboard(activeCount int, runningTasks []TaskItem) string {
	// Layout: Sidebar (Left) | Grid (Right)
	// Footer (Bottom)

	// Dimensions
	footer := m.viewStatusFooter()
	footerHeight := lipgloss.Height(footer)
	contentHeight := m.Height - footerHeight
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Sidebar
	sidebarWidth := int(float64(m.Width) * 0.25)
	if sidebarWidth < 30 {
		sidebarWidth = 30 // Minimum width
	}

	// Ensure sidebar doesn't take too much of small screens
	if sidebarWidth > m.Width/2 {
		sidebarWidth = m.Width / 2
	}

	// Sidebar Focus Styling
	listFocusStyle := StylePaneBorder
	if m.FocusArea == FocusList {
		listFocusStyle = StylePaneBorderFocus
	}

	// Sidebar Header
	sidebarHeader := StyleGridLabel.Width(sidebarWidth - 4).Align(lipgloss.Center).Render("TASK QUEUE")

	taskList := listFocusStyle.Width(sidebarWidth - 2).Height(contentHeight - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			sidebarHeader,
			m.TaskList.View(),
		),
	)

	// Main Grid
	mainWidth := m.Width - sidebarWidth
	if mainWidth < 0 {
		mainWidth = 0
	}

	var workerArea string
	views := make([]string, 0)

	for i, t := range runningTasks {
		if i >= 4 {
			break
		}
		vIdx := i + 1
		vModel := m.WorkerViews[vIdx]

		// Task Label
		label := StyleGridLabel.Render(fmt.Sprintf(" WORKER %d: %s ", vIdx, t.ID))

		// Border Styling
		border := StylePaneBorder
		if t.ID == m.SelectedTaskID {
			border = StylePaneBorderFocus
		}

		// Render Pane
		// Note: vModel.Width/Height are managed by updateLayout in update.go
		// We trust those values are approximately correct.
		pane := border.Width(vModel.Width).Height(vModel.Height).Render(
			lipgloss.JoinVertical(lipgloss.Left, label, vModel.View()),
		)
		views = append(views, pane)
	}

	// Grid Composition
	if len(views) == 0 {
		// No active workers, show System Logs (Orchestrator) to explain why
		// e.g. "Git working directory not clean"

		orchTitle := StyleGridLabel.Render(" SYSTEM LOGS ")
		orchPane := StylePaneBorder.Width(mainWidth).Height(contentHeight).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				orchTitle,
				m.OrchView.View(),
			),
		)

		workerArea = orchPane
	} else {
		switch len(views) {
		case 1:
			workerArea = views[0]
		case 2:
			workerArea = lipgloss.JoinVertical(lipgloss.Left, views[0], views[1])
		case 3:
			top := views[0]
			bottom := lipgloss.JoinHorizontal(lipgloss.Top, views[1], views[2])
			workerArea = lipgloss.JoinVertical(lipgloss.Left, top, bottom)
		case 4:
			top := lipgloss.JoinHorizontal(lipgloss.Top, views[0], views[1])
			bottom := lipgloss.JoinHorizontal(lipgloss.Top, views[2], views[3])
			workerArea = lipgloss.JoinVertical(lipgloss.Left, top, bottom)
		default:
			// Fallback
			workerArea = lipgloss.NewStyle().Width(mainWidth).Height(contentHeight).Render("")
		}
	}

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, taskList, workerArea)

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, footer)
}

func (m Model) viewStatusFooter() string {
	// Bar: [ DIR ] [ STATUS ] [ VERSION ]
	w := lipgloss.Width

	dir := m.LogDir
	// Shorten dir if too long
	if len(dir) > 30 {
		dir = "..." + dir[len(dir)-27:]
	}

	status := " Idle "
	count := len(m.GetRunningTasks())
	styleStatus := StyleStatusPending

	if count > 0 {
		status = fmt.Sprintf(" Active: %d ", count)
		styleStatus = StyleStatusActive
	}

	sDir := lipgloss.NewStyle().Background(ColorSecondary).Foreground(ColorFg).Padding(0, 1).Render(dir)
	sStatus := lipgloss.NewStyle().Background(ColorBg).Foreground(styleStatus.GetForeground()).Padding(0, 1).Bold(true).Render(status)
	sVersion := lipgloss.NewStyle().Background(ColorSecondary).Foreground(ColorFg).Padding(0, 1).Render("v0.2.1")

	// Input Mode Indicator
	sMode := lipgloss.NewStyle().Padding(0, 1).Render(m.getModeString())
	if m.Mode == ModeInsert {
		sMode = lipgloss.NewStyle().Background(ColorPrimary).Foreground(ColorBg).Padding(0, 1).Render(" INSERT ")
	}

	// Middle Spacer
	leftWidth := w(sDir) + w(sStatus)
	rightWidth := w(sMode) + w(sVersion)
	gap := m.Width - leftWidth - rightWidth
	if gap < 0 {
		gap = 0
	}
	spacer := strings.Repeat(" ", gap)

	return lipgloss.JoinHorizontal(lipgloss.Top, sDir, sStatus, spacer, sMode, sVersion)
}

func (m Model) overlay(base, overlay string) string {
	return lipgloss.Place(m.Width, m.Height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(ColorSecondary),
	)
}

func (m Model) getModeString() string {
	if m.Mode == ModeInsert {
		return "INSERT"
	}
	return "NORMAL"
}

func (m Model) viewSuggestions() string {
	if !m.SuggestionActive || len(m.Suggestions) == 0 {
		return ""
	}

	// Limit to 5 items window around selection
	limit := 5
	start := 0
	if m.SuggestionIdx > 2 {
		start = m.SuggestionIdx - 2
	}
	end := start + limit
	if end > len(m.Suggestions) {
		end = len(m.Suggestions)
		start = end - limit
		if start < 0 {
			start = 0
		}
	}

	var items []string
	for i := start; i < end; i++ {
		s := m.Suggestions[i]
		// Truncate if too long logic could go here
		rowStyle := lipgloss.NewStyle().Padding(0, 1).Width(58) // Box width 60 - 2 border
		if i == m.SuggestionIdx {
			rowStyle = rowStyle.Background(ColorPrimary).Foreground(ColorBg)
		} else {
			rowStyle = rowStyle.Foreground(ColorFg)
		}
		items = append(items, rowStyle.Render(s))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(60).
		BorderForeground(ColorSecondary).
		Background(ColorBg). // Ensure opaque
		Render(lipgloss.JoinVertical(lipgloss.Left, items...))
}
