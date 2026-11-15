import type { ReactNode } from "react";

export function AuthFrame({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle: string;
  children: ReactNode;
}) {
  return (
    <div className="min-h-screen bg-background text-foreground flex items-center justify-center px-4 py-10">
      <div className="w-full max-w-md">
        <p className="text-[15px] font-bold tracking-tight text-broadcast">Obelisk</p>
        <h1 className="mt-6 text-page-title">{title}</h1>
        <p className="mt-1 text-body text-muted-foreground">{subtitle}</p>
        <div className="mt-6 rounded-xl border border-border bg-card p-5 shadow-rb-2">
          {children}
        </div>
      </div>
    </div>
  );
}
