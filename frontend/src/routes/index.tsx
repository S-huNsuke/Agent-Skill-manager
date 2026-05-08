import { NavLink, Route, Routes } from "react-router-dom";
import { useState } from "react";
import { AssistantPanel } from "../features/assistant/AssistantPanel";
import { AgentsPage } from "../features/agents/AgentsPage";
import { HomePage } from "../features/home/HomePage";
import { ProjectsPage } from "../features/projects/ProjectsPage";
import { SettingsPage } from "../features/settings/SettingsPage";
import { SkillsPage } from "../features/skills/SkillsPage";
import { StorePage } from "../features/store/StorePage";
import { waitForApi } from "../lib/api";
import type { AppSnapshot } from "../lib/mocks";

interface AppRoutesProps {
  snapshot: AppSnapshot;
  onRefresh: () => void;
}

const navItems = [
  { to: "/", label: "概览", icon: "◉" },
  { to: "/agents", label: "代理", icon: "⬡" },
  { to: "/skills", label: "技能", icon: "✦" },
  { to: "/store", label: "商店", icon: "◈" },
  { to: "/projects", label: "项目", icon: "▣" },
  { to: "/settings", label: "设置", icon: "⚙" },
] as const;

/** 根据导航激活状态和侧边栏折叠状态生成对应的 Tailwind 类名 */
function navLinkClass(isActive: boolean, collapsed: boolean): string {
  const base = collapsed
    ? "flex items-center justify-center rounded-chip px-0 py-2.5 text-sm transition-colors w-full"
    : "flex items-center rounded-chip px-4 py-2.5 text-sm transition-colors";
  return isActive
    ? `${base} bg-accent-glow text-accent font-medium`
    : `${base} text-ink-soft hover:bg-surface-hover`;
}

/** 处理技能操作（安装/卸载/更新/修复） */
async function handleSkillAction(action: string, agentID: string, skillName?: string, cachePath?: string) {
  const api = await waitForApi();
  try {
    let result: string;
    switch (action) {
      case "install":
        result = await api.installSkill(agentID, skillName ?? "", cachePath ?? "");
        break;
      case "uninstall":
        result = await api.uninstallSkill(agentID, skillName ?? "");
        break;
      case "update":
        result = await api.updateSkill(agentID, skillName ?? "", "");
        break;
      case "repair":
        result = "ok";
        break;
      default:
        result = "unknown action";
    }
    if (result !== "ok") {
      alert(`操作失败: ${result}`);
    }
  } catch (err) {
    alert(`操作出错: ${err instanceof Error ? err.message : String(err)}`);
  }
}

/** 推进 AI 助手任务 */
async function handleAdvanceTask(taskID: string, action: string, onRefresh: () => void): Promise<AppSnapshot["assistant"]> {
  const api = await waitForApi();
  const result = await api.advanceAssistantTask(taskID, action);
  // 推进任务后刷新 snapshot 以获取最新状态
  onRefresh();
  return result;
}

/** 重置 AI 助手任务 */
async function handleResetTask(onRefresh: () => void): Promise<AppSnapshot["assistant"]> {
  const api = await waitForApi();
  const result = await api.resetAssistantTask();
  // 重置后刷新 snapshot 以获取最新状态
  onRefresh();
  return result;
}

