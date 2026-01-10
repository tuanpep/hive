// Package config handles loading and validation of orchestrator configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the orchestrator configuration.
type Config struct {
	// AgentCommand is the command to start OpenCode.
	AgentCommand []string `json:"agent_command"`
	// AgentMode is the mode in which the agent operates (currently only "episodic" supported).
	AgentMode string `json:"agent_mode"`

	// NumWorkers is the number of parallel workers to run.
	NumWorkers int `json:"num_workers"`

	// ResponseTimeoutSeconds is the silence timeout for completion detection.
	ResponseTimeoutSeconds int `json:"response_timeout_seconds"`

	// MaxTaskDurationSeconds is the maximum time allowed for a single task.
	MaxTaskDurationSeconds int `json:"max_task_duration_seconds"`

	// MaxReviewCycles is the number of retry attempts for the review phase.
	MaxReviewCycles int `json:"max_review_cycles"`

	// MaxRestartAttempts is the maximum number of agent restart attempts.
	MaxRestartAttempts int `json:"max_restart_attempts"`

	// RestartCooldownSeconds is the exponential backoff for restarts.
	RestartCooldownSeconds []int `json:"restart_cooldown_seconds"`

	// CompletionMarker is the string that indicates task completion.
	CompletionMarker string `json:"completion_marker"`

	// StopTokens are additional tokens that indicate completion.
	StopTokens []string `json:"stop_tokens"`

	// LogDirectory is the directory for log files.
	LogDirectory string `json:"log_directory"`

	// LogLevel sets the logging verbosity (debug, info, warn, error).
	LogLevel string `json:"log_level"`

	// RecoverInProgressOnStartup resets in_progress tasks to pending on startup.
	RecoverInProgressOnStartup bool `json:"recover_in_progress_on_startup"`

	// TasksFile is the path to the tasks JSON file.
	TasksFile string `json:"tasks_file"`

	// WorkDirectory is the working directory for task execution.
	WorkDirectory string `json:"work_directory"`

	// GitIntegration handles git workflow automation.
	GitIntegration GitConfig `json:"git_integration"`

	// Instructions defines system prompts and rules.
	Instructions InstructionConfig `json:"instructions"`
}

// InstructionConfig holds global and role-based instructions.
type InstructionConfig struct {
	GlobalRules      []string          `json:"global_rules"`
	RoleInstructions map[string]string `json:"role_instructions"`
}

// GitConfig holds configuration for git integration.
type GitConfig struct {
	Enabled             bool   `json:"enabled"`
	BaseBranch          string `json:"base_branch"`
	Remote              string `json:"remote"`
	BranchPrefix        string `json:"branch_prefix"`
	CommitMessageFormat string `json:"commit_message_format"`
	CreatePR            bool   `json:"create_pr"`
	PRTitleFormat       string `json:"pr_title_format"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		AgentCommand:               []string{"opencode", "run"},
		AgentMode:                  "episodic",
		NumWorkers:                 1,
		ResponseTimeoutSeconds:     60,
		MaxTaskDurationSeconds:     1800, // 30 minutes
		MaxReviewCycles:            3,
		MaxRestartAttempts:         3,
		RestartCooldownSeconds:     []int{5, 15, 60},
		CompletionMarker:           "### TASK_DONE ###",
		StopTokens:                 []string{"TASK_COMPLETED", "### TASK_DONE ###"},
		LogDirectory:               "./logs",
		LogLevel:                   "info",
		RecoverInProgressOnStartup: true,
		TasksFile:                  "tasks.json",

		WorkDirectory: ".",
		GitIntegration: GitConfig{
			Enabled:             false,
			BaseBranch:          "main",
			Remote:              "origin",
			BranchPrefix:        "agent/task-",
			CommitMessageFormat: "feat: %s (Task %s)",
			CreatePR:            false,
			PRTitleFormat:       "feat: %s",
		},
		Instructions: InstructionConfig{
			GlobalRules: []string{
				"You are a part of an autonomous agent swarm.",
				"Do not usage markdown formatting for file content unless strictly necessary.",
				"Be concise and technical.",
			},
			RoleInstructions: map[string]string{
				"ba":        "You are a Business Analyst. Focus on detailed requirements, user stories, and acceptance criteria (Gherkin). If asked to plan or breakdown a feature, output the tasks in this JSON format between '### PLAN_START ###' and '### PLAN_END ###': `[{\"title\": \"...\", \"description\": \"...\", \"role\": \"...\"}]`.",
				"architect": "You are a Solutions Architect. Focus on high-level system design, patterns, scalability, and trade-offs.",
				"backend":   "You are a Senior Backend Engineer. Focus on robust server-side logic, APIs, database interactions, and performance.",
				"frontend":  "You are a Senior Frontend Engineer. Focus on responsive UI/UX, state management, and modern web frameworks.",
				"qa":        "You are a QA Engineer. Focus on comprehensive testing strategies, edge cases, and security vulnerabilities.",
			},
		},
	}
}

