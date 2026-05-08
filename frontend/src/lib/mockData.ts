import type { AppSnapshot } from "./types";

export const mockSnapshot: AppSnapshot = {
  dashboard: {
    title: "本机技能与运行状态",
    summary: "查看本机 AI 代理、已安装技能和任务执行状态。",
    spotlight: "系统已就绪，等待发现本地代理。",
    highlights: [
      {
        id: "health",
        title: "任务状态",
        value: "0 个任务",
        detail: "暂无运行中的任务。",
        tone: "stable",
        tag: "概况",
      },
    ],
    tasks: [],
    notes: [
      "安装技能前请先查看兼容性信息。",
      "每个项目只能绑定一个技能组。",
    ],
  },
  agents: [],
  skills: [],
  store: [],
  projects: [],
  assistant: {
    id: "assistant-idle",
    request: "",
    status: "queued",
    nextStep: "等待用户输入目标",
    summary: "AI 助手待命中，输入目标即可开始规划。",
    recommendation: "",
    reason: "",
    records: [],
    planJson: "",
    planSteps: [],
    resolvedActions: [],
  },
  diagnostics: [
    {
      id: "diag-1",
      label: "Wails 绑定",
      value: "使用示例数据（未连接后端）",
      tone: "attention",
    },
    {
      id: "diag-2",
      label: "前端路由",
      value: "已就绪",
      tone: "stable",
    },
  ],
};
