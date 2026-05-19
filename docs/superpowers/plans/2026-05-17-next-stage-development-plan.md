# 下一阶段发展计划

> **适用对象：** Agent Skills Manager 下一阶段产品与工程实施。
> **计划日期：** 2026-05-17
> **执行方式：** 按任务逐项推进，每个任务完成后更新 checkbox 状态，并同步更新相关文档。

## 目标

将 Agent Skills Manager 从 MVP 可用状态推进到可日常使用的本地 AI 技能控制台。

下一阶段不以单纯堆叠功能为主，而是围绕三个核心价值收敛：

1. 让用户清楚知道本机有哪些 AI 代理、它们是否健康、当前安装了哪些技能。
2. 让项目能够绑定合适的代理与技能组，并自动发现、协调、修复技能差异。
3. 让 AI 助手从普通聊天升级为可确认、可执行、可验证的技能部署编排器。

## 当前基础

项目已经具备以下能力：

- 支持 Claude Code、Codex、Trae、Gemini CLI、OpenClaw、Hermes 等代理发现与技能目录管理。
- 支持远程技能商店同步、技能搜索、AI 解读和安装。
- 支持本地项目管理、代理绑定、技能组绑定和技能协调。
- 支持 AI 助手任务流，包含规划、解析、执行、验证、报告等阶段。
- 使用 Wails + Go + React + SQLite + Python Worker 构建，前后端模块边界已经基本清晰。

## 产品定位

Agent Skills Manager 的下一阶段定位是：

**面向本地 AI 工作流的技能控制台。**

它应该帮助用户回答并执行三个问题：

- 我的机器上有哪些 AI 代理可以使用？
- 某个项目应该启用哪些技能？
- 我能否一键让目标代理进入适合当前项目的工作状态？

## 阶段 1：稳定性与诊断能力 ✅

**目标：** 让应用在真实 macOS 环境中稳定、可解释、可排查。

**Files:**
- Modify: `docs/ISSUES_AND_FIXES.md`
- Modify: `docs/DEVELOPMENT.md`
- Modify: `go.mod`
- Modify: `internal/app/bindings_settings.go`
- Modify: `internal/platform/logging/logger.go`
- Add: `internal/app/errormap.go`
- Modify: `frontend/src/features/settings/SettingsPage.tsx`
- Modify: `frontend/src/lib/api.ts`
- Modify: `internal/app/bootstrap_test.go`
- Modify: `frontend/src/features/assistant/AssistantPanel.test.tsx`

- [x] 重新梳理 `docs/ISSUES_AND_FIXES.md`，将历史问题标记为已修复、仍需处理或已废弃。
- [x] 明确 Trae、Hermes 为当前新增支持代理，不再归类为范围蔓延问题。
- [x] 校准 Go、Node、Python、Wails、mise、uv、pnpm 的推荐版本与实际构建要求。
- [x] 检查 `go.mod` 中的 Go 版本是否与实际工具链一致。
- [x] 实现或完善应用日志查看能力，让设置页能展示最近运行日志。
- [x] 增加诊断导出能力，包含代理发现结果、技能目录权限、商店同步状态、最近错误。
- [x] 对关键失败场景补充用户可读错误：权限不足、目录缺失、商店同步失败、AI Key 配置错误、技能校验失败。
- [x] 跑通基础验证：`go test ./...`、前端测试、Python Worker 测试、生产构建。

## 阶段 2：项目绑定与技能协调产品化

**目标：** 让项目成为技能管理的中心，而不是单纯的技能列表附属物。

**Files:**
- Modify: `internal/projects/service.go`
- Modify: `internal/reconcile/service.go`
- Modify: `internal/skillgroups/service.go`
- Modify: `internal/app/bindings_projects.go`
- Modify: `frontend/src/features/projects/ProjectsPage.tsx`
- Modify: `frontend/src/lib/types.ts`
- Add migration if persistent fields are needed

- [ ] 支持自定义项目扫描路径，保留默认扫描目录作为兜底。
- [ ] 为项目增加语言、框架、包管理器等轻量识别能力。
- [ ] 在项目详情中展示推荐技能组和推荐理由。
- [ ] 优化协调计划展示，明确列出需安装、需卸载、需更新、需修复的技能。
- [ ] 在执行协调前展示影响范围：目标代理、目标目录、写入/删除动作。
- [ ] 协调执行后记录项目级事件历史。
- [ ] 增加常用技能组模板，例如 Go 后端、Python 工具、前端应用、文档写作、飞书工作流。
- [ ] 补充项目服务、技能组服务、协调服务的测试。

## 阶段 3：AI 助手升级为技能编排器

**目标：** 让 AI 助手可以生成结构化计划，并在用户确认后执行和验证。

