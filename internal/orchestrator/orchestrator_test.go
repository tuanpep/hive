package orchestrator_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/tuanbt/hive/internal/config"
	"github.com/tuanbt/hive/internal/orchestrator"
	"github.com/tuanbt/hive/internal/task"
)

// MockGitClient implements git.Client for testing
type MockGitClient struct {
	IsCleanFunc           func() (bool, error)
	CheckoutNewBranchFunc func(branch, base string) error
	AddAllFunc            func() error
	CommitFunc            func(message string) error
	PushFunc              func(remote, branch string) error
	CreatePRFunc          func(title, body string) error
}

func (m *MockGitClient) IsInstalled() bool { return true }
func (m *MockGitClient) IsClean() (bool, error) {
	if m.IsCleanFunc != nil {
		return m.IsCleanFunc()
	}
	return true, nil
}
func (m *MockGitClient) CheckoutNewBranch(branch, base string) error {
	if m.CheckoutNewBranchFunc != nil {
		return m.CheckoutNewBranchFunc(branch, base)
	}
	return nil
}
func (m *MockGitClient) AddAll() error {
	if m.AddAllFunc != nil {
		return m.AddAllFunc()
	}
	return nil
}
func (m *MockGitClient) Commit(message string) error {
	if m.CommitFunc != nil {
		return m.CommitFunc(message)
	}
	return nil
}
func (m *MockGitClient) Push(remote, branch string) error {
	if m.PushFunc != nil {
		return m.PushFunc(remote, branch)
	}
	return nil
}
func (m *MockGitClient) CreatePR(title, body string) error {
	if m.CreatePRFunc != nil {
		return m.CreatePRFunc(title, body)
	}
	return nil
}

func setupTest(t *testing.T) (*config.Config, string) {
	t.Helper()

	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "orchestrator_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create tasks file
	tasksPath := filepath.Join(tmpDir, "tasks.json")
	initialTasks := []task.Task{} // Empty initially
	data, _ := json.Marshal(initialTasks)
	if err := os.WriteFile(tasksPath, data, 0644); err != nil {
		t.Fatalf("failed to create tasks file: %v", err)
	}

	// Setup config
	cfg := config.DefaultConfig()
	cfg.TasksFile = tasksPath
	cfg.WorkDirectory = tmpDir
	cfg.LogDirectory = filepath.Join(tmpDir, "logs")
	os.MkdirAll(cfg.LogDirectory, 0755)

	// Use echo for unit tests to avoid needing actual agents
	cfg.AgentCommand = []string{"echo", "Worker Ready"}
	cfg.ResponseTimeoutSeconds = 2
	cfg.NumWorkers = 1

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return cfg, tmpDir
}

func TestNew(t *testing.T) {
	cfg, _ := setupTest(t)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	o, err := orchestrator.New(cfg, logger, &MockGitClient{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if o == nil {
		t.Fatal("New() returned nil")
	}
}

func TestRun_Lifecycle(t *testing.T) {
	cfg, _ := setupTest(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	o, err := orchestrator.New(cfg, logger, &MockGitClient{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := o.Run(ctx); err != nil {
			t.Logf("Run() returned: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not exit after context cancellation")
	}
}

func TestRun_TaskProcessing(t *testing.T) {
	cfg, tmpDir := setupTest(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg.AgentMode = "episodic"
	cfg.AgentCommand = []string{"echo", "Here is some work.\n### TASK_DONE ###"}

	tasksPath := filepath.Join(tmpDir, "tasks.json")
	testTask := task.Task{
		ID:        "test-task-1",
		Title:     "Unit Test Task",
		Status:    task.StatusPending,
		CreatedAt: time.Now(),
	}

	tasks := []task.Task{testTask}
	data, _ := json.Marshal(tasks)
	os.WriteFile(tasksPath, data, 0644)

	o, err := orchestrator.New(cfg, logger, &MockGitClient{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.Run(ctx)
	}()

	success := false
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)

		currentTasks, err := task.NewManager(tasksPath).LoadAll()
		if err == nil && len(currentTasks) > 0 {
			if currentTasks[0].Status == task.StatusCompleted {
				success = true
				break
			}
			if currentTasks[0].Status == task.StatusFailed {
				t.Fatalf("Task failed unexpectedly. Log: %s", currentTasks[0].FailReason)
			}
		}
	}

	cancel()
	wg.Wait()

	if !success {
		t.Fatal("Task did not transition to 'completed' within timeout")
	}
}

func TestRecoverInProgressOnStartup(t *testing.T) {
	cfg, tmpDir := setupTest(t)
	cfg.RecoverInProgressOnStartup = true
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	tasksPath := filepath.Join(tmpDir, "tasks.json")
	stuckTask := task.Task{
		ID:     "stuck-1",
		Title:  "Stuck Task",
		Status: task.StatusInProgress,
	}
	data, _ := json.Marshal([]task.Task{stuckTask})
	os.WriteFile(tasksPath, data, 0644)

	o, err := orchestrator.New(cfg, logger, &MockGitClient{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.Run(ctx)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	wg.Wait()

	mgr := task.NewManager(tasksPath)
	loaded, _ := mgr.LoadAll()
	if len(loaded) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(loaded))
	}
	if loaded[0].Status != task.StatusPending {
		t.Errorf("Task status should be reset to 'pending', got '%s'", loaded[0].Status)
	}
}

func TestGitIntegration(t *testing.T) {
	cfg, tmpDir := setupTest(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg.AgentMode = "episodic"
	cfg.AgentCommand = []string{"echo", "Done\n### TASK_DONE ###"}
	cfg.GitIntegration.Enabled = true
	cfg.GitIntegration.BaseBranch = "main"

	mockGit := &MockGitClient{}

	var checkoutCalled, commitCalled, pushCalled bool
	var checkoutBranch string

	mockGit.CheckoutNewBranchFunc = func(branch, base string) error {
		checkoutCalled = true
		checkoutBranch = branch
		return nil
	}
	mockGit.CommitFunc = func(msg string) error {
		commitCalled = true
		return nil
	}
	mockGit.PushFunc = func(remote, branch string) error {
		pushCalled = true
		return nil
	}

	tasksPath := filepath.Join(tmpDir, "tasks.json")
	testTask := task.Task{
		ID:        "git-task",
		Title:     "Git Task",
		Status:    task.StatusPending,
		CreatedAt: time.Now(),
	}
	data, _ := json.Marshal([]task.Task{testTask})
	os.WriteFile(tasksPath, data, 0644)

	o, err := orchestrator.New(cfg, logger, mockGit)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.Run(ctx)
	}()

	// Wait for completion
	success := false
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		currentTasks, _ := task.NewManager(tasksPath).LoadAll()
		if len(currentTasks) > 0 && currentTasks[0].Status == task.StatusCompleted {
			success = true
			break
		}
	}

	cancel()
	wg.Wait()

	if !success {
		t.Fatal("Task not completed")
	}

	if !checkoutCalled {
		t.Error("Git checkout not called")
	}
	if checkoutBranch != "agent/task-git-task" {
		t.Errorf("Expected branch 'agent/task-git-task', got '%s'", checkoutBranch)
	}
	if !commitCalled {
		t.Error("Git commit not called")
	}
	if !pushCalled {
		t.Error("Git push not called")
	}
}
