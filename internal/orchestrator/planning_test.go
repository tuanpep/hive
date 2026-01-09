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

	"github.com/tuanbt/hive/internal/orchestrator"
	"github.com/tuanbt/hive/internal/task"
)

func TestAutoPlanning(t *testing.T) {
	// Setup environment
	cfg, tmpDir := setupTest(t)
	// Mock an agent that returns a plan
	cfg.AgentMode = "episodic"

	// The mock command effectively says:
	// "I am done. Here is my plan. ### TASK_DONE ###"
	// We use echo to simulate this output.
	// We need to pass valid JSON as a string literal to the echo command, which will then be parsed by the code.
	// The problem in the previous run was double escaping in the echo string simulating the agent output.
	// The agent outputs raw text.
	jsonPlan := `[{"title": "Subtask 1", "description": "Do subtask 1", "role": "backend"}, {"title": "Subtask 2", "description": "Do subtask 2", "role": "frontend"}]`

	output := "Analysis complete. Here is the plan.\n### PLAN_START ###\n" + jsonPlan + "\n### PLAN_END ###\nTask complete.\n### TASK_DONE ###"
	cfg.AgentCommand = []string{"echo", output}

	tasksPath := filepath.Join(tmpDir, "tasks.json")
	initialTask := task.Task{
		ID:          "planning-task",
		Title:       "Create Plan",
		Description: "Breakdown the work",
		Role:        "ba",
		Status:      task.StatusPending,
		CreatedAt:   time.Now(),
	}

	data, _ := json.Marshal([]task.Task{initialTask})
	os.WriteFile(tasksPath, data, 0644)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	o, err := orchestrator.New(cfg, logger, &MockGitClient{}, task.NewManager(tasksPath))
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

	// Wait for processing
	success := false
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		currentTasks, _ := task.NewManager(tasksPath).LoadAll()

		// We expect 3 tasks: 1 completed (planning), 2 pending (subtasks)
		if len(currentTasks) == 3 {
			if currentTasks[0].Status == task.StatusCompleted {
				// Verify subtasks
				sub1 := currentTasks[1]
				sub2 := currentTasks[2]

				if sub1.Title == "Subtask 1" && sub1.Role == "backend" &&
					sub2.Title == "Subtask 2" && sub2.Role == "frontend" {
					success = true
					break
				}
			}
		}
	}

	cancel()
	wg.Wait()

	if !success {
		currentTasks, _ := task.NewManager(tasksPath).LoadAll()
		t.Fatalf("Auto-planning failed. Expected 3 tasks, found %d. Task 0 Status: %s", len(currentTasks), currentTasks[0].Status)
	}
}
