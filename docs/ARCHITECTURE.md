# 项目架构说明

本文档描述 Agent Skills Manager 当前代码结构、模块职责和主要边界。它用于新功能开发、问题排查和后续重构时统一判断代码应该放在哪里。

## 总体运行链路

```text
React UI
  -> frontend/src/lib/api.ts
  -> Wails JavaScript bridge
  -> internal/app App 绑定方法
  -> 领域服务 / 仓库 / 适配器
  -> SQLite / 本地文件系统 / Python AI Worker
```

前端只负责界面状态、用户交互和调用 Wails 绑定。Go 后端负责本地状态、技能安装、代理适配、任务编排和持久化。AI 助手由 Go 通过 `internal/ai` 启动 Python Worker 完成规划、解析、对话和报告生成。

## 顶层目录

| 路径 | 作用 |
| --- | --- |
| `cmd/agent-skills-manager` | 桌面应用入口，负责启动 Wails App。 |
| `internal` | Go 后端核心代码，包含领域服务、存储、适配器和 Wails 绑定。 |
| `frontend` | React/TypeScript 前端应用。 |
| `python/worker` | AI 助手 Worker，封装 LLM 供应商和任务流水线。 |
| `scripts` | 当前推荐的构建、运行和验证脚本。 |
| `script` | 兼容旧命令的包装脚本，新脚本优先放在 `scripts`。 |
| `docs` | 项目文档、历史计划、问题记录和图片资源。 |
| `build` | Wails 构建资源和产物目录。构建产物不应作为源码依赖。 |

## Go 后端模块

| 路径 | 职责 | 主要规则 |
| --- | --- | --- |
| `internal/app` | Wails App 门面、生命周期、前端绑定、视图模型转换、调度器。 | 绑定方法保持薄层，只做参数校验、调用服务、转换结果。不要把业务流程长期堆在绑定方法里。 |
| `internal/agents` | 代理注册表、代理接口、本地技能目录读写。 | 新代理必须实现统一 Adapter 接口，并通过注册表暴露能力。 |
| `internal/agents/<agent>` | Claude Code、Codex、Gemini CLI、OpenClaw 等具体适配器。 | 适配器只处理该代理的路径、安装格式和健康检查差异。 |
| `internal/catalog` | 远程技能目录同步、缓存、兼容性过滤和目录类型。 | 远程数据必须经过兼容性和缓存层后再进入安装流程。 |
| `internal/installer` | 技能下载、解压、校验和安装服务。 | 安装前必须执行 preflight/validator，避免写入不完整技能。 |
| `internal/projects` | 本地项目扫描、校验、激活和项目相关设置。 | 项目路径需要标准化和校验，不应由前端直接信任。 |
| `internal/skillgroups` | 技能组定义、校验和期望技能集合。 | 技能组只描述目标状态，不直接执行安装。 |
| `internal/reconcile` | 根据当前状态和目标状态生成安装、更新、修复计划。 | 协调层产出计划，具体写入仍交给 installer/agents/storage。 |
| `internal/tasks` | AI 任务历史、任务状态恢复和任务服务。 | 任务状态必须可恢复，避免 UI 关闭后丢失执行结果。 |
| `internal/storage/sqlite` | SQLite 连接、迁移和仓库实现。 | 所有持久化访问通过仓库完成； schema 变更必须新增 migration。 |
| `internal/domain` | 跨模块共享的领域类型。 | 保持稳定、纯数据结构，避免引入 UI 或存储细节。 |
| `internal/ai` | Go 到 Python Worker 的桥接。 | 只传递结构化 payload；不要把 UI 文案和供应商细节散落到 app 层。 |
| `internal/platform` | 文件系统、日志、错误等基础设施封装。 | 平台差异放在这里，业务模块不要直接复制平台判断。 |

## 前端模块

