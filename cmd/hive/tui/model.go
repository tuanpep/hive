package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

type tickMsg time.Time

type ViewMode int

const (
	ModeSelection ViewMode = iota
	ModeInsert
)

type FocusArea int

const (
	FocusList FocusArea = iota
	FocusLogs
	FocusInput
)

type Model struct {
	// State sources
	ConfigPath string
	TasksFile  string
	LogDir     string

	// Models
	TaskList    list.Model
	OrchView    viewport.Model
	WorkerViews map[int]viewport.Model
	Input       textinput.Model

	// State
	SelectedTaskID string
	LogContent     string
	Width          int
	Height         int
	Err            error
	Ready          bool
	FocusArea      FocusArea
	Mode           ViewMode

	// Hacker V3 State
	ShowModal    bool
	ModalContent string
	Quitting     bool

	// Grid state
	WorkerTaskIDs map[int]string

	// Real-time tracking state
	WatcherActive   bool               // Whether file watchers are running
	TailerCtx       context.Context    // Context for active tailer
	TailerCancel    context.CancelFunc // Cancel function for tailer
	LogOffsets      map[string]int64   // Track file offsets for each task log
	FallbackPolling bool               // True if watchers failed, using polling
}

// TaskItem implements list.Item
type TaskItem struct {
	ID          string
	Title       string
	Status      string
	Description string
}

func (i TaskItem) FilterValue() string       { return i.Title }
func (t TaskItem) TitleString() string       { return t.Title }
func (t TaskItem) DescriptionString() string { return t.Description }

// GetActiveTasks returns a list of TaskItems that are currently running
func (m Model) GetActiveTasks() []TaskItem {
	var active []TaskItem
	for _, item := range m.TaskList.Items() {
		if t, ok := item.(TaskItem); ok {
			// specific statuses that count as "Active" for view purposes
			if t.Status == "in_progress" || t.Status == "reviewing" || t.Status == "pending" {
				active = append(active, t)
			}
		}
	}
	return active
}

// GetRunningTasks returns tasks that specifically warrant a "Worker Window" (in_progress/reviewing)
func (m Model) GetRunningTasks() []TaskItem {
	var running []TaskItem
	for _, item := range m.TaskList.Items() {
		if t, ok := item.(TaskItem); ok {
			if t.Status == "in_progress" || t.Status == "reviewing" {
				running = append(running, t)
			}
		}
	}
	return running
}
