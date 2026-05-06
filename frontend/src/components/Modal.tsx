import { useEffect, type ReactNode } from "react";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  subtitle?: string;
  children: ReactNode;
  width?: string;
}

/** 通用模态框组件，支持标题、副标题和自定义内容 */
export function Modal({ open, onClose, title, subtitle, children, width = "max-w-2xl" }: ModalProps) {
  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={onClose}>
      <article
        className={`bg-surface-warm rounded-panel shadow-panel ${width} w-full mx-4 max-h-[85vh] flex flex-col`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between p-6 border-b border-border-soft shrink-0">
          <div>
            <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">{subtitle}</p>
            <h2 className="font-display text-xl font-semibold text-ink">{title}</h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="w-8 h-8 rounded-full bg-surface hover:bg-surface-hover flex items-center justify-center text-ink-soft hover:text-ink transition-colors"
          >
            ✕
          </button>
        </div>
        <div className="p-6 overflow-y-auto flex-1">
          {children}
        </div>
      </article>
    </div>
  );
}