| 路径 | 职责 | 主要规则 |
| --- | --- | --- |
| `frontend/src/App.tsx` | 应用级布局、全局状态和路由挂载。 | 不放具体业务实现，页面逻辑下沉到 feature。 |
| `frontend/src/routes` | React Router 配置。 | 路由入口保持清晰，页面组件从 features 引入。 |
| `frontend/src/features` | 按功能划分页面和局部组件。 | 新页面优先新建 feature 目录，避免把页面逻辑塞进共享组件。 |
| `frontend/src/features/assistant` | AI 助手聊天面板、历史对话、最近任务交互。 | 对话输出必须过滤模型推理内容；交互状态要覆盖加载、失败、空状态。 |
| `frontend/src/features/ai` | AI 设置相关前端辅助定义。 | 供应商、模型默认值和设置 UI 的常量优先集中在这里。 |
| `frontend/src/features/settings` | 设置页，包含 AI 配置、主题、语言等。 | 设置保存后必须同步到后端持久化配置。 |
| `frontend/src/components` | 跨页面复用 UI 组件。 | 组件应保持无业务或低业务耦合。 |
| `frontend/src/lib/api.ts` | Wails 绑定调用封装和浏览器 mock fallback。 | 前端页面不得直接拼 Wails 方法名，统一经过 API 层。 |
| `frontend/src/lib/types.ts` | 前端共享类型。 | 与 Go view model 对齐，字段变更需要同步测试。 |
| `frontend/src/lib/mockData.ts` | 浏览器开发和测试用 mock 数据。 | mock 只用于非 Wails 环境，不应成为生产逻辑依赖。 |
| `frontend/src/lib/mocks.ts` | 兼容旧导入的 re-export。 | 待旧引用完全迁移后再删除。 |
| `frontend/src/test` | Vitest/Testing Library 测试环境配置。 | 新交互功能应补充组件或 API 层测试。 |

## Python AI Worker

| 路径 | 职责 | 主要规则 |
| --- | --- | --- |
| `python/worker/main.py` | Worker 入口，读取 Go 传入请求并输出结构化结果。 | 标准输入输出协议要保持向后兼容。 |
| `python/worker/pipeline/planner.py` | 根据用户目标生成技能部署计划。 | 输出应是结构化计划，不能依赖长篇自然语言解析。 |
| `python/worker/pipeline/resolver.py` | 解析计划中的技能、项目和代理需求。 | 失败要返回可展示错误，不要静默吞掉。 |
| `python/worker/pipeline/reporter.py` | 汇总执行结果和报告。 | 面向 UI 的报告应去除模型推理内容。 |
| `python/worker/pipeline/chat.py` | 聊天机器人式对话回复。 | 回复只输出用户可读答案，不输出 chain-of-thought 或 `<think>` 内容。 |
| `python/worker/providers` | LLM 供应商适配层。 | 新供应商实现统一 Provider 接口，密钥和 Base URL 从设置传入。 |
| `python/tests` | Worker 流水线测试。 | 修改提示词、解析协议或供应商适配时需要补测试。 |

## 数据和配置

SQLite 是本地持久化核心。迁移文件位于 `internal/storage/sqlite/migrations`，仓库实现位于 `internal/storage/sqlite/repos.go`。

配置和状态原则：

- 用户设置、AI 配置、任务历史等需要持久化的数据应进入 SQLite 或明确的配置存储。
- 前端本地 state 只保存临时 UI 状态，例如输入框内容、折叠状态、选中项。
- API 密钥不应出现在日志、测试快照或错误详情中。
- 数据库 schema 变更必须新增顺序 migration，不直接改历史 migration。

## 模块边界规范

- `internal/app` 是 UI-facing facade，不是业务逻辑仓库。
- 前端调用后端只通过 `frontend/src/lib/api.ts`，不要在页面里直接访问 `window.go...`。
- 服务层不依赖 React/Wails 类型；跨边界使用 view model 或 domain type 转换。
- 文件系统写操作集中在代理适配器、安装器和平台封装中。
- AI Worker 只接收结构化上下文，不读取前端状态，也不直接修改数据库。
- 新增供应商时同时更新前端设置、Go 设置保存、Python provider、测试用例和文档。

## 新功能放置建议

| 场景 | 推荐位置 |
| --- | --- |
| 新增 Wails 调用 | `internal/app/bindings_<domain>.go` + `frontend/src/lib/api.ts` |
| 新增页面 | `frontend/src/features/<name>` + `frontend/src/routes` |
| 新增共享 UI | `frontend/src/components` |
| 新增数据库字段 | `internal/storage/sqlite/migrations` + repository + tests |
| 新增代理 | `internal/agents/<agent>` + registry tests |
| 新增技能安装规则 | `internal/installer` 或 `internal/reconcile` |
| 新增 AI 供应商 | `python/worker/providers` + AI 设置 UI + bridge tests |
| 新增构建流程 | `scripts`，必要时在 `script` 下保留兼容包装 |
