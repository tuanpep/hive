// Package agent provides the agent process driver.
package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tuanbt/hive/internal/config"
)

// Driver manages the lifecycle of an autonomous agent process.
// It supports episodic (one-shot command execution) mode.
type Driver struct {
	// Episodic mode state
	inputBuf strings.Builder

	config  *config.Config
	logger  *slog.Logger
	workDir string

	isRunning    atomic.Bool
	restartCount int
	mu           sync.Mutex

	stopOnce sync.Once
	stopChan chan struct{}
}

// New initializes a new agent Driver instance.
func New(cfg *config.Config, logger *slog.Logger, workDir string) *Driver {
	return &Driver{
		config:   cfg,
		logger:   logger,
		workDir:  workDir,
		stopChan: make(chan struct{}),
	}
}

// Start launches the agent logic.
func (d *Driver) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isRunning.Load() {
		return fmt.Errorf("agent is already running")
	}

	d.stopChan = make(chan struct{})
	d.stopOnce = sync.Once{}
	d.isRunning.Store(true)
	d.logger.Info("started episodic agent")
	return nil
}

// Stop terminates the agent process.
func (d *Driver) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isRunning.Load() {
		return nil
	}

	d.logger.Info("stopping agent")

	d.stopOnce.Do(func() {
		close(d.stopChan)
	})

	d.isRunning.Store(false)
	return nil
}

// Restart stops and starts the agent.
func (d *Driver) Restart() error {
	d.mu.Lock()
	if d.restartCount >= d.config.MaxRestartAttempts {
		d.mu.Unlock()
		return fmt.Errorf("max restart attempts exceeded")
	}
	// Simple backoff
	cooldown := 5 * time.Second
	if len(d.config.RestartCooldownSeconds) > 0 {
		cooldown = time.Duration(d.config.RestartCooldownSeconds[0]) * time.Second
	}
	d.restartCount++
	count := d.restartCount
	d.mu.Unlock() // unlock to allow Stop/Start to lock

	d.logger.Warn("restarting agent", "attempt", count)
	d.Stop()
	time.Sleep(cooldown)
	return d.Start()
}

// IsAlive returns true if the agent is logically running.
func (d *Driver) IsAlive() bool {
	return d.isRunning.Load()
}

// EnsureAlive restarts the agent if it's not running.
func (d *Driver) EnsureAlive() error {
	if !d.IsAlive() {
		return d.Restart()
	}
	return nil
}

// ResetRestartCount resets the restart counter.
func (d *Driver) ResetRestartCount() {
	d.mu.Lock()
	d.restartCount = 0
	d.mu.Unlock()
}

// SendInput sends text to the agent.
func (d *Driver) SendInput(text string) error {
	if !d.IsAlive() {
		return fmt.Errorf("agent is not running")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.inputBuf.WriteString(text + "\n")
	return nil
}

// WaitForResponse waits for agent output.
func (d *Driver) WaitForResponse(ctx context.Context, taskLogger io.Writer) (string, bool, error) {
	return d.execute(ctx, taskLogger)
}

func (d *Driver) execute(ctx context.Context, taskLogger io.Writer) (string, bool, error) {
	d.mu.Lock()
	input := d.inputBuf.String()
	d.inputBuf.Reset()
	d.mu.Unlock()

	args := append([]string{}, d.config.AgentCommand[1:]...)
	// Add input as positional arguments for episodic commands (e.g. 'opencode run [message]')
	if input != "" {
		args = append(args, input)
	}

	cmd := exec.Command(d.config.AgentCommand[0], args...)
	cmd.Dir = d.workDir
	cmd.Env = os.Environ()

	// Capture combined stdout and stderr
	var output strings.Builder
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Create stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", false, fmt.Errorf("stdin pipe: %w", err)
	}

	d.logger.Info("executing episodic command", "cmd", cmd.String())

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return "", false, err
	}

	// Write input to stdin and close
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()

	// Wait for command completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for completion or context cancellation
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		d.logger.Warn("command cancelled")
		return output.String(), false, ctx.Err()

	case err := <-done:
		finalOutput := stdoutBuf.String() + stderrBuf.String()
		output.WriteString(finalOutput)

		if taskLogger != nil {
			fmt.Fprintln(taskLogger, finalOutput)
		}

		if err != nil {
			d.logger.Warn("episodic cmd finished with error", "error", err)
		} else {
			d.logger.Info("episodic cmd finished successfully")
		}

		// Check for completion marker
		markerFound := strings.Contains(finalOutput, d.config.CompletionMarker)
		for _, token := range d.config.StopTokens {
			if strings.Contains(finalOutput, token) {
				markerFound = true
				break
			}
		}

		// Implicit success for episodic if exit code 0 or marker found
		success := markerFound || (err == nil)
		return output.String(), success, nil
	}
}
