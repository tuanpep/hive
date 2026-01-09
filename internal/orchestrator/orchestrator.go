// Package orchestrator provides the main orchestration logic.
package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/tuanbt/hive/internal/config"
	"github.com/tuanbt/hive/internal/git"
	"github.com/tuanbt/hive/internal/task"
	"github.com/tuanbt/hive/internal/worker"
)

// Orchestrator manages the end-to-end task processing workflow.
// It coordinates between the task manager (registry), the worker pool,
// and optional git integration for automated pull requests.
type Orchestrator struct {
	config      *config.Config
	taskManager *task.Manager
	workerPool  *worker.Pool
	logger      *slog.Logger
	gitClient   git.Client

	wg       sync.WaitGroup
	stopChan chan struct{}
}

// New initializes a new Orchestrator instance with the provided dependencies.
// It ensures the task registry file exists before returning.
func New(cfg *config.Config, logger *slog.Logger, gitClient git.Client) (*Orchestrator, error) {
	taskMgr := task.NewManager(cfg.TasksFile)
	if err := taskMgr.EnsureFile(); err != nil {
		return nil, err
	}

	pool := worker.NewPool(cfg, logger, cfg.WorkDirectory)

	return &Orchestrator{
		config:      cfg,
		taskManager: taskMgr,
		workerPool:  pool,
		logger:      logger,
		gitClient:   gitClient,
		stopChan:    make(chan struct{}),
	}, nil
}

// Run starts the orchestrator and blocks until context is cancelled.
func (o *Orchestrator) Run(ctx context.Context) error {
	o.logger.Info("orchestrator starting",
		"num_workers", o.config.NumWorkers,
		"tasks_file", o.config.TasksFile,
	)

	// Recover stuck tasks
	if o.config.RecoverInProgressOnStartup {
		recovered, err := o.taskManager.RecoverInProgress()
		if err != nil {
			o.logger.Error("failed to recover in-progress tasks", "error", err)
		} else if recovered > 0 {
			o.logger.Info("recovered stuck tasks", "count", recovered)
		}
	}

	// Log initial task counts
	counts, _ := o.taskManager.CountByStatus()
	o.logger.Info("task status summary",
		"pending", counts[task.StatusPending],
		"in_progress", counts[task.StatusInProgress],
		"completed", counts[task.StatusCompleted],
		"failed", counts[task.StatusFailed],
	)

	// Start worker pool
	if err := o.workerPool.Start(ctx); err != nil {
		return err
	}

	// Start dispatcher goroutine
	o.wg.Add(1)
	go o.dispatchTasks(ctx)

	// Start result handler goroutine
	o.wg.Add(1)
	go o.handleResults(ctx)

	// Wait for shutdown
	<-ctx.Done()
	o.logger.Info("shutdown signal received")

	return o.Shutdown(ctx)
}

// dispatchTasks polls for pending tasks and submits them to the pool.
func (o *Orchestrator) dispatchTasks(ctx context.Context) {
	defer o.wg.Done()

	o.logger.Info("task dispatcher started")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			o.logger.Info("task dispatcher stopping")
			return

		case <-ticker.C:
			// Check if pool can accept tasks
			if o.workerPool.IsFull() {
				continue
			}

			// Get next pending task
			t, err := o.taskManager.GetNextPending()
			if err != nil {
				o.logger.Error("failed to get next task", "error", err)
				continue
			}

			if t == nil {
				// No pending tasks
				continue
			}

			// Try to claim the task
			workerID := 0 // Will be set by worker
			if err := o.taskManager.ClaimTask(t.ID, workerID); err != nil {
				o.logger.Warn("failed to claim task", "task_id", t.ID, "error", err)
				continue
			}

			// Handle Git Integration
			if o.config.GitIntegration.Enabled {
				// Ensure workspace is clean
				if clean, err := o.gitClient.IsClean(); err != nil || !clean {
					o.logger.Warn("cannot dispatch task: git working directory not clean", "task_id", t.ID)
					o.taskManager.UpdateStatus(t.ID, task.StatusPending, "")
					continue
				}

				// Create and checkout feature branch
				branchName := fmt.Sprintf("%s%s", o.config.GitIntegration.BranchPrefix, t.ID)
				if err := o.gitClient.CheckoutNewBranch(branchName, o.config.GitIntegration.BaseBranch); err != nil {
					o.logger.Error("failed to create git branch", "task_id", t.ID, "error", err)
					o.taskManager.UpdateStatus(t.ID, task.StatusFailed, fmt.Sprintf("git branch failed: %v", err))
					continue
				}
				o.logger.Info("created git branch", "branch", branchName)
			}

			// Submit to pool
			if !o.workerPool.Submit(t) {
				// Failed to submit, reset task status
				o.taskManager.UpdateStatus(t.ID, task.StatusPending, "")
				o.logger.Warn("failed to submit task to pool", "task_id", t.ID)
				continue
			}

			o.logger.Info("task dispatched", "task_id", t.ID, "title", t.Title)
		}
	}
}

