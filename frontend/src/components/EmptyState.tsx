interface EmptyStateProps {
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}

/** 空状态占位组件，用于列表或内容为空时展示 */
export function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <div className="bg-surface rounded-panel shadow-panel p-12 text-center">
      <p className="text-ink-soft text-lg">{title}</p>
      {description && (
        <p className="text-ink-muted text-sm mt-2">{description}</p>
      )}
      {action && (
        <button
          type="button"
          onClick={action.onClick}
          className="mt-4 bg-accent text-white rounded-pill px-6 py-2.5 font-medium shadow-accent hover:bg-accent-warm transition-colors"
        >
          {action.label}
        </button>
      )}
    </div>
  );
}
