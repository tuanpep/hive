package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.Width == 0 || !m.Ready {
		return "Initializing..."
	}

	// Main layout: two panes
	leftPane := m.renderTaskList()
	rightPane := m.renderLogView()

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Footer with input and help
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, footer)
}

func (m Model) renderTaskList() string {
	header := StyleTitle.Render(" TASKS ")
	content := m.TaskList.View()

	border := StyleBorder
	width := m.Width * 30 / 100
	if width < 30 {
		width = 30
	}

	return border.Width(width).Height(m.Height - 3).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, content),
	)
}

func (m Model) renderLogView() string {
	title := "LOGS"
	if m.SelectedTaskID != "" {
		// Shorten task ID for display
		shortID := m.SelectedTaskID
		if len(shortID) > 20 {
			shortID = shortID[:17] + "..."
		}
		title = fmt.Sprintf("LOGS: %s", shortID)
	}

	header := StyleTitle.Render(" " + title + " ")
	content := m.LogView.View()

	if content == "" {
		content = StyleDimmed.Render("No task selected")
	}

	border := StyleBorderFocused
	width := m.Width * 70 / 100

	return border.Width(width).Height(m.Height - 3).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, content),
	)
}

func (m Model) renderFooter() string {
	// Input line
	prompt := ">"
	if m.Mode == ModeInsert {
		prompt = StyleInput.Render(">")
	}
	inputLine := prompt + " " + m.Input.View()

	// Status/error line (if any)
	var status string
	if m.Err != nil {
		status = StyleError.Render(fmt.Sprintf(" [ERROR: %s]", m.Err.Error()))
	}

	// Help line
	help := StyleHelp.Render("i=insert j/k=nav d=del r=retry @=file !=shell /=cmd q=quit")

	// Combine input line
	inputWithStatus := inputLine
	if status != "" {
		inputWithStatus += " " + status
	}

	// Calculate spacing for help to right-align
	helpGap := m.Width - lipgloss.Width(inputWithStatus)
	if helpGap < 0 {
		helpGap = 0
	}

	topLine := lipgloss.JoinHorizontal(lipgloss.Left,
		inputWithStatus,
		strings.Repeat(" ", helpGap),
	)

	return lipgloss.JoinVertical(lipgloss.Left, topLine, help)
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
		// Truncate if too long
		if len(s) > 58 {
			s = s[:55] + "..."
		}
		rowStyle := lipgloss.NewStyle().Padding(0, 1).Width(58)
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
		BorderForeground(ColorDim).
		Background(ColorBg).
		Render(lipgloss.JoinVertical(lipgloss.Left, items...))
}
