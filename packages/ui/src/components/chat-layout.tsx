import type { ReactNode } from "react";
import { cn } from "../lib/cn";

interface ChatLayoutProps {
  chatList: ReactNode;
  composer: ReactNode;
  transcript: ReactNode;
  sidePanel?: ReactNode;
}

export function ChatLayout({ chatList, composer, transcript, sidePanel }: ChatLayoutProps) {
  return (
    <div className="grid gap-5 xl:grid-cols-[320px_minmax(0,1fr)_340px]">
      <section className={cn("rounded-[2rem] border border-white/10 bg-[linear-gradient(180deg,rgba(12,19,31,0.9),rgba(8,13,22,0.82))] p-4 shadow-[0_20px_80px_rgba(0,0,0,0.22)]")}>
        {chatList}
      </section>
      <section className="flex min-h-[72vh] flex-col overflow-hidden rounded-[2rem] border border-white/10 bg-[linear-gradient(180deg,rgba(12,19,31,0.94),rgba(7,10,18,0.92))] shadow-[0_20px_80px_rgba(0,0,0,0.28)]">
        <div className="flex-1 overflow-hidden p-4 sm:p-5">{transcript}</div>
        <div className="border-t border-white/10 bg-white/[0.03] p-4 sm:p-5">{composer}</div>
      </section>
      <section className="rounded-[2rem] border border-white/10 bg-[linear-gradient(180deg,rgba(12,19,31,0.9),rgba(8,13,22,0.82))] p-4 shadow-[0_20px_80px_rgba(0,0,0,0.22)]">
        {sidePanel ?? <p className="text-sm text-slate-300">Agent events, citations, and metadata appear here.</p>}
      </section>
    </div>
  );
}
