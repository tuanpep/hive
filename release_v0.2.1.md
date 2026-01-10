# Release Notes: v0.2.1

## 🚀 Key Features
- **Optimistic UI**: Zero-latency feedback when creating tasks.
- **Hive Mind Sidebar**: Real-time visualization of agent "thought" logs.
- **Slash Commands**: 
  - `/quit`, `/help`
  - `/retry`: Resets failed tasks.
  - `/nuke`: Fails all active tasks (Emergency Stop).
- **Context Mentions**: User can type `@` to mention/include files in task context.
- **Headless Mode**: Run orchestrator without UI using `./hive -headless`.
- **Git Flexibility**: Use `./hive -no-git` to bypass dirty workspace checks.
- **System Visibility**: Dashboard now displays system logs (e.g., Git errors) when no workers are active.

## 🛠️ Fixes
- Fixed TUI freeze when file watchers stopped.
- Fixed "Silent Failure" state by exposing system logs.
