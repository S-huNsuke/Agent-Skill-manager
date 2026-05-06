interface ConfirmModalProps {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: "danger" | "default";
  onConfirm: () => void;
  onCancel: () => void;
}

/** 确认对话框组件，用于需要用户确认的操作 */
export function ConfirmModal({
  title,
  message,
  confirmLabel = "确认",
  cancelLabel = "取消",
  tone = "default",
  onConfirm,
  onCancel,
}: ConfirmModalProps) {
  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={onCancel}>
      <article
        className="bg-surface-warm rounded-panel shadow-panel max-w-md w-full mx-4"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-6">
          <h2 className="font-display text-xl font-semibold text-ink mb-2">{title}</h2>
          <p className="text-ink-soft">{message}</p>
        </div>
        <div className="flex justify-end gap-3 px-6 py-4 border-t border-border-soft">
          <button
            type="button"
            onClick={onCancel}
            className="rounded-pill px-5 py-2 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className={`rounded-pill px-5 py-2 text-sm font-medium transition-colors ${
              tone === "danger"
                ? "bg-red-500 text-white hover:bg-red-600"
                : "bg-accent text-white hover:bg-accent-warm"
            }`}
          >
            {confirmLabel}
          </button>
        </div>
      </article>
    </div>
  );
}
