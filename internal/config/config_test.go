package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.NumWorkers != 1 {
		t.Errorf("expected NumWorkers=1, got %d", cfg.NumWorkers)
	}
	if cfg.ResponseTimeoutSeconds != 60 {
		t.Errorf("expected ResponseTimeoutSeconds=60, got %d", cfg.ResponseTimeoutSeconds)
	}
	if cfg.MaxReviewCycles != 3 {
		t.Errorf("expected MaxReviewCycles=3, got %d", cfg.MaxReviewCycles)
	}
	if cfg.CompletionMarker != "### TASK_DONE ###" {
		t.Errorf("expected CompletionMarker='### TASK_DONE ###', got %s", cfg.CompletionMarker)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"agent_command": ["test-agent"],
		"num_workers": 3,
		"response_timeout_seconds": 120,
		"log_level": "debug"
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.NumWorkers != 3 {
		t.Errorf("expected NumWorkers=3, got %d", cfg.NumWorkers)
	}
	if cfg.ResponseTimeoutSeconds != 120 {
		t.Errorf("expected ResponseTimeoutSeconds=120, got %d", cfg.ResponseTimeoutSeconds)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel=debug, got %s", cfg.LogLevel)
	}
	if len(cfg.AgentCommand) != 1 || cfg.AgentCommand[0] != "test-agent" {
		t.Errorf("expected AgentCommand=[test-agent], got %v", cfg.AgentCommand)
	}

	// Check defaults applied for unspecified fields
	if cfg.MaxReviewCycles != 3 {
		t.Errorf("expected default MaxReviewCycles=3, got %d", cfg.MaxReviewCycles)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/config.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}

	// Should return defaults
	if cfg.NumWorkers != 1 {
		t.Errorf("expected default NumWorkers=1, got %d", cfg.NumWorkers)
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(configPath, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name:    "zero workers",
			modify:  func(c *Config) { c.NumWorkers = 0 },
			wantErr: true,
		},
		{
			name:    "too many workers",
			modify:  func(c *Config) { c.NumWorkers = 100 },
			wantErr: true,
		},
		{
			name:    "zero timeout",
			modify:  func(c *Config) { c.ResponseTimeoutSeconds = 0 },
			wantErr: true,
		},
		{
			name:    "invalid log level",
			modify:  func(c *Config) { c.LogLevel = "verbose" },
			wantErr: true,
		},
		{
			name:    "empty agent command",
			modify:  func(c *Config) { c.AgentCommand = []string{} },
			wantErr: true,
		},
		{
			name:    "valid 5 workers",
			modify:  func(c *Config) { c.NumWorkers = 5 },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()

			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	cfg.NumWorkers = 5
	cfg.LogLevel = "debug"

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load it back
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	if loaded.NumWorkers != 5 {
		t.Errorf("expected NumWorkers=5, got %d", loaded.NumWorkers)
	}
	if loaded.LogLevel != "debug" {
		t.Errorf("expected LogLevel=debug, got %s", loaded.LogLevel)
	}
}
