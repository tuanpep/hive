package worker

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/tuanbt/hive/internal/config"
	"github.com/tuanbt/hive/internal/task"
)

// Pool manages a pool of workers for parallel task execution.
type Pool struct {
	workers    []*Worker
	taskChan   chan *task.Task
	resultChan chan *TaskResult
	config     *config.Config
	logger     *slog.Logger
	workDir    string

	activeCount atomic.Int32
	wg          sync.WaitGroup
	started     bool
	mu          sync.Mutex
}

// NewPool creates a new worker pool.
func NewPool(cfg *config.Config, logger *slog.Logger, workDir string) *Pool {
	return &Pool{
		taskChan:   make(chan *task.Task, cfg.NumWorkers*2), // Buffer for smooth dispatching
		resultChan: make(chan *TaskResult, cfg.NumWorkers*2),
		config:     cfg,
		logger:     logger,
		workDir:    workDir,
	}
}

// Start launches all workers in the pool.
func (p *Pool) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return nil
	}
	p.started = true
	p.mu.Unlock()

	p.logger.Info("starting worker pool", "num_workers", p.config.NumWorkers)

	// Create and start workers
	for i := 1; i <= p.config.NumWorkers; i++ {
		worker := New(i, p.config, p.taskChan, p.resultChan, p.logger, p.workDir)
		p.workers = append(p.workers, worker)

		p.wg.Add(1)
		go func(w *Worker) {
			defer p.wg.Done()
			p.activeCount.Add(1)
			defer p.activeCount.Add(-1)

			if err := w.Start(ctx); err != nil {
				if ctx.Err() == nil {
					p.logger.Error("worker exited with error", "worker_id", w.ID, "error", err)
				}
			}
		}(worker)
	}

	p.logger.Info("worker pool started", "active_workers", p.config.NumWorkers)
	return nil
}

// Stop gracefully shuts down all workers.
func (p *Pool) Stop() {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	p.logger.Info("stopping worker pool")

	// Close task channel to signal workers to stop
	close(p.taskChan)

	// Wait for all workers to finish
	p.wg.Wait()

	// Close result channel after all workers are done
	close(p.resultChan)

	p.logger.Info("worker pool stopped")
}

// Submit sends a task to the pool for processing.
// Returns false if the pool is not accepting tasks (channel full or closed).
func (p *Pool) Submit(t *task.Task) bool {
	select {
	case p.taskChan <- t:
		p.logger.Debug("task submitted", "task_id", t.ID)
		return true
	default:
		p.logger.Warn("task channel full, task not submitted", "task_id", t.ID)
		return false
	}
}

// SubmitBlocking sends a task to the pool, blocking until accepted.
func (p *Pool) SubmitBlocking(ctx context.Context, t *task.Task) error {
	select {
	case p.taskChan <- t:
		p.logger.Debug("task submitted", "task_id", t.ID)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Results returns the channel for receiving task results.
func (p *Pool) Results() <-chan *TaskResult {
	return p.resultChan
}

// ActiveWorkers returns the number of currently active workers.
func (p *Pool) ActiveWorkers() int {
	return int(p.activeCount.Load())
}

// PendingTasks returns the number of tasks waiting in the queue.
func (p *Pool) PendingTasks() int {
	return len(p.taskChan)
}

// IsFull returns true if the task channel is full.
func (p *Pool) IsFull() bool {
	return len(p.taskChan) >= cap(p.taskChan)
}
