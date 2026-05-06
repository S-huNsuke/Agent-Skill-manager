interface SearchBarProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

/** 统一的搜索栏组件，支持输入搜索和清除 */
export function SearchBar({ value, onChange, placeholder = "搜索..." }: SearchBarProps) {
  return (
    <div className="relative flex-1 min-w-[200px]">
      <input
        type="text"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full bg-surface rounded-card shadow-panel px-4 py-2 text-sm text-ink placeholder:text-ink-muted focus:outline-none focus:ring-2 focus:ring-accent/40 pr-8"
      />
      {value && (
        <button
          type="button"
          onClick={() => onChange("")}
          className="absolute right-2 top-1/2 -translate-y-1/2 w-5 h-5 flex items-center justify-center rounded-full bg-surface-hover text-ink-muted hover:text-ink text-xs"
        >
          ✕
        </button>
      )}
    </div>
  );
}
