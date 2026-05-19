# 全面问题修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复项目审查中发现的 28 个问题，涵盖安全、并发、性能、功能缺陷、架构和代码质量。

**Architecture:** 保持 Wails + Go + React + Python Worker 整体架构不变，Python Worker 保持每次请求启动新进程模式。修复聚焦于：安全加固、并发安全、性能优化、功能修复、代码清理。

**Tech Stack:** Go/Wails, React/TypeScript/Vite, SQLite, Python worker.

---

## Phase 1: 高严重度功能缺陷修复（🔴 6 项）

### Task 1: 修复前端 handleSubmitGoal 调用错误 API

**问题 #18**: `AssistantPanel.tsx` 中 `handleSubmitGoal` 调用 `chatAssistant` 而非 `submitGoal`，导致 AI 助手技能规划流程完全不可用。

**Files:**
- Modify: `frontend/src/features/assistant/AssistantPanel.tsx`

- [ ] 在 `handleSubmitGoal` 中，将 `api.chatAssistant(trimmed, chatHistory)` 改为 `api.submitGoal(trimmed)`
- [ ] 调整返回值处理逻辑，从 `AssistantChatResponseViewModel` 适配为 `AssistantTaskViewModel`
- [ ] 提交目标后更新 `activeTask` 状态，设置 `currentStatus` 为返回的 task.status
- [ ] 保留聊天功能：新增独立的 `handleChatMessage` 函数用于纯聊天场景
- [ ] 在 UI 中区分"提交目标"和"发送聊天消息"两种交互模式
- [ ] 运行前端类型检查验证

### Task 2: 修复 BatchUpdateSkills 缺少 SourcePath

**问题 #6**: `BatchUpdateSkills` 创建 `SkillMutation` 时没有设置 `SourcePath`，导致更新操作失败。

**Files:**
- Modify: `internal/app/bindings_skills.go`

- [ ] 在 `BatchUpdateSkills` 中，为每个 `SkillMutation` 查找对应的 `SourcePath`（调用 `findSkillCachePath`）
- [ ] 如果找不到缓存路径，跳过该技能的更新并记录到返回消息中
- [ ] 运行 Go 编译验证

### Task 3: 修复 API Key 安全问题

**问题 #1**: API Key 明文传递到所有供应商环境变量，且进程列表可见。

**Files:**
- Modify: `internal/ai/local_bridge.go`

- [ ] 只设置当前选择供应商对应的环境变量，而非同时设置所有供应商的 Key
- [ ] 使用 `cmd.Env` 构建时，将 API Key 相关变量放在命令末尾，减少暴露风险
- [ ] 在环境变量中移除通用 Key 注入（`OPENAI_API_KEY` 等），只保留 `ASM_AI_API_KEY`
- [ ] Python Worker 端确认只读取 `ASM_AI_API_KEY` 和对应供应商的专用变量
- [ ] 运行 Go 编译验证

### Task 4: 优化 DiscoverAll 频繁调用性能

