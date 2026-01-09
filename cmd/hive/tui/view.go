package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.Width == 0 || !m.Ready {
		return "Initialising system..."
	}

	// 1. Header (Status Bar)
	headerStr := fmt.Sprintf(" 🤖 HIVE-SWARM | TASKS: %d | ID: %s | MODE: %s ",
		len(m.TaskList.Items()), m.SelectedTaskID, m.getModeString())
	header := StyleHeader.Width(m.Width).Render(headerStr)

	// 2. Panes (Complex Grid)
	// Grid logic
	orchStyle := StylePaneBorder
	if m.FocusArea == FocusLogs {
		orchStyle = StylePaneBorderFocus
	}
	orchLabel := StyleGridLabel.Render(" ORCHESTRATOR ")
	orchView := orchStyle.Width(m.OrchView.Width + 2).Height(m.OrchView.Height + 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, orchLabel, m.OrchView.View()),
	)

	// Worker 1 & 2 (Right Stack)
	w1Label := StyleGridLabel.Render(" WORKER-1 ")
	w1 := StylePaneBorder.Width(m.WorkerViews[1].Width + 2).Height(m.WorkerViews[1].Height + 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, w1Label, m.WorkerViews[1].View()),
	)
	w2Label := StyleGridLabel.Render(" WORKER-2 ")
	w2 := StylePaneBorder.Width(m.WorkerViews[2].Width + 2).Height(m.WorkerViews[2].Height + 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, w2Label, m.WorkerViews[2].View()),
	)
	rightStack := lipgloss.JoinVertical(lipgloss.Left, w1, w2)

	topSection := lipgloss.JoinHorizontal(lipgloss.Top, orchView, rightStack)

	// Bottom Row (Worker 3, 4, Tasks)
	w3Label := StyleGridLabel.Render(" WORKER-3 ")
	w3 := StylePaneBorder.Width(m.WorkerViews[3].Width + 2).Height(m.WorkerViews[3].Height + 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, w3Label, m.WorkerViews[3].View()),
	)
	w4Label := StyleGridLabel.Render(" WORKER-4 ")
	w4 := StylePaneBorder.Width(m.WorkerViews[4].Width + 2).Height(m.WorkerViews[4].Height + 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, w4Label, m.WorkerViews[4].View()),
	)

	listFocusStyle := StylePaneBorder
	if m.FocusArea == FocusList {
		listFocusStyle = StylePaneBorderFocus
	}
	taskLabel := StyleGridLabel.Background(ColorBlue).Render(" TASK-QUEUE ")
	taskView := listFocusStyle.Width(m.TaskList.Width() + 2).Height(m.TaskList.Height() + 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, taskLabel, m.TaskList.View()),
	)

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, w3, w4, taskView)

	panes := lipgloss.JoinVertical(lipgloss.Left, topSection, bottomRow)

	// 3. Footer (Input Deck)
	inputPrefix := StyleInputPrefix.Render(">_ ")
	footer := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center, inputPrefix, m.Input.View()),
		StyleDimmed.Render(" [i] Insert [ESC] Selection [d] Delete [q] Quit"),
	)

	content := lipgloss.JoinVertical(lipgloss.Left, header, panes, footer)

	if m.ShowModal {
		modal := StyleModal.Render(m.ModalContent)
		return m.overlay(content, modal)
	}

	return content
}

func (m Model) getModeString() string {
	if m.Mode == ModeInsert {
		return "INSERT"
	}
	return "SELECTION"
}

func (m Model) overlay(base, overlay string) string {
	return lipgloss.Place(m.Width, m.Height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(ColorBorder),
	)
}
