# 开发指南

## 环境要求

| 工具 | 最低版本 | 验证版本 | 用途 |
|------|---------|---------|------|
| Go | 1.26+ | 1.26.3 | 后端编译 |
| Node.js | 22+ | 26.0.0 | 前端构建 |
| pnpm | 10+ | 10.0.0 | 前端包管理 |
| Python | 3.12+ | 3.14.5 | AI 工作进程 |
| Wails CLI | v2.12+ | v2.12.0 | 桌面应用框架 |
| mise | 2024+ | 2026.5.8 | Go/Node 版本管理 |
| uv | 0.5+ | 0.11.14 | Python 依赖管理 |

安装 Wails CLI：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 启动开发环境

```bash
# 安装前端依赖
pnpm --dir frontend install

# 安装 Python 依赖
uv pip install -e .

# 启动开发模式（前后端热重载）
wails dev
```

## 生产构建

```bash
bash scripts/build.sh
# 输出：build/bin/agent-skills-manager.app
```

构建脚本依次执行：
1. 前端：`tsc -b && vite build`
2. 后端：`go build`
3. 打包：组装 `.app` bundle
4. 签名：`codesign --deep --force --sign -`

## 目录结构

```
├── cmd/agent-skills-manager/   # main.go 入口
├── internal/
│   ├── agents/                 # 代理适配器（Claude Code、Codex、Gemini CLI 等）
│   ├── ai/                     # Python 工作进程桥接
│   ├── app/                    # Wails 绑定层和视图模型
│   ├── catalog/                # 远程目录同步
│   ├── domain/                 # 领域类型
│   ├── installer/              # 技能下载、解压、安装
│   ├── platform/               # 日志、文件系统工具
│   ├── projects/               # 项目扫描与激活
│   ├── reconcile/              # 技能协调计划生成
│   ├── skillgroups/            # 技能组管理
│   ├── storage/sqlite/         # SQLite 持久化
│   └── tasks/                  # 任务历史管理
├── frontend/src/
│   ├── components/             # 通用 UI 组件
│   ├── features/               # 功能页面
│   │   ├── agents/             # 代理管理
│   │   ├── ai/                 # AI 设置常量
│   │   ├── assistant/          # AI 助手面板
│   │   ├── home/               # 首页仪表盘
│   │   ├── projects/           # 项目管理
│   │   ├── settings/           # 设置页
│   │   ├── skills/             # 技能管理
│   │   └── store/              # 技能商店
│   ├── lib/
│   │   ├── api.ts              # Wails 调用封装（生产）和 mock（开发）
│   │   ├── types.ts            # 共享 TypeScript 类型
│   │   ├── mockData.ts         # 开发用 mock 数据
│   │   ├── status.ts           # 状态工具函数
│   │   └── utils.ts            # 通用工具函数
│   └── routes/index.tsx        # 路由与操作分发
├── python/worker/
│   ├── main.py                 # 工作进程入口（stdin/stdout JSON）
│   ├── pipeline/               # 四阶段流水线（plan/resolve/report/chat）
│   └── providers/              # LLM 提供商适配器
└── scripts/
    ├── build.sh                # 生产构建
    └── build-and-run.sh        # 构建并启动
```

## 架构说明

### 前后端通信

Wails 通过 Go 方法绑定实现前后端通信。前端调用统一经过 `frontend/src/lib/api.ts` 封装：

```typescript
// 生产环境：调用 Go 方法
wailsCall<T>("MethodName", ...args)

// 开发/测试：使用 mock 数据
mockApi.methodName(args)
```

`api.ts` 根据 `window.__WAILS__` 是否存在自动切换模式。

### Go 绑定层

`internal/app/` 是 Wails 绑定层，按功能拆分：

| 文件 | 职责 |
|------|------|
| `app.go` | App 结构体定义、初始化、生命周期 |
| `bindings.go` | 快照（`GetSnapshot`）、仪表盘、诊断 |
| `bindings_skills.go` | 技能查询、安装、卸载、更新、修复 |
| `bindings_catalog.go` | 商店目录同步、AI 技能解释 |
| `bindings_projects.go` | 项目 CRUD、代理/技能组绑定、协调 |
| `bindings_settings.go` | 设置读写、最近活动 |
| `bindings_assistant.go` | AI 助手任务管理 |
| `bootstrap.go` | 启动时数据加载 |
| `converters.go` | 领域类型 ↔ 视图模型转换 |
| `viewmodels.go` | 所有视图模型类型定义 |
| `scheduler.go` | 定时任务（目录同步、健康检查） |

### Python AI 工作进程

Go 通过 `internal/ai/local_bridge.go` 启动 Python 子进程，通过 stdin/stdout 传递 JSON 消息：

**请求格式：**
```json
{"action": "chat", "payload": {"message": "...", "history": []}}
```

**响应格式：**
```json
{"status": "ok", "data": {"reply": "..."}}
```

支持的 action：`plan`、`resolve`、`report`、`chat`

### 数据库

SQLite 数据库位于 `~/Library/Application Support/agent-skills-manager/data.db`，包含 4 个迁移：

| 迁移 | 内容 |
|------|------|
| `001_initial.sql` | projects、skill_groups、catalog_sources 表 |
| `002_settings.sql` | settings 表 |
| `003_project_extensions.sql` | 项目扩展字段 |
| `004_tasks_nullable_project.sql` | tasks 表，project_id 可为空 |

## 运行测试

```bash
# Go 测试
go test ./...

# 前端测试
pnpm --dir frontend test

# Python 测试
uv run python -m pytest python/tests/
```

## 代码规范

**Go**
- 错误处理：所有错误必须处理或显式忽略（`_ = err`）
- 并发：共享状态使用 `sync.RWMutex` 保护
- 视图模型：领域类型不直接暴露给前端，通过 `converters.go` 转换

**TypeScript**
- 所有后端调用通过 `api.ts` 的接口，不直接调用 `window.go.*`
- 组件 props 必须有明确的 TypeScript 类型
- 异步操作必须有错误处理，不允许静默吞掉错误

**Python**
- 所有 LLM 调用通过 `providers/base.py` 的抽象接口
- 工作进程通过环境变量读取配置，不硬编码 API Key

## 常见问题

**构建时找不到 Go**

使用 mise 管理 Go 版本时，需要激活环境：
```bash
eval "$(mise activate bash)"
```

或直接指定路径：
```bash
export PATH="$HOME/.local/share/mise/installs/go/latest/bin:$PATH"
```

**前端热重载不生效**

Wails dev 模式下前端由 Vite 提供，修改 `frontend/src/` 下的文件会自动热重载。修改 Go 文件需要等待 Go 重新编译。

**AI 助手无响应**

检查设置页中的 AI 提供商配置是否正确（API Key、模型名称）。查看日志：
```bash
~/Library/Logs/agent-skills-manager/app.log
```
