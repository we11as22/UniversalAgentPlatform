import type { ReactNode } from "react";

interface AppShellProps {
  sidebar: ReactNode;
  header: ReactNode;
  children?: ReactNode;
}

export function AppShell({ sidebar, header, children }: AppShellProps) {
  return (
    <div className="relative min-h-screen overflow-hidden bg-[#08111b] text-slate-100">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(56,189,248,0.14),transparent_32%),radial-gradient(circle_at_top_right,rgba(251,191,36,0.12),transparent_30%),linear-gradient(180deg,#09111b_0%,#050810_100%)]" />
      <div className="pointer-events-none absolute inset-x-0 top-0 h-[320px] bg-[linear-gradient(180deg,rgba(255,255,255,0.04),transparent)]" />
      <div className="relative mx-auto grid min-h-screen max-w-[1680px] lg:grid-cols-[19rem_minmax(0,1fr)]">
        <aside className="hidden border-r border-white/10 bg-[linear-gradient(180deg,rgba(8,13,22,0.96),rgba(8,13,22,0.82))] lg:block">
          <div className="sticky top-0 flex h-screen flex-col">{sidebar}</div>
        </aside>
        <div className="flex min-h-screen flex-1 flex-col">
          <header className="sticky top-0 z-20 border-b border-white/10 bg-[rgba(8,12,20,0.82)] px-4 py-4 backdrop-blur-xl sm:px-6 lg:px-8">
            {header}
          </header>
          <div className="border-b border-white/10 bg-white/[0.02] lg:hidden">
            {sidebar}
          </div>
          <main className="flex-1 px-4 py-6 sm:px-6 lg:px-8 lg:py-8">{children}</main>
        </div>
      </div>
    </div>
  );
}
