# Migration Guide: HIVE v0.2.0 - OpenCode-Only Integration

**Date:** 2025-01-10

## Overview

HIVE v0.2.0 removes support for Claude CLI and persistent agent mode, focusing exclusively on **OpenCode** in **episodic mode**. This simplifies the codebase and provides a more focused tooling experience.

## Breaking Changes

### 1. Agent Configuration

**Before:**
```json
{
  "agent_command": ["claude"],
  "agent_mode": "persistent"
}
```

**After:**
```json
{
  "agent_command": ["opencode", "run"],
  "agent_mode": "episodic"
}
```

### 2. Agent Mode

- **Persistent mode** (long-running REPL) has been **removed**
- Only **episodic mode** (one-shot command execution) is now supported

### 3. Agent Driver

- Removed ~150 lines of code related to persistent mode
- Simplified to use synchronous command execution
- No changes to orchestrator, worker pool, or task management

## Migration Steps

### If You're Using Custom `config.json`

**Step 1:** Update your agent command
```json
{
  "agent_command": ["opencode", "run"]
}
```

**Step 2:** Update your agent mode
```json
{
  "agent_mode": "episodic"
}
```

**Step 3:** Verify OpenCode is installed
```bash
opencode --version
```

If OpenCode is not installed:
```bash
npm install -g @opencode/sdk
```

### If You Were Using Claude

HIVE no longer supports Claude CLI. You must migrate to OpenCode:

1. **Install OpenCode:**
   ```bash
   npm install -g @opencode/sdk
   ```

2. **Update `config.json`:**
   - Change `agent_command` to `["opencode", "run"]`
   - Change `agent_mode` to `"episodic"`

3. **Test Your Configuration:**
   - Run a simple task to verify OpenCode works correctly
   - Check logs for any errors

## What Changed?

### Removed
- Persistent mode (long-running REPL agents)
- Claude CLI support
- Process monitoring goroutines
- Output channel infrastructure
- `DrainOutput()` method
- `waitForResponsePersistent()` method
- `readOutput()` goroutines
- `monitorProcess()` goroutine

### Added
- OpenCode integration as default
- Simplified synchronous execution
- Streamlined agent driver
- Updated documentation

## Testing Your Migration

After migrating, verify your setup:

1. **Run HIVE:**
   ```bash
   hive
   ```

2. **Add a Test Task:**
   - Press `i` in TUI
   - Type `Create a simple Go hello world program`
   - Press Enter

3. **Monitor Logs:**
   - Check that OpenCode executes correctly
   - Verify task completes successfully

## Troubleshooting

### "agent_command cannot be empty"

Ensure your `config.json` has:
```json
{
  "agent_command": ["opencode", "run"]
}
```

### "opencode: command not found"

Install OpenCode:
```bash
npm install -g @opencode/sdk
```

### "invalid log_level"

Valid log levels are: `debug`, `info`, `warn`, `error`

### Tasks Not Executing

1. Check that `agent_mode` is set to `"episodic"`
2. Verify OpenCode works independently:
   ```bash
   opencode run "test command"
   ```

## Rollback Plan

If you encounter issues and need to rollback to the previous version:

```bash
# Uninstall current version (if installed globally)
rm -f $(which hive)

# Clone and checkout previous version
git clone https://github.com/tuanpep/hive.git
cd hive
git checkout <previous-version-tag>

# Build and install
make build-all
sudo make install
```

## Questions?

If you have questions about this migration:
- Check the updated documentation in `README.md`
- Review `ARCHITECTURE.md` for implementation details
- Open an issue on GitHub: https://github.com/tuanpep/hive/issues
