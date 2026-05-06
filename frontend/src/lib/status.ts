import type { StatusTone, TaskStatus } from "./mocks";

export const taskStatusMeta: Record<
  TaskStatus,
  { label: string; detail: string; tone: StatusTone }
> = {
  queued: {
    label: "等待中",
    detail: "任务已创建，等待执行。",
    tone: "muted",
  },
  planning: {
    label: "规划中",
    detail: "正在分析任务步骤和影响范围。",
    tone: "attention",
  },
  resolving: {
    label: "分析中",
    detail: "正在检查依赖、路径和兼容性。",
    tone: "attention",
  },
  executing: {
    label: "执行中",
    detail: "正在执行安装或配置操作。",
    tone: "attention",
  },
  verifying: {
    label: "验证中",
    detail: "执行完成，正在验证结果。",
    tone: "stable",
  },
  recovering: {
    label: "恢复中",
    detail: "正在回退或重试操作。",
    tone: "attention",
  },
  completed: {
    label: "已完成",
    detail: "任务执行成功。",
    tone: "stable",
  },
  failed: {
    label: "已失败",
    detail: "任务执行失败，需要手动处理。",
    tone: "critical",
  },
  blocked: {
    label: "已阻塞",
    detail: "任务被权限或前置条件阻止。",
    tone: "critical",
  },
  cancelled: {
    label: "已取消",
    detail: "任务已被手动取消。",
    tone: "muted",
  },
};

export function getTaskStatusMeta(status: TaskStatus) {
  return taskStatusMeta[status];
}
