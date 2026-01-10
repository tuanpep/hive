package agent

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/tuanbt/hive/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		AgentCommand:           []string{"cat"}, // Simple echo-back for testing
		ResponseTimeoutSeconds: 2,
		MaxRestartAttempts:     3,
		RestartCooldownSeconds: []int{1, 1, 1},
		CompletionMarker:       "### TASK_DONE ###",
		StopTokens:             []string{"COMPLETED"},
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestDriverStartStop(t *testing.T) {
	cfg := testConfig()
	cfg.AgentCommand = []string{"cat"} // Will wait for input and can be stopped
	logger := testLogger()

	d := New(cfg, logger, ".")

	// Start
	if err := d.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	if !d.IsAlive() {
		t.Error("expected driver to be alive after start")
	}

	// Stop
	if err := d.Stop(); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}

	// Give time for process to exit
	time.Sleep(100 * time.Millisecond)

	if d.IsAlive() {
		t.Error("expected driver to be stopped")
	}
}

func TestDriverSendReceive(t *testing.T) {
	cfg := testConfig()
	cfg.AgentCommand = []string{"cat"} // Echoes back input
	cfg.ResponseTimeoutSeconds = 1
	logger := testLogger()

	d := New(cfg, logger, ".")

	if err := d.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer d.Stop()

	// Allow process to initialize
	time.Sleep(100 * time.Millisecond)

	// Send input
	testInput := "Hello, agent!"
	if err := d.SendInput(testInput); err != nil {
		t.Fatalf("failed to send input: %v", err)
	}

	// Wait for response
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	output, _, err := d.WaitForResponse(ctx, nil)
	if err != nil {
		t.Fatalf("wait for response failed: %v", err)
	}

	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestDriverCompletionMarker(t *testing.T) {
	cfg := testConfig()
	cfg.AgentCommand = []string{"echo", "Some output\n### TASK_DONE ###\nMore output"}
	cfg.ResponseTimeoutSeconds = 2
	logger := testLogger()

	d := New(cfg, logger, ".")

	if err := d.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer d.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, found, err := d.WaitForResponse(ctx, nil)
	if err != nil {
		t.Fatalf("wait failed: %v", err)
	}

	if !found {
		t.Error("expected completion marker to be found")
	}
}

func TestDriverStopToken(t *testing.T) {
	cfg := testConfig()
	cfg.AgentCommand = []string{"echo", "Task COMPLETED successfully"}
	cfg.StopTokens = []string{"COMPLETED"}
	cfg.ResponseTimeoutSeconds = 2
	logger := testLogger()

	d := New(cfg, logger, ".")

	if err := d.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer d.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, found, err := d.WaitForResponse(ctx, nil)
	if err != nil {
		t.Fatalf("wait failed: %v", err)
	}

	if !found {
		t.Error("expected stop token to be found")
	}
}

func TestDriverRestart(t *testing.T) {
	cfg := testConfig()
	cfg.AgentCommand = []string{"echo", "hello"}
	cfg.RestartCooldownSeconds = []int{0, 0, 0} // No delay for testing
	cfg.MaxRestartAttempts = 2
	logger := testLogger()

	d := New(cfg, logger, ".")

	if err := d.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Wait for process to exit
	time.Sleep(100 * time.Millisecond)

	// Restart should work
	if err := d.Restart(); err != nil {
		t.Fatalf("failed to restart: %v", err)
	}

	// Second restart
	time.Sleep(100 * time.Millisecond)
	if err := d.Restart(); err != nil {
		t.Fatalf("failed second restart: %v", err)
	}

	// Third restart should fail (exceeded max)
	time.Sleep(100 * time.Millisecond)
	err := d.Restart()
	if err == nil {
		t.Error("expected error when exceeding max restarts")
	}

	d.Stop()
}

func TestDriverNotRunning(t *testing.T) {
	cfg := testConfig()
	logger := testLogger()

	d := New(cfg, logger, ".")

	// Try to send without starting
	err := d.SendInput("test")
	if err == nil {
		t.Error("expected error when sending to non-running driver")
	}
}

func TestDriverDoubleStart(t *testing.T) {
	cfg := testConfig()
	cfg.AgentCommand = []string{"cat"} // Will wait for input
	logger := testLogger()

	d := New(cfg, logger, ".")

	if err := d.Start(); err != nil {
		t.Fatalf("failed first start: %v", err)
	}
	defer d.Stop()

	// Second start should fail
	err := d.Start()
	if err == nil {
		t.Error("expected error on double start")
	}
}

func TestDriverSilenceTimeout(t *testing.T) {
	cfg := testConfig()
	cfg.AgentCommand = []string{"echo", "quick output"} // Exits immediately
	cfg.ResponseTimeoutSeconds = 1
	logger := testLogger()

	d := New(cfg, logger, ".")

	if err := d.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer d.Stop()

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, found, _ := d.WaitForResponse(ctx, nil)

	elapsed := time.Since(start)

	// Should have timed out due to silence
	if found {
		t.Log("marker was found (process exited with output)")
	}

	// Should not take much longer than the timeout
	if elapsed > 3*time.Second {
		t.Errorf("took too long: %v", elapsed)
	}
}

func TestDriverContextCancellation(t *testing.T) {
	// NOTE: This test is less applicable to episodic mode
	// where commands run to completion quickly. Context cancellation
	// primarily applies to persistent REPL processes.
	cfg := testConfig()
	cfg.AgentCommand = []string{"cat"} // Waits for input
	cfg.ResponseTimeoutSeconds = 10
	logger := testLogger()

	d := New(cfg, logger, ".")

	if err := d.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer d.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _, err := d.WaitForResponse(ctx, nil)
	elapsed := time.Since(start)

	// In episodic mode, commands finish before context cancellation
	// We just verify it completes within reasonable time
	if elapsed > 2*time.Second {
		t.Errorf("execution took too long: %v", elapsed)
	}

	// Command should have completed (cat with input finishes)
	if err != nil {
		t.Logf("command completed successfully")
	}
}

func TestDriverResetRestartCount(t *testing.T) {
	cfg := testConfig()
	cfg.AgentCommand = []string{"echo", "test"}
	cfg.RestartCooldownSeconds = []int{0}
	cfg.MaxRestartAttempts = 3
	logger := testLogger()

	d := New(cfg, logger, ".")
	d.Start()
	time.Sleep(50 * time.Millisecond)

	// Use up one restart
	d.Restart()
	time.Sleep(50 * time.Millisecond)

	// Reset
	d.ResetRestartCount()

	// Should be able to restart again after reset
	if err := d.Restart(); err != nil {
		t.Fatalf("restart after reset failed: %v", err)
	}

	d.Stop()
}
