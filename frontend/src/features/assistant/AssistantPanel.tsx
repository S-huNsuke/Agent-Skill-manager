import { useEffect, useRef, useState } from "react";
import type { AssistantTaskViewModel, SuggestionTemplate, TaskHistoryItem, TaskStatus, StatusTone } from "../../lib/mocks";
import { selectApi } from "../../lib/api";
import { getTaskStatusMeta, taskStatusMeta } from "../../lib/status";
import { StatusBadge } from "../../components/StatusBadge";

interface AssistantPanelProps {
  task: AssistantTaskViewModel;
  onSubmitGoal?: (goal: string) => void;
  onAdvance?: (taskID: string, action: string) => Promise<AssistantTaskViewModel>;
  onReset?: () => Promise<AssistantTaskViewModel>;
  onClose?: () => void;
}

interface ChatMessage {
  id: string;
  role: "user" | "assistant" | "system";
  content: string;
  time: string;
  status?: TaskStatus;
}

/** 将任务状态转为对话消息列表 */
function taskToMessages(task: AssistantTaskViewModel, submittedGoal: string): ChatMessage[] {
  const messages: ChatMessage[] = [];
  const now = new Date();
  const fmt = (d: Date) => `${d.getHours().toString().padStart(2, "0")}:${d.getMinutes().toString().padStart(2, "0")}`;

  if (submittedGoal || task.request) {
    messages.push({
      id: "user-goal",
      role: "user",
      content: submittedGoal || task.request,
      time: fmt(now),
    });
  }

  if (task.summary) {
    messages.push({
      id: "task-summary",
      role: "assistant",
      content: task.summary,
      time: fmt(now),
      status: task.status,
    });
  }

  if (task.recommendation) {
    messages.push({
      id: "task-rec",
      role: "assistant",
      content: `💡 建议：${task.recommendation}`,
      time: fmt(now),
    });
  }

  if (task.nextStep) {
    messages.push({
      id: "task-next",
      role: "system",
      content: `下一步：${task.nextStep}`,
      time: fmt(now),
    });
  }

  return messages;
}

