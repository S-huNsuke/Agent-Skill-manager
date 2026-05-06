<div align="center">

# Agent Skills Manager

⚡ **一站式 AI 代理技能桌面管理器 — 轻松安装、更新与编排**

<p align="center">
  <strong>简体中文</strong> |
  <a href="./README.en.md">English</a>
</p>

<p align="center">
  <img src="https://img.shields.io/github/license/S-huNsuke/Agent-Skill-manager?color=brightgreen" alt="license">
  <img src="https://img.shields.io/github/v/release/S-huNsuke/Agent-Skill-manager?color=brightgreen&include_prereleases" alt="release">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-blue" alt="platform">
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8" alt="go">
  <img src="https://img.shields.io/badge/React-19+-61DAFB" alt="react">
  <img src="https://img.shields.io/badge/Wails-v2-orange" alt="wails">
  <img src="https://img.shields.io/badge/Docker-支持-2496ED" alt="docker">
</p>

<p align="center">
  <a href="#-快速开始">快速开始</a> •
  <a href="#-核心功能">核心功能</a> •
  <a href="#-架构设计">架构设计</a> •
  <a href="#-开发指南">开发指南</a>
</p>

</div>

---

## 🤖 支持的代理

<p align="center">
  <a href="https://github.com/anthropics/claude-code" target="_blank">
    <img src="/docs/images/claude-code.png" alt="Claude Code" height="60" />
  </a><!--
  --><a href="https://github.com/openai/codex" target="_blank">
    <img src="/docs/images/codex.png" alt="Codex" height="60" />
  </a><!--
  --><a href="https://www.trae.ai/" target="_blank">
    <img src="/docs/images/trae.png" alt="Trae" height="60" />
  </a><!--
  --><a href="https://github.com/google-gemini/gemini-cli" target="_blank">
    <img src="/docs/images/gemini-cli.png" alt="Gemini CLI" height="60" />
  </a><!--
  --><a href="https://github.com/openclaw" target="_blank">
    <img src="/docs/images/openclaw.png" alt="OpenClaw" height="60" />
  </a><!--
  --><a href="https://github.com/nousresearch" target="_blank">
    <img src="/docs/images/hermes.png" alt="Hermes" height="60" />
  </a>
</p>

<p align="center">
  <em>Claude Code · Codex · Trae / Trae CN · Gemini CLI · OpenClaw · Hermes</em>
</p>

---

## 📝 项目简介

Agent Skills Manager 是一款原生 macOS 桌面应用，为 AI 代理技能提供统一的管理界面。支持自动发现本地已安装的 AI 代理、浏览和安装远程目录中的技能、将技能绑定到项目，以及利用 AI 助手自动规划技能部署。

基于 **Wails v2**（Go + React/TypeScript）构建，兼具原生性能和现代化 Web UI。

---

## ✨ 核心功能

### 🤖 代理管理
| 功能 | 描述 |
|------|------|
| 🔍 自动发现 | 自动检测本地已安装的 AI 代理（Claude Code、Codex、Trae 等） |
| 📊 健康监控 | 实时检查每个代理的健康状态 |
| 🔧 一键修复 | 修复缺失的技能目录和损坏的安装 |
| 📋 批量操作 | 批量更新或卸载跨代理的技能 |

### 🏪 技能商店
| 功能 | 描述 |
|------|------|
| 🌐 远程目录 | 浏览来自 GitHub 的技能目录（Anthropic、ComposioHQ、Vercel） |
| 🔄 自动同步 | 定时同步技能目录 |
| 📦 本地缓存 | 已下载的技能缓存在本地，支持离线安装 |
| 🔍 搜索过滤 | 按名称、描述、来源和代理兼容性搜索 |

### 📂 项目管理
| 功能 | 描述 |
|------|------|
| 🗂️ 项目绑定 | 将代理和技能组绑定到特定项目 |
| ⚖️ 技能协调 | 检测缺失/过时的技能，一键协调 |
| 📁 自动扫描 | 自动发现本地 Git 项目 |

### 🧠 AI 助手
| 功能 | 描述 |
|------|------|
| 💬 对话式交互 | 描述你的目标，AI 自动规划技能安装步骤 |
| 🔄 多阶段工作流 | 规划 → 解析 → 执行 → 验证 → 报告 |
| 🤖 多供应商 | 支持 OpenAI、Anthropic 或本地回退模式 |
| 📝 任务历史 | 完整的 AI 辅助任务历史记录 |

### ⚙️ 设置与自动化
| 功能 | 描述 |
|------|------|
| 🎨 主题与语言 | 亮色/暗色/跟随系统主题，中文/英文界面 |
| ⏰ 定时任务 | 可配置的自动同步目录、健康检查计划 |
| 🔐 AI 配置 | 配置 LLM 供应商、模型、API 密钥和自定义 Base URL |
| 🩺 诊断工具 | 内置 Wails 绑定诊断和系统健康仪表盘 |

---

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────┐
│                   前端 (React)                    │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐  │
│  │ 首页 │ │ 代理 │ │ 技能 │ │ 商店 │ │ 项目 │  │
│  └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘  │
│     └────────┴────────┴────────┴────────┘       │
│                    │ Wails 绑定                   │
├────────────────────┼────────────────────────────┤
│                    ▼                              │
│              Go 后端 (App)                        │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ 注册表   │ │  目录    │ │   AI 桥接        │ │
│  │ (代理)   │ │ (技能)   │ │ (Python Worker)  │ │
│  └────┬─────┘ └────┬─────┘ └────────┬─────────┘ │
│       │             │                │            │
│  ┌────┴─────────────┴────────────────┴─────────┐ │
│  │           SQLite 持久化层                    │ │
│  └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

