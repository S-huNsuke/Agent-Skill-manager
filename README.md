# Agent Skills Manager

一个用于管理 AI 代理技能的 macOS 桌面应用。支持 Claude Code、Codex、Gemini CLI 等主流 AI 代理，提供技能安装、项目绑定、AI 辅助规划等功能。

## 功能

**代理管理**
- 自动发现本机已安装的 AI 代理（Claude Code、Codex、Trae、Gemini CLI、OpenClaw）
- 实时健康状态监控，一键修复损坏的技能目录
- 查看每个代理的已安装技能列表

**技能管理**
- 浏览、安装、卸载、更新各代理的技能
- 批量操作：批量更新或卸载多个技能
- 技能健康检查与验证

**技能商店**
- 从 GitHub 远程目录同步技能列表（Anthropic、ComposioHQ、Vercel 等来源）
- 搜索和筛选技能，支持按名称、描述、来源、兼容性过滤
- AI 解读：调用 AI 生成通俗易懂的技能用途说明
- 一键安装到指定代理

**项目管理**
- 自动扫描本地 Git 项目
- 为项目绑定代理和技能组
- 技能协调：检测项目所需技能与已安装技能的差异，生成修复计划并一键执行

**AI 助手**
- 对话式界面，描述目标后自动规划并部署技能
- 支持 Anthropic、OpenAI、Google Gemini 及自定义 Base URL
- 四阶段流水线：规划 → 解析 → 执行 → 报告
- 任务历史持久化，重启后可恢复

**设置**
- 主题（浅色 / 深色 / 跟随系统）
- 语言（中文 / 英文）
- AI 提供商配置（厂商、模型、API Key、Base URL）

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | React 19 + TypeScript 5 + Tailwind CSS 4 |
| 后端 | Go 1.25 + Wails v2 |
| 数据库 | SQLite（modernc.org/sqlite） |
| AI 工作进程 | Python 3.11+ |
| 构建工具 | Vite 6 + pnpm |

## 快速开始

**环境要求**

- Go 1.25+
- Node.js 20+ 和 pnpm
- Python 3.11+
- Wails CLI v2（`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）

**开发模式**

```bash
pnpm --dir frontend install
wails dev
```

**生产构建**

```bash
bash scripts/build.sh
# 输出：build/bin/agent-skills-manager.app
```

**Docker（仅前端 Web 模式）**

```bash
docker-compose up -d
# 访问 http://localhost:8080
```

## 项目结构

```
├── cmd/                    # 应用入口
├── internal/
│   ├── agents/             # 代理适配器（Claude Code、Codex、Gemini CLI 等）
│   ├── ai/                 # Python AI 工作进程桥接
│   ├── app/                # Wails 绑定层和视图模型
│   ├── catalog/            # 远程技能目录同步
│   ├── domain/             # 核心领域类型
│   ├── installer/          # 技能下载、解压、安装
│   ├── platform/           # 日志、文件系统工具
│   ├── projects/           # 项目扫描与激活
│   ├── reconcile/          # 技能协调计划生成
│   ├── skillgroups/        # 技能组管理
│   ├── storage/sqlite/     # SQLite 持久化
│   └── tasks/              # 任务历史管理
├── frontend/
│   └── src/
│       ├── components/     # 通用 UI 组件
│       ├── features/       # 功能模块（agents、skills、store、projects 等）
│       ├── lib/            # API 层、类型定义、工具函数
│       └── routes/         # 路由配置
├── python/
│   └── worker/
│       ├── pipeline/       # 四阶段 AI 流水线
│       └── providers/      # LLM 提供商适配器
├── docs/                   # 开发文档
└── scripts/                # 构建脚本
```

## 许可证

MIT
