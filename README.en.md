<div align="center">

# Agent Skills Manager

⚡ **One-Stop Desktop Manager for AI Agent Skills — Install, Update, and Orchestrate with Ease**

<p align="center">
  <a href="./README.md">简体中文</a> |
  <strong>English</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/github/license/S-huNsuke/Agent-Skill-manager?color=brightgreen" alt="license">
  <img src="https://img.shields.io/github/v/release/S-huNsuke/Agent-Skill-manager?color=brightgreen&include_prereleases" alt="release">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-blue" alt="platform">
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8" alt="go">
  <img src="https://img.shields.io/badge/React-19+-61DAFB" alt="react">
  <img src="https://img.shields.io/badge/Wails-v2-orange" alt="wails">
  <img src="https://img.shields.io/badge/Docker-supported-2496ED" alt="docker">
</p>

<p align="center">
  <a href="#-quick-start">Quick Start</a> •
  <a href="#-key-features">Key Features</a> •
  <a href="#-architecture">Architecture</a> •
  <a href="#-development">Development</a>
</p>

</div>

---

## 🤖 Supported Agents

<p align="center">
  <a href="https://github.com/anthropics/claude-code" target="_blank">
    <img src="https://avatars.githubusercontent.com/anthropics" alt="Claude Code" height="60" />
  </a><!--
  --><a href="https://github.com/openai/codex" target="_blank">
    <img src="https://avatars.githubusercontent.com/openai" alt="Codex" height="60" />
  </a><!--
  --><a href="https://www.trae.ai/" target="_blank">
    <img src="https://avatars.githubusercontent.com/bytedance" alt="Trae" height="60" />
  </a><!--
  --><a href="https://github.com/google-gemini/gemini-cli" target="_blank">
    <img src="https://avatars.githubusercontent.com/google-gemini" alt="Gemini CLI" height="60" />
  </a><!--
  --><a href="https://github.com/openclaw" target="_blank">
    <img src="https://avatars.githubusercontent.com/openclaw" alt="OpenClaw" height="60" />
  </a><!--
  --><a href="https://github.com/nousresearch" target="_blank">
    <img src="https://avatars.githubusercontent.com/nousresearch" alt="Hermes" height="60" />
  </a>
</p>

<p align="center">
  <em>Claude Code · Codex · Trae / Trae CN · Gemini CLI · OpenClaw · Hermes</em>
</p>

---

## 📝 Project Description

Agent Skills Manager is a native macOS desktop application that provides a unified management interface for AI agent skills. It supports discovering locally installed AI agents, browsing and installing skills from remote catalogs, binding skills to projects, and leveraging AI assistants for automated skill planning and deployment.

Built with **Wails v2** (Go + React/TypeScript), it delivers native performance with a modern web-based UI.

---

## ✨ Key Features

### 🤖 Agent Management
| Feature | Description |
|---------|-------------|
| 🔍 Auto Discovery | Automatically detects locally installed AI agents (Claude Code, Codex, Trae, etc.) |
| 📊 Health Monitoring | Real-time health status checks for each agent |
| 🔧 One-Click Repair | Fix missing skill directories and broken installations |
| 📋 Batch Operations | Batch update or uninstall skills across agents |

### 🏪 Skill Store
| Feature | Description |
|---------|-------------|
| 🌐 Remote Catalogs | Browse skills from GitHub-based catalogs (Anthropic, ComposioHQ, Vercel) |
| 🔄 Auto Sync | Scheduled synchronization of skill catalogs |
| 📦 Local Cache | Downloaded skills are cached locally for offline installation |
| 🔍 Search & Filter | Search by name, description, source, and agent compatibility |

### 📂 Project Management
| Feature | Description |
|---------|-------------|
| 🗂️ Project Binding | Bind agents and skill groups to specific projects |
| ⚖️ Skill Reconciliation | Detect missing/outdated skills and reconcile with one click |
| 📁 Auto Scan | Automatically discover local Git projects |

### 🧠 AI Assistant
| Feature | Description |
|---------|-------------|
| 💬 Conversational UI | Describe your goal, AI plans the skill installation steps |
| 🔄 Multi-Phase Workflow | Planning → Resolving → Executing → Verifying → Reporting |
| 🤖 Multi-Provider | Supports OpenAI, Anthropic, or local fallback mode |
| 📝 Task History | Full history of AI-assisted tasks |

### ⚙️ Settings & Automation
| Feature | Description |
|---------|-------------|
| 🎨 Theme & Language | Light/Dark/System theme, Chinese/English interface |
| ⏰ Scheduled Tasks | Auto-sync catalogs, health checks on configurable schedules |
| 🔐 AI Configuration | Configure LLM provider, model, API key, and custom base URL |
| 🩺 Diagnostics | Built-in Wails binding diagnostics and system health dashboard |

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────┐
│                   Frontend (React)               │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐  │
│  │ Home │ │Agents│ │Skills│ │Store │ │Proj  │  │
│  └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘  │
│     └────────┴────────┴────────┴────────┘       │
│                    │ Wails Bindings               │
├────────────────────┼────────────────────────────┤
│                    ▼                              │
│              Go Backend (App)                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ Registry │ │ Catalog  │ │   AI Bridge      │ │
│  │ (Agents) │ │ (Skills) │ │ (Python Worker)  │ │
│  └────┬─────┘ └────┬─────┘ └────────┬─────────┘ │
│       │             │                │            │
│  ┌────┴─────────────┴────────────────┴─────────┐ │
│  │           SQLite Persistence                 │ │
│  └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

