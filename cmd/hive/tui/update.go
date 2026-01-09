package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		startWatchers(m.TasksFile, m.LogDir), // Start real-time file watchers
		fallbackTick(),                       // Fallback polling at 2s
	)
}

// fallbackTick provides a safety net polling mechanism at 2s intervals
// This is used when file watchers fail or as a backup
func fallbackTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// tick is the original fast tick, now only used when watchers fail
func tick() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Track previous selection for tailer management
		prevSelected := m.SelectedTaskID

		// Global Keys
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			if m.Mode == ModeSelection {
				// Cancel any active tailer before quitting
				if m.TailerCancel != nil {
					m.TailerCancel()
				}
				m.Quitting = true
				return m, tea.Quit
			}
		}

		// Modal Handle
		if m.ShowModal {
			switch msg.String() {
			case "enter":
				m.ShowModal = false
				return m, nil
			case "esc":
				m.ShowModal = false
				return m, nil
			}
			return m, nil
		}

		// Mode Switching
		if msg.String() == "i" && m.Mode == ModeSelection {
			m.Mode = ModeInsert
			m.FocusArea = FocusInput
			m.Input.Focus()
			return m, textinput.Blink
		}
		if msg.String() == "esc" && m.Mode == ModeModeInsert() {
			m.Mode = ModeSelection
			m.FocusArea = FocusList
			m.Input.Blur()
			return m, nil
		}

		// Navigation & Actions (Selection Mode)
		if m.Mode == ModeSelection {
			switch msg.String() {
			case "j", "down":
				m.TaskList.CursorDown()
			case "k", "up":
				m.TaskList.CursorUp()
			case "l", "right", "tab":
				m.FocusArea = FocusLogs
			case "h", "left":
				m.FocusArea = FocusList
			case "d":
				if m.SelectedTaskID != "" {
					m.DeleteTask(m.SelectedTaskID)
				}
			}
		}

		// Input Handling (Insert Mode)
		if m.Mode == ModeInsert {
			if msg.String() == "enter" {
				val := m.Input.Value()
				if val != "" {
					if err := m.AddTask(val); err != nil {
						m.Err = err
					}
					m.Input.SetValue("")
				}
			}
		}

		// Check if task selection changed - start new tailer
		if m.Mode == ModeSelection {
			// Get currently selected item
			if item, ok := m.TaskList.SelectedItem().(TaskItem); ok {
				m.SelectedTaskID = item.ID
				if m.SelectedTaskID != prevSelected && m.SelectedTaskID != "" {
					// Selection changed, start tailing new log
					cmds = append(cmds, m.startLogTailer(m.SelectedTaskID))
				}
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Ready = true
		m.updateLayout()

	// === Real-Time Event Handlers ===

	case TasksUpdatedMsg:
		// Tasks file changed - reload immediately
		items := m.LoadTasks()
		m.TaskList.SetItems(items)

		// Re-arm the watcher
		cmds = append(cmds, watchTasksFile(WatchConfig{
			TasksFile: m.TasksFile,
			LogDir:    m.LogDir,
		}))

	case LogLineMsg:
		// New log line received - append to the appropriate viewport
		if msg.TaskID != "" && msg.Line != "" {
			// Update Worker 1 viewport with the selected task's logs
			if msg.TaskID == m.SelectedTaskID {
				if v, ok := m.WorkerViews[1]; ok {
					currentContent := v.View()
					// Avoid appending if it's the initial content load (contains existing content)
					if strings.HasPrefix(msg.Line, "Waiting for logs") || strings.HasPrefix(msg.Line, "Log file empty") {
						v.SetContent(msg.Line)
					} else {
						v.SetContent(currentContent + msg.Line)
					}
					v.GotoBottom()
					m.WorkerViews[1] = v
				}

				// Continue tailing
				if m.TailerCtx != nil {
					logPath := filepath.Join(m.LogDir, fmt.Sprintf("%s.log", msg.TaskID))
					offset := m.LogOffsets[msg.TaskID]
					offset += int64(len(msg.Line))
					m.LogOffsets[msg.TaskID] = offset
					cmds = append(cmds, continueTailing(msg.TaskID, logPath, m.TailerCtx, offset))
				}
			}

			// Also handle orchestrator logs
			if msg.TaskID == "orchestrator" {
				m.OrchView.SetContent(m.OrchView.View() + msg.Line)
				m.OrchView.GotoBottom()
			}
		}

	case LogFileCreatedMsg:
		// New log file created - re-arm the directory watcher
		cmds = append(cmds, watchLogDirectory(WatchConfig{
			TasksFile: m.TasksFile,
			LogDir:    m.LogDir,
		}))

	case TailerStoppedMsg:
		// Tailer stopped (task completed or error)
		if msg.Error != nil {
			// Log error but continue
			m.Err = msg.Error
		}

	case WatcherErrorMsg:
		// File watcher failed - fall back to fast polling
		m.FallbackPolling = true
		m.WatcherActive = false
		if msg.Error != nil {
			m.Err = msg.Error
		}
		// Switch to faster polling as fallback
		cmds = append(cmds, tick())

	case tickMsg:
		// Fallback polling (2s normally, 250ms if watchers failed)
		items := m.LoadTasks()
		m.TaskList.SetItems(items)

		// Update Orchestrator logs
		orchLogs := m.ReadLogs("orchestrator")
		if orchLogs != m.OrchView.View() {
			m.OrchView.SetContent(orchLogs)
			m.OrchView.GotoBottom()
		}

		// Update worker views from assigned task logs
		for i := 1; i <= 4; i++ {
			taskID := ""
			// Find task assigned to this worker
			for _, item := range items {
				ti := item.(TaskItem)
				if ti.Status == "in_progress" {
					// Extract worker ID if we had it
				}
			}

			// Simple auto-fill: show selected task in Worker 1
			if i == 1 && m.SelectedTaskID != "" {
				taskID = m.SelectedTaskID
			}

			if taskID != "" {
				logs := m.ReadLogs(taskID)
				view := m.WorkerViews[i]
				if logs != view.View() {
					view.SetContent(logs)
					view.GotoBottom()
					m.WorkerViews[i] = view
				}
			}
		}

		// Use faster tick if in fallback mode, slower otherwise
		if m.FallbackPolling {
			cmds = append(cmds, tick())
		} else {
			cmds = append(cmds, fallbackTick())
		}
	}

	// Dynamic component updates
	if m.Mode == ModeInsert {
		m.Input, cmd = m.Input.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.FocusArea == FocusLogs {
		m.OrchView, cmd = m.OrchView.Update(msg)
		cmds = append(cmds, cmd)
		// Also update first worker for scrolling if needed
		if v, ok := m.WorkerViews[1]; ok {
			newV, c := v.Update(msg)
			m.WorkerViews[1] = newV
			cmds = append(cmds, c)
		}
	} else {
		m.TaskList, cmd = m.TaskList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// startLogTailer starts tailing a log file for the given task ID
func (m *Model) startLogTailer(taskID string) tea.Cmd {
	// Cancel previous tailer if exists
	if m.TailerCancel != nil {
		m.TailerCancel()
	}

	// Create new context for this tailer
	ctx, cancel := context.WithCancel(context.Background())
	m.TailerCtx = ctx
	m.TailerCancel = cancel

	// Initialize log offsets map if needed
	if m.LogOffsets == nil {
		m.LogOffsets = make(map[string]int64)
	}
	m.LogOffsets[taskID] = 0

	logPath := filepath.Join(m.LogDir, fmt.Sprintf("%s.log", taskID))

	// Check if file exists and get initial content
	if _, err := os.Stat(logPath); err == nil {
		return startTailing(taskID, logPath, ctx)
	}

	// File doesn't exist yet
	return func() tea.Msg {
		return LogLineMsg{TaskID: taskID, Line: "Waiting for logs..."}
	}
}

func (m *Model) updateFocus() {
	if m.FocusArea == FocusInput {
		m.Input.Focus()
	} else {
		m.Input.Blur()
	}
}

func (m *Model) updateLayout() {
	if m.Width == 0 || m.Height == 0 {
		return
	}

	headerHeight := 1
	footerHeight := 3
	panesHeight := m.Height - headerHeight - footerHeight

	// Main Area (Orchestrator) - 60% Width, 70% Height
	orchWidth := int(float64(m.Width) * 0.6)
	orchHeight := int(float64(panesHeight) * 0.7)

	m.OrchView.Width = orchWidth - 2
	m.OrchView.Height = orchHeight - 2

	// Side Column (Worker 1 & 2) - 40% Width
	sideWidth := m.Width - orchWidth
	sideHeight := orchHeight / 2

	if v, ok := m.WorkerViews[1]; ok {
		v.Width = sideWidth - 2
		v.Height = sideHeight - 2
		m.WorkerViews[1] = v
	}
	if v, ok := m.WorkerViews[2]; ok {
		v.Width = sideWidth - 2
		v.Height = sideHeight - 2
		m.WorkerViews[2] = v
	}

	// Bottom Row (Worker 3, 4, Tasks) - 30% Height
	bottomHeight := panesHeight - orchHeight
	thirdWidth := m.Width / 3

	if v, ok := m.WorkerViews[3]; ok {
		v.Width = thirdWidth - 2
		v.Height = bottomHeight - 2
		m.WorkerViews[3] = v
	}
	if v, ok := m.WorkerViews[4]; ok {
		v.Width = thirdWidth - 2
		v.Height = bottomHeight - 2
		m.WorkerViews[4] = v
	}

	m.TaskList.SetSize(thirdWidth-2, bottomHeight-2)
	m.Input.Width = m.Width - 4
}

func containsPlan(s string) bool {
	return strings.Contains(s, "### PLAN_START ###")
}

func ModeModeInsert() ViewMode { return ModeInsert }
