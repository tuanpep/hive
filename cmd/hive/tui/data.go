package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/tuanbt/hive/internal/task"
)

// LoadTasks reads tasks from the tasks.json file via TaskManager
func (m *Model) LoadTasks() []list.Item {
	tasks, err := m.TaskManager.LoadAll()
	if err != nil {
		return []list.Item{} // Return empty on error or log it via some debug channel?
	}

	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		statusIcon := "⏳"
		switch t.Status {
		case task.StatusInProgress:
			statusIcon = "🏃"
		case task.StatusReviewing:
			statusIcon = "👀"
		case task.StatusCompleted:
			statusIcon = "✅"
		case task.StatusFailed:
			statusIcon = "❌"
		}

		desc := string(t.Status)
		if t.Status == task.StatusInProgress || t.Status == task.StatusReviewing {
			desc = fmt.Sprintf("%s | ID: %s", t.Status, t.ID)
		} else if t.Status == task.StatusFailed {
			desc = fmt.Sprintf("Failed: %s", t.FailReason)
		}

		items[i] = TaskItem{
			ID:          t.ID,
			Title:       fmt.Sprintf("%s %s", statusIcon, t.Title),
			Status:      string(t.Status),
			Description: desc,
		}
	}
	return items
}

// AddTask appends a new task to the file
func (m *Model) AddTask(title string) error {
	t := task.NewTask(
		fmt.Sprintf("task-%d", time.Now().UnixNano()),
		title,
		title,
	)

	return m.TaskManager.AddTask(t)
}

// ReadLogs reads the log file for the selected task
func (m *Model) ReadLogs(taskID string) string {
	if taskID == "" {
		return "No task selected."
	}

	path := filepath.Join(m.LogDir, fmt.Sprintf("%s.log", taskID))
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "Waiting for logs..."
		}
		return fmt.Sprintf("Error reading logs: %v", err)
	}
	if len(content) == 0 {
		return "Log file empty..."
	}
	return string(content)
}

// DeleteTask removes a task from the file
func (m *Model) DeleteTask(taskID string) error {
	return m.TaskManager.DeleteTask(taskID)
}