// handleResults processes results from the worker pool.
func (o *Orchestrator) handleResults(ctx context.Context) {
	defer o.wg.Done()

	o.logger.Info("result handler started")

	for result := range o.workerPool.Results() {
		o.processResult(result)
	}

	o.logger.Info("result handler stopped")
}

// processResult handles a single task result.
func (o *Orchestrator) processResult(result *worker.TaskResult) {
	t := result.Task

	o.logger.Info("task completed",
		"task_id", t.ID,
		"title", t.Title,
		"status", result.Status,
		"worker_id", result.WorkerID,
		"duration", result.Duration,
	)

	// Update task status
	reason := ""
	if result.Error != nil {
		reason = result.Error.Error()
		o.logger.Error("task failed", "task_id", t.ID, "error", reason)
	}

	if err := o.taskManager.UpdateStatus(t.ID, result.Status, reason); err != nil {
		o.logger.Error("failed to update task status", "task_id", t.ID, "error", err)
	}

	// Add new tasks if any (auto-planning)
	if len(result.NewTasks) > 0 {
		o.logger.Info("adding new tasks from agent plan", "count", len(result.NewTasks))
		for _, nt := range result.NewTasks {
			if err := o.taskManager.AddTask(nt); err != nil {
				o.logger.Error("failed to add new task", "title", nt.Title, "error", err)
			}
		}
	}

	// Handle Git Integration (Commit/Push)
	if result.Status == task.StatusCompleted && o.config.GitIntegration.Enabled {
		o.logger.Info("committing changes to git", "task_id", t.ID)

		if err := o.gitClient.AddAll(); err != nil {
			o.logger.Error("git add failed", "task_id", t.ID, "error", err)
		} else {
			msg := fmt.Sprintf(o.config.GitIntegration.CommitMessageFormat, t.Title, t.ID)
			if err := o.gitClient.Commit(msg); err != nil {
				o.logger.Error("git commit failed", "task_id", t.ID, "error", err)
			} else {
				branchName := fmt.Sprintf("%s%s", o.config.GitIntegration.BranchPrefix, t.ID)
				if err := o.gitClient.Push(o.config.GitIntegration.Remote, branchName); err != nil {
					// Don't fail the task, just log error
					o.logger.Error("git push failed", "task_id", t.ID, "error", err)
				} else if o.config.GitIntegration.CreatePR {
					if err := o.gitClient.CreatePR(t.Title, t.Description); err != nil {
						o.logger.Error("git pr create failed", "task_id", t.ID, "error", err)
					} else {
						o.logger.Info("git pr created successfully", "task_id", t.ID)
					}
				}
			}
		}
	}

	// Log current counts
	counts, _ := o.taskManager.CountByStatus()
	o.logger.Debug("task status summary",
		"pending", counts[task.StatusPending],
		"in_progress", counts[task.StatusInProgress],
		"completed", counts[task.StatusCompleted],
		"failed", counts[task.StatusFailed],
	)
}

// Shutdown gracefully stops the orchestrator.
func (o *Orchestrator) Shutdown(ctx context.Context) error {
	o.logger.Info("shutting down orchestrator")

	// Signal stop
	close(o.stopChan)

	// Stop worker pool (waits for in-flight tasks)
	o.workerPool.Stop()

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		o.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		o.logger.Info("orchestrator shutdown complete")
	case <-time.After(30 * time.Second):
		o.logger.Warn("shutdown timeout, forcing exit")
	}

	// Final status report
	counts, _ := o.taskManager.CountByStatus()
	o.logger.Info("final task status",
		"pending", counts[task.StatusPending],
		"in_progress", counts[task.StatusInProgress],
		"completed", counts[task.StatusCompleted],
		"failed", counts[task.StatusFailed],
	)

	return nil
}
