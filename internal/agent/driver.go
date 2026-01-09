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

// OutputLine represents a line of output from the agent.
type OutputLine struct {
	Source string
	Line   string
	Time   time.Time
}

// Driver manages the lifecycle of an autonomous agent process.
// It supports both "episodic" (one-shot command execution) and
// "persistent" (long-running REPL) modes.
type Driver struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	// Episodic mode state
	inputBuf  bytes.Buffer
	sessionID string

	outputChan chan OutputLine
	config     *config.Config
	logger     *slog.Logger
	workDir    string

	isRunning    atomic.Bool
	restartCount int
	mu           sync.Mutex
	wg           sync.WaitGroup

	stopOnce sync.Once
	stopChan chan struct{}
}

// New initializes a new agent Driver instance.
func New(cfg *config.Config, logger *slog.Logger, workDir string) *Driver {
	return &Driver{
		config:     cfg,
		logger:     logger,
		workDir:    workDir,
		outputChan: make(chan OutputLine, 1000),
		stopChan:   make(chan struct{}),
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

	// Episodic Mode (e.g. OpenCode)
	if d.config.AgentMode == "episodic" {
		d.isRunning.Store(true)
		d.logger.Info("started episodic agent logic")
		return nil
	}

	// Persistent Mode (e.g. Claude)
	d.logger.Info("starting agent", "command", d.config.AgentCommand)

	cmd := exec.Command(d.config.AgentCommand[0], d.config.AgentCommand[1:]...)
	cmd.Dir = d.workDir
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("failed to create stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return fmt.Errorf("failed to create stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return fmt.Errorf("failed to start agent: %w", err)
	}

	d.cmd = cmd
	d.stdin = stdin
	d.stdout = stdout
	d.stderr = stderr
	d.isRunning.Store(true)

	d.wg.Add(2)
	go d.readOutput(stdout, "stdout")
	go d.readOutput(stderr, "stderr")

	d.wg.Add(1)
	go d.monitorProcess()

	d.logger.Info("agent started", "pid", cmd.Process.Pid)
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

	if d.config.AgentMode == "episodic" {
		d.sessionID = ""
		d.isRunning.Store(false)
		return nil
	}

	d.stopOnce.Do(func() {
		close(d.stopChan)
	})

	if d.stdin != nil {
		d.stdin.Close()
	}

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		d.logger.Debug("agent stopped gracefully")
	case <-time.After(5 * time.Second):
		if d.cmd != nil && d.cmd.Process != nil {
			d.logger.Warn("force killing agent")
			d.cmd.Process.Kill()
		}
	}

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

	if d.config.AgentMode == "episodic" {
		d.inputBuf.WriteString(text + "\n")
		return nil
	}

	if d.stdin == nil {
		return fmt.Errorf("stdin not available")
	}

	d.logger.Debug("sending input", "length", len(text))
	_, err := d.stdin.Write([]byte(text + "\n"))
	return err
}

// WaitForResponse waits for agent output.
func (d *Driver) WaitForResponse(ctx context.Context, taskLogger io.Writer) (string, bool, error) {
	if d.config.AgentMode == "episodic" {
		return d.runEpisodic(ctx, taskLogger)
	}
	return d.waitForResponsePersistent(ctx, taskLogger)
}

func (d *Driver) runEpisodic(ctx context.Context, taskLogger io.Writer) (string, bool, error) {
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

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", false, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", false, fmt.Errorf("stderr pipe: %w", err)
	}

	d.logger.Info("executing episodic command", "cmd", cmd.String())

	if err := cmd.Start(); err != nil {
		return "", false, err
	}

	// We use d.readOutput to keep consistent logging and processing.
	d.wg.Add(2)
	go d.readOutput(stdout, "stdout")
	go d.readOutput(stderr, "stderr")

	// Write input background
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var output strings.Builder
	markerFound := false

	// Wait loop
	for {
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			return output.String(), false, ctx.Err()

		case err := <-done:
			if err != nil {
				d.logger.Warn("episodic cmd finished with error", "error", err)
			} else {
				d.logger.Info("episodic cmd finished successfully")
			}

			// Drain lingering output
			drainTimeout := time.After(500 * time.Millisecond)

		DrainLoop:
			for {
				select {
				case line := <-d.outputChan:
					output.WriteString(line.Line + "\n")
					if taskLogger != nil {
						fmt.Fprintln(taskLogger, line.Line)
					}
					d.logger.Debug("output (drain)", "line", line.Line)
					if strings.Contains(line.Line, d.config.CompletionMarker) {
						markerFound = true
					}
				case <-drainTimeout:
					break DrainLoop
				default:
					if len(d.outputChan) == 0 {
						break DrainLoop
					}
					time.Sleep(10 * time.Millisecond)
				}
			}

			// Implicit success for episodic if exit code 0
			success := markerFound || (err == nil)
			return output.String(), success, nil

		case line := <-d.outputChan:
			output.WriteString(line.Line + "\n")
			if taskLogger != nil {
				fmt.Fprintln(taskLogger, line.Line)
			}
			d.logger.Debug("output", "line", line.Line)
			if strings.Contains(line.Line, d.config.CompletionMarker) {
				markerFound = true
			}
		}
	}
}

