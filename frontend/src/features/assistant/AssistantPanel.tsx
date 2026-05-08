import { useEffect, useRef, useState } from "react";
import type { AISettingsViewModel, AssistantChatMessageViewModel, AssistantTaskViewModel, SuggestionTemplate, TaskHistoryItem, TaskStatus } from "../../lib/mocks";
import { waitForApi } from "../../lib/api";
import { getTaskStatusMeta, taskStatusMeta } from "../../lib/status";
import { StatusBadge } from "../../components/StatusBadge";
import { aiProviderPresets, applyPreset, defaultAISettings, getModelOptions, getPresetFromSettings, type AiPreset } from "../ai/aiSettings";

interface AssistantPanelProps {
  task: AssistantTaskViewModel;
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

interface ChatSession {
  id: string;
  title: string;
  messages: ChatMessage[];
  createdAt: number;
  updatedAt: number;
}

const CHAT_SESSIONS_STORAGE_KEY = "agent-skills-manager.chatSessions";

function createMessageId(role: ChatMessage["role"]) {
  return `${role}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function createSessionTitle(message: string) {
  const compact = message.replace(/\s+/g, " ").trim();
  return compact.length > 18 ? `${compact.slice(0, 18)}...` : compact || "新对话";
}

function loadChatSessions(): ChatSession[] {
  try {
    const storage = window.localStorage;
    if (typeof storage?.getItem !== "function") return [];
    const raw = storage.getItem(CHAT_SESSIONS_STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as ChatSession[];
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter((item) => item && typeof item.id === "string" && Array.isArray(item.messages))
      .slice(0, 30);
  } catch {
    return [];
  }
}

function saveChatSessions(sessions: ChatSession[]) {
  try {
    const storage = window.localStorage;
    if (typeof storage?.setItem !== "function") return;
    storage.setItem(CHAT_SESSIONS_STORAGE_KEY, JSON.stringify(sessions.slice(0, 30)));
  } catch {
    // localStorage can be unavailable in embedded runtimes or tests.
  }
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
export function AssistantPanel({ task, onAdvance, onReset, onClose }: AssistantPanelProps) {
  const [goalText, setGoalText] = useState("");
  const [submittedGoal, setSubmittedGoal] = useState("");
  const [activeTask, setActiveTask] = useState<AssistantTaskViewModel>(task);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [suggestions, setSuggestions] = useState<SuggestionTemplate[]>([]);
  const [currentStatus, setCurrentStatus] = useState<TaskStatus>(task.status);
  const [showHistory, setShowHistory] = useState(false);
  const [taskHistory, setTaskHistory] = useState<TaskHistoryItem[]>([]);
  const [chatSessions, setChatSessions] = useState<ChatSession[]>(() => loadChatSessions());
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [aiSettings, setAiSettings] = useState<AISettingsViewModel>(defaultAISettings);
  const [aiPreset, setAiPreset] = useState<AiPreset>("none");
  const [aiSettingsStatus, setAiSettingsStatus] = useState<string | null>(null);
  const [savingAISettings, setSavingAISettings] = useState(false);
  const [showAISettings, setShowAISettings] = useState(false);
  const [sending, setSending] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    async function loadData() {
      try {
        const api = await waitForApi();
        const [tplsRes, historyRes, aiRes] = await Promise.allSettled([
          api.getSuggestionTemplates(),
          api.getTaskHistory(20),
          api.getAISettings(),
        ]);
        if (cancelled) return;
        if (tplsRes.status === "fulfilled") {
          setSuggestions(tplsRes.value);
        }
        if (historyRes.status === "fulfilled") {
          setTaskHistory(historyRes.value);
        }
        if (aiRes.status === "fulfilled") {
          setAiSettings(aiRes.value);
          setAiPreset(getPresetFromSettings(aiRes.value));
        }
      } catch {
        // 静默处理
      }
    }
    void loadData();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    setActiveTask(task);
  }, [task]);

  useEffect(() => {
    if (activeSessionId || messages.length > 0) {
      setCurrentStatus(activeTask.status);
      return;
    }
    const msgs = taskToMessages(activeTask, submittedGoal);
    if (msgs.length > 0 && activeTask.request) {
      setMessages(msgs);
    }
    // 同步 task.status 到 currentStatus
    setCurrentStatus(activeTask.status);
  }, [activeTask, activeSessionId, messages.length, submittedGoal]);

  useEffect(() => {
    saveChatSessions(chatSessions);
  }, [chatSessions]);

  useEffect(() => {
    if (messagesEndRef.current && typeof messagesEndRef.current.scrollIntoView === 'function') {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages]);

  const currentMeta = getTaskStatusMeta(currentStatus);
  const chatHistory: AssistantChatMessageViewModel[] = messages
    .filter((msg) => msg.role === "user" || msg.role === "assistant")
    .map((msg) => ({ role: msg.role, content: msg.content }));

  async function handleSaveAISettings() {
    setSavingAISettings(true);
    setAiSettingsStatus(null);
    try {
      const api = await waitForApi();
      const result = await api.saveAISettings(aiSettings);
      if (result !== "ok") {
        throw new Error(result);
      }
      setAiSettingsStatus("AI 配置已保存");
    } catch (err) {
      setAiSettingsStatus(`AI 配置保存失败: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setSavingAISettings(false);
    }
  }

  /** 提交用户输入的目标 */
  async function handleSubmitGoal() {
    const trimmed = goalText.trim();
    if (!trimmed || sending) return;
    setSending(true);
    setSubmitError(null);
    setCurrentStatus("queued");
    setGoalText("");
    const time = new Date().toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
    const sessionId = activeSessionId ?? `chat-${Date.now()}`;
    const sessionTitle = createSessionTitle(trimmed);
    const pendingMessages: ChatMessage[] = [
      ...messages,
      { id: createMessageId("user"), role: "user", content: trimmed, time },
    ];
    setActiveSessionId(sessionId);
    setMessages(pendingMessages);
    updateChatSession(sessionId, sessionTitle, pendingMessages);

    try {
      const api = await waitForApi();
      const response = await api.chatAssistant(trimmed, chatHistory);
      const nextMessages: ChatMessage[] = [
        ...pendingMessages.filter((msg) => !msg.id.startsWith("assistant-pending-")),
        {
          id: createMessageId("assistant"),
          role: "assistant",
          content: response.reply,
          time: new Date().toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }),
          status: response.error ? "failed" : undefined,
        },
      ];
      setMessages(nextMessages);
      updateChatSession(sessionId, sessionTitle, nextMessages);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setSubmitError(message);
      setCurrentStatus("failed");
      const nextMessages: ChatMessage[] = [
        ...pendingMessages.filter((msg) => !msg.id.startsWith("assistant-pending-")),
        {
          id: createMessageId("assistant"),
          role: "assistant",
          content: `AI 请求失败：${message}`,
          time: new Date().toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }),
          status: "failed",
        },
      ];
      setMessages(nextMessages);
      updateChatSession(sessionId, sessionTitle, nextMessages);
    } finally {
      setSending(false);
    }
  }

  function updateChatSession(sessionId: string, title: string, nextMessages: ChatMessage[]) {
    const now = Date.now();
    setChatSessions((sessions) => {
      const existing = sessions.find((session) => session.id === sessionId);
      const nextSession: ChatSession = {
        id: sessionId,
        title: existing?.title || title,
        messages: nextMessages,
        createdAt: existing?.createdAt || now,
        updatedAt: now,
      };
      return [nextSession, ...sessions.filter((session) => session.id !== sessionId)].slice(0, 30);
    });
  }

  /** 重置对话 */
  function clearCurrentChat() {
    setGoalText("");
    setSubmittedGoal("");
    setCurrentStatus("queued");
    setMessages([]);
    setActiveSessionId(null);
    setSubmitError(null);
    setSending(false);
  }

  function openChatSession(session: ChatSession) {
    setActiveSessionId(session.id);
    setMessages(session.messages);
    setGoalText("");
    setSubmitError(null);
    setSending(false);
  }

  function deleteChatSession(sessionId: string) {
    setChatSessions((sessions) => sessions.filter((session) => session.id !== sessionId));
    if (activeSessionId === sessionId) {
      clearCurrentChat();
    }
  }

  async function deleteTaskHistoryItem(item: TaskHistoryItem) {
    setTaskHistory((items) => items.filter((current) => current.id !== item.id));
    try {
      const api = await waitForApi();
      const result = await api.deleteTaskHistoryItem(item.id);
      if (result !== "ok") {
        throw new Error(result);
      }
    } catch (err) {
      setSubmitError(`删除任务失败：${err instanceof Error ? err.message : String(err)}`);
      setTaskHistory((items) => {
        if (items.some((current) => current.id === item.id)) return items;
        return [item, ...items];
      });
    }
  }

  async function handleReset() {
    clearCurrentChat();
    if (onReset) {
      try {
        const resetTask = await onReset();
        setActiveTask(resetTask);
      } catch {
        // 静默处理
      }
    }
  }

  /** 使用建议模板填充目标输入 */
  function handleUseSuggestion(tpl: SuggestionTemplate) {
    setGoalText(tpl.promptTemplate);
  }

  const isTaskActive = activeTask.request && activeTask.status !== "queued";

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
          <button
            type="button"
            onClick={clearCurrentChat}
            className="rounded-chip px-2.5 py-1 text-[11px] font-medium bg-surface text-ink-soft hover:bg-surface-hover transition-colors"
          >
            新对话
          </button>
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

      <div className="shrink-0 px-4 py-3 border-b border-border-soft bg-surface/80">
        <div className="flex items-center justify-between gap-2">
          <button
            type="button"
            onClick={() => setShowAISettings((value) => !value)}
            className="text-left flex-1"
          >
            <p className="text-[10px] uppercase tracking-wider text-ink-muted">AI 配置</p>
            <p className="text-xs text-ink-soft">{showAISettings ? "收起配置区" : "展开配置区，选择供应商和密钥"}</p>
          </button>
          <div className="flex items-center gap-2">
          <StatusBadge tone={aiSettings.provider === "none" ? "attention" : "stable"} label={aiSettings.provider === "none" ? "本地模式" : "已配置"} size="sm" />
            <button
              type="button"
              onClick={() => setShowAISettings((value) => !value)}
              className="rounded-chip px-2.5 py-1 text-[11px] font-medium bg-surface text-ink-soft hover:bg-surface-hover transition-colors"
            >
              {showAISettings ? "收起 AI 配置" : "展开 AI 配置"}
            </button>
          </div>
        </div>
        {showAISettings && (
          <div className="flex flex-col gap-2 mt-3">
          <label className="block text-[11px] text-ink-muted" htmlFor="assistant-ai-provider">供应商预设</label>
          <select
            id="assistant-ai-provider"
            value={aiPreset}
            onChange={(e) => {
              const nextPreset = e.target.value as AiPreset;
              setAiPreset(nextPreset);
              setAiSettings((current) => applyPreset(nextPreset, current));
            }}
            className="w-full rounded-card border border-border bg-surface px-3 py-2 text-sm text-ink focus:outline-none focus:ring-1 focus:ring-accent"
          >
            {aiProviderPresets.map((preset) => (
              <option key={preset.value} value={preset.value}>{preset.label}</option>
            ))}
          </select>
          <label className="block text-[11px] text-ink-muted" htmlFor="assistant-ai-key">API Key</label>
          <input
            id="assistant-ai-key"
            type="password"
            value={aiSettings.apiKey}
            onChange={(e) => setAiSettings({ ...aiSettings, apiKey: e.target.value })}
            placeholder="输入 API Key"
            className="w-full rounded-card border border-border bg-surface px-3 py-2 text-sm text-ink focus:outline-none focus:ring-1 focus:ring-accent"
            disabled={aiSettings.provider === "none"}
          />
          <label className="block text-[11px] text-ink-muted" htmlFor="assistant-ai-model">模型</label>
          <input
            id="assistant-ai-model"
            type="text"
            value={aiSettings.model}
            onChange={(e) => setAiSettings({ ...aiSettings, model: e.target.value })}
            list="assistant-ai-model-options"
            placeholder="输入模型名称"
            className="w-full rounded-card border border-border bg-surface px-3 py-2 text-sm text-ink focus:outline-none focus:ring-1 focus:ring-accent"
            disabled={aiSettings.provider === "none"}
          />
          <datalist id="assistant-ai-model-options">
            {getModelOptions(aiPreset).map((model) => (
              <option key={model} value={model} />
            ))}
          </datalist>
          <label className="block text-[11px] text-ink-muted" htmlFor="assistant-ai-base-url">Base URL（可选）</label>
          <input
            id="assistant-ai-base-url"
            type="text"
            value={aiSettings.baseUrl}
            onChange={(e) => setAiSettings({ ...aiSettings, baseUrl: e.target.value })}
            placeholder={aiSettings.provider === "openai-compatible" ? "https://openrouter.ai/api/v1" : "留空使用默认地址"}
            className="w-full rounded-card border border-border bg-surface px-3 py-2 text-sm text-ink focus:outline-none focus:ring-1 focus:ring-accent"
            disabled={aiSettings.provider === "none"}
          />
          <div className="flex items-center justify-between gap-2 pt-1">
            <button
              type="button"
              onClick={handleSaveAISettings}
              disabled={savingAISettings}
              className="rounded-pill px-4 py-2 text-xs font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40"
            >
              {savingAISettings ? "保存中..." : "保存 AI 配置"}
            </button>
            {aiSettingsStatus && <span className="text-[11px] text-ink-muted text-right">{aiSettingsStatus}</span>}
          </div>
          </div>
        )}
      </div>

      {/* 对话区域 */}
      <div className="flex-1 overflow-y-auto px-4 py-3 flex flex-col gap-3">
        {submitError && (
          <div className="rounded-card border border-attention-ink/20 bg-attention-bg/40 px-3 py-2 text-xs text-ink">
            {submitError}
          </div>
        )}
        <div role="log" aria-label="当前对话" className="flex flex-col gap-3">
          {messages.length === 0 && !isTaskActive ? (
            <div className="flex-1 flex flex-col items-center justify-center text-center py-8">
              <span className="w-12 h-12 rounded-full bg-accent/10 flex items-center justify-center text-accent text-xl mb-4">✦</span>
              <p className="text-ink font-medium mb-1">我是你的 AI 助手</p>
              <p className="text-xs text-ink-muted leading-relaxed max-w-[220px]">直接输入问题。我会像聊天机器人一样回复；如果未配置模型，会使用本地说明模式。</p>
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
        </div>

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
                      if (onAdvance && activeTask.id && activeTask.id !== "assistant-idle") {
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
                            const updated = await onAdvance(activeTask.id, action);
                            setActiveTask(updated);
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

        {/* 历史对话 */}
        <div className="mt-2">
          <div className="flex items-center justify-between px-1 mb-1.5">
            <p className="text-[10px] text-ink-muted uppercase tracking-wider">历史对话</p>
            {chatSessions.length > 0 && (
              <span className="text-[10px] text-ink-muted">{chatSessions.length} 条</span>
            )}
          </div>
          {chatSessions.length === 0 ? (
            <div className="rounded-card border border-dashed border-border-soft bg-surface/60 px-3 py-2 text-[11px] text-ink-muted">
              暂无历史对话，发送消息后会自动保存。
            </div>
          ) : (
            <div className="flex flex-col gap-1">
              {chatSessions.map((session) => (
                <div
                  key={session.id}
                  className={`flex items-center gap-1.5 rounded-card p-1.5 ${
                    activeSessionId === session.id ? "bg-accent/10" : "bg-surface"
                  }`}
                >
                  <button
                    type="button"
                    onClick={() => openChatSession(session)}
                    aria-label={`打开对话：${session.title}`}
                    className="min-w-0 flex-1 text-left px-2 py-1 rounded-chip hover:bg-surface-hover transition-colors"
                  >
                    <p className="text-xs text-ink truncate">{session.title}</p>
                    <p className="text-[10px] text-ink-muted">{session.messages.length} 条消息</p>
                  </button>
                  <button
                    type="button"
                    onClick={() => deleteChatSession(session.id)}
                    aria-label={`删除对话：${session.title}`}
                    className="shrink-0 rounded-chip px-2 py-1 text-[10px] font-medium text-critical-ink hover:bg-critical-bg transition-colors"
                  >
                    删除
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

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
              <div key={item.id} className="flex items-center gap-1.5 p-2 rounded-card bg-surface mb-1">
                <p className="text-xs text-ink truncate flex-1">{item.goal}</p>
                <StatusBadge
                  tone={item.status === "completed" ? "stable" : "critical"}
                  label={item.status === "completed" ? "完成" : "失败"}
                  size="sm"
                />
                <button
                  type="button"
                  onClick={() => void deleteTaskHistoryItem(item)}
                  aria-label={`删除任务：${item.goal}`}
                  className="shrink-0 rounded-chip px-2 py-1 text-[10px] font-medium text-critical-ink hover:bg-critical-bg transition-colors"
                >
                  删除
                </button>
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
            placeholder="输入消息..."
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
            disabled={goalText.trim().length === 0 || sending}
            className="shrink-0 w-9 h-9 rounded-card bg-accent text-white flex items-center justify-center hover:bg-accent-warm transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            title={sending ? "发送中" : "发送"}
          >
            {sending ? "…" : "↑"}
          </button>
        </div>
        <p className="text-[10px] text-ink-muted mt-1.5 px-1">按 Enter 发送，Shift+Enter 换行</p>
      </div>
    </div>
  );
}
