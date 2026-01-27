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
	"github.com/tuanbt/hive/cmd/hive/tui/files"
	"github.com/tuanbt/hive/cmd/hive/tui/shell"
	"github.com/tuanbt/hive/internal/task"
)

const HELP_TEXT = `
HIVE Commands:
  i          - Enter insert mode
  j/k        - Navigate tasks
  d          - Delete selected task
  r          - Retry selected task
  @file      - Reference file
  !command   - Execute shell command
  /command   - Execute slash command
  esc        - Exit insert mode
  q/ctrl+c   - Quit
`

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		startWatchers(m.TasksFile, m.LogDir),
		fallbackTick(),
	)
}

// fallbackTick provides polling at 2s intervals
func fallbackTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Ready = true
		m.updateLayout()
		return m, nil
	case TasksUpdatedMsg:
		m.TaskList.SetItems(m.LoadTasks())
		m.updateLayout()
		cmds = append(cmds, watchTasksFile(WatchConfig{
			TasksFile: m.TasksFile,
			LogDir:    m.LogDir,
		}))
		return m, tea.Batch(cmds...)
	case LogLineMsg:
		return m.handleLogLine(msg)
	case tickMsg:
		return m.handleTick()
	}

	// Update focused component
	if m.Mode == ModeInsert {
		var cmd tea.Cmd
		m.Input, cmd = m.Input.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.TaskList, cmd = m.TaskList.Update(msg)
	return m, cmd
}

// handleKey - simplified key handling
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	if msg.String() == "ctrl+c" || msg.String() == "q" {
		if m.TailerCancel != nil {
			m.TailerCancel()
		}
		return m, tea.Quit
	}

	// Mode switching
	if msg.String() == "i" && m.Mode == ModeSelection {
		m.Mode = ModeInsert
		m.Input.Focus()
		return m, textinput.Blink
	}
	if msg.String() == "esc" && m.Mode == ModeInsert {
		m.Mode = ModeSelection
		m.Input.Blur()
		m.SuggestionActive = false
		return m, nil
	}

	// Selection mode
	if m.Mode == ModeSelection {
		return m.handleSelectionKey(msg)
	}

	// Insert mode
	return m.handleInsertKey(msg)
}

// handleSelectionKey - task navigation
func (m Model) handleSelectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	prevSelected := m.SelectedTaskID

	switch msg.String() {
	case "j", "down":
		m.TaskList.CursorDown()
	case "k", "up":
		m.TaskList.CursorUp()
	case "d":
		if m.SelectedTaskID != "" {
			m.DeleteTask(m.SelectedTaskID)
		}
	case "r":
		if m.SelectedTaskID != "" {
			m.RetryTask(m.SelectedTaskID)
		}
	case "ctrl+r":
		items := m.LoadTasks()
		m.TaskList.SetItems(items)
	}

	// Check selection change
	if item, ok := m.TaskList.SelectedItem().(TaskItem); ok {
		m.SelectedTaskID = item.ID
		if m.SelectedTaskID != prevSelected {
			return m, m.startLogTailer(m.SelectedTaskID)
		}
	}

	return m, nil
}

// handleInsertKey - simplified input handling
func (m Model) handleInsertKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Suggestion navigation
	if m.SuggestionActive {
		switch msg.String() {
		case "up":
			m.SuggestionIdx--
			if m.SuggestionIdx < 0 {
				m.SuggestionIdx = len(m.Suggestions) - 1
			}
			return m, nil
		case "down":
			m.SuggestionIdx++
			if m.SuggestionIdx >= len(m.Suggestions) {
				m.SuggestionIdx = 0
			}
			return m, nil
		case "enter":
			m.applySuggestion()
			m.SuggestionActive = false
			return m, nil
		case "esc":
			m.SuggestionActive = false
			return m, nil
		}
	}

	// Trigger suggestions
	if msg.String() == "@" {
		m.SuggestionActive = true
		m.SuggestionType = "@"
		m.SuggestionStart = len(m.Input.Value())
		fileResults, _ := files.FindFiles(m.WorkDirectory, "")
		m.Suggestions = files.GetFilenames(fileResults)
		m.SuggestionIdx = 0
		return m, nil
	}

	if msg.String() == "/" && len(m.Input.Value()) == 0 {
		m.SuggestionActive = true
		m.SuggestionType = "/"
		m.SuggestionStart = 0
		m.Suggestions = []string{"/help", "/quit", "/retry", "/nuke"}
		m.SuggestionIdx = 0
		return m, nil
	}

	// Update input
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	cmds = append(cmds, cmd)

	// Update suggestions
	if m.SuggestionActive {
		val := m.Input.Value()
		start := m.SuggestionStart

		if start < len(val) {
			// Check trigger still valid
			if m.SuggestionType == "@" && (start == 0 || val[start-1] != '@') {
				m.SuggestionActive = false
				return m, tea.Batch(cmds...)
			}
			if m.SuggestionType == "/" && (len(val) == 0 || val[0] != '/') {
				m.SuggestionActive = false
				return m, tea.Batch(cmds...)
			}

			// Filter suggestions
			filter := val[start:]
			if strings.Contains(filter, " ") {
				m.SuggestionActive = false
			} else {
				var filtered []string
				for _, s := range m.Suggestions {
					if strings.Contains(strings.ToLower(s), strings.ToLower(filter)) {
						filtered = append(filtered, s)
					}
				}
				m.Suggestions = filtered
				m.SuggestionIdx = 0
			}
		}
	}

	// Submit on enter
	if msg.String() == "enter" && !m.SuggestionActive {
		return m.handleSubmit()
	}

	return m, tea.Batch(cmds...)
}

