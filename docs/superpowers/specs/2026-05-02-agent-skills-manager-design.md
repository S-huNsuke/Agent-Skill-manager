# Agent Skills Manager Design

## Overview

Agent Skills Manager is a macOS desktop app for ordinary users to discover, install, organize, and maintain skills across multiple local AI agent tools from one place. `v1` targets `Codex`, `Claude Code`, `Gemini CLI`, and `OpenClaw`.

The product combines four capabilities into one workflow:

1. Local agent and skill discovery
2. Standardized skill store sync, install, update, and uninstall
3. Project-specific skill groups
4. Cloud-backed AI automation for recommendation, application, and repair

Users should not need to understand per-agent directory layouts, package formats, or manual installation paths.

## Product Goal

Build a macOS desktop app that:

- discovers supported local agents automatically
- scans installed skills and normalizes them into one management view
- installs skills from standardized catalog URLs and built-in stores
- lets users create skill groups and assign exactly one skill group to each project
- uses a cloud-backed AI agent to recommend, install, repair, and maintain skills automatically for each project

## Scope

### In Scope for v1

- macOS desktop application
- ordinary-user-first UX
- support for `Codex`, `Claude Code`, `Gemini CLI`, `OpenClaw`
- standardized skill store catalog URL support
- built-in stores plus user-added catalog sources
- unified skill package distribution with `zip` or `tar.gz`
- one normalized skill entry that can map to multiple agent installation targets
- project support for both local path-backed projects and virtual projects
- one skill group per project
- logical grouping priority instead of destructive environment switching
- full-automatic AI mode backed by cloud models

### Out of Scope for v1

- Windows or Linux support
- account system and cloud sync
- ratings, reviews, or social store features
- arbitrary webpage scraping as store input
- uninstall-on-project-switch behavior
- multiple skill groups per project
- local model inference

## User Model

The primary user is a non-expert who wants useful skills available in the right project context without learning each agent platform's internals.

UX implications:

- minimize exposure of filesystem paths
- use result-oriented language instead of implementation jargon
- expose advanced diagnostics only when needed
- prefer automation by default, but keep actions visible

## Technical Stack

### Desktop App

- `Wails`
- `React + TypeScript + Vite`
- `shadcn/ui + Tailwind CSS`

### Core Runtime

- `Go`
- `SQLite`

### AI Layer

- `Python` worker
- cloud-hosted LLM providers behind a provider abstraction

### Packaging and Storage

- package download, verification, extraction, and installation handled by the `Go` core
- package formats: `zip`, `tar.gz`
- local cache for package artifacts and synced catalogs

## Architecture

The system is composed of six major parts.

### 1. Desktop UI

Responsible for:

- navigation
- status display
- store browsing
- project and skill group management
- AI task display
- diagnostics and settings

### 2. Go App Core

Responsible for:

- app lifecycle
- backend command bridge
- task scheduling
- persistence
- logging
- file operations

### 3. Agent Adapter Layer

Each supported agent gets its own adapter:

- `CodexAdapter`
- `ClaudeCodeAdapter`
- `GeminiCliAdapter`
- `OpenClawAdapter`

Each adapter exposes the same normalized interface:

- `Discover()`
- `ListInstalledSkills()`
- `InstallSkill()`
- `UninstallSkill()`
- `UpdateSkill()`
- `ValidateSkillInstall()`

#### Adapter Discovery Contract

`Discover()` must not be a generic "search somewhere" function. Each adapter must define:

- candidate install path sources
  - known default paths
  - user-configured override paths
  - executable lookup results when applicable
- validation rules proving the agent is real
  - executable or app existence
  - expected config directory shape
  - readable skill root
- degraded states
  - `not_installed`
  - `installed_but_unreadable`
  - `installed_but_skill_path_missing`
  - `installed_but_skill_path_empty`

#### Adapter Normalization Contract

The four agents are assumed to have different storage formats. Each adapter must translate its native layout into a normalized internal shape:

- native skill path
- native manifest or metadata format
- installable version string if present
- enabled or disabled state if supported
- validation errors when metadata is missing or malformed

#### Adapter State

Each adapter must maintain:

