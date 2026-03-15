import type { ReactNode } from "react";
import { cn } from "../lib/cn";

interface PanelProps {
  title?: string;
  kicker?: string;
  description?: string;
  action?: ReactNode;
  children?: ReactNode;
  className?: string;
}

export function Panel({ title, kicker, description, action, children, className }: PanelProps) {
  return (
    <section
      className={cn(
        "relative overflow-hidden rounded-[2rem] border border-white/10 bg-[linear-gradient(180deg,rgba(12,19,31,0.92),rgba(8,13,22,0.84))] p-5 shadow-[0_20px_80px_rgba(0,0,0,0.28)]",
        className
      )}
    >
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(56,189,248,0.08),transparent_35%),radial-gradient(circle_at_bottom_left,rgba(251,191,36,0.08),transparent_35%)]" />
      {(title || kicker || description || action) && (
        <div className="relative z-10 flex items-start justify-between gap-4">
          <div>
            {kicker ? <p className="text-[11px] uppercase tracking-[0.24em] text-cyan-200/80">{kicker}</p> : null}
            {title ? <h2 className="mt-1 text-xl font-semibold text-white">{title}</h2> : null}
            {description ? <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-300">{description}</p> : null}
          </div>
          {action ? <div className="relative z-10 shrink-0">{action}</div> : null}
        </div>
      )}
      <div className="relative z-10 mt-4">{children}</div>
    </section>
  );
}
