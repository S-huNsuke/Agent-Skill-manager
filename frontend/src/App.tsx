import { useCallback, Component, useEffect, useState, startTransition, type ReactNode } from "react";
import { HashRouter } from "react-router-dom";
import { mockApi, isRunningInWails, waitForApi } from "./lib/api";
import type { AppSnapshot } from "./lib/mocks";
import type { FrontendApi } from "./lib/api";
import { AppRoutes } from "./routes";

/** 全屏居中的加载状态组件 */
function LoadingState() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-canvas">
      <div className="bg-surface-warm rounded-card shadow-panel p-10 text-center max-w-md">
        <p className="text-sm text-ink-muted font-body tracking-wide mb-3">正在准备</p>
        <h1 className="text-2xl font-display text-ink mb-4">正在整理本机技能</h1>
        <p className="text-ink-soft font-body leading-relaxed">正在连接本地适配器，加载技能与项目数据。</p>
      </div>
    </div>
  );
}

/** 全屏错误状态组件 */
function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-canvas">
      <div className="bg-surface-warm rounded-card shadow-panel p-10 text-center max-w-md">
        <p className="text-sm text-ink-muted font-body tracking-wide mb-3">加载遇到问题</p>
        <h1 className="text-2xl font-display text-ink mb-4">无法连接后端服务</h1>
        <p className="text-ink-soft font-body leading-relaxed mb-6">{message}</p>
        <button
          onClick={onRetry}
          className="px-6 py-2 bg-accent text-white rounded-pill font-body text-sm hover:opacity-90 transition-opacity"
        >
          使用示例数据继续
        </button>
      </div>
    </div>
  );
}

/** React 错误边界组件，捕获子组件树中的渲染错误 */
class ErrorBoundary extends Component<{ children: ReactNode }, { hasError: boolean; error: Error | null }> {
  state: { hasError: boolean; error: Error | null } = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("[ErrorBoundary] Rendering error:", error, info.componentStack);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen flex items-center justify-center bg-canvas p-8">
          <div className="bg-surface-warm rounded-card shadow-panel p-10 text-center max-w-lg">
            <p className="text-sm text-ink-muted font-body tracking-wide mb-3">渲染错误</p>
            <h1 className="text-2xl font-display text-ink mb-4">页面加载遇到问题</h1>
            <p className="text-ink-soft font-body leading-relaxed mb-4 text-left break-all">{this.state.error?.message ?? "未知错误"}</p>
            <button
              onClick={() => this.setState({ hasError: false, error: null })}
              className="px-6 py-2 bg-accent text-white rounded-pill font-body text-sm hover:opacity-90 transition-opacity"
            >
              重试
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}

/** 确保数组字段不为 null，防止 Go nil 切片序列化为 JSON null 导致前端崩溃 */
function ensureArrays(snapshot: AppSnapshot): AppSnapshot {
  return {
    ...snapshot,
    agents: snapshot.agents ?? [],
    skills: snapshot.skills ?? [],
    store: snapshot.store ?? [],
    projects: snapshot.projects ?? [],
    diagnostics: snapshot.diagnostics ?? [],
    assistant: {
      ...snapshot.assistant,
      records: snapshot.assistant?.records ?? [],
      planSteps: snapshot.assistant?.planSteps ?? [],
      resolvedActions: snapshot.assistant?.resolvedActions ?? [],
    },
    dashboard: {
      ...snapshot.dashboard,
      highlights: snapshot.dashboard?.highlights ?? [],
      tasks: snapshot.dashboard?.tasks ?? [],
      notes: snapshot.dashboard?.notes ?? [],
    },
  };
}

/** 应用根组件，负责加载快照数据并渲染路由 */
export default function App() {
  const [snapshot, setSnapshot] = useState<AppSnapshot | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [api, setApi] = useState<FrontendApi | null>(null);

  const loadSnapshot = useCallback(async (apiInstance: FrontendApi) => {
    try {
      const rawSnapshot = await apiInstance.getSnapshot();
      const nextSnapshot = ensureArrays(rawSnapshot);
      startTransition(() => {
        setSnapshot(nextSnapshot);
      });
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      if (isRunningInWails()) {
        console.warn("Wails API failed, falling back to mock data:", msg);
        try {
          const fallbackSnapshot = await mockApi.getSnapshot();
          startTransition(() => {
            setSnapshot(fallbackSnapshot);
          });
        } catch {
          setError(msg);
        }
      } else {
        setError(msg);
      }
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    async function init() {
      const selectedApi = await waitForApi();
      if (cancelled) return;
      setApi(selectedApi);
      await loadSnapshot(selectedApi);
    }
    void init();
    return () => {
      cancelled = true;
    };
  }, [loadSnapshot]);

  /** 刷新快照数据 */
  const refreshSnapshot = useCallback(async () => {
    try {
      const apiInstance = await waitForApi();
      if (api !== apiInstance) {
        setApi(apiInstance);
      }
      const rawSnapshot = await apiInstance.refreshSnapshot();
      const nextSnapshot = ensureArrays(rawSnapshot);
      startTransition(() => {
        setSnapshot(nextSnapshot);
      });
    } catch (err) {
      console.warn("Refresh failed:", err);
    }
  }, [api]);

  if (error) {
    return (
      <ErrorState
        message={error}
        onRetry={() => {
          setError(null);
          mockApi.getSnapshot().then((s) => {
            startTransition(() => {
              setSnapshot(ensureArrays(s));
            });
          });
        }}
      />
    );
  }

  if (!snapshot) {
    return <LoadingState />;
  }

  return (
    <ErrorBoundary>
      <HashRouter>
        <AppRoutes snapshot={snapshot} onRefresh={refreshSnapshot} />
      </HashRouter>
    </ErrorBoundary>
  );
}