/** 右侧常驻 AI 助手面板：对话式交互界面 */
export function AssistantPanel({ task, onSubmitGoal, onAdvance, onReset, onClose }: AssistantPanelProps) {
  const [goalText, setGoalText] = useState("");
  const [submittedGoal, setSubmittedGoal] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [suggestions, setSuggestions] = useState<SuggestionTemplate[]>([]);
  const [currentStatus, setCurrentStatus] = useState<TaskStatus>(task.status);
  const [showHistory, setShowHistory] = useState(false);
  const [taskHistory, setTaskHistory] = useState<TaskHistoryItem[]>([]);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    async function loadData() {
      try {
        const api = selectApi();
        const [tpls, history] = await Promise.all([api.getSuggestionTemplates(), api.getTaskHistory(20)]);
        setSuggestions(tpls);
        setTaskHistory(history);
      } catch {
        // 静默处理
      }
    }
    loadData();
  }, []);

  useEffect(() => {
    const msgs = taskToMessages(task, submittedGoal);
    if (msgs.length > 0) {
      setMessages(msgs);
    }
  }, [task, submittedGoal]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const currentMeta = getTaskStatusMeta(currentStatus);

  /** 提交用户输入的目标 */
  function handleSubmitGoal() {
    const trimmed = goalText.trim();
    if (!trimmed) return;
    setSubmittedGoal(trimmed);
    setCurrentStatus("planning");
    onSubmitGoal?.(trimmed);
    setGoalText("");
  }

  /** 重置对话 */
  async function handleReset() {
    setGoalText("");
    setSubmittedGoal("");
    setCurrentStatus("queued");
    setMessages([]);
    if (onReset) {
      try {
        await onReset();
      } catch {
        // 静默处理
      }
    }
  }

  /** 使用建议模板填充目标输入 */
  function handleUseSuggestion(tpl: SuggestionTemplate) {
    setGoalText(tpl.promptTemplate);
  }

  const isTaskActive = submittedGoal.length > 0 || (task.request && task.status !== "queued");

  return (
    <div className="flex flex-col h-full bg-surface-warm rounded-l-panel shadow-panel overflow-hidden">
      {/* 面板头部 */}
      <div className="shrink-0 px-5 py-4 border-b border-border-soft flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <span className="w-7 h-7 rounded-full bg-accent/15 flex items-center justify-center text-accent text-sm">✦</span>
          <div>
            <h2 className="font-display text-sm font-semibold text-ink">AI 助手</h2>
            <p className="text-[11px] text-ink-muted">随时为你规划技能方案</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isTaskActive && (
            <StatusBadge tone={currentMeta.tone} label={currentMeta.label} size="sm" />
          )}
          {onClose && (
            <button
              type="button"
              onClick={onClose}
              className="w-6 h-6 rounded-full hover:bg-surface-hover flex items-center justify-center text-ink-muted hover:text-ink transition-colors text-xs"
              title="关闭面板"
            >
              ✕
            </button>
          )}
        </div>
      </div>

      {/* 对话区域 */}
      <div className="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-3">
        {messages.length === 0 && !isTaskActive ? (
          <div className="flex-1 flex flex-col items-center justify-center text-center py-8">
            <span className="w-12 h-12 rounded-full bg-accent/10 flex items-center justify-center text-accent text-xl mb-4">✦</span>
            <p className="text-ink font-medium mb-1">告诉我你想做什么</p>
            <p className="text-xs text-ink-muted leading-relaxed max-w-[200px]">描述你的目标，AI 助手会帮你规划技能安装和环境配置方案。</p>
          </div>
        ) : (
          messages.map((msg) => (
            <div
              key={msg.id}
              className={`flex flex-col ${msg.role === "user" ? "items-end" : "items-start"}`}
            >
              <div
                className={`max-w-[90%] rounded-card px-3.5 py-2.5 text-sm leading-relaxed ${
                  msg.role === "user"
                    ? "bg-accent text-white"
                    : msg.role === "system"
                    ? "bg-badge-bg text-badge-ink border border-border"
                    : "bg-surface shadow-panel text-ink"
                }`}
              >
                {msg.content}
              </div>
              <div className="flex items-center gap-2 mt-1 px-1">
                <span className="text-[10px] text-ink-muted">{msg.time}</span>
                {msg.status && (
                  <StatusBadge
                    tone={taskStatusMeta[msg.status]?.tone ?? "muted"}
                    label={taskStatusMeta[msg.status]?.label ?? msg.status}
                    size="sm"
                  />
                )}
              </div>
            </div>
          ))
        )}

        {/* 任务进度条 */}
        {isTaskActive && currentStatus !== "queued" && (
          <div className="bg-surface rounded-card p-3 shadow-panel">
            <div className="flex items-center gap-1.5 mb-2">
              {(["planning", "resolving", "executing", "verifying", "completed"] as TaskStatus[]).map((step, idx) => {
                const stepIdx = ["planning", "resolving", "executing", "verifying", "completed"].indexOf(currentStatus);
                const isDone = idx < stepIdx;
                const isCurrent = step === currentStatus;
                return (
                  <div key={step} className="flex items-center gap-1.5">
                    <span className={`w-2 h-2 rounded-full shrink-0 ${
                      isDone ? "bg-stable-ink" : isCurrent ? "bg-accent" : "bg-border-soft"
                    }`} />
                    {idx < 4 && <span className={`w-3 h-0.5 ${isDone ? "bg-stable-ink" : "bg-border-soft"}`} />}
                  </div>
                );
              })}
            </div>
            <div className="flex items-center justify-between">
              <p className="text-xs text-ink-muted">{currentMeta.label}</p>
              <div className="flex gap-1.5">
                {currentStatus !== "completed" && currentStatus !== "failed" && (
                  <button
                    type="button"
                    onClick={async () => {
                      if (onAdvance && task.id && task.id !== "assistant-idle") {
                        try {
                          const actionFlow: Partial<Record<TaskStatus, string>> = {
                            planning: "resolve",
                            resolving: "execute",
                            executing: "verify",
                            verifying: "report",
                            blocked: "resolve",
                            recovering: "verify",
                          };
                          const action = actionFlow[currentStatus];
                          if (action) {
                            const updated = await onAdvance(task.id, action);
                            setCurrentStatus(updated.status);
                          }
                        } catch {
                          // 静默处理
                        }
                      } else {
                        const flow: Partial<Record<TaskStatus, TaskStatus>> = {
                          planning: "resolving",
                          resolving: "executing",
                          executing: "verifying",
                          verifying: "completed",
                          blocked: "recovering",
                          recovering: "verifying",
                        };
                        const next = flow[currentStatus];
                        if (next) setCurrentStatus(next);
                      }
                    }}
                    className="rounded-chip px-2 py-0.5 text-[10px] font-medium bg-accent text-white hover:bg-accent-warm transition-colors"
                  >
                    继续
                  </button>
                )}
                <button
                  type="button"
                  onClick={handleReset}
                  className="rounded-chip px-2 py-0.5 text-[10px] font-medium bg-surface-warm text-ink-muted hover:text-ink transition-colors"
                >
                  重置
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 快捷建议 */}
        {!isTaskActive && suggestions.length > 0 && (
          <div className="flex flex-col gap-1.5 mt-2">
            <p className="text-[10px] text-ink-muted uppercase tracking-wider px-1">快捷建议</p>
            {suggestions.slice(0, 4).map((tpl) => (
              <button
                key={tpl.id}
                type="button"
                onClick={() => handleUseSuggestion(tpl)}
                className="text-left w-full p-2 rounded-card bg-surface shadow-panel hover:shadow-panel-hover transition-shadow"
              >
                <p className="text-xs font-medium text-ink">{tpl.title}</p>
                <p className="text-[10px] text-ink-muted truncate">{tpl.description}</p>
              </button>
            ))}
          </div>
        )}

        {/* 历史任务 */}
        {taskHistory.length > 0 && (
          <div className="mt-2">
            <div className="flex items-center justify-between px-1 mb-1.5">
              <p className="text-[10px] text-ink-muted uppercase tracking-wider">最近任务</p>
              <button
                type="button"
                onClick={() => setShowHistory(!showHistory)}
                className="text-[10px] text-accent hover:text-accent-warm transition-colors"
              >
                {showHistory ? "收起" : "全部"}
              </button>
            </div>
            {(showHistory ? taskHistory : taskHistory.slice(0, 2)).map((item) => (
              <div key={item.id} className="flex items-center justify-between p-2 rounded-card bg-surface mb-1">
                <p className="text-xs text-ink truncate flex-1">{item.goal}</p>
                <StatusBadge
                  tone={item.status === "completed" ? "stable" : "critical"}
                  label={item.status === "completed" ? "完成" : "失败"}
                  size="sm"
                />
              </div>
            ))}
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* 输入区域 */}
      <div className="shrink-0 px-4 py-3 border-t border-border-soft">
        <div className="flex gap-2 items-end">
          <textarea
            className="flex-1 min-h-[36px] max-h-[100px] rounded-card border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-ink-muted focus:outline-none focus:ring-2 focus:ring-accent/40 resize-none"
            placeholder="输入你的目标..."
            value={goalText}
            onChange={(e) => setGoalText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSubmitGoal();
              }
            }}
            rows={1}
          />
          <button
            type="button"
            onClick={handleSubmitGoal}
            disabled={goalText.trim().length === 0}
            className="shrink-0 w-9 h-9 rounded-card bg-accent text-white flex items-center justify-center hover:bg-accent-warm transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            title="发送"
          >
            ↑
          </button>
        </div>
        <p className="text-[10px] text-ink-muted mt-1.5 px-1">按 Enter 发送，Shift+Enter 换行</p>
      </div>
    </div>
  );
}
