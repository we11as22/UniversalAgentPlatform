import type { ReactNode } from "react";
import { cn } from "../lib/cn";

const toneClasses = {
  slate: "border-white/10 bg-white/5 text-slate-200",
  cyan: "border-cyan-300/20 bg-cyan-300/10 text-cyan-100",
  emerald: "border-emerald-300/20 bg-emerald-300/10 text-emerald-100",
  amber: "border-amber-300/20 bg-amber-300/10 text-amber-100",
  rose: "border-rose-300/20 bg-rose-300/10 text-rose-100"
};

interface BadgeProps {
  children: ReactNode;
  tone?: keyof typeof toneClasses;
  className?: string;
}

export function Badge({ children, tone = "slate", className }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-3 py-1 text-[11px] font-medium uppercase tracking-[0.22em]",
        toneClasses[tone],
        className
      )}
    >
      {children}
    </span>
  );
}
