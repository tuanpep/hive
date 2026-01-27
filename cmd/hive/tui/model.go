package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/tuanbt/hive/internal/task"
)

type tickMsg time.Time

type ViewMode int

const (
	ModeSelection ViewMode = iota
	ModeInsert
)

type Model struct {
	// Core dependencies
	TaskManager   *task.Manager
	TasksFile     string
	LogDir        string
	WorkDirectory string

	// UI Components
	TaskList list.Model
	LogView viewport.Model // Single viewport for selected task
	Input   textinput.Model

	// State (minimal)
	SelectedTaskID string
	Width          int
	Height         int
	Mode           ViewMode
	Err            error
	Ready          bool

	// Real-time tracking
	TailerCtx    context.Context
	TailerCancel context.CancelFunc
	LogOffsets   map[string]int64

	// Suggestions (for @ and / commands)
	SuggestionActive bool
	SuggestionType   string // "@" or "/"
	Suggestions      []string
	SuggestionIdx    int
	SuggestionStart  int // Cursor index where @ started
}

// TaskItem implements list.Item
type TaskItem struct {
	ID          string
	Title       string
	Status      string
	Description string
	LastLog     string
}

func (i TaskItem) FilterValue() string       { return i.Title }
func (t TaskItem) TitleString() string       { return t.Title }
func (t TaskItem) DescriptionString() string { return t.Description }

// GetRunningTasks returns tasks that are currently in progress
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
