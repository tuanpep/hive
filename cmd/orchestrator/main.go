package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tuanbt/hive/internal/config"
	"github.com/tuanbt/hive/internal/git"
	"github.com/tuanbt/hive/internal/logger"
	"github.com/tuanbt/hive/internal/orchestrator"
)

var (
	version = "dev"
)

func main() {
	// Command-line flags
	configPath := flag.String("config", "config.json", "Path to config file")
	workers := flag.Int("workers", 0, "Override num_workers (0 = use config)")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("agent-orchestrator %s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Override workers if specified
	if *workers > 0 {
		cfg.NumWorkers = *workers
	}

	// Create logger
	log, err := logger.NewSystemLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
		os.Exit(1)
	}

	log.Info("starting agent-orchestrator",
		"version", version,
		"config", *configPath,
		"workers", cfg.NumWorkers,
	)

	// Create git client
	gitClient := git.NewClient(cfg.WorkDirectory)

	// Create orchestrator
	orch, err := orchestrator.New(cfg, log, gitClient)
	if err != nil {
		log.Error("failed to create orchestrator", "error", err)
		os.Exit(1)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info("received signal, initiating shutdown", "signal", sig)
		cancel()
	}()

	// Run orchestrator
	if err := orch.Run(ctx); err != nil && err != context.Canceled {
		log.Error("orchestrator error", "error", err)
		os.Exit(1)
	}

	log.Info("agent-orchestrator exited")
}
