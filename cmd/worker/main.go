package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tuanbt/hive/internal/agent"
	"github.com/tuanbt/hive/internal/config"
	"github.com/tuanbt/hive/internal/logger"
)

func main() {
	taskInput := flag.String("task", "", "The task description to execute")
	flag.Parse()

	if *taskInput == "" {
		fmt.Println("Error: --task argument is required")
		os.Exit(1)
	}

	// Load Config to defaults
	cfg := config.DefaultConfig()
	cfg.AgentCommand = []string{"opencode", "run"}
	cfg.AgentMode = "episodic"
	cfg.MaxRestartAttempts = 0

	// Use Console Logger
	log := logger.NewConsoleLogger(cfg)
	log.Info("Worker started", "task", *taskInput)

	pwd, _ := os.Getwd()
	driver := agent.New(cfg, log, pwd)

	// Start Agent
	if err := driver.Start(); err != nil {
		log.Error("Failed to start agent", "error", err)
		os.Exit(1)
	}
	defer driver.Stop()

	// Execution Context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Printf("\n>>> EXECUTING TASK: %s\n\n", *taskInput)

	if err := driver.SendInput(*taskInput); err != nil {
		log.Error("Failed to send input", "error", err)
		os.Exit(1)
	}

	// Wait for Result
	output, success, err := driver.WaitForResponse(ctx, nil)
	if err != nil {
		log.Error("Execution failed", "error", err)
	}

	fmt.Println("\n>>> AGENT OUTPUT:")
	fmt.Println("---------------------------------------------------")
	fmt.Println(output)
	fmt.Println("---------------------------------------------------")

	if success {
		fmt.Println("\n✅ TASK COMPLETED SUCCESSFULLY")
	} else {
		fmt.Println("\n❌ TASK FAILED OR TIMED OUT")
	}

	// Keep terminal open
	fmt.Println("\nPress Enter to close this worker...")
	fmt.Scanln()
}