- last discovered install path
- last discovered skills path
- last scan timestamp
- health status
- last error code
- last error message

### 4. Catalog and Package Layer

Responsible for:

- syncing catalog sources
- parsing store manifests
- downloading packages
- checksum verification
- unpacking and staging files
- agent-specific install mapping

### 5. AI Agent Layer

The AI worker runs in Python and is responsible for:

- goal understanding
- recommendation
- skill group generation
- execution planning
- repair and retry orchestration
- summary generation

### 6. Policy and Safety Layer

Although the user-facing behavior is full automatic by default, the system still enforces internal guardrails:

- no blind deletion of working user-installed skills
- no uncontrolled mutation of agent core configuration
- bounded retries
- complete task and action logging

## Main Product Surfaces

### Home

Shows:

- discovered agents count
- installed skills count
- update count
- AI assistant entry
- recommended actions
- recent activity
- system health

### My Skills

Unified management page showing:

- installed skills
- updatable skills
- failed installs
- disabled items

Each item shows:

- name
- source
- version
- installed agents
- status

### Agents

Each agent card shows:

- connected or missing status
- skill directory health
- installed count
- diagnostics summary

### Store

Supports:

- browsing built-in stores
- adding catalog URLs
- enabling or disabling sources
- searching skills
- viewing skill details
- installing and updating skills

### AI Assistant

Task-driven entry point where the user describes a goal and the system:

- recommends skills
- installs them automatically
- creates or updates a skill group
- validates results
- reports the outcome in plain language

### Settings and Diagnostics

Includes:

- model and API configuration
- store source management
- automation settings
- logs and environment diagnostics

## Core Domain Model

### Agent

Represents a detected local agent installation.

Suggested fields:

- `id`
- `kind`
- `name`
- `status`
- `install_path`
- `skills_path`
- `last_seen_at`
- `last_error_code`
- `last_error_message`

### Skill

Represents the normalized internal concept of a skill independent of a specific agent.

Suggested fields:

- `id`
- `name`
- `description`
- `tags`

### CatalogSource

Represents a configured store source.

Suggested fields:

- `id`
- `name`
- `url`
- `is_builtin`
- `enabled`
- `last_synced_at`
- `last_sync_status`
- `last_sync_error`
- `cache_expires_at`
- `min_supported_client_version`

### CatalogSkill

Represents a store listing for a normalized skill.

Suggested fields:

- `id`
- `source_id`
- `name`
- `version`
- `author`
- `description`
- `homepage`
- `package_url`
- `checksum_sha256`
- `supported_agents`
- `schema_version`

### InstalledSkill

Represents a concrete installation state of a skill for an agent.

Suggested fields:

- `id`
- `skill_id`
- `agent_id`
- `version`
- `install_state`
- `install_path`
- `source_id`
- `last_checked_at`
- `error_message`
- `conflict_group`
- `is_managed_by_app`

### SkillGroup

Represents a named set of desired skills for a project context.

Suggested fields:

- `id`
- `name`
- `description`
- `source_type` where values are `manual` or `ai-generated`
- `version`
- `goal_prompt`
- `preferred_agents`
- `created_at`
- `updated_at`

### SkillGroupSkill

Join model between a skill group and its desired skills.

Suggested fields:

- `skill_group_id`
- `skill_id`
- `required`
- `priority`
- `reason`

### Project

Represents either a local folder-backed project or a virtual project.

Suggested fields:

- `id`
- `name`
- `type` where values are `path` or `virtual`
- `path`
- `description`
- `skill_group_id`
- `auto_apply_enabled`
- `last_active_at`
- `created_at`
- `updated_at`

### Task

Represents an AI or system task with explicit lifecycle state.

Suggested fields:

- `id`
- `task_type`
- `trigger_source`
- `project_id`
- `skill_group_id`
- `status`
- `status_reason`
- `plan_json`
- `action_log`
- `result_summary`
- `retry_count`
- `started_at`
- `finished_at`

## Project and Skill Group Rules

- a project may be path-backed or virtual
- a project binds exactly one skill group
- a skill group may be reused by multiple projects
- a skill group may be created manually or generated by AI
- a skill group describes desired capability, not guaranteed install state
- installed skill records describe actual machine state