func (d *Driver) waitForResponsePersistent(ctx context.Context, taskLogger io.Writer) (string, bool, error) {
	timeout := time.Duration(d.config.ResponseTimeoutSeconds) * time.Second
	d.logger.Debug("waiting for response", "timeout", timeout)

	var output strings.Builder
	lastOutputTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return output.String(), false, ctx.Err()

		case line, ok := <-d.outputChan:
			if !ok {
				return output.String(), false, fmt.Errorf("output channel closed")
			}

			output.WriteString(line.Line)
			output.WriteString("\n")
			if taskLogger != nil {
				fmt.Fprintln(taskLogger, line.Line)
			}
			lastOutputTime = time.Now()

			d.logger.Debug("agent output", "source", line.Source, "line", line.Line)

			if strings.Contains(line.Line, d.config.CompletionMarker) {
				d.logger.Info("completion marker found")
				return output.String(), true, nil
			}

			for _, token := range d.config.StopTokens {
				if strings.Contains(line.Line, token) {
					d.logger.Info("stop token found", "token", token)
					return output.String(), true, nil
				}
			}

		case <-time.After(100 * time.Millisecond):
			if time.Since(lastOutputTime) > timeout {
				d.logger.Debug("silence timeout reached")
				return output.String(), false, nil
			}
			if !d.IsAlive() {
				return output.String(), false, fmt.Errorf("agent process died")
			}
		}
	}
}

func (d *Driver) readOutput(r io.Reader, source string) {
	defer d.wg.Done()

	buffer := make([]byte, 4096)
	var currentLine strings.Builder

	for {
		select {
		case <-d.stopChan:
			return
		default:
			n, err := r.Read(buffer)
			if n > 0 {
				data := string(buffer[:n])
				lines := strings.Split(data, "\n")
				for i, line := range lines {
					if i < len(lines)-1 {
						fullLine := currentLine.String() + line
						d.sendOutput(source, fullLine)
						currentLine.Reset()
					} else {
						currentLine.WriteString(line)
					}
				}
				// Optimization: flush partial if buffer not full
				if currentLine.Len() > 0 && n < len(buffer) {
					d.sendOutput(source, currentLine.String())
					currentLine.Reset()
				}
			}
			if err != nil {
				if currentLine.Len() > 0 {
					d.sendOutput(source, currentLine.String())
				}
				if err != io.EOF && !strings.Contains(err.Error(), "file already closed") {
					d.logger.Debug("read error", "source", source, "error", err)
				}
				return
			}
		}
	}
}

func (d *Driver) sendOutput(source, line string) {
	if line == "" {
		return
	}
	select {
	case d.outputChan <- OutputLine{
		Source: source,
		Line:   line,
		Time:   time.Now(),
	}:
	default:
	}
}

func (d *Driver) monitorProcess() {
	defer d.wg.Done()
	if d.cmd == nil || d.cmd.Process == nil {
		return
	}
	err := d.cmd.Wait()
	d.isRunning.Store(false)
	if err != nil {
		d.logger.Warn("agent process exited", "error", err)
	} else {
		d.logger.Info("agent process exited normally")
	}
}

func (d *Driver) DrainOutput(duration time.Duration) {
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		select {
		case <-d.outputChan:
		case <-time.After(100 * time.Millisecond):
			return
		}
	}
}

func (d *Driver) Output() <-chan OutputLine {
	return d.outputChan
}
