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

		// Layout might change
		m.updateLayout()

		// Re-arm the watcher
		cmds = append(cmds, watchTasksFile(WatchConfig{
			TasksFile: m.TasksFile,
			LogDir:    m.LogDir,
		}))

	case LogLineMsg:
		// New log line received - append to the appropriate viewport
		if msg.TaskID != "" && msg.Line != "" {
			// Find if this task is currently displayed (is running)
			running := m.GetRunningTasks()
			var viewIdx int = -1

			for i, t := range running {
				if t.ID == msg.TaskID {
					viewIdx = i + 1
					break
				}
			}

			// If displayed, update the viewport
			if viewIdx != -1 {
				if v, ok := m.WorkerViews[viewIdx]; ok {
					currentContent := v.View()
					// Avoid appending if it's the initial content load (contains existing content)
					if strings.HasPrefix(msg.Line, "Waiting for logs") || strings.HasPrefix(msg.Line, "Log file empty") {
						v.SetContent(msg.Line)
					} else {
						v.SetContent(currentContent + msg.Line)
					}
					v.GotoBottom()
					m.WorkerViews[viewIdx] = v
				}
			}

			// Continue tailing regardless of visibility (so we don't drop the stream)
			// Only tail if this is the SELECTED task or we are ensuring background tailing?
			// Actually, startLogTailer is called on Selection.
			// If we want to tail ALL running tasks, we need multiple contexts or a map of tailers.
			// Current architecture has `TailerCtx` (singular).
			// So we only tail the SELECTED task.
			// If the selected task is running, `viewIdx` will be valid.
			// If the selected task is NOT running, `viewIdx` is -1, but we successfully tail (just don't render).

			if msg.TaskID == m.SelectedTaskID && m.TailerCtx != nil {
				logPath := filepath.Join(m.LogDir, fmt.Sprintf("%s.log", msg.TaskID))
				offset := m.LogOffsets[msg.TaskID]
				offset += int64(len(msg.Line))
				m.LogOffsets[msg.TaskID] = offset
				cmds = append(cmds, continueTailing(msg.TaskID, logPath, m.TailerCtx, offset))
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

		// Update layout based on new state
		m.updateLayout()

		// Update Orchestrator logs
		orchLogs := m.ReadLogs("orchestrator")
		if orchLogs != m.OrchView.View() {
			m.OrchView.SetContent(orchLogs)
			m.OrchView.GotoBottom()
		}

		// Update worker views from RUNNING tasks (Dynamic Grid)
		running := m.GetRunningTasks()
		for i, t := range running {
			if i >= 4 {
				break
			}
			idx := i + 1

			logs := m.ReadLogs(t.ID)
			view := m.WorkerViews[idx]

			// Compare content length to avoid flicker or heavy updates
			if logs != view.View() {
				view.SetContent(logs)
				view.GotoBottom()
				m.WorkerViews[idx] = view
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

	runningTasks := m.GetRunningTasks()
	activeCount := len(runningTasks)

	headerHeight := 1
	footerHeight := 3
	contentHeight := m.Height - headerHeight - footerHeight

	if activeCount == 0 {
		// Idle / Chat Mode
		m.TaskList.SetSize(m.Width-2, contentHeight)
		return
	}

	// Active Mode
	sidebarWidth := int(float64(m.Width) * 0.25)
	if sidebarWidth < 30 {
		sidebarWidth = 30
	}
	mainWidth := m.Width - sidebarWidth

	// Update Sidebar
	m.TaskList.SetSize(sidebarWidth-2, contentHeight)

	// Distribute Main Area
	// We need to update up to activeCount viewports
	for i := 0; i < activeCount && i < 4; i++ {
		vIdx := i + 1
		view, ok := m.WorkerViews[vIdx]
		if !ok {
			continue
		}

		var w, h int

		switch activeCount {
		case 1:
			w, h = mainWidth, contentHeight
		case 2:
			w, h = mainWidth, contentHeight/2 // Stacked Vertical
		case 3:
			if i == 0 {
				w, h = mainWidth, contentHeight/2 // Top
			} else {
				w, h = mainWidth/2, contentHeight/2 // Bottom Split
			}
		case 4:
			w, h = mainWidth/2, contentHeight/2 // 2x2 Grid
		default:
			// Fallback (cap at 4 for now in layout)
			w, h = mainWidth/2, contentHeight/2
		}

		// Adjust for borders
		view.Width = w - 2
		view.Height = h - 2
		m.WorkerViews[vIdx] = view
	}
}

func containsPlan(s string) bool {
	return strings.Contains(s, "### PLAN_START ###")
}

func ModeModeInsert() ViewMode { return ModeInsert }
