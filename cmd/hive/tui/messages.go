// Package tui provides the terminal user interface for HIVE.
package tui

// TasksUpdatedMsg signals that the tasks.json file has been modified.
// The TUI should reload the task list when receiving this message.
type TasksUpdatedMsg struct{}

// LogLineMsg contains a new log line for a specific task.
// Used for real-time log streaming in worker viewports.
type LogLineMsg struct {
	TaskID string
	Line   string
}

// WatcherErrorMsg signals that the file watcher encountered an error.
// The TUI should fall back to polling mode when receiving this message.
type WatcherErrorMsg struct {
	Error error
}

// LogFileCreatedMsg signals that a new log file was created in the logs directory.
// Used to start tailing a new task's log file.
type LogFileCreatedMsg struct {
	TaskID string
	Path   string
}

// TailerStoppedMsg signals that a log tailer has stopped (task completed or error).
type TailerStoppedMsg struct {
	TaskID string
	Error  error
}
