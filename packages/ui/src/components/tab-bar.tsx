import { cn } from "../lib/cn";

interface TabItem {
  key: string;
  label: string;
  hint?: string;
}

interface TabBarProps {
  items: TabItem[];
  activeKey: string;
  onChange: (key: string) => void;
  className?: string;
}

export function TabBar({ items, activeKey, onChange, className }: TabBarProps) {
  return (
    <div className={cn("flex flex-wrap gap-2", className)} role="tablist" aria-label="Workspace tabs">
      {items.map((item) => {
        const active = item.key === activeKey;
        return (
          <button
            key={item.key}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onChange(item.key)}
            className={cn(
              "group rounded-[1.25rem] border px-4 py-3 text-left transition duration-200",
              active
                ? "border-cyan-300/30 bg-cyan-300/12 text-white shadow-[0_10px_40px_rgba(34,211,238,0.12)]"
                : "border-white/10 bg-white/[0.03] text-slate-300 hover:border-white/20 hover:bg-white/[0.05]"
            )}
          >
            <div className="text-sm font-medium">{item.label}</div>
            {item.hint ? <div className={cn("mt-1 text-xs", active ? "text-cyan-100/70" : "text-slate-500 group-hover:text-slate-400")}>{item.hint}</div> : null}
          </button>
        );
      })}
    </div>
  );
}
