import type { StatusTone } from "../lib/mocks";

interface StatusBadgeProps {
  tone: StatusTone;
  label: string;
  size?: "sm" | "md";
}

/** 根据状态色调返回对应的样式类名 */
function toneClasses(tone: StatusTone): string {
  const map: Record<StatusTone, string> = {
    stable: "bg-stable-bg text-stable-ink",
    attention: "bg-attention-bg text-attention-ink",
    critical: "bg-critical text-critical-ink",
    muted: "bg-badge-bg text-badge-ink",
  };
  return map[tone];
}

/** 统一的状态标签组件，用于各页面展示状态信息 */
export function StatusBadge({ tone, label, size = "sm" }: StatusBadgeProps) {
  const sizeClass = size === "sm" ? "px-2.5 py-0.5 text-xs" : "px-3 py-1 text-sm";
  return (
    <span className={`rounded-chip font-medium ${toneClasses(tone)} ${sizeClass}`}>
      {label}
    </span>
  );
}