**Files:**
- Modify: `frontend/src/features/assistant/AssistantPanel.tsx`
- Modify: `frontend/src/lib/api.ts`
- Modify: `internal/app/bindings_assistant.go`
- Modify: `internal/ai/local_bridge.go`
- Modify: `python/worker/pipeline/planner.py`
- Modify: `python/worker/pipeline/resolver.py`
- Modify: `python/worker/pipeline/reporter.py`
- Modify: `python/tests/test_pipeline.py`

- [ ] 在 UI 上区分普通聊天和技能部署任务。
- [ ] 让 `SubmitGoal` 返回更稳定的结构化计划，包括目标代理、候选技能、风险提示、执行步骤。
- [ ] 增加用户确认步骤，执行前必须展示将要修改的代理和技能目录。
- [ ] AI 解析阶段优先使用本地商店、已安装技能、项目上下文，而不是只依赖模型自由生成。
- [ ] 执行阶段记录每一步结果，失败时允许重试或跳过。
- [ ] 验证阶段检查 `SKILL.md`、目录结构、代理可读性和安装标记。
- [ ] 报告阶段生成简洁摘要：完成了什么、失败了什么、下一步建议是什么。
- [ ] 继续确保模型输出不会展示 reasoning、thinking、analysis 等内部推理内容。

## 阶段 4：技能商店与生态能力

**目标：** 让技能来源、技能详情和兼容性判断更加可信。

**Files:**
- Modify: `internal/catalog/client.go`
- Modify: `internal/catalog/cache.go`
- Modify: `internal/catalog/compat.go`
- Modify: `internal/installer/validator.go`
- Modify: `frontend/src/features/store/StorePage.tsx`
- Modify: `frontend/src/features/skills/SkillsPage.tsx`
- Modify: `frontend/src/lib/types.ts`

- [ ] 支持在设置或商店页管理多个技能来源：新增、禁用、刷新、删除。
- [ ] 为技能详情增加 README、来源仓库、更新时间、兼容代理、安装状态。
- [ ] 增强兼容性判断，区分明确支持、可能支持、不支持、未知。
- [ ] 支持导入本地技能目录，并在导入前执行结构校验。
- [ ] 对商店同步增加缓存状态展示：上次同步时间、同步结果、失败原因。
- [ ] 对安装来源增加追踪信息，方便后续判断技能是否可更新。
- [ ] 为新增商店来源和本地导入补充测试。

## 阶段 5：发布与日常使用体验

**目标：** 让应用可以被稳定分发、升级和维护。

**Files:**
- Modify: `scripts/build.sh`
- Modify: `scripts/build-and-run.sh`
- Modify: `README.md`
- Modify: `README.en.md`
- Modify: `docs/DEVELOPMENT.md`
- Add release notes as needed

- [ ] 固化 macOS 构建流程，明确 app bundle 输出、签名和验证步骤。
- [ ] 增加版本号和构建信息展示。
- [ ] 编写 release checklist，包含测试、构建、运行、签名、安装验证。
- [ ] 规划自动更新机制，短期先支持检查新版本和打开发布页。
- [ ] 增加首次启动引导：检测代理、配置 AI、同步商店、选择项目。
- [ ] 优化空状态和加载状态，让用户知道下一步可以做什么。
- [ ] 保持中英文 README 与功能现状同步。

## 优先级建议

近期建议优先推进：

1. 阶段 1：稳定性与诊断能力。
2. 阶段 2：项目绑定与技能协调产品化。
3. 阶段 3：AI 助手结构化计划与确认执行。

阶段 4 和阶段 5 可以并行做小步改进，但不建议在稳定性完成前大规模扩展商店能力。

## 验收标准

下一阶段完成后，应满足以下标准：

- 用户可以清楚看到每个代理的健康状态、技能目录和已安装技能。
- 用户可以为项目选择代理和技能组，并看懂协调计划。
- 用户确认后，应用能执行技能安装、更新、修复，并展示结果。
- AI 助手生成的计划可读、可确认、可执行、可验证。
- 常见失败场景有明确错误提示和诊断信息。
- 项目可以稳定完成测试与生产构建。

## 风险与注意事项

- 不要把 AI 助手变成不可控的自动执行器；涉及文件写入和删除的操作必须让用户确认。
- API Key、技能目录路径、日志信息不能泄露敏感内容。
- 新增代理支持时必须同步适配器、前端类型、文档和测试。
- 数据库 schema 变更必须新增 migration，不直接修改历史 migration。
- 技能商店来源可能不稳定，必须保留缓存和失败兜底。
- 前端状态只保存临时交互信息，持久状态应进入 SQLite 或后端配置。
