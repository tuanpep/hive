# Contributing to HIVE

Thank you for your interest in improving HIVE! This project aims to simplify autonomous agent coordination for complex software projects.

## Development Setup

1. **Prerequisites**:
    - Go 1.21+
    - An agent binary (e.g., `opencode` or similar) configured in your path.

2. **Clone & Build**:
    ```bash
    git clone https://github.com/tuanbt/agent_orchestrator.git
    cd agent_orchestrator
    make build
    make build-hive
    ```

3. **Running Tests**:
    ```bash
    go test ./...
    ```

## Adding New Agent Drivers

If you want to support a new AI model or local agent:
1. Review `internal/agent/driver.go`.
2. Implement the interface handles the `episodic` or `persistent` communication logic.
3. Update `config.json` to point to your new agent command.

## Coding Standards

### 1. conventional-commits
We use conventional commits for automated changelog generation.
- `feat(scope): ...` for new features.
- `fix(scope): ...` for bug fixes.
- `docs(scope): ...` for documentation.

### 2. Documentation
- All exported functions and types MUST have GoDoc comments.
- Major logic changes should be reflected in `ARCHITECTURE.md`.

## Pull Request Process

1. Create a feature branch from `main`.
2. Ensure all tests pass.
3. Submit a PR with a clear description of the impact.
4. The HIVE Orchestrator itself might be used to review your code!
