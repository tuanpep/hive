package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tuanbt/hive/cmd/hive/tui"
	"github.com/tuanbt/hive/internal/config"
	"github.com/tuanbt/hive/internal/task"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config.json", "Path to config file")
	showVersion := flag.Bool("version", false, "Show version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <command> [args]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  list           List all tasks\n")
		fmt.Fprintf(os.Stderr, "  add            Add a new task (usage: add -title \"...\" -role \"...\")\n")
		fmt.Fprintf(os.Stderr, "  done           Mark a task as completed (usage: done <id>)\n")
		fmt.Fprintf(os.Stderr, "  delete         Delete a task (usage: delete <id>)\n")
		fmt.Fprintf(os.Stderr, "  retry          Retry a failed task (usage: retry <id>)\n")
		fmt.Fprintf(os.Stderr, "  logs           Show logs for a task (usage: logs <id>)\n")
		fmt.Fprintf(os.Stderr, "  cleanup        Delete all completed tasks\n")
		fmt.Fprintf(os.Stderr, "  tui            Run the Terminal UI (default)\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("hive %s\n", version)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Resolve paths
	pwd, _ := os.Getwd()
	if !filepath.IsAbs(cfg.TasksFile) {
		cfg.TasksFile = filepath.Join(pwd, cfg.TasksFile)
	}
	if !filepath.IsAbs(cfg.LogDirectory) {
		cfg.LogDirectory = filepath.Join(pwd, cfg.LogDirectory)
	}

	args := flag.Args()
	cmd := "tui"
	if len(args) > 0 {
		cmd = args[0]
	}

	tm := task.NewManager(cfg.TasksFile)
	if err := tm.EnsureFile(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing tasks file: %v\n", err)
		os.Exit(1)
	}

	switch cmd {
	case "tui":
		runTUI(cfg)
	case "list":
		handleList(tm)
	case "add":
		handleAdd(tm, args[1:])
	case "done":
		handleStatusChange(tm, args[1:], task.StatusCompleted)
	case "rm", "delete":
		handleDelete(tm, args[1:])
	case "retry":
		handleRetry(tm, args[1:])
	case "logs":
		handleLogs(cfg.LogDirectory, args[1:])
	case "cleanup":
		handleCleanup(tm)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func handleLogs(logDir string, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: logs <id>\n")
		os.Exit(1)
	}
	id := args[0]
	path := filepath.Join(logDir, fmt.Sprintf("%s.log", id))
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading logs: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(content))
}

func handleCleanup(tm *task.Manager) {
	tasks, err := tm.LoadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading tasks: %v\n", err)
		os.Exit(1)
	}

	count := 0
	for _, t := range tasks {
		if t.Status == task.StatusCompleted {
			if err := tm.DeleteTask(t.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Error deleting task %s: %v\n", t.ID, err)
			} else {
				count++
			}
		}
	}
	fmt.Printf("Cleaned up %d completed tasks.\n", count)
}

func handleList(tm *task.Manager) {
	tasks, err := tm.LoadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading tasks: %v\n", err)
		os.Exit(1)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}

	fmt.Printf("%-20s %-30s %-15s %-10s\n", "ID", "TITLE", "ROLE", "STATUS")
	fmt.Println(strings.Repeat("-", 80))
	for _, t := range tasks {
		fmt.Printf("%-20s %-30.30s %-15s %-10s\n", t.ID, t.Title, t.Role, t.Status)
	}
}

func handleAdd(tm *task.Manager, args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	title := fs.String("title", "", "Task title")
	desc := fs.String("desc", "", "Task description")
	role := fs.String("role", "", "Task role (ba, backend, frontend, etc)")
	fs.Parse(args)

	if *title == "" {
		fmt.Fprintf(os.Stderr, "Error: title is required\n")
		fs.Usage()
		os.Exit(1)
	}

	// Simple ID generation
	id := fmt.Sprintf("task-%d", time.Now().Unix())

	t := task.NewTask(id, *title, *desc)
	if *role != "" {
		t.Role = *role
	}

	if err := tm.AddTask(t); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding task: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Task added: %s\n", id)
}

func handleDelete(tm *task.Manager, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: delete <id>\n")
		os.Exit(1)
	}
	id := args[0]
	if err := tm.DeleteTask(id); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting task: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Task deleted: %s\n", id)
}

func handleStatusChange(tm *task.Manager, args []string, status task.Status) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: <cmd> <id>\n")
		os.Exit(1)
	}
	id := args[0]
	if err := tm.UpdateStatus(id, status, "CLI Update"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Task %s marked as %s\n", id, status)
}

func handleRetry(tm *task.Manager, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: retry <id>\n")
		os.Exit(1)
	}
	id := args[0]
	t, err := tm.GetByID(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	t.ResetForRetry()
	if err := tm.UpdateTask(t); err != nil {
		fmt.Fprintf(os.Stderr, "Error resetting task: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Task %s reset for retry\n", id)
}

func runTUI(cfg *config.Config) {
	model := initialModel(cfg)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running hive: %v\n", err)
		os.Exit(1)
	}
}

func initialModel(cfg *config.Config) tui.Model {
	// Task List (Compact Hacker Style)
	l := list.New([]list.Item{}, tui.TaskDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	// Grid Viewports
	orchVp := viewport.New(0, 0)
	workerVps := make(map[int]viewport.Model)
	for i := 1; i <= 4; i++ {
		workerVps[i] = viewport.New(0, 0)
	}

	// Input
	ti := textinput.New()
	ti.Placeholder = "Type task title..."
	ti.Prompt = "" // Handled by View
	ti.Width = 80
	ti.Blur() // Start in selection mode

	pwd, _ := os.Getwd()
	// Resolve relative paths - ALREADY DONE in main, but good to keep for safety if used elsewhere
	tasksFile := cfg.TasksFile
	if !filepath.IsAbs(tasksFile) {
		tasksFile = filepath.Join(pwd, tasksFile)
	}
	logDir := cfg.LogDirectory
	if !filepath.IsAbs(logDir) {
		logDir = filepath.Join(pwd, logDir)
	}

	return tui.Model{
		ConfigPath:    "config.json",
		TasksFile:     tasksFile,
		LogDir:        logDir,
		TaskList:      l,
		OrchView:      orchVp,
		WorkerViews:   workerVps,
		WorkerTaskIDs: make(map[int]string),
		Input:         ti,
		FocusArea:     tui.FocusList,
	}
}