// handleSubmit - simplified command execution
func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	val := m.Input.Value()
	if val == "" {
		return m, nil
	}

	// Slash commands
	if strings.HasPrefix(val, "/") {
		return m.executeSlashCommand(val)
	}

	// Shell commands
	if strings.HasPrefix(val, "!") {
		shellCmd := strings.TrimSpace(strings.TrimPrefix(val, "!"))
		if shellCmd != "" {
			shell.ExecuteShellCommand(m.WorkDirectory, shellCmd)
		}
		m.Input.SetValue("")
		return m, nil
	}

	// Add task
	m.addTask(val)
	m.Input.SetValue("")
	return m, nil
}

// executeSlashCommand - essential commands only
func (m Model) executeSlashCommand(val string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(val)
	if len(parts) == 0 {
		return m, nil
	}

	switch parts[0] {
	case "/quit", "/exit":
		return m, tea.Quit
	case "/help", "/?":
		m.Err = fmt.Errorf(HELP_TEXT)
		m.Input.SetValue("")
	case "/retry":
		if m.SelectedTaskID != "" {
			m.RetryTask(m.SelectedTaskID)
		}
		m.Input.SetValue("")
	case "/nuke":
		m.Nuke()
		m.Input.SetValue("")
	default:
		m.Input.SetValue("")
	}

	return m, nil
}

// addTask - smart task creation
func (m *Model) addTask(title string) {
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())
	t := task.NewTask(id, title, title)

	// Smart role detection
	lowerTitle := strings.ToLower(title)
	if strings.HasPrefix(lowerTitle, "i want") ||
		strings.HasPrefix(lowerTitle, "build") ||
		strings.HasPrefix(lowerTitle, "create") ||
		strings.HasPrefix(lowerTitle, "plan") {
		t.Role = "ba"
	}

	m.TaskManager.AddTask(t)
	items := m.LoadTasks()
	m.TaskList.SetItems(items)
}

// applySuggestion - insert selected suggestion
func (m *Model) applySuggestion() {
	if len(m.Suggestions) == 0 {
		return
	}

	selected := m.Suggestions[m.SuggestionIdx]
	val := m.Input.Value()

	if m.SuggestionType == "@" {
		before := val[:m.SuggestionStart]
		newVal := before + selected + " "
		m.Input.SetValue(newVal)
		m.Input.SetCursor(len(newVal))
	} else if m.SuggestionType == "/" {
		m.Input.SetValue(selected)
		m.Input.SetCursor(len(selected))
	}
}

// handleLogLine - simplified log handling
func (m Model) handleLogLine(msg LogLineMsg) (tea.Model, tea.Cmd) {
	if msg.TaskID == m.SelectedTaskID {
		current := m.LogView.View()
		m.LogView.SetContent(current + msg.Line)
		m.LogView.GotoBottom()
	}
	return m, nil
}

// handleTick - simplified polling
func (m Model) handleTick() (tea.Model, tea.Cmd) {
	items := m.LoadTasks()
	m.TaskList.SetItems(items)

	if m.SelectedTaskID != "" {
		logs := m.ReadLogs(m.SelectedTaskID)
		if logs != m.LogView.View() {
			m.LogView.SetContent(logs)
			m.LogView.GotoBottom()
		}
	}

	return m, fallbackTick()
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

// updateLayout - simplified layout
func (m *Model) updateLayout() {
	if m.Width == 0 || m.Height == 0 {
		return
	}

	footerHeight := 3
	contentHeight := m.Height - footerHeight

	// Task list: 30% width
	listWidth := m.Width * 30 / 100
	if listWidth < 30 {
		listWidth = 30
	}
	m.TaskList.SetSize(listWidth-4, contentHeight-4)

	// Log view: 70% width
	logWidth := m.Width - listWidth
	m.LogView.Width = logWidth - 4
	m.LogView.Height = contentHeight - 4
}

func ModeModeInsert() ViewMode { return ModeInsert }
