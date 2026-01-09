package tui

import (
	"bufio"
	"context"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// LogTailer handles tailing a log file and streaming new lines.
type LogTailer struct {
	taskID string
	path   string
	ctx    context.Context
	cancel context.CancelFunc
}

// NewLogTailer creates a new log tailer for the specified task.
func NewLogTailer(taskID, path string) *LogTailer {
	ctx, cancel := context.WithCancel(context.Background())
	return &LogTailer{
		taskID: taskID,
		path:   path,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Stop stops the log tailer.
func (t *LogTailer) Stop() {
	t.cancel()
}

// tailLogFile returns a tea.Cmd that tails a log file and emits LogLineMsg for each new line.
// This implements a simple "tail -f" behavior.
func tailLogFile(taskID, path string, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		// Open the file
		file, err := os.Open(path)
		if err != nil {
			// File doesn't exist yet, wait and retry
			if os.IsNotExist(err) {
				time.Sleep(100 * time.Millisecond)
				return LogLineMsg{TaskID: taskID, Line: ""}
			}
			return TailerStoppedMsg{TaskID: taskID, Error: err}
		}
		defer file.Close()

		// Seek to end of file for new content only
		_, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			return TailerStoppedMsg{TaskID: taskID, Error: err}
		}

		reader := bufio.NewReader(file)

		// Poll for new content
		for {
			select {
			case <-ctx.Done():
				return TailerStoppedMsg{TaskID: taskID, Error: nil}
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						// No new content, wait a bit
						time.Sleep(50 * time.Millisecond)
						continue
					}
					return TailerStoppedMsg{TaskID: taskID, Error: err}
				}

				// Return the line (this will re-invoke the command for next line)
				if line != "" {
					return LogLineMsg{TaskID: taskID, Line: line}
				}
			}
		}
	}
}

// startTailing returns a tea.Cmd that starts tailing a log file.
// It reads the entire existing content first, then tails new lines.
func startTailing(taskID, path string, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		// First, read existing content
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				// File doesn't exist yet, that's okay
				return LogLineMsg{TaskID: taskID, Line: "Waiting for logs..."}
			}
			return TailerStoppedMsg{TaskID: taskID, Error: err}
		}

		// Return existing content as first message
		if len(content) > 0 {
			return LogLineMsg{TaskID: taskID, Line: string(content)}
		}

		return LogLineMsg{TaskID: taskID, Line: "Log file empty, waiting..."}
	}
}

// continueTailing returns a tea.Cmd that continues tailing after the initial read.
func continueTailing(taskID, path string, ctx context.Context, offset int64) tea.Cmd {
	return func() tea.Msg {
		// Check context
		select {
		case <-ctx.Done():
			return TailerStoppedMsg{TaskID: taskID, Error: nil}
		default:
		}

		// Open file
		file, err := os.Open(path)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			return continueTailing(taskID, path, ctx, offset)()
		}
		defer file.Close()

		// Get current size
		info, _ := file.Stat()
		currentSize := info.Size()

		// If file has grown, read new content
		if currentSize > offset {
			file.Seek(offset, io.SeekStart)
			newContent := make([]byte, currentSize-offset)
			n, err := file.Read(newContent)
			if err != nil && err != io.EOF {
				return TailerStoppedMsg{TaskID: taskID, Error: err}
			}
			if n > 0 {
				return LogLineMsg{TaskID: taskID, Line: string(newContent[:n])}
			}
		}

		// Wait before checking again
		time.Sleep(50 * time.Millisecond)

		// Continue tailing
		select {
		case <-ctx.Done():
			return TailerStoppedMsg{TaskID: taskID, Error: nil}
		default:
			return continueTailing(taskID, path, ctx, currentSize)()
		}
	}
}
