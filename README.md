# 🤖 HIVE: Autonomous Agent Swarm Orchestrator

![HIVE Logo](.github/assets/logo.png)

[![Go Report Card](https://goreportcard.com/badge/github.com/tuanbt/hive)](https://goreportcard.com/report/github.com/tuanbt/hive)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

HIVE is a powerful, multithreaded orchestration engine for autonomous AI agents. Built in Go, it allows you to dispatch complex software engineering requirements to a "swarm" of agents that collaborate via a shared blackboard pattern.

> **"Turn a single prompt into a coordinated software delivery."**

---

## 🚀 Key Features

- **🦾 Multithreaded Worker Pool**: Run up to 32+ agents simultaneously, each with isolated session management.
- **📟 Hacker Grid TUI**: A high-density 3x2 tiled dashboard for real-time monitoring of the entire swarm pulse.
- **🧠 Auto-Planning**: Agents can generate technical plans that HIVE automatically decomposes into specialized sub-tasks.
- **🔗 Git Integration**: Automated feature branching, committing, and Pull Request creation.
- **📓 Blackboard Pattern**: Decoupled coordination via a persistent JSON task registry.

## 📺 The Hacker Grid
HIVE features a state-of-the-art Terminal UI providing real-time log streaming from every active worker in the swarm.

```text
+-----------------------+-----------+
|                       |  WORKER 1 |
|     ORCHESTRATOR      | (BACKEND) |
|      (THE BRAIN)      +-----------+
|                       |  WORKER 2 |
|                       | (FRONTEND)|
+-----------+-----------+-----------+
|  WORKER 3 |  WORKER 4 |   TASK    |
|   (BA)    |   (QA)    |  REGISTRY |
+-----------+-----------+-----------+
```

## 🛠️ Installation

```bash
# Clone the repository
git clone https://github.com/tuanbt/hive.git
cd hive

# Build the binaries
make build       # Build Orchestrator
make build-hive  # Build Hacker TUI
```

## 🚥 Quick Start

1. **Configure your agent** in `config.json` (e.g., set `agent_command` to your preferred AI driver).
2. **Start the Engine**:
   ```bash
   ./hive
   ```
3. **Launch the TUI**:
   ```bash
   ./hive
   ```
4. **Add a task**:
   In a new terminal or via the Hive TUI (Insert Mode `i`):
   ```bash
   ./hive add -title "Build a JWT Auth API" -role "backend"
   ```

## 🧩 How it Works: The Swarm Logic

1. **Planning**: A `BA` agent analyzes your high-level request and outputs a structured technical plan.
2. **Dispatching**: HIVE parses the plan and spawns specialized tasks for `Backend`, `Frontend`, and `QA` roles.
3. **Execution**: Workers pick up tasks from the registry, execute agents, and write code to the shared filesystem.
4. **Validation**: Agents conduct self-reviews and run tests.
5. **Reporting**: The Orchestrator collects results, commits the code, and updates the task status.

## 📖 Documentation

- [Architecture Guide](ARCHITECTURE.md) - Deep dive into the swarm internals.
- [Contributing](CONTRIBUTING.md) - How to build new agent drivers.

---

Built with ❤️ by TuanBT for the age of Autonomous Engineering. 🚀
