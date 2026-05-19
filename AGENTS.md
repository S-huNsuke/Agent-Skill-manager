# Agent Skills Manager — 代理支持说明

本文档描述应用支持的 AI 代理类型、发现机制和技能管理方式。

## 支持的代理

| 代理 | ID | 技能目录 | 状态 |
|------|----|---------|------|
| Claude Code | `claude-code` | `~/.claude/plugins/` | 完整支持 |
| Codex | `codex` | `~/.codex/plugins/` | 完整支持 |
| Trae | `trae` | `~/.trae/skills/` | 完整支持 |
| Gemini CLI | `gemini-cli` | `~/.gemini/plugins/` | 完整支持 |
| OpenClaw | `openclaw` | `~/.openclaw/plugins/` | 完整支持 |
| Hermes | `hermes` | `~/.hermes/skills/` | 完整支持 |

## 代理发现

应用启动时自动扫描上述目录。每个代理的健康状态分为三种：

- `healthy`：代理已安装，技能目录存在且可读写
- `degraded`：代理已安装，但技能目录缺失或权限不足（可一键修复）
- `not_installed`：未检测到该代理

## 技能目录结构

每个代理的技能目录下，每个技能占一个子目录：

```
~/.claude/plugins/
├── my-skill/
│   ├── SKILL.md        # 技能描述（必须）
│   └── ...             # 其他技能文件
└── another-skill/
    └── SKILL.md
```

技能安装时会将技能目录复制到对应代理的 `plugins/` 目录下。

## 代理适配器接口

每个代理在 `internal/agents/` 下有对应的适配器，实现以下接口：

```go
type Adapter interface {
    Discover(ctx context.Context) ([]AgentInstall, error)
    ListInstalledSkills(ctx context.Context, install AgentInstall) ([]Skill, error)
    InstallSkill(ctx context.Context, install AgentInstall, skill SkillSource) error
    UninstallSkill(ctx context.Context, install AgentInstall, skillName string) error
    UpdateSkill(ctx context.Context, install AgentInstall, skill SkillSource) error
    ValidateSkill(ctx context.Context, install AgentInstall, skillName string) (SkillHealth, error)
    RepairInstall(ctx context.Context, install AgentInstall) error
}
```

## 添加新代理

1. 在 `internal/agents/` 下创建新目录，实现 `Adapter` 接口
2. 在 `internal/agents/registry.go` 的 `NewRegistry` 中注册适配器
3. 在 `internal/app/viewmodels.go` 中添加对应的 `AgentID` 常量
4. 前端 `frontend/src/lib/types.ts` 中更新 `AgentViewModel` 的 `agentType` 枚举（如有）

## 技能来源

技能可以从以下来源安装：

- **远程商店**：通过商店页面从 GitHub 仓库同步的技能目录安装
- **本地路径**：直接指定本地目录路径安装
- **AI 助手**：通过 AI 助手对话，自动规划并安装所需技能

## 项目绑定

项目可以绑定一个代理和一个技能组。绑定后：

1. 项目激活时自动将技能组中的技能安装到绑定代理
2. 技能协调功能可检测技能组与实际安装状态的差异
3. 生成协调计划（需安装 / 需卸载 / 需更新），用户确认后执行


<claude-mem-context>
# Memory Context

# [New project 2] recent context, 2026-05-10 5:31pm GMT+8

Legend: 🎯session 🔴bugfix 🟣feature 🔄refactor ✅change 🔵discovery ⚖️decision 🚨security_alert 🔐security_note
Format: ID TIME TYPE TITLE
Fetch details: get_observations([IDs]) | Search: mem-search skill

Stats: 50 obs (16,111t read) | 321,757t work | 95% savings

