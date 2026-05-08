# 开发规范

本文档定义本项目的开发、测试、代码组织和提交前检查规范。目标是让功能变更可验证、模块边界清晰，并避免 AI 助手、安装器和本地文件写入逻辑互相耦合。

## 环境要求

| 工具 | 要求 |
| --- | --- |
| Go | 1.23 或更高 |
| Node.js | 20 或更高 |
| pnpm | 9 或更高，当前根项目声明为 `pnpm@10.0.0` |
| Wails CLI | v2 |
| Python | 3.10 或更高，AI Worker 需要 |
| Xcode Command Line Tools | macOS 构建需要 |

## 常用命令

```bash
# 安装前端依赖
pnpm --dir frontend install

# 前端开发
pnpm --dir frontend dev

# 前端类型检查
pnpm --dir frontend exec tsc --noEmit

# 前端测试
pnpm --dir frontend test -- --run

# Go 测试，建议使用项目内缓存避免污染全局环境
GOCACHE="$PWD/.gocache" go test ./...

# 构建并验证桌面应用可启动
./script/build_and_run.sh --verify
```

## 提交前检查

涉及代码变更时至少执行：

```bash
pnpm --dir frontend exec tsc --noEmit
pnpm --dir frontend test -- --run
GOCACHE="$PWD/.gocache" go test ./...
```

涉及 Wails、构建脚本、启动流程或桌面运行时问题时，再执行：

```bash
./script/build_and_run.sh --verify
```

只修改文档时不强制运行完整构建，但应确认 Markdown 链接和命令仍然准确。

## Go 代码规范

- Wails 绑定方法放在 `internal/app/bindings_<domain>.go`，保持方法名稳定，因为前端通过生成的 JS bridge 调用。
- `internal/app` 只做门面和转换，复杂业务逻辑应放入 `internal/agents`、`internal/catalog`、`internal/installer`、`internal/projects`、`internal/reconcile`、`internal/skillgroups`、`internal/tasks` 或 `internal/ai`。
- 数据库访问必须经过 `internal/storage/sqlite` 仓库，不要在服务层直接拼 SQL。
- schema 变化必须新增 migration 文件，禁止直接修改已经发布或已使用的历史 migration。
- 错误返回应包含可行动信息，但不能泄露 API 密钥、本地敏感路径中的 token 或完整请求头。
- 新增服务行为时同步增加单元测试，优先测试服务边界而不是 UI 绑定细节。

## 前端代码规范

- 页面级代码放在 `frontend/src/features/<feature>`，跨页面组件放在 `frontend/src/components`。
- 所有后端调用统一封装在 `frontend/src/lib/api.ts`，页面不要直接访问 Wails 全局对象。
- 共享类型放在 `frontend/src/lib/types.ts`，mock 数据放在 `frontend/src/lib/mockData.ts`。
- `frontend/src/lib/mocks.ts` 当前用于兼容旧导入，迁移完成前不要删除。
- UI 状态需要覆盖加载、错误、空状态和禁用状态，特别是 AI 对话、删除历史、同步和安装操作。
- 不要在 UI 中展示模型推理过程、`<think>` 标签或供应商原始错误堆栈。
- 修改交互逻辑时补充 Vitest/Testing Library 测试，优先覆盖用户行为而不是实现细节。

## AI 助手规范

- AI 设置必须支持供应商、模型、API Key 和自定义 Base URL，并通过后端持久化。
- 前端聊天体验应像聊天机器人：发送后立即进入 loading 状态，成功追加助手回复，失败显示可重试错误。
- Worker 输出给 UI 前必须清理推理内容，包括 `<think>...</think>` 和常见 reasoning 字段。
- API Key 只用于运行时请求，不写入日志、测试快照或普通错误消息。
- 新增供应商时应同时更新：
  - `frontend/src/features/ai/aiSettings.ts`
  - 设置页 UI 和测试
  - Go 设置保存/读取绑定
  - `python/worker/providers`
  - Worker 或 bridge 测试

## 数据库和迁移规范

- migration 文件使用递增编号，例如 `005_add_xxx.sql`。
- migration 应可重复在新数据库上从 001 顺序执行。
- repository 测试应覆盖新增字段的读写、默认值和旧数据兼容。
- 删除字段或改变含义前先评估历史数据迁移，不要只改 view model。

## 测试规范

| 范围 | 推荐测试 |
| --- | --- |
| Go 服务逻辑 | `go test ./internal/<module>` |
| SQLite repository | 使用临时数据库执行迁移和读写验证 |
| 前端组件 | Vitest + Testing Library，按用户行为测试 |
| AI Worker | `python/tests` 覆盖 planner/resolver/reporter/chat |
| 构建和启动 | `./script/build_and_run.sh --verify` |

测试原则：

- 优先写能复现问题的失败测试，再修复实现。
- 测试数据放在模块内 `testdata` 或测试文件中，不依赖开发者本机真实代理目录。
- 不要提交生成缓存、截图临时文件、构建产物或本机数据库。

## 文件和 Git 规范

- 不提交 `frontend/dist`、`.gocache`、`.pytest_cache`、`test-screenshots`、`tsconfig.tsbuildinfo` 等生成文件。
- 不删除或回滚他人未提交修改，除非任务明确要求。
- 大规模重构应先更新计划文档，再分模块提交，避免混入无关格式化。
- 脚本统一放在 `scripts`，旧路径兼容入口放在 `script`。

## 新模块检查清单

新增模块或大功能时确认：

- 模块职责能用一句话说明，且不重复已有模块职责。
- 已在 `docs/ARCHITECTURE.md` 中补充模块说明。
- 已在 README 或相关文档中补充使用入口。
- 已补充测试或说明无法自动化测试的原因。
- 已执行与变更范围匹配的验证命令。
