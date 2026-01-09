package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles loading, saving, and querying tasks from a JSON file.
type Manager struct {
	filePath string
	mu       sync.RWMutex
}

// NewManager creates a new task manager for the given file path.
func NewManager(filePath string) *Manager {
	return &Manager{
		filePath: filePath,
	}
}

// EnsureFile creates the tasks file if it doesn't exist.
func (m *Manager) EnsureFile() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		// Create directory if needed
		dir := filepath.Dir(m.filePath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		}

		// Create empty tasks file
		if err := os.WriteFile(m.filePath, []byte("[]"), 0644); err != nil {
			return fmt.Errorf("failed to create tasks file: %w", err)
		}
	}
	return nil
}

// LoadAll reads all tasks from the file.
func (m *Manager) LoadAll() ([]Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks file: %w", err)
	}

	return tasks, nil
}

// SaveAll writes all tasks to the file atomically.
func (m *Manager) SaveAll(tasks []Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.saveAllLocked(tasks)
}

// saveAllLocked writes tasks without acquiring the lock (caller must hold lock).
func (m *Manager) saveAllLocked(tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	// Write to temp file first, then rename (atomic)
	tmpPath := m.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, m.filePath); err != nil {
		os.Remove(tmpPath) // Clean up
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// GetNextPending returns the next pending task and marks it as claimed.
// Returns nil if no pending tasks are available.
func (m *Manager) GetNextPending() (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, err := m.loadAllLocked()
	if err != nil {
		return nil, err
	}

	// Find first pending task (respecting priority)
	var bestTask *Task
	var bestIdx int = -1
	for i := range tasks {
		if tasks[i].Status == StatusPending {
			if bestTask == nil || tasks[i].Priority > bestTask.Priority {
				bestTask = &tasks[i]
				bestIdx = i
			}
		}
	}

	if bestTask == nil {
		return nil, nil
	}

	// Return a copy
	result := tasks[bestIdx]
	return &result, nil
}

// ClaimTask atomically marks a task as in_progress for a worker.
// Returns error if task is no longer pending.
func (m *Manager) ClaimTask(taskID string, workerID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, err := m.loadAllLocked()
	if err != nil {
		return err
	}

	for i := range tasks {
		if tasks[i].ID == taskID {
			if tasks[i].Status != StatusPending {
				return fmt.Errorf("task %s is no longer pending (status: %s)", taskID, tasks[i].Status)
			}
			tasks[i].MarkInProgress(workerID)
			return m.saveAllLocked(tasks)
		}
	}

	return fmt.Errorf("task not found: %s", taskID)
}

// GetByID returns a task by its ID.
func (m *Manager) GetByID(id string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks, err := m.loadAllLocked()
	if err != nil {
		return nil, err
	}

	for i := range tasks {
		if tasks[i].ID == id {
			result := tasks[i]
			return &result, nil
		}
	}

	return nil, fmt.Errorf("task not found: %s", id)
}

// UpdateTask updates a task in the file.
func (m *Manager) UpdateTask(updated *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, err := m.loadAllLocked()
	if err != nil {
		return err
	}

	found := false
	for i := range tasks {
		if tasks[i].ID == updated.ID {
			updated.UpdatedAt = time.Now()
			tasks[i] = *updated
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("task not found: %s", updated.ID)
	}

	return m.saveAllLocked(tasks)
}

// UpdateStatus updates just the status of a task.
func (m *Manager) UpdateStatus(taskID string, status Status, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, err := m.loadAllLocked()
	if err != nil {
		return err
	}

	for i := range tasks {
		if tasks[i].ID == taskID {
			tasks[i].Status = status
			tasks[i].UpdatedAt = time.Now()
			if reason != "" {
				tasks[i].FailReason = reason
			}
			if status.IsTerminal() {
				tasks[i].CompletedAt = time.Now()
			}
			return m.saveAllLocked(tasks)
		}
	}

	return fmt.Errorf("task not found: %s", taskID)
}

// RecoverInProgress resets all in_progress tasks to pending.
// Returns the number of tasks recovered.
func (m *Manager) RecoverInProgress() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, err := m.loadAllLocked()
	if err != nil {
		return 0, err
	}

	count := 0
	for i := range tasks {
		if tasks[i].Status.IsActive() {
			tasks[i].ResetForRetry()
			count++
		}
	}

	if count > 0 {
		if err := m.saveAllLocked(tasks); err != nil {
			return 0, err
		}
	}

	return count, nil
}

// AddTask adds a new task to the file.
func (m *Manager) AddTask(t *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, err := m.loadAllLocked()
	if err != nil {
		return err
	}

	// Check for duplicate ID
	for _, existing := range tasks {
		if existing.ID == t.ID {
			return fmt.Errorf("task with ID %s already exists", t.ID)
		}
	}

	tasks = append(tasks, *t)
	return m.saveAllLocked(tasks)
}

// DeleteTask removes a task from the file.
func (m *Manager) DeleteTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, err := m.loadAllLocked()
	if err != nil {
		return err
	}

	newTasks := make([]Task, 0, len(tasks))
	found := false
	for _, t := range tasks {
		if t.ID == taskID {
			found = true
			continue
		}
		newTasks = append(newTasks, t)
	}

	if !found {
		return fmt.Errorf("task not found: %s", taskID)
	}

	return m.saveAllLocked(newTasks)
}

// CountByStatus returns the count of tasks in each status.
func (m *Manager) CountByStatus() (map[Status]int, error) {
	tasks, err := m.LoadAll()
	if err != nil {
		return nil, err
	}

	counts := make(map[Status]int)
	for _, t := range tasks {
		counts[t.Status]++
	}
	return counts, nil
}

// loadAllLocked reads tasks without acquiring lock (caller must hold lock).
func (m *Manager) loadAllLocked() ([]Task, error) {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks file: %w", err)
	}

	return tasks, nil
}
