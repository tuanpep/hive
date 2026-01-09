package tui

import (
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// WatchConfig holds configuration for the file watcher.
type WatchConfig struct {
	TasksFile string
	LogDir    string
}

// watchTasksFile returns a tea.Cmd that watches the tasks.json file for changes.
// When the file is modified, it emits a TasksUpdatedMsg.
// On error, it emits a WatcherErrorMsg.
func watchTasksFile(cfg WatchConfig) tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return WatcherErrorMsg{Error: err}
		}
		defer watcher.Close()

		// Watch the tasks file
		if err := watcher.Add(cfg.TasksFile); err != nil {
			return WatcherErrorMsg{Error: err}
		}

		// Wait for an event
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return WatcherErrorMsg{Error: nil}
				}
				// Check for write or create events
				if event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Create == fsnotify.Create {
					// Small debounce to avoid rapid-fire events
					time.Sleep(10 * time.Millisecond)
					return TasksUpdatedMsg{}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return WatcherErrorMsg{Error: nil}
				}
				return WatcherErrorMsg{Error: err}
			}
		}
	}
}

// watchLogDirectory returns a tea.Cmd that watches the logs directory for new files.
// When a new .log file is created, it emits a LogFileCreatedMsg.
func watchLogDirectory(cfg WatchConfig) tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return WatcherErrorMsg{Error: err}
		}
		defer watcher.Close()

		// Watch the log directory
		if err := watcher.Add(cfg.LogDir); err != nil {
			return WatcherErrorMsg{Error: err}
		}

		// Wait for an event
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return WatcherErrorMsg{Error: nil}
				}
				// Check for new log files
				if event.Op&fsnotify.Create == fsnotify.Create ||
					event.Op&fsnotify.Write == fsnotify.Write {
					if strings.HasSuffix(event.Name, ".log") {
						// Extract task ID from filename
						base := filepath.Base(event.Name)
						taskID := strings.TrimSuffix(base, ".log")
						return LogFileCreatedMsg{
							TaskID: taskID,
							Path:   event.Name,
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return WatcherErrorMsg{Error: nil}
				}
				return WatcherErrorMsg{Error: err}
			}
		}
	}
}

// startWatchers returns a batch of commands to start all file watchers.
func startWatchers(tasksFile, logDir string) tea.Cmd {
	cfg := WatchConfig{
		TasksFile: tasksFile,
		LogDir:    logDir,
	}
	return tea.Batch(
		watchTasksFile(cfg),
		watchLogDirectory(cfg),
	)
}