This keeps intent and execution separate.

## Store Catalog Protocol

Catalogs use a standardized JSON schema.

Top-level structure:

```json
{
  "schema_version": "1.0",
  "store_name": "Example Skill Store",
  "store_url": "https://example.com",
  "min_client_version": "1.0.0",
  "skills": []
}
```

Each skill listing should include metadata and a multi-agent install mapping.

```json
{
  "id": "web-scrape-summary",
  "name": "Web Scrape Summary",
  "version": "1.2.0",
  "author": "Example Author",
  "description": "Scrape pages and summarize results",
  "homepage": "https://example.com/skills/web-scrape-summary",
  "package_url": "https://example.com/packages/web-scrape-summary-1.2.0.zip",
  "checksum": {
    "sha256": "..."
  },
  "tags": ["web", "scraping", "summary"],
  "supported_agents": ["codex", "claude-code", "gemini-cli", "openclaw"],
  "install_mappings": {
    "codex": {
      "skill_path": "dist/codex/skill",
      "manifest_path": "dist/codex/manifest.json"
    },
    "claude-code": {
      "skill_path": "dist/claude/skill",
      "manifest_path": "dist/claude/manifest.json"
    }
  }
}
```

### Protocol Compatibility Rules

- clients must accept compatible minor schema additions without failing hard
- a catalog with an unsupported major schema version must be marked incompatible
- `min_client_version` lets the app show an upgrade prompt before installation is attempted
- when a source is incompatible, the app continues using the last valid cached catalog if one exists

## Automatic Discovery

The application refreshes state from four trigger paths.

### On App Startup

- discover local agents
- scan installed skills
- refresh normalized local state

### After Catalog Sync

- detect available updates
- compare installed skills with store listings
- identify new candidates relevant to existing projects and skill groups

### On Project Activation

Project activation may come from:

- manual project selection inside the app
- automatic recognition of a registered active local project path

After activation, the system checks the bound skill group and reconciles missing requirements.

### Scheduled Health Scans

Background checks detect:

- missing agents
- corrupted skill installations
- externally changed skill folders
- stale versions

## Project Activation and Matching

The system supports both manual and automatic project switching.

Rules:

- if an active local directory exactly matches a registered project path, activate that project
- if multiple candidate paths match, pick the longest exact-prefix path first
- if two candidates still conflict at the same specificity, do not auto-switch and require confirmation
- virtual projects can only be activated manually

## Skill Group Application Model

The product uses logical grouping first rather than destructive environment switching.

When a project becomes active, the system should:

1. read the bound skill group
2. check whether each required skill is present
3. install missing skills
4. update stale skills
5. repair damaged installs

The system should not proactively uninstall unrelated working skills just because they are outside the current project group.

## Error Handling and Boundary Cases

The spec must define degraded behavior, not just the happy path.

### Offline Behavior

- first-ever catalog sync requires network access
- if the network is unavailable after at least one successful sync, the app may continue using cached catalogs until `cache_expires_at`
- if no valid cache exists and the network is unavailable, store installs are blocked but local management remains available

### Low Disk Space

- before downloading or extracting, the installer must estimate required temporary and final disk usage
- if space is insufficient, the task fails early with a user-visible reason and no partial install is committed

### Broken or Empty Agent Paths

- if an agent install path exists but is unreadable, mark the agent degraded and do not run write operations
- if the skill path is missing or empty, distinguish between `empty_valid_root` and `broken_root`
- the UI should offer repair guidance, not silently hide the agent

### Multi-Version or Duplicate Skill Conflicts

- only one managed version of the same normalized skill may be active per agent at a time
- if multiple versions are discovered outside app control, mark the install as conflicted
- automatic repair may deactivate or quarantine only app-managed duplicates
- user-managed duplicates require explicit confirmation before destructive cleanup

## Update Strategy

The installer should prefer incremental operations where safe.

- metadata-only refreshes should not trigger package reinstall
- reinstall only when version changes, checksum mismatches, or validation fails
- use full reinstall when a package does not support safe in-place update
- use repair reinstall when files are missing or corrupted

## AI Agent Capability

The AI assistant is agentic rather than advisory-only.

Primary behaviors:

- recommend skills from user goals
- create skill groups automatically
- install and configure skills automatically
- validate installations
- repair failed or corrupted project environments

Default execution mode:

- full automatic
- cloud-model-backed
- visible task progress in the UI

## AI Task Types

Initial task classes:

- `RecommendSkills`
- `CreateSkillGroup`
- `ApplySkillGroup`
- `RepairEnvironment`
- `OptimizeProjectSetup`

## AI Execution State Machine

The AI pipeline must use explicit lifecycle states to avoid ambiguous looping.

Task states:

- `queued`
- `planning`
- `resolving`
- `executing`
- `verifying`
- `recovering`
- `completed`
- `failed`
- `blocked`
- `cancelled`

Allowed transitions:

- `queued -> planning`
- `planning -> resolving`
- `resolving -> executing`
- `executing -> verifying`
- `verifying -> completed`
- `verifying -> recovering`
- `recovering -> resolving`
- `recovering -> failed`
- any active state -> `blocked` for user or system intervention
- any active state -> `cancelled`

State machine rules:

- only one recovery loop is allowed per action bundle by default
- repeated recovery requires a new plan revision and increments `retry_count`
- tasks must stop at `blocked` instead of looping indefinitely when prerequisites cannot be satisfied

## AI Execution Pipeline

Recommended internal stages:

1. `Planner`
2. `Resolver`
3. `Executor`
4. `Verifier`
5. `Recovery`
6. `Reporter`

The Python worker delegates local filesystem and installation operations to the Go core instead of editing the environment directly.

## AI Task Queue

All AI actions should be recorded as tasks so progress, retries, and diagnostics are visible.

## Safety and Recovery Rules

Safety needs both prohibitions and recovery definitions.

### Reversible vs Irreversible Operations

Reversible operations:

- download to cache
- extract to staging area
- validate without mutation
- enable or disable app-managed metadata markers

Potentially irreversible operations:

- deleting installed skill files
- overwriting existing non-app-managed files
- rewriting agent configuration files

Irreversible or destructive actions require one of:

- a prior app-managed ownership marker proving the app created the files
- an explicit user confirmation

### Recovery Preconditions

Automatic recovery is allowed only when:

- the failing installation target is app-managed or safely replaceable
- a previous valid package or cached artifact exists
- the adapter can prove the target path belongs to the expected agent

If these are not true, the task must move to `blocked`.

### Operation Log Retention

- keep task summaries indefinitely in SQLite unless the user clears history
- keep structured action logs for at least the most recent 30 days
- keep local package cache by policy, with removable old artifacts

## MVP Definition

The MVP should deliver one usable end-to-end loop:

- install and run the macOS app
- detect supported agents
- show installed skills in one UI
- browse built-in and user-added stores
- install, update, and uninstall skills
- create projects
- create skill groups
- bind one skill group to one project
- auto-apply the group logically by filling gaps and repairing state
- let the AI assistant recommend skills, generate groups, and perform automatic setup

## Milestones

### Milestone 1: Local Management Foundation

- Wails shell
- base React UI
- SQLite persistence
- agent discovery
- installed skill scanning
- base pages for skills, agents, and settings

### Milestone 2: Store and Installer

- catalog protocol implementation
- built-in store configuration
- package download, checksum verification, extraction, install, uninstall, update
- store browsing and skill details
- install logs and diagnostics

### Milestone 3: Projects and Skill Groups

- project creation
- skill group creation
- project-to-group binding
- manual project activation
- automatic skill group reconciliation

### Milestone 4: AI Agent

- natural-language goal input
- skill recommendation
- AI-generated skill groups
- automatic apply and repair
- task queue, execution log, and final explanation

## Design Principles

- optimize for ordinary users rather than power users
- hide local complexity, surface useful outcomes
- separate desired capability from actual install state
- keep project behavior logically grouped instead of destructively switching environments
- make automation visible even when automatic
- keep store distribution standardized and predictable

## Summary

This product acts as a project-aware skills operating layer for multiple local AI agents on macOS. It discovers existing environments, installs and maintains skills from standardized stores, organizes desired capabilities through skill groups, and uses an AI agent to keep each project supplied with the right skills automatically while remaining observable and recoverable.
