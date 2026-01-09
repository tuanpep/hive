package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.Width == 0 || !m.Ready {
		return "Initialising system..."
	}

	runningTasks := m.GetRunningTasks()
	activeCount := len(runningTasks)

	// 1. Header is common
	headerStr := fmt.Sprintf(" 🤖 HIVE-SWARM | TASKS: %d | RUNNING: %d | ID: %s | MODE: %s ",
		len(m.TaskList.Items()), activeCount, m.SelectedTaskID, m.getModeString())
	header := StyleHeader.Width(m.Width).Render(headerStr)

	var mainContent string

	// 2. Dynamic Layout Logic
	if activeCount == 0 {
		// === IDLE / CHAT MODE ===
		// Full width Task List + Input
		// We hide the orchestrator/worker views completely to unclutter
		listFocusStyle := StylePaneBorder
		if m.FocusArea == FocusList {
			listFocusStyle = StylePaneBorderFocus
		}

		// List takes full remaining height
		footerHeight := 3                         // input + help
		listHeight := m.Height - 1 - footerHeight // header=1

		taskView := listFocusStyle.Width(m.Width - 2).Height(listHeight).Render(m.TaskList.View())
		mainContent = taskView

	} else {
		// === ACTIVE SWARM MODE ===
		// Left: Task List (Sidebar)
		// Right: Dynamic Worker Grid

		// Layout constants
		sidebarWidth := int(float64(m.Width) * 0.25)
		if sidebarWidth < 30 {
			sidebarWidth = 30
		} // Min width
		contentHeight := m.Height - 4 // header(1) + footer(3)

		// Render Sidebar
		listFocusStyle := StylePaneBorder
		if m.FocusArea == FocusList {
			listFocusStyle = StylePaneBorderFocus
		}
		sidebar := listFocusStyle.Width(sidebarWidth - 2).Height(contentHeight).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				StyleGridLabel.Background(ColorBlue).Render(" TASK QUEUE "),
				m.TaskList.View(),
			),
		)

		// Render Main Grid based on active count
		var workerArea string

		// Map active tasks to views (up to 4 supported for now)
		views := make([]string, 0)
		for i, t := range runningTasks {
			if i >= 4 {
				break
			}

			// Determine which view model to use (1-4)
			vIdx := i + 1
			vModel := m.WorkerViews[vIdx]

			// Render individual worker pane
			label := StyleGridLabel.Render(fmt.Sprintf(" WORKER-%d [%s] ", vIdx, t.ID))
			border := StylePaneBorder
			// Highlight if this task is selected (simple logic)
			if t.ID == m.SelectedTaskID {
				border = StylePaneBorderFocus
			}

			pane := border.Width(vModel.Width).Height(vModel.Height).Render(
				lipgloss.JoinVertical(lipgloss.Left, label, vModel.View()),
			)
			views = append(views, pane)
		}

		// Grid Composition
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
			// Fallback (shouldn't happen with break above)
			workerArea = views[0]
		}

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, workerArea)
	}

	// 3. Footer (Input Deck)
	inputPrefix := StyleInputPrefix.Render(">_ ")
	footer := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center, inputPrefix, m.Input.View()),
		StyleDimmed.Render(" [i] Insert [ESC] Selection [j/k] Nav [q] Quit"),
	)

	// Combine all
	ui := lipgloss.JoinVertical(lipgloss.Left, header, mainContent, footer)

	if m.ShowModal {
		modal := StyleModal.Render(m.ModalContent)
		return m.overlay(ui, modal)
	}

	return ui
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
