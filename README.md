# 🤖 HIVE: Autonomous Agent Swarm Orchestrator

![HIVE Logo](.github/assets/logo.png)

|[![Go Report Card](https://goreportcard.com/badge/github.com/tuanbt/hive)](https://goreportcard.com/report/github.com/tuanbt/hive)
|[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

HIVE is a powerful, multithreaded orchestration engine for autonomous AI agents. Built in Go, it allows you to dispatch complex software engineering requirements to a "swarm" of OpenCode agents that collaborate via a shared blackboard pattern.

> **"Turn a single prompt into a coordinated software delivery with OpenCode."**

---

## 🚀 Key Features

- **🦾 Multithreaded Worker Pool**: Run up to 32+ agents simultaneously, each with isolated session management.
- **📟 Hacker Grid TUI**: A high-density 3x2 tiled dashboard for real-time monitoring of entire swarm pulse.
- **🧠 Auto-Planning**: Agents can generate technical plans that HIVE automatically decomposes into specialized sub-tasks.
- **🔗 Git Integration**: Automated feature branching, committing, and Pull Request creation.
- **📓 Blackboard Pattern**: Decoupled coordination via a persistent JSON task registry.

## 📺 The Hacker Grid
> **"Be the Queen. Command the Swarm."**

HIVE is a lightweight **Go-based Autonomous Agent Orchestrator** designed for developers who want to manage a swarm of AI agents from the terminal. It uses a **Blackboard Pattern** (`tasks.json`) for coordination and features a **"Hacker Grid" TUI** for real-time monitoring.

---

## 🛠️ Installation

### Prerequisites

Before installing HIVE, you need **OpenCode** installed:

```bash
# Install OpenCode CLI
npm install -g @opencode/sdk

# Verify installation
opencode --version
```

OpenCode is an AI agent that HIVE orchestrates for software engineering tasks.

### ⚡ One-Line Install (Recommended)
Install the latest `hive` binaries (`hive` and `hive-core`) directly to your path:

```bash
curl -sL https://raw.githubusercontent.com/tuanpep/hive/main/install.sh | bash
```

### 📦 Manual Build
If you prefer building from source:

```bash
git clone https://github.com/tuanpep/hive.git
cd hive
make build-all
# Binaries will be in ./dist/
```

## 🚀 Quick Start

1. **Ensure OpenCode is installed**:
    ```bash
    opencode --version
    ```

2. **Start Swarm**:
    ```bash
    hive
    ```
    *(The orchestrator runs automatically in the background)*

3. **Command Agents**:
    - Press `i` to enter Insert Mode.
    - Type `Create a new task for the swarm`.
    - Press `Enter` to submit.
    - Watch the **Dynamic Grid** light up as agents pick up tasks!

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