**问题 #3**: 几乎每个绑定方法都调用 `DiscoverAll`，导致大量重复磁盘 I/O。

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/bindings.go`
- Modify: `internal/app/bindings_skills.go`
- Modify: `internal/app/bindings_settings.go`

- [ ] 在 `App` 结构体中添加 `agentsMu sync.RWMutex` 和 `cachedInstalls []agents.AgentInstall` 字段
- [ ] 添加 `refreshAgentCache()` 方法，执行 `DiscoverAll` 并缓存结果
- [ ] 添加 `getCachedInstalls()` 方法，返回缓存的安装列表
- [ ] 在 `Startup` 中调用 `refreshAgentCache()` 初始化缓存
- [ ] 在 `InstallSkill`、`UninstallSkill`、`UpdateSkill`、`RepairAgent` 等写操作后刷新缓存
- [ ] 将所有绑定方法中的 `a.registry.DiscoverAll(context.Background())` 替换为 `a.getCachedInstalls()`
- [ ] `GetSnapshot` 等读操作直接使用缓存，不再触发磁盘扫描
- [ ] 运行 Go 编译和测试验证

### Task 5: 修复 confirmExecutablePresence 无实际效果

**问题 #5**: `confirmExecutablePresence` 查找可执行文件但不保存结果。

**Files:**
- Modify: `internal/agents/registry.go`
- Modify: `internal/agents/types.go`

- [ ] 在 `AgentInstall` 结构体中添加 `ExecutableFound bool` 字段
- [ ] 修改 `confirmExecutablePresence` 返回是否找到可执行文件
- [ ] 在 `inspectCandidate` 中调用 `confirmExecutablePresence` 并设置 `ExecutableFound`
- [ ] 当没有找到可执行文件时，将健康状态降级为 `HealthInstalledButSkillPathMissing`
- [ ] 运行 Go 测试验证

### Task 6: 修复 SubmitGoal 和 AdvanceAssistantTask 并发锁问题

**问题 #4**: `SubmitGoal` 对 `activeTask` 的读写没有全程加锁保护。

**Files:**
- Modify: `internal/app/bindings_assistant.go`

- [ ] 在 `SubmitGoal` 中，将 `a.assistantMu.Lock()/Unlock()` 包裹整个 `activeTask` 写入操作
- [ ] Bridge 调用期间不加锁（Bridge 是独立进程调用，不涉及共享状态），但在设置 `activeTask` 前加锁
- [ ] 确保 `SubmitGoal` 和 `AdvanceAssistantTask` 的锁获取顺序一致，避免死锁
- [ ] 运行 Go 编译验证

---

## Phase 2: 中严重度问题修复（🟡 14 项）

### Task 7: 修复前端类型导入路径

**问题 #7**: `api.ts` 从 `./mocks` 间接导入类型。

**Files:**
- Modify: `frontend/src/lib/api.ts`

- [ ] 将 `api.ts` 中所有类型导入从 `"./mocks"` 改为 `"./types"`
- [ ] 确认 `mockApi` 中的类型引用仍然正确
- [ ] 运行前端类型检查验证

### Task 8: 预编译 sanitizeAssistantReply 正则表达式

**问题 #8**: 每次聊天都重新编译 4 个正则表达式。

**Files:**
- Modify: `internal/app/bindings_assistant.go`

- [ ] 将 `sanitizeAssistantReply` 中的 4 个正则表达式提取为包级 `var regexp.Regexp` 变量
- [ ] 使用 `regexp.MustCompile` 在包初始化时编译
- [ ] 函数内直接使用预编译的正则对象
- [ ] 运行 Go 编译验证

### Task 9: 实现 GetLogs 功能

**问题 #9**: `GetLogs` 始终返回空数组。

**Files:**
- Modify: `internal/app/bindings_settings.go`
- Modify: `internal/platform/logging/logger.go`

- [ ] 在 `logger.go` 中添加内存日志缓冲区（环形缓冲区，容量 200 条）
- [ ] 每次日志写入时同时写入缓冲区
- [ ] 在 `App` 结构体中引用 logger 的日志缓冲区
- [ ] 实现 `GetLogs` 从缓冲区读取并按 level 过滤
- [ ] 运行 Go 编译验证

### Task 10: 修复 loadFromDatabase 忽略写入错误

**问题 #10**: 默认数据写入数据库时忽略错误。

**Files:**
- Modify: `internal/app/app.go`

- [ ] 在 `loadFromDatabase` 中，将 `_ = a.catalogSrcRepo.Put(...)` 改为检查错误并记录日志
- [ ] 同样修复 `_ = a.projectsRepo.Put(...)` 的错误处理
- [ ] 运行 Go 编译验证

### Task 11: 修复 DiscoverAll 不容错问题

**问题 #11**: 一个适配器失败导致全部失败。

**Files:**
- Modify: `internal/agents/registry.go`

- [ ] 修改 `DiscoverAll`，当某个适配器返回错误时记录日志但继续处理其他适配器
- [ ] 只有当所有适配器都失败时才返回错误
- [ ] 运行 Go 测试验证

### Task 12: 移除未使用的 httpx 依赖

**问题 #13**: `pyproject.toml` 声明了 `httpx` 但代码未使用。

**Files:**
- Modify: `pyproject.toml`

- [ ] 从 `pyproject.toml` 的 `dependencies` 中移除 `httpx>=0.27`
- [ ] 运行 `uv lock` 更新锁文件
- [ ] 运行 Python Worker 验证功能正常

### Task 13: 修复 RepairAgent 硬编码 skills 目录名

**问题 #14**: `RepairAgent` 硬编码 `skills` 目录名，未使用适配器配置。

**Files:**
- Modify: `internal/app/bindings_skills.go`

- [ ] 在 `RepairAgent` 中，通过 `a.registry.AdapterFor(agentID)` 获取适配器
- [ ] 从适配器配置中获取正确的 `SkillsRelativePath`
- [ ] 使用 `filepath.Join(install.InstallPath, skillsRelativePath)` 替代硬编码
- [ ] 运行 Go 编译验证

### Task 14: 支持自定义项目扫描路径

**问题 #15**: `scanLocalProjects` 硬编码扫描路径。

**Files:**
- Modify: `internal/app/app.go`

- [ ] 在 `BootstrapConfig` 或 `GeneralSettingsViewModel` 中添加 `ProjectScanPaths []string` 字段
- [ ] `scanLocalProjects` 优先使用用户配置的路径，未配置时使用默认路径
- [ ] 默认路径增加中文用户常见目录（如 `~/工作`、`~/代码`）
- [ ] 运行 Go 编译验证

### Task 15: 修复 Scheduler.ApplySettings 空窗期

**问题 #16**: `ApplySettings` 先 StopAll 再重建，存在任务空窗期。

**Files:**
- Modify: `internal/app/scheduler.go`

- [ ] 修改 `ApplySettings` 逻辑：先计算需要新增和需要停止的任务
- [ ] 先启动新任务，再停止旧任务（或对已存在的任务保持运行）
- [ ] 只有配置变更的任务才重启，未变更的任务保持运行
- [ ] 运行 Go 编译验证

### Task 16: 修复 go.mod 版本号

**问题 #17**: `go 1.25.0` 不存在。

**Files:**
- Modify: `go.mod`

- [ ] 将 `go 1.25.0` 改为当前实际使用的 Go 版本（如 `go 1.24.0`）
- [ ] 运行 `go mod tidy` 验证

### Task 17: 修复 SyncCatalogSource 锁范围过大

**问题 #19**: `SyncCatalogSource` 持有写锁时执行大量 I/O。

**Files:**
- Modify: `internal/app/bindings_catalog.go`

- [ ] 将 `isSkillInstalled` 调用移到锁外部，先获取需要的数据再持锁修改
- [ ] 缩小写锁范围：只在修改 `catalogSources` 和 `catalogItems` 时持锁
- [ ] GitHub API 调用和文件系统操作在锁外执行
- [ ] 运行 Go 编译验证

### Task 18: 统一脚本目录命名

**问题 #20**: 同时存在 `script/` 和 `scripts/` 两个目录。

**Files:**
- Delete: `script/build_and_run.sh`
- Modify: `scripts/build-and-run.sh`（确保内容完整）

- [ ] 确认 `scripts/build-and-run.sh` 内容完整且可用
- [ ] 删除 `script/build_and_run.sh` 旧文件
- [ ] 检查项目中是否有其他地方引用了 `script/` 目录，更新引用
- [ ] 运行构建脚本验证

### Task 19: 修复前端 AgentViewModel.status 类型

**问题 #22**: `status` 类型过于狭窄，"未安装"映射为"降级"不准确。

**Files:**
- Modify: `frontend/src/lib/types.ts`

- [ ] 将 `AgentViewModel.status` 类型从 `"healthy" | "degraded"` 扩展为 `"healthy" | "degraded" | "not_installed"`
- [ ] 更新 Go 后端 `GetAgents` 中 `HealthNotInstalled` 的映射，返回 `"not_installed"` 而非 `"degraded"`
- [ ] 运行前端类型检查验证

### Task 20: 修复 GetDashboard 任务状态硬编码

**问题 #26**: 仪表盘任务状态始终显示"0 个任务"。

**Files:**
- Modify: `internal/app/bindings.go`

- [ ] 在 `GetDashboard` 中读取 `activeTask` 的实际状态
- [ ] 根据任务状态动态生成"任务状态"高亮条的内容
- [ ] 运行 Go 编译验证

---

## Phase 3: 低严重度问题修复（🟢 8 项）

### Task 21: 修复 fronted.pen 文件名拼写

**问题 #21**: `fronted.pen` 疑似拼写错误。

**Files:**
- Rename: `fronted.pen` → `frontend.pen`（或确认用途后删除）

- [ ] 确认 `fronted.pen` 文件用途
- [ ] 如果是前端相关文件，重命名为 `frontend.pen`
- [ ] 如果是无用文件，直接删除

### Task 22: 清理 mocks.ts 间接导出

**问题 #23**: `mocks.ts` 只是重新导出，增加间接层。

**Files:**
- Modify: `frontend/src/lib/mocks.ts`
- Modify: 所有从 `./mocks` 导入类型的前端文件

- [ ] 将所有从 `"./mocks"` 导入类型的文件改为从 `"./types"` 导入类型
- [ ] `mocks.ts` 只保留 `mockApi` 和辅助函数的导出
- [ ] 运行前端类型检查验证

### Task 23: 清理 Docker 配置或添加说明

**问题 #24**: 桌面应用存在 Docker 配置，用途不明。

**Files:**
- Modify: `Dockerfile` 或 `docker-compose.yml`（添加注释说明用途）

- [ ] 在 Dockerfile 顶部添加注释说明其用途（如：仅用于 CI/开发环境）
- [ ] 如果确实无用，删除 Docker 相关文件

### Task 24: 增加 Python 测试覆盖

**问题 #25**: Python 测试只覆盖 pipeline 层。

**Files:**
- Modify: `python/tests/test_pipeline.py`
- Create: `python/tests/test_providers.py`

- [ ] 为 `http_client.py` 添加单元测试（mock HTTP 响应）
- [ ] 为各 Provider 添加请求构造测试（不实际调用 API）
- [ ] 为 `sanitize_reply` 添加边界条件测试
- [ ] 运行 Python 测试验证

### Task 25: 修复 UpdateSkill 硬编码版本号

**问题 #27**: 所有权标记版本号硬编码为 "2.0.0"。

**Files:**
- Modify: `internal/agents/registry.go`

- [ ] 在 `SkillMutation` 中添加 `Version string` 字段
- [ ] `InstallSkill` 使用 `mutation.Version`（如果为空则默认 "1.0.0"）
- [ ] `UpdateSkill` 递增现有版本号或使用 `mutation.Version`
- [ ] 运行 Go 测试验证

### Task 26: 修复 copyDir 不保留文件权限

**问题 #28**: 复制文件时使用固定权限，丢失执行权限。

**Files:**
- Modify: `internal/agents/registry.go`

- [ ] 在 `copyDir` 中，使用 `os.Stat` 获取源文件权限
- [ ] 使用 `os.WriteFile` 时传入源文件的权限位
- [ ] 运行 Go 测试验证

### Task 27: 修复 pyproject.toml 测试配置

**问题（关联 #25）**: `testpaths` 为空列表。

**Files:**
- Modify: `pyproject.toml`

- [ ] 将 `testpaths = []` 改为 `testpaths = ["tests"]`
- [ ] 运行 `uv run pytest` 验证

### Task 28: 最终验证

- [ ] 运行 `go build ./...` 验证 Go 编译
- [ ] 运行 `go test ./...` 验证 Go 测试
- [ ] 运行 `pnpm --dir frontend exec tsc --noEmit` 验证 TypeScript
- [ ] 运行 `pnpm --dir frontend test -- --run` 验证前端测试
- [ ] 运行 `uv run pytest` 验证 Python 测试
- [ ] 汇总所有修复结果
