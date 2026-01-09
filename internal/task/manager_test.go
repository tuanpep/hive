package task

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestManagerLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.json")

	mgr := NewManager(tasksPath)

	// Create some tasks
	task1 := NewTask("task-1", "First Task", "Do something")
	task2 := NewTask("task-2", "Second Task", "Do something else")

	// Save
	if err := mgr.SaveAll([]Task{*task1, *task2}); err != nil {
		t.Fatalf("failed to save tasks: %v", err)
	}

	// Load
	tasks, err := mgr.LoadAll()
	if err != nil {
		t.Fatalf("failed to load tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "task-1" {
		t.Errorf("expected first task ID=task-1, got %s", tasks[0].ID)
	}
}

func TestManagerGetNextPending(t *testing.T) {
	tmpDir := t.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.json")

	mgr := NewManager(tasksPath)

	task1 := NewTask("task-1", "First", "First task")
	task2 := NewTask("task-2", "Second", "Second task")
	task2.Priority = 10 // Higher priority

	if err := mgr.SaveAll([]Task{*task1, *task2}); err != nil {
		t.Fatalf("failed to save tasks: %v", err)
	}

	// Should get higher priority first
	next, err := mgr.GetNextPending()
	if err != nil {
		t.Fatalf("failed to get next pending: %v", err)
	}
	if next == nil {
		t.Fatal("expected a task, got nil")
	}
	if next.ID != "task-2" {
		t.Errorf("expected task-2 (higher priority), got %s", next.ID)
	}
}

func TestManagerClaimTask(t *testing.T) {
	tmpDir := t.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.json")

	mgr := NewManager(tasksPath)

	task1 := NewTask("task-1", "Test Task", "Description")
	if err := mgr.SaveAll([]Task{*task1}); err != nil {
		t.Fatalf("failed to save tasks: %v", err)
	}

	// Claim it
	if err := mgr.ClaimTask("task-1", 1); err != nil {
		t.Fatalf("failed to claim task: %v", err)
	}

	// Verify status changed
	task, err := mgr.GetByID("task-1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if task.Status != StatusInProgress {
		t.Errorf("expected status in_progress, got %s", task.Status)
	}
	if task.WorkerID != 1 {
		t.Errorf("expected worker_id=1, got %d", task.WorkerID)
	}

	// Try to claim again - should fail
	if err := mgr.ClaimTask("task-1", 2); err == nil {
		t.Error("expected error when claiming non-pending task")
	}
}

func TestManagerUpdateStatus(t *testing.T) {
	tmpDir := t.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.json")

	mgr := NewManager(tasksPath)

	task1 := NewTask("task-1", "Test Task", "Description")
	if err := mgr.SaveAll([]Task{*task1}); err != nil {
		t.Fatalf("failed to save tasks: %v", err)
	}

	// Update to completed
	if err := mgr.UpdateStatus("task-1", StatusCompleted, ""); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	task, _ := mgr.GetByID("task-1")
	if task.Status != StatusCompleted {
		t.Errorf("expected status completed, got %s", task.Status)
	}

	// Update to failed with reason
	if err := mgr.UpdateStatus("task-1", StatusFailed, "test failure"); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	task, _ = mgr.GetByID("task-1")
	if task.Status != StatusFailed {
		t.Errorf("expected status failed, got %s", task.Status)
	}
	if task.FailReason != "test failure" {
		t.Errorf("expected fail reason 'test failure', got %s", task.FailReason)
	}
}

func TestManagerRecoverInProgress(t *testing.T) {
	tmpDir := t.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.json")

	mgr := NewManager(tasksPath)

	task1 := NewTask("task-1", "Pending", "Pending task")
	task2 := NewTask("task-2", "In Progress", "Stuck task")
	task2.Status = StatusInProgress
	task3 := NewTask("task-3", "Completed", "Done task")
	task3.Status = StatusCompleted

	if err := mgr.SaveAll([]Task{*task1, *task2, *task3}); err != nil {
		t.Fatalf("failed to save tasks: %v", err)
	}

	count, err := mgr.RecoverInProgress()
	if err != nil {
		t.Fatalf("failed to recover: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 recovered, got %d", count)
	}

	// Verify task2 is now pending
	task, _ := mgr.GetByID("task-2")
	if task.Status != StatusPending {
		t.Errorf("expected task-2 status pending, got %s", task.Status)
	}
}

func TestManagerConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.json")

	mgr := NewManager(tasksPath)

	// Create initial tasks
	tasks := make([]Task, 10)
	for i := range tasks {
		tasks[i] = *NewTask(string(rune('a'+i)), "Task", "Description")
	}
	if err := mgr.SaveAll(tasks); err != nil {
		t.Fatalf("failed to save tasks: %v", err)
	}

	// Concurrent reads and writes
	var wg sync.WaitGroup
	errors := make(chan error, 20)

	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := mgr.LoadAll()
			if err != nil {
				errors <- err
			}
		}()
	}

	// Writers - update status
	for i := 0; i < 10; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			taskID := string(rune('a' + idx))
			err := mgr.UpdateStatus(taskID, StatusInProgress, "")
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent access error: %v", err)
	}
}

func TestManagerAddTask(t *testing.T) {
	tmpDir := t.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.json")

	mgr := NewManager(tasksPath)
	if err := mgr.EnsureFile(); err != nil {
		t.Fatalf("failed to ensure file: %v", err)
	}

	task1 := NewTask("task-1", "First", "First task")
	if err := mgr.AddTask(task1); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Verify it's there
	tasks, _ := mgr.LoadAll()
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// Try to add duplicate
	if err := mgr.AddTask(task1); err == nil {
		t.Error("expected error for duplicate task ID")
	}
}

func TestManagerCountByStatus(t *testing.T) {
	tmpDir := t.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.json")

	mgr := NewManager(tasksPath)

	task1 := NewTask("task-1", "Pending", "")
	task2 := NewTask("task-2", "Pending", "")
	task3 := NewTask("task-3", "Completed", "")
	task3.Status = StatusCompleted

	if err := mgr.SaveAll([]Task{*task1, *task2, *task3}); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	counts, err := mgr.CountByStatus()
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if counts[StatusPending] != 2 {
		t.Errorf("expected 2 pending, got %d", counts[StatusPending])
	}
	if counts[StatusCompleted] != 1 {
		t.Errorf("expected 1 completed, got %d", counts[StatusCompleted])
	}
}

func TestManagerEnsureFile(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "dir")
	tasksPath := filepath.Join(subDir, "tasks.json")

	mgr := NewManager(tasksPath)

	// Should create nested directories
	if err := mgr.EnsureFile(); err != nil {
		t.Fatalf("failed to ensure file: %v", err)
	}

	// File should exist
	if _, err := os.Stat(tasksPath); err != nil {
		t.Errorf("tasks file not created: %v", err)
	}
}
