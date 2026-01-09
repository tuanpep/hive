package worker

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/tuanbt/hive/internal/config"
	"github.com/tuanbt/hive/internal/task"
)

func testConfig() *config.Config {
	return &config.Config{
		AgentCommand:           []string{"bash", "-c", "while read line; do echo \"Received: $line\"; echo '### TASK_DONE ###'; done"},
		NumWorkers:             2,
		ResponseTimeoutSeconds: 5,
		MaxTaskDurationSeconds: 30,
		MaxReviewCycles:        2,
		MaxRestartAttempts:     3,
		RestartCooldownSeconds: []int{0, 0, 0},
		CompletionMarker:       "### TASK_DONE ###",
		StopTokens:             []string{"TASK_DONE"},
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestPoolStartStop(t *testing.T) {
	cfg := testConfig()
	cfg.NumWorkers = 1
	cfg.AgentCommand = []string{"cat"} // Simple command
	logger := testLogger()

	pool := NewPool(cfg, logger, ".")

	ctx, cancel := context.WithCancel(context.Background())

	// Start pool
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}

	// Give workers time to start
	time.Sleep(100 * time.Millisecond)

	if pool.ActiveWorkers() != 1 {
		t.Errorf("expected 1 active worker, got %d", pool.ActiveWorkers())
	}

	// Cancel context to trigger shutdown
	cancel()

	// Stop pool
	pool.Stop()

	// Workers should be stopped
	time.Sleep(100 * time.Millisecond)
}

func TestPoolSubmit(t *testing.T) {
	cfg := testConfig()
	cfg.NumWorkers = 1
	cfg.AgentCommand = []string{"cat"}
	logger := testLogger()

	pool := NewPool(cfg, logger, ".")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx)
	defer pool.Stop()

	// Submit a task
	testTask := task.NewTask("test-1", "Test Task", "Do something")
	if !pool.Submit(testTask) {
		t.Error("failed to submit task")
	}

	if pool.PendingTasks() != 1 {
		t.Errorf("expected 1 pending task, got %d", pool.PendingTasks())
	}
}

func TestPoolMultipleWorkers(t *testing.T) {
	cfg := testConfig()
	cfg.NumWorkers = 3
	cfg.AgentCommand = []string{"cat"}
	logger := testLogger()

	pool := NewPool(cfg, logger, ".")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx)
	defer pool.Stop()

	// Give workers time to start
	time.Sleep(200 * time.Millisecond)

	if pool.ActiveWorkers() != 3 {
		t.Errorf("expected 3 active workers, got %d", pool.ActiveWorkers())
	}
}

func TestPoolIsFull(t *testing.T) {
	cfg := testConfig()
	cfg.NumWorkers = 1 // Buffer will be 2
	cfg.AgentCommand = []string{"cat"}
	logger := testLogger()

	pool := NewPool(cfg, logger, ".")

	// Fill the buffer without starting workers
	for i := 0; i < 2; i++ {
		testTask := task.NewTask(string(rune('a'+i)), "Task", "Description")
		pool.Submit(testTask)
	}

	if !pool.IsFull() {
		t.Error("expected pool to be full")
	}
}