### May 7, 2026
S83 Review project implementation against planning document to verify all features are fully implemented and check for code errors (May 7 at 9:54 PM)
S84 Troubleshoot AI assistant functionality failure and investigate why backend is not being called when user submits goals (May 7 at 9:58 PM)
S85 Remove or disable SessionStart hook in ~/.codex/hooks.json and check for related files causing similar issues (May 7 at 10:02 PM)
### May 8, 2026
S92 Full project code review — verify frontend and backend correctness and functionality (May 8 at 9:27 AM)
### May 10, 2026
468 4:29p 🔵 Go Backend Build Passes Clean After Code Review Changes
S93 Full project code review of agent-skills-manager — verify frontend and backend correctness, fix issues found (May 10 at 4:30 PM)
S94 Full project code review of agent-skills-manager — verify correctness, fix issues, and launch the app (May 10 at 4:31 PM)
469 4:32p 🔵 Go and Wails Toolchain Managed via mise at ~/.local/share/mise
470 " 🔵 Pre-built Wails App Binary Exists in build/
S95 Code review of agent-skills-manager project — verify frontend/backend correctness and fix all issues found (May 10 at 4:33 PM)
471 4:35p 🔵 StorePage Has AI Skill Explanation Feature
472 " 🔵 Delete Operations Lack Confirmation Dialogs in ProjectsPage
473 " 🔵 api.ts Has Three-Layer Architecture: Interface, Mock, and Wails
474 " 🔵 Backend Implementations Confirmed for ExplainStoreSkill and DeleteProject
475 4:36p 🔵 DeleteProject Uses In-Memory Slice Mutation with Mutex Protection
476 " 🔵 ExplainStoreSkill Fetches README from GitHub with Graceful Fallback
477 " 🔵 StorePage AI Explain Modal Renders README as Preformatted Text with Install Action
478 " 🔵 SkillExplanationViewModel Defined in Both Frontend Types and Go Viewmodels
479 " 🔵 SkillExplanationViewModel Frontend/Backend Field Mapping
480 " 🔵 Go and TypeScript SkillExplanationViewModel Are Perfectly In Sync
481 4:37p 🔵 AI Worker Uses Local Python Process via stdin/stdout Bridge
482 " 🔵 AI Bridge Interface and SubmitGoal Uses LocalBridge for Planning
483 " 🔵 Python Worker Structure: Multi-Provider Pipeline with Tests
484 " 🔵 Python Worker Supports Four Actions: plan, resolve, report, chat
485 " 🔵 App Initializes AI Bridge from Persisted Settings on Startup
486 " 🔵 handleDeleteProject Has activeProjectId Fallback Bug After Deletion
487 4:38p 🔵 Python Chat Pipeline: sanitize_reply Strips LLM Reasoning Tags, Graceful No-Provider Fallback
488 " 🟣 Added AiExplanation Field to SkillExplanationViewModel
489 " 🟣 ExplainStoreSkill Now Generates LLM-Powered AI Explanation via Bridge
490 4:39p 🟣 AI Explanation Feature Fully Wired: Go Import, ViewModel, and TypeScript Types All Updated
S96 Implement UX improvements to ProjectsPage: delete confirmation dialog, error surfacing, reconcile button tooltip, and tighter AI explanation prompt (May 10 at 4:39 PM)
S97 整理项目代码结构并更新项目文档，按功能重新编写文档以反映当前完成情况 (May 10 at 4:48 PM)
491 4:55p ✅ Project Code Reorganization and Documentation Update Requested
492 4:59p 🔵 SQLite Migration Files Inventory
493 " 🔵 Full Project Structure and Feature Status Mapped
494 5:00p ✅ Code Reorganization and Documentation Rewrite Tasks Created
495 " 🔵 mocks.ts Is a Thin Re-export Shim for mockData.ts
496 " 🔵 Frontend lib/ File Sizes Mapped
497 " 🔵 Only One Import Site for mocks.ts — Safe to Eliminate
498 " 🔄 api.ts Import Updated from mocks.ts to mockData.ts
499 5:01p 🔄 Deleted Legacy mocks.ts Shim File
500 " 🔵 bindings_catalog.go Contains GitHub Fetching Logic That Belongs in catalog Package
501 " 🔄 Removed Stale/Misplaced Comments from bindings_catalog.go
502 " 🔵 Dangling Comment at End of bindings_skills.go
503 5:02p 🔄 Removed Orphaned Comment from End of bindings_skills.go
504 " 🔄 Frontend and Backend Code Cleanup Tasks Completed
505 5:03p 🔵 AGENTS.md Contains claude-mem Memory Context, Not Agent Documentation
506 5:04p ✅ README.md Rewritten with Current Implementation State
507 " ✅ docs/DEVELOPMENT.md Rewritten as Practical Developer Guide
508 5:05p ✅ AGENTS.md Written with Agent Support Documentation
509 " ✅ README.md Simplified to Concise Practical Format
510 " ✅ All Documentation Rewrite Tasks Completed
S98 整理项目代码结构并重写文档 — 按功能整理代码、删除冗余文件、重写 README/DEVELOPMENT/AGENTS 文档 (May 10 at 5:06 PM)
511 5:09p 🔵 Project Codebase Overview Request
512 " 🔵 Agent Skills Manager — Project Structure Identified
513 " 🔵 Agent Skills Manager — Full Architecture and Feature Set Documented
514 5:10p 🔵 Complete Wails Binding API Surface and Frontend Architecture
515 " 🔵 All Six Frontend Feature Pages — Detailed UI Behavior and Interaction Patterns
516 5:11p 🔵 Go Backend Domain Model, Services, and Infrastructure — Complete Internal Architecture
517 " 🔵 Agent Adapter Install Paths, AI Bridge IPC, and Python Worker Structure

Access 322k tokens of past work via get_observations([IDs]) or mem-search skill.
</claude-mem-context>