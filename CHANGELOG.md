# Changelog

All notable changes to HIVE will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2025-01-10

### Breaking Changes
- **Removed Claude support and persistent agent mode**
- Default agent changed from Claude to OpenCode
- Default mode changed from `persistent` to `episodic`
- Users must update `config.json` to use `["opencode", "run"]`
- Users must set `agent_mode` to `"episodic"`

### Changed
- **Simplified agent driver** by removing persistent mode logic (~150 lines removed)
- **Streamlined execution model** - now uses synchronous command execution
- **Updated documentation** to reflect OpenCode-only integration
- **Removed obsolete code**:
  - Removed `waitForResponsePersistent()` method
  - Removed `readOutput()` goroutines
  - Removed `monitorProcess()` goroutine
  - Removed `DrainOutput()` method
  - Removed persistent REPL process management

### Fixed
- Fixed agent stdin handling for episodic commands
- Removed dependency on output channel (no longer needed)

### Documentation
- Added OpenCode installation prerequisites to README
- Updated CONTRIBUTING.md with OpenCode-specific instructions
- Updated ARCHITECTURE.md to remove persistent mode references
- Created MIGRATION.md guide for existing users

## Migration

See [MIGRATION.md](MIGRATION.md) for detailed migration instructions from previous versions.

## [Previous Versions]

For versions before 2025-01-10, refer to git history.