/** 应用主路由组件，包含侧边栏、内容区域与右侧 AI 助手面板 */
export function AppRoutes({ snapshot, onRefresh }: AppRoutesProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [assistantOpen, setAssistantOpen] = useState(true);

  return (
    <div className="flex min-h-screen">
      {/* 左侧导航栏 */}
      <aside
        className={`fixed left-0 top-0 h-screen bg-surface-warm rounded-r-panel shadow-panel p-6 flex flex-col gap-6 overflow-y-auto overflow-x-hidden transition-all duration-300 ease-in-out z-50 ${collapsed ? "w-16 px-2 py-6" : "w-72"}`}
      >
        <div className={`flex flex-col gap-3 transition-opacity duration-200 ${collapsed ? "opacity-0 h-0 overflow-hidden" : "opacity-100"}`}>
          <p className="uppercase tracking-widest text-xs text-ink-muted font-body">Agent Skills Manager</p>
          <h1 className="font-display text-2xl font-semibold text-ink">技能管理器</h1>
          <p className="text-sm text-ink-soft">管理本机 AI 代理的技能安装、更新和配置。</p>
        </div>

        <nav className="flex flex-col gap-1" aria-label="主导航">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/"}
              aria-label={item.label}
              title={collapsed ? item.label : undefined}
              className={({ isActive }) => navLinkClass(isActive, collapsed)}
            >
              <span className={`text-base ${collapsed ? "" : "mr-2"}`}>{item.icon}</span>
              <span className={`transition-all duration-200 ${collapsed ? "hidden" : ""}`}>{item.label}</span>
            </NavLink>
          ))}
        </nav>

        <div className={`mt-auto pt-4 border-t border-border-soft flex flex-col gap-2 ${collapsed ? "items-center" : ""}`}>
          <button
            type="button"
            onClick={() => setAssistantOpen((o) => !o)}
            title={assistantOpen ? "关闭 AI 助手" : "打开 AI 助手"}
            className={`flex items-center rounded-chip text-sm text-accent hover:bg-accent-glow transition-colors ${collapsed ? "justify-center px-0 py-2 w-full" : "px-4 py-2 w-full"}`}
          >
            <span className="text-base">✦</span>
            <span className={`transition-all duration-200 ${collapsed ? "hidden" : "ml-2"}`}>{assistantOpen ? "关闭助手" : "打开助手"}</span>
          </button>
          <button
            type="button"
            onClick={onRefresh}
            title="刷新数据"
            className={`flex items-center rounded-chip text-sm text-ink-soft hover:bg-surface-hover transition-colors ${collapsed ? "justify-center px-0 py-2 w-full" : "px-4 py-2 w-full"}`}
          >
            <span className="text-base">↻</span>
            <span className={`transition-all duration-200 ${collapsed ? "hidden" : "ml-2"}`}>刷新数据</span>
          </button>
          <button
            type="button"
            onClick={() => setCollapsed((c) => !c)}
            title={collapsed ? "展开侧边栏" : "收起侧边栏"}
            className={`flex items-center rounded-chip text-sm text-ink-soft hover:bg-surface-hover transition-colors ${collapsed ? "justify-center px-0 py-2 w-full" : "px-4 py-2 w-full"}`}
          >
            <span className={`text-base transition-transform duration-300 ${collapsed ? "rotate-180" : ""}`}>‹</span>
            <span className={`transition-all duration-200 ${collapsed ? "hidden" : "ml-2"}`}>{collapsed ? "展开" : "收起"}</span>
          </button>
          <p className={`text-xs text-ink-muted transition-all duration-200 ${collapsed ? "hidden" : ""}`}>
            Agent Skills Manager v0.1.0
          </p>
        </div>
      </aside>

      {/* 中间内容区域 */}
      <main className={`flex-1 overflow-y-auto p-8 min-h-screen transition-all duration-300 ease-in-out ${collapsed ? "ml-16" : "ml-72"} ${assistantOpen ? "mr-80" : "mr-0"}`}>
        <Routes>
          <Route path="/" element={<HomePage dashboard={snapshot.dashboard} onRefresh={onRefresh} onOpenAssistant={() => setAssistantOpen(true)} />} />
          <Route
            path="/agents"
            element={
              <AgentsPage
                agents={snapshot.agents}
                skills={snapshot.skills}
                onRefresh={onRefresh}
                onAction={handleSkillAction}
              />
            }
          />
          <Route
            path="/skills"
            element={
              <SkillsPage
                skills={snapshot.skills}
                onRefresh={onRefresh}
                onAction={handleSkillAction}
              />
            }
          />
          <Route
            path="/store"
            element={
              <StorePage
                items={snapshot.store}
                agents={snapshot.agents}
                onRefresh={onRefresh}
                onAction={handleSkillAction}
              />
            }
          />
          <Route path="/projects" element={<ProjectsPage projects={snapshot.projects} agents={snapshot.agents} skills={snapshot.skills} storeItems={snapshot.store} onRefresh={onRefresh} />} />
          <Route
            path="/settings"
            element={<SettingsPage diagnostics={snapshot.diagnostics} onRefresh={onRefresh} />}
          />
        </Routes>
      </main>

      {/* 右侧 AI 助手面板 */}
      <div
        className={`fixed right-0 top-0 h-screen w-80 transition-all duration-300 ease-in-out z-40 ${
          assistantOpen ? "translate-x-0" : "translate-x-full"
        }`}
      >
        <AssistantPanel
          task={snapshot.assistant}
          onAdvance={async (taskID, action) => {
            return await handleAdvanceTask(taskID, action, onRefresh);
          }}
          onReset={async () => {
            return await handleResetTask(onRefresh);
          }}
          onClose={() => setAssistantOpen(false)}
        />
      </div>
    </div>
  );
}