### 技术栈

| 层级 | 技术 |
|------|------|
| **前端** | React 19, TypeScript, Tailwind CSS, Vite |
| **后端** | Go 1.23+, Wails v2 |
| **数据库** | SQLite (CGO)，内置迁移 |
| **AI 桥接** | Python Worker（子进程），支持 OpenAI/Anthropic/本地模式 |
| **包管理** | pnpm（前端），Go Modules（后端） |

---

## 🚀 快速开始

### 环境要求

| 依赖 | 版本 |
|------|------|
| **Go** | ≥ 1.23 |
| **Node.js** | ≥ 20（通过 fnm 管理） |
| **pnpm** | ≥ 9 |
| **Wails CLI** | v2 最新版 |
| **Xcode 命令行工具** | 最新版 |
| **Python 3** | ≥ 3.10（可选，AI 功能需要） |

### 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 开发模式

```bash
# 克隆仓库
git clone https://github.com/S-huNsuke/Agent-Skill-manager.git
cd Agent-Skill-manager

# 安装前端依赖
cd frontend && pnpm install && cd ..

# 开发模式运行
wails dev
```

### 生产构建

```bash
# 使用构建脚本
bash scripts/build.sh

# 或使用 Wails CLI
wails build -clean

# 构建产物位于：
# build/bin/agent-skills-manager.app
```

### 🐳 Docker 部署

<details>
<summary><strong>使用 Docker Compose（推荐）</strong></summary>

```bash
# 克隆仓库
git clone https://github.com/S-huNsuke/Agent-Skill-manager.git
cd Agent-Skill-manager

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

</details>

<details>
<summary><strong>使用 Docker 命令</strong></summary>

```bash
# 构建镜像
docker build -t agent-skills-manager .

# 运行容器
docker run -d --name agent-skills-manager \
  -p 8080:8080 \
  -e TZ=Asia/Shanghai \
  -v asm-data:/app/data \
  agent-skills-manager
```

</details>

> **💡 提示：** Docker 镜像基于 Linux 构建，包含 GTK3 和 WebKit2 支持。数据持久化在 `asm-data` 卷中。容器启动后访问 `http://localhost:8080` 即可使用。

---

## 📁 项目结构

```
agent-skills-manager/
├── build/                          # 构建资源和输出
│   ├── appicon.png                 # 应用图标 (1024x1024)
│   └── bin/                        # 构建输出
├── cmd/
│   └── agent-skills-manager/       # 应用入口
├── docs/                           # 文档
├── frontend/                       # React 前端
│   ├── src/
│   │   ├── components/             # 共享 UI 组件
│   │   ├── features/               # 功能模块
│   │   │   ├── agents/             # 代理管理页
│   │   │   ├── assistant/          # AI 助手面板和页面
│   │   │   ├── home/               # 仪表盘/首页
│   │   │   ├── projects/           # 项目管理页
│   │   │   ├── settings/           # 设置页
│   │   │   ├── skills/             # 技能管理页
│   │   │   └── store/              # 技能商店页
│   │   ├── lib/                    # API 层、类型、工具
│   │   └── routes/                 # React Router 配置
│   └── package.json
├── internal/
│   ├── agents/                     # 代理适配器（7 个）
│   ├── ai/                         # AI 桥接（Python Worker）
│   ├── app/                        # 核心应用逻辑
│   │   ├── app.go                  # App 结构体、初始化
│   │   ├── bindings.go             # Wails 绑定方法
│   │   ├── converters.go           # 领域 ↔ 视图模型转换
│   │   ├── scheduler.go            # 定时任务调度器
│   │   └── viewmodels.go           # 视图模型类型定义
│   ├── catalog/                    # 目录同步与解析
│   ├── domain/                     # 领域类型
│   ├── installer/                  # 技能安装逻辑
│   ├── projects/                   # 项目扫描
│   ├── reconcile/                  # 技能协调
│   ├── skillgroups/                # 技能组管理
│   ├── storage/
│   │   └── sqlite/                 # SQLite 仓库和迁移
│   └── tasks/                      # 任务历史管理
├── scripts/
│   └── build.sh                    # 生产构建脚本
└── wails.json                      # Wails 配置
```

---

## 🔧 开发指南

### 前端开发

```bash
cd frontend
pnpm install
pnpm dev                # 启动 Vite 开发服务器
pnpm build              # 生产构建
pnpm exec tsc --noEmit  # 类型检查
```

### 后端开发

```bash
go vet ./...              # 静态分析
go test ./... -count=1    # 运行所有测试
go build ./cmd/agent-skills-manager/  # 构建二进制
```

### 添加新的代理适配器

1. 在 `internal/agents/<agent-name>/` 下创建新包
2. 实现 `agents.Adapter` 接口
3. 在 `internal/app/app.go` → `New()` 中注册适配器

### 添加新功能

1. 在 `internal/app/viewmodels.go` 中定义视图模型
2. 在 `internal/app/bindings.go` 中实现 Wails 绑定方法
3. 在 `frontend/src/lib/api.ts` 中添加 API 方法
4. 在 `frontend/src/features/<feature>/` 中创建 UI 页面

---

## 📜 许可证

本项目基于 [MIT 许可证](./LICENSE) 开源。

---

<div align="center">

### 💖 感谢使用 Agent Skills Manager

如果这个项目对你有帮助，请给我们一个 ⭐️ Star！

<sub>使用 Wails v2 + Go + React 用 ❤️ 构建</sub>

</div>
