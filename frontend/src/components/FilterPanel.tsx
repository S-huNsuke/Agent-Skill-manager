interface FilterOption {
  value: string;
  label: string;
}

interface FilterPanelProps {
  filters: {
    key: string;
    label: string;
    options: FilterOption[];
    value: string;
  }[];
  onChange: (key: string, value: string) => void;
}

/** 筛选面板组件，支持多个筛选维度的组合 */
export function FilterPanel({ filters, onChange }: FilterPanelProps) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      {filters.map((filter) => (
        <select
          key={filter.key}
          value={filter.value}
          onChange={(e) => onChange(filter.key, e.target.value)}
          className="bg-surface rounded-card shadow-panel px-4 py-2 text-sm text-ink focus:outline-none focus:ring-2 focus:ring-accent/40"
        >
          <option value="">{filter.label}</option>
          {filter.options.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      ))}
    </div>
  );
}
