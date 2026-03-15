export default function LoginPage() {
  const keycloak = process.env.KEYCLOAK_URL ?? "http://localhost:18081";
  const realm = process.env.KEYCLOAK_REALM ?? "uap";
  const appUrl = process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:3200";
  const redirectUri = encodeURIComponent(appUrl);
  const authUrl = `${keycloak}/realms/${realm}/protocol/openid-connect/auth?client_id=uap-chat-web&response_type=code&scope=openid&redirect_uri=${redirectUri}`;

  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#08111b] px-6 text-white">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(56,189,248,0.18),transparent_32%),radial-gradient(circle_at_bottom_right,rgba(251,191,36,0.14),transparent_28%),linear-gradient(180deg,#08111b_0%,#04070e_100%)]" />
      <div className="relative grid w-full max-w-6xl gap-6 lg:grid-cols-[1.1fr_0.9fr]">
        <section className="rounded-[2.5rem] border border-white/10 bg-[linear-gradient(180deg,rgba(12,18,29,0.92),rgba(8,12,21,0.84))] p-8 shadow-[0_40px_120px_rgba(0,0,0,0.4)] lg:p-12">
          <p className="font-display text-[11px] uppercase tracking-[0.32em] text-cyan-200/80">UniversalAgentPlatform</p>
          <h1 className="font-display mt-4 text-5xl font-semibold tracking-tight">Enterprise agent workspace built for grounded chat, voice and control.</h1>
          <p className="mt-5 max-w-2xl text-lg leading-8 text-slate-300">
            Authenticate through Keycloak to access tenant-scoped conversations, voice sessions, Qdrant-backed RAG agents and operational telemetry across the full platform.
          </p>
          <div className="mt-8 flex flex-wrap gap-2">
            <span className="rounded-full border border-cyan-300/20 bg-cyan-300/10 px-4 py-2 text-xs uppercase tracking-[0.22em] text-cyan-100">SSE streaming</span>
            <span className="rounded-full border border-emerald-300/20 bg-emerald-300/10 px-4 py-2 text-xs uppercase tracking-[0.22em] text-emerald-100">Voice-first agents</span>
            <span className="rounded-full border border-amber-300/20 bg-amber-300/10 px-4 py-2 text-xs uppercase tracking-[0.22em] text-amber-100">RAG knowledge</span>
          </div>
        </section>
        <section className="rounded-[2.5rem] border border-white/10 bg-[linear-gradient(180deg,rgba(12,18,29,0.92),rgba(8,12,21,0.84))] p-8 shadow-[0_40px_120px_rgba(0,0,0,0.4)] lg:p-10">
          <p className="text-sm uppercase tracking-[0.28em] text-slate-400">Secure entry</p>
          <h2 className="font-display mt-4 text-3xl font-semibold">Continue with enterprise identity</h2>
          <p className="mt-4 text-base leading-7 text-slate-300">
            This environment is wired for Keycloak-backed OIDC. Sign in to open the chat cockpit and verify the full path from conversation creation to streaming response and voice projection.
          </p>
          <a className="mt-8 inline-flex rounded-full bg-cyan-300 px-6 py-3 font-medium text-slate-950 transition hover:bg-cyan-200" href={authUrl}>
            Continue with Keycloak
          </a>
          <div className="mt-8 grid gap-3">
            <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Redirect</p>
              <p className="font-mono-ui mt-3 text-xs text-slate-200">{appUrl}</p>
            </div>
            <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Identity realm</p>
              <p className="mt-3 text-sm text-slate-200">{realm}</p>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