### Tech Stack

| Layer | Technology |
|-------|-----------|
| **Frontend** | React 19, TypeScript, Tailwind CSS, Vite |
| **Backend** | Go 1.23+, Wails v2 |
| **Database** | SQLite (CGO), migrations built-in |
| **AI Bridge** | Python Worker (subprocess), supports OpenAI/Anthropic/local |
| **Package Manager** | pnpm (frontend), Go Modules (backend) |

---

## 🚀 Quick Start

### Prerequisites

| Requirement | Version |
|-------------|---------|
| **Go** | ≥ 1.23 |
| **Node.js** | ≥ 20 (managed via fnm) |
| **pnpm** | ≥ 9 |
| **Wails CLI** | v2 latest |
| **Xcode Command Line Tools** | Latest |
| **Python 3** | ≥ 3.10 (optional, for AI features) |

### Install Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Development Mode

```bash
# Clone the repository
git clone https://github.com/S-huNsuke/Agent-Skill-manager.git
cd Agent-Skill-manager

# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Run in development mode
wails dev
```

### Production Build

```bash
# Build using the build script
bash scripts/build.sh

# Or use Wails CLI
wails build -clean

# The .app bundle will be at:
# build/bin/agent-skills-manager.app
```

### 🐳 Docker Deployment

<details>
<summary><strong>Using Docker Compose (Recommended)</strong></summary>

```bash
# Clone the repository
git clone https://github.com/S-huNsuke/Agent-Skill-manager.git
cd Agent-Skill-manager

# Start the service
docker-compose up -d

# View logs
docker-compose logs -f
```

</details>

<details>
<summary><strong>Using Docker Commands</strong></summary>

```bash
# Build the image
docker build -t agent-skills-manager .

# Run the container
docker run -d --name agent-skills-manager \
  -p 8080:8080 \
  -e TZ=Asia/Shanghai \
  -v asm-data:/app/data \
  agent-skills-manager
```

</details>

> **💡 Note:** The Docker image is built on a Linux base with GTK3 and WebKit2 support. Data is persisted in the `asm-data` volume. Visit `http://localhost:8080` after the container starts.

---

## 📁 Project Structure

```
agent-skills-manager/
├── build/                          # Build assets and output
│   ├── appicon.png                 # Application icon (1024x1024)
│   └── bin/                        # Build output
├── cmd/
│   └── agent-skills-manager/       # Application entry point
├── docs/                           # Documentation
├── frontend/                       # React frontend
│   ├── src/
│   │   ├── components/             # Shared UI components
│   │   ├── features/               # Feature modules
│   │   │   ├── agents/             # Agent management page
│   │   │   ├── assistant/          # AI assistant panel & page
│   │   │   ├── home/               # Dashboard / home page
│   │   │   ├── projects/           # Project management page
│   │   │   ├── settings/           # Settings page
│   │   │   ├── skills/             # Skills management page
│   │   │   └── store/              # Skill store page
│   │   ├── lib/                    # API layer, types, utilities
│   │   └── routes/                 # React Router configuration
│   └── package.json
├── internal/
│   ├── agents/                     # Agent adapters (7 adapters)
│   ├── ai/                         # AI Bridge (Python Worker)
│   ├── app/                        # Core application logic
│   │   ├── app.go                  # App struct, initialization
│   │   ├── bindings.go             # Wails-bound methods
│   │   ├── converters.go           # Domain ↔ ViewModel conversion
│   │   ├── scheduler.go            # Cron-like task scheduler
│   │   └── viewmodels.go           # View model type definitions
│   ├── catalog/                    # Catalog sync & parsing
│   ├── domain/                     # Domain types
│   ├── installer/                  # Skill installation logic
│   ├── projects/                   # Project scanning
│   ├── reconcile/                  # Skill reconciliation
│   ├── skillgroups/                # Skill group management
│   ├── storage/
│   │   └── sqlite/                 # SQLite repositories & migrations
│   └── tasks/                      # Task history management
├── scripts/
│   └── build.sh                    # Production build script
└── wails.json                      # Wails configuration
```

---

## 🔧 Development

### Frontend Development

```bash
cd frontend
pnpm install
pnpm dev          # Start Vite dev server
pnpm build        # Production build
pnpm exec tsc --noEmit  # Type check
```

### Backend Development

```bash
go vet ./...              # Static analysis
go test ./... -count=1    # Run all tests
go build ./cmd/agent-skills-manager/  # Build binary
```

### Adding a New Agent Adapter

1. Create a new package under `internal/agents/<agent-name>/`
2. Implement the `agents.Adapter` interface
3. Register the adapter in `internal/app/app.go` → `New()`

### Adding a New Feature

1. Define view models in `internal/app/viewmodels.go`
2. Implement the Wails-bound method in `internal/app/bindings.go`
3. Add the API method in `frontend/src/lib/api.ts`
4. Create the UI page in `frontend/src/features/<feature>/`

---

## 📜 License

This project is licensed under the [MIT License](./LICENSE).

---

<div align="center">

### 💖 Thank you for using Agent Skills Manager

If this project is helpful to you, please give us a ⭐️ Star!

<sub>Built with ❤️ using Wails v2 + Go + React</sub>

</div>