// Load reads configuration from a JSON file.
// If the file doesn't exist, it returns DefaultConfig.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for zero values
	cfg.applyDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// applyDefaults fills in default values for any fields that are zero/empty.
func (c *Config) applyDefaults() {
	defaults := DefaultConfig()

	if len(c.AgentCommand) == 0 {
		c.AgentCommand = defaults.AgentCommand
	}
	if c.NumWorkers <= 0 {
		c.NumWorkers = defaults.NumWorkers
	}
	if c.ResponseTimeoutSeconds <= 0 {
		c.ResponseTimeoutSeconds = defaults.ResponseTimeoutSeconds
	}
	if c.MaxTaskDurationSeconds <= 0 {
		c.MaxTaskDurationSeconds = defaults.MaxTaskDurationSeconds
	}
	if c.MaxReviewCycles <= 0 {
		c.MaxReviewCycles = defaults.MaxReviewCycles
	}
	if c.MaxRestartAttempts <= 0 {
		c.MaxRestartAttempts = defaults.MaxRestartAttempts
	}
	if len(c.RestartCooldownSeconds) == 0 {
		c.RestartCooldownSeconds = defaults.RestartCooldownSeconds
	}
	if c.CompletionMarker == "" {
		c.CompletionMarker = defaults.CompletionMarker
	}
	if len(c.StopTokens) == 0 {
		c.StopTokens = defaults.StopTokens
	}
	if c.LogDirectory == "" {
		c.LogDirectory = defaults.LogDirectory
	}
	if c.LogLevel == "" {
		c.LogLevel = defaults.LogLevel
	}
	if c.TasksFile == "" {
		c.TasksFile = defaults.TasksFile
	}
	if c.WorkDirectory == "" {
		c.WorkDirectory = defaults.WorkDirectory
	}
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.NumWorkers < 1 {
		return fmt.Errorf("num_workers must be at least 1, got %d", c.NumWorkers)
	}
	if c.NumWorkers > 10 {
		return fmt.Errorf("num_workers should not exceed 10, got %d", c.NumWorkers)
	}
	if c.ResponseTimeoutSeconds < 1 {
		return fmt.Errorf("response_timeout_seconds must be at least 1, got %d", c.ResponseTimeoutSeconds)
	}
	if c.MaxTaskDurationSeconds < 60 {
		return fmt.Errorf("max_task_duration_seconds must be at least 60, got %d", c.MaxTaskDurationSeconds)
	}
	if c.MaxReviewCycles < 1 {
		return fmt.Errorf("max_review_cycles must be at least 1, got %d", c.MaxReviewCycles)
	}
	if c.MaxRestartAttempts < 1 {
		return fmt.Errorf("max_restart_attempts must be at least 1, got %d", c.MaxRestartAttempts)
	}
	if len(c.AgentCommand) == 0 {
		return fmt.Errorf("agent_command cannot be empty")
	}

	// Validate log level
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
		// Valid
	default:
		return fmt.Errorf("invalid log_level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	return nil
}

// Save writes the configuration to a JSON file.
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
