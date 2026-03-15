import type { ReactNode } from "react";
import { cn } from "../lib/cn";

interface ActionCardProps {
  title: string;
  description: string;
  href?: string;
  onClick?: () => void;
  meta?: ReactNode;
  className?: string;
}

export function ActionCard({ title, description, href, onClick, meta, className }: ActionCardProps) {
  const content = (
    <div
      className={cn(
        "rounded-[1.75rem] border border-white/10 bg-white/[0.04] p-4 transition duration-200 hover:-translate-y-0.5 hover:border-cyan-300/20 hover:bg-cyan-300/[0.06]",
        className
      )}
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <h3 className="text-base font-medium text-white">{title}</h3>
          <p className="mt-2 text-sm leading-6 text-slate-300">{description}</p>
        </div>
        {meta ? <div className="shrink-0">{meta}</div> : null}
      </div>
    </div>
  );

  if (href) {
    return (
      <a href={href} target="_blank" rel="noreferrer" className="block">
        {content}
      </a>
    );
  }

  return (
    <button type="button" onClick={onClick} className="block w-full text-left">
      {content}
    </button>
  );
}
