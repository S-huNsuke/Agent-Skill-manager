import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import type { DashboardViewModel, ActivityItem, SystemHealthStatus, RecommendedAction, SuggestionTemplate, StatusTone } from "../../lib/types";
import { selectApi } from "../../lib/api";
import { StatusBadge } from "../../components/StatusBadge";
import { EmptyState } from "../../components/EmptyState";

interface HomePageProps {
  dashboard: DashboardViewModel;
  onRefresh?: () => void;
  onOpenAssistant?: () => void;
}

/** 根据状态色调返回高亮卡片的边框与背景样式 */
function getToneClasses(tone: StatusTone): string {
  const map: Record<StatusTone, string> = {
    stable: "border-l-4 border-stable-border bg-stable-bg",
    attention: "border-l-4 border-attention-border bg-attention-bg",
    critical: "border-l-4 border-critical-border",
    muted: "",
  };
  return map[tone];
}

/** 根据活动类型返回图标 */
function activityIcon(type: string): string {
  const map: Record<string, string> = {
    install: "↓",
    uninstall: "↑",
    update: "↻",
    repair: "🔧",
    create: "+",
  };
  return map[type] ?? "•";
}

/** 首页总览组件：AI 助手入口在最上方，下方为统计卡片、系统健康与最近活动 */
export function HomePage({ dashboard, onRefresh, onOpenAssistant }: HomePageProps) {
  const navigate = useNavigate();
  const [activities, setActivities] = useState<ActivityItem[]>([]);
  const [healthStatus, setHealthStatus] = useState<SystemHealthStatus | null>(null);
  const [recommendations, setRecommendations] = useState<RecommendedAction[]>([]);
  const [suggestions, setSuggestions] = useState<SuggestionTemplate[]>([]);
  const [goalInput, setGoalInput] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function loadData() {
      setLoading(true);
      try {
        const api = selectApi();
        const [acts, health, recs, tpls] = await Promise.all([
          api.getRecentActivities(10),
          api.getSystemHealthStatus(),
          api.getRecommendedActions(),
          api.getSuggestionTemplates(),
        ]);
        setActivities(acts);
        setHealthStatus(health);
        setRecommendations(recs);
        setSuggestions(tpls);
      } catch {
        // 静默处理
      } finally {
        setLoading(false);
      }
    }
    loadData();
  }, [dashboard]);

  const overallTone: StatusTone = useMemo(() => {
    if (!healthStatus) return "muted";
    if (healthStatus.overallStatus === "ok") return "stable";
    if (healthStatus.overallStatus === "warning") return "attention";
    return "critical";
  }, [healthStatus]);

  /** 提交 AI 目标并打开助手面板 */
  async function handleQuickGoal() {
    const trimmed = goalInput.trim();
    if (!trimmed) return;
    onOpenAssistant?.();
    try {
      const api = selectApi();
      await api.submitGoal(trimmed);
      setGoalInput("");
      onRefresh?.();
    } catch (err) {
      alert(`提交目标失败: ${err instanceof Error ? err.message : String(err)}`);
    }
  }

  return (
    <section className="animate-page-in">
      {/* AI 助手入口 - 最醒目位置 */}
      <div className="bg-gradient-to-br from-accent/8 via-surface-warm to-surface rounded-panel shadow-panel p-8 mb-6">
        <div className="flex items-start gap-6">
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-2">
              <span className="w-8 h-8 rounded-full bg-accent/15 flex items-center justify-center text-accent text-lg">✦</span>
              <p className="uppercase tracking-widest text-xs text-accent font-body font-medium">AI 助手</p>
            </div>
            <h1 className="font-display text-2xl font-semibold text-ink mb-2">告诉我你想做什么</h1>
            <p className="text-ink-soft leading-relaxed mb-5">描述你的目标，AI 助手会帮你规划技能安装、环境配置和问题修复方案。</p>
            <div className="flex gap-3 items-center">
              <div className="flex-1 relative">
                <input
                  type="text"
                  value={goalInput}
                  onChange={(e) => setGoalInput(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter") handleQuickGoal(); }}
                  placeholder="例如：帮我搭建一个 Web 开发环境..."
                  className="w-full rounded-pill bg-surface border border-border px-5 py-3 text-sm text-ink placeholder:text-ink-muted focus:outline-none focus:ring-2 focus:ring-accent/40 shadow-panel"
                />
              </div>
              <button
                type="button"
                onClick={handleQuickGoal}
                disabled={!goalInput.trim()}
                className="rounded-pill px-6 py-3 text-sm font-medium bg-accent text-white shadow-accent hover:bg-accent-warm transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                开始规划
              </button>
              <button
                type="button"
                onClick={() => onOpenAssistant?.()}
                className="rounded-pill px-5 py-3 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors"
              >
                打开助手
              </button>
            </div>
          </div>
          <div className="hidden lg:flex flex-col gap-2 w-56 shrink-0">
            {suggestions?.slice(0, 3).map((tpl) => (
              <button
                key={tpl.id}
                type="button"
                onClick={() => { setGoalInput(tpl.promptTemplate); }}
                className="text-left p-3 rounded-card bg-surface/80 hover:bg-surface transition-colors shadow-panel"
              >
                <p className="text-sm font-medium text-ink">{tpl.title}</p>
                <p className="text-xs text-ink-muted">{tpl.description}</p>
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* 统计卡片 - 横向一行 */}
      <div className="grid grid-cols-3 gap-5 mb-6">
        {dashboard.highlights.map((item) => (
          <article key={item.id} className={`bg-surface rounded-card shadow-panel p-5 ${getToneClasses(item.tone)}`}>
            <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body">{item.tag}</p>
            <strong className="text-2xl font-semibold text-ink block mt-1">{item.value}</strong>
            <p className="text-sm text-ink-soft mt-1">{item.detail}</p>
          </article>
        ))}
      </div>

      {/* 下方两栏：系统健康 + 最近活动 */}
      <div className="grid grid-cols-[1fr_1fr] gap-6">
        {/* 系统健康 */}
        <article className="bg-surface rounded-panel shadow-panel p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body">系统状态</p>
              <h2 className="font-display text-xl font-semibold text-ink">健康检查</h2>
            </div>
            <StatusBadge
              tone={overallTone}
              label={healthStatus?.overallStatus === "ok" ? "正常" : healthStatus?.overallStatus === "warning" ? "警告" : "异常"}
            />
          </div>
          {loading ? (
            <p className="text-ink-soft text-center py-6">正在加载...</p>
          ) : healthStatus && (healthStatus.agentHealth ?? []).length > 0 ? (
            <div className="flex flex-col gap-2">
              {healthStatus.agentHealth.map((item) => (
                <div key={item.agentId} className="flex items-center justify-between p-3 rounded-card bg-surface-warm">
                  <div className="flex items-center gap-3">
                    <span className={`w-2.5 h-2.5 rounded-full shrink-0 ${
                      item.status === "healthy" ? "bg-stable-ink" : item.status === "warning" ? "bg-attention-ink" : item.status === "not_installed" ? "bg-ink-muted" : "bg-critical-ink"
                    }`} />
                    <span className="text-sm font-medium text-ink">{item.name}</span>
                  </div>
                  <span className="text-xs text-ink-soft">{item.detail}</span>
                </div>
              ))}
              {healthStatus.diskSpace && healthStatus.diskSpace.totalGb > 0 && (
                <div className="flex items-center justify-between p-3 rounded-card bg-surface-warm mt-2">
                  <span className="text-sm font-medium text-ink">磁盘空间</span>
                  <span className="text-xs text-ink-soft">
                    {healthStatus.diskSpace.freeGb.toFixed(1)} GB 可用 / {healthStatus.diskSpace.totalGb.toFixed(1)} GB 总计 ({healthStatus.diskSpace.usedPct.toFixed(0)}%)
                  </span>
                </div>
              )}
            </div>
          ) : (
            <p className="text-ink-soft text-center py-6">暂无健康数据</p>
          )}
          {recommendations.length > 0 && (
            <div className="mt-4 pt-4 border-t border-border-soft">
              <p className="text-xs text-ink-muted font-medium mb-2">待处理建议</p>
              <div className="flex flex-col gap-2">
                {recommendations.slice(0, 3).map((rec) => (
                  <div key={rec.id} className="flex items-center justify-between p-2.5 rounded-card bg-surface-warm">
                    <div className="flex items-center gap-2">
                      <StatusBadge
                        tone={rec.priority === "high" ? "critical" : rec.priority === "medium" ? "attention" : "muted"}
                        label={rec.priority === "high" ? "高" : rec.priority === "medium" ? "中" : "低"}
                        size="sm"
                      />
                      <span className="text-sm text-ink">{rec.action}</span>
                    </div>
                    <button
                      type="button"
                      onClick={() => navigate("/agents")}
                      className="rounded-chip px-3 py-1 text-xs font-medium bg-accent-glow text-accent hover:bg-accent hover:text-white transition-colors"
                    >
                      处理
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}
        </article>

        {/* 最近活动 */}
        <article className="bg-surface rounded-panel shadow-panel p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body">动态</p>
              <h2 className="font-display text-xl font-semibold text-ink">最近活动</h2>
            </div>
            <div className="flex items-center gap-3">
              <span className="text-sm text-ink-muted">{activities?.length ?? 0} 条</span>
              <button
                type="button"
                onClick={() => onRefresh?.()}
                className="rounded-chip px-3 py-1 text-xs font-medium bg-surface-warm text-ink-soft hover:bg-surface-hover hover:text-ink transition-colors"
              >
                刷新
              </button>
            </div>
          </div>
          {activities?.length > 0 ? (
            <div className="flex flex-col gap-2">
              {activities.map((act) => (
                <div key={act.id} className="flex items-center gap-3 p-3 rounded-card hover:bg-surface-hover transition-colors">
                  <span className="w-7 h-7 rounded-full bg-surface-warm flex items-center justify-center text-sm shrink-0">
                    {activityIcon(act.type)}
                  </span>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-ink truncate">{act.detail}</p>
                    <p className="text-xs text-ink-muted">{act.time}</p>
                  </div>
                  <StatusBadge tone={act.status === "success" ? "stable" : "critical"} label={act.status === "success" ? "成功" : "失败"} />
                </div>
              ))}
            </div>
          ) : (
            <EmptyState title="暂无活动记录" description="安装或更新技能后，活动记录将显示在这里" />
          )}
          <div className="mt-4 pt-4 border-t border-border-soft flex flex-col gap-1.5">
            {dashboard.notes?.map((note) => (
              <p key={note} className="text-xs text-ink-muted">• {note}</p>
            ))}
          </div>
        </article>
      </div>
    </section>
  );
}
