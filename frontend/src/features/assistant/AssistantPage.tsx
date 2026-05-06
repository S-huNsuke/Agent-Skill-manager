import { useEffect, useMemo, useState } from "react";
import type { AssistantTaskViewModel, StatusTone, TaskStatus, SuggestionTemplate, TaskHistoryItem } from "../../lib/mocks";
import { selectApi } from "../../lib/api";
import { getTaskStatusMeta, taskStatusMeta } from "../../lib/status";
import { Modal } from "../../components/Modal";
import { StatusBadge } from "../../components/StatusBadge";

interface AssistantPageProps {
  task: AssistantTaskViewModel;
  onSubmitGoal?: (goal: string) => void;
}

function toneTextClass(tone: StatusTone): string {
  const map: Record<StatusTone, string> = { stable: "text-stable-ink", attention: "text-attention-ink", critical: "text-critical-ink", muted: "text-ink-muted" };
  return map[tone];
}

function stepIndicatorClass(isDone: boolean, isCurrent: boolean): string {
  if (isDone) return "bg-stable-ink text-white";
  if (isCurrent) return "bg-accent text-white";
  return "bg-badge-bg text-badge-ink";
}

/** AI 助手页面：提供目标输入、建议提示、任务执行进度和历史记录 */
export function AssistantPage({ task, onSubmitGoal }: AssistantPageProps) {
  const [goalText, setGoalText] = useState("");
  const [submittedGoal, setSubmittedGoal] = useState("");
  const [currentTask, setCurrentTask] = useState<AssistantTaskViewModel>(task);
  const [advancing, setAdvancing] = useState(false);
  const [suggestions, setSuggestions] = useState<SuggestionTemplate[]>([]);
  const [taskHistory, setTaskHistory] = useState<TaskHistoryItem[]>([]);
  const [showHistory, setShowHistory] = useState(false);

  useEffect(() => {
    setCurrentTask(task);
  }, [task]);

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

  const currentMeta = getTaskStatusMeta(currentTask.status);
  const currentStepIndex = useMemo(() => {
    const steps: TaskStatus[] = ["queued", "planning", "resolving", "executing", "blocked", "recovering", "verifying", "completed", "failed", "cancelled"];
    return steps.indexOf(currentTask.status);
  }, [currentTask.status]);

  const visibleSteps: TaskStatus[] = ["queued", "planning", "resolving", "executing", "verifying", "completed"];
  const alertSteps: TaskStatus[] = ["blocked", "recovering", "failed", "cancelled"];
  const isAlert = alertSteps.includes(currentTask.status);
  const hasSubmitted = submittedGoal.length > 0;
  const displayGoal = hasSubmitted ? submittedGoal : currentTask.request;

  /** 提交用户输入的目标 */
  function handleSubmitGoal() {
    const trimmed = goalText.trim();
    if (!trimmed) return;
    setSubmittedGoal(trimmed);
    onSubmitGoal?.(trimmed);
  }

  /** 推进任务到下一阶段 */
  async function handleAdvance(action: string) {
    if (!currentTask.id || currentTask.id === "assistant-idle") return;
    setAdvancing(true);
    try {
      const api = selectApi();
      const updated = await api.advanceAssistantTask(currentTask.id, action);
      setCurrentTask(updated);
    } catch (err) {
      console.error("advance task failed:", err);
    } finally {
      setAdvancing(false);
    }
  }

  /** 重置所有状态 */
  async function handleReset() {
    try {
      const api = selectApi();
      const resetResult = await api.resetAssistantTask();
      setCurrentTask(resetResult);
    } catch {
      // 静默处理
    }
    setGoalText("");
    setSubmittedGoal("");
  }

  /** 使用建议模板填充目标输入 */
  function handleUseSuggestion(tpl: SuggestionTemplate) {
    setGoalText(tpl.promptTemplate);
  }

  /** 根据当前状态决定下一步操作 */
  function getNextAction(): string | null {
    switch (currentTask.status) {
      case "planning": return "resolve";
      case "resolving": return "execute";
      case "executing": return "verify";
      case "verifying": return "report";
      case "blocked": return "resolve";
      default: return null;
    }
  }

  /** 按类别分组建议模板 */
  const groupedSuggestions = useMemo(() => {
    const groups: Record<string, SuggestionTemplate[]> = {};
    for (const tpl of suggestions) {
      if (!groups[tpl.category]) groups[tpl.category] = [];
      groups[tpl.category].push(tpl);
    }
    return groups;
  }, [suggestions]);

  return (
    <section className="animate-page-in">
      {/* Header */}
      <div className="bg-surface rounded-panel shadow-panel p-8 mb-8">
        <p className="uppercase tracking-widest text-xs text-ink-muted font-body">AI 助手</p>
        <h1 className="font-display text-3xl font-semibold text-ink tracking-tight">任务执行助手</h1>
        <p className="text-lg text-ink-soft leading-relaxed">{currentTask.summary}</p>
      </div>

      {!hasSubmitted ? (
        <div className="grid grid-cols-[1fr_320px] gap-6">
          {/* Goal Input */}
          <article className="bg-surface rounded-card shadow-panel p-6">
            <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-3">输入目标</p>
            <textarea
              className="w-full min-h-[120px] rounded-panel border border-border bg-surface-warm p-4 text-ink placeholder:text-ink-muted focus:outline-none focus:ring-2 focus:ring-accent/40 resize-y"
              placeholder="描述你想要实现的目标，AI 助手将帮你规划技能安装方案..."
              value={goalText}
              onChange={(e) => setGoalText(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) handleSubmitGoal(); }}
            />
            <div className="flex items-center justify-between mt-4">
              <p className="text-xs text-ink-muted">按 ⌘+Enter 快速提交</p>
              <button type="button" className="px-5 py-2.5 rounded-pill font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40 disabled:cursor-not-allowed" disabled={goalText.trim().length === 0} onClick={handleSubmitGoal}>
                提交目标
              </button>
            </div>
          </article>

          {/* Suggestion Templates */}
          <aside className="bg-surface-cream rounded-panel p-6">
            <div className="mb-4">
              <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body">建议</p>
              <h2 className="font-display text-lg font-semibold text-ink">快速模板</h2>
            </div>
            <div className="flex flex-col gap-3">
              {Object.entries(groupedSuggestions).map(([category, tpls]) => (
                <div key={category}>
                  <p className="text-xs text-ink-muted font-medium mb-1">{category}</p>
                  {tpls.map((tpl) => (
                    <button key={tpl.id} type="button" onClick={() => handleUseSuggestion(tpl)} className="text-left w-full p-2.5 rounded-card hover:bg-surface transition-colors mb-1">
                      <p className="text-sm font-medium text-ink">{tpl.title}</p>
                      <p className="text-xs text-ink-muted">{tpl.description}</p>
                    </button>
                  ))}
                </div>
              ))}
              {suggestions.length === 0 && <p className="text-sm text-ink-soft">暂无建议模板</p>}
            </div>
          </aside>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 flex flex-col gap-5">
            {/* Current Task */}
            <article className="bg-surface rounded-panel shadow-panel p-6">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">当前任务</p>
                  <h2 className="font-display text-xl font-semibold text-ink">{displayGoal || "暂无执行中的任务"}</h2>
                </div>
                <StatusBadge tone={currentMeta.tone} label={currentMeta.label} size="md" />
              </div>
              <p className="text-ink-soft leading-relaxed mb-5">{currentTask.blocker || currentMeta.detail}</p>
              <div className="flex gap-3">
                {getNextAction() && (
                  <button type="button" className="px-5 py-2.5 rounded-pill font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40 disabled:cursor-not-allowed" disabled={advancing} onClick={() => handleAdvance(getNextAction()!)}>
                    {advancing ? "处理中..." : "继续执行"}
                  </button>
                )}
                {currentTask.status !== "completed" && currentTask.status !== "cancelled" && (
                  <button type="button" className="px-5 py-2.5 rounded-pill font-medium bg-surface-hover text-ink border border-border hover:bg-surface-warm transition-colors" disabled={advancing} onClick={() => handleAdvance("cancel")}>
                    取消任务
                  </button>
                )}
                <button type="button" className="px-5 py-2.5 rounded-pill font-medium bg-surface-hover text-ink border border-border hover:bg-surface-warm transition-colors" onClick={handleReset}>
                  重新开始
                </button>
              </div>
            </article>

            {/* Alert */}
            {isAlert && (
              <article className={`rounded-panel p-5 border-l-4 ${currentTask.status === "failed" || currentTask.status === "blocked" ? "bg-critical/5 border-l-critical" : "bg-attention-bg/50 border-l-attention-ink"}`}>
                <div className="flex items-center gap-3 mb-2">
                  <span className="text-lg">{currentTask.status === "failed" ? "⚠" : currentTask.status === "blocked" ? "🚫" : "🔄"}</span>
                  <h3 className="font-display text-lg font-semibold text-ink">{currentMeta.label}</h3>
                </div>
                <p className="text-ink-soft text-sm">{currentMeta.detail}</p>
                {currentTask.status === "blocked" && (
                  <button type="button" className="mt-3 px-4 py-2 rounded-pill text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors" disabled={advancing} onClick={() => handleAdvance("resolve")}>
                    尝试恢复
                  </button>
                )}
              </article>
            )}

            {/* Progress Timeline */}
            <article className="bg-surface rounded-panel shadow-panel p-6">
              <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-4">执行进度</p>
              <div className="flex flex-col gap-0">
                {visibleSteps.map((step, index) => {
                  const meta = taskStatusMeta[step];
                  const isCurrent = currentTask.status === step;
                  const isDone = index < currentStepIndex && currentTask.status !== "failed";
                  return (
                    <div key={step} className="flex items-start gap-4 relative">
                      {index > 0 && <div className={`absolute left-[15px] -top-3 w-0.5 h-6 ${isDone || isCurrent ? "bg-stable-ink" : "bg-border-soft"}`} />}
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium shrink-0 z-10 ${stepIndicatorClass(isDone, isCurrent)}`}>
                        {isDone ? "✓" : index + 1}
                      </div>
                      <div className="pb-5">
                        <p className={`font-medium ${isCurrent || isDone ? "text-ink" : toneTextClass(meta.tone)}`}>{meta.label}</p>
                        <p className="text-sm text-ink-muted mt-0.5">{meta.detail}</p>
                      </div>
                    </div>
                  );
                })}
              </div>
            </article>
          </div>

          <div className="flex flex-col gap-5">
            {/* Recommendation */}
            {currentTask.recommendation && (
              <article className="bg-surface rounded-card shadow-panel p-5">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">建议</p>
                <p className="text-ink leading-relaxed">{currentTask.recommendation}</p>
              </article>
            )}

            {/* Action Log */}
            {currentTask.records?.length > 0 && (
              <article className="bg-surface rounded-card shadow-panel p-5">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">执行记录</p>
                <div className="flex flex-col gap-2">
                  {currentTask.records?.map((record, idx) => <p key={idx} className="text-sm text-ink-soft">{record}</p>)}
                </div>
              </article>
            )}

            {/* Task History */}
            <article className="bg-surface rounded-card shadow-panel p-5">
              <div className="flex items-center justify-between mb-2">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body">历史</p>
                <button type="button" onClick={() => setShowHistory(true)} className="text-xs text-accent hover:text-accent-warm transition-colors">
                  查看全部
                </button>
              </div>
              {taskHistory.length > 0 ? (
                <div className="flex flex-col gap-2">
                  {taskHistory.slice(0, 3).map((item) => (
                    <div key={item.id} className="flex items-center justify-between p-2 rounded-card bg-surface-warm">
                      <p className="text-sm text-ink truncate">{item.goal}</p>
                      <StatusBadge tone={item.status === "completed" ? "stable" : "critical"} label={item.status === "completed" ? "完成" : "失败"} />
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-ink-soft">暂无历史任务</p>
              )}
            </article>

            {/* Status Legend */}
            <article className="bg-surface-warm rounded-card p-5">
              <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">状态说明</p>
              <div className="flex flex-col gap-2">
                {(["queued", "planning", "executing", "completed", "failed"] as TaskStatus[]).map((s) => {
                  const m = taskStatusMeta[s];
                  return (
                    <div key={s} className="flex items-center gap-2">
                      <span className={`w-2 h-2 rounded-full shrink-0 ${m.tone === "stable" ? "bg-stable-ink" : m.tone === "attention" ? "bg-attention-ink" : m.tone === "critical" ? "bg-critical-ink" : "bg-ink-muted"}`} />
                      <span className="text-sm text-ink-soft">{m.label}</span>
                    </div>
                  );
                })}
              </div>
            </article>
          </div>
        </div>
      )}

      {/* Task History Modal */}
      <Modal open={showHistory} onClose={() => setShowHistory(false)} title="任务历史" subtitle="历史任务记录">
        <div className="flex flex-col gap-3">
          {taskHistory.length > 0 ? taskHistory.map((item) => (
            <div key={item.id} className="bg-surface rounded-card p-4">
              <div className="flex items-center justify-between mb-1">
                <p className="font-medium text-ink">{item.goal}</p>
                <StatusBadge tone={item.status === "completed" ? "stable" : "critical"} label={item.status === "completed" ? "完成" : "失败"} />
              </div>
              <p className="text-sm text-ink-soft">{item.summary}</p>
              <div className="flex gap-3 mt-1 text-xs text-ink-muted">
                <span>开始: {item.startedAt}</span>
                <span>结束: {item.finishedAt}</span>
              </div>
            </div>
          )) : <p className="text-ink-soft text-center py-8">暂无历史任务</p>}
        </div>
      </Modal>
    </section>
  );
}
