package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Client provides an interface for git operations.
type Client interface {
	IsInstalled() bool
	IsClean() (bool, error)
	CheckoutNewBranch(branch, base string) error
	AddAll() error
	Commit(message string) error
	Push(remote, branch string) error
	CreatePR(title, body string) error
}

// OSClient implements Client using the os/exec package.
type OSClient struct {
	workDir string
}

// NewClient returns a new OSClient.
func NewClient(workDir string) *OSClient {
	return &OSClient{workDir: workDir}
}

// Run executes a git command.
func (c *OSClient) Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = c.workDir
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s failed: %w (stderr: %s)", args[0], err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// IsInstalled checks if git is available.
func (c *OSClient) IsInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// IsClean checks if the working directory is clean.
func (c *OSClient) IsClean() (bool, error) {
	out, err := c.Run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out == "", nil
}

// CheckoutNewBranch creates and checks out a new branch from a base.
func (c *OSClient) CheckoutNewBranch(branch, base string) error {
	// Update base
	// c.Run("fetch", "origin", base) // Optional?
	// Check if branch exists?
	// For now, assume it's new.
	_, err := c.Run("checkout", "-b", branch, base)
	return err
}

// AddAll stages all changes.
func (c *OSClient) AddAll() error {
	_, err := c.Run("add", ".")
	return err
}

// Commit creates a commit.
func (c *OSClient) Commit(message string) error {
	_, err := c.Run("commit", "-m", message)
	return err
}

// Push pushes the branch to remote.
func (c *OSClient) Push(remote, branch string) error {
	_, err := c.Run("push", "-u", remote, branch)
	return err
}

// CreatePR creates a PR using gh CLI.
func (c *OSClient) CreatePR(title, body string) error {
	// Check if gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh cli not found")
	}

	cmd := exec.Command("gh", "pr", "create", "--title", title, "--body", body)
	cmd.Dir = c.workDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh pr create failed: %w (output: %s)", err, string(out))
	}
	return nil
}
