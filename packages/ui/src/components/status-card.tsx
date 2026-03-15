interface StatusCardProps {
  title: string;
  value: string;
  hint: string;
}

export function StatusCard({ title, value, hint }: StatusCardProps) {
  return (
    <div className="relative overflow-hidden rounded-[1.75rem] border border-white/10 bg-[linear-gradient(180deg,rgba(16,24,39,0.9),rgba(9,13,23,0.82))] p-5 shadow-[0_20px_60px_rgba(0,0,0,0.2)]">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(56,189,248,0.1),transparent_35%)]" />
      <p className="relative text-[11px] uppercase tracking-[0.24em] text-slate-400">{title}</p>
      <p className="relative mt-5 text-4xl font-semibold tracking-tight text-white">{value}</p>
      <p className="relative mt-3 text-sm leading-6 text-slate-300">{hint}</p>
    </div>
  );
}
