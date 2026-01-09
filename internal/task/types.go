// Package task provides types and management for orchestrator tasks.
package task

import (
	"time"
)

// Status represents the current state of a task.
type Status string

const (
	// StatusPending indicates the task is waiting to be processed.
	StatusPending Status = "pending"

	// StatusInProgress indicates the task is currently being processed.
	StatusInProgress Status = "in_progress"

	// StatusReviewing indicates the task is in the review phase.
	StatusReviewing Status = "reviewing"

	// StatusCompleted indicates the task finished successfully.
	StatusCompleted Status = "completed"

	// StatusFailed indicates the task failed after retries.
	StatusFailed Status = "failed"
)

// IsTerminal returns true if the status is a final state.
func (s Status) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed
}

// IsActive returns true if the task is currently being worked on.
func (s Status) IsActive() bool {
	return s == StatusInProgress || s == StatusReviewing
}

// Task represents a unit of work to be processed by the orchestrator.
type Task struct {
	// ID is the unique identifier for the task.
	ID string `json:"id"`

	// Title is a short description of the task.
	Title string `json:"title"`

	// Description contains the detailed task instructions.
	Description string `json:"description"`

	// Role defines the agent persona (e.g., coder, qa).
	Role string `json:"role,omitempty"`

	// Status is the current state of the task.
	Status Status `json:"status"`

	// ContextFiles are files to load into the agent context.
	ContextFiles []string `json:"context_files,omitempty"`

	// Logs contains execution log entries.
	Logs []LogEntry `json:"logs,omitempty"`

	// CreatedAt is when the task was created.
	CreatedAt time.Time `json:"created_at,omitempty"`

	// UpdatedAt is when the task was last modified.
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	// StartedAt is when the task started processing.
	StartedAt time.Time `json:"started_at,omitempty"`

	// CompletedAt is when the task finished (success or failure).
	CompletedAt time.Time `json:"completed_at,omitempty"`

	// FailReason contains the error message if task failed.
	FailReason string `json:"fail_reason,omitempty"`

	// WorkerID is the ID of the worker processing this task.
	WorkerID int `json:"worker_id,omitempty"`

	// RetryCount tracks how many review retries have been attempted.
	RetryCount int `json:"retry_count,omitempty"`

	// Priority allows ordering tasks (higher = more important).
	Priority int `json:"priority,omitempty"`
}

// LogEntry represents a single log message for a task.
type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Phase   string    `json:"phase,omitempty"`
	Message string    `json:"message"`
	Data    any       `json:"data,omitempty"`
}

// NewTask creates a new task with the given ID, title, and description.
func NewTask(id, title, description string) *Task {
	now := time.Now()
	return &Task{
		ID:          id,
		Title:       title,
		Description: description,
		Status:      StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// AddLog appends a log entry to the task.
func (t *Task) AddLog(level, phase, message string, data any) {
	entry := LogEntry{
		Time:    time.Now(),
		Level:   level,
		Phase:   phase,
		Message: message,
		Data:    data,
	}
	t.Logs = append(t.Logs, entry)
	t.UpdatedAt = time.Now()
}

// MarkInProgress transitions the task to in_progress status.
func (t *Task) MarkInProgress(workerID int) {
	t.Status = StatusInProgress
	t.WorkerID = workerID
	t.StartedAt = time.Now()
	t.UpdatedAt = time.Now()
}

// MarkReviewing transitions the task to reviewing status.
func (t *Task) MarkReviewing() {
	t.Status = StatusReviewing
	t.UpdatedAt = time.Now()
}

// MarkCompleted transitions the task to completed status.
func (t *Task) MarkCompleted() {
	t.Status = StatusCompleted
	t.CompletedAt = time.Now()
	t.UpdatedAt = time.Now()
}

// MarkFailed transitions the task to failed status with a reason.
func (t *Task) MarkFailed(reason string) {
	t.Status = StatusFailed
	t.FailReason = reason
	t.CompletedAt = time.Now()
	t.UpdatedAt = time.Now()
}

// IncrementRetry increases the retry count and returns the new count.
func (t *Task) IncrementRetry() int {
	t.RetryCount++
	t.UpdatedAt = time.Now()
	return t.RetryCount
}

// ResetForRetry resets the task to pending status for reprocessing.
func (t *Task) ResetForRetry() {
	t.Status = StatusPending
	t.WorkerID = 0
	t.RetryCount = 0
	t.FailReason = ""
	t.StartedAt = time.Time{}
	t.CompletedAt = time.Time{}
	t.UpdatedAt = time.Now()
}

// Duration returns how long the task has been/was running.
func (t *Task) Duration() time.Duration {
	if t.StartedAt.IsZero() {
		return 0
	}
	if !t.CompletedAt.IsZero() {
		return t.CompletedAt.Sub(t.StartedAt)
	}
	return time.Since(t.StartedAt)
}
