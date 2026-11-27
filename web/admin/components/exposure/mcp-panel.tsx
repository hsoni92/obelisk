"use client";

import { GatewayCardSkeleton } from "@/components/skeletons";
import { CopyButton } from "@/components/services/copy-button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CardHeading } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import {
  rtkErrorMessage,
  useGetMCPConfigQuery,
  usePatchMCPConfigMutation,
  useTestMCPMutation,
} from "@/lib/api";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import {
  FlaskConical,
  KeyRound,
  Link2,
  Mail,
  Plug,
  Search,
  Wrench,
  X,
  type LucideIcon,
} from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";

function Block({
  icon: Icon,
  title,
  meta,
  action,
  children,
}: {
  icon: LucideIcon;
  title: string;
  meta?: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="space-y-2">
      <div className="flex min-h-6 items-center gap-1.5">
        <Icon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        <h3 className="text-eyebrow text-muted-foreground">{title}</h3>
        {meta && (
          <span className="text-[11px] text-muted-foreground">· {meta}</span>
        )}
        {action && <span className="ml-auto">{action}</span>}
      </div>
      {children}
    </div>
  );
}

const toolIcon: Record<string, LucideIcon> = {
  web_search: Search,
  send_mail: Mail,
};

export function MCPPanel({ onBusy }: { onBusy: (key: string | null) => void }) {
  const { data: config, isLoading } = useGetMCPConfigQuery(undefined, {
    pollingInterval: 15000,
  });
  const [patchConfig] = usePatchMCPConfigMutation();
  const [testMCP, { isLoading: testing }] = useTestMCPMutation();
  const [tokenReveal, setTokenReveal] = useState<string | null>(null);
  const [confirmRotate, setConfirmRotate] = useState(false);

  const endpoint = useMemo(() => {
    const path = config?.public_path ?? "/mcp";
    return typeof window === "undefined"
      ? path
      : `${window.location.origin}${path}`;
  }, [config?.public_path]);

  const snippet = useMemo(
    () =>
      JSON.stringify(
        {
          mcpServers: {
            obelisk: {
              url: endpoint,
              headers: { Authorization: `Bearer ${tokenReveal ?? "<token>"}` },
            },
          },
        },
        null,
        2,
      ),
    [endpoint, tokenReveal],
  );

  async function act(key: string, fn: () => Promise<void>) {
    onBusy(key);
    try {
      await fn();
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    } finally {
      onBusy(null);
    }
  }

  async function rotate() {
    await act("mcp-token", async () => {
      const res = await patchConfig({ regenerate_token: true }).unwrap();
      if ("token" in res) {
        setTokenReveal(res.token);
        toast.info("New MCP token generated");
      }
    });
  }

  if (isLoading) return <GatewayCardSkeleton />;

  const enabled = config?.enabled ?? false;
  const tokenConfigured = config?.token_configured ?? false;
  const ready = enabled && tokenConfigured;
  const tools = config?.runtime.tools ?? [];
  const status = !enabled
    ? { label: "Off", variant: "default" as const }
    : ready
      ? { label: "Live", variant: "success" as const }
      : { label: "Needs token", variant: "warning" as const };

  return (
    <section
      aria-label="MCP gateway"
      className="flex flex-col overflow-hidden rounded-xl border border-border/70 bg-card shadow-sm"
    >
      <CardHeading
        icon={Plug}
        tone={!enabled ? "neutral" : ready ? "green" : "amber"}
        title="MCP gateway"
        description="One MCP endpoint for your editors and agents — web search and mail"
        badge={<Badge variant={status.variant}>{status.label}</Badge>}
        actions={
          <Switch
            aria-label="Enable MCP gateway"
            checked={enabled}
            onCheckedChange={(v) =>
              void act("mcp-enable", async () => {
                await patchConfig({ enabled: v }).unwrap();
                toast.success(
                  v ? "MCP gateway enabled" : "MCP gateway disabled",
                );
              })
            }
          />
        }
        className="flex-nowrap"
      />
      <div className="flex flex-1 flex-col p-4">
        <div className="flex-1 space-y-5">
          <Block icon={Link2} title="Endpoint">
            <div className="flex items-center gap-1.5">
              <code className="min-w-0 flex-1 truncate rounded-md bg-muted px-2.5 py-1.5 font-mono text-[11px]">
                {endpoint}
              </code>
              <CopyButton value={endpoint} label="Endpoint" />
            </div>
          </Block>

          <Block
            icon={KeyRound}
            title="Access token"
            action={
              <Button
                size="sm"
                className="h-6 px-2 text-[11px]"
                variant={tokenConfigured ? "outline" : "default"}
                onClick={() =>
                  tokenConfigured ? setConfirmRotate(true) : void rotate()
                }
              >
                {tokenConfigured ? "Rotate" : "Generate"}
              </Button>
            }
          >
            <span
              className={cn(
                "flex items-center gap-1.5 text-[13px]",
                tokenConfigured ? "text-foreground" : "text-muted-foreground",
              )}
            >
              <span
                className={cn(
                  "size-1.5 rounded-full",
                  tokenConfigured ? "bg-success" : "bg-border",
                )}
              />
              {tokenConfigured ? "Configured" : "Not generated yet"}
            </span>
            {tokenReveal && (
              <div
                role="status"
                className="mt-3 space-y-3 rounded-lg bg-broadcast-tint p-3 ring-1 ring-broadcast/40"
              >
                <div className="flex items-center justify-between">
                  <p className="text-[11px] font-semibold text-foreground">
                    Copy it now — it won’t be shown again
                  </p>
                  <button
                    type="button"
                    aria-label="Dismiss"
                    onClick={() => setTokenReveal(null)}
                    className="rounded-full p-0.5 text-foreground hover:bg-broadcast/30"
                  >
                    <X className="size-3.5" />
                  </button>
                </div>
                <div className="flex items-center gap-2">
                  <code className="min-w-0 flex-1 break-all font-mono text-[11px]">
                    {tokenReveal}
                  </code>
                  <CopyButton value={tokenReveal} label="Token" />
                </div>
                <div className="flex items-start gap-2">
                  <pre className="min-w-0 flex-1 overflow-x-auto rounded-md bg-background/60 p-2 font-mono text-[10px] leading-4">
                    {snippet}
                  </pre>
                  <CopyButton value={snippet} label="Cursor config" />
                </div>
              </div>
            )}
          </Block>

          <Block
            icon={Wrench}
            title="Tools"
            meta={`${tools.filter((t) => t.enabled).length} of ${tools.length} on`}
          >
            <ul className="divide-y divide-border/60 rounded-lg bg-muted/30 ring-1 ring-border/60">
              {tools.map((t) => {
                const Icon = toolIcon[t.name] ?? Wrench;
                return (
                  <li
                    key={t.name}
                    className="flex items-center gap-3 px-3 py-2.5"
                  >
                    <Icon className="size-4 shrink-0 text-muted-foreground" />
                    <div className="min-w-0 flex-1">
                      <p className="font-mono text-[12px] font-medium leading-4">
                        {t.name}
                      </p>
                      <p
                        className={cn(
                          "mt-0.5 text-[11px] leading-4",
                          t.ready ? "text-muted-foreground" : "text-warning",
                        )}
                      >
                        {t.ready
                          ? t.destructive
                            ? `Ready · up to ${config?.runtime.mail_limit_per_hour}/hour, ${config?.runtime.mail_limit_per_day}/day`
                            : "Ready"
                          : t.message}
                      </p>
                    </div>
                    <Switch
                      aria-label={`${t.enabled ? "Disable" : "Enable"} ${t.name}`}
                      checked={t.enabled}
                      onCheckedChange={(v) =>
                        void act(`mcp-tool-${t.name}`, async () => {
                          await patchConfig({
                            tools: { [t.name]: v },
                          }).unwrap();
                          toast.success(
                            `${t.name} ${v ? "enabled" : "disabled"}`,
                          );
                        })
                      }
                    />
                  </li>
                );
              })}
            </ul>
            <p className="text-[11px] leading-4 text-muted-foreground">
              Tools stay listed to agents even when off or not ready; calling
              one returns a message that says what to fix. Mail recipients are
              not restricted, so keep <code>send_mail</code> off unless you need
              it.
            </p>
          </Block>

          {(config?.send_log?.length ?? 0) > 0 && (
            <Block icon={Mail} title="Recent sends">
              <ul className="max-h-32 divide-y divide-border/60 overflow-y-auto rounded-lg bg-muted/30 ring-1 ring-border/60">
                {config!.send_log.slice(0, 8).map((e, i) => (
                  <li
                    key={i}
                    className="flex items-center gap-3 px-3 py-2 text-[11px]"
                  >
                    <span className="shrink-0 text-muted-foreground">
                      {new Date(e.time).toLocaleString()}
                    </span>
                    <span className="min-w-0 flex-1 truncate">
                      {e.to} — {e.subject}
                    </span>
                    <Badge
                      variant={
                        e.outcome === "ok"
                          ? "success"
                          : e.outcome === "deduped"
                            ? "default"
                            : "warning"
                      }
                    >
                      {e.outcome}
                    </Badge>
                  </li>
                ))}
              </ul>
            </Block>
          )}

          {(config?.runtime.auth_failures ?? 0) > 0 && (
            <p className="rounded-lg bg-warning-tint px-3 py-2 text-[11px] leading-4 text-warning">
              {config!.runtime.auth_failures} rejected request(s) since start
              {config!.runtime.last_auth_failure
                ? `, last at ${new Date(config!.runtime.last_auth_failure).toLocaleString()}`
                : ""}
              .
            </p>
          )}
        </div>

        <footer className="mt-5 flex items-center gap-3 border-t border-border/60 pt-3">
          <FlaskConical
            className="size-3.5 shrink-0 text-muted-foreground"
            aria-hidden
          />
          <p className="min-w-0 flex-1 truncate text-[12px] text-muted-foreground">
            {ready
              ? "Ready for clients."
              : "Turn the gateway on and generate a token."}
          </p>
          <Button
            size="sm"
            variant={ready ? "default" : "outline"}
            disabled={!ready || testing}
            onClick={() =>
              void act("mcp-test", async () => {
                const res = await testMCP().unwrap();
                if (res.ok) {
                  toast.success(
                    `MCP OK — ${res.tool_count} tools${res.search ? `, search ${res.search}` : ""}`,
                  );
                } else {
                  toast.error("MCP test failed", res.error ?? "Test failed");
                }
              })
            }
          >
            {testing ? "Testing…" : "Test"}
          </Button>
        </footer>
      </div>

      <AlertDialog open={confirmRotate} onOpenChange={setConfirmRotate}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Rotate the MCP token?</AlertDialogTitle>
            <AlertDialogDescription>
              The old token stops working immediately, on every machine. Update
              each editor’s MCP config with the new token.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                setConfirmRotate(false);
                void rotate();
              }}
            >
              Rotate
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
